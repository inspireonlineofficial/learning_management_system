package bookshop

import (
	"bytes"
	"context"
	"testing"

	domainbookshop "lms-backend/internal/domain/bookshop"

	"github.com/google/uuid"
)

func testAddress() *domainbookshop.ShippingAddress {
	return &domainbookshop.ShippingAddress{Name: "Hasin A.", Phone: "+880 1700-000000", Line1: "House 12, Road 5", City: "Dhaka"}
}

func physicalOrder(d *bookshopPropDeps, bookID uuid.UUID, status domainbookshop.OrderStatus) *domainbookshop.Order {
	o := &domainbookshop.Order{ID: uuid.New(), StudentID: uuid.New(), BookID: bookID, Format: domainbookshop.OrderFormatPhysical,
		Amount: 0, Currency: "BDT", Status: status, ShippingAddress: testAddress()}
	d.orderRepo.orders[o.ID] = o
	return o
}

// Paid printed copies used to be orderable with no payment, taking stock.
func TestCircuit_PaidPhysicalOrderRequiresApproval(t *testing.T) {
	bookID := uuid.New()
	d := newBookshopPropDeps(map[uuid.UUID]int{bookID: 3})
	_ = d.bookRepo.Create(context.Background(), &domainbookshop.Book{ID: bookID, Title: "P", Price: 500, Currency: "BDT", Format: domainbookshop.BookFormatPhysical, IsActive: true, PhysicalStock: 3})
	_, err := d.service().PlaceOrder(context.Background(), PlaceOrderCommand{StudentID: uuid.New(), BookID: bookID, Format: domainbookshop.OrderFormatPhysical, ShippingAddress: testAddress()})
	if bookshopErrCode(err) != "PURCHASE_APPROVAL_REQUIRED" {
		t.Fatalf("expected PURCHASE_APPROVAL_REQUIRED, got %v", err)
	}
	if d.orderRepo.bookStocks[bookID] != 3 || len(d.orderRepo.orders) != 0 {
		t.Fatalf("stock or orders changed: stock=%d orders=%d", d.orderRepo.bookStocks[bookID], len(d.orderRepo.orders))
	}
}

func TestCircuit_PhysicalOrderNeedsShippingAddress(t *testing.T) {
	bookID := uuid.New()
	d := newBookshopPropDeps(map[uuid.UUID]int{bookID: 3})
	_ = d.bookRepo.Create(context.Background(), &domainbookshop.Book{ID: bookID, Title: "P", Price: 0, Currency: "BDT", Format: domainbookshop.BookFormatPhysical, IsActive: true, PhysicalStock: 3})
	svc := d.service()
	for _, addr := range []*domainbookshop.ShippingAddress{nil, {Name: "A", Phone: "abc", Line1: "x", City: "Dhaka"}, {Name: "A", Phone: "01700000000", City: "Dhaka"}} {
		_, err := svc.PlaceOrder(context.Background(), PlaceOrderCommand{StudentID: uuid.New(), BookID: bookID, Format: domainbookshop.OrderFormatPhysical, ShippingAddress: addr})
		if bookshopErrCode(err) != "INVALID_SHIPPING_ADDRESS" {
			t.Fatalf("address %+v: expected INVALID_SHIPPING_ADDRESS, got %v", addr, err)
		}
	}
	if d.orderRepo.bookStocks[bookID] != 3 {
		t.Fatalf("stock changed to %d", d.orderRepo.bookStocks[bookID])
	}
}

func TestCircuit_FreeDigitalNeedsUploadedFileAndIsIdempotent(t *testing.T) {
	d := newBookshopPropDeps(nil)
	svc := d.service()
	noFile := uuid.New()
	_ = d.bookRepo.Create(context.Background(), &domainbookshop.Book{ID: noFile, Title: "N", Format: domainbookshop.BookFormatDigital, IsActive: true, Currency: "BDT"})
	if _, err := svc.PlaceOrder(context.Background(), PlaceOrderCommand{StudentID: uuid.New(), BookID: noFile, Format: domainbookshop.OrderFormatDigital}); bookshopErrCode(err) != "FILE_NOT_AVAILABLE" {
		t.Fatalf("expected FILE_NOT_AVAILABLE, got %v", err)
	}
	bookID := addBook(d, 0, domainbookshop.BookFormatDigital)
	student := uuid.New()
	first, err := svc.PlaceOrder(context.Background(), PlaceOrderCommand{StudentID: student, BookID: bookID, Format: domainbookshop.OrderFormatDigital})
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.PlaceOrder(context.Background(), PlaceOrderCommand{StudentID: student, BookID: bookID, Format: domainbookshop.OrderFormatDigital})
	if err != nil || second.ID != first.ID {
		t.Fatalf("adding an owned free book again must return the same order: %v %v", err, second)
	}
}

