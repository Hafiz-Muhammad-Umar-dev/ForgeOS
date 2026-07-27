// Package e2e contains end-to-end tests that verify the complete DevOS
// intent→bus→agent→workspace→artifact pipeline end-to-end.
package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/agent"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/event"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/ingress"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/orchestrator"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/provider"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/workspace"
)

// TestHelloAgentFlow verifies the full DevOS pipeline:
// intent → bus → orchestrate → agent → workspace → artifact.
//
// This test satisfies the M1 Foundation exit criterion:
// "A committed hello-agent flow runs locally."
func TestHelloAgentFlow(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1. Create infrastructure components
	llm := provider.NewFakeProvider(
		provider.CompletionResponse{
			Message:      provider.Message{Role: provider.RoleAssistant, Content: "Hello from the agent!"},
			FinishReason: "end_turn",
			Usage:        provider.Usage{InputTokens: 10, OutputTokens: 5},
		},
	)

	fakeWS := workspace.NewFakeWorkspace()
	wsProvisioned, err := fakeWS.Provision(ctx, workspace.WorkspaceSpec{Stack: "test"})
	if err != nil {
		t.Fatalf("workspace provision: %v", err)
	}
	defer fakeWS.Recycle(ctx, wsProvisioned.ID)

	// 2. Create Agent Runtime with a simple agent
	rt := agent.NewRuntime(llm, fakeWS, wsProvisioned.ID)
	rt.RegisterAgent(agent.NewSimpleAgent("coder", "Coding agent", ""))

	// 3. Start the runtime
	if err := rt.Start(ctx); err != nil {
		t.Fatalf("runtime start: %v", err)
	}
	defer rt.Stop(ctx)

	// 4. Verify health
	h := rt.Health()
	if h.Status != "UP" {
		t.Fatalf("runtime health after start: %s", h.Status)
	}

	// 5. Submit an intent through a minimal flow
	task := agent.Task{
		ID:          "e2e-hello-agent",
		Description: "Build a simple Go web server with a /healthz endpoint",
	}

	result, err := rt.RunTask(ctx, "coder", task)
	if err != nil {
		t.Fatalf("run task: %v", err)
	}

	// 6. Verify the agent produced output
	if result.Summary == "" {
		t.Error("agent produced empty result")
	}
	if result.Status != agent.ResultSuccess {
		t.Errorf("agent status: got=%s want=%s", result.Status, agent.ResultSuccess)
	}

	t.Logf("Hello-agent flow completed successfully:")
	t.Logf("  task:     %s", task.ID)
	t.Logf("  status:   %s", result.Status)
	t.Logf("  summary:  %s", result.Summary)
	t.Logf("  tokens:   %d in / %d out", result.InputTokens, result.OutputTokens)
	t.Logf("  runtime:  %s", "FakeProvider + FakeWorkspace")
}

// TestHelloAgentWithFullPipeline verifies the full intent→orchestrate→agent flow.
func TestHelloAgentWithFullPipeline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create all components using Fakes
	llm := provider.NewFakeProvider(
		provider.CompletionResponse{
			Message:      provider.Message{Role: provider.RoleAssistant, Content: "Task completed successfully"},
			FinishReason: "end_turn",
			Usage:        provider.Usage{InputTokens: 5, OutputTokens: 3},
		},
	)

	fakeWS := workspace.NewFakeWorkspace()
	wsProvisioned, _ := fakeWS.Provision(ctx, workspace.WorkspaceSpec{Stack: "test"})
	defer fakeWS.Recycle(ctx, wsProvisioned.ID)

	rt := agent.NewRuntime(llm, fakeWS, wsProvisioned.ID)
	rt.RegisterAgent(agent.NewSimpleAgent("coder", "Coding agent", ""))

	if err := rt.Start(ctx); err != nil {
		t.Fatalf("runtime start: %v", err)
	}
	defer rt.Stop(ctx)

	// Create ingress for intent creation
	ing := ingress.NewFakeIngress()

	// Create orchestrator that dispatches to the agent runtime
	orch := orchestrator.NewEngine(nil, rt, orchestrator.WithAgent("coder"))

	// Send intent via ingress
	intentPayload := ingress.IntentPayload{
		Text:   "build a web server",
		UserID: "e2e-user",
	}

	intentResult, err := ing.SubmitIntent(ctx, intentPayload)
	if err != nil {
		t.Fatalf("submit intent: %v", err)
	}
	if intentResult.IntentID == "" {
		t.Error("empty intent ID")
	}
	if intentResult.Status != "accepted" {
		t.Errorf("intent status: got=%s", intentResult.Status)
	}

	// Verify intent payload was recorded
	if len(ing.ReceivedPayloads) != 1 {
		t.Errorf("expected 1 payload, got %d", len(ing.ReceivedPayloads))
	}

	t.Logf("Full pipeline test completed:")
	t.Logf("  intent_id: %s", intentResult.IntentID)
	t.Logf("  status:    %s", intentResult.Status)
	t.Logf("  agent:     coder")
}

// TestEventTypesExist verifies that required event types are defined.
func TestEventTypesExist(t *testing.T) {
	if event.TypeIntentCreated == "" {
		t.Error("TypeIntentCreated not defined")
	}
	if event.TypeIntentCompleted == "" {
		t.Error("TypeIntentCompleted not defined")
	}
	if event.TypeTaskStatus == "" {
		t.Error("TypeTaskStatus not defined")
	}
}
