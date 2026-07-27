package orchestrator

import (
	"context"
	"log"
	"time"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/agent"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/dag"
)

// NodeExecutor executes a single DAG node through the Coordinator and
// tracks its state transitions.
type NodeExecutor struct {
	coordinator *Coordinator
}

// NewNodeExecutor creates a NodeExecutor.
func NewNodeExecutor(coordinator *Coordinator) *NodeExecutor {
	return &NodeExecutor{coordinator: coordinator}
}

// Execute runs a single DAG node, tracks state, and returns the result.
// The node's status is updated in-place on the provided DAG.
func (ne *NodeExecutor) Execute(ctx context.Context, node *dag.Node, d *dag.DAG) (*agent.Result, error) {
	node.Status = dag.NodeRunning
	log.Printf("dag: executing node=%s agent=%s", node.ID, node.Agent)

	start := time.Now()

	result, err := ne.coordinator.ExecuteNode(ctx, *node)
	elapsed := time.Since(start)

	if err != nil {
		node.Status = dag.NodeFailed
		node.Error = err.Error()
		log.Printf("dag: node=%s failed after %s: %v", node.ID, elapsed, err)
		return result, err
	}

	node.Status = dag.NodeCompleted
	node.Result = result.Summary
	log.Printf("dag: node=%s completed in %s", node.ID, elapsed)
	return result, nil
}
