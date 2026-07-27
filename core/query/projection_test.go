package query

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/agent"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/bus"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/event"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/orchestrator"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/store"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func mustMarshalPayload(t *testing.T, v any) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return data
}

// mockBus implements bus.BusPort for testing ProjectionEngine.
type mockBus struct {
	subscribers []string
}

func (b *mockBus) Connect(ctx context.Context) error                     { return nil }
func (b *mockBus) IsConnected() bool                                     { return true }
func (b *mockBus) Close(ctx context.Context) error                       { return nil }
func (b *mockBus) Publish(ctx context.Context, s string, d []byte) error { return nil }
func (b *mockBus) Subscribe(_ context.Context, subject string, _ bus.MessageHandler) (bus.Subscription, error) {
	b.subscribers = append(b.subscribers, subject)
	return &mockSub{}, nil
}

type mockSub struct{}

func (s *mockSub) Unsubscribe() error { return nil }
func (s *mockSub) Subject() string    { return "" }

// ---------------------------------------------------------------------------
// Projection handler tests using FakeStore
// ---------------------------------------------------------------------------

func TestIntentProjectionName(t *testing.T) {
	p := NewIntentProjection(store.NewFakeStore())
	if p.Name() != "intent_projection" {
		t.Errorf("name=%s", p.Name())
	}
}

func TestIntentProjectionSubjects(t *testing.T) {
	p := NewIntentProjection(store.NewFakeStore())
	subjects := p.Subjects()
	if len(subjects) != 2 {
		t.Fatalf("subjects=%d", len(subjects))
	}
}

func TestIntentProjectionHandleCompleted(t *testing.T) {
	fs := store.NewFakeStore()
	p := NewIntentProjection(fs)

	env := event.RawEnvelope{
		ID:         "evt-1",
		Type:       event.TypeIntentCompleted,
		OrgID:      "org-1",
		ProjectID:  "proj-1",
		TraceID:    "trace-1",
		ProducedAt: time.Now().UnixNano(),
		Payload:    mustMarshalPayload(t, orchestrator.IntentLifecyclePayload{IntentID: "intent-1", Summary: "done"}),
	}

	if err := p.Handle(context.Background(), env); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if fs.ExecCount.Load() != 1 {
		t.Errorf("exec count: got=%d", fs.ExecCount.Load())
	}
}

func TestIntentProjectionHandleFailed(t *testing.T) {
	fs := store.NewFakeStore()
	p := NewIntentProjection(fs)

	env := event.RawEnvelope{
		ID:         "evt-2",
		Type:       event.TypeIntentFailed,
		ProducedAt: time.Now().UnixNano(),
		Payload:    mustMarshalPayload(t, orchestrator.IntentLifecyclePayload{IntentID: "intent-1", Error: "fail"}),
	}

	if err := p.Handle(context.Background(), env); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if fs.ExecCount.Load() != 1 {
		t.Errorf("exec count: got=%d", fs.ExecCount.Load())
	}
}

func TestIntentProjectionSkipsWrongType(t *testing.T) {
	fs := store.NewFakeStore()
	p := NewIntentProjection(fs)

	env := event.RawEnvelope{ID: "evt-3", Type: event.TypeTaskStatus}
	if err := p.Handle(context.Background(), env); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if fs.ExecCount.Load() != 0 {
		t.Errorf("should not exec for wrong type")
	}
}

func TestIntentProjectionUnmarshalError(t *testing.T) {
	fs := store.NewFakeStore()
	p := NewIntentProjection(fs)

	env := event.RawEnvelope{ID: "evt-4", Type: event.TypeIntentCompleted, Payload: []byte(`{invalid}`)}
	if err := p.Handle(context.Background(), env); err == nil {
		t.Fatal("expected unmarshal error")
	}
}

func TestTaskProjectionName(t *testing.T) {
	p := NewTaskProjection(store.NewFakeStore())
	if p.Name() != "task_projection" {
		t.Errorf("name=%s", p.Name())
	}
}

func TestTaskProjectionSubjects(t *testing.T) {
	p := NewTaskProjection(store.NewFakeStore())
	subjects := p.Subjects()
	if len(subjects) != 2 {
		t.Fatalf("subjects=%d", len(subjects))
	}
}

func TestTaskProjectionHandleStatus(t *testing.T) {
	fs := store.NewFakeStore()
	p := NewTaskProjection(fs)

	env := event.RawEnvelope{
		ID:         "evt-5",
		Type:       event.TypeTaskStatus,
		ProducedAt: time.Now().UnixNano(),
		Payload:    mustMarshalPayload(t, agent.TaskStatusPayload{TaskID: "task-1", Status: "completed", AgentName: "coder", Summary: "done"}),
	}

	if err := p.Handle(context.Background(), env); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if fs.ExecCount.Load() != 1 {
		t.Errorf("exec count: got=%d", fs.ExecCount.Load())
	}
}

func TestTaskProjectionHandleFailed(t *testing.T) {
	fs := store.NewFakeStore()
	p := NewTaskProjection(fs)

	env := event.RawEnvelope{
		ID:         "evt-6",
		Type:       event.TypeTaskFailed,
		ProducedAt: time.Now().UnixNano(),
		Payload:    mustMarshalPayload(t, agent.TaskStatusPayload{TaskID: "task-1", Error: "timeout"}),
	}

	if err := p.Handle(context.Background(), env); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if fs.ExecCount.Load() != 1 {
		t.Errorf("exec count: got=%d", fs.ExecCount.Load())
	}
}

func TestTaskProjectionSkipsWrongType(t *testing.T) {
	fs := store.NewFakeStore()
	p := NewTaskProjection(fs)

	env := event.RawEnvelope{ID: "evt-7", Type: event.TypeIntentCompleted}
	if err := p.Handle(context.Background(), env); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if fs.ExecCount.Load() != 0 {
		t.Errorf("should not exec for wrong type")
	}
}

// ---------------------------------------------------------------------------
// ProjectionEngine lifecycle tests
// ---------------------------------------------------------------------------

func TestProjectionEngineName(t *testing.T) {
	pe := NewProjectionEngine(&mockBus{}, []Projection{NewIntentProjection(store.NewFakeStore())})
	if pe.Name() != "projections" {
		t.Errorf("name=%s", pe.Name())
	}
}

func TestProjectionEngineInitSuccess(t *testing.T) {
	pe := NewProjectionEngine(&mockBus{}, []Projection{NewIntentProjection(store.NewFakeStore())})
	if err := pe.Init(context.Background()); err != nil {
		t.Fatalf("init: %v", err)
	}
}

func TestProjectionEngineInitNoBus(t *testing.T) {
	pe := NewProjectionEngine(nil, []Projection{NewIntentProjection(store.NewFakeStore())})
	if err := pe.Init(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}

func TestProjectionEngineInitNoProjections(t *testing.T) {
	pe := NewProjectionEngine(&mockBus{}, nil)
	if err := pe.Init(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}

func TestProjectionEngineStartStop(t *testing.T) {
	mb := &mockBus{}
	pe := NewProjectionEngine(mb, []Projection{NewIntentProjection(store.NewFakeStore())})
	ctx := context.Background()

	if err := pe.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	if len(mb.subscribers) == 0 {
		t.Error("expected subscribers")
	}

	h := pe.Health()
	if h.Status != "UP" {
		t.Errorf("health after start: got=%s", h.Status)
	}

	if err := pe.Stop(ctx); err != nil {
		t.Fatalf("stop: %v", err)
	}

	h = pe.Health()
	if h.Status == "UP" {
		t.Errorf("health after stop: got=%s", h.Status)
	}
}
