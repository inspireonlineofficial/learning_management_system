package bkash

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"lms-backend/configs"
)

// BkashClient is the interface consumed by the application layer.
type BkashClient interface {
	CreatePayment(ctx context.Context, req CreatePaymentRequest) (*CreatePaymentResponse, error)
	ExecutePayment(ctx context.Context, paymentID string) (*ExecutePaymentResponse, error)
}

// CreatePaymentRequest holds the parameters for creating a bKash payment.
type CreatePaymentRequest struct {
	Amount                string // formatted as "%.2f"
	PayerReference        string // student UUID string
	MerchantInvoiceNumber string // "INV-{intent_uuid}"
}

// CreatePaymentResponse holds the result of a successful Create Payment call.
type CreatePaymentResponse struct {
	PaymentID string
	BkashURL  string
}

// ExecutePaymentResponse holds the result of a successful Execute Payment call.
type ExecutePaymentResponse struct {
	TrxID                 string
	Amount                string
	Currency              string
	MerchantInvoiceNumber string
}

// tokenCache holds the in-memory token state.
type tokenCache struct {
	idToken      string
	refreshToken string
	expiresAt    time.Time
}

// client is the concrete implementation of BkashClient.
type client struct {
	httpClient  *http.Client
	baseURL     string
	appKey      string
	appSecret   string
	username    string
	password    string
	callbackURL string
	mu          sync.Mutex
	tokenCache  tokenCache
}

// NewClient creates a new BkashClient from the given config.
func NewClient(cfg *configs.Config) BkashClient {
	return &client{
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		baseURL:     cfg.BkashBaseURL,
		appKey:      cfg.BkashAppKey,
		appSecret:   cfg.BkashAppSecret,
		username:    cfg.BkashUsername,
		password:    cfg.BkashPassword,
		callbackURL: cfg.BkashCallbackURL,
	}
}

// tokenGrantRefreshResponse is the shared response shape for grant and refresh endpoints.
type tokenGrantRefreshResponse struct {
	StatusCode    string      `json:"statusCode"`
	StatusMessage string      `json:"statusMessage"`
	IDToken       string      `json:"id_token"`
	RefreshToken  string      `json:"refresh_token"`
	ExpiresIn     interface{} `json:"expires_in"` // can be string or int
}

// getToken returns a valid id_token, refreshing or granting as needed.
func (c *client) getToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()

	// Return cached token if it has more than 60 seconds remaining.
	if c.tokenCache.expiresAt.Sub(now) > 60*time.Second {
		return c.tokenCache.idToken, nil
	}

	// Try refresh if we have a refresh token.
	if c.tokenCache.refreshToken != "" {
		token, err := c.doRefreshToken(ctx)
		if err == nil {
			return token, nil
		}
		// Fall through to grant on refresh failure.
	}

	// Fall back to grant token.
	return c.doGrantToken(ctx)
}

// doGrantToken calls the Grant Token endpoint and updates the cache.
func (c *client) doGrantToken(ctx context.Context) (string, error) {
	body := map[string]string{
		"app_key":    c.appKey,
		"app_secret": c.appSecret,
	}

	resp, err := c.doTokenRequest(ctx, c.baseURL+"/checkout/token/grant", body)
	if err != nil {
		return "", err
	}

	c.updateTokenCache(resp)
	return resp.IDToken, nil
}

// doRefreshToken calls the Refresh Token endpoint and updates the cache.
func (c *client) doRefreshToken(ctx context.Context) (string, error) {
	body := map[string]string{
		"app_key":       c.appKey,
		"app_secret":    c.appSecret,
		"refresh_token": c.tokenCache.refreshToken,
	}

	resp, err := c.doTokenRequest(ctx, c.baseURL+"/checkout/token/refresh", body)
	if err != nil {
		return "", err
	}

	c.updateTokenCache(resp)
	return resp.IDToken, nil
}

// doTokenRequest performs a token grant or refresh HTTP request.
func (c *client) doTokenRequest(ctx context.Context, url string, body map[string]string) (*tokenGrantRefreshResponse, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("bkash: marshal token request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("bkash: create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("username", c.username)
	req.Header.Set("password", c.password)

	httpResp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("bkash: token request failed: %w", err)
	}
	defer httpResp.Body.Close()

	respBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("bkash: read token response: %w", err)
	}

	var tokenResp tokenGrantRefreshResponse
	if err := json.Unmarshal(respBytes, &tokenResp); err != nil {
		return nil, fmt.Errorf("bkash: decode token response: %w", err)
	}

	if tokenResp.StatusCode != "0000" {
		return nil, fmt.Errorf("bkash: token error %s: %s", tokenResp.StatusCode, tokenResp.StatusMessage)
	}

	return &tokenResp, nil
}

