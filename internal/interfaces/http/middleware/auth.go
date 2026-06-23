package middleware

import (
	"context"
	"lms-backend/pkg/apperrors"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

// JWTVerifier defines the interface for JWT verification
type JWTVerifier interface {
	VerifyToken(tokenString string) (userID string, role string, email string, err error)
}

// AuthenticateMiddleware wraps JWT verification
type AuthenticateMiddleware struct {
	verifier JWTVerifier
}

// NewAuthenticateMiddleware creates a new authenticate middleware
func NewAuthenticateMiddleware(verifier JWTVerifier) *AuthenticateMiddleware {
	return &AuthenticateMiddleware{
		verifier: verifier,
	}
}

// Authenticate middleware validates JWT and injects user context
func (m *AuthenticateMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeError(w, apperrors.ErrUnauthorized)
			return
		}

		// Check Bearer prefix
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			writeError(w, apperrors.ErrUnauthorized)
			return
		}

		token := parts[1]

		// Verify token
		userID, role, email, err := m.verifier.VerifyToken(token)
		if err != nil {
			writeError(w, apperrors.ErrUnauthorized)
			return
		}

		// Inject user info into context
		parsedUserID, parseErr := uuid.Parse(userID)
		if parseErr != nil {
			writeError(w, apperrors.ErrUnauthorized)
			return
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, "user_id", parsedUserID)
		ctx = context.WithValue(ctx, "role", role)
		ctx = context.WithValue(ctx, "email", email)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// AuthenticateOptional middleware validates JWT if present, and injects user context. It does not fail if JWT is missing or invalid.
func (m *AuthenticateMiddleware) AuthenticateOptional(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			next.ServeHTTP(w, r)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			next.ServeHTTP(w, r)
			return
		}

		token := parts[1]

		userID, role, email, err := m.verifier.VerifyToken(token)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		parsedUserID, parseErr := uuid.Parse(userID)
		if parseErr != nil {
			next.ServeHTTP(w, r)
			return
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, "user_id", parsedUserID)
		ctx = context.WithValue(ctx, "role", role)
		ctx = context.WithValue(ctx, "email", email)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Authenticate middleware validates JWT and injects user context (legacy function-based version)
func Authenticate(verifier JWTVerifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeError(w, apperrors.ErrUnauthorized)
				return
			}

			// Check Bearer prefix
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				writeError(w, apperrors.ErrUnauthorized)
				return
			}

			token := parts[1]

			// Verify token
			userID, role, email, err := verifier.VerifyToken(token)
			if err != nil {
				writeError(w, apperrors.ErrUnauthorized)
				return
			}

			// Inject user info into context
			parsedUserID, parseErr := uuid.Parse(userID)
			if parseErr != nil {
				writeError(w, apperrors.ErrUnauthorized)
				return
			}

			ctx := r.Context()
			ctx = context.WithValue(ctx, "user_id", parsedUserID)
			ctx = context.WithValue(ctx, "role", role)
			ctx = context.WithValue(ctx, "email", email)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
