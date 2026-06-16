package courses

import (
	"time"

	"github.com/google/uuid"
)

// VideoStatus represents the processing status of a video
type VideoStatus string

const (
	VideoStatusProcessing VideoStatus = "processing"
	VideoStatusReady      VideoStatus = "ready"
	VideoStatusFailed     VideoStatus = "failed"
)

// Video represents a video file stored in RustFS
type Video struct {
	ID                 uuid.UUID
	CourseID           uuid.UUID
	UploaderID         uuid.UUID
	RustFSKey          string // Never exposed in API responses
	Status             VideoStatus
	DurationSeconds    int
	ThumbnailRustFSKey string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}
