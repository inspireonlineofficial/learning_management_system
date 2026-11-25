package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	domainaudit "lms-backend/internal/domain/audit"

	"github.com/google/uuid"
)

// AuditLogRepository implements domainaudit.AuditLogRepository.
// The DB user has no UPDATE/DELETE on audit_logs (Requirements: 9.6).
type AuditLogRepository struct {
	db *sql.DB
}

// NewAuditLogRepository creates a new AuditLogRepository.
func NewAuditLogRepository(db *sql.DB) *AuditLogRepository {
	return &AuditLogRepository{db: db}
}

// List returns paginated, filterable audit log entries.
// Requirements: 9.5
func (r *AuditLogRepository) List(ctx context.Context, filter domainaudit.AuditLogFilter, page, limit int) ([]domainaudit.AuditLog, int, error) {
	where := []string{"1=1"}
	args := []interface{}{}
	argIdx := 1

	if filter.ActorID != nil {
		where = append(where, fmt.Sprintf("actor_id = $%d", argIdx))
		args = append(args, *filter.ActorID)
		argIdx++
	}
	if filter.Action != nil {
		where = append(where, fmt.Sprintf("action = $%d", argIdx))
		args = append(args, *filter.Action)
		argIdx++
	}
	if filter.TargetType != nil {
		where = append(where, fmt.Sprintf("target_type = $%d", argIdx))
		args = append(args, *filter.TargetType)
		argIdx++
	}
	if filter.TargetID != nil {
		where = append(where, fmt.Sprintf("target_id = $%d", argIdx))
		args = append(args, *filter.TargetID)
		argIdx++
	}
	if filter.FromDate != nil {
		where = append(where, fmt.Sprintf("created_at >= $%d", argIdx))
		args = append(args, *filter.FromDate)
		argIdx++
	}
	if filter.ToDate != nil {
		where = append(where, fmt.Sprintf("created_at <= $%d", argIdx))
		args = append(args, *filter.ToDate)
		argIdx++
	}

	whereClause := strings.Join(where, " AND ")

	// Count total
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM audit_logs WHERE %s", whereClause)
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count audit logs: %w", err)
	}

	// Fetch page
	offset := (page - 1) * limit
	dataArgs := append(args, limit, offset)
	dataQuery := fmt.Sprintf(`
		SELECT id, actor_id, actor_name, action, target_type, target_id, metadata, ip_address::text, created_at
		FROM audit_logs
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)

	rows, err := r.db.QueryContext(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("query audit logs: %w", err)
	}
	defer rows.Close()

	var logs []domainaudit.AuditLog
	for rows.Next() {
		var l domainaudit.AuditLog
		var targetIDStr *string
		var ipStr *string
		if err := rows.Scan(
			&l.ID, &l.ActorID, &l.ActorName, &l.Action,
			&l.TargetType, &targetIDStr, &l.Metadata, &ipStr, &l.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan audit log: %w", err)
		}
		if targetIDStr != nil {
			parsed, err := uuid.Parse(*targetIDStr)
			if err == nil {
				l.TargetID = &parsed
			}
		}
		l.IPAddress = ipStr
		logs = append(logs, l)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}
