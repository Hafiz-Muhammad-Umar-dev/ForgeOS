// Package gateway implements the DevOS API Gateway HTTP server.
// It wraps the Intent Ingress with authentication middleware and exposes
// health endpoints. The Gateway implements lifecycle.Component for
// kernel integration.
package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/auth"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/ingress"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/lifecycle"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/apps/gateway/middleware"
)

// Compile-time check.
var _ lifecycle.Component = (*Gateway)(nil)

// Gateway is the API Gateway HTTP server. It authenticates requests via
// AuthProvider, routes authenticated intents to the Intent Ingress, and
// exposes health endpoints. It implements lifecycle.Component.
type Gateway struct {
	config   GatewayConfig
	provider auth.AuthProvider
	ingress  ingress.IntentIngress
	server   *http.Server
	mu       sync.RWMutex
	intents  []IntentItem
}

// IntentItem is a lightweight intent summary returned by GET /v1/intents.
type IntentItem struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// GatewayConfig configures the Gateway HTTP server.
type GatewayConfig struct {
	ListenAddr      string
	ShutdownTimeout time.Duration
}

// DefaultGatewayConfig returns a sensible default configuration.
func DefaultGatewayConfig() GatewayConfig {
	return GatewayConfig{
		ListenAddr:      ":8080",
		ShutdownTimeout: 10 * time.Second,
	}
}

// NewGateway creates a new Gateway server.
func NewGateway(cfg GatewayConfig, provider auth.AuthProvider, ing ingress.IntentIngress) *Gateway {
	return &Gateway{
		config:   cfg,
		provider: provider,
		ingress:  ing,
		intents:  seedIntents(),
	}
}

// seedIntents returns mock data when no database is available.
func seedIntents() []IntentItem {
	return []IntentItem{
		{ID: "intent-1", Title: "Build authentication flow", Status: "running", CreatedAt: time.Now().Add(-2 * time.Hour)},
		{ID: "intent-2", Title: "Implement API Gateway", Status: "completed", CreatedAt: time.Now().Add(-24 * time.Hour)},
		{ID: "intent-3", Title: "Design database schema", Status: "completed", CreatedAt: time.Now().Add(-48 * time.Hour)},
		{ID: "intent-4", Title: "Set up CI/CD pipeline", Status: "failed", CreatedAt: time.Now().Add(-72 * time.Hour)},
		{ID: "intent-5", Title: "Write unit tests for auth", Status: "pending", CreatedAt: time.Now().Add(-1 * time.Hour)},
	}
}

// RecordIntent stores a newly created intent in the mock list.
func (g *Gateway) RecordIntent(id, title string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.intents = append(g.intents, IntentItem{
		ID:        id,
		Title:     title,
		Status:    "pending",
		CreatedAt: time.Now(),
	})
}

// ---------------------------------------------------------------------------
// Lifecycle
// ---------------------------------------------------------------------------

func (g *Gateway) Name() string { return "gateway" }

func (g *Gateway) Init(_ context.Context) error {
	if g.config.ListenAddr == "" {
		return fmt.Errorf("gateway: listen address is required")
	}
	if g.provider == nil {
		return fmt.Errorf("gateway: auth provider is required")
	}
	if g.ingress == nil {
		return fmt.Errorf("gateway: ingress is required")
	}
	return nil
}

func (g *Gateway) Start(_ context.Context) error {
	mux := http.NewServeMux()

	// Wrap the authenticated handler dispatcher.
	authMW := middleware.Authenticate(g.provider, http.HandlerFunc(g.handleAuthenticated))

	// Authenticated routes
	mux.Handle("/v1/intents", authMW)

	// Health routes (no auth)
	mux.HandleFunc("/healthz", g.handleHealthz)
	mux.HandleFunc("/readyz", g.handleReadyz)

	g.server = &http.Server{
		Addr:    g.config.ListenAddr,
		Handler: mux,
	}

	go func() {
		log.Printf("gateway: listening on %s", g.config.ListenAddr)
		if err := g.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("gateway: server error: %v", err)
		}
	}()

	return nil
}

func (g *Gateway) Stop(ctx context.Context) error {
	if g.server == nil {
		return nil
	}
	shutdownCtx, cancel := context.WithTimeout(ctx, g.config.ShutdownTimeout)
	defer cancel()
	err := g.server.Shutdown(shutdownCtx)
	g.server = nil
	return err
}

func (g *Gateway) Health() lifecycle.Health {
	if g.server == nil {
		return lifecycle.Health{Status: lifecycle.StatusDown, Since: time.Now()}
	}
	return lifecycle.Health{Status: lifecycle.StatusUp, Since: time.Now()}
}

// ---------------------------------------------------------------------------
// Authenticated request dispatcher
// ---------------------------------------------------------------------------

func (g *Gateway) handleAuthenticated(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		g.handleListIntents(w, r)
	case http.MethodPost:
		g.handleSubmitIntent(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// ---------------------------------------------------------------------------
// GET /v1/intents
// ---------------------------------------------------------------------------

func (g *Gateway) handleListIntents(w http.ResponseWriter, r *http.Request) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var result []IntentItem
	if len(g.intents) > 0 {
		result = g.intents
	} else {
		result = seedIntents()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// ---------------------------------------------------------------------------
// POST /v1/intents
// ---------------------------------------------------------------------------

func (g *Gateway) handleSubmitIntent(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.ClaimsFromContext(r.Context())

	var req struct {
		Text      string `json:"text"`
		UserID    string `json:"user_id"`
		OrgID     string `json:"org_id"`
		ProjectID string `json:"project_id"`
		TraceID   string `json:"trace_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	userID := req.UserID
	if userID == "" {
		userID = claims.Subject
	}

	payload := ingress.IntentPayload{
		Text:      req.Text,
		UserID:    userID,
		OrgID:     claims.OrgID,
		ProjectID: req.ProjectID,
		TraceID:   req.TraceID,
	}

	result, err := g.ingress.SubmitIntent(r.Context(), payload)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Record in mock list.
	g.RecordIntent(result.IntentID, req.Text)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(result)
}

// ---------------------------------------------------------------------------
// Health
// ---------------------------------------------------------------------------

func (g *Gateway) handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (g *Gateway) handleReadyz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func writeError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}
