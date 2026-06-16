package bookshop

import (
	"context"
	"io"
	"testing"
	"time"

	domainbookshop "lms-backend/internal/domain/bookshop"
	"lms-backend/internal/domain/notifications"

	"github.com/google/uuid"
	"pgregory.net/rapid"
)

// ─── Mock implementations ─────────────────────────────────────────────────────

// mockBookRepo implements domainbookshop.BookRepository
type mockBookRepo struct {
	books map[uuid.UUID]*domainbookshop.Book
}

func newMockBookRepo() *mockBookRepo {
	return &mockBookRepo{books: make(map[uuid.UUID]*domainbookshop.Book)}
}

func (m *mockBookRepo) Create(ctx context.Context, book *domainbookshop.Book) error {
	m.books[book.ID] = book
	return nil
}

func (m *mockBookRepo) FindByID(ctx context.Context, id uuid.UUID) (*domainbookshop.Book, error) {
	b, ok := m.books[id]
	if !ok {
		return nil, nil
	}
	return b, nil
}

func (m *mockBookRepo) Update(ctx context.Context, book *domainbookshop.Book) error {
	m.books[book.ID] = book
	return nil
}

func (m *mockBookRepo) List(ctx context.Context, filter domainbookshop.BookFilter, page, limit int) ([]*domainbookshop.Book, int, error) {
	var result []*domainbookshop.Book
	for _, b := range m.books {
		if filter.ActiveOnly && !b.IsActive {
			continue
		}
		result = append(result, b)
	}
	return result, len(result), nil
}

// mockOrderRepo implements domainbookshop.OrderRepository
type mockOrderRepo struct {
	orders         map[uuid.UUID]*domainbookshop.Order
	idempotencyMap map[string]*domainbookshop.Order
	bookStocks     map[uuid.UUID]int // tracks stock changes
}

func newMockOrderRepo(initialStocks map[uuid.UUID]int) *mockOrderRepo {
	stocks := make(map[uuid.UUID]int)
	for k, v := range initialStocks {
		stocks[k] = v
	}
	return &mockOrderRepo{
		orders:         make(map[uuid.UUID]*domainbookshop.Order),
		idempotencyMap: make(map[string]*domainbookshop.Order),
		bookStocks:     stocks,
	}
}

func (m *mockOrderRepo) Create(ctx context.Context, order *domainbookshop.Order) error {
	m.orders[order.ID] = order
	if order.IdempotencyKey != nil {
		m.idempotencyMap[*order.IdempotencyKey] = order
	}
	return nil
}

func (m *mockOrderRepo) FindByID(ctx context.Context, id uuid.UUID) (*domainbookshop.Order, error) {
	o, ok := m.orders[id]
	if !ok {
		return nil, nil
	}
	return o, nil
}

func (m *mockOrderRepo) FindByIdempotencyKey(ctx context.Context, key string) (*domainbookshop.Order, error) {
	o, ok := m.idempotencyMap[key]
	if !ok {
		return nil, nil
	}
	return o, nil
}

func (m *mockOrderRepo) Update(ctx context.Context, order *domainbookshop.Order) error {
	m.orders[order.ID] = order
	return nil
}

func (m *mockOrderRepo) FindByStudentID(ctx context.Context, studentID uuid.UUID, page, limit int) ([]*domainbookshop.Order, int, error) {
	var result []*domainbookshop.Order
	for _, o := range m.orders {
		if o.StudentID == studentID {
			result = append(result, o)
		}
	}
	return result, len(result), nil
}

func (m *mockOrderRepo) FindNonRefundedByStudentAndBook(ctx context.Context, studentID, bookID uuid.UUID) (*domainbookshop.Order, error) {
	for _, o := range m.orders {
		if o.StudentID == studentID && o.BookID == bookID &&
			o.Status != domainbookshop.OrderStatusRefunded &&
			o.Status != domainbookshop.OrderStatusCancelled {
			return o, nil
		}
	}
	return nil, nil
}

func (m *mockOrderRepo) DecrementPhysicalStock(ctx context.Context, bookID uuid.UUID) error {
	stock, ok := m.bookStocks[bookID]
	if !ok || stock <= 0 {
		return errOutOfStock
	}
	m.bookStocks[bookID] = stock - 1
	return nil
}

func (m *mockOrderRepo) IncrementPhysicalStock(ctx context.Context, bookID uuid.UUID) error {
	m.bookStocks[bookID]++
	return nil
}

// errOutOfStock is a sentinel error for out-of-stock condition.
var errOutOfStock = &outOfStockError{}

