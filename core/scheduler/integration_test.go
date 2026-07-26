//go:build integration

package scheduler

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

// TestIntegrationScheduleAndExecute verifies the full schedule→execute flow.
func TestIntegrationScheduleAndExecute(t *testing.T) {
	var executed atomic.Int32
	workFn := func(_ context.Context, task Task) error {
		executed.Add(1)
		if task.ID == "" {
			t.Error("task ID is empty")
		}
		return nil
	}

	s := NewService(DefaultConfig(), workFn)
	ctx := context.Background()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer s.Stop(ctx)

	const n = 5
	for i := 0; i < n; i++ {
		_, err := s.RequestSchedule(ctx, ScheduleRequest{
			OrgID:    "org-int",
			AgentID:  "coder",
			IntentID: "intent-1",
		})
		if err != nil {
			t.Fatalf("schedule %d: %v", i, err)
		}
	}

	time.Sleep(1 * time.Second)

	got := executed.Load()
	if got != int32(n) {
		t.Errorf("executed: got=%d want=%d", got, n)
	}
}

// TestIntegrationSequentialExecution verifies tasks execute in order.
func TestIntegrationSequentialExecution(t *testing.T) {
	var results []int
	workFn := func(_ context.Context, task Task) error {
		val := task.Payload.(int)
		results = append(results, val)
		return nil
	}

	s := NewService(DefaultConfig(), workFn)
	ctx := context.Background()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer s.Stop(ctx)

	for i := 0; i < 5; i++ {
		s.RequestSchedule(ctx, ScheduleRequest{
			IntentID: "seq-test",
			Payload:  i,
		})
	}

	time.Sleep(500 * time.Millisecond)

	if len(results) != 5 {
		t.Fatalf("results: got=%d want=5", len(results))
	}
	for i, v := range results {
		if v != i {
			t.Errorf("order[%d]: got=%d want=%d", i, v, i)
		}
	}
}

// TestIntegrationCancelDuringExecution verifies cancellation works.
func TestIntegrationCancelDuringExecution(t *testing.T) {
	s := NewService(DefaultConfig(), func(ctx context.Context, task Task) error {
		<-ctx.Done()
		return ctx.Err()
	})
	ctx := context.Background()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer s.Stop(ctx)

	task, _ := s.RequestSchedule(ctx, ScheduleRequest{
		IntentID: "cancel-test",
	})

	time.Sleep(50 * time.Millisecond)

	if err := s.Cancel(ctx, task.ID); err != nil {
		t.Fatalf("cancel: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	stored, _ := s.Status(ctx, task.ID)
	if stored.State != TaskStateFailed {
		t.Errorf("state: got=%s want=failed", stored.State)
	}
}
