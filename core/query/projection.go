package query

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

// Compile-time check.
var _ lifecycle.Component = (*ProjectionEngine)(nil)

// Projection subscribes to bus events and updates a read model.
// Implementations must be idempotent and safe for concurrent calls.
type Projection interface {
	// Name identifies the projection for logging and metrics.
	Name() string

	// Subjects returns the bus subjects this projection subscribes to.
	Subjects() []string

	// Handle processes a single event and updates the read model.
	// Must be idempotent (use UPSERT). Returns error on transient failure.
	Handle(ctx context.Context, env event.RawEnvelope) error
}

// ProjectionEngine manages bus subscriptions for one or more Projection
// implementations. It implements lifecycle.Component.
//
// Ownership:
//   - ProjectionEngine owns all Bus subscriptions.
//   - Projection implementations own only the projection logic.
//   - Subscription lifecycle (Init, Start, Stop) is managed here.
type ProjectionEngine struct {
	bus         bus.BusPort
	projections []Projection
	subs        []bus.Subscription
	mu          sync.Mutex

	successCount int64
	failureCount int64
}

// NewProjectionEngine creates a ProjectionEngine.
func NewProjectionEngine(b bus.BusPort, projections []Projection) *ProjectionEngine {
	return &ProjectionEngine{
		bus:         b,
		projections: projections,
	}
}

// ---------------------------------------------------------------------------
// Lifecycle
// ---------------------------------------------------------------------------

// Name returns "projections" for the lifecycle manager.
func (e *ProjectionEngine) Name() string { return "projections" }

// Init validates the engine has a bus and at least one projection.
func (e *ProjectionEngine) Init(_ context.Context) error {
	if e.bus == nil {
		return fmt.Errorf("projections: bus is required")
	}
	if len(e.projections) == 0 {
		return fmt.Errorf("projections: at least one projection is required")
	}
	return nil
}

// Start subscribes all projections to their bus subjects.
func (e *ProjectionEngine) Start(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, proj := range e.projections {
		for _, subj := range proj.Subjects() {
			sub, err := e.bus.Subscribe(ctx, subj, e.handlerFor(proj))
			if err != nil {
				e.unsubscribeAll()
				return fmt.Errorf("projections: subscribe %s: %w", subj, err)
			}
			e.subs = append(e.subs, sub)
			log.Printf("projections: subscribed %s to %s", proj.Name(), subj)
		}
	}

	return nil
}

// Stop unsubscribes all projections.
func (e *ProjectionEngine) Stop(_ context.Context) error {
	e.unsubscribeAll()
	return nil
}

// Health reports whether the engine has active subscriptions.
func (e *ProjectionEngine) Health() lifecycle.Health {
	e.mu.Lock()
	active := len(e.subs) > 0
	e.mu.Unlock()
	if !active {
		return lifecycle.Health{Status: lifecycle.StatusDown, Since: time.Now()}
	}
	return lifecycle.Health{Status: lifecycle.StatusUp, Since: time.Now()}
}

// ---------------------------------------------------------------------------
// Internal
// ---------------------------------------------------------------------------

func (e *ProjectionEngine) unsubscribeAll() {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, sub := range e.subs {
		_ = sub.Unsubscribe()
	}
	e.subs = nil
}

func (e *ProjectionEngine) handlerFor(proj Projection) bus.MessageHandler {
	return func(ctx context.Context, msg bus.Message) error {
		env, err := event.Deserialize(msg.Data())
		if err != nil {
			_ = msg.Term()
			return nil
		}

		start := time.Now()

		if err := proj.Handle(ctx, env); err != nil {
			e.failureCount++
			log.Printf("projection: %s event=%s type=%s duration=%s status=failed error=%v",
				proj.Name(), env.ID, env.Type, time.Since(start), err)
			_ = msg.Nak()
			return nil
		}

		e.successCount++
		log.Printf("projection: %s event=%s type=%s duration=%s status=completed",
			proj.Name(), env.ID, env.Type, time.Since(start))
		_ = msg.Ack()
		return nil
	}
}
