package bkash

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// newTestClient creates a client wired to the given base URL with test credentials.
func newTestClient(baseURL string) *client {
	return &client{
		httpClient:  &http.Client{Timeout: 5 * time.Second},
		baseURL:     baseURL,
		appKey:      "test-app-key",
		appSecret:   "test-app-secret",
		username:    "test-username",
		password:    "test-password",
		callbackURL: "https://example.com/callback",
	}
}

// grantResponse returns a valid grant/refresh JSON response body.
func grantResponse(idToken, refreshToken string) []byte {
	b, _ := json.Marshal(map[string]interface{}{
		"statusCode":    "0000",
		"statusMessage": "Successful",
		"id_token":      idToken,
		"refresh_token": refreshToken,
		"expires_in":    3600,
	})
	return b
}

// TestGrantRequestBodyAndHeaders verifies that the grant request sends app_key, app_secret,
// and the username/password headers as required by the bKash spec.
func TestGrantRequestBodyAndHeaders(t *testing.T) {
	var capturedBody []byte
	var capturedUsername, capturedPassword string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUsername = r.Header.Get("username")
		capturedPassword = r.Header.Get("password")
		capturedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.Write(grantResponse("id-token-123", "refresh-token-abc"))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	token, err := c.getToken(context.Background())
	if err != nil {
		t.Fatalf("getToken returned error: %v", err)
	}
	if token != "id-token-123" {
		t.Errorf("expected token id-token-123, got %s", token)
	}

	// Verify headers.
	if capturedUsername != "test-username" {
		t.Errorf("expected username header 'test-username', got %q", capturedUsername)
	}
	if capturedPassword != "test-password" {
		t.Errorf("expected password header 'test-password', got %q", capturedPassword)
	}

	// Verify body contains app_key and app_secret.
	var body map[string]string
	if err := json.Unmarshal(capturedBody, &body); err != nil {
		t.Fatalf("failed to parse grant request body: %v", err)
	}
	if body["app_key"] != "test-app-key" {
		t.Errorf("expected app_key 'test-app-key', got %q", body["app_key"])
	}
	if body["app_secret"] != "test-app-secret" {
		t.Errorf("expected app_secret 'test-app-secret', got %q", body["app_secret"])
	}
}

// TestRefreshUsesStoredRefreshToken verifies that when a refresh_token is cached,
// the refresh endpoint is called with that token.
func TestRefreshUsesStoredRefreshToken(t *testing.T) {
	var capturedPath string
	var capturedBody []byte

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.Write(grantResponse("new-id-token", "new-refresh-token"))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	// Seed the cache with a near-expired token and a refresh token.
	c.tokenCache = tokenCache{
		idToken:      "old-id-token",
		refreshToken: "stored-refresh-token",
		expiresAt:    time.Now().Add(30 * time.Second), // within 60s → needs refresh
	}

	token, err := c.getToken(context.Background())
	if err != nil {
		t.Fatalf("getToken returned error: %v", err)
	}
	if token != "new-id-token" {
		t.Errorf("expected new-id-token, got %s", token)
	}

	// Verify the refresh endpoint was called.
	if capturedPath != "/checkout/token/refresh" {
		t.Errorf("expected refresh endpoint, got path %q", capturedPath)
	}

	// Verify the body contains the stored refresh_token.
	var body map[string]string
	if err := json.Unmarshal(capturedBody, &body); err != nil {
		t.Fatalf("failed to parse refresh request body: %v", err)
	}
	if body["refresh_token"] != "stored-refresh-token" {
		t.Errorf("expected refresh_token 'stored-refresh-token', got %q", body["refresh_token"])
	}
}

// TestGetTokenErrorWhenBothFail verifies that getToken returns an error when both
// the refresh and grant endpoints return non-0000 status codes.
func TestGetTokenErrorWhenBothFail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		b, _ := json.Marshal(map[string]string{
			"statusCode":    "2001",
			"statusMessage": "Invalid credentials",
		})
		w.Write(b)
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	// Seed a near-expired token with a refresh token so both paths are exercised.
	c.tokenCache = tokenCache{
		idToken:      "old-token",
		refreshToken: "old-refresh",
		expiresAt:    time.Now().Add(10 * time.Second),
	}

	_, err := c.getToken(context.Background())
	if err == nil {
		t.Fatal("expected error when both refresh and grant fail, got nil")
	}
}

