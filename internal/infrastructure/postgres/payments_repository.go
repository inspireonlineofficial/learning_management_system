package postgres

import (
	"context"
	"database/sql"
	"strconv"
	"strings"

	domainpayments "lms-backend/internal/domain/payments"

	"github.com/google/uuid"
)

// ─── PaymentIntentRepository ──────────────────────────────────────────────────

// PaymentIntentRepository implements domain/payments.PaymentIntentRepository.
type PaymentIntentRepository struct {
	db *sql.DB
}

// NewPaymentIntentRepository creates a new PaymentIntentRepository.
func NewPaymentIntentRepository(db *sql.DB) *PaymentIntentRepository {
	return &PaymentIntentRepository{db: db}
}

// Create inserts a new payment intent record.
func (r *PaymentIntentRepository) Create(ctx context.Context, intent *domainpayments.PaymentIntent) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO payment_intents
			(id, student_id, item_type, item_id, amount, currency, status, provider_intent_id, bkash_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		intent.ID,
		intent.StudentID,
		intent.ItemType,
		intent.ItemID,
		intent.Amount,
		intent.Currency,
		intent.Status,
		intent.ProviderIntentID,
		intent.BkashURL,
		intent.CreatedAt,
		intent.UpdatedAt,
	)
	return err
}

// FindByID returns a payment intent by its ID, or nil if not found.
func (r *PaymentIntentRepository) FindByID(ctx context.Context, id uuid.UUID) (*domainpayments.PaymentIntent, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, student_id, item_type, item_id, amount, currency, status, provider_intent_id, bkash_url, created_at, updated_at
		FROM payment_intents
		WHERE id = $1`, id)
	return scanPaymentIntent(row)
}

// Update persists status changes to an existing payment intent.
func (r *PaymentIntentRepository) Update(ctx context.Context, intent *domainpayments.PaymentIntent) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE payment_intents
		SET status = $1, provider_intent_id = $2, updated_at = $3
		WHERE id = $4`,
		intent.Status,
		intent.ProviderIntentID,
		intent.UpdatedAt,
		intent.ID,
	)
	return err
}

// FindByStudentAndItem returns an existing pending intent for the same student+item.
func (r *PaymentIntentRepository) FindByStudentAndItem(ctx context.Context, studentID, itemID uuid.UUID, itemType domainpayments.ItemType) (*domainpayments.PaymentIntent, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, student_id, item_type, item_id, amount, currency, status, provider_intent_id, bkash_url, created_at, updated_at
		FROM payment_intents
		WHERE student_id = $1 AND item_id = $2 AND item_type = $3 AND status = 'pending'
		ORDER BY created_at DESC
		LIMIT 1`,
		studentID, itemID, itemType,
	)
	return scanPaymentIntent(row)
}

// FindByProviderIntentID returns a payment intent by its provider-assigned ID.
// Used to look up the intent during bKash payment callbacks. Requirements: 4.3
func (r *PaymentIntentRepository) FindByProviderIntentID(ctx context.Context, providerIntentID string) (*domainpayments.PaymentIntent, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, student_id, item_type, item_id, amount, currency, status, provider_intent_id, bkash_url, created_at, updated_at
		FROM payment_intents
		WHERE provider_intent_id = $1`, providerIntentID)
	return scanPaymentIntent(row)
}

func scanPaymentIntent(row *sql.Row) (*domainpayments.PaymentIntent, error) {
	var intent domainpayments.PaymentIntent
	err := row.Scan(
		&intent.ID,
		&intent.StudentID,
		&intent.ItemType,
		&intent.ItemID,
		&intent.Amount,
		&intent.Currency,
		&intent.Status,
		&intent.ProviderIntentID,
		&intent.BkashURL,
		&intent.CreatedAt,
		&intent.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &intent, nil
}

// ─── PaymentRepository ────────────────────────────────────────────────────────

// PaymentRepository implements domain/payments.PaymentRepository.
type PaymentRepository struct {
	db *sql.DB
}

// NewPaymentRepository creates a new PaymentRepository.
func NewPaymentRepository(db *sql.DB) *PaymentRepository {
	return &PaymentRepository{db: db}
}

// Create inserts a new payment record.
func (r *PaymentRepository) Create(ctx context.Context, payment *domainpayments.Payment) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO payments
			(id, payment_intent_id, student_id, idempotency_key, provider_transaction_id, amount, currency, status, receipt_url, paid_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		payment.ID,
		payment.PaymentIntentID,
		payment.StudentID,
		payment.IdempotencyKey,
		payment.ProviderTransactionID,
		payment.Amount,
		payment.Currency,
		payment.Status,
		payment.ReceiptURL,
		payment.PaidAt,
		payment.CreatedAt,
	)
	return err
}

