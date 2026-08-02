package intents

import "errors"

// Sentinel errors returned by the intents service and repository.
var (
	// ErrInvalidInput is returned when a required field is missing or invalid.
	ErrInvalidInput = errors.New("intents: invalid input")

	// ErrNotFound is returned when an intent or task is not found.
	ErrNotFound = errors.New("intents: not found")
)
