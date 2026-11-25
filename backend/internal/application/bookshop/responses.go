package bookshop

import (
	"time"

	"lms-backend/internal/domain/bookshop"

	"github.com/google/uuid"
)

// BookResponse is the public-facing book representation.
// digital_file_rustfs_key and preview_rustfs_key are NEVER included.
// Requirements: 19.1, 19.6
type BookResponse struct {
	ID             uuid.UUID           `json:"id"`
	Title          string              `json:"title"`
	Author         string              `json:"author"`
	Subject        string              `json:"subject"`
	ClassGrade     string              `json:"class_grade"`
	Description    string              `json:"description"`
	Format         bookshop.BookFormat `json:"format"`
	Price          float64             `json:"price"`
	Currency       string              `json:"currency"`
	PhysicalStock  int                 `json:"physical_stock"`
	CoverURL       string              `json:"cover_url,omitempty"`
	HasPreview     bool                `json:"has_preview"`
	HasDigitalFile bool                `json:"has_digital_file"`
	IsActive       bool                `json:"is_active"`
	CreatedAt      time.Time           `json:"created_at"`
	UpdatedAt      time.Time           `json:"updated_at"`
}

// BookListResponse wraps a paginated list of books.
type BookListResponse struct {
	Data []*BookResponse        `json:"data"`
	Meta map[string]interface{} `json:"meta"`
}

// LibraryBookResponse represents a purchased digital book in a student's library.
type LibraryBookResponse struct {
	*BookResponse
	OrderID        uuid.UUID `json:"order_id"`
	PurchasedAt    time.Time `json:"purchased_at"`
	LastPageRead   int       `json:"last_page_read"`
	ReadingEnabled bool      `json:"reading_enabled"`
}

// LibraryBookListResponse wraps a paginated student library list.
type LibraryBookListResponse struct {
	Data []*LibraryBookResponse `json:"data"`
	Meta map[string]interface{} `json:"meta"`
}

// BookPreviewResponse contains the presigned preview URL.
// The raw file URL is never exposed.
// Requirements: 19.2
type BookPreviewResponse struct {
	BookID     uuid.UUID `json:"book_id"`
	PreviewURL string    `json:"preview_url"` // presigned URL, 10-min TTL
	ExpiresIn  int       `json:"expires_in"`  // seconds
}

// DigitalBookAccessResponse contains the presigned full-file URL and bookmark.
// Requirements: 19.3, 19.5
type DigitalBookAccessResponse struct {
	BookID       uuid.UUID `json:"book_id"`
	AccessURL    string    `json:"access_url"`     // presigned URL for full digital file
	ExpiresIn    int       `json:"expires_in"`     // seconds
	LastPageRead int       `json:"last_page_read"` // from bookmark, 0 if no bookmark
}

