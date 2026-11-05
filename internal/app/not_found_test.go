package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestJSONNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{"status":"ok"}`)) })
	mux.HandleFunc("POST /api/v1/auth/login", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
	h := jsonNotFound(mux)

	tests := []struct {
		method, path string
		wantStatus   int
		wantBody     string
		wantAllow    string
	}{
		{"GET", "/", http.StatusOK, `"status":"ok"`, ""},
		{"POST", "/api/v1/auth/login", http.StatusNoContent, "", ""},
		{"GET", "/does-not-exist", http.StatusNotFound, `"message":"not found"`, ""},
		{"GET", "/api/v1/auth/login", http.StatusMethodNotAllowed, `"message":"method not allowed"`, "POST"},
	}
	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(tt.method, tt.path, nil))
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			body := rec.Body.String()
			if !strings.Contains(body, tt.wantBody) {
				t.Fatalf("body = %q, want it to contain %q", body, tt.wantBody)
			}
			if strings.Contains(body, "page not found") || strings.Contains(strings.ToLower(body), "method not allowed\n") {
				t.Fatalf("plain-text net/http body leaked: %q", body)
			}
			if tt.wantStatus >= 400 && !strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json") {
				t.Fatalf("Content-Type = %q, want application/json", rec.Header().Get("Content-Type"))
			}
			if tt.wantAllow != "" && !strings.Contains(rec.Header().Get("Allow"), tt.wantAllow) {
				t.Fatalf("Allow = %q, want it to contain %q", rec.Header().Get("Allow"), tt.wantAllow)
			}
		})
	}
}
