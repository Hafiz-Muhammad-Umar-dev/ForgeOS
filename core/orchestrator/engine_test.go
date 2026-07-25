package orchestrator

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/agent"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/bus"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/event"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/ingress"
)

// ---------------------------------------------------------------------------
// Test bus stub
// ---------------------------------------------------------------------------

type testBus struct {
	mu          sync.Mutex
	subHandlers []struct {
		subject string
		handler bus.MessageHandler
	}
	published []struct {
		subject string
		data    []byte
	}
	connected bool
}

func (b *testBus) Connect(_ context.Context) error { return nil }
func (b *testBus) IsConnected() bool               { return b.connected }
func (b *testBus) Close(_ context.Context) error   { return nil }
func (b *testBus) Publish(_ context.Context, subject string, data []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.published = append(b.published, struct {
		subject string
		data    []byte
	}{subject, data})
	return nil
}
func (b *testBus) Subscribe(_ context.Context, subject string, handler bus.MessageHandler) (bus.Subscription, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subHandlers = append(b.subHandlers, struct {
		subject string
		handler bus.MessageHandler
	}{subject, handler})
	return &testSub{parent: b, subject: subject}, nil
}

type testSub struct {
	parent  *testBus
	subject string
	mu      sync.Mutex
	active  bool
}

func (s *testSub) Unsubscribe() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.active = false
	return nil
}
func (s *testSub) Subject() string { return s.subject }

// deliver simulates receiving a bus message through the subscription handler.
// It copies the handlers under lock, then invokes without the lock to
// prevent deadlock when the handler calls Publish (which also acquires the lock).
func (b *testBus) deliver(ctx context.Context, subject string, data []byte) {
	b.mu.Lock()
	var handlers []bus.MessageHandler
	for _, sh := range b.subHandlers {
		if sh.subject == subject {
			handlers = append(handlers, sh.handler)
		}
	}
	b.mu.Unlock()
	for _, h := range handlers {
		h(ctx, &testMsg{data: data, subject: subject})
	}
}

type testMsg struct {
	data    []byte
	subject string
}

func (m *testMsg) Subject() string { return m.subject }
func (m *testMsg) Data() []byte    { return m.data }
func (m *testMsg) Ack() error      { return nil }
func (m *testMsg) Nak() error      { return nil }
func (m *testMsg) Term() error     { return nil }

// makeIntentEvent creates a serialized intent.created event for testing.
func makeIntentEvent(t *testing.T, text, userID string) []byte {
	t.Helper()
	payload := ingress.IntentPayload{Text: text, UserID: userID}
	env := event.New(event.TypeIntentCreated, "test", payload)
	data, err := event.Serialize(env)
	if err != nil {
		t.Fatalf("serialize: %v", err)
	}
	return data
}

// ---------------------------------------------------------------------------
// Engine lifecycle tests
// ---------------------------------------------------------------------------

func TestEngineName(t *testing.T) {
	e := NewEngine(&testBus{}, NewFakeRunner())
	if e.Name() != "orchestrator" {
		t.Errorf("name=%s", e.Name())
	}
}

func TestEngineInitSuccess(t *testing.T) {
	e := NewEngine(&testBus{}, NewFakeRunner())
	if err := e.Init(context.Background()); err != nil {
		t.Fatalf("init: %v", err)
	}
}

