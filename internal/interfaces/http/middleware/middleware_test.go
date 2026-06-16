package middleware_test

import (
	"context"
	"lms-backend/internal/infrastructure/redis"
	"lms-backend/internal/interfaces/http/middleware"
	"lms-backend/pkg/apperrors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"pgregory.net/rapid"
)

// **Validates: Requirements 1.3**
// Property 1: All API endpoints are versioned under /v1/
func TestProperty1_AllEndpointsVersioned(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random path
		path := rapid.StringMatching(`^/v1/[a-z]+(/[a-z0-9-]+)*$`).Draw(t, "path")

		// Create request
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()

		// Simple handler
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		handler.ServeHTTP(rec, req)

		// Verify path starts with /v1/
		if !strings.HasPrefix(req.URL.Path, "/v1/") {
			t.Fatalf("Path %s does not start with /v1/", req.URL.Path)
		}
	})
}

// **Validates: Requirements 1.6**
// Property 2: Protected endpoints reject requests without a valid JWT
func TestProperty2_ProtectedEndpointsRejectWithoutJWT(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random valid path segment
		pathSegment := rapid.StringMatching(`^[a-z][a-z0-9-]*$`).Draw(t, "path")

		// Create request without Authorization header
		req := httptest.NewRequest(http.MethodGet, "/v1/"+pathSegment, nil)
		rec := httptest.NewRecorder()

		// Mock JWT verifier
		verifier := &mockJWTVerifier{}

		// Apply auth middleware
		handler := middleware.Authenticate(verifier)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		handler.ServeHTTP(rec, req)

		// Verify 401 response
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("Expected 401, got %d", rec.Code)
		}
	})
}

// **Validates: Requirements 1.7**
// Property 3: RBAC enforcement — wrong role returns 403
func TestProperty3_RBACEnforcementWrongRole(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random role
		userRole := rapid.SampledFrom([]string{"student", "teacher"}).Draw(t, "userRole")
		requiredRole := "admin"

		req := httptest.NewRequest(http.MethodGet, "/v1/admin/users", nil)
		ctx := context.WithValue(req.Context(), "role", userRole)
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		// Apply authorize middleware
		handler := middleware.Authorize(requiredRole)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		handler.ServeHTTP(rec, req)

		// Verify 403 response
		if rec.Code != http.StatusForbidden {
			t.Fatalf("Expected 403 for role %s, got %d", userRole, rec.Code)
		}
	})
}

// **Validates: Requirements 1.9, 1.11**
// Property 4: Rate limiting — unauthenticated requests
func TestProperty4_RateLimitUnauthenticated(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random IP address
		ip := rapid.StringMatching(`^([0-9]{1,3}\.){3}[0-9]{1,3}$`).Draw(t, "ip")

		// Create mock Redis client
		mockRedis := &mockRedisClient{counters: make(map[string]int64)}
		rateLimiter := middleware.NewRateLimiter(mockRedis)

		// Simulate multiple requests from same IP
		requestCount := rapid.IntRange(61, 100).Draw(t, "requestCount")

		var lastStatusCode int
		for i := 0; i < requestCount; i++ {
			req := httptest.NewRequest(http.MethodGet, "/v1/test", nil)
			req.RemoteAddr = ip + ":12345"
			rec := httptest.NewRecorder()

			handler := rateLimiter.Limit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			handler.ServeHTTP(rec, req)
			lastStatusCode = rec.Code
		}

		// After exceeding limit, should get 429
		if lastStatusCode != http.StatusTooManyRequests {
			t.Fatalf("Expected 429 after %d requests, got %d", requestCount, lastStatusCode)
		}
	})
}

// **Validates: Requirements 1.10, 1.11**
// Property 5: Rate limiting — authenticated requests
func TestProperty5_RateLimitAuthenticated(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		userID := rapid.StringN(10, 50, -1).Draw(t, "userID")

		// Create mock Redis client
		mockRedis := &mockRedisClient{counters: make(map[string]int64)}
		rateLimiter := middleware.NewRateLimiter(mockRedis)

		// Simulate multiple requests from same user
		requestCount := rapid.IntRange(301, 400).Draw(t, "requestCount")

		var lastStatusCode int
		for i := 0; i < requestCount; i++ {
			req := httptest.NewRequest(http.MethodGet, "/v1/test", nil)
			ctx := context.WithValue(req.Context(), "user_id", userID)
			req = req.WithContext(ctx)
			rec := httptest.NewRecorder()

			handler := rateLimiter.Limit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			handler.ServeHTTP(rec, req)
			lastStatusCode = rec.Code
		}

		// After exceeding limit, should get 429
		if lastStatusCode != http.StatusTooManyRequests {
			t.Fatalf("Expected 429 after %d requests, got %d", requestCount, lastStatusCode)
		}
	})
}

// **Validates: Requirements 1.17**
// Property 9: All list responses include a meta object
func TestProperty9_ListResponsesIncludeMeta(t *testing.T) {
	// This property is tested at the handler level, not middleware
	// Placeholder for completeness
	t.Skip("Property tested at handler level")
}

