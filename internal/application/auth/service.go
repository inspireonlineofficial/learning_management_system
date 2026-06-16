package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"lms-backend/internal/domain/auth"
	infraoauth "lms-backend/internal/infrastructure/oauth"
	"lms-backend/pkg/apperrors"
	"lms-backend/pkg/logger"
	"lms-backend/pkg/validator"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Service defines the auth service interface
type Service interface {
	// Registration and OTP
	Register(ctx context.Context, cmd RegisterCommand) (*RegisterResult, error)
	VerifyOTP(ctx context.Context, cmd VerifyOTPCommand) (*TokenPair, error)
	ResendOTP(ctx context.Context, cmd ResendOTPCommand) error

	// Login, Refresh, Logout
	Login(ctx context.Context, cmd LoginCommand) (*TokenPair, error)
	RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error)
	Logout(ctx context.Context, refreshToken string) error

	// Password Reset and Account Management
	ForgotPassword(ctx context.Context, cmd ForgotPasswordCommand) error
	ResetPassword(ctx context.Context, cmd ResetPasswordCommand) error
	ChangePassword(ctx context.Context, userID uuid.UUID, cmd ChangePasswordCommand) error
	UpdateProfile(ctx context.Context, userID uuid.UUID, cmd UpdateProfileCommand) (*ProfileResult, error)
	GetProfile(ctx context.Context, userID uuid.UUID) (*ProfileResult, error)

	// Admin 2FA
	AdminLogin(ctx context.Context, cmd AdminLoginCommand) (*OTPSessionResult, error)
	AdminVerifyOTP(ctx context.Context, cmd AdminVerifyOTPCommand) (*TokenPair, error)
	AdminResendOTP(ctx context.Context, otpSessionToken string) error
	ImpersonateUser(ctx context.Context, cmd ImpersonateUserCommand) (*ImpersonationResult, error)

	// OAuth
	OAuthRedirect(ctx context.Context, cmd OAuthRedirectCommand) (string, error)
	OAuthCallback(ctx context.Context, cmd OAuthCallbackCommand) (*OAuthResult, error)
	ConnectProvider(ctx context.Context, userID uuid.UUID, cmd ConnectProviderCommand) error
	DisconnectProvider(ctx context.Context, userID uuid.UUID, cmd DisconnectProviderCommand) error
	ListProviders(ctx context.Context, userID uuid.UUID) ([]*OAuthProviderResult, error)
}

// Dependencies for the auth service
type ServiceDeps struct {
	UserRepo          auth.UserRepository
	OTPRepo           auth.OTPRepository
	PasswordResetRepo auth.PasswordResetRepository
	OAuthProviderRepo auth.OAuthProviderRepository
	TokenStore        auth.TokenStore
	JWTService        JWTService
	OAuthFactory      OAuthProviderFactory
	TokenEncryptor    TokenEncryptor
	EmailService      EmailService
	RedisClient       RedisClient
	AuditLogger       AuditLogger
	FrontendBaseURL   string
	AdminDevBypass    bool
	AdminDevOTP       string
}

// JWTService interface for JWT operations
type JWTService interface {
	IssueToken(userID uuid.UUID, role, email string) (string, error)
	VerifyToken(tokenString string) (userID string, role string, email string, err error)
}

// OAuthProviderFactory interface for OAuth providers
type OAuthProviderFactory interface {
	GetProvider(name string) (OAuthProvider, error)
	IsProviderEnabled(name string) bool
}

// OAuthProvider interface for OAuth operations
type OAuthProvider interface {
	GetAuthURL(state string) string
	ExchangeCode(ctx context.Context, code string) (accessToken, refreshToken string, err error)
	GetUserInfo(ctx context.Context, accessToken string) (*OAuthUserInfo, error)
}

// OAuthUserInfo represents user info from OAuth provider.
type OAuthUserInfo = infraoauth.OAuthUserInfo

// TokenEncryptor interface for token encryption
type TokenEncryptor interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(ciphertext string) (string, error)
}

// EmailService interface for sending emails
type EmailService interface {
	SendOTP(ctx context.Context, email, otp string, expiresAt time.Time) error
	SendPasswordReset(ctx context.Context, email, resetLink string) error
	SendWelcome(ctx context.Context, email, name string) error
}

// RedisClient interface for Redis operations
type RedisClient interface {
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	GetDel(ctx context.Context, key string) (string, error)
	Del(ctx context.Context, keys ...string) error
	Incr(ctx context.Context, key string) (int64, error)
	Expire(ctx context.Context, key string, ttl time.Duration) error
}

// AuditLogger interface for audit logging
type AuditLogger interface {
	LogAdminLogin(ctx context.Context, userID uuid.UUID, username, ipAddress string, success bool) error
}

// authService implements the Service interface
type authService struct {
	deps ServiceDeps
}

// NewService creates a new auth service
func NewService(deps ServiceDeps) Service {
	return &authService{deps: deps}
}

