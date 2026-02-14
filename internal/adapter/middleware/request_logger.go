package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"time"
)

// CtxKeyRequestID is the context key for the request ID (set by RequestLogger).
const CtxKeyRequestID CtxKey = "requestID"

// responseWriter wraps http.ResponseWriter to capture status code and bytes written.
type responseWriter struct {
	http.ResponseWriter
	status  int
	written int64
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.written += int64(n)
	return n, err
}

// newRequestID returns a random hex string for request tracing (std lib only).
func newRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(b)
}

// clientIPForLog returns the client IP from X-Forwarded-For or RemoteAddr (for logging).
func clientIPForLog(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return xff[:i]
			}
		}
		return xff
	}
	return r.RemoteAddr
}

// RequestLogger logs every request (except /health) with method, path, status, latency, ip, user_agent, user_id, request_id.
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}

		requestID := newRequestID()
		ctx := context.WithValue(r.Context(), CtxKeyRequestID, requestID)
		r = r.WithContext(ctx)

		start := time.Now()
		wrapped := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(wrapped, r)

		latency := time.Since(start).Milliseconds()
		ip := clientIPForLog(r)
		userAgent := r.Header.Get("User-Agent")
		userID := int64(0)
		if id, ok := r.Context().Value(CtxKeyUserID).(int64); ok {
			userID = id
		}

		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapped.status,
			"latency_ms", latency,
			"ip", ip,
			"user_agent", userAgent,
			"user_id", userID,
			"request_id", requestID,
		)
	})
}
