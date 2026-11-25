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
	// HLSManifestKey is the key of the master .m3u8 produced by the
	// transcoding worker. When non-empty the player should prefer HLS over
	// progressive MP4 for adaptive bitrate.
	HLSManifestKey string
	TranscodedAt   *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
