package agent

import "errors"

// Sentinel errors returned by the agent runtime and agents.
var (
	// ErrMaxIterations is returned when the agent exceeds the iteration limit.
	ErrMaxIterations = errors.New("agent: max iterations exceeded")

	// ErrTaskFailed is returned when the agent determines the task cannot be
	// completed.
	ErrTaskFailed = errors.New("agent: task failed")

	// ErrToolNotFound is returned when an agent requests a tool that is not
	// in the ToolSet.
	ErrToolNotFound = errors.New("agent: tool not found")

	// ErrToolExecutionFailed is returned when a tool execution fails.
	ErrToolExecutionFailed = errors.New("agent: tool execution failed")

	// ErrProviderFailed is returned when the LLM provider call fails.
	ErrProviderFailed = errors.New("agent: provider failed")

	// ErrAgentNotReady is returned when the runtime is not initialized.
	ErrAgentNotReady = errors.New("agent: runtime not ready")
)
