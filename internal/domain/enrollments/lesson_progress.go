package enrollments

import (
	"time"

	"github.com/google/uuid"
)

// LessonProgress tracks a student's progress through a specific lesson
type LessonProgress struct {
	ID              uuid.UUID
	EnrollmentID    uuid.UUID
	LessonID        uuid.UUID
	PositionSeconds int
	WatchedPercent  float64
	Completed       bool
	CompletedAt     *time.Time
	LastWatchedAt   *time.Time
}

// IsComplete returns true if the lesson has been marked as completed
func (lp *LessonProgress) IsComplete() bool {
	return lp.Completed
}

// CanMarkComplete returns true if the watched percentage is sufficient to mark as complete
func (lp *LessonProgress) CanMarkComplete() bool {
	return lp.WatchedPercent >= 80.0
}

// MarkComplete marks the lesson as completed with the current timestamp
func (lp *LessonProgress) MarkComplete() {
	if lp.CanMarkComplete() {
		lp.Completed = true
		now := time.Now()
		lp.CompletedAt = &now
	}
}
