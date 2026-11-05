package app

import (
	"net/http"

	"monity/internal/pkg/response"
)

// jsonNotFound replaces net/http's plain-text "404 page not found" and "405 method not allowed"
// replies, which identify the server as Go, with the API's usual JSON error body.
func jsonNotFound(mux *http.ServeMux) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, pattern := mux.Handler(r); pattern != "" {
			mux.ServeHTTP(w, r)
			return
		}
		// No route matched. Run the mux's own fallback only to learn its status (404 or 405) and
		// headers such as Allow, discarding its plain-text body.
		probe := &statusProbe{header: w.Header()}
		mux.ServeHTTP(probe, r)
		switch probe.status {
		case http.StatusMethodNotAllowed:
			response.Error(w, http.StatusMethodNotAllowed, "method not allowed", nil)
		default:
			response.Error(w, http.StatusNotFound, "not found", nil)
		}
	})
}

// statusProbe records the status code and shares the real response headers, dropping the body.
type statusProbe struct {
	header http.Header
	status int
}

func (p *statusProbe) Header() http.Header { return p.header }
func (p *statusProbe) WriteHeader(code int) {
	if p.status == 0 {
		p.status = code
	}
}
func (p *statusProbe) Write(b []byte) (int, error) {
	if p.status == 0 {
		p.status = http.StatusOK
	}
	return len(b), nil
}
