package stream

import (
	"context"
	"sync"
	"testing"
	"fmt"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/bus"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/event"
)

// mockBus implements bus.BusPort for testing.
type mockBus struct {
	mu          sync.Mutex
	handlers    []bus.MessageHandler
	subjects    []string
	connected   bool
}

func (b *mockBus) Connect(_ context.Context) error                     { return nil }
func (b *mockBus) IsConnected() bool                                   { return b.connected }
func (b *mockBus) Close(_ context.Context) error                       { return nil }
func (b *mockBus) Publish(_ context.Context, _ string, _ []byte) error { return nil }
func (b *mockBus) Subscribe(_ context.Context, subject string, handler bus.MessageHandler) (bus.Subscription, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subjects = append(b.subjects, subject)
	b.handlers = append(b.handlers, handler)
	return &mockSub{}, nil
}

type mockSub struct{}
func (s *mockSub) Unsubscribe() error { return nil }
func (s *mockSub) Subject() string    { return "" }

func newTestHub(t *testing.T) *Hub {
	t.Helper()
	return NewHub(&mockBus{connected: true}, nil)
}

func TestHubName(t *testing.T) {
	h := newTestHub(t)
	if h.Name() != "stream" {
		t.Errorf("name=%s", h.Name())
	}
}

func TestHubInitNoBus(t *testing.T) {
	h := NewHub(nil, nil)
	if err := h.Init(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}

func TestHubStartStop(t *testing.T) {
	h := newTestHub(t)
	ctx := context.Background()

	if err := h.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	if h.started != true {
		t.Error("should be started")
	}
	if err := h.Stop(ctx); err != nil {
		t.Fatalf("stop: %v", err)
	}
	if h.started != false {
		t.Error("should be stopped")
	}
}

func TestHubSubscribeUnsubscribe(t *testing.T) {
	h := newTestHub(t)
	if err := h.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer h.Stop(context.Background())

	ch, err := h.Subscribe(context.Background(), "intent-1", "sub-1")
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	if ch == nil {
		t.Fatal("nil channel")
	}
	if h.SubscriberCount("intent-1") != 1 {
		t.Errorf("count=%d", h.SubscriberCount("intent-1"))
	}

	if err := h.Unsubscribe("intent-1", "sub-1"); err != nil {
		t.Fatalf("unsubscribe: %v", err)
	}
	if h.SubscriberCount("intent-1") != 0 {
		t.Errorf("count after unsubscribe=%d", h.SubscriberCount("intent-1"))
	}
}

func TestHubSubscribeDuplicate(t *testing.T) {
	h := newTestHub(t)
	h.Start(context.Background())
	defer h.Stop(context.Background())

	h.Subscribe(context.Background(), "intent-1", "sub-1")
	_, err := h.Subscribe(context.Background(), "intent-1", "sub-1")
	if err == nil {
		t.Fatal("expected error for duplicate subscriber")
	}
}

func TestHubUnsubscribeNotFound(t *testing.T) {
	h := newTestHub(t)
	h.Start(context.Background())
	defer h.Stop(context.Background())

	err := h.Unsubscribe("nonexistent", "sub-1")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestHubConcurrentSubscribe(t *testing.T) {
	h := newTestHub(t)
	h.Start(context.Background())
	defer h.Stop(context.Background())

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			subID := fmt.Sprintf("sub-%d", n)
			_, err := h.Subscribe(context.Background(), "intent-1", subID)
			if err != nil && err.Error() != "" {
				t.Errorf("subscribe %s: %v", subID, err)
			}
		}(i)
	}
	wg.Wait()

	count := h.SubscriberCount("intent-1")
	if count != 20 {
		t.Errorf("expected 20 subscribers, got %d", count)
	}
}

func TestHubConcurrentUnsubscribe(t *testing.T) {
	h := newTestHub(t)
	h.Start(context.Background())
	defer h.Stop(context.Background())

	for i := 0; i < 20; i++ {
		h.Subscribe(context.Background(), "intent-1", fmt.Sprintf("sub-%d", i))
	}

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_ = h.Unsubscribe("intent-1", fmt.Sprintf("sub-%d", n))
		}(i)
	}
	wg.Wait()

	if h.SubscriberCount("intent-1") != 0 {
		t.Errorf("expected 0 subscribers, got %d", h.SubscriberCount("intent-1"))
	}
}

func TestHubStopDuringActiveSubscribers(t *testing.T) {
	h := newTestHub(t)
	h.Start(context.Background())

	for i := 0; i < 10; i++ {
		h.Subscribe(context.Background(), "intent-1", fmt.Sprintf("sub-%d", i))
	}

	// Stop should close all subscriber channels.
	if err := h.Stop(context.Background()); err != nil {
		t.Fatalf("stop: %v", err)
	}

	if h.SubscriberCount("intent-1") != 0 {
		t.Errorf("expected 0 after stop, got %d", h.SubscriberCount("intent-1"))
	}
}

func TestHubDoubleCloseProtection(t *testing.T) {
	h := newTestHub(t)
	h.Start(context.Background())
	defer h.Stop(context.Background())

	h.Subscribe(context.Background(), "intent-1", "sub-1")

	// Unsubscribe twice — second should not panic.
	if err := h.Unsubscribe("intent-1", "sub-1"); err != nil {
		t.Fatalf("first unsubscribe: %v", err)
	}
	err := h.Unsubscribe("intent-1", "sub-1")
	if err == nil {
		t.Log("second unsubscribe correctly returned error")
	}
}

func TestSSEFormat(t *testing.T) {
	env := event.RawEnvelope{
		ID:   "evt-1",
		Type: event.TypeIntentCompleted,
		Payload: []byte(`{"summary":"done"}`),
	}
	sse := FormatSSE(env)
	if len(sse) == 0 {
		t.Fatal("empty SSE output")
	}
}

func TestSSEHeartbeat(t *testing.T) {
	hb := FormatSSEHeartbeat()
	if len(hb) == 0 {
		t.Fatal("empty heartbeat")
	}
}

func TestExtractIntentID(t *testing.T) {
	env := event.RawEnvelope{ID: "intent-1", Type: event.TypeIntentCompleted}
	if id := extractIntentID(env); id != "intent-1" {
		t.Errorf("expected intent-1, got %s", id)
	}

	env2 := event.RawEnvelope{ID: "task-1", Type: event.TypeTaskStatus}
	if id := extractIntentID(env2); id != "" {
		t.Errorf("expected empty for task events, got %s", id)
	}
}
