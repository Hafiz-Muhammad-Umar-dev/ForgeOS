// Package orchestrator implements the minimal orchestration engine for
// DevOS. It subscribes to intent.created events, dispatches them to the
// Agent Runtime via TaskRunner, and publishes intent.completed / intent.failed.
//
// Sprint 0 scope (event bridge only):
//   - Subscribe to intent.created
//   - Deserialize event → extract IntentPayload
//   - Convert to agent.Task
//   - Dispatch via TaskRunner
//   - Publish intent.completed / intent.failed
//   - Ack/Nak/Term bus messages
//
// Excluded from Sprint 0:
//   - DAG planner, multi-agent, HITL, budget, retry, persistence,
//     scheduling, workflow engine
//
// See SDD §03 (Orchestration Service), ADR-001 (Event-Driven Core).
package orchestrator

import (
	"context"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/agent"
)

// TaskRunner is the subset of the Agent Runtime that the orchestrator needs.
// Defining a narrow interface keeps the orchestrator decoupled from the
// full agent.Runtime struct.
type TaskRunner interface {
	// RunTask executes a task with the named agent and returns the result.
	RunTask(ctx context.Context, agentName string, task agent.Task) (*agent.Result, error)
}
