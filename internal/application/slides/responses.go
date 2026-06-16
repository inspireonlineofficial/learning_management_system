package slides

import (
	"time"

	domainslides "lms-backend/internal/domain/slides"

	"github.com/google/uuid"
)

type SlideResponse struct {
	ID         uuid.UUID `json:"id"`
	Title      string    `json:"title"`
	Subtitle   string    `json:"subtitle"`
	LinkURL    string    `json:"link_url"`
	MediaURL   string    `json:"media_url"`
	MediaType  string    `json:"media_type"`
	DurationMS int       `json:"duration_ms"`
	Position   int       `json:"position"`
	IsActive   bool      `json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type SlideListResponse struct {
	Slides []SlideResponse `json:"slides"`
}

func toResponse(slide *domainslides.PromotionalSlide, mediaURL string) SlideResponse {
	return SlideResponse{
		ID:         slide.ID,
		Title:      slide.Title,
		Subtitle:   slide.Subtitle,
		LinkURL:    slide.LinkURL,
		MediaURL:   mediaURL,
		MediaType:  slide.MediaType,
		DurationMS: slide.DurationMS,
		Position:   slide.Position,
		IsActive:   slide.IsActive,
		CreatedAt:  slide.CreatedAt,
		UpdatedAt:  slide.UpdatedAt,
	}
}
