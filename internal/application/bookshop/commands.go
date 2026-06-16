package bookshop

import (
	"io"

	"lms-backend/internal/domain/bookshop"

	"github.com/google/uuid"
)

// ─── Catalog commands ─────────────────────────────────────────────────────────

// ListBooksCommand holds filter and pagination parameters for the public catalog.
// Requirements: 19.1
type ListBooksCommand struct {
	Search     string
	Subject    string
	ClassGrade string
	Format     bookshop.BookFormat
	MinPrice   *float64
	MaxPrice   *float64
	Page       int
	Limit      int
}

// GetBookPreviewCommand requests a presigned preview URL for a book.
// Requirements: 19.2
type GetBookPreviewCommand struct {
	BookID uuid.UUID
}

// ─── Digital access commands ──────────────────────────────────────────────────

// GetDigitalBookAccessCommand requests full digital access for a purchased book.
// Requirements: 19.3, 19.4
type GetDigitalBookAccessCommand struct {
	StudentID uuid.UUID
	BookID    uuid.UUID
}

// UpsertBookmarkCommand persists the student's current reading position.
// Requirements: 19.5
type UpsertBookmarkCommand struct {
	StudentID    uuid.UUID
	BookID       uuid.UUID
	LastPageRead int
}

// ─── Admin commands ───────────────────────────────────────────────────────────

// CreateBookCommand holds the data for creating a new book.
// digital_file_rustfs_key is stored internally and never exposed.
// Requirements: 19.6
type CreateBookCommand struct {
	ActorID              uuid.UUID
	Title                string
	Author               string
	Subject              string
	ClassGrade           string
	Description          string
	Format               bookshop.BookFormat
	Price                float64
	Currency             string
	PhysicalStock        int
	DigitalFileRustFSKey string // stored internally, never returned
	PreviewRustFSKey     string
}

// UpdateBookCommand holds the fields that can be updated on a book.
// Requirements: 19.7
type UpdateBookCommand struct {
	ActorID       uuid.UUID
	BookID        uuid.UUID
	Title         *string
	Author        *string
	Subject       *string
	ClassGrade    *string
	Description   *string
	Price         *float64
	PhysicalStock *int
	IsActive      *bool
}

type UploadBookCoverCommand struct {
	ActorID    uuid.UUID
	BookID     uuid.UUID
	FileName   string
	FileSize   int64
	MimeType   string
	MagicBytes []byte
	Reader     io.Reader
	IPAddress  string
}

// FulfilOrderCommand updates an order to shipped status with a tracking number.
// Requirements: 20.2
type FulfilOrderCommand struct {
	ActorID        uuid.UUID
	OrderID        uuid.UUID
	TrackingNumber string
}

// ProcessRefundCommand processes a refund for an order.
// Idempotent via IdempotencyKey.
// Requirements: 20.3, 20.4
type ProcessRefundCommand struct {
	ActorID        uuid.UUID
	ActorName      string
	OrderID        uuid.UUID
	IdempotencyKey string
	IPAddress      string
}

// ─── Order commands ───────────────────────────────────────────────────────────

// PlaceOrderCommand creates a new book order.
// Requirements: 19.8, 20.1
type PlaceOrderCommand struct {
	StudentID      uuid.UUID
	BookID         uuid.UUID
	Format         bookshop.OrderFormat
	IdempotencyKey string
}

// ListStudentOrdersCommand lists orders for a student.
// Requirements: 20.5
type ListStudentOrdersCommand struct {
	StudentID uuid.UUID
	Page      int
	Limit     int
}
