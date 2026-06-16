package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"lms-backend/internal/domain/auth"
	"strings"
	"time"

	"github.com/google/uuid"
)

// UserRepository implements auth.UserRepository
type UserRepository struct {
	db *sql.DB
}

// List returns users for admin management with optional filters.
func (r *UserRepository) List(ctx context.Context, role, status, search *string, fromDate, toDate *time.Time, page, limit int) ([]*auth.User, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	where := []string{"deleted_at IS NULL"}
	args := []any{}
	addArg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}

	if role != nil && *role != "" {
		where = append(where, "role = "+addArg(*role))
	}
	if status != nil && *status != "" {
		where = append(where, "status = "+addArg(*status))
	}
	if search != nil && strings.TrimSpace(*search) != "" {
		term := "%" + strings.ToLower(strings.TrimSpace(*search)) + "%"
		where = append(where, "(LOWER(full_name) LIKE "+addArg(term)+" OR LOWER(email) LIKE "+addArg(term)+")")
	}
	if fromDate != nil {
		where = append(where, "created_at >= "+addArg(*fromDate))
	}
	if toDate != nil {
		where = append(where, "created_at <= "+addArg(*toDate))
	}

	whereClause := strings.Join(where, " AND ")
	var total int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE "+whereClause, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	queryArgs := append([]any{}, args...)
	offsetParam := func(value any) string {
		queryArgs = append(queryArgs, value)
		return fmt.Sprintf("$%d", len(queryArgs))
	}
	query := `
		SELECT id, full_name, email, username, password_hash, role, status, profile_complete, created_at, updated_at, deleted_at
		FROM users
		WHERE ` + whereClause + `
		ORDER BY created_at DESC
		LIMIT ` + offsetParam(limit) + ` OFFSET ` + offsetParam((page-1)*limit)

	rows, err := r.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var result []*auth.User
	for rows.Next() {
		u := &auth.User{}
		if err := rows.Scan(
			&u.ID, &u.FullName, &u.Email, &u.Username, &u.PasswordHash,
			&u.Role, &u.Status, &u.ProfileComplete, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan user: %w", err)
		}
		result = append(result, u)
	}
	return result, total, rows.Err()
}

