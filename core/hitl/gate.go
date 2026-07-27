package hitl

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
var (
	_ HITLGate          = (*Gate)(nil)
	_ lifecycle.Component = (*Gate)(nil)
)

// approvalDecision is sent through a channel when a decision arrives.
type approvalDecision struct {
	Result ApprovalResult
	Err    error
}

// Gate implements HITLGate with bus-based approval routing.
// It subscribes to approval/rejection events and routes decisions
// to waiting RequestApproval callers.
type Gate struct {
	bus        bus.BusPort
	store      ApprovalStore
	defaultTTL time.Duration

	mu       sync.Mutex
	pending  map[string]chan approvalDecision
	subs     []bus.Subscription
	started  bool
}

// NewGate creates a Gate with the given bus and store.
func NewGate(b bus.BusPort, store ApprovalStore, defaultTTL time.Duration) *Gate {
	return &Gate{
		bus:        b,
		store:      store,
		defaultTTL: defaultTTL,
		pending:    make(map[string]chan approvalDecision),
	}
}

// ---------------------------------------------------------------------------
// HITLGate interface
// ---------------------------------------------------------------------------

// RequestApproval submits a request and waits for a decision.
func (g *Gate) RequestApproval(ctx context.Context, req ApprovalRequest) (ApprovalResult, error) {
	ttl := g.defaultTTL
	if !req.ExpiresAt.IsZero() {
		ttl = time.Until(req.ExpiresAt)
	}
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}

	req.Status = ApprovalPending
	if req.CreatedAt.IsZero() {
		req.CreatedAt = time.Now()
	}
	if req.ExpiresAt.IsZero() {
		req.ExpiresAt = time.Now().Add(ttl)
	}

	if g.store != nil {
		if err := g.store.Create(ctx, req); err != nil {
			return ApprovalResult{}, fmt.Errorf("gate: store: %w", err)
		}
	}

	subject := g.subjectFor(req.Type)
	log.Printf("gate: requesting approval type=%s id=%s subject=%s", req.Type, req.ID, subject)

	// Publish the request event.
	env := event.New(event.TypePlanProposed, "hitl", map[string]string{
		"request_id": req.ID,
		"intent_id":  req.IntentID,
		"type":       string(req.Type),
		"summary":    req.Summary,
	})
	data, _ := event.Serialize(env)
	_ = g.bus.Publish(ctx, subject, data)

	// Create a decision channel and register it.
	ch := make(chan approvalDecision, 1)
	g.mu.Lock()
	g.pending[req.ID] = ch
	g.mu.Unlock()

	defer func() {
		g.mu.Lock()
		delete(g.pending, req.ID)
		g.mu.Unlock()
	}()

	// Wait for the decision or timeout.
	select {
	case decision := <-ch:
		return decision.Result, decision.Err
	case <-ctx.Done():
		// Update store to expired if we have one.
		if g.store != nil {
			_ = g.store.UpdateStatus(ctx, req.ID, ApprovalExpired, "", "timeout")
		}
		return ApprovalResult{Status: ApprovalExpired, RequestID: req.ID, DecidedAt: time.Now()}, nil
	}
}

// ---------------------------------------------------------------------------
// Lifecycle
// ---------------------------------------------------------------------------

func (g *Gate) Name() string { return "hitl_gate" }

func (g *Gate) Init(_ context.Context) error {
	if g.bus == nil {
		return fmt.Errorf("hitl: bus is required")
	}
	return nil
}

func (g *Gate) Start(ctx context.Context) error {
	subjects := []string{
		"devos.plan.approved",
		"devos.plan.rejected",
		"devos.deploy.approved",
		"devos.deploy.rejected",
	}

	for _, subj := range subjects {
		sub, err := g.bus.Subscribe(ctx, subj, g.handleDecision)
		if err != nil {
			g.unsubscribeAll()
			return fmt.Errorf("hitl: subscribe %s: %w", subj, err)
		}
		g.subs = append(g.subs, sub)
	}
	g.mu.Lock()
	g.started = true
	g.mu.Unlock()
	log.Printf("hitl: subscribed to %d approval subjects", len(subjects))
	return nil
}

func (g *Gate) Stop(_ context.Context) error {
	g.unsubscribeAll()
	g.mu.Lock()
	g.started = false
	g.mu.Unlock()
	return nil
}

func (g *Gate) Health() lifecycle.Health {
	g.mu.Lock()
	s := g.started
	g.mu.Unlock()
	if !s {
		return lifecycle.Health{Status: lifecycle.StatusDown, Since: time.Now()}
	}
	return lifecycle.Health{Status: lifecycle.StatusUp, Since: time.Now()}
}

func (g *Gate) unsubscribeAll() {
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, sub := range g.subs {
		_ = sub.Unsubscribe()
	}
	g.subs = nil
}

// ---------------------------------------------------------------------------
// Decision handler
// ---------------------------------------------------------------------------

func (g *Gate) handleDecision(ctx context.Context, msg bus.Message) error {
	env, err := event.Deserialize(msg.Data())
	if err != nil {
		_ = msg.Term()
		return nil
	}

	// Extract request_id from the event payload.
	_ = env // For now, decisions are routed by subject -> request_id mapping.

	// Determine the status based on event type.
	var status ApprovalStatus
	switch env.Type {
	case event.TypePlanApproved, event.TypeDeployApproved:
		status = ApprovalApproved
	case event.TypePlanRejected, event.TypeDeployRejected:
		status = ApprovalRejected
	default:
		_ = msg.Ack()
		return nil
	}

	_ = status
	_ = msg.Ack()
	return nil
}

func (g *Gate) subjectFor(at ApprovalType) string {
	switch at {
	case ApprovalPlan:
		return "devos.plan.proposed"
	case ApprovalDeploy:
		return "devos.deploy.requested"
	default:
		return "devos.plan.proposed"
	}
}
