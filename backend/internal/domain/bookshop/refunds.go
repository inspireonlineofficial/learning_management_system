package bookshop

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// RefundRequestStatus is the state of a student's refund request.
type RefundRequestStatus string

const (
	RefundRequestPending  RefundRequestStatus = "pending"
	RefundRequestApproved RefundRequestStatus = "approved"
	RefundRequestRejected RefundRequestStatus = "rejected"
)

// PayoutMethod is how refunded money was returned to the student.
type PayoutMethod string

const (
	PayoutBKash  PayoutMethod = "bkash"
	PayoutNagad  PayoutMethod = "nagad"
	PayoutRocket PayoutMethod = "rocket"
	PayoutBank   PayoutMethod = "bank"
	PayoutCash   PayoutMethod = "cash"
	PayoutOther  PayoutMethod = "other"
	PayoutNone   PayoutMethod = "none" // free orders: nothing to return
)

// ValidPayoutMethod reports whether m is a known payout method.
func ValidPayoutMethod(m PayoutMethod) bool {
	switch m {
	case PayoutBKash, PayoutNagad, PayoutRocket, PayoutBank, PayoutCash, PayoutOther, PayoutNone:
		return true
	}
	return false
}

// RefundRequest is a student's request to refund one order.
type RefundRequest struct {
	ID         uuid.UUID
	OrderID    uuid.UUID
	StudentID  uuid.UUID
	Reason     string
	Status     RefundRequestStatus
	AdminNote  *string
	ReviewedBy *uuid.UUID
	ReviewedAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time

	// Joined for display.
	BookTitle    string
	StudentName  string
	StudentEmail string
	Amount       float64
	Currency     string
	Format       OrderFormat
	OrderStatus  OrderStatus
}

// OrderRefund records a refund that was paid out.
type OrderRefund struct {
	ID              uuid.UUID
	OrderID         uuid.UUID
	RefundRequestID *uuid.UUID
	Amount          float64
	Currency        string
	PayoutMethod    PayoutMethod
	PayoutReference *string
	Note            *string
	ProcessedBy     *uuid.UUID
	CreatedAt       time.Time
}

// ErrRefundRequestExists is returned when an order already has an open request.
var ErrRefundRequestExists = errors.New("a refund request for this order is already open")

// RefundRepository stores refund requests and payout records. Methods join
// the transaction carried by ctx.
type RefundRepository interface {
	CreateRequest(ctx context.Context, r *RefundRequest) error
	FindRequestByID(ctx context.Context, id uuid.UUID) (*RefundRequest, error)
	// LockRequest reads a request FOR UPDATE inside the caller's transaction.
	LockRequest(ctx context.Context, id uuid.UUID) (*RefundRequest, error)
	FindPendingRequestByOrder(ctx context.Context, orderID uuid.UUID) (*RefundRequest, error)
	LatestRequestByOrder(ctx context.Context, orderID uuid.UUID) (*RefundRequest, error)
	UpdateRequest(ctx context.Context, r *RefundRequest) error
	CountRequestsForOrder(ctx context.Context, orderID uuid.UUID) (int, error)
	ListRequests(ctx context.Context, status RefundRequestStatus, studentID *uuid.UUID, page, limit int) ([]*RefundRequest, int, error)
	CreateRefund(ctx context.Context, r *OrderRefund) error
	FindRefundByOrder(ctx context.Context, orderID uuid.UUID) (*OrderRefund, error)
}
