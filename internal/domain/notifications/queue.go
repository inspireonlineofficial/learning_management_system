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
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
	Delay   time.Duration   `json:"delay"`
}
