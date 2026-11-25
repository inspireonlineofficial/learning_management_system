package bookshop

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"math"
	"regexp"
	"unicode/utf8"

	"lms-backend/internal/domain/bookshop"
	"lms-backend/internal/domain/notifications"
	tsclient "lms-backend/internal/infrastructure/typesense"
	"lms-backend/pkg/apperrors"
	"lms-backend/pkg/logger"
	"lms-backend/pkg/pagination"

	"github.com/google/uuid"
)

// Service defines the interface for all bookshop use cases.
type Service interface {
	// Catalog & digital access (Requirements: 19.1–19.5)
	ListBooks(ctx context.Context, cmd ListBooksCommand) (*BookListResponse, error)
	GetBookDetail(ctx context.Context, bookID uuid.UUID) (*BookResponse, error)
	GetBookPreview(ctx context.Context, cmd GetBookPreviewCommand) (*BookPreviewResponse, error)
	GetDigitalBookAccess(ctx context.Context, cmd GetDigitalBookAccessCommand) (*DigitalBookAccessResponse, error)
	UpsertBookmark(ctx context.Context, cmd UpsertBookmarkCommand) (*BookmarkResponse, error)

	// Admin use cases (Requirements: 19.6, 19.7, 20.2, 20.3, 20.4)
	ListAdminBooks(ctx context.Context, cmd ListBooksCommand) (*BookListResponse, error)
	GetAdminBook(ctx context.Context, bookID uuid.UUID) (*BookResponse, error)
	ListAdminOrders(ctx context.Context, page, limit int) (*OrderListResponse, error)
	CreateBook(ctx context.Context, cmd CreateBookCommand) (*BookResponse, error)
	UpdateBook(ctx context.Context, cmd UpdateBookCommand) (*BookResponse, error)
	UploadBookCover(ctx context.Context, cmd UploadBookCoverCommand) (*BookResponse, error)
	UploadBookFile(ctx context.Context, cmd UploadBookFileCommand) (*BookResponse, error)
	FulfilOrder(ctx context.Context, cmd FulfilOrderCommand) (*OrderResponse, error)
	ProcessRefund(ctx context.Context, cmd ProcessRefundCommand) (*RefundResponse, error)
	ListRefundQueue(ctx context.Context, page, limit int) (*RefundQueueResponse, error)
	ListRefundRequests(ctx context.Context, status string, page, limit int) (*RefundRequestListResponse, error)
	ApproveRefundRequest(ctx context.Context, cmd ReviewRefundRequestCommand) (*RefundResponse, error)
	RejectRefundRequest(ctx context.Context, cmd ReviewRefundRequestCommand) (*RefundRequestResponse, error)
	ListAdminOrdersByStatus(ctx context.Context, status string, page, limit int) (*OrderListResponse, error)

	// Order use cases (Requirements: 19.8, 20.1, 20.5)
	PlaceOrder(ctx context.Context, cmd PlaceOrderCommand) (*OrderResponse, error)
	PlaceOrders(ctx context.Context, cmds []PlaceOrderCommand) ([]*OrderResponse, error)
	ListStudentOrders(ctx context.Context, cmd ListStudentOrdersCommand) (*OrderListResponse, error)
	GetStudentOrder(ctx context.Context, studentID, orderID uuid.UUID) (*OrderResponse, error)
	ListStudentLibrary(ctx context.Context, cmd ListStudentOrdersCommand) (*LibraryBookListResponse, error)
	GetStudentBookStatus(ctx context.Context, studentID, bookID uuid.UUID) (*StudentBookStatusResponse, error)
	CheckOrderable(ctx context.Context, studentID, bookID uuid.UUID, format bookshop.OrderFormat) (*BookResponse, string)
	RequestRefund(ctx context.Context, cmd RequestRefundCommand) (*RefundRequestResponse, error)
	ListMyRefundRequests(ctx context.Context, studentID uuid.UUID, page, limit int) (*RefundRequestListResponse, error)
}

