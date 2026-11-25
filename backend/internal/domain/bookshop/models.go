package bookshop

import (
	"strings"
	"time"
	"unicode/utf8"

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
	// ShippingAddress is required for physical orders and nil for digital ones.
	ShippingAddress *ShippingAddress `json:"shipping_address,omitempty"`
	ShippedAt       *time.Time       `json:"shipped_at,omitempty"`
	DeliveredAt     *time.Time       `json:"delivered_at,omitempty"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
	DeletedAt       *time.Time       `json:"deleted_at,omitempty"`

	// Read-only details joined in by the repository for display.
	BookTitle    string `json:"-"`
	StudentName  string `json:"-"`
	StudentEmail string `json:"-"`
}

// ShippingAddress is where a physical order is delivered.
type ShippingAddress struct {
	Name       string `json:"name"`
	Phone      string `json:"phone"`
	Line1      string `json:"line1"`
	Line2      string `json:"line2,omitempty"`
	City       string `json:"city"`
	PostalCode string `json:"postal_code,omitempty"`
}

// Normalize trims every field.
func (a *ShippingAddress) Normalize() {
	a.Name = strings.TrimSpace(a.Name)
	a.Phone = strings.TrimSpace(a.Phone)
	a.Line1 = strings.TrimSpace(a.Line1)
	a.Line2 = strings.TrimSpace(a.Line2)
	a.City = strings.TrimSpace(a.City)
	a.PostalCode = strings.TrimSpace(a.PostalCode)
}

// Problem returns a human-readable reason the address is unusable, or "".
func (a *ShippingAddress) Problem() string {
	if a == nil {
		return "a shipping address is required for printed books"
	}
	required := []struct{ name, value string }{{"name", a.Name}, {"phone", a.Phone}, {"address line", a.Line1}, {"city", a.City}}
	for _, f := range required {
		if f.value == "" {
			return "shipping " + f.name + " is required"
		}
	}
	for _, v := range []string{a.Name, a.Phone, a.Line1, a.Line2, a.City, a.PostalCode} {
		if utf8.RuneCountInString(v) > 200 {
			return "shipping address fields must be 200 characters or fewer"
		}
	}
	digits := 0
	for _, r := range a.Phone {
		switch {
		case r >= '0' && r <= '9':
			digits++
		case r == '+' || r == ' ' || r == '-' || r == '(' || r == ')':
		default:
			return "shipping phone may contain only digits, spaces and + - ( )"
		}
	}
	if digits < 6 || digits > 15 {
		return "shipping phone must have between 6 and 15 digits"
	}
	return ""
}

// IsPhysical reports whether the order ships a printed copy.
func (o *Order) IsPhysical() bool { return o.Format == OrderFormatPhysical }

// SupportsFormat reports whether a book can be ordered in the given format.
func (b *Book) SupportsFormat(f OrderFormat) bool {
	switch f {
	case OrderFormatDigital:
		return b.Format == BookFormatDigital || b.Format == BookFormatBoth
	case OrderFormatPhysical:
		return b.Format == BookFormatPhysical || b.Format == BookFormatBoth
	}
	return false
}

// ValidBookFormat reports whether f is a known catalogue format.
func ValidBookFormat(f BookFormat) bool {
	return f == BookFormatDigital || f == BookFormatPhysical || f == BookFormatBoth
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
