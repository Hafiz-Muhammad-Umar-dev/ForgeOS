package hitl

import (
	"context"
	"fmt"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/store"
)

// ApprovalStore persists approval requests and decisions.
type ApprovalStore interface {
	// Create inserts a new approval request.
	Create(ctx context.Context, req ApprovalRequest) error

	// Get returns an approval request by ID.
	Get(ctx context.Context, id string) (ApprovalRequest, error)

	// UpdateStatus updates the status and decision metadata.
	UpdateStatus(ctx context.Context, id string, status ApprovalStatus, approvedBy, reason string) error
}

// PGApprovalStore implements ApprovalStore using the DevOS store.
type PGApprovalStore struct {
	readModel *store.ReadModel
}

func NewPGApprovalStore(s store.Store) *PGApprovalStore {
	return &PGApprovalStore{readModel: store.NewReadModel(s)}
}

func (s *PGApprovalStore) Create(ctx context.Context, req ApprovalRequest) error {
	_, err := s.readModel.Exec(ctx,
		`INSERT INTO hitl_approvals (id, intent_id, type, summary, node_id, status, expires_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 ON CONFLICT (id) DO NOTHING`,
		req.ID, req.IntentID, string(req.Type), req.Summary, req.NodeID,
		string(req.Status), req.ExpiresAt, req.CreatedAt)
	return err
}

func (s *PGApprovalStore) Get(ctx context.Context, id string) (ApprovalRequest, error) {
	row := s.readModel.QueryRow(ctx,
		`SELECT id, intent_id, type, summary, node_id, status, expires_at, created_at
		 FROM hitl_approvals WHERE id = $1`, id)

	var req ApprovalRequest
	var status, aType string
	err := row.Scan(&req.ID, &req.IntentID, &aType, &req.Summary, &req.NodeID,
		&status, &req.ExpiresAt, &req.CreatedAt)
	if err != nil {
		return ApprovalRequest{}, fmt.Errorf("hitl: get approval: %w", store.MapPGError(err))
	}
	req.Status = ApprovalStatus(status)
	req.Type = ApprovalType(aType)
	return req, nil
}

func (s *PGApprovalStore) UpdateStatus(ctx context.Context, id string, status ApprovalStatus, approvedBy, reason string) error {
	_, err := s.readModel.Exec(ctx,
		`UPDATE hitl_approvals SET status = $1, approved_by = $2, reason = $3, decided_at = NOW()
		 WHERE id = $4`,
		string(status), approvedBy, reason, id)
	return err
}

// Migrations returns SQL migrations for the hitl package.
func Migrations() []store.Migration {
	return []store.Migration{
		{
			Version:     1,
			Description: "create hitl_approvals table",
			Up: `
				CREATE TABLE IF NOT EXISTS hitl_approvals (
					id          TEXT PRIMARY KEY,
					intent_id   TEXT NOT NULL,
					type        TEXT NOT NULL,
					summary     TEXT NOT NULL DEFAULT '',
					node_id     TEXT NOT NULL DEFAULT '',
					status      TEXT NOT NULL DEFAULT 'pending',
					approved_by TEXT NOT NULL DEFAULT '',
					reason      TEXT NOT NULL DEFAULT '',
					expires_at  TIMESTAMPTZ NOT NULL,
					created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
					decided_at  TIMESTAMPTZ
				);
			`,
			Down: `DROP TABLE IF EXISTS hitl_approvals;`,
		},
	}
}
