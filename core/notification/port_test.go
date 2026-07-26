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
}

func TestNotificationPayloadError(t *testing.T) {
	p := NotificationPayload{IntentID: "i1", Type: "intent.failed", Error: "build failed"}
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

func TestChannelMessage(t *testing.T) {
	msg := ChannelMessage{
		Channel: "discord",
		Content: "hello",
		Embeds:  []Embed{{Title: "Test", Color: 0x00FF00}},
	}
	if msg.Channel != "discord" {
		t.Errorf("channel=%s", msg.Channel)
	}
	if len(msg.Embeds) != 1 {
		t.Errorf("embeds=%d", len(msg.Embeds))
	}
	if msg.Embeds[0].Color != 0x00FF00 {
		t.Errorf("color=%d", msg.Embeds[0].Color)
	}
}

func TestEmbed(t *testing.T) {
	e := Embed{Title: "Title", Description: "Desc", URL: "https://example.com", Color: 0xFF0000}
	if e.Title != "Title" {
		t.Errorf("title=%s", e.Title)
	}
	if e.Description != "Desc" {
		t.Errorf("desc=%s", e.Description)
	}
	if e.Color != 0xFF0000 {
		t.Errorf("color=%d", e.Color)
	}
}

func TestDefaultColors(t *testing.T) {
	tests := []struct {
		eventType string
		expected  int
	}{
		{"intent.completed", 0x00FF00},
		{"task.status", 0x00FF00},
		{"deploy.completed", 0x00FF00},
		{"intent.failed", 0xFF0000},
		{"task.failed", 0xFF0000},
		{"unknown", 0x808080},
	}
	for _, tt := range tests {
		got := defaultColor(tt.eventType)
		if got != tt.expected {
			t.Errorf("defaultColor(%s): got=%d want=%d", tt.eventType, got, tt.expected)
		}
	}
}

// ---------------------------------------------------------------------------
// TextRenderer tests
// ---------------------------------------------------------------------------

func TestTextRenderer(t *testing.T) {
	r := NewTextRenderer()
	msg, err := r.Render(context.Background(), NotificationPayload{
		IntentID: "intent-1",
		Type:     "intent.completed",
		Summary:  "all good",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if msg.Content == "" {
		t.Error("empty content")
	}
	if len(msg.Embeds) != 1 {
		t.Errorf("embeds=%d", len(msg.Embeds))
	}
}

func TestTextRendererFailed(t *testing.T) {
	r := NewTextRenderer()
	msg, err := r.Render(context.Background(), NotificationPayload{
		IntentID: "intent-1",
		Type:     "intent.failed",
		Error:    "something broke",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if msg.Content == "" {
		t.Error("empty content")
	}
}

func TestTextRendererDeploy(t *testing.T) {
	r := NewTextRenderer()
	msg, err := r.Render(context.Background(), NotificationPayload{
		IntentID: "intent-1",
		Type:     "deploy.completed",
		Summary:  "deployed to production",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if msg.Content == "" {
		t.Error("empty content")
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
}

func TestFakeNotificationRecordsMultiple(t *testing.T) {
	fn := NewFakeNotification()
	fn.Send(nil, NotificationPayload{IntentID: "i1"})
	fn.Send(nil, NotificationPayload{IntentID: "i2"})
	if len(fn.Received) != 2 {
		t.Errorf("received=%d", len(fn.Received))
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
// FakeChannelProvider tests
// ---------------------------------------------------------------------------

func TestFakeChannelProvider(t *testing.T) {
	fc := NewFakeChannelProvider("discord")
	if fc.Name() != "discord" {
		t.Errorf("name=%s", fc.Name())
	}
}

func TestFakeChannelProviderSend(t *testing.T) {
	fc := NewFakeChannelProvider("discord")
	err := fc.Send(nil, ChannelMessage{Content: "test"})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if fc.SendCount.Load() != 1 {
		t.Errorf("count=%d", fc.SendCount.Load())
	}
	if len(fc.Received) != 1 {
		t.Errorf("received=%d", len(fc.Received))
	}
}

func TestFakeChannelProviderSendError(t *testing.T) {
	fc := NewFakeChannelProvider("discord")
	fc.SendError = ErrSendFailed
	err := fc.Send(nil, ChannelMessage{Content: "test"})
	if err != ErrSendFailed {
		t.Errorf("err=%v", err)
	}
}

// ---------------------------------------------------------------------------
// FakeRenderer tests
// ---------------------------------------------------------------------------

func TestFakeRenderer(t *testing.T) {
	fr := NewFakeRenderer()
	msg, err := fr.Render(nil, NotificationPayload{IntentID: "i1"})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if msg.Content != "rendered notification" {
		t.Errorf("content=%s", msg.Content)
	}
	if fr.RenderCount.Load() != 1 {
		t.Errorf("count=%d", fr.RenderCount.Load())
	}
}

func TestFakeRendererRecordsNotifications(t *testing.T) {
	fr := NewFakeRenderer()
	fr.Render(nil, NotificationPayload{IntentID: "i1"})
	fr.Render(nil, NotificationPayload{IntentID: "i2"})
	if len(fr.Received) != 2 {
		t.Errorf("received=%d", len(fr.Received))
	}
}

func TestFakeRendererCustomFunc(t *testing.T) {
	fr := NewFakeRenderer()
	fr.RenderFunc = func(_ context.Context, notification NotificationPayload) (ChannelMessage, error) {
		return ChannelMessage{Content: "custom: " + notification.IntentID}, nil
	}
	msg, _ := fr.Render(nil, NotificationPayload{IntentID: "test123"})
	if msg.Content != "custom: test123" {
		t.Errorf("content=%s", msg.Content)
	}
}

// ---------------------------------------------------------------------------
// LoggingAdapter tests
// ---------------------------------------------------------------------------

func TestLoggingAdapterSend(t *testing.T) {
	la := NewLoggingAdapter()
	err := la.Send(nil, NotificationPayload{IntentID: "i1", Type: "completed", Summary: "ok"})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
}

func TestLoggingAdapterSendFailed(t *testing.T) {
	la := NewLoggingAdapter()
	err := la.Send(nil, NotificationPayload{IntentID: "i1", Type: "failed", Error: "error"})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
}
