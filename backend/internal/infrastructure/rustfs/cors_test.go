package rustfs

import "testing"

func TestValidateCORSOrigins(t *testing.T) {
	good, err := ValidateCORSOrigins([]string{"https://app.example.com/", " http://localhost:3000"})
	if err != nil || len(good) != 2 || good[0] != "https://app.example.com" {
		t.Fatalf("valid origins rejected: %v %v", good, err)
	}
	for _, bad := range []string{"*", "https://*.example.com", "http://app.example.com", "https://app.example.com/path", "ftp://x", "example.com"} {
		if _, err := ValidateCORSOrigins([]string{bad}); err == nil {
			t.Errorf("origin %q should be rejected", bad)
		}
	}
	if _, err := ValidateCORSOrigins([]string{" ", ""}); err == nil {
		t.Error("empty origin list should be rejected")
	}
}

func TestSameRule(t *testing.T) {
	a := FrontendCORSRule([]string{"https://a.test"})
	if !sameRule(a, FrontendCORSRule([]string{"https://a.test"})) || sameRule(a, FrontendCORSRule([]string{"https://b.test"})) {
		t.Fatal("rule comparison wrong")
	}
}
