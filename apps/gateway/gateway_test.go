package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/auth"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/ingress"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/intents"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/store"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/apps/gateway/middleware"
)

// testIntentRow is a store.Row that scans a populated intent.
type testIntentRow struct {
	id string
}

func (r *testIntentRow) Scan(dest ...any) error {
	now := time.Now()
	*(dest[0].(*string)) = r.id
	*(dest[1].(*string)) = "dev-admin"
	*(dest[2].(*string)) = "org-1"
	*(dest[3].(*string)) = ""
	*(dest[4].(*string)) = ""
	*(dest[5].(*string)) = "Build auth"
	*(dest[6].(*string)) = "pending"
	*(dest[7].(*string)) = ""
	*(dest[8].(*string)) = ""
	*(dest[9].(*time.Time)) = now
	*(dest[10].(*time.Time)) = now
	return nil
}

// newTestIntentsService builds an intents service backed by an in-memory store.
func newTestIntentsService() *intents.Service {
	fs := store.NewFakeStore()
	fs.QueryRowFunc = func(ctx context.Context, sql string, args ...any) store.Row {
		return &testIntentRow{id: args[0].(string)}
	}
	fs.QueryFunc = func(ctx context.Context, sql string, args ...any) (store.Rows, error) {
		return &storeEmptyRows{}, nil
	}
	return intents.NewService(intents.NewRepository(fs))
}

// storeEmptyRows is a store.Rows that yields no rows.
type storeEmptyRows struct{}

func (r *storeEmptyRows) Next() bool { return false }
func (r *storeEmptyRows) Scan(dest ...any) error { return nil }
func (r *storeEmptyRows) Close() {}

// ---------------------------------------------------------------------------
// Test ingress stub
// ---------------------------------------------------------------------------

type testIngress struct {
	lastPayload ingress.IntentPayload
}

func (t *testIngress) SubmitIntent(_ context.Context, payload ingress.IntentPayload) (ingress.IntentResult, error) {
	t.lastPayload = payload
	return ingress.IntentResult{
		IntentID: "intent-test-1",
		Status:   "accepted",
		TraceID:  "trace-test-1",
	}, nil
}

// ---------------------------------------------------------------------------
// Gateway integration tests
// ---------------------------------------------------------------------------

func newTestGateway(t *testing.T, provider auth.AuthProvider) *Gateway {
	t.Helper()
	cfg := DefaultGatewayConfig()
	cfg.ListenAddr = ":0" // random port (won't actually be used with httptest)
	return NewGateway(cfg, provider, &testIngress{}, newTestIntentsService())
}

func serveGateway(t *testing.T, g *Gateway) *httptest.Server {
	t.Helper()
	// Build the same middleware chain used by Gateway.Start().
	authHandler := middleware.Authenticate(g.provider, http.HandlerFunc(g.handleSubmitIntent))

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/intents":
			authHandler.ServeHTTP(w, r)
		case "/healthz":
			g.handleHealthz(w, r)
		case "/readyz":
			g.handleReadyz(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
}

func TestGatewaySubmitIntentAuthenticated(t *testing.T) {
	provider := auth.NewFakeAuthProvider()
	g := newTestGateway(t, provider)
	srv := serveGateway(t, g)
	defer srv.Close()

	body := `{"text":"build an app","project_id":"proj-1"}`
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/v1/intents", bytes.NewReader([]byte(body)))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer valid-token")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("status: got=%d want=201", resp.StatusCode)
	}

	var result ingress.IntentResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.IntentID == "" {
		t.Error("intent_id is empty")
	}
	if result.Status != "accepted" {
		t.Errorf("status=%s", result.Status)
	}
}

func TestGatewaySubmitIntentUnauthenticated(t *testing.T) {
	provider := auth.NewFakeAuthProvider()
	g := newTestGateway(t, provider)
	srv := serveGateway(t, g)
	defer srv.Close()

	body := `{"text":"should fail"}`
	resp, err := http.Post(srv.URL+"/v1/intents", "application/json", bytes.NewReader([]byte(body)))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status: got=%d want=401", resp.StatusCode)
	}
}

func TestGatewaySubmitIntentWithAuthFailure(t *testing.T) {
	provider := auth.NewFakeAuthProvider()
	provider.AlwaysValid = false
	provider.AlwaysInvalid = true
	g := newTestGateway(t, provider)
	srv := serveGateway(t, g)
	defer srv.Close()

	body := `{"text":"should fail"}`
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/intents", bytes.NewReader([]byte(body)))
	req.Header.Set("Authorization", "Bearer bad-token")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status: got=%d want=401", resp.StatusCode)
	}
}

