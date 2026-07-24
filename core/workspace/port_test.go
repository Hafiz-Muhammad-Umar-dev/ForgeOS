package workspace

import (
	"context"
	"strings"
	"testing"
)

func TestWorkspaceID(t *testing.T) {
	id := WorkspaceID("ws-123")
	if !id.IsValid() {
		t.Fatal("should be valid")
	}
	if id.String() != "ws-123" {
		t.Errorf("string: got=%s", id.String())
	}
	empty := WorkspaceID("")
	if empty.IsValid() {
		t.Fatal("empty should not be valid")
	}
}

func TestWorkspaceStatusValues(t *testing.T) {
	if WorkspaceStatusUnknown != "" {
		t.Errorf("unknown=%s", WorkspaceStatusUnknown)
	}
	if WorkspaceStatusProvisioning != "provisioning" {
		t.Errorf("provisioning=%s", WorkspaceStatusProvisioning)
	}
	if WorkspaceStatusReady != "ready" {
		t.Errorf("ready=%s", WorkspaceStatusReady)
	}
	if WorkspaceStatusRunning != "running" {
		t.Errorf("running=%s", WorkspaceStatusRunning)
	}
	if WorkspaceStatusRecycling != "recycling" {
		t.Errorf("recycling=%s", WorkspaceStatusRecycling)
	}
	if WorkspaceStatusRecycled != "recycled" {
		t.Errorf("recycled=%s", WorkspaceStatusRecycled)
	}
	if WorkspaceStatusFailed != "failed" {
		t.Errorf("failed=%s", WorkspaceStatusFailed)
	}
}

func TestWorkspaceSpecStackConfigurable(t *testing.T) {
	spec := WorkspaceSpec{Stack: "node:20"}
	if spec.Stack != "node:20" {
		t.Errorf("stack: got=%s", spec.Stack)
	}
}

func TestWorkspaceWithEnv(t *testing.T) {
	spec := WorkspaceSpec{
		Stack: "go:1.23",
		Env:   map[string]string{"KEY": "value"},
	}
	if spec.Env["KEY"] != "value" {
		t.Errorf("env: got=%v", spec.Env)
	}
}

func TestWorkspaceDefaults(t *testing.T) {
	spec := WorkspaceSpec{Stack: "python:3.12"}
	ws := Workspace{
		ID:   "ws-1",
		Spec: spec,
	}
	if ws.Status != "" {
		t.Errorf("status: got=%s", ws.Status)
	}
	if ws.RootDir != "" {
		t.Errorf("root: got=%s", ws.RootDir)
	}
}

func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		err   error
		label string
	}{
		{ErrNotFound, "ErrNotFound"},
		{ErrNotReady, "ErrNotReady"},
		{ErrAlreadyExists, "ErrAlreadyExists"},
		{ErrExecFailed, "ErrExecFailed"},
		{ErrProvisionFailed, "ErrProvisionFailed"},
		{ErrRecycleFailed, "ErrRecycleFailed"},
		{ErrSecretNotResolved, "ErrSecretNotResolved"},
	}
	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			if tt.err == nil {
				t.Fatal("error is nil")
			}
		})
	}
}

func TestExecRequestDefaults(t *testing.T) {
	req := ExecRequest{Command: "echo"}
	if len(req.Args) != 0 {
		t.Errorf("args: got=%v", req.Args)
	}
	if req.WorkDir != "" {
		t.Errorf("workdir: got=%s", req.WorkDir)
	}
}

func TestExecResponse(t *testing.T) {
	resp := ExecResponse{
		Stdout:   "hello",
		Stderr:   "",
		ExitCode: 0,
	}
	if resp.Stdout != "hello" {
		t.Errorf("stdout: got=%s", resp.Stdout)
	}
	if resp.ExitCode != 0 {
		t.Errorf("exit: got=%d", resp.ExitCode)
	}
}

func TestSecretRef(t *testing.T) {
	ref := SecretRef{Key: "DATABASE_URL"}
	if ref.Key != "DATABASE_URL" {
		t.Errorf("key: got=%s", ref.Key)
	}
}

func TestResolvedSecret(t *testing.T) {
	secret := ResolvedSecret{Value: "postgres://...", Exists: true}
	if !secret.Exists {
		t.Error("should exist")
	}
	if secret.Value != "postgres://..." {
		t.Errorf("value: got=%s", secret.Value)
	}

	missing := ResolvedSecret{Exists: false}
	if missing.Exists {
		t.Error("should not exist")
	}
}

// ---------------------------------------------------------------------------
// FakeWorkspace tests
// ---------------------------------------------------------------------------

func TestFakeWorkspaceProvision(t *testing.T) {
	fw := NewFakeWorkspace()
	ws, err := fw.Provision(context.Background(), WorkspaceSpec{Stack: "node:20"})
	if err != nil {
		t.Fatalf("provision: %v", err)
	}
	if ws.ID == "" {
		t.Fatal("empty id")
	}
	if ws.Spec.Stack != "node:20" {
		t.Errorf("stack: got=%s", ws.Spec.Stack)
	}
	if ws.Status != WorkspaceStatusReady {
		t.Errorf("status: got=%s", ws.Status)
	}
	if fw.ProvisionCount.Load() != 1 {
		t.Errorf("count: got=%d", fw.ProvisionCount.Load())
	}
}

