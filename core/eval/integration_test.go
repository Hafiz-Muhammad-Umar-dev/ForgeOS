//go:build integration

package eval

import (
	"context"
	"testing"
	"time"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/agent"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/workspace"
)

func TestIntegrationEvaluatorRunAllGoldenTasks(t *testing.T) {
	llm := agent.NewEchoProvider()
	ws := workspace.NewFakeWorkspace()
	wsProvisioned, err := ws.Provision(context.Background(), workspace.WorkspaceSpec{Stack: "test"})
	if err != nil {
		t.Fatalf("provision: %v", err)
	}
	defer ws.Recycle(context.Background(), wsProvisioned.ID)

	rt := agent.NewRuntime(llm, ws, wsProvisioned.ID)
	rt.RegisterAgent(agent.NewSimpleAgent("coder", "Coding agent", ""))

	e := NewEvaluator(rt, ws, wsProvisioned.ID)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tasks := DefaultGoldenTasks()
	results := e.RunAll(ctx, tasks)

	if len(results) != len(tasks) {
		t.Fatalf("expected %d results, got %d", len(tasks), len(results))
	}

	passed := 0
	for _, r := range results {
		if r.Passed {
			passed++
		}
		t.Logf("task=%s passed=%v duration=%s", r.TaskName, r.Passed, r.Duration)
	}

	t.Logf("Golden tasks: %d/%d passed", passed, len(results))
}

func TestIntegrationEvaluatorCustomTask(t *testing.T) {
	llm := agent.NewEchoProvider()
	ws := workspace.NewFakeWorkspace()
	wsProvisioned, err := ws.Provision(context.Background(), workspace.WorkspaceSpec{Stack: "test"})
	if err != nil {
		t.Fatalf("provision: %v", err)
	}
	defer ws.Recycle(context.Background(), wsProvisioned.ID)

	rt := agent.NewRuntime(llm, ws, wsProvisioned.ID)
	rt.RegisterAgent(agent.NewSimpleAgent("coder", "Coding agent", ""))

	e := NewEvaluator(rt, ws, wsProvisioned.ID)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result := e.Run(ctx, GoldenTask{
		Name:        "integration-test",
		Description: "Verify evaluator works end-to-end",
		Task:        agent.Task{ID: "int-eval-1", Description: "say hello"},
		Assertions: []Assertion{
			AssertNotEmpty(),
			AssertContains("hello"),
		},
	})

	if !result.Passed {
		t.Fatalf("expected pass, got errors: %v", result.Errors)
	}
}
