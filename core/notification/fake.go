package notification

import (
	"context"
	"sync/atomic"
)

// Compile-time checks.
var (
	_ NotificationPort  = (*FakeNotification)(nil)
	_ ChannelProvider   = (*FakeChannelProvider)(nil)
	_ Renderer          = (*FakeRenderer)(nil)
)

// FakeNotification is an in-memory NotificationPort for testing.
type FakeNotification struct {
	SendFunc  func(ctx context.Context, notification NotificationPayload) error
	SendError error
	SendCount atomic.Int64
	Received  []NotificationPayload
}

func NewFakeNotification() *FakeNotification {
	return &FakeNotification{}
}

func (f *FakeNotification) Send(ctx context.Context, notification NotificationPayload) error {
	f.SendCount.Add(1)
	f.Received = append(f.Received, notification)
	if f.SendFunc != nil {
		return f.SendFunc(ctx, notification)
	}
	return f.SendError
}

// ---------------------------------------------------------------------------
// FakeChannelProvider
// ---------------------------------------------------------------------------

// FakeChannelProvider is an in-memory ChannelProvider for testing.
type FakeChannelProvider struct {
	// NameValue is returned by Name().
	NameValue string

	// SendFunc overrides Send behavior.
	SendFunc func(ctx context.Context, msg ChannelMessage) error

	// SendError is returned by the default Send implementation.
	SendError error

	// SendCount tracks the number of Send calls.
	SendCount atomic.Int64

	// Received records every message sent.
	Received []ChannelMessage
}

// NewFakeChannelProvider creates a FakeChannelProvider with the given name.
func NewFakeChannelProvider(name string) *FakeChannelProvider {
	return &FakeChannelProvider{NameValue: name}
}

func (f *FakeChannelProvider) Name() string { return f.NameValue }

func (f *FakeChannelProvider) Send(ctx context.Context, msg ChannelMessage) error {
	f.SendCount.Add(1)
	f.Received = append(f.Received, msg)
	if f.SendFunc != nil {
		return f.SendFunc(ctx, msg)
	}
	return f.SendError
}

// ---------------------------------------------------------------------------
// FakeRenderer
// ---------------------------------------------------------------------------

// FakeRenderer is an in-memory Renderer for testing.
type FakeRenderer struct {
	// RenderFunc overrides Render behavior.
	RenderFunc func(ctx context.Context, notification NotificationPayload) (ChannelMessage, error)

	// Result is returned by the default Render implementation.
	Result ChannelMessage

	// RenderCount tracks the number of Render calls.
	RenderCount atomic.Int64

	// Received records every notification rendered.
	Received []NotificationPayload
}

// NewFakeRenderer creates a FakeRenderer with a default result.
func NewFakeRenderer() *FakeRenderer {
	return &FakeRenderer{
		Result: ChannelMessage{
			Content: "rendered notification",
			Embeds:  []Embed{{Description: "rendered", Color: 0x00FF00}},
		},
	}
}

func (f *FakeRenderer) Render(ctx context.Context, notification NotificationPayload) (ChannelMessage, error) {
	f.RenderCount.Add(1)
	f.Received = append(f.Received, notification)
	if f.RenderFunc != nil {
		return f.RenderFunc(ctx, notification)
	}
	return f.Result, nil
}
