package agents

import "errors"

var (
	// ErrNotFound is returned when an agent is not found.
	ErrNotFound = errors.New("agents: not found")

	// ErrInvalidInput is returned when input validation fails.
	ErrInvalidInput = errors.New("agents: invalid input")
)
