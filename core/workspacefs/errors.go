package workspacefs

import "errors"

// Sentinel errors returned by the workspacefs service and repository.
var (
	// ErrInvalidInput is returned when a required field is missing or invalid.
	ErrInvalidInput = errors.New("workspace: invalid input")

	// ErrNotFound is returned when a workspace or file is not found.
	ErrNotFound = errors.New("workspace: not found")

	// ErrAlreadyExists is returned when creating a file that already exists.
	ErrAlreadyExists = errors.New("workspace: already exists")

	// ErrParentNotFound is returned when the parent folder does not exist.
	ErrParentNotFound = errors.New("workspace: parent folder not found")

	// ErrNotEmpty is returned when deleting a non-empty folder.
	ErrNotEmpty = errors.New("workspace: folder is not empty")
)
