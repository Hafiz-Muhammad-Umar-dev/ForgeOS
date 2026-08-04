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

// ClaudeConfig configures the Claude LLM provider adapter.
type ClaudeConfig struct {
	// APIKey is the Anthropic API key.
	APIKey string

	// BaseURL is the Anthropic API base URL.
	// Defaults to "https://api.anthropic.com".
	BaseURL string

	// DefaultModel is the model to use when CompletionRequest.Model is empty.
	// Defaults to "claude-sonnet-4-20250514".
	DefaultModel string

	// DefaultMaxTokens is the max_tokens to use when CompletionRequest.MaxTokens
	// is zero. If zero, a sensible default (4096) is used.
	DefaultMaxTokens int

	// HTTPClient is the HTTP client used for API calls.
	// If nil, http.DefaultClient is used.
	HTTPClient *http.Client

	// Timeout for a single request. Zero means no timeout.
	Timeout time.Duration
}

// DefaultClaudeConfig returns a sensible default Claude configuration.
func DefaultClaudeConfig() ClaudeConfig {
	return ClaudeConfig{
		BaseURL:          "https://api.anthropic.com",
		DefaultModel:     "claude-sonnet-4-20250514",
		DefaultMaxTokens: 4096,
		Timeout:          120 * time.Second,
	}
}

// ClaudeOption configures the Claude adapter.
type ClaudeOption func(*ClaudeConfig)

// WithClaudeAPIKey sets the Anthropic API key.
func WithClaudeAPIKey(key string) ClaudeOption {
	return func(c *ClaudeConfig) { c.APIKey = key }
}

// WithClaudeBaseURL sets the Anthropic API base URL.
func WithClaudeBaseURL(url string) ClaudeOption {
	return func(c *ClaudeConfig) { c.BaseURL = url }
}

// WithClaudeModel sets the default Claude model.
func WithClaudeModel(model string) ClaudeOption {
	return func(c *ClaudeConfig) { c.DefaultModel = model }
}

// WithClaudeHTTPClient sets the HTTP client for the Claude adapter.
func WithClaudeHTTPClient(client *http.Client) ClaudeOption {
	return func(c *ClaudeConfig) { c.HTTPClient = client }
}

// WithClaudeTimeout sets the request timeout.
func WithClaudeTimeout(timeout time.Duration) ClaudeOption {
	return func(c *ClaudeConfig) { c.Timeout = timeout }
}

// Compile-time check: *Claude implements LLMProvider.
var _ LLMProvider = (*Claude)(nil)

// Claude is an LLMProvider adapter for the Anthropic Claude API (Messages API).
type Claude struct {
	config ClaudeConfig
	client *http.Client
}

// NewClaude creates a new Claude provider adapter.
func NewClaude(opts ...ClaudeOption) *Claude {
	cfg := DefaultClaudeConfig()
	for _, fn := range opts {
		fn(&cfg)
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: cfg.Timeout}
	}
	return &Claude{config: cfg, client: client}
}

// ---------------------------------------------------------------------------
// Anthropic Messages API types (request/response wire format)
// ---------------------------------------------------------------------------

type anthropicRequest struct {
	Model     string         `json:"model"`
	MaxTokens int            `json:"max_tokens"`
	Messages  []anthropicMsg `json:"messages"`
	System    string         `json:"system,omitempty"`
	Stream    bool           `json:"stream"`
}

type anthropicMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	ID         string           `json:"id"`
	Type       string           `json:"type"`
	Role       string           `json:"role"`
	Content    []anthropicBlock `json:"content"`
	StopReason string           `json:"stop_reason"`
	Usage      anthropicUsage   `json:"usage"`
}

type anthropicBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// anthropicError represents an API error response.
type anthropicError struct {
	Type  string             `json:"type"`
	Error anthropicErrorBody `json:"error"`
}

type anthropicErrorBody struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// Complete sends a non-streaming completion request to the Anthropic API.
func (c *Claude) Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
	anthropicReq := c.buildRequest(req, false)
	data, err := json.Marshal(anthropicReq)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("claude: marshal request: %w", err)
	}

	body, err := c.doRequest(ctx, data)
	if err != nil {
		return CompletionResponse{}, err
	}

	var anthropicResp anthropicResponse
	if err := json.Unmarshal(body, &anthropicResp); err != nil {
		return CompletionResponse{}, fmt.Errorf("claude: unmarshal response: %w", err)
	}

	return c.toCompletionResponse(anthropicResp), nil
}

