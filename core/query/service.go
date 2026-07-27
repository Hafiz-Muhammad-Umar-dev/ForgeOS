package query

import (
	"context"
	"fmt"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/store"
)

// Compile-time check.
var _ QueryService = (*Service)(nil)

// Service implements QueryService by reading from the database repository.
type Service struct {
	repo *Repository
}

// NewService creates a new QueryService backed by the given store.
func NewService(s store.Store) *Service {
	return &Service{repo: NewRepository(s)}
}

// GetIntent returns a single intent by ID.
func (s *Service) GetIntent(ctx context.Context, id string) (IntentView, error) {
	return s.repo.GetIntent(ctx, id)
}

// ListIntents returns intents for an organization with pagination.
func (s *Service) ListIntents(ctx context.Context, orgID string, limit, offset int) ([]IntentView, error) {
	return s.repo.ListIntents(ctx, orgID, limit, offset)
}

// GetTask returns a single task by ID.
func (s *Service) GetTask(ctx context.Context, id string) (TaskView, error) {
	return s.repo.GetTask(ctx, id)
}

// ListTasks returns all tasks for an intent.
func (s *Service) ListTasks(ctx context.Context, intentID string) ([]TaskView, error) {
	return s.repo.ListTasksByIntent(ctx, intentID)
}

// Ping verifies the backing store is reachable.
func (s *Service) Ping(ctx context.Context) error {
	if pg, ok := s.repo.readModel.Store().(interface{ Ping(context.Context) error }); ok {
		return pg.Ping(ctx)
	}
	return fmt.Errorf("query: store does not support ping")
}
