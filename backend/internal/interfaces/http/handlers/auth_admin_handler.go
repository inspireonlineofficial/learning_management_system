package handlers

import (
	"encoding/json"
	"lms-backend/internal/application/auth"
	"net/http"
)

// AdminLogin handles POST /v1/auth/admin/login
//
// @Summary      Admin login
// @Description  Authenticates an admin user with username and password; returns an OTP session token for two-factor verification
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      object{username=string,password=string}  true  "Admin login request"
// @Success      200   {object}  auth.OTPSessionResult
// @Failure      400   {object}  ValidationErrorResponse
// @Failure      401   {object}  ErrorResponse
// @Router       /v1/auth/admin/login [post]
func (h *AuthHandler) AdminLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	cmd := auth.AdminLoginCommand{
		Username: req.Username,
		Password: req.Password,
	}

	result, err := h.authService.AdminLogin(r.Context(), cmd)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// AdminVerifyOTP handles POST /v1/auth/admin/verify-otp
//
// @Summary      Admin verify OTP
// @Description  Verifies the admin's OTP using the session token from AdminLogin and returns JWT tokens on success
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      object{otp_session_token=string,otp=string}  true  "Admin OTP verification request"
// @Success      200   {object}  auth.TokenPair
// @Failure      400   {object}  ValidationErrorResponse
// @Failure      401   {object}  ErrorResponse
// @Router       /v1/auth/admin/verify-otp [post]
func (h *AuthHandler) AdminVerifyOTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	var req struct {
		OTPSessionToken string `json:"otp_session_token"`
		OTP             string `json:"otp"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	cmd := auth.AdminVerifyOTPCommand{
		OTPSessionToken: req.OTPSessionToken,
		OTP:             req.OTP,
	}

	tokens, err := h.authService.AdminVerifyOTP(r.Context(), cmd)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, tokens)
}

// AdminResendOTP handles POST /v1/auth/admin/resend-otp
//
// @Summary      Admin resend OTP
// @Description  Resends the OTP for an active admin login session identified by the OTP session token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      object{otp_session_token=string}  true  "Admin resend OTP request"
// @Success      200   {object}  object{message=string}
// @Failure      400   {object}  ValidationErrorResponse
// @Failure      401   {object}  ErrorResponse
// @Failure      429   {object}  ErrorResponse
// @Router       /v1/auth/admin/resend-otp [post]
func (h *AuthHandler) AdminResendOTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	var req struct {
		OTPSessionToken string `json:"otp_session_token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	err := h.authService.AdminResendOTP(r.Context(), req.OTPSessionToken)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "OTP resent successfully"})
}
