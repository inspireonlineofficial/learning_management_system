package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

	req, err := http.NewRequestWithContext(ctx, "POST", "https://oauth2.googleapis.com/token", nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}
	req.URL.RawQuery = data.Encode()
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

// GitHubProvider implements OAuth 2.0 for GitHub
type GitHubProvider struct {
	clientID     string
	clientSecret string
	redirectURL  string
}

// NewGitHubProvider creates a new GitHub OAuth provider
func NewGitHubProvider(clientID, clientSecret, redirectURL string) *GitHubProvider {
	return &GitHubProvider{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURL:  redirectURL,
	}
}

// GetAuthURL returns the GitHub OAuth authorization URL
func (p *GitHubProvider) GetAuthURL(state string) string {
	params := url.Values{}
	params.Set("client_id", p.clientID)
	params.Set("redirect_uri", p.redirectURL)
	params.Set("scope", "user:email")
	params.Set("state", state)

	return "https://github.com/login/oauth/authorize?" + params.Encode()
}

// ExchangeCode exchanges the authorization code for tokens
func (p *GitHubProvider) ExchangeCode(ctx context.Context, code string) (accessToken, refreshToken string, err error) {
	data := url.Values{}
	data.Set("code", code)
	data.Set("client_id", p.clientID)
	data.Set("client_secret", p.clientSecret)
	data.Set("redirect_uri", p.redirectURL)

	req, err := http.NewRequestWithContext(ctx, "POST", "https://github.com/login/oauth/access_token", nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}
	req.URL.RawQuery = data.Encode()
	req.Header.Set("Accept", "application/json")

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
		AccessToken string `json:"access_token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", fmt.Errorf("failed to decode response: %w", err)
	}

	return result.AccessToken, "", nil // GitHub doesn't provide refresh tokens in this flow
}

// GetUserInfo retrieves user information from GitHub
func (p *GitHubProvider) GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	// Get user profile
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get user info: %s", string(body))
	}

	var user struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		AvatarURL string `json:"avatar_url"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	// Get primary email
	emailReq, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user/emails", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create email request: %w", err)
	}
	emailReq.Header.Set("Authorization", "Bearer "+accessToken)
	emailReq.Header.Set("Accept", "application/json")

	emailResp, err := http.DefaultClient.Do(emailReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get emails: %w", err)
	}
	defer emailResp.Body.Close()

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}

	if err := json.NewDecoder(emailResp.Body).Decode(&emails); err != nil {
		return nil, fmt.Errorf("failed to decode emails: %w", err)
	}

	var primaryEmail string
	for _, e := range emails {
		if e.Primary && e.Verified {
			primaryEmail = e.Email
			break
		}
	}

	if primaryEmail == "" && len(emails) > 0 {
		primaryEmail = emails[0].Email
	}

	name := user.Name
	if name == "" {
		name = user.Login
	}

	return &UserInfo{
		ProviderUserID: fmt.Sprintf("%d", user.ID),
		Email:          primaryEmail,
		Name:           name,
		AvatarURL:      user.AvatarURL,
	}, nil
}

// MicrosoftProvider implements OAuth 2.0 for Microsoft
type MicrosoftProvider struct {
	clientID     string
	clientSecret string
	redirectURL  string
}

// NewMicrosoftProvider creates a new Microsoft OAuth provider
func NewMicrosoftProvider(clientID, clientSecret, redirectURL string) *MicrosoftProvider {
	return &MicrosoftProvider{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURL:  redirectURL,
	}
}

// GetAuthURL returns the Microsoft OAuth authorization URL
func (p *MicrosoftProvider) GetAuthURL(state string) string {
	params := url.Values{}
	params.Set("client_id", p.clientID)
	params.Set("redirect_uri", p.redirectURL)
	params.Set("response_type", "code")
	params.Set("scope", "openid email profile")
	params.Set("state", state)
	params.Set("response_mode", "query")

	return "https://login.microsoftonline.com/common/oauth2/v2.0/authorize?" + params.Encode()
}

// ExchangeCode exchanges the authorization code for tokens
func (p *MicrosoftProvider) ExchangeCode(ctx context.Context, code string) (accessToken, refreshToken string, err error) {
	data := url.Values{}
	data.Set("code", code)
	data.Set("client_id", p.clientID)
	data.Set("client_secret", p.clientSecret)
	data.Set("redirect_uri", p.redirectURL)
	data.Set("grant_type", "authorization_code")

	req, err := http.NewRequestWithContext(ctx, "POST", "https://login.microsoftonline.com/common/oauth2/v2.0/token", nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}
	req.URL.RawQuery = data.Encode()
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

// GetUserInfo retrieves user information from Microsoft
func (p *MicrosoftProvider) GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://graph.microsoft.com/v1.0/me", nil)
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
		ID                string `json:"id"`
		DisplayName       string `json:"displayName"`
		Mail              string `json:"mail"`
		UserPrincipalName string `json:"userPrincipalName"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	email := result.Mail
	if email == "" {
		email = result.UserPrincipalName
	}

	return &UserInfo{
		ProviderUserID: result.ID,
		Email:          email,
		Name:           result.DisplayName,
		AvatarURL:      "",
	}, nil
}

// ProviderFactory creates OAuth providers based on configuration
type ProviderFactory struct {
	providers map[string]Provider
}

// NewProviderFactory creates a new provider factory
func NewProviderFactory(
	googleClientID, googleClientSecret, googleRedirectURL string,
	githubClientID, githubClientSecret, githubRedirectURL string,
	microsoftClientID, microsoftClientSecret, microsoftRedirectURL string,
) *ProviderFactory {
	factory := &ProviderFactory{
		providers: make(map[string]Provider),
	}

	// Google is required
	if googleClientID != "" && googleClientSecret != "" {
		factory.providers["google"] = NewGoogleProvider(googleClientID, googleClientSecret, googleRedirectURL)
	}

	// GitHub is optional
	if githubClientID != "" && githubClientSecret != "" {
		factory.providers["github"] = NewGitHubProvider(githubClientID, githubClientSecret, githubRedirectURL)
	}

	// Microsoft is optional
	if microsoftClientID != "" && microsoftClientSecret != "" {
		factory.providers["microsoft"] = NewMicrosoftProvider(microsoftClientID, microsoftClientSecret, microsoftRedirectURL)
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
