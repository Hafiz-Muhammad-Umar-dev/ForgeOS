// Package notification implements the Notification Service port and a
// logging adapter for Sprint 0. It subscribes to intent.completed and
// intent.failed events from the bus and dispatches them through a
// NotificationPort.
//
// Sprint 0 scope:
//   - NotificationPort interface (Send)
//   - Subscription to intent.completed / intent.failed
//   - LoggingAdapter (logs what would be sent)
//   - FakeNotification for tests
//
// Excluded from Sprint 0:
//   - Email, SMS, push, Slack, Discord, WhatsApp
//   - Retry queues, templates, persistence, scheduling
//   - Rate limiting, user preferences, external providers
//
// See SDD §07 (Notification Service), ADR-006 (Channel Adapters).
package notification

import "context"

// NotificationPort is the outbound notification abstraction.
// Implementations send notifications through channel adapters
// (Discord, Slack, etc.) or, in Sprint 0, log them.
type NotificationPort interface {
	// Send dispatches a notification.
	Send(ctx context.Context, notification NotificationPayload) error
}

// NotificationPayload carries the data for a single notification.
type NotificationPayload struct {
	// IntentID identifies the originating intent.
	IntentID string `json:"intent_id"`

	// Type is the event type that triggered this notification
	// (e.g., "intent.completed", "intent.failed").
	Type string `json:"type"`

	// Summary is a human-readable result summary.
	Summary string `json:"summary,omitempty"`

	// Error is set for failure notifications.
	Error string `json:"error,omitempty"`

	// OrgID is the tenant organization.
	OrgID string `json:"org_id,omitempty"`

	// TraceID for distributed tracing.
	TraceID string `json:"trace_id,omitempty"`
}
