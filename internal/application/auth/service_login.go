package auth

import (
	"context"
	"fmt"
	"lms-backend/pkg/apperrors"
	"lms-backend/pkg/logger"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Login implements login with brute force protection
func (s *authService) Login(ctx context.Context, cmd LoginCommand) (*TokenPair, error) {
	// Check brute force protection
	ipAddress := getIPFromContext(ctx)
	if ipAddress != "" {
		key := fmt.Sprintf("rate:login:%s", ipAddress)
		count, err := s.deps.RedisClient.Incr(ctx, key)
		if err == nil {
			if count == 1 {
				_ = s.deps.RedisClient.Expire(ctx, key, 15*time.Minute)
			}
			if count > 5 {
				return nil, apperrors.ErrTooManyAttempts
			}
		}
	}

	// Find user by email
	user, err := s.deps.UserRepo.FindByEmail(ctx, cmd.Email)
	if err != nil {
		return nil, apperrors.ErrInvalidCredentials
	}

	// Verify password
	if user.PasswordHash == nil {
		return nil, apperrors.ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(cmd.Password)); err != nil {
		return nil, apperrors.ErrInvalidCredentials
	}

	// Check account status
	if user.Status != "active" {
		return nil, apperrors.ErrAccountInactive
	}

	// Issue tokens
	return s.issueTokens(ctx, user, cmd.RememberMe, true)
}

// RefreshToken implements token refresh with rotation
func (s *authService) RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error) {
	// Validate and delete old token atomically (GETDEL)
	userID, err := s.deps.TokenStore.ValidateRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, apperrors.ErrInvalidRefreshToken
	}

	// Delete the old token (rotation)
	if err := s.deps.TokenStore.DeleteRefreshToken(ctx, refreshToken); err != nil {
		logger.Error(ctx, "failed to delete old refresh token", "error", err)
	}

	// Find user
	user, err := s.deps.UserRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, apperrors.ErrInvalidRefreshToken
	}

	// Check account status
	if user.Status != "active" {
		return nil, apperrors.ErrAccountInactive
	}

	// Issue new tokens
	return s.issueTokens(ctx, user, true, false) // Assume remember_me for refresh
}

// Logout implements logout by invalidating the refresh token
func (s *authService) Logout(ctx context.Context, refreshToken string) error {
	if err := s.deps.TokenStore.DeleteRefreshToken(ctx, refreshToken); err != nil {
		return fmt.Errorf("failed to logout: %w", err)
	}
	return nil
}

// getIPFromContext extracts IP address from context
func getIPFromContext(ctx context.Context) string {
	if ip := ctx.Value("ip_address"); ip != nil {
		if ipStr, ok := ip.(string); ok {
			return ipStr
		}
	}
	return ""
}
