package provider

import (
	"context"
	"fmt"
	"sync/atomic"
)

// FakeProvider is an in-memory LLMProvider implementation for testing.
// It returns predefined responses and records all received requests.
type FakeProvider struct {
	// CompleteFunc overrides the Complete behavior. If nil, a default
	// implementation using Responses is used.
	CompleteFunc func(ctx context.Context, req CompletionRequest) (CompletionResponse, error)

	// StreamFunc overrides the Stream behavior. If nil, a default
	// implementation using Responses is used.
	StreamFunc func(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error)

	// Responses is the default set of responses for Complete and Stream.
	// When set, each call consumes and returns the next response.
	// If empty, a default "Hello!" response is used.
	Responses []CompletionResponse

	// RequestsReceived records every CompletionRequest received for
	// assertions in tests.
	RequestsReceived []CompletionRequest

	// RequestCount is the total number of requests received.
	RequestCount atomic.Int64

	// ConfiguredCapabilities is returned by Capabilities(). If nil, a
	// sensible default is used.
	ConfiguredCapabilities *Capabilities
}

// NewFakeProvider creates a FakeProvider with an optional set of responses.
func NewFakeProvider(responses ...CompletionResponse) *FakeProvider {
	if len(responses) == 0 {
		responses = []CompletionResponse{
			{
				Message:      Message{Role: RoleAssistant, Content: "Hello! How can I help you?"},
				FinishReason: "end_turn",
				Usage:        Usage{InputTokens: 10, OutputTokens: 5},
			},
		}
	}
	return &FakeProvider{Responses: responses}
}

// Complete returns the next predefined response or an error if exhausted.
func (f *FakeProvider) Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
	f.RequestCount.Add(1)
	f.RequestsReceived = append(f.RequestsReceived, req)

	if f.CompleteFunc != nil {
		return f.CompleteFunc(ctx, req)
	}

	if len(f.Responses) == 0 {
		return CompletionResponse{}, fmt.Errorf("fake: no more responses")
	}

	resp := f.Responses[0]
	f.Responses = f.Responses[1:]
	return resp, nil
}

// Stream returns a channel streaming the next predefined response chunks.
func (f *FakeProvider) Stream(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error) {
	f.RequestCount.Add(1)
	f.RequestsReceived = append(f.RequestsReceived, req)

	if f.StreamFunc != nil {
		return f.StreamFunc(ctx, req)
	}

	ch := make(chan StreamChunk, 10)

	if len(f.Responses) == 0 {
		close(ch)
		return ch, nil
	}

	resp := f.Responses[0]
	f.Responses = f.Responses[1:]

	go func() {
		defer close(ch)
		for _, r := range splitContent(resp.Message.Content, 10) {
			ch <- StreamChunk{Content: r}
		}
		ch <- StreamChunk{Done: true, Usage: resp.Usage}
	}()

	return ch, nil
}

// Capabilities returns the configured capabilities or a sensible default.
func (f *FakeProvider) Capabilities() Capabilities {
	if f.ConfiguredCapabilities != nil {
		return *f.ConfiguredCapabilities
	}
	return Capabilities{
		Provider:  "fake",
		Models:    []string{"fake-model"},
		Streaming: true,
		MaxTokens: 4096,
	}
}

// splitContent splits a string into chunks for fake streaming.
func splitContent(s string, chunkSize int) []string {
	if len(s) == 0 {
		return []string{""}
	}
	var chunks []string
	runes := []rune(s)
	for i := 0; i < len(runes); i += chunkSize {
		end := i + chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[i:end]))
	}
	return chunks
}

// Compile-time check.
var _ DeployProvider = (*FakeDeployProvider)(nil)

// FakeDeployProvider is an in-memory DeployProvider implementation for testing.
type FakeDeployProvider struct {
	DeployFunc func(ctx context.Context, req DeployRequest) (DeployResponse, error)
	StatusFunc func(ctx context.Context, deploymentID string) (DeployStatus, error)
	DefaultResponse DeployResponse
	DefaultStatus   DeployStatus
	DeployCount     atomic.Int64
	ReceivedRequests []DeployRequest
	ConfiguredCapabilities *DeployCapabilities
}

func NewFakeDeployProvider() *FakeDeployProvider {
	return &FakeDeployProvider{
		DefaultResponse: DeployResponse{
			DeploymentID: "deploy-fake-1",
			URL:          "https://fake-deploy.example.com",
			Status:       DeployStatusReady,
		},
		DefaultStatus: DeployStatusReady,
	}
}

func (f *FakeDeployProvider) Deploy(ctx context.Context, req DeployRequest) (DeployResponse, error) {
	f.DeployCount.Add(1)
	f.ReceivedRequests = append(f.ReceivedRequests, req)
	if f.DeployFunc != nil {
		return f.DeployFunc(ctx, req)
	}
	return f.DefaultResponse, nil
}

func (f *FakeDeployProvider) Status(_ context.Context, _ string) (DeployStatus, error) {
	if f.StatusFunc != nil {
		return f.StatusFunc(nil, "")
	}
	return f.DefaultStatus, nil
}

func (f *FakeDeployProvider) Capabilities() DeployCapabilities {
	if f.ConfiguredCapabilities != nil {
		return *f.ConfiguredCapabilities
	}
	return DeployCapabilities{Provider: "fake", Regions: []string{"us-east"}}
}
