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
	CreatedAt                  time.Time
	UpdatedAt                  time.Time
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
