package assessments

import (
	"time"

	"github.com/google/uuid"
)

// SubmissionGrade represents a grade record for an assignment submission
// This is an append-only entity - each grade action creates a new record
type SubmissionGrade struct {
	ID                uuid.UUID
	SubmissionID      uuid.UUID
	GradedBy          uuid.UUID
	Score             float64
	Feedback          string
	RevisionRequested bool
	RevisionNotes     string
	GradedAt          time.Time
}

// IsPassingGrade checks if the score meets a passing threshold
func (sg *SubmissionGrade) IsPassingGrade(totalMarks float64, passingPercent float64) bool {
	if totalMarks == 0 {
		return false
	}
	scorePercent := (sg.Score / totalMarks) * 100
	return scorePercent >= passingPercent
}

// RequestsRevision returns true if this grade requests a revision
func (sg *SubmissionGrade) RequestsRevision() bool {
	return sg.RevisionRequested
}
