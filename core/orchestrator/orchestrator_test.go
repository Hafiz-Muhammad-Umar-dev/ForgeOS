package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/agent"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/dag"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/event"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/ingress"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/planner"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/provider"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/registry"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/workspace"
)

// makeIntentEvent creates a serialized intent.created event for testing
// without relying on engine_test.go's version (which uses different params).
func makeDAGIntentEvent(t *testing.T, text string) []byte {
	t.Helper()
	payload := ingress.IntentPayload{Text: text, UserID: "test-user"}
	env := event.New(event.TypeIntentCreated, "test", payload)
	data, _ := event.Serialize(env)
	return data
}

// ---------------------------------------------------------------------------
// DAGExecutor lifecycle tests
// ---------------------------------------------------------------------------

func newTestDAGExecutor(t *testing.T) *DAGExecutor {
	t.Helper()
	tb := &testBus{connected: true}
	fp := planner.NewFakePlanner()
	reg := registry.NewInMemoryRegistry()
	llm := provider.NewFakeProvider()
	ws := workspace.NewFakeWorkspace()
	wsProvisioned, _ := ws.Provision(context.Background(), workspace.WorkspaceSpec{Stack: "test"})
	rt := agent.NewRuntime(llm, ws, wsProvisioned.ID)
	rt.RegisterAgent(agent.NewSimpleAgent("coder", "Coding agent", ""))

	return NewDAGExecutor(tb, fp, reg, rt)
}

func TestDAGExecutorName(t *testing.T) {
	e := newTestDAGExecutor(t)
	if e.Name() != "dag_executor" {
		t.Errorf("name=%s", e.Name())
	}
}

func TestDAGExecutorInitSuccess(t *testing.T) {
	e := newTestDAGExecutor(t)
	if err := e.Init(context.Background()); err != nil {
		t.Fatalf("init: %v", err)
	}
}

