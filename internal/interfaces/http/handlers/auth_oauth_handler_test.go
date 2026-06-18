package handlers

import (
	"net/http/httptest"
	"testing"
)

func TestOAuthFrontendRedirectBaseURLUsesForwardedOriginWhenConfiguredForLocalhost(t *testing.T) {
	handler := NewAuthHandler(nil, "http://localhost:3000")
	req := httptest.NewRequest("GET", "http://127.0.0.1:8080/v1/auth/oauth/google/callback", nil)
	req.Host = "inspire-online.com"
	req.Header.Set("X-Forwarded-Proto", "https")

	got := handler.oauthFrontendRedirectBaseURL(req)
	if got != "https://inspire-online.com" {
		t.Fatalf("expected production origin, got %q", got)
	}
}

func TestOAuthFrontendRedirectBaseURLKeepsConfiguredProductionURL(t *testing.T) {
	handler := NewAuthHandler(nil, "https://inspire-online.com")
	req := httptest.NewRequest("GET", "http://127.0.0.1:8080/v1/auth/oauth/google/callback", nil)
	req.Host = "internal.example"
	req.Header.Set("X-Forwarded-Proto", "https")

	got := handler.oauthFrontendRedirectBaseURL(req)
	if got != "https://inspire-online.com" {
		t.Fatalf("expected configured production URL, got %q", got)
	}
}

func TestOAuthFrontendRedirectBaseURLKeepsLocalhostForLocalRequests(t *testing.T) {
	handler := NewAuthHandler(nil, "http://localhost:3000")
	req := httptest.NewRequest("GET", "http://localhost:8080/v1/auth/oauth/google/callback", nil)
	req.Host = "localhost:8080"

	got := handler.oauthFrontendRedirectBaseURL(req)
	if got != "http://localhost:3000" {
		t.Fatalf("expected local frontend URL, got %q", got)
	}
}
