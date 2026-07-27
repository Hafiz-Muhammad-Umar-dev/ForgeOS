package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/agent"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/budget"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/event"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/hitl"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/planner"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/provider"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/registry"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/workspace"
)

// ---------------------------------------------------------------------------
// HITL Gate integration tests
// ---------------------------------------------------------------------------

func TestDAGExecutorHITLApprovalBlocksExecution(t *testing.T) {
	tb := &testBus{connected: true}
	gate := hitl.NewFakeHITLGate()
	gate.AutoApprove = true
	// Add a delay so we can verify the gate blocks
	gate.Delay = 50 * time.Millisecond

	e := newTestDAGExecutorWithHITL(t, tb, gate)
	ctx := context.Background()

	if err := e.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer e.Stop(ctx)

	start := time.Now()
	tb.deliver(ctx, "devos.intent.created", makeDAGIntentEvent(t, "build an app"))
	elapsed := time.Since(start)

	// Should have taken at least 50ms due to gate delay
	if elapsed < 40*time.Millisecond {
		t.Errorf("execution too fast (gate may not have blocked): %s", elapsed)
	}
	t.Logf("HITL gate blocked for %s (expected >= 50ms)", elapsed)
}

func TestDAGExecutorHITLApprovalPasses(t *testing.T) {
	tb := &testBus{connected: true}
	recorder := hitl.NewFakeHITLGate()
	recorder.AutoApprove = true

	e := newTestDAGExecutorWithHITL(t, tb, recorder)
	ctx := context.Background()

	if err := e.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer e.Stop(ctx)

	tb.deliver(ctx, "devos.intent.created", makeDAGIntentEvent(t, "test approval"))

	if recorder.RequestCount.Load() != 1 {
		t.Errorf("expected 1 approval request, got %d", recorder.RequestCount.Load())
	}
	if len(recorder.ReceivedRequests) == 0 {
		t.Fatal("no approval request received")
	}
	if recorder.ReceivedRequests[0].Type != hitl.ApprovalPlan {
		t.Errorf("type: got=%s", recorder.ReceivedRequests[0].Type)
	}
}

func TestDAGExecutorHITLRejectionStopsDAG(t *testing.T) {
	tb := &testBus{connected: true}
	gate := hitl.NewFakeHITLGate()
	gate.AutoApprove = false
	gate.AutoReject = true

	e := newTestDAGExecutorWithHITL(t, tb, gate)
	ctx := context.Background()

	if err := e.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer e.Stop(ctx)

	tb.deliver(ctx, "devos.intent.created", makeDAGIntentEvent(t, "test rejection"))

	tb.mu.Lock()
	defer tb.mu.Unlock()

	// Should have published intent.failed (not completed)
	for _, p := range tb.published {
		if env, err := event.Deserialize(p.data); err == nil {
			if env.Type == event.TypeIntentCompleted {
				t.Error("expected rejection, got completed")
			}
		}
	}
}

