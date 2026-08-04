package bus

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/lifecycle"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Compile-time check: *NatsBus implements BusPort.
var _ BusPort = (*NatsBus)(nil)

// NatsBus is a NATS JetStream adapter implementing BusPort and
// lifecycle.Component.
//
// It owns a single NATS connection with JetStream and manages stream
// creation, publishing, and subscribing.
type NatsBus struct {
	cfg NatsConfig

	// connection
	conn        *nats.Conn
	js          jetstream.JetStream
	connCreated bool

	// stream lookup: subject prefix -> stream name
	streamIndex map[string]string

	// subscriptions
	subsMu sync.RWMutex
	subs   []*natsSub

	// state
	closed  atomic.Bool
	startMu sync.Mutex
}

// natsSub wraps a JetStream consumer subscription.
type natsSub struct {
	subject string
	msgCtx  jetstream.MessagesContext
}

// Unsubscribe stops the message iterator, which causes the consume loop to
// exit cleanly.
func (s *natsSub) Unsubscribe() error {
	if s.msgCtx != nil {
		s.msgCtx.Stop()
	}
	return nil
}

// Subject returns the subject pattern.
func (s *natsSub) Subject() string { return s.subject }

// natsMsg wraps a JetStream message as a bus.Message.
type natsMsg struct {
	subject string
	data    []byte
	msg     jetstream.Msg
}

func (m *natsMsg) Subject() string { return m.subject }
func (m *natsMsg) Data() []byte    { return m.data }
func (m *natsMsg) Ack() error {
	if m.msg != nil {
		return m.msg.Ack()
	}
	return nil
}
func (m *natsMsg) Nak() error {
	if m.msg != nil {
		return m.msg.Nak()
	}
	return nil
}
func (m *natsMsg) Term() error {
	if m.msg != nil {
		return m.msg.Term()
	}
	return nil
}

// NewNatsBus creates a new NATS bus adapter with the given options.
func NewNatsBus(opts ...NatsOption) *NatsBus {
	o := &natsOptions{config: DefaultNatsConfig()}
	for _, fn := range opts {
		fn(o)
	}
	return &NatsBus{
		cfg: o.config,
	}
}

// Name returns the component name for the lifecycle manager.
func (b *NatsBus) Name() string { return "bus" }

// Init validates configuration. Part of lifecycle.Component.
func (b *NatsBus) Init(ctx context.Context) error {
	if b.cfg.URL == "" {
		return errors.New("bus: NATS URL is required")
	}
	return nil
}

// Start connects to NATS and sets up JetStream streams. Part of lifecycle.Component.
func (b *NatsBus) Start(ctx context.Context) error {
	return b.Connect(ctx)
}

// Stop gracefully closes the bus connection. Part of lifecycle.Component.
func (b *NatsBus) Stop(ctx context.Context) error {
	return b.Close(ctx)
}

// Health returns the component health as required by lifecycle.Component.
func (b *NatsBus) Health() lifecycle.Health {
	if b.closed.Load() || b.conn == nil || !b.conn.IsConnected() {
		return lifecycle.Health{Status: lifecycle.StatusDown, Since: time.Now()}
	}
	return lifecycle.Health{Status: lifecycle.StatusUp, Since: time.Now()}
}

