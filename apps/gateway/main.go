// Package gateway implements the DevOS API Gateway HTTP server.
package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/apps/gateway/middleware"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/agents"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/auth"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/execution"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/ingress"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/intents"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/lifecycle"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/workspacefs"
)

var _ lifecycle.Component = (*Gateway)(nil)

type Gateway struct {
	config          GatewayConfig
	provider        auth.AuthProvider
	ingress         ingress.IntentIngress
	intentsSvc      *intents.Service
	workspaceSvc    *workspacefs.Service
	agentsSvc       *agents.Service
	execSvc         *execution.Service
	defaultWsID     string
	defaultWsLoaded bool
	server          *http.Server
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

// NewGateway creates a Gateway backed by persistence services.
func NewGateway(cfg GatewayConfig, provider auth.AuthProvider, ing ingress.IntentIngress, svc *intents.Service, ws *workspacefs.Service, as *agents.Service, es *execution.Service) *Gateway {
	return &Gateway{
		config:       cfg,
		provider:     provider,
		ingress:      ing,
		intentsSvc:   svc,
		workspaceSvc: ws,
		agentsSvc:    as,
		execSvc:      es,
	}
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
	if g.intentsSvc == nil {
		return fmt.Errorf("gateway: intents service is required")
	}
	if g.workspaceSvc == nil {
		return fmt.Errorf("gateway: workspace service is required")
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
	case strings.HasPrefix(path, "/v1/workspace/") && method == http.MethodGet:
		g.handleWorkspaceGet(w, r, path)
	case strings.HasPrefix(path, "/v1/workspace/") && method == http.MethodPost:
		g.handleWorkspacePost(w, r, path)
	case strings.HasPrefix(path, "/v1/workspace/") && method == http.MethodPut:
		g.handleWorkspacePut(w, r, path)
	case strings.HasPrefix(path, "/v1/workspace/") && method == http.MethodPatch:
		g.handleWorkspacePatch(w, r, path)
	case strings.HasPrefix(path, "/v1/workspace/") && method == http.MethodDelete:
		g.handleWorkspaceDelete(w, r, path)
	case strings.HasPrefix(path, "/v1/agents/") && method == http.MethodGet:
		g.handleGetAgent(w, r, path)
	case path == "/v1/agents" && method == http.MethodGet:
		g.handleListAgents(w, r)
	case path == "/v1/executions" && method == http.MethodGet:
		g.handleListExecutions(w, r)
	case strings.HasPrefix(path, "/v1/executions/") && method == http.MethodGet:
		g.handleExecutionGetOrSub(w, r, path)
	case path == "/v1/executions" && method == http.MethodPost:
		// Not used, but reserve for future
		g.handleV1Mock(w, r, path, method)
	case strings.HasPrefix(path, "/v1/executions/") && method == http.MethodPost:
		// Check for intent-based action routes: /v1/executions/{intentId}/{action}
		// or /v1/executions/{intentId}/metrics or /v1/executions/{intentId}/events
		if strings.Contains(path, "/metrics") {
			g.handleExecutionMetricsByIntent(w, r)
			return
		}
		if strings.Contains(path, "/events") {
			g.handleExecutionEventsByIntent(w, r)
			return
		}
		// Check if it's an action route (run, pause, resume, stop)
		rest := strings.TrimPrefix(path, "/v1/executions/")
		parts := strings.Split(rest, "/")
		if len(parts) == 2 {
			// Could be intent-based action or execution-based action
			// We'll try intent-based first for frontend compatibility
			g.handleExecutionActionByIntent(w, r)
			return
		}
		g.handleExecutionAction(w, r)
	default:
		g.handleV1Mock(w, r, path, method)
	}
}

// ---------------------------------------------------------------------------
// GET /v1/intents
// ---------------------------------------------------------------------------

func (g *Gateway) handleListIntents(w http.ResponseWriter, r *http.Request) {
	orgID := r.URL.Query().Get("org_id")
	intents, err := g.intentsSvc.ListIntents(r.Context(), orgID, 100, 0)
	if err != nil {
		log.Printf("gateway: list intents: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list intents")
		return
	}
	writeJSON(w, http.StatusOK, intents)
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
	intent, err := g.intentsSvc.GetIntent(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "intent not found")
		return
	}
	writeJSON(w, http.StatusOK, intent)
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
	orgID := req.OrgID
	if orgID == "" {
		orgID = claims.OrgID
	}

	// Persist the intent via the database-backed service.
	intent, err := g.intentsSvc.CreateIntent(r.Context(), intents.NewIntentRequest{
		Text: req.Text, UserID: userID, OrgID: orgID,
		ProjectID: req.ProjectID, TraceID: req.TraceID,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Publish to the bus via the ingress.
	payload := ingress.IntentPayload{
		Text: req.Text, UserID: userID, OrgID: orgID,
		ProjectID: req.ProjectID, TraceID: req.TraceID,
	}
	result, err := g.ingress.SubmitIntent(r.Context(), payload)
	if err != nil {
		log.Printf("gateway: publish intent: %v", err)
	}

	writeJSON(w, http.StatusCreated, ingress.IntentResult{
		IntentID: intent.ID,
		Status:   "accepted",
		TraceID:  result.TraceID,
	})
}

// ---------------------------------------------------------------------------
// GET /v1/tasks?intentId=
// ---------------------------------------------------------------------------

func (g *Gateway) handleListTasks(w http.ResponseWriter, r *http.Request) {
	intentID := r.URL.Query().Get("intentId")
	tasks, err := g.intentsSvc.ListTasks(r.Context(), intentID)
	if err != nil {
		log.Printf("gateway: list tasks: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list tasks")
		return
	}
	writeJSON(w, http.StatusOK, tasks)
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
