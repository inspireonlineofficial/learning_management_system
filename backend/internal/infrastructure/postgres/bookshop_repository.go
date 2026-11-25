package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	domainbookshop "lms-backend/internal/domain/bookshop"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// ─── BookRepository ───────────────────────────────────────────────────────────

// BookRepository implements domain/bookshop.BookRepository.
type BookRepository struct {
	db *sql.DB
}

// NewBookRepository creates a new BookRepository.
func NewBookRepository(db *sql.DB) *BookRepository {
	return &BookRepository{db: db}
}

// Create inserts a new book record.
func (r *BookRepository) Create(ctx context.Context, book *domainbookshop.Book) error {
	_, err := executorForContext(ctx, r.db).ExecContext(ctx, `
		INSERT INTO books
			(id, title, author, subject, class_grade, description, format, price, currency,
			 physical_stock, digital_file_rustfs_key, preview_rustfs_key, cover_rustfs_key, is_active, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`,
		book.ID, book.Title, book.Author, book.Subject, book.ClassGrade, book.Description,
		book.Format, book.Price, book.Currency, book.PhysicalStock,
		book.DigitalFileRustFSKey, book.PreviewRustFSKey, book.CoverRustFSKey, book.IsActive,
		book.CreatedAt, book.UpdatedAt,
	)
	return err
}

// FindByID returns a book by its ID.
func (r *BookRepository) FindByID(ctx context.Context, id uuid.UUID) (*domainbookshop.Book, error) {
	row := executorForContext(ctx, r.db).QueryRowContext(ctx, `
		SELECT id, title, author, subject, class_grade, description, format, price, currency,
		       physical_stock, digital_file_rustfs_key, preview_rustfs_key, cover_rustfs_key, is_active, created_at, updated_at
		FROM books WHERE id = $1`, id)
	return scanBook(row)
}

// Update persists changes to an existing book.
func (r *BookRepository) Update(ctx context.Context, book *domainbookshop.Book) error {
	_, err := executorForContext(ctx, r.db).ExecContext(ctx, `
		UPDATE books SET
			title = $1, author = $2, subject = $3, class_grade = $4, description = $5,
			format = $6, price = $7, currency = $8, physical_stock = $9,
			digital_file_rustfs_key = $10, preview_rustfs_key = $11, cover_rustfs_key = $12,
			is_active = $13, updated_at = $14
		WHERE id = $15`,
		book.Title, book.Author, book.Subject, book.ClassGrade, book.Description,
		book.Format, book.Price, book.Currency, book.PhysicalStock,
		book.DigitalFileRustFSKey, book.PreviewRustFSKey, book.CoverRustFSKey, book.IsActive,
		book.UpdatedAt, book.ID,
	)
	return err
}

