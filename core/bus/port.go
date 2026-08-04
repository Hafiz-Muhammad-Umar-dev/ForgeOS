// Package bus defines the DevOS message-bus port abstraction (BusPort) and
// its NATS JetStream adapter. It follows the ports/adapters (hexagonal)
// architecture: core/domain uses BusPort, and the nats_adapter.go
// implementation satisfies it without leaking NATS SDK types into domain code.
//
// Sprint 0 scope (see planning/03-build-order.md):
//   - Connect / Close (lifecycle management)
//   - Publish (JetStream publish with at-least-once delivery)
//   - Subscribe (JetStream push consumer with handler callback)
//   - Event envelope serialization (core/event/serde.go)
//   - Subject conventions (core/event/envelope.go Subject type)
//   - Connection and error handling
//
// Excluded from Sprint 0 (deferred to later components):
//   - Request / Reply
//   - HITL flow support
//   - Replay / stream management
//   - KV / Object Store
package bus

import "context"

// BusPort is the core abstraction for the DevOS message bus. All cross-context
// communication flows through this interface. Implementations provide the
// transport (NATS JetStream, in-memory for tests, etc.).
type BusPort interface {
	// Connect establishes a connection to the message bus. It is safe to call
	// multiple times (idempotent when already connected).
	Connect(ctx context.Context) error

	// Publish sends a message to the given subject with at-least-once delivery
	// semantics. data is typically a serialized RawEnvelope (JSON bytes).
	Publish(ctx context.Context, subject string, data []byte) error

	// Subscribe registers a handler for messages matching the subject pattern.
	// The returned Subscription can be used to unsubscribe. The handler is
	// called sequentially for each message; implementations must not call the
	// handler concurrently for the same subscription.
	Subscribe(ctx context.Context, subject string, handler MessageHandler) (Subscription, error)

	// Close gracefully shuts down the bus connection and releases all resources.
	// It is safe to call multiple times.
	Close(ctx context.Context) error

	// IsConnected reports whether the bus is currently connected.
	IsConnected() bool
}

// MessageHandler receives a single message from a subscription. Implementations
// must Ack or Nak the message to signal successful (or failed) processing.
type MessageHandler func(ctx context.Context, msg Message) error

// Message represents a single message received from a subscription.
type Message interface {
	// Subject returns the NATS subject the message was published to.
	Subject() string

	// Data returns the message payload bytes.
	Data() []byte

	// Ack acknowledges successful processing of the message.
	Ack() error

	// Nak marks the message as not acknowledged, triggering redelivery.
	Nak() error

	// Term marks the message as failed permanently (dead-letter).
	Term() error
}

// Subscription represents an active subscription to a subject pattern.
type Subscription interface {
	// Unsubscribe cancels the subscription. It is safe to call multiple times.
	Unsubscribe() error

	// Subject returns the subject pattern this subscription matches.
	Subject() string
}
