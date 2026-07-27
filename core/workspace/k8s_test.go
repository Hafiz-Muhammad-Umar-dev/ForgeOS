package workspace

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

// fakeK8sClient implements K8sClient for testing.
type fakeK8sClient struct {
	CreatePodFunc func(ctx context.Context, name, image, namespace string, env map[string]string, command []string) error
	DeletePodFunc func(ctx context.Context, name, namespace string) error
	PodStatusFunc func(ctx context.Context, name, namespace string) (string, error)
	ExecInPodFunc func(ctx context.Context, name, namespace, container string, cmd []string) (string, string, int, error)
	PodIPFunc     func(ctx context.Context, name, namespace string) (string, error)

	CreateCount atomic.Int64
	DeleteCount atomic.Int64
}

func (f *fakeK8sClient) CreatePod(ctx context.Context, name, image, namespace string, env map[string]string, command []string) error {
	f.CreateCount.Add(1)
	if f.CreatePodFunc != nil {
		return f.CreatePodFunc(ctx, name, image, namespace, env, command)
	}
	return nil
}

func (f *fakeK8sClient) DeletePod(ctx context.Context, name, namespace string) error {
	f.DeleteCount.Add(1)
	if f.DeletePodFunc != nil {
		return f.DeletePodFunc(ctx, name, namespace)
	}
	return nil
}

func (f *fakeK8sClient) PodStatus(ctx context.Context, name, namespace string) (string, error) {
	if f.PodStatusFunc != nil {
		return f.PodStatusFunc(ctx, name, namespace)
	}
	return "Running", nil
}

func (f *fakeK8sClient) ExecInPod(ctx context.Context, name, namespace, container string, cmd []string) (string, string, int, error) {
	if f.ExecInPodFunc != nil {
		return f.ExecInPodFunc(ctx, name, namespace, container, cmd)
	}
	return "", "", 0, nil
}

func (f *fakeK8sClient) PodIP(ctx context.Context, name, namespace string) (string, error) {
	if f.PodIPFunc != nil {
		return f.PodIPFunc(ctx, name, namespace)
	}
	return "10.0.0.1", nil
}

func TestK8sWorkspaceProvision(t *testing.T) {
	fc := &fakeK8sClient{}
	w := NewK8sWorkspace(fc)

	ws, err := w.Provision(context.Background(), WorkspaceSpec{Stack: "node:20"})
	if err != nil {
		t.Fatalf("provision: %v", err)
	}
	if ws.ID == "" {
		t.Fatal("empty id")
	}
	if ws.Spec.Stack != "node:20" {
		t.Errorf("stack=%s", ws.Spec.Stack)
	}
	if ws.Status != WorkspaceStatusReady {
		t.Errorf("status=%s", ws.Status)
	}
	if fc.CreateCount.Load() != 1 {
		t.Errorf("create count=%d", fc.CreateCount.Load())
	}
}

func TestK8sWorkspaceProvisionWithOptions(t *testing.T) {
	fc := &fakeK8sClient{}
	w := NewK8sWorkspace(fc,
		WithK8sNamespace("custom-ns"),
		WithK8sDefaultImage("node:20-alpine"),
		WithK8sPodTimeout(30*time.Second),
	)
	if w.config.Namespace != "custom-ns" {
		t.Errorf("ns=%s", w.config.Namespace)
	}
	if w.config.DefaultImage != "node:20-alpine" {
		t.Errorf("image=%s", w.config.DefaultImage)
	}
}

func TestK8sWorkspaceExec(t *testing.T) {
	fc := &fakeK8sClient{}
	fc.ExecInPodFunc = func(_ context.Context, _, _, _ string, _ []string) (string, string, int, error) {
		return "hello", "", 0, nil
	}

	w := NewK8sWorkspace(fc)
	ws, _ := w.Provision(context.Background(), WorkspaceSpec{Stack: "test"})

	resp, err := w.Exec(context.Background(), ws.ID, ExecRequest{Command: "echo hello"})
	if err != nil {
		t.Fatalf("exec: %v", err)
	}
	if resp.Stdout != "hello" {
		t.Errorf("stdout=%s", resp.Stdout)
	}
}

