package auth

import (
	"context"
	"lms-backend/internal/domain/auth"
	"lms-backend/pkg/apperrors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"pgregory.net/rapid"
)

// Mock implementations for testing

type mockUserRepo struct {
	users map[string]*auth.User
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{users: make(map[string]*auth.User)}
}

func (m *mockUserRepo) Create(ctx context.Context, u *auth.User) error {
	if _, exists := m.users[u.Email]; exists {
		return apperrors.ErrEmailExists
	}
	u.ID = uuid.New()
	u.CreatedAt = time.Now().UTC()
	u.UpdatedAt = time.Now().UTC()
	m.users[u.Email] = u
	return nil
}

func (m *mockUserRepo) FindByEmail(ctx context.Context, email string) (*auth.User, error) {
	if u, ok := m.users[email]; ok {
		return u, nil
	}
	return nil, apperrors.ErrUserNotFound
}

func (m *mockUserRepo) FindByUsername(ctx context.Context, username string) (*auth.User, error) {
	for _, u := range m.users {
		if u.Username != nil && *u.Username == username {
			return u, nil
		}
	}
	return nil, apperrors.ErrUserNotFound
}

func (m *mockUserRepo) FindByID(ctx context.Context, id uuid.UUID) (*auth.User, error) {
	for _, u := range m.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, apperrors.ErrUserNotFound
}

func (m *mockUserRepo) Update(ctx context.Context, u *auth.User) error {
	if _, ok := m.users[u.Email]; !ok {
		return apperrors.ErrUserNotFound
	}
	u.UpdatedAt = time.Now().UTC()
	m.users[u.Email] = u
	return nil
}

func (m *mockUserRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	for _, u := range m.users {
		if u.ID == id {
			now := time.Now().UTC()
			u.DeletedAt = &now
			return nil
		}
	}
	return apperrors.ErrUserNotFound
}

type mockOTPRepo struct {
	otps map[uuid.UUID]*auth.OTPRecord
}

func newMockOTPRepo() *mockOTPRepo {
	return &mockOTPRepo{otps: make(map[uuid.UUID]*auth.OTPRecord)}
}

func (m *mockOTPRepo) Store(ctx context.Context, otp *auth.OTPRecord) error {
	otp.ID = uuid.New()
	otp.CreatedAt = time.Now().UTC()
	m.otps[otp.UserID] = otp
	return nil
}

func (m *mockOTPRepo) FindByUserID(ctx context.Context, userID uuid.UUID, purpose string) (*auth.OTPRecord, error) {
	if otp, ok := m.otps[userID]; ok && otp.Purpose == purpose {
		return otp, nil
	}
	return nil, apperrors.ErrOTPNotFound
}

func (m *mockOTPRepo) IncrementAttempts(ctx context.Context, id uuid.UUID) error {
	for _, otp := range m.otps {
		if otp.ID == id {
			otp.Attempts++
			return nil
		}
	}
	return apperrors.ErrOTPNotFound
}

func (m *mockOTPRepo) IncrementResendCount(ctx context.Context, id uuid.UUID) error {
	for _, otp := range m.otps {
		if otp.ID == id {
			otp.ResendCount++
			return nil
		}
	}
	return apperrors.ErrOTPNotFound
}

func (m *mockOTPRepo) Invalidate(ctx context.Context, id uuid.UUID) error {
	for _, otp := range m.otps {
		if otp.ID == id {
			now := time.Now().UTC()
			otp.InvalidatedAt = &now
			return nil
		}
	}
	return apperrors.ErrOTPNotFound
}

type mockTokenStore struct {
	tokens map[string]uuid.UUID
}

func newMockTokenStore() *mockTokenStore {
	return &mockTokenStore{tokens: make(map[string]uuid.UUID)}
}

func (m *mockTokenStore) StoreRefreshToken(ctx context.Context, userID uuid.UUID, token string, ttl time.Duration) error {
	m.tokens[token] = userID
	return nil
}

func (m *mockTokenStore) ValidateRefreshToken(ctx context.Context, token string) (uuid.UUID, error) {
	if userID, ok := m.tokens[token]; ok {
		return userID, nil
	}
	return uuid.Nil, apperrors.ErrInvalidRefreshToken
}

func (m *mockTokenStore) DeleteRefreshToken(ctx context.Context, token string) error {
	delete(m.tokens, token)
	return nil
}

func (m *mockTokenStore) DeleteAllRefreshTokens(ctx context.Context, userID uuid.UUID) error {
	for token, uid := range m.tokens {
		if uid == userID {
			delete(m.tokens, token)
		}
	}
	return nil
}