// Register implements registration with OTP
func (s *authService) Register(ctx context.Context, cmd RegisterCommand) (*RegisterResult, error) {
	// Validate input
	if err := s.validateRegistration(cmd); err != nil {
		return nil, err
	}

	// Check if email already exists
	existingUser, err := s.deps.UserRepo.FindByEmail(ctx, cmd.Email)
	if err == nil && existingUser != nil {
		return nil, apperrors.ErrEmailExists
	}

	// Create user (inactive until OTP verified)
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(cmd.Password), 12)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &auth.User{
		FullName:        cmd.FullName,
		Email:           strings.ToLower(strings.TrimSpace(cmd.Email)),
		PasswordHash:    stringPtr(string(passwordHash)),
		Role:            cmd.Role,
		Status:          "inactive",
		ProfileComplete: false,
	}

	if err := s.deps.UserRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Generate and store OTP
	otp := generateOTP()
	otpHash, err := bcrypt.GenerateFromPassword([]byte(otp), 12)
	if err != nil {
		return nil, fmt.Errorf("failed to hash OTP: %w", err)
	}

	expiresAt := time.Now().UTC().Add(10 * time.Minute)
	otpRecord := &auth.OTPRecord{
		UserID:      user.ID,
		OTPHash:     string(otpHash),
		Purpose:     "registration",
		Attempts:    0,
		ResendCount: 0,
		ExpiresAt:   expiresAt,
	}

	if err := s.deps.OTPRepo.Store(ctx, otpRecord); err != nil {
		return nil, fmt.Errorf("failed to store OTP: %w", err)
	}

	// Send OTP email (async via job queue would be better in production)
	if err := s.deps.EmailService.SendOTP(ctx, user.Email, otp, expiresAt); err != nil {
		logger.Error(ctx, "failed to send OTP email", "error", err)
		// Don't fail the registration if email fails
	}

	return &RegisterResult{
		Message:   "Registration successful. Please check your email for the OTP.",
		Email:     maskEmail(user.Email),
		ExpiresAt: expiresAt,
	}, nil
}

// VerifyOTP implements OTP verification
func (s *authService) VerifyOTP(ctx context.Context, cmd VerifyOTPCommand) (*TokenPair, error) {
	// Find user
	user, err := s.deps.UserRepo.FindByEmail(ctx, cmd.Email)
	if err != nil {
		return nil, apperrors.ErrInvalidCredentials
	}

	// Find OTP record
	otpRecord, err := s.deps.OTPRepo.FindByUserID(ctx, user.ID, cmd.Purpose)
	if err != nil {
		return nil, &apperrors.AppError{
			Code:       "OTP_NOT_FOUND",
			Message:    "No OTP found for this email",
			HTTPStatus: 400,
		}
	}

	// Check if OTP is expired
	if time.Now().UTC().After(otpRecord.ExpiresAt) {
		return nil, apperrors.ErrOTPExpired
	}

	// Check if OTP is invalidated
	if otpRecord.InvalidatedAt != nil {
		return nil, apperrors.ErrOTPMaxAttempts
	}

	// Verify OTP
	if err := bcrypt.CompareHashAndPassword([]byte(otpRecord.OTPHash), []byte(cmd.OTP)); err != nil {
		// Increment attempts
		_ = s.deps.OTPRepo.IncrementAttempts(ctx, otpRecord.ID)

		// Check if max attempts reached
		if otpRecord.Attempts+1 >= 5 {
			_ = s.deps.OTPRepo.Invalidate(ctx, otpRecord.ID)
			return nil, apperrors.ErrOTPMaxAttempts
		}

		return nil, &apperrors.AppError{
			Code:       "INVALID_OTP",
			Message:    "Invalid OTP",
			HTTPStatus: 400,
		}
	}

	// Activate user
	user.Status = "active"
	if err := s.deps.UserRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to activate user: %w", err)
	}

	// Invalidate OTP
	_ = s.deps.OTPRepo.Invalidate(ctx, otpRecord.ID)

	// Issue tokens
	return s.issueTokens(ctx, user, false)
}

// ResendOTP implements OTP resend with rate limiting
func (s *authService) ResendOTP(ctx context.Context, cmd ResendOTPCommand) error {
	// Find user
	user, err := s.deps.UserRepo.FindByEmail(ctx, cmd.Email)
	if err != nil {
		// Don't reveal if email exists
		return nil
	}

	// Find existing OTP
	otpRecord, err := s.deps.OTPRepo.FindByUserID(ctx, user.ID, cmd.Purpose)
	if err != nil {
		return &apperrors.AppError{
			Code:       "OTP_NOT_FOUND",
			Message:    "No OTP found for this email",
			HTTPStatus: 400,
		}
	}

	// Check resend limit
	if otpRecord.ResendCount >= 3 {
		return apperrors.ErrOTPResendLimit
	}

	// Check cooldown (60 seconds)
	if time.Since(otpRecord.CreatedAt) < 60*time.Second {
		return &apperrors.AppError{
			Code:       "OTP_COOLDOWN",
			Message:    "Please wait before requesting another OTP",
			HTTPStatus: 429,
		}
	}

	// Generate new OTP
	otp := generateOTP()
	otpHash, err := bcrypt.GenerateFromPassword([]byte(otp), 12)
	if err != nil {
		return fmt.Errorf("failed to hash OTP: %w", err)
	}

	// Invalidate old OTP
	_ = s.deps.OTPRepo.Invalidate(ctx, otpRecord.ID)

	// Store new OTP
	expiresAt := time.Now().UTC().Add(10 * time.Minute)
	newOTPRecord := &auth.OTPRecord{
		UserID:      user.ID,
		OTPHash:     string(otpHash),
		Purpose:     cmd.Purpose,
		Attempts:    0,
		ResendCount: otpRecord.ResendCount + 1,
		ExpiresAt:   expiresAt,
	}

	if err := s.deps.OTPRepo.Store(ctx, newOTPRecord); err != nil {
		return fmt.Errorf("failed to store OTP: %w", err)
	}

	// Send OTP email
	if err := s.deps.EmailService.SendOTP(ctx, user.Email, otp, expiresAt); err != nil {
		logger.Error(ctx, "failed to send OTP email", "error", err)
	}

	return nil
}

