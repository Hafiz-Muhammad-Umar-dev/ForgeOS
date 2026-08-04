package planner

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/dag"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/ingress"
)

// Compile-time check.
var _ Planner = (*FakePlanner)(nil)

// FakePlanner is an in-memory Planner implementation for testing.
// It returns a configurable DAG and records all received intents.
type FakePlanner struct {
	// PlanFunc overrides the Plan behavior. If nil, Result is used.
	PlanFunc func(ctx context.Context, intent ingress.IntentPayload) (*dag.DAG, error)

	// Result is returned by the default Plan implementation.
	Result *dag.DAG

	// PlanError is returned by the default Plan implementation when set.
	PlanError error

	// PlanCount tracks the number of Plan calls.
	PlanCount atomic.Int64

	// ReceivedIntents records every intent received.
	ReceivedIntents []ingress.IntentPayload
}

// NewFakePlanner creates a FakePlanner with a default single-node DAG.
func NewFakePlanner() *FakePlanner {
	return &FakePlanner{
		Result: &dag.DAG{
			ID:        "fake-dag-1",
			Status:    dag.DAGProposed,
			Nodes:     []dag.Node{{ID: "node-1", Agent: "coder", Description: "implement the task"}},
			CreatedAt: time.Now(),
		},
	}
}

// Plan records the call and returns the configured result.
func (f *FakePlanner) Plan(_ context.Context, intent ingress.IntentPayload) (*dag.DAG, error) {
	f.PlanCount.Add(1)
	f.ReceivedIntents = append(f.ReceivedIntents, intent)

	if f.PlanFunc != nil {
		return f.PlanFunc(nil, intent)
	}
	if f.PlanError != nil {
		return nil, f.PlanError
	}
	result := *f.Result
	result.IntentID = intent.Text
	return &result, nil
}
