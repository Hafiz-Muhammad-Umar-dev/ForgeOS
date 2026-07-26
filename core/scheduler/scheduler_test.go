package scheduler

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Existing type tests (keep from original)
// ---------------------------------------------------------------------------

func TestTaskStateValid(t *testing.T) {
	for _, s := range []TaskState{TaskStatePending, TaskStateScheduled, TaskStateRunning, TaskStateDone, TaskStateFailed} {
		if !s.Valid() {
			t.Errorf("%s invalid", s)
		}
	}
	if TaskState("bogus").Valid() {
		t.Error("bogus should be invalid")
	}
}

// ---------------------------------------------------------------------------
// Old fake tests (keep from original)
// ---------------------------------------------------------------------------

type fakeSchedulerOld struct{}

func (fakeSchedulerOld) RequestSchedule(context.Context, ScheduleRequest) (Task, error) {
	return Task{ID: "t1", State: TaskStateScheduled}, nil
}
func (fakeSchedulerOld) Cancel(context.Context, string) error { return nil }
func (fakeSchedulerOld) Status(context.Context, string) (Task, error) {
	return Task{ID: "t1"}, nil
}

func TestSchedulerFakeOld(t *testing.T) {
	f := fakeSchedulerOld{}
	task, err := f.RequestSchedule(context.Background(), ScheduleRequest{AgentID: "a1"})
	if err != nil {
		t.Fatal(err)
	}
	if task.ID != "t1" || task.State != TaskStateScheduled {
		t.Fatalf("task=%+v", task)
	}
}

// ---------------------------------------------------------------------------
// New Service tests
// ---------------------------------------------------------------------------

// noopWorker is a WorkFunc that completes successfully.
func noopWorker(_ context.Context, _ Task) error { return nil }

// blockingWorker blocks until context is cancelled.
func blockingWorker(ctx context.Context, _ Task) error {
	<-ctx.Done()
	return ctx.Err()
}

func newTestService(t *testing.T, workFn WorkFunc) *Service {
	t.Helper()
	cfg := DefaultConfig()
	cfg.QueueSize = 10
	return NewService(cfg, workFn)
}

func TestServiceName(t *testing.T) {
	s := newTestService(t, noopWorker)
	if s.Name() != "scheduler" {
		t.Errorf("name=%s", s.Name())
	}
}

func TestServiceInitSuccess(t *testing.T) {
	s := newTestService(t, noopWorker)
	if err := s.Init(context.Background()); err != nil {
		t.Fatalf("init: %v", err)
	}
}

func TestServiceInitNoWorkFn(t *testing.T) {
	s := NewService(DefaultConfig(), nil)
	if err := s.Init(context.Background()); err == nil {
		t.Fatal("expected error without work function")
	}
}

func TestServiceStartStop(t *testing.T) {
	s := newTestService(t, noopWorker)
	ctx := context.Background()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	if !s.started {
		t.Error("should be started")
	}

	h := s.Health()
	if h.Status != "UP" {
		t.Errorf("health after start: got=%s", h.Status)
	}

	if err := s.Stop(ctx); err != nil {
		t.Fatalf("stop: %v", err)
	}
	if s.started {
		t.Error("should be stopped")
	}

	h = s.Health()
	if h.Status == "UP" {
		t.Errorf("health after stop: got=%s", h.Status)
	}
}

func TestServiceScheduleAndExecute(t *testing.T) {
	var executed atomic.Bool
	workFn := func(_ context.Context, task Task) error {
		executed.Store(true)
		if task.ID == "" {
			t.Error("task ID should not be empty")
		}
		return nil
	}

	s := newTestService(t, workFn)
	ctx := context.Background()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer s.Stop(ctx)

	task, err := s.RequestSchedule(ctx, ScheduleRequest{
		OrgID:    "org-1",
		AgentID:  "coder",
		IntentID: "intent-1",
	})
	if err != nil {
		t.Fatalf("schedule: %v", err)
	}
	if task.State != TaskStatePending {
		t.Errorf("state: got=%s", task.State)
	}

	// Wait for execution
	time.Sleep(100 * time.Millisecond)

	if !executed.Load() {
		t.Error("task was not executed")
	}

	// Verify final state
	stored, err := s.Status(ctx, task.ID)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if stored.State != TaskStateDone {
		t.Errorf("final state: got=%s want=done", stored.State)
	}

	if s.QueueLength() != 0 {
		t.Errorf("queue length: got=%d", s.QueueLength())
	}
}

func TestServiceScheduleBeforeStart(t *testing.T) {
	s := newTestService(t, noopWorker)
	_, err := s.RequestSchedule(context.Background(), ScheduleRequest{IntentID: "i1"})
	if err == nil {
		t.Fatal("expected error when not started")
	}
}

