package agent

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/bus"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/event"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/lifecycle"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/provider"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/workspace"
)

// Compile-time check: *Runtime implements lifecycle.Component.
var _ lifecycle.Component = (*Runtime)(nil)

// Runtime manages agent execution. It owns the agent registry, dispatches
// tasks to agents, and provides the execution loop with LLM and workspace
// access. It implements lifecycle.Component for kernel integration.
type Runtime struct {
	mu       sync.RWMutex
	agents   map[string]Agent
	provider provider.LLMProvider
	ws       workspace.WorkspacePort
	wsID     workspace.WorkspaceID
	bus      bus.BusPort
	cfg      Config
	ready    bool
}

// RuntimeOption configures the Runtime.
type RuntimeOption func(*Runtime)

// WithRuntimeConfig sets the runtime configuration.
func WithRuntimeConfig(cfg Config) RuntimeOption {
	return func(r *Runtime) { r.cfg = cfg }
}

// WithAgentBus attaches a message bus for event publishing.
func WithAgentBus(b bus.BusPort) RuntimeOption {
	return func(r *Runtime) { r.bus = b }
}

// NewRuntime creates a new agent runtime with the given provider and workspace.
// At least one agent must be registered before RunTask is called.
func NewRuntime(
	llm provider.LLMProvider,
	ws workspace.WorkspacePort,
	wsID workspace.WorkspaceID,
	opts ...RuntimeOption,
) *Runtime {
	r := &Runtime{
		agents:   make(map[string]Agent),
		provider: llm,
		ws:       ws,
		wsID:     wsID,
		cfg:      DefaultConfig(),
	}
	for _, fn := range opts {
		fn(r)
	}
	return r
}

// RegisterAgent adds an agent to the runtime's registry.
func (r *Runtime) RegisterAgent(a Agent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.agents[a.Name()] = a
}

// Agent returns the named agent, or nil.
func (r *Runtime) Agent(name string) Agent {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.agents[name]
}

// RunTask executes a task with the named agent. It creates the execution
// context, provisions tools from the workspace, runs the agent, and returns
// the result.
func (r *Runtime) RunTask(ctx context.Context, agentName string, task Task) (*Result, error) {
	r.mu.RLock()
	agent, ok := r.agents[agentName]
	cfg := r.cfg
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("runtime: %w: agent %q not registered", ErrTaskFailed, agentName)
	}

	maxIter := task.MaxIterations
	if maxIter <= 0 {
		maxIter = cfg.DefaultMaxIterations
	}
	task.MaxIterations = maxIter

	r.publishEvent(ctx, newTaskStartedEvent(task, agentName))

	ws := r.ws
	wsID := r.wsID
	tools := DefaultTools(ws, wsID)

	ac := NewContext(
		ctx,
		task,
		tools,
		r.provider,
		ws,
		WithBus(r.bus),
	)

	result, err := agent.Run(ac)
	if err != nil {
		r.publishEvent(ctx, newTaskFailedEvent(task, agentName, err))
		return nil, fmt.Errorf("runtime: %w: %w", ErrTaskFailed, err)
	}

	r.publishEvent(ctx, newTaskCompletedEvent(task, agentName, result))
	return result, nil
}

// RunTaskStream executes a task and returns a channel of AgentEvents for
// real-time observation. The channel is closed when execution completes.
func (r *Runtime) RunTaskStream(ctx context.Context, agentName string, task Task) (<-chan AgentEvent, error) {
	r.mu.RLock()
	agent, ok := r.agents[agentName]
	cfg := r.cfg
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("runtime: %w: agent %q not registered", ErrTaskFailed, agentName)
	}

	maxIter := task.MaxIterations
	if maxIter <= 0 {
		maxIter = cfg.DefaultMaxIterations
	}
	task.MaxIterations = maxIter

	ws := r.ws
	wsID := r.wsID
	tools := DefaultTools(ws, wsID)

	ac := NewContext(
		ctx,
		task,
		tools,
		r.provider,
		ws,
		WithBus(r.bus),
	)

	ch := make(chan AgentEvent, 64)
	go r.runStream(ctx, agent, ac, ch)
	return ch, nil
}

