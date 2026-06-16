package bookshop

import (
	"context"

	"github.com/google/uuid"
)

// BookRepository defines the persistence port for books.
// Requirements: 19.1, 19.6, 19.7
type BookRepository interface {
	// Create inserts a new book record.
	Create(ctx context.Context, book *Book) error

	// FindByID returns a book by its ID, or nil if not found.
	FindByID(ctx context.Context, id uuid.UUID) (*Book, error)

	// Update persists changes to an existing book.
	Update(ctx context.Context, book *Book) error

	// List returns a paginated list of active books with optional filters.
	// Only active books (is_active = true) are returned for public endpoints.
	List(ctx context.Context, filter BookFilter, page, limit int) ([]*Book, int, error)
}

// BookFilter holds optional filter parameters for listing books.
type BookFilter struct {
	Search     string
	Subject    string
	ClassGrade string
	Format     BookFormat
	MinPrice   *float64
	MaxPrice   *float64
	ActiveOnly bool
}

// OrderRepository defines the persistence port for orders.
// Requirements: 20.1, 20.2, 20.3, 20.4
type OrderRepository interface {
	// Create inserts a new order record within the provided transaction context.
	Create(ctx context.Context, order *Order) error

	// FindByID returns an order by its ID, or nil if not found.
	FindByID(ctx context.Context, id uuid.UUID) (*Order, error)

	// FindByIdempotencyKey returns an order matching the given idempotency key, or nil.
	FindByIdempotencyKey(ctx context.Context, key string) (*Order, error)

	// Update persists changes to an existing order.
	Update(ctx context.Context, order *Order) error

	// FindByStudentID returns all orders for a student (paginated).
	FindByStudentID(ctx context.Context, studentID uuid.UUID, page, limit int) ([]*Order, int, error)

	// FindNonRefundedByStudentAndBook returns the most recent non-refunded order
	// for a student+book pair, used to verify digital access.
	FindNonRefundedByStudentAndBook(ctx context.Context, studentID, bookID uuid.UUID) (*Order, error)

	// DecrementPhysicalStock decrements physical_stock by 1 atomically.
	// Returns an error if stock is already 0.
	DecrementPhysicalStock(ctx context.Context, bookID uuid.UUID) error

	// IncrementPhysicalStock increments physical_stock by 1 (used on refund).
	IncrementPhysicalStock(ctx context.Context, bookID uuid.UUID) error
}

// BookBookmarkRepository defines the persistence port for reading bookmarks.
// Requirements: 19.5
type BookBookmarkRepository interface {
	// Upsert creates or updates the bookmark for a student+book pair.
	Upsert(ctx context.Context, bookmark *BookBookmark) error

	// FindByStudentAndBook returns the bookmark for a student+book pair, or nil.
	FindByStudentAndBook(ctx context.Context, studentID, bookID uuid.UUID) (*BookBookmark, error)
}
