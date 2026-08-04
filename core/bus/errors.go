package bus

import "errors"

// Sentinel errors returned by BusPort implementations.
var (
	// ErrNotConnected is returned when an operation is attempted on a bus
	// that has not been connected or has already been closed.
	ErrNotConnected = errors.New("bus: not connected")

	// ErrClosed is returned when an operation is attempted on a bus that
	// has been permanently closed.
	ErrClosed = errors.New("bus: closed")

	// ErrSubscribeFailed is returned when subscription setup fails.
	ErrSubscribeFailed = errors.New("bus: subscribe failed")

	// ErrPublishFailed is returned when message publication fails.
	ErrPublishFailed = errors.New("bus: publish failed")
)
