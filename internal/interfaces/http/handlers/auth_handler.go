package handlers

import (
	"encoding/json"
	"lms-backend/internal/application/auth"
	"lms-backend/pkg/apperrors"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

// AuthHandler handles authentication HTTP requests
type AuthHandler struct {
	authService     auth.Service
	frontendBaseURL string
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authService auth.Service, frontendBaseURL string) *AuthHandler {
	return &AuthHandler{
		authService:     authService,
		frontendBaseURL: frontendBaseURL,
	}
}

// Register handles POST /v1/auth/register
//
// @Summary      Register a new user
// @Description  Creates a new student account and sends an OTP verification email
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      object{full_name=string,email=string,password=string,confirm_password=string}  true  "Registration request"
// @Success      201   {object}  auth.RegisterResult
// @Failure      400   {object}  ValidationErrorResponse
// @Failure      409   {object}  ErrorResponse
// @Router       /v1/auth/register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	var req struct {
		FullName        string `json:"full_name"`
		Email           string `json:"email"`
		Password        string `json:"password"`
		ConfirmPassword string `json:"confirm_password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	cmd := auth.RegisterCommand{
		FullName:        req.FullName,
		Email:           req.Email,
		Password:        req.Password,
		ConfirmPassword: req.ConfirmPassword,
		Role:            "student",
	}

	result, err := h.authService.Register(r.Context(), cmd)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, result)
}

// VerifyOTP handles POST /v1/auth/verify-otp
//
// @Summary      Verify OTP
// @Description  Verifies the one-time password sent to the user's email and returns JWT tokens on success
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      object{email=string,otp=string}  true  "OTP verification request"
// @Success      200   {object}  auth.TokenPair
// @Failure      400   {object}  ValidationErrorResponse
// @Failure      401   {object}  ErrorResponse
// @Router       /v1/auth/verify-otp [post]
func (h *AuthHandler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	var req struct {
		Email string `json:"email"`
		OTP   string `json:"otp"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	cmd := auth.VerifyOTPCommand{
		Email: req.Email,
		OTP:   req.OTP,
	}

	tokens, err := h.authService.VerifyOTP(r.Context(), cmd)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, tokens)
}

// ResendOTP handles POST /v1/auth/resend-otp
//
// @Summary      Resend OTP
// @Description  Resends the one-time password to the user's email address
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      object{email=string}  true  "Resend OTP request"
// @Success      200   {object}  object{message=string}
// @Failure      400   {object}  ValidationErrorResponse
// @Failure      429   {object}  ErrorResponse
// @Router       /v1/auth/resend-otp [post]
func (h *AuthHandler) ResendOTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	var req struct {
		Email string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	cmd := auth.ResendOTPCommand{
		Email: req.Email,
	}

	err := h.authService.ResendOTP(r.Context(), cmd)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "OTP resent successfully"})
}

// Login handles POST /v1/auth/login
//
// @Summary      Login
// @Description  Authenticates a user with email and password and returns JWT access and refresh tokens
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      object{email=string,password=string,remember_me=bool}  true  "Login request"
// @Success      200   {object}  auth.TokenPair
// @Failure      400   {object}  ValidationErrorResponse
// @Failure      401   {object}  ErrorResponse
// @Failure      403   {object}  ErrorResponse
// @Router       /v1/auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	var req struct {
		Email      string `json:"email"`
		Password   string `json:"password"`
		RememberMe bool   `json:"remember_me"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	cmd := auth.LoginCommand{
		Email:      req.Email,
		Password:   req.Password,
		RememberMe: req.RememberMe,
	}

	tokens, err := h.authService.Login(r.Context(), cmd)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, tokens)
}

// RefreshToken handles POST /v1/auth/refresh
//
// @Summary      Refresh tokens
// @Description  Exchanges a valid refresh token for a new JWT access and refresh token pair
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      object{refresh_token=string}  true  "Refresh token request"
// @Success      200   {object}  auth.TokenPair
// @Failure      400   {object}  ValidationErrorResponse
// @Failure      401   {object}  ErrorResponse
// @Router       /v1/auth/refresh [post]
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	var req struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	tokens, err := h.authService.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, tokens)
}

// Logout handles POST /v1/auth/logout
//
// @Summary      Logout
// @Description  Invalidates the provided refresh token, ending the user's session
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      object{refresh_token=string}  true  "Logout request"
// @Success      200   {object}  object{message=string}
// @Failure      400   {object}  ValidationErrorResponse
// @Failure      401   {object}  ErrorResponse
// @Router       /v1/auth/logout [post]
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	var req struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	err := h.authService.Logout(r.Context(), req.RefreshToken)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Logged out successfully"})
}

// ForgotPassword handles POST /v1/auth/forgot-password
//
// @Summary      Forgot password
// @Description  Sends a password reset link to the provided email address. Always returns 200 to prevent email enumeration.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      object{email=string}  true  "Forgot password request"
// @Success      200   {object}  object{message=string}
// @Failure      400   {object}  ValidationErrorResponse
// @Router       /v1/auth/forgot-password [post]
func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	var req struct {
		Email string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	// Always return 200 regardless of email existence (prevent enumeration)
	cmd := auth.ForgotPasswordCommand{
		Email: req.Email,
	}
	_ = h.authService.ForgotPassword(r.Context(), cmd)

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "If an account exists for this email, a reset link has been sent",
	})
}

// ResetPassword handles POST /v1/auth/reset-password
//
// @Summary      Reset password
// @Description  Resets the user's password using a valid reset token received via email
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      object{token=string,new_password=string,confirm_password=string}  true  "Reset password request"
// @Success      200   {object}  object{message=string}
// @Failure      400   {object}  ValidationErrorResponse
// @Failure      401   {object}  ErrorResponse
// @Router       /v1/auth/reset-password [post]
func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	var req struct {
		Token           string `json:"token"`
		NewPassword     string `json:"new_password"`
		ConfirmPassword string `json:"confirm_password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	cmd := auth.ResetPasswordCommand{
		Token:           req.Token,
		NewPassword:     req.NewPassword,
		ConfirmPassword: req.ConfirmPassword,
	}

	err := h.authService.ResetPassword(r.Context(), cmd)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Password reset successfully"})
}

