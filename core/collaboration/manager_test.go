package collaboration

import (
	"context"
	"sync"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// PipeConn — in-memory SessionConn for tests
// ---------------------------------------------------------------------------

type pipeConn struct {
	mu     sync.Mutex
	cond   *sync.Cond
	buf    [][]byte
	closed bool
}

func newPipeConn() *pipeConn {
	c := &pipeConn{}
	c.cond = sync.NewCond(&c.mu)
	return c
}

func (p *pipeConn) Send(data []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return nil
	}
	p.buf = append(p.buf, data)
	p.cond.Signal()
	return nil
}

func (p *pipeConn) Receive() ([]byte, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for len(p.buf) == 0 && !p.closed {
		p.cond.Wait()
	}
	if p.closed && len(p.buf) == 0 {
		return nil, nil
	}
	data := p.buf[0]
	p.buf = p.buf[1:]
	return data, nil
}

func (p *pipeConn) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closed = true
	p.cond.Broadcast()
	return nil
}

func newTestSession(id, userID, intentID string) *Session {
	return NewSession(id, userID, intentID, newPipeConn())
}

// ---------------------------------------------------------------------------
// Room tests
// ---------------------------------------------------------------------------

func TestRoomNew(t *testing.T) {
	r := NewRoom("intent-1")
	if r.IntentID() != "intent-1" {
		t.Errorf("intentID=%s", r.IntentID())
	}
	if r.SessionCount() != 0 {
		t.Errorf("count=%d", r.SessionCount())
	}
}

func TestRoomAddSession(t *testing.T) {
	r := NewRoom("intent-1")
	s1 := newTestSession("s1", "u1", "intent-1")

	if err := r.AddSession(s1); err != nil {
		t.Fatalf("add: %v", err)
	}
	if r.SessionCount() != 1 {
		t.Errorf("count=%d", r.SessionCount())
	}
}

func TestRoomAddDuplicateSession(t *testing.T) {
	r := NewRoom("intent-1")
	s1 := newTestSession("s1", "u1", "intent-1")

	_ = r.AddSession(s1)
	err := r.AddSession(s1)
	if err == nil {
		t.Fatal("expected error for duplicate session")
	}
}

func TestRoomRemoveSession(t *testing.T) {
	r := NewRoom("intent-1")
	s1 := newTestSession("s1", "u1", "intent-1")

	_ = r.AddSession(s1)
	r.RemoveSession("s1")

	if r.SessionCount() != 0 {
		t.Errorf("count=%d", r.SessionCount())
	}
}

func TestRoomBroadcast(t *testing.T) {
	r := NewRoom("intent-1")
	sender := newTestSession("sender", "u1", "intent-1")
	recv := newTestSession("recv", "u2", "intent-1")

	_ = r.AddSession(sender)
	_ = r.AddSession(recv)

	n := r.Broadcast("sender", []byte("hello"))
	if n != 1 {
		t.Errorf("broadcast count=%d", n)
	}

	// Verify the recipient got the message.
	data, err := recv.Receive()
	if err != nil {
		t.Fatalf("receive: %v", err)
	}
	if string(data) != "hello" {
		t.Errorf("data=%s", string(data))
	}
}

func TestRoomBroadcastSkipsSender(t *testing.T) {
	r := NewRoom("intent-1")
	s1 := newTestSession("s1", "u1", "intent-1")
	s2 := newTestSession("s2", "u2", "intent-1")
	s3 := newTestSession("s3", "u3", "intent-1")

	_ = r.AddSession(s1)
	_ = r.AddSession(s2)
	_ = r.AddSession(s3)

	n := r.Broadcast("s1", []byte("update"))
	if n != 2 {
		t.Errorf("broadcast count=%d (expected 2)", n)
	}
}

func TestRoomSessions(t *testing.T) {
	r := NewRoom("intent-1")
	_ = r.AddSession(newTestSession("s1", "u1", "intent-1"))
	_ = r.AddSession(newTestSession("s2", "u2", "intent-1"))

	ids := r.Sessions()
	if len(ids) != 2 {
		t.Errorf("sessions=%d", len(ids))
	}
}

func TestRoomClose(t *testing.T) {
	r := NewRoom("intent-1")
	s1 := newTestSession("s1", "u1", "intent-1")
	_ = r.AddSession(s1)

	r.Close(context.Background())

	if r.SessionCount() != 0 {
		t.Errorf("count after close=%d", r.SessionCount())
	}
	if !s1.IsClosed() {
		t.Error("session should be closed")
	}
}

func TestRoomCloseEmpty(t *testing.T) {
	r := NewRoom("intent-1")
	r.Close(context.Background())
	// Should not panic.
}

// ---------------------------------------------------------------------------
// Session tests
// ---------------------------------------------------------------------------

func TestSessionSend(t *testing.T) {
	conn := newPipeConn()
	s := NewSession("s1", "u1", "intent-1", conn)

	if !s.Send([]byte("hello")) {
		t.Fatal("send returned false")
	}

	data, err := conn.Receive()
	if err != nil {
		t.Fatalf("receive: %v", err)
	}
	if string(data) != "hello" {
		t.Errorf("data=%s", string(data))
	}
}

func TestSessionSendAfterClose(t *testing.T) {
	conn := newPipeConn()
	s := NewSession("s1", "u1", "intent-1", conn)

	_ = s.Close(context.Background())
	if s.Send([]byte("data")) {
		t.Error("send should return false after close")
	}
}

