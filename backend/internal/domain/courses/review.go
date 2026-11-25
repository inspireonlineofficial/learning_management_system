package courses

import (
	"time"

	"github.com/google/uuid"
)

// CourseReview represents a student's review of a course
type CourseReview struct {
	ID        uuid.UUID
	CourseID  uuid.UUID
	StudentID uuid.UUID
	Rating    int // 1-5
	Comment   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// IsValid checks if the review has valid rating
func (r *CourseReview) IsValid() bool {
	return r.Rating >= 1 && r.Rating <= 5
}
