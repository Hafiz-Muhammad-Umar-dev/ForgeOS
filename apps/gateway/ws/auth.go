package ws

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/auth"
)

// Authenticate performs the WebSocket authentication handshake. It reads the
// first client message, expects an "auth" type with a Bearer token, validates
// it via AuthProvider, and returns the claims on success.
//
// The caller passes a read function that reads one raw message from the
// WebSocket connection. authenticate handles the protocol framing internally.
func Authenticate(
	ctx context.Context,
	readMsg func() ([]byte, error),
	writeMsg func([]byte) error,
	provider auth.AuthProvider,
) (auth.Claims, error) {
	// Read the first client message.
	raw, err := readMsg()
	if err != nil {
		return auth.Claims{}, fmt.Errorf("ws auth: read: %w", err)
	}

	var clientMsg struct {
		Auth struct {
			Token string `json:"token"`
		} `json:"auth"`
	}

	// Accept both { "type": "auth", "token": "..." } and
	// { "auth": { "token": "..." } } formats.
	if err := json.Unmarshal(raw, &clientMsg); err != nil || clientMsg.Auth.Token == "" {
		// Try flat format.
		var flatMsg struct {
			Type  string `json:"type"`
			Token string `json:"token"`
		}
		if err2 := json.Unmarshal(raw, &flatMsg); err2 != nil || flatMsg.Token == "" {
			errData, _ := Encode(NewAuthFailed("invalid auth message"))
			_ = writeMsg(errData)
			return auth.Claims{}, fmt.Errorf("ws auth: invalid message format")
		}
		clientMsg.Auth.Token = flatMsg.Token
	}

	// Validate the token.
	claims, err := provider.Authenticate(ctx, clientMsg.Auth.Token)
	if err != nil {
		errData, _ := Encode(NewAuthFailed(err.Error()))
		_ = writeMsg(errData)
		return auth.Claims{}, fmt.Errorf("ws auth: %w", err)
	}

	return claims, nil
}