// FindByIdempotencyKey returns a payment matching the given idempotency key, or nil.
func (r *PaymentRepository) FindByIdempotencyKey(ctx context.Context, key string) (*domainpayments.Payment, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, payment_intent_id, student_id, idempotency_key, provider_transaction_id, amount, currency, status, receipt_url, paid_at, created_at
		FROM payments
		WHERE idempotency_key = $1`, key)
	return scanPayment(row)
}

// FindByStudentID returns paginated payment history for a student.
func (r *PaymentRepository) FindByStudentID(ctx context.Context, studentID uuid.UUID, page, limit int) ([]*domainpayments.Payment, int, error) {
	offset := (page - 1) * limit

	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM payments WHERE student_id = $1`, studentID).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT p.id, p.payment_intent_id, p.student_id, pi.item_type, p.idempotency_key, p.provider_transaction_id, p.amount, p.currency, p.status, p.receipt_url, p.paid_at, p.created_at
		FROM payments p
		JOIN payment_intents pi ON pi.id = p.payment_intent_id
		WHERE p.student_id = $1
		ORDER BY p.created_at DESC
		LIMIT $2 OFFSET $3`,
		studentID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var payments []*domainpayments.Payment
	for rows.Next() {
		p, err := scanPaymentWithItemTypeRow(rows)
		if err != nil {
			return nil, 0, err
		}
		payments = append(payments, p)
	}
	return payments, total, rows.Err()
}

// List returns all payments with optional filters (admin view).
func (r *PaymentRepository) List(ctx context.Context, filter domainpayments.PaymentFilter, page, limit int) ([]*domainpayments.Payment, int, error) {
	offset := (page - 1) * limit

	where, args := buildPaymentFilter(filter)

	countQuery := "SELECT COUNT(*) FROM payments p JOIN payment_intents pi ON pi.id = p.payment_intent_id" + where
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	query := `SELECT p.id, p.payment_intent_id, p.student_id, pi.item_type, p.idempotency_key, p.provider_transaction_id, p.amount, p.currency, p.status, p.receipt_url, p.paid_at, p.created_at
		FROM payments p JOIN payment_intents pi ON pi.id = p.payment_intent_id` + where + ` ORDER BY p.created_at DESC LIMIT $` + itoa(len(args)-1) + ` OFFSET $` + itoa(len(args))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var payments []*domainpayments.Payment
	for rows.Next() {
		p, err := scanPaymentWithItemTypeRow(rows)
		if err != nil {
			return nil, 0, err
		}
		payments = append(payments, p)
	}
	return payments, total, rows.Err()
}

// buildPaymentFilter constructs a WHERE clause and args slice from a PaymentFilter.
func buildPaymentFilter(filter domainpayments.PaymentFilter) (string, []interface{}) {
	var conditions []string
	var args []interface{}
	n := 1

	if filter.StudentID != nil {
		conditions = append(conditions, "p.student_id = $"+itoa(n))
		args = append(args, *filter.StudentID)
		n++
	}
	if filter.ItemType != nil {
		conditions = append(conditions, "pi.item_type = $"+itoa(n))
		args = append(args, *filter.ItemType)
		n++
	}
	if filter.Status != nil {
		conditions = append(conditions, "p.status = $"+itoa(n))
		args = append(args, *filter.Status)
		n++
	}
	if filter.FromDate != nil {
		conditions = append(conditions, "p.created_at >= $"+itoa(n))
		args = append(args, *filter.FromDate)
		n++
	}
	if filter.ToDate != nil {
		conditions = append(conditions, "p.created_at <= $"+itoa(n))
		args = append(args, *filter.ToDate)
		n++
	}

	if len(conditions) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(conditions, " AND "), args
}

func scanPayment(row *sql.Row) (*domainpayments.Payment, error) {
	var p domainpayments.Payment
	var paidAt sql.NullTime
	var receiptURL sql.NullString
	err := row.Scan(
		&p.ID,
		&p.PaymentIntentID,
		&p.StudentID,
		&p.IdempotencyKey,
		&p.ProviderTransactionID,
		&p.Amount,
		&p.Currency,
		&p.Status,
		&receiptURL,
		&paidAt,
		&p.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if receiptURL.Valid {
		p.ReceiptURL = &receiptURL.String
	}
	if paidAt.Valid {
		t := paidAt.Time
		p.PaidAt = &t
	}
	return &p, nil
}

func scanPaymentRow(rows *sql.Rows) (*domainpayments.Payment, error) {
	var p domainpayments.Payment
	var paidAt sql.NullTime
	var receiptURL sql.NullString
	err := rows.Scan(
		&p.ID,
		&p.PaymentIntentID,
		&p.StudentID,
		&p.IdempotencyKey,
		&p.ProviderTransactionID,
		&p.Amount,
		&p.Currency,
		&p.Status,
		&receiptURL,
		&paidAt,
		&p.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if receiptURL.Valid {
		p.ReceiptURL = &receiptURL.String
	}
	if paidAt.Valid {
		t := paidAt.Time
		p.PaidAt = &t
	}
	return &p, nil
}

func scanPaymentWithItemTypeRow(rows *sql.Rows) (*domainpayments.Payment, error) {
	var p domainpayments.Payment
	var paidAt sql.NullTime
	var receiptURL sql.NullString
	err := rows.Scan(
		&p.ID,
		&p.PaymentIntentID,
		&p.StudentID,
		&p.ItemType,
		&p.IdempotencyKey,
		&p.ProviderTransactionID,
		&p.Amount,
		&p.Currency,
		&p.Status,
		&receiptURL,
		&paidAt,
		&p.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if receiptURL.Valid {
		p.ReceiptURL = &receiptURL.String
	}
	if paidAt.Valid {
		t := paidAt.Time
		p.PaidAt = &t
	}
	return &p, nil
}

// itoa converts an int to a decimal string.
func itoa(n int) string {
	return strconv.Itoa(n)
}
