package registry

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Existing type tests
// ---------------------------------------------------------------------------

func TestServiceInfoHas(t *testing.T) {
	info := ServiceInfo{Capabilities: []Capability{"llm", "embed"}}
	if !info.Has("llm") {
		t.Error("should have llm")
	}
	if info.Has("vector") {
		t.Error("should not have vector")
	}
}

func TestCapabilitySet(t *testing.T) {
	s := NewCapabilitySet("a", "b", "a")
	if len(s) != 2 {
		t.Fatalf("len=%d", len(s))
	}
	if !s.Contains("a") || !s.Contains("b") {
		t.Error("missing")
	}
	if s.Contains("c") {
		t.Error("extra")
	}
	if len(s.Slice()) != 2 {
		t.Fatalf("slice len=%d", len(s.Slice()))
	}
}

// ---------------------------------------------------------------------------
// memRegistry test double (updated for the extended Registry interface)
// ---------------------------------------------------------------------------

type memRegistry struct {
	store map[string]ServiceInfo
	mu    sync.RWMutex
}

func newMem() *memRegistry { return &memRegistry{store: map[string]ServiceInfo{}} }

func (m *memRegistry) Register(_ context.Context, info ServiceInfo) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.store[info.ID]; exists {
		return fmt.Errorf("%w: id=%s", ErrAlreadyRegistered, info.ID)
	}
	m.store[info.ID] = info
	return nil
}
func (m *memRegistry) Deregister(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.store, id)
	return nil
}
func (m *memRegistry) Discover(_ context.Context, kind string) ([]ServiceInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []ServiceInfo
	for _, v := range m.store {
		if v.Kind == kind {
			out = append(out, v)
		}
	}
	return out, nil
}
func (m *memRegistry) Resolve(_ context.Context, cap Capability) (ServiceInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, v := range m.store {
		if v.Has(cap) {
			return v, nil
		}
	}
	return ServiceInfo{}, nil
}
func (m *memRegistry) ResolveByID(_ context.Context, id string) (ServiceInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.store[id], nil
}
func (m *memRegistry) Watch(_ context.Context, _ Capability) (<-chan ServiceInfo, error) {
	ch := make(chan ServiceInfo)
	close(ch)
	return ch, nil
}
func (m *memRegistry) RegisterAgent(ctx context.Context, info ServiceInfo) error {
	if info.Kind == "" {
		info.Kind = KindAgent
	}
	return m.Register(ctx, info)
}
func (m *memRegistry) RegisterTool(ctx context.Context, info ServiceInfo) error {
	if info.Kind == "" {
		info.Kind = KindTool
	}
	return m.Register(ctx, info)
}
func (m *memRegistry) RegisterProvider(ctx context.Context, info ServiceInfo) error {
	if info.Kind == "" {
		info.Kind = KindProvider
	}
	return m.Register(ctx, info)
}
func (m *memRegistry) GetAgent(ctx context.Context, id string) (ServiceInfo, error) {
	info, err := m.ResolveByID(ctx, id)
	return info, err
}
func (m *memRegistry) GetTool(ctx context.Context, id string) (ServiceInfo, error) {
	info, err := m.ResolveByID(ctx, id)
	return info, err
}
func (m *memRegistry) GetProvider(ctx context.Context, id string) (ServiceInfo, error) {
	info, err := m.ResolveByID(ctx, id)
	return info, err
}
func (m *memRegistry) ListAgents(ctx context.Context) ([]ServiceInfo, error) {
	return m.Discover(ctx, KindAgent)
}
func (m *memRegistry) ListTools(ctx context.Context) ([]ServiceInfo, error) {
	return m.Discover(ctx, KindTool)
}
func (m *memRegistry) ListProviders(ctx context.Context) ([]ServiceInfo, error) {
	return m.Discover(ctx, KindProvider)
}

var _ Registry = (*memRegistry)(nil)

func TestRegistryFake(t *testing.T) {
	r := newMem()
	ctx := context.Background()
	if err := r.Register(ctx, ServiceInfo{ID: "1", Kind: "agent", Capabilities: []Capability{"plan"}}); err != nil {
		t.Fatal(err)
	}
	got, err := r.Resolve(ctx, "plan")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "1" {
		t.Errorf("resolved=%s", got.ID)
	}
	disc, _ := r.Discover(ctx, "agent")
	if len(disc) != 1 {
		t.Errorf("discover=%d", len(disc))
	}
}