// NewUserRepository creates a new UserRepository
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create creates a new user
func (r *UserRepository) Create(ctx context.Context, u *auth.User) error {
	query := `
		INSERT INTO users (id, full_name, email, username, password_hash, role, status, profile_complete, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	u.ID = uuid.New()
	u.CreatedAt = time.Now().UTC()
	u.UpdatedAt = time.Now().UTC()

	_, err := r.db.ExecContext(ctx, query,
		u.ID, u.FullName, u.Email, u.Username, u.PasswordHash,
		u.Role, u.Status, u.ProfileComplete, u.CreatedAt, u.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

// FindByEmail finds a user by email
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*auth.User, error) {
	query := `
		SELECT id, full_name, email, username, password_hash, role, status, profile_complete, created_at, updated_at, deleted_at
		FROM users
		WHERE email = $1 AND deleted_at IS NULL
	`
	u := &auth.User{}
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&u.ID, &u.FullName, &u.Email, &u.Username, &u.PasswordHash,
		&u.Role, &u.Status, &u.ProfileComplete, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find user by email: %w", err)
	}
	return u, nil
}

// FindByUsername finds a user by username
func (r *UserRepository) FindByUsername(ctx context.Context, username string) (*auth.User, error) {
	query := `
		SELECT id, full_name, email, username, password_hash, role, status, profile_complete, created_at, updated_at, deleted_at
		FROM users
		WHERE username = $1 AND deleted_at IS NULL
	`
	u := &auth.User{}
	err := r.db.QueryRowContext(ctx, query, username).Scan(
		&u.ID, &u.FullName, &u.Email, &u.Username, &u.PasswordHash,
		&u.Role, &u.Status, &u.ProfileComplete, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find user by username: %w", err)
	}
	return u, nil
}

// FindByID finds a user by ID
func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (*auth.User, error) {
	query := `
		SELECT id, full_name, email, username, password_hash, role, status, profile_complete, created_at, updated_at, deleted_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`
	u := &auth.User{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&u.ID, &u.FullName, &u.Email, &u.Username, &u.PasswordHash,
		&u.Role, &u.Status, &u.ProfileComplete, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find user by ID: %w", err)
	}
	return u, nil
}

// Update updates a user
func (r *UserRepository) Update(ctx context.Context, u *auth.User) error {
	query := `
		UPDATE users
		SET full_name = $2, email = $3, username = $4, password_hash = $5, role = $6, status = $7, profile_complete = $8, updated_at = $9
		WHERE id = $1 AND deleted_at IS NULL
	`
	u.UpdatedAt = time.Now().UTC()

	result, err := r.db.ExecContext(ctx, query,
		u.ID, u.FullName, u.Email, u.Username, u.PasswordHash,
		u.Role, u.Status, u.ProfileComplete, u.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// SoftDelete soft-deletes a user
func (r *UserRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE users
		SET deleted_at = $2, updated_at = $2
		WHERE id = $1 AND deleted_at IS NULL
	`
	now := time.Now().UTC()
	result, err := r.db.ExecContext(ctx, query, id, now)
	if err != nil {
		return fmt.Errorf("failed to soft delete user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// OTPRepository implements auth.OTPRepository
type OTPRepository struct {
	db *sql.DB
}

// NewOTPRepository creates a new OTPRepository
func NewOTPRepository(db *sql.DB) *OTPRepository {
	return &OTPRepository{db: db}
}

// Store stores an OTP record
func (r *OTPRepository) Store(ctx context.Context, otp *auth.OTPRecord) error {
	query := `
		INSERT INTO otp_records (id, user_id, otp_hash, purpose, attempts, resend_count, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	otp.ID = uuid.New()
	otp.CreatedAt = time.Now().UTC()

	_, err := r.db.ExecContext(ctx, query,
		otp.ID, otp.UserID, otp.OTPHash, otp.Purpose,
		otp.Attempts, otp.ResendCount, otp.ExpiresAt, otp.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to store OTP: %w", err)
	}
	return nil
}

// FindByUserID finds the most recent OTP for a user and purpose
func (r *OTPRepository) FindByUserID(ctx context.Context, userID uuid.UUID, purpose string) (*auth.OTPRecord, error) {
	query := `
		SELECT id, user_id, otp_hash, purpose, attempts, resend_count, expires_at, invalidated_at, created_at
		FROM otp_records
		WHERE user_id = $1 AND purpose = $2 AND invalidated_at IS NULL
		ORDER BY created_at DESC
		LIMIT 1
	`
	otp := &auth.OTPRecord{}
	err := r.db.QueryRowContext(ctx, query, userID, purpose).Scan(
		&otp.ID, &otp.UserID, &otp.OTPHash, &otp.Purpose,
		&otp.Attempts, &otp.ResendCount, &otp.ExpiresAt, &otp.InvalidatedAt, &otp.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("OTP not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find OTP: %w", err)
	}
	return otp, nil
}

// IncrementAttempts increments the attempt counter
func (r *OTPRepository) IncrementAttempts(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE otp_records SET attempts = attempts + 1 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to increment attempts: %w", err)
	}
	return nil
}

// IncrementResendCount increments the resend counter
func (r *OTPRepository) IncrementResendCount(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE otp_records SET resend_count = resend_count + 1 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to increment resend count: %w", err)
	}
	return nil
}

// Invalidate marks an OTP as invalidated
func (r *OTPRepository) Invalidate(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE otp_records SET invalidated_at = $2 WHERE id = $1`
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, query, id, now)
	if err != nil {
		return fmt.Errorf("failed to invalidate OTP: %w", err)
	}
	return nil
}

// PasswordResetRepository implements auth.PasswordResetRepository
type PasswordResetRepository struct {
	db *sql.DB
}

// NewPasswordResetRepository creates a new PasswordResetRepository
func NewPasswordResetRepository(db *sql.DB) *PasswordResetRepository {
	return &PasswordResetRepository{db: db}
}

// Store stores a password reset token
func (r *PasswordResetRepository) Store(ctx context.Context, token *auth.PasswordResetToken) error {
	query := `
		INSERT INTO password_reset_tokens (id, user_id, token_hash, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	token.ID = uuid.New()
	token.CreatedAt = time.Now().UTC()

	_, err := r.db.ExecContext(ctx, query,
		token.ID, token.UserID, token.TokenHash, token.ExpiresAt, token.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to store password reset token: %w", err)
	}
	return nil
}

// FindByTokenHash finds a password reset token by its hash
func (r *PasswordResetRepository) FindByTokenHash(ctx context.Context, tokenHash string) (*auth.PasswordResetToken, error) {
	query := `
		SELECT id, user_id, token_hash, expires_at, used_at, created_at
		FROM password_reset_tokens
		WHERE token_hash = $1
	`
	token := &auth.PasswordResetToken{}
	err := r.db.QueryRowContext(ctx, query, tokenHash).Scan(
		&token.ID, &token.UserID, &token.TokenHash, &token.ExpiresAt, &token.UsedAt, &token.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("token not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find token: %w", err)
	}
	return token, nil
}

// MarkAsUsed marks a password reset token as used
func (r *PasswordResetRepository) MarkAsUsed(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE password_reset_tokens SET used_at = $2 WHERE id = $1`
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, query, id, now)
	if err != nil {
		return fmt.Errorf("failed to mark token as used: %w", err)
	}
	return nil
}

// OAuthProviderRepository implements auth.OAuthProviderRepository
type OAuthProviderRepository struct {
	db *sql.DB
}

// NewOAuthProviderRepository creates a new OAuthProviderRepository
func NewOAuthProviderRepository(db *sql.DB) *OAuthProviderRepository {
	return &OAuthProviderRepository{db: db}
}

// Create creates a new OAuth provider link
func (r *OAuthProviderRepository) Create(ctx context.Context, provider *auth.OAuthProvider) error {
	query := `
		INSERT INTO oauth_providers (id, user_id, provider, provider_user_id, access_token_encrypted, refresh_token_encrypted, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	provider.ID = uuid.New()
	provider.CreatedAt = time.Now().UTC()

	_, err := r.db.ExecContext(ctx, query,
		provider.ID, provider.UserID, provider.Provider, provider.ProviderUserID,
		provider.AccessTokenEncrypted, provider.RefreshTokenEncrypted, provider.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create OAuth provider: %w", err)
	}
	return nil
}

// FindByUserIDAndProvider finds an OAuth provider by user ID and provider name
func (r *OAuthProviderRepository) FindByUserIDAndProvider(ctx context.Context, userID uuid.UUID, provider string) (*auth.OAuthProvider, error) {
	query := `
		SELECT id, user_id, provider, provider_user_id, access_token_encrypted, refresh_token_encrypted, created_at
		FROM oauth_providers
		WHERE user_id = $1 AND provider = $2
	`
	p := &auth.OAuthProvider{}
	err := r.db.QueryRowContext(ctx, query, userID, provider).Scan(
		&p.ID, &p.UserID, &p.Provider, &p.ProviderUserID,
		&p.AccessTokenEncrypted, &p.RefreshTokenEncrypted, &p.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("OAuth provider not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find OAuth provider: %w", err)
	}
	return p, nil
}

// FindByProviderAndProviderUserID finds an OAuth provider by provider name and provider user ID
func (r *OAuthProviderRepository) FindByProviderAndProviderUserID(ctx context.Context, provider, providerUserID string) (*auth.OAuthProvider, error) {
	query := `
		SELECT id, user_id, provider, provider_user_id, access_token_encrypted, refresh_token_encrypted, created_at
		FROM oauth_providers
		WHERE provider = $1 AND provider_user_id = $2
	`
	p := &auth.OAuthProvider{}
	err := r.db.QueryRowContext(ctx, query, provider, providerUserID).Scan(
		&p.ID, &p.UserID, &p.Provider, &p.ProviderUserID,
		&p.AccessTokenEncrypted, &p.RefreshTokenEncrypted, &p.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("OAuth provider not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find OAuth provider: %w", err)
	}
	return p, nil
}

// ListByUserID lists all OAuth providers for a user
func (r *OAuthProviderRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]*auth.OAuthProvider, error) {
	query := `
		SELECT id, user_id, provider, provider_user_id, access_token_encrypted, refresh_token_encrypted, created_at
		FROM oauth_providers
		WHERE user_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list OAuth providers: %w", err)
	}
	defer rows.Close()

	var providers []*auth.OAuthProvider
	for rows.Next() {
		p := &auth.OAuthProvider{}
		err := rows.Scan(
			&p.ID, &p.UserID, &p.Provider, &p.ProviderUserID,
			&p.AccessTokenEncrypted, &p.RefreshTokenEncrypted, &p.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan OAuth provider: %w", err)
		}
		providers = append(providers, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating OAuth providers: %w", err)
	}

	return providers, nil
}

// Delete deletes an OAuth provider link
func (r *OAuthProviderRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM oauth_providers WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete OAuth provider: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("OAuth provider not found")
	}

	return nil
}

// Update updates an OAuth provider
func (r *OAuthProviderRepository) Update(ctx context.Context, provider *auth.OAuthProvider) error {
	query := `
		UPDATE oauth_providers
		SET access_token_encrypted = $2, refresh_token_encrypted = $3
		WHERE id = $1
	`
	result, err := r.db.ExecContext(ctx, query,
		provider.ID, provider.AccessTokenEncrypted, provider.RefreshTokenEncrypted,
	)
	if err != nil {
		return fmt.Errorf("failed to update OAuth provider: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("OAuth provider not found")
	}

	return nil
}