// BookmarkResponse is returned after upserting a bookmark.
// Requirements: 19.5
type BookmarkResponse struct {
	BookID       uuid.UUID `json:"book_id"`
	StudentID    uuid.UUID `json:"student_id"`
	LastPageRead int       `json:"last_page_read"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// OrderResponse is the public-facing order representation.
// idempotency_key is never exposed.
// Requirements: 20.1, 20.5
type OrderResponse struct {
	ID              uuid.UUID                 `json:"id"`
	StudentID       uuid.UUID                 `json:"student_id"`
	BookID          uuid.UUID                 `json:"book_id"`
	Format          bookshop.OrderFormat      `json:"format"`
	Amount          float64                   `json:"amount"`
	Currency        string                    `json:"currency"`
	Status          bookshop.OrderStatus      `json:"status"`
	TrackingNumber  *string                   `json:"tracking_number"`
	BookTitle       string                    `json:"book_title"`
	StudentName     string                    `json:"student_name,omitempty"`
	StudentEmail    string                    `json:"student_email,omitempty"`
	ShippingAddress *bookshop.ShippingAddress `json:"shipping_address,omitempty"`
	ShippedAt       *time.Time                `json:"shipped_at,omitempty"`
	DeliveredAt     *time.Time                `json:"delivered_at,omitempty"`
	RefundInfo      *OrderRefundInfo          `json:"refund_info,omitempty"`
	CreatedAt       time.Time                 `json:"created_at"`
	UpdatedAt       time.Time                 `json:"updated_at"`
}

// StudentBookStatusResponse says whether the student owns a book's digital edition.
type StudentBookStatusResponse struct {
	BookID    uuid.UUID  `json:"book_id"`
	InLibrary bool       `json:"in_library"`
	OrderID   *uuid.UUID `json:"order_id,omitempty"`
}

// RefundRequester identifies the student who placed a refundable order.
type RefundRequester struct {
	ID       uuid.UUID `json:"id"`
	FullName string    `json:"full_name"`
	Email    string    `json:"email,omitempty"`
}

// RefundQueueItem is one refundable (or refunded) order in the admin queue.
type RefundQueueItem struct {
	ID          uuid.UUID            `json:"id"`
	OrderID     uuid.UUID            `json:"order_id"`
	Requester   RefundRequester      `json:"requester"`
	BookID      uuid.UUID            `json:"book_id"`
	BookTitle   string               `json:"book_title"`
	Format      bookshop.OrderFormat `json:"format"`
	Amount      float64              `json:"amount"`
	Currency    string               `json:"currency"`
	OrderStatus bookshop.OrderStatus `json:"order_status"`
	Refundable  bool                 `json:"refundable"`
	// Refund is the recorded payout once refunded; PendingRequestID is set
	// when the student has asked for a refund.
	Refund           *OrderRefundResponse `json:"refund,omitempty"`
	PendingRequestID *uuid.UUID           `json:"pending_request_id,omitempty"`
	CreatedAt        time.Time            `json:"created_at"`
	UpdatedAt        time.Time            `json:"updated_at"`
}

// RefundQueueResponse wraps the admin refund queue.
type RefundQueueResponse struct {
	Items []RefundQueueItem      `json:"items"`
	Meta  map[string]interface{} `json:"meta"`
}

// OrderListResponse wraps a paginated list of orders.
type OrderListResponse struct {
	Data []*OrderResponse       `json:"data"`
	Meta map[string]interface{} `json:"meta"`
}

// RefundResponse summarises the result of a refund operation.
// Requirements: 20.3
type RefundResponse struct {
	OrderID       uuid.UUID            `json:"order_id"`
	Status        string               `json:"status"`
	StockRestored bool                 `json:"stock_restored"`
	AccessRevoked bool                 `json:"access_revoked"`
	ProcessedAt   time.Time            `json:"processed_at"`
	Refund        *OrderRefundResponse `json:"refund,omitempty"`
}

// OrderRefundResponse is the recorded payout of a refunded order.
type OrderRefundResponse struct {
	Amount          float64               `json:"amount"`
	Currency        string                `json:"currency"`
	PayoutMethod    bookshop.PayoutMethod `json:"payout_method"`
	PayoutReference *string               `json:"payout_reference,omitempty"`
	Note            *string               `json:"note,omitempty"`
	RefundedAt      time.Time             `json:"refunded_at"`
}

func toOrderRefundResponse(r *bookshop.OrderRefund) *OrderRefundResponse {
	if r == nil {
		return nil
	}
	return &OrderRefundResponse{Amount: r.Amount, Currency: r.Currency, PayoutMethod: r.PayoutMethod,
		PayoutReference: r.PayoutReference, Note: r.Note, RefundedAt: r.CreatedAt}
}

// RefundRequestResponse is a refund request as students and admins see it.
type RefundRequestResponse struct {
	ID           uuid.UUID                    `json:"id"`
	OrderID      uuid.UUID                    `json:"order_id"`
	StudentID    uuid.UUID                    `json:"student_id"`
	StudentName  string                       `json:"student_name,omitempty"`
	StudentEmail string                       `json:"student_email,omitempty"`
	BookTitle    string                       `json:"book_title"`
	Format       bookshop.OrderFormat         `json:"format"`
	Amount       float64                      `json:"amount"`
	Currency     string                       `json:"currency"`
	OrderStatus  bookshop.OrderStatus         `json:"order_status"`
	Reason       string                       `json:"reason"`
	Status       bookshop.RefundRequestStatus `json:"status"`
	AdminNote    *string                      `json:"admin_note,omitempty"`
	ReviewedAt   *time.Time                   `json:"reviewed_at,omitempty"`
	CreatedAt    time.Time                    `json:"created_at"`
}

func toRefundRequestResponse(r *bookshop.RefundRequest) *RefundRequestResponse {
	return &RefundRequestResponse{
		ID: r.ID, OrderID: r.OrderID, StudentID: r.StudentID, StudentName: r.StudentName, StudentEmail: r.StudentEmail,
		BookTitle: r.BookTitle, Format: r.Format, Amount: r.Amount, Currency: r.Currency, OrderStatus: r.OrderStatus,
		Reason: r.Reason, Status: r.Status, AdminNote: r.AdminNote, ReviewedAt: r.ReviewedAt, CreatedAt: r.CreatedAt,
	}
}

// RefundRequestListResponse is a paginated list of refund requests.
type RefundRequestListResponse struct {
	Data []*RefundRequestResponse `json:"data"`
	Meta map[string]interface{}   `json:"meta"`
}

// OrderRefundInfo is the refund state shown on a student's order page.
type OrderRefundInfo struct {
	Eligible      bool                   `json:"eligible"`
	Deadline      *time.Time             `json:"deadline,omitempty"`
	BlockedReason string                 `json:"blocked_reason,omitempty"`
	Request       *RefundRequestResponse `json:"request,omitempty"`
	Refund        *OrderRefundResponse   `json:"refund,omitempty"`
}

// toBookResponse converts a domain Book to a BookResponse (no internal fields).
func toBookResponse(b *bookshop.Book) *BookResponse {
	return &BookResponse{
		ID:             b.ID,
		Title:          b.Title,
		Author:         b.Author,
		Subject:        b.Subject,
		ClassGrade:     b.ClassGrade,
		Description:    b.Description,
		Format:         b.Format,
		Price:          b.Price,
		Currency:       b.Currency,
		PhysicalStock:  b.PhysicalStock,
		CoverURL:       "",
		HasPreview:     b.PreviewRustFSKey != nil && *b.PreviewRustFSKey != "",
		HasDigitalFile: b.DigitalFileRustFSKey != nil && *b.DigitalFileRustFSKey != "",
		IsActive:       b.IsActive,
		CreatedAt:      b.CreatedAt,
		UpdatedAt:      b.UpdatedAt,
	}
}

// toOrderResponse converts a domain Order to an OrderResponse (no internal fields).
func toOrderResponse(o *bookshop.Order) *OrderResponse {
	return &OrderResponse{
		ID:              o.ID,
		StudentID:       o.StudentID,
		BookID:          o.BookID,
		Format:          o.Format,
		Amount:          o.Amount,
		Currency:        o.Currency,
		Status:          o.Status,
		TrackingNumber:  o.TrackingNumber,
		BookTitle:       o.BookTitle,
		StudentName:     o.StudentName,
		StudentEmail:    o.StudentEmail,
		ShippingAddress: o.ShippingAddress,
		ShippedAt:       o.ShippedAt,
		DeliveredAt:     o.DeliveredAt,
		CreatedAt:       o.CreatedAt,
		UpdatedAt:       o.UpdatedAt,
	}
}
