package eval

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/agent"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/workspace"
)

// ---------------------------------------------------------------------------
// Assertion tests
// ---------------------------------------------------------------------------

func TestAssertContainsPasses(t *testing.T) {
	a := AssertContains("hello")
	result := &agent.Result{Summary: "hello world"}
	if err := a.Check(result); err != nil {
		t.Fatalf("assertion failed: %v", err)
	}
}

func TestAssertContainsFails(t *testing.T) {
	a := AssertContains("goodbye")
	result := &agent.Result{Summary: "hello world"}
	if err := a.Check(result); err == nil {
		t.Fatal("expected assertion to fail")
	}
}

func TestAssertNotEmptyPasses(t *testing.T) {
	a := AssertNotEmpty()
	if err := a.Check(&agent.Result{Summary: "content"}); err != nil {
		t.Fatalf("assertion failed: %v", err)
	}
}

func TestAssertNotEmptyFails(t *testing.T) {
	a := AssertNotEmpty()
	if err := a.Check(&agent.Result{Summary: ""}); err == nil {
		t.Fatal("expected assertion to fail")
	}
}

func TestAssertMatchPasses(t *testing.T) {
	a := AssertMatch("(?i)hello")
	if err := a.Check(&agent.Result{Summary: "Hello World"}); err != nil {
		t.Fatalf("assertion failed: %v", err)
	}
}

func TestAssertMatchFails(t *testing.T) {
	a := AssertMatch("goodbye")
	if err := a.Check(&agent.Result{Summary: "hello"}); err == nil {
		t.Fatal("expected assertion to fail")
	}
}

