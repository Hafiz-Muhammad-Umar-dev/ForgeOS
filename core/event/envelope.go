// Package event defines the DevOS event envelope and the canonical event
// catalog shared across all services. It is pure data + helpers; it contains
// no transport logic (NATS integration lives in core/bus, a later milestone).
//
// See specs/02-specification/04-message-bus.md (Event Envelope, Subject
// Hierarchy, Event Catalog) and ADR-001.
package event

import (
	"crypto/rand"
	"fmt"
	"strings"
	"time"
)

// SchemaVersion is the envelope schema version.
type SchemaVersion int

// CurrentSchemaVersion is the schema version emitted by New.
const CurrentSchemaVersion SchemaVersion = 1

// EventType identifies an event (e.g., "intent.created").
type EventType string

// Canonical event catalog (specs/02-specification/04-message-bus.md §6).
const (
	TypeIntentCreated     EventType = "intent.created"
	TypeIntentCancelled   EventType = "intent.cancelled"
	TypeIntentCompleted   EventType = "intent.completed"
	TypeIntentFailed      EventType = "intent.failed"
	TypePlanProposed      EventType = "plan.proposed"
	TypePlanApproved      EventType = "plan.approved"
	TypePlanRejected      EventType = "plan.rejected"
	TypePlanStarted       EventType = "plan.started"
	TypeNodeStatus        EventType = "node.status"
	TypeNodeFailed        EventType = "node.failed"
	TypeTaskAssigned      EventType = "task.assigned"
	TypeTaskStatus        EventType = "task.status"
	TypeTaskFailed        EventType = "task.failed"
	TypeAgentToken        EventType = "agent.token"
	TypeArtifactPublished EventType = "artifact.published"
	TypeReviewPassed      EventType = "review.passed"
	TypeReviewComment     EventType = "review.comment"
	TypeTestResult        EventType = "test.result"
	TypeSecurityFinding   EventType = "security.finding"
	TypeGitCommit         EventType = "git.commit"
	TypeDeployCompleted   EventType = "deploy.completed"
	TypeBudgetExceeded    EventType = "budget.exceeded"
	TypeWorkspaceReady    EventType = "workspace.ready"
	TypeAgentRegistered   EventType = "agent.registered"
)

// Subject is the NATS subject for an event: devos.<entity>.<action>.
type Subject struct {
	Entity string
	Action string
}

// String renders the full NATS subject.
func (s Subject) String() string {
	return "devos." + s.Entity + "." + s.Action
}

// EventType returns the corresponding event type (entity.action, no prefix).
func (s Subject) EventType() EventType {
	return EventType(s.Entity + "." + s.Action)
}

// Subject parses the entity/action from an EventType.
func (t EventType) Subject() Subject {
	parts := strings.SplitN(string(t), ".", 2)
	if len(parts) == 2 {
		return Subject{Entity: parts[0], Action: parts[1]}
	}
	return Subject{Entity: string(t)}
}

// EventEnvelope is the uniform envelope wrapping every domain event.
// It matches specs/02-specification/04-message-bus.md §4.
type EventEnvelope[T any] struct {
	ID            string
	Type          EventType
	SchemaVersion SchemaVersion
	TraceID       string
	OrgID         string
	ProjectID     string
	ProducedBy    string
	ProducedAt    time.Time
	Payload       T
}

// Option customizes envelope construction.
type Option func(*options)

type options struct {
	traceID   string
	orgID     string
	projectID string
	now       time.Time
}

// WithTraceID sets the trace id (distributed tracing correlation).
func WithTraceID(id string) Option { return func(o *options) { o.traceID = id } }

// WithOrgID sets the organization id.
func WithOrgID(id string) Option { return func(o *options) { o.orgID = id } }

// WithProjectID sets the project id.
func WithProjectID(id string) Option { return func(o *options) { o.projectID = id } }

// WithTime overrides the production timestamp (used by tests for determinism).
func WithTime(t time.Time) Option { return func(o *options) { o.now = t } }

// New constructs an envelope, generating a unique ID and defaulting the
// timestamp to time.Now when not overridden.
func New[T any](typ EventType, producedBy string, payload T, opts ...Option) EventEnvelope[T] {
	o := options{now: time.Now()}
	for _, fn := range opts {
		fn(&o)
	}
	id, _ := newID()
	return EventEnvelope[T]{
		ID:            id,
		Type:          typ,
		SchemaVersion: CurrentSchemaVersion,
		TraceID:       o.traceID,
		OrgID:         o.orgID,
		ProjectID:     o.projectID,
		ProducedBy:    producedBy,
		ProducedAt:    o.now,
		Payload:       payload,
	}
}

// newID returns a random RFC 4122 v4 UUID (dependency-free).
func newID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // RFC 4122 variant
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}
