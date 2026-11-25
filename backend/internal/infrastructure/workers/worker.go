package workers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"lms-backend/internal/domain/notifications"
	"lms-backend/internal/infrastructure/redis"
	"lms-backend/pkg/logger"
	"sync"
	"time"
)

// JobHandler processes a specific job type
type JobHandler interface {
	Handle(ctx context.Context, job notifications.Job) error
}

// PermanentError marks a failure that retrying cannot fix (bad payload,
// rejected recipient). Such jobs go straight to the dead-letter list.
type PermanentError struct{ Err error }

func (e PermanentError) Error() string { return e.Err.Error() }
func (e PermanentError) Unwrap() error { return e.Err }

// Permanent wraps err so the worker does not retry it.
func Permanent(err error) error {
	if err == nil {
		return nil
	}
	return PermanentError{Err: err}
}

// MaxJobAttempts is how many times a job runs before it is dead-lettered.
const MaxJobAttempts = 5

// retryBackoff is the wait before retry n (1-based).
func retryBackoff(attempt int) time.Duration {
	switch attempt {
	case 1:
		return 10 * time.Second
	case 2:
		return 30 * time.Second
	case 3:
		return 2 * time.Minute
	default:
		return 10 * time.Minute
	}
}

// Worker consumes jobs from a Redis list with a small pool of goroutines.
//
// Earlier it ran one job at a time and dropped any job whose handler failed,
// so a single long video transcode delayed every email and notification, and
// an SMTP outage lost emails for good. Now:
//   - Concurrency goroutines consume the queue; "heavy" job types (video
//     transcodes) are limited to one at a time so they cannot starve the rest.
//   - A failed job is retried with backoff (10s, 30s, 2m, 10m) via a sorted
//     set "<queue>:retry"; after MaxJobAttempts (or a per-type limit), or on a
//     PermanentError, it is kept in "<queue>:dead" (last 1000) for
//     inspection instead of vanishing.
//   - Retries are safe: every job carries an ID, notifications it creates
//     derive their IDs from it (inserted with ON CONFLICT DO NOTHING), and a
//     job ID that already completed is skipped.
type Worker struct {
	redis       *redis.Client
	queue       string
	router      map[string]JobHandler
	heavy       map[string]bool
	maxAttempts map[string]int
	concurrency int
	heavySlot   chan struct{}
	now         func() time.Time
}

// NewWorker creates a new background worker
func NewWorker(redisClient *redis.Client, queue string) *Worker {
	return &Worker{
		redis:       redisClient,
		queue:       queue,
		router:      make(map[string]JobHandler),
		heavy:       make(map[string]bool),
		maxAttempts: make(map[string]int),
		concurrency: 4,
		heavySlot:   make(chan struct{}, 1),
		now:         time.Now,
	}
}

// RegisterHandler registers a job handler for a specific job type
func (w *Worker) RegisterHandler(jobType string, handler JobHandler) {
	w.router[jobType] = handler
}

// SetMaxAttempts overrides MaxJobAttempts for one job type (for example
// fewer attempts for expensive transcodes).
func (w *Worker) SetMaxAttempts(jobType string, n int) {
	if n < 1 {
		n = 1
	}
	w.maxAttempts[jobType] = n
}

func (w *Worker) attemptsFor(jobType string) int {
	if n, ok := w.maxAttempts[jobType]; ok {
		return n
	}
	return MaxJobAttempts
}

// doneTTL is how long a completed job ID is remembered, so a job that was
// queued twice (or retried after it actually succeeded) runs only once.
const doneTTL = 7 * 24 * time.Hour

func (w *Worker) doneKey(id string) string { return w.key() + ":done:" + id }

// MarkHeavy limits a job type to one concurrent run.
func (w *Worker) MarkHeavy(jobType string) { w.heavy[jobType] = true }

// SetConcurrency sets how many jobs run at once (minimum 1).
func (w *Worker) SetConcurrency(n int) {
	if n < 1 {
		n = 1
	}
	w.concurrency = n
}

func (w *Worker) key() string      { return fmt.Sprintf("queue:%s", w.queue) }
func (w *Worker) retryKey() string { return w.key() + ":retry" }
func (w *Worker) deadKey() string  { return w.key() + ":dead" }

// Run starts the worker loop and blocks until ctx is cancelled.
func (w *Worker) Run(ctx context.Context) {
	logger.Info(ctx, "Worker started", "queue", w.queue, "concurrency", w.concurrency)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		w.promoteLoop(ctx)
	}()
	for i := 0; i < w.concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.consume(ctx)
		}()
	}
	wg.Wait()
	logger.Info(ctx, "Worker shut down", "queue", w.queue)
}

