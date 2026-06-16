package middleware

import (
	"bytes"
	"context"
	"net/http"
)

// IdempotencyStore defines the Redis-backed store for idempotency keys.
// Key pattern: idempotency:{key} → response JSON, TTL 24h.
// Requirements: 24.3, 20.4
type IdempotencyStore interface {
	Get(ctx context.Context, key string) (string, bool, error)
	Set(ctx context.Context, key string, response string) error
}

// Idempotency returns middleware that caches responses keyed by the Idempotency-Key header.
// On a duplicate request (key already in Redis), the cached response is returned immediately.
// Requirements: 24.3, 20.4
func Idempotency(store IdempotencyStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("Idempotency-Key")
			if key == "" {
				// No key provided — pass through without caching
				next.ServeHTTP(w, r)
				return
			}

			// Check cache
			cached, found, err := store.Get(r.Context(), key)
			if err == nil && found {
				// Return cached response
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-Idempotency-Replayed", "true")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(cached))
				return
			}

			// Capture the response
			rec := &responseRecorder{
				ResponseWriter: w,
				buf:            &bytes.Buffer{},
				statusCode:     http.StatusOK,
			}
			next.ServeHTTP(rec, r)

			// Only cache successful responses (2xx)
			if rec.statusCode >= 200 && rec.statusCode < 300 {
				_ = store.Set(r.Context(), key, rec.buf.String())
			}
		})
	}
}

// responseRecorder captures the response body and status code.
type responseRecorder struct {
	http.ResponseWriter
	buf        *bytes.Buffer
	statusCode int
}

func (r *responseRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	r.buf.Write(b)
	return r.ResponseWriter.Write(b)
}
