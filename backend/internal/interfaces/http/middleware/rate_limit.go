package middleware

import (
	"context"
	"fmt"
	"lms-backend/internal/infrastructure/redis"
	"lms-backend/pkg/apperrors"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// RateLimiter provides Redis-backed rate limiting
type RateLimiter struct {
	redis    redis.RedisClient
	verifier JWTVerifier
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(redisClient redis.RedisClient) *RateLimiter {
	return &RateLimiter{redis: redisClient}
}

// WithIdentity lets the limiter key authenticated traffic by the verified
// user ID from the bearer token. The global limiter runs before the per-route
// authentication middleware, so without this every request — including
// authenticated SPA traffic — would share the tight anonymous per-IP budget
// (a whole classroom behind one NAT address would share 60 requests/minute).
// Only cryptographically verified tokens are trusted; an invalid token falls
// back to the per-IP budget.
func (rl *RateLimiter) WithIdentity(verifier JWTVerifier) *RateLimiter {
	rl.verifier = verifier
	return rl
}

// Limit applies rate limiting:
//   - 60 requests/minute per IP for unauthenticated requests (Requirement 1.9)
//   - 300 requests/minute per user for authenticated requests (Requirement 1.10)
func (rl *RateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var key string
		var limit int64
		var window time.Duration

		if userID := rl.identify(r); userID != "" {
			key = fmt.Sprintf("rate:user:%s", userID)
			limit = 300
			window = 60 * time.Second
		} else {
			ip := getClientIP(r)
			key = fmt.Sprintf("rate:ip:%s", ip)
			limit = 60
			window = 60 * time.Second
		}

		count, err := rl.redis.Incr(ctx, key)
		if err != nil {
			// Fail open if Redis is unavailable
			next.ServeHTTP(w, r)
			return
		}

		if count == 1 {
			rl.redis.Expire(ctx, key, window)
		}

		if count > limit {
			retryAfter := int(window.Seconds())
			w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
			writeError(w, apperrors.ErrTooManyAttempts)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// identify returns the authenticated user ID for the request, if any.
func (rl *RateLimiter) identify(r *http.Request) string {
	if userID := r.Context().Value("user_id"); userID != nil {
		return fmt.Sprintf("%v", userID)
	}
	if rl.verifier == nil {
		return ""
	}
	authHeader := r.Header.Get("Authorization")
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" || parts[1] == "" {
		return ""
	}
	userID, _, _, err := rl.verifier.VerifyToken(parts[1])
	if err != nil {
		return ""
	}
	return userID
}

// ClientIP stores the caller's IP address in the request context under
// "ip_address" so application services (login brute-force protection, audit
// logging) can use it. Before this middleware existed nothing populated that
// key, which silently disabled the login brute-force counters.
func ClientIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), "ip_address", getClientIP(r))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// getClientIP resolves the client IP address.
//
// The production Nginx config sets X-Real-IP to $remote_addr, so that header
// is trusted first. X-Forwarded-For is only a fallback, and then the
// right-most entry is used (the one appended by the nearest proxy): left-most
// entries are supplied by the client and can be forged to evade per-IP limits.
func getClientIP(r *http.Request) string {
	if xri := strings.TrimSpace(r.Header.Get("X-Real-IP")); xri != "" {
		return xri
	}

	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if last := strings.TrimSpace(ips[len(ips)-1]); last != "" {
			return last
		}
	}

	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}
