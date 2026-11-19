package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"monity/internal/config"
	"monity/internal/core/port"
	"monity/internal/models"
	"monity/internal/pkg/authcookie"
)

type fakeAuthService struct {
	port.AuthService
	refreshGot              string
	logoutAccess, logoutRef string
	refreshErr              error
}

func (f *fakeAuthService) Login(context.Context, port.LoginRequest) (*port.AuthResponse, error) {
	return &port.AuthResponse{Token: "access-1", RefreshToken: "refresh-1", User: &models.User{Email: "a@b.c"}}, nil
}

func (f *fakeAuthService) Refresh(_ context.Context, rt string) (*port.AuthResponse, error) {
	f.refreshGot = rt
	if f.refreshErr != nil {
		return nil, f.refreshErr
	}
	return &port.AuthResponse{Token: "access-2", RefreshToken: "refresh-2", User: &models.User{}}, nil
}

func (f *fakeAuthService) Logout(_ context.Context, access, refresh string) error {
	f.logoutAccess, f.logoutRef = access, refresh
	return nil
}

func newTestAuthHandler(svc port.AuthService) *AuthHandler {
	cfg := &config.Config{Jwt: config.JwtConfig{RefreshExpiration: "1h"}, Cookie: config.CookieConfig{Secure: true}}
	return NewAuthHandler(svc, authcookie.NewManager(cfg))
}

func cookiesByNameAndPath(rec *httptest.ResponseRecorder) map[string]*http.Cookie {
	out := map[string]*http.Cookie{}
	for _, c := range rec.Result().Cookies() {
		out[c.Name+" "+c.Path] = c
	}
	return out
}

func TestLoginSetsHttpOnlyCookiesAndHidesTokens(t *testing.T) {
	h := newTestAuthHandler(&fakeAuthService{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"email":"a@b.c","password":"x"}`))
	rec := httptest.NewRecorder()
	h.Login(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	if body := rec.Body.String(); strings.Contains(body, "access-1") || strings.Contains(body, "refresh-1") {
		t.Fatalf("token leaked in body: %s", body)
	}
	var payload struct {
		Data struct {
			User *models.User `json:"user"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil || payload.Data.User == nil || payload.Data.User.Email != "a@b.c" {
		t.Fatalf("user missing from body: %s", rec.Body)
	}

	cookies := cookiesByNameAndPath(rec)
	access := cookies[authcookie.AccessName+" /"]
	refresh := cookies[authcookie.RefreshName+" "+authcookie.RefreshPath]
	if access == nil || access.Value != "access-1" || refresh == nil || refresh.Value != "refresh-1" {
		t.Fatalf("auth cookies not set: %v", rec.Result().Cookies())
	}
	for _, c := range []*http.Cookie{access, refresh} {
		if !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteLaxMode || c.MaxAge != 3600 {
			t.Fatalf("cookie %s has wrong flags: %+v", c.Name, c)
		}
	}
	if legacy := cookies[authcookie.RefreshName+" /"]; legacy == nil || legacy.MaxAge >= 0 {
		t.Fatalf("legacy refresh cookie on / not expired: %+v", legacy)
	}
}

func TestRefreshReadsCookieAndRotates(t *testing.T) {
	svc := &fakeAuthService{}
	h := newTestAuthHandler(svc)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: authcookie.RefreshName, Value: "refresh-1"})
	rec := httptest.NewRecorder()
	h.Refresh(rec, req)

	if rec.Code != http.StatusOK || svc.refreshGot != "refresh-1" {
		t.Fatalf("status %d, refresh got %q", rec.Code, svc.refreshGot)
	}
	cookies := cookiesByNameAndPath(rec)
	if c := cookies[authcookie.AccessName+" /"]; c == nil || c.Value != "access-2" {
		t.Fatalf("access cookie not rotated: %v", rec.Result().Cookies())
	}
	if c := cookies[authcookie.RefreshName+" "+authcookie.RefreshPath]; c == nil || c.Value != "refresh-2" {
		t.Fatalf("refresh cookie not rotated: %v", rec.Result().Cookies())
	}
}

func TestRefreshFailureClearsCookies(t *testing.T) {
	h := newTestAuthHandler(&fakeAuthService{refreshErr: errors.New("invalid or expired refresh token")})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: authcookie.RefreshName, Value: "stale"})
	rec := httptest.NewRecorder()
	h.Refresh(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d", rec.Code)
	}
	assertCleared(t, rec)
}

func TestRefreshWithoutTokenIsUnauthorized(t *testing.T) {
	h := newTestAuthHandler(&fakeAuthService{})
	rec := httptest.NewRecorder()
	h.Refresh(rec, httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestLogoutRevokesCookieTokensAndClears(t *testing.T) {
	svc := &fakeAuthService{}
	h := newTestAuthHandler(svc)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: authcookie.AccessName, Value: "access-1"})
	req.AddCookie(&http.Cookie{Name: authcookie.RefreshName, Value: "refresh-1"})
	rec := httptest.NewRecorder()
	h.Logout(rec, req)

	if rec.Code != http.StatusOK || svc.logoutAccess != "access-1" || svc.logoutRef != "refresh-1" {
		t.Fatalf("status %d, revoked %q/%q", rec.Code, svc.logoutAccess, svc.logoutRef)
	}
	assertCleared(t, rec)
}

func assertCleared(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	cookies := cookiesByNameAndPath(rec)
	for _, key := range []string{authcookie.AccessName + " /", authcookie.RefreshName + " " + authcookie.RefreshPath, authcookie.RefreshName + " /"} {
		if c := cookies[key]; c == nil || c.MaxAge >= 0 {
			t.Fatalf("cookie %q not cleared: %+v", key, c)
		}
	}
}
