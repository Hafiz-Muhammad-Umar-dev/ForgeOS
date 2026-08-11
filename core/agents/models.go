// Package agents provides PostgreSQL-backed persistence for agent metadata.
// It follows the Handler → Service → Repository → Store (database) architecture,
// mirroring the pattern established by core/intents.
package agents

import "time"

// AgentStatus represents the runtime status of an agent.
type AgentStatus string

const (
	AgentIdle      AgentStatus = "idle"
	AgentRunning   AgentStatus = "running"
	AgentThinking  AgentStatus = "thinking"
	AgentWaiting   AgentStatus = "waiting"
	AgentCompleted AgentStatus = "completed"
	AgentFailed    AgentStatus = "failed"
)

// AgentRole represents the role/capability of an agent.
type AgentRole string

const (
	RolePlanner   AgentRole = "planner"
	RoleResearcher AgentRole = "researcher"
	RoleCoder     AgentRole = "coder"
	RoleReviewer  AgentRole = "reviewer"
	RoleTester    AgentRole = "tester"
	RoleDeployer  AgentRole = "deployer"
)

// Agent is an agent persisted to the database, matching the frontend AgentInfo shape.
type Agent struct {
	ID             string    `json:"id"`
	OrgID          string    `json:"org_id"`
	Name           string    `json:"name"`
	Role           AgentRole `json:"role"`
	Status         AgentStatus `json:"status"`
	Model          string    `json:"model"`
	Temperature    float64   `json:"temperature"`
	CurrentTool    string    `json:"current_tool,omitempty"`
	Reasoning      string    `json:"reasoning,omitempty"`
	Memory         string    `json:"memory,omitempty"`
	Output         string    `json:"output,omitempty"`
	QueueLength    int       `json:"queue_length"`
	ExecutionTime  int64     `json:"execution_time"` // milliseconds
	PromptTokens   int       `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	Cost           float64   `json:"cost"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
