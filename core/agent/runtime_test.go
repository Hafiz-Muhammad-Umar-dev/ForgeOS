package agent

import (
	"context"
	"testing"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/workspace"
)

func TestNewRuntime(t *testing.T) {
	llm := NewEchoProvider()
	ws := workspace.NewFakeWorkspace()

	wsProvisioned, _ := ws.Provision(context.Background(), workspace.WorkspaceSpec{Stack: "test"})

	r := NewRuntime(llm, ws, wsProvisioned.ID)
	if r == nil {
		t.Fatal("runtime is nil")
	}
	if r.provider == nil {
		t.Error("provider is nil")
	}
	if r.ws == nil {
		t.Error("workspace is nil")
	}
}

func TestRuntimeRegisterAgent(t *testing.T) {
	r := newTestRuntime(t)
	fa := NewFakeAgent("test-agent")
	r.RegisterAgent(fa)

	if got := r.Agent("test-agent"); got == nil {
		t.Error("agent not found")
	}
	if got := r.Agent("nonexistent"); got != nil {
		t.Error("nonexistent agent should be nil")
	}
}

func TestRuntimeRunTask(t *testing.T) {
	r := newTestRuntime(t)
	r.RegisterAgent(NewFakeAgent("coder"))

	result, err := r.RunTask(context.Background(), "coder", Task{
		ID:          "task-1",
		Description: "write a function",
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if result.Status != ResultSuccess {
		t.Errorf("status=%s", result.Status)
	}
}

func TestRuntimeRunTaskUnknownAgent(t *testing.T) {
	r := newTestRuntime(t)
	_, err := r.RunTask(context.Background(), "ghost", Task{
		ID: "task-1", Description: "test",
	})
	if err == nil {
		t.Fatal("expected error for unknown agent")
	}
}

func TestRuntimeRunTaskDefaultIterations(t *testing.T) {
	fa := NewFakeAgent("coder")
	r := newTestRuntime(t)
	r.RegisterAgent(fa)

	_, err := r.RunTask(context.Background(), "coder", Task{
		ID: "t1", Description: "test",
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	if len(fa.ReceivedTasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(fa.ReceivedTasks))
	}
	if fa.ReceivedTasks[0].MaxIterations != 10 {
		t.Errorf("maxIter: got=%d want=10", fa.ReceivedTasks[0].MaxIterations)
	}
}

func TestRuntimeRunTaskCustomIterations(t *testing.T) {
	fa := NewFakeAgent("coder")
	r := newTestRuntime(t)
	r.RegisterAgent(fa)

	_, err := r.RunTask(context.Background(), "coder", Task{
		ID: "t1", Description: "test", MaxIterations: 3,
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	if fa.ReceivedTasks[0].MaxIterations != 3 {
		t.Errorf("maxIter: got=%d want=3", fa.ReceivedTasks[0].MaxIterations)
	}
}

func TestRuntimeRunTaskAgentError(t *testing.T) {
	fa := NewFakeAgent("failing")
	fa.RunError = ErrTaskFailed
	r := newTestRuntime(t)
	r.RegisterAgent(fa)

	_, err := r.RunTask(context.Background(), "failing", Task{
		ID: "t1", Description: "test",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRuntimeRunTaskStream(t *testing.T) {
	r := newTestRuntime(t)
	r.RegisterAgent(NewFakeAgent("streamer"))

	ch, err := r.RunTaskStream(context.Background(), "streamer", Task{
		ID: "t1", Description: "test",
	})
	if err != nil {
		t.Fatalf("stream: %v", err)
	}

	events := 0
	for evt := range ch {
		events++
		if evt.Err != nil {
			t.Fatalf("event err: %v", evt.Err)
		}
	}
	if events == 0 {
		t.Error("no events received")
	}
}

func TestRuntimeRunTaskStreamUnknownAgent(t *testing.T) {
	r := newTestRuntime(t)
	_, err := r.RunTaskStream(context.Background(), "ghost", Task{
		ID: "t1", Description: "test",
	})
	if err == nil {
		t.Fatal("expected error for unknown agent")
	}
}

func TestRuntimeLifecycle(t *testing.T) {
	r := newTestRuntime(t)

	if err := r.Init(context.Background()); err != nil {
		t.Fatalf("init: %v", err)
	}
	if r.Name() != "agent" {
		t.Errorf("name=%s", r.Name())
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

func TestRuntimeInitWithoutProvider(t *testing.T) {
	r := &Runtime{ws: workspace.NewFakeWorkspace()}
	err := r.Init(context.Background())
	if err == nil {
		t.Fatal("expected error without provider")
	}
}

func TestRuntimeInitWithoutWorkspace(t *testing.T) {
	r := &Runtime{provider: NewEchoProvider()}
	err := r.Init(context.Background())
	if err == nil {
		t.Fatal("expected error without workspace")
	}
}

func TestRuntimeConfig(t *testing.T) {
	llm := NewEchoProvider()
	ws := workspace.NewFakeWorkspace()
	wsProvisioned, _ := ws.Provision(context.Background(), workspace.WorkspaceSpec{Stack: "test"})

	r := NewRuntime(llm, ws, wsProvisioned.ID,
		WithRuntimeConfig(Config{DefaultMaxIterations: 5, PublishEvents: false}),
	)
	if r.cfg.DefaultMaxIterations != 5 {
		t.Errorf("maxIter: got=%d", r.cfg.DefaultMaxIterations)
	}
	if r.cfg.PublishEvents {
		t.Error("should not publish events")
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newTestRuntime(t *testing.T) *Runtime {
	t.Helper()
	llm := NewEchoProvider()
	ws := workspace.NewFakeWorkspace()
	wsProvisioned, err := ws.Provision(context.Background(), workspace.WorkspaceSpec{Stack: "test"})
	if err != nil {
		t.Fatalf("provision: %v", err)
	}
	return NewRuntime(llm, ws, wsProvisioned.ID)
}
