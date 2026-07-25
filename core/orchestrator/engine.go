package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/agent"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/bus"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/event"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/ingress"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/lifecycle"
)

// Compile-time check.
var _ lifecycle.Component = (*Engine)(nil)

// IntentLifecyclePayload is the event payload for intent.completed and
// intent.failed events.
type IntentLifecyclePayload struct {
	IntentID  string `json:"intent_id"`
	UserID    string `json:"user_id,omitempty"`
	OrgID     string `json:"org_id,omitempty"`
	ProjectID string `json:"project_id,omitempty"`
	TraceID   string `json:"trace_id,omitempty"`
	Summary   string `json:"summary,omitempty"`
	Error     string `json:"error,omitempty"`
}

// Engine subscribes to intent.created, dispatches to the Agent Runtime
// via TaskRunner, and publishes intent.completed / intent.failed events.
// It implements lifecycle.Component for kernel integration.
type Engine struct {
	bus     bus.BusPort
	runner  TaskRunner
	agent   string
	sub     bus.Subscription
	mu      sync.Mutex
	started bool
}

// EngineOption configures the Engine.
type EngineOption func(*Engine)

// WithAgent sets the default agent name for dispatch.
func WithAgent(name string) EngineOption {
	return func(e *Engine) { e.agent = name }
}

// NewEngine creates a new orchestration engine.
func NewEngine(b bus.BusPort, runner TaskRunner, opts ...EngineOption) *Engine {
	e := &Engine{
		bus:    b,
		runner: runner,
		agent:  "coder",
	}
	for _, fn := range opts {
		fn(e)
	}
	return e
}

// ---------------------------------------------------------------------------
// Lifecycle
// ---------------------------------------------------------------------------

// Name returns "orchestrator" for the lifecycle manager.
func (e *Engine) Name() string { return "orchestrator" }

// Init validates dependencies.
func (e *Engine) Init(_ context.Context) error {
	if e.bus == nil {
		return fmt.Errorf("orchestrator: %w: bus is required", ErrNotStarted)
	}
	if e.runner == nil {
		return fmt.Errorf("orchestrator: %w: runner is required", ErrNotStarted)
	}
	return nil
}

// Start subscribes to intent.created events.
func (e *Engine) Start(ctx context.Context) error {
	sub, err := e.bus.Subscribe(ctx, "devos.intent.created", e.handleIntentCreated)
	if err != nil {
		return fmt.Errorf("orchestrator: subscribe: %w", err)
	}
	e.mu.Lock()
	e.sub = sub
	e.started = true
	e.mu.Unlock()
	return nil
}

// Stop unsubscribes from intent.created.
func (e *Engine) Stop(_ context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.sub != nil {
		_ = e.sub.Unsubscribe()
		e.sub = nil
	}
	e.started = false
	return nil
}

// Health reports whether the engine is subscribed.
func (e *Engine) Health() lifecycle.Health {
	e.mu.Lock()
	s := e.started
	e.mu.Unlock()
	if !s {
		return lifecycle.Health{Status: lifecycle.StatusDown, Since: time.Now()}
	}
	return lifecycle.Health{Status: lifecycle.StatusUp, Since: time.Now()}
}

// ---------------------------------------------------------------------------
// Message handler
// ---------------------------------------------------------------------------

func (e *Engine) handleIntentCreated(ctx context.Context, msg bus.Message) error {
	env, err := event.Deserialize(msg.Data())
	if err != nil {
		_ = msg.Term()
		return nil
	}

	if env.Type != event.TypeIntentCreated {
		_ = msg.Ack()
		return nil
	}

	payload, err := event.UnmarshalPayload[ingress.IntentPayload](env)
	if err != nil {
		_ = msg.Term()
		return nil
	}

	task := agent.Task{
		ID:          env.ID,
		Description: payload.Text,
		Payload: map[string]any{
			"user_id":    payload.UserID,
			"org_id":     env.OrgID,
			"project_id": env.ProjectID,
			"trace_id":   env.TraceID,
		},
	}

	result, err := e.runner.RunTask(ctx, e.agent, task)
	if err != nil {
		e.publishFailed(ctx, env, payload, err)
		_ = msg.Ack()
		return nil
	}

	e.publishCompleted(ctx, env, payload, result)
	_ = msg.Ack()
	return nil
}

// ---------------------------------------------------------------------------
// Event publishing
// ---------------------------------------------------------------------------

func (e *Engine) publishCompleted(ctx context.Context, env event.RawEnvelope, p ingress.IntentPayload, result *agent.Result) {
	pl := IntentLifecyclePayload{
		IntentID:  env.ID,
		UserID:    p.UserID,
		OrgID:     env.OrgID,
		ProjectID: env.ProjectID,
		TraceID:   env.TraceID,
		Summary:   result.Summary,
	}
	_ = e.publish(ctx, event.TypeIntentCompleted, pl, env)
}

func (e *Engine) publishFailed(ctx context.Context, env event.RawEnvelope, p ingress.IntentPayload, err error) {
	pl := IntentLifecyclePayload{
		IntentID:  env.ID,
		UserID:    p.UserID,
		OrgID:     env.OrgID,
		ProjectID: env.ProjectID,
		TraceID:   env.TraceID,
		Error:     err.Error(),
	}
	_ = e.publish(ctx, event.TypeIntentFailed, pl, env)
}

func (e *Engine) publish(ctx context.Context, typ event.EventType, pl IntentLifecyclePayload, env event.RawEnvelope) error {
	envelope := event.New(typ, "orchestrator", pl,
		event.WithTraceID(env.TraceID),
		event.WithOrgID(env.OrgID),
		event.WithProjectID(env.ProjectID),
	)
	data, err := event.Serialize(envelope)
	if err != nil {
		return err
	}
	return e.bus.Publish(ctx, typ.Subject().String(), data)
}
