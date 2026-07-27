// Package query implements the CQRS read side for DevOS. It provides the
// QueryService port, a ProjectionEngine that consumes bus events and writes
// read models, and strongly-typed view types.
//
// Sprint 0 scope:
//   - QueryService interface (GetIntent, ListIntents, GetTask, ListTasks)
//   - IntentProjection (subscribes to intent.completed, intent.failed)
//   - TaskProjection (subscribes to task.status, task.failed)
//   - ProjectionEngine (lifecycle.Component managing bus subscriptions)
//   - Repository with named SQL constants
//   - FakeQueryService for testing
//
// See SDD §08 (Query/Read Service), Build Order Step 11.
package query

import (
	"context"
	"time"
)

// QueryService serves projected read-model data to clients.
// It is the CQRS read-side port.
type QueryService interface {
	// GetIntent returns a single intent by ID.
	GetIntent(ctx context.Context, id string) (IntentView, error)

	// ListIntents returns intents for an organization with pagination.
	ListIntents(ctx context.Context, orgID string, limit, offset int) ([]IntentView, error)

	// GetTask returns a single task by ID.
	GetTask(ctx context.Context, id string) (TaskView, error)

	// ListTasks returns all tasks for an intent.
	ListTasks(ctx context.Context, intentID string) ([]TaskView, error)

	// Ping verifies the backing store is reachable.
	Ping(ctx context.Context) error
}

// IntentView is the projected read model for an intent.
type IntentView struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id,omitempty"`
	OrgID     string    `json:"org_id"`
	ProjectID string    `json:"project_id,omitempty"`
	TraceID   string    `json:"trace_id,omitempty"`
	Text      string    `json:"text,omitempty"`
	Status    string    `json:"status"`
	Summary   string    `json:"summary,omitempty"`
	Error     string    `json:"error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TaskView is the projected read model for a task.
type TaskView struct {
	ID           string    `json:"id"`
	IntentID     string    `json:"intent_id"`
	AgentName    string    `json:"agent_name,omitempty"`
	Status       string    `json:"status"`
	Summary      string    `json:"summary,omitempty"`
	Error        string    `json:"error,omitempty"`
	InputTokens  int       `json:"input_tokens"`
	OutputTokens int       `json:"output_tokens"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
