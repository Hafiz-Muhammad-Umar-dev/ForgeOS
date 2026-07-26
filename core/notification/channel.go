package notification

import "context"

// ChannelProvider sends pre-rendered messages to a specific outbound channel
// (Discord, Slack, etc.). Each channel adapter implements this interface.
type ChannelProvider interface {
	// Name returns the channel name (e.g., "discord", "slack").
	Name() string

	// Send delivers a pre-rendered message to the channel.
	// Implementations must respect context cancellation.
	Send(ctx context.Context, msg ChannelMessage) error
}

// ChannelMessage is a pre-rendered message ready for delivery to a channel.
// It contains the plain-text content and optional rich embeds.
type ChannelMessage struct {
	// Channel identifies the target channel adapter.
	Channel string `json:"channel"`

	// Content is the plain-text message body.
	Content string `json:"content"`

	// Embeds are rich content blocks (Discord embed, Slack blocks, etc.).
	Embeds []Embed `json:"embeds,omitempty"`
}

// Embed is a rich content embed for platforms that support them (Discord, Slack).
type Embed struct {
	// Title is the embed title.
	Title string `json:"title,omitempty"`

	// Description is the main body of the embed.
	Description string `json:"description,omitempty"`

	// URL is an optional link for the title.
	URL string `json:"url,omitempty"`

	// Color is the embed accent color (hex integer, e.g., 0x00FF00).
	Color int `json:"color,omitempty"`
}
