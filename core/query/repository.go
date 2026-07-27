package query

import (
	"context"
	"fmt"
	"time"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/store"
)

// Named SQL query constants.
const (
	sqlInsertIntent = `INSERT INTO query_intents
		(id, user_id, org_id, project_id, trace_id, text, status, summary, error, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			summary = EXCLUDED.summary,
			error = EXCLUDED.error,
			updated_at = NOW()`

	sqlSelectIntent = `SELECT id, user_id, org_id, project_id, trace_id, text, status, summary, error,
		created_at, updated_at FROM query_intents WHERE id = $1`

	sqlSelectIntentsByOrg = `SELECT id, user_id, org_id, project_id, trace_id, text, status, summary, error,
		created_at, updated_at FROM query_intents WHERE org_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`

	sqlSelectIntentsByOrgAndStatus = `SELECT id, user_id, org_id, project_id, trace_id, text, status, summary, error,
		created_at, updated_at FROM query_intents WHERE org_id = $1 AND status = $2 ORDER BY created_at DESC LIMIT $3 OFFSET $4`

	sqlInsertTask = `INSERT INTO query_tasks
		(id, intent_id, agent_name, status, summary, error, input_tokens, output_tokens, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			summary = EXCLUDED.summary,
			error = EXCLUDED.error,
			updated_at = NOW()`

	sqlSelectTask = `SELECT id, intent_id, agent_name, status, summary, error, input_tokens, output_tokens,
		created_at, updated_at FROM query_tasks WHERE id = $1`

	sqlSelectTasksByIntent = `SELECT id, intent_id, agent_name, status, summary, error, input_tokens, output_tokens,
		created_at, updated_at FROM query_tasks WHERE intent_id = $1 ORDER BY created_at`
)

// Repository provides all database operations for the Query Service.
// All SQL is defined as named constants for maintainability.
type Repository struct {
	readModel *store.ReadModel
}

// NewRepository creates a Repository backed by the given store.
func NewRepository(s store.Store) *Repository {
	return &Repository{readModel: store.NewReadModel(s)}
}

// ---------------------------------------------------------------------------
// Intent operations
// ---------------------------------------------------------------------------

// UpsertIntent inserts or updates an intent read model row.
func (r *Repository) UpsertIntent(ctx context.Context, id, userID, orgID, projectID, traceID, text, status, summary, errMsg string, createdAt time.Time) error {
	_, err := r.readModel.Exec(ctx, sqlInsertIntent,
		id, userID, orgID, projectID, traceID, text, status, summary, errMsg, createdAt)
	return err
}

// GetIntent returns a single intent by ID.
func (r *Repository) GetIntent(ctx context.Context, id string) (IntentView, error) {
	row := r.readModel.QueryRow(ctx, sqlSelectIntent, id)
	return scanIntentView(row)
}

// ListIntents returns intents for an organization with pagination.
func (r *Repository) ListIntents(ctx context.Context, orgID string, limit, offset int) ([]IntentView, error) {
	rows, err := r.readModel.Query(ctx, sqlSelectIntentsByOrg, orgID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanIntentViews(rows)
}

// ListIntentsByStatus returns intents filtered by status with pagination.
func (r *Repository) ListIntentsByStatus(ctx context.Context, orgID, status string, limit, offset int) ([]IntentView, error) {
	rows, err := r.readModel.Query(ctx, sqlSelectIntentsByOrgAndStatus, orgID, status, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanIntentViews(rows)
}

// ---------------------------------------------------------------------------
// Task operations
// ---------------------------------------------------------------------------

// UpsertTask inserts or updates a task read model row.
func (r *Repository) UpsertTask(ctx context.Context, id, intentID, agentName, status, summary, errMsg string, inputTokens, outputTokens int, createdAt time.Time) error {
	_, err := r.readModel.Exec(ctx, sqlInsertTask,
		id, intentID, agentName, status, summary, errMsg, inputTokens, outputTokens, createdAt)
	return err
}

// GetTask returns a single task by ID.
func (r *Repository) GetTask(ctx context.Context, id string) (TaskView, error) {
	row := r.readModel.QueryRow(ctx, sqlSelectTask, id)
	return scanTaskView(row)
}

// ListTasksByIntent returns all tasks for an intent.
func (r *Repository) ListTasksByIntent(ctx context.Context, intentID string) ([]TaskView, error) {
	rows, err := r.readModel.Query(ctx, sqlSelectTasksByIntent, intentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTaskViews(rows)
}

// ---------------------------------------------------------------------------
// Scan helpers
// ---------------------------------------------------------------------------

func scanIntentView(row store.Row) (IntentView, error) {
	var v IntentView
	err := row.Scan(&v.ID, &v.UserID, &v.OrgID, &v.ProjectID, &v.TraceID,
		&v.Text, &v.Status, &v.Summary, &v.Error, &v.CreatedAt, &v.UpdatedAt)
	if err != nil {
		return IntentView{}, fmt.Errorf("%w: %w", ErrIntentNotFound, store.MapPGError(err))
	}
	return v, nil
}

func scanIntentViews(rows store.Rows) ([]IntentView, error) {
	var results []IntentView
	for rows.Next() {
		var v IntentView
		if err := rows.Scan(&v.ID, &v.UserID, &v.OrgID, &v.ProjectID, &v.TraceID,
			&v.Text, &v.Status, &v.Summary, &v.Error, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, err
		}
		results = append(results, v)
	}
	return results, nil
}

func scanTaskView(row store.Row) (TaskView, error) {
	var v TaskView
	err := row.Scan(&v.ID, &v.IntentID, &v.AgentName, &v.Status, &v.Summary, &v.Error,
		&v.InputTokens, &v.OutputTokens, &v.CreatedAt, &v.UpdatedAt)
	if err != nil {
		return TaskView{}, fmt.Errorf("%w: %w", ErrTaskNotFound, store.MapPGError(err))
	}
	return v, nil
}

func scanTaskViews(rows store.Rows) ([]TaskView, error) {
	var results []TaskView
	for rows.Next() {
		var v TaskView
		if err := rows.Scan(&v.ID, &v.IntentID, &v.AgentName, &v.Status, &v.Summary, &v.Error,
			&v.InputTokens, &v.OutputTokens, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, err
		}
		results = append(results, v)
	}
	return results, nil
}
