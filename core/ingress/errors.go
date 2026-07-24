package ingress

import "errors"

// Sentinel errors returned by the ingress adapter.
var (
	// ErrInvalidRequest is returned when the request payload fails
	// validation (e.g., empty text, malformed JSON).
	ErrInvalidRequest = errors.New("ingress: invalid request")
)
