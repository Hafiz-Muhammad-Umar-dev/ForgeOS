package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestOpenRouterName(t *testing.T) {
	o := NewOpenRouter()
	if o == nil {
		t.Fatal("nil provider")
	}
}

func TestOpenRouterDefaultConfig(t *testing.T) {
	cfg := DefaultOpenRouterConfig()
	if cfg.BaseURL != "https://openrouter.ai/api/v1" {
		t.Errorf("baseURL=%s", cfg.BaseURL)
	}
	if cfg.DefaultModel != "openai/gpt-4o" {
		t.Errorf("model=%s", cfg.DefaultModel)
	}
	if cfg.MaxRetries != 3 {
		t.Errorf("retries=%d", cfg.MaxRetries)
	}
}

func TestOpenRouterOptions(t *testing.T) {
	o := NewOpenRouter(
		WithOpenRouterAPIKey("test-key"),
		WithOpenRouterBaseURL("https://custom.example.com"),
		WithOpenRouterModel("custom-model"),
		WithOpenRouterTimeout(5*time.Second),
		WithOpenRouterRetries(5),
	)
	if o.config.APIKey != "test-key" {
		t.Errorf("key=%s", o.config.APIKey)
	}
	if o.config.BaseURL != "https://custom.example.com" {
		t.Errorf("baseURL=%s", o.config.BaseURL)
	}
	if o.config.DefaultModel != "custom-model" {
		t.Errorf("model=%s", o.config.DefaultModel)
	}
}

func newOpenRouterTestServer(t *testing.T, response any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
}

func TestOpenRouterComplete(t *testing.T) {
	srv := newOpenRouterTestServer(t, openRouterResponse{
		Choices: []openRouterChoice{
			{
				Message:      openRouterMessage{Role: "assistant", Content: "Hello!"},
				FinishReason: "stop",
			},
		},
		Usage: &openRouterUsage{PromptTokens: 10, CompletionTokens: 5},
	})
	defer srv.Close()

	o := NewOpenRouter(WithOpenRouterBaseURL(srv.URL), WithOpenRouterAPIKey("test-key"))

	resp, err := o.Complete(context.Background(), CompletionRequest{
		Messages: []Message{{Role: RoleUser, Content: "Hi"}},
	})
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if resp.Message.Content != "Hello!" {
		t.Errorf("content=%s", resp.Message.Content)
	}
	if resp.Usage.InputTokens != 10 {
		t.Errorf("input=%d", resp.Usage.InputTokens)
	}
	if resp.Usage.OutputTokens != 5 {
		t.Errorf("output=%d", resp.Usage.OutputTokens)
	}
}

func TestOpenRouterCompleteWithSystemPrompt(t *testing.T) {
	srv := newOpenRouterTestServer(t, openRouterResponse{
		Choices: []openRouterChoice{
			{Message: openRouterMessage{Role: "assistant", Content: "ok"}, FinishReason: "stop"},
		},
	})
	defer srv.Close()

	o := NewOpenRouter(WithOpenRouterBaseURL(srv.URL), WithOpenRouterAPIKey("key"))
	resp, err := o.Complete(context.Background(), CompletionRequest{
		System:   "Be concise.",
		Messages: []Message{{Role: RoleUser, Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if resp.Message.Content == "" {
		t.Error("empty response")
	}
}

func TestOpenRouterCompleteEmptyResponse(t *testing.T) {
	srv := newOpenRouterTestServer(t, openRouterResponse{Choices: []openRouterChoice{}})
	defer srv.Close()

	o := NewOpenRouter(WithOpenRouterBaseURL(srv.URL), WithOpenRouterAPIKey("key"))
	resp, err := o.Complete(context.Background(), CompletionRequest{
		Messages: []Message{{Role: RoleUser, Content: "test"}},
	})
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if resp.Message.Content != "" {
		t.Errorf("expected empty, got %s", resp.Message.Content)
	}
}

func TestOpenRouterCapabilities(t *testing.T) {
	o := NewOpenRouter(WithOpenRouterAPIKey("key"))
	caps := o.Capabilities()
	if caps.Provider != "openrouter" {
		t.Errorf("provider=%s", caps.Provider)
	}
	if !caps.Streaming {
		t.Error("should support streaming")
	}
}

func TestOpenRouterStream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.WriteHeader(http.StatusOK)

		chunks := []string{
			`data: {"choices":[{"delta":{"content":"Hello"}}]}`,
			`data: {"choices":[{"delta":{"content":" world"}}]}`,
			`data: [DONE]`,
		}
		for _, c := range chunks {
			_, _ = w.Write([]byte(c + "\n\n"))
			w.(http.Flusher).Flush()
		}
	}))
	defer srv.Close()

	o := NewOpenRouter(WithOpenRouterBaseURL(srv.URL), WithOpenRouterAPIKey("key"))
	ch, err := o.Stream(context.Background(), CompletionRequest{
		Messages: []Message{{Role: RoleUser, Content: "Hi"}},
	})
	if err != nil {
		t.Fatalf("stream: %v", err)
	}

	var full string
	for chunk := range ch {
		if chunk.Err != nil {
			t.Fatalf("chunk err: %v", chunk.Err)
		}
		full += chunk.Content
	}
	if full != "Hello world" {
		t.Errorf("streamed: got=%s", full)
	}
}

