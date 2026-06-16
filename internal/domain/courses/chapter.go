package courses

import (
	"time"

	"github.com/google/uuid"
)

// Chapter represents a grouping of lessons within a module
type Chapter struct {
	ID        uuid.UUID
	ModuleID  uuid.UUID
	Title     string
	Position  int
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}
