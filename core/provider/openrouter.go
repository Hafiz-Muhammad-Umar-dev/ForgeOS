package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Compile-time check.
var _ LLMProvider = (*OpenRouter)(nil)

// OpenRouterConfig configures the OpenRouter LLM provider adapter.
// OpenRouter provides a unified API for many LLM providers.
type OpenRouterConfig struct {
	// APIKey is the OpenRouter API key.
	APIKey string

	// BaseURL is the OpenRouter API base URL.
	// Defaults to "https://openrouter.ai/api/v1".
	BaseURL string

	// DefaultModel is the model to use when CompletionRequest.Model is empty.
	// Configurable via environment or options.
	DefaultModel string

	// DefaultMaxTokens is the max_tokens to use when CompletionRequest.MaxTokens is zero.
	DefaultMaxTokens int

	// HTTPClient is the HTTP client used for API calls.
	HTTPClient *http.Client

	// Timeout for a single request. Zero means no timeout.
	Timeout time.Duration

	// MaxRetries is the maximum number of retry attempts for transient failures.
	MaxRetries int
}

// DefaultOpenRouterConfig returns a sensible default configuration.
func DefaultOpenRouterConfig() OpenRouterConfig {
	return OpenRouterConfig{
		BaseURL:          "https://openrouter.ai/api/v1",
		DefaultModel:     "openai/gpt-4o",
		DefaultMaxTokens: 4096,
		Timeout:          120 * time.Second,
		MaxRetries:       3,
	}
}

// OpenRouterOption configures the OpenRouter adapter.
type OpenRouterOption func(*OpenRouterConfig)

func WithOpenRouterAPIKey(key string) OpenRouterOption {
	return func(c *OpenRouterConfig) { c.APIKey = key }
}

func WithOpenRouterBaseURL(url string) OpenRouterOption {
	return func(c *OpenRouterConfig) { c.BaseURL = url }
}

func WithOpenRouterModel(model string) OpenRouterOption {
	return func(c *OpenRouterConfig) { c.DefaultModel = model }
}

func WithOpenRouterTimeout(timeout time.Duration) OpenRouterOption {
	return func(c *OpenRouterConfig) { c.Timeout = timeout }
}

func WithOpenRouterRetries(n int) OpenRouterOption {
	return func(c *OpenRouterConfig) { c.MaxRetries = n }
}

// OpenRouter is an LLMProvider adapter for the OpenRouter API.
// OpenRouter exposes an OpenAI-compatible chat completions endpoint.
type OpenRouter struct {
	config OpenRouterConfig
	client *http.Client
}

// NewOpenRouter creates a new OpenRouter provider adapter.
func NewOpenRouter(opts ...OpenRouterOption) *OpenRouter {
	cfg := DefaultOpenRouterConfig()
	for _, fn := range opts {
		fn(&cfg)
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: cfg.Timeout}
	}
	return &OpenRouter{config: cfg, client: client}
}

// ---------------------------------------------------------------------------
// OpenAI-compatible Chat Completions types
// ---------------------------------------------------------------------------

type openRouterMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openRouterRequest struct {
	Model     string              `json:"model"`
	Messages  []openRouterMessage `json:"messages"`
	MaxTokens int                 `json:"max_tokens,omitempty"`
	Stream    bool                `json:"stream,omitempty"`
}

type openRouterResponse struct {
	ID      string             `json:"id"`
	Choices []openRouterChoice `json:"choices"`
	Usage   *openRouterUsage   `json:"usage,omitempty"`
}

type openRouterChoice struct {
	Message      openRouterMessage `json:"message"`
	FinishReason string            `json:"finish_reason"`
}

type openRouterUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
}

type openRouterError struct {
	Error openRouterErrorBody `json:"error"`
}

type openRouterErrorBody struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code,omitempty"`
}

type openRouterStreamChoice struct {
	Delta openRouterDelta `json:"delta"`
}

type openRouterDelta struct {
	Content string `json:"content"`
}

type openRouterStreamResponse struct {
	Choices []openRouterStreamChoice `json:"choices"`
	Usage   *openRouterUsage         `json:"usage,omitempty"`
}

// ---------------------------------------------------------------------------
// LLMProvider implementation
// ---------------------------------------------------------------------------

// Complete sends a chat completion request to OpenRouter.
func (o *OpenRouter) Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
	orReq := o.buildRequest(req, false)
	body, err := o.doRequest(ctx, orReq)
	if err != nil {
		return CompletionResponse{}, err
	}

	var orResp openRouterResponse
	if err := json.Unmarshal(body, &orResp); err != nil {
		return CompletionResponse{}, fmt.Errorf("openrouter: unmarshal: %w", err)
	}

	return o.toCompletionResponse(orResp), nil
}

