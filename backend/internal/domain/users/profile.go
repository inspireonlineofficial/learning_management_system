package users

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// StudentProfile represents the academic profile of a student
type StudentProfile struct {
	UserID          uuid.UUID `json:"user_id"`
	SchoolName      string    `json:"school_name"`
	ClassGrade      string    `json:"class_grade"`
	RollNumber      string    `json:"roll_number"`
	DateOfBirth     time.Time `json:"date_of_birth"`
	Gender          *string   `json:"gender,omitempty"` // male, female, other, prefer_not_to_say
	GuardianName    *string   `json:"guardian_name,omitempty"`
	GuardianContact *string   `json:"guardian_contact,omitempty"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// StudentProfileRepository defines the interface for student profile persistence
type StudentProfileRepository interface {
	// Upsert creates or updates a student profile and sets profile_complete atomically
	Upsert(ctx context.Context, profile *StudentProfile) error

	// FindByUserID retrieves a student profile by user ID
	FindByUserID(ctx context.Context, userID uuid.UUID) (*StudentProfile, error)

	// UpdateByAdmin updates a student profile by admin (for admin operations)
	UpdateByAdmin(ctx context.Context, profile *StudentProfile) error
}
