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
	ID            uuid.UUID           `json:"id"`
	Title         string              `json:"title"`
	Author        string              `json:"author"`
	Subject       string              `json:"subject"`
	ClassGrade    string              `json:"class_grade"`
	Description   string              `json:"description"`
	Format        bookshop.BookFormat `json:"format"`
	Price         float64             `json:"price"`
	Currency      string              `json:"currency"`
	PhysicalStock int                 `json:"physical_stock"`
	CoverURL      string              `json:"cover_url,omitempty"`
	IsActive      bool                `json:"is_active"`
	CreatedAt     time.Time           `json:"created_at"`
	UpdatedAt     time.Time           `json:"updated_at"`
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
	ID             uuid.UUID            `json:"id"`
	StudentID      uuid.UUID            `json:"student_id"`
	BookID         uuid.UUID            `json:"book_id"`
	Format         bookshop.OrderFormat `json:"format"`
	Amount         float64              `json:"amount"`
	Currency       string               `json:"currency"`
	Status         bookshop.OrderStatus `json:"status"`
	TrackingNumber *string              `json:"tracking_number"`
	CreatedAt      time.Time            `json:"created_at"`
	UpdatedAt      time.Time            `json:"updated_at"`
}

// OrderListResponse wraps a paginated list of orders.
type OrderListResponse struct {
	Data []*OrderResponse       `json:"data"`
	Meta map[string]interface{} `json:"meta"`
}

// RefundResponse summarises the result of a refund operation.
// Requirements: 20.3
type RefundResponse struct {
	OrderID       uuid.UUID `json:"order_id"`
	Status        string    `json:"status"`
	StockRestored bool      `json:"stock_restored"`
	AccessRevoked bool      `json:"access_revoked"`
	ProcessedAt   time.Time `json:"processed_at"`
}

// toBookResponse converts a domain Book to a BookResponse (no internal fields).
func toBookResponse(b *bookshop.Book) *BookResponse {
	return &BookResponse{
		ID:            b.ID,
		Title:         b.Title,
		Author:        b.Author,
		Subject:       b.Subject,
		ClassGrade:    b.ClassGrade,
		Description:   b.Description,
		Format:        b.Format,
		Price:         b.Price,
		Currency:      b.Currency,
		PhysicalStock: b.PhysicalStock,
		CoverURL:      "",
		IsActive:      b.IsActive,
		CreatedAt:     b.CreatedAt,
		UpdatedAt:     b.UpdatedAt,
	}
}

// toOrderResponse converts a domain Order to an OrderResponse (no internal fields).
func toOrderResponse(o *bookshop.Order) *OrderResponse {
	return &OrderResponse{
		ID:             o.ID,
		StudentID:      o.StudentID,
		BookID:         o.BookID,
		Format:         o.Format,
		Amount:         o.Amount,
		Currency:       o.Currency,
		Status:         o.Status,
		TrackingNumber: o.TrackingNumber,
		CreatedAt:      o.CreatedAt,
		UpdatedAt:      o.UpdatedAt,
	}
}
