package collaboration

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// SessionConn abstracts the transport connection so the core collaboration
// package does not depend on WebSocket types. Implementations provide the
// transport (WebSocket, in-memory for tests, etc.).
type SessionConn interface {
	// Send writes data to the connection. Must be safe for concurrent calls
	// or serialized externally.
	Send(data []byte) error

	// Receive blocks until a message arrives or the connection closes.
	Receive() ([]byte, error)

	// Close terminates the connection.
	Close() error
}

// Session manages a single client connection within a collaboration room.
// Each session has an authenticated identity and belongs to one intent room.
type Session struct {
	ID       string
	UserID   string
	IntentID string

	conn SessionConn

	mu     sync.Mutex
	closed bool
	done   chan struct{}

	lastHeartbeat atomic.Int64 // unix nanos
}

// NewSession creates a new session with the given connection.
func NewSession(id, userID, intentID string, conn SessionConn) *Session {
	return &Session{
		ID:       id,
		UserID:   userID,
		IntentID: intentID,
		conn:     conn,
		done:     make(chan struct{}),
	}
}

// Send enqueues data to the session's connection. Returns false if the
// session is closed.
func (s *Session) Send(data []byte) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return false
	}
	if err := s.conn.Send(data); err != nil {
		return false
	}
	return true
}

// Receive blocks until the next message arrives or the session is closed.
func (s *Session) Receive() ([]byte, error) {
	type result struct {
		data []byte
		err  error
	}

	ch := make(chan result, 1)
	go func() {
		data, err := s.conn.Receive()
		ch <- result{data, err}
	}()

	select {
	case r := <-ch:
		return r.data, r.err
	case <-s.done:
		return nil, fmt.Errorf("session: closed")
	}
}

// Close terminates the session and its connection.
func (s *Session) Close(ctx context.Context) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	close(s.done)
	s.mu.Unlock()

	return s.conn.Close()
}

// IsClosed reports whether the session has been closed.
func (s *Session) IsClosed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}

// SetHeartbeat records a heartbeat timestamp.
func (s *Session) SetHeartbeat() {
	s.lastHeartbeat.Store(time.Now().UnixNano())
}

// LastHeartbeat returns the time of the last heartbeat.
func (s *Session) LastHeartbeat() time.Time {
	return time.Unix(0, s.lastHeartbeat.Load())
}
