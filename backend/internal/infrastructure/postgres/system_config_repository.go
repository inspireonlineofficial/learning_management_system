package postgres

import (
	"context"
	"database/sql"
	"encoding/json"

	domainsysconfig "lms-backend/internal/domain/system_config"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// ─── SystemSettingRepository ──────────────────────────────────────────────────

// SystemSettingRepository implements domain/system_config.SystemSettingRepository.
type SystemSettingRepository struct {
	db *sql.DB
}

// NewSystemSettingRepository creates a new SystemSettingRepository.
func NewSystemSettingRepository(db *sql.DB) *SystemSettingRepository {
	return &SystemSettingRepository{db: db}
}

// Get returns the singleton system settings row (id=1).
func (r *SystemSettingRepository) Get(ctx context.Context) (*domainsysconfig.SystemSetting, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, platform_name, default_timezone, oauth_providers_enabled,
		       maintenance_mode, feature_flags, updated_at, updated_by
		FROM system_settings
		WHERE id = 1`)
	return scanSystemSetting(row)
}

// Update persists changes to the singleton settings record.
func (r *SystemSettingRepository) Update(ctx context.Context, s *domainsysconfig.SystemSetting) error {
	flagsJSON, err := json.Marshal(s.FeatureFlags)
	if err != nil {
		flagsJSON = []byte("{}")
	}
	_, err = r.db.ExecContext(ctx, `
		UPDATE system_settings
		SET platform_name = $1,
		    default_timezone = $2,
		    oauth_providers_enabled = $3,
		    maintenance_mode = $4,
		    feature_flags = $5,
		    updated_at = $6,
		    updated_by = $7
		WHERE id = 1`,
		s.PlatformName,
		s.DefaultTimezone,
		pq.Array(s.OAuthProvidersEnabled),
		s.MaintenanceMode,
		flagsJSON,
		s.UpdatedAt,
		s.UpdatedBy,
	)
	return err
}

func scanSystemSetting(row *sql.Row) (*domainsysconfig.SystemSetting, error) {
	var s domainsysconfig.SystemSetting
	var featureFlagsRaw []byte
	var updatedBy sql.NullString

	err := row.Scan(
		&s.ID,
		&s.PlatformName,
		&s.DefaultTimezone,
		pq.Array(&s.OAuthProvidersEnabled),
		&s.MaintenanceMode,
		&featureFlagsRaw,
		&s.UpdatedAt,
		&updatedBy,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	s.FeatureFlags = featureFlagsRaw
	if updatedBy.Valid {
		id, _ := uuid.Parse(updatedBy.String)
		s.UpdatedBy = &id
	}
	return &s, nil
}

// ─── SystemSettingHistoryRepository ──────────────────────────────────────────

// SystemSettingHistoryRepository implements domain/system_config.SystemSettingHistoryRepository.
// This is append-only — no Update or Delete methods.
type SystemSettingHistoryRepository struct {
	db *sql.DB
}

// NewSystemSettingHistoryRepository creates a new SystemSettingHistoryRepository.
func NewSystemSettingHistoryRepository(db *sql.DB) *SystemSettingHistoryRepository {
	return &SystemSettingHistoryRepository{db: db}
}

// Create appends a new history record.
func (r *SystemSettingHistoryRepository) Create(ctx context.Context, h *domainsysconfig.SystemSettingHistory) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO system_setting_history (id, changed_by, diff, snapshot, changed_at)
		VALUES ($1, $2, $3, $4, $5)`,
		h.ID,
		h.ChangedBy,
		h.Diff,
		h.Snapshot,
		h.ChangedAt,
	)
	return err
}

// FindByID returns a single history record by ID.
func (r *SystemSettingHistoryRepository) FindByID(ctx context.Context, id uuid.UUID) (*domainsysconfig.SystemSettingHistory, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, changed_by, diff, snapshot, changed_at
		FROM system_setting_history
		WHERE id = $1`, id)
	return scanHistoryRow(row)
}

// List returns paginated history records ordered by changed_at DESC.
func (r *SystemSettingHistoryRepository) List(ctx context.Context, page, limit int) ([]*domainsysconfig.SystemSettingHistory, int, error) {
	offset := (page - 1) * limit

	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM system_setting_history`).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, changed_by, diff, snapshot, changed_at
		FROM system_setting_history
		ORDER BY changed_at DESC
		LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var records []*domainsysconfig.SystemSettingHistory
	for rows.Next() {
		var h domainsysconfig.SystemSettingHistory
		if err := rows.Scan(&h.ID, &h.ChangedBy, &h.Diff, &h.Snapshot, &h.ChangedAt); err != nil {
			return nil, 0, err
		}
		records = append(records, &h)
	}
	return records, total, rows.Err()
}

func scanHistoryRow(row *sql.Row) (*domainsysconfig.SystemSettingHistory, error) {
	var h domainsysconfig.SystemSettingHistory
	err := row.Scan(&h.ID, &h.ChangedBy, &h.Diff, &h.Snapshot, &h.ChangedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &h, nil
}
