package orchestrator

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/agent"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/budget"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/bus"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/dag"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/event"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/hitl"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/ingress"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/lifecycle"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/planner"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/registry"
)

// Compile-time check.
var _ lifecycle.Component = (*DAGExecutor)(nil)

// DAGExecutor subscribes to intent.created events, plans DAGs via the Planner,
// executes nodes via the Coordinator, and publishes plan/node lifecycle events.
type DAGExecutor struct {
	bus         bus.BusPort
	planner     planner.Planner
	coordinator *Coordinator
	nodeExec    *NodeExecutor
	gate        hitl.HITLGate   // optional human-in-the-loop approval
	gov         budget.Governor // optional budget enforcement
	sub         bus.Subscription
	mu          sync.Mutex
	started     bool
}

// DAGExecutorOption configures the DAGExecutor.
type DAGExecutorOption func(*DAGExecutor)

// WithHITLGate attaches a human-in-the-loop approval gate.
func WithHITLGate(g hitl.HITLGate) DAGExecutorOption {
	return func(e *DAGExecutor) { e.gate = g }
}

// WithBudgetGovernor attaches a budget governor for token/cost enforcement.
func WithBudgetGovernor(g budget.Governor) DAGExecutorOption {
	return func(e *DAGExecutor) { e.gov = g }
}

// NewDAGExecutor creates a new DAGExecutor.
func NewDAGExecutor(b bus.BusPort, p planner.Planner, reg registry.Registry, runner TaskRunner, opts ...DAGExecutorOption) *DAGExecutor {
	coord := NewCoordinator(reg, runner)
	e := &DAGExecutor{
		bus:         b,
		planner:     p,
		coordinator: coord,
		nodeExec:    NewNodeExecutor(coord),
	}
	for _, fn := range opts {
		fn(e)
	}
	return e
}

// ---------------------------------------------------------------------------
// Lifecycle
// ---------------------------------------------------------------------------

// Name returns "dag_executor" for the lifecycle manager.
func (e *DAGExecutor) Name() string { return "dag_executor" }

// Init validates dependencies.
func (e *DAGExecutor) Init(_ context.Context) error {
	if e.bus == nil {
		return fmt.Errorf("dag_executor: bus is required")
	}
	if e.planner == nil {
		return fmt.Errorf("dag_executor: planner is required")
	}
	if e.coordinator == nil {
		return fmt.Errorf("dag_executor: coordinator is required")
	}
	return nil
}

// Start subscribes to intent.created events.
func (e *DAGExecutor) Start(ctx context.Context) error {
	sub, err := e.bus.Subscribe(ctx, "devos.intent.created", e.handleIntentCreated)
	if err != nil {
		return fmt.Errorf("dag_executor: subscribe: %w", err)
	}
	e.mu.Lock()
	e.sub = sub
	e.started = true
	e.mu.Unlock()
	log.Printf("dag_executor: subscribed to devos.intent.created")
	return nil
}

// Stop unsubscribes from intent.created.
func (e *DAGExecutor) Stop(_ context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.sub != nil {
		_ = e.sub.Unsubscribe()
		e.sub = nil
	}
	e.started = false
	return nil
}

// Health reports whether the executor is subscribed.
func (e *DAGExecutor) Health() lifecycle.Health {
	e.mu.Lock()
	s := e.started
	e.mu.Unlock()
	if !s {
		return lifecycle.Health{Status: lifecycle.StatusDown, Since: time.Now()}
	}
	return lifecycle.Health{Status: lifecycle.StatusUp, Since: time.Now()}
}

// ---------------------------------------------------------------------------
// Message handler
// ---------------------------------------------------------------------------

