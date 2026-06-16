package users

import (
	"time"

	"github.com/google/uuid"
)

// StudentProfileResult represents the result of student profile operations
type StudentProfileResult struct {
	UserID          uuid.UUID `json:"user_id"`
	SchoolName      string    `json:"school_name,omitempty"`
	ClassGrade      string    `json:"class_grade,omitempty"`
	RollNumber      string    `json:"roll_number,omitempty"`
	DateOfBirth     time.Time `json:"date_of_birth,omitempty"`
	Gender          *string   `json:"gender,omitempty"`
	GuardianName    *string   `json:"guardian_name,omitempty"`
	GuardianContact *string   `json:"guardian_contact,omitempty"`
	ProfileComplete bool      `json:"profile_complete"`
	UpdatedAt       time.Time `json:"updated_at,omitempty"`
}

// UserResult represents a user in list or detail responses
type UserResult struct {
	ID              uuid.UUID `json:"id"`
	FullName        string    `json:"full_name"`
	Email           string    `json:"email"`
	Role            string    `json:"role"`
	Status          string    `json:"status"`
	ProfileComplete bool      `json:"profile_complete"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// UserDetailResult represents detailed user information including student profile
type UserDetailResult struct {
	ID              uuid.UUID             `json:"id"`
	FullName        string                `json:"full_name"`
	Email           string                `json:"email"`
	Role            string                `json:"role"`
	Status          string                `json:"status"`
	ProfileComplete bool                  `json:"profile_complete"`
	CreatedAt       time.Time             `json:"created_at"`
	UpdatedAt       time.Time             `json:"updated_at"`
	StudentProfile  *StudentProfileResult `json:"student_profile,omitempty"`
}

// ListUsersResult represents paginated list of users
type ListUsersResult struct {
	Users []UserResult       `json:"users"`
	Meta  PaginationMetadata `json:"meta"`
}

// PaginationMetadata represents pagination metadata
type PaginationMetadata struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}
