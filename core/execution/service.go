package execution

import (
	"context"
	"time"
)

// Service is the application service for executions.
// It enforces state transitions and orchestrates repository access.
type Service struct {
	repo *Repository
	now  func() time.Time
}

// NewService creates an executions service backed by the given repository.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo, now: time.Now}
}

// GetExecution returns a single execution by ID.
func (s *Service) GetExecution(ctx context.Context, id string) (*Execution, error) {
	return s.repo.GetExecution(ctx, id)
}

// ListExecutions returns executions for an org or intent.
func (s *Service) ListExecutions(ctx context.Context, orgID, intentID string, limit, offset int) ([]Execution, error) {
	return s.repo.ListExecutions(ctx, orgID, intentID, limit, offset)
}

// ApplyAction executes a state-transition action against an execution.
func (s *Service) ApplyAction(ctx context.Context, id string, action Action) (*Execution, error) {
	e, err := s.repo.GetExecution(ctx, id)
	if err != nil {
		return nil, err
	}

	target, err := TargetStatusForAction(e.Status, action)
	if err != nil {
		return nil, err
	}

	now := s.now()
	switch target {
	case StatusRunning:
		if e.Status == StatusPending {
			e.StartedAt = &now
		}
	case StatusFailed:
		if action == ActionStop {
			e.CompletedAt = &now
		}
	case StatusCompleted:
		e.CompletedAt = &now
	}

	e.Status = target
	e.UpdatedAt = now

	if err := s.repo.UpsertExecution(ctx, *e); err != nil {
		return nil, err
	}
	return e, nil
}

// SavePlan persists the DAG plan for an execution.
func (s *Service) SavePlan(ctx context.Context, plan ExecutionPlan) error {
	return s.repo.SavePlan(ctx, plan)
}

// GetPlan returns the DAG plan for an execution.
func (s *Service) GetPlan(ctx context.Context, executionID string) (*ExecutionPlan, error) {
	return s.repo.GetPlan(ctx, executionID)
}

// AddEvent records an execution event.
func (s *Service) AddEvent(ctx context.Context, executionID, eventType, agentID, content string, metadata map[string]any) error {
	e := Event{
		ID:          newID(),
		ExecutionID: executionID,
		Type:        eventType,
		AgentID:     agentID,
		Content:     content,
		Metadata:    metadata,
		Timestamp:   s.now(),
	}
	return s.repo.AddEvent(ctx, e)
}

// ListEvents returns events for an execution.
func (s *Service) ListEvents(ctx context.Context, executionID string) ([]Event, error) {
	return s.repo.ListEvents(ctx, executionID)
}

// UpdateMetrics stores execution metrics.
func (s *Service) UpdateMetrics(ctx context.Context, m Metrics) error {
	return s.repo.UpdateMetrics(ctx, m)
}

// GetMetrics returns metrics for an execution.
func (s *Service) GetMetrics(ctx context.Context, executionID string) (*Metrics, error) {
	return s.repo.GetMetrics(ctx, executionID)
}

// GetOrCreateExecutionForIntent returns the latest execution for an intent,
// or creates a new one if none exists.
func (s *Service) GetOrCreateExecutionForIntent(ctx context.Context, intentID, agentName string) (*Execution, error) {
	e, err := s.repo.GetLatestExecutionForIntent(ctx, intentID)
	if err == nil {
		return e, nil
	}
	// Create a new execution for this intent
	now := s.now()
	e = &Execution{
		ID:        newID(),
		IntentID:  intentID,
		AgentName: agentName,
		Status:    StatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repo.UpsertExecution(ctx, *e); err != nil {
		return nil, err
	}
	return e, nil
}