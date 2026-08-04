package budget

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/lifecycle"
)

// Compile-time check.
var (
	_ Governor            = (*Service)(nil)
	_ lifecycle.Component = (*Service)(nil)
)

// Service implements the Governor port with a store-backed budget ledger.
// It enforces token ceilings and records usage.
type Service struct {
	store BudgetStore
	mu    sync.Mutex
	cache map[string]Budget
}

// NewService creates a new budget governor service.
func NewService(s BudgetStore) *Service {
	return &Service{
		store: s,
		cache: make(map[string]Budget),
	}
}

// Name returns "budget" for the lifecycle manager.
func (s *Service) Name() string { return "budget" }

// Init validates dependencies.
func (s *Service) Init(_ context.Context) error {
	if s.store == nil {
		return fmt.Errorf("budget: store is required")
	}
	return nil
}

// Start loads budgets.
func (s *Service) Start(_ context.Context) error {
	log.Printf("budget: started")
	return nil
}

// Stop flushes and clears the cache.
func (s *Service) Stop(_ context.Context) error {
	s.mu.Lock()
	s.cache = nil
	s.mu.Unlock()
	return nil
}

// Health reports whether the service is ready.
func (s *Service) Health() lifecycle.Health {
	return lifecycle.Health{Status: lifecycle.StatusUp, Since: time.Now()}
}

// Check verifies the budget has capacity.
func (s *Service) Check(ctx context.Context, orgID string) (Budget, error) {
	b, err := s.getOrCreate(ctx, orgID)
	if err != nil {
		return Budget{}, err
	}

	if b.TokenRemaining <= 0 {
		return b, fmt.Errorf("%w: org=%s remaining=%d", ErrBudgetExceeded, orgID, b.TokenRemaining)
	}
	return b, nil
}

// Consume records token usage. Safe for concurrent calls.
func (s *Service) Consume(ctx context.Context, orgID string, usage Usage) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	b, err := s.store.AddUsage(ctx, orgID, usage.TotalTokens())
	if err != nil {
		return fmt.Errorf("budget: consume: %w", err)
	}
	s.cache[orgID] = b

	log.Printf("budget: consumed %d tokens for org=%s (used=%d/%d)",
		usage.TotalTokens(), orgID, b.TokenUsed, b.TokenCeiling)
	return nil
}

func (s *Service) getOrCreate(ctx context.Context, orgID string) (Budget, error) {
	s.mu.Lock()
	if b, ok := s.cache[orgID]; ok {
		s.mu.Unlock()
		return b, nil
	}
	s.mu.Unlock()

	b, err := s.store.GetOrCreate(ctx, orgID)
	if err != nil {
		return Budget{}, fmt.Errorf("budget: get: %w", err)
	}

	s.mu.Lock()
	s.cache[orgID] = b
	s.mu.Unlock()
	return b, nil
}
