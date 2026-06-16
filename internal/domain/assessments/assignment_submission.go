package assessments

import (
	"time"

	"github.com/google/uuid"
)

// AssignmentSubmissionStatus represents the state of a submission
type AssignmentSubmissionStatus string

const (
	AssignmentSubmissionStatusDraft             AssignmentSubmissionStatus = "draft"
	AssignmentSubmissionStatusSubmitted         AssignmentSubmissionStatus = "submitted"
	AssignmentSubmissionStatusGraded            AssignmentSubmissionStatus = "graded"
	AssignmentSubmissionStatusRevisionRequested AssignmentSubmissionStatus = "revision_requested"
)

// AssignmentSubmission represents a student's submission for an assignment
type AssignmentSubmission struct {
	ID           uuid.UUID
	AssignmentID uuid.UUID
	StudentID    uuid.UUID
	Status       AssignmentSubmissionStatus
	TextContent  string
	SubmittedAt  *time.Time
	IsLate       bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// IsDraft returns true if the submission is still a draft
func (as *AssignmentSubmission) IsDraft() bool {
	return as.Status == AssignmentSubmissionStatusDraft
}

// IsSubmitted returns true if the submission has been submitted
func (as *AssignmentSubmission) IsSubmitted() bool {
	return as.Status == AssignmentSubmissionStatusSubmitted ||
		as.Status == AssignmentSubmissionStatusGraded ||
		as.Status == AssignmentSubmissionStatusRevisionRequested
}

// IsGraded returns true if the submission has been graded
func (as *AssignmentSubmission) IsGraded() bool {
	return as.Status == AssignmentSubmissionStatusGraded
}

// NeedsRevision returns true if revision has been requested
func (as *AssignmentSubmission) NeedsRevision() bool {
	return as.Status == AssignmentSubmissionStatusRevisionRequested
}

// CanResubmit returns true if the student can resubmit
func (as *AssignmentSubmission) CanResubmit() bool {
	return as.Status == AssignmentSubmissionStatusRevisionRequested
}
