package scheduler

import (
	"context"
	"fmt"
	"sync/atomic"
)

// Compile-time check.
var _ Scheduler = (*FakeScheduler)(nil)

// FakeScheduler is an in-memory Scheduler implementation for testing.
// It records all requests and returns configurable results.
type FakeScheduler struct {
	// RequestScheduleFunc overrides RequestSchedule behavior.
	RequestScheduleFunc func(ctx context.Context, req ScheduleRequest) (Task, error)

	// CancelFunc overrides Cancel behavior.
	CancelFunc func(ctx context.Context, taskID string) error

	// StatusFunc overrides Status behavior.
	StatusFunc func(ctx context.Context, taskID string) (Task, error)

	// Result is returned by the default RequestSchedule implementation.
	Result Task

	// RequestCount tracks the number of RequestSchedule calls.
	RequestCount atomic.Int64

	// Received records every request received.
	Received []ScheduleRequest
}

// NewFakeScheduler creates a FakeScheduler with a default success result.
func NewFakeScheduler() *FakeScheduler {
	return &FakeScheduler{
		Result: Task{ID: "fake-task-1", State: TaskStateScheduled},
	}
}

// RequestSchedule records the call and returns the configured result.
func (f *FakeScheduler) RequestSchedule(_ context.Context, req ScheduleRequest) (Task, error) {
	f.RequestCount.Add(1)
	f.Received = append(f.Received, req)

	if f.RequestScheduleFunc != nil {
		return f.RequestScheduleFunc(nil, req)
	}
	return f.Result, nil
}

// Cancel records and delegates.
func (f *FakeScheduler) Cancel(ctx context.Context, taskID string) error {
	if f.CancelFunc != nil {
		return f.CancelFunc(ctx, taskID)
	}
	return nil
}

// Status records and delegates.
func (f *FakeScheduler) Status(ctx context.Context, taskID string) (Task, error) {
	if f.StatusFunc != nil {
		return f.StatusFunc(ctx, taskID)
	}
	return f.Result, nil
}

// QueueLength returns 0 for the fake.
func (f *FakeScheduler) QueueLength() int { return 0 }

// IsRunning returns false for the fake.
func (f *FakeScheduler) IsRunning() bool { return false }

// ---------------------------------------------------------------------------
// test helpers
// ---------------------------------------------------------------------------

// NewFakeSchedulerWithError creates a FakeScheduler that always returns
// the given error for RequestSchedule.
func NewFakeSchedulerWithError(err error) *FakeScheduler {
	return &FakeScheduler{
		RequestScheduleFunc: func(_ context.Context, _ ScheduleRequest) (Task, error) {
			return Task{}, err
		},
		Result: Task{},
	}
}

// NewFakeSchedulerWithResult creates a FakeScheduler that always returns
// the given task for RequestSchedule.
func NewFakeSchedulerWithResult(t Task) *FakeScheduler {
	return &FakeScheduler{
		Result: t,
	}
}

// compile-time check for vararg constructors (no-op but documents intent).
func init() {
	var _ Scheduler = (*FakeScheduler)(nil)
	_ = fmt.Sprintf("") // suppress unused import
}