type mockJWTService struct{}

func (m *mockJWTService) IssueToken(userID uuid.UUID, role, email string) (string, error) {
	return "mock.jwt.token", nil
}

func (m *mockJWTService) VerifyToken(tokenString string) (userID string, role string, email string, err error) {
	return uuid.New().String(), "student", "test@example.com", nil
}

type mockEmailService struct{}

func (m *mockEmailService) SendOTP(ctx context.Context, email, otp string, expiresAt time.Time) error {
	return nil
}

func (m *mockEmailService) SendPasswordReset(ctx context.Context, email, resetLink string) error {
	return nil
}

func (m *mockEmailService) SendWelcome(ctx context.Context, email, name string) error {
	return nil
}

// Helper to create a test service
func createTestService() Service {
	return NewService(ServiceDeps{
		UserRepo:     newMockUserRepo(),
		OTPRepo:      newMockOTPRepo(),
		TokenStore:   newMockTokenStore(),
		JWTService:   &mockJWTService{},
		EmailService: &mockEmailService{},
	})
}

// **Property 12: Registration rejects invalid input fields**
// Validates: Requirements 2.2
func TestProperty12_RegistrationRejectsInvalidInput(t *testing.T) {
	service := createTestService()

	t.Run("invalid full_name length", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			nameLen := rapid.OneOf(
				rapid.IntRange(0, 1),
				rapid.IntRange(101, 200),
			).Draw(t, "nameLen")

			cmd := RegisterCommand{
				FullName:        strings.Repeat("a", nameLen),
				Email:           "valid@example.com",
				Password:        "password123",
				ConfirmPassword: "password123",
				Role:            "student",
			}

			_, err := service.Register(context.Background(), cmd)
			if err == nil {
				t.Fatalf("expected error for invalid full_name length %d", nameLen)
			}
		})
	})

	t.Run("invalid email format", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			invalidEmail := rapid.StringMatching("[a-z]+").Draw(t, "email")

			cmd := RegisterCommand{
				FullName:        "Valid Name",
				Email:           invalidEmail,
				Password:        "password123",
				ConfirmPassword: "password123",
				Role:            "student",
			}

			_, err := service.Register(context.Background(), cmd)
			if err == nil {
				t.Fatalf("expected error for invalid email: %s", invalidEmail)
			}
		})
	})

	t.Run("password too short", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			pwdLen := rapid.IntRange(0, 7).Draw(t, "pwdLen")
			pwd := strings.Repeat("a", pwdLen)

			cmd := RegisterCommand{
				FullName:        "Valid Name",
				Email:           "valid@example.com",
				Password:        pwd,
				ConfirmPassword: pwd,
				Role:            "student",
			}

			_, err := service.Register(context.Background(), cmd)
			if err == nil {
				t.Fatalf("expected error for password length %d", pwdLen)
			}
		})
	})

	t.Run("password without digit", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			pwd := rapid.StringMatching("[a-zA-Z]{8,20}").Draw(t, "password")

			cmd := RegisterCommand{
				FullName:        "Valid Name",
				Email:           "valid@example.com",
				Password:        pwd,
				ConfirmPassword: pwd,
				Role:            "student",
			}

			_, err := service.Register(context.Background(), cmd)
			if err == nil {
				t.Fatalf("expected error for password without digit: %s", pwd)
			}
		})
	})

	t.Run("password mismatch", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			pwd1 := rapid.StringMatching("[a-zA-Z0-9]{8,20}").Draw(t, "password1")
			pwd2 := rapid.StringMatching("[a-zA-Z0-9]{8,20}").Draw(t, "password2")

			if pwd1 == pwd2 {
				t.Skip("passwords match")
			}

			cmd := RegisterCommand{
				FullName:        "Valid Name",
				Email:           "valid@example.com",
				Password:        pwd1,
				ConfirmPassword: pwd2,
				Role:            "student",
			}

			_, err := service.Register(context.Background(), cmd)
			if err == nil {
				t.Fatal("expected error for password mismatch")
			}
		})
	})
}

