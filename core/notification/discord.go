package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Compile-time check.
var _ ChannelProvider = (*DiscordAdapter)(nil)

// discordWebhookPayload is the JSON payload sent to Discord's webhook API.
type discordWebhookPayload struct {
	Content string        `json:"content"`
	Embeds  []discordEmbed `json:"embeds,omitempty"`
}

type discordEmbed struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	URL         string `json:"url,omitempty"`
	Color       int    `json:"color,omitempty"`
}

// DiscordAdapter is a ChannelProvider that sends notifications through a
// Discord webhook. It converts ChannelMessage into Discord's webhook JSON
// format and POSTs it to the configured webhook URL.
type DiscordAdapter struct {
	webhookURL string
	client     *http.Client
}

// NewDiscordAdapter creates a DiscordAdapter that posts to the given webhook URL.
func NewDiscordAdapter(webhookURL string, opts ...DiscordOption) *DiscordAdapter {
	a := &DiscordAdapter{
		webhookURL: webhookURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
	for _, fn := range opts {
		fn(a)
	}
	return a
}

// DiscordOption configures the DiscordAdapter.
type DiscordOption func(*DiscordAdapter)

// WithDiscordHTTPClient sets the HTTP client for the adapter.
func WithDiscordHTTPClient(client *http.Client) DiscordOption {
	return func(a *DiscordAdapter) { a.client = client }
}

// Name returns "discord".
func (a *DiscordAdapter) Name() string { return "discord" }

// Send posts a message to the Discord webhook.
func (a *DiscordAdapter) Send(ctx context.Context, msg ChannelMessage) error {
	payload := a.buildPayload(msg)

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("discord: marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.webhookURL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("discord: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("discord: request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("discord: %s", resp.Status)
	}

	return nil
}

// buildPayload converts a ChannelMessage into Discord's webhook format.
func (a *DiscordAdapter) buildPayload(msg ChannelMessage) discordWebhookPayload {
	p := discordWebhookPayload{
		Content: msg.Content,
	}
	for _, e := range msg.Embeds {
		p.Embeds = append(p.Embeds, discordEmbed{
			Title:       e.Title,
			Description: e.Description,
			URL:         e.URL,
			Color:       e.Color,
		})
	}
	return p
}
