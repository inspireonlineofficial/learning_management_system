package assessments

import (
	"time"

	"github.com/google/uuid"
)

// SubmissionType represents the type of submission allowed
type SubmissionType string

const (
	SubmissionTypeFile SubmissionType = "file"
	SubmissionTypeText SubmissionType = "text"
	SubmissionTypeBoth SubmissionType = "both"
)

// Assignment is the aggregate root for assignment assessments
type Assignment struct {
	ID                  uuid.UUID
	CourseID            uuid.UUID
	Title               string
	Description         string
	DueAt               time.Time
	SubmissionType      SubmissionType
	MaxFileSizeMB       int
	AllowLateSubmission bool
	TotalMarks          float64
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// IsPastDeadline checks if the current time is past the due date
func (a *Assignment) IsPastDeadline(now time.Time) bool {
	return now.After(a.DueAt)
}

// CanSubmitLate checks if late submissions are allowed
func (a *Assignment) CanSubmitLate() bool {
	return a.AllowLateSubmission
}

// AcceptsFileSubmissions returns true if file submissions are allowed
func (a *Assignment) AcceptsFileSubmissions() bool {
	return a.SubmissionType == SubmissionTypeFile || a.SubmissionType == SubmissionTypeBoth
}

// AcceptsTextSubmissions returns true if text submissions are allowed
func (a *Assignment) AcceptsTextSubmissions() bool {
	return a.SubmissionType == SubmissionTypeText || a.SubmissionType == SubmissionTypeBoth
}

// ValidateFileSize checks if a file size is within the allowed limit
func (a *Assignment) ValidateFileSize(sizeBytes int64) bool {
	maxBytes := int64(a.MaxFileSizeMB) * 1024 * 1024
	return sizeBytes <= maxBytes
}
