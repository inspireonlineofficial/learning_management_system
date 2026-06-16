package handlers

import (
	"encoding/json"
	"fmt"
	"lms-backend/internal/application/auth"
	"net/http"
	"net/url"
	"strings"
)

// OAuthRedirect handles GET /v1/auth/oauth/:provider
//
// @Summary      OAuth redirect
// @Description  Redirects the user to the OAuth provider's authorization page. Supports google, github, and microsoft.
// @Tags         auth
// @Produce      json
// @Param        provider  path      string  true  "OAuth provider (google, github, microsoft)"
// @Success      302       {string}  string  "Redirect to provider authorization URL"
// @Failure      400       {object}  ValidationErrorResponse
// @Router       /v1/auth/oauth/{provider} [get]
func (h *AuthHandler) OAuthRedirect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	// Extract provider from path
	provider := extractProviderFromPath(r.URL.Path)
	if provider == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid provider", nil)
		return
	}

	cmd := auth.OAuthRedirectCommand{
		Provider: provider,
		ReturnTo: r.URL.Query().Get("return"),
	}

	authURL, err := h.authService.OAuthRedirect(r.Context(), cmd)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	// Redirect to provider's authorization URL
	http.Redirect(w, r, authURL, http.StatusFound)
}

// OAuthCallback handles GET /v1/auth/oauth/:provider/callback
//
// @Summary      OAuth callback
// @Description  Handles the OAuth provider callback, exchanges the authorization code for JWT tokens
// @Tags         auth
// @Produce      json
// @Param        provider  path      string  true  "OAuth provider (google, github, microsoft)"
// @Param        code      query     string  true  "Authorization code from provider"
// @Param        state     query     string  true  "State parameter for CSRF protection"
// @Success      200       {object}  auth.OAuthResult
// @Failure      400       {object}  ValidationErrorResponse
// @Failure      401       {object}  ErrorResponse
// @Router       /v1/auth/oauth/{provider}/callback [get]
func (h *AuthHandler) OAuthCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	// Extract provider from path
	provider := extractProviderFromPath(r.URL.Path)
	if provider == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid provider", nil)
		return
	}

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" || state == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Missing code or state parameter", nil)
		return
	}

	cmd := auth.OAuthCallbackCommand{
		Provider: provider,
		Code:     code,
		State:    state,
	}

	result, err := h.authService.OAuthCallback(r.Context(), cmd)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	if h.frontendBaseURL != "" {
		redirectURL := strings.TrimRight(h.frontendBaseURL, "/") + "/auth/callback"
		fragment := url.Values{}
		fragment.Set("access_token", result.AccessToken)
		fragment.Set("refresh_token", result.RefreshToken)
		fragment.Set("expires_in", fmt.Sprintf("%d", result.ExpiresIn))
		if result.IsNewUser {
			fragment.Set("is_new_user", "true")
		}
		if result.ReturnTo != "" {
			fragment.Set("return_to", result.ReturnTo)
		}
		http.Redirect(w, r, redirectURL+"#"+fragment.Encode(), http.StatusFound)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// ConnectProvider handles POST /v1/auth/me/oauth/connect
//
// @Summary      Connect OAuth provider
// @Description  Links an OAuth provider (google, github, microsoft) to the authenticated user's account using an OAuth authorization code
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      object{provider=string,code=string,state=string}  true  "Connect provider request"
// @Success      200   {object}  object{message=string}
// @Failure      400   {object}  ValidationErrorResponse
// @Failure      401   {object}  ErrorResponse
// @Failure      409   {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/auth/me/oauth/connect [post]
func (h *AuthHandler) ConnectProvider(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil)
		return
	}

	var req struct {
		Provider string `json:"provider"`
		Code     string `json:"code"`
		State    string `json:"state"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	cmd := auth.ConnectProviderCommand{
		Provider: req.Provider,
		Code:     req.Code,
		State:    req.State,
	}

	err = h.authService.ConnectProvider(r.Context(), userID, cmd)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Provider connected successfully"})
}

// DisconnectProvider handles DELETE /v1/auth/me/oauth/:provider
//
// @Summary      Disconnect OAuth provider
// @Description  Unlinks the specified OAuth provider from the authenticated user's account
// @Tags         auth
// @Produce      json
// @Param        provider  path      string  true  "OAuth provider name (google, github, microsoft)"
// @Success      200       {object}  object{message=string}
// @Failure      400       {object}  ValidationErrorResponse
// @Failure      401       {object}  ErrorResponse
// @Failure      422       {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/auth/me/oauth/{provider} [delete]
func (h *AuthHandler) DisconnectProvider(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil)
		return
	}

	// Extract provider from path
	provider := extractProviderFromPath(r.URL.Path)
	if provider == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid provider", nil)
		return
	}

	cmd := auth.DisconnectProviderCommand{
		Provider: provider,
	}

	err = h.authService.DisconnectProvider(r.Context(), userID, cmd)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Provider disconnected successfully"})
}

// ListProviders handles GET /v1/auth/me/oauth/providers
func (h *AuthHandler) ListProviders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil)
		return
	}

	providers, err := h.authService.ListProviders(r.Context(), userID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"providers": providers})
}

// Helper function to extract provider from URL path
func extractProviderFromPath(path string) string {
	// Extract provider from paths like:
	// /v1/auth/oauth/google
	// /v1/auth/oauth/google/callback
	// /v1/auth/me/oauth/google

	parts := splitPath(path)
	for i, part := range parts {
		if part == "oauth" && i+1 < len(parts) {
			provider := parts[i+1]
			// Remove "callback" suffix if present
			if provider == "callback" && i+2 < len(parts) {
				return parts[i+2]
			}
			if provider != "connect" {
				return provider
			}
		}
	}
	return ""
}

func splitPath(path string) []string {
	var parts []string
	for _, part := range splitString(path, '/') {
		if part != "" {
			parts = append(parts, part)
		}
	}
	return parts
}

func splitString(s string, sep rune) []string {
	var parts []string
	var current string
	for _, c := range s {
		if c == sep {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}
