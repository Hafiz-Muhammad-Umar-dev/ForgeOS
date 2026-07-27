// Package hitl provides human-in-the-loop approval gates for the DevOS
// orchestration engine. HITL gates pause DAG execution until a human
// approves or rejects a proposed action.
//
// See ADR-007 (Human-in-the-Loop as First-Class Gate).
package hitl

import (
	"context"
	"errors"
	"time"
)

// ApprovalType identifies what kind of action requires approval.
type ApprovalType string

const (
	ApprovalPlan   ApprovalType = "plan"
	ApprovalDeploy ApprovalType = "deploy"
)

// ApprovalStatus represents the lifecycle state of an approval request.
type ApprovalStatus string

const (
	ApprovalPending  ApprovalStatus = "pending"
	ApprovalApproved ApprovalStatus = "approved"
	ApprovalRejected ApprovalStatus = "rejected"
	ApprovalExpired  ApprovalStatus = "expired"
)

// ApprovalRequest is created when a HITL gate pauses execution.
type ApprovalRequest struct {
	// ID uniquely identifies this approval request.
	ID string `json:"id"`

	// IntentID links the request to the originating intent.
	IntentID string `json:"intent_id"`

	// Type identifies what kind of approval is needed.
	Type ApprovalType `json:"type"`

	// Summary is a human-readable description of what requires approval.
	Summary string `json:"summary"`

	// NodeID is the DAG node ID that requires approval (if applicable).
	NodeID string `json:"node_id,omitempty"`

	// Status tracks the current state of this request.
	Status ApprovalStatus `json:"status"`

	// ExpiresAt sets the deadline for a decision.
	ExpiresAt time.Time `json:"expires_at"`

	// CreatedAt is when the request was created.
	CreatedAt time.Time `json:"created_at"`
}

// ApprovalResult is returned when a decision is reached.
type ApprovalResult struct {
	// RequestID matches the original request.
	RequestID string `json:"request_id"`

	// Status tells whether the request was approved, rejected, or expired.
	Status ApprovalStatus `json:"status"`

	// ApprovedBy identifies who made the decision.
	ApprovedBy string `json:"approved_by,omitempty"`

	// Reason is an optional justification.
	Reason string `json:"reason,omitempty"`

	// DecidedAt is when the decision was made.
	DecidedAt time.Time `json:"decided_at"`
}

// HITLGate is the port for human-in-the-loop approval. Implementations
// publish approval requests, wait for a decision, and return the result.
type HITLGate interface {
	// RequestApproval submits a request and blocks until a decision is
	// reached or the context is cancelled.
	RequestApproval(ctx context.Context, req ApprovalRequest) (ApprovalResult, error)
}

// Sentinel errors returned by HITLGate implementations.
var (
	ErrTimeout        = errors.New("hitl: approval timeout")
	ErrNotStarted     = errors.New("hitl: not started")
)
