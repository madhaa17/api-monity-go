package handler

import (
	"errors"
	"net/http"

	"monity/internal/core/port"
	"monity/internal/pkg/response"
)

// writeServiceError maps typed service errors to HTTP responses: validation errors to 400 with
// their message, ErrNotFound to 404 with notFoundMsg, and anything else to 500 with internalMsg.
func writeServiceError(w http.ResponseWriter, r *http.Request, err error, notFoundMsg, internalMsg string) {
	var validationErr *port.ValidationError
	switch {
	case errors.As(err, &validationErr):
		response.ErrorWithLog(w, r, http.StatusBadRequest, validationErr.Message, nil)
	case errors.Is(err, port.ErrNotFound):
		response.ErrorWithLog(w, r, http.StatusNotFound, notFoundMsg, nil)
	default:
		response.ErrorWithLog(w, r, http.StatusInternalServerError, internalMsg, err.Error())
	}
}