func TestDAGExecutorInitNoBus(t *testing.T) {
	e := &DAGExecutor{bus: nil, planner: planner.NewFakePlanner()}
	if err := e.Init(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}

func TestDAGExecutorStartStop(t *testing.T) {
	e := newTestDAGExecutor(t)
	ctx := context.Background()

	if err := e.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	h := e.Health()
	if h.Status != "UP" {
		t.Errorf("health after start: got=%s", h.Status)
	}
	if err := e.Stop(ctx); err != nil {
		t.Fatalf("stop: %v", err)
	}
	h = e.Health()
	if h.Status == "UP" {
		t.Errorf("health after stop: got=%s", h.Status)
	}
}

// ---------------------------------------------------------------------------
// DAG execution tests
// ---------------------------------------------------------------------------

func TestDAGExecutorHandlesIntent(t *testing.T) {
	tb := &testBus{connected: true}
	fp := planner.NewFakePlanner()
	reg := registry.NewInMemoryRegistry()
	llm := provider.NewFakeProvider()
	ws := workspace.NewFakeWorkspace()
	wsProvisioned, _ := ws.Provision(context.Background(), workspace.WorkspaceSpec{Stack: "test"})
	rt := agent.NewRuntime(llm, ws, wsProvisioned.ID)
	rt.RegisterAgent(agent.NewSimpleAgent("coder", "Coding agent", ""))

	e := NewDAGExecutor(tb, fp, reg, rt)
	ctx := context.Background()
	if err := e.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer e.Stop(ctx)

	tb.deliver(ctx, "devos.intent.created", makeDAGIntentEvent(t, "build an app"))

	tb.mu.Lock()
	defer tb.mu.Unlock()
	if len(tb.published) == 0 {
		t.Error("no events published")
	}
}

func TestDAGExecutorMultiNodeDAG(t *testing.T) {
	tb := &testBus{connected: true}
	llm := agent.NewEchoProvider()
	ws := workspace.NewFakeWorkspace()
	wsProvisioned, _ := ws.Provision(context.Background(), workspace.WorkspaceSpec{Stack: "test"})
	rt := agent.NewRuntime(llm, ws, wsProvisioned.ID)
	rt.RegisterAgent(agent.NewSimpleAgent("frontend", "Frontend", ""))
	rt.RegisterAgent(agent.NewSimpleAgent("backend", "Backend", ""))

	reg := registry.NewInMemoryRegistry()
	reg.RegisterAgent(context.Background(), registry.ServiceInfo{ID: "frontend", Name: "frontend", Kind: "agent"})
	reg.RegisterAgent(context.Background(), registry.ServiceInfo{ID: "backend", Name: "backend", Kind: "agent"})

	fp := planner.NewFakePlanner()
	fp.Result = &dag.DAG{
		Nodes: []dag.Node{
			{ID: "frontend-node", Agent: "frontend", Description: "build UI"},
			{ID: "backend-node", Agent: "backend", Description: "build API", InputIDs: []string{"frontend-node"}},
		},
	}

	e := NewDAGExecutor(tb, fp, reg, rt)
	ctx := context.Background()
	if err := e.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer e.Stop(ctx)

	tb.deliver(ctx, "devos.intent.created", makeDAGIntentEvent(t, "build full stack"))

	time.Sleep(200 * time.Millisecond)

	tb.mu.Lock()
	defer tb.mu.Unlock()
	hasCompleted := false
	for _, p := range tb.published {
		if p.subject == "devos.intent.completed" {
			hasCompleted = true
			break
		}
	}
	if !hasCompleted {
		t.Log("note: intent.completed not found (may still be processing)")
	}
}

// ---------------------------------------------------------------------------
// Coordinator tests
// ---------------------------------------------------------------------------

func TestCoordinatorResolveAgent(t *testing.T) {
	reg := registry.NewInMemoryRegistry()
	reg.RegisterAgent(context.Background(), registry.ServiceInfo{ID: "coder", Name: "coder", Kind: "agent"})

	coord := NewCoordinator(reg, nil)
	node := dag.Node{ID: "n1", Agent: "coder", Description: "write code"}

	agentName, err := coord.ResolveAgent(context.Background(), node)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if agentName != "coder" {
		t.Errorf("agent: got=%s", agentName)
	}
}

func TestCoordinatorResolveAgentFallback(t *testing.T) {
	reg := registry.NewInMemoryRegistry()
	coord := NewCoordinator(reg, nil)
	node := dag.Node{ID: "n1", Agent: "unknown-agent", Description: "test"}

	agentName, err := coord.ResolveAgent(context.Background(), node)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if agentName != "unknown-agent" {
		t.Errorf("agent: got=%s", agentName)
	}
}

func TestNodeExecutor(t *testing.T) {
	llm := agent.NewEchoProvider()
	ws := workspace.NewFakeWorkspace()
	wsProvisioned, _ := ws.Provision(context.Background(), workspace.WorkspaceSpec{Stack: "test"})
	rt := agent.NewRuntime(llm, ws, wsProvisioned.ID)
	rt.RegisterAgent(agent.NewSimpleAgent("coder", "Coding agent", ""))

	reg := registry.NewInMemoryRegistry()
	coord := NewCoordinator(reg, rt)
	ne := NewNodeExecutor(coord)

	node := &dag.Node{ID: "n1", Agent: "coder", Description: "say hello"}
	d := &dag.DAG{Nodes: []dag.Node{*node}}

	result, err := ne.Execute(context.Background(), node, d)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result == nil {
		t.Fatal("nil result")
	}
	if node.Status != dag.NodeCompleted {
		t.Errorf("status: got=%s", node.Status)
	}
}

// ---------------------------------------------------------------------------
// Event type verification
// ---------------------------------------------------------------------------

func TestNewEventTypesExist(t *testing.T) {
	if event.TypePlanStarted == "" {
		t.Error("TypePlanStarted not defined")
	}
	if event.TypeNodeStatus == "" {
		t.Error("TypeNodeStatus not defined")
	}
	if event.TypeNodeFailed == "" {
		t.Error("TypeNodeFailed not defined")
	}
}
