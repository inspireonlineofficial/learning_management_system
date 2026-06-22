package enrollments

import (
	"time"

	"github.com/google/uuid"
)

// CourseSummaryResponse represents course details inside an enrollment response
type CourseSummaryResponse struct {
	ID       uuid.UUID                `json:"id"`
	Title    string                   `json:"title"`
	CoverURL string                   `json:"cover_url,omitempty"`
	Category *CategorySummaryResponse `json:"category,omitempty"`
}

// CategorySummaryResponse represents category/subject of a course
type CategorySummaryResponse struct {
	ID   uuid.UUID `json:"id,omitempty"`
	Name string    `json:"name"`
}

// EnrollmentResponse represents an enrollment in API responses
type EnrollmentResponse struct {
	ID              uuid.UUID              `json:"id"`
	StudentID       uuid.UUID              `json:"student_id"`
	CourseID        uuid.UUID              `json:"course_id"`
	EnrollmentType  string                 `json:"enrollment_type"`
	Status          string                 `json:"status"`
	ProgressPercent float64                `json:"progress_percent"`
	CompletedAt     *time.Time             `json:"completed_at,omitempty"`
	EnrolledAt      time.Time              `json:"enrolled_at"`
	Course          *CourseSummaryResponse `json:"course,omitempty"`
}

// LessonProgressResponse represents lesson progress in API responses
type LessonProgressResponse struct {
	ID              uuid.UUID  `json:"id"`
	EnrollmentID    uuid.UUID  `json:"enrollment_id"`
	LessonID        uuid.UUID  `json:"lesson_id"`
	PositionSeconds int        `json:"position_seconds"`
	WatchedPercent  float64    `json:"watched_percent"`
	Completed       bool       `json:"completed"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	LastWatchedAt   *time.Time `json:"last_watched_at,omitempty"`
}

// StreamingSignedURLResponse represents a presigned URL for video streaming.
// When the video has been transcoded to HLS, HLSURL is set and the player
// should prefer it over SignedURL for adaptive bitrate. SignedURL always
// points at the original progressive MP4 so untranscoded videos still play.
type StreamingSignedURLResponse struct {
	SignedURL   string    `json:"signed_url"`
	HLSURL      string    `json:"hls_url,omitempty"`
	ExpiresAt   time.Time `json:"expires_at"`
	PosterURL   string    `json:"poster_url,omitempty"`
	ContentType string    `json:"content_type,omitempty"`
	HasHLS      bool      `json:"has_hls"`
}
