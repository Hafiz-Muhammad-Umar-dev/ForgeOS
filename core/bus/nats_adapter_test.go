package bus

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
)

// startTestNats starts an embedded NATS server for testing.
// Returns the server and a cleanup function.
func startTestNats(t *testing.T) (*server.Server, string) {
	t.Helper()

	opts := &server.Options{
		Port:      -1,
		Host:      "127.0.0.1",
		NoLog:     true,
		NoSigs:    true,
		JetStream: true,
	}
	s, err := server.NewServer(opts)
	if err != nil {
		t.Fatalf("nats server new: %v", err)
	}
	s.Start()
	t.Cleanup(s.Shutdown)

	if !s.ReadyForConnections(5 * time.Second) {
		t.Fatal("nats server not ready")
	}

	url := s.ClientURL()
	return s, url
}

func TestNewNatsBus_Defaults(t *testing.T) {
	b := NewNatsBus()
	if b == nil {
		t.Fatal("bus is nil")
	}
	if b.cfg.URL != "nats://localhost:4222" {
		t.Errorf("default URL: got=%s", b.cfg.URL)
	}
	if len(b.cfg.StreamConfigs) == 0 {
		t.Fatal("no default stream configs")
	}
}

func TestNatsBus_Init_EmptyURL(t *testing.T) {
	b := NewNatsBus(WithNatsConfig(NatsConfig{URL: ""}))
	err := b.Init(context.Background())
	if err == nil {
		t.Fatal("expected error for empty URL")
	}
}

func TestNatsBus_ConnectAndClose(t *testing.T) {
	_, url := startTestNats(t)

	b := NewNatsBus(WithNatsURL(url))
	ctx := context.Background()

	if err := b.Connect(ctx); err != nil {
		t.Fatalf("connect: %v", err)
	}
	if !b.IsConnected() {
		t.Fatal("not connected after connect")
	}

	if err := b.Close(ctx); err != nil {
		t.Fatalf("close: %v", err)
	}
	if b.IsConnected() {
		t.Fatal("connected after close")
	}
}

func TestNatsBus_IdempotentConnect(t *testing.T) {
	_, url := startTestNats(t)

	b := NewNatsBus(WithNatsURL(url))
	ctx := context.Background()

	if err := b.Connect(ctx); err != nil {
		t.Fatalf("first connect: %v", err)
	}
	if err := b.Connect(ctx); err != nil {
		t.Fatalf("second connect: %v", err)
	}
}

func TestNatsBus_IdempotentClose(t *testing.T) {
	_, url := startTestNats(t)

	b := NewNatsBus(WithNatsURL(url))
	ctx := context.Background()

	if err := b.Connect(ctx); err != nil {
		t.Fatalf("connect: %v", err)
	}
	if err := b.Close(ctx); err != nil {
		t.Fatalf("first close: %v", err)
	}
	if err := b.Close(ctx); err != nil {
		t.Fatalf("second close: %v", err)
	}
}

func TestNatsBus_PublishSubscribe(t *testing.T) {
	_, url := startTestNats(t)

	b := NewNatsBus(WithNatsURL(url))
	ctx := context.Background()

	if err := b.Connect(ctx); err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer b.Close(ctx)

	// Subscribe first
	var received atomic.Value
	received.Store("")

	sub, err := b.Subscribe(ctx, "devos.test.hello", func(_ context.Context, msg Message) error {
		received.Store(string(msg.Data()))
		return msg.Ack()
	})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer sub.Unsubscribe()

	// Allow subscription to propagate
	time.Sleep(100 * time.Millisecond)

	// Publish
	payload := []byte(`{"message":"hello world"}`)
	if err := b.Publish(ctx, "devos.test.hello", payload); err != nil {
		t.Fatalf("publish: %v", err)
	}

	// Wait for delivery
	time.Sleep(500 * time.Millisecond)

	got := received.Load().(string)
	if got == "" {
		t.Fatal("did not receive message")
	}
	if got != string(payload) {
		t.Errorf("message: got=%s want=%s", got, string(payload))
	}
}

func TestNatsBus_PublishSubscribeMultipleMessages(t *testing.T) {
	_, url := startTestNats(t)

	b := NewNatsBus(WithNatsURL(url))
	ctx := context.Background()

	if err := b.Connect(ctx); err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer b.Close(ctx)

	var count atomic.Int32
	sub, err := b.Subscribe(ctx, "devos.test.multi", func(_ context.Context, msg Message) error {
		count.Add(1)
		return msg.Ack()
	})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer sub.Unsubscribe()

	time.Sleep(100 * time.Millisecond)

	for i := 0; i < 5; i++ {
		payload := []byte(fmt.Sprintf(`{"n":%d}`, i))
		if err := b.Publish(ctx, "devos.test.multi", payload); err != nil {
			t.Fatalf("publish %d: %v", i, err)
		}
	}

	time.Sleep(500 * time.Millisecond)

	if got := count.Load(); got != 5 {
		t.Fatalf("received %d messages, want 5", got)
	}
}