// **Property 13: OTP is stored as bcrypt hash, never plaintext**
// Validates: Requirements 2.4
func TestProperty13_OTPStoredAsBcryptHash(t *testing.T) {
	mockOTPRepo := newMockOTPRepo()
	service := NewService(ServiceDeps{
		UserRepo:     newMockUserRepo(),
		OTPRepo:      mockOTPRepo,
		TokenStore:   newMockTokenStore(),
		JWTService:   &mockJWTService{},
		EmailService: &mockEmailService{},
	})

	cmd := RegisterCommand{
		FullName:        "Test User",
		Email:           "otp-hash@example.com",
		Password:        "password123",
		ConfirmPassword: "password123",
		Role:            "student",
	}

	_, err := service.Register(context.Background(), cmd)
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	var storedOTP *auth.OTPRecord
	for _, otp := range mockOTPRepo.otps {
		if otp.Purpose == "registration" {
			storedOTP = otp
			break
		}
	}

	if storedOTP == nil {
		t.Fatal("OTP not stored")
	}

	if !strings.HasPrefix(storedOTP.OTPHash, "$2") {
		t.Fatalf("OTP hash does not appear to be bcrypt: %s", storedOTP.OTPHash)
	}

	cost, err := bcrypt.Cost([]byte(storedOTP.OTPHash))
	if err != nil {
		t.Fatalf("failed to get bcrypt cost: %v", err)
	}
	if cost < 12 {
		t.Fatalf("bcrypt cost %d is less than 12", cost)
	}
}

// **Property 14: OTP invalidated after 5 incorrect attempts**
// Validates: Requirements 2.6
func TestProperty14_OTPInvalidatedAfter5Attempts(t *testing.T) {
	mockOTPRepo := newMockOTPRepo()
	mockUserRepo := newMockUserRepo()
	service := NewService(ServiceDeps{
		UserRepo:     mockUserRepo,
		OTPRepo:      mockOTPRepo,
		TokenStore:   newMockTokenStore(),
		JWTService:   &mockJWTService{},
		EmailService: &mockEmailService{},
	})

	// Register a user first
	email := "test@example.com"
	cmd := RegisterCommand{
		FullName:        "Test User",
		Email:           email,
		Password:        "password123",
		ConfirmPassword: "password123",
		Role:            "student",
	}

	_, err := service.Register(context.Background(), cmd)
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	// Try 5 incorrect OTPs
	for i := 0; i < 5; i++ {
		verifyCmd := VerifyOTPCommand{
			Email:   email,
			OTP:     "000000", // Wrong OTP
			Purpose: "registration",
		}
		_, _ = service.VerifyOTP(context.Background(), verifyCmd)
	}

	// 6th attempt should fail with OTP_MAX_ATTEMPTS
	verifyCmd := VerifyOTPCommand{
		Email:   email,
		OTP:     "000000",
		Purpose: "registration",
	}
	_, err = service.VerifyOTP(context.Background(), verifyCmd)
	if err == nil {
		t.Fatal("expected error after 5 failed attempts")
	}
	if err != apperrors.ErrOTPMaxAttempts {
		t.Fatalf("expected OTP_MAX_ATTEMPTS error, got: %v", err)
	}
}

// **Property 15: OTP resend enforces max 3 resends per registration**
// Validates: Requirements 2.8
func TestProperty15_OTPResendEnforcesMax3Resends(t *testing.T) {
	mockOTPRepo := newMockOTPRepo()
	mockUserRepo := newMockUserRepo()
	service := NewService(ServiceDeps{
		UserRepo:     mockUserRepo,
		OTPRepo:      mockOTPRepo,
		TokenStore:   newMockTokenStore(),
		JWTService:   &mockJWTService{},
		EmailService: &mockEmailService{},
	})

	// Register a user first
	email := "test@example.com"
	cmd := RegisterCommand{
		FullName:        "Test User",
		Email:           email,
		Password:        "password123",
		ConfirmPassword: "password123",
		Role:            "student",
	}

	_, err := service.Register(context.Background(), cmd)
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	// Manually set the OTP created time to bypass cooldown
	for _, otp := range mockOTPRepo.otps {
		otp.CreatedAt = time.Now().UTC().Add(-2 * time.Minute)
	}

	// Resend 3 times
	for i := 0; i < 3; i++ {
		resendCmd := ResendOTPCommand{
			Email:   email,
			Purpose: "registration",
		}
		err := service.ResendOTP(context.Background(), resendCmd)
		if err != nil {
			t.Fatalf("resend %d failed: %v", i+1, err)
		}

		// Update created time again
		for _, otp := range mockOTPRepo.otps {
			otp.CreatedAt = time.Now().UTC().Add(-2 * time.Minute)
		}
	}

	// 4th resend should fail
	resendCmd := ResendOTPCommand{
		Email:   email,
		Purpose: "registration",
	}
	err = service.ResendOTP(context.Background(), resendCmd)
	if err == nil {
		t.Fatal("expected error after 3 resends")
	}
	if err != apperrors.ErrOTPResendLimit {
		t.Fatalf("expected OTP_RESEND_LIMIT error, got: %v", err)
	}
}