func (e *DAGExecutor) handleIntentCreated(ctx context.Context, msg bus.Message) error {
	env, err := event.Deserialize(msg.Data())
	if err != nil {
		_ = msg.Term()
		return nil
	}

	if env.Type != event.TypeIntentCreated {
		_ = msg.Ack()
		return nil
	}

	payload, err := event.UnmarshalPayload[ingress.IntentPayload](env)
	if err != nil {
		_ = msg.Term()
		return nil
	}

	dagID := env.ID
	log.Printf("dag_executor: planning for intent=%s", dagID)

	d, err := e.planner.Plan(ctx, payload)
	if err != nil {
		log.Printf("dag_executor: plan failed for intent=%s: %v", dagID, err)
		e.publishIntentFailed(ctx, env, payload, err)
		_ = msg.Ack()
		return nil
	}
	d.ID = dagID
	d.IntentID = dagID

	e.publishPlanStarted(ctx, d, env)
	e.publishPlanProposed(ctx, d, env)

	// HITL gate: pause execution until human approves/rejects the plan.
	if e.gate != nil {
		// Use a timeout context so approval does not block indefinitely.
		approvalCtx, cancel := context.WithTimeout(ctx, 15*time.Minute)
		defer cancel()

		approvalReq := hitl.ApprovalRequest{
			ID:       dagID + "-approval",
			IntentID: dagID,
			Type:     hitl.ApprovalPlan,
			Summary:  payload.Text,
		}
		result, err := e.gate.RequestApproval(approvalCtx, approvalReq)
		if err != nil || result.Status == hitl.ApprovalRejected || result.Status == hitl.ApprovalExpired {
			reason := "plan rejected"
			if err != nil {
				reason = err.Error()
			} else if result.Reason != "" {
				reason = result.Reason
			}
			log.Printf("dag_executor: plan rejected for intent=%s: %s", dagID, reason)
			e.publishPlanRejected(ctx, d, env, reason)
			e.publishIntentFailed(ctx, env, payload, fmt.Errorf("plan rejected: %s", reason))
			_ = msg.Ack()
			return nil
		}
		log.Printf("dag_executor: plan approved for intent=%s", dagID)
	}

	if err := e.executeDAG(ctx, d, env, payload); err != nil {
		log.Printf("dag_executor: execution failed for intent=%s: %v", dagID, err)
		_ = msg.Ack()
		return nil
	}

	_ = msg.Ack()
	return nil
}

// executeDAG runs all DAG nodes in topological order.
func (e *DAGExecutor) executeDAG(ctx context.Context, d *dag.DAG, env event.RawEnvelope, payload ingress.IntentPayload) error {
	d.Status = dag.DAGRunning

	sorted, err := d.TopologicalSort()
	if err != nil {
		e.publishIntentFailed(ctx, env, payload, fmt.Errorf("dag: %w", err))
		return nil
	}

	for i := range sorted {
		node := d.FindNode(sorted[i].ID)
		if node == nil {
			continue
		}

		if node.Status == dag.NodeSkipped {
			continue
		}

		// Budget check: verify capacity before dispatching.
		if e.gov != nil {
			if _, err := e.gov.Check(ctx, env.OrgID); err != nil {
				log.Printf("dag_executor: budget exceeded for org=%s node=%s", env.OrgID, node.ID)
				e.publishNodeFailed(ctx, node, d, env)
				node.Error = err.Error()
				e.publishBudgetExceeded(ctx, env)
				e.markDownstreamSkipped(d, node.ID)
				e.publishIntentFailed(ctx, env, payload, err)
				return nil
			}
		}

		result, execErr := e.nodeExec.Execute(ctx, node, d)
		if execErr != nil {
			e.publishNodeFailed(ctx, node, d, env)
			e.markDownstreamSkipped(d, node.ID)
			e.publishIntentFailed(ctx, env, payload, execErr)
			return nil
		}

		e.publishNodeStatus(ctx, node, d, result, env)

		// Budget consumption: record usage after successful dispatch.
		if e.gov != nil && result != nil {
			usage := budget.Usage{
				InputTokens:  result.InputTokens,
				OutputTokens: result.OutputTokens,
				AgentName:    node.Agent,
			}
			if err := e.gov.Consume(ctx, env.OrgID, usage); err != nil {
				log.Printf("dag_executor: budget consume failed for org=%s: %v", env.OrgID, err)
			}
		}
	}

	d.Status = dag.DAGCompleted
	e.publishIntentCompleted(ctx, env, payload, fmt.Sprintf("dag %s completed with %d nodes", d.ID, len(d.Nodes)))
	return nil
}

