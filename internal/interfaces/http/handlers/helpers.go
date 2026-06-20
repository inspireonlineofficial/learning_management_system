package handlers

import (
	"encoding/json"
	"lms-backend/pkg/apperrors"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

// writeErrorResponse writes an error response in the standard format
func writeErrorResponse(w http.ResponseWriter, err error) {
	appErr, ok := err.(*apperrors.AppError)
	if !ok {
		appErr = &apperrors.AppError{
			Code:       "INTERNAL_ERROR",
			Message:    "An internal error occurred",
			HTTPStatus: http.StatusInternalServerError,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(appErr.HTTPStatus)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]interface{}{
			"code":    appErr.Code,
			"message": appErr.Message,
			"details": appErr.Details,
		},
	})
}

// writeJSONResponse writes a JSON response with the given status code
func writeJSONResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// getUserIDFromContext extracts the user ID from the request context
func getUserIDFromContext(r *http.Request) (uuid.UUID, error) {
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		return uuid.Nil, apperrors.ErrUnauthorized
	}
	return userID, nil
}

func getUserRoleFromContext(r *http.Request) (string, error) {
	role, ok := r.Context().Value("role").(string)
	if !ok || role == "" {
		return "", apperrors.ErrUnauthorized
	}
	return role, nil
}

func decodeJSONBody(r *http.Request, target interface{}) error {
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		return apperrors.NewSimpleValidationError("INVALID_JSON", "invalid request body")
	}
	return nil
}

func requestIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		return strings.TrimSpace(strings.Split(forwarded, ",")[0])
	}
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}
	return r.RemoteAddr
}
