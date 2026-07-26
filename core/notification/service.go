package notification

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/bus"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/event"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/lifecycle"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/orchestrator"
)

// Compile-time check.
var _ lifecycle.Component = (*Service)(nil)

// Service subscribes to intent.completed and intent.failed events,
// deserializes them, and dispatches notifications through the
// NotificationPort. It implements lifecycle.Component.
type Service struct {
	bus      bus.BusPort
	notifier NotificationPort

	subs []bus.Subscription
	mu   sync.Mutex
}

// NewService creates a new notification service.
func NewService(b bus.BusPort, notifier NotificationPort) *Service {
	return &Service{
		bus:      b,
		notifier: notifier,
	}
}

// ---------------------------------------------------------------------------
// Lifecycle
// ---------------------------------------------------------------------------

// Name returns "notification" for the lifecycle manager.
func (s *Service) Name() string { return "notification" }

// Init validates dependencies.
func (s *Service) Init(_ context.Context) error {
	if s.bus == nil {
		return fmt.Errorf("notification: %w: bus is required", ErrNotStarted)
	}
	if s.notifier == nil {
		return fmt.Errorf("notification: %w: notifier is required", ErrNotStarted)
	}
	return nil
}

// Start subscribes to intent.completed and intent.failed.
func (s *Service) Start(ctx context.Context) error {
	subjects := []string{
		"devos.intent.completed",
		"devos.intent.failed",
	}

	for _, subj := range subjects {
		sub, err := s.bus.Subscribe(ctx, subj, s.handleEvent)
		if err != nil {
			// Unwind on failure
			s.unsubscribeAll()
			return fmt.Errorf("notification: subscribe %s: %w", subj, err)
		}
		s.mu.Lock()
		s.subs = append(s.subs, sub)
		s.mu.Unlock()
	}

	log.Printf("notification: subscribed to %v", subjects)
	return nil
}

// Stop unsubscribes all subscriptions.
func (s *Service) Stop(_ context.Context) error {
	s.unsubscribeAll()
	return nil
}

// Health reports whether the service has active subscriptions.
func (s *Service) Health() lifecycle.Health {
	s.mu.Lock()
	active := len(s.subs) > 0
	s.mu.Unlock()
	if !active {
		return lifecycle.Health{Status: lifecycle.StatusDown, Since: time.Now()}
	}
	return lifecycle.Health{Status: lifecycle.StatusUp, Since: time.Now()}
}

// unsubscribeAll safely removes all subscriptions.
func (s *Service) unsubscribeAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, sub := range s.subs {
		_ = sub.Unsubscribe()
	}
	s.subs = nil
}

// ---------------------------------------------------------------------------
// Message handler
// ---------------------------------------------------------------------------

func (s *Service) handleEvent(ctx context.Context, msg bus.Message) error {
	env, err := event.Deserialize(msg.Data())
	if err != nil {
		_ = msg.Term()
		return nil
	}

	if env.Type != event.TypeIntentCompleted && env.Type != event.TypeIntentFailed {
		_ = msg.Ack()
		return nil
	}

	payload, err := event.UnmarshalPayload[orchestrator.IntentLifecyclePayload](env)
	if err != nil {
		_ = msg.Term()
		return nil
	}

	notification := NotificationPayload{
		IntentID: payload.IntentID,
		Type:     string(env.Type),
		Summary:  payload.Summary,
		Error:    payload.Error,
		OrgID:    env.OrgID,
		TraceID:  env.TraceID,
	}

	if err := s.notifier.Send(ctx, notification); err != nil {
		log.Printf("notification: send failed: %v", err)
		_ = msg.Nak()
		return nil
	}

	_ = msg.Ack()
	return nil
}
