package authservice

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
)

// devUser is the hardcoded development account.
type devUser struct {
	ID       string
	Username string
	Password string
	Role     string
	Name     string
	OrgID    string
}

// devAccount is the single development user.
var devAccount = devUser{
	ID:       "dev-admin",
	Username: "admin",
	Password: "admin123",
	Role:     "admin",
	Name:     "Administrator",
	OrgID:    "org-1",
}

const (
	tokenExpiry = 86400 // 24 hours in seconds
)

// Authenticator handles JWT signing and user authentication.
type Authenticator struct {
	secret []byte
}

// NewAuthenticator creates an Authenticator with the given HMAC secret.
func NewAuthenticator(secret []byte) *Authenticator {
	return &Authenticator{secret: secret}
}

// Login authenticates a user and returns a signed JWT.
func (a *Authenticator) Login(username, password string) (*LoginResponse, error) {
	if username != devAccount.Username || password != devAccount.Password {
		return nil, fmt.Errorf("invalid credentials")
	}

	token, err := a.signJWT(devAccount)
	if err != nil {
		return nil, fmt.Errorf("sign jwt: %w", err)
	}

	return &LoginResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   tokenExpiry,
		User: User{
			ID:   devAccount.ID,
			Name: devAccount.Name,
			Role: devAccount.Role,
		},
	}, nil
}

// ValidateUser returns the dev user without checking credentials (used by /auth/me).
func (a *Authenticator) ValidateUser() User {
	return User{
		ID:   devAccount.ID,
		Name: devAccount.Name,
		Role: devAccount.Role,
	}
}

// jwtHeader is the JWT header (must match gateway's jwtHeader for compatibility).
type jwtHeader struct {
	Alg string `json:"alg"`
}

// jwtPayload is the JWT claims payload (compatible with gateway's jwtClaims).
type jwtPayload struct {
	Subject string   `json:"sub"`
	OrgID   string   `json:"org_id,omitempty"`
	Role    string   `json:"role,omitempty"`
	Scopes  []string `json:"scopes,omitempty"`
	Exp     int64    `json:"exp"`
	Iat     int64    `json:"iat"`
}

// signJWT creates an HS256 JWT signed with the authenticator's secret,
// compatible with apps/gateway/auth/jwt.go's JWTAdapter.Authenticate verification.
func (a *Authenticator) signJWT(user devUser) (string, error) {
	// Header
	header := jwtHeader{Alg: "HS256"}
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("marshal header: %w", err)
	}
	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)

	// Payload
	now := time.Now().Unix()
	payload := jwtPayload{
		Subject: user.ID,
		OrgID:   user.OrgID,
		Role:    user.Role,
		Scopes:  []string{user.Role},
		Exp:     now + tokenExpiry,
		Iat:     now,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal payload: %w", err)
	}
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)

	// Signature: HMAC-SHA256 of "header.payload"
	signingInput := headerB64 + "." + payloadB64
	mac := hmac.New(sha256.New, a.secret)
	mac.Write([]byte(signingInput))
	sig := mac.Sum(nil)
	sigB64 := base64.RawURLEncoding.EncodeToString(sig)

	return signingInput + "." + sigB64, nil
}