func TestEngineInitNoBus(t *testing.T) {
	e := NewEngine(nil, NewFakeRunner())
	if err := e.Init(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}

func TestEngineInitNoRunner(t *testing.T) {
	e := NewEngine(&testBus{}, nil)
	if err := e.Init(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}

func TestEngineStartStop(t *testing.T) {
	tb := &testBus{connected: true}
	e := NewEngine(tb, NewFakeRunner())
	ctx := context.Background()

	if err := e.Init(ctx); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := e.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}

	h := e.Health()
	if h.Status != "UP" {
		t.Errorf("health after start: got=%s", h.Status)
	}

	if err := e.Stop(ctx); err != nil {
		t.Fatalf("stop: %v", err)
	}

	h = e.Health()
	if h.Status == "UP" {
		t.Errorf("health after stop: got=%s", h.Status)
	}
}

// ---------------------------------------------------------------------------
// Dispatch tests
// ---------------------------------------------------------------------------

func TestEngineDispatchSuccess(t *testing.T) {
	tb := &testBus{connected: true}
	fr := NewFakeRunner()
	e := NewEngine(tb, fr)
	ctx := context.Background()

	if err := e.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer e.Stop(ctx)

	data := makeIntentEvent(t, "build an app", "user-42")
	tb.deliver(ctx, "devos.intent.created", data)

	if fr.RunCount.Load() != 1 {
		t.Fatalf("run count: got=%d", fr.RunCount.Load())
	}
	if fr.Received[0].Task.Description != "build an app" {
		t.Errorf("description=%s", fr.Received[0].Task.Description)
	}

	// Should have published intent.completed
	tb.mu.Lock()
	defer tb.mu.Unlock()
	found := false
	for _, p := range tb.published {
		if p.subject == "devos.intent.completed" {
			found = true
			break
		}
	}
	if !found {
		t.Error("intent.completed not published")
	}
}

func TestEngineDispatchFailure(t *testing.T) {
	tb := &testBus{connected: true}
	fr := NewFakeRunner()
	fr.RunError = ErrDispatchFailed
	e := NewEngine(tb, fr)
	ctx := context.Background()

	if err := e.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer e.Stop(ctx)

	data := makeIntentEvent(t, "do something", "user-1")
	tb.deliver(ctx, "devos.intent.created", data)

	if fr.RunCount.Load() != 1 {
		t.Fatalf("run count: got=%d", fr.RunCount.Load())
	}

	// Should have published intent.failed
	tb.mu.Lock()
	defer tb.mu.Unlock()
	found := false
	for _, p := range tb.published {
		if p.subject == "devos.intent.failed" {
			found = true
			break
		}
	}
	if !found {
		t.Error("intent.failed not published")
	}
}

func TestEngineDispatchMalformedEvent(t *testing.T) {
	tb := &testBus{connected: true}
	fr := NewFakeRunner()
	e := NewEngine(tb, fr)
	ctx := context.Background()

	if err := e.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer e.Stop(ctx)

	// Send garbage
	tb.deliver(ctx, "devos.intent.created", []byte(`{invalid}`))

	if fr.RunCount.Load() != 0 {
		t.Errorf("run count: got=%d", fr.RunCount.Load())
	}
}

func TestEngineDispatchWrongEventType(t *testing.T) {
	tb := &testBus{connected: true}
	fr := NewFakeRunner()
	e := NewEngine(tb, fr)
	ctx := context.Background()

	if err := e.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer e.Stop(ctx)

	// Send a task.status event instead of intent.created (wrong type)
	payload := map[string]string{"status": "running"}
	env := event.New(event.TypeTaskStatus, "test", payload)
	data, _ := event.Serialize(env)

	tb.deliver(ctx, "devos.intent.created", data)

	if fr.RunCount.Load() != 0 {
		t.Errorf("run count: got=%d", fr.RunCount.Load())
	}
}

func TestEngineDispatchContextCancelled(t *testing.T) {
	tb := &testBus{connected: true}
	fr := NewFakeRunner()
	// Make RunTask block until context is done
	fr.RunFunc = func(ctx context.Context, _ string, _ agent.Task) (*agent.Result, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}

	e := NewEngine(tb, fr)
	ctx, cancel := context.WithCancel(context.Background())

	if err := e.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer e.Stop(ctx)

	data := makeIntentEvent(t, "test cancellation", "user-1")

	// Deliver in a goroutine since it will block
	done := make(chan struct{})
	go func() {
		tb.deliver(ctx, "devos.intent.created", data)
		close(done)
	}()

	// Wait a moment for delivery to start
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// Handler exited
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not exit after cancellation")
	}
}

func TestEngineDispatchWithPayload(t *testing.T) {
	tb := &testBus{connected: true}
	fr := NewFakeRunner()
	e := NewEngine(tb, fr)
	ctx := context.Background()

	if err := e.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer e.Stop(ctx)

	// Create an event with full metadata
	payload := ingress.IntentPayload{
		Text:    "full test",
		UserID:  "user-1",
		TraceID: "trace-custom",
	}
	payload.ProjectID = "proj-1"
	payload.OrgID = "org-1"

	env := event.New(event.TypeIntentCreated, "test", payload,
		event.WithTraceID("trace-custom"),
		event.WithOrgID("org-1"),
		event.WithProjectID("proj-1"),
	)
	data, _ := event.Serialize(env)

	tb.deliver(ctx, "devos.intent.created", data)

	if fr.RunCount.Load() != 1 {
		t.Fatalf("run count: got=%d", fr.RunCount.Load())
	}

	// Verify task payload includes metadata
	task := fr.Received[0].Task
	if task.Payload["trace_id"] != "trace-custom" {
		t.Errorf("trace_id=%v", task.Payload["trace_id"])
	}
	if task.Payload["org_id"] != "org-1" {
		t.Errorf("org_id=%v", task.Payload["org_id"])
	}
}

// ---------------------------------------------------------------------------
// Concurrency tests
// ---------------------------------------------------------------------------

