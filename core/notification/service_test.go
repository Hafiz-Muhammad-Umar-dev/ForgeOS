package notification

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/bus"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/event"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/orchestrator"
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
	published []struct{ subject string; data []byte }
	connected bool
}

func (b *testBus) Connect(_ context.Context) error { return nil }
func (b *testBus) IsConnected() bool                { return b.connected }
func (b *testBus) Close(_ context.Context) error     { return nil }
func (b *testBus) Publish(_ context.Context, subject string, data []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.published = append(b.published, struct{ subject string; data []byte }{subject, data})
	return nil
}
func (b *testBus) Subscribe(_ context.Context, subject string, handler bus.MessageHandler) (bus.Subscription, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subHandlers = append(b.subHandlers, struct {
		subject string
		handler bus.MessageHandler
	}{subject, handler})
	return &testSub{subject: subject}, nil
}

type testSub struct{ subject string }

func (s *testSub) Unsubscribe() error { return nil }
func (s *testSub) Subject() string    { return s.subject }

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
	subject string
	data    []byte
}

func (m *testMsg) Subject() string { return m.subject }
func (m *testMsg) Data() []byte    { return m.data }
func (m *testMsg) Ack() error      { return nil }
func (m *testMsg) Nak() error      { return nil }
func (m *testMsg) Term() error     { return nil }

// makeLifecycleEvent creates a serialized intent.completed event for testing.
func makeLifecycleEvent(t *testing.T, typ event.EventType, intentID, summary string) []byte {
	t.Helper()
	payload := orchestrator.IntentLifecyclePayload{
		IntentID: intentID,
		Summary:  summary,
		OrgID:    "org-1",
		TraceID:  "trace-1",
	}
	env := event.New(typ, "test", payload)
	data, err := event.Serialize(env)
	if err != nil {
		t.Fatalf("serialize: %v", err)
	}
	return data
}

// ---------------------------------------------------------------------------
// Lifecycle tests
// ---------------------------------------------------------------------------

func TestServiceName(t *testing.T) {
	s := NewService(&testBus{}, NewFakeNotification())
	if s.Name() != "notification" {
		t.Errorf("name=%s", s.Name())
	}
}

func TestServiceInitSuccess(t *testing.T) {
	s := NewService(&testBus{}, NewFakeNotification())
	if err := s.Init(context.Background()); err != nil {
		t.Fatalf("init: %v", err)
	}
}

