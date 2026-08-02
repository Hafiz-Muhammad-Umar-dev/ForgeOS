package intents

import "context"

// Service is the application service for intents and tasks.
// It encapsulates business rules and orchestrates repository access.
type Service struct {
	repo *Repository
}

// NewService creates an intents service backed by the given repository.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// CreateIntent creates a new intent and returns it.
func (s *Service) CreateIntent(ctx context.Context, in NewIntentRequest) (*Intent, error) {
	if in.Text == "" {
		return nil, ErrInvalidInput
	}
	if in.OrgID == "" {
		in.OrgID = "default"
	}
	return s.repo.CreateIntent(ctx, in)
}

// GetIntent returns a single intent by ID.
func (s *Service) GetIntent(ctx context.Context, id string) (*Intent, error) {
	return s.repo.GetIntent(ctx, id)
}

// ListIntents returns intents with pagination, optionally filtered by org.
func (s *Service) ListIntents(ctx context.Context, orgID string, limit, offset int) ([]Intent, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListIntents(ctx, orgID, limit, offset)
}

// ListTasks returns tasks for an intent, or all tasks when intentID is empty.
func (s *Service) ListTasks(ctx context.Context, intentID string) ([]Task, error) {
	return s.repo.ListTasks(ctx, intentID)
}
