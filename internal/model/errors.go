package model

import "errors"

var (
	// ErrNotFound indicates the requested resource does not exist.
	ErrNotFound = errors.New("not found")

	// ErrValidation indicates invalid input from the caller.
	ErrValidation = errors.New("validation error")

	// ErrConflict indicates a uniqueness constraint violation.
	ErrConflict = errors.New("conflict")
)

// ValidationError carries a field-level validation message.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

func (e *ValidationError) Unwrap() error {
	return ErrValidation
}
