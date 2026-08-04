package workspace

import "errors"

// Sentinel errors returned by WorkspacePort and SecretProxy implementations.
var (
	// ErrNotFound is returned when a workspace or secret is not found.
	ErrNotFound = errors.New("workspace: not found")

	// ErrNotReady is returned when attempting to exec in a workspace
	// that is not in Ready status.
	ErrNotReady = errors.New("workspace: not ready")

	// ErrAlreadyExists is returned when attempting to provision a
	// workspace with an ID that already exists.
	ErrAlreadyExists = errors.New("workspace: already exists")

	// ErrExecFailed is returned when a command execution fails.
	ErrExecFailed = errors.New("workspace: exec failed")

	// ErrProvisionFailed is returned when workspace provisioning fails.
	ErrProvisionFailed = errors.New("workspace: provision failed")

	// ErrRecycleFailed is returned when workspace recycling fails.
	ErrRecycleFailed = errors.New("workspace: recycle failed")

	// ErrSecretNotResolved is returned when the secret proxy cannot
	// resolve a secret reference.
	ErrSecretNotResolved = errors.New("workspace: secret not resolved")
)
