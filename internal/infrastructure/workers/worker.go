package workers

import (
	"context"
	"encoding/json"
	"fmt"
	"lms-backend/internal/domain/notifications"
	"lms-backend/internal/infrastructure/redis"
	"lms-backend/pkg/logger"
	"time"
)

// JobHandler processes a specific job type
type JobHandler interface {
	Handle(ctx context.Context, job notifications.Job) error
}

// Worker consumes jobs from Redis queue
type Worker struct {
	redis   *redis.Client
	queue   string
	router  map[string]JobHandler
	timeout time.Duration
}

// NewWorker creates a new background worker
func NewWorker(redisClient *redis.Client, queue string) *Worker {
	return &Worker{
		redis:   redisClient,
		queue:   queue,
		router:  make(map[string]JobHandler),
		timeout: 5 * time.Second,
	}
}

// RegisterHandler registers a job handler for a specific job type
func (w *Worker) RegisterHandler(jobType string, handler JobHandler) {
	w.router[jobType] = handler
}

// Run starts the worker loop
func (w *Worker) Run(ctx context.Context) {
	logger.Info(ctx, "Worker started", "queue", w.queue)

	for {
		select {
		case <-ctx.Done():
			logger.Info(ctx, "Worker shutting down", "queue", w.queue)
			return
		default:
			job, err := w.dequeue(ctx)
			if err != nil {
				// Log error but continue
				logger.Error(ctx, "Failed to dequeue job", "error", err)
				time.Sleep(1 * time.Second)
				continue
			}

			if job == nil {
				// No job available, sleep to avoid tight CPU loop
				time.Sleep(1 * time.Second)
				continue
			}

			// Process job
			if err := w.processJob(ctx, job); err != nil {
				logger.Error(ctx, "Failed to process job",
					"job_type", job.Type,
					"error", err,
				)
			}
		}
	}
}

func (w *Worker) dequeue(ctx context.Context) (*notifications.Job, error) {
	// Use LPOP to atomically fetch and remove the next job from the list
	result, err := w.redis.LPop(ctx, fmt.Sprintf("queue:%s", w.queue))
	if err != nil {
		// No job available or connection issue
		return nil, nil
	}

	var job notifications.Job
	if err := json.Unmarshal([]byte(result), &job); err != nil {
		return nil, fmt.Errorf("failed to unmarshal job: %w", err)
	}

	return &job, nil
}

func (w *Worker) processJob(ctx context.Context, job *notifications.Job) error {
	handler, ok := w.router[job.Type]
	if !ok {
		return fmt.Errorf("no handler registered for job type: %s", job.Type)
	}

	logger.Info(ctx, "Processing job", "job_type", job.Type)

	if err := handler.Handle(ctx, *job); err != nil {
		return fmt.Errorf("handler failed: %w", err)
	}

	logger.Info(ctx, "Job processed successfully", "job_type", job.Type)
	return nil
}
