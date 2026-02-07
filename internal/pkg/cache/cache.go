package cache

import (
	"context"
	"time"
)

// Cache is a key-value cache with TTL. Used for price data and other API response caching.
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
}
