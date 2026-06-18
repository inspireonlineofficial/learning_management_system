package jwt

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims represents the JWT claims
type Claims struct {
	UserID string `json:"sub"`
	Role   string `json:"role"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// keyEntry holds a single RSA key pair with its kid.
type keyEntry struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	kid        string
}

// JWTService handles JWT operations with support for key rotation.
// Multiple public keys can be active simultaneously so that tokens signed
// with an old key remain valid until they expire. Requirements: 28.7
type JWTService struct {
	// activeKey is the key used to sign new tokens.
	activeKey keyEntry

	// rotatedKeys holds previous keys that are still valid for verification.
	// Key: kid → keyEntry
	rotatedKeys map[string]keyEntry

	issuer string

	// JWKS cache
	jwksCache    *JWKSResponse
	jwksCacheMu  sync.RWMutex
	jwksCachedAt time.Time
	jwksCacheTTL time.Duration
}

// JWKSResponse represents the JWKS endpoint response
type JWKSResponse struct {
	Keys []JWK `json:"keys"`
}

// JWK represents a JSON Web Key
type JWK struct {
	Kty string `json:"kty"` // Key type (RSA)
	Use string `json:"use"` // Public key use (sig)
	Kid string `json:"kid"` // Key ID
	Alg string `json:"alg"` // Algorithm (RS256)
	N   string `json:"n"`   // Modulus
	E   string `json:"e"`   // Exponent
}

// NewJWTService creates a new JWT service
func NewJWTService(privateKeyPath, publicKeyPath, issuer string) (*JWTService, error) {
	privateKeyPath = strings.TrimSpace(privateKeyPath)
	privateKeyPath = strings.Trim(privateKeyPath, "\"'")
	privateKeyPath = strings.ReplaceAll(privateKeyPath, "\\n", "\n")

	publicKeyPath = strings.TrimSpace(publicKeyPath)
	publicKeyPath = strings.Trim(publicKeyPath, "\"'")
	publicKeyPath = strings.ReplaceAll(publicKeyPath, "\\n", "\n")

	// Load private key
	var privateKeyData []byte
	if len(privateKeyPath) > 0 && (privateKeyPath[0] == '-' || len(privateKeyPath) > 100) {
		privateKeyData = cleanPEM([]byte(privateKeyPath))
	} else {
		var err error
		privateKeyData, err = os.ReadFile(privateKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read private key: %w", err)
		}
		privateKeyData = cleanPEM(privateKeyData)
	}

	privateKeyBlock, _ := pem.Decode(privateKeyData)
	if privateKeyBlock == nil {
		prefix := ""
		if len(privateKeyData) > 30 {
			prefix = string(privateKeyData[:30])
		} else {
			prefix = string(privateKeyData)
		}
		suffix := ""
		if len(privateKeyData) > 30 {
			suffix = string(privateKeyData[len(privateKeyData)-30:])
		}
		return nil, fmt.Errorf("failed to decode private key PEM (len: %d, hex: %x, prefix: %q, suffix: %q)", len(privateKeyData), privateKeyData, prefix, suffix)
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(privateKeyBlock.Bytes)
	if err != nil {
		// Try PKCS8 format
		key, err2 := x509.ParsePKCS8PrivateKey(privateKeyBlock.Bytes)
		if err2 != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		var ok bool
		privateKey, ok = key.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("private key is not RSA")
		}
	}

	// Load public key. If loading or parsing fails, fallback to deriving it from the private key.
	var publicKey *rsa.PublicKey
	var pubErr error

	if len(publicKeyPath) > 0 {
		var publicKeyData []byte
		if publicKeyPath[0] == '-' || len(publicKeyPath) > 100 {
			publicKeyData = cleanPEM([]byte(publicKeyPath))
		} else {
			var err error
			publicKeyData, err = os.ReadFile(publicKeyPath)
			if err != nil {
				pubErr = fmt.Errorf("failed to read public key file: %w", err)
			} else {
				publicKeyData = cleanPEM(publicKeyData)
			}
		}

		if pubErr == nil {
			publicKeyBlock, _ := pem.Decode(publicKeyData)
			if publicKeyBlock != nil {
				publicKeyInterface, err := x509.ParsePKIXPublicKey(publicKeyBlock.Bytes)
				if err == nil {
					var ok bool
					publicKey, ok = publicKeyInterface.(*rsa.PublicKey)
					if !ok {
						pubErr = fmt.Errorf("public key is not RSA")
					}
				} else {
					pubErr = fmt.Errorf("failed to parse public key: %w", err)
				}
			} else {
				pubErr = fmt.Errorf("failed to decode public key PEM")
			}
		}
	} else {
		pubErr = fmt.Errorf("public key path is empty")
	}

	// Fallback to deriving public key from private key
	if pubErr != nil || publicKey == nil {
		publicKey = &privateKey.PublicKey
	}

	// Generate a key ID for the active key
	keyID := uuid.New().String()[:8]

	return &JWTService{
		activeKey: keyEntry{
			privateKey: privateKey,
			publicKey:  publicKey,
			kid:        keyID,
		},
		rotatedKeys:  make(map[string]keyEntry),
		issuer:       issuer,
		jwksCacheTTL: 1 * time.Hour,
	}, nil
}

// AddRotatedKey registers an additional public key for verification only.
// This allows tokens signed with an old key to remain valid until they expire.
// Requirements: 28.7
func (s *JWTService) AddRotatedKey(kid string, publicKey *rsa.PublicKey) {
	s.jwksCacheMu.Lock()
	defer s.jwksCacheMu.Unlock()
	s.rotatedKeys[kid] = keyEntry{kid: kid, publicKey: publicKey}
	// Invalidate JWKS cache so the new key appears immediately
	s.jwksCache = nil
}

// IssueToken issues a new JWT access token signed with the active key.
func (s *JWTService) IssueToken(userID uuid.UUID, role, email string) (string, error) {
	now := time.Now().UTC()
	claims := Claims{
		UserID: userID.String(),
		Role:   role,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = s.activeKey.kid

	tokenString, err := token.SignedString(s.activeKey.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// VerifyToken verifies a JWT token and returns the claims.
// Supports multiple active keys via the kid header. Requirements: 28.7
func (s *JWTService) VerifyToken(tokenString string) (userID string, role string, email string, err error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Select public key by kid header
		kid, _ := token.Header["kid"].(string)
		if kid == s.activeKey.kid {
			return s.activeKey.publicKey, nil
		}
		if entry, ok := s.rotatedKeys[kid]; ok {
			return entry.publicKey, nil
		}
		// Fallback: use active key (for tokens issued before kid was set)
		return s.activeKey.publicKey, nil
	})

	if err != nil {
		return "", "", "", fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return "", "", "", fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return "", "", "", fmt.Errorf("invalid claims")
	}

	// Verify issuer
	if claims.Issuer != s.issuer {
		return "", "", "", fmt.Errorf("invalid issuer")
	}

	return claims.UserID, claims.Role, claims.Email, nil
}

// GetJWKS returns the JWKS response including all active keys (cached for 1 hour).
// Supports key rotation by including both the active key and any rotated keys.
// Requirements: 28.7
func (s *JWTService) GetJWKS() (*JWKSResponse, error) {
	s.jwksCacheMu.RLock()
	if s.jwksCache != nil && time.Since(s.jwksCachedAt) < s.jwksCacheTTL {
		defer s.jwksCacheMu.RUnlock()
		return s.jwksCache, nil
	}
	s.jwksCacheMu.RUnlock()

	s.jwksCacheMu.Lock()
	defer s.jwksCacheMu.Unlock()

	// Double-check after acquiring write lock
	if s.jwksCache != nil && time.Since(s.jwksCachedAt) < s.jwksCacheTTL {
		return s.jwksCache, nil
	}

	keys := []JWK{publicKeyToJWK(s.activeKey.publicKey, s.activeKey.kid)}

	// Include rotated keys so tokens signed with old keys remain verifiable
	for _, entry := range s.rotatedKeys {
		keys = append(keys, publicKeyToJWK(entry.publicKey, entry.kid))
	}

	jwks := &JWKSResponse{Keys: keys}
	s.jwksCache = jwks
	s.jwksCachedAt = time.Now()
	return jwks, nil
}

// publicKeyToJWK converts an RSA public key to JWK format.
func publicKeyToJWK(pub *rsa.PublicKey, kid string) JWK {
	nBytes := pub.N.Bytes()
	eBytes := make([]byte, 4)
	eBytes[0] = byte(pub.E >> 24)
	eBytes[1] = byte(pub.E >> 16)
	eBytes[2] = byte(pub.E >> 8)
	eBytes[3] = byte(pub.E)
	for len(eBytes) > 1 && eBytes[0] == 0 {
		eBytes = eBytes[1:]
	}
	return JWK{
		Kty: "RSA",
		Use: "sig",
		Kid: kid,
		Alg: "RS256",
		N:   base64URLEncode(nBytes),
		E:   base64URLEncode(eBytes),
	}
}

// base64URLEncode encodes bytes to base64 URL encoding without padding
func base64URLEncode(data []byte) string {
	encoded := make([]byte, (len(data)*8+5)/6)
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"

	var bits uint32
	var bitsLen uint
	var pos int

	for _, b := range data {
		bits = (bits << 8) | uint32(b)
		bitsLen += 8

		for bitsLen >= 6 {
			bitsLen -= 6
			encoded[pos] = alphabet[(bits>>bitsLen)&0x3F]
			pos++
		}
	}

	if bitsLen > 0 {
		bits <<= (6 - bitsLen)
		encoded[pos] = alphabet[bits&0x3F]
		pos++
	}

	return string(encoded[:pos])
}

// ServeJWKS returns the JWKS as JSON bytes
func (s *JWTService) ServeJWKS() ([]byte, error) {
	jwks, err := s.GetJWKS()
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(jwks)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JWKS: %w", err)
	}

	return data, nil
}

// cleanPEM sanitizes and normalizes PEM blocks (handling newlines, escaping, quotes, and outer text)
func cleanPEM(data []byte) []byte {
	s := string(data)
	s = strings.TrimSpace(s)

	// Remove outer escaped quotes first, then regular quotes
	s = strings.TrimPrefix(s, "\\\"")
	s = strings.TrimSuffix(s, "\\\"")
	s = strings.TrimPrefix(s, "\\'")
	s = strings.TrimSuffix(s, "\\'")
	s = strings.Trim(s, "\"'")
	s = strings.TrimSpace(s)

	s = strings.ReplaceAll(s, "\\n", "\n")
	s = strings.ReplaceAll(s, "\\r", "\r")
	s = strings.ReplaceAll(s, "\\t", "\t")
	s = strings.ReplaceAll(s, "\\", "")

	// Locate -----BEGIN
	beginIdx := strings.Index(s, "-----BEGIN")
	if beginIdx != -1 {
		s = s[beginIdx:]
	}

	// Locate -----END and trim everything after its closing dashes
	endIdx := strings.Index(s, "-----END")
	if endIdx != -1 {
		afterEnd := s[endIdx+8:]
		closingDashes := strings.Index(afterEnd, "-----")
		if closingDashes != -1 {
			s = s[:endIdx+8+closingDashes+5]
		}
	}

	return []byte(strings.TrimSpace(s) + "\n")
}
