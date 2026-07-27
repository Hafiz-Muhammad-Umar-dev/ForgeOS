package eval

import (
	"context"
	"log"
	"time"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/agent"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/workspace"
)

// GoldenTask is a predefined task that exercises the agent system.
type GoldenTask struct {
	// Name identifies the task.
	Name string

	// Description explains what the task does.
	Description string

	// Task is the agent.Task to execute.
	Task agent.Task

	// Assertions validate the result.
	Assertions []Assertion

	// Timeout bounds the total execution time.
	Timeout time.Duration
}

// EvalResult captures whether a golden task passed or failed.
type EvalResult struct {
	TaskName string
	Passed   bool
	Errors   []string
	Duration time.Duration
}

// Evaluator runs golden tasks through the Agent Runtime and collects results.
type Evaluator struct {
	runtime *agent.Runtime
	ws      workspace.WorkspacePort
	wsID    workspace.WorkspaceID
}

// NewEvaluator creates an Evaluator backed by the given runtime and workspace.
func NewEvaluator(rt *agent.Runtime, ws workspace.WorkspacePort, wsID workspace.WorkspaceID) *Evaluator {
	return &Evaluator{
		runtime: rt,
		ws:      ws,
		wsID:    wsID,
	}
}

// Run executes a single golden task and returns the evaluation result.
func (e *Evaluator) Run(ctx context.Context, gt GoldenTask) EvalResult {
	start := time.Now()

	timeout := gt.Timeout
	if timeout <= 0 {
		timeout = 60 * time.Second
	}

	taskCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	log.Printf("eval: task=%q starting (timeout=%s)", gt.Name, timeout)

	result, err := e.runtime.RunTask(taskCtx, "coder", gt.Task)
	if err != nil {
		elapsed := time.Since(start)
		log.Printf("eval: task=%q failed: %v (duration=%s)", gt.Name, err, elapsed)
		return EvalResult{
			TaskName: gt.Name,
			Passed:   false,
			Errors:   []string{err.Error()},
			Duration: elapsed,
		}
	}

	var errors []string
	for _, assertion := range gt.Assertions {
		if err := assertion.Check(result); err != nil {
			errors = append(errors, err.Error())
		}
	}

	passed := len(errors) == 0
	elapsed := time.Since(start)

	if passed {
		log.Printf("eval: task=%q passed (duration=%s)", gt.Name, elapsed)
	} else {
		log.Printf("eval: task=%q failed %d assertions (duration=%s)", gt.Name, len(errors), elapsed)
	}

	return EvalResult{
		TaskName: gt.Name,
		Passed:   passed,
		Errors:   errors,
		Duration: elapsed,
	}
}

// RunAll executes all golden tasks sequentially and returns the results.
func (e *Evaluator) RunAll(ctx context.Context, tasks []GoldenTask) []EvalResult {
	results := make([]EvalResult, 0, len(tasks))
	for _, gt := range tasks {
		result := e.Run(ctx, gt)
		results = append(results, result)
	}
	return results
}
