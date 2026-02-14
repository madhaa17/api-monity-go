package middleware

import (
	"compress/gzip"
	"net/http"
	"strings"
)

// gzipResponseWriter wraps the response writer to compress the body when the client
// sends Accept-Encoding: gzip. It delays WriteHeader until the first Write so we can
// set Content-Encoding: gzip before sending headers.
type gzipResponseWriter struct {
	http.ResponseWriter
	code        int
	wroteHeader bool
	gz          *gzip.Writer
}

func (w *gzipResponseWriter) WriteHeader(code int) {
	if w.wroteHeader {
		return
	}
	w.code = code
}

func (w *gzipResponseWriter) Write(p []byte) (int, error) {
	if !w.wroteHeader {
		w.wroteHeader = true
		w.ResponseWriter.Header().Set("Content-Encoding", "gzip")
		if w.code != 0 {
			w.ResponseWriter.WriteHeader(w.code)
		} else {
			w.ResponseWriter.WriteHeader(http.StatusOK)
		}
		w.gz = gzip.NewWriter(w.ResponseWriter)
	}
	return w.gz.Write(p)
}

// Gzip compresses the response body with gzip when the client sends Accept-Encoding: gzip.
// It wraps the response as the outermost layer so the response is compressed before
// being sent to the client.
func Gzip(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		gw := &gzipResponseWriter{ResponseWriter: w, code: http.StatusOK}
		next.ServeHTTP(gw, r)
		if gw.gz != nil {
			_ = gw.gz.Close()
		} else if !gw.wroteHeader {
			w.WriteHeader(gw.code)
		}
	})
}
