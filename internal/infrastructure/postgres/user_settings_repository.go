package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"lms-backend/internal/domain/auth"
	"time"

	"github.com/google/uuid"
)

// UserSettingsRepository implements auth.UserSettingsRepository.
type UserSettingsRepository struct {
	db *sql.DB
}

// NewUserSettingsRepository creates a user settings repository.
func NewUserSettingsRepository(db *sql.DB) *UserSettingsRepository {
	return &UserSettingsRepository{db: db}
}

// GetByUserID returns settings for a user.
func (r *UserSettingsRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*auth.UserSettings, error) {
	query := `
		SELECT user_id, email_notifications, push_notifications, newsletter_opt_in, language, timezone, created_at, updated_at
		FROM user_settings
		WHERE user_id = $1
	`
	settings := &auth.UserSettings{}
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&settings.UserID,
		&settings.EmailNotifications,
		&settings.PushNotifications,
		&settings.NewsletterOptIn,
		&settings.Language,
		&settings.Timezone,
		&settings.CreatedAt,
		&settings.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("user settings not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user settings: %w", err)
	}
	return settings, nil
}

// Upsert creates or updates settings for a user.
func (r *UserSettingsRepository) Upsert(ctx context.Context, settings *auth.UserSettings) error {
	now := time.Now().UTC()
	if settings.CreatedAt.IsZero() {
		settings.CreatedAt = now
	}
	settings.UpdatedAt = now

	query := `
		INSERT INTO user_settings (
			user_id, email_notifications, push_notifications, newsletter_opt_in,
			language, timezone, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (user_id) DO UPDATE SET
			email_notifications = EXCLUDED.email_notifications,
			push_notifications = EXCLUDED.push_notifications,
			newsletter_opt_in = EXCLUDED.newsletter_opt_in,
			language = EXCLUDED.language,
			timezone = EXCLUDED.timezone,
			updated_at = EXCLUDED.updated_at
	`
	_, err := r.db.ExecContext(
		ctx,
		query,
		settings.UserID,
		settings.EmailNotifications,
		settings.PushNotifications,
		settings.NewsletterOptIn,
		settings.Language,
		settings.Timezone,
		settings.CreatedAt,
		settings.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert user settings: %w", err)
	}
	return nil
}
