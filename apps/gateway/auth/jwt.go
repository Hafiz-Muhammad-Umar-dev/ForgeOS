// Package auth implements AuthProvider adapters for the DevOS API Gateway.
package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/auth"
)

// Compile-time check.
var _ auth.AuthProvider = (*JWTAdapter)(nil)

// JWTAdapter verifies HS256 JWTs using HMAC-SHA256.
// It parses the JWT, verifies the signature, extracts standard claims,
// and checks expiration.
type JWTAdapter struct {
	secret []byte
}

// NewJWTAdapter creates a JWTAdapter that verifies tokens signed with the
// given secret key.
func NewJWTAdapter(secret []byte) *JWTAdapter {
	return &JWTAdapter{secret: secret}
}

// jwtHeader is the parsed JWT header (only the alg field).
type jwtHeader struct {
	Alg string `json:"alg"`
	Kid string `json:"kid,omitempty"`
}

// jwtClaims is the parsed JWT claims payload.
type jwtClaims struct {
	Subject string   `json:"sub"`
	OrgID   string   `json:"org_id,omitempty"`
	Scopes  []string `json:"scopes,omitempty"`
	Exp     int64    `json:"exp"`
}

// Authenticate verifies the JWT and returns the embedded claims.
func (j *JWTAdapter) Authenticate(_ context.Context, token string) (auth.Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return auth.Claims{}, fmt.Errorf("%w: malformed jwt", auth.ErrInvalidToken)
	}

	// Verify signature.
	signingInput := parts[0] + "." + parts[1]
	sigBytes, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return auth.Claims{}, fmt.Errorf("%w: invalid signature encoding", auth.ErrInvalidToken)
	}

	mac := hmac.New(sha256.New, j.secret)
	mac.Write([]byte(signingInput))
	expected := mac.Sum(nil)

	if !hmac.Equal(sigBytes, expected) {
		return auth.Claims{}, fmt.Errorf("%w: signature mismatch", auth.ErrInvalidToken)
	}

	// Decode header (validate alg).
	var header jwtHeader
	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err == nil {
		json.Unmarshal(headerJSON, &header)
	}
	if header.Alg != "" && header.Alg != "HS256" {
		return auth.Claims{}, fmt.Errorf("%w: algorithm %q not accepted", auth.ErrInvalidToken, header.Alg)
	}

	// Decode payload.
	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return auth.Claims{}, fmt.Errorf("%w: invalid payload encoding", auth.ErrInvalidToken)
	}

	var claims jwtClaims
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return auth.Claims{}, fmt.Errorf("%w: invalid payload", auth.ErrInvalidToken)
	}

	// Check expiration.
	if claims.Exp > 0 && time.Now().Unix() > claims.Exp {
		return auth.Claims{}, fmt.Errorf("%w: token expired at %d", auth.ErrTokenExpired, claims.Exp)
	}

	return auth.Claims{
		Subject: claims.Subject,
		OrgID:   claims.OrgID,
		Scopes:  claims.Scopes,
	}, nil
}
