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

// CORS returns a handler that sets CORS headers based on allowed origins.
// allowedOrigins can be "*" or comma-separated list, e.g. "https://app.example.com,https://admin.example.com"
func CORS(allowedOrigins string) func(next http.Handler) http.Handler {
	origins := parseOrigins(allowedOrigins)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" {
				if allowOrigin(origins, origin) {
					w.Header().Set("Access-Control-Allow-Origin", origin)
				}
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Max-Age", "86400")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func parseOrigins(s string) []string {
	if s == "" || s == "*" {
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
