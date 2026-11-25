package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The refund and order endpoints rely on an Idempotency-Key header; a
// browser on another origin can only send it if the preflight allows it.
func TestCORSAllowsIdempotencyKey(t *testing.T) {
	h := CORS([]string{"https://app.example.com"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	req := httptest.NewRequest(http.MethodOptions, "/v1/bookshop/orders", nil)
	req.Header.Set("Origin", "https://app.example.com")
	req.Header.Set("Access-Control-Request-Headers", "idempotency-key")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if !strings.Contains(rec.Header().Get("Access-Control-Allow-Headers"), "Idempotency-Key") {
		t.Fatalf("preflight must allow Idempotency-Key, got %q", rec.Header().Get("Access-Control-Allow-Headers"))
	}
}
