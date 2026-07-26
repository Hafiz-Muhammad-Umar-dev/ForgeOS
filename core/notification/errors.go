package notification

import "errors"

var (
	// ErrInvalidEvent is returned when a bus message cannot be deserialized.
	ErrInvalidEvent = errors.New("notification: invalid event")

	// ErrSendFailed is returned when the notification port fails to send.
	ErrSendFailed = errors.New("notification: send failed")

	// ErrNotStarted is returned when the service is not started.
	ErrNotStarted = errors.New("notification: not started")
)
