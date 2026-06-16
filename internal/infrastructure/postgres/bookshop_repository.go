package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	domainbookshop "lms-backend/internal/domain/bookshop"

	"github.com/google/uuid"
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
	_, err := r.db.ExecContext(ctx, `
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
	row := r.db.QueryRowContext(ctx, `
		SELECT id, title, author, subject, class_grade, description, format, price, currency,
		       physical_stock, digital_file_rustfs_key, preview_rustfs_key, cover_rustfs_key, is_active, created_at, updated_at
		FROM books WHERE id = $1`, id)
	return scanBook(row)
}

// Update persists changes to an existing book.
func (r *BookRepository) Update(ctx context.Context, book *domainbookshop.Book) error {
	_, err := r.db.ExecContext(ctx, `
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
		conditions = append(conditions, fmt.Sprintf("(title ILIKE $%d OR author ILIKE $%d)", argIdx, argIdx+1))
		like := "%" + filter.Search + "%"
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
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Fetch page
	offset := (page - 1) * limit
	dataArgs := append(args, limit, offset)
	dataQuery := fmt.Sprintf(`
		SELECT id, title, author, subject, class_grade, description, format, price, currency,
		       physical_stock, digital_file_rustfs_key, preview_rustfs_key, cover_rustfs_key, is_active, created_at, updated_at
		FROM books %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`, where, argIdx, argIdx+1)

	rows, err := r.db.QueryContext(ctx, dataQuery, dataArgs...)
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

// OrderRepository implements domain/bookshop.OrderRepository.
type OrderRepository struct {
	db *sql.DB
}

// NewOrderRepository creates a new OrderRepository.
func NewOrderRepository(db *sql.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

// Create inserts a new order record.
func (r *OrderRepository) Create(ctx context.Context, order *domainbookshop.Order) error {
	exec := executorForContext(ctx, r.db)
	_, err := exec.ExecContext(ctx, `
		INSERT INTO orders
			(id, student_id, book_id, format, amount, currency, status, tracking_number, idempotency_key, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		order.ID, order.StudentID, order.BookID, order.Format, order.Amount, order.Currency,
		order.Status, order.TrackingNumber, order.IdempotencyKey,
		order.CreatedAt, order.UpdatedAt,
	)
	return err
}

// FindByID returns an order by its ID (excludes soft-deleted).
func (r *OrderRepository) FindByID(ctx context.Context, id uuid.UUID) (*domainbookshop.Order, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, student_id, book_id, format, amount, currency, status, tracking_number, idempotency_key, created_at, updated_at, deleted_at
		FROM orders WHERE id = $1 AND deleted_at IS NULL`, id)
	return scanOrder(row)
}

// FindByIdempotencyKey returns an order matching the given idempotency key.
func (r *OrderRepository) FindByIdempotencyKey(ctx context.Context, key string) (*domainbookshop.Order, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, student_id, book_id, format, amount, currency, status, tracking_number, idempotency_key, created_at, updated_at, deleted_at
		FROM orders WHERE idempotency_key = $1 AND deleted_at IS NULL`, key)
	return scanOrder(row)
}

// Update persists changes to an existing order.
func (r *OrderRepository) Update(ctx context.Context, order *domainbookshop.Order) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE orders SET
			status = $1, tracking_number = $2, updated_at = $3
		WHERE id = $4`,
		order.Status, order.TrackingNumber, order.UpdatedAt, order.ID,
	)
	return err
}

// FindByStudentID returns all orders for a student (paginated, excludes soft-deleted).
func (r *OrderRepository) FindByStudentID(ctx context.Context, studentID uuid.UUID, page, limit int) ([]*domainbookshop.Order, int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM orders WHERE student_id = $1 AND deleted_at IS NULL`, studentID,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, student_id, book_id, format, amount, currency, status, tracking_number, idempotency_key, created_at, updated_at, deleted_at
		FROM orders WHERE student_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		studentID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var orders []*domainbookshop.Order
	for rows.Next() {
		o, err := scanOrderRow(rows)
		if err != nil {
			return nil, 0, err
		}
		orders = append(orders, o)
	}
	return orders, total, rows.Err()
}

// List returns all orders for the admin fulfilment queue.
func (r *OrderRepository) List(ctx context.Context, page, limit int) ([]*domainbookshop.Order, int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM orders WHERE deleted_at IS NULL`).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, student_id, book_id, format, amount, currency, status, tracking_number, idempotency_key, created_at, updated_at, deleted_at
		FROM orders WHERE deleted_at IS NULL
		ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var orders []*domainbookshop.Order
	for rows.Next() {
		order, err := scanOrderRow(rows)
		if err != nil {
			return nil, 0, err
		}
		orders = append(orders, order)
	}
	return orders, total, rows.Err()
}

// FindNonRefundedByStudentAndBook returns the most recent non-refunded order for a student+book.
func (r *OrderRepository) FindNonRefundedByStudentAndBook(ctx context.Context, studentID, bookID uuid.UUID) (*domainbookshop.Order, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, student_id, book_id, format, amount, currency, status, tracking_number, idempotency_key, created_at, updated_at, deleted_at
		FROM orders
		WHERE student_id = $1 AND book_id = $2
		  AND status NOT IN ('refunded', 'cancelled')
		  AND deleted_at IS NULL
		ORDER BY created_at DESC LIMIT 1`,
		studentID, bookID,
	)
	return scanOrder(row)
}

// DecrementPhysicalStock decrements physical_stock by 1 atomically.
// Returns an error if stock is already 0.
func (r *OrderRepository) DecrementPhysicalStock(ctx context.Context, bookID uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, `
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
		return fmt.Errorf("out of stock")
	}
	return nil
}

// IncrementPhysicalStock increments physical_stock by 1 (used on refund).
func (r *OrderRepository) IncrementPhysicalStock(ctx context.Context, bookID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE books SET physical_stock = physical_stock + 1, updated_at = $1
		WHERE id = $2`,
		time.Now().UTC(), bookID,
	)
	return err
}

// scanOrder scans a single row into an Order.
func scanOrder(row *sql.Row) (*domainbookshop.Order, error) {
	o := &domainbookshop.Order{}
	err := row.Scan(
		&o.ID, &o.StudentID, &o.BookID, &o.Format, &o.Amount, &o.Currency,
		&o.Status, &o.TrackingNumber, &o.IdempotencyKey,
		&o.CreatedAt, &o.UpdatedAt, &o.DeletedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return o, err
}

// scanOrderRow scans a rows.Next() row into an Order.
func scanOrderRow(rows *sql.Rows) (*domainbookshop.Order, error) {
	o := &domainbookshop.Order{}
	err := rows.Scan(
		&o.ID, &o.StudentID, &o.BookID, &o.Format, &o.Amount, &o.Currency,
		&o.Status, &o.TrackingNumber, &o.IdempotencyKey,
		&o.CreatedAt, &o.UpdatedAt, &o.DeletedAt,
	)
	return o, err
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
	_, err := r.db.ExecContext(ctx, `
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
	row := r.db.QueryRowContext(ctx, `
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