func TestFakeWorkspaceProvisionMultiple(t *testing.T) {
	fw := NewFakeWorkspace()
	ws1, _ := fw.Provision(context.Background(), WorkspaceSpec{Stack: "go:1.23"})
	ws2, _ := fw.Provision(context.Background(), WorkspaceSpec{Stack: "python:3.12"})

	if ws1.ID == ws2.ID {
		t.Error("expected unique ids")
	}
	if len(fw.Workspaces) != 2 {
		t.Errorf("workspaces: got=%d", len(fw.Workspaces))
	}
}

func TestFakeWorkspaceExec(t *testing.T) {
	fw := NewFakeWorkspace()
	ws, _ := fw.Provision(context.Background(), WorkspaceSpec{Stack: "go:1.23"})

	resp, err := fw.Exec(context.Background(), ws.ID, ExecRequest{Command: "go", Args: []string{"version"}})
	if err != nil {
		t.Fatalf("exec: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Errorf("exit: got=%d", resp.ExitCode)
	}
	if !strings.Contains(resp.Stdout, "executed: go") {
		t.Errorf("stdout: got=%s", resp.Stdout)
	}
	if fw.ExecCount.Load() != 1 {
		t.Errorf("count: got=%d", fw.ExecCount.Load())
	}
}

func TestFakeWorkspaceExecNotFound(t *testing.T) {
	fw := NewFakeWorkspace()
	_, err := fw.Exec(context.Background(), "nonexistent", ExecRequest{Command: "ls"})
	if err == nil {
		t.Fatal("expected error for nonexistent workspace")
	}
}

func TestFakeWorkspaceStatus(t *testing.T) {
	fw := NewFakeWorkspace()
	ws, _ := fw.Provision(context.Background(), WorkspaceSpec{Stack: "test"})

	status, err := fw.Status(context.Background(), ws.ID)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status != WorkspaceStatusReady {
		t.Errorf("status: got=%s", status)
	}

	// After exec, status should change to Running
	_, _ = fw.Exec(context.Background(), ws.ID, ExecRequest{Command: "echo"})
	status, err = fw.Status(context.Background(), ws.ID)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status != WorkspaceStatusRunning {
		t.Errorf("status after exec: got=%s", status)
	}
}

func TestFakeWorkspaceStatusNotFound(t *testing.T) {
	fw := NewFakeWorkspace()
	_, err := fw.Status(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFakeWorkspaceRecycle(t *testing.T) {
	fw := NewFakeWorkspace()
	ws, _ := fw.Provision(context.Background(), WorkspaceSpec{Stack: "test"})

	if err := fw.Recycle(context.Background(), ws.ID); err != nil {
		t.Fatalf("recycle: %v", err)
	}
	if len(fw.Workspaces) != 0 {
		t.Errorf("workspaces after recycle: %d", len(fw.Workspaces))
	}

	// Double recycle should error
	err := fw.Recycle(context.Background(), ws.ID)
	if err == nil {
		t.Fatal("expected error on double recycle")
	}
}

func TestFakeWorkspaceCustomProvisionFunc(t *testing.T) {
	fw := NewFakeWorkspace()
	fw.ProvisionFunc = func(ctx context.Context, spec WorkspaceSpec) (Workspace, error) {
		return Workspace{
			ID:      "custom",
			Spec:    spec,
			Status:  WorkspaceStatusReady,
			RootDir: "/custom",
		}, nil
	}

	ws, err := fw.Provision(context.Background(), WorkspaceSpec{Stack: "custom-stack"})
	if err != nil {
		t.Fatalf("provision: %v", err)
	}
	if ws.ID != "custom" {
		t.Errorf("id: got=%s", ws.ID)
	}
	if ws.RootDir != "/custom" {
		t.Errorf("root: got=%s", ws.RootDir)
	}
}

// ---------------------------------------------------------------------------
// FakeSecretProxy tests
// ---------------------------------------------------------------------------

func TestFakeSecretProxyResolve(t *testing.T) {
	fsp := NewFakeSecretProxy(map[string]string{
		"DATABASE_URL": "postgres://localhost",
	})

	secret, err := fsp.Resolve(context.Background(), SecretRef{Key: "DATABASE_URL"})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if !secret.Exists {
		t.Fatal("secret should exist")
	}
	if secret.Value != "postgres://localhost" {
		t.Errorf("value: got=%s", secret.Value)
	}
	if fsp.ResolveCount.Load() != 1 {
		t.Errorf("count: got=%d", fsp.ResolveCount.Load())
	}
}

func TestFakeSecretProxyResolveMissing(t *testing.T) {
	fsp := NewFakeSecretProxy(nil)

	secret, err := fsp.Resolve(context.Background(), SecretRef{Key: "MISSING"})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if secret.Exists {
		t.Fatal("secret should not exist")
	}
	if secret.Value != "" {
		t.Errorf("value: got=%s", secret.Value)
	}
}

func TestFakeSecretProxyManySecrets(t *testing.T) {
	fsp := NewFakeSecretProxy(map[string]string{
		"API_KEY":     "sk-123",
		"DATABASE":    "pg://localhost",
		"SECRET_KEY":  "super-secret",
	})

	for key, expected := range map[string]string{
		"API_KEY": "sk-123", "DATABASE": "pg://localhost", "SECRET_KEY": "super-secret",
	} {
		secret, err := fsp.Resolve(context.Background(), SecretRef{Key: key})
		if err != nil {
			t.Fatalf("resolve %s: %v", key, err)
		}
		if !secret.Exists {
			t.Errorf("%s: not found", key)
		}
		if secret.Value != expected {
			t.Errorf("%s: got=%s want=%s", key, secret.Value, expected)
		}
	}
}
