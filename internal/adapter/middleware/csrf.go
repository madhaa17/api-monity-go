package middleware

import (
	"log/slog"
	"net/http"

	"monity/internal/pkg/authcookie"
	"monity/internal/pkg/response"
)

// CSRF rejects state-changing requests that ride the auth cookies from another site. SameSite=Lax
// already keeps the cookies off cross-site POSTs; this also covers sibling subdomains (same-site)
// and browsers without SameSite.
//
// A request passes when it is safe (GET/HEAD/OPTIONS), carries an Authorization header (a cross-site
// page cannot set one without a CORS preflight), carries no auth cookie, comes from an origin listed
// explicitly in allowedOrigins, or is marked Sec-Fetch-Site: same-origin. Requests without
// Sec-Fetch-Site (non-browser clients, old browsers) fall back to SameSite.
func CSRF(allowedOrigins string) func(next http.Handler) http.Handler {
	var explicit []string
	for _, o := range parseOrigins(allowedOrigins) {
		if o != "*" {
			explicit = append(explicit, o)
		}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if csrfAllowed(r, explicit) {
				next.ServeHTTP(w, r)
				return
			}
			slog.Warn("csrf_rejected", "path", r.URL.Path, "origin", r.Header.Get("Origin"),
				"sec_fetch_site", r.Header.Get("Sec-Fetch-Site"), "ip", clientIP(r))
			response.Error(w, http.StatusForbidden, "cross-site request rejected", nil)
		})
	}
}

func csrfAllowed(r *http.Request, explicitOrigins []string) bool {
	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	}
	if r.Header.Get("Authorization") != "" || !authcookie.HasAuthCookie(r) {
		return true
	}
	if origin := r.Header.Get("Origin"); origin != "" && allowOrigin(explicitOrigins, origin) {
		return true
	}
	if site := r.Header.Get("Sec-Fetch-Site"); site != "" {
		return site == "same-origin"
	}
	return true
}