// TxRunner runs fn inside one database transaction carried by its context.
type TxRunner interface {
	RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// Configurer lets the composition root attach optional collaborators.
type Configurer interface {
	SetTxRunner(TxRunner)
	SetRefundRepository(bookshop.RefundRepository)
}

// RefundWindow is how long after purchase (digital) or delivery (printed) a
// student may ask for a refund.
const RefundWindow = 7 * 24 * time.Hour

// maxRefundRequestsPerOrder stops repeated requests after declines.
const maxRefundRequestsPerOrder = 3

type noTx struct{}

func (noTx) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error { return fn(ctx) }

// StorageClient defines the interface for object storage operations.
type StorageClient interface {
	PutObject(ctx context.Context, bucket, key string, r io.Reader, size int64, contentType string) error
	PresignGetURL(ctx context.Context, bucket, key string, ttl time.Duration) (string, error)
	DeleteObject(ctx context.Context, bucket, key string) error
}

// AuditLogger records privileged admin actions.
type AuditLogger interface {
	LogAction(ctx context.Context, actorID uuid.UUID, actorName, action, targetType string, targetID uuid.UUID, metadata map[string]interface{}, ipAddress string) error
}

// IdempotencyStore checks and stores idempotency keys to prevent duplicate processing.
type IdempotencyStore interface {
	// Get returns the cached response for a key, or ("", false) if not found.
	Get(ctx context.Context, key string) (string, bool, error)
	// Set stores a response for a key with a 24h TTL.
	Set(ctx context.Context, key string, response string) error
}

type service struct {
	bookRepo     bookshop.BookRepository
	orderRepo    bookshop.OrderRepository
	bookmarkRepo bookshop.BookBookmarkRepository
	jobQueue     notifications.JobQueue
	storage      StorageClient
	audit        AuditLogger
	idempotency  IdempotencyStore
	booksBucket  string
	indexer      tsclient.Indexer
	tx           TxRunner
	refunds      bookshop.RefundRepository
	now          func() time.Time
}

// SetRefundRepository enables refund requests and payout records.
func (s *service) SetRefundRepository(r bookshop.RefundRepository) { s.refunds = r }

// SetTxRunner makes stock changes and order writes atomic.
func (s *service) SetTxRunner(tx TxRunner) {
	if tx != nil {
		s.tx = tx
	}
}

// NewService creates a new bookshop service.
func NewService(
	bookRepo bookshop.BookRepository,
	orderRepo bookshop.OrderRepository,
	bookmarkRepo bookshop.BookBookmarkRepository,
	jobQueue notifications.JobQueue,
	storage StorageClient,
	audit AuditLogger,
	idempotency IdempotencyStore,
	booksBucket string,
	indexer tsclient.Indexer,
) Service {
	return &service{
		bookRepo:     bookRepo,
		orderRepo:    orderRepo,
		bookmarkRepo: bookmarkRepo,
		jobQueue:     jobQueue,
		storage:      storage,
		audit:        audit,
		idempotency:  idempotency,
		booksBucket:  booksBucket,
		indexer:      indexer,
		tx:           noTx{},
		now:          time.Now,
	}
}

// ─── Catalog & Digital Access ─────────────────────────────────────────────────

// ListBooks returns a paginated, filterable list of active books.
// Requirements: 19.1
func (s *service) ListBooks(ctx context.Context, cmd ListBooksCommand) (*BookListResponse, error) {
	return s.listBooks(ctx, cmd, true)
}

// ListAdminBooks returns all books, including records hidden from the public catalog.
func (s *service) ListAdminBooks(ctx context.Context, cmd ListBooksCommand) (*BookListResponse, error) {
	return s.listBooks(ctx, cmd, false)
}

func (s *service) listBooks(ctx context.Context, cmd ListBooksCommand, activeOnly bool) (*BookListResponse, error) {
	if cmd.Page < 1 {
		cmd.Page = 1
	}
	if cmd.Limit < 1 || cmd.Limit > 100 {
		cmd.Limit = 20
	}

	filter := bookshop.BookFilter{
		Search:     cmd.Search,
		Subject:    cmd.Subject,
		ClassGrade: cmd.ClassGrade,
		Format:     cmd.Format,
		MinPrice:   cmd.MinPrice,
		MaxPrice:   cmd.MaxPrice,
		ActiveOnly: activeOnly,
		Sort:       cmd.Sort,
	}

	books, total, err := s.bookRepo.List(ctx, filter, cmd.Page, cmd.Limit)
	if err != nil {
		return nil, apperrors.NewInternalError("LIST_BOOKS_FAILED", "failed to list books")
	}

	data := make([]*BookResponse, 0, len(books))
	for _, b := range books {
		data = append(data, s.toBookResponse(ctx, b))
	}

	meta := pagination.NewMeta(total, cmd.Page, cmd.Limit)

	return &BookListResponse{
		Data: data,
		Meta: map[string]interface{}{
			"page":        meta.Page,
			"limit":       meta.Limit,
			"total":       meta.Total,
			"total_pages": meta.TotalPages,
		},
	}, nil
}

// ListAdminOrders returns a paginated fulfilment queue for administrators.
func (s *service) ListAdminOrders(ctx context.Context, page, limit int) (*OrderListResponse, error) {
	return s.ListAdminOrdersByStatus(ctx, "", page, limit)
}

// ListAdminOrdersByStatus returns the admin order list, optionally filtered by status.
func (s *service) ListAdminOrdersByStatus(ctx context.Context, status string, page, limit int) (*OrderListResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	var statuses []bookshop.OrderStatus
	if status != "" {
		st := bookshop.OrderStatus(status)
		if !validOrderStatus(st) {
			return nil, apperrors.NewSimpleValidationError("INVALID_STATUS", "status must be placed, shipped, delivered, refunded or cancelled")
		}
		statuses = []bookshop.OrderStatus{st}
	}

	orders, total, err := s.orderRepo.ListByStatuses(ctx, statuses, page, limit)
	if err != nil {
		return nil, apperrors.NewInternalError("LIST_ORDERS_FAILED", "failed to list book orders")
	}

	data := make([]*OrderResponse, 0, len(orders))
	for _, order := range orders {
		data = append(data, toOrderResponse(order))
	}
	meta := pagination.NewMeta(total, page, limit)
	return &OrderListResponse{
		Data: data,
		Meta: map[string]interface{}{
			"page":        meta.Page,
			"limit":       meta.Limit,
			"total":       meta.Total,
			"total_pages": meta.TotalPages,
		},
	}, nil
}

// GetBookDetail returns the public metadata for a single active book.
func (s *service) GetBookDetail(ctx context.Context, bookID uuid.UUID) (*BookResponse, error) {
	book, err := s.bookRepo.FindByID(ctx, bookID)
	if err != nil || book == nil || !book.IsActive {
		return nil, apperrors.NewNotFoundError("BOOK_NOT_FOUND", "book not found")
	}

	return s.toBookResponse(ctx, book), nil
}

// GetAdminBook returns a book whatever its visibility, so an admin can edit
// (and re-publish) a hidden book.
func (s *service) GetAdminBook(ctx context.Context, bookID uuid.UUID) (*BookResponse, error) {
	book, err := s.bookRepo.FindByID(ctx, bookID)
	if err != nil || book == nil {
		return nil, apperrors.NewNotFoundError("BOOK_NOT_FOUND", "book not found")
	}
	return s.toBookResponse(ctx, book), nil
}

// GetBookPreview returns a presigned URL for the book's preview pages (10-min TTL).
// The raw file URL is never exposed.
// Requirements: 19.2
func (s *service) GetBookPreview(ctx context.Context, cmd GetBookPreviewCommand) (*BookPreviewResponse, error) {
	book, err := s.bookRepo.FindByID(ctx, cmd.BookID)
	if err != nil || book == nil {
		return nil, apperrors.NewNotFoundError("BOOK_NOT_FOUND", "book not found")
	}

	if !book.IsActive {
		return nil, apperrors.NewNotFoundError("BOOK_NOT_FOUND", "book not found")
	}

	if book.PreviewRustFSKey == nil || *book.PreviewRustFSKey == "" {
		return nil, apperrors.NewNotFoundError("PREVIEW_NOT_AVAILABLE", "no preview available for this book")
	}

	const previewTTL = 10 * time.Minute
	url, err := s.storage.PresignGetURL(ctx, s.booksBucket, *book.PreviewRustFSKey, previewTTL)
	if err != nil {
		return nil, apperrors.NewInternalError("PRESIGN_FAILED", "failed to generate preview URL")
	}

	return &BookPreviewResponse{
		BookID:     book.ID,
		PreviewURL: url,
		ExpiresIn:  int(previewTTL.Seconds()),
	}, nil
}

// GetDigitalBookAccess verifies a non-refunded purchase and returns a presigned URL.
// Returns NO_ACCESS if the purchase has been refunded.
// Requirements: 19.3, 19.4
func (s *service) GetDigitalBookAccess(ctx context.Context, cmd GetDigitalBookAccessCommand) (*DigitalBookAccessResponse, error) {
	book, err := s.bookRepo.FindByID(ctx, cmd.BookID)
	if err != nil || book == nil {
		return nil, apperrors.NewNotFoundError("BOOK_NOT_FOUND", "book not found")
	}

	if book.Format == bookshop.BookFormatPhysical {
		return nil, apperrors.NewForbiddenError("NO_ACCESS", "this book is not available in digital format")
	}

	// Verify the student has a valid (non-refunded) purchase
	order, err := s.orderRepo.FindNonRefundedByStudentAndBook(ctx, cmd.StudentID, cmd.BookID)
	if err != nil {
		return nil, apperrors.NewInternalError("ACCESS_CHECK_FAILED", "failed to verify purchase")
	}
	if order == nil || order.Format != bookshop.OrderFormatDigital {
		return nil, apperrors.NewForbiddenError("NO_ACCESS", "no valid purchase found for this book")
	}

	if book.DigitalFileRustFSKey == nil || *book.DigitalFileRustFSKey == "" {
		return nil, apperrors.NewConflictError("FILE_NOT_AVAILABLE", "the digital edition has not been uploaded yet")
	}

	const accessTTL = 2 * time.Hour
	url, err := s.storage.PresignGetURL(ctx, s.booksBucket, *book.DigitalFileRustFSKey, accessTTL)
	if err != nil {
		return nil, apperrors.NewInternalError("PRESIGN_FAILED", "failed to generate access URL")
	}

	// Fetch bookmark for last_page_read
	lastPageRead := 0
	bookmark, _ := s.bookmarkRepo.FindByStudentAndBook(ctx, cmd.StudentID, cmd.BookID)
	if bookmark != nil {
		lastPageRead = bookmark.LastPageRead
	}

	return &DigitalBookAccessResponse{
		BookID:       book.ID,
		AccessURL:    url,
		ExpiresIn:    int(accessTTL.Seconds()),
		LastPageRead: lastPageRead,
	}, nil
}

// UpsertBookmark persists the student's current reading position.
// Requirements: 19.5
func (s *service) UpsertBookmark(ctx context.Context, cmd UpsertBookmarkCommand) (*BookmarkResponse, error) {
	if cmd.LastPageRead < 0 || cmd.LastPageRead > maxBookPage {
		return nil, apperrors.NewSimpleValidationError("INVALID_PAGE", "last_page_read must be between 0 and 100000")
	}
	// Only a student who can read the book may store a reading position.
	order, err := s.orderRepo.FindNonRefundedByStudentAndBook(ctx, cmd.StudentID, cmd.BookID)
	if err != nil {
		return nil, apperrors.NewInternalError("ACCESS_CHECK_FAILED", "failed to verify purchase")
	}
	if order == nil || order.Format != bookshop.OrderFormatDigital {
		return nil, apperrors.NewForbiddenError("NO_ACCESS", "no valid purchase found for this book")
	}
	now := time.Now().UTC()
	bookmark := &bookshop.BookBookmark{
		ID:           uuid.New(),
		StudentID:    cmd.StudentID,
		BookID:       cmd.BookID,
		LastPageRead: cmd.LastPageRead,
		UpdatedAt:    now,
	}

	if err := s.bookmarkRepo.Upsert(ctx, bookmark); err != nil {
		return nil, apperrors.NewInternalError("BOOKMARK_FAILED", "failed to save bookmark")
	}

	return &BookmarkResponse{
		BookID:       cmd.BookID,
		StudentID:    cmd.StudentID,
		LastPageRead: cmd.LastPageRead,
		UpdatedAt:    now,
	}, nil
}

// ─── Admin Use Cases ──────────────────────────────────────────────────────────

// CreateBook creates a new book record.
// digital_file_rustfs_key is stored internally and never exposed.
// Requirements: 19.6
func (s *service) CreateBook(ctx context.Context, cmd CreateBookCommand) (*BookResponse, error) {
	now := time.Now().UTC()

	cmd.Title, cmd.Author = strings.TrimSpace(cmd.Title), strings.TrimSpace(cmd.Author)
	cmd.Currency = strings.ToUpper(strings.TrimSpace(cmd.Currency))
	if cmd.Currency == "" {
		cmd.Currency = "BDT"
	}
	if err := validateBookFields(cmd.Title, cmd.Author, cmd.Format, cmd.Price, cmd.Currency, cmd.PhysicalStock); err != nil {
		return nil, err
	}

	// Files are normally attached through the upload endpoints. Raw keys are
	// still accepted for API compatibility, but only for objects stored under
	// the bookshop's own prefixes.
	var digitalKey *string
	if cmd.DigitalFileRustFSKey != "" {
		if !strings.HasPrefix(cmd.DigitalFileRustFSKey, "book-files/") && !strings.HasPrefix(cmd.DigitalFileRustFSKey, "books/") {
			return nil, apperrors.NewSimpleValidationError("INVALID_FILE_KEY", "upload the digital edition with the file upload endpoint")
		}
		digitalKey = &cmd.DigitalFileRustFSKey
	}
	var previewKey *string
	if cmd.PreviewRustFSKey != "" {
		if !strings.HasPrefix(cmd.PreviewRustFSKey, "book-previews/") && !strings.HasPrefix(cmd.PreviewRustFSKey, "books/") {
			return nil, apperrors.NewSimpleValidationError("INVALID_FILE_KEY", "upload the preview with the preview upload endpoint")
		}
		previewKey = &cmd.PreviewRustFSKey
	}

	book := &bookshop.Book{
		ID:                   uuid.New(),
		Title:                cmd.Title,
		Author:               cmd.Author,
		Subject:              cmd.Subject,
		ClassGrade:           cmd.ClassGrade,
		Description:          cmd.Description,
		Format:               cmd.Format,
		Price:                cmd.Price,
		Currency:             cmd.Currency,
		PhysicalStock:        cmd.PhysicalStock,
		DigitalFileRustFSKey: digitalKey,
		PreviewRustFSKey:     previewKey,
		IsActive:             true,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	if err := s.bookRepo.Create(ctx, book); err != nil {
		return nil, apperrors.NewInternalError("CREATE_BOOK_FAILED", "failed to create book")
	}

	if s.indexer != nil {
		if err := s.indexer.UpsertBook(ctx, tsclient.BookDocument{
			ID:       book.ID.String(),
			Title:    book.Title,
			Author:   book.Author,
			Format:   string(book.Format),
			IsActive: book.IsActive,
		}); err != nil {
			log.Printf("typesense index error: %v", err)
		}
	}
	if s.audit != nil {
		_ = s.audit.LogAction(ctx, cmd.ActorID, "", "book_created", "book", book.ID, nil, "")
	}

	logger.Info(ctx, "Book created", "book_id", book.ID, "title", book.Title)
	return s.toBookResponse(ctx, book), nil
}

// UpdateBook updates an existing book.
// Setting is_active: false removes the book from the public catalog.
// Requirements: 19.7
func (s *service) UpdateBook(ctx context.Context, cmd UpdateBookCommand) (*BookResponse, error) {
	book, err := s.bookRepo.FindByID(ctx, cmd.BookID)
	if err != nil || book == nil {
		return nil, apperrors.NewNotFoundError("BOOK_NOT_FOUND", "book not found")
	}

	if cmd.Title != nil {
		book.Title = strings.TrimSpace(*cmd.Title)
	}
	if cmd.Author != nil {
		book.Author = strings.TrimSpace(*cmd.Author)
	}
	if cmd.Format != nil {
		book.Format = *cmd.Format
	}
	if cmd.Currency != nil {
		book.Currency = strings.ToUpper(strings.TrimSpace(*cmd.Currency))
	}
	if cmd.Subject != nil {
		book.Subject = *cmd.Subject
	}
	if cmd.ClassGrade != nil {
		book.ClassGrade = *cmd.ClassGrade
	}
	if cmd.Description != nil {
		book.Description = *cmd.Description
	}
	if cmd.Price != nil {
		book.Price = *cmd.Price
	}
	if cmd.PhysicalStock != nil {
		book.PhysicalStock = *cmd.PhysicalStock
	}
	if cmd.IsActive != nil {
		book.IsActive = *cmd.IsActive
	}
	if err := validateBookFields(book.Title, book.Author, book.Format, book.Price, book.Currency, book.PhysicalStock); err != nil {
		return nil, err
	}
	book.UpdatedAt = time.Now().UTC()

	if err := s.bookRepo.Update(ctx, book); err != nil {
		return nil, apperrors.NewInternalError("UPDATE_BOOK_FAILED", "failed to update book")
	}

	if s.indexer != nil {
		if err := s.indexer.UpsertBook(ctx, tsclient.BookDocument{
			ID:       book.ID.String(),
			Title:    book.Title,
			Author:   book.Author,
			Format:   string(book.Format),
			IsActive: book.IsActive,
		}); err != nil {
			log.Printf("typesense index error: %v", err)
		}
	}
	if s.audit != nil {
		_ = s.audit.LogAction(ctx, cmd.ActorID, "", "book_updated", "book", book.ID, nil, "")
	}

	return s.toBookResponse(ctx, book), nil
}

func (s *service) UploadBookCover(ctx context.Context, cmd UploadBookCoverCommand) (*BookResponse, error) {
	book, err := s.bookRepo.FindByID(ctx, cmd.BookID)
	if err != nil || book == nil {
		return nil, apperrors.NewNotFoundError("BOOK_NOT_FOUND", "book not found")
	}
	contentType, err := validateCoverUpload(cmd.FileSize, cmd.MimeType, cmd.MagicBytes)
	if err != nil {
		return nil, err
	}
	key := generatedBookCoverKey(cmd.FileName)
	if err := s.storage.PutObject(ctx, s.booksBucket, key, cmd.Reader, cmd.FileSize, contentType); err != nil {
		return nil, apperrors.NewInternalError("COVER_UPLOAD_FAILED", "failed to store cover image")
	}
	book.CoverRustFSKey = &key
	book.UpdatedAt = time.Now().UTC()
	if err := s.bookRepo.Update(ctx, book); err != nil {
		return nil, err
	}
	if s.audit != nil {
		_ = s.audit.LogAction(ctx, cmd.ActorID, "", "book_cover_updated", "book", book.ID, nil, cmd.IPAddress)
	}
	return s.toBookResponse(ctx, book), nil
}

// FulfilOrder moves an order through its fulfilment states:
//
//	placed  → shipped   (physical only, tracking number required)
//	shipped → delivered (physical only)
//	placed  → cancelled (physical orders get their copy back in stock)
//
// Each transition is a compare-and-swap, so concurrent admin actions apply
// exactly once. The student is notified of every change.
// Requirements: 20.2
func (s *service) FulfilOrder(ctx context.Context, cmd FulfilOrderCommand) (*OrderResponse, error) {
	target := bookshop.OrderStatus(cmd.Status)
	if target == "" {
		target = bookshop.OrderStatusShipped
	}
	var updated *bookshop.Order
	err := s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		order, err := s.orderRepo.FindByID(txCtx, cmd.OrderID)
		if err != nil || order == nil {
			return apperrors.NewNotFoundError("ORDER_NOT_FOUND", "order not found")
		}
		from := order.Status
		now := time.Now().UTC()
		switch target {
		case bookshop.OrderStatusShipped:
			if !order.IsPhysical() {
				return apperrors.NewSimpleValidationError("INVALID_TRANSITION", "digital orders are delivered through the reader and cannot be shipped")
			}
			if from != bookshop.OrderStatusPlaced {
				return apperrors.NewSimpleValidationError("INVALID_STATUS", "only placed orders can be shipped")
			}
			tracking := strings.TrimSpace(cmd.TrackingNumber)
			if tracking == "" || len(tracking) > 100 {
				return apperrors.NewSimpleValidationError("VALIDATION_ERROR", "tracking_number is required (100 characters or fewer)")
			}
			order.TrackingNumber = &tracking
			order.ShippedAt = &now
		case bookshop.OrderStatusDelivered:
			if !order.IsPhysical() || from != bookshop.OrderStatusShipped {
				return apperrors.NewSimpleValidationError("INVALID_STATUS", "only shipped printed orders can be marked delivered")
			}
			order.DeliveredAt = &now
		case bookshop.OrderStatusCancelled:
			if from != bookshop.OrderStatusPlaced {
				return apperrors.NewSimpleValidationError("INVALID_STATUS", "only orders that have not shipped can be cancelled; refund shipped orders instead")
			}
		default:
			return apperrors.NewSimpleValidationError("INVALID_STATUS", "status must be shipped, delivered or cancelled")
		}
		order.Status = target
		order.UpdatedAt = now
		ok, err := s.orderRepo.UpdateStatusIfCurrent(txCtx, order, from)
		if err != nil {
			return apperrors.NewInternalError("FULFIL_ORDER_FAILED", "failed to update order status")
		}
		if !ok {
			return apperrors.NewConflictError("ORDER_CHANGED", "the order was changed by someone else; reload and try again")
		}
		if target == bookshop.OrderStatusCancelled && order.IsPhysical() {
			if err := s.orderRepo.IncrementPhysicalStock(txCtx, order.BookID); err != nil {
				return apperrors.NewInternalError("STOCK_RESTORE_FAILED", "failed to restore physical stock")
			}
		}
		updated = order
		return nil
	})
	if err != nil {
		return nil, err
	}

	if s.audit != nil {
		_ = s.audit.LogAction(ctx, cmd.ActorID, "", "order_"+string(target), "order", updated.ID,
			map[string]interface{}{"tracking_number": updated.TrackingNumber}, "")
	}
	s.enqueueOrderNotification(ctx, updated, "order_"+string(target))
	logger.Info(ctx, "Order status changed", "order_id", updated.ID, "status", target)
	return toOrderResponse(updated), nil
}

