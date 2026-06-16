package courses

import (
	"time"

	"github.com/google/uuid"
)

// Module represents a top-level grouping of chapters within a course
type Module struct {
	ID        uuid.UUID
	CourseID  uuid.UUID
	Title     string
	Position  int
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}
