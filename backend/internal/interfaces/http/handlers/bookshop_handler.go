package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"

	appbookshop "lms-backend/internal/application/bookshop"
	domainbookshop "lms-backend/internal/domain/bookshop"
	"lms-backend/pkg/apperrors"

	"github.com/google/uuid"
)

// BookshopHandler handles HTTP requests for the bookshop bounded context.
type BookshopHandler struct {
	service   appbookshop.Service
	cartStore stateStore
	purchases PurchaseRequester
}

// SetPurchaseRequester lets cart checkout turn paid items into purchase requests.
func (h *BookshopHandler) SetPurchaseRequester(p PurchaseRequester) { h.purchases = p }

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
	cmd, err := parseBookListCommand(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
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

// ListStudentLibrary handles GET /v1/student/bookshop/library.
func (h *BookshopHandler) ListStudentLibrary(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	page, limit := parseBookshopPagination(r)
	result, err := h.service.ListStudentLibrary(r.Context(), appbookshop.ListStudentOrdersCommand{
		StudentID: userID,
		Page:      page,
		Limit:     limit,
	})
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

// PlaceOrder handles POST /v1/bookshop/orders.
func (h *BookshopHandler) PlaceOrder(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	var req orderItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_JSON", "invalid request body"))
		return
	}

	cmd, err := buildPlaceOrderCommand(userID, req, r.Header.Get("Idempotency-Key"))
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	order, err := h.service.PlaceOrder(r.Context(), cmd)
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
		Items           []orderItemRequest              `json:"items"`
		ShippingAddress *domainbookshop.ShippingAddress `json:"shipping_address"`
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
	cmds := make([]appbookshop.PlaceOrderCommand, 0, len(req.Items))
	for i, item := range req.Items {
		itemKey := idempotencyKey
		if itemKey != "" {
			itemKey = itemKey + ":" + strconv.Itoa(i)
		}
		if item.ShippingAddress == nil {
			item.ShippingAddress = req.ShippingAddress
		}
		cmd, err := buildPlaceOrderCommand(userID, item, itemKey)
		if err != nil {
			writeErrorResponse(w, err)
			return
		}
		cmds = append(cmds, cmd)
	}
	// All items succeed or none do.
	orders, err := h.service.PlaceOrders(r.Context(), cmds)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	// Remove only what was ordered; it used to empty the whole cart.
	ordered := make([]string, 0, len(cmds))
	for _, c := range cmds {
		ordered = append(ordered, cartEntry{BookID: c.BookID, Format: c.Format}.id())
	}
	if h.cartStore != nil {
		_ = h.removeCartEntries(r.Context(), userID, ordered...)
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
	cmd, err := parseBookListCommand(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	result, err := h.service.ListAdminBooks(r.Context(), cmd)
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
	result, err := h.service.ListAdminOrdersByStatus(r.Context(), r.URL.Query().Get("status"), page, limit)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, result)
}

// ListRefunds handles GET /v1/admin/bookshop/refunds.
// It lists real orders an admin can refund, plus already refunded ones.
// No student refund requests or reasons exist in the system, so none are
// invented here.
func (h *BookshopHandler) ListRefunds(w http.ResponseWriter, r *http.Request) {
	page, limit := parseBookshopPagination(r)
	result, err := h.service.ListRefundQueue(r.Context(), page, limit)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, result)
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
		Title         *string                    `json:"title"`
		Author        *string                    `json:"author"`
		Subject       *string                    `json:"subject"`
		ClassGrade    *string                    `json:"class_grade"`
		Description   *string                    `json:"description"`
		Format        *domainbookshop.BookFormat `json:"format"`
		Currency      *string                    `json:"currency"`
		Price         *float64                   `json:"price"`
		PhysicalStock *int                       `json:"physical_stock"`
		IsActive      *bool                      `json:"is_active"`
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
		Format:        req.Format,
		Currency:      req.Currency,
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
		file, header, err = r.FormFile("file")
	}
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

// UploadBookFile handles POST /v1/admin/bookshop/books/{bookId}/file (the full
// digital edition) and /preview (a free sample). Field "file", PDF only.
func (h *BookshopHandler) UploadBookFile(kind appbookshop.BookFileKind) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		limit := int64(appbookshop.MaxBookFileBytes)
		if kind == appbookshop.BookFilePreview {
			limit = appbookshop.MaxBookPreviewBytes
		}
		r.Body = http.MaxBytesReader(w, r.Body, limit+1024*1024)
		// Parts beyond 8 MB spill to temporary files instead of memory.
		if err := r.ParseMultipartForm(8 * 1024 * 1024); err != nil {
			writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_MULTIPART", "invalid multipart form or file too large"))
			return
		}
		defer func() {
			if r.MultipartForm != nil {
				_ = r.MultipartForm.RemoveAll()
			}
		}()
		file, header, err := r.FormFile("file")
		if err != nil {
			writeErrorResponse(w, apperrors.NewSimpleValidationError("FILE_REQUIRED", "a PDF file is required in the \"file\" field"))
			return
		}
		defer file.Close()
		magic := make([]byte, 512)
		n, readErr := io.ReadFull(file, magic)
		if readErr != nil && readErr != io.ErrUnexpectedEOF {
			writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_FILE", "could not read the file"))
			return
		}
		magic = magic[:n]
		result, err := h.service.UploadBookFile(r.Context(), appbookshop.UploadBookFileCommand{
			ActorID:    userID,
			BookID:     bookID,
			Kind:       kind,
			FileSize:   header.Size,
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
}

// GetAdminBook handles GET /v1/admin/bookshop/books/{bookId}.
func (h *BookshopHandler) GetAdminBook(w http.ResponseWriter, r *http.Request) {
	bookID, err := uuid.Parse(r.PathValue("bookId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid book ID"))
		return
	}
	result, err := h.service.GetAdminBook(r.Context(), bookID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, result)
}

// GetStudentBookStatus handles GET /v1/student/bookshop/books/{bookId}/status.
func (h *BookshopHandler) GetStudentBookStatus(w http.ResponseWriter, r *http.Request) {
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
	result, err := h.service.GetStudentBookStatus(r.Context(), userID, bookID)
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

	switch req.Status {
	case "shipped", "delivered", "cancelled":
	default:
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_STATUS", "status must be shipped, delivered or cancelled"))
		return
	}

	cmd := appbookshop.FulfilOrderCommand{
		ActorID:        userID,
		OrderID:        orderID,
		Status:         req.Status,
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
		OrderID         string   `json:"order_id"`
		Action          string   `json:"action"`
		PayoutMethod    string   `json:"payout_method"`
		PayoutReference string   `json:"payout_reference"`
		Amount          *float64 `json:"amount"`
		Note            string   `json:"note"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_JSON", "invalid request body"))
		return
	}
	// This endpoint only issues refunds. The old admin page also sent
	// action "reject" here, which silently refunded the order.
	if req.Action != "" && req.Action != "refund" && req.Action != "approve" {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ACTION", "this endpoint only issues refunds; omit action or use \"refund\""))
		return
	}

	orderID, err := uuid.Parse(req.OrderID)
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid order_id"))
		return
	}

	actorName, _ := r.Context().Value("user_name").(string)

	cmd := appbookshop.ProcessRefundCommand{
		ActorID:         userID,
		ActorName:       actorName,
		OrderID:         orderID,
		IdempotencyKey:  idempotencyKey,
		IPAddress:       requestIP(r),
		PayoutMethod:    req.PayoutMethod,
		PayoutReference: req.PayoutReference,
		Amount:          req.Amount,
		Note:            req.Note,
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

func parseBookListCommand(r *http.Request) (appbookshop.ListBooksCommand, error) {
	q := r.URL.Query()
	page, limit := parseBookshopPagination(r)
	cmd := appbookshop.ListBooksCommand{
		Search:     strings.TrimSpace(q.Get("search")),
		Subject:    q.Get("subject"),
		ClassGrade: q.Get("class_grade"),
		Format:     domainbookshop.BookFormat(q.Get("format")),
		Sort:       q.Get("sort"),
		Page:       page,
		Limit:      limit,
	}
	if len(cmd.Search) > 200 {
		return cmd, apperrors.NewSimpleValidationError("INVALID_SEARCH", "search must be 200 characters or fewer")
	}
	if cmd.Format != "" && !domainbookshop.ValidBookFormat(cmd.Format) {
		return cmd, apperrors.NewSimpleValidationError("INVALID_FORMAT", "format must be digital, physical or both")
	}
	switch cmd.Sort {
	case "", "newest", "price_asc", "price_desc", "title":
	default:
		return cmd, apperrors.NewSimpleValidationError("INVALID_SORT", "sort must be newest, price_asc, price_desc or title")
	}
	for name, dst := range map[string]**float64{"min_price": &cmd.MinPrice, "max_price": &cmd.MaxPrice} {
		if v := q.Get(name); v != "" {
			value, err := strconv.ParseFloat(v, 64)
			if err != nil || value < 0 || math.IsNaN(value) || math.IsInf(value, 0) {
				return cmd, apperrors.NewSimpleValidationError("INVALID_PRICE", name+" must be a non-negative number")
			}
			*dst = &value
		}
	}
	return cmd, nil
}

type orderItemRequest struct {
	BookID          string                          `json:"book_id"`
	Format          string                          `json:"format"`
	ShippingAddress *domainbookshop.ShippingAddress `json:"shipping_address"`
}

func buildPlaceOrderCommand(studentID uuid.UUID, req orderItemRequest, idempotencyKey string) (appbookshop.PlaceOrderCommand, error) {
	bookID, err := uuid.Parse(req.BookID)
	if err != nil {
		return appbookshop.PlaceOrderCommand{}, apperrors.NewSimpleValidationError("INVALID_BOOK_ID", "invalid book_id")
	}

	format := domainbookshop.OrderFormat(strings.TrimSpace(req.Format))
	if format == "" {
		format = domainbookshop.OrderFormatDigital
	}
	if format != domainbookshop.OrderFormatDigital && format != domainbookshop.OrderFormatPhysical {
		return appbookshop.PlaceOrderCommand{}, apperrors.NewSimpleValidationError("INVALID_FORMAT", "format must be digital or physical")
	}
	if len(idempotencyKey) > 200 {
		return appbookshop.PlaceOrderCommand{}, apperrors.NewSimpleValidationError("INVALID_IDEMPOTENCY_KEY", "Idempotency-Key must be 200 characters or fewer")
	}

	return appbookshop.PlaceOrderCommand{
		StudentID:       studentID,
		BookID:          bookID,
		Format:          format,
		IdempotencyKey:  idempotencyKey,
		ShippingAddress: req.ShippingAddress,
	}, nil
}

func cartStoreKey(userID uuid.UUID) string {
	return "bookshop:cart:" + userID.String()
}

// RequestRefund handles POST /v1/student/bookshop/orders/{orderId}/refund-request.
func (h *BookshopHandler) RequestRefund(w http.ResponseWriter, r *http.Request) {
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
	var req struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64*1024)).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_JSON", "invalid request body"))
		return
	}
	result, err := h.service.RequestRefund(r.Context(), appbookshop.RequestRefundCommand{StudentID: userID, OrderID: orderID, Reason: req.Reason})
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusCreated, result)
}

// ListMyRefundRequests handles GET /v1/student/bookshop/refund-requests.
func (h *BookshopHandler) ListMyRefundRequests(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	page, limit := parseBookshopPagination(r)
	result, err := h.service.ListMyRefundRequests(r.Context(), userID, page, limit)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, result)
}

// ListRefundRequests handles GET /v1/admin/bookshop/refund-requests.
func (h *BookshopHandler) ListRefundRequests(w http.ResponseWriter, r *http.Request) {
	page, limit := parseBookshopPagination(r)
	result, err := h.service.ListRefundRequests(r.Context(), r.URL.Query().Get("status"), page, limit)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, result)
}

func (h *BookshopHandler) reviewRefundCommand(w http.ResponseWriter, r *http.Request) (appbookshop.ReviewRefundRequestCommand, bool) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return appbookshop.ReviewRefundRequestCommand{}, false
	}
	requestID, err := uuid.Parse(r.PathValue("requestId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid refund request ID"))
		return appbookshop.ReviewRefundRequestCommand{}, false
	}
	var req struct {
		PayoutMethod    string   `json:"payout_method"`
		PayoutReference string   `json:"payout_reference"`
		Amount          *float64 `json:"amount"`
		Note            string   `json:"note"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64*1024)).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_JSON", "invalid request body"))
		return appbookshop.ReviewRefundRequestCommand{}, false
	}
	return appbookshop.ReviewRefundRequestCommand{
		ActorID: userID, RequestID: requestID, PayoutMethod: req.PayoutMethod, PayoutReference: req.PayoutReference,
		Amount: req.Amount, Note: req.Note, IPAddress: requestIP(r),
	}, true
}

// ApproveRefundRequest handles POST /v1/admin/bookshop/refund-requests/{requestId}/approve.
func (h *BookshopHandler) ApproveRefundRequest(w http.ResponseWriter, r *http.Request) {
	cmd, ok := h.reviewRefundCommand(w, r)
	if !ok {
		return
	}
	result, err := h.service.ApproveRefundRequest(r.Context(), cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, result)
}

// RejectRefundRequest handles POST /v1/admin/bookshop/refund-requests/{requestId}/reject.
func (h *BookshopHandler) RejectRefundRequest(w http.ResponseWriter, r *http.Request) {
	cmd, ok := h.reviewRefundCommand(w, r)
	if !ok {
		return
	}
	result, err := h.service.RejectRefundRequest(r.Context(), cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, result)
}