func TestOpenRouterRateLimited(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(openRouterError{
			Error: openRouterErrorBody{Message: "Rate limited", Type: "rate_limit_error"},
		})
	}))
	defer srv.Close()

	o := NewOpenRouter(WithOpenRouterBaseURL(srv.URL), WithOpenRouterAPIKey("key"))
	_, err := o.Complete(context.Background(), CompletionRequest{
		Messages: []Message{{Role: RoleUser, Content: "Hi"}},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), ErrRateLimited.Error()) {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestOpenRouterAuthFailed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(openRouterError{
			Error: openRouterErrorBody{Message: "Invalid API key", Type: "authentication_error"},
		})
	}))
	defer srv.Close()

	o := NewOpenRouter(WithOpenRouterBaseURL(srv.URL), WithOpenRouterAPIKey("bad-key"))
	_, err := o.Complete(context.Background(), CompletionRequest{
		Messages: []Message{{Role: RoleUser, Content: "Hi"}},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), ErrAuthFailed.Error()) {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestOpenRouterContextWindowExceeded(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(openRouterError{
			Error: openRouterErrorBody{
				Message: "This model's maximum context length is 8192 tokens",
				Type:    "invalid_request_error",
			},
		})
	}))
	defer srv.Close()

	o := NewOpenRouter(WithOpenRouterBaseURL(srv.URL), WithOpenRouterAPIKey("key"))
	_, err := o.Complete(context.Background(), CompletionRequest{
		Messages: []Message{{Role: RoleUser, Content: strings.Repeat("a", 10000)}},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), ErrContextWindowExceeded.Error()) {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestOpenRouterServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	o := NewOpenRouter(WithOpenRouterBaseURL(srv.URL), WithOpenRouterAPIKey("key"))
	_, err := o.Complete(context.Background(), CompletionRequest{
		Messages: []Message{{Role: RoleUser, Content: "Hi"}},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), ErrProviderUnavailable.Error()) {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestOpenRouterContextCancellation(t *testing.T) {
	srv := newOpenRouterTestServer(t, openRouterResponse{
		Choices: []openRouterChoice{
			{Message: openRouterMessage{Role: "assistant", Content: "should not get here"}, FinishReason: "stop"},
		},
	})
	defer srv.Close()

	o := NewOpenRouter(WithOpenRouterBaseURL(srv.URL), WithOpenRouterAPIKey("key"))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := o.Complete(ctx, CompletionRequest{
		Messages: []Message{{Role: RoleUser, Content: "Hi"}},
	})
	if err == nil {
		t.Fatal("expected cancellation error")
	}
	t.Logf("cancellation error (expected): %v", err)
}

func TestOpenRouterBuildRequestWithSystem(t *testing.T) {
	o := NewOpenRouter(WithOpenRouterAPIKey("key"))
	req := CompletionRequest{
		Model:    "test-model",
		System:   "You are helpful.",
		Messages: []Message{{Role: RoleUser, Content: "hello"}},
	}
	orReq := o.buildRequest(req, false)
	if orReq.Model != "test-model" {
		t.Errorf("model=%s", orReq.Model)
	}
	if len(orReq.Messages) != 2 {
		t.Fatalf("messages=%d", len(orReq.Messages))
	}
	if orReq.Messages[0].Content != "You are helpful." {
		t.Errorf("system=%s", orReq.Messages[0].Content)
	}
}

func TestOpenRouterBuildRequestNoSystem(t *testing.T) {
	o := NewOpenRouter(WithOpenRouterAPIKey("key"), WithOpenRouterModel("m1"))
	req := CompletionRequest{
		Messages: []Message{
			{Role: RoleUser, Content: "hello"},
			{Role: RoleAssistant, Content: "world"},
		},
	}
	orReq := o.buildRequest(req, false)
	if len(orReq.Messages) != 2 {
		t.Fatalf("messages=%d", len(orReq.Messages))
	}
	if orReq.Messages[0].Role != "user" {
		t.Errorf("role0=%s", orReq.Messages[0].Role)
	}
}

func TestOpenRouterStreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	o := NewOpenRouter(WithOpenRouterBaseURL(srv.URL), WithOpenRouterAPIKey("key"))
	_, err := o.Stream(context.Background(), CompletionRequest{
		Messages: []Message{{Role: RoleUser, Content: "Hi"}},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestOpenRouterStreamCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, ok := w.(http.Flusher)
		if !ok {
			return
		}
		// Write chunks until the context is cancelled.
		for ctx.Err() == nil {
			_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"a\"}}]}\n\n"))
			flusher.Flush()
		}
	}))

	o := NewOpenRouter(WithOpenRouterBaseURL(srv.URL), WithOpenRouterAPIKey("key"))

	ch, err := o.Stream(ctx, CompletionRequest{
		Messages: []Message{{Role: RoleUser, Content: "Hi"}},
	})
	if err != nil {
		srv.Close()
		t.Fatalf("stream: %v", err)
	}

	// Read one chunk, then cancel
	for chunk := range ch {
		if chunk.Content != "" {
			cancel()
			break
		}
	}
	srv.Close()
	t.Log("stream cancelled successfully")
}
