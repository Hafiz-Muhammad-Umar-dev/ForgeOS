package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/workspace"
)

// DefaultTools returns the built-in tool set available to all agents.
// The tools are configured to execute commands through the given workspace.
func DefaultTools(ws workspace.WorkspacePort, wsID workspace.WorkspaceID) ToolSet {
	if ws == nil {
		return ToolSet{}
	}
	return ToolSet{
		executeCommandTool(ws, wsID),
	}
}

// executeCommandTool creates the "execute_command" tool that runs a shell
// command inside the workspace.
func executeCommandTool(ws workspace.WorkspacePort, wsID workspace.WorkspaceID) Tool {
	return Tool{
		Name:        "execute_command",
		Description: "Execute a shell command inside the workspace. Returns stdout, stderr, and exit code. Use this to run build commands, tests, file operations, and any CLI tool.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"command": map[string]any{
					"type":        "string",
					"description": "The shell command to execute",
				},
				"work_dir": map[string]any{
					"type":        "string",
					"description": "Working directory relative to workspace root. Empty means workspace root.",
				},
				"timeout_seconds": map[string]any{
					"type":        "number",
					"description": "Maximum execution time in seconds. Zero means no timeout.",
				},
			},
			"required": []string{"command"},
		},
		Execute: func(ctx context.Context, input map[string]any) (ToolResult, error) {
			command, _ := input["command"].(string)
			if command == "" {
				return ToolResult{Error: "command is required", ExitCode: -1}, nil
			}

			req := workspace.ExecRequest{Command: "sh", Args: []string{"-c", command}}

			if wd, ok := input["work_dir"].(string); ok && wd != "" {
				req.WorkDir = wd
			}
			if ts, ok := input["timeout_seconds"].(float64); ok && ts > 0 {
				req.Timeout = durationFromSeconds(ts)
			}

			resp, err := ws.Exec(ctx, wsID, req)
			if err != nil {
				return ToolResult{
					Output:   resp.Stdout,
					Error:    fmt.Sprintf("exec failed: %v\nstderr: %s", err, resp.Stderr),
					ExitCode: resp.ExitCode,
				}, nil
			}
			return ToolResult{
				Output:   resp.Stdout,
				Error:    resp.Stderr,
				ExitCode: resp.ExitCode,
			}, nil
		},
	}
}

// durationFromSeconds converts a float64 seconds value to a time.Duration.
func durationFromSeconds(s float64) time.Duration {
	return time.Duration(s * float64(time.Second))
}