// Stream sends a streaming chat completion request.
func (o *OpenRouter) Stream(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error) {
	orReq := o.buildRequest(req, true)
	data, err := json.Marshal(orReq)
	if err != nil {
		return nil, fmt.Errorf("openrouter: marshal stream: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, o.config.BaseURL+"/chat/completions", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("openrouter: create stream request: %w", err)
	}
	o.setHeaders(httpReq)

	httpResp, err := o.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("openrouter: stream request: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		err := o.handleErrorStatus(httpResp)
		httpResp.Body.Close()
		return nil, err
	}

	ch := make(chan StreamChunk, 64)
	go o.readStream(ctx, httpResp.Body, ch)
	return ch, nil
}

// Capabilities returns the OpenRouter provider's capabilities.
func (o *OpenRouter) Capabilities() Capabilities {
	return Capabilities{
		Provider:  "openrouter",
		Models:    []string{o.config.DefaultModel},
		Streaming: true,
		MaxTokens: o.config.DefaultMaxTokens,
	}
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

func (o *OpenRouter) buildRequest(req CompletionRequest, stream bool) openRouterRequest {
	model := req.Model
	if model == "" {
		model = o.config.DefaultModel
	}
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = o.config.DefaultMaxTokens
	}

	msgs := make([]openRouterMessage, 0, len(req.Messages)+1)
	if req.System != "" {
		msgs = append(msgs, openRouterMessage{Role: "system", Content: req.System})
	}
	for _, m := range req.Messages {
		msgs = append(msgs, openRouterMessage{Role: string(m.Role), Content: m.Content})
	}

	return openRouterRequest{
		Model:     model,
		Messages:  msgs,
		MaxTokens: maxTokens,
		Stream:    stream,
	}
}

func (o *OpenRouter) doRequest(ctx context.Context, orReq openRouterRequest) ([]byte, error) {
	data, err := json.Marshal(orReq)
	if err != nil {
		return nil, fmt.Errorf("openrouter: marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, o.config.BaseURL+"/chat/completions", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("openrouter: create request: %w", err)
	}
	o.setHeaders(httpReq)

	resp, err := o.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("openrouter: request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, o.handleErrorStatus(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("openrouter: read: %w", err)
	}
	return body, nil
}

func (o *OpenRouter) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.config.APIKey)
}

func (o *OpenRouter) toCompletionResponse(resp openRouterResponse) CompletionResponse {
	content := ""
	finishReason := ""
	if len(resp.Choices) > 0 {
		content = resp.Choices[0].Message.Content
		finishReason = resp.Choices[0].FinishReason
	}

	inputTokens, outputTokens := 0, 0
	if resp.Usage != nil {
		inputTokens = resp.Usage.PromptTokens
		outputTokens = resp.Usage.CompletionTokens
	}

	return CompletionResponse{
		Message:      Message{Role: RoleAssistant, Content: content},
		FinishReason: finishReason,
		Usage:        Usage{InputTokens: inputTokens, OutputTokens: outputTokens},
	}
}

// readStream reads OpenAI-compatible SSE stream data.
func (o *OpenRouter) readStream(ctx context.Context, body io.ReadCloser, ch chan<- StreamChunk) {
	defer body.Close()
	defer close(ch)

	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 65536), 65536)

	for scanner.Scan() {
		line := scanner.Text()

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		// OpenAI signals stream end with "data: [DONE]"
		if data == "[DONE]" {
			ch <- StreamChunk{Done: true}
			return
		}

		var orResp openRouterStreamResponse
		if err := json.Unmarshal([]byte(data), &orResp); err != nil {
			continue
		}

		for _, choice := range orResp.Choices {
			if choice.Delta.Content != "" {
				ch <- StreamChunk{Content: choice.Delta.Content}
			}
		}

		// Check for final usage info
		if orResp.Usage != nil {
			ch <- StreamChunk{
				Done: true,
				Usage: Usage{
					InputTokens:  orResp.Usage.PromptTokens,
					OutputTokens: orResp.Usage.CompletionTokens,
				},
			}
			return
		}
	}

	if err := scanner.Err(); err != nil {
		ch <- StreamChunk{Err: fmt.Errorf("openrouter: stream read: %w", err)}
	}
}

// handleErrorStatus maps HTTP error responses to sentinel errors.
func (o *OpenRouter) handleErrorStatus(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusTooManyRequests {
		return fmt.Errorf("openrouter: %w", ErrRateLimited)
	}

	var apiErr openRouterError
	if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Error.Message != "" {
		switch apiErr.Error.Type {
		case "authentication_error":
			return fmt.Errorf("openrouter: %w: %s", ErrAuthFailed, apiErr.Error.Message)
		case "rate_limit_error":
			return fmt.Errorf("openrouter: %w: %s", ErrRateLimited, apiErr.Error.Message)
		case "invalid_request_error":
			if strings.Contains(strings.ToLower(apiErr.Error.Message), "context length") ||
				strings.Contains(strings.ToLower(apiErr.Error.Message), "too many tokens") {
				return fmt.Errorf("openrouter: %w: %s", ErrContextWindowExceeded, apiErr.Error.Message)
			}
			return fmt.Errorf("openrouter: %w: %s", ErrBadRequest, apiErr.Error.Message)
		default:
			return fmt.Errorf("openrouter: %s: %s", apiErr.Error.Type, apiErr.Error.Message)
		}
	}

	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return fmt.Errorf("openrouter: %w", ErrAuthFailed)
	case resp.StatusCode >= http.StatusInternalServerError:
		return fmt.Errorf("openrouter: %w", ErrProviderUnavailable)
	default:
		return fmt.Errorf("openrouter: %s", resp.Status)
	}
}