func TestGatewaySubmitIntentInjectsClaimsIdentity(t *testing.T) {
	ing := &testIngress{}
	provider := auth.NewFakeAuthProvider()
	provider.ClaimsValue = auth.Claims{
		Subject: "jwt-user-abc",
		OrgID:   "jwt-org-xyz",
		Scopes:  []string{"intent:write"},
	}

	g := NewGateway(DefaultGatewayConfig(), provider, ing, newTestIntentsService())
	srv := serveGateway(t, g)
	defer srv.Close()

	body := `{"text":"build an app"}`
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/intents", bytes.NewReader([]byte(body)))
	req.Header.Set("Authorization", "Bearer any-token")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("status: got=%d want=201", resp.StatusCode)
	}

	// Verify the ingress received the claims identity
	if ing.lastPayload.UserID != "jwt-user-abc" {
		t.Errorf("userID: got=%s", ing.lastPayload.UserID)
	}
	if ing.lastPayload.OrgID != "jwt-org-xyz" {
		t.Errorf("orgID: got=%s", ing.lastPayload.OrgID)
	}
}

func TestGatewaySubmitIntentUserIDOverride(t *testing.T) {
	ing := &testIngress{}
	provider := auth.NewFakeAuthProvider()
	provider.ClaimsValue = auth.Claims{Subject: "jwt-user", OrgID: "jwt-org"}

	g := NewGateway(DefaultGatewayConfig(), provider, ing, newTestIntentsService())
	srv := serveGateway(t, g)
	defer srv.Close()

	// Request provides its own user_id — should be used instead of claims
	body := `{"text":"build","user_id":"override-user"}`
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/intents", bytes.NewReader([]byte(body)))
	req.Header.Set("Authorization", "Bearer token")

	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("status: got=%d want=201", resp.StatusCode)
	}

	if ing.lastPayload.UserID != "override-user" {
		t.Errorf("userID: got=%s", ing.lastPayload.UserID)
	}
}

func TestGatewayHealthz(t *testing.T) {
	provider := auth.NewFakeAuthProvider()
	g := newTestGateway(t, provider)
	srv := serveGateway(t, g)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status: got=%d", resp.StatusCode)
	}

	var body map[string]string
	json.NewDecoder(resp.Body).Decode(&body)
	if body["status"] != "ok" {
		t.Errorf("status=%s", body["status"])
	}
}

func TestGatewayReadyz(t *testing.T) {
	provider := auth.NewFakeAuthProvider()
	g := newTestGateway(t, provider)
	srv := serveGateway(t, g)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/readyz")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status: got=%d", resp.StatusCode)
	}

	var body map[string]string
	json.NewDecoder(resp.Body).Decode(&body)
	if body["status"] != "ready" {
		t.Errorf("status=%s", body["status"])
	}
}

func TestGatewayLifecycle(t *testing.T) {
	provider := auth.NewFakeAuthProvider()
	g := NewGateway(DefaultGatewayConfig(), provider, &testIngress{}, newTestIntentsService())

	if g.Name() != "gateway" {
		t.Errorf("name=%s", g.Name())
	}

	if err := g.Init(context.Background()); err != nil {
		t.Fatalf("init: %v", err)
	}

	// Start
	if err := g.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}

	h := g.Health()
	if h.Status != "UP" {
		t.Errorf("health after start: got=%s", h.Status)
	}

	// Stop
	if err := g.Stop(context.Background()); err != nil {
		t.Fatalf("stop: %v", err)
	}

	h = g.Health()
	if h.Status == "UP" {
		t.Errorf("health after stop: got=%s", h.Status)
	}
}

func TestGatewayInitNoAddress(t *testing.T) {
	g := NewGateway(GatewayConfig{ListenAddr: ""}, auth.NewFakeAuthProvider(), &testIngress{}, newTestIntentsService())
	if err := g.Init(context.Background()); err == nil {
		t.Fatal("expected error with empty listen address")
	}
}

func TestGatewayInitNoAuthProvider(t *testing.T) {
	g := NewGateway(DefaultGatewayConfig(), nil, &testIngress{}, newTestIntentsService())
	if err := g.Init(context.Background()); err == nil {
		t.Fatal("expected error without auth provider")
	}
}

func TestGatewayInitNoIngress(t *testing.T) {
	g := NewGateway(DefaultGatewayConfig(), auth.NewFakeAuthProvider(), nil, newTestIntentsService())
	if err := g.Init(context.Background()); err == nil {
		t.Fatal("expected error without ingress")
	}
}