type outOfStockError struct{}

func (e *outOfStockError) Error() string { return "out of stock" }

// mockBookmarkRepo implements domainbookshop.BookBookmarkRepository
type mockBookmarkRepo struct {
	bookmarks map[[2]uuid.UUID]*domainbookshop.BookBookmark
}

func newMockBookmarkRepo() *mockBookmarkRepo {
	return &mockBookmarkRepo{bookmarks: make(map[[2]uuid.UUID]*domainbookshop.BookBookmark)}
}

func (m *mockBookmarkRepo) Upsert(ctx context.Context, bm *domainbookshop.BookBookmark) error {
	m.bookmarks[[2]uuid.UUID{bm.StudentID, bm.BookID}] = bm
	return nil
}

func (m *mockBookmarkRepo) FindByStudentAndBook(ctx context.Context, studentID, bookID uuid.UUID) (*domainbookshop.BookBookmark, error) {
	bm, ok := m.bookmarks[[2]uuid.UUID{studentID, bookID}]
	if !ok {
		return nil, nil
	}
	return bm, nil
}

// mockJobQueue implements notifications.JobQueue
type mockJobQueue struct {
	jobs []notifications.Job
}

func (m *mockJobQueue) Enqueue(ctx context.Context, job notifications.Job) error {
	m.jobs = append(m.jobs, job)
	return nil
}

func (m *mockJobQueue) Dequeue(ctx context.Context, timeout time.Duration) (*notifications.Job, error) {
	if len(m.jobs) == 0 {
		return nil, nil
	}
	job := m.jobs[0]
	m.jobs = m.jobs[1:]
	return &job, nil
}

// mockStorageClient implements StorageClient
type mockStorageClient struct{}

func (m *mockStorageClient) PutObject(ctx context.Context, bucket, key string, r io.Reader, size int64, contentType string) error {
	return nil
}

func (m *mockStorageClient) PresignGetURL(ctx context.Context, bucket, key string, ttl time.Duration) (string, error) {
	return "https://example.com/presigned/" + key, nil
}

func (m *mockStorageClient) DeleteObject(ctx context.Context, bucket, key string) error {
	return nil
}

// mockAuditLogger implements AuditLogger
type mockAuditLogger struct {
	actions []string
}

func (m *mockAuditLogger) LogAction(ctx context.Context, actorID uuid.UUID, actorName, action, targetType string, targetID uuid.UUID, metadata map[string]interface{}, ipAddress string) error {
	m.actions = append(m.actions, action)
	return nil
}

// mockIdempotencyStore implements IdempotencyStore
type mockIdempotencyStore struct {
	cache map[string]string
}

func newMockIdempotencyStore() *mockIdempotencyStore {
	return &mockIdempotencyStore{cache: make(map[string]string)}
}

func (m *mockIdempotencyStore) Get(ctx context.Context, key string) (string, bool, error) {
	v, ok := m.cache[key]
	return v, ok, nil
}

func (m *mockIdempotencyStore) Set(ctx context.Context, key string, response string) error {
	m.cache[key] = response
	return nil
}

// ─── Helper: build a service with all mocks ──────────────────────────────────

type bookshopPropDeps struct {
	bookRepo     *mockBookRepo
	orderRepo    *mockOrderRepo
	bookmarkRepo *mockBookmarkRepo
	jobQueue     *mockJobQueue
	storage      *mockStorageClient
	audit        *mockAuditLogger
	idempotency  *mockIdempotencyStore
}

func newBookshopPropDeps(initialStocks map[uuid.UUID]int) *bookshopPropDeps {
	return &bookshopPropDeps{
		bookRepo:     newMockBookRepo(),
		orderRepo:    newMockOrderRepo(initialStocks),
		bookmarkRepo: newMockBookmarkRepo(),
		jobQueue:     &mockJobQueue{},
		storage:      &mockStorageClient{},
		audit:        &mockAuditLogger{},
		idempotency:  newMockIdempotencyStore(),
	}
}

func (d *bookshopPropDeps) service() Service {
	return NewService(
		d.bookRepo,
		d.orderRepo,
		d.bookmarkRepo,
		d.jobQueue,
		d.storage,
		d.audit,
		d.idempotency,
		"lms-books",
		nil,
	)
}

// ─── Property 52 ─────────────────────────────────────────────────────────────

