// Package ingress defines the Intent Ingress port and a REST (HTTP) adapter
// for submitting intents into the DevOS bus. It follows the ports/adapters
// (hexagonal) architecture: core services use IntentIngress, and the REST
// adapter satisfies it without leaking HTTP types into domain code.
//
// Sprint 0 scope:
//   - IntentIngress interface (SubmitIntent)
//   - IntentPayload, IntentResult, Attachment types
//   - REST adapter (POST /v1/intents, GET /healthz, GET /readyz)
//   - FakeIngress for tests
//   - Bus integration via existing event.EventEnvelope + event.Serialize
//
// Excluded from Sprint 0 (deferred to later components):
//   - Authentication / identity injection (API Gateway)
//   - Webhook signature verification
//   - Rate limiting
//   - Channel-specific adapters (Discord, Slack, etc.)
//   - WebSocket / SSE
//   - Persistence
//
// See ADR-006 (Channel Adapters Funnel to Uniform Intent), SDD §01
// (Intent Ingress), SDD §10 (Channel Adapters).
package ingress

import "context"

// IntentIngress is the port for submitting user intents into the system.
// Implementations accept an intent payload and return a tracking result.
type IntentIngress interface {
	// SubmitIntent accepts an intent payload and returns a result with
	// the assigned intent ID. The implementation is responsible for
	// wrapping the payload in a canonical event envelope and publishing
	// it to the bus.
	SubmitIntent(ctx context.Context, payload IntentPayload) (IntentResult, error)
}

// IntentPayload is the domain payload carried by an intent.created event.
// It contains the user's natural-language request and associated metadata.
type IntentPayload struct {
	// Text is the natural-language intent (required).
	Text string `json:"text"`

	// UserID identifies the submitting user. In Sprint 0 this is
	// provided by the caller; identity injection via API Gateway is
	// deferred to a later sprint.
	UserID string `json:"user_id,omitempty"`

	// OrgID is the tenant organization. When empty a default is used.
	OrgID string `json:"org_id,omitempty"`

	// ProjectID scopes the intent within an organization.
	ProjectID string `json:"project_id,omitempty"`

	// TraceID for distributed tracing. Generated when empty.
	TraceID string `json:"trace_id,omitempty"`

	// Attachments are references to external content (metadata only;
	// no raw payload is stored in the event model).
	Attachments []Attachment `json:"attachments,omitempty"`
}

// Attachment is a metadata-only reference to external content.
type Attachment struct {
	Name     string `json:"name"`
	MIMEType string `json:"mime_type"`
	URI      string `json:"uri"`
}

// IntentResult is returned after a successful intent submission.
type IntentResult struct {
	IntentID string `json:"intent_id"`
	Status   string `json:"status"`
	TraceID  string `json:"trace_id,omitempty"`
}