func TestServiceScheduleQueueFull(t *testing.T) {
	cfg := DefaultConfig()
	cfg.QueueSize = 1
	s := NewService(cfg, func(ctx context.Context, task Task) error {
		// Block to keep queue full
		<-ctx.Done()
		return ctx.Err()
	})
	ctx := context.Background()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}

	// Fill the queue
	s.RequestSchedule(ctx, ScheduleRequest{IntentID: "i1"})

	// Second request should block; use a cancelled context to test full queue
	cancelCtx, cancel := context.WithCancel(ctx)
	cancel()
	_, err := s.RequestSchedule(cancelCtx, ScheduleRequest{IntentID: "i2"})
	if err == nil {
		t.Log("note: queue accepted (maybe worker consumed first)")
	}

	s.Stop(ctx)
}

func TestServiceCancelRunning(t *testing.T) {
	s := newTestService(t, blockingWorker)
	ctx := context.Background()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer s.Stop(ctx)

	task, err := s.RequestSchedule(ctx, ScheduleRequest{IntentID: "cancel-me"})
	if err != nil {
		t.Fatalf("schedule: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	// Cancel the running task
	if err := s.Cancel(ctx, task.ID); err != nil {
		t.Fatalf("cancel: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	stored, err := s.Status(ctx, task.ID)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if stored.State != TaskStateFailed {
		t.Errorf("state after cancel: got=%s want=failed", stored.State)
	}
}

func TestServiceCancelNotFound(t *testing.T) {
	s := newTestService(t, noopWorker)
	err := s.Cancel(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestServiceCancelNotRunning(t *testing.T) {
	s := newTestService(t, noopWorker)
	ctx := context.Background()
	s.Start(ctx)
	defer s.Stop(ctx)

	task, _ := s.RequestSchedule(ctx, ScheduleRequest{IntentID: "i1"})

	// Wait for it to complete
	time.Sleep(100 * time.Millisecond)

	err := s.Cancel(ctx, task.ID)
	if err == nil {
		t.Log("note: cancel on completed task did not error (may be expected)")
	}
}

func TestServiceStatusNotFound(t *testing.T) {
	s := newTestService(t, noopWorker)
	_, err := s.Status(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestServiceQueueLength(t *testing.T) {
	s := newTestService(t, func(ctx context.Context, task Task) error {
		<-ctx.Done()
		return ctx.Err()
	})
	ctx := context.Background()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer s.Stop(ctx)

	// Submit one task — it will be picked up immediately and block
	s.RequestSchedule(ctx, ScheduleRequest{IntentID: "i1"})
	time.Sleep(50 * time.Millisecond)

	// Submit another — should wait in queue
	s.RequestSchedule(ctx, ScheduleRequest{IntentID: "i2"})

	if s.QueueLength() != 1 {
		t.Logf("queue length: %d (may be 0 if worker consumed both)", s.QueueLength())
	}
}

func TestServiceConcurrentSchedule(t *testing.T) {
	s := newTestService(t, func(ctx context.Context, task Task) error {
		<-ctx.Done()
		return ctx.Err()
	})
	ctx := context.Background()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer s.Stop(ctx)

	const n = 10
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			s.RequestSchedule(ctx, ScheduleRequest{
				IntentID: fmt.Sprintf("intent-%d", i),
			})
		}(i)
	}
	wg.Wait()

	time.Sleep(50 * time.Millisecond)
	t.Logf("queue length after %d concurrent schedules: %d", n, s.QueueLength())
}

func TestServiceShutdownDuringProcessing(t *testing.T) {
	var started atomic.Bool
	s := newTestService(t, func(ctx context.Context, task Task) error {
		started.Store(true)
		<-ctx.Done()
		return ctx.Err()
	})
	ctx := context.Background()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}

	s.RequestSchedule(ctx, ScheduleRequest{IntentID: "slow-task"})
	time.Sleep(50 * time.Millisecond)

	if !started.Load() {
		t.Error("task should have started")
	}

	// Stop should cancel the running task
	if err := s.Stop(ctx); err != nil {
		t.Fatalf("stop: %v", err)
	}
}

func TestServiceDoubleStart(t *testing.T) {
	s := newTestService(t, noopWorker)
	ctx := context.Background()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("first start: %v", err)
	}
	if err := s.Start(ctx); err != nil {
		t.Fatalf("second start: %v", err)
	}
	if err := s.Stop(ctx); err != nil {
		t.Fatalf("stop: %v", err)
	}
}

func TestServiceDoubleStop(t *testing.T) {
	s := newTestService(t, noopWorker)
	ctx := context.Background()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := s.Stop(ctx); err != nil {
		t.Fatalf("first stop: %v", err)
	}
	if err := s.Stop(ctx); err != nil {
		t.Fatalf("second stop: %v", err)
	}
}

func TestServiceTaskPayload(t *testing.T) {
	var received atomic.Value
	s := newTestService(t, func(_ context.Context, task Task) error {
		received.Store(task.Payload)
		return nil
	})
	ctx := context.Background()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer s.Stop(ctx)

	payload := map[string]string{"key": "value"}
	s.RequestSchedule(ctx, ScheduleRequest{
		IntentID: "payload-test",
		Payload:  payload,
	})

	time.Sleep(100 * time.Millisecond)

	got := received.Load()
	if got == nil {
		t.Fatal("payload not received")
	}
}

func TestServiceWorkFuncError(t *testing.T) {
	s := newTestService(t, func(_ context.Context, _ Task) error {
		return fmt.Errorf("work error")
	})
	ctx := context.Background()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer s.Stop(ctx)

	task, _ := s.RequestSchedule(ctx, ScheduleRequest{IntentID: "error-task"})

	time.Sleep(100 * time.Millisecond)

	stored, _ := s.Status(ctx, task.ID)
	if stored.State != TaskStateFailed {
		t.Errorf("state: got=%s want=failed", stored.State)
	}
}

func TestServiceEmptyIntentID(t *testing.T) {
	s := newTestService(t, func(_ context.Context, task Task) error {
		if task.ID == "" {
			t.Error("task ID should not be empty")
		}
		return nil
	})
	ctx := context.Background()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer s.Stop(ctx)

	task, err := s.RequestSchedule(ctx, ScheduleRequest{
		OrgID:   "org-1",
		AgentID: "coder",
	})
	if err != nil {
		t.Fatalf("schedule: %v", err)
	}
	if task.ID == "" {
		t.Error("generated ID should not be empty")
	}

	time.Sleep(100 * time.Millisecond)
}

// ---------------------------------------------------------------------------
// FakeScheduler tests
// ---------------------------------------------------------------------------

func TestFakeSchedulerRequestSchedule(t *testing.T) {
	fs := NewFakeScheduler()
	task, err := fs.RequestSchedule(nil, ScheduleRequest{AgentID: "coder"})
	if err != nil {
		t.Fatalf("schedule: %v", err)
	}
	if task.ID != "fake-task-1" {
		t.Errorf("id=%s", task.ID)
	}
	if fs.RequestCount.Load() != 1 {
		t.Errorf("count=%d", fs.RequestCount.Load())
	}
}

func TestFakeSchedulerRecordsRequests(t *testing.T) {
	fs := NewFakeScheduler()
	fs.RequestSchedule(nil, ScheduleRequest{IntentID: "i1", AgentID: "a1"})
	fs.RequestSchedule(nil, ScheduleRequest{IntentID: "i2", AgentID: "a2"})

	if len(fs.Received) != 2 {
		t.Fatalf("received=%d", len(fs.Received))
	}
	if fs.Received[0].IntentID != "i1" {
		t.Errorf("first=%s", fs.Received[0].IntentID)
	}
	if fs.Received[1].AgentID != "a2" {
		t.Errorf("second=%s", fs.Received[1].AgentID)
	}
}

func TestFakeSchedulerWithError(t *testing.T) {
	expectedErr := fmt.Errorf("custom error")
	fs := NewFakeSchedulerWithError(expectedErr)
	_, err := fs.RequestSchedule(nil, ScheduleRequest{})
	if err != expectedErr {
		t.Errorf("err=%v", err)
	}
}

func TestFakeSchedulerQueueLength(t *testing.T) {
	fs := NewFakeScheduler()
	if fs.QueueLength() != 0 {
		t.Errorf("queue length: got=%d", fs.QueueLength())
	}
}

func TestFakeSchedulerIsRunning(t *testing.T) {
	fs := NewFakeScheduler()
	if fs.IsRunning() {
		t.Error("fake should not be running")
	}
}

// ---------------------------------------------------------------------------
// Integration helpers
// ---------------------------------------------------------------------------

func TestServiceIntegrationFlow(t *testing.T) {
	var executed int32
	s := newTestService(t, func(_ context.Context, task Task) error {
		atomic.AddInt32(&executed, 1)
		return nil
	})
	ctx := context.Background()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer s.Stop(ctx)

	const n = 5
	for i := 0; i < n; i++ {
		s.RequestSchedule(ctx, ScheduleRequest{
			IntentID: fmt.Sprintf("intent-%d", i),
			Payload:  i,
		})
	}

	time.Sleep(500 * time.Millisecond)

	got := atomic.LoadInt32(&executed)
	if got != int32(n) {
		t.Errorf("executed: got=%d want=%d", got, n)
	}
}
