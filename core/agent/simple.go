package agent

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/provider"
)

// Compile-time check: *SimpleAgent implements Agent.
var _ Agent = (*SimpleAgent)(nil)

// SimpleAgent is a general-purpose agent that uses a text-based tool-calling
// protocol. It sends the task description and tool definitions to the LLM,
// parses structured tool calls from the response, executes them, and loops
// until completion or the iteration limit.
//
// Tool calling protocol:
//   The agent outputs a JSON block with the format:
//     ```tool
//     {"name": "tool_name", "args": {...}}
//     ```
//   The runtime parses this, executes the tool, and feeds the result back.
//
// When no tool call is detected in the response, the agent treats the
// response as the final answer and completes.
type SimpleAgent struct {
	name         string
	description  string
	model        string
	systemPrompt string // optional override; empty = auto-generated
}

// NewSimpleAgent creates a new SimpleAgent with the given name and
// description. If model is empty, the LLM provider's default is used.
func NewSimpleAgent(name, description, model string) *SimpleAgent {
	return &SimpleAgent{
		name:        name,
		description: description,
		model:       model,
	}
}

// NewSimpleAgentWithPrompt creates a SimpleAgent with a custom system prompt.
// When set, systemPrompt replaces the auto-generated prompt from buildSystemPrompt.
func NewSimpleAgentWithPrompt(name, description, model, systemPrompt string) *SimpleAgent {
	return &SimpleAgent{
		name:         name,
		description:  description,
		model:        model,
		systemPrompt: systemPrompt,
	}
}

// Name returns the agent name.
func (a *SimpleAgent) Name() string { return a.name }

// Description returns the agent description.
func (a *SimpleAgent) Description() string { return a.description }

// Run executes the agent loop: LLM → parse → tool → loop → result.
func (a *SimpleAgent) Run(ctx Context) (*Result, error) {
systemPrompt := a.systemPrompt
	if systemPrompt == "" {
		systemPrompt = a.buildSystemPrompt(ctx.Tools)
	}
	messages := []provider.Message{
		{Role: provider.RoleUser, Content: ctx.Task.Description},
	}

	var lastContent string
	iterations := 0
	maxIter := ctx.Task.MaxIterations
	if maxIter <= 0 {
		maxIter = 10
	}

	for iterations < maxIter {
		iterations++

		llmResp, err := ctx.Provider.Complete(ctx, provider.CompletionRequest{
			Model:    a.model,
			Messages: messages,
			System:   systemPrompt,
		})
		if err != nil {
			return nil, fmt.Errorf("%w: iteration %d: %w", ErrProviderFailed, iterations, err)
		}

		output := llmResp.Message.Content
		lastContent = output

		// Try to parse a tool call from the response.
		toolCall := parseToolCall(output)
		if toolCall == nil {
			// No tool call — this is the final answer.
			return &Result{
				Summary:      output,
				Status:       ResultSuccess,
				InputTokens:  llmResp.Usage.InputTokens,
				OutputTokens: llmResp.Usage.OutputTokens,
				Iterations:   iterations,
			}, nil
		}

		// Execute the tool.
		tool := ctx.Tools.Find(toolCall.Tool)
		if tool == nil {
			return nil, fmt.Errorf("%w: %q", ErrToolNotFound, toolCall.Tool)
		}

		result, err := tool.Execute(ctx, toolCall.Args)
		if err != nil {
			return nil, fmt.Errorf("%w: %q: %w", ErrToolExecutionFailed, toolCall.Tool, err)
		}

		// Feed tool result back to the conversation.
		messages = append(messages,
			provider.Message{Role: provider.RoleAssistant, Content: output},
			provider.Message{Role: provider.RoleUser, Content: formatToolResult(toolCall.Tool, result)},
		)
	}

	return &Result{
		Summary:    lastContent,
		Status:     ResultMaxIterations,
		Iterations: iterations,
	}, nil
}

// buildSystemPrompt creates the system prompt describing the agent's tools.
func (a *SimpleAgent) buildSystemPrompt(tools ToolSet) string {
	if len(tools) == 0 {
		return fmt.Sprintf(`You are %s. %s

Complete the task based on the user's request. Provide a clear and concise response.`, a.name, a.description)
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf(`You are %s. %s

You have access to the following tools. When you need to use a tool, respond with a JSON code block in this exact format:

`+"```"+`tool
{"name": "tool_name", "args": {"arg1": "value1"}}
`+"```"+`

Available tools:
`, a.name, a.description))

	for _, t := range tools {
		b.WriteString(fmt.Sprintf("\n### %s\n%s\n", t.Name, t.Description))
		if params, err := json.MarshalIndent(t.Parameters, "", "  "); err == nil {
			b.WriteString(fmt.Sprintf("Arguments schema:\n%s\n", string(params)))
		}
	}

	b.WriteString(`

After you receive the tool result, continue working. When the task is complete, provide your final answer without a tool block.`)
	return b.String()
}

// parseToolCall tries to extract a tool call from the agent's output.
// It looks for a JSON block wrapped in ```tool ... ``` markers.
func parseToolCall(output string) *ToolCall {
	startMarker := "```tool\n"
	endMarker := "```"

	startIdx := strings.Index(output, startMarker)
	if startIdx < 0 {
		return nil
	}
	startIdx += len(startMarker)

	endIdx := strings.Index(output[startIdx:], endMarker)
	if endIdx < 0 {
		return nil
	}

	jsonStr := output[startIdx : startIdx+endIdx]
	jsonStr = strings.TrimSpace(jsonStr)

	var raw struct {
		Name string         `json:"name"`
		Args map[string]any `json:"args"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &raw); err != nil {
		return nil
	}
	if raw.Name == "" {
		return nil
	}

	return &ToolCall{
		ID:   fmt.Sprintf("tc-%d", len(raw.Args)),
		Tool: raw.Name,
		Args: raw.Args,
	}
}

// formatToolResult creates the user-facing message with the tool output.
func formatToolResult(toolName string, result ToolResult) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Result from %s:\n", toolName))
	if result.Output != "" {
		b.WriteString(result.Output)
		b.WriteString("\n")
	}
	if result.Error != "" {
		b.WriteString(fmt.Sprintf("stderr:\n%s\n", result.Error))
	}
	b.WriteString(fmt.Sprintf("\nExit code: %d", result.ExitCode))
	return b.String()
}
