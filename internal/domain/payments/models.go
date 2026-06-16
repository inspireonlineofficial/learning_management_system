package payments

import (
	"time"

	"github.com/google/uuid"
)

// ItemType represents the type of item being purchased.
type ItemType string

const (
	ItemTypeCourse ItemType = "course"
	ItemTypeBook   ItemType = "book"
)

// PaymentIntentStatus represents the lifecycle of a payment intent.
type PaymentIntentStatus string

const (
	PaymentIntentStatusPending   PaymentIntentStatus = "pending"
	PaymentIntentStatusConfirmed PaymentIntentStatus = "confirmed"
	PaymentIntentStatusFailed    PaymentIntentStatus = "failed"
)

// PaymentStatus represents the outcome of a payment transaction.
type PaymentStatus string

const (
	PaymentStatusSuccess  PaymentStatus = "success"
	PaymentStatusFailed   PaymentStatus = "failed"
	PaymentStatusRefunded PaymentStatus = "refunded"
)

// PaymentIntent is the aggregate root for a checkout session.
// bkash_url is returned to the frontend to redirect the user to the bKash payment page.
// Raw card data is NEVER stored here — all sensitive data is handled by the provider.
// Requirements: 24.1, 24.7, 4.1, 4.2, 4.3
type PaymentIntent struct {
	ID               uuid.UUID           `json:"id"`
	StudentID        uuid.UUID           `json:"student_id"`
	ItemType         ItemType            `json:"item_type"`
	ItemID           uuid.UUID           `json:"item_id"`
	Amount           float64             `json:"amount"`
	Currency         string              `json:"currency"`
	Status           PaymentIntentStatus `json:"status"`
	ProviderIntentID *string             `json:"-"` // never exposed in API
	BkashURL         *string             `json:"-"` // redirect URL returned at creation, never logged
	CreatedAt        time.Time           `json:"created_at"`
	UpdatedAt        time.Time           `json:"updated_at"`
}

// Payment records a completed (or failed) payment transaction.
// idempotency_key ensures duplicate confirm requests are not double-processed.
// Raw card data is NEVER stored. Requirements: 24.2, 24.3, 24.7
type Payment struct {
	ID                    uuid.UUID     `json:"id"`
	PaymentIntentID       uuid.UUID     `json:"payment_intent_id"`
	StudentID             uuid.UUID     `json:"student_id"`
	ItemType              ItemType      `json:"item_type"`
	IdempotencyKey        string        `json:"-"` // never exposed in API
	ProviderTransactionID string        `json:"-"` // never exposed in API
	Amount                float64       `json:"amount"`
	Currency              string        `json:"currency"`
	Status                PaymentStatus `json:"status"`
	ReceiptURL            *string       `json:"receipt_url"`
	PaidAt                *time.Time    `json:"paid_at"`
	CreatedAt             time.Time     `json:"created_at"`
}

// PaymentFilter holds optional filter parameters for listing payments (admin).
// Requirements: 24.6
type PaymentFilter struct {
	StudentID *uuid.UUID
	ItemType  *ItemType
	Status    *PaymentStatus
	FromDate  *time.Time
	ToDate    *time.Time
}
