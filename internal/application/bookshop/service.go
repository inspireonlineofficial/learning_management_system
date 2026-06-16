package bookshop

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

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
	ListAdminOrders(ctx context.Context, page, limit int) (*OrderListResponse, error)
	CreateBook(ctx context.Context, cmd CreateBookCommand) (*BookResponse, error)
	UpdateBook(ctx context.Context, cmd UpdateBookCommand) (*BookResponse, error)
	UploadBookCover(ctx context.Context, cmd UploadBookCoverCommand) (*BookResponse, error)
	FulfilOrder(ctx context.Context, cmd FulfilOrderCommand) (*OrderResponse, error)
	ProcessRefund(ctx context.Context, cmd ProcessRefundCommand) (*RefundResponse, error)

	// Order use cases (Requirements: 19.8, 20.1, 20.5)
	PlaceOrder(ctx context.Context, cmd PlaceOrderCommand) (*OrderResponse, error)
	ListStudentOrders(ctx context.Context, cmd ListStudentOrdersCommand) (*OrderListResponse, error)
	GetStudentOrder(ctx context.Context, studentID, orderID uuid.UUID) (*OrderResponse, error)
}

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

type adminOrderRepository interface {
	List(ctx context.Context, page, limit int) ([]*bookshop.Order, int, error)
}

