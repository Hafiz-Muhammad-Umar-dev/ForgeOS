package orchestrator

import (
	"context"
	"fmt"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/agent"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/dag"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/registry"
)

// Coordinator resolves agents from the Registry and dispatches DAG nodes
// to the Agent Runtime.
type Coordinator struct {
	registry registry.Registry
	runner   TaskRunner
}

// NewCoordinator creates a Coordinator backed by the given Registry and TaskRunner.
func NewCoordinator(reg registry.Registry, runner TaskRunner) *Coordinator {
	return &Coordinator{registry: reg, runner: runner}
}

// ResolveAgent looks up an agent name by capability from the Registry.
// Falls back to using the agent name directly when the Registry has no match.
func (c *Coordinator) ResolveAgent(ctx context.Context, node dag.Node) (string, error) {
	// Try to resolve by capability first (agent name matches capability).
	services, err := c.registry.Discover(ctx, "agent")
	if err == nil && len(services) > 0 {
		for _, svc := range services {
			if svc.Name == node.Agent {
				return svc.Name, nil
			}
		}
	}

	// Fall back to using the node's agent name directly.
	return node.Agent, nil
}

// ExecuteNode dispatches a DAG node to the Agent Runtime and returns the result.
func (c *Coordinator) ExecuteNode(ctx context.Context, node dag.Node) (*agent.Result, error) {
	agentName, err := c.ResolveAgent(ctx, node)
	if err != nil {
		return nil, fmt.Errorf("coordinator: resolve agent for %q: %w", node.ID, err)
	}

	task := agent.Task{
		ID:          node.ID,
		Description: node.Description,
		MaxIterations: 3,
	}

	result, err := c.runner.RunTask(ctx, agentName, task)
	if err != nil {
		return result, fmt.Errorf("coordinator: run %q on %q: %w", node.ID, agentName, err)
	}
	return result, nil
}
