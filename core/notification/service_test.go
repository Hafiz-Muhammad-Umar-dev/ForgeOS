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
	subHandlers []struct{ subject string; handler bus.MessageHandler }
	published   []struct{ subject string; data []byte }
	connected   bool
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
	b.subHandlers = append(b.subHandlers, struct{ subject string; handler bus.MessageHandler }{subject, handler})
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

func makeLifecycleEvent(t *testing.T, typ event.EventType, intentID, summary string) []byte {
	t.Helper()
	payload := orchestrator.IntentLifecyclePayload{IntentID: intentID, Summary: summary, OrgID: "org-1", TraceID: "trace-1"}
	env := event.New(typ, "test", payload)
	data, err := event.Serialize(env)
	if err != nil {
		t.Fatalf("serialize: %v", err)
	}
	return data
}

func makeDeployEvent(t *testing.T, summary string) []byte {
	t.Helper()
	payload := orchestrator.IntentLifecyclePayload{IntentID: "deploy-intent", Summary: summary, OrgID: "org-1"}
	env := event.New(event.TypeDeployCompleted, "test", payload)
	data, _ := event.Serialize(env)
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
// Notification dispatch tests
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

	tb.deliver(ctx, "devos.intent.completed", makeLifecycleEvent(t, event.TypeIntentCompleted, "intent-1", "ok"))

	if fn.SendCount.Load() != 1 {
		t.Fatalf("send count: got=%d", fn.SendCount.Load())
	}
	if fn.Received[0].IntentID != "intent-1" {
		t.Errorf("intentId=%s", fn.Received[0].IntentID)
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

	payload := orchestrator.IntentLifecyclePayload{IntentID: "intent-fail", Error: "timeout", OrgID: "org-1"}
	env := event.New(event.TypeIntentFailed, "test", payload)
	data, _ := event.Serialize(env)
	tb.deliver(ctx, "devos.intent.failed", data)

	if fn.SendCount.Load() != 1 {
		t.Fatalf("send count: got=%d", fn.SendCount.Load())
	}
	if fn.Received[0].IntentID != "intent-fail" {
		t.Errorf("intentId=%s", fn.Received[0].IntentID)
	}
}

func TestServiceDeployNotification(t *testing.T) {
	tb := &testBus{connected: true}
	fn := NewFakeNotification()
	s := NewService(tb, fn)
	ctx := context.Background()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer s.Stop(ctx)

	tb.deliver(ctx, "devos.deploy.completed", makeDeployEvent(t, "deployed to prod"))

	if fn.SendCount.Load() != 1 {
		t.Fatalf("send count: got=%d", fn.SendCount.Load())
	}
	if fn.Received[0].Type != "deploy.completed" {
		t.Errorf("type=%s", fn.Received[0].Type)
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

	payload := map[string]string{"status": "running"}
	env := event.New(event.TypeTaskAssigned, "test", payload)
	data, _ := event.Serialize(env)
	tb.deliver(ctx, "devos.intent.completed", data)

	if fn.SendCount.Load() != 0 {
		t.Errorf("send count: got=%d", fn.SendCount.Load())
	}
}

// ---------------------------------------------------------------------------
// Renderer + Channel tests
// ---------------------------------------------------------------------------

func TestServiceWithRendererAndChannel(t *testing.T) {
	tb := &testBus{connected: true}
	fn := NewFakeNotification()
	fc := NewFakeChannelProvider("discord")
	fr := NewFakeRenderer()
	s := NewService(tb, fn, WithRenderer(fr), WithChannel(fc))
	ctx := context.Background()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer s.Stop(ctx)

	tb.deliver(ctx, "devos.intent.completed", makeLifecycleEvent(t, event.TypeIntentCompleted, "intent-1", "rendered"))

	// NotificationPort should be called
	if fn.SendCount.Load() != 1 {
		t.Errorf("notifier count: got=%d", fn.SendCount.Load())
	}

	// Renderer should be called
	if fr.RenderCount.Load() != 1 {
		t.Errorf("renderer count: got=%d", fr.RenderCount.Load())
	}

	// Channel should be called
	if fc.SendCount.Load() != 1 {
		t.Errorf("channel count: got=%d", fc.SendCount.Load())
	}
}

func TestServiceWithChannelOnly(t *testing.T) {
	tb := &testBus{connected: true}
	fn := NewFakeNotification()
	s := NewService(tb, fn)
	ctx := context.Background()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer s.Stop(ctx)

	// Without renderer+channel, only NotificationPort is used
	tb.deliver(ctx, "devos.intent.completed", makeLifecycleEvent(t, event.TypeIntentCompleted, "intent-1", ""))

	if fn.SendCount.Load() != 1 {
		t.Errorf("notifier count: got=%d", fn.SendCount.Load())
	}
}

// ---------------------------------------------------------------------------
// Concurrency tests
// ---------------------------------------------------------------------------

func TestServiceConcurrentEvents(t *testing.T) {
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
		go func() {
			defer wg.Done()
			tb.deliver(ctx, "devos.intent.completed", makeLifecycleEvent(t, event.TypeIntentCompleted, "intent-1", ""))
		}()
	}
	wg.Wait()

	if fn.SendCount.Load() != int64(n) {
		t.Errorf("send count: got=%d want=%d", fn.SendCount.Load(), n)
	}
}

func TestServiceMultipleSubjects(t *testing.T) {
	tb := &testBus{connected: true}
	fn := NewFakeNotification()
	s := NewService(tb, fn)
	ctx := context.Background()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer s.Stop(ctx)

	// Send events on different subjects
	tb.deliver(ctx, "devos.intent.completed", makeLifecycleEvent(t, event.TypeIntentCompleted, "ic-1", ""))
	tb.deliver(ctx, "devos.intent.failed", makeLifecycleEvent(t, event.TypeIntentFailed, "if-1", ""))
	tb.deliver(ctx, "devos.task.status", makeLifecycleEvent(t, event.TypeTaskStatus, "ts-1", ""))
	tb.deliver(ctx, "devos.deploy.completed", makeDeployEvent(t, "deploy-1"))

	if fn.SendCount.Load() != 4 {
		t.Errorf("send count: got=%d want=4", fn.SendCount.Load())
	}
}

func TestServiceContextCancellation(t *testing.T) {
	tb := &testBus{connected: true}
	fn := NewFakeNotification()
	s := NewService(tb, fn)
	ctx, cancel := context.WithCancel(context.Background())

	// Make notifier block until context is cancelled
	fn.SendFunc = func(ctx context.Context, _ NotificationPayload) error {
		<-ctx.Done()
		return ctx.Err()
	}

	if err := s.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer s.Stop(ctx)

	done := make(chan struct{})
	go func() {
		tb.deliver(ctx, "devos.intent.completed", makeLifecycleEvent(t, event.TypeIntentCompleted, "intent-1", ""))
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not exit after cancellation")
	}
}

func TestServiceShutdownDuringDispatch(t *testing.T) {
	tb := &testBus{connected: true}
	fn := NewFakeNotification()
	fn.SendFunc = func(ctx context.Context, _ NotificationPayload) error {
		time.Sleep(200 * time.Millisecond)
		return nil
	}
	s := NewService(tb, fn)
	ctx := context.Background()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}

	go tb.deliver(ctx, "devos.intent.completed", makeLifecycleEvent(t, event.TypeIntentCompleted, "intent-1", ""))
	time.Sleep(50 * time.Millisecond)

	if err := s.Stop(ctx); err != nil {
		t.Fatalf("stop: %v", err)
	}

	if fn.SendCount.Load() != 1 {
		t.Errorf("send count: got=%d want=1", fn.SendCount.Load())
	}
}