func TestSessionClose(t *testing.T) {
	conn := newPipeConn()
	s := NewSession("s1", "u1", "intent-1", conn)

	if s.IsClosed() {
		t.Error("should not be closed initially")
	}

	_ = s.Close(context.Background())
	if !s.IsClosed() {
		t.Error("should be closed")
	}

	// Double close should not panic.
	_ = s.Close(context.Background())
}

func TestSessionHeartbeat(t *testing.T) {
	conn := newPipeConn()
	s := NewSession("s1", "u1", "intent-1", conn)

	before := s.LastHeartbeat()
	if before.IsZero() {
		t.Error("zero heartbeat before first set")
	}

	s.SetHeartbeat()
	after := s.LastHeartbeat()
	if !after.After(before) {
		t.Error("heartbeat should advance")
	}
}

func TestSessionID(t *testing.T) {
	s := NewSession("s1", "u1", "intent-1", newPipeConn())
	if s.ID != "s1" {
		t.Errorf("id=%s", s.ID)
	}
	if s.UserID != "u1" {
		t.Errorf("userID=%s", s.UserID)
	}
	if s.IntentID != "intent-1" {
		t.Errorf("intentID=%s", s.IntentID)
	}
}

// ---------------------------------------------------------------------------
// Manager tests
// ---------------------------------------------------------------------------

func TestManagerNew(t *testing.T) {
	m := NewManager()
	if m.RoomCount() != 0 {
		t.Errorf("count=%d", m.RoomCount())
	}
}

func TestManagerJoinRoom(t *testing.T) {
	m := NewManager()
	s := newTestSession("s1", "u1", "intent-1")

	if err := m.JoinRoom(context.Background(), "intent-1", s); err != nil {
		t.Fatalf("join: %v", err)
	}

	room := m.Room("intent-1")
	if room == nil {
		t.Fatal("room should exist")
	}
	if room.SessionCount() != 1 {
		t.Errorf("count=%d", room.SessionCount())
	}
}

func TestManagerJoinRoomCreatesNew(t *testing.T) {
	m := NewManager()
	s := newTestSession("s1", "u1", "intent-1")

	_ = m.JoinRoom(context.Background(), "intent-1", s)
	_ = m.JoinRoom(context.Background(), "intent-2", s)

	if m.RoomCount() != 2 {
		t.Errorf("room count=%d", m.RoomCount())
	}
}

func TestManagerLeaveRoom(t *testing.T) {
	m := NewManager()
	s := newTestSession("s1", "u1", "intent-1")

	_ = m.JoinRoom(context.Background(), "intent-1", s)

	if err := m.LeaveRoom(context.Background(), "intent-1", "s1"); err != nil {
		t.Fatalf("leave: %v", err)
	}

	// Room should be removed when empty.
	if m.Room("intent-1") != nil {
		t.Error("room should be nil after last session leaves")
	}
}

func TestManagerLeaveRoomNotFound(t *testing.T) {
	m := NewManager()
	err := m.LeaveRoom(context.Background(), "nonexistent", "s1")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestManagerRoomNotFound(t *testing.T) {
	m := NewManager()
	if r := m.Room("nonexistent"); r != nil {
		t.Error("expected nil room")
	}
}

func TestManagerClose(t *testing.T) {
	m := NewManager()
	_ = m.JoinRoom(context.Background(), "intent-1", newTestSession("s1", "u1", "intent-1"))
	_ = m.JoinRoom(context.Background(), "intent-2", newTestSession("s2", "u2", "intent-2"))

	if err := m.Close(context.Background()); err != nil {
		t.Fatalf("close: %v", err)
	}
	if m.RoomCount() != 0 {
		t.Errorf("count after close=%d", m.RoomCount())
	}
}

func TestManagerConcurrentJoins(t *testing.T) {
	m := NewManager()
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			id := "s" + string(rune('0'+n))
			s := newTestSession(id, "u"+id, "intent-concurrent")
			_ = m.JoinRoom(context.Background(), "intent-concurrent", s)
		}(i)
	}
	wg.Wait()

	room := m.Room("intent-concurrent")
	if room == nil {
		t.Fatal("room should exist")
	}
	if room.SessionCount() != 10 {
		t.Errorf("expected 10 sessions, got %d", room.SessionCount())
	}
}

func TestManagerConcurrentBroadcast(t *testing.T) {
	m := NewManager()
	ctx := context.Background()

	// Create a room with 5 sessions.
	for i := 0; i < 5; i++ {
		id := "s" + string(rune('0'+i))
		s := newTestSession(id, "u"+id, "intent-bc")
		_ = m.JoinRoom(ctx, "intent-bc", s)
	}

	room := m.Room("intent-bc")
	if room == nil {
		t.Fatal("room should exist")
	}

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			id := "s" + string(rune('0'+n))
			room.Broadcast(id, []byte("hello"))
		}(i)
	}
	wg.Wait()
}

func TestSessionReceiveBlocking(t *testing.T) {
	conn := newPipeConn()
	s := NewSession("s1", "u1", "intent-1", conn)

	// Send in background after a short delay.
	go func() {
		time.Sleep(10 * time.Millisecond)
		conn.Send([]byte("data"))
	}()

	data, err := s.Receive()
	if err != nil {
		t.Fatalf("receive: %v", err)
	}
	if string(data) != "data" {
		t.Errorf("data=%s", string(data))
	}
}

func TestSessionReceiveAfterClose(t *testing.T) {
	conn := newPipeConn()
	s := NewSession("s1", "u1", "intent-1", conn)

	_ = s.Close(context.Background())

	_, err := s.Receive()
	if err == nil {
		t.Fatal("expected error after close")
	}
}
