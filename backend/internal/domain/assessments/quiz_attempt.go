package assessments

import (
	"encoding/json"
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
	DraftAnswers     json.RawMessage
	DraftRevision    int64
	Answers          json.RawMessage // final answers, set on submission
	Results          json.RawMessage // per-question snapshot ([]QuestionOutcome), set on submission
	ScorePoints      *float64
	MaxPoints        *float64
	GradingStatus    GradingStatus
	GradedAt         *time.Time
	GradedBy         *uuid.UUID
	ShuffleSeed      int64
}

// GradingStatus tracks whether all marks of a submitted attempt are final.
type GradingStatus string

const (
	GradingStatusAuto    GradingStatus = "auto"    // fully auto-graded
	GradingStatusPending GradingStatus = "pending" // waiting for manual marks
	GradingStatusGraded  GradingStatus = "graded"  // manual marks complete
)

// Deadline returns when the attempt must be submitted, or nil if untimed and
// the quiz has no closing date. The closing date caps a timed attempt.
func (qa *QuizAttempt) Deadline(quiz *Quiz) *time.Time {
	var deadline *time.Time
	if quiz.TimeLimitSeconds > 0 {
		d := qa.StartedAt.Add(time.Duration(quiz.TimeLimitSeconds) * time.Second)
		deadline = &d
	}
	if quiz.AvailableUntil != nil && (deadline == nil || quiz.AvailableUntil.Before(*deadline)) {
		d := *quiz.AvailableUntil
		deadline = &d
	}
	return deadline
}

// ExpiredAt reports whether the attempt's deadline has passed at t.
func (qa *QuizAttempt) ExpiredAt(quiz *Quiz, t time.Time) bool {
	d := qa.Deadline(quiz)
	return d != nil && t.After(*d)
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
