package scheduler

import "errors"

var (
	// ErrNotStarted is returned when the service is not started.
	ErrNotStarted = errors.New("scheduler: not started")

	// ErrQueueFull is returned when the task queue is full.
	ErrQueueFull = errors.New("scheduler: queue full")

	// ErrTaskNotFound is returned when a task cannot be found.
	ErrTaskNotFound = errors.New("scheduler: task not found")

	// ErrTaskNotRunning is returned when trying to cancel a task that
	// is not currently running.
	ErrTaskNotRunning = errors.New("scheduler: task not running")
)