// Connect establishes a NATS connection and initializes JetStream.
func (b *NatsBus) Connect(ctx context.Context) error {
	b.startMu.Lock()
	defer b.startMu.Unlock()

	if b.connCreated && b.conn != nil && b.conn.IsConnected() {
		return nil
	}

	nc, err := nats.Connect(b.cfg.URL,
		nats.Name(b.cfg.Name),
		nats.Timeout(b.cfg.Timeout),
		nats.ReconnectWait(b.cfg.ReconnectWait),
		nats.MaxReconnects(b.cfg.MaxReconnects),
		nats.DisconnectErrHandler(func(_ *nats.Conn, _ error) {}),
		nats.ReconnectHandler(func(_ *nats.Conn) {}),
		nats.ClosedHandler(func(_ *nats.Conn) {
			b.closed.Store(true)
		}),
	)
	if err != nil {
		return fmt.Errorf("bus: connect to %s: %w", b.cfg.URL, err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return fmt.Errorf("bus: jetstream: %w", err)
	}

	b.conn = nc
	b.js = js
	b.connCreated = true
	b.closed.Store(false)

	if err := b.initStreams(ctx); err != nil {
		nc.Close()
		b.connCreated = false
		return fmt.Errorf("bus: init streams: %w", err)
	}

	return nil
}

// initStreams builds the subject→stream index and ensures all configured
// streams exist on the server.
func (b *NatsBus) initStreams(ctx context.Context) error {
	b.streamIndex = make(map[string]string, len(b.cfg.StreamConfigs))

	for _, sc := range b.cfg.StreamConfigs {
		for _, subj := range sc.Subjects {
			if strings.HasSuffix(subj, ".>") {
				prefix := strings.TrimSuffix(subj, ".>") + "."
				b.streamIndex[prefix] = sc.Name
			} else {
				b.streamIndex[subj] = sc.Name
			}
		}

		streamCfg := b.toStreamConfig(sc)
		if _, err := b.js.CreateOrUpdateStream(ctx, streamCfg); err != nil {
			return fmt.Errorf("bus: create stream %s: %w", sc.Name, err)
		}
	}

	return nil
}

// toStreamConfig converts a StreamConfig to a NATS stream configuration.
func (b *NatsBus) toStreamConfig(sc StreamConfig) jetstream.StreamConfig {
	storage := jetstream.FileStorage
	if sc.Storage == "memory" {
		storage = jetstream.MemoryStorage
	}

	retention := jetstream.LimitsPolicy
	switch sc.Retention {
	case "workQueue":
		retention = jetstream.WorkQueuePolicy
	case "interest":
		retention = jetstream.InterestPolicy
	}

	replicas := sc.Replicas
	if replicas < 1 {
		replicas = 1
	}

	return jetstream.StreamConfig{
		Name:              sc.Name,
		Subjects:          sc.Subjects,
		MaxAge:            sc.MaxAge,
		Storage:           storage,
		Retention:         retention,
		Replicas:          replicas,
		MaxMsgsPerSubject: 1_000_000,
	}
}

// resolveStream maps a subject to the stream name by prefix matching.
// Falls back to finding an existing stream on the server when no match
// is found in the local index.
func (b *NatsBus) resolveStream(ctx context.Context, subject string) (string, error) {
	// Try exact match first.
	if name, ok := b.streamIndex[subject]; ok {
		return name, nil
	}

	// Try prefix match: "devos.intent.created" -> match on "devos.intent."
	for prefix, name := range b.streamIndex {
		if strings.HasPrefix(subject, prefix) {
			return name, nil
		}
	}

	// No configured match. Try to find an existing stream on the server.
	if name, err := b.js.StreamNameBySubject(ctx, subject); err == nil && name != "" {
		return name, nil
	}

	// Auto-create an on-the-fly stream.
	parts := strings.SplitN(subject, ".", 3)
	domain := "unknown"
	if len(parts) >= 2 {
		domain = parts[1]
	}
	streamName := "STREAM_" + strings.ToUpper(domain)
	subjPattern := "devos." + domain + ".>"

	cfg := jetstream.StreamConfig{
		Name:      streamName,
		Subjects:  []string{subjPattern},
		MaxAge:    7 * 24 * time.Hour,
		Storage:   jetstream.FileStorage,
		Retention: jetstream.LimitsPolicy,
		Replicas:  1,
	}
	if _, err := b.js.CreateOrUpdateStream(ctx, cfg); err != nil {
		// Overlap with an existing stream — find it by subject.
		if name, lookupErr := b.js.StreamNameBySubject(ctx, subject); lookupErr == nil && name != "" {
			return name, nil
		}
		return "", fmt.Errorf("bus: auto-create stream for %s: %w", subject, err)
	}

	return streamName, nil
}

// Publish sends data bytes to the given subject via JetStream.
func (b *NatsBus) Publish(ctx context.Context, subject string, data []byte) error {
	if b.closed.Load() || b.conn == nil || !b.conn.IsConnected() {
		return fmt.Errorf("publish %s: %w", subject, ErrNotConnected)
	}

	stream, err := b.resolveStream(ctx, subject)
	if err != nil {
		return fmt.Errorf("publish %s: %w", subject, err)
	}

	if _, err = b.js.Publish(ctx, subject, data, jetstream.WithExpectStream(stream)); err != nil {
		return fmt.Errorf("publish %s: %w (%s)", subject, ErrPublishFailed, err)
	}
	return nil
}

// Subscribe registers a handler for messages on the given subject pattern.
func (b *NatsBus) Subscribe(ctx context.Context, subject string, handler MessageHandler) (Subscription, error) {
	if b.closed.Load() || b.conn == nil || !b.conn.IsConnected() {
		return nil, fmt.Errorf("subscribe %s: %w", subject, ErrNotConnected)
	}

	stream, err := b.resolveStream(ctx, subject)
	if err != nil {
		return nil, fmt.Errorf("subscribe %s: %w", subject, err)
	}

	cons, err := b.js.CreateOrUpdateConsumer(ctx, stream, jetstream.ConsumerConfig{
		Name:          b.consumerName(subject),
		FilterSubject: subject,
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxDeliver:    3,
	})
	if err != nil {
		return nil, fmt.Errorf("subscribe %s: %w", subject, ErrSubscribeFailed)
	}

	msgCtx, err := cons.Messages()
	if err != nil {
		return nil, fmt.Errorf("subscribe %s: messages: %w", subject, err)
	}

	sub := &natsSub{subject: subject, msgCtx: msgCtx}

	b.subsMu.Lock()
	b.subs = append(b.subs, sub)
	b.subsMu.Unlock()

	go b.consumeLoop(ctx, msgCtx, handler)

	return sub, nil
}

// consumeLoop reads messages from the JetStream consumer channel and calls
// the handler for each.
func (b *NatsBus) consumeLoop(
	ctx context.Context,
	msgCtx jetstream.MessagesContext,
	handler MessageHandler,
) {
	defer msgCtx.Stop()

	for {
		msg, err := msgCtx.Next()
		if err != nil {
			if errors.Is(err, nats.ErrConnectionClosed) ||
				errors.Is(err, jetstream.ErrMsgIteratorClosed) ||
				errors.Is(err, context.Canceled) {
				return
			}
			continue
		}

		bm := &natsMsg{
			subject: msg.Subject(),
			data:    msg.Data(),
			msg:     msg,
		}

		if err := handler(ctx, bm); err != nil {
			_ = bm.Nak()
		}
	}
}

// consumerName generates a deterministic consumer name from a subject pattern.
func (b *NatsBus) consumerName(subject string) string {
	s := strings.NewReplacer(".", "_", ">", "all", "*", "any").Replace(subject)
	return b.cfg.Name + "_" + s
}

// Close gracefully shuts down the NATS connection.
func (b *NatsBus) Close(ctx context.Context) error {
	b.startMu.Lock()
	defer b.startMu.Unlock()

	if !b.connCreated || b.conn == nil {
		return nil
	}

	b.subsMu.Lock()
	for _, s := range b.subs {
		_ = s.Unsubscribe()
	}
	b.subs = nil
	b.subsMu.Unlock()

	b.closed.Store(true)
	b.conn.Close()
	b.conn = nil
	b.connCreated = false
	return nil
}

// IsConnected reports whether the bus connection is active.
func (b *NatsBus) IsConnected() bool {
	return !b.closed.Load() && b.conn != nil && b.conn.IsConnected()
}
