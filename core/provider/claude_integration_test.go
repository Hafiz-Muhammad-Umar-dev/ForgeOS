//go:build integration

package provider

import (
	"context"
	"os"
	"strings"
	"testing"
)

// TestIntegrationClaudeComplete sends a real completion request to the
// Anthropic API. Requires the ANTHROPIC_API_KEY environment variable.
//
// Usage: go test -tags=integration -run TestIntegrationClaudeComplete ./core/provider/...
func TestIntegrationClaudeComplete(t *testing.T) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("ANTHROPIC_API_KEY not set")
	}

	cl := NewClaude(WithClaudeAPIKey(apiKey))

	resp, err := cl.Complete(context.Background(), CompletionRequest{
		Model: "claude-sonnet-4-20250514",
		Messages: []Message{
			{Role: RoleUser, Content: "Reply with exactly one word: hello"},
		},
		MaxTokens: 10,
	})
	if err != nil {
		t.Fatalf("complete: %v", err)
	}

	if resp.Message.Role != RoleAssistant {
		t.Errorf("role: got=%s", resp.Message.Role)
	}
	if resp.Message.Content == "" {
		t.Error("empty response content")
	}
	if resp.Usage.InputTokens == 0 || resp.Usage.OutputTokens == 0 {
		t.Errorf("usage: got=%+v", resp.Usage)
	}
	t.Logf("response: %q (in=%d, out=%d)", resp.Message.Content, resp.Usage.InputTokens, resp.Usage.OutputTokens)
}

// TestIntegrationClaudeStream sends a real streaming request to the
// Anthropic API. Requires the ANTHROPIC_API_KEY environment variable.
//
// Usage: go test -tags=integration -run TestIntegrationClaudeStream ./core/provider/...
func TestIntegrationClaudeStream(t *testing.T) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("ANTHROPIC_API_KEY not set")
	}

	cl := NewClaude(WithClaudeAPIKey(apiKey))

	ch, err := cl.Stream(context.Background(), CompletionRequest{
		Model: "claude-sonnet-4-20250514",
		Messages: []Message{
			{Role: RoleUser, Content: "Count from one to three. Use commas."},
		},
		MaxTokens: 50,
	})
	if err != nil {
		t.Fatalf("stream: %v", err)
	}

	var full strings.Builder
	chunkCount := 0
	var finalUsage Usage

	for chunk := range ch {
		if chunk.Err != nil {
			t.Fatalf("chunk error: %v", chunk.Err)
		}
		full.WriteString(chunk.Content)
		chunkCount++
		if chunk.Done {
			finalUsage = chunk.Usage
		}
	}

	if full.Len() == 0 {
		t.Error("empty streamed response")
	}
	if chunkCount == 0 {
		t.Error("no chunks received")
	}
	if finalUsage.OutputTokens == 0 {
		t.Errorf("expected output tokens, got=%+v", finalUsage)
	}
	t.Logf("streamed %d chunks: %q (in=%d, out=%d)", chunkCount, full.String(), finalUsage.InputTokens, finalUsage.OutputTokens)
}

// TestIntegrationClaudeSystemPrompt verifies system prompt handling.
func TestIntegrationClaudeSystemPrompt(t *testing.T) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("ANTHROPIC_API_KEY not set")
	}

	cl := NewClaude(WithClaudeAPIKey(apiKey))

	resp, err := cl.Complete(context.Background(), CompletionRequest{
		Model:  "claude-sonnet-4-20250514",
		System: "You are a terse assistant. Reply in as few words as possible.",
		Messages: []Message{
			{Role: RoleUser, Content: "What color is the sky?"},
		},
		MaxTokens: 50,
	})
	if err != nil {
		t.Fatalf("complete: %v", err)
	}

	t.Logf("system prompt response: %q", resp.Message.Content)
	if resp.Message.Content == "" {
		t.Error("empty response")
	}
}

// TestIntegrationClaudeContextWindow verifies error handling for oversized
// prompts.
func TestIntegrationClaudeContextWindow(t *testing.T) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("ANTHROPIC_API_KEY not set")
	}

	cl := NewClaude(WithClaudeAPIKey(apiKey))

	// Send a very large prompt to trigger context window error
	largeContent := strings.Repeat("hello world ", 100000)

	_, err := cl.Complete(context.Background(), CompletionRequest{
		Model: "claude-sonnet-4-20250514",
		Messages: []Message{
			{Role: RoleUser, Content: largeContent},
		},
		MaxTokens: 10,
	})
	if err == nil {
		t.Skip("prompt was accepted (may vary by model/account limits)")
	}

	errStr := err.Error()
	t.Logf("context error (expected): %v", err)
	if !strings.Contains(errStr, ErrContextWindowExceeded.Error()) &&
		!strings.Contains(errStr, ErrBadRequest.Error()) {
		t.Errorf("unexpected error type: %v", err)
	}
}
