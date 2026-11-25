package points

import (
	"time"

	"github.com/google/uuid"
)

// PointEventType represents the action that triggered a point award
type PointEventType string

const (
	PointEventTypeVideoComplete PointEventType = "video_complete"
	PointEventTypeQuizPass      PointEventType = "quiz_pass"
	PointEventTypeQuizPerfect   PointEventType = "quiz_perfect"
)

// PointEvent is an append-only entity recording a single point award to a student
type PointEvent struct {
	ID          uuid.UUID
	StudentID   uuid.UUID
	Type        PointEventType
	SourceID    uuid.UUID // lesson_id or quiz_id
	SourceTitle string
	Points      int
	BonusPoints int
	EarnedAt    time.Time
}

// TotalPoints returns the sum of base points and bonus points for this event
func (p *PointEvent) TotalPoints() int {
	return p.Points + p.BonusPoints
}
