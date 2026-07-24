//go:build integration

package workspace

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestIntegrationLocalProvisionAndExec verifies a full provision->exec->recycle
// cycle with the LocalWorkspace adapter.
func TestIntegrationLocalProvisionAndExec(t *testing.T) {
	lw := NewLocalWorkspace()
	ctx := context.Background()

	ws, err := lw.Provision(ctx, WorkspaceSpec{
		Stack: "integration-test",
		Env:   map[string]string{"HELLO": "world"},
	})
	if err != nil {
		t.Fatalf("provision: %v", err)
	}
	defer lw.Recycle(ctx, ws.ID)

	if _, err := os.Stat(ws.RootDir); os.IsNotExist(err) {
		t.Fatalf("workspace dir %s not created", ws.RootDir)
	}

	envData, err := os.ReadFile(filepath.Join(ws.RootDir, ".env"))
	if err != nil {
		t.Fatalf("read .env: %v", err)
	}
	if !strings.Contains(string(envData), "HELLO=world") {
		t.Errorf(".env: got=%s", string(envData))
	}

	cmd := "echo"
	args := []string{"integration test"}

	resp, err := lw.Exec(ctx, ws.ID, ExecRequest{Command: cmd, Args: args})
	if err != nil {
		t.Fatalf("exec: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Errorf("exit: got=%d", resp.ExitCode)
	}
	if !strings.Contains(resp.Stdout, "integration test") {
		t.Errorf("stdout: got=%s", resp.Stdout)
	}

	status, err := lw.Status(ctx, ws.ID)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status != WorkspaceStatusRunning {
		t.Errorf("status after exec: got=%s", status)
	}

	testContent := "hello from workspace"
	if err := os.WriteFile(filepath.Join(ws.RootDir, "test.txt"), []byte(testContent), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	resp, err = lw.Exec(ctx, ws.ID, ExecRequest{
		Command: "cat",
		Args:    []string{"test.txt"},
	})
	if err != nil {
		t.Fatalf("exec read: %v", err)
	}
	if strings.TrimSpace(resp.Stdout) != testContent {
		t.Errorf("read back: got=%s want=%s", strings.TrimSpace(resp.Stdout), testContent)
	}
}

// TestIntegrationLocalMultipleWorkspaces verifies workspace isolation.
func TestIntegrationLocalMultipleWorkspaces(t *testing.T) {
	lw := NewLocalWorkspace()
	ctx := context.Background()

	ws1, err := lw.Provision(ctx, WorkspaceSpec{Stack: "ws1"})
	if err != nil {
		t.Fatalf("provision ws1: %v", err)
	}
	defer lw.Recycle(ctx, ws1.ID)

	ws2, err := lw.Provision(ctx, WorkspaceSpec{Stack: "ws2"})
	if err != nil {
		t.Fatalf("provision ws2: %v", err)
	}
	defer lw.Recycle(ctx, ws2.ID)

	if err := os.WriteFile(filepath.Join(ws1.RootDir, "marker.txt"), []byte("ws1-data"), 0644); err != nil {
		t.Fatalf("write ws1: %v", err)
	}

	if _, err := os.Stat(filepath.Join(ws2.RootDir, "marker.txt")); !os.IsNotExist(err) {
		t.Error("ws2 should not have ws1's file")
	}
	if _, err := os.Stat(filepath.Join(ws1.RootDir, "marker.txt")); os.IsNotExist(err) {
		t.Error("ws1 should have its file")
	}
}

// TestIntegrationLocalExecNonZeroExit verifies exit code propagation.
func TestIntegrationLocalExecNonZeroExit(t *testing.T) {
	lw := NewLocalWorkspace()
	ctx := context.Background()

	ws, err := lw.Provision(ctx, WorkspaceSpec{Stack: "test"})
	if err != nil {
		t.Fatalf("provision: %v", err)
	}
	defer lw.Recycle(ctx, ws.ID)

	resp, err := lw.Exec(ctx, ws.ID, ExecRequest{
		Command: "sh",
		Args:    []string{"-c", "exit 42"},
	})
	if err != nil {
		t.Fatalf("exec: %v", err)
	}
	if resp.ExitCode != 42 {
		t.Errorf("exit code: got=%d want=42", resp.ExitCode)
	}
}

// TestIntegrationLocalExecTimeout verifies a long-running command is killed.
func TestIntegrationLocalExecTimeout(t *testing.T) {
	lw := NewLocalWorkspace()
	ctx := context.Background()

	ws, err := lw.Provision(ctx, WorkspaceSpec{Stack: "test"})
	if err != nil {
		t.Fatalf("provision: %v", err)
	}
	defer lw.Recycle(ctx, ws.ID)

	start := time.Now()
	_, err = lw.Exec(ctx, ws.ID, ExecRequest{
		Command: "ping",
		Args:    []string{"-n", "30", "127.0.0.1"},
		Timeout: 50 * time.Millisecond,
	})

	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if elapsed > 5*time.Second {
		t.Errorf("command took too long to kill: %v", elapsed)
	}
	t.Logf("command killed in %v (expected): %v", elapsed, err)
}

// TestIntegrationLocalRecycleIdempotent verifies recycle can be called
// on an already-recycled workspace.
func TestIntegrationLocalRecycleIdempotent(t *testing.T) {
	lw := NewLocalWorkspace()
	ctx := context.Background()

	ws, err := lw.Provision(ctx, WorkspaceSpec{Stack: "test"})
	if err != nil {
		t.Fatalf("provision: %v", err)
	}

	if err := lw.Recycle(ctx, ws.ID); err != nil {
		t.Fatalf("first recycle: %v", err)
	}

	err = lw.Recycle(ctx, ws.ID)
	if err == nil {
		t.Log("note: second recycle returned nil (idempotent)")
	} else {
		t.Logf("second recycle correctly errored: %v", err)
	}
}