// TestProperty52_DigitalBookAccessRequiresValidNonRefundedPurchase verifies that
// GetDigitalBookAccess returns NO_ACCESS (HTTP 403) when the student has no valid
// (non-refunded) purchase for the book.
//
// **Validates: Requirements 19.3**
func TestProperty52_DigitalBookAccessRequiresValidNonRefundedPurchase(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ctx := context.Background()

		// Generate a digital book
		bookID := uuid.New()
		digitalKey := "lms-books/" + bookID.String() + "/full/file.pdf"
		book := &domainbookshop.Book{
			ID:                   bookID,
			Title:                "Test Book",
			Author:               "Author",
			Format:               domainbookshop.BookFormatDigital,
			Price:                10.0,
			Currency:             "BDT",
			IsActive:             true,
			DigitalFileRustFSKey: &digitalKey,
			CreatedAt:            time.Now().UTC(),
			UpdatedAt:            time.Now().UTC(),
		}

		// Generate a student
		studentID := uuid.New()

		// Randomly choose the order scenario
		scenario := rapid.IntRange(0, 3).Draw(t, "scenario")
		// 0: no order at all
		// 1: order exists but is refunded
		// 2: order exists but is cancelled
		// 3: valid order exists (should succeed)

		deps := newBookshopPropDeps(nil)
		deps.bookRepo.books[bookID] = book
		svc := deps.service()

		switch scenario {
		case 0:
			// No order — must return NO_ACCESS
			_, err := svc.GetDigitalBookAccess(ctx, GetDigitalBookAccessCommand{
				StudentID: studentID,
				BookID:    bookID,
			})
			if err == nil {
				t.Fatal("expected NO_ACCESS error when no purchase exists, got nil")
			}
			appErr, ok := err.(interface{ Error() string })
			if !ok {
				t.Fatalf("expected AppError, got %T", err)
			}
			_ = appErr

		case 1:
			// Refunded order — must return NO_ACCESS
			refundedOrder := &domainbookshop.Order{
				ID:        uuid.New(),
				StudentID: studentID,
				BookID:    bookID,
				Format:    domainbookshop.OrderFormatDigital,
				Status:    domainbookshop.OrderStatusRefunded,
				Amount:    10.0,
				Currency:  "BDT",
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			}
			deps.orderRepo.orders[refundedOrder.ID] = refundedOrder

			_, err := svc.GetDigitalBookAccess(ctx, GetDigitalBookAccessCommand{
				StudentID: studentID,
				BookID:    bookID,
			})
			if err == nil {
				t.Fatal("expected NO_ACCESS error for refunded purchase, got nil")
			}

		case 2:
			// Cancelled order — must return NO_ACCESS
			cancelledOrder := &domainbookshop.Order{
				ID:        uuid.New(),
				StudentID: studentID,
				BookID:    bookID,
				Format:    domainbookshop.OrderFormatDigital,
				Status:    domainbookshop.OrderStatusCancelled,
				Amount:    10.0,
				Currency:  "BDT",
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			}
			deps.orderRepo.orders[cancelledOrder.ID] = cancelledOrder

			_, err := svc.GetDigitalBookAccess(ctx, GetDigitalBookAccessCommand{
				StudentID: studentID,
				BookID:    bookID,
			})
			if err == nil {
				t.Fatal("expected NO_ACCESS error for cancelled purchase, got nil")
			}

		case 3:
			// Valid order — must succeed and return a presigned URL
			validOrder := &domainbookshop.Order{
				ID:        uuid.New(),
				StudentID: studentID,
				BookID:    bookID,
				Format:    domainbookshop.OrderFormatDigital,
				Status:    domainbookshop.OrderStatusPlaced,
				Amount:    10.0,
				Currency:  "BDT",
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			}
			deps.orderRepo.orders[validOrder.ID] = validOrder

			result, err := svc.GetDigitalBookAccess(ctx, GetDigitalBookAccessCommand{
				StudentID: studentID,
				BookID:    bookID,
			})
			if err != nil {
				t.Fatalf("expected success for valid purchase, got error: %v", err)
			}
			if result == nil {
				t.Fatal("expected non-nil result for valid purchase")
			}
			if result.AccessURL == "" {
				t.Fatal("expected non-empty access URL for valid purchase")
			}
			// Property: the raw digital_file_rustfs_key must NOT appear in the response
			if result.AccessURL == digitalKey {
				t.Fatal("access URL must be a presigned URL, not the raw RustFS key")
			}
		}

		// Property: a different student without a purchase must always get NO_ACCESS
		otherStudentID := uuid.New()
		_, err := svc.GetDigitalBookAccess(ctx, GetDigitalBookAccessCommand{
			StudentID: otherStudentID,
			BookID:    bookID,
		})
		if err == nil {
			t.Fatal("student without purchase must always get NO_ACCESS")
		}
	})
}

