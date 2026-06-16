package bookshop

import (
	"time"

	"github.com/google/uuid"
)

// BookFormat represents the available format types for a book.
type BookFormat string

const (
	BookFormatPhysical BookFormat = "physical"
	BookFormatDigital  BookFormat = "digital"
	BookFormatBoth     BookFormat = "both"
)

// OrderFormat represents the format chosen when placing an order.
type OrderFormat string

const (
	OrderFormatPhysical OrderFormat = "physical"
	OrderFormatDigital  OrderFormat = "digital"
)

// OrderStatus represents the lifecycle status of an order.
type OrderStatus string

const (
	OrderStatusPlaced    OrderStatus = "placed"
	OrderStatusShipped   OrderStatus = "shipped"
	OrderStatusDelivered OrderStatus = "delivered"
	OrderStatusRefunded  OrderStatus = "refunded"
	OrderStatusCancelled OrderStatus = "cancelled"
)

// Book is the aggregate root for the bookshop catalog.
// digital_file_rustfs_key is stored internally and NEVER exposed in public API responses.
// Requirements: 19.1, 19.6
type Book struct {
	ID                   uuid.UUID  `json:"id"`
	Title                string     `json:"title"`
	Author               string     `json:"author"`
	Subject              string     `json:"subject"`
	ClassGrade           string     `json:"class_grade"`
	Description          string     `json:"description"`
	Format               BookFormat `json:"format"`
	Price                float64    `json:"price"`
	Currency             string     `json:"currency"`
	PhysicalStock        int        `json:"physical_stock"`
	DigitalFileRustFSKey *string    `json:"-"` // never exposed in API
	PreviewRustFSKey     *string    `json:"-"` // never exposed in API
	CoverRustFSKey       *string    `json:"-"` // never exposed in API
	IsActive             bool       `json:"is_active"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

// Order is the aggregate root for book orders.
// Supports soft-delete via DeletedAt.
// Requirements: 20.1, 20.3, 20.4
type Order struct {
	ID             uuid.UUID   `json:"id"`
	StudentID      uuid.UUID   `json:"student_id"`
	BookID         uuid.UUID   `json:"book_id"`
	Format         OrderFormat `json:"format"`
	Amount         float64     `json:"amount"`
	Currency       string      `json:"currency"`
	Status         OrderStatus `json:"status"`
	TrackingNumber *string     `json:"tracking_number"`
	IdempotencyKey *string     `json:"-"` // never exposed in API
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
	DeletedAt      *time.Time  `json:"deleted_at,omitempty"`
}

// BookBookmark tracks a student's reading progress in a digital book.
// UNIQUE(student_id, book_id).
// Requirements: 19.5
type BookBookmark struct {
	ID           uuid.UUID `json:"id"`
	StudentID    uuid.UUID `json:"student_id"`
	BookID       uuid.UUID `json:"book_id"`
	LastPageRead int       `json:"last_page_read"`
	UpdatedAt    time.Time `json:"updated_at"`
}