// ProcessRefund processes a refund for an order.
// Idempotent via IdempotencyKey — duplicate requests return the cached response.
// Restores physical_stock for physical orders and revokes digital access.
// Records audit log entry with action 'refund_issued'.
// Requirements: 20.3, 20.4
func (s *service) ProcessRefund(ctx context.Context, cmd ProcessRefundCommand) (*RefundResponse, error) {
	// Idempotency check
	if cmd.IdempotencyKey != "" {
		cached, found, err := s.idempotency.Get(ctx, "refund:"+cmd.IdempotencyKey)
		if err == nil && found {
			var resp RefundResponse
			if jsonErr := json.Unmarshal([]byte(cached), &resp); jsonErr == nil {
				return &resp, nil
			}
		}
	}

	var result refundResult
	err := s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		var err error
		result, err = s.refundOrderTx(txCtx, cmd)
		return err
	})
	if err != nil {
		return nil, err
	}
	resp := s.finishRefund(ctx, cmd, result)
	if cmd.IdempotencyKey != "" && !result.already {
		if data, err := json.Marshal(resp); err == nil {
			_ = s.idempotency.Set(ctx, "refund:"+cmd.IdempotencyKey, string(data))
		}
	}
	return resp, nil
}

type refundResult struct {
	order         *bookshop.Order
	refund        *bookshop.OrderRefund
	request       *bookshop.RefundRequest
	already       bool
	stockRestored bool
	accessRevoked bool
}

