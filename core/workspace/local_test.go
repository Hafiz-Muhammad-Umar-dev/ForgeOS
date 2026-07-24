package workspace

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLocalProvision(t *testing.T) {
	lw := NewLocalWorkspace()
	ws, err := lw.Provision(context.Background(), WorkspaceSpec{Stack: "test"})
	if err != nil {
		t.Fatalf("provision: %v", err)
	}
	defer lw.Recycle(context.Background(), ws.ID)

	if ws.ID == "" {
		t.Fatal("empty id")
	}
	if ws.Spec.Stack != "test" {
		t.Errorf("stack: got=%s", ws.Spec.Stack)
	}
	if ws.Status != WorkspaceStatusReady {
		t.Errorf("status: got=%s", ws.Status)
	}
	if ws.RootDir == "" {
		t.Fatal("empty root dir")
	}

	// Verify directory exists
	if _, err := os.Stat(ws.RootDir); os.IsNotExist(err) {
		t.Fatalf("root dir %s does not exist", ws.RootDir)
	}
}

func TestLocalProvisionEmptyStack(t *testing.T) {
	lw := NewLocalWorkspace()
	_, err := lw.Provision(context.Background(), WorkspaceSpec{Stack: ""})
	if err == nil {
		t.Fatal("expected error for empty stack")
	}
}

func TestLocalProvisionWithBaseDir(t *testing.T) {
	baseDir := t.TempDir()
	lw := NewLocalWorkspace(WithBaseDir(baseDir))

	ws, err := lw.Provision(context.Background(), WorkspaceSpec{Stack: "test"})
	if err != nil {
		t.Fatalf("provision: %v", err)
	}
	defer lw.Recycle(context.Background(), ws.ID)

	if !strings.HasPrefix(ws.RootDir, baseDir) {
		t.Errorf("root dir %s not under base %s", ws.RootDir, baseDir)
	}
}

func TestLocalProvisionWithEnv(t *testing.T) {
	lw := NewLocalWorkspace()
	ws, err := lw.Provision(context.Background(), WorkspaceSpec{
		Stack: "test",
		Env:   map[string]string{"KEY": "value", "ANOTHER": "val2"},
	})
	if err != nil {
		t.Fatalf("provision: %v", err)
	}
	defer lw.Recycle(context.Background(), ws.ID)

	// Check .env file was created
	envData, err := os.ReadFile(filepath.Join(ws.RootDir, ".env"))
	if err != nil {
		t.Fatalf("read .env: %v", err)
	}
	content := string(envData)
	if !strings.Contains(content, "KEY=value") {
		t.Errorf(".env missing KEY: %s", content)
	}
	if !strings.Contains(content, "ANOTHER=val2") {
		t.Errorf(".env missing ANOTHER: %s", content)
	}
}

