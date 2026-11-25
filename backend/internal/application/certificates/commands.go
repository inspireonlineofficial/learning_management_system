package certificates

import "github.com/google/uuid"

// AutoGenerateCertificateCommand is issued when a student reaches 100% course completion.
// Requirements: 18.1
type AutoGenerateCertificateCommand struct {
	StudentID      uuid.UUID
	CourseID       uuid.UUID
	StudentName    string
	CourseTitle    string
	InstructorName string
}

// GetStudentCertificateCommand retrieves a student's certificate for a course.
// Requirements: 18.3
type GetStudentCertificateCommand struct {
	StudentID uuid.UUID
	CourseID  uuid.UUID
}

// VerifyCertificateCommand is used by the public verification endpoint.
// Requirements: 18.4, 18.6
type VerifyCertificateCommand struct {
	VerificationID string
}

// GenerateCertificatePDFCommand is the background job payload for PDF generation.
// Requirements: 18.5
type GenerateCertificatePDFCommand struct {
	CertificateID uuid.UUID
}
