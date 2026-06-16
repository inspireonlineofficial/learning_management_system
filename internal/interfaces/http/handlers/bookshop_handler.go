package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	appbookshop "lms-backend/internal/application/bookshop"
	domainbookshop "lms-backend/internal/domain/bookshop"
	"lms-backend/pkg/apperrors"

	"github.com/google/uuid"
)

// BookshopHandler handles HTTP requests for the bookshop bounded context.
type BookshopHandler struct {
	service   appbookshop.Service
	cartStore stateStore
}

// NewBookshopHandler creates a new BookshopHandler.
func NewBookshopHandler(service appbookshop.Service, stores ...stateStore) *BookshopHandler {
	var cartStore stateStore
	if len(stores) > 0 {
		cartStore = stores[0]
	}
	return &BookshopHandler{service: service, cartStore: cartStore}
}

// ─── Public endpoints ─────────────────────────────────────────────────────────

// ListBooks handles GET /v1/bookshop/books (public)
// Returns a paginated, filterable catalog of active books.
// Requirements: 19.1
//
// @Summary      List books
// @Description  Returns a paginated, filterable catalog of active books
// @Tags         bookshop
// @Produce      json
// @Param        page        query  int     false  "Page number"        default(1)
// @Param        limit       query  int     false  "Items per page"     default(20)
// @Param        search      query  string  false  "Search query"
// @Param        subject     query  string  false  "Filter by subject"
// @Param        class_grade query  string  false  "Filter by class grade"
// @Param        format      query  string  false  "Filter by format"
// @Param        min_price   query  number  false  "Minimum price"
// @Param        max_price   query  number  false  "Maximum price"
// @Success      200  {object}  bookshop.BookListResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Router       /v1/bookshop/books [get]
func (h *BookshopHandler) ListBooks(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	page := 1
	limit := 20
	if p := q.Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if l := q.Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	var minPrice, maxPrice *float64
	if v := q.Get("min_price"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			minPrice = &f
		}
	}
	if v := q.Get("max_price"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			maxPrice = &f
		}
	}

	cmd := appbookshop.ListBooksCommand{
		Search:     q.Get("search"),
		Subject:    q.Get("subject"),
		ClassGrade: q.Get("class_grade"),
		Format:     domainbookshop.BookFormat(q.Get("format")),
		MinPrice:   minPrice,
		MaxPrice:   maxPrice,
		Page:       page,
		Limit:      limit,
	}

	result, err := h.service.ListBooks(r.Context(), cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

// GetBookDetail handles GET /v1/bookshop/books/:bookId
func (h *BookshopHandler) GetBookDetail(w http.ResponseWriter, r *http.Request) {
	bookIDStr := r.PathValue("bookId")
	bookID, err := uuid.Parse(bookIDStr)
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid book ID"))
		return
	}

	result, err := h.service.GetBookDetail(r.Context(), bookID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

// GetBookPreview handles GET /v1/bookshop/books/:bookId/preview
// Returns a presigned preview URL (10-min TTL). Requires authentication.
// Requirements: 19.2
//
// @Summary      Get book preview
// @Description  Returns a presigned preview URL (10-min TTL) for the specified book
// @Tags         bookshop
// @Produce      json
// @Param        bookId  path  string  true  "Book ID"
// @Success      200  {object}  bookshop.BookPreviewResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Router       /v1/bookshop/books/{bookId}/preview [get]
func (h *BookshopHandler) GetBookPreview(w http.ResponseWriter, r *http.Request) {
	bookIDStr := r.PathValue("bookId")
	bookID, err := uuid.Parse(bookIDStr)
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid book ID"))
		return
	}

	cmd := appbookshop.GetBookPreviewCommand{BookID: bookID}
	result, err := h.service.GetBookPreview(r.Context(), cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

// ─── Student endpoints ────────────────────────────────────────────────────────

// GetDigitalBookAccess handles GET /v1/student/bookshop/reader/:bookId/access
// Verifies a non-refunded purchase and returns a presigned URL for the full digital file.
// Requirements: 19.3, 19.4
//
// @Summary      Get digital book access
// @Description  Verifies a non-refunded purchase and returns a presigned URL for the full digital file
// @Tags         bookshop
// @Produce      json
// @Param        bookId  path  string  true  "Book ID"
// @Success      200  {object}  bookshop.DigitalBookAccessResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/student/bookshop/reader/{bookId}/access [get]
func (h *BookshopHandler) GetDigitalBookAccess(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	bookIDStr := r.PathValue("bookId")
	bookID, err := uuid.Parse(bookIDStr)
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid book ID"))
		return
	}

	cmd := appbookshop.GetDigitalBookAccessCommand{
		StudentID: userID,
		BookID:    bookID,
	}

	result, err := h.service.GetDigitalBookAccess(r.Context(), cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

// UpsertBookmark handles POST /v1/student/bookshop/reader/:bookId/bookmark
// Persists the student's current reading position.
// Requirements: 19.5
//
// @Summary      Upsert bookmark
// @Description  Persists the student's current reading position for the specified book
// @Tags         bookshop
// @Accept       json
// @Produce      json
// @Param        bookId  path  string  true  "Book ID"
// @Param        body    body  object  true  "Bookmark data"
// @Success      200  {object}  bookshop.BookmarkResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/student/bookshop/reader/{bookId}/bookmark [post]
func (h *BookshopHandler) UpsertBookmark(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	bookIDStr := r.PathValue("bookId")
	bookID, err := uuid.Parse(bookIDStr)
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid book ID"))
		return
	}

	var req struct {
		LastPageRead int `json:"last_page_read"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_JSON", "invalid request body"))
		return
	}
	if req.LastPageRead < 1 {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_PAGE", "last_page_read must be >= 1"))
		return
	}

	cmd := appbookshop.UpsertBookmarkCommand{
		StudentID:    userID,
		BookID:       bookID,
		LastPageRead: req.LastPageRead,
	}

	result, err := h.service.UpsertBookmark(r.Context(), cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

// ListStudentOrders handles GET /v1/student/bookshop/orders
// Returns all orders for the authenticated student.
// Requirements: 20.5
//
// @Summary      List student orders
// @Description  Returns a paginated list of orders for the authenticated student
// @Tags         bookshop
// @Produce      json
// @Param        page   query  int  false  "Page number"     default(1)
// @Param        limit  query  int  false  "Items per page"  default(20)
// @Success      200  {object}  bookshop.OrderListResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/student/bookshop/orders [get]
func (h *BookshopHandler) ListStudentOrders(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	page := 1
	limit := 20
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	cmd := appbookshop.ListStudentOrdersCommand{
		StudentID: userID,
		Page:      page,
		Limit:     limit,
	}

	result, err := h.service.ListStudentOrders(r.Context(), cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

// GetStudentOrder handles GET /v1/student/bookshop/orders/:orderId
func (h *BookshopHandler) GetStudentOrder(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	orderID, err := uuid.Parse(r.PathValue("orderId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid order ID"))
		return
	}

	result, err := h.service.GetStudentOrder(r.Context(), userID, orderID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

func (h *BookshopHandler) GetCart(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	cart, err := h.buildCartResponse(r, userID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, cart)
}

func (h *BookshopHandler) AddCartItem(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	var req struct {
		BookID   string `json:"book_id"`
		Quantity int    `json:"quantity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_JSON", "invalid request body"))
		return
	}
	bookID, err := uuid.Parse(req.BookID)
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_BOOK_ID", "invalid book_id"))
		return
	}
	if req.Quantity < 1 {
		req.Quantity = 1
	}
	if req.Quantity > 99 {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_QUANTITY", "quantity must be 99 or fewer"))
		return
	}
	if _, err := h.service.GetBookDetail(r.Context(), bookID); err != nil {
		writeErrorResponse(w, err)
		return
	}

	cart, err := h.readCart(r.Context(), userID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	cart[bookID] += req.Quantity
	if err := h.writeCart(r.Context(), userID, cart); err != nil {
		writeErrorResponse(w, err)
		return
	}

	response, err := h.buildCartResponse(r, userID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, response)
}

func (h *BookshopHandler) UpdateCartItem(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	bookID, err := uuid.Parse(r.PathValue("itemId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ITEM_ID", "invalid item ID"))
		return
	}
	var req struct {
		Quantity int `json:"quantity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_JSON", "invalid request body"))
		return
	}
	if req.Quantity < 0 || req.Quantity > 99 {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_QUANTITY", "quantity must be between 0 and 99"))
		return
	}

	cart, err := h.readCart(r.Context(), userID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	if req.Quantity == 0 {
		delete(cart, bookID)
	} else {
		cart[bookID] = req.Quantity
	}
	if err := h.writeCart(r.Context(), userID, cart); err != nil {
		writeErrorResponse(w, err)
		return
	}

	response, err := h.buildCartResponse(r, userID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, response)
}

func (h *BookshopHandler) RemoveCartItem(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	bookID, err := uuid.Parse(r.PathValue("itemId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ITEM_ID", "invalid item ID"))
		return
	}
	cart, err := h.readCart(r.Context(), userID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	delete(cart, bookID)
	if err := h.writeCart(r.Context(), userID, cart); err != nil {
		writeErrorResponse(w, err)
		return
	}

	response, err := h.buildCartResponse(r, userID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, response)
}

// PlaceOrder handles POST /v1/bookshop/orders.
func (h *BookshopHandler) PlaceOrder(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	var req struct {
		BookID string `json:"book_id"`
		Format string `json:"format"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_JSON", "invalid request body"))
		return
	}

	order, err := h.placeOrder(r, userID, req.BookID, req.Format, r.Header.Get("Idempotency-Key"))
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusCreated, order)
}

