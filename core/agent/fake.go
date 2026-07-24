package agent

import (
	"context"
	"sync/atomic"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/provider"
)

// Compile-time checks.
var _ Agent = (*FakeAgent)(nil)

// FakeAgent is an in-memory Agent implementation for testing.
// It returns a configurable result and records all calls.
type FakeAgent struct {
	// NameValue is returned by Name().
	NameValue string

	// DescriptionValue is returned by Description().
	DescriptionValue string

	// RunFunc overrides the Run behavior. If nil, a default
	// implementation using ResultValue is used.
	RunFunc func(ctx Context) (*Result, error)

	// ResultValue is returned by the default Run implementation.
	ResultValue *Result

	// RunError is returned by the default Run implementation when set.
	RunError error

	// RunCount tracks the number of Run calls.
	RunCount atomic.Int64

	// ReceivedContexts records every Context received.
	ReceivedContexts []Context

	// ReceivedTasks records every Task received.
	ReceivedTasks []Task
}

// NewFakeAgent creates a FakeAgent with the given name.
func NewFakeAgent(name string) *FakeAgent {
	return &FakeAgent{
		NameValue:        name,
		DescriptionValue: "fake agent for testing",
		ResultValue: &Result{
			Summary: "fake result",
			Status:  ResultSuccess,
		},
	}
}

// Name returns the agent name.
func (f *FakeAgent) Name() string { return f.NameValue }

// Description returns the agent description.
func (f *FakeAgent) Description() string { return f.DescriptionValue }

// Run records the call and returns the configured result.
func (f *FakeAgent) Run(ctx Context) (*Result, error) {
	f.RunCount.Add(1)
	f.ReceivedContexts = append(f.ReceivedContexts, ctx)
	f.ReceivedTasks = append(f.ReceivedTasks, ctx.Task)

	if f.RunFunc != nil {
		return f.RunFunc(ctx)
	}
	if f.RunError != nil {
		return nil, f.RunError
	}
	if f.ResultValue != nil {
		return f.ResultValue, nil
	}
	return &Result{Summary: "ok", Status: ResultSuccess}, nil
}

// ---------------------------------------------------------------------------
// FakeLLMProvider helpers
// ---------------------------------------------------------------------------

// NewEchoProvider returns a FakeProvider that echoes the user's last message
// as the assistant response. Useful for testing agent loop behavior without
// a real LLM.
func NewEchoProvider() *EchoProvider {
	return &EchoProvider{}
}

// EchoProvider is a test helper that echoes the last user message.
type EchoProvider struct {
	CallCount atomic.Int64
}

// Complete returns the last user message as the assistant response.
func (e *EchoProvider) Complete(ctx context.Context, req provider.CompletionRequest) (provider.CompletionResponse, error) {
	e.CallCount.Add(1)
	content := ""
	if len(req.Messages) > 0 {
		content = req.Messages[len(req.Messages)-1].Content
	}
	return provider.CompletionResponse{
		Message:      provider.Message{Role: provider.RoleAssistant, Content: content},
		FinishReason: "end_turn",
		Usage:        provider.Usage{InputTokens: 10, OutputTokens: 5},
	}, nil
}

// Capabilities returns the provider capabilities.
func (e *EchoProvider) Capabilities() provider.Capabilities {
	return provider.Capabilities{Provider: "echo", Streaming: false, MaxTokens: 100}
}

// Stream implements provider.LLMProvider. Not supported in the echo provider.
func (e *EchoProvider) Stream(ctx context.Context, req provider.CompletionRequest) (<-chan provider.StreamChunk, error) {
	ch := make(chan provider.StreamChunk, 1)
	ch <- provider.StreamChunk{Done: true, Usage: provider.Usage{InputTokens: 0, OutputTokens: 0}}
	close(ch)
	return ch, nil
}
