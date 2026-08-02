package intents

import (
	"context"
	"fmt"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/store"
)

// Named SQL statements for intents and tasks.
const (
	sqlCreateIntent = `INSERT INTO intents
		(id, user_id, org_id, project_id, trace_id, text, status, summary, error, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, 'pending', '', '', NOW(), NOW())
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			text = EXCLUDED.text,
			updated_at = NOW()`

	sqlGetIntent = `SELECT id, user_id, org_id, project_id, trace_id, text, status, summary, error,
		created_at, updated_at FROM intents WHERE id = $1`

	sqlListIntents = `SELECT id, user_id, org_id, project_id, trace_id, text, status, summary, error,
		created_at, updated_at FROM intents ORDER BY created_at DESC LIMIT $1 OFFSET $2`

	sqlListIntentsByOrg = `SELECT id, user_id, org_id, project_id, trace_id, text, status, summary, error,
		created_at, updated_at FROM intents WHERE org_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`

	sqlListTasksByIntent = `SELECT id, intent_id, agent_name, status, summary, error, input_tokens, output_tokens,
		created_at, updated_at FROM tasks WHERE intent_id = $1 ORDER BY created_at`

	sqlListTasks = `SELECT id, intent_id, agent_name, status, summary, error, input_tokens, output_tokens,
		created_at, updated_at FROM tasks ORDER BY created_at`
)

// Repository provides database operations for intents and tasks.
type Repository struct {
	store store.Store
}

// NewRepository creates an intents repository backed by the given store.
func NewRepository(s store.Store) *Repository {
	return &Repository{store: s}
}

// CreateIntent inserts a new intent and returns it.
func (r *Repository) CreateIntent(ctx context.Context, in NewIntentRequest) (*Intent, error) {
	id := newID()
	if _, err := r.store.Exec(ctx, sqlCreateIntent,
		id, in.UserID, in.OrgID, in.ProjectID, in.TraceID, in.Text,
	); err != nil {
		return nil, fmt.Errorf("intents: create: %w", err)
	}
	return r.GetIntent(ctx, id)
}

// GetIntent returns a single intent by ID.
func (r *Repository) GetIntent(ctx context.Context, id string) (*Intent, error) {
	row := r.store.QueryRow(ctx, sqlGetIntent, id)
	var it Intent
	if err := row.Scan(&it.ID, &it.UserID, &it.OrgID, &it.ProjectID, &it.TraceID,
		&it.Text, &it.Status, &it.Summary, &it.Error, &it.CreatedAt, &it.UpdatedAt); err != nil {
		return nil, fmt.Errorf("intents: get %s: %w", id, err)
	}
	return &it, nil
}

// ListIntents returns intents, optionally filtered by org, with pagination.
func (r *Repository) ListIntents(ctx context.Context, orgID string, limit, offset int) ([]Intent, error) {
	var (
		rows store.Rows
		err  error
	)
	if orgID != "" {
		rows, err = r.store.Query(ctx, sqlListIntentsByOrg, orgID, limit, offset)
	} else {
		rows, err = r.store.Query(ctx, sqlListIntents, limit, offset)
	}
	if err != nil {
		return nil, fmt.Errorf("intents: list: %w", err)
	}
	defer rows.Close()

	results := make([]Intent, 0)
	for rows.Next() {
		var it Intent
		if err := rows.Scan(&it.ID, &it.UserID, &it.OrgID, &it.ProjectID, &it.TraceID,
			&it.Text, &it.Status, &it.Summary, &it.Error, &it.CreatedAt, &it.UpdatedAt); err != nil {
			return nil, fmt.Errorf("intents: scan: %w", err)
		}
		results = append(results, it)
	}
	return results, nil
}

// ListTasks returns tasks for an intent, or all tasks when intentID is empty.
func (r *Repository) ListTasks(ctx context.Context, intentID string) ([]Task, error) {
	var (
		rows store.Rows
		err  error
	)
	if intentID != "" {
		rows, err = r.store.Query(ctx, sqlListTasksByIntent, intentID)
	} else {
		rows, err = r.store.Query(ctx, sqlListTasks)
	}
	if err != nil {
		return nil, fmt.Errorf("intents: list tasks: %w", err)
	}
	defer rows.Close()

	results := make([]Task, 0)
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.ID, &t.IntentID, &t.AgentName, &t.Status, &t.Summary,
			&t.Error, &t.InputTokens, &t.OutputTokens, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("intents: scan task: %w", err)
		}
		results = append(results, t)
	}
	return results, nil
}