func (w *Worker) consume(ctx context.Context) {
	for ctx.Err() == nil {
		raw, ok, err := w.redis.BLPop(ctx, 2*time.Second, w.key())
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			logger.Error(ctx, "Failed to dequeue job", "queue", w.queue, "error", err)
			time.Sleep(time.Second)
			continue
		}
		if !ok {
			continue
		}
		var job notifications.Job
		if err := json.Unmarshal([]byte(raw), &job); err != nil {
			logger.Error(ctx, "Discarding malformed job", "queue", w.queue, "error", err)
			w.deadLetter(ctx, raw)
			continue
		}
		w.run(ctx, &job)
	}
}

func (w *Worker) run(ctx context.Context, job *notifications.Job) {
	if w.heavy[job.Type] {
		select {
		case w.heavySlot <- struct{}{}:
			defer func() { <-w.heavySlot }()
		case <-ctx.Done():
			// Put it back so it is not lost on shutdown.
			_ = w.push(context.Background(), job)
			return
		}
	}
	if job.ID != "" {
		if _, err := w.redis.Get(ctx, w.doneKey(job.ID)); err == nil {
			logger.Info(ctx, "Skipping job that already completed", "job_type", job.Type, "job_id", job.ID)
			return
		}
	}
	jobCtx := ctx
	if job.ID != "" {
		jobCtx = notifications.WithJobKey(ctx, job.ID)
	}
	err := w.processJob(jobCtx, job)
	if err == nil {
		if job.ID != "" {
			_ = w.redis.Set(context.Background(), w.doneKey(job.ID), "1", doneTTL)
		}
		return
	}
	job.Attempts++
	job.LastError = err.Error()
	var permanent PermanentError
	if errors.As(err, &permanent) || job.Attempts >= w.attemptsFor(job.Type) {
		logger.Error(ctx, "Job failed permanently", "job_type", job.Type, "attempts", job.Attempts, "error", err)
		if data, mErr := json.Marshal(job); mErr == nil {
			w.deadLetter(ctx, string(data))
		}
		return
	}
	delay := retryBackoff(job.Attempts)
	logger.Warn(ctx, "Job failed; will retry", "job_type", job.Type, "attempt", job.Attempts, "retry_in", delay.String(), "error", err)
	data, mErr := json.Marshal(job)
	if mErr != nil {
		return
	}
	if zErr := w.redis.ZAdd(context.Background(), w.retryKey(), float64(w.now().Add(delay).UnixMilli()), string(data)); zErr != nil {
		logger.Error(ctx, "Failed to schedule job retry", "job_type", job.Type, "error", zErr)
	}
}

// promoteLoop moves due retries back onto the queue. ZRem decides which
// process moves a member, so several servers never duplicate a retry.
func (w *Worker) promoteLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.promoteDue(ctx)
		}
	}
}

func (w *Worker) promoteDue(ctx context.Context) {
	due, err := w.redis.ZRangeByScoreUpTo(ctx, w.retryKey(), float64(w.now().UnixMilli()), 100)
	if err != nil {
		return
	}
	for _, member := range due {
		won, err := w.redis.ZRem(ctx, w.retryKey(), member)
		if err != nil || !won {
			continue
		}
		if err := w.redis.RPush(ctx, w.key(), member); err != nil {
			// Put it back for the next tick rather than lose it.
			_ = w.redis.ZAdd(ctx, w.retryKey(), float64(w.now().UnixMilli()), member)
		}
	}
}

func (w *Worker) push(ctx context.Context, job *notifications.Job) error {
	data, err := json.Marshal(job)
	if err != nil {
		return err
	}
	return w.redis.RPush(ctx, w.key(), string(data))
}

func (w *Worker) deadLetter(ctx context.Context, raw string) {
	if err := w.redis.RPush(context.Background(), w.deadKey(), raw); err != nil {
		logger.Error(ctx, "Failed to store dead-lettered job", "error", err)
		return
	}
	_ = w.redis.LTrim(context.Background(), w.deadKey(), -1000, -1)
}

func (w *Worker) processJob(ctx context.Context, job *notifications.Job) error {
	handler, ok := w.router[job.Type]
	if !ok {
		return Permanent(fmt.Errorf("no handler registered for job type: %s", job.Type))
	}

	logger.Info(ctx, "Processing job", "job_type", job.Type, "attempt", job.Attempts+1)

	if err := handler.Handle(ctx, *job); err != nil {
		return fmt.Errorf("handler failed: %w", err)
	}

	logger.Info(ctx, "Job processed successfully", "job_type", job.Type)
	return nil
}

// QueueStats reports queue depth for monitoring.
type QueueStats struct {
	Pending int64 `json:"pending"`
	Retry   int64 `json:"retry"`
	Dead    int64 `json:"dead"`
}

// Stats returns the current queue depths.
func (w *Worker) Stats(ctx context.Context) (QueueStats, error) {
	var s QueueStats
	var err error
	if s.Pending, err = w.redis.LLen(ctx, w.key()); err != nil {
		return s, err
	}
	if s.Retry, err = w.redis.ZCard(ctx, w.retryKey()); err != nil {
		return s, err
	}
	s.Dead, err = w.redis.LLen(ctx, w.deadKey())
	return s, err
}