func TestEngineConcurrentPublish(t *testing.T) {
	tb := &testBus{connected: true}
	fr := NewFakeRunner()
	e := NewEngine(tb, fr)
	ctx := context.Background()

	if err := e.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer e.Stop(ctx)

	const n = 10
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			data := makeIntentEvent(t, "concurrent", "user")
			tb.deliver(ctx, "devos.intent.created", data)
		}(i)
	}
	wg.Wait()

	if fr.RunCount.Load() != int64(n) {
		t.Errorf("runs: got=%d want=%d", fr.RunCount.Load(), n)
	}
}

// ---------------------------------------------------------------------------
// Published event content tests
// ---------------------------------------------------------------------------

func TestEnginePublishedEventContent(t *testing.T) {
	tb := &testBus{connected: true}
	fr := NewFakeRunner()
	e := NewEngine(tb, fr)
	ctx := context.Background()

	if err := e.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer e.Stop(ctx)

	payload := ingress.IntentPayload{Text: "test content", UserID: "user-99"}
	env := event.New(event.TypeIntentCreated, "test", payload,
		event.WithTraceID("trace-99"),
		event.WithOrgID("org-99"),
	)
	data, _ := event.Serialize(env)

	tb.deliver(ctx, "devos.intent.created", data)

	tb.mu.Lock()
	defer tb.mu.Unlock()

	var completedData []byte
	for _, p := range tb.published {
		if p.subject == "devos.intent.completed" {
			completedData = p.data
			break
		}
	}
	if completedData == nil {
		t.Fatal("no intent.completed event published")
	}

	// Verify the published event envelope
	var raw map[string]any
	if err := json.Unmarshal(completedData, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if raw["type"] != "intent.completed" {
		t.Errorf("type=%v", raw["type"])
	}
	if raw["orgId"] != "org-99" {
		t.Errorf("orgId=%v", raw["orgId"])
	}
	if raw["traceId"] != "trace-99" {
		t.Errorf("traceId=%v", raw["traceId"])
	}
}

func TestEnginePublishedFailedEvent(t *testing.T) {
	tb := &testBus{connected: true}
	fr := NewFakeRunner()
	fr.RunError = ErrDispatchFailed
	e := NewEngine(tb, fr)
	ctx := context.Background()

	if err := e.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer e.Stop(ctx)

	data := makeIntentEvent(t, "will fail", "user-1")
	tb.deliver(ctx, "devos.intent.created", data)

	tb.mu.Lock()
	defer tb.mu.Unlock()

	var failedData []byte
	for _, p := range tb.published {
		if p.subject == "devos.intent.failed" {
			failedData = p.data
			break
		}
	}
	if failedData == nil {
		t.Fatal("no intent.failed event published")
	}

	var raw map[string]any
	if err := json.Unmarshal(failedData, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if raw["type"] != "intent.failed" {
		t.Errorf("type=%v", raw["type"])
	}
}

// ---------------------------------------------------------------------------
// Shutdown tests
// ---------------------------------------------------------------------------

func TestEngineStopDuringDispatch(t *testing.T) {
	tb := &testBus{connected: true}
	fr := NewFakeRunner()
	// Block RunTask so we can test stop during dispatch
	fr.RunFunc = func(ctx context.Context, _ string, _ agent.Task) (*agent.Result, error) {
		time.Sleep(200 * time.Millisecond)
		return &agent.Result{Summary: "ok", Status: agent.ResultSuccess}, nil
	}

	e := NewEngine(tb, fr)
	ctx := context.Background()

	if err := e.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}

	data := makeIntentEvent(t, "slow task", "user-1")
	go tb.deliver(ctx, "devos.intent.created", data)

	// Wait a bit for delivery to start
	time.Sleep(50 * time.Millisecond)

	// Stop should unsubscribe and prevent new dispatches
	if err := e.Stop(ctx); err != nil {
		t.Fatalf("stop: %v", err)
	}

	// RunTask should have been called exactly once
	if fr.RunCount.Load() != 1 {
		t.Errorf("runs: got=%d want=1", fr.RunCount.Load())
	}
}

func TestEngineDoubleStart(t *testing.T) {
	tb := &testBus{connected: true}
	e := NewEngine(tb, NewFakeRunner())
	ctx := context.Background()

	if err := e.Start(ctx); err != nil {
		t.Fatalf("first start: %v", err)
	}
	if err := e.Stop(ctx); err != nil {
		t.Fatalf("stop: %v", err)
	}
	// Second stop should be fine
	if err := e.Stop(ctx); err != nil {
		t.Fatalf("second stop: %v", err)
	}
}
