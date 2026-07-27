// Package dag defines the DAG (Directed Acyclic Graph) types for the DevOS
// orchestration engine. A DAG represents a plan decomposed into task nodes
// with dependency edges. Nodes are executed in topological order.
//
// See ADR-002 (Stateful DAG Orchestration), SDD §03 (Orchestration Service).
package dag

import "time"

// NodeStatus represents the lifecycle state of a DAG node.
type NodeStatus string

const (
	NodePending   NodeStatus = "pending"
	NodeRunning   NodeStatus = "running"
	NodeCompleted NodeStatus = "completed"
	NodeFailed    NodeStatus = "failed"
	NodeSkipped   NodeStatus = "skipped"
)

// NodePayload is the event payload for node.status and node.failed events.
type NodePayload struct {
	NodeID  string `json:"node_id"`
	DAGID   string `json:"dag_id"`
	Agent   string `json:"agent,omitempty"`
	Status  string `json:"status"`
	Summary string `json:"summary,omitempty"`
	Error   string `json:"error,omitempty"`
}

// Node is a single unit of work within a DAG.
type Node struct {
	// ID is a unique identifier within the DAG.
	ID string `json:"id"`

	// Name is a human-readable label (e.g., "Generate frontend scaffold").
	Name string `json:"name"`

	// Agent is the agent name resolved from the Registry (e.g., "frontend").
	Agent string `json:"agent"`

	// Description is the natural-language task description for the agent.
	Description string `json:"description"`

	// InputIDs lists the node IDs that must complete before this node runs.
	InputIDs []string `json:"input_ids,omitempty"`

	// Status tracks execution progress.
	Status NodeStatus `json:"status"`

	// Result is the agent output summary after completion.
	Result string `json:"result,omitempty"`

	// Error is set when the node fails.
	Error string `json:"error,omitempty"`
}

// Edge is a dependency between two nodes.
type Edge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// DAGStatus represents the lifecycle state of a DAG execution.
type DAGStatus string

const (
	DAGProposed  DAGStatus = "proposed"
	DAGRunning   DAGStatus = "running"
	DAGCompleted DAGStatus = "completed"
	DAGFailed    DAGStatus = "failed"
)

// DAG is a Directed Acyclic Graph of work nodes.
type DAG struct {
	// ID uniquely identifies this DAG execution.
	ID string `json:"id"`

	// IntentID links the DAG to the originating intent.
	IntentID string `json:"intent_id"`

	// Nodes is the set of work nodes in this DAG.
	Nodes []Node `json:"nodes"`

	// Edges define dependencies between nodes.
	Edges []Edge `json:"edges,omitempty"`

	// Status tracks overall DAG execution progress.
	Status DAGStatus `json:"status"`

	// CreatedAt is when the DAG was created.
	CreatedAt time.Time `json:"created_at"`
}

// FindNode returns a node by ID, or nil if not found.
func (d *DAG) FindNode(id string) *Node {
	for i := range d.Nodes {
		if d.Nodes[i].ID == id {
			return &d.Nodes[i]
		}
	}
	return nil
}