// Stream sends a streaming completion request and returns a channel of
// StreamChunk values.
func (c *Claude) Stream(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error) {
	anthropicReq := c.buildRequest(req, true)
	data, err := json.Marshal(anthropicReq)
	if err != nil {
		return nil, fmt.Errorf("claude: marshal stream request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.config.BaseURL+"/v1/messages", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("claude: create stream request: %w", err)
	}
	c.setHeaders(httpReq)

	httpResp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("claude: stream request: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		err := c.handleErrorStatus(httpResp)
		httpResp.Body.Close()
		return nil, err
	}

	ch := make(chan StreamChunk, 64)
	go c.readStream(ctx, httpResp.Body, ch)
	return ch, nil
}

// Capabilities returns the Claude provider's capabilities.
func (c *Claude) Capabilities() Capabilities {
	return Capabilities{
		Provider:  "anthropic",
		Models:    supportedModels(),
		Streaming: true,
		MaxTokens: 4096,
	}
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

func (c *Claude) buildRequest(req CompletionRequest, stream bool) anthropicRequest {
	model := req.Model
	if model == "" {
		model = c.config.DefaultModel
	}
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = c.config.DefaultMaxTokens
	}
	if maxTokens == 0 {
		maxTokens = 4096
	}

	messages := make([]anthropicMsg, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = anthropicMsg{Role: string(m.Role), Content: m.Content}
	}

	return anthropicRequest{
		Model:     model,
		MaxTokens: maxTokens,
		Messages:  messages,
		System:    req.System,
		Stream:    stream,
	}
}

func (c *Claude) doRequest(ctx context.Context, data []byte) ([]byte, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.config.BaseURL+"/v1/messages", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("claude: create request: %w", err)
	}
	c.setHeaders(httpReq)

	httpResp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("claude: request: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		return nil, c.handleErrorStatus(httpResp)
	}

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("claude: read response: %w", err)
	}
	return body, nil
}

func (c *Claude) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.config.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")
}

func (c *Claude) toCompletionResponse(resp anthropicResponse) CompletionResponse {
	content := ""
	for _, block := range resp.Content {
		content += block.Text
	}

	return CompletionResponse{
		Message: Message{
			Role:    RoleAssistant,
			Content: content,
		},
		FinishReason: resp.StopReason,
		Usage: Usage{
			InputTokens:  resp.Usage.InputTokens,
			OutputTokens: resp.Usage.OutputTokens,
		},
	}
}

