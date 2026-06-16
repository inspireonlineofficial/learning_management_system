package postgres

import (
	"context"
	"database/sql"
	"time"

	domainnotif "lms-backend/internal/domain/notifications"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// ─── NotificationRepository ───────────────────────────────────────────────────

// NotificationRepository implements domainnotif.NotificationRepository.
type NotificationRepository struct {
	db *sql.DB
}

// NewNotificationRepository creates a new NotificationRepository.
func NewNotificationRepository(db *sql.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

func (r *NotificationRepository) Create(ctx context.Context, n *domainnotif.Notification) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO notifications (id, user_id, type, title, body, is_read, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		n.ID, n.UserID, n.Type, n.Title, n.Body, n.IsRead, n.CreatedAt,
	)
	return err
}

func (r *NotificationRepository) FindByID(ctx context.Context, id uuid.UUID) (*domainnotif.Notification, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, type, title, body, is_read, created_at
		FROM notifications WHERE id = $1`, id)
	return scanNotification(row)
}

func (r *NotificationRepository) ListByUserID(ctx context.Context, userID uuid.UUID, page, limit int) ([]*domainnotif.Notification, int, error) {
	offset := (page - 1) * limit

	var total int
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM notifications WHERE user_id = $1`, userID).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, type, title, body, is_read, created_at
		FROM notifications WHERE user_id = $1
		ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []*domainnotif.Notification
	for rows.Next() {
		n, err := scanNotificationRow(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, n)
	}
	return items, total, rows.Err()
}

func (r *NotificationRepository) CountUnread(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = false`, userID).Scan(&count)
	return count, err
}

func (r *NotificationRepository) MarkRead(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE notifications SET is_read = true WHERE id = $1`, id)
	return err
}

func (r *NotificationRepository) MarkAllRead(ctx context.Context, userID uuid.UUID) (int, error) {
	result, err := r.db.ExecContext(ctx,
		`UPDATE notifications SET is_read = true WHERE user_id = $1 AND is_read = false`, userID)
	if err != nil {
		return 0, err
	}
	count, err := result.RowsAffected()
	return int(count), err
}

func (r *NotificationRepository) ListAllUserIDs(ctx context.Context, role *string) ([]uuid.UUID, error) {
	var rows *sql.Rows
	var err error

	if role != nil {
		rows, err = r.db.QueryContext(ctx,
			`SELECT id FROM users WHERE role = $1 AND status = 'active' AND deleted_at IS NULL`, *role)
	} else {
		rows, err = r.db.QueryContext(ctx,
			`SELECT id FROM users WHERE status = 'active' AND deleted_at IS NULL`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func scanNotification(row *sql.Row) (*domainnotif.Notification, error) {
	var n domainnotif.Notification
	err := row.Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Body, &n.IsRead, &n.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &n, err
}

func scanNotificationRow(rows *sql.Rows) (*domainnotif.Notification, error) {
	var n domainnotif.Notification
	err := rows.Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Body, &n.IsRead, &n.CreatedAt)
	return &n, err
}

// ─── NotificationTemplateRepository ──────────────────────────────────────────

// NotificationTemplateRepository implements domainnotif.NotificationTemplateRepository.
type NotificationTemplateRepository struct {
	db *sql.DB
}

// NewNotificationTemplateRepository creates a new NotificationTemplateRepository.
func NewNotificationTemplateRepository(db *sql.DB) *NotificationTemplateRepository {
	return &NotificationTemplateRepository{db: db}
}

func (r *NotificationTemplateRepository) FindByID(ctx context.Context, id uuid.UUID) (*domainnotif.NotificationTemplate, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, type, channel, subject_template, body_template, allowed_variables, updated_at
		FROM notification_templates WHERE id = $1`, id)
	return scanTemplate(row)
}

func (r *NotificationTemplateRepository) FindByType(ctx context.Context, notifType string) (*domainnotif.NotificationTemplate, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, type, channel, subject_template, body_template, allowed_variables, updated_at
		FROM notification_templates WHERE type = $1`, notifType)
	return scanTemplate(row)
}

func (r *NotificationTemplateRepository) Update(ctx context.Context, t *domainnotif.NotificationTemplate) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE notification_templates
		SET subject_template = $1, body_template = $2, updated_at = $3
		WHERE id = $4`,
		t.SubjectTemplate, t.BodyTemplate, t.UpdatedAt, t.ID,
	)
	return err
}

func (r *NotificationTemplateRepository) List(ctx context.Context) ([]*domainnotif.NotificationTemplate, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, type, channel, subject_template, body_template, allowed_variables, updated_at
		FROM notification_templates ORDER BY type`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []*domainnotif.NotificationTemplate
	for rows.Next() {
		t, err := scanTemplateRow(rows)
		if err != nil {
			return nil, err
		}
		templates = append(templates, t)
	}
	return templates, rows.Err()
}

func scanTemplate(row *sql.Row) (*domainnotif.NotificationTemplate, error) {
	var t domainnotif.NotificationTemplate
	var allowedVars pq.StringArray
	err := row.Scan(&t.ID, &t.Type, &t.Channel, &t.SubjectTemplate, &t.BodyTemplate, &allowedVars, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	t.AllowedVariables = []string(allowedVars)
	return &t, nil
}

func scanTemplateRow(rows *sql.Rows) (*domainnotif.NotificationTemplate, error) {
	var t domainnotif.NotificationTemplate
	var allowedVars pq.StringArray
	err := rows.Scan(&t.ID, &t.Type, &t.Channel, &t.SubjectTemplate, &t.BodyTemplate, &allowedVars, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	t.AllowedVariables = []string(allowedVars)
	return &t, nil
}

// ─── BroadcastWorker helper ───────────────────────────────────────────────────

// BroadcastNotificationHandler processes broadcast_notification jobs by fanning out
// individual send_notification inserts in batches.
// Requirements: 22.6, 22.7
type BroadcastNotificationHandler struct {
	notifRepo domainnotif.NotificationRepository
	db        *sql.DB
}

// NewBroadcastNotificationHandler creates a new BroadcastNotificationHandler.
func NewBroadcastNotificationHandler(notifRepo domainnotif.NotificationRepository, db *sql.DB) *BroadcastNotificationHandler {
	return &BroadcastNotificationHandler{notifRepo: notifRepo, db: db}
}

// FanOut inserts notifications for all target users in batches of 500.
func (h *BroadcastNotificationHandler) FanOut(ctx context.Context, role *string, notifType, title, body string) error {
	userIDs, err := h.notifRepo.ListAllUserIDs(ctx, role)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	const batchSize = 500

	for i := 0; i < len(userIDs); i += batchSize {
		end := i + batchSize
		if end > len(userIDs) {
			end = len(userIDs)
		}
		batch := userIDs[i:end]

		tx, err := h.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}

		stmt, err := tx.PrepareContext(ctx, `
			INSERT INTO notifications (id, user_id, type, title, body, is_read, created_at)
			VALUES ($1, $2, $3, $4, $5, false, $6)`)
		if err != nil {
			tx.Rollback()
			return err
		}

		for _, uid := range batch {
			if _, err := stmt.ExecContext(ctx, uuid.New(), uid, notifType, title, body, now); err != nil {
				stmt.Close()
				tx.Rollback()
				return err
			}
		}
		stmt.Close()

		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}
