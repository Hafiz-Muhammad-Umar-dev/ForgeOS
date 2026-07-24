package agent

import (
	"context"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/bus"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/provider"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/workspace"
)

// Context carries task-scoped resources that an agent uses during execution.
// It embeds the standard library context for cancellation and deadlines.
//
// The zero value is not usable; construct via NewContext.
type Context struct {
	context.Context

	// Task is the work item being executed.
	Task Task

	// Tools are the capabilities available to the agent.
	Tools ToolSet

	// Provider is the LLM provider the agent uses for reasoning.
	Provider provider.LLMProvider

	// Workspace is the execution environment for tools.
	Workspace workspace.WorkspacePort

	// Bus is the message bus for publishing events. May be nil when no bus
	// is available; publishing must be best-effort.
	Bus bus.BusPort
}

// NewContext builds an agent context from its required dependencies.
func NewContext(
	ctx context.Context,
	task Task,
	tools ToolSet,
	llm provider.LLMProvider,
	ws workspace.WorkspacePort,
	opts ...ContextOption,
) Context {
	ac := Context{
		Context:   ctx,
		Task:      task,
		Tools:     tools,
		Provider:  llm,
		Workspace: ws,
	}
	for _, fn := range opts {
		fn(&ac)
	}
	return ac
}

// ContextOption configures the agent context.
type ContextOption func(*Context)

// WithBus attaches a bus to the context for event publishing.
func WithBus(b bus.BusPort) ContextOption {
	return func(ac *Context) { ac.Bus = b }
}
