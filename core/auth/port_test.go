package auth

import (
	"context"
	"testing"
)

func TestClaimsDefaults(t *testing.T) {
	c := Claims{Subject: "user-1", OrgID: "org-1"}
	if c.Subject != "user-1" {
		t.Errorf("subject=%s", c.Subject)
	}
	if c.OrgID != "org-1" {
		t.Errorf("orgId=%s", c.OrgID)
	}
}

func TestClaimsWithScopes(t *testing.T) {
	c := Claims{Subject: "user-1", Scopes: []string{"intent:write", "deploy:execute"}}
	if len(c.Scopes) != 2 {
		t.Errorf("scopes=%d", len(c.Scopes))
	}
	if c.Scopes[0] != "intent:write" {
		t.Errorf("scope0=%s", c.Scopes[0])
	}
}

func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		err   error
		label string
	}{
		{ErrInvalidToken, "ErrInvalidToken"},
		{ErrTokenExpired, "ErrTokenExpired"},
		{ErrForbidden, "ErrForbidden"},
	}
	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			if tt.err == nil {
				t.Fatal("error is nil")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// FakeAuthProvider tests
// ---------------------------------------------------------------------------

func TestFakeAuthProviderDefaults(t *testing.T) {
	fp := NewFakeAuthProvider()
	claims, err := fp.Authenticate(nil, "any-token")
	if err != nil {
		t.Fatalf("auth: %v", err)
	}
	if claims.Subject != "fake-user" {
		t.Errorf("subject=%s", claims.Subject)
	}
	if claims.OrgID != "fake-org" {
		t.Errorf("orgId=%s", claims.OrgID)
	}
}

func TestFakeAuthProviderAlwaysInvalid(t *testing.T) {
	fp := NewFakeAuthProvider()
	fp.AlwaysValid = false
	fp.AlwaysInvalid = true

	_, err := fp.Authenticate(nil, "bad-token")
	if err != ErrInvalidToken {
		t.Errorf("err=%v", err)
	}
}

func TestFakeAuthProviderCustomClaims(t *testing.T) {
	fp := NewFakeAuthProvider()
	fp.ClaimsValue = Claims{Subject: "custom-user", OrgID: "custom-org"}

	claims, err := fp.Authenticate(nil, "token")
	if err != nil {
		t.Fatalf("auth: %v", err)
	}
	if claims.Subject != "custom-user" {
		t.Errorf("subject=%s", claims.Subject)
	}
}

func TestFakeAuthProviderRecordsTokens(t *testing.T) {
	fp := NewFakeAuthProvider()

	fp.Authenticate(nil, "token-1")
	fp.Authenticate(nil, "token-2")

	if fp.AuthenticateCount.Load() != 2 {
		t.Errorf("count=%d", fp.AuthenticateCount.Load())
	}
	if len(fp.ReceivedTokens) != 2 {
		t.Fatalf("tokens=%d", len(fp.ReceivedTokens))
	}
	if fp.ReceivedTokens[0] != "token-1" {
		t.Errorf("token0=%s", fp.ReceivedTokens[0])
	}
}

func TestFakeAuthProviderCustomFunc(t *testing.T) {
	fp := NewFakeAuthProvider()
	fp.AuthenticateFunc = func(_ context.Context, token string) (Claims, error) {
		if token == "allowed" {
			return Claims{Subject: "admin"}, nil
		}
		return Claims{}, ErrForbidden
	}

	claims, err := fp.Authenticate(nil, "allowed")
	if err != nil {
		t.Fatalf("auth: %v", err)
	}
	if claims.Subject != "admin" {
		t.Errorf("subject=%s", claims.Subject)
	}

	_, err = fp.Authenticate(nil, "denied")
	if err != ErrForbidden {
		t.Errorf("err=%v", err)
	}
}
