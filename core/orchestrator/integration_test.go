//go:build integration

package orchestrator

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/agent"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/event"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/ingress"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/workspace"
)

// TestIntegrationOrchestratorFullCycle verifies the entire intent lifecycle:
// subscribe → receive event → dispatch → publish lifecycle event.
func TestIntegrationOrchestratorFullCycle(t *testing.T) {
	tb := &testBus{connected: true}
	fr := NewFakeRunner()
	e := NewEngine(tb, fr)
	ctx := context.Background()

	if err := e.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer e.Stop(ctx)

	payload := ingress.IntentPayload{
		Text:    "build a react app",
		UserID:  "user-integration",
		TraceID: "trace-integration",
	}
	env := event.New(event.TypeIntentCreated, "integration-test", payload,
		event.WithTraceID("trace-integration"),
		event.WithOrgID("org-integration"),
	)
	data, err := event.Serialize(env)
	if err != nil {
		t.Fatalf("serialize: %v", err)
	}

	tb.deliver(ctx, "devos.intent.created", data)

	// Verify dispatch
	if fr.RunCount.Load() != 1 {
		t.Fatalf("run count: got=%d", fr.RunCount.Load())
	}
	if fr.Received[0].Task.Description != "build a react app" {
		t.Errorf("description=%s", fr.Received[0].Task.Description)
	}

	// Verify lifecycle event was published
	tb.mu.Lock()
	defer tb.mu.Unlock()
	var found bool
	for _, p := range tb.published {
		if p.subject == "devos.intent.completed" {
			found = true
			var raw map[string]any
			if err := json.Unmarshal(p.data, &raw); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if raw["type"] != "intent.completed" {
				t.Errorf("type=%v", raw["type"])
			}
			if raw["traceId"] != "trace-integration" {
				t.Errorf("traceId=%v", raw["traceId"])
			}
			break
		}
	}
	if !found {
		t.Error("intent.completed not published")
	}
}

// TestIntegrationOrchestratorDispatchFailure verifies the failure path.
func TestIntegrationOrchestratorDispatchFailure(t *testing.T) {
	tb := &testBus{connected: true}
	fr := NewFakeRunner()
	fr.RunError = ErrDispatchFailed
	e := NewEngine(tb, fr)
	ctx := context.Background()

	if err := e.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer e.Stop(ctx)

	payload := ingress.IntentPayload{Text: "task that fails"}
	env := event.New(event.TypeIntentCreated, "integration-test", payload)
	data, _ := event.Serialize(env)

	tb.deliver(ctx, "devos.intent.created", data)

	tb.mu.Lock()
	defer tb.mu.Unlock()
	var found bool
	for _, p := range tb.published {
		if p.subject == "devos.intent.failed" {
			found = true
			break
		}
	}
	if !found {
		t.Error("intent.failed not published")
	}
}

// TestIntegrationOrchestratorWithRealAgentRuntime verifies that the
// orchestrator can dispatch to an agent.Runtime with a real FakeProvider.
func TestIntegrationOrchestratorWithRealAgentRuntime(t *testing.T) {
	// This test verifies the integration boundary between orchestrator
	// and the agent runtime by using a real agent.Runtime and FakeAgent.
	tb := &testBus{connected: true}

	// Create a minimal agent runtime with a fake agent
	ar := newTestAgentRuntime(t)
	ar.RegisterAgent(agent.NewSimpleAgent("coder", "Coding agent", ""))

	// Create orchestrator using the agent runtime as TaskRunner
	e := NewEngine(tb, ar, WithAgent("coder"))
	ctx := context.Background()

	if err := e.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer e.Stop(ctx)

	payload := ingress.IntentPayload{Text: "write a function"}
	env := event.New(event.TypeIntentCreated, "integration-test", payload)
	data, _ := event.Serialize(env)

	tb.deliver(ctx, "devos.intent.created", data)

	// Wait for async dispatch
	time.Sleep(200 * time.Millisecond)

	// Verify lifecycle event was published
	tb.mu.Lock()
	defer tb.mu.Unlock()
	var found bool
	for _, p := range tb.published {
		if p.subject == "devos.intent.completed" {
			found = true
			break
		}
	}
	if !found {
		t.Error("intent.completed not published with real agent runtime")
	}
}

// newTestAgentRuntime creates a minimal agent runtime for integration testing.
func newTestAgentRuntime(t *testing.T) *agent.Runtime {
	t.Helper()
	llm := agent.NewEchoProvider()
	ws := workspace.NewFakeWorkspace()
	wsProvisioned, err := ws.Provision(context.Background(), workspace.WorkspaceSpec{Stack: "test"})
	if err != nil {
		t.Fatalf("provision: %v", err)
	}
	return agent.NewRuntime(llm, ws, wsProvisioned.ID)
}
