// Package intents provides PostgreSQL-backed persistence for intents and tasks.
// It follows the Handler → Service → Repository → Store (database) architecture.
//
// This replaces the previous in-memory mock data with a durable, transactional
// storage layer. The service exposes a clean API that the API Gateway consumes.
package intents

import "time"

// Intent is a user intent persisted to the database.
type Intent struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	OrgID     string    `json:"org_id"`
	ProjectID string    `json:"project_id"`
	TraceID   string    `json:"trace_id"`
	Text      string    `json:"text"`
	Status    string    `json:"status"`
	Summary   string    `json:"summary"`
	Error     string    `json:"error"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Task is a task associated with an intent.
type Task struct {
	ID           string    `json:"id"`
	IntentID     string    `json:"intent_id"`
	AgentName    string    `json:"agent_name"`
	Status       string    `json:"status"`
	Summary      string    `json:"summary"`
	Error        string    `json:"error"`
	InputTokens  int       `json:"input_tokens"`
	OutputTokens int       `json:"output_tokens"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// NewIntentRequest is the input for creating a new intent.
type NewIntentRequest struct {
	Text      string
	UserID    string
	OrgID     string
	ProjectID string
	TraceID   string
}
