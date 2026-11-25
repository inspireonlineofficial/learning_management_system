package audit

import "testing"

func TestNormalizeIP(t *testing.T) {
	cases := map[string]interface{}{
		"":                nil,
		"127.0.0.1:51922": "127.0.0.1",
		"203.0.113.9":     "203.0.113.9",
		"[::1]:8080":      "::1",
		"not-an-ip":       nil,
	}
	for in, want := range cases {
		if got := normalizeIP(in); got != want {
			t.Errorf("normalizeIP(%q) = %v, want %v", in, got, want)
		}
	}
}
