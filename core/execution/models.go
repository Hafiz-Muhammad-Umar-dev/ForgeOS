package execution

import "time"

// ExecutionPlan is the DAG-based execution model served to the frontend.
// It mirrors the frontend ExecutionPlan type and the core/dag.DAG structure.
type ExecutionPlan struct {
	ID          string          `json:"id"`
	IntentID    string          `json:"intent_id"`
	Status      Status          `json:"status"`
	Nodes       []ExecutionNode `json:"nodes"`
	Edges       []ExecutionEdge `json:"edges"`
	CreatedAt   time.Time       `json:"created_at"`
	StartedAt   *time.Time      `json:"started_at,omitempty"`
	CompletedAt *time.Time      `json:"completed_at,omitempty"`
}

// ExecutionNode represents a single unit of work in an execution DAG.
type ExecutionNode struct {
	ID        string `json:"id"`
	AgentRole string `json:"agent_role"`
	Label     string `json:"label"`
	Status    string `json:"status"`
	Progress  int    `json:"progress"`
	Runtime   int64  `json:"runtime"` // milliseconds
	Tokens    int    `json:"tokens"`
	Cost      float64 `json:"cost"`
	ParentID  string `json:"parent_id,omitempty"`
}

// ExecutionEdge represents a dependency between execution nodes.
type ExecutionEdge struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
}
