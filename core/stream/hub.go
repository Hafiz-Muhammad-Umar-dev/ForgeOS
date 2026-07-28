package stream

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/bus"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/event"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/lifecycle"
)

// Compile-time checks.
var (
	_ Streamer          = (*Hub)(nil)
	_ lifecycle.Component = (*Hub)(nil)
)

// subscriber represents a single streaming connection.
type subscriber struct {
	id       string
	intentID string
	ch       chan event.RawEnvelope
	closeOnce sync.Once
	closed    bool
	mu        sync.Mutex
}

func (s *subscriber) safeClose() {
	s.closeOnce.Do(func() {
		s.mu.Lock()
		s.closed = true
		s.mu.Unlock()
		close(s.ch)
	})
}

func (s *subscriber) isClosed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}

// Hub implements Streamer with a single dispatcher goroutine that processes
// bus events and fans them out to all matching subscribers. Event ordering
// is deterministic per subscriber.
type Hub struct {
	bus      bus.BusPort
	subjects []string

	mu          sync.Mutex
	subscribers map[string]map[string]*subscriber // intentID → subID → sub

	sub      bus.Subscription
	events   chan event.RawEnvelope
	started  bool
	done     chan struct{}
	cancel   context.CancelFunc // cancels the dispatch loop context
}

// NewHub creates a new streaming hub.
func NewHub(b bus.BusPort, subjects []string) *Hub {
	if subjects == nil {
		subjects = defaultSubjects()
	}
	return &Hub{
		bus:         b,
		subjects:    subjects,
		subscribers: make(map[string]map[string]*subscriber),
		events:      make(chan event.RawEnvelope, 256),
	}
}

func defaultSubjects() []string {
	return []string{
		"devos.intent.completed",
		"devos.intent.failed",
		"devos.task.status",
		"devos.task.failed",
		"devos.node.status",
		"devos.node.failed",
		"devos.plan.started",
		"devos.plan.proposed",
	}
}

// ---------------------------------------------------------------------------
// Streamer interface
// ---------------------------------------------------------------------------

// Subscribe registers a subscriber for events on the given intent.
func (h *Hub) Subscribe(_ context.Context, intentID, subID string) (<-chan event.RawEnvelope, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.subscribers[intentID]; !ok {
		h.subscribers[intentID] = make(map[string]*subscriber)
	}

	if _, ok := h.subscribers[intentID][subID]; ok {
		return nil, fmt.Errorf("stream: subscriber %q already exists for intent %q", subID, intentID)
	}

	sub := &subscriber{
		id:       subID,
		intentID: intentID,
		ch:       make(chan event.RawEnvelope, 64),
	}
	h.subscribers[intentID][subID] = sub

	return sub.ch, nil
}

// Unsubscribe removes a subscriber and cleans up resources.
func (h *Hub) Unsubscribe(intentID, subID string) error {
	h.mu.Lock()
	subMap, ok := h.subscribers[intentID]
	if !ok {
		h.mu.Unlock()
		return fmt.Errorf("stream: no subscribers for intent %q", intentID)
	}
	sub, ok := subMap[subID]
	if !ok {
		h.mu.Unlock()
		return fmt.Errorf("stream: subscriber %q not found for intent %q", subID, intentID)
	}
	delete(subMap, subID)
	if len(subMap) == 0 {
		delete(h.subscribers, intentID)
	}
	h.mu.Unlock()

	sub.safeClose()
	return nil
}

// SubscriberCount returns the number of active subscribers for an intent.
func (h *Hub) SubscriberCount(intentID string) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	subMap, ok := h.subscribers[intentID]
	if !ok {
		return 0
	}
	return len(subMap)
}

// ---------------------------------------------------------------------------
// Lifecycle
// ---------------------------------------------------------------------------

// Name returns "stream" for the lifecycle manager.
func (h *Hub) Name() string { return "stream" }

// Init validates dependencies.
func (h *Hub) Init(_ context.Context) error {
	if h.bus == nil {
		return fmt.Errorf("stream: bus is required")
	}
	return nil
}

