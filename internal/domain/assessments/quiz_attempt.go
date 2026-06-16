package assessments

import (
	"time"

	"github.com/google/uuid"
)

// QuizAttemptStatus represents the state of a quiz attempt
type QuizAttemptStatus string

const (
	QuizAttemptStatusInProgress    QuizAttemptStatus = "in_progress"
	QuizAttemptStatusSubmitted     QuizAttemptStatus = "submitted"
	QuizAttemptStatusAutoSubmitted QuizAttemptStatus = "auto_submitted"
)

// QuizAttempt represents a student's attempt at a quiz
type QuizAttempt struct {
	ID               uuid.UUID
	QuizID           uuid.UUID
	StudentID        uuid.UUID
	StartedAt        time.Time
	SubmittedAt      *time.Time
	ScorePercent     *float64
	Passed           *bool
	TimeTakenSeconds *int
	PointsAwarded    int
	Status           QuizAttemptStatus
}

// IsInProgress returns true if the attempt is still in progress
func (qa *QuizAttempt) IsInProgress() bool {
	return qa.Status == QuizAttemptStatusInProgress
}

// IsSubmitted returns true if the attempt has been submitted
func (qa *QuizAttempt) IsSubmitted() bool {
	return qa.Status == QuizAttemptStatusSubmitted || qa.Status == QuizAttemptStatusAutoSubmitted
}

// HasExpired checks if the attempt has exceeded the time limit
func (qa *QuizAttempt) HasExpired(timeLimitSeconds int) bool {
	if timeLimitSeconds == 0 {
		return false
	}
	elapsed := time.Since(qa.StartedAt).Seconds()
	return elapsed > float64(timeLimitSeconds)
}

// CalculateTimeTaken returns the time taken in seconds
func (qa *QuizAttempt) CalculateTimeTaken() int {
	if qa.SubmittedAt == nil {
		return 0
	}
	return int(qa.SubmittedAt.Sub(qa.StartedAt).Seconds())
}
