package registry

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/lifecycle"
)

// Compile-time checks.
var (
	_ Registry            = (*InMemoryRegistry)(nil)
	_ lifecycle.Component = (*InMemoryRegistry)(nil)
)

// InMemoryRegistry is an in-memory implementation of the Registry port.
// It is fully thread-safe and suitable for single-process deployments,
// development, and testing. Production deployments should use the
// NATS-KV-backed implementation (future sprint).
type InMemoryRegistry struct {
	mu      sync.RWMutex
	entries map[string]ServiceInfo
	ready   bool
	nowFn   func() time.Time
}

// InMemoryOption configures the InMemoryRegistry.
type InMemoryOption func(*InMemoryRegistry)

// NewInMemoryRegistry creates a new InMemoryRegistry.
func NewInMemoryRegistry(opts ...InMemoryOption) *InMemoryRegistry {
	r := &InMemoryRegistry{
		entries: make(map[string]ServiceInfo),
		nowFn:   time.Now,
	}
	for _, fn := range opts {
		fn(r)
	}
	return r
}

// ---------------------------------------------------------------------------
// Generic Registry methods
// ---------------------------------------------------------------------------

// Register adds a service to the registry. Idempotent: calling Register
// twice with the same ID is an error (ErrAlreadyRegistered).
func (r *InMemoryRegistry) Register(_ context.Context, info ServiceInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.entries[info.ID]; exists {
		return fmt.Errorf("%w: id=%s", ErrAlreadyRegistered, info.ID)
	}

	info.RegisteredAt = r.nowFn()
	r.entries[info.ID] = info
	return nil
}

// Deregister removes a service by ID.
func (r *InMemoryRegistry) Deregister(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.entries[id]; !exists {
		return fmt.Errorf("%w: id=%s", ErrNotFound, id)
	}
	delete(r.entries, id)
	return nil
}

// Discover returns all services of the given kind.
func (r *InMemoryRegistry) Discover(_ context.Context, kind string) ([]ServiceInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var out []ServiceInfo
	for _, v := range r.entries {
		if v.Kind == kind {
			out = append(out, v)
		}
	}
	return out, nil
}

// Resolve returns the first service that provides the given capability.
func (r *InMemoryRegistry) Resolve(_ context.Context, cap Capability) (ServiceInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, v := range r.entries {
		if v.Has(cap) {
			return v, nil
		}
	}
	return ServiceInfo{}, fmt.Errorf("%w: capability=%s", ErrNotFound, cap)
}

// ResolveByID returns the service with the given ID.
func (r *InMemoryRegistry) ResolveByID(_ context.Context, id string) (ServiceInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	v, ok := r.entries[id]
	if !ok {
		return ServiceInfo{}, fmt.Errorf("%w: id=%s", ErrNotFound, id)
	}
	return v, nil
}

// Watch returns a channel that receives capability-matched updates.
// Sprint 0 implementation: returns a closed channel. A full
// implementation with NATS KV watch is deferred to a later sprint.
func (r *InMemoryRegistry) Watch(_ context.Context, _ Capability) (<-chan ServiceInfo, error) {
	ch := make(chan ServiceInfo)
	close(ch)
	return ch, nil
}

// ---------------------------------------------------------------------------
// Typed convenience methods
// ---------------------------------------------------------------------------

// RegisterAgent registers a service with KindAgent.
func (r *InMemoryRegistry) RegisterAgent(ctx context.Context, info ServiceInfo) error {
	if info.Kind == "" {
		info.Kind = KindAgent
	}
	if info.Kind != KindAgent {
		return fmt.Errorf("%w: expected agent, got %s", ErrInvalidKind, info.Kind)
	}
	return r.Register(ctx, info)
}

// RegisterTool registers a service with KindTool.
func (r *InMemoryRegistry) RegisterTool(ctx context.Context, info ServiceInfo) error {
	if info.Kind == "" {
		info.Kind = KindTool
	}
	if info.Kind != KindTool {
		return fmt.Errorf("%w: expected tool, got %s", ErrInvalidKind, info.Kind)
	}
	return r.Register(ctx, info)
}

// RegisterProvider registers a service with KindProvider.
func (r *InMemoryRegistry) RegisterProvider(ctx context.Context, info ServiceInfo) error {
	if info.Kind == "" {
		info.Kind = KindProvider
	}
	if info.Kind != KindProvider {
		return fmt.Errorf("%w: expected provider, got %s", ErrInvalidKind, info.Kind)
	}
	return r.Register(ctx, info)
}

// GetAgent returns a registered agent by ID.
func (r *InMemoryRegistry) GetAgent(ctx context.Context, id string) (ServiceInfo, error) {
	return r.getByKind(ctx, id, KindAgent)
}

// GetTool returns a registered tool by ID.
func (r *InMemoryRegistry) GetTool(ctx context.Context, id string) (ServiceInfo, error) {
	return r.getByKind(ctx, id, KindTool)
}

// GetProvider returns a registered provider by ID.
func (r *InMemoryRegistry) GetProvider(ctx context.Context, id string) (ServiceInfo, error) {
	return r.getByKind(ctx, id, KindProvider)
}

// ListAgents returns all registered agents.
func (r *InMemoryRegistry) ListAgents(ctx context.Context) ([]ServiceInfo, error) {
	return r.Discover(ctx, KindAgent)
}

// ListTools returns all registered tools.
func (r *InMemoryRegistry) ListTools(ctx context.Context) ([]ServiceInfo, error) {
	return r.Discover(ctx, KindTool)
}

// ListProviders returns all registered providers.
func (r *InMemoryRegistry) ListProviders(ctx context.Context) ([]ServiceInfo, error) {
	return r.Discover(ctx, KindProvider)
}

// getByKind is a helper that resolves by ID and verifies the kind.
func (r *InMemoryRegistry) getByKind(_ context.Context, id, kind string) (ServiceInfo, error) {
	r.mu.RLock()
	v, ok := r.entries[id]
	r.mu.RUnlock()

	if !ok {
		return ServiceInfo{}, fmt.Errorf("%w: id=%s", ErrNotFound, id)
	}
	if v.Kind != kind {
		return ServiceInfo{}, fmt.Errorf("%w: id=%s kind=%s, expected %s", ErrNotFound, id, v.Kind, kind)
	}
	return v, nil
}

// ---------------------------------------------------------------------------
// Lifecycle integration
// ---------------------------------------------------------------------------

// Name returns "registry" for the lifecycle manager.
func (r *InMemoryRegistry) Name() string { return "registry" }

// Init validates the registry state.
func (r *InMemoryRegistry) Init(_ context.Context) error {
	return nil
}

// Start marks the registry as ready.
func (r *InMemoryRegistry) Start(_ context.Context) error {
	r.mu.Lock()
	r.ready = true
	r.mu.Unlock()
	return nil
}

// Stop marks the registry as not ready.
func (r *InMemoryRegistry) Stop(_ context.Context) error {
	r.mu.Lock()
	r.ready = false
	r.mu.Unlock()
	return nil
}

// Health reports whether the registry is ready.
func (r *InMemoryRegistry) Health() lifecycle.Health {
	r.mu.RLock()
	ready := r.ready
	r.mu.RUnlock()
	if !ready {
		return lifecycle.Health{Status: lifecycle.StatusDown, Since: time.Now()}
	}
	return lifecycle.Health{Status: lifecycle.StatusUp, Since: time.Now()}
}
