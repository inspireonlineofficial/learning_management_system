package payments

import (
	"lms-backend/internal/domain/payments"

	"github.com/google/uuid"
)

// CreateCheckoutIntentCommand initiates a legacy checkout intent for a course or book.
// Requirements: 24.1
type CreateCheckoutIntentCommand struct {
	StudentID uuid.UUID
	ItemType  payments.ItemType
	ItemID    uuid.UUID
}

// ExecutePaymentCommand records a legacy payment execution result.
// Requirements: 6.1
type ExecutePaymentCommand struct {
	PaymentID string // provider payment ID from callback query param
	Status    string // "success", "failure", or "cancel"
}

// GetPaymentHistoryCommand retrieves a student's payment history.
// Requirements: 24.5
type GetPaymentHistoryCommand struct {
	StudentID uuid.UUID
	Page      int
	Limit     int
}

// ListAllPaymentsCommand retrieves all payments with optional filters (admin).
// Requirements: 24.6
type ListAllPaymentsCommand struct {
	Filter payments.PaymentFilter
	Page   int
	Limit  int
}

// CreatePurchaseRequestCommand initiates a student purchase approval request.
type CreatePurchaseRequestCommand struct {
	StudentID      uuid.UUID
	ItemType       payments.PurchaseRequestItemType
	ItemID         uuid.UUID
	FileName       string
	IdempotencyKey string
}

// ReviewPurchaseRequestCommand captures an admin approval or rejection action.
type ReviewPurchaseRequestCommand struct {
	RequestID uuid.UUID
	ActorID   uuid.UUID
	ActorName string
	IPAddress string
	Reason    string
}

// ListPurchaseRequestsCommand retrieves approval requests with optional filters.
type ListPurchaseRequestsCommand struct {
	StudentID *uuid.UUID
	ItemType  *payments.PurchaseRequestItemType
	Status    *payments.PurchaseRequestStatus
	Page      int
	Limit     int
}
