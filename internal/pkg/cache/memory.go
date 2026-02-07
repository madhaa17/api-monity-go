package cache

import (
	"context"
	"sync"
	"time"
)

type memoryEntry struct {
	value     []byte
	expiresAt time.Time
}

// MemoryCache is an in-memory cache implementation (single instance).
type MemoryCache struct {
	mu    sync.RWMutex
	store map[string]*memoryEntry
}

func NewMemoryCache() *MemoryCache {
	return &MemoryCache{store: make(map[string]*memoryEntry)}
}

func (c *MemoryCache) Get(ctx context.Context, key string) ([]byte, error) {
	c.mu.RLock()
	entry, ok := c.store[key]
	c.mu.RUnlock()
	if !ok || entry == nil || time.Now().After(entry.expiresAt) {
		return nil, ErrMiss
	}
	return entry.value, nil
}

func (c *MemoryCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	expiresAt := time.Now().Add(ttl)
	if ttl <= 0 {
		expiresAt = time.Now().Add(24 * time.Hour)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store[key] = &memoryEntry{value: value, expiresAt: expiresAt}
	return nil
}
