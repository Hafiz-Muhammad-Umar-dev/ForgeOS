// Package middleware provides HTTP middleware for the DevOS API Gateway.
package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/auth"
)

// Sentinel errors returned by the middleware.
var (
	ErrMissingAuth       = errors.New("missing authorization header")
	ErrInvalidAuthHeader = errors.New("invalid authorization header")
)

// contextKey is a private type for context keys to avoid collisions.
type contextKey string

// ClaimsKey is the context key for storing auth claims.
const ClaimsKey contextKey = "auth.claims"

// ClaimsFromContext extracts auth claims from a request context.
// Returns zero Claims and false if not present.
func ClaimsFromContext(ctx context.Context) (auth.Claims, bool) {
	c, ok := ctx.Value(ClaimsKey).(auth.Claims)
	return c, ok
}

// Authenticate wraps an http.Handler with JWT/API key authentication.
// It extracts the Bearer token from the Authorization header, calls the
// AuthProvider, and injects Claims into the request context on success.
// On failure, it returns a 401 JSON response.
func Authenticate(provider auth.AuthProvider, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := extractBearerToken(r)
		if err != nil {
			writeAuthError(w, err.Error())
			return
		}

		claims, err := provider.Authenticate(r.Context(), token)
		if err != nil {
			writeAuthError(w, err.Error())
			return
		}

		ctx := context.WithValue(r.Context(), ClaimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// extractBearerToken extracts the Bearer token from the Authorization header.
func extractBearerToken(r *http.Request) (string, error) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return "", ErrMissingAuth
	}

	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", ErrInvalidAuthHeader
	}

	return parts[1], nil
}

// writeAuthError writes a 401 JSON error response.
func writeAuthError(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
