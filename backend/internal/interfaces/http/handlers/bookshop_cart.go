package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	appbookshop "lms-backend/internal/application/bookshop"
	apppayments "lms-backend/internal/application/payments"
	domainbookshop "lms-backend/internal/domain/bookshop"
	domainpayments "lms-backend/internal/domain/payments"
	"lms-backend/pkg/apperrors"

	"github.com/google/uuid"
)

// PurchaseRequester creates purchase requests (implemented by the payments service).
type PurchaseRequester interface {
	CreatePurchaseRequest(ctx context.Context, cmd apppayments.CreatePurchaseRequestCommand) (*apppayments.PurchaseRequestItem, error)
}

// The cart holds one entry per book and edition, one copy each (every order
// is a single copy). It used to key items by book only, accept quantities up
// to 99 that checkout ignored, and return items in random order.
const (
	cartTTL      = 30 * 24 * time.Hour
	maxCartItems = 20
)

type cartEntry struct {
	BookID  uuid.UUID                  `json:"book_id"`
	Format  domainbookshop.OrderFormat `json:"format"`
	AddedAt time.Time                  `json:"added_at"`
}

func (e cartEntry) id() string { return e.BookID.String() + ":" + string(e.Format) }

type storedCart struct {
	Version int         `json:"v"`
	Items   []cartEntry `json:"items"`
}

func parseCartItemID(raw string) (uuid.UUID, domainbookshop.OrderFormat, error) {
	idPart, formatPart, _ := strings.Cut(raw, ":")
	bookID, err := uuid.Parse(idPart)
	if err != nil {
		return uuid.Nil, "", apperrors.NewSimpleValidationError("INVALID_ITEM_ID", "invalid cart item")
	}
	return bookID, domainbookshop.OrderFormat(formatPart), nil
}

// cartHashStore is the Redis subset the cart needs. Each cart is a hash
// (field "<book_id>:<format>", value = time added), so adds and removes are
// atomic per item. It used to be one JSON value rewritten on every change,
// and concurrent adds (two tabs, a double click) silently lost items.
type cartHashStore interface {
	HSetNX(ctx context.Context, key, field string, value interface{}) (bool, error)
	HDel(ctx context.Context, key string, fields ...string) error
	HGetAll(ctx context.Context, key string) (map[string]string, error)
	HLen(ctx context.Context, key string) (int64, error)
	Expire(ctx context.Context, key string, ttl time.Duration) error
}

func cartHashKey(userID uuid.UUID) string { return "bookshop:cart:v2:" + userID.String() }

func (h *BookshopHandler) cartHash() (cartHashStore, error) {
	store, ok := h.cartStore.(cartHashStore)
	if !ok || h.cartStore == nil {
		return nil, apperrors.NewInternalError("CART_STORE_UNAVAILABLE", "bookshop cart store is not configured")
	}
	return store, nil
}

func (h *BookshopHandler) readCart(ctx context.Context, userID uuid.UUID) ([]cartEntry, error) {
	store, err := h.cartHash()
	if err != nil {
		return nil, err
	}
	h.migrateLegacyCart(ctx, store, userID)
	fields, err := store.HGetAll(ctx, cartHashKey(userID))
	if err != nil {
		return nil, apperrors.NewInternalError("CART_READ_FAILED", "failed to read bookshop cart")
	}
	items := make([]cartEntry, 0, len(fields))
	for field, added := range fields {
		bookID, format, err := parseCartItemID(field)
		if err != nil {
			continue
		}
		nanos, _ := strconv.ParseInt(added, 10, 64)
		items = append(items, cartEntry{BookID: bookID, Format: format, AddedAt: time.Unix(0, nanos).UTC()})
	}
	return items, nil
}

// migrateLegacyCart moves a cart stored in the old single-value format.
func (h *BookshopHandler) migrateLegacyCart(ctx context.Context, store cartHashStore, userID uuid.UUID) {
	raw, err := h.cartStore.Get(ctx, cartStoreKey(userID))
	if err != nil || raw == "" {
		return
	}
	var v2 storedCart
	var ids []cartEntry
	if json.Unmarshal([]byte(raw), &v2) == nil && v2.Version == 2 {
		ids = v2.Items
	} else {
		var legacy map[string]int
		if json.Unmarshal([]byte(raw), &legacy) == nil {
			for id := range legacy {
				if bookID, err := uuid.Parse(id); err == nil {
					ids = append(ids, cartEntry{BookID: bookID, AddedAt: time.Now().UTC()})
				}
			}
		}
	}
	for _, e := range ids {
		format := e.Format
		if format == "" {
			book, _ := h.service.CheckOrderable(ctx, userID, e.BookID, domainbookshop.OrderFormatDigital)
			format = defaultFormat(book)
		}
		_, _ = store.HSetNX(ctx, cartHashKey(userID), cartEntry{BookID: e.BookID, Format: format}.id(), e.AddedAt.UnixNano())
	}
	_ = store.Expire(ctx, cartHashKey(userID), cartTTL)
	_ = h.cartStore.Del(ctx, cartStoreKey(userID))
}

