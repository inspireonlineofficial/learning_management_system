package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"lms-backend/internal/domain/auth"
	"lms-backend/pkg/apperrors"
	"lms-backend/pkg/logger"
	"lms-backend/pkg/validator"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// ForgotPassword implements forgot password (always returns 200)
func (s *authService) ForgotPassword(ctx context.Context, cmd ForgotPasswordCommand) error {
	// Always return success to prevent email enumeration
	// But only send email if user exists and has password auth

	user, err := s.deps.UserRepo.FindByEmail(ctx, cmd.Email)
	if err != nil {
		// User doesn't exist, but return success anyway
		return nil
	}

	// Check if user has password authentication
	if user.PasswordHash == nil {
		// OAuth-only account, but don't reveal this
		return nil
	}

	// Generate reset token
	tokenBytes := make([]byte, 32)
	rand.Read(tokenBytes)
	token := hex.EncodeToString(tokenBytes)

	// Hash the token for storage
	tokenHash := hashToken(token)

	// Store token
	expiresAt := time.Now().UTC().Add(30 * time.Minute)
	resetToken := &auth.PasswordResetToken{
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
	}

	if err := s.deps.PasswordResetRepo.Store(ctx, resetToken); err != nil {
		logger.Error(ctx, "failed to store password reset token", "error", err)
		return nil // Don't reveal error
	}

	// Send reset email
	baseURL := s.deps.FrontendBaseURL
	if baseURL == "" {
		baseURL = "http://localhost:5173"
	}
	resetLink := fmt.Sprintf("%s/reset-password?token=%s", strings.TrimRight(baseURL, "/"), token)
	if err := s.deps.EmailService.SendPasswordReset(ctx, user.Email, resetLink); err != nil {
		logger.Error(ctx, "failed to send password reset email", "error", err)
	}

	return nil
}

// ResetPassword implements password reset
func (s *authService) ResetPassword(ctx context.Context, cmd ResetPasswordCommand) error {
	// Validate passwords
	if err := s.validatePasswordReset(cmd); err != nil {
		return err
	}

	// Hash the token
	tokenHash := hashToken(cmd.Token)

	// Find token
	resetToken, err := s.deps.PasswordResetRepo.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		return &apperrors.AppError{
			Code:       "INVALID_RESET_TOKEN",
			Message:    "Invalid or expired reset token",
			HTTPStatus: 400,
		}
	}

	// Check if token is expired
	if time.Now().UTC().After(resetToken.ExpiresAt) {
		return &apperrors.AppError{
			Code:       "INVALID_RESET_TOKEN",
			Message:    "Invalid or expired reset token",
			HTTPStatus: 400,
		}
	}

	// Check if token is already used
	if resetToken.UsedAt != nil {
		return &apperrors.AppError{
			Code:       "TOKEN_ALREADY_USED",
			Message:    "This reset token has already been used",
			HTTPStatus: 400,
		}
	}

	// Find user
	user, err := s.deps.UserRepo.FindByID(ctx, resetToken.UserID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Hash new password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(cmd.NewPassword), 12)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password
	user.PasswordHash = stringPtr(string(passwordHash))
	if err := s.deps.UserRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Mark token as used
	if err := s.deps.PasswordResetRepo.MarkAsUsed(ctx, resetToken.ID); err != nil {
		logger.Error(ctx, "failed to mark token as used", "error", err)
	}

	// Invalidate all refresh tokens
	if err := s.deps.TokenStore.DeleteAllRefreshTokens(ctx, user.ID); err != nil {
		logger.Error(ctx, "failed to invalidate refresh tokens", "error", err)
	}

	return nil
}

