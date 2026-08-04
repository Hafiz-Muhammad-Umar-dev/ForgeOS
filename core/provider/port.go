// Package provider defines the LLM and Deploy provider port abstractions for
// the DevOS kernel. It follows the ports/adapters (hexagonal) architecture:
// core/domain uses LLMProvider and DeployProvider, and adapters satisfy them
// without leaking provider SDK types into domain code.
//
// See ADR-003 (Provider Abstraction via Ports), SDD §05 (Provider Gateway).
package provider

import "context"

// LLMProvider is the core abstraction for a large-language model provider.
// All LLM interactions go through this interface. Implementations provide
// the transport (HTTP to Anthropic, OpenRouter, etc.).
type LLMProvider interface {
	// Complete sends a completion request and returns the full response.
	// It blocks until the entire response is received.
	Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error)

	// Stream sends a completion request and returns a channel of stream
	// chunks. The channel is closed when streaming is complete or an error
	// occurs. Consumers should check StreamChunk.Err for errors.
	Stream(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error)

	// Capabilities returns the provider's advertised capabilities.
	Capabilities() Capabilities
}

// Role identifies the speaker of a message.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
)

// Message is a single turn in a conversation.
type Message struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
}

// CompletionRequest is the input to an LLM completion call.
type CompletionRequest struct {
	// Model is the model identifier (e.g., "claude-sonnet-4-20250514").
	// Empty means the provider default.
	Model string `json:"model,omitempty"`

	// Messages is the conversation history, newest last.
	Messages []Message `json:"messages"`

	// System is an optional system prompt.
	System string `json:"system,omitempty"`

	// MaxTokens is the maximum number of tokens to generate.
	// Zero means the provider default.
	MaxTokens int `json:"max_tokens,omitempty"`

	// Temperature controls randomness (0.0–1.0). Zero means provider default.
	Temperature float64 `json:"temperature,omitempty"`
}

// CompletionResponse is the result of a successful completion call.
type CompletionResponse struct {
	// Message is the assistant's response.
	Message Message `json:"message"`

	// FinishReason describes why generation stopped.
	FinishReason string `json:"finish_reason"`

	// Usage contains token counts.
	Usage Usage `json:"usage"`
}

// Usage contains token accounting for a completion.
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// StreamChunk is a single chunk from a streaming completion.
type StreamChunk struct {
	// Content is a partial delta of the assistant's response.
	Content string `json:"content,omitempty"`

	// Done is true when the stream is complete.
	Done bool `json:"done"`

	// Usage is set when Done is true.
	Usage Usage `json:"usage,omitempty"`

	// Err is set when a streaming error occurs.
	Err error `json:"-"`
}

// Capabilities describes what a provider supports.
type Capabilities struct {
	// Provider is the provider name (e.g., "anthropic").
	Provider string `json:"provider"`

	// Models is the list of available model identifiers.
	Models []string `json:"models"`

	// Streaming indicates whether the provider supports streaming.
	Streaming bool `json:"streaming"`

	// MaxTokens is the maximum allowed output tokens.
	MaxTokens int `json:"max_tokens"`
}

// ---------------------------------------------------------------------------
// DeployProvider
// ---------------------------------------------------------------------------

// DeployProvider is the abstraction for a deployment provider (Vercel, Fly.io).
// Implementations handle source upload, build, release, and status tracking.
type DeployProvider interface {
	// Deploy triggers a deployment from the given source and returns tracking info.
	Deploy(ctx context.Context, req DeployRequest) (DeployResponse, error)

	// Status returns the current status of a deployment.
	Status(ctx context.Context, deploymentID string) (DeployStatus, error)

	// Capabilities returns the provider's advertised capabilities.
	Capabilities() DeployCapabilities
}

// DeployRequest is the input to a deployment call.
type DeployRequest struct {
	// Source is the git repository URL to deploy.
	Source string `json:"source"`

	// ProjectName is the target project or app name.
	ProjectName string `json:"project_name"`

	// Env are environment variables to inject during the build.
	Env map[string]string `json:"env,omitempty"`
}

// DeployResponse is the result of a deployment request.
type DeployResponse struct {
	// DeploymentID uniquely identifies the deployment.
	DeploymentID string `json:"deployment_id"`

	// URL is the public URL of the deployed application.
	URL string `json:"url"`

	// Status is the initial deployment status.
	Status DeployStatus `json:"status"`
}

// DeployStatus represents the lifecycle state of a deployment.
type DeployStatus string

const (
	DeployStatusPending   DeployStatus = "pending"
	DeployStatusDeploying DeployStatus = "deploying"
	DeployStatusReady     DeployStatus = "ready"
	DeployStatusFailed    DeployStatus = "failed"
)

// DeployCapabilities describes a deploy provider's capabilities.
type DeployCapabilities struct {
	// Provider is the provider name (e.g., "vercel", "fly").
	Provider string `json:"provider"`

	// Regions is the list of supported deployment regions.
	Regions []string `json:"regions"`
}
