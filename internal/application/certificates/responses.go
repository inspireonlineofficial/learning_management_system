package certificates

import (
	"time"

	"github.com/google/uuid"
)

// CertificateResponse is returned to the student when they request their certificate.
// pdf_url is a presigned URL (1h TTL); pdf_rustfs_key is never exposed.
// Requirements: 18.3, 18.5
type CertificateResponse struct {
	ID             uuid.UUID `json:"id"`
	StudentID      uuid.UUID `json:"student_id"`
	CourseID       uuid.UUID `json:"course_id"`
	VerificationID string    `json:"verification_id"`
	StudentName    string    `json:"student_name"`
	CourseTitle    string    `json:"course_title"`
	InstructorName string    `json:"instructor_name"`
	CompletionDate string    `json:"completion_date"` // ISO 8601 date string
	PDFURL         *string   `json:"pdf_url"`         // presigned URL, nil if PDF not yet generated
	CreatedAt      time.Time `json:"created_at"`
}

// VerifyCertificateResponse is returned by the public verification endpoint.
// Requirements: 18.4, 18.6
type VerifyCertificateResponse struct {
	Valid          bool      `json:"valid"`
	VerificationID string    `json:"verification_id"`
	StudentName    string    `json:"student_name"`
	CourseTitle    string    `json:"course_title"`
	InstructorName string    `json:"instructor_name"`
	CompletionDate string    `json:"completion_date"` // ISO 8601 date string
	IssuedAt       time.Time `json:"issued_at"`
}
