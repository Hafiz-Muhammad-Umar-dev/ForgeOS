package execution

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/store"
)

// Named SQL statements for executions.
const (
	sqlUpsertExecution = `INSERT INTO executions
		(id, intent_id, agent_name, status, org_id, created_at, updated_at, started_at, completed_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW(), $6, $7)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			updated_at = NOW(),
			started_at = COALESCE(EXCLUDED.started_at, executions.started_at),
			completed_at = COALESCE(EXCLUDED.completed_at, executions.completed_at)`

	sqlGetExecution = `SELECT id, intent_id, agent_name, status, org_id, created_at, updated_at, started_at, completed_at
		FROM executions WHERE id = $1`

	sqlListExecutionsByOrg = `SELECT id, intent_id, agent_name, status, org_id, created_at, updated_at, started_at, completed_at
		FROM executions WHERE org_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`

	sqlListExecutionsByIntent = `SELECT id, intent_id, agent_name, status, org_id, created_at, updated_at, started_at, completed_at
		FROM executions WHERE intent_id = $1 ORDER BY created_at`

	sqlUpsertExecutionNode = `INSERT INTO execution_nodes
		(id, execution_id, agent_role, label, status, progress, runtime, tokens, cost, parent_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			progress = EXCLUDED.progress,
			runtime = EXCLUDED.runtime,
			tokens = EXCLUDED.tokens,
			cost = EXCLUDED.cost,
			parent_id = EXCLUDED.parent_id`

	sqlListExecutionNodes = `SELECT id, execution_id, agent_role, label, status, progress, runtime, tokens, cost, parent_id
		FROM execution_nodes WHERE execution_id = $1 ORDER BY id`

	sqlUpsertExecutionEdge = `INSERT INTO execution_edges (id, execution_id, source, target)
		VALUES ($1, $2, $3, $4) ON CONFLICT (id) DO NOTHING`

	sqlListExecutionEdges = `SELECT id, execution_id, source, target
		FROM execution_edges WHERE execution_id = $1 ORDER BY id`

	sqlUpsertExecutionEvent = `INSERT INTO execution_events
		(id, execution_id, type, agent_id, content, metadata, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	sqlListExecutionEvents = `SELECT id, execution_id, type, agent_id, content, metadata, timestamp
		FROM execution_events WHERE execution_id = $1 ORDER BY timestamp`

	sqlUpsertExecutionMetrics = `INSERT INTO execution_metrics
		(execution_id, total_tokens, prompt_tokens, completion_tokens, estimated_cost,
		 execution_duration, average_latency, tools_executed, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (execution_id) DO UPDATE SET
			total_tokens = EXCLUDED.total_tokens,
			prompt_tokens = EXCLUDED.prompt_tokens,
			completion_tokens = EXCLUDED.completion_tokens,
			estimated_cost = EXCLUDED.estimated_cost,
			execution_duration = EXCLUDED.execution_duration,
			average_latency = EXCLUDED.average_latency,
			tools_executed = EXCLUDED.tools_executed,
			timestamp = NOW()`

	sqlGetExecutionMetrics = `SELECT execution_id, total_tokens, prompt_tokens, completion_tokens,
		estimated_cost, execution_duration, average_latency, tools_executed, timestamp
		FROM execution_metrics WHERE execution_id = $1`
)

// Repository provides database operations for executions.
type Repository struct {
	store store.Store
}

// NewRepository creates an executions repository.
func NewRepository(s store.Store) *Repository {
	return &Repository{store: s}
}

// UpsertExecution inserts or updates an execution.
func (r *Repository) UpsertExecution(ctx context.Context, e Execution) error {
	_, err := r.store.Exec(ctx, sqlUpsertExecution,
		e.ID, e.IntentID, e.AgentName, string(e.Status), e.OrgID, e.StartedAt, e.CompletedAt)
	if err != nil {
		return fmt.Errorf("execution: upsert: %w", err)
	}
	return nil
}

// GetExecution returns a single execution by ID.
func (r *Repository) GetExecution(ctx context.Context, id string) (*Execution, error) {
	row := r.store.QueryRow(ctx, sqlGetExecution, id)
	return scanExecution(row, id)
}

// ListExecutions returns executions for an org or intent.
func (r *Repository) ListExecutions(ctx context.Context, orgID, intentID string, limit, offset int) ([]Execution, error) {
	var (
		rows store.Rows
		err  error
	)
	if intentID != "" {
		rows, err = r.store.Query(ctx, sqlListExecutionsByIntent, intentID)
	} else if orgID != "" {
		rows, err = r.store.Query(ctx, sqlListExecutionsByOrg, orgID, limit, offset)
	}
	if err != nil {
		return nil, fmt.Errorf("execution: list: %w", err)
	}
	defer rows.Close()

	results := make([]Execution, 0)
	for rows.Next() {
		var e Execution
		if err := scanExecutionInto(&e, rows); err != nil {
			return nil, err
		}
		results = append(results, e)
	}
	return results, nil
}

// SavePlan persists the DAG nodes and edges for an execution.
func (r *Repository) SavePlan(ctx context.Context, plan ExecutionPlan) error {
	for _, n := range plan.Nodes {
		if _, err := r.store.Exec(ctx, sqlUpsertExecutionNode,
			n.ID, plan.ID, n.AgentRole, n.Label, n.Status, n.Progress, n.Runtime, n.Tokens, n.Cost, n.ParentID); err != nil {
			return fmt.Errorf("execution: save node %s: %w", n.ID, err)
		}
	}
	for _, e := range plan.Edges {
		if _, err := r.store.Exec(ctx, sqlUpsertExecutionEdge, e.ID, plan.ID, e.Source, e.Target); err != nil {
			return fmt.Errorf("execution: save edge %s: %w", e.ID, err)
		}
	}
	return nil
}

// GetPlan reconstructs the DAG plan for an execution.
func (r *Repository) GetPlan(ctx context.Context, executionID string) (*ExecutionPlan, error) {
	plan := &ExecutionPlan{ID: executionID}

	nodeRows, err := r.store.Query(ctx, sqlListExecutionNodes, executionID)
	if err != nil {
		return nil, err
	}
	defer nodeRows.Close()
	for nodeRows.Next() {
		var n ExecutionNode
		if err := nodeRows.Scan(&n.ID, &n.ParentID, &n.AgentRole, &n.Label, &n.Status, &n.Progress, &n.Runtime, &n.Tokens, &n.Cost, &n.ParentID); err != nil {
			return nil, fmt.Errorf("execution: scan node: %w", err)
		}
		plan.Nodes = append(plan.Nodes, n)
	}

	edgeRows, err := r.store.Query(ctx, sqlListExecutionEdges, executionID)
	if err != nil {
		return nil, err
	}
	defer edgeRows.Close()
	for edgeRows.Next() {
		var e ExecutionEdge
		if err := edgeRows.Scan(&e.ID, &e.Source, &e.Target, &e.Source, &e.Target); err != nil {
			return nil, fmt.Errorf("execution: scan edge: %w", err)
		}
		plan.Edges = append(plan.Edges, e)
	}

	return plan, nil
}

// AddEvent records an execution event.
func (r *Repository) AddEvent(ctx context.Context, e Event) error {
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now()
	}
	meta := []byte("{}")
	if e.Metadata != nil {
		meta, _ = json.Marshal(e.Metadata)
	}
	_, err := r.store.Exec(ctx, sqlUpsertExecutionEvent,
		e.ID, e.ExecutionID, e.Type, e.AgentID, e.Content, meta, e.Timestamp)
	if err != nil {
		return fmt.Errorf("execution: add event: %w", err)
	}
	return nil
}

// ListEvents returns events for an execution.
func (r *Repository) ListEvents(ctx context.Context, executionID string) ([]Event, error) {
	rows, err := r.store.Query(ctx, sqlListExecutionEvents, executionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]Event, 0)
	for rows.Next() {
		var e Event
		var meta []byte
		if err := rows.Scan(&e.ID, &e.ExecutionID, &e.Type, &e.AgentID, &e.Content, &meta, &e.Timestamp); err != nil {
			return nil, fmt.Errorf("execution: scan event: %w", err)
		}
		if len(meta) > 0 {
			_ = json.Unmarshal(meta, &e.Metadata)
		}
		results = append(results, e)
	}
	return results, nil
}

// UpdateMetrics stores execution metrics.
func (r *Repository) UpdateMetrics(ctx context.Context, m Metrics) error {
	m.Timestamp = time.Now()
	_, err := r.store.Exec(ctx, sqlUpsertExecutionMetrics,
		m.ExecutionID, m.TotalTokens, m.PromptTokens, m.CompletionTokens, m.EstimatedCost,
		m.Duration, m.AverageLatency, m.ToolsExecuted, m.Timestamp)
	if err != nil {
		return fmt.Errorf("execution: update metrics: %w", err)
	}
	return nil
}

// GetMetrics returns metrics for an execution.
func (r *Repository) GetMetrics(ctx context.Context, executionID string) (*Metrics, error) {
	row := r.store.QueryRow(ctx, sqlGetExecutionMetrics, executionID)
	var m Metrics
	err := row.Scan(&m.ExecutionID, &m.TotalTokens, &m.PromptTokens, &m.CompletionTokens,
		&m.EstimatedCost, &m.Duration, &m.AverageLatency, &m.ToolsExecuted, &m.Timestamp)
	if err != nil {
		return &Metrics{ExecutionID: executionID}, nil
	}
	return &m, nil
}

// scanner matches the Scan method on both store.Row and store.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanExecution(row store.Row, id string) (*Execution, error) {
	var e Execution
	e.ID = id
	if err := scanExecutionInto(&e, row); err != nil {
		return nil, err
	}
	return &e, nil
}

func scanExecutionInto(e *Execution, row scanner) error {
	var status string
	var startedAt, completedAt *time.Time
	if err := row.Scan(&e.ID, &e.IntentID, &e.AgentName, &status, &e.OrgID,
		&e.CreatedAt, &e.UpdatedAt, &startedAt, &completedAt); err != nil {
		return fmt.Errorf("execution: scan: %w", err)
	}
	e.Status = Status(status)
	e.StartedAt = startedAt
	e.CompletedAt = completedAt
	return nil
}