// **Property 16: Forgot-password always returns 200 regardless of email existence**
// Validates: Requirements 5.1
func TestProperty16_ForgotPasswordAlwaysReturns200(t *testing.T) {
	service := createTestService()

	rapid.Check(t, func(t *rapid.T) {
		email := rapid.StringMatching("[a-z]+@[a-z]+\\.[a-z]+").Draw(t, "email")

		cmd := ForgotPasswordCommand{
			Email: email,
		}

		// Should not return error regardless of whether email exists
		err := service.ForgotPassword(context.Background(), cmd)
		if err != nil {
			t.Fatalf("ForgotPassword returned error: %v", err)
		}
	})
}

// **Property 65: All stored password hashes use bcrypt with cost factor >= 12**
// Validates: Requirements 28.4
func TestProperty65_PasswordHashesBcryptCost12(t *testing.T) {
	mockUserRepo := newMockUserRepo()
	service := NewService(ServiceDeps{
		UserRepo:     mockUserRepo,
		OTPRepo:      newMockOTPRepo(),
		TokenStore:   newMockTokenStore(),
		JWTService:   &mockJWTService{},
		EmailService: &mockEmailService{},
	})

	password := "password123"
	cmd := RegisterCommand{
		FullName:        "Test User",
		Email:           "password-hash@example.com",
		Password:        password,
		ConfirmPassword: password,
		Role:            "student",
	}

	_, err := service.Register(context.Background(), cmd)
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	user, err := mockUserRepo.FindByEmail(context.Background(), cmd.Email)
	if err != nil {
		t.Fatal("user not found after registration")
	}

	if user.PasswordHash == nil {
		t.Fatal("password hash is nil")
	}

	if !strings.HasPrefix(*user.PasswordHash, "$2") {
		t.Fatalf("password hash does not appear to be bcrypt: %s", *user.PasswordHash)
	}

	cost, err := bcrypt.Cost([]byte(*user.PasswordHash))
	if err != nil {
		t.Fatalf("failed to get bcrypt cost: %v", err)
	}
	if cost < 12 {
		t.Fatalf("bcrypt cost %d is less than 12", cost)
	}
}

// **Property 19: Refresh token rotation — old token invalidated after use**
// Validates: Requirements 3.6
func TestProperty19_RefreshTokenRotation(t *testing.T) {
	mockTokenStore := newMockTokenStore()
	mockUserRepo := newMockUserRepo()
	service := NewService(ServiceDeps{
		UserRepo:     mockUserRepo,
		OTPRepo:      newMockOTPRepo(),
		TokenStore:   mockTokenStore,
		JWTService:   &mockJWTService{},
		EmailService: &mockEmailService{},
	})

	// Create a user and store a refresh token
	userID := uuid.New()
	user := &auth.User{
		ID:       userID,
		FullName: "Test User",
		Email:    "test@example.com",
		Role:     "student",
		Status:   "active",
	}
	mockUserRepo.users[user.Email] = user

	oldToken := "old-refresh-token"
	mockTokenStore.StoreRefreshToken(context.Background(), userID, oldToken, 24*time.Hour)

	// Use the refresh token
	_, err := service.RefreshToken(context.Background(), oldToken)
	if err != nil {
		t.Fatalf("refresh token failed: %v", err)
	}

	// Try to use the old token again - should fail
	_, err = service.RefreshToken(context.Background(), oldToken)
	if err == nil {
		t.Fatal("expected error when reusing old refresh token")
	}
}

// **Property 20: Logout invalidates the submitted refresh token**
// Validates: Requirements 3.8
func TestProperty20_LogoutInvalidatesToken(t *testing.T) {
	mockTokenStore := newMockTokenStore()
	service := NewService(ServiceDeps{
		UserRepo:     newMockUserRepo(),
		OTPRepo:      newMockOTPRepo(),
		TokenStore:   mockTokenStore,
		JWTService:   &mockJWTService{},
		EmailService: &mockEmailService{},
	})

	rapid.Check(t, func(t *rapid.T) {
		userID := uuid.New()
		token := rapid.String().Draw(t, "token")

		// Store a refresh token
		mockTokenStore.StoreRefreshToken(context.Background(), userID, token, 24*time.Hour)

		// Logout
		err := service.Logout(context.Background(), token)
		if err != nil {
			t.Fatalf("logout failed: %v", err)
		}

		// Try to validate the token - should fail
		_, err = mockTokenStore.ValidateRefreshToken(context.Background(), token)
		if err == nil {
			t.Fatal("token should be invalidated after logout")
		}
	})
}
