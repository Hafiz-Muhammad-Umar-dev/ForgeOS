package orchestrator

import (
	"context"
	"testing"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/agent"
)

func TestSentinelErrors(t *testing.T) {
	errs := []struct {
		err   error
		label string
	}{
		{ErrInvalidEvent, "ErrInvalidEvent"},
		{ErrDispatchFailed, "ErrDispatchFailed"},
		{ErrNotStarted, "ErrNotStarted"},
	}
	for _, tt := range errs {
		t.Run(tt.label, func(t *testing.T) {
			if tt.err == nil {
				t.Fatal("error is nil")
			}
		})
	}
}

func TestIntentLifecyclePayload(t *testing.T) {
	p := IntentLifecyclePayload{
		IntentID:  "intent-1",
		UserID:    "user-1",
		OrgID:     "org-1",
		ProjectID: "proj-1",
		TraceID:   "trace-1",
		Summary:   "done",
	}
	if p.IntentID != "intent-1" {
		t.Errorf("intentId=%s", p.IntentID)
	}
	if p.UserID != "user-1" {
		t.Errorf("userId=%s", p.UserID)
	}
}

func TestIntentLifecyclePayloadError(t *testing.T) {
	p := IntentLifecyclePayload{
		IntentID: "intent-1",
		Error:    "something failed",
	}
	if p.Error != "something failed" {
		t.Errorf("error=%s", p.Error)
	}
}

// ---------------------------------------------------------------------------
// FakeRunner tests
// ---------------------------------------------------------------------------

func TestFakeRunnerDefaults(t *testing.T) {
	fr := NewFakeRunner()
	if fr.ResultValue.Status != agent.ResultSuccess {
		t.Errorf("status=%s", fr.ResultValue.Status)
	}
}

func TestFakeRunnerRunTask(t *testing.T) {
	fr := NewFakeRunner()

	result, err := fr.RunTask(context.Background(), "coder", agent.Task{ID: "t1", Description: "test"})
	if err != nil {
		t.Fatalf("runTask: %v", err)
	}
	if result.Summary != "completed" {
		t.Errorf("summary=%s", result.Summary)
	}
	if fr.RunCount.Load() != 1 {
		t.Errorf("count=%d", fr.RunCount.Load())
	}
	if len(fr.Received) != 1 {
		t.Fatalf("received=%d", len(fr.Received))
	}
	if fr.Received[0].AgentName != "coder" {
		t.Errorf("agent=%s", fr.Received[0].AgentName)
	}
	if fr.Received[0].Task.ID != "t1" {
		t.Errorf("taskID=%s", fr.Received[0].Task.ID)
	}
}

func TestFakeRunnerRunTaskError(t *testing.T) {
	fr := NewFakeRunner()
	fr.RunError = ErrDispatchFailed

	_, err := fr.RunTask(context.Background(), "coder", agent.Task{ID: "t1"})
	if err != ErrDispatchFailed {
		t.Errorf("err=%v", err)
	}
}

func TestFakeRunnerCustomRunFunc(t *testing.T) {
	fr := NewFakeRunner()
	fr.RunFunc = func(_ context.Context, name string, task agent.Task) (*agent.Result, error) {
		return &agent.Result{Summary: "custom", Status: agent.ResultSuccess}, nil
	}

	result, err := fr.RunTask(context.Background(), "custom", agent.Task{ID: "t1"})
	if err != nil {
		t.Fatalf("runTask: %v", err)
	}
	if result.Summary != "custom" {
		t.Errorf("summary=%s", result.Summary)
	}
}