// addCartEntry adds one item atomically; false when it was already there.
func (h *BookshopHandler) addCartEntry(ctx context.Context, userID uuid.UUID, e cartEntry) (bool, error) {
	store, err := h.cartHash()
	if err != nil {
		return false, err
	}
	key := cartHashKey(userID)
	added, err := store.HSetNX(ctx, key, e.id(), e.AddedAt.UnixNano())
	if err != nil {
		return false, apperrors.NewInternalError("CART_WRITE_FAILED", "failed to update bookshop cart")
	}
	if added {
		if n, err := store.HLen(ctx, key); err == nil && n > maxCartItems {
			_ = store.HDel(ctx, key, e.id())
			return false, apperrors.NewSimpleValidationError("CART_FULL", fmt.Sprintf("the cart holds up to %d items", maxCartItems))
		}
	}
	_ = store.Expire(ctx, key, cartTTL)
	return added, nil
}

func (h *BookshopHandler) removeCartEntries(ctx context.Context, userID uuid.UUID, ids ...string) error {
	if len(ids) == 0 {
		return nil
	}
	store, err := h.cartHash()
	if err != nil {
		return err
	}
	if err := store.HDel(ctx, cartHashKey(userID), ids...); err != nil {
		return apperrors.NewInternalError("CART_WRITE_FAILED", "failed to update bookshop cart")
	}
	return nil
}

type cartItemResponse struct {
	ID             string                     `json:"id"`
	Book           *appbookshop.BookResponse  `json:"book"`
	Format         domainbookshop.OrderFormat `json:"format"`
	Quantity       int                        `json:"quantity"`
	UnitPriceCents int                        `json:"unit_price_cents"`
	Free           bool                       `json:"free"`
	Available      bool                       `json:"available"`
	Problem        string                     `json:"problem,omitempty"`
	AddedAt        time.Time                  `json:"added_at"`
}

type cartResponse struct {
	ID            string             `json:"id"`
	Items         []cartItemResponse `json:"items"`
	SubtotalCents int                `json:"subtotal_cents"`
	TotalCents    int                `json:"total_cents"`
	Currency      string             `json:"currency"`
	NeedsAddress  bool               `json:"needs_address"`
	HasPaidItems  bool               `json:"has_paid_items"`
}

// defaultFormat picks an edition for legacy entries that had none.
func defaultFormat(book *appbookshop.BookResponse) domainbookshop.OrderFormat {
	if book != nil && book.Format == domainbookshop.BookFormatPhysical {
		return domainbookshop.OrderFormatPhysical
	}
	return domainbookshop.OrderFormatDigital
}

func (h *BookshopHandler) buildCart(ctx context.Context, userID uuid.UUID, entries []cartEntry) cartResponse {
	resp := cartResponse{ID: userID.String(), Items: []cartItemResponse{}, Currency: "BDT"}
	sort.SliceStable(entries, func(i, j int) bool { return entries[i].AddedAt.Before(entries[j].AddedAt) })
	for _, e := range entries {
		format := e.Format
		if format == "" {
			book, _ := h.service.CheckOrderable(ctx, userID, e.BookID, domainbookshop.OrderFormatDigital)
			format = defaultFormat(book)
		}
		book, problem := h.service.CheckOrderable(ctx, userID, e.BookID, format)
		if book == nil {
			continue // deleted book: drop silently
		}
		item := cartItemResponse{
			ID: cartEntry{BookID: e.BookID, Format: format}.id(), Book: book, Format: format, Quantity: 1,
			UnitPriceCents: int(math.Round(book.Price * 100)), Free: book.Price == 0,
			Available: problem == "", Problem: problem, AddedAt: e.AddedAt,
		}
		if item.Available {
			resp.SubtotalCents += item.UnitPriceCents
			resp.Currency = book.Currency
			if format == domainbookshop.OrderFormatPhysical {
				resp.NeedsAddress = true
			}
			if !item.Free {
				resp.HasPaidItems = true
			}
		}
		resp.Items = append(resp.Items, item)
	}
	resp.TotalCents = resp.SubtotalCents
	return resp
}

