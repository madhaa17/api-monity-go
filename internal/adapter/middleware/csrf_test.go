package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"monity/internal/pkg/authcookie"
)

func TestCSRF(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	handler := CSRF("https://app.example.com")(ok)

	tests := []struct {
		name    string
		method  string
		cookie  bool
		headers map[string]string
		want    int
	}{
		{"safe method", http.MethodGet, true, map[string]string{"Sec-Fetch-Site": "cross-site"}, http.StatusOK},
		{"no auth cookie", http.MethodPost, false, map[string]string{"Sec-Fetch-Site": "cross-site"}, http.StatusOK},
		{"bearer header", http.MethodPost, true, map[string]string{"Authorization": "Bearer x", "Sec-Fetch-Site": "cross-site"}, http.StatusOK},
		{"same-origin", http.MethodPost, true, map[string]string{"Sec-Fetch-Site": "same-origin"}, http.StatusOK},
		{"cross-site", http.MethodPost, true, map[string]string{"Sec-Fetch-Site": "cross-site", "Origin": "https://evil.example"}, http.StatusForbidden},
		{"sibling subdomain", http.MethodDelete, true, map[string]string{"Sec-Fetch-Site": "same-site", "Origin": "https://evil.example.com"}, http.StatusForbidden},
		{"listed origin", http.MethodPut, true, map[string]string{"Sec-Fetch-Site": "same-site", "Origin": "https://app.example.com"}, http.StatusOK},
		{"non-browser client", http.MethodPost, true, nil, http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/v1/expenses", nil)
			if tt.cookie {
				req.AddCookie(&http.Cookie{Name: authcookie.AccessName, Value: "token"})
			}
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != tt.want {
				t.Fatalf("status %d, want %d", rec.Code, tt.want)
			}
		})
	}
}

func TestCSRFWildcardOriginIsNotTrusted(t *testing.T) {
	handler := CSRF("*")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/expenses", nil)
	req.AddCookie(&http.Cookie{Name: authcookie.AccessName, Value: "token"})
	req.Header.Set("Origin", "https://evil.example")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status %d, want 403", rec.Code)
	}
}
