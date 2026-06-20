package payments

import (
	"context"

	"github.com/google/uuid"
)

// PaymentIntentRepository defines the persistence port for payment intents.
// Requirements: 24.1
type PaymentIntentRepository interface {
	// Create inserts a new payment intent record.
	Create(ctx context.Context, intent *PaymentIntent) error

	// FindByID returns a payment intent by its ID, or nil if not found.
	FindByID(ctx context.Context, id uuid.UUID) (*PaymentIntent, error)

	// Update persists status changes to an existing payment intent.
	Update(ctx context.Context, intent *PaymentIntent) error

	// FindByStudentAndItem returns an existing pending intent for the same student+item,
	// used to avoid creating duplicate intents.
	FindByStudentAndItem(ctx context.Context, studentID, itemID uuid.UUID, itemType ItemType) (*PaymentIntent, error)

	// FindByProviderIntentID returns a payment intent by its provider-assigned ID.
	// Retained for legacy payment records. Requirements: 4.3
	FindByProviderIntentID(ctx context.Context, providerIntentID string) (*PaymentIntent, error)
}

// PaymentRepository defines the persistence port for payment records.
// Requirements: 24.2, 24.3, 24.5, 24.6
type PaymentRepository interface {
	// Create inserts a new payment record.
	Create(ctx context.Context, payment *Payment) error

	// FindByIdempotencyKey returns a payment matching the given idempotency key, or nil.
	// Used to enforce idempotency on confirm requests. Requirements: 24.3
	FindByIdempotencyKey(ctx context.Context, key string) (*Payment, error)

	// FindByStudentID returns paginated payment history for a student.
	// Requirements: 24.5
	FindByStudentID(ctx context.Context, studentID uuid.UUID, page, limit int) ([]*Payment, int, error)

	// List returns all payments with optional filters (admin view).
	// Requirements: 24.6
	List(ctx context.Context, filter PaymentFilter, page, limit int) ([]*Payment, int, error)
}