// GetCart handles GET /v1/bookshop/cart.
func (h *BookshopHandler) GetCart(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	entries, err := h.readCart(r.Context(), userID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, h.buildCart(r.Context(), userID, entries))
}

// AddCartItem handles POST /v1/bookshop/cart/items {book_id, format}.
func (h *BookshopHandler) AddCartItem(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	var req struct {
		BookID   string `json:"book_id"`
		Format   string `json:"format"`
		Quantity int    `json:"quantity"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16*1024)).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_JSON", "invalid request body"))
		return
	}
	bookID, err := uuid.Parse(req.BookID)
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_BOOK_ID", "invalid book_id"))
		return
	}
	if req.Quantity > 1 {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_QUANTITY", "each order is one copy; add the book once"))
		return
	}
	format := domainbookshop.OrderFormat(strings.TrimSpace(req.Format))
	if format == "" {
		book, _ := h.service.CheckOrderable(r.Context(), userID, bookID, domainbookshop.OrderFormatDigital)
		format = defaultFormat(book)
	}
	if format != domainbookshop.OrderFormatDigital && format != domainbookshop.OrderFormatPhysical {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_FORMAT", "format must be digital or physical"))
		return
	}
	book, problem := h.service.CheckOrderable(r.Context(), userID, bookID, format)
	if book == nil {
		writeErrorResponse(w, apperrors.NewNotFoundError("BOOK_NOT_FOUND", "book not found"))
		return
	}
	if problem != "" {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("NOT_AVAILABLE", problem))
		return
	}
	if _, err := h.addCartEntry(r.Context(), userID, cartEntry{BookID: bookID, Format: format, AddedAt: time.Now().UTC()}); err != nil {
		writeErrorResponse(w, err)
		return
	}
	entries, err := h.readCart(r.Context(), userID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, h.buildCart(r.Context(), userID, entries))
}

// UpdateCartItem handles PATCH /v1/bookshop/cart/items/{itemId}: quantity 0
// removes the item; quantity 1 keeps it; nothing larger is allowed.
func (h *BookshopHandler) UpdateCartItem(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Quantity int `json:"quantity"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16*1024)).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_JSON", "invalid request body"))
		return
	}
	if req.Quantity < 0 || req.Quantity > 1 {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_QUANTITY", "each order is one copy (quantity 0 removes the item)"))
		return
	}
	if req.Quantity == 0 {
		h.RemoveCartItem(w, r)
		return
	}
	h.GetCart(w, r)
}

// RemoveCartItem handles DELETE /v1/bookshop/cart/items/{itemId}.
func (h *BookshopHandler) RemoveCartItem(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	bookID, format, err := parseCartItemID(r.PathValue("itemId"))
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	ids := []string{}
	if format == "" {
		ids = append(ids, cartEntry{BookID: bookID, Format: domainbookshop.OrderFormatDigital}.id(),
			cartEntry{BookID: bookID, Format: domainbookshop.OrderFormatPhysical}.id())
	} else {
		ids = append(ids, cartEntry{BookID: bookID, Format: format}.id())
	}
	if err := h.removeCartEntries(r.Context(), userID, ids...); err != nil {
		writeErrorResponse(w, err)
		return
	}
	entries, err := h.readCart(r.Context(), userID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, h.buildCart(r.Context(), userID, entries))
}

type cartCheckoutResult struct {
	Orders   []*appbookshop.OrderResponse       `json:"orders"`
	Requests []*apppayments.PurchaseRequestItem `json:"requests"`
	Failed   []cartCheckoutFailure              `json:"failed"`
	Cart     cartResponse                       `json:"cart"`
}

type cartCheckoutFailure struct {
	ItemID string `json:"item_id"`
	Title  string `json:"title"`
	Error  string `json:"error"`
}