// updateTokenCache updates the in-memory token cache from a token response.
func (c *client) updateTokenCache(resp *tokenGrantRefreshResponse) {
	expiresInSecs := 3600 // default
	switch v := resp.ExpiresIn.(type) {
	case string:
		if n, err := strconv.Atoi(v); err == nil {
			expiresInSecs = n
		}
	case float64:
		expiresInSecs = int(v)
	case int:
		expiresInSecs = v
	}

	c.tokenCache = tokenCache{
		idToken:      resp.IDToken,
		refreshToken: resp.RefreshToken,
		expiresAt:    time.Now().Add(time.Duration(expiresInSecs) * time.Second),
	}
}

// createPaymentAPIRequest is the JSON body sent to bKash Create Payment.
type createPaymentAPIRequest struct {
	Mode                  string `json:"mode"`
	PayerReference        string `json:"payerReference"`
	CallbackURL           string `json:"callbackURL"`
	Amount                string `json:"amount"`
	Currency              string `json:"currency"`
	Intent                string `json:"intent"`
	MerchantInvoiceNumber string `json:"merchantInvoiceNumber"`
}

// createPaymentAPIResponse is the JSON response from bKash Create Payment.
type createPaymentAPIResponse struct {
	StatusCode    string `json:"statusCode"`
	StatusMessage string `json:"statusMessage"`
	PaymentID     string `json:"paymentID"`
	BkashURL      string `json:"bkashURL"`
}

// CreatePayment calls the bKash Create Payment endpoint.
func (c *client) CreatePayment(ctx context.Context, req CreatePaymentRequest) (*CreatePaymentResponse, error) {
	token, err := c.getToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("bkash: get token for create payment: %w", err)
	}

	apiReq := createPaymentAPIRequest{
		Mode:                  "0011",
		PayerReference:        req.PayerReference,
		CallbackURL:           c.callbackURL,
		Amount:                req.Amount,
		Currency:              "BDT",
		Intent:                "authorization",
		MerchantInvoiceNumber: req.MerchantInvoiceNumber,
	}

	data, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("bkash: marshal create payment request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/tokenized/checkout/payment/create", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("bkash: create payment request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", token)
	httpReq.Header.Set("X-App-Key", c.appKey)

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("bkash: create payment HTTP call: %w", err)
	}
	defer httpResp.Body.Close()

	respBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("bkash: read create payment response: %w", err)
	}

	var apiResp createPaymentAPIResponse
	if err := json.Unmarshal(respBytes, &apiResp); err != nil {
		return nil, fmt.Errorf("bkash: decode create payment response: %w", err)
	}

	if apiResp.StatusCode != "0000" {
		return nil, fmt.Errorf("bkash: create payment error %s: %s", apiResp.StatusCode, apiResp.StatusMessage)
	}

	return &CreatePaymentResponse{
		PaymentID: apiResp.PaymentID,
		BkashURL:  apiResp.BkashURL,
	}, nil
}

// executePaymentAPIRequest is the JSON body sent to bKash Execute Payment.
type executePaymentAPIRequest struct {
	PaymentID string `json:"paymentID"`
}

// executePaymentAPIResponse is the JSON response from bKash Execute Payment.
type executePaymentAPIResponse struct {
	StatusCode            string `json:"statusCode"`
	StatusMessage         string `json:"statusMessage"`
	TransactionStatus     string `json:"transactionStatus"`
	TrxID                 string `json:"trxID"`
	Amount                string `json:"amount"`
	Currency              string `json:"currency"`
	MerchantInvoiceNumber string `json:"merchantInvoiceNumber"`
}

// ExecutePayment calls the bKash Execute Payment endpoint.
func (c *client) ExecutePayment(ctx context.Context, paymentID string) (*ExecutePaymentResponse, error) {
	token, err := c.getToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("bkash: get token for execute payment: %w", err)
	}

	apiReq := executePaymentAPIRequest{PaymentID: paymentID}
	data, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("bkash: marshal execute payment request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/tokenized/checkout/execute/", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("bkash: execute payment request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", token)
	httpReq.Header.Set("X-App-Key", c.appKey)

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("bkash: execute payment HTTP call: %w", err)
	}
	defer httpResp.Body.Close()

	respBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("bkash: read execute payment response: %w", err)
	}

	var apiResp executePaymentAPIResponse
	if err := json.Unmarshal(respBytes, &apiResp); err != nil {
		return nil, fmt.Errorf("bkash: decode execute payment response: %w", err)
	}

	if apiResp.StatusCode != "0000" {
		return nil, fmt.Errorf("bkash: execute payment error %s: %s", apiResp.StatusCode, apiResp.StatusMessage)
	}

	if apiResp.TransactionStatus != "Completed" {
		return nil, fmt.Errorf("bkash: execute payment transaction not completed: %s", apiResp.TransactionStatus)
	}

	return &ExecutePaymentResponse{
		TrxID:                 apiResp.TrxID,
		Amount:                apiResp.Amount,
		Currency:              apiResp.Currency,
		MerchantInvoiceNumber: apiResp.MerchantInvoiceNumber,
	}, nil
}
