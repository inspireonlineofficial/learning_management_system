package slides

import (
	"io"

	"github.com/google/uuid"
)

type CreateSlideCommand struct {
	ActorID    uuid.UUID
	Title      string
	Subtitle   string
	LinkURL    string
	DurationMS int
	Position   int
	FileName   string
	FileSize   int64
	MimeType   string
	MagicBytes []byte
	Reader     io.Reader
	IPAddress  string
}

type UpdateSlideCommand struct {
	ActorID    uuid.UUID
	SlideID    uuid.UUID
	Title      *string
	Subtitle   *string
	LinkURL    *string
	DurationMS *int
	Position   *int
	IsActive   *bool
	FileName   string
	FileSize   int64
	MimeType   string
	MagicBytes []byte
	Reader     io.Reader
	IPAddress  string
}

type ReorderSlidesCommand struct {
	ActorID   uuid.UUID
	Positions map[uuid.UUID]int
	IPAddress string
}

type DeactivateSlideCommand struct {
	ActorID   uuid.UUID
	SlideID   uuid.UUID
	IPAddress string
}
