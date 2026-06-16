package auth

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// User represents an authenticated account in the system
type User struct {
	ID              uuid.UUID  `json:"id"`
	FullName        string     `json:"full_name"`
	Email           string     `json:"email"`
	Username        *string    `json:"username,omitempty"` // Admin only
	PasswordHash    *string    `json:"-"`                  // Never exposed in JSON
	Role            string     `json:"role"`               // student, teacher, admin
	Status          string     `json:"status"`             // active, inactive
	ProfileComplete bool       `json:"profile_complete"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	DeletedAt       *time.Time `json:"deleted_at,omitempty"`
}

// OTPRecord represents a one-time password for verification
type OTPRecord struct {
	ID            uuid.UUID  `json:"id"`
	UserID        uuid.UUID  `json:"user_id"`
	OTPHash       string     `json:"-"` // Never exposed
	Purpose       string     `json:"purpose"`
	Attempts      int        `json:"attempts"`
	ResendCount   int        `json:"resend_count"`
	ExpiresAt     time.Time  `json:"expires_at"`
	InvalidatedAt *time.Time `json:"invalidated_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

// OAuthProvider represents a linked OAuth provider for a user
type OAuthProvider struct {
	ID                    uuid.UUID `json:"id"`
	UserID                uuid.UUID `json:"user_id"`
	Provider              string    `json:"provider"` // google, github, microsoft
	ProviderUserID        string    `json:"provider_user_id"`
	AccessTokenEncrypted  *string   `json:"-"` // Never exposed
	RefreshTokenEncrypted *string   `json:"-"` // Never exposed
	CreatedAt             time.Time `json:"created_at"`
}

// PasswordResetToken represents a single-use password reset token
type PasswordResetToken struct {
	ID        uuid.UUID  `json:"id"`
	UserID    uuid.UUID  `json:"user_id"`
	TokenHash string     `json:"-"` // Never exposed
	ExpiresAt time.Time  `json:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// UserRepository defines the interface for user persistence
type UserRepository interface {
	Create(ctx context.Context, u *User) error
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByUsername(ctx context.Context, username string) (*User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*User, error)
	Update(ctx context.Context, u *User) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
}

// OTPRepository defines the interface for OTP persistence
type OTPRepository interface {
	Store(ctx context.Context, otp *OTPRecord) error
	FindByUserID(ctx context.Context, userID uuid.UUID, purpose string) (*OTPRecord, error)
	IncrementAttempts(ctx context.Context, id uuid.UUID) error
	IncrementResendCount(ctx context.Context, id uuid.UUID) error
	Invalidate(ctx context.Context, id uuid.UUID) error
}

// PasswordResetRepository defines the interface for password reset token persistence
type PasswordResetRepository interface {
	Store(ctx context.Context, token *PasswordResetToken) error
	FindByTokenHash(ctx context.Context, tokenHash string) (*PasswordResetToken, error)
	MarkAsUsed(ctx context.Context, id uuid.UUID) error
}

// OAuthProviderRepository defines the interface for OAuth provider persistence
type OAuthProviderRepository interface {
	Create(ctx context.Context, provider *OAuthProvider) error
	FindByUserIDAndProvider(ctx context.Context, userID uuid.UUID, provider string) (*OAuthProvider, error)
	FindByProviderAndProviderUserID(ctx context.Context, provider, providerUserID string) (*OAuthProvider, error)
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]*OAuthProvider, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Update(ctx context.Context, provider *OAuthProvider) error
}

// TokenStore defines the interface for refresh token operations in Redis
type TokenStore interface {
	StoreRefreshToken(ctx context.Context, userID uuid.UUID, token string, ttl time.Duration) error
	ValidateRefreshToken(ctx context.Context, token string) (uuid.UUID, error)
	DeleteRefreshToken(ctx context.Context, token string) error
	DeleteAllRefreshTokens(ctx context.Context, userID uuid.UUID) error
}
