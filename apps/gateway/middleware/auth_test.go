package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/auth"
)

func newOKHandler(t *testing.T) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "no claims in context"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(claims)
	})
}

func TestAuthenticateValidToken(t *testing.T) {
	provider := auth.NewFakeAuthProvider()
	handler := Authenticate(provider, newOKHandler(t))

	req := httptest.NewRequest(http.MethodPost, "/v1/intents", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status: got=%d want=200", rec.Code)
	}

	var claims auth.Claims
	if err := json.NewDecoder(rec.Body).Decode(&claims); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if claims.Subject != "fake-user" {
		t.Errorf("subject=%s", claims.Subject)
	}
}

func TestAuthenticateMissingHeader(t *testing.T) {
	provider := auth.NewFakeAuthProvider()
	handler := Authenticate(provider, newOKHandler(t))

	req := httptest.NewRequest(http.MethodPost, "/v1/intents", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status: got=%d want=401", rec.Code)
	}
}

func TestAuthenticateInvalidAuthFormat(t *testing.T) {
	provider := auth.NewFakeAuthProvider()
	handler := Authenticate(provider, newOKHandler(t))

	req := httptest.NewRequest(http.MethodPost, "/v1/intents", nil)
	req.Header.Set("Authorization", "Invalid-Format") // not Bearer
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status: got=%d want=401", rec.Code)
	}
}

func TestAuthenticateProviderError(t *testing.T) {
	provider := auth.NewFakeAuthProvider()
	provider.AlwaysValid = false
	provider.AlwaysInvalid = true

	handler := Authenticate(provider, newOKHandler(t))

	req := httptest.NewRequest(http.MethodPost, "/v1/intents", nil)
	req.Header.Set("Authorization", "Bearer bad-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status: got=%d want=401", rec.Code)
	}
}

func TestAuthenticateClaimsInContext(t *testing.T) {
	provider := auth.NewFakeAuthProvider()
	provider.ClaimsValue = auth.Claims{
		Subject: "custom-sub",
		OrgID:   "custom-org",
		Scopes:  []string{"custom:scope"},
	}

	var capturedClaims auth.Claims
	handler := Authenticate(provider, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedClaims, _ = ClaimsFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/v1/intents", nil)
	req.Header.Set("Authorization", "Bearer any-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if capturedClaims.Subject != "custom-sub" {
		t.Errorf("subject=%s", capturedClaims.Subject)
	}
	if capturedClaims.OrgID != "custom-org" {
		t.Errorf("orgId=%s", capturedClaims.OrgID)
	}
}

func TestClaimsFromContextMissing(t *testing.T) {
	_, ok := ClaimsFromContext(context.Background())
	if ok {
		t.Error("should be false for missing claims")
	}
}

func TestAuthenticateBearerTokenExtraction(t *testing.T) {
	tests := []struct {
		name        string
		header      string
		wantStatus  int
		wantSubject string
	}{
		{"valid Bearer", "Bearer mytoken", http.StatusOK, "fake-user"},
		{"lowercase bearer", "bearer mytoken", http.StatusOK, "fake-user"},
		{"missing space", "Bearermytoken", http.StatusUnauthorized, ""},
		{"empty header", "", http.StatusUnauthorized, ""},
		{"wrong prefix", "Token mytoken", http.StatusUnauthorized, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := auth.NewFakeAuthProvider()
			handler := Authenticate(provider, newOKHandler(t))

			req := httptest.NewRequest(http.MethodPost, "/v1/intents", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status: got=%d want=%d", rec.Code, tt.wantStatus)
			}
		})
	}
}