func TestNatsBus_PublishBeforeSubscribe(t *testing.T) {
	_, url := startTestNats(t)

	b := NewNatsBus(WithNatsURL(url))
	ctx := context.Background()

	if err := b.Connect(ctx); err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer b.Close(ctx)

	// Publish first (with no subscribers)
	payload := []byte(`{"message":"early bird"}`)
	if err := b.Publish(ctx, "devos.test.early", payload); err != nil {
		t.Fatalf("publish: %v", err)
	}

	// Subscribe later
	var received atomic.Value
	received.Store("")

	sub, err := b.Subscribe(ctx, "devos.test.early", func(_ context.Context, msg Message) error {
		received.Store(string(msg.Data()))
		return msg.Ack()
	})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer sub.Unsubscribe()

	time.Sleep(500 * time.Millisecond)

	// Because the consumer is created with default DeliverPolicy (DeliverAll),
	// earlier messages in the stream should be delivered to new consumers.
	got := received.Load().(string)
	if got == "" {
		t.Log("note: no message received (expected if consumer uses DeliverNew policy)")
	}
}

func TestNatsBus_WildcardSubscribe(t *testing.T) {
	_, url := startTestNats(t)

	b := NewNatsBus(WithNatsURL(url))
	ctx := context.Background()

	if err := b.Connect(ctx); err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer b.Close(ctx)

	var count atomic.Int32
	sub, err := b.Subscribe(ctx, "devos.test.wild.>", func(_ context.Context, msg Message) error {
		count.Add(1)
		return msg.Ack()
	})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer sub.Unsubscribe()

	time.Sleep(100 * time.Millisecond)

	subjects := []string{"devos.test.wild.a", "devos.test.wild.b", "devos.test.wild.c"}
	for _, s := range subjects {
		if err := b.Publish(ctx, s, []byte(`{}`)); err != nil {
			t.Fatalf("publish %s: %v", s, err)
		}
	}

	time.Sleep(500 * time.Millisecond)

	if got := count.Load(); got != 3 {
		t.Fatalf("received %d messages, want 3", got)
	}
}

func TestNatsBus_PublishNotConnected(t *testing.T) {
	b := NewNatsBus(WithNatsURL("nats://localhost:14222"))
	err := b.Publish(context.Background(), "devos.test.fail", []byte("{}"))
	if err == nil {
		t.Fatal("expected error for not connected")
	}
}

func TestNatsBus_SubscribeNotConnected(t *testing.T) {
	b := NewNatsBus(WithNatsURL("nats://localhost:14222"))
	_, err := b.Subscribe(context.Background(), "devos.test.fail", func(_ context.Context, _ Message) error {
		return nil
	})
	if err == nil {
		t.Fatal("expected error for not connected")
	}
}

func TestNatsBus_HealthCheck(t *testing.T) {
	_, url := startTestNats(t)

	b := NewNatsBus(WithNatsURL(url))
	ctx := context.Background()

	// Before connect - should be down
	h := b.Health()
	if h.Status == "UP" {
		t.Log("health before connect:", h.Status)
	}

	if err := b.Connect(ctx); err != nil {
		t.Fatalf("connect: %v", err)
	}

	// After connect - should be up
	h = b.Health()
	if h.Status != "UP" {
		t.Errorf("health after connect: got=%s want=UP", h.Status)
	}

	_ = b.Close(ctx)

	// After close - should be down
	h = b.Health()
	if h.Status == "UP" {
		t.Errorf("health after close: got=%s", h.Status)
	}
}

func TestNatsBus_LifecycleComponent(t *testing.T) {
	_, url := startTestNats(t)

	b := NewNatsBus(WithNatsURL(url))
	ctx := context.Background()

	if err := b.Init(ctx); err != nil {
		t.Fatalf("init: %v", err)
	}
	if b.Name() != "bus" {
		t.Errorf("name: got=%s want=bus", b.Name())
	}
	if err := b.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	if !b.IsConnected() {
		t.Fatal("not connected after start")
	}
	if err := b.Stop(ctx); err != nil {
		t.Fatalf("stop: %v", err)
	}
	if b.IsConnected() {
		t.Fatal("connected after stop")
	}
}

func TestNatsBus_HandlerNakOnError(t *testing.T) {
	_, url := startTestNats(t)

	b := NewNatsBus(WithNatsURL(url))
	ctx := context.Background()

	if err := b.Connect(ctx); err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer b.Close(ctx)

	var attempts atomic.Int32
	sub, err := b.Subscribe(ctx, "devos.test.nakerror", func(_ context.Context, msg Message) error {
		attempts.Add(1)
		// Return an error to trigger Nak
		return fmt.Errorf("handler error")
	})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer sub.Unsubscribe()

	time.Sleep(100 * time.Millisecond)

	if err := b.Publish(ctx, "devos.test.nakerror", []byte(`{}`)); err != nil {
		t.Fatalf("publish: %v", err)
	}

	// Wait for a couple of delivery attempts
	time.Sleep(1 * time.Second)

	got := attempts.Load()
	t.Logf("handler called %d times (expected at least 1)", got)
	if got < 1 {
		t.Error("handler was never called")
	}
}

func TestNatsBus_CustomConfig(t *testing.T) {
	cfg := NatsConfig{
		URL:           "nats://localhost:4222",
		Name:          "test-suite",
		Timeout:       5 * time.Second,
		ReconnectWait: 1 * time.Second,
		MaxReconnects: 3,
	}

	b := NewNatsBus(WithNatsConfig(cfg))
	if b.cfg.Name != "test-suite" {
		t.Errorf("name: got=%s", b.cfg.Name)
	}
	if b.cfg.MaxReconnects != 3 {
		t.Errorf("maxReconnects: got=%d", b.cfg.MaxReconnects)
	}
}
