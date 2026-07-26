//go:build integration

package notification

import (
	"context"
	"testing"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/event"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/orchestrator"
)

// TestIntegrationNotificationCompleted verifies the notification service
// correctly consumes an intent.completed event.
func TestIntegrationNotificationCompleted(t *testing.T) {
	tb := &testBus{connected: true}
	fn := NewFakeNotification()
	s := NewService(tb, fn)
	ctx := context.Background()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer s.Stop(ctx)

	payload := orchestrator.IntentLifecyclePayload{IntentID: "int-1", Summary: "integration pass", OrgID: "org-int"}
	env := event.New(event.TypeIntentCompleted, "test", payload)
	data, _ := event.Serialize(env)

	tb.deliver(ctx, "devos.intent.completed", data)

	if fn.SendCount.Load() != 1 {
		t.Fatalf("send count: got=%d", fn.SendCount.Load())
	}
	if fn.Received[0].Summary != "integration pass" {
		t.Errorf("summary=%s", fn.Received[0].Summary)
	}
}

// TestIntegrationNotificationFailed verifies the failure path.
func TestIntegrationNotificationFailed(t *testing.T) {
	tb := &testBus{connected: true}
	fn := NewFakeNotification()
	s := NewService(tb, fn)
	ctx := context.Background()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer s.Stop(ctx)

	payload := orchestrator.IntentLifecyclePayload{IntentID: "int-2", Error: "agent error", OrgID: "org-int"}
	env := event.New(event.TypeIntentFailed, "test", payload)
	data, _ := event.Serialize(env)

	tb.deliver(ctx, "devos.intent.failed", data)

	if fn.SendCount.Load() != 1 {
		t.Fatalf("send count: got=%d", fn.SendCount.Load())
	}
	if fn.Received[0].Error != "agent error" {
		t.Errorf("error=%s", fn.Received[0].Error)
	}
}

// TestIntegrationNotificationDeploy verifies deploy events.
func TestIntegrationNotificationDeploy(t *testing.T) {
	tb := &testBus{connected: true}
	fn := NewFakeNotification()
	s := NewService(tb, fn)
	ctx := context.Background()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer s.Stop(ctx)

	payload := orchestrator.IntentLifecyclePayload{IntentID: "int-3", Summary: "deployed", OrgID: "org-int"}
	env := event.New(event.TypeDeployCompleted, "test", payload)
	data, _ := event.Serialize(env)

	tb.deliver(ctx, "devos.deploy.completed", data)

	if fn.SendCount.Load() != 1 {
		t.Fatalf("send count: got=%d", fn.SendCount.Load())
	}
	if fn.Received[0].Type != "deploy.completed" {
		t.Errorf("type=%s", fn.Received[0].Type)
	}
	if fn.Received[0].Summary != "deployed" {
		t.Errorf("summary=%s", fn.Received[0].Summary)
	}
}

// TestIntegrationNotificationWithRendererAndChannel verifies the full
// notification pipeline: event → render → send.
func TestIntegrationNotificationWithRendererAndChannel(t *testing.T) {
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

	payload := orchestrator.IntentLifecyclePayload{IntentID: "int-4", Summary: "full pipeline", OrgID: "org-int"}
	env := event.New(event.TypeIntentCompleted, "test", payload)
	data, _ := event.Serialize(env)

	tb.deliver(ctx, "devos.intent.completed", data)

	if fr.RenderCount.Load() != 1 {
		t.Errorf("render count: got=%d", fr.RenderCount.Load())
	}
	if fc.SendCount.Load() != 1 {
		t.Errorf("channel send count: got=%d", fc.SendCount.Load())
	}
}

// TestIntegrationNotificationWithLoggingAdapter verifies the logging adapter.
func TestIntegrationNotificationWithLoggingAdapter(t *testing.T) {
	tb := &testBus{connected: true}
	la := NewLoggingAdapter()
	s := NewService(tb, la)
	ctx := context.Background()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer s.Stop(ctx)

	payload := orchestrator.IntentLifecyclePayload{IntentID: "int-log", Summary: "log test", OrgID: "org-log"}
	env := event.New(event.TypeIntentCompleted, "test", payload)
	data, _ := event.Serialize(env)

	tb.deliver(ctx, "devos.intent.completed", data)
	// Logging adapter writes to stdout — no return value to assert.
}

// TestIntegrationMultipleSubjects verifies all subjects are subscribed.
func TestIntegrationMultipleSubjects(t *testing.T) {
	tb := &testBus{connected: true}
	fn := NewFakeNotification()
	s := NewService(tb, fn)
	ctx := context.Background()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer s.Stop(ctx)

	completed := orchestrator.IntentLifecyclePayload{IntentID: "i1", Summary: "completed"}
	failed := orchestrator.IntentLifecyclePayload{IntentID: "i2", Error: "failed"}
	deploy := orchestrator.IntentLifecyclePayload{IntentID: "i3", Summary: "deployed"}

	ce, _ := event.Serialize(event.New(event.TypeIntentCompleted, "test", completed))
	fe, _ := event.Serialize(event.New(event.TypeIntentFailed, "test", failed))
	de, _ := event.Serialize(event.New(event.TypeDeployCompleted, "test", deploy))

	tb.deliver(ctx, "devos.intent.completed", ce)
	tb.deliver(ctx, "devos.intent.failed", fe)
	tb.deliver(ctx, "devos.deploy.completed", de)

	if fn.SendCount.Load() != 3 {
		t.Errorf("send count: got=%d want=3", fn.SendCount.Load())
	}
}