// refundOrderTx refunds one order inside the caller's transaction:
//
//   - compare-and-swap on the order status, so of several concurrent refunds
//     exactly one applies (the others see "already refunded");
//   - an unshipped printed copy goes back into stock (a shipped one is with
//     the student);
//   - the payout (amount, method, reference) is recorded in order_refunds,
//     and an open refund request for the order is marked approved.
func (s *service) refundOrderTx(txCtx context.Context, cmd ProcessRefundCommand) (refundResult, error) {
	var res refundResult
	for attempt := 0; attempt < 3; attempt++ {
		current, err := s.orderRepo.FindByID(txCtx, cmd.OrderID)
		if err != nil || current == nil {
			return res, apperrors.NewNotFoundError("ORDER_NOT_FOUND", "order not found")
		}
		res.order = current
		switch current.Status {
		case bookshop.OrderStatusRefunded:
			res.already = true
			return res, nil
		case bookshop.OrderStatusCancelled:
			return res, apperrors.NewSimpleValidationError("INVALID_STATUS", "cancelled orders cannot be refunded")
		}
		payout, err := buildPayout(current, cmd)
		if err != nil {
			return res, err
		}
		from := current.Status
		current.Status = bookshop.OrderStatusRefunded
		current.UpdatedAt = time.Now().UTC()
		ok, err := s.orderRepo.UpdateStatusIfCurrent(txCtx, current, from)
		if err != nil {
			return res, apperrors.NewInternalError("REFUND_FAILED", "failed to update order status")
		}
		if !ok {
			continue // someone else changed it; re-read and decide again
		}
		if current.IsPhysical() && from == bookshop.OrderStatusPlaced {
			if err := s.orderRepo.IncrementPhysicalStock(txCtx, current.BookID); err != nil {
				return res, apperrors.NewInternalError("STOCK_RESTORE_FAILED", "failed to restore physical stock")
			}
			res.stockRestored = true
		}
		res.accessRevoked = current.Format == bookshop.OrderFormatDigital

		if s.refunds != nil {
			request := cmd.request
			if request == nil {
				request, _ = s.refunds.FindPendingRequestByOrder(txCtx, current.ID)
			}
			if request != nil {
				payout.RefundRequestID = &request.ID
				now := time.Now().UTC()
				actor := cmd.ActorID
				request.Status = bookshop.RefundRequestApproved
				request.ReviewedBy = &actor
				request.ReviewedAt = &now
				request.UpdatedAt = now
				if note := strings.TrimSpace(cmd.Note); note != "" {
					request.AdminNote = &note
				}
				if err := s.refunds.UpdateRequest(txCtx, request); err != nil {
					return res, apperrors.NewInternalError("REFUND_FAILED", "failed to update the refund request")
				}
				res.request = request
			}
			if cmd.ActorID != uuid.Nil {
				payout.ProcessedBy = &cmd.ActorID
			}
			if err := s.refunds.CreateRefund(txCtx, payout); err != nil {
				return res, apperrors.NewInternalError("REFUND_FAILED", "failed to record the refund")
			}
			res.refund = payout
		}
		return res, nil
	}
	return res, apperrors.NewConflictError("ORDER_CHANGED", "the order is being changed by someone else; try again")
}

var payoutNeedsReference = map[bookshop.PayoutMethod]bool{
	bookshop.PayoutBKash: true, bookshop.PayoutNagad: true, bookshop.PayoutRocket: true, bookshop.PayoutBank: true,
}

// buildPayout validates how the money is returned. Paid orders need a real
// method (and a transaction reference for mobile money and bank); the
// amount defaults to the order amount and may be partial, never more.
func buildPayout(order *bookshop.Order, cmd ProcessRefundCommand) (*bookshop.OrderRefund, error) {
	ref := &bookshop.OrderRefund{
		ID:        uuid.New(),
		OrderID:   order.ID,
		Currency:  order.Currency,
		CreatedAt: time.Now().UTC(),
	}
	if note := strings.TrimSpace(cmd.Note); note != "" {
		if len(note) > 2000 {
			return nil, apperrors.NewSimpleValidationError("INVALID_NOTE", "note must be 2000 characters or fewer")
		}
		ref.Note = &note
	}
	if order.Amount <= 0 {
		ref.PayoutMethod = bookshop.PayoutNone
		return ref, nil
	}
	method := bookshop.PayoutMethod(strings.ToLower(strings.TrimSpace(cmd.PayoutMethod)))
	if !bookshop.ValidPayoutMethod(method) || method == bookshop.PayoutNone {
		return nil, apperrors.NewValidationError([]map[string]string{{"field": "payout_method", "message": "how the money was returned: bkash, nagad, rocket, bank, cash or other"}})
	}
	reference := strings.TrimSpace(cmd.PayoutReference)
	if len(reference) > 200 {
		return nil, apperrors.NewValidationError([]map[string]string{{"field": "payout_reference", "message": "200 characters or fewer"}})
	}
	if payoutNeedsReference[method] && reference == "" {
		return nil, apperrors.NewValidationError([]map[string]string{{"field": "payout_reference", "message": "the transaction ID of the refund payment is required"}})
	}
	amount := order.Amount
	if cmd.Amount != nil {
		amount = math.Round(*cmd.Amount*100) / 100
		if amount <= 0 || amount > order.Amount {
			return nil, apperrors.NewValidationError([]map[string]string{{"field": "amount", "message": fmt.Sprintf("must be more than 0 and at most %.2f", order.Amount)}})
		}
	}
	ref.PayoutMethod = method
	ref.Amount = amount
	if reference != "" {
		ref.PayoutReference = &reference
	}
	return ref, nil
}