func TestServiceInitNoBus(t *testing.T) {
	s := NewService(nil, NewFakeNotification())
	if err := s.Init(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}

func TestServiceInitNoNotifier(t *testing.T) {
	s := NewService(&testBus{}, nil)
	if err := s.Init(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}

func TestServiceStartStop(t *testing.T) {
	tb := &testBus{connected: true}
	s := NewService(tb, NewFakeNotification())
	ctx := context.Background()

	if err := s.Init(ctx); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := s.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}

	h := s.Health()
	if h.Status != "UP" {
		t.Errorf("health after start: got=%s", h.Status)
	}

	if err := s.Stop(ctx); err != nil {
		t.Fatalf("stop: %v", err)
	}

	h = s.Health()
	if h.Status == "UP" {
		t.Errorf("health after stop: got=%s", h.Status)
	}
}

// ---------------------------------------------------------------------------
// Dispatch tests
// ---------------------------------------------------------------------------

func TestServiceCompletedNotification(t *testing.T) {
	tb := &testBus{connected: true}
	fn := NewFakeNotification()
	s := NewService(tb, fn)
	ctx := context.Background()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer s.Stop(ctx)

	data := makeLifecycleEvent(t, event.TypeIntentCompleted, "intent-1", "build completed")
	tb.deliver(ctx, "devos.intent.completed", data)

	if fn.SendCount.Load() != 1 {
		t.Fatalf("send count: got=%d", fn.SendCount.Load())
	}
	if fn.Received[0].IntentID != "intent-1" {
		t.Errorf("intentId=%s", fn.Received[0].IntentID)
	}
	if fn.Received[0].Type != "intent.completed" {
		t.Errorf("type=%s", fn.Received[0].Type)
	}
	if fn.Received[0].Summary != "build completed" {
		t.Errorf("summary=%s", fn.Received[0].Summary)
	}
}

func TestServiceFailedNotification(t *testing.T) {
	tb := &testBus{connected: true}
	fn := NewFakeNotification()
	s := NewService(tb, fn)
	ctx := context.Background()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer s.Stop(ctx)

	payload := orchestrator.IntentLifecyclePayload{
		IntentID: "intent-fail",
		Error:    "agent timeout",
		OrgID:    "org-1",
	}
	env := event.New(event.TypeIntentFailed, "test", payload)
	data, _ := event.Serialize(env)

	tb.deliver(ctx, "devos.intent.failed", data)

	if fn.SendCount.Load() != 1 {
		t.Fatalf("send count: got=%d", fn.SendCount.Load())
	}
	if fn.Received[0].IntentID != "intent-fail" {
		t.Errorf("intentId=%s", fn.Received[0].IntentID)
	}
	if fn.Received[0].Type != "intent.failed" {
		t.Errorf("type=%s", fn.Received[0].Type)
	}
	if fn.Received[0].Error != "agent timeout" {
		t.Errorf("error=%s", fn.Received[0].Error)
	}
}

func TestServiceMalformedEvent(t *testing.T) {
	tb := &testBus{connected: true}
	fn := NewFakeNotification()
	s := NewService(tb, fn)
	ctx := context.Background()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer s.Stop(ctx)

	tb.deliver(ctx, "devos.intent.completed", []byte(`{invalid}`))

	if fn.SendCount.Load() != 0 {
		t.Errorf("send count: got=%d", fn.SendCount.Load())
	}
}

func TestServiceWrongEventType(t *testing.T) {
	tb := &testBus{connected: true}
	fn := NewFakeNotification()
	s := NewService(tb, fn)
	ctx := context.Background()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer s.Stop(ctx)

	// Send task.status instead of intent.*
	payload := map[string]string{"status": "running"}
	env := event.New(event.TypeTaskStatus, "test", payload)
	data, _ := event.Serialize(env)

	tb.deliver(ctx, "devos.intent.completed", data)

	if fn.SendCount.Load() != 0 {
		t.Errorf("send count: got=%d", fn.SendCount.Load())
	}
}

func TestServiceContextCancellation(t *testing.T) {
	tb := &testBus{connected: true}
	fn := NewFakeNotification()
	s := NewService(tb, fn)
	ctx, cancel := context.WithCancel(context.Background())

	if err := s.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer s.Stop(ctx)

	// Make the notifier block until context is cancelled
	fn.SendFunc = func(ctx context.Context, _ NotificationPayload) error {
		<-ctx.Done()
		return ctx.Err()
	}

	data := makeLifecycleEvent(t, event.TypeIntentCompleted, "intent-1", "")

	done := make(chan struct{})
	go func() {
		tb.deliver(ctx, "devos.intent.completed", data)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// handler exited
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not exit after cancellation")
	}
}

func TestServiceConcurrentPublish(t *testing.T) {
	tb := &testBus{connected: true}
	fn := NewFakeNotification()
	s := NewService(tb, fn)
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
			data := makeLifecycleEvent(t, event.TypeIntentCompleted, "intent-1", "")
			tb.deliver(ctx, "devos.intent.completed", data)
		}(i)
	}
	wg.Wait()

	if fn.SendCount.Load() != int64(n) {
		t.Errorf("send count: got=%d want=%d", fn.SendCount.Load(), n)
	}
}

func TestServiceMultipleSubscriptions(t *testing.T) {
	tb := &testBus{connected: true}
	fn := NewFakeNotification()
	s := NewService(tb, fn)
	ctx := context.Background()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer s.Stop(ctx)

	// Send a completed event
	compData := makeLifecycleEvent(t, event.TypeIntentCompleted, "intent-c", "ok")
	tb.deliver(ctx, "devos.intent.completed", compData)

	// Send a failed event
	failPayload := orchestrator.IntentLifecyclePayload{IntentID: "intent-f", Error: "fail"}
	failEnv := event.New(event.TypeIntentFailed, "test", failPayload)
	failData, _ := event.Serialize(failEnv)
	tb.deliver(ctx, "devos.intent.failed", failData)

	if fn.SendCount.Load() != 2 {
		t.Errorf("send count: got=%d want=2", fn.SendCount.Load())
	}
}

func TestServiceShutdownDuringDispatch(t *testing.T) {
	tb := &testBus{connected: true}
	fn := NewFakeNotification()
	s := NewService(tb, fn)
	ctx := context.Background()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}

	// Block the notifier so we can test shutdown
	fn.SendFunc = func(ctx context.Context, _ NotificationPayload) error {
		time.Sleep(200 * time.Millisecond)
		return nil
	}

	data := makeLifecycleEvent(t, event.TypeIntentCompleted, "intent-1", "")
	go tb.deliver(ctx, "devos.intent.completed", data)

	time.Sleep(50 * time.Millisecond)

	if err := s.Stop(ctx); err != nil {
		t.Fatalf("stop: %v", err)
	}

	// Should have sent exactly 1
	if fn.SendCount.Load() != 1 {
		t.Errorf("send count: got=%d want=1", fn.SendCount.Load())
	}
}
