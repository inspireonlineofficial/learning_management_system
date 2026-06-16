package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	domainpayments "lms-backend/internal/domain/payments"

	"github.com/google/uuid"
)

// PurchaseRequestRepository implements domain/payments.PurchaseRequestRepository.
type PurchaseRequestRepository struct {
	db *sql.DB
}

// NewPurchaseRequestRepository creates a new repository for purchase requests.
func NewPurchaseRequestRepository(db *sql.DB) *PurchaseRequestRepository {
	return &PurchaseRequestRepository{db: db}
}

func (r *PurchaseRequestRepository) Create(ctx context.Context, request *domainpayments.PurchaseRequest) error {
	exec := executorForContext(ctx, r.db)
	_, err := exec.ExecContext(ctx, `
		INSERT INTO purchase_requests
			(id, student_id, item_type, item_id, file_name, idempotency_key, status, rejection_reason, result_enrollment_id, result_order_id, reviewed_by, reviewed_at, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		request.ID, request.StudentID, request.ItemType, request.ItemID, request.FileName, request.IdempotencyKey,
		request.Status, request.RejectionReason, request.ResultEnrollmentID, request.ResultOrderID, request.ReviewedBy, request.ReviewedAt,
		request.CreatedAt, request.UpdatedAt,
	)
	return err
}

func (r *PurchaseRequestRepository) FindByID(ctx context.Context, id uuid.UUID) (*domainpayments.PurchaseRequest, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, student_id, item_type, item_id, file_name, idempotency_key, status, rejection_reason, result_enrollment_id, result_order_id, reviewed_by, reviewed_at, created_at, updated_at
		FROM purchase_requests
		WHERE id = $1`, id)
	return scanPurchaseRequest(row)
}

func (r *PurchaseRequestRepository) FindByIdempotencyKey(ctx context.Context, key string) (*domainpayments.PurchaseRequest, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, student_id, item_type, item_id, file_name, idempotency_key, status, rejection_reason, result_enrollment_id, result_order_id, reviewed_by, reviewed_at, created_at, updated_at
		FROM purchase_requests
		WHERE idempotency_key = $1
		ORDER BY created_at DESC
		LIMIT 1`, key)
	return scanPurchaseRequest(row)
}

func (r *PurchaseRequestRepository) FindLatestByStudentAndItem(ctx context.Context, studentID, itemID uuid.UUID, itemType domainpayments.PurchaseRequestItemType) (*domainpayments.PurchaseRequest, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, student_id, item_type, item_id, file_name, idempotency_key, status, rejection_reason, result_enrollment_id, result_order_id, reviewed_by, reviewed_at, created_at, updated_at
		FROM purchase_requests
		WHERE student_id = $1 AND item_id = $2 AND item_type = $3
		ORDER BY created_at DESC
		LIMIT 1`, studentID, itemID, itemType)
	return scanPurchaseRequest(row)
}

func (r *PurchaseRequestRepository) Update(ctx context.Context, request *domainpayments.PurchaseRequest) error {
	exec := executorForContext(ctx, r.db)
	_, err := exec.ExecContext(ctx, `
		UPDATE purchase_requests
		SET file_name = $2, idempotency_key = $3, status = $4, rejection_reason = $5,
		    result_enrollment_id = $6, result_order_id = $7, reviewed_by = $8, reviewed_at = $9, updated_at = $10
		WHERE id = $1`,
		request.ID, request.FileName, request.IdempotencyKey, request.Status, request.RejectionReason,
		request.ResultEnrollmentID, request.ResultOrderID, request.ReviewedBy, request.ReviewedAt, request.UpdatedAt,
	)
	return err
}

func (r *PurchaseRequestRepository) List(ctx context.Context, filter domainpayments.PurchaseRequestFilter, page, limit int) ([]*domainpayments.PurchaseRequest, int, error) {
	where, args := buildPurchaseRequestFilter(filter)
	countQuery := "SELECT COUNT(*) FROM purchase_requests" + where
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, student_id, item_type, item_id, file_name, idempotency_key, status, rejection_reason, result_enrollment_id, result_order_id, reviewed_by, reviewed_at, created_at, updated_at
		FROM purchase_requests` + where + ` ORDER BY created_at DESC`
	if limit > 0 {
		offset := (page - 1) * limit
		args = append(args, limit, offset)
		query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)-1, len(args))
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var requests []*domainpayments.PurchaseRequest
	for rows.Next() {
		request, err := scanPurchaseRequest(rows)
		if err != nil {
			return nil, 0, err
		}
		requests = append(requests, request)
	}
	return requests, total, rows.Err()
}

func (r *PurchaseRequestRepository) ListAll(ctx context.Context, filter domainpayments.PurchaseRequestFilter) ([]*domainpayments.PurchaseRequest, error) {
	requests, _, err := r.List(ctx, filter, 1, 0)
	return requests, err
}

func buildPurchaseRequestFilter(filter domainpayments.PurchaseRequestFilter) (string, []interface{}) {
	var conditions []string
	var args []interface{}
	index := 1

	if filter.StudentID != nil {
		conditions = append(conditions, fmt.Sprintf("student_id = $%d", index))
		args = append(args, *filter.StudentID)
		index++
	}
	if filter.ItemType != nil {
		conditions = append(conditions, fmt.Sprintf("item_type = $%d", index))
		args = append(args, *filter.ItemType)
		index++
	}
	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", index))
		args = append(args, *filter.Status)
		index++
	}

	if len(conditions) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(conditions, " AND "), args
}

func scanPurchaseRequest(row interface {
	Scan(dest ...any) error
}) (*domainpayments.PurchaseRequest, error) {
	request := &domainpayments.PurchaseRequest{}
	err := row.Scan(
		&request.ID,
		&request.StudentID,
		&request.ItemType,
		&request.ItemID,
		&request.FileName,
		&request.IdempotencyKey,
		&request.Status,
		&request.RejectionReason,
		&request.ResultEnrollmentID,
		&request.ResultOrderID,
		&request.ReviewedBy,
		&request.ReviewedAt,
		&request.CreatedAt,
		&request.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return request, err
}
