package agents

import "context"

// Service is the application service for agents.
// It encapsulates business rules and orchestrates repository access.
type Service struct {
	repo *Repository
}

// NewService creates an agents service backed by the given repository.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// ListAgents returns all agents, optionally scoped to an org.
func (s *Service) ListAgents(ctx context.Context, orgID string) ([]Agent, error) {
	return s.repo.ListAgents(ctx, orgID)
}

// GetAgent returns a single agent by ID.
func (s *Service) GetAgent(ctx context.Context, id string) (*Agent, error) {
	return s.repo.GetAgent(ctx, id)
}

// RegisterAgent inserts or updates an agent's metadata.
func (s *Service) RegisterAgent(ctx context.Context, a Agent) error {
	if a.ID == "" {
		return ErrInvalidInput
	}
	if a.Name == "" {
		a.Name = a.ID
	}
	if a.Role == "" {
		a.Role = RoleCoder
	}
	if a.Status == "" {
		a.Status = AgentIdle
	}
	return s.repo.UpsertAgent(ctx, a)
}

// UpdateAgentStatus updates an agent's runtime status.
func (s *Service) UpdateAgentStatus(ctx context.Context, id string, status AgentStatus) error {
	if status == "" {
		return ErrInvalidInput
	}
	return s.repo.UpdateAgentStatus(ctx, id, status)
}