func TestDAGExecutorHITLTimeoutStopsDAG(t *testing.T) {
	tb := &testBus{connected: true}
	gate := hitl.NewFakeHITLGate()
	gate.SimulateTimeout = true

	e := newTestDAGExecutorWithHITL(t, tb, gate)
	ctx := context.Background()

	if err := e.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer e.Stop(ctx)

	tb.deliver(ctx, "devos.intent.created", makeDAGIntentEvent(t, "test timeout"))

	tb.mu.Lock()
	defer tb.mu.Unlock()

	// The DAG executor should not have published intent.completed
	for _, p := range tb.published {
		if env, err := event.Deserialize(p.data); err == nil {
			if env.Type == event.TypeIntentCompleted {
				t.Error("expected timeout, got completed")
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Budget Governor integration tests
// ---------------------------------------------------------------------------

func TestDAGExecutorBudgetCheckPasses(t *testing.T) {
	tb := &testBus{connected: true}
	gov := budget.NewFakeGovernor(1000)

	e := newTestDAGExecutorWithBudget(t, tb, gov)
	ctx := context.Background()

	if err := e.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer e.Stop(ctx)

	tb.deliver(ctx, "devos.intent.created", makeDAGIntentEvent(t, "test budget check"))

	if gov.CheckCount.Load() == 0 {
		t.Error("budget check was not called")
	}
}

func TestDAGExecutorBudgetExceededStopsDAG(t *testing.T) {
	tb := &testBus{connected: true}
	gov := budget.NewFakeGovernor(0) // zero ceiling — always exceeded

	e := newTestDAGExecutorWithBudget(t, tb, gov)
	ctx := context.Background()

	if err := e.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer e.Stop(ctx)

	tb.deliver(ctx, "devos.intent.created", makeDAGIntentEvent(t, "test budget exceeded"))

	tb.mu.Lock()
	defer tb.mu.Unlock()

	// Should have published intent.failed, not completed
	for _, p := range tb.published {
		if env, err := event.Deserialize(p.data); err == nil {
			if env.Type == event.TypeIntentCompleted {
				t.Error("expected budget exceeded, got completed")
			}
		}
	}
}

func TestDAGExecutorBudgetConsumeAfterNode(t *testing.T) {
	tb := &testBus{connected: true}
	gov := budget.NewFakeGovernor(10000)

	e := newTestDAGExecutorWithBudget(t, tb, gov)
	ctx := context.Background()

	if err := e.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer e.Stop(ctx)

	tb.deliver(ctx, "devos.intent.created", makeDAGIntentEvent(t, "test budget consume"))

	if gov.ConsumeCount.Load() == 0 {
		t.Error("budget consume was not called after node execution")
	}
}

func TestDAGExecutorBudgetCheckAndConsumeOrder(t *testing.T) {
	tb := &testBus{connected: true}
	gov := budget.NewFakeGovernor(10000)

	e := newTestDAGExecutorWithBudget(t, tb, gov)
	ctx := context.Background()

	if err := e.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer e.Stop(ctx)

	tb.deliver(ctx, "devos.intent.created", makeDAGIntentEvent(t, "test budget order"))

	// Check should be called before Consume
	if gov.CheckCount.Load() == 0 {
		t.Error("budget check not called")
	}
	if gov.ConsumeCount.Load() == 0 {
		t.Error("budget consume not called")
	}
}

// ---------------------------------------------------------------------------
// Combined HITL + Budget test
// ---------------------------------------------------------------------------

func TestDAGExecutorHITLAndBudgetCombined(t *testing.T) {
	tb := &testBus{connected: true}
	gate := hitl.NewFakeHITLGate()
	gov := budget.NewFakeGovernor(10000)

	llm := agent.NewEchoProvider()
	ws := workspace.NewFakeWorkspace()
	wsProvisioned, _ := ws.Provision(context.Background(), workspace.WorkspaceSpec{Stack: "test"})
	rt := agent.NewRuntime(llm, ws, wsProvisioned.ID)
	rt.RegisterAgent(agent.NewSimpleAgent("coder", "Coding agent", ""))

	fp := planner.NewFakePlanner()
	reg := registry.NewInMemoryRegistry()

	e := NewDAGExecutor(tb, fp, reg, rt, WithHITLGate(gate), WithBudgetGovernor(gov))
	ctx := context.Background()

	if err := e.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer e.Stop(ctx)

	tb.deliver(ctx, "devos.intent.created", makeDAGIntentEvent(t, "combined test"))

	if gate.RequestCount.Load() == 0 {
		t.Error("HITL gate not called")
	}
	if gov.CheckCount.Load() == 0 {
		t.Error("budget check not called")
	}
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func newTestDAGExecutorWithHITL(t *testing.T, tb *testBus, gate hitl.HITLGate) *DAGExecutor {
	t.Helper()
	llm := provider.NewFakeProvider()
	ws := workspace.NewFakeWorkspace()
	wsProvisioned, _ := ws.Provision(context.Background(), workspace.WorkspaceSpec{Stack: "test"})
	rt := agent.NewRuntime(llm, ws, wsProvisioned.ID)
	rt.RegisterAgent(agent.NewSimpleAgent("coder", "Coding agent", ""))

	fp := planner.NewFakePlanner()
	reg := registry.NewInMemoryRegistry()

	return NewDAGExecutor(tb, fp, reg, rt, WithHITLGate(gate))
}

func newTestDAGExecutorWithBudget(t *testing.T, tb *testBus, gov budget.Governor) *DAGExecutor {
	t.Helper()
	llm := provider.NewFakeProvider()
	ws := workspace.NewFakeWorkspace()
	wsProvisioned, _ := ws.Provision(context.Background(), workspace.WorkspaceSpec{Stack: "test"})
	rt := agent.NewRuntime(llm, ws, wsProvisioned.ID)
	rt.RegisterAgent(agent.NewSimpleAgent("coder", "Coding agent", ""))

	fp := planner.NewFakePlanner()
	reg := registry.NewInMemoryRegistry()

	return NewDAGExecutor(tb, fp, reg, rt, WithBudgetGovernor(gov))
}