func (e *DAGExecutor) markDownstreamSkipped(d *dag.DAG, failedID string) {
	for i := range d.Nodes {
		for _, inputID := range d.Nodes[i].InputIDs {
			if inputID == failedID && d.Nodes[i].Status == dag.NodePending {
				d.Nodes[i].Status = dag.NodeSkipped
				log.Printf("dag: node=%s skipped due to upstream failure", d.Nodes[i].ID)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Event publishing
// ---------------------------------------------------------------------------

func (e *DAGExecutor) publishPlanStarted(ctx context.Context, d *dag.DAG, env event.RawEnvelope) {
	e.publish(ctx, event.TypePlanStarted, d.ID, env)
}

func (e *DAGExecutor) publishPlanProposed(ctx context.Context, d *dag.DAG, env event.RawEnvelope) {
	e.publish(ctx, event.TypePlanProposed, d.ID, env)
}

func (e *DAGExecutor) publishPlanRejected(ctx context.Context, d *dag.DAG, env event.RawEnvelope, reason string) {
	_ = e.publishPayload(ctx, event.TypePlanRejected, map[string]string{
		"dag_id": d.ID, "reason": reason,
	}, env)
}

func (e *DAGExecutor) publishBudgetExceeded(ctx context.Context, env event.RawEnvelope) {
	_ = e.publishPayload(ctx, event.TypeBudgetExceeded, map[string]string{
		"org_id": env.OrgID,
	}, env)
}

func (e *DAGExecutor) publishNodeStatus(ctx context.Context, node *dag.Node, d *dag.DAG, result *agent.Result, env event.RawEnvelope) {
	pl := dag.NodePayload{
		NodeID:  node.ID,
		DAGID:   d.ID,
		Agent:   node.Agent,
		Status:  string(node.Status),
		Summary: result.Summary,
	}
	_ = e.publishPayload(ctx, event.TypeNodeStatus, pl, env)
}

func (e *DAGExecutor) publishNodeFailed(ctx context.Context, node *dag.Node, d *dag.DAG, env event.RawEnvelope) {
	pl := dag.NodePayload{
		NodeID: node.ID,
		DAGID:  d.ID,
		Agent:  node.Agent,
		Status: "failed",
		Error:  node.Error,
	}
	_ = e.publishPayload(ctx, event.TypeNodeFailed, pl, env)
}

func (e *DAGExecutor) publishIntentCompleted(ctx context.Context, env event.RawEnvelope, p ingress.IntentPayload, summary string) {
	pl := IntentLifecyclePayload{
		IntentID:  env.ID,
		UserID:    p.UserID,
		OrgID:     env.OrgID,
		ProjectID: env.ProjectID,
		TraceID:   env.TraceID,
		Summary:   summary,
	}
	_ = e.publishPayload(ctx, event.TypeIntentCompleted, pl, env)
}

func (e *DAGExecutor) publishIntentFailed(ctx context.Context, env event.RawEnvelope, p ingress.IntentPayload, err error) {
	pl := IntentLifecyclePayload{
		IntentID:  env.ID,
		UserID:    p.UserID,
		OrgID:     env.OrgID,
		ProjectID: env.ProjectID,
		TraceID:   env.TraceID,
		Error:     err.Error(),
	}
	_ = e.publishPayload(ctx, event.TypeIntentFailed, pl, env)
}

func (e *DAGExecutor) publish(ctx context.Context, typ event.EventType, id string, env event.RawEnvelope) {
	_ = e.publishPayload(ctx, typ, map[string]string{"dag_id": id}, env)
}

func (e *DAGExecutor) publishPayload(ctx context.Context, typ event.EventType, payload any, env event.RawEnvelope) error {
	envelope := event.New(typ, "dag_executor", payload,
		event.WithTraceID(env.TraceID),
		event.WithOrgID(env.OrgID),
		event.WithProjectID(env.ProjectID),
	)
	data, err := event.Serialize(envelope)
	if err != nil {
		return err
	}
	return e.bus.Publish(ctx, typ.Subject().String(), data)
}
