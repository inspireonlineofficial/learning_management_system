package auth

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"testing"
	"time"

	"lms-backend/internal/domain/auth"
	"lms-backend/pkg/apperrors"

	"golang.org/x/crypto/bcrypt"
)

type memRedis struct{ data map[string]string }

func (m *memRedis) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	m.data[key] = fmt.Sprint(value)
	return nil
}
func (m *memRedis) Get(ctx context.Context, key string) (string, error) {
	v, ok := m.data[key]
	if !ok {
		return "", errors.New("nil")
	}
	return v, nil
}
func (m *memRedis) GetDel(ctx context.Context, key string) (string, error) {
	v, err := m.Get(ctx, key)
	delete(m.data, key)
	return v, err
}
func (m *memRedis) Del(ctx context.Context, keys ...string) error {
	for _, k := range keys {
		delete(m.data, k)
	}
	return nil
}
func (m *memRedis) Incr(ctx context.Context, key string) (int64, error) {
	n, _ := strconv.ParseInt(m.data[key], 10, 64)
	n++
	m.data[key] = strconv.FormatInt(n, 10)
	return n, nil
}
func (m *memRedis) Expire(ctx context.Context, key string, ttl time.Duration) error { return nil }

func newThrottleService(t *testing.T) (Service, *memRedis) {
	t.Helper()
	repo := newMockUserRepo()
	hash, _ := bcrypt.GenerateFromPassword([]byte("correct-horse1"), bcrypt.MinCost)
	h := string(hash)
	_ = repo.Create(context.Background(), &auth.User{Email: "student@example.com", PasswordHash: &h, Role: "student", Status: "active"})
	r := &memRedis{data: map[string]string{}}
	svc := NewService(ServiceDeps{UserRepo: repo, OTPRepo: newMockOTPRepo(), TokenStore: newMockTokenStore(), JWTService: &mockJWTService{}, EmailService: &mockEmailService{}, RedisClient: r})
	return svc, r
}

func ipCtx(ip string) context.Context {
	return context.WithValue(context.Background(), "ip_address", ip)
}

// Audit: the login counter never received an IP (nothing populated
// "ip_address") so brute-force protection was effectively disabled; it also
// counted successful logins, so busy shared IPs locked themselves out.
func TestAudit_LoginLocksAccountAfterRepeatedFailuresFromManyIPs(t *testing.T) {
	svc, _ := newThrottleService(t)
	for i := 0; i < maxLoginFailuresPerAcc; i++ {
		_, err := svc.Login(ipCtx(fmt.Sprintf("203.0.113.%d", i)), LoginCommand{Email: "student@example.com", Password: "wrong"})
		if !errors.Is(err, apperrors.ErrInvalidCredentials) {
			t.Fatalf("attempt %d: expected invalid credentials, got %v", i, err)
		}
	}
	_, err := svc.Login(ipCtx("198.51.100.1"), LoginCommand{Email: "student@example.com", Password: "correct-horse1"})
	if !errors.Is(err, apperrors.ErrTooManyAttempts) {
		t.Fatalf("account should be temporarily locked after %d failures, got %v", maxLoginFailuresPerAcc, err)
	}
}

func TestAudit_SuccessfulLoginsDoNotLockSharedIP(t *testing.T) {
	svc, _ := newThrottleService(t)
	for i := 0; i < 30; i++ {
		if _, err := svc.Login(ipCtx("192.0.2.10"), LoginCommand{Email: "student@example.com", Password: "correct-horse1"}); err != nil {
			t.Fatalf("login %d failed: %v", i, err)
		}
	}
}

func TestAudit_SuccessResetsAccountFailures(t *testing.T) {
	svc, r := newThrottleService(t)
	for i := 0; i < maxLoginFailuresPerAcc-1; i++ {
		_, _ = svc.Login(ipCtx("192.0.2.1"), LoginCommand{Email: "student@example.com", Password: "nope"})
	}
	if _, err := svc.Login(ipCtx("192.0.2.1"), LoginCommand{Email: "student@example.com", Password: "correct-horse1"}); err != nil {
		t.Fatalf("login should succeed before the limit: %v", err)
	}
	if _, ok := r.data[loginAccountKey("user", "student@example.com")]; ok {
		t.Fatalf("account failure counter should be cleared on success")
	}
}

func TestAudit_LoginEmailIsCaseInsensitive(t *testing.T) {
	svc, _ := newThrottleService(t)
	if _, err := svc.Login(ipCtx("192.0.2.2"), LoginCommand{Email: "  Student@Example.COM ", Password: "correct-horse1"}); err != nil {
		t.Fatalf("mixed-case email should log in: %v", err)
	}
}
