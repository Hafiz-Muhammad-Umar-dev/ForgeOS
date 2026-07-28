package stream

import (
	"fmt"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/event"
)

// FormatSSE formats an event envelope as an SSE data frame.
// Format:
//
//	event: <event_type>
//	id: <event_id>
//	data: <json_payload>
//
func FormatSSE(env event.RawEnvelope) []byte {
	return []byte(fmt.Sprintf("event: %s\nid: %s\ndata: %s\n\n", env.Type, env.ID, string(env.Payload)))
}

// FormatSSEHeartbeat returns an SSE comment line for connection keepalive.
// These are ignored by the EventSource API but prevent proxy timeouts.
func FormatSSEHeartbeat() []byte {
	return []byte(": heartbeat\n\n")
}
