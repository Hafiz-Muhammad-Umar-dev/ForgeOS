// Package stream provides event streaming for DevOS. The Streamer interface
// allows consumers to subscribe to events for a specific intent and receive
// them in real-time through Server-Sent Events or WebSocket.
package stream

import (
	"context"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/event"
)

// Streamer is the port for real-time event streaming. Implementations manage
// subscriber connections and fan out bus events to authorized subscribers.
type Streamer interface {
	// Subscribe registers a subscriber for events on the given intent.
	// Each subscriber receives a read-only channel of events.
	Subscribe(ctx context.Context, intentID, subID string) (<-chan event.RawEnvelope, error)

	// Unsubscribe removes a subscriber and cleans up resources.
	Unsubscribe(intentID, subID string) error

	// SubscriberCount returns the number of active subscribers for an intent.
	SubscriberCount(intentID string) int
}
