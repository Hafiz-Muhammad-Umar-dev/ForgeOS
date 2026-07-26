//go:build integration

package notification

import (
	"context"
	"testing"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/event"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/orchestrator"
)

// TestIntegrationNotificationCompleted verifies the notification service
// correctly consumes an intent.completed event and dispatches it.
func TestIntegrationNotificationCompleted(t *testing.T) {
	tb := &testBus{connected: true}
	fn := NewFakeNotification()
	s := NewService(tb, fn)
	ctx := context.Background()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer s.Stop(ctx)

	payload := orchestrator.IntentLifecyclePayload{
		IntentID: "int-integration-1",
		Summary:  "integration test passed",
		OrgID:    "org-int",
		TraceID:  "trace-int",
	}
	env := event.New(event.TypeIntentCompleted, "integration-test", payload)
	data, err := event.Serialize(env)
	if err != nil {
		t.Fatalf("serialize: %v", err)
	}

	tb.deliver(ctx, "devos.intent.completed", data)

	if fn.SendCount.Load() != 1 {
		t.Fatalf("send count: got=%d", fn.SendCount.Load())
	}
	if fn.Received[0].IntentID != "int-integration-1" {
		t.Errorf("intentId=%s", fn.Received[0].IntentID)
	}
	if fn.Received[0].Summary != "integration test passed" {
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

	payload := orchestrator.IntentLifecyclePayload{
		IntentID: "int-integration-fail",
		Error:    "agent error",
		OrgID:    "org-int",
	}
	env := event.New(event.TypeIntentFailed, "integration-test", payload)
	data, _ := event.Serialize(env)

	tb.deliver(ctx, "devos.intent.failed", data)

	if fn.SendCount.Load() != 1 {
		t.Fatalf("send count: got=%d", fn.SendCount.Load())
	}
	if fn.Received[0].IntentID != "int-integration-fail" {
		t.Errorf("intentId=%s", fn.Received[0].IntentID)
	}
	if fn.Received[0].Error != "agent error" {
		t.Errorf("error=%s", fn.Received[0].Error)
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

	payload := orchestrator.IntentLifecyclePayload{
		IntentID: "int-log-test",
		Summary:  "logging test",
		OrgID:    "org-log",
	}
	env := event.New(event.TypeIntentCompleted, "integration-test", payload)
	data, _ := event.Serialize(env)

	tb.deliver(ctx, "devos.intent.completed", data)
	// Logging adapter writes to stdout — no return value to assert.
	// Test passes if no panic or error occurs.
}
