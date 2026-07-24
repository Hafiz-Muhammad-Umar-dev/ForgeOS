package registry

import "errors"

// Sentinel errors returned by Registry implementations.
var (
	// ErrNotFound is returned when a service is not found by ID.
	ErrNotFound = errors.New("registry: not found")

	// ErrAlreadyRegistered is returned when a service with the same ID
	// is already registered and the implementation requires unique IDs.
	ErrAlreadyRegistered = errors.New("registry: already registered")

	// ErrInvalidKind is returned when a typed method is called with a
	// ServiceInfo whose Kind does not match the expected kind.
	ErrInvalidKind = errors.New("registry: invalid kind")
)
