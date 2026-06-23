package redis

import (
	"context"
	"time"
)

// AnalyticsCache implements application/analytics.Cache using Redis.
type AnalyticsCache struct {
	client *Client
}

// NewAnalyticsCache creates a new AnalyticsCache.
func NewAnalyticsCache(client *Client) *AnalyticsCache {
	return &AnalyticsCache{client: client}
}

// Get retrieves a cached analytics response by key.
func (c *AnalyticsCache) Get(ctx context.Context, key string) (string, error) {
	return c.client.Get(ctx, key)
}

// Set stores a cached analytics response with the given TTL.
func (c *AnalyticsCache) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	return c.client.Set(ctx, key, value, ttl)
}

// Delete removes a cached analytics response by key. Missing keys are not an
// error — cache invalidation is best-effort, and the next read will simply
// repopulate from the database.
func (c *AnalyticsCache) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, key)
}