// TestCreatePaymentAmountSerializedAsTwoDecimalString verifies that CreatePayment
// sends the amount as a string with 2 decimal places (e.g., "100.00").
func TestCreatePaymentAmountSerializedAsTwoDecimalString(t *testing.T) {
	var capturedCreateBody []byte

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/checkout/token/grant":
			w.Write(grantResponse("tok", "ref"))
		case "/tokenized/checkout/payment/create":
			capturedCreateBody, _ = io.ReadAll(r.Body)
			b, _ := json.Marshal(map[string]string{
				"statusCode": "0000",
				"paymentID":  "pay-001",
				"bkashURL":   "https://bkash.example.com/pay",
			})
			w.Write(b)
		}
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, err := c.CreatePayment(context.Background(), CreatePaymentRequest{
		Amount:                "100.00",
		PayerReference:        "student-uuid",
		MerchantInvoiceNumber: "INV-001",
	})
	if err != nil {
		t.Fatalf("CreatePayment returned error: %v", err)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(capturedCreateBody, &body); err != nil {
		t.Fatalf("failed to parse create payment body: %v", err)
	}

	amount, ok := body["amount"].(string)
	if !ok {
		t.Fatalf("expected amount to be a string, got %T", body["amount"])
	}
	if amount != "100.00" {
		t.Errorf("expected amount '100.00', got %q", amount)
	}
}

// TestExecutePaymentDeserializesResponseFields verifies that ExecutePayment correctly
// deserializes trxID, amount, currency, and merchantInvoiceNumber from the response.
func TestExecutePaymentDeserializesResponseFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/checkout/token/grant":
			w.Write(grantResponse("tok", "ref"))
		case "/tokenized/checkout/execute/":
			b, _ := json.Marshal(map[string]string{
				"statusCode":            "0000",
				"transactionStatus":     "Completed",
				"trxID":                 "TRX123456",
				"amount":                "250.00",
				"currency":              "BDT",
				"merchantInvoiceNumber": "INV-abc-123",
			})
			w.Write(b)
		}
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	resp, err := c.ExecutePayment(context.Background(), "pay-xyz")
	if err != nil {
		t.Fatalf("ExecutePayment returned error: %v", err)
	}

	if resp.TrxID != "TRX123456" {
		t.Errorf("expected TrxID 'TRX123456', got %q", resp.TrxID)
	}
	if resp.Amount != "250.00" {
		t.Errorf("expected Amount '250.00', got %q", resp.Amount)
	}
	if resp.Currency != "BDT" {
		t.Errorf("expected Currency 'BDT', got %q", resp.Currency)
	}
	if resp.MerchantInvoiceNumber != "INV-abc-123" {
		t.Errorf("expected MerchantInvoiceNumber 'INV-abc-123', got %q", resp.MerchantInvoiceNumber)
	}
}

// TestCachedTokenNotExpired verifies that a cached token with TTL > 60s is returned
// without calling the grant endpoint.
func TestCachedTokenNotExpired(t *testing.T) {
	grantCalled := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		grantCalled++
		w.Header().Set("Content-Type", "application/json")
		w.Write(grantResponse("new-tok", "new-ref"))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	c.tokenCache = tokenCache{
		idToken:      "cached-token",
		refreshToken: "",
		expiresAt:    time.Now().Add(120 * time.Second), // > 60s remaining
	}

	token, err := c.getToken(context.Background())
	if err != nil {
		t.Fatalf("getToken returned error: %v", err)
	}
	if token != "cached-token" {
		t.Errorf("expected cached-token, got %s", token)
	}
	if grantCalled != 0 {
		t.Errorf("expected grant endpoint not to be called, but it was called %d time(s)", grantCalled)
	}
}

// TestConcurrentTokenGrantCalledOnce verifies that concurrent calls to getToken
// with no cached token result in the grant endpoint being called exactly once.
func TestConcurrentTokenGrantCalledOnce(t *testing.T) {
	var mu sync.Mutex
	grantCalled := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/checkout/token/grant" {
			mu.Lock()
			grantCalled++
			mu.Unlock()
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(grantResponse("shared-token", "shared-refresh"))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)

	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_, _ = c.getToken(context.Background())
		}()
	}
	wg.Wait()

	if grantCalled != 1 {
		t.Errorf("expected grant endpoint called exactly once, got %d", grantCalled)
	}
}
