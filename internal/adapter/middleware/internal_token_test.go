package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireInternalToken(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	const secret = "s3cret-s3cret-s3cret-s3cret-s3cret"

	tests := []struct {
		name, token, method, path, header string
		want                              int
	}{
		{"disabled when empty", "", http.MethodPost, "/api/v1/auth/login", "", http.StatusOK},
		{"missing header", secret, http.MethodPost, "/api/v1/auth/login", "", http.StatusForbidden},
		{"wrong header", secret, http.MethodGet, "/api/v1/auth/me", "nope", http.StatusForbidden},
		{"prefix of secret", secret, http.MethodGet, "/api/v1/auth/me", secret[:10], http.StatusForbidden},
		{"correct header", secret, http.MethodPost, "/api/v1/auth/login", secret, http.StatusOK},
		{"health open", secret, http.MethodGet, "/health", "", http.StatusOK},
		{"root open", secret, http.MethodGet, "/", "", http.StatusOK},
		{"post to health not exempt", secret, http.MethodPost, "/health", "", http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			if tt.header != "" {
				req.Header.Set(InternalTokenHeader, tt.header)
			}
			rec := httptest.NewRecorder()
			RequireInternalToken(tt.token)(ok).ServeHTTP(rec, req)
			if rec.Code != tt.want {
				t.Fatalf("status %d, want %d", rec.Code, tt.want)
			}
		})
	}
}
