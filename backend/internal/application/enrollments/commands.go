package enrollments

import "github.com/google/uuid"

// EnrollFreeCommand represents a free course enrollment request
type EnrollFreeCommand struct {
	StudentID uuid.UUID
	CourseID  uuid.UUID
}

// RevokeEnrollmentCommand represents an enrollment revocation request
type RevokeEnrollmentCommand struct {
	StudentID uuid.UUID
	CourseID  uuid.UUID
	Status    string // "cancelled" or "refunded"
}

// UpdateLessonProgressCommand represents a lesson progress update request
type UpdateLessonProgressCommand struct {
	EnrollmentID    uuid.UUID
	LessonID        uuid.UUID
	PositionSeconds int
	WatchedPercent  float64
	Completed       bool
}

// GetStreamingSignedURLCommand represents a request for a streaming signed URL
type GetStreamingSignedURLCommand struct {
	UserID   uuid.UUID
	LessonID uuid.UUID
}
