package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORS(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	tests := []struct {
		name, allowed, origin, want string
	}{
		{"empty list allows nobody", "", "https://evil.example", ""},
		{"listed origin", "https://app.example.com, https://b.example.com", "https://b.example.com", "https://b.example.com"},
		{"unlisted origin", "https://app.example.com", "https://evil.example", ""},
		{"wildcard", "*", "https://any.example", "https://any.example"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/me", nil)
			req.Header.Set("Origin", tt.origin)
			rec := httptest.NewRecorder()
			CORS(tt.allowed)(ok).ServeHTTP(rec, req)
			if got := rec.Header().Get("Access-Control-Allow-Origin"); got != tt.want {
				t.Fatalf("Allow-Origin %q, want %q", got, tt.want)
			}
			if tt.want == "" && rec.Header().Get("Access-Control-Allow-Methods") != "" {
				t.Fatal("CORS headers sent to a disallowed origin")
			}
		})
	}
}