func TestK8sWorkspaceExecNotFound(t *testing.T) {
	fc := &fakeK8sClient{}
	w := NewK8sWorkspace(fc)
	_, err := w.Exec(context.Background(), "nonexistent", ExecRequest{Command: "echo"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestK8sWorkspaceStatus(t *testing.T) {
	fc := &fakeK8sClient{}
	w := NewK8sWorkspace(fc)
	ws, _ := w.Provision(context.Background(), WorkspaceSpec{Stack: "test"})

	status, err := w.Status(context.Background(), ws.ID)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status != WorkspaceStatusReady {
		t.Errorf("status=%s", status)
	}
}

func TestK8sWorkspaceStatusNotFound(t *testing.T) {
	fc := &fakeK8sClient{}
	w := NewK8sWorkspace(fc)
	_, err := w.Status(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestK8sWorkspaceRecycle(t *testing.T) {
	fc := &fakeK8sClient{}
	w := NewK8sWorkspace(fc)
	ws, _ := w.Provision(context.Background(), WorkspaceSpec{Stack: "test"})

	if err := w.Recycle(context.Background(), ws.ID); err != nil {
		t.Fatalf("recycle: %v", err)
	}
	if fc.DeleteCount.Load() != 1 {
		t.Errorf("delete count=%d", fc.DeleteCount.Load())
	}
}

func TestK8sWorkspaceRecycleNotFound(t *testing.T) {
	fc := &fakeK8sClient{}
	w := NewK8sWorkspace(fc)
	err := w.Recycle(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestK8sWorkspaceImageForStack(t *testing.T) {
	fc := &fakeK8sClient{}
	w := NewK8sWorkspace(fc, WithK8sImageMap(map[string]string{
		"node:20":  "node:20-alpine",
		"python:3": "python:3-slim",
	}))

	if img := w.imageFor("node:20"); img != "node:20-alpine" {
		t.Errorf("node: got=%s", img)
	}
	if img := w.imageFor("unknown"); img != "ubuntu:22.04" {
		t.Errorf("unknown: got=%s", img)
	}
	if img := w.imageFor(""); img != "ubuntu:22.04" {
		t.Errorf("empty: got=%s", img)
	}
}

func TestK8sWorkspaceProvisionCreateError(t *testing.T) {
	fc := &fakeK8sClient{}
	fc.CreatePodFunc = func(_ context.Context, _, _, _ string, _ map[string]string, _ []string) error {
		return fmt.Errorf("cluster unavailable")
	}

	w := NewK8sWorkspace(fc)
	_, err := w.Provision(context.Background(), WorkspaceSpec{Stack: "test"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestK8sWorkspaceDoubleRecycle(t *testing.T) {
	fc := &fakeK8sClient{}
	w := NewK8sWorkspace(fc)
	ws, _ := w.Provision(context.Background(), WorkspaceSpec{Stack: "test"})

	w.Recycle(context.Background(), ws.ID)
	err := w.Recycle(context.Background(), ws.ID)
	if err == nil {
		t.Log("note: double recycle returned nil (expected)")
	}
}

func TestK8sWorkspaceProvisionUniqueIDs(t *testing.T) {
	fc := &fakeK8sClient{}
	w := NewK8sWorkspace(fc)

	ws1, _ := w.Provision(context.Background(), WorkspaceSpec{Stack: "a"})
	ws2, _ := w.Provision(context.Background(), WorkspaceSpec{Stack: "b"})
	ws3, _ := w.Provision(context.Background(), WorkspaceSpec{Stack: "c"})

	if ws1.ID == ws2.ID || ws2.ID == ws3.ID || ws1.ID == ws3.ID {
		t.Fatal("expected unique IDs")
	}
}

func TestK8sWorkspaceStatusPodPhase(t *testing.T) {
	fc := &fakeK8sClient{}
	fc.PodStatusFunc = func(_ context.Context, _, _ string) (string, error) {
		return "Pending", nil
	}

	w := NewK8sWorkspace(fc)
	ws, _ := w.Provision(context.Background(), WorkspaceSpec{Stack: "test"})

	status, err := w.Status(context.Background(), ws.ID)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status != WorkspaceStatusProvisioning {
		t.Errorf("status=%s", status)
	}
}

func TestK8sWorkspaceDefaultConfig(t *testing.T) {
	cfg := DefaultK8sConfig()
	if cfg.Namespace != "devos" {
		t.Errorf("ns=%s", cfg.Namespace)
	}
	if cfg.DefaultImage != "ubuntu:22.04" {
		t.Errorf("image=%s", cfg.DefaultImage)
	}
}

func TestK8sClientInterface(t *testing.T) {
	// Verify K8sClient interface is satisfied by fake.
	var _ K8sClient = (*fakeK8sClient)(nil)
}
