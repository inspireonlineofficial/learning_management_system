package enrollments

import (
	"time"

	"github.com/google/uuid"
)

// EnrollmentType represents whether the enrollment was free or paid
type EnrollmentType string

const (
	EnrollmentTypeFree EnrollmentType = "free"
	EnrollmentTypePaid EnrollmentType = "paid"
)

// EnrollmentStatus represents the current state of an enrollment
type EnrollmentStatus string

const (
	EnrollmentStatusActive    EnrollmentStatus = "active"
	EnrollmentStatusCancelled EnrollmentStatus = "cancelled"
	EnrollmentStatusRefunded  EnrollmentStatus = "refunded"
)

// Enrollment is the aggregate root for the enrollments bounded context
type Enrollment struct {
	ID              uuid.UUID
	StudentID       uuid.UUID
	CourseID        uuid.UUID
	EnrollmentType  EnrollmentType
	Status          EnrollmentStatus
	ProgressPercent float64
	CompletedAt     *time.Time
	EnrolledAt      time.Time
}

// IsActive returns true if the enrollment is in active status
func (e *Enrollment) IsActive() bool {
	return e.Status == EnrollmentStatusActive
}

// IsComplete returns true if the course has been completed
func (e *Enrollment) IsComplete() bool {
	return e.CompletedAt != nil
}

// CanAccess returns true if the student can access course content
func (e *Enrollment) CanAccess() bool {
	return e.Status == EnrollmentStatusActive
}
