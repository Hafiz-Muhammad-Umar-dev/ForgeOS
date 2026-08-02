// Package gateway implements the DevOS API Gateway HTTP server.
package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/auth"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/ingress"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/lifecycle"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/apps/gateway/middleware"
)

var _ lifecycle.Component = (*Gateway)(nil)

type Gateway struct {
	config   GatewayConfig
	provider auth.AuthProvider
	ingress  ingress.IntentIngress
	server   *http.Server
	mu       sync.RWMutex
	intents  []IntentItem
	tasks    []TaskItem
}

type IntentItem struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type TaskItem struct {
	ID        string    `json:"id"`
	IntentID  string    `json:"intent_id"`
	Summary   string    `json:"summary"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type GatewayConfig struct {
	ListenAddr      string
	ShutdownTimeout time.Duration
}

func DefaultGatewayConfig() GatewayConfig {
	return GatewayConfig{
		ListenAddr:      ":8080",
		ShutdownTimeout: 10 * time.Second,
	}
}

func NewGateway(cfg GatewayConfig, provider auth.AuthProvider, ing ingress.IntentIngress) *Gateway {
	return &Gateway{
		config:   cfg,
		provider: provider,
		ingress:  ing,
		intents:  seedIntents(),
		tasks:    seedTasks(),
	}
}

func seedIntents() []IntentItem {
	return []IntentItem{
		{ID: "intent-1", Title: "Build authentication flow", Status: "running", CreatedAt: time.Now().Add(-2 * time.Hour)},
		{ID: "intent-2", Title: "Implement API Gateway", Status: "completed", CreatedAt: time.Now().Add(-24 * time.Hour)},
		{ID: "intent-3", Title: "Design database schema", Status: "completed", CreatedAt: time.Now().Add(-48 * time.Hour)},
		{ID: "intent-4", Title: "Set up CI/CD pipeline", Status: "failed", CreatedAt: time.Now().Add(-72 * time.Hour)},
		{ID: "intent-5", Title: "Write unit tests for auth", Status: "pending", CreatedAt: time.Now().Add(-1 * time.Hour)},
	}
}

func seedTasks() []TaskItem {
	return []TaskItem{
		{ID: "task-1", IntentID: "intent-1", Summary: "Design auth schema", Status: "completed", CreatedAt: time.Now().Add(-2 * time.Hour)},
		{ID: "task-2", IntentID: "intent-1", Summary: "Implement JWT signing", Status: "running", CreatedAt: time.Now().Add(-1 * time.Hour)},
		{ID: "task-3", IntentID: "intent-1", Summary: "Write integration tests", Status: "pending", CreatedAt: time.Now().Add(-30 * time.Minute)},
		{ID: "task-4", IntentID: "intent-2", Summary: "Route planning", Status: "completed", CreatedAt: time.Now().Add(-24 * time.Hour)},
		{ID: "task-5", IntentID: "intent-3", Summary: "Schema migration", Status: "completed", CreatedAt: time.Now().Add(-48 * time.Hour)},
	}
}

func (g *Gateway) RecordIntent(id, title string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.intents = append(g.intents, IntentItem{
		ID: id, Title: title, Status: "pending", CreatedAt: time.Now(),
	})
}

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
	authMW := middleware.Authenticate(g.provider, http.HandlerFunc(g.handleAuthenticated))

	mux.Handle("GET /v1/intents", authMW)
	mux.Handle("POST /v1/intents", authMW)
	mux.Handle("GET /v1/tasks", authMW)
	mux.Handle("/v1/{rest...}", authMW)

	mux.HandleFunc("/healthz", g.handleHealthz)
	mux.HandleFunc("/readyz", g.handleReadyz)

	srv := &http.Server{Addr: g.config.ListenAddr, Handler: mux}
	g.server = srv

	go func() {
		log.Printf("gateway: listening on %s", g.config.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
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
	path := r.URL.Path
	method := r.Method

	switch {
	case path == "/v1/intents" && method == http.MethodGet:
		g.handleListIntents(w, r)
	case path == "/v1/intents" && method == http.MethodPost:
		g.handleSubmitIntent(w, r)
	case strings.HasPrefix(path, "/v1/intents/") && method == http.MethodGet:
		g.handleGetIntentByPath(w, r, path)
	case path == "/v1/tasks" && method == http.MethodGet:
		g.handleListTasks(w, r)
	default:
		g.handleV1Mock(w, r, path, method)
	}
}

// ---------------------------------------------------------------------------
// GET /v1/intents
// ---------------------------------------------------------------------------

func (g *Gateway) handleListIntents(w http.ResponseWriter, r *http.Request) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	writeJSON(w, http.StatusOK, g.intents)
}

// ---------------------------------------------------------------------------
// GET /v1/intents/{id}
// ---------------------------------------------------------------------------

func (g *Gateway) handleGetIntentByPath(w http.ResponseWriter, r *http.Request, path string) {
	id := strings.TrimPrefix(path, "/v1/intents/")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	for _, intent := range g.intents {
		if intent.ID == id {
			writeJSON(w, http.StatusOK, intent)
			return
		}
	}
	writeError(w, http.StatusNotFound, "intent not found")
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
		Text: req.Text, UserID: userID, OrgID: claims.OrgID,
		ProjectID: req.ProjectID, TraceID: req.TraceID,
	}
	result, err := g.ingress.SubmitIntent(r.Context(), payload)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	g.RecordIntent(result.IntentID, req.Text)
	writeJSON(w, http.StatusCreated, result)
}

// ---------------------------------------------------------------------------
// GET /v1/tasks?intentId=
// ---------------------------------------------------------------------------

func (g *Gateway) handleListTasks(w http.ResponseWriter, r *http.Request) {
	intentID := r.URL.Query().Get("intentId")
	g.mu.RLock()
	defer g.mu.RUnlock()
	if intentID != "" {
		var filtered []TaskItem
		for _, t := range g.tasks {
			if t.IntentID == intentID {
				filtered = append(filtered, t)
			}
		}
		writeJSON(w, http.StatusOK, filtered)
		return
	}
	writeJSON(w, http.StatusOK, g.tasks)
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