func TestLocalExec(t *testing.T) {
	lw := NewLocalWorkspace()
	ws, err := lw.Provision(context.Background(), WorkspaceSpec{Stack: "test"})
	if err != nil {
		t.Fatalf("provision: %v", err)
	}
	defer lw.Recycle(context.Background(), ws.ID)

	// Use platform-appropriate command
	cmd, args := "echo", []string{"hello"}

	resp, err := lw.Exec(context.Background(), ws.ID, ExecRequest{
		Command: cmd,
		Args:    args,
	})
	if err != nil {
		t.Fatalf("exec: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Errorf("exit: got=%d", resp.ExitCode)
	}
	if !strings.Contains(resp.Stdout, "hello") {
		t.Errorf("stdout: got=%s", resp.Stdout)
	}
}

func TestLocalExecInWorkDir(t *testing.T) {
	lw := NewLocalWorkspace()
	ws, err := lw.Provision(context.Background(), WorkspaceSpec{Stack: "test"})
	if err != nil {
		t.Fatalf("provision: %v", err)
	}
	defer lw.Recycle(context.Background(), ws.ID)

	// Create a subdirectory
	subDir := filepath.Join(ws.RootDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Write a file in the subdirectory
	if err := os.WriteFile(filepath.Join(subDir, "test.txt"), []byte("content"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Read from the subdirectory
	cmd := "cat"
	args := []string{"test.txt"}

	resp, err := lw.Exec(context.Background(), ws.ID, ExecRequest{
		Command: cmd,
		Args:    args,
		WorkDir: "subdir",
	})
	if err != nil {
		t.Fatalf("exec: %v", err)
	}
	if !strings.Contains(resp.Stdout, "content") {
		t.Errorf("stdout: got=%s", resp.Stdout)
	}
}

func TestLocalExecNotFound(t *testing.T) {
	lw := NewLocalWorkspace()
	_, err := lw.Exec(context.Background(), "nonexistent", ExecRequest{Command: "echo"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLocalExecNotReady(t *testing.T) {
	lw := NewLocalWorkspace()
	// Manually insert a workspace with Failed status
	lw.mu.Lock()
	lw.dirs["failed-ws"] = Workspace{ID: "failed-ws", Status: WorkspaceStatusFailed, RootDir: os.TempDir()}
	lw.mu.Unlock()

	_, err := lw.Exec(context.Background(), "failed-ws", ExecRequest{Command: "echo"})
	if err == nil {
		t.Fatal("expected error for failed workspace")
	}
}

func TestLocalStatus(t *testing.T) {
	lw := NewLocalWorkspace()
	ws, err := lw.Provision(context.Background(), WorkspaceSpec{Stack: "test"})
	if err != nil {
		t.Fatalf("provision: %v", err)
	}
	defer lw.Recycle(context.Background(), ws.ID)

	status, err := lw.Status(context.Background(), ws.ID)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status != WorkspaceStatusReady {
		t.Errorf("status: got=%s", status)
	}
}

func TestLocalStatusNotFound(t *testing.T) {
	lw := NewLocalWorkspace()
	_, err := lw.Status(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLocalRecycle(t *testing.T) {
	lw := NewLocalWorkspace()
	ws, err := lw.Provision(context.Background(), WorkspaceSpec{Stack: "test"})
	if err != nil {
		t.Fatalf("provision: %v", err)
	}

	// Verify dir exists
	if _, err := os.Stat(ws.RootDir); os.IsNotExist(err) {
		t.Fatal("dir should exist before recycle")
	}

	if err := lw.Recycle(context.Background(), ws.ID); err != nil {
		t.Fatalf("recycle: %v", err)
	}

	// Verify dir is removed
	if _, err := os.Stat(ws.RootDir); !os.IsNotExist(err) {
		t.Error("dir should not exist after recycle")
	}

	// Double recycle should error
	err = lw.Recycle(context.Background(), ws.ID)
	if err == nil {
		t.Fatal("expected error on double recycle")
	}
}

func TestLocalRecycleNotFound(t *testing.T) {
	lw := NewLocalWorkspace()
	err := lw.Recycle(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLocalExecTimeout(t *testing.T) {
	lw := NewLocalWorkspace()
	ws, err := lw.Provision(context.Background(), WorkspaceSpec{Stack: "test"})
	if err != nil {
		t.Fatalf("provision: %v", err)
	}
	defer lw.Recycle(context.Background(), ws.ID)

	cmd := "ping"
	args := []string{"-n", "10", "127.0.0.1"}

	_, err = lw.Exec(context.Background(), ws.ID, ExecRequest{
		Command: cmd,
		Args:    args,
		Timeout: 10 * time.Millisecond,
	})
	if err == nil {
		t.Fatal("expected timeout error")
	}
	t.Logf("timeout error (expected): %v", err)
}

func TestLocalContextCancellation(t *testing.T) {
	lw := NewLocalWorkspace()
	ws, err := lw.Provision(context.Background(), WorkspaceSpec{Stack: "test"})
	if err != nil {
		t.Fatalf("provision: %v", err)
	}
	defer lw.Recycle(context.Background(), ws.ID)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	cmd := "echo"

	_, err = lw.Exec(ctx, ws.ID, ExecRequest{
		Command: cmd,
		Args:    []string{"should not run"},
	})
	if err == nil {
		t.Fatal("expected cancellation error")
	}
	t.Logf("cancellation error (expected): %v", err)
}

func TestLocalExecUpdatesStatus(t *testing.T) {
	lw := NewLocalWorkspace()
	ws, err := lw.Provision(context.Background(), WorkspaceSpec{Stack: "test"})
	if err != nil {
		t.Fatalf("provision: %v", err)
	}
	defer lw.Recycle(context.Background(), ws.ID)

	// Status should be Ready initially
	status, _ := lw.Status(context.Background(), ws.ID)
	if status != WorkspaceStatusReady {
		t.Errorf("initial status: got=%s", status)
	}

	cmd := "echo"
	args := []string{"hello"}

	_, err = lw.Exec(context.Background(), ws.ID, ExecRequest{Command: cmd, Args: args})
	if err != nil {
		t.Fatalf("exec: %v", err)
	}

	// After Exec, status should be Running
	status, _ = lw.Status(context.Background(), ws.ID)
	if status != WorkspaceStatusRunning {
		t.Errorf("after exec: got=%s", status)
	}
}

func TestLocalProvisionUniqueIDs(t *testing.T) {
	lw := NewLocalWorkspace()
	ws1, _ := lw.Provision(context.Background(), WorkspaceSpec{Stack: "a"})
	ws2, _ := lw.Provision(context.Background(), WorkspaceSpec{Stack: "b"})
	ws3, _ := lw.Provision(context.Background(), WorkspaceSpec{Stack: "c"})

	if ws1.ID == ws2.ID || ws2.ID == ws3.ID || ws1.ID == ws3.ID {
		t.Fatal("expected unique IDs")
	}
}

func TestLocalProvisionWithRepoURL(t *testing.T) {
	lw := NewLocalWorkspace()
	// Use a context with a short timeout so the test does not hang on
	// network timeouts when git clone cannot reach the remote.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := lw.Provision(ctx, WorkspaceSpec{
		Stack:   "test",
		RepoURL: "https://github.com/nonexistent/repo.git",
	})
	if err == nil {
		t.Log("note: repo clone succeeded (unexpected)")
	} else {
		t.Logf("repo clone error (expected): %v", err)
	}
}

func TestLocalRecycleIdempotentOnDirMissing(t *testing.T) {
	lw := NewLocalWorkspace()
	ws, err := lw.Provision(context.Background(), WorkspaceSpec{Stack: "test"})
	if err != nil {
		t.Fatalf("provision: %v", err)
	}

	// Remove the directory manually
	os.RemoveAll(ws.RootDir)

	// Recycle should still clean up metadata
	err = lw.Recycle(context.Background(), ws.ID)
	if err != nil {
		t.Errorf("expected clean recycle even without dir, got: %v", err)
	}
}
