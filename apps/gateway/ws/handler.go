package ws

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/auth"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/collaboration"

	"github.com/coder/websocket"
)

// HandlerConfig configures the WebSocket handler.
type HandlerConfig struct {
	// ReadTimeout is the max time to wait for a client message.
	ReadTimeout time.Duration

	// WriteTimeout is the max time to write a message to the client.
	WriteTimeout time.Duration

	// PingInterval is how often the server sends pings.
	PingInterval time.Duration

	// MaxMessageSize is the maximum allowed message payload in bytes.
	MaxMessageSize int64
}

// DefaultHandlerConfig returns sensible defaults.
func DefaultHandlerConfig() HandlerConfig {
	return HandlerConfig{
		ReadTimeout:    60 * time.Second,
		WriteTimeout:   10 * time.Second,
		PingInterval:   30 * time.Second,
		MaxMessageSize: 1 << 20, // 1 MB
	}
}

// CollaborationHandler is the HTTP handler for the WebSocket collaboration
// endpoint at GET /v1/stream.
type CollaborationHandler struct {
	cfg     HandlerConfig
	auth    auth.AuthProvider
	manager *collaboration.Manager
}

// NewCollaborationHandler creates a new WebSocket collaboration handler.
func NewCollaborationHandler(
	cfg HandlerConfig,
	provider auth.AuthProvider,
	manager *collaboration.Manager,
) *CollaborationHandler {
	return &CollaborationHandler{
		cfg:     cfg,
		auth:    provider,
		manager: manager,
	}
}

// ServeHTTP handles the WebSocket upgrade and collaboration session.
// Pattern: GET /v1/stream
func (h *CollaborationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true, // Allow any origin during development
	})
	if err != nil {
		log.Printf("ws: upgrade failed: %v", err)
		return
	}
	defer c.Close(websocket.StatusNormalClosure, "connection closed")

	// Set read limit.
	c.SetReadLimit(h.cfg.MaxMessageSize)

	// Authentication handshake.
	claims, err := h.authenticate(r.Context(), c)
	if err != nil {
		log.Printf("ws: auth failed: %v", err)
		return
	}

	// Wait for the join message.
	intentID, err := h.waitForJoin(r.Context(), c, claims)
	if err != nil {
		return
	}

	// Create a session and join the room.
	sessionID := newSessionID()
	session := collaboration.NewSession(sessionID, claims.Subject, intentID, &wsConn{c: c})

	if err := h.manager.JoinRoom(r.Context(), intentID, session); err != nil {
		log.Printf("ws: join room %s: %v", intentID, err)
		writeWSError(c, "failed to join room")
		return
	}
	defer func() {
		_ = h.manager.LeaveRoom(context.Background(), intentID, sessionID)
	}()

	log.Printf("ws: session %s joined room %s (user=%s)", sessionID, intentID, claims.Subject)

	// Run the receive loop.
	h.receiveLoop(r.Context(), c, session, intentID)
}

// authenticate performs the authentication handshake.
func (h *CollaborationHandler) authenticate(ctx context.Context, c *websocket.Conn) (auth.Claims, error) {
	readFn := func() ([]byte, error) {
		_, data, err := c.Read(ctx)
		return data, err
	}
	writeFn := func(data []byte) error {
		return c.Write(ctx, websocket.MessageText, data)
	}

	return Authenticate(ctx, readFn, writeFn, h.auth)
}

// waitForJoin reads messages until a "join" message is received.
func (h *CollaborationHandler) waitForJoin(ctx context.Context, c *websocket.Conn, claims auth.Claims) (string, error) {
	for {
		_, data, err := c.Read(ctx)
		if err != nil {
			return "", err
		}

		msg, err := Decode(data)
		if err != nil {
			continue
		}

		switch msg.Type {
		case "join":
			if msg.IntentID == "" {
				writeWSError(c, "intent_id is required")
				continue
			}
			// Send auth_ok now that we have the intent.
			authOK, _ := Encode(NewAuthOK(claims.Subject))
			_ = c.Write(ctx, websocket.MessageText, authOK)
			return msg.IntentID, nil
		case "ping":
			pong, _ := Encode(NewPong())
			_ = c.Write(ctx, websocket.MessageText, pong)
		default:
			writeWSError(c, "expected join message")
		}
	}
}

// receiveLoop reads messages from the WebSocket and broadcasts updates to the
// room. It runs until the connection is closed or an error occurs.
func (h *CollaborationHandler) receiveLoop(
	ctx context.Context,
	c *websocket.Conn,
	session *collaboration.Session,
	intentID string,
) {
	for {
		msgType, data, err := c.Read(ctx)
		if err != nil {
			return
		}

		session.SetHeartbeat()

		msg, err := Decode(data)
		if err != nil {
			continue
		}

		switch msg.Type {
		case "update":
			// Broadcast the binary update to all other sessions in the room.
			payload := msg.Data
			if payload == nil {
				// If data is nil, use the raw message data as binary.
				if msgType == websocket.MessageBinary {
					payload = data
				}
			}
			if payload != nil {
				h.manager.Room(intentID).Broadcast(session.ID, payload)
			}
		case "awareness":
			// Relay awareness state to all other sessions.
			h.manager.Room(intentID).Broadcast(session.ID, data)
		case "ping":
			pong, _ := Encode(NewPong())
			_ = c.Write(ctx, websocket.MessageText, pong)
		case "leave":
			return
		}
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// wsConn adapts a *websocket.Conn to collaboration.SessionConn.
type wsConn struct {
	c *websocket.Conn
}

func (w *wsConn) Send(data []byte) error {
	return w.c.Write(context.Background(), websocket.MessageText, data)
}

func (w *wsConn) Receive() ([]byte, error) {
	_, data, err := w.c.Read(context.Background())
	return data, err
}

func (w *wsConn) Close() error {
	return w.c.Close(websocket.StatusNormalClosure, "session closed")
}

func writeWSError(c *websocket.Conn, msg string) {
	errMsg, _ := Encode(NewError(msg))
	_ = c.Write(context.Background(), websocket.MessageText, errMsg)
}

func newSessionID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "unknown"
	}
	return fmt.Sprintf("ws-%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
