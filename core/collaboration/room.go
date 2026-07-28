package collaboration

import (
	"context"
	"fmt"
	"sync"
)

// MessageType identifies the kind of data being broadcast.
type MessageType int

const (
	MessageUpdate     MessageType = iota // Binary document update (Yjs update)
	MessageAwareness                     // Awareness state change
)

// Room manages a set of sessions joined to the same intent. It receives
// binary updates from any session and broadcasts them to all other sessions.
type Room struct {
	intentID string
	mu       sync.RWMutex
	sessions map[string]*Session
}

// NewRoom creates an empty room for the given intent.
func NewRoom(intentID string) *Room {
	return &Room{
		intentID: intentID,
		sessions: make(map[string]*Session),
	}
}

// IntentID returns the intent this room belongs to.
func (r *Room) IntentID() string { return r.intentID }

// SessionCount returns the number of connected sessions.
func (r *Room) SessionCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.sessions)
}

// AddSession joins a session to the room.
func (r *Room) AddSession(s *Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.sessions[s.ID]; ok {
		return fmt.Errorf("room: session %q already joined", s.ID)
	}
	r.sessions[s.ID] = s
	return nil
}

// RemoveSession removes a session from the room.
func (r *Room) RemoveSession(sessionID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.sessions, sessionID)
}

// Broadcast sends a binary message to every session in the room except the
// sender. Returns the number of recipients.
func (r *Room) Broadcast(senderID string, data []byte) int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var count int
	for id, s := range r.sessions {
		if id == senderID {
			continue
		}
		if s.Send(data) {
			count++
		}
	}
	return count
}

// Sessions returns a snapshot of all session IDs in the room.
func (r *Room) Sessions() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := make([]string, 0, len(r.sessions))
	for id := range r.sessions {
		ids = append(ids, id)
	}
	return ids
}

// Close disconnects all sessions in the room.
func (r *Room) Close(ctx context.Context) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for id, s := range r.sessions {
		_ = s.Close(ctx)
		delete(r.sessions, id)
	}
}