// finishRefund writes the audit entry and notifies the student once the
// refund transaction has committed.
func (s *service) finishRefund(ctx context.Context, cmd ProcessRefundCommand, res refundResult) *RefundResponse {
	order := res.order
	if res.already {
		return &RefundResponse{
			OrderID:       order.ID,
			Status:        "refunded",
			AccessRevoked: order.Format == bookshop.OrderFormatDigital,
			ProcessedAt:   order.UpdatedAt,
		}
	}
	metadata := map[string]interface{}{
		"book_id":        order.BookID,
		"student_id":     order.StudentID,
		"format":         order.Format,
		"amount":         order.Amount,
		"stock_restored": res.stockRestored,
		"access_revoked": res.accessRevoked,
	}
	if res.refund != nil {
		metadata["refunded_amount"] = res.refund.Amount
		metadata["payout_method"] = res.refund.PayoutMethod
	}
	if res.request != nil {
		metadata["refund_request_id"] = res.request.ID
	}
	if s.audit != nil {
		_ = s.audit.LogAction(ctx, cmd.ActorID, cmd.ActorName, "refund_issued", "order", order.ID, metadata, cmd.IPAddress)
	}
	s.enqueueOrderNotification(ctx, order, "order_refunded")
	logger.Info(ctx, "Refund processed", "order_id", order.ID, "actor_id", cmd.ActorID)

	resp := &RefundResponse{
		OrderID:       order.ID,
		Status:        "refunded",
		StockRestored: res.stockRestored,
		AccessRevoked: res.accessRevoked,
		ProcessedAt:   time.Now().UTC(),
	}
	if res.refund != nil {
		resp.Refund = toOrderRefundResponse(res.refund)
	}
	return resp
}

// ─── Order Use Cases ──────────────────────────────────────────────────────────

// PlaceOrder creates a new book order.
// Decrements physical_stock atomically in the same transaction.
// Rejects if stock = 0 (HTTP 422).
// Sends a receipt email.
// Requirements: 19.8, 20.1
func (s *service) PlaceOrder(ctx context.Context, cmd PlaceOrderCommand) (*OrderResponse, error) {
	order, created, err := s.placeOrder(ctx, cmd)
	if err != nil {
		return nil, err
	}
	if created {
		s.enqueueOrderNotification(ctx, order, "order_placed")
		logger.Info(ctx, "Order placed", "order_id", order.ID, "student_id", cmd.StudentID, "book_id", cmd.BookID)
	}
	return toOrderResponse(order), nil
}

