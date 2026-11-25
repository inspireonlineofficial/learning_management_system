package middleware

import (
	"lms-backend/pkg/apperrors"
	"net/http"
)

// ProfileGateChecker defines the interface for checking profile completion
type ProfileGateChecker interface {
	IsProfileComplete(userID string) (bool, error)
}

// ProfileGate middleware checks if student profile is complete
func ProfileGate(checker ProfileGateChecker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := r.Context().Value("role")
			if role == nil {
				writeError(w, apperrors.ErrUnauthorized)
				return
			}

			// Only apply to students
			if role.(string) != "student" {
				next.ServeHTTP(w, r)
				return
			}

			userID := r.Context().Value("user_id")
			if userID == nil {
				writeError(w, apperrors.ErrUnauthorized)
				return
			}

			// Check profile completion
			complete, err := checker.IsProfileComplete(userID.(string))
			if err != nil {
				writeError(w, apperrors.ErrInternal)
				return
			}

			if !complete {
				writeError(w, apperrors.ErrProfileIncomplete)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
