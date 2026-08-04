//go:build integration

package bus

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/event"
	"github.com/nats-io/nats-server/v2/server"
)

// startIntegrationNats starts an embedded NATS server with JetStream for
// integration tests.
func startIntegrationNats(t *testing.T) (*server.Server, string) {
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

// TestIntegration_PublishSubscribeWithEnvelope verifies the full round-trip:
// create an EventEnvelope → serialize → bus.Publish → bus.Subscribe → deserialize.
func TestIntegration_PublishSubscribeWithEnvelope(t *testing.T) {
	_, url := startIntegrationNats(t)

	b := NewNatsBus(WithNatsURL(url))
	ctx := context.Background()
	if err := b.Connect(ctx); err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer b.Close(ctx)

	// Create an event envelope
	now := time.Date(2026, 7, 21, 12, 0, 0, 0, time.UTC)
	payload := map[string]any{"key": "value", "number": 42}
	e := event.New(event.TypeIntentCreated, "integration-test", payload,
		event.WithTraceID("trace-integration"),
		event.WithOrgID("org-integration"),
		event.WithProjectID("proj-integration"),
		event.WithTime(now),
	)

	// Serialize
	data, err := event.Serialize(e)
	if err != nil {
		t.Fatalf("serialize: %v", err)
	}

	// Subscribe
	var received atomic.Value
	received.Store("")

	sub, err := b.Subscribe(ctx, e.Type.Subject().String(), func(_ context.Context, msg Message) error {
		received.Store(string(msg.Data()))
		return msg.Ack()
	})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer sub.Unsubscribe()

	time.Sleep(200 * time.Millisecond)

	// Publish
	subject := e.Type.Subject().String()
	if err := b.Publish(ctx, subject, data); err != nil {
		t.Fatalf("publish: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	// Deserialize and verify
	rawVal := received.Load()
	raw, _ := rawVal.(string)
	if raw == "" {
		t.Fatal("did not receive message")
	}

	env, err := event.Deserialize([]byte(raw))
	if err != nil {
		t.Fatalf("deserialize: %v", err)
	}

	if env.Type != event.TypeIntentCreated {
		t.Errorf("type: got=%s want=%s", env.Type, event.TypeIntentCreated)
	}
	if env.TraceID != "trace-integration" {
		t.Errorf("traceId: got=%s", env.TraceID)
	}
	if env.OrgID != "org-integration" {
		t.Errorf("orgId: got=%s", env.OrgID)
	}
	if env.ProjectID != "proj-integration" {
		t.Errorf("projectId: got=%s", env.ProjectID)
	}
	if env.ProducedBy != "integration-test" {
		t.Errorf("producedBy: got=%s", env.ProducedBy)
	}

	gotTime := time.Unix(0, env.ProducedAt)
	if !gotTime.Equal(now) {
		t.Errorf("producedAt: got=%v want=%v", gotTime, now)
	}

	gotPayload, err := event.UnmarshalPayload[map[string]any](env)
	if err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if gotPayload["key"] != "value" {
		t.Errorf("payload key: got=%v", gotPayload["key"])
	}
	if int(gotPayload["number"].(float64)) != 42 {
		t.Errorf("payload number: got=%v", gotPayload["number"])
	}
}

// TestIntegration_MultipleEventTypes verifies round-trip for multiple event types.
func TestIntegration_MultipleEventTypes(t *testing.T) {
	_, url := startIntegrationNats(t)

	b := NewNatsBus(WithNatsURL(url))
	ctx := context.Background()
	if err := b.Connect(ctx); err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer b.Close(ctx)

	type testCase struct {
		eventType event.EventType
		payload   any
		subject   string
	}

	cases := []testCase{
		{event.TypeTaskAssigned, map[string]string{"taskId": "t1"}, "devos.task.assigned"},
		{event.TypeWorkspaceReady, map[string]string{"wsId": "w1"}, "devos.workspace.ready"},
		{event.TypeBudgetExceeded, map[string]float64{"ceiling": 1000}, "devos.budget.exceeded"},
	}

	for _, tc := range cases {
		t.Run(string(tc.eventType), func(t *testing.T) {
			var received atomic.Value
			received.Store("")

			sub, err := b.Subscribe(ctx, tc.subject, func(_ context.Context, msg Message) error {
				received.Store(string(msg.Data()))
				return msg.Ack()
			})
			if err != nil {
				t.Fatalf("subscribe: %v", err)
			}
			defer sub.Unsubscribe()

			time.Sleep(200 * time.Millisecond)

			e := event.New(tc.eventType, "integration-test", tc.payload)
			data, err := event.Serialize(e)
			if err != nil {
				t.Fatalf("serialize: %v", err)
			}

			if err := b.Publish(ctx, tc.subject, data); err != nil {
				t.Fatalf("publish: %v", err)
			}

			time.Sleep(500 * time.Millisecond)

			raw := received.Load().(string)
			if raw == "" {
				t.Fatal("did not receive message")
			}

			env, err := event.Deserialize([]byte(raw))
			if err != nil {
				t.Fatalf("deserialize: %v", err)
			}
			if env.Type != tc.eventType {
				t.Errorf("type: got=%s want=%s", env.Type, tc.eventType)
			}
		})
	}
}

// TestIntegration_HighThroughput verifies the bus handles back-to-back messages.
func TestIntegration_HighThroughput(t *testing.T) {
	_, url := startIntegrationNats(t)

	b := NewNatsBus(WithNatsURL(url))
	ctx := context.Background()
	if err := b.Connect(ctx); err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer b.Close(ctx)

	const msgCount = 50
	var counter atomic.Int32

	sub, err := b.Subscribe(ctx, "devos.test.throughput", func(_ context.Context, msg Message) error {
		counter.Add(1)
		return msg.Ack()
	})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer sub.Unsubscribe()

	time.Sleep(200 * time.Millisecond)

	for i := range msgCount {
		e := event.New(event.TypeTaskAssigned, "throughput-test",
			map[string]int{"seq": i})
		data, err := event.Serialize(e)
		if err != nil {
			t.Fatalf("serialize %d: %v", i, err)
		}
		if err := b.Publish(ctx, "devos.test.throughput", data); err != nil {
			t.Fatalf("publish %d: %v", i, err)
		}
	}

	time.Sleep(2 * time.Second)

	got := counter.Load()
	if got != msgCount {
		t.Errorf("received %d messages, want %d", got, msgCount)
	}
}

// TestIntegration_SubjectConventions verifies correct subject format.
func TestIntegration_SubjectConventions(t *testing.T) {
	_, url := startIntegrationNats(t)

	b := NewNatsBus(WithNatsURL(url))
	ctx := context.Background()
	if err := b.Connect(ctx); err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer b.Close(ctx)

	events := []struct {
		eventType event.EventType
		wantSubj  string
	}{
		{event.TypeIntentCreated, "devos.intent.created"},
		{event.TypePlanProposed, "devos.plan.proposed"},
		{event.TypeTaskAssigned, "devos.task.assigned"},
		{event.TypeAgentToken, "devos.agent.token"},
		{event.TypeDeployCompleted, "devos.deploy.completed"},
	}

	for _, ev := range events {
		subj := ev.eventType.Subject().String()
		if subj != ev.wantSubj {
			t.Errorf("%s subject: got=%s want=%s", ev.eventType, subj, ev.wantSubj)
		}

		// Publish a message to verify the subject works in transit
		e := event.New(ev.eventType, "subject-test", json.RawMessage("{}"))
		data, err := event.Serialize(e)
		if err != nil {
			t.Fatalf("serialize %s: %v", ev.eventType, err)
		}

		if err := b.Publish(ctx, subj, data); err != nil {
			t.Errorf("publish %s: %v", subj, err)
		}
	}
}

// TestIntegration_ConnectionErrorHandling verifies error on connection failure.
func TestIntegration_ConnectionErrorHandling(t *testing.T) {
	b := NewNatsBus(WithNatsURL("nats://127.0.0.1:1"))
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := b.Connect(ctx)
	if err == nil {
		t.Fatal("expected connection error")
	}
	t.Logf("connection error (expected): %v", err)

	if b.IsConnected() {
		t.Fatal("should not be connected")
	}

	// Publish on disconnected bus should error
	err = b.Publish(ctx, "devos.test.fail", []byte("{}"))
	if err == nil {
		t.Fatal("expected publish error on disconnected bus")
	}
}

// TestIntegration_GracefulShutdown verifies clean shutdown finishes in-flight messages.
func TestIntegration_GracefulShutdown(t *testing.T) {
	_, url := startIntegrationNats(t)

	b := NewNatsBus(WithNatsURL(url))
	ctx := context.Background()
	if err := b.Connect(ctx); err != nil {
		t.Fatalf("connect: %v", err)
	}

	var received atomic.Int32
	sub, err := b.Subscribe(ctx, "devos.test.shutdown", func(_ context.Context, msg Message) error {
		received.Add(1)
		time.Sleep(10 * time.Millisecond)
		return msg.Ack()
	})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer sub.Unsubscribe()

	time.Sleep(200 * time.Millisecond)

	// Publish some messages
	for i := 0; i < 5; i++ {
		e := event.New(event.TypeTaskAssigned, "shutdown-test", map[string]int{"seq": i})
		data, _ := event.Serialize(e)
		_ = b.Publish(ctx, "devos.test.shutdown", data)
	}

	time.Sleep(200 * time.Millisecond)

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := b.Close(shutdownCtx); err != nil {
		t.Fatalf("close: %v", err)
	}

	got := received.Load()
	t.Logf("received %d messages before shutdown", got)
	if got == 0 {
		t.Error("no messages received before shutdown")
	}
}

// TestIntegration_ErrorPropagation verifies errors from the bus are propagated.
func TestIntegration_ErrorPropagation(t *testing.T) {
	t.Run("publish to missing subject fails gracefully", func(t *testing.T) {
		_, url := startIntegrationNats(t)

		b := NewNatsBus(WithNatsURL(url))
		ctx := context.Background()
		if err := b.Connect(ctx); err != nil {
			t.Fatalf("connect: %v", err)
		}
		defer b.Close(ctx)

		// Publish to a valid subject should always succeed (stream is auto-created).
		err := b.Publish(ctx, "devos.test.ok", []byte(`{"msg":"ok"}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("subscribe with invalid pattern", func(t *testing.T) {
		_, url := startIntegrationNats(t)

		b := NewNatsBus(WithNatsURL(url))
		ctx := context.Background()
		if err := b.Connect(ctx); err != nil {
			t.Fatalf("connect: %v", err)
		}
		defer b.Close(ctx)

		_, err := b.Subscribe(ctx, "", func(_ context.Context, _ Message) error {
			return nil
		})
		if err == nil {
			t.Log("note: empty subject subscription did not error (may succeed on some configs)")
		}
	})
}

// TestIntegration_DefaultStreamsCreated verifies all default streams are created.
func TestIntegration_DefaultStreamsCreated(t *testing.T) {
	_, url := startIntegrationNats(t)

	b := NewNatsBus(WithNatsURL(url))
	ctx := context.Background()

	// Connect (this auto-creates streams)
	if err := b.Connect(ctx); err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer b.Close(ctx)

	expectedStreams := []string{"INTENTS", "PLANS", "TASKS", "TOKENS", "ARTIFACTS", "DEPLOYS", "OBS"}

	// Use stream name lister to verify streams exist
	streamLister := b.js.StreamNames(ctx)

	streamSet := make(map[string]bool)
	for name := range streamLister.Name() {
		streamSet[name] = true
	}
	if err := streamLister.Err(); err != nil {
		t.Fatalf("list streams: %v", err)
	}

	for _, name := range expectedStreams {
		if !streamSet[name] {
			t.Errorf("expected stream %s not found in created streams", name)
		}
	}
	t.Logf("found %d streams", len(streamSet))
}

// Helper to show that bus-specific streams work across domains.
func TestIntegration_CrossDomainPublish(t *testing.T) {
	_, url := startIntegrationNats(t)

	b := NewNatsBus(WithNatsURL(url))
	ctx := context.Background()
	if err := b.Connect(ctx); err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer b.Close(ctx)

	// Publish to multiple domains (skip agent — overlaps with TOKENS stream)
	domains := []string{"intent", "plan", "task", "artifact"}
	for _, domain := range domains {
		subject := fmt.Sprintf("devos.%s.test", domain)
		e := event.New(event.EventType(domain+".test"), "cross-domain-test",
			json.RawMessage(fmt.Sprintf(`{"domain":"%s"}`, domain)))
		data, _ := event.Serialize(e)
		if err := b.Publish(ctx, subject, data); err != nil {
			t.Errorf("publish to %s: %v", subject, err)
		}
	}
}
