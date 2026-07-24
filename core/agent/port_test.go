package agent

import (
	"context"
	"strings"
	"testing"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/provider"
)

func TestResultStatusValues(t *testing.T) {
	if ResultSuccess != "success" {
		t.Errorf("ResultSuccess=%s", ResultSuccess)
	}
	if ResultFailed != "failed" {
		t.Errorf("ResultFailed=%s", ResultFailed)
	}
	if ResultMaxIterations != "max_iterations" {
		t.Errorf("ResultMaxIterations=%s", ResultMaxIterations)
	}
	if ResultCancelled != "cancelled" {
		t.Errorf("ResultCancelled=%s", ResultCancelled)
	}
}

func TestArtifact(t *testing.T) {
	a := Artifact{Name: "test.txt", Type: "file", Content: []byte("hello")}
	if a.Name != "test.txt" {
		t.Errorf("name=%s", a.Name)
	}
	if string(a.Content) != "hello" {
		t.Errorf("content=%s", a.Content)
	}
}

func TestTaskDefaults(t *testing.T) {
	task := Task{ID: "t1", Description: "do something"}
	if task.ID != "t1" {
		t.Errorf("id=%s", task.ID)
	}
	if task.MaxIterations != 0 {
		t.Errorf("maxIter=%d", task.MaxIterations)
	}
}

func TestToolSetFind(t *testing.T) {
	ts := ToolSet{
		{Name: "tool_a", Description: "Tool A"},
		{Name: "tool_b", Description: "Tool B"},
	}

	if found := ts.Find("tool_a"); found == nil || found.Name != "tool_a" {
		t.Error("tool_a not found")
	}
	if found := ts.Find("tool_b"); found == nil || found.Name != "tool_b" {
		t.Error("tool_b not found")
	}
	if found := ts.Find("nonexistent"); found != nil {
		t.Error("nonexistent tool should not be found")
	}
}

func TestToolSetNames(t *testing.T) {
	ts := ToolSet{
		{Name: "alpha"},
		{Name: "beta"},
	}
	names := ts.Names()
	if len(names) != 2 || names[0] != "alpha" || names[1] != "beta" {
		t.Errorf("names=%v", names)
	}
}

func TestSentinelErrors(t *testing.T) {
	errs := []struct {
		err   error
		label string
	}{
		{ErrMaxIterations, "ErrMaxIterations"},
		{ErrTaskFailed, "ErrTaskFailed"},
		{ErrToolNotFound, "ErrToolNotFound"},
		{ErrToolExecutionFailed, "ErrToolExecutionFailed"},
		{ErrProviderFailed, "ErrProviderFailed"},
		{ErrAgentNotReady, "ErrAgentNotReady"},
	}
	for _, tt := range errs {
		t.Run(tt.label, func(t *testing.T) {
			if tt.err == nil {
				t.Fatal("error is nil")
			}
		})
	}
}

func TestAgentEventTypes(t *testing.T) {
	// Content event
	e := AgentEvent{Content: "thinking..."}
	if e.Content != "thinking..." {
		t.Errorf("content=%s", e.Content)
	}
	if e.Done {
		t.Error("should not be done")
	}

	// Done event
	e2 := AgentEvent{Done: true, Result: &Result{Summary: "done"}}
	if !e2.Done {
		t.Error("should be done")
	}
	if e2.Result.Summary != "done" {
		t.Errorf("summary=%s", e2.Result.Summary)
	}

	// Error event
	e3 := AgentEvent{Err: ErrTaskFailed}
	if e3.Err != ErrTaskFailed {
		t.Errorf("err=%v", e3.Err)
	}

	// ToolCall event
	tc := &ToolCall{Tool: "echo", Args: map[string]any{"msg": "hi"}}
	e4 := AgentEvent{ToolCall: tc}
	if e4.ToolCall.Tool != "echo" {
		t.Errorf("tool=%s", e4.ToolCall.Tool)
	}
}

// ---------------------------------------------------------------------------
// FakeAgent tests
// ---------------------------------------------------------------------------

func TestFakeAgent(t *testing.T) {
	fa := NewFakeAgent("test-agent")
	if fa.Name() != "test-agent" {
		t.Errorf("name=%s", fa.Name())
	}
	if fa.Description() != "fake agent for testing" {
		t.Errorf("desc=%s", fa.Description())
	}
}

func TestFakeAgentRun(t *testing.T) {
	fa := NewFakeAgent("test")
	task := Task{ID: "t1", Description: "test task"}
	ctx := NewContext(context.Background(), task, ToolSet{}, NewEchoProvider(), nil)

	result, err := fa.Run(ctx)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if result.Summary != "fake result" {
		t.Errorf("summary=%s", result.Summary)
	}
	if fa.RunCount.Load() != 1 {
		t.Errorf("count=%d", fa.RunCount.Load())
	}
	if len(fa.ReceivedTasks) != 1 {
		t.Errorf("tasks=%d", len(fa.ReceivedTasks))
	}
}

func TestFakeAgentRunError(t *testing.T) {
	fa := NewFakeAgent("test")
	fa.RunError = ErrTaskFailed
	task := Task{ID: "t1", Description: "test"}
	ctx := NewContext(context.Background(), task, ToolSet{}, NewEchoProvider(), nil)

	_, err := fa.Run(ctx)
	if err != ErrTaskFailed {
		t.Errorf("err=%v", err)
	}
}

