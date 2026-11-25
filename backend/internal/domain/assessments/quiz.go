package assessments

import (
	"time"

	"github.com/google/uuid"
)

// Quiz is the aggregate root for quiz assessments
type Quiz struct {
	ID                         uuid.UUID
	CourseID                   uuid.UUID
	LessonID                   *uuid.UUID // nullable - can be module-level
	Title                      string
	TimeLimitSeconds           int     // 0 = no limit
	MaxAttempts                int     // 0 = unlimited
	PassingScorePercent        float64 // default 60
	ShuffleQuestions           bool
	ShowAnswersAfterSubmission bool
	IsFree                     bool
	IsPublished                bool
	Description                string
	Instructions               string
	Mode                       QuizMode
	ShuffleOptions             bool
	PartialCredit              bool       // multiple-select questions earn proportional marks
	AvailableFrom              *time.Time // nil = open immediately once published
	AvailableUntil             *time.Time // nil = no closing date
	CalculatorPolicy           string     // none | basic | scientific | graphing
	ReferenceMaterial          string     // permitted resources, shown to students
	CreatedAt                  time.Time
	UpdatedAt                  time.Time
}

// QuizMode selects the student experience. Practice mode shows hints and
// allows per-question answer checks; exam mode never reveals answers before
// submission.
type QuizMode string

const (
	QuizModeExam     QuizMode = "exam"
	QuizModePractice QuizMode = "practice"
)

// IsPractice reports whether hints and answer checks are allowed.
func (q *Quiz) IsPractice() bool { return q.Mode == QuizModePractice }

// AvailabilityAt reports whether a new attempt may start at t.
func (q *Quiz) AvailabilityAt(t time.Time) string {
	if q.AvailableFrom != nil && t.Before(*q.AvailableFrom) {
		return "not_open"
	}
	if q.AvailableUntil != nil && !t.Before(*q.AvailableUntil) {
		return "closed"
	}
	return "open"
}

// HasTimeLimit returns true if the quiz has a time limit
func (q *Quiz) HasTimeLimit() bool {
	return q.TimeLimitSeconds > 0
}

// HasAttemptLimit returns true if the quiz has a maximum attempt limit
func (q *Quiz) HasAttemptLimit() bool {
	return q.MaxAttempts > 0
}

// CanAttempt checks if a student can start a new attempt
func (q *Quiz) CanAttempt(attemptsUsed int) bool {
	if !q.HasAttemptLimit() {
		return true
	}
	return attemptsUsed < q.MaxAttempts
}

// IsPassing checks if a score meets the passing threshold
func (q *Quiz) IsPassing(scorePercent float64) bool {
	return scorePercent >= q.PassingScorePercent
}
