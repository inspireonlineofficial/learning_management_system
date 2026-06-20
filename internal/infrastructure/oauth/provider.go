package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// UserInfo represents the user information returned by OAuth providers
type UserInfo struct {
	ProviderUserID string
	Email          string
	Name           string
	AvatarURL      string
}

// Provider defines the OAuth 2.0 provider interface
type Provider interface {
	GetAuthURL(state string) string
	ExchangeCode(ctx context.Context, code string) (accessToken, refreshToken string, err error)
	GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error)
}

// GoogleProvider implements OAuth 2.0 for Google
type GoogleProvider struct {
	clientID     string
	clientSecret string
	redirectURL  string
}

// NewGoogleProvider creates a new Google OAuth provider
func NewGoogleProvider(clientID, clientSecret, redirectURL string) *GoogleProvider {
	return &GoogleProvider{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURL:  redirectURL,
	}
}

// GetAuthURL returns the Google OAuth authorization URL
func (p *GoogleProvider) GetAuthURL(state string) string {
	params := url.Values{}
	params.Set("client_id", p.clientID)
	params.Set("redirect_uri", p.redirectURL)
	params.Set("response_type", "code")
	params.Set("scope", "openid email profile")
	params.Set("state", state)
	params.Set("access_type", "offline")
	params.Set("prompt", "consent")

	return "https://accounts.google.com/o/oauth2/v2/auth?" + params.Encode()
}

// ExchangeCode exchanges the authorization code for tokens
func (p *GoogleProvider) ExchangeCode(ctx context.Context, code string) (accessToken, refreshToken string, err error) {
	data := url.Values{}
	data.Set("code", code)
	data.Set("client_id", p.clientID)
	data.Set("client_secret", p.clientSecret)
	data.Set("redirect_uri", p.redirectURL)
	data.Set("grant_type", "authorization_code")

	req, err := http.NewRequestWithContext(ctx, "POST", "https://oauth2.googleapis.com/token", strings.NewReader(data.Encode()))
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("token exchange failed: %s", string(body))
	}

	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", fmt.Errorf("failed to decode response: %w", err)
	}

	return result.AccessToken, result.RefreshToken, nil
}

// GetUserInfo retrieves user information from Google
func (p *GoogleProvider) GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get user info: %s", string(body))
	}

	var result struct {
		ID      string `json:"id"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	return &UserInfo{
		ProviderUserID: result.ID,
		Email:          result.Email,
		Name:           result.Name,
		AvatarURL:      result.Picture,
	}, nil
}

// ProviderFactory creates OAuth providers based on configuration
type ProviderFactory struct {
	providers map[string]Provider
}

// NewProviderFactory creates a new provider factory
func NewProviderFactory(
	googleClientID, googleClientSecret, googleRedirectURL string,
) *ProviderFactory {
	factory := &ProviderFactory{
		providers: make(map[string]Provider),
	}

	// Google is required
	if googleClientID != "" && googleClientSecret != "" {
		factory.providers["google"] = NewGoogleProvider(googleClientID, googleClientSecret, googleRedirectURL)
	}

	return factory
}

// GetProvider returns a provider by name
func (f *ProviderFactory) GetProvider(name string) (Provider, error) {
	provider, ok := f.providers[name]
	if !ok {
		return nil, fmt.Errorf("provider %s not configured", name)
	}
	return provider, nil
}

// IsProviderEnabled checks if a provider is enabled
func (f *ProviderFactory) IsProviderEnabled(name string) bool {
	_, ok := f.providers[name]
	return ok
}

// Factory implements the OAuthProviderFactory interface
type Factory struct {
	providers map[string]Provider
}

// NewFactory creates a new OAuth provider factory
func NewFactory(providers map[string]Provider) *Factory {
	return &Factory{
		providers: providers,
	}
}

// GetProvider returns the OAuth provider for the given name
func (f *Factory) GetProvider(name string) (ProviderAdapter, error) {
	provider, ok := f.providers[name]
	if !ok {
		return ProviderAdapter{}, fmt.Errorf("OAuth provider %s not found", name)
	}
	return ProviderAdapter{provider: provider}, nil
}

// IsProviderEnabled checks if the provider is enabled
func (f *Factory) IsProviderEnabled(name string) bool {
	_, ok := f.providers[name]
	return ok
}

// ProviderAdapter adapts oauth.Provider to auth.OAuthProvider
type ProviderAdapter struct {
	provider Provider
}

// GetAuthURL returns the authorization URL
func (a ProviderAdapter) GetAuthURL(state string) string {
	return a.provider.GetAuthURL(state)
}

// ExchangeCode exchanges the authorization code for tokens
func (a ProviderAdapter) ExchangeCode(ctx context.Context, code string) (accessToken, refreshToken string, err error) {
	return a.provider.ExchangeCode(ctx, code)
}

// OAuthUserInfo represents user info from OAuth provider (matches auth.OAuthUserInfo)
type OAuthUserInfo struct {
	ProviderUserID string
	Email          string
	Name           string
	AvatarURL      string
}

// GetUserInfo gets user information from the provider
func (a ProviderAdapter) GetUserInfo(ctx context.Context, accessToken string) (*OAuthUserInfo, error) {
	info, err := a.provider.GetUserInfo(ctx, accessToken)
	if err != nil {
		return nil, err
	}
	return &OAuthUserInfo{
		ProviderUserID: info.ProviderUserID,
		Email:          info.Email,
		Name:           info.Name,
		AvatarURL:      info.AvatarURL,
	}, nil
}
