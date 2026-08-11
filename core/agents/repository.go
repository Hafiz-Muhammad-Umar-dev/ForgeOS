package agents

import (
	"context"
	"fmt"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/store"
)

// Named SQL statements for agents.
const (
	sqlUpsertAgent = `INSERT INTO agents
		(id, org_id, name, role, status, model, temperature, current_tool,
		 reasoning, memory, output, queue_length, execution_time,
		 prompt_tokens, completion_tokens, cost, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, NOW(), NOW())
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			current_tool = EXCLUDED.current_tool,
			reasoning = EXCLUDED.reasoning,
			output = EXCLUDED.output,
			queue_length = EXCLUDED.queue_length,
			execution_time = EXCLUDED.execution_time,
			prompt_tokens = EXCLUDED.prompt_tokens,
			completion_tokens = EXCLUDED.completion_tokens,
			cost = EXCLUDED.cost,
			updated_at = NOW()`

	sqlGetAgent = `SELECT id, org_id, name, role, status, model, temperature, current_tool,
		reasoning, memory, output, queue_length, execution_time,
		prompt_tokens, completion_tokens, cost, created_at, updated_at
		FROM agents WHERE id = $1`

	sqlListAgents = `SELECT id, org_id, name, role, status, model, temperature, current_tool,
		reasoning, memory, output, queue_length, execution_time,
		prompt_tokens, completion_tokens, cost, created_at, updated_at
		FROM agents ORDER BY name`

	sqlListAgentsByOrg = `SELECT id, org_id, name, role, status, model, temperature, current_tool,
		reasoning, memory, output, queue_length, execution_time,
		prompt_tokens, completion_tokens, cost, created_at, updated_at
		FROM agents WHERE org_id = $1 ORDER BY name`

	sqlUpdateAgentStatus = `UPDATE agents SET status = $1, updated_at = NOW() WHERE id = $2`
)

// Repository provides database operations for agents.
type Repository struct {
	store store.Store
}

// NewRepository creates an agents repository backed by the given store.
func NewRepository(s store.Store) *Repository {
	return &Repository{store: s}
}

// UpsertAgent inserts or updates an agent.
func (r *Repository) UpsertAgent(ctx context.Context, a Agent) error {
	_, err := r.store.Exec(ctx, sqlUpsertAgent,
		a.ID, a.OrgID, a.Name, string(a.Role), string(a.Status), a.Model, a.Temperature,
		a.CurrentTool, a.Reasoning, a.Memory, a.Output, a.QueueLength, a.ExecutionTime,
		a.PromptTokens, a.CompletionTokens, a.Cost)
	if err != nil {
		return fmt.Errorf("agents: upsert: %w", err)
	}
	return nil
}

// GetAgent returns a single agent by ID.
func (r *Repository) GetAgent(ctx context.Context, id string) (*Agent, error) {
	row := r.store.QueryRow(ctx, sqlGetAgent, id)
	return scanAgent(row, id)
}

// ListAgents returns agents, optionally scoped to an org.
func (r *Repository) ListAgents(ctx context.Context, orgID string) ([]Agent, error) {
	var (
		rows store.Rows
		err  error
	)
	if orgID != "" {
		rows, err = r.store.Query(ctx, sqlListAgentsByOrg, orgID)
	} else {
		rows, err = r.store.Query(ctx, sqlListAgents)
	}
	if err != nil {
		return nil, fmt.Errorf("agents: list: %w", err)
	}
	defer rows.Close()

	results := make([]Agent, 0)
	for rows.Next() {
		var a Agent
		if err := scanAgentInto(&a, rows); err != nil {
			return nil, err
		}
		results = append(results, a)
	}
	return results, nil
}

// UpdateAgentStatus updates an agent's runtime status.
func (r *Repository) UpdateAgentStatus(ctx context.Context, id string, status AgentStatus) error {
	_, err := r.store.Exec(ctx, sqlUpdateAgentStatus, string(status), id)
	if err != nil {
		return fmt.Errorf("agents: update status: %w", err)
	}
	return nil
}

// scanner matches the Scan method on both store.Row and store.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanAgent(row store.Row, id string) (*Agent, error) {
	var a Agent
	a.ID = id
	if err := scanAgentInto(&a, row); err != nil {
		return nil, err
	}
	return &a, nil
}

func scanAgentInto(a *Agent, row scanner) error {
	var role, status string
	if err := row.Scan(&a.ID, &a.OrgID, &a.Name, &role, &status, &a.Model, &a.Temperature,
		&a.CurrentTool, &a.Reasoning, &a.Memory, &a.Output, &a.QueueLength, &a.ExecutionTime,
		&a.PromptTokens, &a.CompletionTokens, &a.Cost, &a.CreatedAt, &a.UpdatedAt); err != nil {
		return fmt.Errorf("agents: scan: %w", err)
	}
	a.Role = AgentRole(role)
	a.Status = AgentStatus(status)
	return nil
}