// Checkout handles POST /v1/bookshop/checkout.
func (h *BookshopHandler) Checkout(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	var req struct {
		Items []struct {
			BookID string `json:"book_id"`
			Format string `json:"format"`
		} `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_JSON", "invalid request body"))
		return
	}
	if len(req.Items) == 0 {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("EMPTY_CART", "checkout requires at least one item"))
		return
	}
	if len(req.Items) > 50 {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("CART_TOO_LARGE", "checkout supports at most 50 items"))
		return
	}

	idempotencyKey := r.Header.Get("Idempotency-Key")
	orders := make([]*appbookshop.OrderResponse, 0, len(req.Items))
	for i, item := range req.Items {
		itemKey := idempotencyKey
		if itemKey != "" {
			itemKey = itemKey + ":" + strconv.Itoa(i)
		}
		order, err := h.placeOrder(r, userID, item.BookID, item.Format, itemKey)
		if err != nil {
			writeErrorResponse(w, err)
			return
		}
		orders = append(orders, order)
	}

	if err := h.clearCart(r.Context(), userID); err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusCreated, map[string]interface{}{
		"orders": orders,
	})
}

// ─── Admin endpoints ──────────────────────────────────────────────────────────

// ListAdminBooks handles GET /v1/admin/bookshop/books.
//
// @Summary      List admin books
// @Description  Returns the paginated admin catalog, including books hidden from the public storefront
// @Tags         bookshop
// @Produce      json
// @Param        page   query  int  false  "Page number" default(1)
// @Param        limit  query  int  false  "Items per page" default(20)
// @Success      200  {object}  bookshop.BookListResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/admin/bookshop/books [get]
func (h *BookshopHandler) ListAdminBooks(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.ListAdminBooks(r.Context(), parseBookListCommand(r))
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, result)
}

// ListAdminOrders handles GET /v1/admin/bookshop/orders.
//
// @Summary      List admin book orders
// @Description  Returns the paginated book fulfilment queue for administrators
// @Tags         bookshop
// @Produce      json
// @Param        page   query  int  false  "Page number" default(1)
// @Param        limit  query  int  false  "Items per page" default(20)
// @Success      200  {object}  bookshop.OrderListResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/admin/bookshop/orders [get]
func (h *BookshopHandler) ListAdminOrders(w http.ResponseWriter, r *http.Request) {
	page, limit := parseBookshopPagination(r)
	result, err := h.service.ListAdminOrders(r.Context(), page, limit)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, result)
}

// ListRefunds handles GET /v1/admin/bookshop/refunds.
func (h *BookshopHandler) ListRefunds(w http.ResponseWriter, r *http.Request) {
	page, limit := parseBookshopPagination(r)
	result, err := h.service.ListAdminOrders(r.Context(), page, limit)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	type refundQueueItem struct {
		ID        string                 `json:"id"`
		OrderID   string                 `json:"order_id"`
		Requester map[string]string      `json:"requester"`
		Reason    string                 `json:"reason"`
		Amount    float64                `json:"amount"`
		Currency  string                 `json:"currency"`
		Status    string                 `json:"status"`
		CreatedAt string                 `json:"created_at"`
		Metadata  map[string]interface{} `json:"metadata,omitempty"`
	}

	items := make([]refundQueueItem, 0, len(result.Data))
	for _, order := range result.Data {
		if order.Status == domainbookshop.OrderStatusRefunded || order.Status == domainbookshop.OrderStatusCancelled {
			continue
		}
		items = append(items, refundQueueItem{
			ID:      order.ID.String(),
			OrderID: order.ID.String(),
			Requester: map[string]string{
				"id":        order.StudentID.String(),
				"full_name": "Student",
			},
			Reason:    "Refund eligible book order",
			Amount:    order.Amount,
			Currency:  order.Currency,
			Status:    "pending",
			CreatedAt: order.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			Metadata: map[string]interface{}{
				"book_id": order.BookID,
				"format":  order.Format,
				"status":  order.Status,
			},
		})
	}

	writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"items": items,
		"meta":  result.Meta,
	})
}

// CreateBook handles POST /v1/admin/bookshop/books
// Creates a new book. digital_file_rustfs_key is stored internally, never exposed.
// Requirements: 19.6
//
// @Summary      Create book
// @Description  Creates a new book; digital_file_rustfs_key is stored internally and never exposed
// @Tags         bookshop
// @Accept       json
// @Produce      json
// @Param        body  body  object  true  "Book data"
// @Success      201  {object}  bookshop.BookResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/admin/bookshop/books [post]
func (h *BookshopHandler) CreateBook(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	var req struct {
		Title                string                    `json:"title"`
		Author               string                    `json:"author"`
		Subject              string                    `json:"subject"`
		ClassGrade           string                    `json:"class_grade"`
		Description          string                    `json:"description"`
		Format               domainbookshop.BookFormat `json:"format"`
		Price                float64                   `json:"price"`
		Currency             string                    `json:"currency"`
		PhysicalStock        int                       `json:"physical_stock"`
		DigitalFileRustFSKey string                    `json:"digital_file_rustfs_key"`
		PreviewRustFSKey     string                    `json:"preview_rustfs_key"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_JSON", "invalid request body"))
		return
	}

	if req.Title == "" || req.Author == "" {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("VALIDATION_ERROR", "title and author are required"))
		return
	}

	cmd := appbookshop.CreateBookCommand{
		ActorID:              userID,
		Title:                req.Title,
		Author:               req.Author,
		Subject:              req.Subject,
		ClassGrade:           req.ClassGrade,
		Description:          req.Description,
		Format:               req.Format,
		Price:                req.Price,
		Currency:             req.Currency,
		PhysicalStock:        req.PhysicalStock,
		DigitalFileRustFSKey: req.DigitalFileRustFSKey,
		PreviewRustFSKey:     req.PreviewRustFSKey,
	}

	result, err := h.service.CreateBook(r.Context(), cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusCreated, result)
}

