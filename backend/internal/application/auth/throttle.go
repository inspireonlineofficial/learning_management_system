package auth

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Brute-force protection limits.
//
// Failures are counted (not every attempt) so that legitimate users who log in
// repeatedly from a shared address — a school lab behind one NAT IP — are not
// locked out. Two independent budgets apply:
//   - per IP:      stops one source from guessing many accounts;
//   - per account: stops distributed guessing against one account
//     (credential stuffing from many IPs).
const (
	loginFailureWindow     = 15 * time.Minute
	maxLoginFailuresPerIP  = 20
	maxLoginFailuresPerAcc = 10

	passwordResetWindow     = time.Hour
	maxPasswordResetsPerAcc = 3
)

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func loginIPKey(scope, ip string) string {
	return fmt.Sprintf("rate:login:%s:ip:%s", scope, ip)
}

func loginAccountKey(scope, account string) string {
	return fmt.Sprintf("rate:login:%s:acct:%s", scope, strings.ToLower(account))
}

func (s *authService) counterValue(ctx context.Context, key string) int64 {
	if s.deps.RedisClient == nil {
		return 0
	}
	raw, err := s.deps.RedisClient.Get(ctx, key)
	if err != nil || raw == "" {
		return 0
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0
	}
	return n
}

func (s *authService) bumpCounter(ctx context.Context, key string, window time.Duration) int64 {
	if s.deps.RedisClient == nil {
		return 0
	}
	count, err := s.deps.RedisClient.Incr(ctx, key)
	if err != nil {
		return 0
	}
	if count == 1 {
		_ = s.deps.RedisClient.Expire(ctx, key, window)
	}
	return count
}

// loginBlocked reports whether the IP or the account has exhausted its
// failed-login budget.
func (s *authService) loginBlocked(ctx context.Context, scope, ip, account string) bool {
	if ip != "" && s.counterValue(ctx, loginIPKey(scope, ip)) >= maxLoginFailuresPerIP {
		return true
	}
	if account != "" && s.counterValue(ctx, loginAccountKey(scope, account)) >= maxLoginFailuresPerAcc {
		return true
	}
	return false
}

func (s *authService) recordLoginFailure(ctx context.Context, scope, ip, account string) {
	if ip != "" {
		s.bumpCounter(ctx, loginIPKey(scope, ip), loginFailureWindow)
	}
	if account != "" {
		s.bumpCounter(ctx, loginAccountKey(scope, account), loginFailureWindow)
	}
}

func (s *authService) clearAccountFailures(ctx context.Context, scope, account string) {
	if s.deps.RedisClient == nil || account == "" {
		return
	}
	_ = s.deps.RedisClient.Del(ctx, loginAccountKey(scope, account))
}

// passwordResetAllowed throttles reset emails per account so the endpoint
// cannot be used to flood a victim's inbox. Callers stay silent when the
// budget is exhausted to avoid revealing whether the account exists.
func (s *authService) passwordResetAllowed(ctx context.Context, email string) bool {
	key := fmt.Sprintf("rate:pwreset:acct:%s", email)
	return s.bumpCounter(ctx, key, passwordResetWindow) <= maxPasswordResetsPerAcc
}

var (
	dummyHashOnce sync.Once
	dummyHash     []byte
)

// equalizeLoginTiming performs a bcrypt comparison against a fixed hash so
// that "unknown email" responses take as long as "wrong password" responses,
// which prevents account enumeration through response timing.
func equalizeLoginTiming(password string) {
	dummyHashOnce.Do(func() {
		dummyHash, _ = bcrypt.GenerateFromPassword([]byte("timing-equalizer-not-a-password"), 12)
	})
	_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
}
