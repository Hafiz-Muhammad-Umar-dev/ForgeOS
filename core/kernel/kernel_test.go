package kernel

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/bus"
	"github.com/nats-io/nats-server/v2/server"
)

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
	return s, s.ClientURL()
}

func TestNewWithBus(t *testing.T) {
	_, url := startTestNats(t)

	k, err := New(WithBusURL(url))
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	if k.Bus == nil {
		t.Fatal("Bus is nil")
	}
	if !k.Bus.IsConnected() {
		t.Fatal("bus not connected after New")
	}

	// Clean up
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := k.Lifecycle.Stop(ctx); err != nil {
		t.Fatalf("stop: %v", err)
	}
	if k.Bus.IsConnected() {
		t.Fatal("bus connected after stop")
	}
}

func TestNewBusPublishSubscribe(t *testing.T) {
	_, url := startTestNats(t)

	k, err := New(WithBusURL(url))
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = k.Lifecycle.Stop(ctx)
	}()

	var received atomic.Value
	received.Store("")

	ctx := context.Background()
	sub, err := k.Bus.Subscribe(ctx, "devos.test.kernel", func(_ context.Context, msg bus.Message) error {
		received.Store(string(msg.Data()))
		return msg.Ack()
	})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer sub.Unsubscribe()

	time.Sleep(200 * time.Millisecond)

	payload := []byte(`{"msg":"kernel-integration"}`)
	if err := k.Bus.Publish(ctx, "devos.test.kernel", payload); err != nil {
		t.Fatalf("publish: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	raw := received.Load().(string)
	if raw == "" {
		t.Fatal("did not receive message via bus")
	}
	if raw != string(payload) {
		t.Errorf("message: got=%s want=%s", raw, string(payload))
	}
}

func TestNewWithoutBus(t *testing.T) {
	k, err := New(WithoutBus())
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	if k.Bus != nil {
		t.Fatal("expected nil bus when WithoutBus is set")
	}
}

func TestNewBusLifecycleRun(t *testing.T) {
	_, url := startTestNats(t)

	k, err := New(WithBusURL(url))
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Run should start the bus, block until ctx cancels, then stop cleanly.
	if err := k.Run(ctx); err != nil {
		t.Fatalf("run: %v", err)
	}
	if k.Bus.IsConnected() {
		t.Fatal("bus still connected after Run returns")
	}
}

func TestNewBusHealthCheck(t *testing.T) {
	_, url := startTestNats(t)

	k, err := New(WithBusURL(url))
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = k.Lifecycle.Stop(ctx)
	}()

	health := k.Lifecycle.Health()
	foundBus := false
	for _, h := range health {
		if h.Name == "bus" {
			foundBus = true
			if h.Health.Status == "DOWN" {
				t.Errorf("bus health=DOWN, expected UP")
			}
		}
	}
	if !foundBus {
		t.Fatal("bus component not found in health check")
	}
}

func TestNewWithoutBusHealthCheck(t *testing.T) {
	k, err := New(WithoutBus())
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	health := k.Lifecycle.Health()
	for _, h := range health {
		if h.Name == "bus" {
			t.Fatal("bus component should not be registered")
		}
	}
}
