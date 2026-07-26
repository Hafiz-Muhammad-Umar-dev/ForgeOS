package provider

import "errors"

// Sentinel errors returned by LLMProvider implementations.
var (
	// ErrRateLimited is returned when the provider responds with a rate
	// limit status (HTTP 429).
	ErrRateLimited = errors.New("provider: rate limited")

	// ErrContextWindowExceeded is returned when the request exceeds the
	// provider's context window (HTTP 400 with context-length error).
	ErrContextWindowExceeded = errors.New("provider: context window exceeded")

	// ErrProviderUnavailable is returned when the provider responds with a
	// server error (HTTP 5xx) or is unreachable.
	ErrProviderUnavailable = errors.New("provider: unavailable")

	// ErrAuthFailed is returned when authentication fails (HTTP 401/403).
	ErrAuthFailed = errors.New("provider: authentication failed")

	// ErrBadRequest is returned when the request is invalid (HTTP 400
	// without a context-length error).
	ErrBadRequest = errors.New("provider: bad request")

	// ErrStreamInterrupted is returned when the stream connection is lost
	// before completion.
	ErrStreamInterrupted = errors.New("provider: stream interrupted")

	// ErrDeployFailed is returned when a deployment fails.
	ErrDeployFailed = errors.New("provider: deploy failed")

	// ErrDeployNotFound is returned when a deployment ID is not found.
	ErrDeployNotFound = errors.New("provider: deploy not found")
)