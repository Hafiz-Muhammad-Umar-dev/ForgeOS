package registry

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Compile-time check.
var _ Registry = (*FakeRegistry)(nil)

// FakeRegistry is an in-memory Registry implementation for testing.
// It records all registration requests and returns configurable results.
type FakeRegistry struct {
	// Entries holds the registered services.
	Entries map[string]ServiceInfo

	// RegisterCount tracks calls to Register.
	RegisterCount atomic.Int64

	// DeregisterCount tracks calls to Deregister.
	DeregisterCount atomic.Int64

	mu sync.Mutex
}

// NewFakeRegistry creates an empty FakeRegistry.
func NewFakeRegistry() *FakeRegistry {
	return &FakeRegistry{
		Entries: make(map[string]ServiceInfo),
	}
}

// Register adds an entry to the registry.
func (f *FakeRegistry) Register(_ context.Context, info ServiceInfo) error {
	f.RegisterCount.Add(1)
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, exists := f.Entries[info.ID]; exists {
		return fmt.Errorf("fake: %w: id=%s", ErrAlreadyRegistered, info.ID)
	}
	if info.RegisteredAt.IsZero() {
		info.RegisteredAt = time.Now()
	}
	f.Entries[info.ID] = info
	return nil
}

// Deregister removes an entry by ID.
func (f *FakeRegistry) Deregister(_ context.Context, id string) error {
	f.DeregisterCount.Add(1)
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, exists := f.Entries[id]; !exists {
		return fmt.Errorf("fake: %w: id=%s", ErrNotFound, id)
	}
	delete(f.Entries, id)
	return nil
}

// Discover returns entries of the given kind.
func (f *FakeRegistry) Discover(_ context.Context, kind string) ([]ServiceInfo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []ServiceInfo
	for _, v := range f.Entries {
		if v.Kind == kind {
			out = append(out, v)
		}
	}
	return out, nil
}

// Resolve returns the first entry with the given capability.
func (f *FakeRegistry) Resolve(_ context.Context, cap Capability) (ServiceInfo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, v := range f.Entries {
		if v.Has(cap) {
			return v, nil
		}
	}
	return ServiceInfo{}, fmt.Errorf("fake: %w: capability=%s", ErrNotFound, cap)
}

// ResolveByID returns an entry by ID.
func (f *FakeRegistry) ResolveByID(_ context.Context, id string) (ServiceInfo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	v, ok := f.Entries[id]
	if !ok {
		return ServiceInfo{}, fmt.Errorf("fake: %w: id=%s", ErrNotFound, id)
	}
	return v, nil
}

// Watch returns a closed channel. Full implementation deferred.
func (f *FakeRegistry) Watch(_ context.Context, _ Capability) (<-chan ServiceInfo, error) {
	ch := make(chan ServiceInfo)
	close(ch)
	return ch, nil
}

// RegisterAgent registers an agent entry.
func (f *FakeRegistry) RegisterAgent(ctx context.Context, info ServiceInfo) error {
	if info.Kind == "" {
		info.Kind = KindAgent
	}
	return f.Register(ctx, info)
}

// RegisterTool registers a tool entry.
func (f *FakeRegistry) RegisterTool(ctx context.Context, info ServiceInfo) error {
	if info.Kind == "" {
		info.Kind = KindTool
	}
	return f.Register(ctx, info)
}

// RegisterProvider registers a provider entry.
func (f *FakeRegistry) RegisterProvider(ctx context.Context, info ServiceInfo) error {
	if info.Kind == "" {
		info.Kind = KindProvider
	}
	return f.Register(ctx, info)
}

// GetAgent returns an agent by ID.
func (f *FakeRegistry) GetAgent(ctx context.Context, id string) (ServiceInfo, error) {
	info, err := f.ResolveByID(ctx, id)
	if err != nil {
		return info, err
	}
	if info.Kind != KindAgent {
		return ServiceInfo{}, fmt.Errorf("fake: %w: id=%s kind=%s", ErrNotFound, id, info.Kind)
	}
	return info, nil
}

// GetTool returns a tool by ID.
func (f *FakeRegistry) GetTool(ctx context.Context, id string) (ServiceInfo, error) {
	info, err := f.ResolveByID(ctx, id)
	if err != nil {
		return info, err
	}
	if info.Kind != KindTool {
		return ServiceInfo{}, fmt.Errorf("fake: %w: id=%s kind=%s", ErrNotFound, id, info.Kind)
	}
	return info, nil
}

// GetProvider returns a provider by ID.
func (f *FakeRegistry) GetProvider(ctx context.Context, id string) (ServiceInfo, error) {
	info, err := f.ResolveByID(ctx, id)
	if err != nil {
		return info, err
	}
	if info.Kind != KindProvider {
		return ServiceInfo{}, fmt.Errorf("fake: %w: id=%s kind=%s", ErrNotFound, id, info.Kind)
	}
	return info, nil
}

// ListAgents returns all registered agents.
func (f *FakeRegistry) ListAgents(ctx context.Context) ([]ServiceInfo, error) {
	return f.Discover(ctx, KindAgent)
}

// ListTools returns all registered tools.
func (f *FakeRegistry) ListTools(ctx context.Context) ([]ServiceInfo, error) {
	return f.Discover(ctx, KindTool)
}

// ListProviders returns all registered providers.
func (f *FakeRegistry) ListProviders(ctx context.Context) ([]ServiceInfo, error) {
	return f.Discover(ctx, KindProvider)
}
