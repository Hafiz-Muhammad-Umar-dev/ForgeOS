package notification

import (
	"context"
	"fmt"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/bus"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/event"
)

// Compile-time check.
var _ NotificationPort = (*WebAdapter)(nil)

// WebAdapter is a NotificationPort implementation that pushes notifications
// to Web PWA clients. It publishes the notification as a bus event on the
// intent's subject; the StreamHub subscribes to these events and fans them
// out to the Web PWA subscribers for that intent.
type WebAdapter struct {
	bus bus.BusPort
}

// NewWebAdapter creates a WebAdapter that publishes notifications to the bus.
func NewWebAdapter(b bus.BusPort) *WebAdapter {
	return &WebAdapter{bus: b}
}

// Send publishes a notification event to the bus for delivery to Web PWA
// subscribers of the notification's intent.
func (w *WebAdapter) Send(ctx context.Context, notification NotificationPayload) error {
	if w.bus == nil || notification.IntentID == "" {
		return nil
	}

	env := event.New("notification.sent", "notification", notification,
		event.WithOrgID(notification.OrgID),
		event.WithTraceID(notification.TraceID),
	)

	data, err := event.Serialize(env)
	if err != nil {
		return fmt.Errorf("web: serialize: %w", err)
	}

	// Publish on the intent's subject so the StreamHub can fan it out.
	return w.bus.Publish(ctx, "devos.intent.notify", data)
}
