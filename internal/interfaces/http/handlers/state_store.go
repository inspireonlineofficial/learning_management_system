package handlers

import (
	"context"
	"strings"
	"time"
)

type stateStore interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Del(ctx context.Context, keys ...string) error
}

func stateStoreMiss(err error) bool {
	return err != nil && strings.Contains(err.Error(), "redis: nil")
}
