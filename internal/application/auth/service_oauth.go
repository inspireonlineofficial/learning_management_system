package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"lms-backend/internal/domain/auth"
	"lms-backend/pkg/apperrors"
	"lms-backend/pkg/logger"
	"strings"
	"time"

	"github.com/google/uuid"
)

const googleOAuthAdminEmail = "inspireonlineofficial@gmail.com"

// OAuthRedirect implements OAuth redirect with state parameter
func (s *authService) OAuthRedirect(ctx context.Context, cmd OAuthRedirectCommand) (string, error) {
	// Get provider
	provider, err := s.deps.OAuthFactory.GetProvider(cmd.Provider)
	if err != nil {
		return "", &apperrors.AppError{
			Code:       "PROVIDER_NOT_CONFIGURED",
			Message:    fmt.Sprintf("OAuth provider %s is not configured", cmd.Provider),
			HTTPStatus: 400,
		}
	}

	// Generate random state
	stateBytes := make([]byte, 32)
	rand.Read(stateBytes)
	state := hex.EncodeToString(stateBytes)

	// Store state in Redis with 10-minute TTL
	key := fmt.Sprintf("oauth_state:%s", state)
	statePayload, err := json.Marshal(map[string]string{
		"provider":  cmd.Provider,
		"return_to": cmd.ReturnTo,
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal OAuth state: %w", err)
	}
	if err := s.deps.RedisClient.Set(ctx, key, string(statePayload), 10*time.Minute); err != nil {
		return "", fmt.Errorf("failed to store OAuth state: %w", err)
	}

	// Get authorization URL
	authURL := provider.GetAuthURL(state)

	return authURL, nil
}

// OAuthCallback implements OAuth callback with user creation/linking
func (s *authService) OAuthCallback(ctx context.Context, cmd OAuthCallbackCommand) (*OAuthResult, error) {
	// Verify state parameter
	key := fmt.Sprintf("oauth_state:%s", cmd.State)
	storedState, err := s.deps.RedisClient.GetDel(ctx, key)
	if err != nil {
		return nil, &apperrors.AppError{
			Code:       "INVALID_OAUTH_STATE",
			Message:    "Invalid or expired OAuth state parameter",
			HTTPStatus: 400,
		}
	}

	statePayload := struct {
		Provider string `json:"provider"`
		ReturnTo string `json:"return_to"`
	}{}
	if jsonErr := json.Unmarshal([]byte(storedState), &statePayload); jsonErr != nil {
		// Backward compatibility for older provider-only state values.
		statePayload.Provider = storedState
	}

	if statePayload.Provider != cmd.Provider {
		return nil, &apperrors.AppError{
			Code:       "INVALID_OAUTH_STATE",
			Message:    "Invalid or expired OAuth state parameter",
			HTTPStatus: 400,
		}
	}

	// Get provider
	provider, err := s.deps.OAuthFactory.GetProvider(cmd.Provider)
	if err != nil {
		return nil, fmt.Errorf("provider not configured: %w", err)
	}

	// Exchange code for tokens
	accessToken, refreshToken, err := provider.ExchangeCode(ctx, cmd.Code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	// Get user info
	userInfo, err := provider.GetUserInfo(ctx, accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	// Check if OAuth provider already exists
	existingProvider, err := s.deps.OAuthProviderRepo.FindByProviderAndProviderUserID(ctx, cmd.Provider, userInfo.ProviderUserID)
	if err == nil && existingProvider != nil {
		// User already linked - just login
		user, err := s.deps.UserRepo.FindByID(ctx, existingProvider.UserID)
		if err != nil {
			return nil, fmt.Errorf("user not found: %w", err)
		}

		// Check account status
		if user.Status != "active" {
			return nil, apperrors.ErrAccountInactive
		}
		if err := s.ensureGoogleOAuthAdminRole(ctx, cmd.Provider, user); err != nil {
			return nil, err
		}

		// Issue tokens
		tokens, err := s.issueTokens(ctx, user, false)
		if err != nil {
			return nil, err
		}

		return &OAuthResult{
			TokenPair: *tokens,
			IsNewUser: false,
			ReturnTo:  statePayload.ReturnTo,
		}, nil
	}

	// Check if user with this email already exists
	existingUser, err := s.deps.UserRepo.FindByEmail(ctx, userInfo.Email)
	if err == nil && existingUser != nil {
		if existingUser.Status != "active" {
			return nil, apperrors.ErrAccountInactive
		}
		if err := s.ensureGoogleOAuthAdminRole(ctx, cmd.Provider, existingUser); err != nil {
			return nil, err
		}

		// Link provider to existing account
		encryptedAccessToken, err := s.deps.TokenEncryptor.Encrypt(accessToken)
		if err != nil {
			logger.Error(ctx, "failed to encrypt access token", "error", err)
			encryptedAccessToken = ""
		}

		encryptedRefreshToken, err := s.deps.TokenEncryptor.Encrypt(refreshToken)
		if err != nil {
			logger.Error(ctx, "failed to encrypt refresh token", "error", err)
			encryptedRefreshToken = ""
		}

		oauthProvider := &auth.OAuthProvider{
			UserID:                existingUser.ID,
			Provider:              cmd.Provider,
			ProviderUserID:        userInfo.ProviderUserID,
			AccessTokenEncrypted:  stringPtrIfNotEmpty(encryptedAccessToken),
			RefreshTokenEncrypted: stringPtrIfNotEmpty(encryptedRefreshToken),
		}

		if err := s.deps.OAuthProviderRepo.Create(ctx, oauthProvider); err != nil {
			return nil, fmt.Errorf("failed to link provider: %w", err)
		}

		// Issue tokens
		tokens, err := s.issueTokens(ctx, existingUser, false)
		if err != nil {
			return nil, err
		}

		return &OAuthResult{
			TokenPair: *tokens,
			IsNewUser: false,
			ReturnTo:  statePayload.ReturnTo,
		}, nil
	}

	// Create new user
	user := &auth.User{
		FullName:        userInfo.Name,
		Email:           strings.ToLower(strings.TrimSpace(userInfo.Email)),
		PasswordHash:    nil, // OAuth-only account
		Role:            roleForGoogleOAuthEmail(cmd.Provider, userInfo.Email),
		Status:          "active", // No OTP verification for OAuth
		ProfileComplete: false,
	}

	if err := s.deps.UserRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Link OAuth provider
	encryptedAccessToken, err := s.deps.TokenEncryptor.Encrypt(accessToken)
	if err != nil {
		logger.Error(ctx, "failed to encrypt access token", "error", err)
		encryptedAccessToken = ""
	}

	encryptedRefreshToken, err := s.deps.TokenEncryptor.Encrypt(refreshToken)
	if err != nil {
		logger.Error(ctx, "failed to encrypt refresh token", "error", err)
		encryptedRefreshToken = ""
	}

	oauthProvider := &auth.OAuthProvider{
		UserID:                user.ID,
		Provider:              cmd.Provider,
		ProviderUserID:        userInfo.ProviderUserID,
		AccessTokenEncrypted:  stringPtrIfNotEmpty(encryptedAccessToken),
		RefreshTokenEncrypted: stringPtrIfNotEmpty(encryptedRefreshToken),
	}

	if err := s.deps.OAuthProviderRepo.Create(ctx, oauthProvider); err != nil {
		return nil, fmt.Errorf("failed to create OAuth provider: %w", err)
	}

	// Issue tokens
	tokens, err := s.issueTokens(ctx, user, false)
	if err != nil {
		return nil, err
	}

	return &OAuthResult{
		TokenPair: *tokens,
		IsNewUser: true,
		ReturnTo:  statePayload.ReturnTo,
	}, nil
}

func (s *authService) ensureGoogleOAuthAdminRole(ctx context.Context, provider string, user *auth.User) error {
	role := roleForGoogleOAuthEmail(provider, user.Email)
	if role != "admin" || user.Role == "admin" {
		return nil
	}
	user.Role = "admin"
	user.ProfileComplete = true
	if err := s.deps.UserRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to promote OAuth admin user: %w", err)
	}
	return nil
}

func roleForGoogleOAuthEmail(provider, email string) string {
	if strings.EqualFold(provider, "google") &&
		strings.EqualFold(strings.TrimSpace(email), googleOAuthAdminEmail) {
		return "admin"
	}
	return "student"
}

// ConnectProvider implements connecting a new OAuth provider to an existing account
func (s *authService) ConnectProvider(ctx context.Context, userID uuid.UUID, cmd ConnectProviderCommand) error {
	// Verify state parameter
	key := fmt.Sprintf("oauth_state:%s", cmd.State)
	storedProvider, err := s.deps.RedisClient.GetDel(ctx, key)
	if err != nil || storedProvider != cmd.Provider {
		return &apperrors.AppError{
			Code:       "INVALID_OAUTH_STATE",
			Message:    "Invalid or expired OAuth state parameter",
			HTTPStatus: 400,
		}
	}

	// Get provider
	provider, err := s.deps.OAuthFactory.GetProvider(cmd.Provider)
	if err != nil {
		return fmt.Errorf("provider not configured: %w", err)
	}

	// Exchange code for tokens
	accessToken, refreshToken, err := provider.ExchangeCode(ctx, cmd.Code)
	if err != nil {
		return fmt.Errorf("failed to exchange code: %w", err)
	}

	// Get user info
	userInfo, err := provider.GetUserInfo(ctx, accessToken)
	if err != nil {
		return fmt.Errorf("failed to get user info: %w", err)
	}

	// Check if provider is already linked to this user
	existingProvider, err := s.deps.OAuthProviderRepo.FindByUserIDAndProvider(ctx, userID, cmd.Provider)
	if err == nil && existingProvider != nil {
		return &apperrors.AppError{
			Code:       "PROVIDER_ALREADY_LINKED",
			Message:    "This provider is already linked to your account",
			HTTPStatus: 409,
		}
	}

	// Encrypt tokens
	encryptedAccessToken, err := s.deps.TokenEncryptor.Encrypt(accessToken)
	if err != nil {
		logger.Error(ctx, "failed to encrypt access token", "error", err)
		encryptedAccessToken = ""
	}

	encryptedRefreshToken, err := s.deps.TokenEncryptor.Encrypt(refreshToken)
	if err != nil {
		logger.Error(ctx, "failed to encrypt refresh token", "error", err)
		encryptedRefreshToken = ""
	}

	// Create OAuth provider link
	oauthProvider := &auth.OAuthProvider{
		UserID:                userID,
		Provider:              cmd.Provider,
		ProviderUserID:        userInfo.ProviderUserID,
		AccessTokenEncrypted:  stringPtrIfNotEmpty(encryptedAccessToken),
		RefreshTokenEncrypted: stringPtrIfNotEmpty(encryptedRefreshToken),
	}

	if err := s.deps.OAuthProviderRepo.Create(ctx, oauthProvider); err != nil {
		return fmt.Errorf("failed to link provider: %w", err)
	}

	return nil
}

// DisconnectProvider implements disconnecting an OAuth provider
func (s *authService) DisconnectProvider(ctx context.Context, userID uuid.UUID, cmd DisconnectProviderCommand) error {
	// Find the provider
	provider, err := s.deps.OAuthProviderRepo.FindByUserIDAndProvider(ctx, userID, cmd.Provider)
	if err != nil {
		return &apperrors.AppError{
			Code:       "PROVIDER_NOT_FOUND",
			Message:    "This provider is not linked to your account",
			HTTPStatus: 404,
		}
	}

	// Check if this is the only authentication method
	user, err := s.deps.UserRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// If user has no password, check if they have other OAuth providers
	if user.PasswordHash == nil {
		providers, err := s.deps.OAuthProviderRepo.ListByUserID(ctx, userID)
		if err != nil {
			return fmt.Errorf("failed to list providers: %w", err)
		}

		if len(providers) <= 1 {
			return &apperrors.AppError{
				Code:       "CANNOT_REMOVE_ONLY_AUTH_METHOD",
				Message:    "Cannot remove the only authentication method. Please set a password first.",
				HTTPStatus: 422,
			}
		}
	}

	// Delete the provider
	if err := s.deps.OAuthProviderRepo.Delete(ctx, provider.ID); err != nil {
		return fmt.Errorf("failed to disconnect provider: %w", err)
	}

	return nil
}

// ListProviders implements listing linked OAuth providers
func (s *authService) ListProviders(ctx context.Context, userID uuid.UUID) ([]*OAuthProviderResult, error) {
	providers, err := s.deps.OAuthProviderRepo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list providers: %w", err)
	}

	results := make([]*OAuthProviderResult, len(providers))
	for i, p := range providers {
		results[i] = &OAuthProviderResult{
			Provider:  p.Provider,
			CreatedAt: p.CreatedAt,
		}
	}

	return results, nil
}

// Helper functions

func stringPtrIfNotEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