// Start subscribes to bus events and begins the dispatch loop.
func (h *Hub) Start(ctx context.Context) error {
	sub, err := h.bus.Subscribe(ctx, h.subjects[0], h.handleBusEvent)
	if err != nil {
		return fmt.Errorf("stream: subscribe: %w", err)
	}

	// Create a cancellable context so Stop() can signal the dispatch loop
	// to exit even when the caller's context is never cancelled.
	dispatchCtx, cancel := context.WithCancel(ctx)

	h.mu.Lock()
	h.sub = sub
	h.started = true
	h.done = make(chan struct{})
	h.cancel = cancel
	h.mu.Unlock()

	go h.dispatchLoop(dispatchCtx)

	log.Printf("stream: hub started with %d subjects", len(h.subjects))
	return nil
}

// Stop shuts down the hub and all subscribers.
func (h *Hub) Stop(_ context.Context) error {
	h.mu.Lock()
	if !h.started {
		h.mu.Unlock()
		return nil
	}
	h.started = false
	if h.sub != nil {
		_ = h.sub.Unsubscribe()
		h.sub = nil
	}
	cancel := h.cancel
	h.cancel = nil
	h.mu.Unlock()

	// Cancel the dispatch loop context so it exits even when no bus events
	// arrive (prevents the deadlock where Stop waits forever on <-h.done).
	if cancel != nil {
		cancel()
	}

	// Wait for dispatch loop to finish.
	<-h.done

	// Close all subscriber channels.
	h.mu.Lock()
	for intentID, subMap := range h.subscribers {
		for subID, sub := range subMap {
			sub.safeClose()
			delete(subMap, subID)
		}
		delete(h.subscribers, intentID)
	}
	h.mu.Unlock()

	return nil
}

// Health reports whether the hub is running.
func (h *Hub) Health() lifecycle.Health {
	h.mu.Lock()
	s := h.started
	h.mu.Unlock()
	if !s {
		return lifecycle.Health{Status: lifecycle.StatusDown, Since: time.Now()}
	}
	return lifecycle.Health{Status: lifecycle.StatusUp, Since: time.Now()}
}

// ---------------------------------------------------------------------------
// Dispatch
// ---------------------------------------------------------------------------

// handleBusEvent is called by the bus adapter for each event (sequential).
func (h *Hub) handleBusEvent(ctx context.Context, msg bus.Message) error {
	env, err := event.Deserialize(msg.Data())
	if err != nil {
		_ = msg.Term()
		return nil
	}

	// Send to dispatch loop for fan-out.
	select {
	case h.events <- env:
	case <-ctx.Done():
		return nil
	}

	_ = msg.Ack()
	return nil
}

// dispatchLoop runs in a single goroutine. It reads events from the bus
// channel and fans them out to matching subscribers. This design ensures
// deterministic ordering for all subscribers with no concurrent fan-out.
func (h *Hub) dispatchLoop(ctx context.Context) {
	defer close(h.done)

	for {
		select {
		case <-ctx.Done():
			return
		case env := <-h.events:
			h.fanOut(env)
		}
	}
}

// fanOut sends an event to all subscribers whose intentID matches the event.
// Non-blocking send: if a subscriber's channel is full, the subscriber is
// evicted (slow consumer).
func (h *Hub) fanOut(env event.RawEnvelope) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// The intent ID may be in the envelope's ID for intent events.
	// For node events, we match based on the event type and envelope.
	intentID := extractIntentID(env)
	if intentID == "" {
		return
	}

	subMap, ok := h.subscribers[intentID]
	if !ok {
		return
	}

	for subID, sub := range subMap {
		if sub.isClosed() {
			delete(subMap, subID)
			continue
		}

		select {
		case sub.ch <- env:
		default:
			// Slow consumer — evict.
			log.Printf("stream: evicting slow subscriber %s for intent %s", subID, intentID)
			delete(subMap, subID)
			sub.safeClose()
		}
	}
}

// extractIntentID attempts to determine the intent ID from an event envelope.
// For intent events, the envelope ID is the intent ID.
// For other events, returns empty (not matched to a specific intent stream).
func extractIntentID(env event.RawEnvelope) string {
	switch env.Type {
	case event.TypeIntentCompleted, event.TypeIntentFailed,
		event.TypePlanStarted, event.TypePlanProposed,
		event.TypePlanApproved, event.TypePlanRejected:
		return env.ID
	default:
		return ""
	}
}
