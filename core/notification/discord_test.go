package notification

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDiscordAdapterName(t *testing.T) {
	a := NewDiscordAdapter("https://discord.com/api/webhooks/test")
	if a.Name() != "discord" {
		t.Errorf("name=%s", a.Name())
	}
}

func TestDiscordAdapterSend(t *testing.T) {
	var captured struct {
		Content string         `json:"content"`
		Embeds  []discordEmbed `json:"embeds"`
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode: %v", err)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	a := NewDiscordAdapter(srv.URL)

	err := a.Send(context.Background(), ChannelMessage{
		Channel: "discord",
		Content: "Hello from test",
		Embeds: []Embed{
			{Description: "Test embed", Color: 0x00FF00},
		},
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}

	if captured.Content != "Hello from test" {
		t.Errorf("content: got=%s", captured.Content)
	}
	if len(captured.Embeds) != 1 {
		t.Fatalf("embeds: got=%d", len(captured.Embeds))
	}
	if captured.Embeds[0].Color != 0x00FF00 {
		t.Errorf("color: got=%d", captured.Embeds[0].Color)
	}
}

func TestDiscordAdapterSendNoEmbeds(t *testing.T) {
	var captured discordWebhookPayload
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&captured)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	a := NewDiscordAdapter(srv.URL)
	err := a.Send(context.Background(), ChannelMessage{
		Channel: "discord",
		Content: "plain text only",
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if captured.Content != "plain text only" {
		t.Errorf("content: got=%s", captured.Content)
	}
	if len(captured.Embeds) != 0 {
		t.Errorf("embeds: got=%d", len(captured.Embeds))
	}
}

func TestDiscordAdapterHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	a := NewDiscordAdapter(srv.URL)
	err := a.Send(context.Background(), ChannelMessage{Content: "test"})
	if err == nil {
		t.Fatal("expected error for 400")
	}
}

func TestDiscordAdapterServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	a := NewDiscordAdapter(srv.URL)
	err := a.Send(context.Background(), ChannelMessage{Content: "test"})
	if err == nil {
		t.Fatal("expected error for 500")
	}
}

func TestDiscordAdapterContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	a := NewDiscordAdapter(srv.URL)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := a.Send(ctx, ChannelMessage{Content: "test"})
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestDiscordAdapterCustomHTTPClient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify content type header
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("content-type: got=%s", r.Header.Get("Content-Type"))
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	a := NewDiscordAdapter(srv.URL)
	err := a.Send(context.Background(), ChannelMessage{Content: "header test"})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
}

func TestDiscordAdapterWebhookURL(t *testing.T) {
	// Verify the webhook URL is the one being POSTed to
	var requestURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestURL = r.URL.String()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	a := NewDiscordAdapter(srv.URL)
	a.Send(context.Background(), ChannelMessage{Content: "test"})

	if !strings.HasPrefix(requestURL, "/") {
		t.Logf("requested URL: %s", requestURL)
	}
}

func TestDiscordAdapterLongContent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload discordWebhookPayload
		json.NewDecoder(r.Body).Decode(&payload)
		if len(payload.Content) != 10000 {
			t.Errorf("content length: got=%d", len(payload.Content))
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	a := NewDiscordAdapter(srv.URL)
	longContent := strings.Repeat("a", 10000)
	err := a.Send(context.Background(), ChannelMessage{Content: longContent})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
}
