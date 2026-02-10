package middleware

import (
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	"monity/internal/config"
	"monity/internal/pkg/response"
)

type rateLimitEntry struct {
	count       int
	windowStart time.Time
}

type RateLimitMiddleware struct {
	cfg   *config.RateLimitConfig
	store map[string]*rateLimitEntry
	mu    sync.RWMutex
}

func NewRateLimitMiddleware(cfg *config.RateLimitConfig) *RateLimitMiddleware {
	if cfg.TTLSeconds <= 0 {
		cfg.TTLSeconds = 60
	}
	if cfg.Limit <= 0 {
		cfg.Limit = 100
	}
	return &RateLimitMiddleware{
		cfg:   cfg,
		store: make(map[string]*rateLimitEntry),
	}
}

func (m *RateLimitMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := clientIP(r)

		m.mu.Lock()
		entry, ok := m.store[key]
		now := time.Now()
		windowDur := time.Duration(m.cfg.TTLSeconds) * time.Second
		if !ok || now.Sub(entry.windowStart) >= windowDur {
			m.store[key] = &rateLimitEntry{count: 1, windowStart: now}
			m.mu.Unlock()
			next.ServeHTTP(w, r)
			return
		}
		entry.count++
		if entry.count > m.cfg.Limit {
			m.mu.Unlock()
			slog.Warn("rate_limit_exceeded", "ip", key, "path", r.URL.Path, "count", entry.count, "limit", m.cfg.Limit)
			w.Header().Set("Retry-After", strconv.Itoa(m.cfg.TTLSeconds))
			response.Error(w, http.StatusTooManyRequests, "rate limit exceeded", nil)
			return
		}
		m.mu.Unlock()
		next.ServeHTTP(w, r)
	})
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// First IP is the client when behind proxy
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return xff[:i]
			}
		}
		return xff
	}
	return r.RemoteAddr
}