func TestCircuit_FulfilmentTransitions(t *testing.T) {
	bookID := uuid.New()
	d := newBookshopPropDeps(map[uuid.UUID]int{bookID: 0})
	svc := d.service()
	ctx := context.Background()
	digital := &domainbookshop.Order{ID: uuid.New(), StudentID: uuid.New(), BookID: bookID, Format: domainbookshop.OrderFormatDigital, Status: domainbookshop.OrderStatusPlaced}
	d.orderRepo.orders[digital.ID] = digital
	if _, err := svc.FulfilOrder(ctx, FulfilOrderCommand{OrderID: digital.ID, Status: "shipped", TrackingNumber: "T"}); bookshopErrCode(err) != "INVALID_TRANSITION" {
		t.Fatalf("a digital order must not ship, got %v", err)
	}

	o := physicalOrder(d, bookID, domainbookshop.OrderStatusPlaced)
	if _, err := svc.FulfilOrder(ctx, FulfilOrderCommand{OrderID: o.ID, Status: "delivered"}); err == nil {
		t.Fatal("an unshipped order must not be delivered")
	}
	if _, err := svc.FulfilOrder(ctx, FulfilOrderCommand{OrderID: o.ID, Status: "shipped"}); err == nil {
		t.Fatal("shipping needs a tracking number")
	}
	res, err := svc.FulfilOrder(ctx, FulfilOrderCommand{OrderID: o.ID, Status: "shipped", TrackingNumber: " PATHAO-1 "})
	if err != nil || res.Status != domainbookshop.OrderStatusShipped || *res.TrackingNumber != "PATHAO-1" || res.ShippedAt == nil {
		t.Fatalf("ship: %v %+v", err, res)
	}
	if _, err := svc.FulfilOrder(ctx, FulfilOrderCommand{OrderID: o.ID, Status: "cancelled"}); err == nil {
		t.Fatal("a shipped order must not be cancelled")
	}
	res, err = svc.FulfilOrder(ctx, FulfilOrderCommand{OrderID: o.ID, Status: "delivered"})
	if err != nil || res.Status != domainbookshop.OrderStatusDelivered || res.DeliveredAt == nil {
		t.Fatalf("deliver: %v %+v", err, res)
	}

	c := physicalOrder(d, bookID, domainbookshop.OrderStatusPlaced)
	if _, err := svc.FulfilOrder(ctx, FulfilOrderCommand{OrderID: c.ID, Status: "cancelled"}); err != nil {
		t.Fatal(err)
	}
	if d.orderRepo.bookStocks[bookID] != 1 {
		t.Fatalf("cancelling an unshipped copy must restore stock once, got %d", d.orderRepo.bookStocks[bookID])
	}
	var notified []string
	for _, j := range d.jobQueue.jobs {
		notified = append(notified, j.Type)
	}
	if len(notified) != 3 {
		t.Fatalf("expected shipped, delivered and cancelled notifications, got %v", notified)
	}
}

func TestCircuit_RefundRestoresOnlyUnshippedStock(t *testing.T) {
	bookID := uuid.New()
	d := newBookshopPropDeps(map[uuid.UUID]int{bookID: 0})
	svc := d.service()
	shipped := physicalOrder(d, bookID, domainbookshop.OrderStatusShipped)
	res, err := svc.ProcessRefund(context.Background(), ProcessRefundCommand{OrderID: shipped.ID})
	if err != nil || res.StockRestored || d.orderRepo.bookStocks[bookID] != 0 {
		t.Fatalf("a shipped copy is with the student; stock must not be restored: %v %+v stock=%d", err, res, d.orderRepo.bookStocks[bookID])
	}
	placed := physicalOrder(d, bookID, domainbookshop.OrderStatusPlaced)
	res, err = svc.ProcessRefund(context.Background(), ProcessRefundCommand{OrderID: placed.ID})
	if err != nil || !res.StockRestored || d.orderRepo.bookStocks[bookID] != 1 {
		t.Fatalf("an unshipped copy returns to stock: %v %+v stock=%d", err, res, d.orderRepo.bookStocks[bookID])
	}
	last := d.jobQueue.jobs[len(d.jobQueue.jobs)-1]
	if last.Type != "order_refunded" {
		t.Fatalf("the student must be notified of the refund, last job %q", last.Type)
	}
}

// When another request changes the order between read and write, the
// refund re-reads instead of double-processing.
func TestCircuit_RefundRetriesAfterLostRace(t *testing.T) {
	bookID := uuid.New()
	d := newBookshopPropDeps(map[uuid.UUID]int{bookID: 0})
	o := physicalOrder(d, bookID, domainbookshop.OrderStatusPlaced)
	d.orderRepo.casConflicts = 1
	res, err := d.service().ProcessRefund(context.Background(), ProcessRefundCommand{OrderID: o.ID})
	if err != nil || res.Status != "refunded" || d.orderRepo.bookStocks[bookID] != 1 {
		t.Fatalf("refund after a lost race: %v %+v stock=%d", err, res, d.orderRepo.bookStocks[bookID])
	}
}

