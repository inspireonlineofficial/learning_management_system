package payments

import (
	"time"

	domainpayments "lms-backend/internal/domain/payments"
	"lms-backend/pkg/pagination"

	"github.com/google/uuid"
)

// CheckoutIntentResponse is returned after creating a payment intent.
// bkash_url is included here (only at creation time) for the frontend to redirect to bKash.
// Requirements: 24.1, 4.1, 4.2
type CheckoutIntentResponse struct {
	PaymentIntentID uuid.UUID `json:"payment_intent_id"`
	Amount          float64   `json:"amount"`
	Currency        string    `json:"currency"`
	BkashURL        string    `json:"bkash_url"`
}

// ExecutePaymentResponse is returned by the BkashCallback handler.
// Either enrollment_id or order_id will be set depending on item_type.
// Requirements: 6.1, 7.2
type ExecutePaymentResponse struct {
	Status       string     `json:"status"`
	PaymentID    uuid.UUID  `json:"payment_id"`
	EnrollmentID *uuid.UUID `json:"enrollment_id,omitempty"`
	OrderID      *uuid.UUID `json:"order_id,omitempty"`
}

// PaymentHistoryItem represents a single payment in the student's history.
// Requirements: 24.5
type PaymentHistoryItem struct {
	ID         uuid.UUID                    `json:"id"`
	ItemType   domainpayments.ItemType      `json:"item_type" swaggertype:"string" enums:"course,book"`
	ItemTitle  string                       `json:"item_title"`
	Amount     float64                      `json:"amount"`
	Currency   string                       `json:"currency"`
	Status     domainpayments.PaymentStatus `json:"status" swaggertype:"string" enums:"success,failed,refunded"`
	ReceiptURL *string                      `json:"receipt_url"`
	PaidAt     *time.Time                   `json:"paid_at"`
}

// PaymentHistoryResponse wraps a paginated list of payment history items.
type PaymentHistoryResponse struct {
	Data []*PaymentHistoryItem `json:"data"`
	Meta pagination.Meta       `json:"meta"`
}

// AdminPaymentItem represents a payment record in the admin view.
// Requirements: 24.6
type AdminPaymentItem struct {
	ID              uuid.UUID                    `json:"id"`
	StudentID       uuid.UUID                    `json:"student_id"`
	PaymentIntentID uuid.UUID                    `json:"payment_intent_id"`
	ItemType        domainpayments.ItemType      `json:"item_type" swaggertype:"string" enums:"course,book"`
	Amount          float64                      `json:"amount"`
	Currency        string                       `json:"currency"`
	Status          domainpayments.PaymentStatus `json:"status" swaggertype:"string" enums:"success,failed,refunded"`
	ReceiptURL      *string                      `json:"receipt_url"`
	PaidAt          *time.Time                   `json:"paid_at"`
	CreatedAt       time.Time                    `json:"created_at"`
}

// AdminPaymentListResponse wraps a paginated list of admin payment records.
type AdminPaymentListResponse struct {
	Data []*AdminPaymentItem `json:"data"`
	Meta pagination.Meta     `json:"meta"`
}

// PurchaseRequestItem represents a request in approval queues and CSV exports.
type PurchaseRequestItem struct {
	ID                 uuid.UUID                              `json:"id"`
	StudentID          uuid.UUID                              `json:"student_id"`
	StudentName        string                                 `json:"student_name"`
	StudentEmail       string                                 `json:"student_email"`
	ItemType           domainpayments.PurchaseRequestItemType `json:"item_type" swaggertype:"string" enums:"course,book"`
	ItemID             uuid.UUID                              `json:"item_id"`
	ItemTitle          string                                 `json:"item_title"`
	ItemSubtitle       string                                 `json:"item_subtitle,omitempty"`
	FileName           string                                 `json:"file_name"`
	Status             domainpayments.PurchaseRequestStatus   `json:"status" swaggertype:"string" enums:"pending,approved,rejected"`
	RejectionReason    *string                                `json:"rejection_reason,omitempty"`
	ResultEnrollmentID *uuid.UUID                             `json:"result_enrollment_id,omitempty"`
	ResultOrderID      *uuid.UUID                             `json:"result_order_id,omitempty"`
	ReviewedBy         *uuid.UUID                             `json:"reviewed_by,omitempty"`
	ReviewedAt         *time.Time                             `json:"reviewed_at,omitempty"`
	CreatedAt          time.Time                              `json:"created_at"`
	UpdatedAt          time.Time                              `json:"updated_at"`
}

// PurchaseRequestListResponse wraps paginated approval requests.
type PurchaseRequestListResponse struct {
	Data []*PurchaseRequestItem `json:"data"`
	Meta pagination.Meta        `json:"meta"`
}

// PurchaseRequestExportRow is written to CSV for admin downloads.
type PurchaseRequestExportRow struct {
	RequestID          string
	StudentID          string
	StudentName        string
	StudentEmail       string
	ItemType           string
	ItemID             string
	ItemTitle          string
	ItemSubtitle       string
	FileName           string
	Status             string
	RejectionReason    string
	ResultEnrollmentID string
	ResultOrderID      string
	ReviewedBy         string
	ReviewedAt         string
	CreatedAt          string
	UpdatedAt          string
}
