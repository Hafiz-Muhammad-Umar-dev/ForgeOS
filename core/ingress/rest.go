package ingress

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/bus"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/event"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/lifecycle"
)

// Compile-time checks.
var (
	_ IntentIngress       = (*RESTAdapter)(nil)
	_ lifecycle.Component = (*RESTAdapter)(nil)
)

// RESTAdapter is a thin HTTP transport layer that accepts intent requests,
// validates them, wraps them in a canonical event envelope, and publishes
// them to the bus. It implements both IntentIngress and lifecycle.Component.
type RESTAdapter struct {
	config RESTConfig
	server *http.Server
	bus    bus.BusPort
}

// NewRESTAdapter creates a new REST adapter with the given configuration.
func NewRESTAdapter(opts ...RESTOption) *RESTAdapter {
	cfg := DefaultRESTConfig()
	for _, fn := range opts {
		fn(&cfg)
	}
	return &RESTAdapter{config: cfg, bus: cfg.Bus}
}

// ---------------------------------------------------------------------------
// IntentIngress implementation
// ---------------------------------------------------------------------------

// SubmitIntent validates the payload, builds the canonical event envelope,
// and publishes it to the bus.
func (a *RESTAdapter) SubmitIntent(ctx context.Context, payload IntentPayload) (IntentResult, error) {
	if payload.Text == "" {
		return IntentResult{}, fmt.Errorf("%w: text is required", ErrInvalidRequest)
	}
	if a.config.MaxTextLength > 0 && len(payload.Text) > a.config.MaxTextLength {
		return IntentResult{}, fmt.Errorf("%w: text exceeds %d characters", ErrInvalidRequest, a.config.MaxTextLength)
	}

	intentID := newIntentID()
	traceID := payload.TraceID
	if traceID == "" {
		traceID = newTraceID()
	}
	orgID := payload.OrgID
	if orgID == "" {
		orgID = "default"
	}

	env := event.New(event.TypeIntentCreated, "ingress", payload,
		event.WithTraceID(traceID),
		event.WithOrgID(orgID),
		event.WithProjectID(payload.ProjectID),
	)

	// Override the generated ID with our own (event.New generates a
	// random UUID; we use our own format for the intent ID).
	env.ID = intentID

	data, err := event.Serialize(env)
	if err != nil {
		return IntentResult{}, fmt.Errorf("ingress: serialize: %w", err)
	}

	if err := a.bus.Publish(ctx, "devos.intent.created", data); err != nil {
		return IntentResult{}, fmt.Errorf("ingress: publish: %w", err)
	}

	return IntentResult{
		IntentID: intentID,
		Status:   "accepted",
		TraceID:  traceID,
	}, nil
}

// ---------------------------------------------------------------------------
// Lifecycle component
// ---------------------------------------------------------------------------

// Name returns "ingress" for the lifecycle manager.
func (a *RESTAdapter) Name() string { return "ingress" }

// Init validates configuration.
func (a *RESTAdapter) Init(_ context.Context) error {
	if a.bus == nil {
		return fmt.Errorf("ingress: bus is required")
	}
	return nil
}

// Start begins serving HTTP requests.
func (a *RESTAdapter) Start(_ context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/intents", a.handleSubmitIntent)
	mux.HandleFunc("/healthz", a.handleHealthz)
	mux.HandleFunc("/readyz", a.handleReadyz)

	a.server = &http.Server{
		Addr:         a.config.ListenAddr,
		Handler:      mux,
		ReadTimeout:  a.config.ReadTimeout,
		WriteTimeout: a.config.WriteTimeout,
	}

	go func() {
		log.Printf("ingress: listening on %s", a.config.ListenAddr)
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("ingress: server error: %v", err)
		}
	}()

	return nil
}

// Stop gracefully shuts down the HTTP server.
func (a *RESTAdapter) Stop(ctx context.Context) error {
	if a.server == nil {
		return nil
	}
	shutdownCtx, cancel := context.WithTimeout(ctx, a.config.ShutdownTimeout)
	defer cancel()
	err := a.server.Shutdown(shutdownCtx)
	a.server = nil
	return err
}

// Health reports whether the server is running.
func (a *RESTAdapter) Health() lifecycle.Health {
	if a.server == nil {
		return lifecycle.Health{Status: lifecycle.StatusDown, Since: time.Now()}
	}
	return lifecycle.Health{Status: lifecycle.StatusUp, Since: time.Now()}
}

// ---------------------------------------------------------------------------
// HTTP handlers
// ---------------------------------------------------------------------------

func (a *RESTAdapter) handleSubmitIntent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		Text        string       `json:"text"`
		UserID      string       `json:"user_id"`
		OrgID       string       `json:"org_id"`
		ProjectID   string       `json:"project_id"`
		TraceID     string       `json:"trace_id"`
		Attachments []Attachment `json:"attachments"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	payload := IntentPayload{
		Text:        req.Text,
		UserID:      req.UserID,
		OrgID:       req.OrgID,
		ProjectID:   req.ProjectID,
		TraceID:     req.TraceID,
		Attachments: req.Attachments,
	}

	result, err := a.SubmitIntent(r.Context(), payload)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(result)
}

func (a *RESTAdapter) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (a *RESTAdapter) handleReadyz(w http.ResponseWriter, r *http.Request) {
	if a.bus == nil || !a.bus.IsConnected() {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"status": "not ready"})
		return
	}
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

func newIntentID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("intent-%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("intent-%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func newTraceID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("trace-%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("trace-%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