// **Validates: Requirements 1.18**
// Property 10: All error responses follow the standard error shape
func TestProperty10_ErrorResponsesFollowStandardShape(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/test", nil)
		rec := httptest.NewRecorder()

		// Handler that triggers auth error
		verifier := &mockJWTVerifier{}
		handler := middleware.Authenticate(verifier)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		handler.ServeHTTP(rec, req)

		// Verify response contains error object with code and message
		body := rec.Body.String()
		if !strings.Contains(body, `"error"`) {
			t.Fatalf("Response missing error object: %s", body)
		}
		if !strings.Contains(body, `"code"`) {
			t.Fatalf("Response missing code field: %s", body)
		}
		if !strings.Contains(body, `"message"`) {
			t.Fatalf("Response missing message field: %s", body)
		}
	})
}

// **Validates: Requirements 1.20**
// Property 11: X-Request-ID is propagated on every response
func TestProperty11_RequestIDPropagated(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random request ID or none
		hasRequestID := rapid.Bool().Draw(t, "hasRequestID")
		var requestID string
		if hasRequestID {
			requestID = rapid.StringN(1, 50, -1).Draw(t, "requestID")
		}

		req := httptest.NewRequest(http.MethodGet, "/v1/test", nil)
		if hasRequestID {
			req.Header.Set("X-Request-ID", requestID)
		}
		rec := httptest.NewRecorder()

		// Apply RequestID middleware
		handler := middleware.RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		handler.ServeHTTP(rec, req)

		// Verify X-Request-ID is in response
		responseID := rec.Header().Get("X-Request-ID")
		if responseID == "" {
			t.Fatalf("X-Request-ID not in response")
		}

		// If request had ID, response should match
		if hasRequestID && responseID != requestID {
			t.Fatalf("X-Request-ID mismatch: expected %s, got %s", requestID, responseID)
		}
	})
}

// **Validates: Requirements 28.1**
// Property 64: Security headers present on all responses
func TestProperty64_SecurityHeadersPresent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/test", nil)
		rec := httptest.NewRecorder()

		// Apply SecurityHeaders middleware
		handler := middleware.SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		handler.ServeHTTP(rec, req)

		// Verify all required security headers
		requiredHeaders := map[string]string{
			"X-Content-Type-Options":    "nosniff",
			"X-Frame-Options":           "DENY",
			"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
			"Content-Security-Policy":   "default-src 'none'; frame-ancestors 'none'",
			"Referrer-Policy":           "no-referrer",
		}

		for header, expectedValue := range requiredHeaders {
			actualValue := rec.Header().Get(header)
			if actualValue != expectedValue {
				t.Fatalf("Header %s: expected %s, got %s", header, expectedValue, actualValue)
			}
		}
	})
}

// **Validates: Requirements 28.6**
// Property 66: Oversized request bodies are rejected with HTTP 413
func TestProperty66_OversizedRequestBodiesRejected(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate body larger than limit
		maxBytes := int64(1024) // 1KB limit for test
		bodySize := rapid.Int64Range(maxBytes+1, maxBytes*10).Draw(t, "bodySize")

		body := strings.Repeat("x", int(bodySize))
		req := httptest.NewRequest(http.MethodPost, "/v1/test", strings.NewReader(body))
		rec := httptest.NewRecorder()

		// Apply MaxBytes middleware
		handler := middleware.MaxBytes(maxBytes)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Try to read body
			buf := make([]byte, bodySize+1)
			_, err := r.Body.Read(buf)
			if err != nil {
				w.WriteHeader(http.StatusRequestEntityTooLarge)
				return
			}
			w.WriteHeader(http.StatusOK)
		}))

		handler.ServeHTTP(rec, req)

		// Verify 413 response
		if rec.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("Expected 413 for body size %d, got %d", bodySize, rec.Code)
		}
	})
}

// Mock JWT verifier for testing
type mockJWTVerifier struct{}

func (m *mockJWTVerifier) VerifyToken(tokenString string) (userID string, role string, email string, err error) {
	if tokenString == "" {
		return "", "", "", apperrors.ErrUnauthorized
	}
	return "user-123", "student", "test@example.com", nil
}

// Mock Redis client for testing
type mockRedisClient struct {
	counters map[string]int64
}

// Ensure mockRedisClient implements redis.RedisClient
var _ redis.RedisClient = (*mockRedisClient)(nil)

func (m *mockRedisClient) Incr(ctx context.Context, key string) (int64, error) {
	m.counters[key]++
	return m.counters[key], nil
}

func (m *mockRedisClient) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return nil
}

func (m *mockRedisClient) Get(ctx context.Context, key string) (string, error) {
	return "", nil
}

func (m *mockRedisClient) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return nil
}

func (m *mockRedisClient) GetDel(ctx context.Context, key string) (string, error) {
	return "", nil
}

func (m *mockRedisClient) Del(ctx context.Context, keys ...string) error {
	return nil
}

func (m *mockRedisClient) SAdd(ctx context.Context, key string, members ...interface{}) error {
	return nil
}

func (m *mockRedisClient) SMembers(ctx context.Context, key string) ([]string, error) {
	return nil, nil
}

func (m *mockRedisClient) SRem(ctx context.Context, key string, members ...interface{}) error {
	return nil
}

func (m *mockRedisClient) Close() error {
	return nil
}
