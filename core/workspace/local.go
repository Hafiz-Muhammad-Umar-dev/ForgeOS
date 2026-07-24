package workspace

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Compile-time check: *LocalWorkspace implements WorkspacePort.
var _ WorkspacePort = (*LocalWorkspace)(nil)

// LocalWorkspace is a WorkspacePort adapter that provisions workspaces as
// temporary directories on the local filesystem and executes commands via
// os/exec. It is designed for development, testing, and Sprint 0 — not for
// production isolation.
//
// Future adapters (Docker, Kubernetes, Firecracker) should satisfy the same
// WorkspacePort interface without changing consumers.
type LocalWorkspace struct {
	mu    sync.Mutex
	dirs  map[WorkspaceID]Workspace
	base  string
	nowFn func() time.Time
}

// LocalOption configures the LocalWorkspace adapter.
type LocalOption func(*LocalWorkspace)

// WithBaseDir sets the parent directory for workspace temp dirs.
func WithBaseDir(dir string) LocalOption {
	return func(l *LocalWorkspace) { l.base = dir }
}

// NewLocalWorkspace creates a new LocalWorkspace adapter.
func NewLocalWorkspace(opts ...LocalOption) *LocalWorkspace {
	lw := &LocalWorkspace{
		dirs:  make(map[WorkspaceID]Workspace),
		nowFn: time.Now,
	}
	for _, fn := range opts {
		fn(lw)
	}
	return lw
}

// Provision creates a new workspace as a temporary directory.
func (l *LocalWorkspace) Provision(ctx context.Context, spec WorkspaceSpec) (Workspace, error) {
	if spec.Stack == "" {
		return Workspace{}, fmt.Errorf("local: %w: stack is required", ErrProvisionFailed)
	}

	id, err := newWorkspaceID()
	if err != nil {
		return Workspace{}, fmt.Errorf("local: %w: generate id: %w", ErrProvisionFailed, err)
	}

	parentDir := l.base
	if parentDir == "" {
		parentDir = os.TempDir()
	}

	rootDir, err := os.MkdirTemp(parentDir, "devos-workspace-*")
	if err != nil {
		return Workspace{}, fmt.Errorf("local: %w: create dir: %w", ErrProvisionFailed, err)
	}

	if len(spec.Env) > 0 {
		if err := writeEnvFile(rootDir, spec.Env); err != nil {
			os.RemoveAll(rootDir)
			return Workspace{}, fmt.Errorf("local: %w: write env: %w", ErrProvisionFailed, err)
		}
	}

	if spec.RepoURL != "" {
		cloneDir := filepath.Join(rootDir, "repo")
		if err := cloneRepo(ctx, spec.RepoURL, cloneDir); err != nil {
			os.RemoveAll(rootDir)
			return Workspace{}, fmt.Errorf("local: %w: clone repo: %w", ErrProvisionFailed, err)
		}
	}

	ws := Workspace{
		ID:      id,
		Spec:    spec,
		Status:  WorkspaceStatusReady,
		RootDir: rootDir,
		Created: l.nowFn(),
	}

	l.mu.Lock()
	l.dirs[id] = ws
	l.mu.Unlock()

	return ws, nil
}

// Exec runs a command inside the workspace directory.
func (l *LocalWorkspace) Exec(ctx context.Context, id WorkspaceID, req ExecRequest) (ExecResponse, error) {
	l.mu.Lock()
	ws, ok := l.dirs[id]
	l.mu.Unlock()

	if !ok {
		return ExecResponse{}, fmt.Errorf("local: %w: id=%s", ErrNotFound, id)
	}
	if ws.Status != WorkspaceStatusReady && ws.Status != WorkspaceStatusRunning {
		return ExecResponse{}, fmt.Errorf("local: %w: id=%s status=%s", ErrNotReady, id, ws.Status)
	}

	workDir := ws.RootDir
	if req.WorkDir != "" {
		workDir = filepath.Join(ws.RootDir, req.WorkDir)
	}

	cmdCtx := ctx
	if req.Timeout > 0 {
		var cancel context.CancelFunc
		cmdCtx, cancel = context.WithTimeout(ctx, req.Timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(cmdCtx, req.Command, req.Args...)
	cmd.Dir = workDir

	cmd.Env = os.Environ()
	for k, v := range ws.Spec.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	for k, v := range req.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	stdout, stderr := new(strings.Builder), new(strings.Builder)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err := cmd.Run()

	// Check for context cancellation before ExitError handling.
	// On some platforms a killed process still reports an ExitError,
	// so we check the context first to distinguish "timed out" from
	// "command exited with non-zero code".
	if err != nil {
		if cmdCtx.Err() != nil {
			return ExecResponse{}, fmt.Errorf("local: %w: %w", ErrExecFailed, cmdCtx.Err())
		}
	}

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
			err = nil
		}
	}

	if ws.Status == WorkspaceStatusReady {
		ws.Status = WorkspaceStatusRunning
		l.mu.Lock()
		l.dirs[id] = ws
		l.mu.Unlock()
	}

	resp := ExecResponse{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
	}

	if err != nil {
		return resp, fmt.Errorf("local: %w: %w", ErrExecFailed, err)
	}
	return resp, nil
}

// Status returns the current status of a workspace.
func (l *LocalWorkspace) Status(ctx context.Context, id WorkspaceID) (WorkspaceStatus, error) {
	l.mu.Lock()
	ws, ok := l.dirs[id]
	l.mu.Unlock()

	if !ok {
		return WorkspaceStatusUnknown, fmt.Errorf("local: %w: id=%s", ErrNotFound, id)
	}

	if _, err := os.Stat(ws.RootDir); os.IsNotExist(err) {
		return WorkspaceStatusRecycled, nil
	}

	return ws.Status, nil
}

// Recycle removes the workspace directory and its metadata.
func (l *LocalWorkspace) Recycle(ctx context.Context, id WorkspaceID) error {
	l.mu.Lock()
	ws, ok := l.dirs[id]
	if !ok {
		l.mu.Unlock()
		return fmt.Errorf("local: %w: id=%s", ErrNotFound, id)
	}
	delete(l.dirs, id)
	l.mu.Unlock()

	if err := os.RemoveAll(ws.RootDir); err != nil {
		return fmt.Errorf("local: %w: remove %s: %w", ErrRecycleFailed, ws.RootDir, err)
	}
	return nil
}

func newWorkspaceID() (WorkspaceID, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return WorkspaceID(fmt.Sprintf("ws-%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])), nil
}

func writeEnvFile(dir string, env map[string]string) error {
	var sb strings.Builder
	for k, v := range env {
		sb.WriteString(k)
		sb.WriteString("=")
		sb.WriteString(v)
		sb.WriteString("\n")
	}
	return os.WriteFile(filepath.Join(dir, ".env"), []byte(sb.String()), 0644)
}

func cloneRepo(ctx context.Context, repoURL, cloneDir string) error {
	cmd := exec.CommandContext(ctx, "git", "clone", repoURL, cloneDir)
	cmd.Stderr = new(strings.Builder)
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := exitErr.Stderr
			if len(stderr) == 0 && cmd.Stderr != nil {
				stderr = []byte(cmd.Stderr.(*strings.Builder).String())
			}
			return fmt.Errorf("git clone: exit %d: %s", exitErr.ExitCode(), string(stderr))
		}
		return err
	}
	return nil
}
