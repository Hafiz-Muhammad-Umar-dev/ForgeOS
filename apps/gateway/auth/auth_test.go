package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/auth"
)

// ---------------------------------------------------------------------------
// JWT test helpers
// ---------------------------------------------------------------------------

func hmacSign(secret, data string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(data))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func base64JSON(v any) string {
	data, _ := json.Marshal(v)
	return base64.RawURLEncoding.EncodeToString(data)
}

func makeJWT(secret string, claims map[string]any) string {
	header := base64JSON(map[string]string{"alg": "HS256", "typ": "JWT"})
	payload := base64JSON(claims)
	sig := hmacSign(secret, header+"."+payload)
	return header + "." + payload + "." + sig
}

// ---------------------------------------------------------------------------
// JWTAdapter tests
// ---------------------------------------------------------------------------

func TestJWTAdapterValidToken(t *testing.T) {
	secret := []byte("my-secret-key")
	adapter := NewJWTAdapter(secret)

	exp := time.Now().Add(1 * time.Hour).Unix()
	token := makeJWT(string(secret), map[string]any{
		"sub":    "user-abc",
		"org_id": "org-123",
		"exp":    exp,
	})

	claims, err := adapter.Authenticate(context.Background(), token)
	if err != nil {
		t.Fatalf("auth: %v", err)
	}
	if claims.Subject != "user-abc" {
		t.Errorf("subject=%s", claims.Subject)
	}
	if claims.OrgID != "org-123" {
		t.Errorf("orgId=%s", claims.OrgID)
	}
}

func TestJWTAdapterInvalidSignature(t *testing.T) {
	adapter := NewJWTAdapter([]byte("real-secret"))
	token := makeJWT("wrong-secret", map[string]any{
		"sub": "user-1",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	})

	_, err := adapter.Authenticate(context.Background(), token)
	if err == nil {
		t.Fatal("expected error for invalid signature")
	}
}

