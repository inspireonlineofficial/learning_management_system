package middleware

import (
	"fmt"
	"lms-backend/internal/infrastructure/redis"
	"lms-backend/pkg/apperrors"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// RateLimiter provides Redis-backed rate limiting
type RateLimiter struct {
	redis redis.RedisClient
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(redisClient redis.RedisClient) *RateLimiter {
	return &RateLimiter{redis: redisClient}
}

// Limit returns a middleware that enforces rate limits
func (rl *RateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Check if user is authenticated
		userID := ctx.Value("user_id")
		var key string
		var limit int64
		var window time.Duration

		if userID != nil {
			// Authenticated: 300 req/min per user
			key = fmt.Sprintf("rate:user:%v", userID)
			limit = 300
			window = 60 * time.Second
		} else {
			// Unauthenticated: 60 req/min per IP
			ip := getClientIP(r)
			key = fmt.Sprintf("rate:ip:%s", ip)
			limit = 60
			window = 60 * time.Second
		}

		// Increment counter
		count, err := rl.redis.Incr(ctx, key)
		if err != nil {
			// On Redis error, allow request (fail open)
			next.ServeHTTP(w, r)
			return
		}

		// Set expiry on first request
		if count == 1 {
			rl.redis.Expire(ctx, key, window)
		}

		// Check limit
		if count > limit {
			retryAfter := int(window.Seconds())
			w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
			writeError(w, apperrors.ErrTooManyAttempts)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}
