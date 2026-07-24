package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/provider"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/workspace"
)

func TestSimpleAgentName(t *testing.T) {
	a := NewSimpleAgent("coder", "Writes code", "")
	if a.Name() != "coder" {
		t.Errorf("name=%s", a.Name())
	}
	if a.Description() != "Writes code" {
		t.Errorf("desc=%s", a.Description())
	}
}

func TestSimpleAgentRunNoTools(t *testing.T) {
	llm := NewEchoProvider()
	a := NewSimpleAgent("echo", "Echoes input", "")

	task := Task{ID: "t1", Description: "say hello"}
	ctx := NewContext(context.Background(), task, ToolSet{}, llm, nil)

	result, err := a.Run(ctx)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if result.Status != ResultSuccess {
		t.Errorf("status=%s", result.Status)
	}
	if !strings.Contains(result.Summary, "say hello") {
		t.Errorf("summary=%s", result.Summary)
	}
	if result.Iterations != 1 {
		t.Errorf("iterations=%d", result.Iterations)
	}
}

func TestSimpleAgentRunWithTool(t *testing.T) {
	llm := &toolCallProvider{
		responses: []provider.CompletionResponse{
			{
				Message:      provider.Message{Role: provider.RoleAssistant, Content: "Let me run that.\n```tool\n{\"name\": \"execute_command\", \"args\": {\"command\": \"echo hi\"}}\n```\n"},
				FinishReason: "tool_use",
			},
			{
				Message:      provider.Message{Role: provider.RoleAssistant, Content: "Done! The result was hi."},
				FinishReason: "end_turn",
			},
		},
	}
	a := NewSimpleAgent("tool-user", "Uses tools", "")

	task := Task{ID: "t1", Description: "run echo hi"}
	ws := workspace.NewFakeWorkspace()
	wsProvisioned, _ := ws.Provision(context.Background(), workspace.WorkspaceSpec{Stack: "test"})
	tools := DefaultTools(ws, wsProvisioned.ID)

	ctx := NewContext(context.Background(), task, tools, llm, ws)

	result, err := a.Run(ctx)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if result.Status != ResultSuccess {
		t.Errorf("status=%s", result.Status)
	}
	if result.Iterations != 2 {
		t.Errorf("iterations=%d", result.Iterations)
	}
	if !strings.Contains(result.Summary, "Done") {
		t.Errorf("summary=%s", result.Summary)
	}
}

func TestSimpleAgentMaxIterations(t *testing.T) {
	// Provider that always asks for a tool call, causing max iterations.
	llm := &toolCallProvider{
		responses: []provider.CompletionResponse{
			{
				Message:      provider.Message{Role: provider.RoleAssistant, Content: "```tool\n{\"name\": \"execute_command\", \"args\": {\"command\": \"echo loop\"}}\n```\n"},
				FinishReason: "tool_use",
			},
		},
		// Single response — loop will repeat it.
		loop: true,
	}

	a := NewSimpleAgent("looper", "Loops", "")
	task := Task{ID: "t1", Description: "loop test", MaxIterations: 2}

	ws := workspace.NewFakeWorkspace()
	wsProvisioned, _ := ws.Provision(context.Background(), workspace.WorkspaceSpec{Stack: "test"})
	tools := DefaultTools(ws, wsProvisioned.ID)

	ctx := NewContext(context.Background(), task, tools, llm, ws)

	result, err := a.Run(ctx)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if result.Status != ResultMaxIterations {
		t.Errorf("status=%s want=max_iterations", result.Status)
	}
	if result.Iterations != 2 {
		t.Errorf("iterations=%d want=2", result.Iterations)
	}
}

func TestSimpleAgentBuildSystemPrompt(t *testing.T) {
	t.Run("with tools", func(t *testing.T) {
		a := NewSimpleAgent("test", "Test agent", "")
		tools := ToolSet{
			{Name: "echo", Description: "Echoes input"},
		}
		prompt := a.buildSystemPrompt(tools)
		if !strings.Contains(prompt, "echo") {
			t.Errorf("prompt missing tool: %s", prompt)
		}
		if !strings.Contains(prompt, "```tool") {
			t.Errorf("prompt missing tool marker: %s", prompt)
		}
	})

	t.Run("without tools", func(t *testing.T) {
		a := NewSimpleAgent("test", "Test agent", "")
		prompt := a.buildSystemPrompt(ToolSet{})
		if strings.Contains(prompt, "```tool") {
			t.Errorf("prompt should not have tool marker: %s", prompt)
		}
	})
}

// ---------------------------------------------------------------------------
// Tool call provider helper
// ---------------------------------------------------------------------------

type toolCallProvider struct {
	responses []provider.CompletionResponse
	index     int
	loop      bool
}

func (t *toolCallProvider) Capabilities() provider.Capabilities {
	return provider.Capabilities{Provider: "test", Streaming: false, MaxTokens: 100}
}

func (t *toolCallProvider) Complete(ctx context.Context, req provider.CompletionRequest) (provider.CompletionResponse, error) {
	if t.index >= len(t.responses) {
		if t.loop && len(t.responses) > 0 {
			return t.responses[len(t.responses)-1], nil
		}
		return provider.CompletionResponse{
			Message:      provider.Message{Role: provider.RoleAssistant, Content: "final answer"},
			FinishReason: "end_turn",
		}, nil
	}
	resp := t.responses[t.index]
	t.index++
	return resp, nil
}

func (t *toolCallProvider) Stream(ctx context.Context, req provider.CompletionRequest) (<-chan provider.StreamChunk, error) {
	ch := make(chan provider.StreamChunk, 1)
	defer close(ch)
	ch <- provider.StreamChunk{Done: true}
	return ch, nil
}
