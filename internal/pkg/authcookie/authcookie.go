// Package authcookie sets, clears and reads the HttpOnly cookies that carry the auth tokens, so
// browser code never sees a token.
package authcookie

import (
	"net/http"
	"time"

	"monity/internal/config"
)

const (
	// AccessName holds the access token. Sent on every path so the frontend server (proxy.ts,
	// Server Components) can read it and forward it as a Bearer token.
	AccessName = "MoneyTor_token"
	// RefreshName holds the refresh token. Scoped to RefreshPath, so the browser only sends it to
	// the auth endpoints that need it.
	RefreshName = "MoneyTor_refresh_token"
	RefreshPath = "/api/v1/auth"
)

// Manager writes the auth cookies with the configured flags.
type Manager struct {
	secure bool
	domain string
	// maxAge is the refresh token lifetime. The access cookie lives as long, although the JWT in it
	// expires sooner: its presence tells the frontend a session exists, and a 401 then triggers a refresh.
	maxAge time.Duration
}

func NewManager(cfg *config.Config) *Manager {
	maxAge, err := time.ParseDuration(cfg.Jwt.RefreshExpiration)
	if err != nil || maxAge <= 0 {
		maxAge = 168 * time.Hour // same fallback as the auth service
	}
	return &Manager{secure: cfg.Cookie.Secure, domain: cfg.Cookie.Domain, maxAge: maxAge}
}

// Set writes both auth cookies.
func (m *Manager) Set(w http.ResponseWriter, accessToken, refreshToken string) {
	seconds := int(m.maxAge / time.Second)
	http.SetCookie(w, m.cookie(AccessName, accessToken, "/", seconds))
	http.SetCookie(w, m.cookie(RefreshName, refreshToken, RefreshPath, seconds))
	// The frontend used to store the refresh token on path "/". Drop that copy so a stale token
	// is not sent alongside the new one.
	http.SetCookie(w, m.cookie(RefreshName, "", "/", -1))
}

// Clear expires both auth cookies, including the legacy refresh cookie on path "/".
func (m *Manager) Clear(w http.ResponseWriter) {
	http.SetCookie(w, m.cookie(AccessName, "", "/", -1))
	http.SetCookie(w, m.cookie(RefreshName, "", RefreshPath, -1))
	http.SetCookie(w, m.cookie(RefreshName, "", "/", -1))
}

func (m *Manager) cookie(name, value, path string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		Domain:   m.domain,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   m.secure,
		SameSite: http.SameSiteLaxMode,
	}
}

// AccessToken returns the access token cookie value, or "".
func AccessToken(r *http.Request) string { return value(r, AccessName) }

// RefreshToken returns the refresh token cookie value, or "". When both the scoped cookie and a
// legacy path "/" copy are sent, browsers list the longer path first, so the current one wins.
func RefreshToken(r *http.Request) string { return value(r, RefreshName) }

// HasAuthCookie reports whether the request carries either auth cookie.
func HasAuthCookie(r *http.Request) bool {
	return AccessToken(r) != "" || RefreshToken(r) != ""
}

func value(r *http.Request, name string) string {
	c, err := r.Cookie(name)
	if err != nil || c.Value == "undefined" {
		return ""
	}
	return c.Value
}