func TestCircuit_BookmarkRequiresOwnershipAndValidPage(t *testing.T) {
	d := newBookshopPropDeps(nil)
	svc := d.service()
	bookID := addBook(d, 0, domainbookshop.BookFormatDigital)
	student := uuid.New()
	if _, err := svc.UpsertBookmark(context.Background(), UpsertBookmarkCommand{StudentID: student, BookID: bookID, LastPageRead: 3}); bookshopErrCode(err) != "NO_ACCESS" {
		t.Fatalf("expected NO_ACCESS, got %v", err)
	}
	if _, err := svc.PlaceOrder(context.Background(), PlaceOrderCommand{StudentID: student, BookID: bookID, Format: domainbookshop.OrderFormatDigital}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.UpsertBookmark(context.Background(), UpsertBookmarkCommand{StudentID: student, BookID: bookID, LastPageRead: -1}); bookshopErrCode(err) != "INVALID_PAGE" {
		t.Fatalf("expected INVALID_PAGE, got %v", err)
	}
	if _, err := svc.UpsertBookmark(context.Background(), UpsertBookmarkCommand{StudentID: student, BookID: bookID, LastPageRead: 12}); err != nil {
		t.Fatal(err)
	}
}

func TestCircuit_BookValidation(t *testing.T) {
	svc := newBookshopPropDeps(nil).service()
	bad := []CreateBookCommand{
		{Title: "T", Author: "A", Format: "digital", Price: -1},
		{Title: "T", Author: "A", Format: "audio"},
		{Title: "T", Author: "A", Format: "physical", PhysicalStock: -2},
		{Title: "T", Author: "A", Format: "digital", Currency: "TAKA"},
		{Title: " ", Author: "A", Format: "digital"},
		{Title: "T", Author: "A", Format: "digital", DigitalFileRustFSKey: "videos/secret.mp4"},
	}
	for _, cmd := range bad {
		if _, err := svc.CreateBook(context.Background(), cmd); err == nil {
			t.Fatalf("expected %+v to be rejected", cmd)
		}
	}
	ok, err := svc.CreateBook(context.Background(), CreateBookCommand{Title: "T", Author: "A", Format: "both", Price: 120, Currency: "bdt", PhysicalStock: 4})
	if err != nil || ok.Currency != "BDT" {
		t.Fatalf("valid book rejected: %v", err)
	}
	neg := -5.0
	if _, err := svc.UpdateBook(context.Background(), UpdateBookCommand{BookID: ok.ID, Price: &neg}); err == nil {
		t.Fatal("negative price update must be rejected")
	}
}

func TestCircuit_UploadAcceptsOnlyPDF(t *testing.T) {
	d := newBookshopPropDeps(nil)
	svc := d.service()
	bookID := addBook(d, 0, domainbookshop.BookFormatDigital)
	png := []byte("\x89PNG\r\n\x1a\n....")
	if _, err := svc.UploadBookFile(context.Background(), UploadBookFileCommand{BookID: bookID, Kind: BookFileFull, FileSize: int64(len(png)), MagicBytes: png, Reader: bytes.NewReader(png)}); bookshopErrCode(err) != "INVALID_FILE_TYPE" {
		t.Fatalf("expected INVALID_FILE_TYPE, got %v", err)
	}
	pdf := []byte("%PDF-1.4\n...")
	res, err := svc.UploadBookFile(context.Background(), UploadBookFileCommand{BookID: bookID, Kind: BookFilePreview, FileSize: int64(len(pdf)), MagicBytes: pdf, Reader: bytes.NewReader(pdf)})
	if err != nil || !res.HasPreview {
		t.Fatalf("preview upload: %v %+v", err, res)
	}
	if _, err := svc.UploadBookFile(context.Background(), UploadBookFileCommand{BookID: bookID, Kind: BookFilePreview, FileSize: MaxBookPreviewBytes + 1, MagicBytes: pdf, Reader: bytes.NewReader(pdf)}); bookshopErrCode(err) != "INVALID_FILE_SIZE" {
		t.Fatalf("expected INVALID_FILE_SIZE, got %v", err)
	}
}

func TestCircuit_CheckoutIsAllOrNothing(t *testing.T) {
	d := newBookshopPropDeps(nil)
	svc := d.service()
	free := addBook(d, 0, domainbookshop.BookFormatDigital)
	paid := addBook(d, 200, domainbookshop.BookFormatDigital)
	student := uuid.New()
	_, err := svc.PlaceOrders(context.Background(), []PlaceOrderCommand{
		{StudentID: student, BookID: free, Format: domainbookshop.OrderFormatDigital},
		{StudentID: student, BookID: paid, Format: domainbookshop.OrderFormatDigital},
	})
	if err == nil {
		t.Fatal("checkout with a paid item must fail")
	}
	// The mock has no real transaction; the service must still not notify
	// about orders from a failed checkout.
	if len(d.jobQueue.jobs) != 0 {
		t.Fatalf("no notification may be sent for a failed checkout, got %d", len(d.jobQueue.jobs))
	}
}
