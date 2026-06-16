package certificates

import (
	"time"

	"github.com/google/uuid"
)

// Certificate is the aggregate root for the certificates bounded context.
// It is auto-generated when a student reaches 100% course completion.
// Requirements: 18.1, 18.2
type Certificate struct {
	ID             uuid.UUID `json:"id"`
	StudentID      uuid.UUID `json:"student_id"`
	CourseID       uuid.UUID `json:"course_id"`
	VerificationID string    `json:"verification_id"` // VARCHAR(64) UNIQUE NOT NULL
	StudentName    string    `json:"student_name"`    // snapshot at issuance
	CourseTitle    string    `json:"course_title"`    // snapshot
	InstructorName string    `json:"instructor_name"` // snapshot
	CompletionDate time.Time `json:"completion_date"` // DATE NOT NULL
	PDFRustFSKey   *string   `json:"-"`               // nullable until async generation completes; never exposed
	CreatedAt      time.Time `json:"created_at"`
}