// UpdateBook handles PATCH /v1/admin/bookshop/books/:bookId
// Updates a book. Setting is_active: false removes it from the public catalog.
// Requirements: 19.7
//
// @Summary      Update book
// @Description  Updates an existing book; setting is_active to false removes it from the public catalog
// @Tags         bookshop
// @Accept       json
// @Produce      json
// @Param        bookId  path  string  true  "Book ID"
// @Param        body    body  object  true  "Book update data"
// @Success      200  {object}  bookshop.BookResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/admin/bookshop/books/{bookId} [patch]
func (h *BookshopHandler) UpdateBook(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	bookIDStr := r.PathValue("bookId")
	bookID, err := uuid.Parse(bookIDStr)
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid book ID"))
		return
	}

	var req struct {
		Title         *string  `json:"title"`
		Author        *string  `json:"author"`
		Subject       *string  `json:"subject"`
		ClassGrade    *string  `json:"class_grade"`
		Description   *string  `json:"description"`
		Price         *float64 `json:"price"`
		PhysicalStock *int     `json:"physical_stock"`
		IsActive      *bool    `json:"is_active"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_JSON", "invalid request body"))
		return
	}

	cmd := appbookshop.UpdateBookCommand{
		ActorID:       userID,
		BookID:        bookID,
		Title:         req.Title,
		Author:        req.Author,
		Subject:       req.Subject,
		ClassGrade:    req.ClassGrade,
		Description:   req.Description,
		Price:         req.Price,
		PhysicalStock: req.PhysicalStock,
		IsActive:      req.IsActive,
	}

	result, err := h.service.UpdateBook(r.Context(), cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

func (h *BookshopHandler) UploadBookCover(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	bookID, err := uuid.Parse(r.PathValue("bookId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid book ID"))
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 3*1024*1024)
	if err := r.ParseMultipartForm(3 * 1024 * 1024); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_MULTIPART", "invalid multipart form"))
		return
	}
	file, header, err := r.FormFile("cover")
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("COVER_REQUIRED", "cover file is required"))
		return
	}
	defer file.Close()
	magic := make([]byte, 512)
	n, readErr := io.ReadFull(file, magic)
	if readErr != nil && readErr != io.ErrUnexpectedEOF {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_COVER", "could not read cover file"))
		return
	}
	magic = magic[:n]
	result, err := h.service.UploadBookCover(r.Context(), appbookshop.UploadBookCoverCommand{
		ActorID:    userID,
		BookID:     bookID,
		FileName:   header.Filename,
		FileSize:   header.Size,
		MimeType:   header.Header.Get("Content-Type"),
		MagicBytes: magic,
		Reader:     io.MultiReader(bytes.NewReader(magic), file),
		IPAddress:  requestIP(r),
	})
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, result)
}

// FulfilOrder handles PATCH /v1/admin/bookshop/orders/:orderId
// Updates an order to shipped status with a tracking number.
// Requirements: 20.2
//
// @Summary      Fulfil order
// @Description  Updates an order to shipped status with a tracking number
// @Tags         bookshop
// @Accept       json
// @Produce      json
// @Param        orderId  path  string  true  "Order ID"
// @Param        body     body  object  true  "Fulfilment data"
// @Success      200  {object}  bookshop.OrderResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/admin/bookshop/orders/{orderId} [patch]
func (h *BookshopHandler) FulfilOrder(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	orderIDStr := r.PathValue("orderId")
	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid order ID"))
		return
	}

	var req struct {
		Status         string `json:"status"`
		TrackingNumber string `json:"tracking_number"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_JSON", "invalid request body"))
		return
	}

	if req.Status != "shipped" {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_STATUS", "only 'shipped' status is supported via this endpoint"))
		return
	}
	if req.TrackingNumber == "" {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("VALIDATION_ERROR", "tracking_number is required"))
		return
	}

	cmd := appbookshop.FulfilOrderCommand{
		ActorID:        userID,
		OrderID:        orderID,
		TrackingNumber: req.TrackingNumber,
	}

	result, err := h.service.FulfilOrder(r.Context(), cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

// ProcessRefund handles POST /v1/admin/bookshop/refunds
// Processes a refund. Idempotent via Idempotency-Key header.
// Requirements: 20.3, 20.4
//
// @Summary      Process refund
// @Description  Processes a refund for an order; idempotent via Idempotency-Key header
// @Tags         bookshop
// @Accept       json
// @Produce      json
// @Param        Idempotency-Key  header  string  false  "Idempotency key"
// @Param        body             body    object  true   "Refund data"
// @Success      200  {object}  bookshop.RefundResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/admin/bookshop/refunds [post]
func (h *BookshopHandler) ProcessRefund(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	idempotencyKey := r.Header.Get("Idempotency-Key")

	var req struct {
		OrderID string `json:"order_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_JSON", "invalid request body"))
		return
	}

	orderID, err := uuid.Parse(req.OrderID)
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid order_id"))
		return
	}

	actorName, _ := r.Context().Value("user_name").(string)

	cmd := appbookshop.ProcessRefundCommand{
		ActorID:        userID,
		ActorName:      actorName,
		OrderID:        orderID,
		IdempotencyKey: idempotencyKey,
		IPAddress:      r.RemoteAddr,
	}

	result, err := h.service.ProcessRefund(r.Context(), cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

func parseBookshopPagination(r *http.Request) (int, int) {
	q := r.URL.Query()
	page := 1
	limit := 20
	if p := q.Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if l := q.Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	return page, limit
}

func parseBookListCommand(r *http.Request) appbookshop.ListBooksCommand {
	q := r.URL.Query()
	page, limit := parseBookshopPagination(r)
	var minPrice, maxPrice *float64
	if v := q.Get("min_price"); v != "" {
		if value, err := strconv.ParseFloat(v, 64); err == nil {
			minPrice = &value
		}
	}
	if v := q.Get("max_price"); v != "" {
		if value, err := strconv.ParseFloat(v, 64); err == nil {
			maxPrice = &value
		}
	}
	return appbookshop.ListBooksCommand{
		Search:     q.Get("search"),
		Subject:    q.Get("subject"),
		ClassGrade: q.Get("class_grade"),
		Format:     domainbookshop.BookFormat(q.Get("format")),
		MinPrice:   minPrice,
		MaxPrice:   maxPrice,
		Page:       page,
		Limit:      limit,
	}
}

func (h *BookshopHandler) placeOrder(r *http.Request, studentID uuid.UUID, rawBookID, rawFormat, idempotencyKey string) (*appbookshop.OrderResponse, error) {
	bookID, err := uuid.Parse(rawBookID)
	if err != nil {
		return nil, apperrors.NewSimpleValidationError("INVALID_BOOK_ID", "invalid book_id")
	}

	format := domainbookshop.OrderFormat(strings.TrimSpace(rawFormat))
	if format == "" {
		format = domainbookshop.OrderFormatDigital
	}
	if format != domainbookshop.OrderFormatDigital && format != domainbookshop.OrderFormatPhysical {
		return nil, apperrors.NewSimpleValidationError("INVALID_FORMAT", "format must be digital or physical")
	}

	return h.service.PlaceOrder(r.Context(), appbookshop.PlaceOrderCommand{
		StudentID:      studentID,
		BookID:         bookID,
		Format:         format,
		IdempotencyKey: idempotencyKey,
	})
}

func (h *BookshopHandler) buildCartResponse(r *http.Request, userID uuid.UUID) (map[string]interface{}, error) {
	rawItems, err := h.readCart(r.Context(), userID)
	if err != nil {
		return nil, err
	}

	type cartItem struct {
		ID             string                    `json:"id"`
		Book           *appbookshop.BookResponse `json:"book"`
		Quantity       int                       `json:"quantity"`
		UnitPriceCents int                       `json:"unit_price_cents"`
	}

	items := make([]cartItem, 0, len(rawItems))
	subtotal := 0
	currency := "USD"
	for bookID, quantity := range rawItems {
		book, err := h.service.GetBookDetail(r.Context(), bookID)
		if err != nil {
			continue
		}
		unitPriceCents := int(math.Round(book.Price * 100))
		subtotal += unitPriceCents * quantity
		currency = book.Currency
		items = append(items, cartItem{
			ID:             bookID.String(),
			Book:           book,
			Quantity:       quantity,
			UnitPriceCents: unitPriceCents,
		})
	}

	return map[string]interface{}{
		"id":             userID.String(),
		"items":          items,
		"subtotal_cents": subtotal,
		"total_cents":    subtotal,
		"currency":       currency,
	}, nil
}

const cartTTL = 30 * 24 * time.Hour

func (h *BookshopHandler) readCart(ctx context.Context, userID uuid.UUID) (map[uuid.UUID]int, error) {
	if h.cartStore == nil {
		return nil, apperrors.NewInternalError("CART_STORE_UNAVAILABLE", "bookshop cart store is not configured")
	}
	raw, err := h.cartStore.Get(ctx, cartStoreKey(userID))
	if err != nil {
		if stateStoreMiss(err) {
			return make(map[uuid.UUID]int), nil
		}
		return nil, apperrors.NewInternalError("CART_READ_FAILED", "failed to read bookshop cart")
	}
	var serialized map[string]int
	if err := json.Unmarshal([]byte(raw), &serialized); err != nil {
		return nil, apperrors.NewInternalError("CART_READ_FAILED", "failed to read bookshop cart")
	}
	cart := make(map[uuid.UUID]int, len(serialized))
	for rawBookID, quantity := range serialized {
		bookID, err := uuid.Parse(rawBookID)
		if err != nil || quantity <= 0 {
			continue
		}
		cart[bookID] = quantity
	}
	return cart, nil
}

func (h *BookshopHandler) writeCart(ctx context.Context, userID uuid.UUID, cart map[uuid.UUID]int) error {
	if h.cartStore == nil {
		return apperrors.NewInternalError("CART_STORE_UNAVAILABLE", "bookshop cart store is not configured")
	}
	serialized := make(map[string]int, len(cart))
	for bookID, quantity := range cart {
		if quantity > 0 {
			serialized[bookID.String()] = quantity
		}
	}
	if len(serialized) == 0 {
		return h.clearCart(ctx, userID)
	}
	data, err := json.Marshal(serialized)
	if err != nil {
		return apperrors.NewInternalError("CART_WRITE_FAILED", "failed to write bookshop cart")
	}
	if err := h.cartStore.Set(ctx, cartStoreKey(userID), string(data), cartTTL); err != nil {
		return apperrors.NewInternalError("CART_WRITE_FAILED", "failed to write bookshop cart")
	}
	return nil
}

func (h *BookshopHandler) clearCart(ctx context.Context, userID uuid.UUID) error {
	if h.cartStore == nil {
		return apperrors.NewInternalError("CART_STORE_UNAVAILABLE", "bookshop cart store is not configured")
	}
	if err := h.cartStore.Del(ctx, cartStoreKey(userID)); err != nil {
		return apperrors.NewInternalError("CART_WRITE_FAILED", "failed to clear bookshop cart")
	}
	return nil
}

func cartStoreKey(userID uuid.UUID) string {
	return "bookshop:cart:" + userID.String()
}
