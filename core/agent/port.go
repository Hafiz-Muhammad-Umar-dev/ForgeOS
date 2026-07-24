// Package agent defines the DevOS agent abstraction (Agent), the agent runtime
// (Runtime), and the tool execution pipeline. It follows the ports/adapters
// (hexagonal) architecture: the Runtime uses LLMProvider and WorkspacePort to
// execute agents, and agents are plugins that implement the Agent interface.
//
// Sprint 0 scope:
//   - Agent interface (Name, Description, Run)
//   - Runtime (manages agent execution, config, lifecycle)
//   - Tool interface and built-in tools (execute_command)
//   - Context (task-scoped dependencies)
//   - SimpleAgent demo agent
//   - FakeAgent for tests
//   - Streaming support via RunStream
//   - Event publishing over BusPort
//
// Excluded from Sprint 0 (deferred to later components):
//   - Registry-backed agent loading
//   - Multi-agent orchestration
//   - Memory / RAG
//   - Planning engine
//   - HITL / budget gates
//   - Tool marketplace / dynamic plugin loading
//   - Cost tracking
//
// See SDD §04 (Agent Runtime), ADR-003 (Provider Abstraction via Ports),
// ADR-004 (Workspace Isolation).
package agent

import (
	"context"
)

//---------------------------------------------------------------------------
// Agent interface
//---------------------------------------------------------------------------

// Agent is the core abstraction for an agent plugin. Each agent defines its
// own behavior by implementing Run.
type Agent interface {
	// Name returns the agent's unique identifier (e.g., "coder", "reviewer").
	Name() string

	// Description returns a human-readable description of the agent's role.
	Description() string

	// Run executes the agent's logic for the given task. The context carries
	// the LLM provider, workspace, and tools the agent may use. It returns
	// the result or an error.
	Run(ctx Context) (*Result, error)
}

//---------------------------------------------------------------------------
// Result types
//---------------------------------------------------------------------------

// ResultStatus indicates how the agent run completed.
type ResultStatus string

const (
	ResultSuccess            ResultStatus = "success"
	ResultFailed             ResultStatus = "failed"
	ResultMaxIterations      ResultStatus = "max_iterations"
	ResultCancelled          ResultStatus = "cancelled"
)

// Result is the outcome of a single agent run.
type Result struct {
	TaskID      string        `json:"task_id"`
	Summary     string        `json:"summary"`
	Status      ResultStatus  `json:"status"`
	Artifacts   []Artifact    `json:"artifacts,omitempty"`
	InputTokens  int          `json:"input_tokens"`
	OutputTokens int          `json:"output_tokens"`
	Iterations  int           `json:"iterations"`
}

// Artifact is a named output produced by an agent (file, report, etc.).
type Artifact struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	MIMEType string `json:"mime_type,omitempty"`
	Content  []byte `json:"content,omitempty"`
}

//---------------------------------------------------------------------------
// Task types
//---------------------------------------------------------------------------

// Task is a unit of work assigned to an agent.
type Task struct {
	// ID is the unique task identifier.
	ID string `json:"id"`

	// Description is the natural-language task description.
	Description string `json:"description"`

	// Payload carries optional structured data.
	Payload map[string]any `json:"payload,omitempty"`

	// MaxIterations limits the number of LLM-tool cycles. Zero means a
	// default (10) is used.
	MaxIterations int `json:"max_iterations,omitempty"`
}

//---------------------------------------------------------------------------
// Tool types
//---------------------------------------------------------------------------

// Tool defines an executable capability an agent can invoke. In Sprint 0
// tools are text-protocol based: the agent describes the tool call in its
// output, and the Runtime parses and dispatches it.
type Tool struct {
	// Name is the tool identifier (e.g., "execute_command").
	Name string `json:"name"`

	// Description explains to the LLM what the tool does and when to use it.
	Description string `json:"description"`

	// Parameters is a JSON Schema object describing the expected arguments.
	Parameters map[string]any `json:"parameters"`

	// Execute performs the tool and returns the result.
	Execute func(ctx context.Context, input map[string]any) (ToolResult, error)
}

// ToolResult is the outcome of a single tool execution.
type ToolResult struct {
	// Output is the human-readable result (stdout, file contents, etc.).
	Output string `json:"output"`

	// Error is set when the tool fails.
	Error string `json:"error,omitempty"`

	// ExitCode is the process exit code (for command tools).
	ExitCode int `json:"exit_code"`
}

// ToolSet is an ordered collection of tools available to an agent.
type ToolSet []Tool

// Find returns the tool with the given name, or nil.
func (ts ToolSet) Find(name string) *Tool {
	for _, t := range ts {
		if t.Name == name {
			return &t
		}
	}
	return nil
}

// Names returns the list of tool names.
func (ts ToolSet) Names() []string {
	names := make([]string, len(ts))
	for i, t := range ts {
		names[i] = t.Name
	}
	return names
}

//---------------------------------------------------------------------------
// AgentEvent (streaming)
//---------------------------------------------------------------------------

// AgentEvent is emitted during a streaming agent run. Each event represents
// one observable step.
type AgentEvent struct {
	// Content is a text delta from the agent.
	Content string `json:"content,omitempty"`

	// ToolCall is set when the agent requests a tool execution.
	ToolCall *ToolCall `json:"tool_call,omitempty"`

	// ToolResult is set when a tool execution completes.
	ToolResult *ToolResult `json:"tool_result,omitempty"`

	// Done is true when the run is complete.
	Done bool `json:"done"`

	// Result is set when Done is true.
	Result *Result `json:"result,omitempty"`

	// Err is set when a fatal error occurs.
	Err error `json:"-"`
}

// ToolCall represents an agent's request to invoke a tool.
type ToolCall struct {
	// ID uniquely identifies this tool call within the run.
	ID string `json:"id"`

	// Tool is the name of the tool to invoke.
	Tool string `json:"tool"`

	// Args are the tool arguments.
	Args map[string]any `json:"args"`
}
