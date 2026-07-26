package notification

import (
	"context"
	"fmt"
)

// Renderer converts a NotificationPayload into a ChannelMessage ready
// for delivery through a ChannelProvider.
type Renderer interface {
	// Render converts a notification into a channel message.
	Render(ctx context.Context, notification NotificationPayload) (ChannelMessage, error)
}

// Compile-time check.
var _ Renderer = (*TextRenderer)(nil)

// TextRenderer is a Renderer that produces plain-text messages.
// It is the Sprint 0 default renderer.
type TextRenderer struct{}

// NewTextRenderer creates a TextRenderer.
func NewTextRenderer() *TextRenderer {
	return &TextRenderer{}
}

// Render converts a notification to a plain-text message.
func (r *TextRenderer) Render(_ context.Context, notification NotificationPayload) (ChannelMessage, error) {
	content := r.formatContent(notification)

	embedColor := defaultColor(notification.Type)

	return ChannelMessage{
		Content: content,
		Embeds: []Embed{
			{
				Description: content,
				Color:       embedColor,
			},
		},
	}, nil
}

func (r *TextRenderer) formatContent(notification NotificationPayload) string {
	switch notification.Type {
	case "intent.completed", "task.status":
		return fmt.Sprintf("✅ %s\nIntent: %s\nSummary: %s",
			notification.Type, notification.IntentID, notification.Summary)
	case "intent.failed", "task.failed":
		return fmt.Sprintf("❌ %s\nIntent: %s\nError: %s",
			notification.Type, notification.IntentID, notification.Error)
	case "deploy.completed":
		return fmt.Sprintf("🚀 %s\nIntent: %s\nSummary: %s",
			notification.Type, notification.IntentID, notification.Summary)
	default:
		if notification.Error != "" {
			return fmt.Sprintf("⚠️ %s\nIntent: %s\nError: %s",
				notification.Type, notification.IntentID, notification.Error)
		}
		return fmt.Sprintf("ℹ️ %s\nIntent: %s\nSummary: %s",
			notification.Type, notification.IntentID, notification.Summary)
	}
}

func defaultColor(eventType string) int {
	switch eventType {
	case "intent.completed", "task.status", "deploy.completed":
		return 0x00FF00 // green
	case "intent.failed", "task.failed":
		return 0xFF0000 // red
	default:
		return 0x808080 // grey
	}
}
