package enrollments

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"path"
	"strings"
	"time"
)

// EdgeDeliveryConfig configures optional authorization-aware CDN delivery.
//
// Background: the Spaces CDN does not cache requests made with presigned URLs
// (each one is forwarded to the origin), so with a private bucket the default
// delivery path (backend-signed playlists + presigned origin segments) gets no
// edge caching. In "bunny" mode, a Bunny CDN pull zone pulls from the private
// Spaces bucket using S3 origin authentication and validates a short-lived,
// directory-scoped token at the edge before serving cached content. The token
// lives in the URL path (/bcdn_token=...), so relative HLS segment URIs inherit
// it and the static playlists in storage need no rewriting.
type EdgeDeliveryConfig struct {
	// Mode is "presigned" (default) or "bunny".
	Mode string
	// Host is the pull zone hostname, e.g. "lms-video.b-cdn.net" or a custom
	// CNAME such as "video.example.com".
	Host string
	// TokenKey is the pull zone's token authentication key.
	TokenKey string
	// TTL bounds the playback session (default 2h).
	TTL time.Duration
}

// EdgeDeliverySetter lets the composition root enable edge delivery.
type EdgeDeliverySetter interface {
	SetEdgeDelivery(cfg EdgeDeliveryConfig)
}

// SetEdgeDelivery enables an edge delivery mode.
func (s *service) SetEdgeDelivery(cfg EdgeDeliveryConfig) {
	cfg.Mode = strings.ToLower(strings.TrimSpace(cfg.Mode))
	cfg.Host = strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(cfg.Host), "https://"), "http://"), "/")
	if cfg.TTL <= 0 {
		cfg.TTL = streamURLTTL
	}
	s.edge = cfg
}

func (s *service) bunnyEnabled() bool {
	return s.edge.Mode == "bunny" && s.edge.Host != "" && s.edge.TokenKey != ""
}

// bunnySignDirectory builds a Bunny "advanced" (HS256) path-based token URL
// for objectKey, valid for every file under its directory (token_path), per
// https://bunny.net/docs/cdn/security/token-authentication/advanced:
//
//	HMAC-SHA256(key, signaturePath + expires + signingData), base64url without
//	padding, prefixed "HS256-"; signingData = sorted "token_*" params.
func bunnySignDirectory(host, key, objectKey string, expires int64) string {
	objectPath := "/" + strings.TrimPrefix(objectKey, "/")
	tokenPath := path.Dir(objectPath) + "/"
	signingData := "token_path=" + tokenPath
	mac := hmac.New(sha256.New, []byte(key))
	fmt.Fprintf(mac, "%s%d%s", tokenPath, expires, signingData)
	token := "HS256-" + strings.TrimRight(base64.URLEncoding.EncodeToString(mac.Sum(nil)), "=")
	return fmt.Sprintf("https://%s/bcdn_token=%s&expires=%d&token_path=%s%s",
		host, token, expires, url.QueryEscape(tokenPath), objectPath)
}
