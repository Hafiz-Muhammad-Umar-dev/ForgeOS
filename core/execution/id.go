package execution

import (
	"crypto/rand"
	"fmt"
	"time"
)

// newID returns a random UUID-like identifier.
func newID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("e-%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}