// PlaceOrders places several orders atomically (checkout): if any item
// fails, no order is created and no stock is taken.
func (s *service) PlaceOrders(ctx context.Context, cmds []PlaceOrderCommand) ([]*OrderResponse, error) {
	type placed struct {
		order   *bookshop.Order
		created bool
	}
	results := make([]placed, 0, len(cmds))
	err := s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		for _, cmd := range cmds {
			order, created, err := s.placeOrder(txCtx, cmd)
			if err != nil {
				return err
			}
			results = append(results, placed{order, created})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	out := make([]*OrderResponse, 0, len(results))
	for _, r := range results {
		if r.created {
			s.enqueueOrderNotification(ctx, r.order, "order_placed")
		}
		out = append(out, toOrderResponse(r.order))
	}
	return out, nil
}

// placeOrder validates and creates one order. created is false when an
// existing order is returned (idempotent replay or a free book already owned).
func (s *service) placeOrder(ctx context.Context, cmd PlaceOrderCommand) (*bookshop.Order, bool, error) {
	// Idempotency check (scoped to the caller: a key reused by another
	// student must never return someone else's order).
	if cmd.IdempotencyKey != "" {
		existing, err := s.orderRepo.FindByIdempotencyKey(ctx, cmd.IdempotencyKey)
		if err == nil && existing != nil {
			if existing.StudentID != cmd.StudentID || existing.BookID != cmd.BookID {
				return nil, false, apperrors.NewConflictError("IDEMPOTENCY_KEY_REUSED", "idempotency key was already used for a different request")
			}
			return existing, false, nil
		}
	}

	book, err := s.bookRepo.FindByID(ctx, cmd.BookID)
	if err != nil || book == nil {
		return nil, false, apperrors.NewNotFoundError("BOOK_NOT_FOUND", "book not found")
	}

	if !book.IsActive {
		return nil, false, apperrors.NewNotFoundError("BOOK_NOT_FOUND", "book not found")
	}

	if !book.SupportsFormat(cmd.Format) {
		return nil, false, apperrors.NewSimpleValidationError("FORMAT_UNAVAILABLE", "this book is not available in "+string(cmd.Format)+" format")
	}

	// There is no payment gateway: every paid book — digital or printed — is
	// sold through the purchase request workflow (POST /v1/purchase-requests,
	// approved by an admin after checking the payment), which creates the
	// order. Ordering directly used to unlock the reader or take a printed
	// copy out of stock without any payment.
	if book.Price > 0 {
		return nil, false, &apperrors.AppError{
			Code:       "PURCHASE_APPROVAL_REQUIRED",
			Message:    "paid books must be requested through the purchase approval workflow",
			HTTPStatus: 402,
		}
	}

	var address *bookshop.ShippingAddress
	if cmd.Format == bookshop.OrderFormatDigital {
		if book.DigitalFileRustFSKey == nil || *book.DigitalFileRustFSKey == "" {
			return nil, false, apperrors.NewConflictError("FILE_NOT_AVAILABLE", "the digital edition has not been uploaded yet")
		}
		// Adding a free book twice returns the copy already in the library.
		if existing, _ := s.orderRepo.FindNonRefundedByStudentAndBook(ctx, cmd.StudentID, cmd.BookID); existing != nil {
			return existing, false, nil
		}
	} else {
		if cmd.ShippingAddress != nil {
			a := *cmd.ShippingAddress
			a.Normalize()
			address = &a
		}
		if problem := address.Problem(); problem != "" {
			return nil, false, apperrors.NewSimpleValidationError("INVALID_SHIPPING_ADDRESS", problem)
		}
	}

	now := time.Now().UTC()
	var idempotencyKey *string
	if cmd.IdempotencyKey != "" {
		idempotencyKey = &cmd.IdempotencyKey
	}

	order := &bookshop.Order{
		ID:              uuid.New(),
		StudentID:       cmd.StudentID,
		BookID:          cmd.BookID,
		Format:          cmd.Format,
		Amount:          book.Price,
		Currency:        book.Currency,
		Status:          bookshop.OrderStatusPlaced,
		IdempotencyKey:  idempotencyKey,
		ShippingAddress: address,
		CreatedAt:       now,
		UpdatedAt:       now,
		BookTitle:       book.Title,
	}

	// The stock decrement and the order row commit together.
	err = s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		if cmd.Format == bookshop.OrderFormatPhysical {
			if err := s.orderRepo.DecrementPhysicalStock(txCtx, cmd.BookID); err != nil {
				if errors.Is(err, bookshop.ErrOutOfStock) {
					return apperrors.NewSimpleValidationError("OUT_OF_STOCK", "this book is out of stock")
				}
				return apperrors.NewInternalError("PLACE_ORDER_FAILED", "failed to reserve a copy")
			}
		}
		if err := s.orderRepo.Create(txCtx, order); err != nil {
			return apperrors.NewInternalError("PLACE_ORDER_FAILED", "failed to create order")
		}
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	return order, true, nil
}

// ListStudentOrders returns all orders for a student.
// Requirements: 20.5
func (s *service) ListStudentOrders(ctx context.Context, cmd ListStudentOrdersCommand) (*OrderListResponse, error) {
	if cmd.Page < 1 {
		cmd.Page = 1
	}
	if cmd.Limit < 1 || cmd.Limit > 100 {
		cmd.Limit = 20
	}

	orders, total, err := s.orderRepo.FindByStudentID(ctx, cmd.StudentID, cmd.Page, cmd.Limit)
	if err != nil {
		return nil, apperrors.NewInternalError("LIST_ORDERS_FAILED", "failed to list orders")
	}

	data := make([]*OrderResponse, 0, len(orders))
	for _, o := range orders {
		data = append(data, toOrderResponse(o))
	}

	meta := pagination.NewMeta(total, cmd.Page, cmd.Limit)

	return &OrderListResponse{
		Data: data,
		Meta: map[string]interface{}{
			"page":        meta.Page,
			"limit":       meta.Limit,
			"total":       meta.Total,
			"total_pages": meta.TotalPages,
		},
	}, nil
}

// ListStudentLibrary returns purchased, non-refunded digital books for a student.
func (s *service) ListStudentLibrary(ctx context.Context, cmd ListStudentOrdersCommand) (*LibraryBookListResponse, error) {
	if cmd.Page < 1 {
		cmd.Page = 1
	}
	if cmd.Limit < 1 || cmd.Limit > 100 {
		cmd.Limit = 20
	}

	// One entry per owned digital book. A book withdrawn from sale stays in
	// the library: the student paid for it and the reader still opens it.
	orders, total, err := s.orderRepo.FindLibrary(ctx, cmd.StudentID, cmd.Page, cmd.Limit)
	if err != nil {
		return nil, apperrors.NewInternalError("LIST_LIBRARY_FAILED", "failed to list library")
	}

	data := make([]*LibraryBookResponse, 0, len(orders))
	for _, order := range orders {
		book, err := s.bookRepo.FindByID(ctx, order.BookID)
		if err != nil || book == nil {
			continue
		}
		response := &LibraryBookResponse{
			BookResponse:   s.toBookResponse(ctx, book),
			OrderID:        order.ID,
			PurchasedAt:    order.CreatedAt,
			ReadingEnabled: book.DigitalFileRustFSKey != nil && *book.DigitalFileRustFSKey != "",
		}
		if bookmark, _ := s.bookmarkRepo.FindByStudentAndBook(ctx, cmd.StudentID, order.BookID); bookmark != nil {
			response.LastPageRead = bookmark.LastPageRead
		}
		data = append(data, response)
	}

	meta := pagination.NewMeta(total, cmd.Page, cmd.Limit)
	return &LibraryBookListResponse{
		Data: data,
		Meta: map[string]interface{}{
			"page":        meta.Page,
			"limit":       meta.Limit,
			"total":       meta.Total,
			"total_pages": meta.TotalPages,
		},
	}, nil
}

// GetStudentBookStatus tells the book page whether the student already owns
// the digital edition, so it can offer "Read now" instead of a purchase.
func (s *service) GetStudentBookStatus(ctx context.Context, studentID, bookID uuid.UUID) (*StudentBookStatusResponse, error) {
	order, err := s.orderRepo.FindNonRefundedByStudentAndBook(ctx, studentID, bookID)
	if err != nil {
		return nil, apperrors.NewInternalError("ACCESS_CHECK_FAILED", "failed to check ownership")
	}
	resp := &StudentBookStatusResponse{BookID: bookID}
	if order != nil && order.Format == bookshop.OrderFormatDigital {
		resp.InLibrary = true
		resp.OrderID = &order.ID
	}
	return resp, nil
}

// refundEligibility decides whether a student may ask to refund an order.
// Rules: paid orders only; not cancelled or refunded; no open request; at
// most 3 requests per order; digital within 7 days of purchase; printed
// before delivery or within 7 days of it.
func (s *service) refundEligibility(ctx context.Context, order *bookshop.Order) (bool, *time.Time, string) {
	if order.Amount <= 0 {
		return false, nil, "Free orders have nothing to refund."
	}
	switch order.Status {
	case bookshop.OrderStatusRefunded:
		return false, nil, "This order has been refunded."
	case bookshop.OrderStatusCancelled:
		return false, nil, "This order was cancelled."
	}
	var deadline *time.Time
	if order.IsPhysical() {
		if order.DeliveredAt != nil {
			d := order.DeliveredAt.Add(RefundWindow)
			deadline = &d
		}
	} else {
		d := order.CreatedAt.Add(RefundWindow)
		deadline = &d
	}
	if deadline != nil && s.now().After(*deadline) {
		return false, deadline, "The 7-day refund window has passed."
	}
	if s.refunds != nil {
		if pending, _ := s.refunds.FindPendingRequestByOrder(ctx, order.ID); pending != nil {
			return false, deadline, "A refund request is already waiting for review."
		}
		if n, err := s.refunds.CountRequestsForOrder(ctx, order.ID); err == nil && n >= maxRefundRequestsPerOrder {
			return false, deadline, "The limit of refund requests for this order has been reached."
		}
	}
	return true, deadline, ""
}

// CheckOrderable reports whether a student can get a book in a format right
// now, and if not, why (shown next to cart items). The book is returned even
// when withdrawn from sale so the cart can still name it.
func (s *service) CheckOrderable(ctx context.Context, studentID, bookID uuid.UUID, format bookshop.OrderFormat) (*BookResponse, string) {
	book, err := s.bookRepo.FindByID(ctx, bookID)
	if err != nil || book == nil {
		return nil, "This book no longer exists."
	}
	resp := s.toBookResponse(ctx, book)
	switch {
	case !book.IsActive:
		return resp, "This book is no longer for sale."
	case !book.SupportsFormat(format):
		return resp, "Not sold in this edition."
	case format == bookshop.OrderFormatDigital && (book.DigitalFileRustFSKey == nil || *book.DigitalFileRustFSKey == ""):
		return resp, "The digital edition isn't available yet."
	case format == bookshop.OrderFormatPhysical && book.PhysicalStock <= 0:
		return resp, "Out of stock."
	}
	if format == bookshop.OrderFormatDigital {
		if owned, _ := s.orderRepo.FindNonRefundedByStudentAndBook(ctx, studentID, bookID); owned != nil {
			return resp, "Already in your library."
		}
	}
	return resp, ""
}

// RequestRefund records a student's refund request for one of their orders.
func (s *service) RequestRefund(ctx context.Context, cmd RequestRefundCommand) (*RefundRequestResponse, error) {
	if s.refunds == nil {
		return nil, apperrors.NewInternalError("REFUNDS_UNAVAILABLE", "refund requests are not available")
	}
	reason := strings.TrimSpace(cmd.Reason)
	if reason == "" || utf8.RuneCountInString(reason) > 2000 {
		return nil, apperrors.NewValidationError([]map[string]string{{"field": "reason", "message": "tell us why (up to 2000 characters)"}})
	}
	order, err := s.orderRepo.FindByID(ctx, cmd.OrderID)
	if err != nil || order == nil || order.StudentID != cmd.StudentID {
		return nil, apperrors.NewNotFoundError("ORDER_NOT_FOUND", "order not found")
	}
	if ok, _, why := s.refundEligibility(ctx, order); !ok {
		return nil, apperrors.NewSimpleValidationError("REFUND_NOT_ALLOWED", why)
	}
	now := s.now().UTC()
	req := &bookshop.RefundRequest{
		ID: uuid.New(), OrderID: order.ID, StudentID: cmd.StudentID, Reason: reason,
		Status: bookshop.RefundRequestPending, CreatedAt: now, UpdatedAt: now,
		BookTitle: order.BookTitle, Amount: order.Amount, Currency: order.Currency, Format: order.Format, OrderStatus: order.Status,
	}
	if err := s.refunds.CreateRequest(ctx, req); err != nil {
		if errors.Is(err, bookshop.ErrRefundRequestExists) {
			return nil, apperrors.NewConflictError("REFUND_REQUEST_EXISTS", "a refund request for this order is already waiting for review")
		}
		return nil, apperrors.NewInternalError("REFUND_REQUEST_FAILED", "failed to save the refund request")
	}
	s.enqueueRefundRequestJob(ctx, "refund_request_received", req)
	s.alertAdminsOfRefundRequest(ctx, req)
	return toRefundRequestResponse(req), nil
}

func (s *service) enqueueRefundRequestJob(ctx context.Context, jobType string, req *bookshop.RefundRequest) {
	if s.jobQueue == nil {
		return
	}
	note := ""
	if req.AdminNote != nil {
		note = *req.AdminNote
	}
	payload, _ := json.Marshal(map[string]string{
		"student_id": req.StudentID.String(), "order_id": req.OrderID.String(), "book_title": req.BookTitle, "note": note,
	})
	if err := s.jobQueue.Enqueue(ctx, notifications.Job{Type: jobType, Payload: payload}); err != nil {
		logger.Error(ctx, "Failed to queue refund request notification", "request_id", req.ID, "error", err)
	}
}

// alertAdminsOfRefundRequest queues an in-app + email alert to every admin.
func (s *service) alertAdminsOfRefundRequest(ctx context.Context, req *bookshop.RefundRequest) {
	if s.jobQueue == nil {
		return
	}
	name := req.StudentName
	if name == "" {
		if full, _ := s.refunds.FindRequestByID(ctx, req.ID); full != nil {
			name = full.StudentName
		}
	}
	payload, _ := json.Marshal(map[string]interface{}{
		"request_id": req.ID, "order_id": req.OrderID, "book_title": req.BookTitle, "student_name": name,
		"amount": req.Amount, "currency": req.Currency, "reason": req.Reason,
	})
	if err := s.jobQueue.Enqueue(ctx, notifications.Job{Type: "refund_request_admin_alert", Payload: payload}); err != nil {
		logger.Error(ctx, "Failed to queue admin refund alert", "request_id", req.ID, "error", err)
	}
}

func (s *service) listRefundRequests(ctx context.Context, status string, studentID *uuid.UUID, page, limit int) (*RefundRequestListResponse, error) {
	if s.refunds == nil {
		return &RefundRequestListResponse{Data: []*RefundRequestResponse{}, Meta: map[string]interface{}{"total": 0}}, nil
	}
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	st := bookshop.RefundRequestStatus(status)
	switch st {
	case "", bookshop.RefundRequestPending, bookshop.RefundRequestApproved, bookshop.RefundRequestRejected:
	default:
		return nil, apperrors.NewSimpleValidationError("INVALID_STATUS", "status must be pending, approved or rejected")
	}
	reqs, total, err := s.refunds.ListRequests(ctx, st, studentID, page, limit)
	if err != nil {
		return nil, apperrors.NewInternalError("LIST_REFUND_REQUESTS_FAILED", "failed to list refund requests")
	}
	data := make([]*RefundRequestResponse, 0, len(reqs))
	for _, r := range reqs {
		data = append(data, toRefundRequestResponse(r))
	}
	meta := pagination.NewMeta(total, page, limit)
	return &RefundRequestListResponse{Data: data, Meta: map[string]interface{}{
		"page": meta.Page, "limit": meta.Limit, "total": meta.Total, "total_pages": meta.TotalPages,
	}}, nil
}

// ListMyRefundRequests lists a student's own refund requests.
func (s *service) ListMyRefundRequests(ctx context.Context, studentID uuid.UUID, page, limit int) (*RefundRequestListResponse, error) {
	return s.listRefundRequests(ctx, "", &studentID, page, limit)
}

// ListRefundRequests lists refund requests for admins.
func (s *service) ListRefundRequests(ctx context.Context, status string, page, limit int) (*RefundRequestListResponse, error) {
	return s.listRefundRequests(ctx, status, nil, page, limit)
}

// ApproveRefundRequest refunds the order of a pending request and records
// how the money was returned, in one transaction with the request locked.
func (s *service) ApproveRefundRequest(ctx context.Context, cmd ReviewRefundRequestCommand) (*RefundResponse, error) {
	if s.refunds == nil {
		return nil, apperrors.NewInternalError("REFUNDS_UNAVAILABLE", "refund requests are not available")
	}
	refundCmd := ProcessRefundCommand{ActorID: cmd.ActorID, ActorName: cmd.ActorName, IPAddress: cmd.IPAddress,
		PayoutMethod: cmd.PayoutMethod, PayoutReference: cmd.PayoutReference, Amount: cmd.Amount, Note: cmd.Note}
	var result refundResult
	err := s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		req, err := s.refunds.LockRequest(txCtx, cmd.RequestID)
		if err != nil || req == nil {
			return apperrors.NewNotFoundError("REFUND_REQUEST_NOT_FOUND", "refund request not found")
		}
		if req.Status != bookshop.RefundRequestPending {
			return apperrors.NewConflictError("REFUND_REQUEST_DECIDED", "this refund request was already "+string(req.Status))
		}
		refundCmd.OrderID = req.OrderID
		refundCmd.request = req
		result, err = s.refundOrderTx(txCtx, refundCmd)
		if err != nil {
			return err
		}
		if result.already {
			return apperrors.NewConflictError("ORDER_ALREADY_REFUNDED", "the order was already refunded")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.finishRefund(ctx, refundCmd, result), nil
}

// RejectRefundRequest declines a pending request with a note to the student.
func (s *service) RejectRefundRequest(ctx context.Context, cmd ReviewRefundRequestCommand) (*RefundRequestResponse, error) {
	if s.refunds == nil {
		return nil, apperrors.NewInternalError("REFUNDS_UNAVAILABLE", "refund requests are not available")
	}
	note := strings.TrimSpace(cmd.Note)
	if note == "" || utf8.RuneCountInString(note) > 2000 {
		return nil, apperrors.NewValidationError([]map[string]string{{"field": "note", "message": "explain the decision to the student (up to 2000 characters)"}})
	}
	var req *bookshop.RefundRequest
	err := s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		var err error
		req, err = s.refunds.LockRequest(txCtx, cmd.RequestID)
		if err != nil || req == nil {
			return apperrors.NewNotFoundError("REFUND_REQUEST_NOT_FOUND", "refund request not found")
		}
		if req.Status != bookshop.RefundRequestPending {
			return apperrors.NewConflictError("REFUND_REQUEST_DECIDED", "this refund request was already "+string(req.Status))
		}
		now := s.now().UTC()
		actor := cmd.ActorID
		req.Status = bookshop.RefundRequestRejected
		req.AdminNote = &note
		req.ReviewedBy = &actor
		req.ReviewedAt = &now
		req.UpdatedAt = now
		return s.refunds.UpdateRequest(txCtx, req)
	})
	if err != nil {
		return nil, err
	}
	if s.audit != nil {
		_ = s.audit.LogAction(ctx, cmd.ActorID, cmd.ActorName, "refund_request_rejected", "refund_request", req.ID,
			map[string]interface{}{"order_id": req.OrderID}, cmd.IPAddress)
	}
	s.enqueueRefundRequestJob(ctx, "refund_request_rejected", req)
	return toRefundRequestResponse(req), nil
}

