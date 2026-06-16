package users

import (
	"time"
)

// SubmitStudentProfileCommand represents the command to submit a student profile
type SubmitStudentProfileCommand struct {
	SchoolName      string    `json:"school_name"`
	ClassGrade      string    `json:"class_grade"`
	RollNumber      string    `json:"roll_number"`
	DateOfBirth     time.Time `json:"date_of_birth"`
	Gender          *string   `json:"gender,omitempty"`
	GuardianName    *string   `json:"guardian_name,omitempty"`
	GuardianContact *string   `json:"guardian_contact,omitempty"`
}

// ListUsersFilters represents filters for listing users
type ListUsersFilters struct {
	Role     *string    `json:"role,omitempty"`
	Status   *string    `json:"status,omitempty"`
	Search   *string    `json:"search,omitempty"`
	FromDate *time.Time `json:"from_date,omitempty"`
	ToDate   *time.Time `json:"to_date,omitempty"`
	Page     int        `json:"page"`
	Limit    int        `json:"limit"`
}

// CreateUserCommand represents the command to create a user
type CreateUserCommand struct {
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

// UpdateUserCommand represents the command to update a user
type UpdateUserCommand struct {
	Role   *string `json:"role,omitempty"`
	Status *string `json:"status,omitempty"`
}

// UpdateStudentProfileCommand represents the command to update a student profile by admin
type UpdateStudentProfileCommand struct {
	SchoolName      *string    `json:"school_name,omitempty"`
	ClassGrade      *string    `json:"class_grade,omitempty"`
	RollNumber      *string    `json:"roll_number,omitempty"`
	DateOfBirth     *time.Time `json:"date_of_birth,omitempty"`
	Gender          *string    `json:"gender,omitempty"`
	GuardianName    *string    `json:"guardian_name,omitempty"`
	GuardianContact *string    `json:"guardian_contact,omitempty"`
}
