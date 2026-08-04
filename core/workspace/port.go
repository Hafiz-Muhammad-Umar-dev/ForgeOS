// Package workspace defines the workspace port abstraction (WorkspacePort) and
// the secret proxy interface (SecretProxy) for the DevOS kernel. It follows the
// ports/adapters (hexagonal) architecture: core/domain uses WorkspacePort, and
// adapters (e.g., local.go) satisfy it without leaking container/K8s SDK types
// into domain code.
//
// Sprint 0 scope:
//   - WorkspacePort interface (Provision, Exec, Status, Recycle)
//   - SecretProxy interface (Resolve)
//   - Workspace / WorkspaceSpec / ExecRequest / ExecResponse types
//   - LocalWorkspace adapter (temp dir + os/exec)
//   - FakeWorkspace / FakeSecretProxy for tests
//
// Excluded from Sprint 0 (deferred to later components):
//   - Docker / Kubernetes / Firecracker adapters
//   - Snapshot / restore
//   - Warm pools
//   - Bus events
//   - Browser automation / DB services / package managers
//   - Secret provider integration (OIDC / vault)
//
// See ADR-004 (Workspace Isolation via Containers/MicroVMs), SDD §06
// (Workspace Manager).
package workspace

import (
	"context"
	"time"
)

// WorkspaceID uniquely identifies a workspace.
type WorkspaceID string

// IsValid reports whether the identifier is non-empty.
func (id WorkspaceID) IsValid() bool { return id != "" }

// String returns the underlying value.
func (id WorkspaceID) String() string { return string(id) }

// WorkspaceStatus represents the lifecycle phase of a workspace.
type WorkspaceStatus string

const (
	WorkspaceStatusUnknown      WorkspaceStatus = ""
	WorkspaceStatusProvisioning WorkspaceStatus = "provisioning"
	WorkspaceStatusReady        WorkspaceStatus = "ready"
	WorkspaceStatusRunning      WorkspaceStatus = "running"
	WorkspaceStatusRecycling    WorkspaceStatus = "recycling"
	WorkspaceStatusRecycled     WorkspaceStatus = "recycled"
	WorkspaceStatusFailed       WorkspaceStatus = "failed"
)

// WorkspaceSpec describes what kind of workspace to provision.
// The Stack field is provider-agnostic — each adapter maps it to its
// own environment (e.g., "node:20" → a Docker image, a Firecracker
// rootfs, or a local toolchain).
type WorkspaceSpec struct {
	// Stack identifies the runtime environment (e.g., "node:20",
	// "go:1.23", "python:3.12"). Adapters use this to select the
	// appropriate base image or toolchain.
	Stack string `json:"stack"`

	// Env is a set of environment variables injected into the workspace.
	Env map[string]string `json:"env,omitempty"`

	// RepoURL is an optional git repository to clone into the workspace.
	RepoURL string `json:"repo_url,omitempty"`

	// Entrypoint is an optional command to run on start.
	Entrypoint string `json:"entrypoint,omitempty"`
}

// Workspace represents a provisioned workspace.
type Workspace struct {
	ID      WorkspaceID     `json:"id"`
	Spec    WorkspaceSpec   `json:"spec"`
	Status  WorkspaceStatus `json:"status"`
	RootDir string          `json:"root_dir"`
	Created time.Time       `json:"created"`
}

// ExecRequest is the input to execute a command inside a workspace.
type ExecRequest struct {
	// Command is the executable name or path.
	Command string `json:"command"`

	// Args are the command arguments.
	Args []string `json:"args,omitempty"`

	// WorkDir is an optional working directory relative to the workspace
	// root. If empty, the workspace root is used.
	WorkDir string `json:"work_dir,omitempty"`

	// Timeout bounds the execution time. Zero means no timeout.
	Timeout time.Duration `json:"timeout,omitempty"`

	// Env overrides or augments the workspace environment for this call.
	Env map[string]string `json:"env,omitempty"`
}

// ExecResponse is the result of a command execution.
type ExecResponse struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
}

// WorkspacePort is the core abstraction for provisioning and interacting
// with isolated workspaces. Implementations provide the execution backend
// (local process, Docker, Kubernetes, Firecracker, etc.).
type WorkspacePort interface {
	// Provision creates a new workspace from the given spec and returns
	// a Workspace handle. The workspace is ready for Exec calls when the
	// returned status is WorkspaceStatusReady.
	Provision(ctx context.Context, spec WorkspaceSpec) (Workspace, error)

	// Exec runs a command inside the workspace and returns the result.
	Exec(ctx context.Context, id WorkspaceID, req ExecRequest) (ExecResponse, error)

	// Status returns the current status of a workspace.
	Status(ctx context.Context, id WorkspaceID) (WorkspaceStatus, error)

	// Recycle destroys the workspace and releases all associated
	// resources. It is safe to call multiple times.
	Recycle(ctx context.Context, id WorkspaceID) error
}

// SecretRef identifies a secret to resolve through the SecretProxy.
type SecretRef struct {
	// Key is the logical name of the secret (e.g., "DATABASE_URL").
	Key string `json:"key"`
}

// ResolvedSecret is the result of resolving a secret reference.
type ResolvedSecret struct {
	// Value is the resolved secret value.
	Value string `json:"value,omitempty"`

	// Exists is true when the secret was found.
	Exists bool `json:"exists"`
}

// SecretProxy resolves secrets at egress time so that agent workspaces
// never see raw secret values. Implementations use short-lived credentials
// and audit every resolution (Constitution T4, T10).
type SecretProxy interface {
	// Resolve resolves a secret reference. Returns a ResolvedSecret with
	// Exists=false when the secret is not found.
	Resolve(ctx context.Context, ref SecretRef) (ResolvedSecret, error)
}
