package slides

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, slide *PromotionalSlide) error
	FindByID(ctx context.Context, id uuid.UUID) (*PromotionalSlide, error)
	Update(ctx context.Context, slide *PromotionalSlide) error
	List(ctx context.Context, activeOnly bool) ([]*PromotionalSlide, error)
	Reorder(ctx context.Context, positions map[uuid.UUID]int) error
}