// ---------------------------------------------------------------------------
// InMemoryRegistry tests
// ---------------------------------------------------------------------------

func newTestRegistry() *InMemoryRegistry {
	return NewInMemoryRegistry()
}

func TestRegisterAndResolve(t *testing.T) {
	r := newTestRegistry()
	ctx := context.Background()

	err := r.Register(ctx, ServiceInfo{
		ID: "agent-1", Name: "Test Agent", Kind: KindAgent,
		Capabilities: []Capability{"code.write"},
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	got, err := r.Resolve(ctx, "code.write")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got.ID != "agent-1" {
		t.Errorf("id: got=%s", got.ID)
	}
}

func TestRegisterDuplicate(t *testing.T) {
	r := newTestRegistry()
	ctx := context.Background()

	info := ServiceInfo{ID: "dup", Name: "Duplicate", Kind: KindAgent}
	if err := r.Register(ctx, info); err != nil {
		t.Fatalf("first register: %v", err)
	}
	err := r.Register(ctx, info)
	if err == nil {
		t.Fatal("expected error for duplicate")
	}
}

func TestRegisterAndResolveByID(t *testing.T) {
	r := newTestRegistry()
	ctx := context.Background()

	info := ServiceInfo{ID: "agent-1", Kind: KindAgent, Name: "Agent One"}
	r.Register(ctx, info)

	got, err := r.ResolveByID(ctx, "agent-1")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got.Name != "Agent One" {
		t.Errorf("name: got=%s", got.Name)
	}
}

func TestResolveByIDNotFound(t *testing.T) {
	r := newTestRegistry()
	_, err := r.ResolveByID(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent")
	}
}

func TestDeregister(t *testing.T) {
	r := newTestRegistry()
	ctx := context.Background()

	r.Register(ctx, ServiceInfo{ID: "to-remove", Kind: KindAgent})
	if err := r.Deregister(ctx, "to-remove"); err != nil {
		t.Fatalf("deregister: %v", err)
	}

	_, err := r.ResolveByID(ctx, "to-remove")
	if err == nil {
		t.Fatal("expected error after deregister")
	}
}

func TestDeregisterNotFound(t *testing.T) {
	r := newTestRegistry()
	err := r.Deregister(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDiscoverByKind(t *testing.T) {
	r := newTestRegistry()
	ctx := context.Background()

	r.Register(ctx, ServiceInfo{ID: "a1", Kind: KindAgent, Name: "Agent A"})
	r.Register(ctx, ServiceInfo{ID: "a2", Kind: KindAgent, Name: "Agent B"})
	r.Register(ctx, ServiceInfo{ID: "t1", Kind: KindTool, Name: "Tool A"})

	agents, _ := r.Discover(ctx, KindAgent)
	if len(agents) != 2 {
		t.Errorf("agents: got=%d want=2", len(agents))
	}

	tools, _ := r.Discover(ctx, KindTool)
	if len(tools) != 1 {
		t.Errorf("tools: got=%d want=1", len(tools))
	}

	providers, _ := r.Discover(ctx, KindProvider)
	if len(providers) != 0 {
		t.Errorf("providers: got=%d want=0", len(providers))
	}
}

func TestResolveCapabilityNotFound(t *testing.T) {
	r := newTestRegistry()
	_, err := r.Resolve(context.Background(), "nonexistent.cap")
	if err == nil {
		t.Fatal("expected error")
	}
}

// ---------------------------------------------------------------------------
// Typed method tests
// ---------------------------------------------------------------------------

func TestRegisterAgent(t *testing.T) {
	r := newTestRegistry()
	ctx := context.Background()

	err := r.RegisterAgent(ctx, ServiceInfo{ID: "agent-1", Name: "Coder", Capabilities: []Capability{"code.write"}})
	if err != nil {
		t.Fatalf("register agent: %v", err)
	}

	info, err := r.GetAgent(ctx, "agent-1")
	if err != nil {
		t.Fatalf("get agent: %v", err)
	}
	if info.Kind != KindAgent {
		t.Errorf("kind: got=%s", info.Kind)
	}

	agents, err := r.ListAgents(ctx)
	if err != nil {
		t.Fatalf("list agents: %v", err)
	}
	if len(agents) != 1 {
		t.Errorf("agents: got=%d", len(agents))
	}
}

func TestRegisterTool(t *testing.T) {
	r := newTestRegistry()
	ctx := context.Background()

	err := r.RegisterTool(ctx, ServiceInfo{ID: "tool-1", Name: "ExecuteCommand"})
	if err != nil {
		t.Fatalf("register tool: %v", err)
	}

	info, err := r.GetTool(ctx, "tool-1")
	if err != nil {
		t.Fatalf("get tool: %v", err)
	}
	if info.Kind != KindTool {
		t.Errorf("kind: got=%s", info.Kind)
	}

	tools, err := r.ListTools(ctx)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	if len(tools) != 1 {
		t.Errorf("tools: got=%d", len(tools))
	}
}

func TestRegisterProvider(t *testing.T) {
	r := newTestRegistry()
	ctx := context.Background()

	err := r.RegisterProvider(ctx, ServiceInfo{ID: "provider-1", Name: "Claude", Capabilities: []Capability{"llm"}})
	if err != nil {
		t.Fatalf("register provider: %v", err)
	}

	info, err := r.GetProvider(ctx, "provider-1")
	if err != nil {
		t.Fatalf("get provider: %v", err)
	}
	if info.Kind != KindProvider {
		t.Errorf("kind: got=%s", info.Kind)
	}

	providers, err := r.ListProviders(ctx)
	if err != nil {
		t.Fatalf("list providers: %v", err)
	}
	if len(providers) != 1 {
		t.Errorf("providers: got=%d", len(providers))
	}
}

func TestGetAgentNotFound(t *testing.T) {
	r := newTestRegistry()
	_, err := r.GetAgent(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetAgentWrongKind(t *testing.T) {
	r := newTestRegistry()
	ctx := context.Background()

	r.Register(ctx, ServiceInfo{ID: "tool-1", Kind: KindTool, Name: "A Tool"})
	_, err := r.GetAgent(ctx, "tool-1")
	if err == nil {
		t.Fatal("expected error for wrong kind")
	}
}

func TestRegisterAgentWithInvalidKind(t *testing.T) {
	r := newTestRegistry()
	err := r.RegisterAgent(context.Background(), ServiceInfo{ID: "bad", Kind: KindTool})
	if err == nil {
		t.Fatal("expected error for invalid kind")
	}
}

// ---------------------------------------------------------------------------
// Concurrent access tests
// ---------------------------------------------------------------------------

func TestConcurrentRegistration(t *testing.T) {
	r := newTestRegistry()
	ctx := context.Background()

	const n = 50
	errs := make(chan error, n)
	ids := make([]string, n)
	for i := 0; i < n; i++ {
		ids[i] = fmt.Sprintf("agent-%d", i)
	}

	for i := 0; i < n; i++ {
		id := ids[i]
		go func() {
			errs <- r.Register(ctx, ServiceInfo{ID: id, Kind: KindAgent, Name: "Agent " + id})
		}()
	}

	for i := 0; i < n; i++ {
		if err := <-errs; err != nil {
			t.Errorf("concurrent register %d: %v", i, err)
		}
	}

	agents, _ := r.ListAgents(ctx)
	if len(agents) != n {
		t.Errorf("agents: got=%d want=%d", len(agents), n)
	}
}

func TestConcurrentLookup(t *testing.T) {
	r := newTestRegistry()
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		id := fmt.Sprintf("agent-%d", i)
		r.Register(ctx, ServiceInfo{ID: id, Kind: KindAgent})
	}

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			id := fmt.Sprintf("agent-%d", n%10)
			_, err := r.ResolveByID(ctx, id)
			if err != nil {
				t.Errorf("lookup %s: %v", id, err)
			}
		}(i)
	}
	wg.Wait()
}

func TestConcurrentRegisterAndDeregister(t *testing.T) {
	r := newTestRegistry()
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			id := fmt.Sprintf("agent-%d", n)
			_ = r.Register(ctx, ServiceInfo{ID: id, Kind: KindAgent})
			_, _ = r.ResolveByID(ctx, id)
			_ = r.Deregister(ctx, id)
		}(i)
	}
	wg.Wait()
}

// ---------------------------------------------------------------------------
// Lifecycle tests
// ---------------------------------------------------------------------------

func TestInMemoryLifecycle(t *testing.T) {
	r := NewInMemoryRegistry()

	if r.Name() != "registry" {
		t.Errorf("name=%s", r.Name())
	}
	if err := r.Init(context.Background()); err != nil {
		t.Fatalf("init: %v", err)
	}

	health := r.Health()
	if health.Status == "UP" {
		t.Log("health before start:", health.Status)
	}

	if err := r.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	health = r.Health()
	if health.Status != "UP" {
		t.Errorf("health after start: got=%s", health.Status)
	}

	if err := r.Stop(context.Background()); err != nil {
		t.Fatalf("stop: %v", err)
	}
	health = r.Health()
	if health.Status == "UP" {
		t.Errorf("health after stop: got=%s", health.Status)
	}
}

// ---------------------------------------------------------------------------
// FakeRegistry tests
// ---------------------------------------------------------------------------

func TestFakeRegistryRegister(t *testing.T) {
	fr := NewFakeRegistry()
	ctx := context.Background()

	err := fr.Register(ctx, ServiceInfo{ID: "test-1", Kind: KindAgent})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if fr.RegisterCount.Load() != 1 {
		t.Errorf("count: got=%d", fr.RegisterCount.Load())
	}
}

func TestFakeRegistryDuplicate(t *testing.T) {
	fr := NewFakeRegistry()
	ctx := context.Background()

	fr.Register(ctx, ServiceInfo{ID: "dup", Kind: KindAgent})
	err := fr.Register(ctx, ServiceInfo{ID: "dup", Kind: KindAgent})
	if err == nil {
		t.Fatal("expected duplicate error")
	}
}

func TestFakeRegistryDiscover(t *testing.T) {
	fr := NewFakeRegistry()
	ctx := context.Background()

	fr.RegisterAgent(ctx, ServiceInfo{ID: "a1", Name: "Agent A"})
	fr.RegisterAgent(ctx, ServiceInfo{ID: "a2", Name: "Agent B"})
	fr.RegisterTool(ctx, ServiceInfo{ID: "t1", Name: "Tool T"})

	agents, _ := fr.ListAgents(ctx)
	if len(agents) != 2 {
		t.Errorf("agents: got=%d", len(agents))
	}

	tools, _ := fr.ListTools(ctx)
	if len(tools) != 1 {
		t.Errorf("tools: got=%d", len(tools))
	}
}

func TestFakeRegistryDeregister(t *testing.T) {
	fr := NewFakeRegistry()
	ctx := context.Background()

	fr.Register(ctx, ServiceInfo{ID: "r1", Kind: KindAgent})
	if err := fr.Deregister(ctx, "r1"); err != nil {
		t.Fatalf("deregister: %v", err)
	}
	if fr.DeregisterCount.Load() != 1 {
		t.Errorf("count: got=%d", fr.DeregisterCount.Load())
	}

	_, err := fr.ResolveByID(ctx, "r1")
	if err == nil {
		t.Fatal("expected error after deregister")
	}
}

func TestFakeRegistryGetAgent(t *testing.T) {
	fr := NewFakeRegistry()
	ctx := context.Background()

	fr.RegisterAgent(ctx, ServiceInfo{ID: "agent-1", Name: "Coder"})
	info, err := fr.GetAgent(ctx, "agent-1")
	if err != nil {
		t.Fatalf("get agent: %v", err)
	}
	if info.Name != "Coder" {
		t.Errorf("name: got=%s", info.Name)
	}
}

func TestFakeRegistryGetAgentWrongKind(t *testing.T) {
	fr := NewFakeRegistry()
	ctx := context.Background()

	fr.RegisterTool(ctx, ServiceInfo{ID: "tool-id", Name: "A Tool"})
	_, err := fr.GetAgent(ctx, "tool-id")
	if err == nil {
		t.Fatal("expected error for wrong kind")
	}
}

func TestFakeRegistryRegisteredAt(t *testing.T) {
	fr := NewFakeRegistry()
	ctx := context.Background()

	before := time.Now()
	fr.RegisterAgent(ctx, ServiceInfo{ID: "a1", Name: "Agent A"})

	info, _ := fr.GetAgent(ctx, "a1")
	if info.RegisteredAt.IsZero() {
		t.Error("RegisteredAt should not be zero")
	}
	if info.RegisteredAt.Before(before) {
		t.Error("RegisteredAt should be after 'before'")
	}
}

func TestFakeRegistryWatch(t *testing.T) {
	fr := NewFakeRegistry()
	ch, err := fr.Watch(context.Background(), "test")
	if err != nil {
		t.Fatalf("watch: %v", err)
	}
	// Should be closed immediately
	_, ok := <-ch
	if ok {
		t.Error("expected closed channel")
	}
}
