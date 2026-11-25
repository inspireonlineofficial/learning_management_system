package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	domainbookshop "lms-backend/internal/domain/bookshop"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// RefundRepository implements domain/bookshop.RefundRepository.
type RefundRepository struct{ db *sql.DB }

// NewRefundRepository creates a RefundRepository.
func NewRefundRepository(db *sql.DB) *RefundRepository { return &RefundRepository{db: db} }

const refundRequestSelect = `
	SELECT rr.id, rr.order_id, rr.student_id, rr.reason, rr.status, rr.admin_note, rr.reviewed_by, rr.reviewed_at,
	       rr.created_at, rr.updated_at,
	       COALESCE(b.title, ''), COALESCE(u.full_name, ''), COALESCE(u.email, ''),
	       o.amount, o.currency, o.format, o.status
	FROM refund_requests rr
	JOIN orders o ON o.id = rr.order_id
	LEFT JOIN books b ON b.id = o.book_id
	LEFT JOIN users u ON u.id = rr.student_id`

func scanRefundRequest(s bookScanner) (*domainbookshop.RefundRequest, error) {
	r := &domainbookshop.RefundRequest{}
	err := s.Scan(&r.ID, &r.OrderID, &r.StudentID, &r.Reason, &r.Status, &r.AdminNote, &r.ReviewedBy, &r.ReviewedAt,
		&r.CreatedAt, &r.UpdatedAt, &r.BookTitle, &r.StudentName, &r.StudentEmail,
		&r.Amount, &r.Currency, &r.Format, &r.OrderStatus)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return r, err
}

func (r *RefundRepository) CreateRequest(ctx context.Context, req *domainbookshop.RefundRequest) error {
	_, err := executorForContext(ctx, r.db).ExecContext(ctx, `
		INSERT INTO refund_requests (id, order_id, student_id, reason, status, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		req.ID, req.OrderID, req.StudentID, req.Reason, req.Status, req.CreatedAt, req.UpdatedAt)
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23505" {
		return domainbookshop.ErrRefundRequestExists
	}
	return err
}

func (r *RefundRepository) FindRequestByID(ctx context.Context, id uuid.UUID) (*domainbookshop.RefundRequest, error) {
	return scanRefundRequest(executorForContext(ctx, r.db).QueryRowContext(ctx, refundRequestSelect+` WHERE rr.id = $1`, id))
}

func (r *RefundRepository) LockRequest(ctx context.Context, id uuid.UUID) (*domainbookshop.RefundRequest, error) {
	return scanRefundRequest(executorForContext(ctx, r.db).QueryRowContext(ctx, refundRequestSelect+` WHERE rr.id = $1 FOR UPDATE OF rr`, id))
}

func (r *RefundRepository) FindPendingRequestByOrder(ctx context.Context, orderID uuid.UUID) (*domainbookshop.RefundRequest, error) {
	return scanRefundRequest(executorForContext(ctx, r.db).QueryRowContext(ctx,
		refundRequestSelect+` WHERE rr.order_id = $1 AND rr.status = 'pending'`, orderID))
}

func (r *RefundRepository) LatestRequestByOrder(ctx context.Context, orderID uuid.UUID) (*domainbookshop.RefundRequest, error) {
	return scanRefundRequest(executorForContext(ctx, r.db).QueryRowContext(ctx,
		refundRequestSelect+` WHERE rr.order_id = $1 ORDER BY rr.created_at DESC LIMIT 1`, orderID))
}

func (r *RefundRepository) UpdateRequest(ctx context.Context, req *domainbookshop.RefundRequest) error {
	_, err := executorForContext(ctx, r.db).ExecContext(ctx, `
		UPDATE refund_requests SET status = $2, admin_note = $3, reviewed_by = $4, reviewed_at = $5, updated_at = $6
		WHERE id = $1`, req.ID, req.Status, req.AdminNote, req.ReviewedBy, req.ReviewedAt, req.UpdatedAt)
	return err
}

func (r *RefundRepository) ListRequests(ctx context.Context, status domainbookshop.RefundRequestStatus, studentID *uuid.UUID, page, limit int) ([]*domainbookshop.RefundRequest, int, error) {
	where := ` WHERE TRUE`
	args := []interface{}{}
	if status != "" {
		args = append(args, status)
		where += fmt.Sprintf(` AND rr.status = $%d`, len(args))
	}
	if studentID != nil {
		args = append(args, *studentID)
		where += fmt.Sprintf(` AND rr.student_id = $%d`, len(args))
	}
	exec := executorForContext(ctx, r.db)
	var total int
	if err := exec.QueryRowContext(ctx, `SELECT COUNT(*) FROM refund_requests rr`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	n := len(args)
	args = append(args, limit, (page-1)*limit)
	rows, err := exec.QueryContext(ctx, refundRequestSelect+where+fmt.Sprintf(` ORDER BY rr.created_at DESC, rr.id LIMIT $%d OFFSET $%d`, n+1, n+2), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []*domainbookshop.RefundRequest{}
	for rows.Next() {
		req, err := scanRefundRequest(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, req)
	}
	return out, total, rows.Err()
}

func (r *RefundRepository) CreateRefund(ctx context.Context, ref *domainbookshop.OrderRefund) error {
	_, err := executorForContext(ctx, r.db).ExecContext(ctx, `
		INSERT INTO order_refunds (id, order_id, refund_request_id, amount, currency, payout_method, payout_reference, note, processed_by, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		ref.ID, ref.OrderID, ref.RefundRequestID, ref.Amount, ref.Currency, ref.PayoutMethod, ref.PayoutReference, ref.Note, ref.ProcessedBy, ref.CreatedAt)
	return err
}

func (r *RefundRepository) FindRefundByOrder(ctx context.Context, orderID uuid.UUID) (*domainbookshop.OrderRefund, error) {
	ref := &domainbookshop.OrderRefund{}
	err := executorForContext(ctx, r.db).QueryRowContext(ctx, `
		SELECT id, order_id, refund_request_id, amount, currency, payout_method, payout_reference, note, processed_by, created_at
		FROM order_refunds WHERE order_id = $1`, orderID).Scan(
		&ref.ID, &ref.OrderID, &ref.RefundRequestID, &ref.Amount, &ref.Currency, &ref.PayoutMethod, &ref.PayoutReference, &ref.Note, &ref.ProcessedBy, &ref.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return ref, err
}

func (r *RefundRepository) CountRequestsForOrder(ctx context.Context, orderID uuid.UUID) (int, error) {
	var n int
	err := executorForContext(ctx, r.db).QueryRowContext(ctx, `SELECT COUNT(*) FROM refund_requests WHERE order_id = $1`, orderID).Scan(&n)
	return n, err
}
