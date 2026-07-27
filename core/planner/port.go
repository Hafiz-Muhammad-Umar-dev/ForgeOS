// Package planner provides the Planner interface and an LLM-based
// implementation that decomposes an intent into a DAG of tasks.
//
// See ADR-002 (Stateful DAG Orchestration), SDD §03 §2 (Planner).
package planner

import (
	"context"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/dag"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/ingress"
)

// Planner decomposes an intent into a DAG of executable task nodes.
type Planner interface {
	// Plan accepts a user intent and returns a DAG. The DAG's nodes
	// reference agents by name (e.g., "frontend", "backend") and
	// include dependency edges between them.
	Plan(ctx context.Context, intent ingress.IntentPayload) (*dag.DAG, error)
}
