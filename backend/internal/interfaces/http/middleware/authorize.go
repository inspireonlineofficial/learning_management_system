package middleware

import (
	"lms-backend/pkg/apperrors"
	"net/http"
)

// Authorize middleware checks if the authenticated user has the required role
func Authorize(requiredRole string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := r.Context().Value("role")
			if role == nil {
				writeError(w, apperrors.ErrUnauthorized)
				return
			}

			userRole := role.(string)

			// Admin has access to everything
			if userRole == "admin" {
				next.ServeHTTP(w, r)
				return
			}

			// Check if user role matches required role
			if userRole != requiredRole {
				writeError(w, apperrors.ErrForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
