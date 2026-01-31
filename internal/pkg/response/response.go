package response

import (
	"encoding/json"
	"net/http"
)

type Wrapper struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Errors  interface{} `json:"errors,omitempty"`
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

// Error sends an error JSON response
func Error(w http.ResponseWriter, status int, message string, errors interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Wrapper{
		Success: false,
		Message: message,
		Errors:  errors,
	})
}