// ─── Property 53 ─────────────────────────────────────────────────────────────

// TestProperty53_BookOrderDecrementsPhysicalStockAtomically verifies that placing
// a physical book order decrements physical_stock by exactly 1, and that orders
// are rejected when stock is 0.
//
// **Validates: Requirements 20.1**
func TestProperty53_BookOrderDecrementsPhysicalStockAtomically(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ctx := context.Background()

		// Generate initial stock between 0 and 5
		initialStock := rapid.IntRange(0, 5).Draw(t, "initial_stock")
		bookID := uuid.New()

		deps := newBookshopPropDeps(map[uuid.UUID]int{bookID: initialStock})
		svc := deps.service()

		// Create a physical book with the given stock
		book := &domainbookshop.Book{
			ID:            bookID,
			Title:         "Physical Book",
			Author:        "Author",
			Format:        domainbookshop.BookFormatPhysical,
			Price:         20.0,
			Currency:      "BDT",
			PhysicalStock: initialStock,
			IsActive:      true,
			CreatedAt:     time.Now().UTC(),
			UpdatedAt:     time.Now().UTC(),
		}
		deps.bookRepo.books[bookID] = book

		studentID := uuid.New()
		idempotencyKey := uuid.New().String()

		result, err := svc.PlaceOrder(ctx, PlaceOrderCommand{
			StudentID:      studentID,
			BookID:         bookID,
			Format:         domainbookshop.OrderFormatPhysical,
			IdempotencyKey: idempotencyKey,
		})

		if initialStock == 0 {
			// Property: order must be rejected when stock is 0
			if err == nil {
				t.Fatal("expected error when placing order with stock=0, got nil")
			}
			// Property: no order must be created
			if len(deps.orderRepo.orders) != 0 {
				t.Fatalf("expected 0 orders when stock=0, got %d", len(deps.orderRepo.orders))
			}
			// Property: stock must remain 0
			if deps.orderRepo.bookStocks[bookID] != 0 {
				t.Fatalf("stock must remain 0 after rejected order, got %d", deps.orderRepo.bookStocks[bookID])
			}
		} else {
			// Property: order must succeed when stock > 0
			if err != nil {
				t.Fatalf("expected success when stock=%d, got error: %v", initialStock, err)
			}
			if result == nil {
				t.Fatal("expected non-nil order result")
			}
			if result.Status != domainbookshop.OrderStatusPlaced {
				t.Fatalf("expected order status 'placed', got %q", result.Status)
			}

			// Property: stock must be decremented by exactly 1
			expectedStock := initialStock - 1
			actualStock := deps.orderRepo.bookStocks[bookID]
			if actualStock != expectedStock {
				t.Fatalf("expected stock to be %d after order, got %d", expectedStock, actualStock)
			}

			// Property: exactly one order must be created
			if len(deps.orderRepo.orders) != 1 {
				t.Fatalf("expected exactly 1 order, got %d", len(deps.orderRepo.orders))
			}

			// Property: idempotent re-submission must return the same order without decrementing stock again
			result2, err2 := svc.PlaceOrder(ctx, PlaceOrderCommand{
				StudentID:      studentID,
				BookID:         bookID,
				Format:         domainbookshop.OrderFormatPhysical,
				IdempotencyKey: idempotencyKey,
			})
			if err2 != nil {
				t.Fatalf("idempotent re-submission failed: %v", err2)
			}
			if result2.ID != result.ID {
				t.Fatal("idempotent re-submission must return the same order ID")
			}
			// Stock must not be decremented again
			if deps.orderRepo.bookStocks[bookID] != expectedStock {
				t.Fatalf("idempotent re-submission must not decrement stock again: expected %d, got %d",
					expectedStock, deps.orderRepo.bookStocks[bookID])
			}
		}
	})
}

// ─── Property 54 ─────────────────────────────────────────────────────────────

