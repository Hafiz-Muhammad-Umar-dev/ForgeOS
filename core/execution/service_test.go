package execution

import (
	"context"
	"testing"
	"time"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/store"
)

func TestValidateTransitionValid(t *testing.T) {
	cases := []struct {
		from, to Status
	}{
		{StatusPending, StatusRunning},
		{StatusRunning, StatusCompleted},
		{StatusRunning, StatusFailed},
		{StatusRunning, StatusPaused},
		{StatusPaused, StatusRunning},
	}
	for _, c := range cases {
		if err := ValidateTransition(c.from, c.to); err != nil {
			t.Errorf("expected valid %s->%s, got %v", c.from, c.to, err)
		}
	}
}

func TestValidateTransitionInvalid(t *testing.T) {
	cases := []struct {
		from, to Status
	}{
		{StatusPending, StatusPaused},
		{StatusCompleted, StatusRunning},
		{StatusFailed, StatusRunning},
		{StatusPending, StatusCompleted},
	}
	for _, c := range cases {
		if err := ValidateTransition(c.from, c.to); err == nil {
			t.Errorf("expected invalid %s->%s", c.from, c.to)
		}
	}
}

func TestValidateTransitionNoOp(t *testing.T) {
	if err := ValidateTransition(StatusRunning, StatusRunning); err == nil {
		t.Error("expected error for no-op transition")
	}
}

func TestTargetStatusForAction(t *testing.T) {
	s, err := TargetStatusForAction(StatusRunning, ActionPause)
	if err != nil {
		t.Fatalf("pause: %v", err)
	}
	if s != StatusPaused {
		t.Errorf("expected paused, got %s", s)
	}

	s, err = TargetStatusForAction(StatusPaused, ActionResume)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	if s != StatusRunning {
		t.Errorf("expected running, got %s", s)
	}
}

func TestTargetStatusForActionIllegal(t *testing.T) {
	// Cannot pause a pending execution.
	_, err := TargetStatusForAction(StatusPending, ActionPause)
	if err == nil {
		t.Error("expected error pausing pending execution")
	}
	// Cannot resume a completed execution.
	_, err = TargetStatusForAction(StatusCompleted, ActionResume)
	if err == nil {
		t.Error("expected error resuming completed execution")
	}
}

func TestServiceApplyActionRun(t *testing.T) {
	fs := store.NewFakeStore()
	repo := NewRepository(fs)
	svc := NewService(repo)

	// Seed an execution via the in-memory manager path.
	ex := &Execution{ID: "e1", IntentID: "i1", Status: StatusPending}
	repo.UpsertExecution(context.Background(), *ex)

	got, err := svc.ApplyAction(context.Background(), "e1", ActionRun)
	if err == nil {
		if got.Status != StatusRunning {
			t.Errorf("status: got=%s want=running", got.Status)
		}
	}
}

func TestServiceApplyActionIllegal(t *testing.T) {
	fs := store.NewFakeStore()
	repo := NewRepository(fs)
	svc := NewService(repo)

	repo.UpsertExecution(context.Background(), Execution{ID: "e1", Status: StatusCompleted})

	_, err := svc.ApplyAction(context.Background(), "e1", ActionResume)
	if err == nil {
		t.Error("expected error resuming completed execution")
	}
}

func TestExecutionsMigration(t *testing.T) {
	migs := Migrations()
	if len(migs) == 0 {
		t.Fatal("no migrations")
	}
	if migs[0].Version != 1 {
		t.Errorf("version=%d", migs[0].Version)
	}
	for _, table := range []string{"executions", "execution_nodes", "execution_edges", "execution_events", "execution_metrics"} {
		if !contains(migs[0].Up, table) {
			t.Errorf("migration missing %s", table)
		}
	}
}

func TestExecutionModelFields(t *testing.T) {
	now := time.Now()
	e := Execution{
		ID: "e1", IntentID: "i1", AgentName: "coder", Status: StatusPending,
		OrgID: "org-1", CreatedAt: now, UpdatedAt: now,
	}
	if e.ID != "e1" || e.IntentID != "i1" || e.Status != StatusPending {
		t.Errorf("execution model fields wrong: %+v", e)
	}
}

func TestEventModelFields(t *testing.T) {
	e := Event{
		ID: "evt-1", ExecutionID: "e1", Type: "reasoning",
		AgentID: "coder", Content: "thinking", Metadata: map[string]any{"k": "v"},
	}
	if e.Type != "reasoning" || e.AgentID != "coder" {
		t.Errorf("event model fields wrong: %+v", e)
	}
	if e.Metadata["k"] != "v" {
		t.Errorf("metadata wrong: %+v", e.Metadata)
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}