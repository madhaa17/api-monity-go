package database

import (
	"context"
	"fmt"

	"monity/internal/config"

	"github.com/redis/go-redis/v9"
)

// NewRedis creates a Redis client from config. Call Close() when shutting down.
func NewRedis(ctx context.Context, cfg *config.RedisConfig) (*redis.Client, error) {
	if !cfg.Enabled() {
		return nil, nil
	}

	opts := &redis.Options{
		Addr:     cfg.Addr(),
		Username: cfg.Username,
		Password: cfg.Password,
		DB:       cfg.DB,
	}

	client := redis.NewClient(opts)
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	return client, nil
}
