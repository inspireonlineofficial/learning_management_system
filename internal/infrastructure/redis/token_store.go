package redis

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// TokenStore implements auth.TokenStore
type TokenStore struct {
	client *Client
}

// NewTokenStore creates a new TokenStore
func NewTokenStore(client *Client) *TokenStore {
	return &TokenStore{client: client}
}

// hashToken creates a SHA256 hash of the token
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// StoreRefreshToken stores a refresh token in Redis
func (s *TokenStore) StoreRefreshToken(ctx context.Context, userID uuid.UUID, token string, ttl time.Duration) error {
	tokenHash := hashToken(token)
	key := fmt.Sprintf("refresh:%s", tokenHash)

	// Store the token with user ID
	if err := s.client.Set(ctx, key, userID.String(), ttl); err != nil {
		return fmt.Errorf("failed to store refresh token: %w", err)
	}

	// Add to user's session set for bulk invalidation
	sessionKey := fmt.Sprintf("user_sessions:%s", userID.String())
	if err := s.client.SAdd(ctx, sessionKey, tokenHash); err != nil {
		return fmt.Errorf("failed to add token to user sessions: %w", err)
	}

	// Set TTL on the session set
	if err := s.client.Expire(ctx, sessionKey, ttl); err != nil {
		return fmt.Errorf("failed to set TTL on user sessions: %w", err)
	}

	return nil
}

// ValidateRefreshToken validates a refresh token and returns the user ID
func (s *TokenStore) ValidateRefreshToken(ctx context.Context, token string) (uuid.UUID, error) {
	tokenHash := hashToken(token)
	key := fmt.Sprintf("refresh:%s", tokenHash)

	userIDStr, err := s.client.Get(ctx, key)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid or expired refresh token")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid user ID in token: %w", err)
	}

	return userID, nil
}

// DeleteRefreshToken deletes a refresh token from Redis (atomic GETDEL for rotation)
func (s *TokenStore) DeleteRefreshToken(ctx context.Context, token string) error {
	tokenHash := hashToken(token)
	key := fmt.Sprintf("refresh:%s", tokenHash)

	// Use GETDEL to atomically get and delete
	userIDStr, err := s.client.GetDel(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to delete refresh token: %w", err)
	}

	// Remove from user's session set
	userID, err := uuid.Parse(userIDStr)
	if err == nil {
		sessionKey := fmt.Sprintf("user_sessions:%s", userID.String())
		_ = s.client.SRem(ctx, sessionKey, tokenHash) // Best effort
	}

	return nil
}

// DeleteAllRefreshTokens deletes all refresh tokens for a user
func (s *TokenStore) DeleteAllRefreshTokens(ctx context.Context, userID uuid.UUID) error {
	sessionKey := fmt.Sprintf("user_sessions:%s", userID.String())

	// Get all token hashes for this user
	tokenHashes, err := s.client.SMembers(ctx, sessionKey)
	if err != nil {
		return fmt.Errorf("failed to get user sessions: %w", err)
	}

	// Delete each token
	for _, tokenHash := range tokenHashes {
		key := fmt.Sprintf("refresh:%s", tokenHash)
		_ = s.client.Del(ctx, key) // Best effort
	}

	// Delete the session set
	if err := s.client.Del(ctx, sessionKey); err != nil {
		return fmt.Errorf("failed to delete user sessions: %w", err)
	}

	return nil
}
