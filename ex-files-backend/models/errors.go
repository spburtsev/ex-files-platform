package models

import "errors"

var (
	ErrNotFound     = errors.New("not found")
	ErrForbidden    = errors.New("forbidden")
	ErrConflict     = errors.New("conflict")
	ErrInvalidInput = errors.New("invalid input")
)

// TransitionError is returned when a document status transition is not allowed.
type TransitionError struct {
	From DocumentStatus
	To   DocumentStatus
}

func (e *TransitionError) Error() string {
	return "invalid transition from " + string(e.From) + " to " + string(e.To)
}
