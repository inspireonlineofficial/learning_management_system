package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"lms-backend/pkg/apperrors"
	"lms-backend/pkg/logger"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func (s *authService) adminOTPCode() string {
	if s.deps.AdminDevBypass && s.deps.AdminDevOTP != "" {
		return s.deps.AdminDevOTP
	}
	return generateOTP()
}

// AdminLogin implements admin login with 2FA
func (s *authService) AdminLogin(ctx context.Context, cmd AdminLoginCommand) (*OTPSessionResult, error) {
	ipAddress := getIPFromContext(ctx)

	// Check brute force protection
	if ipAddress != "" {
		key := fmt.Sprintf("rate:login:%s", ipAddress)
		count, err := s.deps.RedisClient.Incr(ctx, key)
		if err == nil {
			if count == 1 {
				_ = s.deps.RedisClient.Expire(ctx, key, 15*time.Minute)
			}
			if count > 5 {
				// Log failed attempt
				if s.deps.AuditLogger != nil {
					_ = s.deps.AuditLogger.LogAdminLogin(ctx, uuid.Nil, cmd.Username, ipAddress, false)
				}
				return nil, apperrors.ErrTooManyAttempts
			}
		}
	}

	// Find user by username
	user, err := s.deps.UserRepo.FindByUsername(ctx, cmd.Username)
	if err != nil {
		// Log failed attempt
		if s.deps.AuditLogger != nil {
			_ = s.deps.AuditLogger.LogAdminLogin(ctx, uuid.Nil, cmd.Username, ipAddress, false)
		}
		return nil, apperrors.ErrInvalidCredentials
	}

	// Verify role is admin
	if user.Role != "admin" {
		// Log failed attempt
		if s.deps.AuditLogger != nil {
			_ = s.deps.AuditLogger.LogAdminLogin(ctx, user.ID, cmd.Username, ipAddress, false)
		}
		return nil, apperrors.ErrInvalidCredentials
	}

	// Verify password
	if user.PasswordHash == nil {
		// Log failed attempt
		if s.deps.AuditLogger != nil {
			_ = s.deps.AuditLogger.LogAdminLogin(ctx, user.ID, cmd.Username, ipAddress, false)
		}
		return nil, apperrors.ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(cmd.Password)); err != nil {
		// Log failed attempt
		if s.deps.AuditLogger != nil {
			_ = s.deps.AuditLogger.LogAdminLogin(ctx, user.ID, cmd.Username, ipAddress, false)
		}
		return nil, apperrors.ErrInvalidCredentials
	}

	// Check account status
	if user.Status != "active" {
		// Log failed attempt
		if s.deps.AuditLogger != nil {
			_ = s.deps.AuditLogger.LogAdminLogin(ctx, user.ID, cmd.Username, ipAddress, false)
		}
		return nil, apperrors.ErrAccountInactive
	}

	// Generate OTP
	otp := s.adminOTPCode()
	otpHash, err := bcrypt.GenerateFromPassword([]byte(otp), 12)
	if err != nil {
		return nil, fmt.Errorf("failed to hash OTP: %w", err)
	}

	// Generate OTP session token
	sessionToken := generateRefreshToken()
	expiresAt := time.Now().UTC().Add(5 * time.Minute)

	// Store OTP session in Redis
	sessionData := map[string]interface{}{
		"user_id":  user.ID.String(),
		"otp_hash": string(otpHash),
		"attempts": 0,
		"resends":  0,
		"email":    user.Email,
		"username": cmd.Username,
	}

	sessionJSON, err := json.Marshal(sessionData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal session data: %w", err)
	}

	key := fmt.Sprintf("otp_session:%s", sessionToken)
	if err := s.deps.RedisClient.Set(ctx, key, string(sessionJSON), 5*time.Minute); err != nil {
		return nil, fmt.Errorf("failed to store OTP session: %w", err)
	}

	// Send OTP email
	if !s.deps.AdminDevBypass {
		if err := s.deps.EmailService.SendOTP(ctx, user.Email, otp, expiresAt); err != nil {
			logger.Error(ctx, "failed to send admin OTP email", "error", err)
		}
	} else {
		logger.Info(ctx, "admin dev bypass enabled; skipping OTP email", "username", cmd.Username)
	}

	return &OTPSessionResult{
		OTPSessionToken: sessionToken,
		MaskedEmail:     maskEmail(user.Email),
		ExpiresAt:       expiresAt,
	}, nil
}

