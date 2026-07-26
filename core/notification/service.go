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

// Service subscribes to orchestration lifecycle events, dispatches
// notifications through a NotificationPort, and optionally renders and
// sends them through a ChannelProvider. It implements lifecycle.Component.
type Service struct {
	bus      bus.BusPort
	notifier NotificationPort
	renderer Renderer
	channel  ChannelProvider
	subs     []bus.Subscription
	subjects []string
	mu       sync.Mutex
}

// ServiceOption configures the Service.
type ServiceOption func(*Service)

// WithRenderer attaches a message renderer for channel-formatting.
func WithRenderer(r Renderer) ServiceOption {
	return func(s *Service) { s.renderer = r }
}

// WithChannel attaches a channel provider for outbound delivery.
func WithChannel(c ChannelProvider) ServiceOption {
	return func(s *Service) { s.channel = c }
}

// WithNotificationSubjects sets the event subjects to subscribe to.
// If empty, defaults are used.
func WithNotificationSubjects(subjects []string) ServiceOption {
	return func(s *Service) { s.subjects = subjects }
}

// NewService creates a new notification service.
// The notifier is always used; renderer and channel are optional.
func NewService(b bus.BusPort, notifier NotificationPort, opts ...ServiceOption) *Service {
	s := &Service{
		bus:      b,
		notifier: notifier,
		subjects: defaultSubjects(),
	}
	for _, fn := range opts {
		fn(s)
	}
	return s
}

// defaultSubjects returns the default set of event subjects to subscribe to.
func defaultSubjects() []string {
	return []string{
		"devos.intent.completed",
		"devos.intent.failed",
		"devos.task.status",
		"devos.task.failed",
		"devos.deploy.completed",
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

// Start subscribes to all configured event subjects.
func (s *Service) Start(ctx context.Context) error {
	for _, subj := range s.subjects {
		sub, err := s.bus.Subscribe(ctx, subj, s.handleEvent)
		if err != nil {
			s.unsubscribeAll()
			return fmt.Errorf("notification: subscribe %s: %w", subj, err)
		}
		s.mu.Lock()
		s.subs = append(s.subs, sub)
		s.mu.Unlock()
	}

	log.Printf("notification: subscribed to %d subjects", len(s.subjects))
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

	if !s.isRelevantType(env.Type) {
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

	// Always send through the NotificationPort (logging, etc.).
	if err := s.notifier.Send(ctx, notification); err != nil {
		log.Printf("notification: port send failed: %v", err)
	}

	// Optionally render and deliver through a channel adapter.
	if s.renderer != nil && s.channel != nil {
		channelMsg, renderErr := s.renderer.Render(ctx, notification)
		if renderErr != nil {
			log.Printf("notification: render failed: %v", renderErr)
		} else {
			if sendErr := s.channel.Send(ctx, channelMsg); sendErr != nil {
				log.Printf("notification: channel send failed: %v", sendErr)
			}
		}
	}

	_ = msg.Ack()
	return nil
}

// isRelevantType checks whether the event type should be handled.
func (s *Service) isRelevantType(typ event.EventType) bool {
	switch typ {
	case event.TypeIntentCompleted,
		event.TypeIntentFailed,
		event.TypeTaskStatus,
		event.TypeTaskFailed,
		event.TypeDeployCompleted:
		return true
	default:
		return false
	}
}
