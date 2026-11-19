package middleware

import (
	"crypto/subtle"
	"log/slog"
	"net/http"

	"monity/internal/pkg/response"
)

// InternalTokenHeader carries the secret shared by the frontend server and this API.
const InternalTokenHeader = "X-Internal-Token"

// RequireInternalToken refuses every request that lacks the shared secret, so the API answers only
// the frontend server (its /api/v1 proxy and Server Components), never direct callers such as curl
// or scripts. The health endpoints stay open for uptime checks and container health checks.
// An empty token disables the check (development); config.Validate requires one in production.
func RequireInternalToken(token string) func(next http.Handler) http.Handler {
	want := []byte(token)
	return func(next http.Handler) http.Handler {
		if token == "" {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isHealthCheck(r) || subtle.ConstantTimeCompare([]byte(r.Header.Get(InternalTokenHeader)), want) == 1 {
				next.ServeHTTP(w, r)
				return
			}
			slog.Warn("internal_token_rejected", "method", r.Method, "path", r.URL.Path, "ip", clientIP(r))
			response.Error(w, http.StatusForbidden, "forbidden", nil)
		})
	}
}

func isHealthCheck(r *http.Request) bool {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}
	return r.URL.Path == "/" || r.URL.Path == "/health"
}