func TestFakeAgentCustomRunFunc(t *testing.T) {
	fa := NewFakeAgent("test")
	fa.RunFunc = func(ctx Context) (*Result, error) {
		return &Result{Summary: "custom", Status: ResultSuccess}, nil
	}
	task := Task{ID: "t1", Description: "test"}
	ctx := NewContext(context.Background(), task, ToolSet{}, NewEchoProvider(), nil)

	result, err := fa.Run(ctx)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if result.Summary != "custom" {
		t.Errorf("summary=%s", result.Summary)
	}
}

// ---------------------------------------------------------------------------
// EchoProvider tests
// ---------------------------------------------------------------------------

func TestEchoProviderComplete(t *testing.T) {
	ep := NewEchoProvider()
	resp, err := ep.Complete(context.Background(), provider.CompletionRequest{
		Messages: []provider.Message{
			{Role: provider.RoleUser, Content: "hello"},
		},
	})
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if resp.Message.Content != "hello" {
		t.Errorf("content=%s", resp.Message.Content)
	}
	if ep.CallCount.Load() != 1 {
		t.Errorf("count=%d", ep.CallCount.Load())
	}
}

func TestEchoProviderEmptyMessages(t *testing.T) {
	ep := NewEchoProvider()
	resp, err := ep.Complete(context.Background(), provider.CompletionRequest{})
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if resp.Message.Content != "" {
		t.Errorf("content=%s", resp.Message.Content)
	}
}

// ---------------------------------------------------------------------------
// Context tests
// ---------------------------------------------------------------------------

func TestNewContext(t *testing.T) {
	task := Task{ID: "t1", Description: "test"}
	tools := ToolSet{{Name: "tool1"}}
	provider := NewEchoProvider()

	ctx := NewContext(context.Background(), task, tools, provider, nil)
	if ctx.Task.ID != "t1" {
		t.Errorf("task id=%s", ctx.Task.ID)
	}
	if len(ctx.Tools) != 1 {
		t.Errorf("tools=%d", len(ctx.Tools))
	}
	if ctx.Provider == nil {
		t.Error("provider is nil")
	}
}

func TestContextCancellation(t *testing.T) {
	task := Task{ID: "t1", Description: "test"}

	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	ctx := NewContext(cancelledCtx, task, ToolSet{}, NewEchoProvider(), nil)

	select {
	case <-ctx.Done():
		// Expected
	default:
		t.Error("context should be done")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.DefaultMaxIterations != 10 {
		t.Errorf("maxIter=%d", cfg.DefaultMaxIterations)
	}
	if !cfg.PublishEvents {
		t.Error("should publish events by default")
	}
}

func TestParseToolCall(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		wantTool string
		wantArg  string
		wantNil  bool
	}{
		{
			name:     "valid tool call",
			output:   "Let me run that.\n```tool\n{\"name\": \"execute_command\", \"args\": {\"command\": \"ls\"}}\n```\n",
			wantTool: "execute_command",
			wantArg:  "ls",
		},
		{
			name:     "no tool call",
			output:   "The answer is 42.",
			wantNil:  true,
		},
		{
			name:     "empty output",
			output:   "",
			wantNil:  true,
		},
		{
			name:     "malformed JSON",
			output:   "```tool\n{invalid}\n```",
			wantNil:  true,
		},
		{
			name:     "missing tool name",
			output:   "```tool\n{\"args\": {\"cmd\": \"ls\"}}\n```",
			wantNil:  true,
		},
		{
			name:     "multiple tool blocks - uses first",
			output:   "```tool\n{\"name\": \"first\", \"args\": {}}\n```\n...\n```tool\n{\"name\": \"second\", \"args\": {}}\n```",
			wantTool: "first",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := parseToolCall(tt.output)
			if tt.wantNil {
				if tc != nil {
					t.Errorf("expected nil, got %+v", tc)
				}
				return
			}
			if tc == nil {
				t.Fatal("expected tool call, got nil")
			}
			if tc.Tool != tt.wantTool {
				t.Errorf("tool: got=%s want=%s", tc.Tool, tt.wantTool)
			}
			if arg, ok := tc.Args["command"]; ok && arg != tt.wantArg {
				t.Errorf("arg: got=%v want=%s", arg, tt.wantArg)
			}
		})
	}
}

func TestFormatToolResult(t *testing.T) {
	result := ToolResult{Output: "hello", ExitCode: 0}
	formatted := formatToolResult("echo", result)
	if !strings.Contains(formatted, "hello") {
		t.Errorf("missing output: %s", formatted)
	}
	if !strings.Contains(formatted, "Exit code: 0") {
		t.Errorf("missing exit code: %s", formatted)
	}
}

func TestFormatToolResultWithError(t *testing.T) {
	result := ToolResult{Output: "out", Error: "err", ExitCode: 1}
	formatted := formatToolResult("bad_tool", result)
	if !strings.Contains(formatted, "err") {
		t.Errorf("missing stderr: %s", formatted)
	}
}
