package workspace

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
)

// Compile-time check: *FakeWorkspace implements WorkspacePort.
var _ WorkspacePort = (*FakeWorkspace)(nil)

// FakeWorkspace is an in-memory WorkspacePort implementation for testing.
// It records all requests and returns configurable responses.
type FakeWorkspace struct {
	// ProvisionFunc overrides the Provision behavior. If nil, a default
	// implementation using Specs is used.
	ProvisionFunc func(ctx context.Context, spec WorkspaceSpec) (Workspace, error)

	// ExecFunc overrides the Exec behavior. If nil, a default
	// implementation using Responses is used.
	ExecFunc func(ctx context.Context, id WorkspaceID, req ExecRequest) (ExecResponse, error)

	// StatusFunc overrides the Status behavior. If nil, a default
	// implementation is used.
	StatusFunc func(ctx context.Context, id WorkspaceID) (WorkspaceStatus, error)

	// RecycleFunc overrides the Recycle behavior. If nil, a default
	// implementation is used.
	RecycleFunc func(ctx context.Context, id WorkspaceID) error

	// Workspaces holds the provisioned workspaces, keyed by ID.
	Workspaces map[WorkspaceID]Workspace

	// ProvisionCount tracks the number of Provision calls.
	ProvisionCount atomic.Int64

	// ExecCount tracks the number of Exec calls.
	ExecCount atomic.Int64

	// nextID is used to generate deterministic workspace IDs.
	nextID int64

	mu sync.Mutex
}

// NewFakeWorkspace creates a new FakeWorkspace.
func NewFakeWorkspace() *FakeWorkspace {
	return &FakeWorkspace{
		Workspaces: make(map[WorkspaceID]Workspace),
	}
}

// Provision records a provisioning request. Returns the workspace or an error.
func (f *FakeWorkspace) Provision(ctx context.Context, spec WorkspaceSpec) (Workspace, error) {
	f.ProvisionCount.Add(1)

	if f.ProvisionFunc != nil {
		return f.ProvisionFunc(ctx, spec)
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	id := WorkspaceID(fmt.Sprintf("fake-ws-%d", f.nextID))
	f.nextID++

	ws := Workspace{
		ID:      id,
		Spec:    spec,
		Status:  WorkspaceStatusReady,
		RootDir: "/tmp/fake-" + string(id),
	}
	f.Workspaces[id] = ws
	return ws, nil
}

// Exec records and simulates command execution.
func (f *FakeWorkspace) Exec(ctx context.Context, id WorkspaceID, req ExecRequest) (ExecResponse, error) {
	f.ExecCount.Add(1)

	if f.ExecFunc != nil {
		return f.ExecFunc(ctx, id, req)
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if _, ok := f.Workspaces[id]; !ok {
		return ExecResponse{}, fmt.Errorf("fake: %w", ErrNotFound)
	}

	// Update status to Running if it was Ready.
	if ws := f.Workspaces[id]; ws.Status == WorkspaceStatusReady {
		ws.Status = WorkspaceStatusRunning
		f.Workspaces[id] = ws
	}

	return ExecResponse{
		Stdout:   fmt.Sprintf("executed: %s %v", req.Command, req.Args),
		ExitCode: 0,
	}, nil
}

// Status returns the workspace status.
func (f *FakeWorkspace) Status(ctx context.Context, id WorkspaceID) (WorkspaceStatus, error) {
	if f.StatusFunc != nil {
		return f.StatusFunc(ctx, id)
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	ws, ok := f.Workspaces[id]
	if !ok {
		return WorkspaceStatusUnknown, fmt.Errorf("fake: %w", ErrNotFound)
	}
	return ws.Status, nil
}

// Recycle removes the workspace.
func (f *FakeWorkspace) Recycle(ctx context.Context, id WorkspaceID) error {
	if f.RecycleFunc != nil {
		return f.RecycleFunc(ctx, id)
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if _, ok := f.Workspaces[id]; !ok {
		return fmt.Errorf("fake: %w", ErrNotFound)
	}
	delete(f.Workspaces, id)
	return nil
}

// ---------------------------------------------------------------------------
// FakeSecretProxy
// ---------------------------------------------------------------------------

// FakeSecretProxy is an in-memory SecretProxy implementation for testing.
type FakeSecretProxy struct {
	// Secrets is the backing store of resolvable secrets.
	Secrets map[string]string

	// ResolveCount tracks the number of Resolve calls.
	ResolveCount atomic.Int64

	mu sync.RWMutex
}

// NewFakeSecretProxy creates a FakeSecretProxy with the given secrets.
func NewFakeSecretProxy(secrets map[string]string) *FakeSecretProxy {
	if secrets == nil {
		secrets = make(map[string]string)
	}
	return &FakeSecretProxy{Secrets: secrets}
}

// Resolve looks up a secret by key.
func (f *FakeSecretProxy) Resolve(ctx context.Context, ref SecretRef) (ResolvedSecret, error) {
	f.ResolveCount.Add(1)
	f.mu.RLock()
	val, ok := f.Secrets[ref.Key]
	f.mu.RUnlock()
	if !ok {
		return ResolvedSecret{Exists: false}, nil
	}
	return ResolvedSecret{Value: val, Exists: true}, nil
}
