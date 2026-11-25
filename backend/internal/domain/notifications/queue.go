package notifications

import (
	"context"
	"encoding/json"
	"time"
)

// JobQueue defines the interface for job queue operations
type JobQueue interface {
	Enqueue(ctx context.Context, job Job) error
	Dequeue(ctx context.Context, timeout time.Duration) (*Job, error)
}

// Job represents a background job
type Job struct {
	// ID identifies the job across retries (set when enqueued). Notifications
	// created by the job derive their IDs from it, so a retry never
	// duplicates them, and a job ID that already completed is skipped.
	ID      string          `json:"id,omitempty"`
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
	Delay   time.Duration   `json:"delay"`
	// Attempts counts failed runs; the worker retries with backoff and moves
	// the job to a dead-letter list after the last attempt.
	Attempts  int    `json:"attempts,omitempty"`
	LastError string `json:"last_error,omitempty"`
}
