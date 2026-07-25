package orchestrator

import (
	"context"
	"sync/atomic"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/agent"
)

// Compile-time check.
var _ TaskRunner = (*FakeRunner)(nil)

// FakeRunner is an in-memory TaskRunner implementation for testing.
// It records all calls and returns configurable results.
type FakeRunner struct {
	// RunFunc overrides the RunTask behavior.
	RunFunc func(ctx context.Context, agentName string, task agent.Task) (*agent.Result, error)

	// ResultValue is returned by the default RunTask implementation.
	ResultValue *agent.Result

	// RunError is returned by the default RunTask implementation when set.
	RunError error

	// RunCount tracks the number of RunTask calls.
	RunCount atomic.Int64

	// Received records every task dispatched.
	Received []dispatchCall
}

type dispatchCall struct {
	AgentName string
	Task      agent.Task
}

// NewFakeRunner creates a FakeRunner with a default success result.
func NewFakeRunner() *FakeRunner {
	return &FakeRunner{
		ResultValue: &agent.Result{
			Summary: "completed",
			Status:  agent.ResultSuccess,
		},
	}
}

// RunTask records the call and returns the configured result.
func (f *FakeRunner) RunTask(ctx context.Context, agentName string, task agent.Task) (*agent.Result, error) {
	f.RunCount.Add(1)
	f.Received = append(f.Received, dispatchCall{AgentName: agentName, Task: task})

	if f.RunFunc != nil {
		return f.RunFunc(ctx, agentName, task)
	}
	if f.RunError != nil {
		return nil, f.RunError
	}
	return f.ResultValue, nil
}
