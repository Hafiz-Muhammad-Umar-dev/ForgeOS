package deployment

import (
	"fmt"
	"sync"
	"time"
)

// Package deployment provides deployment lifecycle management.

type Status string

const (
	StatusPending   Status = "pending"
	StatusDeploying Status = "deploying"
	StatusBuilding  Status = "building"
	StatusUploading Status = "uploading"
	StatusRunning   Status = "running"
	StatusHealthy   Status = "healthy"
	StatusStopped   Status = "stopped"
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"
)

type Deployment struct {
	ID          string    `json:"id"`
	ProjectName string    `json:"project_name"`
	Status      Status    `json:"status"`
	Branch      string    `json:"branch,omitempty"`
	Commit      string    `json:"commit,omitempty"`
	Image       string    `json:"image,omitempty"`
	Region      string    `json:"region,omitempty"`
	URL         string    `json:"url,omitempty"`
	CreatedBy   string    `json:"created_by,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	StartedAt   time.Time `json:"started_at,omitempty"`
	FinishedAt  time.Time `json:"finished_at,omitempty"`
}

type Log struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
}

type Stage struct {
	Name       string    `json:"name"`
	Status     Status    `json:"status"`
	StartedAt  time.Time `json:"started_at,omitempty"`
	FinishedAt time.Time `json:"finished_at,omitempty"`
}

type Manager struct {
	mu          sync.RWMutex
	deployments map[string]*Deployment
	logs        map[string][]Log
	stages      map[string][]Stage
	counter     int
}

func NewManager() *Manager {
	return &Manager{
		deployments: make(map[string]*Deployment),
		logs:        make(map[string][]Log),
		stages:      make(map[string][]Stage),
	}
}

func (m *Manager) Create(projectName, branch, commit, createdBy string) *Deployment {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counter++
	d := &Deployment{
		ID:          fmt.Sprintf("deploy-%d", m.counter),
		ProjectName: projectName,
		Status:      StatusPending,
		Branch:      branch,
		Commit:      commit,
		CreatedBy:   createdBy,
		CreatedAt:   time.Now(),
	}
	m.deployments[d.ID] = d
	return d
}

func (m *Manager) Get(id string) (*Deployment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	d, ok := m.deployments[id]
	if !ok {
		return nil, fmt.Errorf("deployment not found: %s", id)
	}
	return d, nil
}

func (m *Manager) List(project, status string, page, limit int) []*Deployment {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*Deployment
	for _, d := range m.deployments {
		if project != "" && d.ProjectName != project {
			continue
		}
		if status != "" && string(d.Status) != status {
			continue
		}
		result = append(result, d)
	}
	return result
}

func (m *Manager) UpdateStatus(id string, status Status) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	d, ok := m.deployments[id]
	if !ok {
		return fmt.Errorf("deployment not found: %s", id)
	}
	d.Status = status
	if status == StatusRunning || status == StatusHealthy {
		if d.StartedAt.IsZero() {
			d.StartedAt = time.Now()
		}
	}
	if status == StatusStopped || status == StatusFailed || status == StatusCancelled {
		d.FinishedAt = time.Now()
	}
	return nil
}

func (m *Manager) AddLog(id, level, message string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.deployments[id]; !ok {
		return fmt.Errorf("deployment not found: %s", id)
	}
	m.logs[id] = append(m.logs[id], Log{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
	})
	return nil
}

func (m *Manager) Logs(id string) []Log {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.logs[id]
}

func (m *Manager) AddStage(id, name string, status Status) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.deployments[id]; !ok {
		return fmt.Errorf("deployment not found: %s", id)
	}
	m.stages[id] = append(m.stages[id], Stage{
		Name:      name,
		Status:    status,
		StartedAt: time.Now(),
	})
	return nil
}

func (m *Manager) Stages(id string) []Stage {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stages[id]
}
