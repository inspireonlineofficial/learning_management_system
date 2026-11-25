package middleware

import (
	"net/http/httptest"
	"testing"
)

func TestAudit_ClientIPIgnoresSpoofedForwardedFor(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "10.0.0.5:1234"
	r.Header.Set("X-Forwarded-For", "1.2.3.4, 203.0.113.9")
	if got := getClientIP(r); got != "203.0.113.9" {
		t.Fatalf("expected the proxy-appended (right-most) address, got %q", got)
	}
	r.Header.Set("X-Real-IP", "198.51.100.7")
	if got := getClientIP(r); got != "198.51.100.7" {
		t.Fatalf("expected X-Real-IP from the trusted proxy, got %q", got)
	}
	r2 := httptest.NewRequest("GET", "/", nil)
	r2.RemoteAddr = "[2001:db8::1]:443"
	if got := getClientIP(r2); got != "2001:db8::1" {
		t.Fatalf("IPv6 remote address not parsed: %q", got)
	}
}

type stubVerifier struct{ user string }

func (s stubVerifier) VerifyToken(token string) (string, string, string, error) {
	if token == "good" {
		return s.user, "student", "a@b.c", nil
	}
	return "", "", "", errBadToken
}

type tokenErr string

func (e tokenErr) Error() string { return string(e) }

const errBadToken = tokenErr("bad token")

func TestAudit_RateLimiterKeysAuthenticatedTrafficByUser(t *testing.T) {
	rl := NewRateLimiter(nil).WithIdentity(stubVerifier{user: "u-1"})
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "Bearer good")
	if got := rl.identify(r); got != "u-1" {
		t.Fatalf("expected verified user id, got %q", got)
	}
	r.Header.Set("Authorization", "Bearer forged")
	if got := rl.identify(r); got != "" {
		t.Fatalf("unverified tokens must fall back to IP limits, got %q", got)
	}
}
