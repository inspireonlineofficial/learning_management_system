package payments

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// PurchaseRequestItemType identifies the item being approved.
type PurchaseRequestItemType string

const (
	PurchaseRequestItemTypeCourse PurchaseRequestItemType = "course"
	PurchaseRequestItemTypeBook   PurchaseRequestItemType = "book"
)

// PurchaseRequestStatus represents the approval lifecycle.
type PurchaseRequestStatus string

const (
	PurchaseRequestStatusPending  PurchaseRequestStatus = "pending"
	PurchaseRequestStatusApproved PurchaseRequestStatus = "approved"
	PurchaseRequestStatusRejected PurchaseRequestStatus = "rejected"
)

// PurchaseRequest stores a student purchase that requires admin approval.
type PurchaseRequest struct {
	ID                 uuid.UUID               `json:"id"`
	StudentID          uuid.UUID               `json:"student_id"`
	ItemType           PurchaseRequestItemType `json:"item_type"`
	ItemID             uuid.UUID               `json:"item_id"`
	FileName           string                  `json:"file_name"`
	IdempotencyKey     *string                 `json:"-"`
	Status             PurchaseRequestStatus   `json:"status"`
	RejectionReason    *string                 `json:"rejection_reason,omitempty"`
	ResultEnrollmentID *uuid.UUID              `json:"result_enrollment_id,omitempty"`
	ResultOrderID      *uuid.UUID              `json:"result_order_id,omitempty"`
	ReviewedBy         *uuid.UUID              `json:"reviewed_by,omitempty"`
	ReviewedAt         *time.Time              `json:"reviewed_at,omitempty"`
	CreatedAt          time.Time               `json:"created_at"`
	UpdatedAt          time.Time               `json:"updated_at"`
}

// PurchaseRequestFilter holds optional list filters for approval requests.
type PurchaseRequestFilter struct {
	StudentID *uuid.UUID
	ItemType  *PurchaseRequestItemType
	Status    *PurchaseRequestStatus
}

// PurchaseRequestRepository defines the persistence port for approval requests.
type PurchaseRequestRepository interface {
	Create(ctx context.Context, request *PurchaseRequest) error
	FindByID(ctx context.Context, id uuid.UUID) (*PurchaseRequest, error)
	FindByIdempotencyKey(ctx context.Context, key string) (*PurchaseRequest, error)
	FindLatestByStudentAndItem(ctx context.Context, studentID, itemID uuid.UUID, itemType PurchaseRequestItemType) (*PurchaseRequest, error)
	Update(ctx context.Context, request *PurchaseRequest) error
	List(ctx context.Context, filter PurchaseRequestFilter, page, limit int) ([]*PurchaseRequest, int, error)
	ListAll(ctx context.Context, filter PurchaseRequestFilter) ([]*PurchaseRequest, error)
}