// ListRefundQueue lists orders an admin can refund (placed, shipped or
// delivered) followed by already refunded ones, with the real student and
// book. The platform records no student refund requests or reasons, so none
// are shown.
func (s *service) ListRefundQueue(ctx context.Context, page, limit int) (*RefundQueueResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	orders, total, err := s.orderRepo.ListByStatuses(ctx, []bookshop.OrderStatus{
		bookshop.OrderStatusPlaced, bookshop.OrderStatusShipped, bookshop.OrderStatusDelivered, bookshop.OrderStatusRefunded,
	}, page, limit)
	if err != nil {
		return nil, apperrors.NewInternalError("LIST_REFUNDS_FAILED", "failed to list refundable orders")
	}
	items := make([]RefundQueueItem, 0, len(orders))
	for _, o := range orders {
		items = append(items, RefundQueueItem{
			ID:          o.ID,
			OrderID:     o.ID,
			Requester:   RefundRequester{ID: o.StudentID, FullName: o.StudentName, Email: o.StudentEmail},
			BookID:      o.BookID,
			BookTitle:   o.BookTitle,
			Format:      o.Format,
			Amount:      o.Amount,
			Currency:    o.Currency,
			OrderStatus: o.Status,
			Refundable:  o.Status != bookshop.OrderStatusRefunded,
			CreatedAt:   o.CreatedAt,
			UpdatedAt:   o.UpdatedAt,
		})
		if s.refunds != nil {
			item := &items[len(items)-1]
			if o.Status == bookshop.OrderStatusRefunded {
				if ref, _ := s.refunds.FindRefundByOrder(ctx, o.ID); ref != nil {
					item.Refund = toOrderRefundResponse(ref)
				}
			} else if req, _ := s.refunds.FindPendingRequestByOrder(ctx, o.ID); req != nil {
				item.PendingRequestID = &req.ID
			}
		}
	}
	meta := pagination.NewMeta(total, page, limit)
	return &RefundQueueResponse{Items: items, Meta: map[string]interface{}{
		"page": meta.Page, "limit": meta.Limit, "total": meta.Total, "total_pages": meta.TotalPages,
	}}, nil
}

