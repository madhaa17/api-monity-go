package middleware

import (
	"net/http"
	"strings"
)

// SecurityHeaders adds common security headers to responses.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent MIME sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")
		// Clickjacking protection
		w.Header().Set("X-Frame-Options", "DENY")
		// XSS filter (legacy browsers)
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		// Referrer policy: don't send full URL to other origins
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		// HTTPS only in production (optional; enable if behind TLS)
		// w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		next.ServeHTTP(w, r)
	})
}

// CORS lets browser pages on the listed origins read API responses. allowedOrigins is a
// comma-separated list, e.g. "https://app.example.com,https://admin.example.com", or "*" (refused in
// production by config.Validate). Empty allows no origin: the frontend reaches the API through its
// same-origin /api/v1 rewrite and needs no CORS. CORS only limits what browsers let other sites read;
// it does not stop direct calls (curl, scripts, servers).
func CORS(allowedOrigins string) func(next http.Handler) http.Handler {
	origins := parseOrigins(allowedOrigins)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if origin := r.Header.Get("Origin"); origin != "" && allowOrigin(origins, origin) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Add("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
				w.Header().Set("Access-Control-Max-Age", "86400")
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func parseOrigins(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	if s == "*" {
		return []string{"*"}
	}
	var out []string
	for _, o := range strings.Split(s, ",") {
		o = strings.TrimSpace(o)
		if o != "" {
			out = append(out, o)
		}
	}
	return out
}

func allowOrigin(origins []string, origin string) bool {
	for _, o := range origins {
		if o == "*" || o == origin {
			return true
		}
	}
	return false
}