// ChangePassword handles POST /v1/auth/me/change-password
//
// @Summary      Change password
// @Description  Changes the authenticated user's password by verifying the current password first
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      object{current_password=string,new_password=string,confirm_password=string}  true  "Change password request"
// @Success      200   {object}  object{message=string}
// @Failure      400   {object}  ValidationErrorResponse
// @Failure      401   {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/auth/me/change-password [post]
func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
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
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
		ConfirmPassword string `json:"confirm_password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	cmd := auth.ChangePasswordCommand{
		CurrentPassword: req.CurrentPassword,
		NewPassword:     req.NewPassword,
		ConfirmPassword: req.ConfirmPassword,
	}

	err = h.authService.ChangePassword(r.Context(), userID, cmd)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Password changed successfully"})
}

// UpdateProfile handles PATCH /v1/auth/me
//
// @Summary      Update profile
// @Description  Updates the authenticated user's profile information
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      object{full_name=string}  false  "Update profile request"
// @Success      200   {object}  auth.ProfileResult
// @Failure      400   {object}  ValidationErrorResponse
// @Failure      401   {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/auth/me [patch]
func (h *AuthHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil)
		return
	}

	var req struct {
		FullName *string `json:"full_name,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	cmd := auth.UpdateProfileCommand{
		FullName: req.FullName,
	}

	user, err := h.authService.UpdateProfile(r.Context(), userID, cmd)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, user)
}

// GetProfile handles GET /v1/auth/me
func (h *AuthHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil)
		return
	}

	profile, err := h.authService.GetProfile(r.Context(), userID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, profile)
}

// GetUserSettings handles GET /v1/auth/me/settings.
func (h *AuthHandler) GetUserSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil)
		return
	}

	settings, err := h.authService.GetUserSettings(r.Context(), userID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, settings)
}

// UpdateUserSettings handles PATCH /v1/auth/me/settings.
func (h *AuthHandler) UpdateUserSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil)
		return
	}

	var req auth.UpdateUserSettingsCommand
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	settings, err := h.authService.UpdateUserSettings(r.Context(), userID, req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, settings)
}

// ImpersonateUser handles POST /v1/admin/users/:userId/impersonate.
func (h *AuthHandler) ImpersonateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	actorID, err := getUserIDFromContext(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/v1/admin/users/")
	targetUserIDStr := strings.Split(path, "/")[0]
	targetUserID, err := uuid.Parse(targetUserIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_USER_ID", "Invalid user ID", nil)
		return
	}

	result, err := h.authService.ImpersonateUser(r.Context(), auth.ImpersonateUserCommand{
		ActorID:      actorID,
		TargetUserID: targetUserID,
	})
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// Helper functions

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, code, message string, details []map[string]string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
			"details": details,
		},
	})
}

func handleServiceError(w http.ResponseWriter, err error) {
	appErr, ok := err.(*apperrors.AppError)
	if !ok {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred", nil)
		return
	}

	status := http.StatusInternalServerError
	switch appErr.Code {
	case "EMAIL_EXISTS", "ALREADY_ENROLLED":
		status = http.StatusConflict
	case "INVALID_CREDENTIALS", "INVALID_REFRESH_TOKEN", "INVALID_RESET_TOKEN", "TOKEN_ALREADY_USED":
		status = http.StatusUnauthorized
	case "ACCOUNT_INACTIVE", "PROFILE_INCOMPLETE":
		status = http.StatusForbidden
	case "OTP_EXPIRED", "OTP_MAX_ATTEMPTS", "VALIDATION_ERROR":
		status = http.StatusBadRequest
	case "OTP_RESEND_LIMIT", "TOO_MANY_ATTEMPTS":
		status = http.StatusTooManyRequests
	case "NOT_FOUND":
		status = http.StatusNotFound
	case "CANNOT_REMOVE_ONLY_AUTH_METHOD", "PAST_DEADLINE", "FILE_TOO_LARGE":
		status = http.StatusUnprocessableEntity
	}

	writeError(w, status, appErr.Code, appErr.Message, appErr.Details)
}

func getIPAddress(r *http.Request) string {
	// Check X-Forwarded-For header first
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		// Take the first IP if multiple are present
		ips := strings.Split(forwarded, ",")
		return strings.TrimSpace(ips[0])
	}

	// Check X-Real-IP header
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	// Fall back to RemoteAddr
	return r.RemoteAddr
}
