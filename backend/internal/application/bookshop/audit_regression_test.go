package bookshop

import (
	"context"
	"errors"
	"testing"

	domainbookshop "lms-backend/internal/domain/bookshop"
	"lms-backend/pkg/apperrors"

	"github.com/google/uuid"
)

func bookshopErrCode(err error) string {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		return appErr.Code
	}
	return ""
}

func addBook(d *bookshopPropDeps, price float64, format domainbookshop.BookFormat) uuid.UUID {
	id := uuid.New()
	key := "books/" + id.String() + ".pdf"
	_ = d.bookRepo.Create(context.Background(), &domainbookshop.Book{
		ID: id, Title: "Book", Price: price, Currency: "BDT", Format: format,
		IsActive: true, PhysicalStock: 5, DigitalFileRustFSKey: &key,
	})
	return id
}

// Audit: POST /v1/bookshop/orders used to create a "placed" digital order for a
// paid book with no payment, which immediately unlocked the reader.
func TestAudit_PaidDigitalOrderRequiresApproval(t *testing.T) {
	d := newBookshopPropDeps(nil)
	svc := d.service()
	bookID := addBook(d, 450, domainbookshop.BookFormatDigital)
	student := uuid.New()

	_, err := svc.PlaceOrder(context.Background(), PlaceOrderCommand{StudentID: student, BookID: bookID, Format: domainbookshop.OrderFormatDigital})
	if bookshopErrCode(err) != "PURCHASE_APPROVAL_REQUIRED" {
		t.Fatalf("expected PURCHASE_APPROVAL_REQUIRED, got %v", err)
	}
	if _, err := svc.GetDigitalBookAccess(context.Background(), GetDigitalBookAccessCommand{StudentID: student, BookID: bookID}); bookshopErrCode(err) != "NO_ACCESS" {
		t.Fatalf("reader access must stay locked, got %v", err)
	}
}

func TestAudit_FreeDigitalOrderStillAllowed(t *testing.T) {
	d := newBookshopPropDeps(nil)
	svc := d.service()
	bookID := addBook(d, 0, domainbookshop.BookFormatDigital)
	if _, err := svc.PlaceOrder(context.Background(), PlaceOrderCommand{StudentID: uuid.New(), BookID: bookID, Format: domainbookshop.OrderFormatDigital}); err != nil {
		t.Fatalf("free digital books should be orderable directly: %v", err)
	}
}

func TestAudit_PhysicalOrderDoesNotUnlockReader(t *testing.T) {
	bookID := uuid.New()
	d := newBookshopPropDeps(map[uuid.UUID]int{bookID: 5})
	svc := d.service()
	key := "books/x.pdf"
	_ = d.bookRepo.Create(context.Background(), &domainbookshop.Book{ID: bookID, Title: "B", Price: 0, Format: domainbookshop.BookFormatBoth, IsActive: true, PhysicalStock: 5, DigitalFileRustFSKey: &key})
	student := uuid.New()
	if _, err := svc.PlaceOrder(context.Background(), PlaceOrderCommand{StudentID: student, BookID: bookID, Format: domainbookshop.OrderFormatPhysical, ShippingAddress: testAddress()}); err != nil {
		t.Fatalf("physical order: %v", err)
	}
	if _, err := svc.GetDigitalBookAccess(context.Background(), GetDigitalBookAccessCommand{StudentID: student, BookID: bookID}); bookshopErrCode(err) != "NO_ACCESS" {
		t.Fatalf("a physical order must not grant digital access, got %v", err)
	}
}

func TestAudit_IdempotencyKeyIsScopedToStudent(t *testing.T) {
	d := newBookshopPropDeps(nil)
	svc := d.service()
	bookID := addBook(d, 0, domainbookshop.BookFormatDigital)
	if _, err := svc.PlaceOrder(context.Background(), PlaceOrderCommand{StudentID: uuid.New(), BookID: bookID, Format: domainbookshop.OrderFormatDigital, IdempotencyKey: "k1"}); err != nil {
		t.Fatalf("first order: %v", err)
	}
	_, err := svc.PlaceOrder(context.Background(), PlaceOrderCommand{StudentID: uuid.New(), BookID: bookID, Format: domainbookshop.OrderFormatDigital, IdempotencyKey: "k1"})
	if bookshopErrCode(err) != "IDEMPOTENCY_KEY_REUSED" {
		t.Fatalf("another student's idempotency key must not return their order, got %v", err)
	}
}