// TestProperty54_RefundIdempotencyDuplicateRequestsNotDoubleProcessed verifies that
// duplicate refund requests with the same idempotency key are not double-processed.
//
// **Validates: Requirements 20.4**
func TestProperty54_RefundIdempotencyDuplicateRequestsNotDoubleProcessed(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ctx := context.Background()

		// Generate a physical or digital order
		isPhysical := rapid.Bool().Draw(t, "is_physical")
		bookID := uuid.New()
		initialStock := 5

		deps := newBookshopPropDeps(map[uuid.UUID]int{bookID: initialStock})
		svc := deps.service()

		// Create the book
		format := domainbookshop.BookFormatDigital
		if isPhysical {
			format = domainbookshop.BookFormatPhysical
		}
		book := &domainbookshop.Book{
			ID:            bookID,
			Title:         "Test Book",
			Author:        "Author",
			Format:        format,
			Price:         15.0,
			Currency:      "BDT",
			PhysicalStock: initialStock,
			IsActive:      true,
			CreatedAt:     time.Now().UTC(),
			UpdatedAt:     time.Now().UTC(),
		}
		deps.bookRepo.books[bookID] = book

		// Create an existing order in 'placed' status
		orderFormat := domainbookshop.OrderFormatDigital
		if isPhysical {
			orderFormat = domainbookshop.OrderFormatPhysical
		}
		orderID := uuid.New()
		order := &domainbookshop.Order{
			ID:        orderID,
			StudentID: uuid.New(),
			BookID:    bookID,
			Format:    orderFormat,
			Amount:    15.0,
			Currency:  "BDT",
			Status:    domainbookshop.OrderStatusPlaced,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}
		deps.orderRepo.orders[orderID] = order

		// Generate a unique idempotency key
		idempotencyKey := uuid.New().String()
		actorID := uuid.New()

		// First refund request
		result1, err := svc.ProcessRefund(ctx, ProcessRefundCommand{
			ActorID:        actorID,
			ActorName:      "Admin",
			OrderID:        orderID,
			IdempotencyKey: idempotencyKey,
			IPAddress:      "127.0.0.1",
		})
		if err != nil {
			t.Fatalf("first refund request failed: %v", err)
		}
		if result1 == nil {
			t.Fatal("expected non-nil refund response")
		}
		if result1.Status != "refunded" {
			t.Fatalf("expected status 'refunded', got %q", result1.Status)
		}

		// Record state after first refund
		stockAfterFirstRefund := deps.orderRepo.bookStocks[bookID]
		auditActionsAfterFirst := len(deps.audit.actions)

		// Property: order must be in 'refunded' status after first refund
		storedOrder := deps.orderRepo.orders[orderID]
		if storedOrder.Status != domainbookshop.OrderStatusRefunded {
			t.Fatalf("order must be 'refunded' after first refund, got %q", storedOrder.Status)
		}

		// Generate 1–3 duplicate refund requests
		numDuplicates := rapid.IntRange(1, 3).Draw(t, "num_duplicates")

		for i := 0; i < numDuplicates; i++ {
			result2, err := svc.ProcessRefund(ctx, ProcessRefundCommand{
				ActorID:        actorID,
				ActorName:      "Admin",
				OrderID:        orderID,
				IdempotencyKey: idempotencyKey,
				IPAddress:      "127.0.0.1",
			})
			if err != nil {
				t.Fatalf("duplicate refund request %d failed: %v", i+1, err)
			}
			if result2 == nil {
				t.Fatalf("duplicate refund request %d returned nil", i+1)
			}

			// Property: duplicate must return the same status
			if result2.Status != "refunded" {
				t.Fatalf("duplicate refund %d must return 'refunded', got %q", i+1, result2.Status)
			}

			// Property: stock must not be incremented again for physical orders
			if isPhysical {
				if deps.orderRepo.bookStocks[bookID] != stockAfterFirstRefund {
					t.Fatalf("duplicate refund %d must not increment stock again: expected %d, got %d",
						i+1, stockAfterFirstRefund, deps.orderRepo.bookStocks[bookID])
				}
			}

			// Property: audit log must not grow (no new audit entry for duplicate)
			if len(deps.audit.actions) != auditActionsAfterFirst {
				t.Fatalf("duplicate refund %d must not create new audit log entry: expected %d actions, got %d",
					i+1, auditActionsAfterFirst, len(deps.audit.actions))
			}
		}

		// Property: for physical orders, stock must be restored exactly once
		if isPhysical {
			expectedStock := initialStock + 1 // restored once
			if deps.orderRepo.bookStocks[bookID] != expectedStock {
				t.Fatalf("physical stock must be restored exactly once: expected %d, got %d",
					expectedStock, deps.orderRepo.bookStocks[bookID])
			}
		}

		// Property: audit log must have exactly one 'refund_issued' entry
		refundAuditCount := 0
		for _, action := range deps.audit.actions {
			if action == "refund_issued" {
				refundAuditCount++
			}
		}
		if refundAuditCount != 1 {
			t.Fatalf("expected exactly 1 'refund_issued' audit entry, got %d", refundAuditCount)
		}
	})
}