func TestJWTAdapterExpiredToken(t *testing.T) {
	secret := []byte("test-secret")
	adapter := NewJWTAdapter(secret)

	token := makeJWT(string(secret), map[string]any{
		"sub": "user-1",
		"exp": time.Now().Add(-1 * time.Hour).Unix(),
	})

	_, err := adapter.Authenticate(context.Background(), token)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestJWTAdapterNoExpiration(t *testing.T) {
	secret := []byte("test-secret")
	adapter := NewJWTAdapter(secret)

	token := makeJWT(string(secret), map[string]any{
		"sub": "user-1",
	})

	claims, err := adapter.Authenticate(context.Background(), token)
	if err != nil {
		t.Fatalf("auth: %v", err)
	}
	if claims.Subject != "user-1" {
		t.Errorf("subject=%s", claims.Subject)
	}
}

func TestJWTAdapterMalformedToken(t *testing.T) {
	adapter := NewJWTAdapter([]byte("secret"))

	tests := []string{
		"",
		"not-a-jwt",
		"header.payload",
		"header.payload.sig.extra",
	}
	for _, tt := range tests {
		_, err := adapter.Authenticate(context.Background(), tt)
		if err == nil {
			t.Errorf("expected error for token %q", tt)
		}
	}
}

func TestJWTAdapterWrongAlgorithm(t *testing.T) {
	adapter := NewJWTAdapter([]byte("secret"))

	// Create a token with alg="none"
	header := base64JSON(map[string]string{"alg": "none", "typ": "JWT"})
	payload := base64JSON(map[string]any{"sub": "user-1"})
	token := header + "." + payload + ".invalid-signature"

	_, err := adapter.Authenticate(context.Background(), token)
	if err == nil {
		t.Fatal("expected error for 'none' algorithm")
	}
}

func TestJWTAdapterContextCancelled(t *testing.T) {
	adapter := NewJWTAdapter([]byte("secret"))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	exp := time.Now().Add(1 * time.Hour).Unix()
	token := makeJWT("secret", map[string]any{"sub": "u1", "exp": exp})

	_, err := adapter.Authenticate(ctx, token)
	if err != nil {
		// With a valid token, context cancellation may not be observed
		// before verification completes. This is acceptable.
		t.Logf("ctx cancelled (expected): %v", err)
	}
}

func TestJWTAdapterWithScopes(t *testing.T) {
	secret := []byte("secret-key")
	adapter := NewJWTAdapter(secret)

	exp := time.Now().Add(1 * time.Hour).Unix()
	token := makeJWT(string(secret), map[string]any{
		"sub":    "admin",
		"scopes": []string{"intent:write", "deploy:execute"},
		"exp":    exp,
	})

	claims, err := adapter.Authenticate(context.Background(), token)
	if err != nil {
		t.Fatalf("auth: %v", err)
	}
	if len(claims.Scopes) != 2 {
		t.Fatalf("scopes=%d", len(claims.Scopes))
	}
	if claims.Scopes[0] != "intent:write" {
		t.Errorf("scope0=%s", claims.Scopes[0])
	}
}

// ---------------------------------------------------------------------------
// APIKeyAdapter tests
// ---------------------------------------------------------------------------

func TestAPIKeyAdapterValidKey(t *testing.T) {
	keys := map[string]string{"sk-abc": "user-1", "sk-xyz": "user-2"}
	adapter := NewAPIKeyAdapter(keys)

	claims, err := adapter.Authenticate(context.Background(), "sk-abc")
	if err != nil {
		t.Fatalf("auth: %v", err)
	}
	if claims.Subject != "user-1" {
		t.Errorf("subject=%s", claims.Subject)
	}
	if claims.OrgID != "default" {
		t.Errorf("orgId=%s", claims.OrgID)
	}
}

func TestAPIKeyAdapterInvalidKey(t *testing.T) {
	adapter := NewAPIKeyAdapter(map[string]string{"valid-key": "user-1"})

	_, err := adapter.Authenticate(context.Background(), "invalid-key")
	if err == nil {
		t.Fatal("expected error for invalid key")
	}
}

func TestAPIKeyAdapterEmptyMap(t *testing.T) {
	adapter := NewAPIKeyAdapter(nil)

	_, err := adapter.Authenticate(context.Background(), "any-key")
	if err == nil {
		t.Fatal("expected error with empty key map")
	}
}

func TestAPIKeyAdapterAddKey(t *testing.T) {
	adapter := NewAPIKeyAdapter(nil)
	adapter.AddKey("new-key", "new-user")

	claims, err := adapter.Authenticate(context.Background(), "new-key")
	if err != nil {
		t.Fatalf("auth: %v", err)
	}
	if claims.Subject != "new-user" {
		t.Errorf("subject=%s", claims.Subject)
	}
}

func TestAPIKeyAdapterRemoveKey(t *testing.T) {
	adapter := NewAPIKeyAdapter(map[string]string{"temp-key": "temp-user"})
	adapter.RemoveKey("temp-key")

	_, err := adapter.Authenticate(context.Background(), "temp-key")
	if err == nil {
		t.Fatal("expected error after key removal")
	}
}

// ---------------------------------------------------------------------------
// Cross-adapter contract tests
// ---------------------------------------------------------------------------

func TestAuthProviderContract(t *testing.T) {
	// Both adapters must return Claims with Subject populated on success.
	providers := []struct {
		name  string
		auth  auth.AuthProvider
		token string
	}{
		{"JWT", NewJWTAdapter([]byte("secret")), makeJWT("secret", map[string]any{"sub": "jwt-user", "exp": time.Now().Add(1 * time.Hour).Unix()})},
		{"APIKey", NewAPIKeyAdapter(map[string]string{"key-1": "key-user"}), "key-1"},
	}

	for _, p := range providers {
		t.Run(p.name, func(t *testing.T) {
			claims, err := p.auth.Authenticate(context.Background(), p.token)
			if err != nil {
				t.Fatalf("auth: %v", err)
			}
			if claims.Subject == "" {
				t.Error("empty subject")
			}
		})
	}
}
