//go:build integration

package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/provider"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/workspace"
)

// TestIntegrationSimpleAgentRun verifies the full agent loop:
// provider + workspace → agent runs a task with a tool call.
func TestIntegrationSimpleAgentRun(t *testing.T) {
	llm := provider.NewFakeProvider(
		provider.CompletionResponse{
			Message:      provider.Message{Role: provider.RoleAssistant, Content: "I'll execute a command.\n```tool\n{\"name\": \"execute_command\", \"args\": {\"command\": \"echo hello from agent\"}}\n```\n"},
			FinishReason: "tool_use",
		},
		provider.CompletionResponse{
			Message:      provider.Message{Role: provider.RoleAssistant, Content: "Task complete! The command ran successfully."},
			FinishReason: "end_turn",
		},
	)

	ws := workspace.NewFakeWorkspace()
	wsProvisioned, err := ws.Provision(context.Background(), workspace.WorkspaceSpec{Stack: "integration-test"})
	if err != nil {
		t.Fatalf("provision: %v", err)
	}

	a := NewSimpleAgent("integration-agent", "Integration test agent", "")
	task := Task{ID: "int-1", Description: "run echo hello from agent"}
	tools := DefaultTools(ws, wsProvisioned.ID)

	ctx := NewContext(context.Background(), task, tools, llm, ws)
	result, err := a.Run(ctx)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if result.Status != ResultSuccess {
		t.Errorf("status=%s", result.Status)
	}
	if !strings.Contains(result.Summary, "complete") && !strings.Contains(result.Summary, "success") {
		t.Logf("summary: %s", result.Summary)
	}
	if result.Iterations != 2 {
		t.Errorf("iterations=%d want=2", result.Iterations)
	}
}

// TestIntegrationRuntimeFullCycle runs a task through the Runtime with
// a real FakeProvider and a real FakeWorkspace.
func TestIntegrationRuntimeFullCycle(t *testing.T) {
	llm := provider.NewFakeProvider(
		provider.CompletionResponse{
			Message:      provider.Message{Role: provider.RoleAssistant, Content: "Final answer: hello world"},
			FinishReason: "end_turn",
		},
	)

	ws := workspace.NewFakeWorkspace()
	wsProvisioned, err := ws.Provision(context.Background(), workspace.WorkspaceSpec{Stack: "integration-test"})
	if err != nil {
		t.Fatalf("provision: %v", err)
	}

	r := NewRuntime(llm, ws, wsProvisioned.ID)
	r.RegisterAgent(NewSimpleAgent("coder", "Coding agent", ""))

	result, err := r.RunTask(context.Background(), "coder", Task{
		ID:          "int-2",
		Description: "say hello",
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if result.Status != ResultSuccess {
		t.Errorf("status=%s", result.Status)
	}
	if !strings.Contains(result.Summary, "hello") {
		t.Errorf("summary=%s", result.Summary)
	}
}

// TestIntegrationAgentWithToolExecution verifies that the tool execution
// pipeline works end-to-end with a FakeWorkspace.
func TestIntegrationAgentWithToolExecution(t *testing.T) {
	llm := provider.NewFakeProvider(
		provider.CompletionResponse{
			Message:      provider.Message{Role: provider.RoleAssistant, Content: "Let me run a command.\n```tool\n{\"name\": \"execute_command\", \"args\": {\"command\": \"echo success\"}}\n```\n"},
			FinishReason: "tool_use",
		},
		provider.CompletionResponse{
			Message:      provider.Message{Role: provider.RoleAssistant, Content: "The command succeeded."},
			FinishReason: "end_turn",
		},
	)

	ws := workspace.NewFakeWorkspace()
	wsProvisioned, err := ws.Provision(context.Background(), workspace.WorkspaceSpec{Stack: "integration-test"})
	if err != nil {
		t.Fatalf("provision: %v", err)
	}

	// Execute the tool via workspace
	tools := DefaultTools(ws, wsProvisioned.ID)
	task := Task{ID: "int-3", Description: "run a command"}
	ctx := NewContext(context.Background(), task, tools, llm, ws)

	a := NewSimpleAgent("tool-agent", "Uses tools", "")
	result, err := a.Run(ctx)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if result.Iterations == 0 {
		t.Error("expected at least 1 iteration")
	}
	t.Logf("result: status=%s iterations=%d summary=%q", result.Status, result.Iterations, result.Summary)
}

// TestIntegrationRuntimeStreaming verifies the streaming execution path.
func TestIntegrationRuntimeStreaming(t *testing.T) {
	llm := provider.NewFakeProvider(
		provider.CompletionResponse{
			Message:      provider.Message{Role: provider.RoleAssistant, Content: "Final output from stream"},
			FinishReason: "end_turn",
		},
	)

	ws := workspace.NewFakeWorkspace()
	wsProvisioned, err := ws.Provision(context.Background(), workspace.WorkspaceSpec{Stack: "integration-stream"})
	if err != nil {
		t.Fatalf("provision: %v", err)
	}

	r := NewRuntime(llm, ws, wsProvisioned.ID)
	r.RegisterAgent(NewSimpleAgent("streamer", "Streaming agent", ""))

	ch, err := r.RunTaskStream(context.Background(), "streamer", Task{
		ID:          "int-4",
		Description: "stream test",
	})
	if err != nil {
		t.Fatalf("stream: %v", err)
	}

	var events []AgentEvent
	for evt := range ch {
		events = append(events, evt)
	}

	if len(events) == 0 {
		t.Fatal("no events received")
	}

	// Last event should be Done
	last := events[len(events)-1]
	if !last.Done {
		t.Error("last event should be Done")
	}
	if last.Result == nil {
		t.Error("Done event should have a Result")
	}
	if last.Result != nil && last.Result.Summary == "" {
		t.Error("Result should have a summary")
	}
}
