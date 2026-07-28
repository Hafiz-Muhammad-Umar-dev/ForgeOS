// Package collaboration provides intent-scoped collaborative editing rooms.
// Each room manages a set of connected clients for one intent and relays
// binary document updates between participants.
//
// Sprint 1 scope:
//   - Room creation, lookup, and removal
//   - Session lifecycle (connect, disconnect, send queue, heartbeat)
//   - Binary update relay between clients in the same room
//   - Awareness state stub (state relay without schema enforcement)
//   - JSON protocol for WebSocket handshake and control messages
//   - Reuse of existing auth.AuthProvider for authentication
//   - CollaborationService port for lifecycle integration
//
// Excluded from Sprint 2 (deferred):
//   - Yjs document persistence
//   - Document history / snapshot
//   - Version vectors / conflict-free resolution beyond relay
//
// See ADR-005 (CRDT Client Sync), SDD §08 (Query Service).
package collaboration

import "context"

// CollaborationService is the port for real-time collaborative editing.
// Implementations manage rooms, sessions, and binary update relay.
type CollaborationService interface {
	// JoinRoom adds a session to an intent-scoped room.
	JoinRoom(ctx context.Context, intentID string, session *Session) error

	// LeaveRoom removes a session from a room.
	LeaveRoom(ctx context.Context, intentID string, sessionID string) error

	// Room returns the room for an intent, or nil.
	Room(intentID string) *Room

	// Close gracefully shuts down all rooms and sessions.
	Close(ctx context.Context) error
}
