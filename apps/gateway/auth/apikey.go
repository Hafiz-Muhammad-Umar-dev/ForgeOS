package auth

import (
	"context"
	"fmt"
	"sync"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/auth"
)

// Compile-time check.
var _ auth.AuthProvider = (*APIKeyAdapter)(nil)

// APIKeyAdapter validates static API keys from an in-memory map.
// Each key maps to an identity label.
type APIKeyAdapter struct {
	keys map[string]string // key → owner label
	mu   sync.RWMutex
}

// NewAPIKeyAdapter creates an adapter with the given key→owner mapping.
func NewAPIKeyAdapter(keys map[string]string) *APIKeyAdapter {
	if keys == nil {
		keys = make(map[string]string)
	}
	return &APIKeyAdapter{keys: keys}
}

// Authenticate looks up the token in the key map.
func (a *APIKeyAdapter) Authenticate(_ context.Context, token string) (auth.Claims, error) {
	a.mu.RLock()
	owner, ok := a.keys[token]
	a.mu.RUnlock()

	if !ok {
		return auth.Claims{}, fmt.Errorf("%w: unknown api key", auth.ErrInvalidToken)
	}

	return auth.Claims{
		Subject: owner,
		OrgID:   "default",
		Scopes:  []string{"intent:write"},
	}, nil
}

// AddKey adds or updates an API key at runtime.
func (a *APIKeyAdapter) AddKey(key, owner string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.keys[key] = owner
}

// RemoveKey removes an API key at runtime.
func (a *APIKeyAdapter) RemoveKey(key string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.keys, key)
}