// ---------------------------------------------------------------------------
// Published event content tests
// ---------------------------------------------------------------------------

func TestServiceDeployEventContent(t *testing.T) {
	tb := &testBus{connected: true}
	fn := NewFakeNotification()
	s := NewService(tb, fn)
	ctx := context.Background()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer s.Stop(ctx)

	tb.deliver(ctx, "devos.deploy.completed", makeDeployEvent(t, "deployed successfully"))

	if fn.Received[0].Summary != "deployed successfully" {
		t.Errorf("summary=%s", fn.Received[0].Summary)
	}
}

func TestServiceRendererChannelIntegration(t *testing.T) {
	tb := &testBus{connected: true}
	fn := NewFakeNotification()
	fc := NewFakeChannelProvider("discord")
	fr := NewFakeRenderer()
	s := NewService(tb, fn, WithRenderer(fr), WithChannel(fc))
	ctx := context.Background()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer s.Stop(ctx)

	tb.deliver(ctx, "devos.intent.completed", makeLifecycleEvent(t, event.TypeIntentCompleted, "intent-1", "integration"))

	time.Sleep(50 * time.Millisecond)

	if fr.RenderCount.Load() != 1 {
		t.Errorf("render count: got=%d", fr.RenderCount.Load())
	}
	if fc.SendCount.Load() != 1 {
		t.Errorf("channel send count: got=%d", fc.SendCount.Load())
	}

	// Verify the message sent through the channel matches the rendered output
	if len(fc.Received) > 0 && fc.Received[0].Content == "" {
		t.Error("channel received empty message")
	}
}
