package response

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

type Wrapper struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Errors  interface{} `json:"errors,omitempty"`
}

// normalizeErrors ensures errors is always an object for FE. String -> {"message": s}; nil -> {}.
func normalizeErrors(errors interface{}) interface{} {
	if errors == nil {
		return map[string]interface{}{}
	}
	if s, ok := errors.(string); ok {
		return map[string]interface{}{"message": s}
	}
	return errors
}

// Success sends a success JSON response
func Success(w http.ResponseWriter, status int, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Wrapper{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// Error sends an error JSON response (no logging). Errors is normalized to an object.
func Error(w http.ResponseWriter, status int, message string, errors interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Wrapper{
		Success: false,
		Message: message,
		Errors:  normalizeErrors(errors),
	})
}

// ErrorWithLog sends an error JSON response and logs: 5xx at Error, 4xx at Debug.
// Errors is normalized to an object. Pass r so request context can be included. If r is nil, no log is written.
func ErrorWithLog(w http.ResponseWriter, r *http.Request, status int, message string, errors interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Wrapper{
		Success: false,
		Message: message,
		Errors:  normalizeErrors(errors),
	})
	if r == nil {
		return
	}
	attrs := []any{"method", r.Method, "path", r.URL.Path, "status", status, "message", message}
	if status >= 500 {
		if errors != nil {
			attrs = append(attrs, "error", errors)
		}
		slog.Error("handler_error", attrs...)
	} else if status >= 400 {
		slog.Debug("client_error", attrs...)
	}
}
