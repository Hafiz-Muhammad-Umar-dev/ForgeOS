package notification

import (
	"context"
	"encoding/json"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/event"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/stream"
)

// notificationToJSON serializes a notification payload to JSON bytes.
func notificationToJSON(n NotificationPayload) []byte {
	data, _ := json.Marshal(n)
	return data
}

// Compile-time check.
var _ NotificationPort = (*WebAdapter)(nil)

// WebAdapter is a NotificationPort implementation that pushes notifications
// to Web PWA clients via the StreamHub. It is one of several adapters called
// by the Notification Service (alongside LoggingAdapter, DiscordAdapter).
type WebAdapter struct {
	hub stream.Streamer
}

// NewWebAdapter creates a WebAdapter backed by the given Streamer.
func NewWebAdapter(hub stream.Streamer) *WebAdapter {
	return &WebAdapter{hub: hub}
}

// Send pushes a notification to the Web PWA. It maps the notification's
// intent ID to a stream subscriber and delivers an event envelope.
func (w *WebAdapter) Send(ctx context.Context, notification NotificationPayload) error {
	if w.hub == nil || notification.IntentID == "" {
		return nil
	}

	// Build a raw envelope for delivery.
	env := event.RawEnvelope{
		Type:    "notification.sent",
		Payload: []byte(notificationToJSON(notification)),
	}

	// Subscribe (ephemeral) to the intent's stream, deliver, unsubscribe.
	subID := "web-" + notification.IntentID
	ch, err := w.hub.Subscribe(ctx, notification.IntentID, subID)
	if err != nil {
		return err
	}
	defer w.hub.Unsubscribe(notification.IntentID, subID)

	select {
	case ch <- env:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