// Helper functions

func (s *authService) validateRegistration(cmd RegisterCommand) error {
	var details []map[string]string

	// Validate full name
	if len(cmd.FullName) < 2 || len(cmd.FullName) > 100 {
		details = append(details, map[string]string{
			"field":   "full_name",
			"message": "must be between 2 and 100 characters",
		})
	}

	// Validate email
	if !validator.IsValidEmail(cmd.Email) {
		details = append(details, map[string]string{
			"field":   "email",
			"message": "must be a valid email address",
		})
	}

	// Validate password
	if len(cmd.Password) < 8 {
		details = append(details, map[string]string{
			"field":   "password",
			"message": "must be at least 8 characters",
		})
	}

	if !validator.ContainsDigit(cmd.Password) {
		details = append(details, map[string]string{
			"field":   "password",
			"message": "must contain at least one digit",
		})
	}

	// Validate password confirmation
	if cmd.Password != cmd.ConfirmPassword {
		details = append(details, map[string]string{
			"field":   "confirm_password",
			"message": "passwords do not match",
		})
	}

	// Validate role
	if cmd.Role != "student" && cmd.Role != "teacher" {
		details = append(details, map[string]string{
			"field":   "role",
			"message": "must be either 'student' or 'teacher'",
		})
	}

	if len(details) > 0 {
		return apperrors.NewValidationError(details)
	}

	return nil
}

func (s *authService) issueTokens(ctx context.Context, user *auth.User, rememberMe bool) (*TokenPair, error) {
	// Issue JWT access token
	accessToken, err := s.deps.JWTService.IssueToken(user.ID, user.Role, user.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to issue access token: %w", err)
	}

	// Generate refresh token
	refreshToken := generateRefreshToken()

	// Determine TTL
	ttl := 24 * time.Hour
	if rememberMe {
		ttl = 30 * 24 * time.Hour
	}

	// Store refresh token
	if err := s.deps.TokenStore.StoreRefreshToken(ctx, user.ID, refreshToken, ttl); err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    900, // 15 minutes
	}, nil
}

// ImpersonateUser issues an admin-scoped temporary session for another active user.
func (s *authService) ImpersonateUser(ctx context.Context, cmd ImpersonateUserCommand) (*ImpersonationResult, error) {
	actor, err := s.deps.UserRepo.FindByID(ctx, cmd.ActorID)
	if err != nil {
		return nil, apperrors.ErrUnauthorized
	}
	if actor.Role != "admin" || actor.Status != "active" {
		return nil, apperrors.NewForbiddenError("FORBIDDEN", "only active admins can impersonate users")
	}
	if actor.ID == cmd.TargetUserID {
		return nil, apperrors.NewSimpleValidationError("INVALID_TARGET", "cannot impersonate your own account")
	}

	target, err := s.deps.UserRepo.FindByID(ctx, cmd.TargetUserID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("USER_NOT_FOUND", "target user not found")
	}
	if target.Status != "active" {
		return nil, apperrors.NewForbiddenError("USER_INACTIVE", "target user is not active")
	}

	tokens, err := s.issueTokens(ctx, target, false)
	if err != nil {
		return nil, err
	}

	return &ImpersonationResult{
		TokenPair: *tokens,
		User: ProfileResult{
			ID:              target.ID.String(),
			FullName:        target.FullName,
			Email:           target.Email,
			Role:            target.Role,
			ProfileComplete: target.ProfileComplete,
			CreatedAt:       target.CreatedAt,
			UpdatedAt:       target.UpdatedAt,
		},
	}, nil
}

func generateOTP() string {
	// Generate a 6-digit OTP
	max := big.NewInt(1000000)
	n, _ := rand.Int(rand.Reader, max)
	return fmt.Sprintf("%06d", n.Int64())
}

func generateRefreshToken() string {
	// Generate a 256-bit random token
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func maskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return email
	}

	local := parts[0]
	domain := parts[1]

	if len(local) <= 2 {
		return email
	}

	masked := string(local[0]) + strings.Repeat("*", len(local)-2) + string(local[len(local)-1])
	return masked + "@" + domain
}

func stringPtr(s string) *string {
	return &s
}
