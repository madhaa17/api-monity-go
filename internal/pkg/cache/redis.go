package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisCache uses Redis for distributed caching (shared across instances).
type RedisCache struct {
	client *redis.Client
}

func NewRedisCache(client *redis.Client) *RedisCache {
	return &RedisCache{client: client}
}

func (c *RedisCache) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, ErrMiss
	}
	if err != nil {
		return nil, err
	}
	return val, nil
}

func (c *RedisCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return c.client.Set(ctx, key, value, ttl).Err()
}