// runStream executes the agent in a goroutine and sends events on the channel.
// Every channel send is guarded by a ctx.Done() check so the goroutine exits
// cleanly when the consumer stops reading or the context is cancelled.
func (r *Runtime) runStream(ctx context.Context, agent Agent, ac Context, ch chan<- AgentEvent) {
	defer close(ch)

	r.publishEvent(ctx, newTaskStartedEvent(ac.Task, agent.Name()))

	select {
	case <-ctx.Done():
		return
	case ch <- AgentEvent{Content: fmt.Sprintf("starting agent %q for task %q\n", agent.Name(), ac.Task.Description)}:
	}

	result, err := agent.Run(ac)
	if err != nil {
		r.publishEvent(ctx, newTaskFailedEvent(ac.Task, agent.Name(), err))
		select {
		case <-ctx.Done():
			return
		case ch <- AgentEvent{Err: err}:
		}
		return
	}

	select {
	case <-ctx.Done():
		return
	case ch <- AgentEvent{Done: true, Result: result}:
	}
	r.publishEvent(ctx, newTaskCompletedEvent(ac.Task, agent.Name(), result))
}

// ---------------------------------------------------------------------------
// Lifecycle integration
// ---------------------------------------------------------------------------

// Name returns "agent" for the lifecycle manager.
func (r *Runtime) Name() string { return "agent" }

// Init validates the runtime configuration.
func (r *Runtime) Init(ctx context.Context) error {
	if r.provider == nil {
		return fmt.Errorf("agent: %w: provider is required", ErrAgentNotReady)
	}
	if r.ws == nil {
		return fmt.Errorf("agent: %w: workspace is required", ErrAgentNotReady)
	}
	return nil
}

// Start marks the runtime as ready.
func (r *Runtime) Start(ctx context.Context) error {
	r.mu.Lock()
	r.ready = true
	r.mu.Unlock()
	return nil
}

// Stop marks the runtime as not ready. In-flight tasks are cancelled via
// their context, not by Stop.
func (r *Runtime) Stop(ctx context.Context) error {
	r.mu.Lock()
	r.ready = false
	r.mu.Unlock()
	return nil
}

// Health reports whether the runtime is ready.
func (r *Runtime) Health() lifecycle.Health {
	r.mu.RLock()
	ready := r.ready
	r.mu.RUnlock()
	if !ready {
		return lifecycle.Health{Status: lifecycle.StatusDown, Since: time.Now()}
	}
	return lifecycle.Health{Status: lifecycle.StatusUp, Since: time.Now()}
}

// ---------------------------------------------------------------------------
// Event publishing
// ---------------------------------------------------------------------------

func (r *Runtime) publishEvent(ctx context.Context, evt event.RawEnvelope) {
	if r.bus == nil || !r.cfg.PublishEvents {
		return
	}
	data, err := json.Marshal(evt)
	if err != nil {
		return
	}
	_ = r.bus.Publish(ctx, evt.Subject(), data)
}

func newTaskStartedEvent(task Task, agentName string) event.RawEnvelope {
	return event.RawEnvelope{
		ID:         newEventID(),
		Type:       event.TypeTaskStatus,
		ProducedBy: agentName,
		ProducedAt: time.Now().UnixNano(),
		Payload:    mustMarshal(map[string]any{"task_id": task.ID, "status": "started"}),
	}
}

func newTaskCompletedEvent(task Task, agentName string, result *Result) event.RawEnvelope {
	return event.RawEnvelope{
		ID:         newEventID(),
		Type:       event.TypeTaskStatus,
		ProducedBy: agentName,
		ProducedAt: time.Now().UnixNano(),
		Payload:    mustMarshal(map[string]any{"task_id": task.ID, "status": "completed", "summary": result.Summary}),
	}
}

func newTaskFailedEvent(task Task, agentName string, err error) event.RawEnvelope {
	return event.RawEnvelope{
		ID:         newEventID(),
		Type:       event.TypeTaskFailed,
		ProducedBy: agentName,
		ProducedAt: time.Now().UnixNano(),
		Payload:    mustMarshal(map[string]any{"task_id": task.ID, "status": "failed", "error": err.Error()}),
	}
}

func newEventID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("evt-%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func mustMarshal(v any) json.RawMessage {
	data, _ := json.Marshal(v)
	return data
}