// Package auth defines the AuthProvider port for DevOS. It follows the
// ports/adapters (hexagonal) architecture: core services use AuthProvider,
// and adapters (JWT, API key, OIDC) satisfy it without leaking credential
// handling into domain code.
//
// See ADR-003 (Provider Abstraction via Ports), SDD §02 (API Gateway).
package auth

import (
	"context"
	"errors"
)

// AuthProvider verifies credentials and returns identity claims.
// Implementations must be stateless and safe for concurrent use.
type AuthProvider interface {
	// Authenticate verifies a credential token and returns the identity
	// claims. Implementations must respect context cancellation.
	Authenticate(ctx context.Context, token string) (Claims, error)
}

// Claims carries verified identity information after authentication.
type Claims struct {
	// Subject identifies the user or service (e.g., "user-abc").
	Subject string `json:"sub"`

	// OrgID is the tenant organization identifier.
	OrgID string `json:"org_id,omitempty"`

	// Scopes are the granted permission scopes (e.g., ["intent:write"]).
	Scopes []string `json:"scopes,omitempty"`
}

// Sentinel errors returned by AuthProvider implementations.
var (
	// ErrInvalidToken is returned when the token is malformed or invalid.
	ErrInvalidToken = errors.New("auth: invalid token")

	// ErrTokenExpired is returned when the token has expired.
	ErrTokenExpired = errors.New("auth: token expired")

	// ErrForbidden is returned when the token is valid but lacks the
	// required permissions.
	ErrForbidden = errors.New("auth: forbidden")
)