// AdminVerifyOTP implements admin OTP verification
func (s *authService) AdminVerifyOTP(ctx context.Context, cmd AdminVerifyOTPCommand) (*TokenPair, error) {
	ipAddress := getIPFromContext(ctx)

	// Get OTP session from Redis
	key := fmt.Sprintf("otp_session:%s", cmd.OTPSessionToken)
	sessionJSON, err := s.deps.RedisClient.Get(ctx, key)
	if err != nil {
		return nil, &apperrors.AppError{
			Code:       "INVALID_OTP_SESSION",
			Message:    "Invalid or expired OTP session",
			HTTPStatus: 400,
		}
	}

	var sessionData map[string]interface{}
	if err := json.Unmarshal([]byte(sessionJSON), &sessionData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session data: %w", err)
	}

	// Extract session data
	userIDStr, _ := sessionData["user_id"].(string)
	otpHash, _ := sessionData["otp_hash"].(string)
	attempts, _ := sessionData["attempts"].(float64)
	username, _ := sessionData["username"].(string)

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID in session: %w", err)
	}

	// Verify OTP
	if err := bcrypt.CompareHashAndPassword([]byte(otpHash), []byte(cmd.OTP)); err != nil {
		// Increment attempts
		attempts++
		sessionData["attempts"] = attempts

		if attempts >= 5 {
			// Invalidate session
			_ = s.deps.RedisClient.Del(ctx, key)

			// Log failed attempt
			if s.deps.AuditLogger != nil {
				_ = s.deps.AuditLogger.LogAdminLogin(ctx, userID, username, ipAddress, false)
			}

			return nil, apperrors.ErrOTPMaxAttempts
		}

		// Update session with new attempt count
		updatedJSON, _ := json.Marshal(sessionData)
		_ = s.deps.RedisClient.Set(ctx, key, string(updatedJSON), 5*time.Minute)

		return nil, &apperrors.AppError{
			Code:       "INVALID_OTP",
			Message:    "Invalid OTP",
			HTTPStatus: 400,
		}
	}

	// OTP is correct - delete session
	_ = s.deps.RedisClient.Del(ctx, key)

	// Find user
	user, err := s.deps.UserRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Log successful login
	if s.deps.AuditLogger != nil {
		_ = s.deps.AuditLogger.LogAdminLogin(ctx, user.ID, username, ipAddress, true)
	}

	// Issue tokens
	return s.issueTokens(ctx, user, false, true)
}

// AdminResendOTP implements admin OTP resend
func (s *authService) AdminResendOTP(ctx context.Context, otpSessionToken string) error {
	// Get OTP session from Redis
	key := fmt.Sprintf("otp_session:%s", otpSessionToken)
	sessionJSON, err := s.deps.RedisClient.Get(ctx, key)
	if err != nil {
		return &apperrors.AppError{
			Code:       "INVALID_OTP_SESSION",
			Message:    "Invalid or expired OTP session",
			HTTPStatus: 400,
		}
	}

	var sessionData map[string]interface{}
	if err := json.Unmarshal([]byte(sessionJSON), &sessionData); err != nil {
		return fmt.Errorf("failed to unmarshal session data: %w", err)
	}

	// Check resend limit
	resends, _ := sessionData["resends"].(float64)
	if resends >= 3 {
		return apperrors.ErrOTPResendLimit
	}

	// Extract email
	email, _ := sessionData["email"].(string)

	// Generate new OTP
	otp := s.adminOTPCode()
	otpHash, err := bcrypt.GenerateFromPassword([]byte(otp), 12)
	if err != nil {
		return fmt.Errorf("failed to hash OTP: %w", err)
	}

	// Update session
	sessionData["otp_hash"] = string(otpHash)
	sessionData["attempts"] = 0
	sessionData["resends"] = resends + 1

	updatedJSON, err := json.Marshal(sessionData)
	if err != nil {
		return fmt.Errorf("failed to marshal session data: %w", err)
	}

	if err := s.deps.RedisClient.Set(ctx, key, string(updatedJSON), 5*time.Minute); err != nil {
		return fmt.Errorf("failed to update OTP session: %w", err)
	}

	// Send OTP email
	expiresAt := time.Now().UTC().Add(5 * time.Minute)
	if !s.deps.AdminDevBypass {
		if err := s.deps.EmailService.SendOTP(ctx, email, otp, expiresAt); err != nil {
			logger.Error(ctx, "failed to send admin OTP email", "error", err)
		}
	} else {
		logger.Info(ctx, "admin dev bypass enabled; skipping OTP email resend", "email", email)
	}

	return nil
}
