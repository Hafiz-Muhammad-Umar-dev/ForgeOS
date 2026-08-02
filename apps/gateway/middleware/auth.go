// Package middleware provides HTTP middleware for the DevOS API Gateway.
package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/auth"
)

var (
	ErrMissingAuth       = errors.New("missing authorization header")
	ErrInvalidAuthHeader = errors.New("invalid authorization header")
)

type contextKey string

const ClaimsKey contextKey = "auth.claims"

func ClaimsFromContext(ctx context.Context) (auth.Claims, bool) {
	c, ok := ctx.Value(ClaimsKey).(auth.Claims)
	return c, ok
}

// Authenticate wraps an http.Handler with JWT/API key authentication.
func Authenticate(provider auth.AuthProvider, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("AUTH: path=%s method=%s", r.URL.Path, r.Method)
		authHeader := r.Header.Get("Authorization")
		log.Printf("AUTH: Authorization header: %q", authHeader)

		token, err := extractBearerToken(r)
		if err != nil {
			log.Printf("AUTH: extract error: %v", err)
			writeAuthError(w, err.Error())
			return
		}
		log.Printf("AUTH: extracted token (len=%d): %.30s...", len(token), token)

		claims, err := provider.Authenticate(r.Context(), token)
		if err != nil {
			log.Printf("AUTH: Authenticate failed: %v", err)
			writeAuthError(w, err.Error())
			return
		}

		log.Printf("AUTH: success user=%s org=%s", claims.Subject, claims.OrgID)
		ctx := context.WithValue(r.Context(), ClaimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

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

func writeAuthError(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