// readStream reads SSE events from the Anthropic streaming response and sends
// StreamChunk values on the channel.
func (c *Claude) readStream(ctx context.Context, body io.ReadCloser, ch chan<- StreamChunk) {
	defer body.Close()
	defer close(ch)

	scanner := bufio.NewScanner(body)
	// SSE lines can be long, so increase the scanner buffer.
	scanner.Buffer(make([]byte, 0, 65536), 65536)

	var currentEvent string
	var currentMessage *anthropicResponse
	var streamUsage anthropicUsage
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "event: ") {
			currentEvent = strings.TrimPrefix(line, "event: ")
			continue
		}

		if strings.HasPrefix(line, "data: ") {
			dataStr := strings.TrimPrefix(line, "data: ")

			switch currentEvent {
			case "message_start":
				var msgStart struct {
					Type    string            `json:"type"`
					Message anthropicResponse `json:"message"`
				}
				if err := json.Unmarshal([]byte(dataStr), &msgStart); err != nil {
					ch <- StreamChunk{Err: fmt.Errorf("claude: parse message_start: %w", err)}
					return
				}
				currentMessage = &msgStart.Message

			case "content_block_delta":
				var delta struct {
					Type  string `json:"type"`
					Index int    `json:"index"`
					Delta struct {
						Type string `json:"type"`
						Text string `json:"text"`
					} `json:"delta"`
				}
				if err := json.Unmarshal([]byte(dataStr), &delta); err != nil {
					ch <- StreamChunk{Err: fmt.Errorf("claude: parse delta: %w", err)}
					return
				}
				if delta.Delta.Type == "text_delta" && delta.Delta.Text != "" {
					ch <- StreamChunk{Content: delta.Delta.Text}
				}

			case "message_delta":
				var msgDelta struct {
					Type  string `json:"type"`
					Delta struct {
						StopReason string `json:"stop_reason"`
					} `json:"delta"`
					Usage anthropicUsage `json:"usage"`
				}
				if err := json.Unmarshal([]byte(dataStr), &msgDelta); err != nil {
					ch <- StreamChunk{Err: fmt.Errorf("claude: parse message_delta: %w", err)}
					return
				}
				streamUsage = msgDelta.Usage

			case "message_stop":
				usage := Usage{
					InputTokens:  streamUsage.InputTokens,
					OutputTokens: streamUsage.OutputTokens,
				}
				if usage.InputTokens == 0 && currentMessage != nil {
					usage.InputTokens = currentMessage.Usage.InputTokens
				}
				ch <- StreamChunk{Done: true, Usage: usage}
				return

			case "error":
				var apiErr anthropicError
				if err := json.Unmarshal([]byte(dataStr), &apiErr); err == nil {
					ch <- StreamChunk{Err: c.mapError(apiErr)}
				} else {
					ch <- StreamChunk{Err: fmt.Errorf("claude: stream error: %s", dataStr)}
				}
				return

			case "ping":
				// Heartbeat; ignore.
			}
		}
	}

	if err := scanner.Err(); err != nil {
		ch <- StreamChunk{Err: fmt.Errorf("claude: stream read: %w", err)}
	}
}

// handleErrorStatus maps an HTTP error response to a sentinel error.
func (c *Claude) handleErrorStatus(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusTooManyRequests {
		return fmt.Errorf("claude: %w", ErrRateLimited)
	}

	var apiErr anthropicError
	if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Error.Type != "" {
		return c.mapError(apiErr)
	}

	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return fmt.Errorf("claude: %w: %s", ErrAuthFailed, resp.Status)
	case resp.StatusCode == http.StatusBadRequest:
		if strings.Contains(strings.ToLower(string(body)), "context") ||
			strings.Contains(strings.ToLower(string(body)), "length") ||
			strings.Contains(strings.ToLower(string(body)), "too large") {
			return fmt.Errorf("claude: %w: %s", ErrContextWindowExceeded, resp.Status)
		}
		return fmt.Errorf("claude: %w: %s", ErrBadRequest, resp.Status)
	case resp.StatusCode >= http.StatusInternalServerError:
		return fmt.Errorf("claude: %w: %s", ErrProviderUnavailable, resp.Status)
	default:
		return fmt.Errorf("claude: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
}

func (c *Claude) mapError(apiErr anthropicError) error {
	switch apiErr.Error.Type {
	case "authentication_error", "permission_error":
		return fmt.Errorf("claude: %w: %s", ErrAuthFailed, apiErr.Error.Message)
	case "rate_limit_error":
		return fmt.Errorf("claude: %w: %s", ErrRateLimited, apiErr.Error.Message)
	case "invalid_request_error":
		if strings.Contains(strings.ToLower(apiErr.Error.Message), "context") ||
			strings.Contains(strings.ToLower(apiErr.Error.Message), "length") ||
			strings.Contains(strings.ToLower(apiErr.Error.Message), "too long") {
			return fmt.Errorf("claude: %w: %s", ErrContextWindowExceeded, apiErr.Error.Message)
		}
		return fmt.Errorf("claude: %w: %s", ErrBadRequest, apiErr.Error.Message)
	case "api_error", "overloaded_error":
		return fmt.Errorf("claude: %w: %s", ErrProviderUnavailable, apiErr.Error.Message)
	default:
		return fmt.Errorf("claude: %s: %s", apiErr.Error.Type, apiErr.Error.Message)
	}
}

// supportedModels returns the known Claude models available through the API.
func supportedModels() []string {
	return []string{
		"claude-sonnet-4-20250514",
		"claude-sonnet-4-20250514",
		"claude-3-5-sonnet-20241022",
		"claude-3-5-haiku-20241022",
		"claude-3-opus-20240229",
		"claude-3-haiku-20240307",
	}
}
