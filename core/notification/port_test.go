package notification

import (
	"context"
	"testing"
)

func TestNotificationPayload(t *testing.T) {
	p := NotificationPayload{
		IntentID: "intent-1",
		Type:     "intent.completed",
		Summary:  "build succeeded",
		OrgID:    "org-1",
		TraceID:  "trace-1",
	}
	if p.IntentID != "intent-1" {
		t.Errorf("intentId=%s", p.IntentID)
	}
	if p.Type != "intent.completed" {
		t.Errorf("type=%s", p.Type)
	}
	if p.Summary != "build succeeded" {
		t.Errorf("summary=%s", p.Summary)
	}
}

func TestNotificationPayloadError(t *testing.T) {
	p := NotificationPayload{
		IntentID: "intent-1",
		Type:     "intent.failed",
		Error:    "build failed",
		OrgID:    "org-1",
	}
	if p.Error != "build failed" {
		t.Errorf("error=%s", p.Error)
	}
}

func TestSentinelErrors(t *testing.T) {
	errs := []struct {
		err   error
		label string
	}{
		{ErrInvalidEvent, "ErrInvalidEvent"},
		{ErrSendFailed, "ErrSendFailed"},
		{ErrNotStarted, "ErrNotStarted"},
	}
	for _, tt := range errs {
		t.Run(tt.label, func(t *testing.T) {
			if tt.err == nil {
				t.Fatal("error is nil")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// FakeNotification tests
// ---------------------------------------------------------------------------

func TestFakeNotificationSend(t *testing.T) {
	fn := NewFakeNotification()

	err := fn.Send(nil, NotificationPayload{IntentID: "i1", Type: "completed"})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if fn.SendCount.Load() != 1 {
		t.Errorf("count=%d", fn.SendCount.Load())
	}
	if len(fn.Received) != 1 {
		t.Fatalf("received=%d", len(fn.Received))
	}
	if fn.Received[0].IntentID != "i1" {
		t.Errorf("intentId=%s", fn.Received[0].IntentID)
	}
}

func TestFakeNotificationSendError(t *testing.T) {
	fn := NewFakeNotification()
	fn.SendError = ErrSendFailed

	err := fn.Send(nil, NotificationPayload{IntentID: "i1"})
	if err != ErrSendFailed {
		t.Errorf("err=%v", err)
	}
}

func TestFakeNotificationRecordsAll(t *testing.T) {
	fn := NewFakeNotification()

	fn.Send(nil, NotificationPayload{IntentID: "i1", Type: "completed"})
	fn.Send(nil, NotificationPayload{IntentID: "i2", Type: "failed"})

	if len(fn.Received) != 2 {
		t.Fatalf("received=%d", len(fn.Received))
	}
	if fn.Received[0].IntentID != "i1" {
		t.Errorf("first=%s", fn.Received[0].IntentID)
	}
	if fn.Received[1].IntentID != "i2" {
		t.Errorf("second=%s", fn.Received[1].IntentID)
	}
}

func TestFakeNotificationCustomSendFunc(t *testing.T) {
	fn := NewFakeNotification()
	fn.SendFunc = func(_ context.Context, notification NotificationPayload) error {
		if notification.IntentID == "allowed" {
			return nil
		}
		return ErrSendFailed
	}

	if err := fn.Send(nil, NotificationPayload{IntentID: "allowed"}); err != nil {
		t.Errorf("should be allowed: %v", err)
	}
	if err := fn.Send(nil, NotificationPayload{IntentID: "denied"}); err == nil {
		t.Error("should be denied")
	}
}

// ---------------------------------------------------------------------------
// LoggingAdapter tests
// ---------------------------------------------------------------------------

func TestLoggingAdapterSend(t *testing.T) {
	la := NewLoggingAdapter()

	err := la.Send(nil, NotificationPayload{
		IntentID: "intent-1",
		Type:     "intent.completed",
		Summary:  "all good",
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
}

func TestLoggingAdapterSendFailed(t *testing.T) {
	la := NewLoggingAdapter()

	err := la.Send(nil, NotificationPayload{
		IntentID: "intent-1",
		Type:     "intent.failed",
		Error:    "something went wrong",
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
}
