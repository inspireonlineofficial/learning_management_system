package rustfs

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// FrontendCORSRuleID names the one CORS rule the LMS manages on a bucket.
const FrontendCORSRuleID = "lms-frontend"

// FrontendCORSRule lets the given frontend origins read objects through
// signed URLs with range requests (PDF.js book reader, HLS player).
func FrontendCORSRule(origins []string) CORSRule {
	return CORSRule{
		ID:             FrontendCORSRuleID,
		AllowedOrigins: origins,
		AllowedMethods: []string{"GET", "HEAD"},
		AllowedHeaders: []string{"Range"},
		ExposeHeaders:  []string{"Accept-Ranges", "Content-Range", "Content-Length", "ETag"},
		MaxAgeSeconds:  3600,
	}
}

// ValidateCORSOrigins accepts exact https origins (http only for localhost)
// and rejects wildcards, paths and anything else.
func ValidateCORSOrigins(origins []string) ([]string, error) {
	out := []string{}
	for _, o := range origins {
		o = strings.TrimRight(strings.TrimSpace(o), "/")
		if o == "" {
			continue
		}
		u, err := url.Parse(o)
		local := u != nil && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1")
		if err != nil || u.Host == "" || u.Path != "" || u.RawQuery != "" || strings.Contains(o, "*") ||
			!(u.Scheme == "https" || (u.Scheme == "http" && local)) {
			return nil, fmt.Errorf("refusing CORS origin %q: use an exact https origin (http only for localhost)", o)
		}
		out = append(out, o)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no CORS origins given")
	}
	return out, nil
}

// EnsureFrontendCORS adds or updates the LMS rule on a bucket, keeping every
// other rule. It writes only when something differs and reports whether it
// changed the bucket.
func (c *Client) EnsureFrontendCORS(ctx context.Context, bucket string, origins []string) (bool, error) {
	current, err := c.GetBucketCORS(ctx, bucket)
	if err != nil {
		return false, fmt.Errorf("read CORS on %s: %w", bucket, err)
	}
	want := FrontendCORSRule(origins)
	next := make([]CORSRule, 0, len(current)+1)
	for _, r := range current {
		if r.ID == FrontendCORSRuleID {
			if sameRule(r, want) {
				return false, nil
			}
			continue
		}
		next = append(next, r)
	}
	next = append(next, want)
	if err := c.PutBucketCORS(ctx, bucket, next); err != nil {
		return false, fmt.Errorf("write CORS on %s: %w", bucket, err)
	}
	return true, nil
}

func sameRule(a, b CORSRule) bool {
	eq := func(x, y []string) bool { return strings.Join(x, ",") == strings.Join(y, ",") }
	return eq(a.AllowedOrigins, b.AllowedOrigins) && eq(a.AllowedMethods, b.AllowedMethods) &&
		eq(a.AllowedHeaders, b.AllowedHeaders) && eq(a.ExposeHeaders, b.ExposeHeaders) && a.MaxAgeSeconds == b.MaxAgeSeconds
}
