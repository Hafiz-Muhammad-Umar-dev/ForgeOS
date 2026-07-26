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
}

// GatewayConfig configures the Gateway HTTP server.
type GatewayConfig struct {
	// ListenAddr is the TCP address for the HTTP server.
	ListenAddr string

	// ShutdownTimeout is the max time to wait for in-flight requests
	// to complete during graceful shutdown.
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
	}
}

// ---------------------------------------------------------------------------
// Lifecycle
// ---------------------------------------------------------------------------

// Name returns "gateway" for the lifecycle manager.
func (g *Gateway) Name() string { return "gateway" }

// Init validates configuration and dependencies.
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

// Start begins serving HTTP requests.
func (g *Gateway) Start(_ context.Context) error {
	mux := http.NewServeMux()

	// Authenticated routes
	authHandler := middleware.Authenticate(g.provider, http.HandlerFunc(g.handleSubmitIntent))
	mux.Handle("/v1/intents", authHandler)

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

// Stop gracefully shuts down the HTTP server.
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

// Health reports whether the gateway is running.
func (g *Gateway) Health() lifecycle.Health {
	if g.server == nil {
		return lifecycle.Health{Status: lifecycle.StatusDown, Since: time.Now()}
	}
	return lifecycle.Health{Status: lifecycle.StatusUp, Since: time.Now()}
}

// ---------------------------------------------------------------------------
// HTTP handlers
// ---------------------------------------------------------------------------

func (g *Gateway) handleSubmitIntent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	claims, _ := middleware.ClaimsFromContext(r.Context())

	var req struct {
		Text      string             `json:"text"`
		UserID    string             `json:"user_id"`
		OrgID     string             `json:"org_id"`
		ProjectID string             `json:"project_id"`
		TraceID   string             `json:"trace_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	// Use claims identity when user_id is not provided.
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(result)
}

func (g *Gateway) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (g *Gateway) handleReadyz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func writeError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
