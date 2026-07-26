package auth

import (
	"context"
	"sync/atomic"
)

// Compile-time check.
var _ AuthProvider = (*FakeAuthProvider)(nil)

// FakeAuthProvider is an in-memory AuthProvider for testing.
// It records all received tokens and returns configurable results.
type FakeAuthProvider struct {
	// AuthenticateFunc overrides the Authenticate behavior.
	AuthenticateFunc func(ctx context.Context, token string) (Claims, error)

	// AlwaysValid makes any token succeed, returning default claims.
	AlwaysValid bool

	// AlwaysInvalid makes any token fail with ErrInvalidToken.
	AlwaysInvalid bool

	// ClaimsValue is returned when AlwaysValid is true.
	ClaimsValue Claims

	// AuthenticateCount tracks the number of calls.
	AuthenticateCount atomic.Int64

	// ReceivedTokens records every token received.
	ReceivedTokens []string
}

// NewFakeAuthProvider creates a FakeAuthProvider that always succeeds.
func NewFakeAuthProvider() *FakeAuthProvider {
	return &FakeAuthProvider{
		AlwaysValid: true,
		ClaimsValue: Claims{
			Subject: "fake-user",
			OrgID:   "fake-org",
			Scopes:  []string{"intent:write"},
		},
	}
}

// Authenticate records the call and returns based on configuration.
func (f *FakeAuthProvider) Authenticate(_ context.Context, token string) (Claims, error) {
	f.AuthenticateCount.Add(1)
	f.ReceivedTokens = append(f.ReceivedTokens, token)

	if f.AuthenticateFunc != nil {
		return f.AuthenticateFunc(nil, token)
	}
	if f.AlwaysInvalid {
		return Claims{}, ErrInvalidToken
	}
	if f.AlwaysValid {
		return f.ClaimsValue, nil
	}
	return Claims{}, ErrInvalidToken
}
