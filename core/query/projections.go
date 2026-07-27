package query

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/agent"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/event"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/orchestrator"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/store"
)

// Compile-time checks.
var (
	_ Projection = (*IntentProjection)(nil)
	_ Projection = (*TaskProjection)(nil)
)

// ---------------------------------------------------------------------------
// IntentProjection
// ---------------------------------------------------------------------------

// IntentProjection projects intent.completed and intent.failed events into
// the query_intents read model table.
type IntentProjection struct {
	repo *Repository
}

// NewIntentProjection creates an IntentProjection.
func NewIntentProjection(s store.Store) *IntentProjection {
	return &IntentProjection{repo: NewRepository(s)}
}

// Name returns "intent_projection".
func (p *IntentProjection) Name() string { return "intent_projection" }

// Subjects returns the bus subjects this projection handles.
func (p *IntentProjection) Subjects() []string {
	return []string{
		"devos.intent.completed",
		"devos.intent.failed",
	}
}

// Handle processes a single event and updates the intent read model.
func (p *IntentProjection) Handle(ctx context.Context, env event.RawEnvelope) error {
	if env.Type != event.TypeIntentCompleted && env.Type != event.TypeIntentFailed {
		return nil
	}

	payload, err := event.UnmarshalPayload[orchestrator.IntentLifecyclePayload](env)
	if err != nil {
		return fmt.Errorf("unmarshal intent payload: %w", err)
	}

	status := "completed"
	if env.Type == event.TypeIntentFailed {
		status = "failed"
	}

	if err := p.repo.UpsertIntent(ctx,
		payload.IntentID,
		payload.UserID,
		env.OrgID,
		env.ProjectID,
		env.TraceID,
		"", // text — not available in lifecycle events
		status,
		payload.Summary,
		payload.Error,
		time.Unix(0, env.ProducedAt),
	); err != nil {
		return fmt.Errorf("upsert intent: %w", err)
	}

	log.Printf("projection: intent_projection projected intent=%s status=%s", payload.IntentID, status)
	return nil
}

// ---------------------------------------------------------------------------
// TaskProjection
// ---------------------------------------------------------------------------

// TaskProjection projects task.status and task.failed events into the
// query_tasks read model table.
type TaskProjection struct {
	repo *Repository
}

// NewTaskProjection creates a TaskProjection.
func NewTaskProjection(s store.Store) *TaskProjection {
	return &TaskProjection{repo: NewRepository(s)}
}

// Name returns "task_projection".
func (p *TaskProjection) Name() string { return "task_projection" }

// Subjects returns the bus subjects this projection handles.
func (p *TaskProjection) Subjects() []string {
	return []string{
		"devos.task.status",
		"devos.task.failed",
	}
}

// Handle processes a single event and updates the task read model.
func (p *TaskProjection) Handle(ctx context.Context, env event.RawEnvelope) error {
	if env.Type != event.TypeTaskStatus && env.Type != event.TypeTaskFailed {
		return nil
	}

	payload, err := event.UnmarshalPayload[agent.TaskStatusPayload](env)
	if err != nil {
		return fmt.Errorf("unmarshal task payload: %w", err)
	}

	status := payload.Status
	if env.Type == event.TypeTaskFailed {
		status = "failed"
	}

	if err := p.repo.UpsertTask(ctx,
		payload.TaskID,
		"", // intent_id — requires cross-reference from event metadata (future)
		payload.AgentName,
		status,
		payload.Summary,
		payload.Error,
		0, // input_tokens — not available in current task events
		0, // output_tokens — not available in current task events
		time.Unix(0, env.ProducedAt),
	); err != nil {
		return fmt.Errorf("upsert task: %w", err)
	}

	log.Printf("projection: task_projection projected task=%s status=%s agent=%s", payload.TaskID, status, payload.AgentName)
	return nil
}
