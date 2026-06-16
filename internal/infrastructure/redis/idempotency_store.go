package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

const idempotencyTTL = 24 * time.Hour

// IdempotencyStore implements the bookshop.IdempotencyStore interface using Redis.
// Key pattern: idempotency:{key} → response JSON, TTL 24h
type IdempotencyStore struct {
	client *Client
}

// NewIdempotencyStore creates a new IdempotencyStore.
func NewIdempotencyStore(client *Client) *IdempotencyStore {
	return &IdempotencyStore{client: client}
}

// Get returns the cached response for a key, or ("", false, nil) if not found.
func (s *IdempotencyStore) Get(ctx context.Context, key string) (string, bool, error) {
	val, err := s.client.Get(ctx, "idempotency:"+key)
	if err == redis.Nil {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return val, true, nil
}

// Set stores a response for a key with a 24h TTL.
func (s *IdempotencyStore) Set(ctx context.Context, key string, response string) error {
	return s.client.Set(ctx, "idempotency:"+key, response, idempotencyTTL)
}