// CheckoutCart handles POST /v1/bookshop/cart/checkout.
//
// Free items become orders, all in one transaction (all or none). Paid items
// become purchase requests that an admin approves after checking the
// payment (there is no gateway); the payment reference covers the basket.
// Every item is checked first, so the usual failures (out of stock, missing
// address, already owned) stop checkout before anything is created. Items
// that succeed leave the cart; a rare failure after the check (stock taken
// in the meantime) is reported per item and the item stays in the cart.
func (h *BookshopHandler) CheckoutCart(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	var req struct {
		ShippingAddress  *domainbookshop.ShippingAddress `json:"shipping_address"`
		PaymentReference string                          `json:"payment_reference"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32*1024)).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_JSON", "invalid request body"))
		return
	}
	entries, err := h.readCart(r.Context(), userID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	cart := h.buildCart(r.Context(), userID, entries)
	if len(cart.Items) == 0 {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("CART_EMPTY", "your cart is empty"))
		return
	}
	for _, item := range cart.Items {
		if !item.Available {
			writeErrorResponse(w, apperrors.NewSimpleValidationError("CART_ITEM_UNAVAILABLE", item.Book.Title+": "+item.Problem+" Remove it to continue."))
			return
		}
	}
	if cart.NeedsAddress {
		var a *domainbookshop.ShippingAddress
		if req.ShippingAddress != nil {
			c := *req.ShippingAddress
			c.Normalize()
			a = &c
		}
		if problem := a.Problem(); problem != "" {
			writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_SHIPPING_ADDRESS", problem))
			return
		}
	}
	reference := strings.TrimSpace(req.PaymentReference)
	if cart.HasPaidItems {
		if reference == "" || len(reference) > 120 {
			writeErrorResponse(w, apperrors.NewValidationError([]map[string]string{{"field": "payment_reference", "message": "enter the transaction ID of your payment (up to 120 characters)"}}))
			return
		}
		if h.purchases == nil {
			writeErrorResponse(w, apperrors.NewInternalError("PURCHASES_UNAVAILABLE", "purchase requests are not available"))
			return
		}
	}

	result := cartCheckoutResult{Orders: []*appbookshop.OrderResponse{}, Requests: []*apppayments.PurchaseRequestItem{}, Failed: []cartCheckoutFailure{}}
	done := map[string]bool{}

	var freeCmds []appbookshop.PlaceOrderCommand
	var freeIDs []string
	for _, item := range cart.Items {
		if item.Free {
			freeCmds = append(freeCmds, appbookshop.PlaceOrderCommand{
				StudentID: userID, BookID: item.Book.ID, Format: item.Format,
				ShippingAddress: addressFor(item.Format, req.ShippingAddress),
			})
			freeIDs = append(freeIDs, item.ID)
		}
	}
	if len(freeCmds) > 0 {
		orders, err := h.service.PlaceOrders(r.Context(), freeCmds)
		if err != nil {
			writeErrorResponse(w, err) // nothing was created
			return
		}
		result.Orders = orders
		for _, id := range freeIDs {
			done[id] = true
		}
	}

	paidCount, paidTotal := 0, 0
	for _, item := range cart.Items {
		if !item.Free {
			paidCount++
			paidTotal += item.UnitPriceCents
		}
	}
	note := fmt.Sprintf("%s · cart of %d paid item(s), total %.2f %s", reference, paidCount, float64(paidTotal)/100, cart.Currency)
	for _, item := range cart.Items {
		if item.Free {
			continue
		}
		created, err := h.purchases.CreatePurchaseRequest(r.Context(), apppayments.CreatePurchaseRequestCommand{
			StudentID: userID, ItemType: domainpayments.PurchaseRequestItemTypeBook, ItemID: item.Book.ID,
			FileName: note, BookFormat: string(item.Format), ShippingAddress: addressFor(item.Format, req.ShippingAddress),
		})
		if err != nil {
			result.Failed = append(result.Failed, cartCheckoutFailure{ItemID: item.ID, Title: item.Book.Title, Error: errorMessage(err)})
			continue
		}
		result.Requests = append(result.Requests, created)
		done[item.ID] = true
	}

	doneIDs := []string{}
	for id := range done {
		doneIDs = append(doneIDs, id)
	}
	if err := h.removeCartEntries(r.Context(), userID, doneIDs...); err != nil {
		writeErrorResponse(w, err)
		return
	}
	remaining, err := h.readCart(r.Context(), userID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	result.Cart = h.buildCart(r.Context(), userID, remaining)
	status := http.StatusCreated
	if len(result.Failed) > 0 {
		status = http.StatusMultiStatus
	}
	writeJSONResponse(w, status, result)
}

func addressFor(format domainbookshop.OrderFormat, a *domainbookshop.ShippingAddress) *domainbookshop.ShippingAddress {
	if format == domainbookshop.OrderFormatPhysical {
		return a
	}
	return nil
}

func errorMessage(err error) string {
	if appErr, ok := err.(*apperrors.AppError); ok {
		return appErr.Message
	}
	return "could not be requested"
}
