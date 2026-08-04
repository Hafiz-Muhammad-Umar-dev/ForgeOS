// Package ws implements the WebSocket endpoint for real-time collaboration.
// It handles connection upgrade, authentication handshake, and protocol
// message encoding/decoding.
package ws

import (
	"encoding/json"
	"fmt"
)

// ---------------------------------------------------------------------------
// Message types
// ---------------------------------------------------------------------------

const (
	msgAuth       = "auth"
	msgAuthOK     = "auth_ok"
	msgAuthFailed = "auth_failed"
	msgJoin       = "join"
	msgLeave      = "leave"
	msgUpdate     = "update"
	msgAwareness  = "awareness"
	msgPing       = "ping"
	msgPong       = "pong"
	msgError      = "error"
)

// Message is the JSON frame exchanged over WebSocket.
type Message struct {
	Type      string          `json:"type"`
	IntentID  string          `json:"intent_id,omitempty"`
	SessionID string          `json:"session_id,omitempty"`
	Data      json.RawMessage `json:"data,omitempty"`
	Error     string          `json:"error,omitempty"`
}

// ---------------------------------------------------------------------------
// Auth payloads
// ---------------------------------------------------------------------------

// AuthMessage is the first message a client must send after connecting.
type AuthMessage struct {
	Token string `json:"token"`
}

// AuthOKMessage is sent on successful authentication.
type AuthOKMessage struct {
	SessionID string `json:"session_id"`
}

// AuthFailedMessage is sent on authentication failure.
type AuthFailedMessage struct {
	Error string `json:"error"`
}

// ---------------------------------------------------------------------------
// Codec
// ---------------------------------------------------------------------------

// Encode serializes a message to JSON bytes.
func Encode(msg Message) ([]byte, error) {
	data, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("ws: encode: %w", err)
	}
	return data, nil
}

// Decode parses a JSON message from bytes.
func Decode(data []byte) (Message, error) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return Message{}, fmt.Errorf("ws: decode: %w", err)
	}
	if msg.Type == "" {
		return Message{}, fmt.Errorf("ws: message type is required")
	}
	return msg, nil
}

// ---------------------------------------------------------------------------
// Message constructors
// ---------------------------------------------------------------------------

// NewAuthOK creates an auth_ok message.
func NewAuthOK(sessionID string) Message {
	data, _ := json.Marshal(AuthOKMessage{SessionID: sessionID})
	return Message{Type: msgAuthOK, SessionID: sessionID, Data: data}
}

// NewAuthFailed creates an auth_failed message.
func NewAuthFailed(err string) Message {
	return Message{Type: msgAuthFailed, Error: err}
}

// NewPong creates a pong message (response to ping).
func NewPong() Message {
	return Message{Type: msgPong}
}

// NewError creates an error message.
func NewError(err string) Message {
	return Message{Type: msgError, Error: err}
}
