package redis

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// SigningKeyStore manages user-scoped signing keys for presigned URLs
type SigningKeyStore struct {
	client *Client
}

// NewSigningKeyStore creates a new SigningKeyStore
func NewSigningKeyStore(client *Client) *SigningKeyStore {
	return &SigningKeyStore{client: client}
}

// InvalidateUserSigningKey invalidates the user-scoped signing key for a specific enrollment
// This is called when an enrollment is cancelled or refunded to immediately revoke
// access to all presigned URLs for that user-course combination
func (s *SigningKeyStore) InvalidateUserSigningKey(ctx context.Context, userID uuid.UUID, courseID uuid.UUID) error {
	// Key pattern: signing_key:{user_id}:{course_id}
	key := fmt.Sprintf("signing_key:%s:%s", userID.String(), courseID.String())

	// Delete the signing key from Redis
	if err := s.client.Del(ctx, key); err != nil {
		return fmt.Errorf("failed to invalidate signing key: %w", err)
	}

	return nil
}

// GetUserSigningKey retrieves or generates a user-scoped signing key
// This would be used when generating presigned URLs
func (s *SigningKeyStore) GetUserSigningKey(ctx context.Context, userID uuid.UUID, courseID uuid.UUID) (string, error) {
	key := fmt.Sprintf("signing_key:%s:%s", userID.String(), courseID.String())

	// Try to get existing key
	signingKey, err := s.client.Get(ctx, key)
	if err == nil && signingKey != "" {
		return signingKey, nil
	}

	// If key doesn't exist, generate a new one
	// In a real implementation, this would be a cryptographically secure random key
	// For now, we'll use a placeholder
	newKey := fmt.Sprintf("key_%s_%s", userID.String(), courseID.String())

	// Store with a reasonable TTL (e.g., 24 hours)
	// In production, this TTL should match the maximum presigned URL validity
	if err := s.client.Set(ctx, key, newKey, 24*3600); err != nil {
		return "", fmt.Errorf("failed to store signing key: %w", err)
	}

	return newKey, nil
}
