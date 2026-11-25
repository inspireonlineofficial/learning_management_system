package points

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// PointEventRepository defines the interface for point event persistence.
// This is an append-only repository — no Update or Delete methods.
type PointEventRepository interface {
	Create(ctx context.Context, event *PointEvent) error
	FindByID(ctx context.Context, id uuid.UUID) (*PointEvent, error)
	FindByStudentID(ctx context.Context, studentID uuid.UUID, page, limit int) ([]*PointEvent, int, error)
	// ExistsForSourceOnDay checks daily dedup: returns true if a video_complete event
	// already exists for the given student, source, and UTC calendar day.
	ExistsForSourceOnDay(ctx context.Context, studentID, sourceID uuid.UUID, eventType PointEventType, day time.Time) (bool, error)
	// ExistsPassingForSource returns true if a quiz_pass event already exists for
	// the given student and source (used to prevent re-awarding on repeat passes).
	ExistsPassingForSource(ctx context.Context, studentID, sourceID uuid.UUID, eventType PointEventType) (bool, error)
	// SumByStudentID returns the total points earned by a student across all events.
	SumByStudentID(ctx context.Context, studentID uuid.UUID) (int, error)
	// SumByStudentIDSince returns the total points earned by a student since the given time.
	SumByStudentIDSince(ctx context.Context, studentID uuid.UUID, since time.Time) (int, error)
}

// PointsConfigRepository defines the interface for the singleton points configuration.
type PointsConfigRepository interface {
	Get(ctx context.Context) (*PointsConfig, error)
	Update(ctx context.Context, config *PointsConfig) error
}