func TestAssertStatusPasses(t *testing.T) {
	a := AssertStatus(agent.ResultSuccess)
	if err := a.Check(&agent.Result{Status: agent.ResultSuccess}); err != nil {
		t.Fatalf("assertion failed: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Golden task tests
// ---------------------------------------------------------------------------

func TestDefaultGoldenTasksExist(t *testing.T) {
	tasks := DefaultGoldenTasks()
	if len(tasks) != 5 {
		t.Fatalf("expected 5 tasks, got %d", len(tasks))
	}
}

func TestGoldenTasksHaveNames(t *testing.T) {
	for _, gt := range DefaultGoldenTasks() {
		if gt.Name == "" {
			t.Error("task has empty name")
		}
		if gt.Description == "" {
			t.Errorf("task %q has empty description", gt.Name)
		}
		if len(gt.Assertions) == 0 {
			t.Errorf("task %q has no assertions", gt.Name)
		}
	}
}

func TestGoldenTaskNamesAreUnique(t *testing.T) {
	seen := make(map[string]bool)
	for _, gt := range DefaultGoldenTasks() {
		if seen[gt.Name] {
			t.Errorf("duplicate task name: %s", gt.Name)
		}
		seen[gt.Name] = true
	}
}

// ---------------------------------------------------------------------------
// Evaluator tests
// ---------------------------------------------------------------------------

func newTestEvaluator(t testing.TB) *Evaluator {
	t.Helper()
	// Use EchoProvider instead of FakeProvider to avoid exhausting
	// predefined responses across multiple task calls.
	llm := agent.NewEchoProvider()
	ws := workspace.NewFakeWorkspace()
	wsProvisioned, err := ws.Provision(context.Background(), workspace.WorkspaceSpec{Stack: "test"})
	if err != nil {
		t.Fatalf("provision: %v", err)
	}
	rt := agent.NewRuntime(llm, ws, wsProvisioned.ID)
	rt.RegisterAgent(agent.NewSimpleAgent("coder", "Coding agent", ""))
	return NewEvaluator(rt, ws, wsProvisioned.ID)
}

func TestEvaluatorRunPasses(t *testing.T) {
	e := newTestEvaluator(t)
	ctx := context.Background()

	result := e.Run(ctx, GoldenTask{
		Name:        "test-task",
		Description: "A test task",
		Task: agent.Task{
			ID:          "test-1",
			Description: "say hello",
		},
		Assertions: []Assertion{
			AssertNotEmpty(),
		},
	})

	if !result.Passed {
		t.Fatalf("expected pass, got errors: %v", result.Errors)
	}
	if result.TaskName != "test-task" {
		t.Errorf("task name: got=%s", result.TaskName)
	}
	if result.Duration <= 0 {
		t.Error("expected positive duration")
	}
}

func TestEvaluatorRunFailsOnAssertion(t *testing.T) {
	e := newTestEvaluator(t)
	ctx := context.Background()

	result := e.Run(ctx, GoldenTask{
		Name:        "fail-task",
		Description: "A task that should fail assertions",
		Task: agent.Task{
			ID:          "test-2",
			Description: "say hello",
		},
		Assertions: []Assertion{
			AssertContains("nonexistent-pattern-xyz"),
		},
	})

	if result.Passed {
		t.Fatal("expected failure")
	}
	if len(result.Errors) == 0 {
		t.Fatal("expected at least one error")
	}
}

func TestEvaluatorRunAll(t *testing.T) {
	e := newTestEvaluator(t)
	ctx := context.Background()

	tasks := []GoldenTask{
		{
			Name: "task-1", Description: "first",
			Task:       agent.Task{ID: "all-1", Description: "say hello"},
			Assertions: []Assertion{AssertNotEmpty()},
			Timeout:    5 * time.Second,
		},
		{
			Name: "task-2", Description: "second",
			Task:       agent.Task{ID: "all-2", Description: "say world"},
			Assertions: []Assertion{AssertNotEmpty()},
			Timeout:    5 * time.Second,
		},
	}

	results := e.RunAll(ctx, tasks)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	for _, r := range results {
		if !r.Passed {
			t.Errorf("task %s failed: %v", r.TaskName, r.Errors)
		}
	}
}

func TestEvaluatorRunTimeout(t *testing.T) {
	e := newTestEvaluator(t)
	ctx := context.Background()

	result := e.Run(ctx, GoldenTask{
		Name:        "timeout-task",
		Description: "A task with very short timeout",
		Task: agent.Task{
			ID:            "timeout-1",
			Description:   "say hello",
			MaxIterations: 5,
		},
		Assertions: []Assertion{AssertNotEmpty()},
		Timeout:    1 * time.Millisecond, // extremely short
	})

	t.Logf("timeout result: passed=%v errors=%v duration=%s", result.Passed, result.Errors, result.Duration)
	// The task may or may not timeout depending on echo speed
}

// ---------------------------------------------------------------------------
// Report tests
// ---------------------------------------------------------------------------

func TestSummaryFormat(t *testing.T) {
	results := []EvalResult{
		{TaskName: "a", Passed: true, Duration: time.Second},
		{TaskName: "b", Passed: false, Errors: []string{"assertion failed"}, Duration: time.Second},
	}

	s := Summary(results)
	if !strings.Contains(s, "2 total") {
		t.Errorf("missing total count: %s", s)
	}
	if !strings.Contains(s, "1 passed") {
		t.Errorf("missing pass count: %s", s)
	}
	if !strings.Contains(s, "1 failed") {
		t.Errorf("missing fail count: %s", s)
	}
	if !strings.Contains(s, "assertion failed") {
		t.Errorf("missing error detail: %s", s)
	}
}

func TestSummaryEmpty(t *testing.T) {
	s := Summary(nil)
	if s == "" {
		t.Error("expected non-empty summary")
	}
}

// ---------------------------------------------------------------------------
// Benchmark tests
// ---------------------------------------------------------------------------

func BenchmarkEvaluatorRun(b *testing.B) {
	e := newTestEvaluator(b)
	ctx := context.Background()

	gt := GoldenTask{
		Name: "bench", Description: "benchmark",
		Task:       agent.Task{ID: "bench-1", Description: "say hello"},
		Assertions: []Assertion{AssertNotEmpty()},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.Run(ctx, gt)
	}
}
