package orchestrator

import "errors"

var (
	// ErrInvalidEvent is returned when a bus message cannot be deserialized.
	ErrInvalidEvent = errors.New("orchestrator: invalid event")

	// ErrDispatchFailed is returned when the agent runtime task fails.
	ErrDispatchFailed = errors.New("orchestrator: dispatch failed")

	// ErrNotStarted is returned when the engine is not started.
	ErrNotStarted = errors.New("orchestrator: not started")
)
