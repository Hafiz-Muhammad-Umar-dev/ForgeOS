package budget

import (
	"context"
	"fmt"
	"time"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/store"
)

// BudgetStore persists budget state and usage history.
type BudgetStore interface {
	// GetOrCreate returns the current budget for an org, creating a default if none exists.
	GetOrCreate(ctx context.Context, orgID string) (Budget, error)

	// AddUsage atomically adds token usage and returns the updated budget.
	AddUsage(ctx context.Context, orgID string, tokens int64) (Budget, error)
}

// PGBudgetStore implements BudgetStore using the DevOS store.
type PGBudgetStore struct {
	readModel *store.ReadModel
}

// NewPGBudgetStore creates a PGBudgetStore.
func NewPGBudgetStore(s store.Store) *PGBudgetStore {
	return &PGBudgetStore{readModel: store.NewReadModel(s)}
}

var defaultTokenCeiling int64 = 1000000

func (s *PGBudgetStore) GetOrCreate(ctx context.Context, orgID string) (Budget, error) {
	row := s.readModel.QueryRow(ctx,
		`SELECT org_id, token_ceiling, token_used, created_at
		 FROM budget_ledger WHERE org_id = $1`, orgID)

	var b Budget
	var createdAt time.Time
	err := row.Scan(&b.OrgID, &b.TokenCeiling, &b.TokenUsed, &createdAt)
	if err == nil {
		b.TokenRemaining = b.TokenCeiling - b.TokenUsed
		if b.TokenRemaining <= 0 {
			b.Status = BudgetExhausted
		} else {
			b.Status = BudgetOK
		}
		return b, nil
	}

	// Create a default budget.
	b = Budget{
		OrgID:          orgID,
		TokenCeiling:   defaultTokenCeiling,
		TokenUsed:      0,
		TokenRemaining: defaultTokenCeiling,
		Status:         BudgetOK,
		ResetAt:        time.Now().Add(24 * time.Hour),
	}
	_, err = s.readModel.Exec(ctx,
		`INSERT INTO budget_ledger (org_id, token_ceiling, token_used, created_at)
		 VALUES ($1, $2, 0, NOW())`, orgID, defaultTokenCeiling)
	if err != nil {
		return Budget{}, fmt.Errorf("budget: create: %w", store.MapPGError(err))
	}
	return b, nil
}

func (s *PGBudgetStore) AddUsage(ctx context.Context, orgID string, tokens int64) (Budget, error) {
	_, err := s.readModel.Exec(ctx,
		`UPDATE budget_ledger SET token_used = token_used + $1 WHERE org_id = $2`, tokens, orgID)
	if err != nil {
		return Budget{}, fmt.Errorf("budget: add usage: %w", store.MapPGError(err))
	}
	return s.GetOrCreate(ctx, orgID)
}

// Migrations returns SQL migrations for the budget package.
func Migrations() []store.Migration {
	return []store.Migration{
		{
			Version:     1,
			Description: "create budget_ledger table",
			Up: `
				CREATE TABLE IF NOT EXISTS budget_ledger (
					org_id         TEXT PRIMARY KEY,
					token_ceiling  BIGINT NOT NULL DEFAULT 1000000,
					token_used     BIGINT NOT NULL DEFAULT 0,
					created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
					updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
				);
			`,
			Down: `DROP TABLE IF EXISTS budget_ledger;`,
		},
	}
}
