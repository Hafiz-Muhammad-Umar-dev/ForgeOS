// Package authservice implements the ForgeOS authentication service.
// It issues and verifies HS256 JWTs compatible with the Gateway's JWTAdapter.
//
// This is a DEVELOPMENT-ONLY auth service. It uses a hardcoded admin account
// and is not suitable for production.
package authservice

// LoginRequest is the JSON body for POST /auth/login.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse is the JSON response for a successful login.
type LoginResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
	User        User   `json:"user"`
}

// User represents an authenticated user.
type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}

// ErrorResponse is a generic JSON error response.
type ErrorResponse struct {
	Error string `json:"error"`
}
