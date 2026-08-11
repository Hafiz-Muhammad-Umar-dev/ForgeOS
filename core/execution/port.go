package execution

import (
	"context"
	"fmt"
	"time"
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusPaused    Status = "paused"
)

type Execution struct {
	ID          string     `json:"id"`
	IntentID    string     `json:"intent_id"`
	AgentName   string     `json:"agent_name"`
	Status      Status     `json:"status"`
	OrgID       string     `json:"org_id,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

type Event struct {
	ID          string            `json:"id"`
	ExecutionID string            `json:"execution_id"`
	Type        string            `json:"type"`
	AgentID     string            `json:"agent_id,omitempty"`
	Content     string            `json:"content"`
	Metadata    map[string]any    `json:"metadata,omitempty"`
	Timestamp   time.Time         `json:"timestamp"`
}

type Metrics struct {
	ExecutionID      string    `json:"execution_id"`
	TotalTokens      int       `json:"total_tokens"`
	PromptTokens     int       `json:"prompt_tokens"`
	CompletionTokens int       `json:"completion_tokens"`
	EstimatedCost    float64   `json:"estimated_cost"`
	Duration         float64   `json:"duration"`
	AverageLatency   float64   `json:"average_latency"`
	ToolsExecuted    int       `json:"tools_executed"`
	Timestamp        time.Time `json:"timestamp"`
}

type Manager struct {
	executions map[string]*Execution
	events     map[string][]Event
	metrics    map[string]*Metrics
}

func NewManager() *Manager {
	return &Manager{
		executions: make(map[string]*Execution),
		events:     make(map[string][]Event),
		metrics:    make(map[string]*Metrics),
	}
}

func (m *Manager) Create(ctx context.Context, intentID, agentName string) (*Execution, error) {
	e := &Execution{
		ID:        fmt.Sprintf("exec-%d", time.Now().UnixNano()),
		IntentID:  intentID,
		AgentName: agentName,
		Status:    StatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.executions[e.ID] = e
	return e, nil
}

func (m *Manager) Get(id string) (*Execution, error) {
	e, ok := m.executions[id]
	if !ok {
		return nil, fmt.Errorf("execution not found: %s", id)
	}
	return e, nil
}

func (m *Manager) List(intentID string) []*Execution {
	var result []*Execution
	for _, e := range m.executions {
		if intentID == "" || e.IntentID == intentID {
			result = append(result, e)
		}
	}
	return result
}

func (m *Manager) UpdateStatus(id string, status Status) error {
	e, ok := m.executions[id]
	if !ok {
		return fmt.Errorf("execution not found: %s", id)
	}
	e.Status = status
	e.UpdatedAt = time.Now()
	return nil
}

func (m *Manager) AddEvent(executionID, eventType, content string) (*Event, error) {
	e := &Event{
		ID:          fmt.Sprintf("evt-%d", time.Now().UnixNano()),
		ExecutionID: executionID,
		Type:        eventType,
		Content:     content,
		Timestamp:   time.Now(),
	}
	m.events[executionID] = append(m.events[executionID], *e)
	return e, nil
}

func (m *Manager) Events(executionID string) []Event {
	return m.events[executionID]
}

func (m *Manager) UpdateMetrics(id string, metrics *Metrics) error {
	metrics.ExecutionID = id
	metrics.Timestamp = time.Now()
	m.metrics[id] = metrics
	return nil
}

func (m *Manager) GetMetrics(id string) (*Metrics, error) {
	mt, ok := m.metrics[id]
	if !ok {
		return &Metrics{ExecutionID: id}, nil
	}
	return mt, nil
}
