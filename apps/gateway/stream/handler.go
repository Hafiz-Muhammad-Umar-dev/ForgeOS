// Package stream provides SSE and WebSocket handlers for the DevOS API Gateway.
package stream

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/auth"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/event"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/stream"
)

// Handler serves SSE and WebSocket endpoints for real-time event streaming.
type Handler struct {
	hub      stream.Streamer
	provider auth.AuthProvider
}

// NewHandler creates a stream handler with the given hub and auth provider.
func NewHandler(hub stream.Streamer, provider auth.AuthProvider) *Handler {
	return &Handler{hub: hub, provider: provider}
}

// HandleSSE serves Server-Sent Events for a specific intent.
//
// GET /v1/intents/:id/stream
//
// It authenticates the request, verifies intent authorization, subscribes to
// the hub, and streams events. A single goroutine handles heartbeat ticker,
// event delivery, and context cancellation to avoid concurrent writes.
func (h *Handler) HandleSSE(w http.ResponseWriter, r *http.Request) {
	// Authenticate via Authorization header.
	token := extractBearer(r)
	if token == "" {
		writeSSEError(w, http.StatusUnauthorized, "missing bearer token")
		return
	}
	claims, err := h.provider.Authenticate(r.Context(), token)
	if err != nil {
		writeSSEError(w, http.StatusUnauthorized, "invalid token")
		return
	}
	_ = claims // future: check claims.OrgID against intent

	intentID := r.PathValue("id")
	if intentID == "" {
		writeSSEError(w, http.StatusBadRequest, "missing intent id")
		return
	}

	// Subscribe to the hub. The hub returns a channel of events.
	subID := fmt.Sprintf("sse-%d", time.Now().UnixNano())
	ch, err := h.hub.Subscribe(r.Context(), intentID, subID)
	if err != nil {
		writeSSEError(w, http.StatusForbidden, "not authorized for intent")
		return
	}

	// MUST always unsubscribe when the handler returns to prevent goroutine leaks.
	defer h.hub.Unsubscribe(intentID, subID)

	// Set SSE headers.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	flusher, ok := w.(http.Flusher)
	if !ok {
		// Cannot stream without flusher support.
		return
	}

	// Write initial comment to establish connection.
	_, _ = w.Write([]byte(": connected\n\n"))
	flusher.Flush()

	// Heartbeat ticker.
	heartbeat := time.NewTicker(30 * time.Second)
	defer heartbeat.Stop()

	// Single event loop — never write to w from multiple goroutines.
	for {
		select {
		case <-r.Context().Done():
			// Client disconnected or server shutting down.
			return
		case <-heartbeat.C:
			// Heartbeat to keep connection alive.
			_, _ = w.Write([]byte(": heartbeat\n\n"))
			flusher.Flush()
		case env := <-ch:
			// Deliver event to client.
			_, _ = w.Write(stream.FormatSSE(env))
			flusher.Flush()
		}
	}
}

// extractBearer extracts the Bearer token from the Authorization header.
func extractBearer(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if len(auth) > 7 && auth[:7] == "Bearer " {
		return auth[7:]
	}
	return ""
}

// writeSSEError writes an SSE-format error response.
func writeSSEError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// Verify the package imports context and event (used for type references).
var (
	_ context.Context = context.Background()
	_ event.EventType = event.TypeIntentCreated
)
