package notification

import (
	"context"
	"sync/atomic"
)

// Compile-time check.
var _ NotificationPort = (*FakeNotification)(nil)

// FakeNotification is an in-memory NotificationPort for testing.
// It records all sent notifications and returns configurable results.
type FakeNotification struct {
	// SendFunc overrides the Send behavior.
	SendFunc func(ctx context.Context, notification NotificationPayload) error

	// SendError is returned by the default Send implementation.
	SendError error

	// SendCount tracks the number of Send calls.
	SendCount atomic.Int64

	// Received records every notification sent.
	Received []NotificationPayload
}

// NewFakeNotification creates a FakeNotification.
func NewFakeNotification() *FakeNotification {
	return &FakeNotification{}
}

// Send records the notification and returns the configured result.
func (f *FakeNotification) Send(ctx context.Context, notification NotificationPayload) error {
	f.SendCount.Add(1)
	f.Received = append(f.Received, notification)

	if f.SendFunc != nil {
		return f.SendFunc(ctx, notification)
	}
	return f.SendError
}
