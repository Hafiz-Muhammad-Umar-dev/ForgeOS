package query

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Compile-time check.
var _ QueryService = (*FakeQueryService)(nil)

// FakeQueryService is an in-memory QueryService implementation for testing.
type FakeQueryService struct {
	Intents map[string]IntentView
	Tasks   map[string]TaskView

	mu        sync.Mutex
	GetCount  atomic.Int64
	ListCount atomic.Int64
}

// NewFakeQueryService creates an empty FakeQueryService.
func NewFakeQueryService() *FakeQueryService {
	return &FakeQueryService{
		Intents: make(map[string]IntentView),
		Tasks:   make(map[string]TaskView),
	}
}

func (f *FakeQueryService) GetIntent(_ context.Context, id string) (IntentView, error) {
	f.GetCount.Add(1)
	f.mu.Lock()
	defer f.mu.Unlock()
	v, ok := f.Intents[id]
	if !ok {
		return IntentView{}, fmt.Errorf("%w: id=%s", ErrIntentNotFound, id)
	}
	return v, nil
}

func (f *FakeQueryService) ListIntents(_ context.Context, orgID string, limit, offset int) ([]IntentView, error) {
	f.ListCount.Add(1)
	f.mu.Lock()
	defer f.mu.Unlock()
	var results []IntentView
	for _, v := range f.Intents {
		if v.OrgID == orgID {
			results = append(results, v)
		}
	}
	// Apply pagination
	if offset >= len(results) {
		return nil, nil
	}
	end := offset + limit
	if end > len(results) {
		end = len(results)
	}
	return results[offset:end], nil
}

func (f *FakeQueryService) GetTask(_ context.Context, id string) (TaskView, error) {
	f.GetCount.Add(1)
	f.mu.Lock()
	defer f.mu.Unlock()
	v, ok := f.Tasks[id]
	if !ok {
		return TaskView{}, fmt.Errorf("%w: id=%s", ErrTaskNotFound, id)
	}
	return v, nil
}

func (f *FakeQueryService) ListTasks(_ context.Context, intentID string) ([]TaskView, error) {
	f.ListCount.Add(1)
	f.mu.Lock()
	defer f.mu.Unlock()
	var results []TaskView
	for _, v := range f.Tasks {
		if v.IntentID == intentID {
			results = append(results, v)
		}
	}
	if results == nil {
		return []TaskView{}, nil
	}
	return results, nil
}

func (f *FakeQueryService) Ping(_ context.Context) error { return nil }

// AddIntent adds a test intent to the fake service.
func (f *FakeQueryService) AddIntent(v IntentView) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if v.CreatedAt.IsZero() {
		v.CreatedAt = time.Now()
	}
	if v.UpdatedAt.IsZero() {
		v.UpdatedAt = time.Now()
	}
	f.Intents[v.ID] = v
}

// AddTask adds a test task to the fake service.
func (f *FakeQueryService) AddTask(v TaskView) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if v.CreatedAt.IsZero() {
		v.CreatedAt = time.Now()
	}
	if v.UpdatedAt.IsZero() {
		v.UpdatedAt = time.Now()
	}
	f.Tasks[v.ID] = v
}
