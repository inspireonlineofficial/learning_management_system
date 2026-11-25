package apperrors

import "net/http"

// AppError represents a structured application error
type AppError struct {
	Code       string              `json:"code"`
	Message    string              `json:"message"`
	Details    []map[string]string `json:"details,omitempty"`
	HTTPStatus int                 `json:"-"`
}

func (e *AppError) Error() string {
	return e.Message
}

// Common error codes
var (
	ErrEmailExists = &AppError{
		Code:       "EMAIL_EXISTS",
		Message:    "An account with this email already exists",
		HTTPStatus: http.StatusConflict,
	}

	ErrInvalidCredentials = &AppError{
		Code:       "INVALID_CREDENTIALS",
		Message:    "Invalid credentials",
		HTTPStatus: http.StatusUnauthorized,
	}

	ErrAccountInactive = &AppError{
		Code:       "ACCOUNT_INACTIVE",
		Message:    "Account is inactive",
		HTTPStatus: http.StatusForbidden,
	}

	ErrInvalidRefreshToken = &AppError{
		Code:       "INVALID_REFRESH_TOKEN",
		Message:    "Invalid or expired refresh token",
		HTTPStatus: http.StatusUnauthorized,
	}

	ErrOTPExpired = &AppError{
		Code:       "OTP_EXPIRED",
		Message:    "OTP has expired",
		HTTPStatus: http.StatusBadRequest,
	}

	ErrOTPMaxAttempts = &AppError{
		Code:       "OTP_MAX_ATTEMPTS",
		Message:    "Maximum OTP attempts exceeded",
		HTTPStatus: http.StatusBadRequest,
	}

	ErrOTPResendLimit = &AppError{
		Code:       "OTP_RESEND_LIMIT",
		Message:    "Maximum OTP resend limit reached",
		HTTPStatus: http.StatusTooManyRequests,
	}

	ErrTooManyAttempts = &AppError{
		Code:       "TOO_MANY_ATTEMPTS",
		Message:    "Too many attempts, please try again later",
		HTTPStatus: http.StatusTooManyRequests,
	}

	ErrProfileIncomplete = &AppError{
		Code:       "PROFILE_INCOMPLETE",
		Message:    "Student profile must be completed before accessing this resource",
		HTTPStatus: http.StatusForbidden,
	}

	ErrUnauthorized = &AppError{
		Code:       "UNAUTHORIZED",
		Message:    "Authentication required",
		HTTPStatus: http.StatusUnauthorized,
	}

	ErrForbidden = &AppError{
		Code:       "FORBIDDEN",
		Message:    "You do not have permission to access this resource",
		HTTPStatus: http.StatusForbidden,
	}

	ErrNotFound = &AppError{
		Code:       "NOT_FOUND",
		Message:    "Resource not found",
		HTTPStatus: http.StatusNotFound,
	}

	ErrValidation = &AppError{
		Code:       "VALIDATION_ERROR",
		Message:    "Request validation failed",
		HTTPStatus: http.StatusBadRequest,
	}

	ErrInternal = &AppError{
		Code:       "INTERNAL_ERROR",
		Message:    "An internal error occurred",
		HTTPStatus: http.StatusInternalServerError,
	}

	ErrUserNotFound = &AppError{
		Code:       "USER_NOT_FOUND",
		Message:    "User not found",
		HTTPStatus: http.StatusNotFound,
	}

	ErrOTPNotFound = &AppError{
		Code:       "OTP_NOT_FOUND",
		Message:    "OTP not found",
		HTTPStatus: http.StatusBadRequest,
	}
)

// NewValidationError creates a validation error with field details
func NewValidationError(details []map[string]string) *AppError {
	return &AppError{
		Code:       "VALIDATION_ERROR",
		Message:    "Request validation failed",
		Details:    details,
		HTTPStatus: http.StatusBadRequest,
	}
}

// NewNotFoundError creates a not found error with custom code and message
func NewNotFoundError(code, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: http.StatusNotFound,
	}
}

// NewForbiddenError creates a forbidden error with custom code and message
func NewForbiddenError(code, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: http.StatusForbidden,
	}
}

// NewInternalError creates an internal error with custom code and message
func NewInternalError(code, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: http.StatusInternalServerError,
	}
}

// NewConflictError creates a conflict error with custom code and message
func NewConflictError(code, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: http.StatusConflict,
	}
}

// NewValidationErrorWithDetails creates a validation error with custom code, message and details
func NewValidationErrorWithDetails(code, message string, details []map[string]string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		Details:    details,
		HTTPStatus: http.StatusBadRequest,
	}
}

// NewSimpleValidationError creates a validation error with code and message (convenience function)
func NewSimpleValidationError(code, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: http.StatusBadRequest,
	}
}
