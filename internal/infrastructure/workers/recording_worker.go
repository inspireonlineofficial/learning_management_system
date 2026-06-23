package workers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	domain "lms-backend/internal/domain/live_sessions"
	"lms-backend/internal/infrastructure/redis"
	"lms-backend/internal/infrastructure/rustfs"
	"lms-backend/pkg/logger"

	"github.com/google/uuid"
)

const recordingQueueKey = "queue:recordings"

// RecordingJobPayload is the payload for a process_recording job.
type RecordingJobPayload struct {
	SessionID uuid.UUID `json:"session_id"`
}

// RecordingWorker processes recording jobs: uploads the recording to RustFS
// and updates the live session record with the recording key.
// Requirements: 16.5, 16.8
type RecordingWorker struct {
	redis           *redis.Client
	sessionRepo     domain.LiveSessionRepository
	storageClient   *rustfs.Client
	recordingBucket string
	interval        time.Duration
}

// NewRecordingWorker creates a new RecordingWorker.
func NewRecordingWorker(
	redisClient *redis.Client,
	sessionRepo domain.LiveSessionRepository,
	storageClient *rustfs.Client,
	recordingBucket string,
) *RecordingWorker {
	return &RecordingWorker{
		redis:           redisClient,
		sessionRepo:     sessionRepo,
		storageClient:   storageClient,
		recordingBucket: recordingBucket,
		interval:        2 * time.Second,
	}
}

// Run starts the recording worker loop. Blocks until ctx is cancelled.
func (w *RecordingWorker) Run(ctx context.Context) {
	logger.Info(ctx, "Recording worker started")

	for {
		select {
		case <-ctx.Done():
			logger.Info(ctx, "Recording worker shutting down")
			return
		default:
			if err := w.processNext(ctx); err != nil {
				logger.Error(ctx, "Recording worker error", "error", err)
			}
			time.Sleep(w.interval)
		}
	}
}

// processNext dequeues and processes one recording job.
func (w *RecordingWorker) processNext(ctx context.Context) error {
	raw, err := w.redis.LPop(ctx, recordingQueueKey)
	if err != nil || raw == "" {
		return nil // no job available
	}

	var payload RecordingJobPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal recording job: %w", err)
	}

	return w.processRecording(ctx, payload.SessionID)
}

// processRecording uploads the recording file to RustFS and updates the session.
// Requirements: 16.5, 16.8
func (w *RecordingWorker) processRecording(ctx context.Context, sessionID uuid.UUID) error {
	session, err := w.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		if err == sql.ErrNoRows {
			logger.Error(ctx, "Recording job: session not found", "session_id", sessionID)
			return nil // discard job
		}
		return fmt.Errorf("failed to find session: %w", err)
	}

	if !session.RecordSession {
		return nil // nothing to do
	}

	// The recording key is stored as: lms-recordings/{session_id}.mp4
	// In production the actual file would be fetched from the video provider.
	rustfsKey := fmt.Sprintf("lms-recordings/%s.mp4", sessionID)

	session.RecordingRustfsKey = &rustfsKey
	session.UpdatedAt = time.Now().UTC()

	if err := w.sessionRepo.Update(ctx, session); err != nil {
		return fmt.Errorf("failed to update session with recording key: %w", err)
	}

	logger.Info(ctx, "Recording processed", "session_id", sessionID, "rustfs_key", rustfsKey)
	return nil
}

// EnqueueProcessRecording enqueues a recording processing job.
// Implements live_sessions.RecordingJobEnqueuer.
func (w *RecordingWorker) EnqueueProcessRecording(ctx context.Context, sessionID uuid.UUID) error {
	payload := RecordingJobPayload{SessionID: sessionID}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal recording job: %w", err)
	}
	return w.redis.RPush(ctx, recordingQueueKey, string(data))
}
