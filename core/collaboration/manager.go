package collaboration

import (
	"context"
	"fmt"
	"sync"
)

// Manager maintains the set of active collaboration rooms. It provides
// room creation, lookup, and removal, and implements CollaborationService.
type Manager struct {
	mu    sync.Mutex
	rooms map[string]*Room
}

// NewManager creates an empty manager.
func NewManager() *Manager {
	return &Manager{
		rooms: make(map[string]*Room),
	}
}

// JoinRoom adds a session to the room for the given intent, creating the
// room if it does not exist.
func (m *Manager) JoinRoom(ctx context.Context, intentID string, session *Session) error {
	m.mu.Lock()
	room, ok := m.rooms[intentID]
	if !ok {
		room = NewRoom(intentID)
		m.rooms[intentID] = room
	}
	m.mu.Unlock()

	return room.AddSession(session)
}

// LeaveRoom removes a session from its room. If the room becomes empty, it
// is removed.
func (m *Manager) LeaveRoom(ctx context.Context, intentID string, sessionID string) error {
	m.mu.Lock()
	room, ok := m.rooms[intentID]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("manager: room %q not found", intentID)
	}
	m.mu.Unlock()

	room.RemoveSession(sessionID)

	if room.SessionCount() == 0 {
		m.mu.Lock()
		// Double-check after re-acquiring the lock.
		if room.SessionCount() == 0 {
			delete(m.rooms, intentID)
		}
		m.mu.Unlock()
	}

	return nil
}

// Room returns the room for the given intent, or nil.
func (m *Manager) Room(intentID string) *Room {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.rooms[intentID]
}

// RoomCount returns the number of active rooms.
func (m *Manager) RoomCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.rooms)
}

// Close shuts down all rooms and their sessions.
func (m *Manager) Close(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, room := range m.rooms {
		room.Close(ctx)
		delete(m.rooms, id)
	}
	return nil
}
