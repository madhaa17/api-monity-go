package port

import "errors"

// ErrNotFound is returned when a requested resource does not exist or is not owned by the user.
var ErrNotFound = errors.New("not found")

// ErrInsufficientBalance is returned by repositories when a balance change would make it negative.
var ErrInsufficientBalance = errors.New("insufficient balance")

// ErrPriceUnavailable is returned when no current market price can be had in the needed currency
// (the price source is unreachable or the exchange rate is missing).
var ErrPriceUnavailable = errors.New("price unavailable")

// ErrConflict is returned by repositories when a write violates a uniqueness constraint.
var ErrConflict = errors.New("conflict")

// ValidationError is a client-facing error: its Message is safe to return to the caller as-is.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// NewValidationError returns a *ValidationError with the given client-facing message.
func NewValidationError(message string) error {
	return &ValidationError{Message: message}
}
