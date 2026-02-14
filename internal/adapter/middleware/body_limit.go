package middleware

import (
	"net/http"
)

// MaxBodyBytes is the maximum allowed request body size (1 MB).
const MaxBodyBytes = 1 << 20

// BodyLimit wraps the request body with http.MaxBytesReader for POST, PUT, PATCH methods
// so that handlers reading the body will get an error if the body exceeds maxBytes.
func BodyLimit(maxBytes int64) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body == nil {
				next.ServeHTTP(w, r)
				return
			}
			switch r.Method {
			case http.MethodPost, http.MethodPut, http.MethodPatch:
				r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			}
			next.ServeHTTP(w, r)
		})
	}
}