// List returns a paginated list of books with optional filters.
func (r *BookRepository) List(ctx context.Context, filter domainbookshop.BookFilter, page, limit int) ([]*domainbookshop.Book, int, error) {
	args := []interface{}{}
	conditions := []string{}
	argIdx := 1

	if filter.ActiveOnly {
		conditions = append(conditions, fmt.Sprintf("is_active = $%d", argIdx))
		args = append(args, true)
		argIdx++
	}
	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(title ILIKE $%d ESCAPE '\\' OR author ILIKE $%d ESCAPE '\\')", argIdx, argIdx+1))
		like := "%" + escapeLike(filter.Search) + "%"
		args = append(args, like, like)
		argIdx += 2
	}
	if filter.Subject != "" {
		conditions = append(conditions, fmt.Sprintf("subject = $%d", argIdx))
		args = append(args, filter.Subject)
		argIdx++
	}
	if filter.ClassGrade != "" {
		conditions = append(conditions, fmt.Sprintf("class_grade = $%d", argIdx))
		args = append(args, filter.ClassGrade)
		argIdx++
	}
	if filter.Format != "" {
		conditions = append(conditions, fmt.Sprintf("format = $%d", argIdx))
		args = append(args, filter.Format)
		argIdx++
	}
	if filter.MinPrice != nil {
		conditions = append(conditions, fmt.Sprintf("price >= $%d", argIdx))
		args = append(args, *filter.MinPrice)
		argIdx++
	}
	if filter.MaxPrice != nil {
		conditions = append(conditions, fmt.Sprintf("price <= $%d", argIdx))
		args = append(args, *filter.MaxPrice)
		argIdx++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM books %s", where)
	if err := executorForContext(ctx, r.db).QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Fetch page
	offset := (page - 1) * limit
	dataArgs := append(args, limit, offset)
	dataQuery := fmt.Sprintf(`
		SELECT id, title, author, subject, class_grade, description, format, price, currency,
		       physical_stock, digital_file_rustfs_key, preview_rustfs_key, cover_rustfs_key, is_active, created_at, updated_at
		FROM books %s
		ORDER BY %s
		LIMIT $%d OFFSET $%d`, where, bookOrderBy(filter.Sort), argIdx, argIdx+1)

	rows, err := executorForContext(ctx, r.db).QueryContext(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var books []*domainbookshop.Book
	for rows.Next() {
		b, err := scanBookScanner(rows)
		if err != nil {
			return nil, 0, err
		}
		books = append(books, b)
	}
	return books, total, rows.Err()
}

// bookOrderBy maps a sort option to a fixed ORDER BY clause (never input).
func bookOrderBy(sort string) string {
	switch sort {
	case "price_asc":
		return "price ASC, created_at DESC, id"
	case "price_desc":
		return "price DESC, created_at DESC, id"
	case "title":
		return "title ASC, id"
	default:
		return "created_at DESC, id"
	}
}

// escapeLike makes user search text match literally inside ILIKE patterns.
func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

type bookScanner interface {
	Scan(dest ...interface{}) error
}

// scanBook scans a single row into a Book.
func scanBook(row *sql.Row) (*domainbookshop.Book, error) {
	b, err := scanBookScanner(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return b, err
}

func scanBookScanner(scanner bookScanner) (*domainbookshop.Book, error) {
	b := &domainbookshop.Book{}
	var subject, classGrade, description sql.NullString
	err := scanner.Scan(
		&b.ID, &b.Title, &b.Author, &subject, &classGrade, &description,
		&b.Format, &b.Price, &b.Currency, &b.PhysicalStock,
		&b.DigitalFileRustFSKey, &b.PreviewRustFSKey, &b.CoverRustFSKey, &b.IsActive,
		&b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	b.Subject = subject.String
	b.ClassGrade = classGrade.String
	b.Description = description.String
	return b, nil
}

// ─── OrderRepository ──────────────────────────────────────────────────────────

// OrderRepository implements domain/bookshop.OrderRepository. Every method
// joins the transaction carried by ctx, when there is one.
type OrderRepository struct {
	db *sql.DB
}

// NewOrderRepository creates a new OrderRepository.
func NewOrderRepository(db *sql.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

// orderSelect reads an order together with the book title and student name
// shown in the student and admin order lists.
const orderSelect = `
	SELECT o.id, o.student_id, o.book_id, o.format, o.amount, o.currency, o.status, o.tracking_number,
	       o.idempotency_key, o.shipping_address, o.shipped_at, o.delivered_at,
	       o.created_at, o.updated_at, o.deleted_at,
	       COALESCE(b.title, ''), COALESCE(u.full_name, ''), COALESCE(u.email, '')
	FROM orders o
	LEFT JOIN books b ON b.id = o.book_id
	LEFT JOIN users u ON u.id = o.student_id`

func shippingAddressValue(a *domainbookshop.ShippingAddress) (interface{}, error) {
	if a == nil {
		return nil, nil
	}
	data, err := json.Marshal(a)
	if err != nil {
		return nil, err
	}
	return string(data), nil
}

// Create inserts a new order record.
func (r *OrderRepository) Create(ctx context.Context, order *domainbookshop.Order) error {
	address, err := shippingAddressValue(order.ShippingAddress)
	if err != nil {
		return err
	}
	_, err = executorForContext(ctx, r.db).ExecContext(ctx, `
		INSERT INTO orders
			(id, student_id, book_id, format, amount, currency, status, tracking_number, idempotency_key,
			 shipping_address, shipped_at, delivered_at, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10::jsonb,$11,$12,$13,$14)`,
		order.ID, order.StudentID, order.BookID, order.Format, order.Amount, order.Currency,
		order.Status, order.TrackingNumber, order.IdempotencyKey,
		address, order.ShippedAt, order.DeliveredAt,
		order.CreatedAt, order.UpdatedAt,
	)
	return err
}

// FindByID returns an order by its ID (excludes soft-deleted).
func (r *OrderRepository) FindByID(ctx context.Context, id uuid.UUID) (*domainbookshop.Order, error) {
	row := executorForContext(ctx, r.db).QueryRowContext(ctx, orderSelect+` WHERE o.id = $1 AND o.deleted_at IS NULL`, id)
	return scanOrder(row)
}

// FindByIdempotencyKey returns an order matching the given idempotency key.
func (r *OrderRepository) FindByIdempotencyKey(ctx context.Context, key string) (*domainbookshop.Order, error) {
	row := executorForContext(ctx, r.db).QueryRowContext(ctx, orderSelect+` WHERE o.idempotency_key = $1 AND o.deleted_at IS NULL`, key)
	return scanOrder(row)
}

// Update persists changes to an existing order.
func (r *OrderRepository) Update(ctx context.Context, order *domainbookshop.Order) error {
	_, err := executorForContext(ctx, r.db).ExecContext(ctx, `
		UPDATE orders SET
			status = $1, tracking_number = $2, shipped_at = $3, delivered_at = $4, updated_at = $5
		WHERE id = $6`,
		order.Status, order.TrackingNumber, order.ShippedAt, order.DeliveredAt, order.UpdatedAt, order.ID,
	)
	return err
}

// UpdateStatusIfCurrent is a compare-and-swap on the order status.
func (r *OrderRepository) UpdateStatusIfCurrent(ctx context.Context, order *domainbookshop.Order, expected domainbookshop.OrderStatus) (bool, error) {
	result, err := executorForContext(ctx, r.db).ExecContext(ctx, `
		UPDATE orders SET
			status = $1, tracking_number = $2, shipped_at = $3, delivered_at = $4, updated_at = $5
		WHERE id = $6 AND status = $7 AND deleted_at IS NULL`,
		order.Status, order.TrackingNumber, order.ShippedAt, order.DeliveredAt, order.UpdatedAt, order.ID, expected,
	)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n == 1, err
}

func (r *OrderRepository) queryOrders(ctx context.Context, query string, args ...interface{}) ([]*domainbookshop.Order, error) {
	rows, err := executorForContext(ctx, r.db).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	orders := []*domainbookshop.Order{}
	for rows.Next() {
		o, err := scanOrderRow(rows)
		if err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	return orders, rows.Err()
}

// FindByStudentID returns all orders for a student (paginated, excludes soft-deleted).
func (r *OrderRepository) FindByStudentID(ctx context.Context, studentID uuid.UUID, page, limit int) ([]*domainbookshop.Order, int, error) {
	var total int
	if err := executorForContext(ctx, r.db).QueryRowContext(ctx,
		`SELECT COUNT(*) FROM orders WHERE student_id = $1 AND deleted_at IS NULL`, studentID,
	).Scan(&total); err != nil {
		return nil, 0, err
	}
	orders, err := r.queryOrders(ctx, orderSelect+`
		WHERE o.student_id = $1 AND o.deleted_at IS NULL
		ORDER BY o.created_at DESC, o.id LIMIT $2 OFFSET $3`,
		studentID, limit, (page-1)*limit)
	return orders, total, err
}

// FindLibrary returns the newest valid digital order per book for a student.
func (r *OrderRepository) FindLibrary(ctx context.Context, studentID uuid.UUID, page, limit int) ([]*domainbookshop.Order, int, error) {
	const owned = `
		FROM orders
		WHERE student_id = $1 AND format = 'digital'
		  AND status NOT IN ('refunded', 'cancelled') AND deleted_at IS NULL`
	var total int
	if err := executorForContext(ctx, r.db).QueryRowContext(ctx,
		`SELECT COUNT(DISTINCT book_id)`+owned, studentID).Scan(&total); err != nil {
		return nil, 0, err
	}
	orders, err := r.queryOrders(ctx, orderSelect+`
		WHERE o.id IN (
			SELECT DISTINCT ON (book_id) id`+owned+`
			ORDER BY book_id, created_at DESC
		)
		ORDER BY o.created_at DESC, o.id LIMIT $2 OFFSET $3`,
		studentID, limit, (page-1)*limit)
	return orders, total, err
}

// List returns all orders for the admin fulfilment queue.
func (r *OrderRepository) List(ctx context.Context, page, limit int) ([]*domainbookshop.Order, int, error) {
	return r.ListByStatuses(ctx, nil, page, limit)
}

// ListByStatuses returns orders filtered by status for the admin queues.
func (r *OrderRepository) ListByStatuses(ctx context.Context, statuses []domainbookshop.OrderStatus, page, limit int) ([]*domainbookshop.Order, int, error) {
	where := ` WHERE o.deleted_at IS NULL`
	args := []interface{}{}
	if len(statuses) > 0 {
		values := make([]string, len(statuses))
		for i, st := range statuses {
			values[i] = string(st)
		}
		where += ` AND o.status::text = ANY($1)`
		args = append(args, pq.Array(values))
	}
	var total int
	if err := executorForContext(ctx, r.db).QueryRowContext(ctx, `SELECT COUNT(*) FROM orders o`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	n := len(args)
	args = append(args, limit, (page-1)*limit)
	orders, err := r.queryOrders(ctx, orderSelect+where+fmt.Sprintf(` ORDER BY o.created_at DESC, o.id LIMIT $%d OFFSET $%d`, n+1, n+2), args...)
	return orders, total, err
}

// FindNonRefundedByStudentAndBook returns the most recent non-refunded digital
// order for a student+book. Only digital orders grant reader access; a
// physical (shipped) order does not.
func (r *OrderRepository) FindNonRefundedByStudentAndBook(ctx context.Context, studentID, bookID uuid.UUID) (*domainbookshop.Order, error) {
	row := executorForContext(ctx, r.db).QueryRowContext(ctx, orderSelect+`
		WHERE o.student_id = $1 AND o.book_id = $2
		  AND o.format = 'digital'
		  AND o.status NOT IN ('refunded', 'cancelled')
		  AND o.deleted_at IS NULL
		ORDER BY o.created_at DESC LIMIT 1`,
		studentID, bookID,
	)
	return scanOrder(row)
}

// DecrementPhysicalStock decrements physical_stock by 1 atomically.
// Returns an error if stock is already 0.
func (r *OrderRepository) DecrementPhysicalStock(ctx context.Context, bookID uuid.UUID) error {
	result, err := executorForContext(ctx, r.db).ExecContext(ctx, `
		UPDATE books SET physical_stock = physical_stock - 1, updated_at = $1
		WHERE id = $2 AND physical_stock > 0`,
		time.Now().UTC(), bookID,
	)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domainbookshop.ErrOutOfStock
	}
	return nil
}

// IncrementPhysicalStock increments physical_stock by 1 (used on refund or cancellation).
func (r *OrderRepository) IncrementPhysicalStock(ctx context.Context, bookID uuid.UUID) error {
	_, err := executorForContext(ctx, r.db).ExecContext(ctx, `
		UPDATE books SET physical_stock = physical_stock + 1, updated_at = $1
		WHERE id = $2`,
		time.Now().UTC(), bookID,
	)
	return err
}

func scanOrderFields(scanner bookScanner) (*domainbookshop.Order, error) {
	o := &domainbookshop.Order{}
	var address []byte
	err := scanner.Scan(
		&o.ID, &o.StudentID, &o.BookID, &o.Format, &o.Amount, &o.Currency,
		&o.Status, &o.TrackingNumber, &o.IdempotencyKey, &address, &o.ShippedAt, &o.DeliveredAt,
		&o.CreatedAt, &o.UpdatedAt, &o.DeletedAt,
		&o.BookTitle, &o.StudentName, &o.StudentEmail,
	)
	if err != nil {
		return nil, err
	}
	if len(address) > 0 {
		var a domainbookshop.ShippingAddress
		if json.Unmarshal(address, &a) == nil {
			o.ShippingAddress = &a
		}
	}
	return o, nil
}

// scanOrder scans a single row into an Order.
func scanOrder(row *sql.Row) (*domainbookshop.Order, error) {
	o, err := scanOrderFields(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return o, err
}

// scanOrderRow scans a rows.Next() row into an Order.
func scanOrderRow(rows *sql.Rows) (*domainbookshop.Order, error) {
	return scanOrderFields(rows)
}

// ─── BookBookmarkRepository ───────────────────────────────────────────────────

// BookBookmarkRepository implements domain/bookshop.BookBookmarkRepository.
type BookBookmarkRepository struct {
	db *sql.DB
}

// NewBookBookmarkRepository creates a new BookBookmarkRepository.
func NewBookBookmarkRepository(db *sql.DB) *BookBookmarkRepository {
	return &BookBookmarkRepository{db: db}
}

// Upsert creates or updates the bookmark for a student+book pair.
func (r *BookBookmarkRepository) Upsert(ctx context.Context, bookmark *domainbookshop.BookBookmark) error {
	_, err := executorForContext(ctx, r.db).ExecContext(ctx, `
		INSERT INTO book_bookmarks (id, student_id, book_id, last_page_read, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (student_id, book_id)
		DO UPDATE SET last_page_read = EXCLUDED.last_page_read, updated_at = EXCLUDED.updated_at`,
		bookmark.ID, bookmark.StudentID, bookmark.BookID, bookmark.LastPageRead, bookmark.UpdatedAt,
	)
	return err
}

// FindByStudentAndBook returns the bookmark for a student+book pair.
func (r *BookBookmarkRepository) FindByStudentAndBook(ctx context.Context, studentID, bookID uuid.UUID) (*domainbookshop.BookBookmark, error) {
	row := executorForContext(ctx, r.db).QueryRowContext(ctx, `
		SELECT id, student_id, book_id, last_page_read, updated_at
		FROM book_bookmarks WHERE student_id = $1 AND book_id = $2`,
		studentID, bookID,
	)
	bm := &domainbookshop.BookBookmark{}
	err := row.Scan(&bm.ID, &bm.StudentID, &bm.BookID, &bm.LastPageRead, &bm.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return bm, err
}