// GetStudentOrder returns a single order that belongs to the authenticated student.
func (s *service) GetStudentOrder(ctx context.Context, studentID, orderID uuid.UUID) (*OrderResponse, error) {
	order, err := s.orderRepo.FindByID(ctx, orderID)
	if err != nil || order == nil {
		return nil, apperrors.NewNotFoundError("ORDER_NOT_FOUND", "order not found")
	}

	if order.StudentID != studentID {
		return nil, apperrors.NewForbiddenError("FORBIDDEN", "you do not have access to this order")
	}

	resp := toOrderResponse(order)
	if s.refunds != nil {
		info := &OrderRefundInfo{}
		info.Eligible, info.Deadline, info.BlockedReason = s.refundEligibility(ctx, order)
		if req, _ := s.refunds.LatestRequestByOrder(ctx, order.ID); req != nil {
			info.Request = toRefundRequestResponse(req)
		}
		if ref, _ := s.refunds.FindRefundByOrder(ctx, order.ID); ref != nil {
			info.Refund = toOrderRefundResponse(ref)
		}
		resp.RefundInfo = info
	}
	return resp, nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// enqueueOrderNotification enqueues a notification job for order status changes.
func (s *service) enqueueOrderNotification(ctx context.Context, order *bookshop.Order, jobType string) {
	type orderNotificationPayload struct {
		OrderID        uuid.UUID `json:"order_id"`
		StudentID      uuid.UUID `json:"student_id"`
		BookID         uuid.UUID `json:"book_id"`
		BookTitle      string    `json:"book_title"`
		Format         string    `json:"format"`
		Status         string    `json:"status"`
		TrackingNumber string    `json:"tracking_number,omitempty"`
	}

	title := order.BookTitle
	if title == "" {
		if book, _ := s.bookRepo.FindByID(ctx, order.BookID); book != nil {
			title = book.Title
		}
	}
	tracking := ""
	if order.TrackingNumber != nil {
		tracking = *order.TrackingNumber
	}
	payload, err := json.Marshal(orderNotificationPayload{
		OrderID:        order.ID,
		StudentID:      order.StudentID,
		BookID:         order.BookID,
		BookTitle:      title,
		Format:         string(order.Format),
		Status:         string(order.Status),
		TrackingNumber: tracking,
	})
	if err != nil {
		logger.Error(ctx, "Failed to marshal order notification payload", "order_id", order.ID, "error", err)
		return
	}

	job := notifications.Job{
		Type:    jobType,
		Payload: json.RawMessage(payload),
	}
	if err := s.jobQueue.Enqueue(ctx, job); err != nil {
		logger.Error(ctx, "Failed to enqueue order notification", "order_id", order.ID, "job_type", jobType, "error", err)
	}
}

func (s *service) toBookResponse(ctx context.Context, book *bookshop.Book) *BookResponse {
	response := toBookResponse(book)
	if book.CoverRustFSKey != nil && *book.CoverRustFSKey != "" && s.storage != nil {
		if url, err := s.storage.PresignGetURL(ctx, s.booksBucket, *book.CoverRustFSKey, 15*time.Minute); err == nil {
			response.CoverURL = url
		}
	}
	return response
}

const maxBookPage = 100000

var currencyCode = regexp.MustCompile(`^[A-Z]{3}$`)

func validOrderStatus(st bookshop.OrderStatus) bool {
	switch st {
	case bookshop.OrderStatusPlaced, bookshop.OrderStatusShipped, bookshop.OrderStatusDelivered,
		bookshop.OrderStatusRefunded, bookshop.OrderStatusCancelled:
		return true
	}
	return false
}

// validateBookFields checks the catalogue rules shared by create and update.
func validateBookFields(title, author string, format bookshop.BookFormat, price float64, currency string, stock int) error {
	var problems []map[string]string
	add := func(field, message string) {
		problems = append(problems, map[string]string{"field": field, "message": message})
	}
	if title == "" || utf8.RuneCountInString(title) > 255 {
		add("title", "required, 255 characters or fewer")
	}
	if author == "" || utf8.RuneCountInString(author) > 255 {
		add("author", "required, 255 characters or fewer")
	}
	if !bookshop.ValidBookFormat(format) {
		add("format", "must be digital, physical or both")
	}
	if math.IsNaN(price) || math.IsInf(price, 0) || price < 0 || price > 10_000_000 {
		add("price", "must be between 0 and 10,000,000")
	}
	if !currencyCode.MatchString(currency) {
		add("currency", "must be a three-letter ISO code such as BDT")
	}
	if stock < 0 || stock > 1_000_000 {
		add("physical_stock", "must be between 0 and 1,000,000")
	}
	if len(problems) > 0 {
		return apperrors.NewValidationError(problems)
	}
	return nil
}

// UploadBookFile stores the full digital edition ("file") or the free sample
// ("preview") as a PDF and attaches it to the book. Students only ever get
// short-lived signed URLs for these objects.
func (s *service) UploadBookFile(ctx context.Context, cmd UploadBookFileCommand) (*BookResponse, error) {
	book, err := s.bookRepo.FindByID(ctx, cmd.BookID)
	if err != nil || book == nil {
		return nil, apperrors.NewNotFoundError("BOOK_NOT_FOUND", "book not found")
	}
	limit, prefix := int64(MaxBookFileBytes), "book-files/"
	if cmd.Kind == BookFilePreview {
		limit, prefix = MaxBookPreviewBytes, "book-previews/"
	} else if cmd.Kind != BookFileFull {
		return nil, apperrors.NewSimpleValidationError("INVALID_FILE_KIND", "file kind must be file or preview")
	}
	if cmd.FileSize <= 0 || cmd.FileSize > limit {
		return nil, apperrors.NewSimpleValidationError("INVALID_FILE_SIZE", fmt.Sprintf("the PDF must be at most %d MB", limit/(1024*1024)))
	}
	if !bytes.HasPrefix(cmd.MagicBytes, []byte("%PDF-")) {
		return nil, apperrors.NewSimpleValidationError("INVALID_FILE_TYPE", "the file must be a PDF")
	}
	key := prefix + uuid.New().String() + ".pdf"
	if err := s.storage.PutObject(ctx, s.booksBucket, key, cmd.Reader, cmd.FileSize, "application/pdf"); err != nil {
		return nil, apperrors.NewInternalError("FILE_UPLOAD_FAILED", "failed to store the PDF")
	}
	// The previous object is kept (not deleted) so a mistaken upload can be
	// recovered; it is no longer referenced by the book.
	if cmd.Kind == BookFilePreview {
		book.PreviewRustFSKey = &key
	} else {
		book.DigitalFileRustFSKey = &key
	}
	book.UpdatedAt = time.Now().UTC()
	if err := s.bookRepo.Update(ctx, book); err != nil {
		return nil, apperrors.NewInternalError("UPDATE_BOOK_FAILED", "failed to attach the PDF")
	}
	if s.audit != nil {
		_ = s.audit.LogAction(ctx, cmd.ActorID, "", "book_"+string(cmd.Kind)+"_uploaded", "book", book.ID,
			map[string]interface{}{"size": cmd.FileSize}, cmd.IPAddress)
	}
	return s.toBookResponse(ctx, book), nil
}

func validateCoverUpload(size int64, declared string, magic []byte) (string, error) {
	if size <= 0 || size > 2*1024*1024 {
		return "", apperrors.NewSimpleValidationError("INVALID_COVER_SIZE", "cover image must be greater than 0 and at most 2 MB")
	}
	detected := http.DetectContentType(magic)
	allowed := map[string]bool{"image/jpeg": true, "image/png": true, "image/webp": true}
	if !allowed[detected] {
		return "", apperrors.NewSimpleValidationError("INVALID_COVER_TYPE", "cover image must be JPEG, PNG, or WebP")
	}
	if declared != "" && declared != "application/octet-stream" && !strings.HasPrefix(declared, detected) {
		return "", apperrors.NewSimpleValidationError("CONTENT_TYPE_MISMATCH", "declared content type does not match cover contents")
	}
	return detected, nil
}

func generatedBookCoverKey(fileName string) string {
	ext := strings.ToLower(filepath.Ext(fileName))
	if ext == "" {
		ext = ".bin"
	}
	return fmt.Sprintf("book-covers/%s%s", uuid.New().String(), ext)
}
