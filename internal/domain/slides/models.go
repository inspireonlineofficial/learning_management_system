package slides

import (
	"time"

	"github.com/google/uuid"
)

type PromotionalSlide struct {
	ID            uuid.UUID
	Title         string
	Subtitle      string
	LinkURL       string
	MediaKey      string
	MediaType     string
	DurationMS    int
	Position      int
	IsActive      bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeactivatedAt *time.Time
}