// ChangePassword implements password change
func (s *authService) ChangePassword(ctx context.Context, userID uuid.UUID, cmd ChangePasswordCommand) error {
	// Validate passwords
	if err := s.validatePasswordChange(cmd); err != nil {
		return err
	}

	// Find user
	user, err := s.deps.UserRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Check if user has password authentication
	if user.PasswordHash == nil {
		return &apperrors.AppError{
			Code:       "NO_PASSWORD_SET",
			Message:    "This account does not have a password set. Please use OAuth to login.",
			HTTPStatus: 400,
		}
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(cmd.CurrentPassword)); err != nil {
		return &apperrors.AppError{
			Code:       "INVALID_CURRENT_PASSWORD",
			Message:    "Current password is incorrect",
			HTTPStatus: 400,
		}
	}

	// Hash new password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(cmd.NewPassword), 12)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password
	user.PasswordHash = stringPtr(string(passwordHash))
	if err := s.deps.UserRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Invalidate all other refresh tokens (keep current session)
	if err := s.deps.TokenStore.DeleteAllRefreshTokens(ctx, user.ID); err != nil {
		logger.Error(ctx, "failed to invalidate refresh tokens", "error", err)
	}

	return nil
}

// UpdateProfile implements profile update
func (s *authService) UpdateProfile(ctx context.Context, userID uuid.UUID, cmd UpdateProfileCommand) (*ProfileResult, error) {
	// Find user
	user, err := s.deps.UserRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Update fields
	if cmd.FullName != nil {
		if len(*cmd.FullName) < 2 || len(*cmd.FullName) > 100 {
			return nil, apperrors.NewValidationError([]map[string]string{
				{"field": "full_name", "message": "must be between 2 and 100 characters"},
			})
		}
		user.FullName = *cmd.FullName
	}

	// Update user
	if err := s.deps.UserRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to update profile: %w", err)
	}

	return &ProfileResult{
		ID:              user.ID.String(),
		FullName:        user.FullName,
		Email:           user.Email,
		Role:            user.Role,
		ProfileComplete: user.ProfileComplete,
		CreatedAt:       user.CreatedAt,
		UpdatedAt:       user.UpdatedAt,
	}, nil
}

// GetProfile implements profile retrieval
func (s *authService) GetProfile(ctx context.Context, userID uuid.UUID) (*ProfileResult, error) {
	user, err := s.deps.UserRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	return &ProfileResult{
		ID:              user.ID.String(),
		FullName:        user.FullName,
		Email:           user.Email,
		Role:            user.Role,
		ProfileComplete: user.ProfileComplete,
		CreatedAt:       user.CreatedAt,
		UpdatedAt:       user.UpdatedAt,
	}, nil
}

// Validation helpers

func (s *authService) validatePasswordReset(cmd ResetPasswordCommand) error {
	var details []map[string]string

	if len(cmd.NewPassword) < 8 {
		details = append(details, map[string]string{
			"field":   "new_password",
			"message": "must be at least 8 characters",
		})
	}

	if !validator.ContainsDigit(cmd.NewPassword) {
		details = append(details, map[string]string{
			"field":   "new_password",
			"message": "must contain at least one digit",
		})
	}

	if cmd.NewPassword != cmd.ConfirmPassword {
		details = append(details, map[string]string{
			"field":   "confirm_password",
			"message": "passwords do not match",
		})
	}

	if len(details) > 0 {
		return apperrors.NewValidationError(details)
	}

	return nil
}

func (s *authService) validatePasswordChange(cmd ChangePasswordCommand) error {
	var details []map[string]string

	if len(cmd.NewPassword) < 8 {
		details = append(details, map[string]string{
			"field":   "new_password",
			"message": "must be at least 8 characters",
		})
	}

	if !validator.ContainsDigit(cmd.NewPassword) {
		details = append(details, map[string]string{
			"field":   "new_password",
			"message": "must contain at least one digit",
		})
	}

	if cmd.NewPassword != cmd.ConfirmPassword {
		details = append(details, map[string]string{
			"field":   "confirm_password",
			"message": "passwords do not match",
		})
	}

	if len(details) > 0 {
		return apperrors.NewValidationError(details)
	}

	return nil
}

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
