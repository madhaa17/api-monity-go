package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"monity/internal/config"
	"monity/internal/pkg/cache"

	"github.com/golang-jwt/jwt/v5"
)

func TestRequireAuthRejectsTokensIssuedBeforePasswordChange(t *testing.T) {
	cfg := &config.Config{Jwt: config.JwtConfig{Secret: "access-secret"}}
	c := cache.NewMemoryCache()
	m := NewAuthMiddleware(cfg, c)
	handler := m.RequireAuth(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	changedAt := time.Now().Add(-time.Minute)
	if err := c.Set(context.Background(), passwordChangedKeyPrefix+"7", []byte(strconv.FormatInt(changedAt.Unix(), 10)), time.Hour); err != nil {
		t.Fatal(err)
	}

	sign := func(claims jwt.MapClaims) string {
		claims["uuid"] = "user-uuid"
		claims["exp"] = time.Now().Add(time.Hour).Unix()
		token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(cfg.Jwt.Secret))
		if err != nil {
			t.Fatal(err)
		}
		return token
	}

	tests := []struct {
		name   string
		claims jwt.MapClaims
		want   int
	}{
		{"issued before change", jwt.MapClaims{"sub": 7, "iat": changedAt.Add(-time.Second).Unix()}, http.StatusUnauthorized},
		{"no iat", jwt.MapClaims{"sub": 7}, http.StatusUnauthorized},
		{"issued at change", jwt.MapClaims{"sub": 7, "iat": changedAt.Unix()}, http.StatusOK},
		{"issued after change", jwt.MapClaims{"sub": 7, "iat": time.Now().Unix()}, http.StatusOK},
		{"other user", jwt.MapClaims{"sub": 8, "iat": changedAt.Add(-time.Hour).Unix()}, http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
			req.Header.Set("Authorization", "Bearer "+sign(tt.claims))
			rec := httptest.NewRecorder()
			handler(rec, req)
			if rec.Code != tt.want {
				t.Fatalf("status %d, want %d", rec.Code, tt.want)
			}
		})
	}
}