// ListAdminOrders returns a paginated fulfilment queue for administrators.
func (s *service) ListAdminOrders(ctx context.Context, page, limit int) (*OrderListResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	repo, ok := s.orderRepo.(adminOrderRepository)
	if !ok {
		return nil, apperrors.NewInternalError("LIST_ORDERS_FAILED", "failed to list book orders")
	}

	orders, total, err := repo.List(ctx, page, limit)
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
	if order == nil {
		return nil, apperrors.NewForbiddenError("NO_ACCESS", "no valid purchase found for this book")
	}

	if book.DigitalFileRustFSKey == nil || *book.DigitalFileRustFSKey == "" {
		return nil, apperrors.NewInternalError("FILE_NOT_AVAILABLE", "digital file not available")
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

	var digitalKey *string
	if cmd.DigitalFileRustFSKey != "" {
		digitalKey = &cmd.DigitalFileRustFSKey
	}
	var previewKey *string
	if cmd.PreviewRustFSKey != "" {
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
		book.Title = *cmd.Title
	}
	if cmd.Author != nil {
		book.Author = *cmd.Author
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

// FulfilOrder updates an order to shipped status with a tracking number.
// Sends an in-app notification and email to the student.
// Requirements: 20.2
func (s *service) FulfilOrder(ctx context.Context, cmd FulfilOrderCommand) (*OrderResponse, error) {
	order, err := s.orderRepo.FindByID(ctx, cmd.OrderID)
	if err != nil || order == nil {
		return nil, apperrors.NewNotFoundError("ORDER_NOT_FOUND", "order not found")
	}

	if order.Status != bookshop.OrderStatusPlaced {
		return nil, apperrors.NewSimpleValidationError("INVALID_STATUS", "order cannot be fulfilled in its current status")
	}

	order.Status = bookshop.OrderStatusShipped
	order.TrackingNumber = &cmd.TrackingNumber
	order.UpdatedAt = time.Now().UTC()

	if err := s.orderRepo.Update(ctx, order); err != nil {
		return nil, apperrors.NewInternalError("FULFIL_ORDER_FAILED", "failed to update order status")
	}

	// Enqueue notification to student
	s.enqueueOrderNotification(ctx, order, "order_shipped")

	logger.Info(ctx, "Order fulfilled", "order_id", order.ID, "tracking_number", cmd.TrackingNumber)
	return toOrderResponse(order), nil
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

	order, err := s.orderRepo.FindByID(ctx, cmd.OrderID)
	if err != nil || order == nil {
		return nil, apperrors.NewNotFoundError("ORDER_NOT_FOUND", "order not found")
	}

	if order.Status == bookshop.OrderStatusRefunded {
		// Already refunded — return idempotent response
		return &RefundResponse{
			OrderID:       order.ID,
			Status:        "refunded",
			StockRestored: order.Format == bookshop.OrderFormatPhysical,
			AccessRevoked: order.Format == bookshop.OrderFormatDigital,
			ProcessedAt:   order.UpdatedAt,
		}, nil
	}

	if order.Status == bookshop.OrderStatusCancelled {
		return nil, apperrors.NewSimpleValidationError("INVALID_STATUS", "cancelled orders cannot be refunded")
	}

	stockRestored := false
	accessRevoked := false

	// Restore physical stock if applicable
	if order.Format == bookshop.OrderFormatPhysical {
		if err := s.orderRepo.IncrementPhysicalStock(ctx, order.BookID); err != nil {
			return nil, apperrors.NewInternalError("STOCK_RESTORE_FAILED", "failed to restore physical stock")
		}
		stockRestored = true
	}

	// Revoke digital access (mark order as refunded — access check uses order status)
	if order.Format == bookshop.OrderFormatDigital {
		accessRevoked = true
	}

	order.Status = bookshop.OrderStatusRefunded
	order.UpdatedAt = time.Now().UTC()

	if err := s.orderRepo.Update(ctx, order); err != nil {
		return nil, apperrors.NewInternalError("REFUND_FAILED", "failed to update order status")
	}

	// Record audit log
	if s.audit != nil {
		_ = s.audit.LogAction(ctx, cmd.ActorID, cmd.ActorName, "refund_issued", "order", order.ID,
			map[string]interface{}{
				"book_id":        order.BookID,
				"student_id":     order.StudentID,
				"format":         order.Format,
				"amount":         order.Amount,
				"stock_restored": stockRestored,
				"access_revoked": accessRevoked,
			},
			cmd.IPAddress,
		)
	}

	now := time.Now().UTC()
	resp := &RefundResponse{
		OrderID:       order.ID,
		Status:        "refunded",
		StockRestored: stockRestored,
		AccessRevoked: accessRevoked,
		ProcessedAt:   now,
	}

	// Cache the response for idempotency
	if cmd.IdempotencyKey != "" {
		if data, err := json.Marshal(resp); err == nil {
			_ = s.idempotency.Set(ctx, "refund:"+cmd.IdempotencyKey, string(data))
		}
	}

	logger.Info(ctx, "Refund processed", "order_id", order.ID, "actor_id", cmd.ActorID)
	return resp, nil
}

// ─── Order Use Cases ──────────────────────────────────────────────────────────

// PlaceOrder creates a new book order.
// Decrements physical_stock atomically in the same transaction.
// Rejects if stock = 0 (HTTP 422).
// Sends a receipt email.
// Requirements: 19.8, 20.1
func (s *service) PlaceOrder(ctx context.Context, cmd PlaceOrderCommand) (*OrderResponse, error) {
	// Idempotency check
	if cmd.IdempotencyKey != "" {
		existing, err := s.orderRepo.FindByIdempotencyKey(ctx, cmd.IdempotencyKey)
		if err == nil && existing != nil {
			return toOrderResponse(existing), nil
		}
	}

	book, err := s.bookRepo.FindByID(ctx, cmd.BookID)
	if err != nil || book == nil {
		return nil, apperrors.NewNotFoundError("BOOK_NOT_FOUND", "book not found")
	}

	if !book.IsActive {
		return nil, apperrors.NewNotFoundError("BOOK_NOT_FOUND", "book not found")
	}

	// Validate format availability
	if cmd.Format == bookshop.OrderFormatPhysical && book.Format == bookshop.BookFormatDigital {
		return nil, apperrors.NewSimpleValidationError("FORMAT_UNAVAILABLE", "this book is not available in physical format")
	}
	if cmd.Format == bookshop.OrderFormatDigital && book.Format == bookshop.BookFormatPhysical {
		return nil, apperrors.NewSimpleValidationError("FORMAT_UNAVAILABLE", "this book is not available in digital format")
	}

	// Check physical stock before attempting decrement
	if cmd.Format == bookshop.OrderFormatPhysical {
		if book.PhysicalStock <= 0 {
			return nil, apperrors.NewSimpleValidationError("OUT_OF_STOCK", "this book is out of stock")
		}

		// Atomically decrement stock
		if err := s.orderRepo.DecrementPhysicalStock(ctx, cmd.BookID); err != nil {
			return nil, apperrors.NewSimpleValidationError("OUT_OF_STOCK", "this book is out of stock")
		}
	}

	now := time.Now().UTC()
	var idempotencyKey *string
	if cmd.IdempotencyKey != "" {
		idempotencyKey = &cmd.IdempotencyKey
	}

	order := &bookshop.Order{
		ID:             uuid.New(),
		StudentID:      cmd.StudentID,
		BookID:         cmd.BookID,
		Format:         cmd.Format,
		Amount:         book.Price,
		Currency:       book.Currency,
		Status:         bookshop.OrderStatusPlaced,
		IdempotencyKey: idempotencyKey,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.orderRepo.Create(ctx, order); err != nil {
		// If order creation fails after stock decrement, restore stock
		if cmd.Format == bookshop.OrderFormatPhysical {
			_ = s.orderRepo.IncrementPhysicalStock(ctx, cmd.BookID)
		}
		return nil, apperrors.NewInternalError("PLACE_ORDER_FAILED", "failed to create order")
	}

	// Enqueue receipt email
	s.enqueueOrderNotification(ctx, order, "order_placed")

	logger.Info(ctx, "Order placed", "order_id", order.ID, "student_id", cmd.StudentID, "book_id", cmd.BookID)
	return toOrderResponse(order), nil
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

// GetStudentOrder returns a single order that belongs to the authenticated student.
func (s *service) GetStudentOrder(ctx context.Context, studentID, orderID uuid.UUID) (*OrderResponse, error) {
	order, err := s.orderRepo.FindByID(ctx, orderID)
	if err != nil || order == nil {
		return nil, apperrors.NewNotFoundError("ORDER_NOT_FOUND", "order not found")
	}

	if order.StudentID != studentID {
		return nil, apperrors.NewForbiddenError("FORBIDDEN", "you do not have access to this order")
	}

	return toOrderResponse(order), nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// enqueueOrderNotification enqueues a notification job for order status changes.
func (s *service) enqueueOrderNotification(ctx context.Context, order *bookshop.Order, jobType string) {
	type orderNotificationPayload struct {
		OrderID   uuid.UUID `json:"order_id"`
		StudentID uuid.UUID `json:"student_id"`
		BookID    uuid.UUID `json:"book_id"`
		Status    string    `json:"status"`
	}

	payload, err := json.Marshal(orderNotificationPayload{
		OrderID:   order.ID,
		StudentID: order.StudentID,
		BookID:    order.BookID,
		Status:    string(order.Status),
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
