package scheduler

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/lifecycle"
)

// Compile-time checks.
var (
	_ Scheduler           = (*Service)(nil)
	_ lifecycle.Component = (*Service)(nil)
)

// WorkFunc processes a scheduled task. Implementations receive a task
// and its derived context. Returning an error marks the task as failed.
type WorkFunc func(ctx context.Context, task Task) error

// Service implements the Scheduler port with a sequential task queue.
// It processes one task at a time through a configurable WorkFunc.
// Tasks are queued via RequestSchedule and executed by a background worker.
type Service struct {
	config Config
	workFn WorkFunc
	queue  chan Task
	done   chan struct{}

	mu         sync.Mutex
	tasks      map[string]Task
	taskCancel context.CancelFunc
	started    bool

	// cancelWorkers is used to signal worker shutdown.
	cancelWorkers context.CancelFunc

	queueLen atomic.Int64
	running  atomic.Bool
}

// NewService creates a new scheduler service.
// workFn is called for each task that is dequeued for execution.
func NewService(cfg Config, workFn WorkFunc) *Service {
	return &Service{
		config: cfg,
		workFn: workFn,
		queue:  make(chan Task, cfg.QueueSize),
		tasks:  make(map[string]Task),
	}
}

// ---------------------------------------------------------------------------
// Scheduler interface
// ---------------------------------------------------------------------------

// RequestSchedule enqueues a task for execution. Returns an error when
// the queue is full or the service is not started.
func (s *Service) RequestSchedule(ctx context.Context, req ScheduleRequest) (Task, error) {
	if !s.started {
		return Task{}, fmt.Errorf("%w: scheduler not running", ErrNotStarted)
	}

	task := Task{
		ID:        req.IntentID,
		OrgID:     req.OrgID,
		ProjectID: req.ProjectID,
		AgentID:   req.AgentID,
		IntentID:  req.IntentID,
		State:     TaskStatePending,
		Payload:   req.Payload,
		CreatedAt: time.Now(),
	}
	// Use a placeholder ID if IntentID is empty.
	if task.ID == "" {
		task.ID = fmt.Sprintf("task-%d", time.Now().UnixNano())
	}

	s.mu.Lock()
	s.tasks[task.ID] = task
	s.mu.Unlock()

	select {
	case s.queue <- task:
		s.queueLen.Add(1)
		return task, nil
	case <-ctx.Done():
		// Context cancelled while waiting to enqueue.
		s.mu.Lock()
		delete(s.tasks, task.ID)
		s.mu.Unlock()
		return Task{}, ctx.Err()
	default:
		// Queue full.
		s.mu.Lock()
		delete(s.tasks, task.ID)
		s.mu.Unlock()
		return Task{}, fmt.Errorf("%w: queue capacity %d", ErrQueueFull, s.config.QueueSize)
	}
}

// Cancel cancels a running task. Returns an error if the task is not found
// or not currently running.
func (s *Service) Cancel(_ context.Context, taskID string) error {
	s.mu.Lock()
	task, ok := s.tasks[taskID]
	s.mu.Unlock()

	if !ok {
		return fmt.Errorf("%w: id=%s", ErrTaskNotFound, taskID)
	}
	if task.State != TaskStateRunning {
		return fmt.Errorf("%w: id=%s state=%s", ErrTaskNotRunning, taskID, task.State)
	}

	s.mu.Lock()
	if s.taskCancel != nil {
		s.taskCancel()
	}
	s.mu.Unlock()
	return nil
}

// Status returns the current state of a task.
func (s *Service) Status(_ context.Context, taskID string) (Task, error) {
	s.mu.Lock()
	task, ok := s.tasks[taskID]
	s.mu.Unlock()

	if !ok {
		return Task{}, fmt.Errorf("%w: id=%s", ErrTaskNotFound, taskID)
	}
	return task, nil
}

// QueueLength returns the number of tasks currently in the queue.
func (s *Service) QueueLength() int {
	return int(s.queueLen.Load())
}

// IsRunning reports whether a task is currently being executed.
func (s *Service) IsRunning() bool {
	return s.running.Load()
}

// ---------------------------------------------------------------------------
// Lifecycle
// ---------------------------------------------------------------------------

// Name returns "scheduler" for the lifecycle manager.
func (s *Service) Name() string { return "scheduler" }

// Init validates configuration.
func (s *Service) Init(_ context.Context) error {
	if s.workFn == nil {
		return fmt.Errorf("scheduler: work function is required")
	}
	return nil
}

// Start begins processing the task queue.
func (s *Service) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return nil
	}

	ctx, s.cancelWorkers = context.WithCancel(ctx)
	s.done = make(chan struct{})

	go s.workerLoop(ctx)

	s.started = true
	return nil
}

// Stop gracefully shuts down the scheduler. In-flight tasks receive
// a cancelled context. Pending tasks remain in the queue and are not
// processed after shutdown.
func (s *Service) Stop(ctx context.Context) error {
	s.mu.Lock()
	if !s.started {
		s.mu.Unlock()
		return nil
	}
	s.started = false
	s.mu.Unlock()

	// Signal the worker to stop.
	if s.cancelWorkers != nil {
		s.cancelWorkers()
	}

	// Wait for the worker to finish, with timeout.
	done := make(chan struct{})
	go func() {
		<-s.done
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(s.config.ShutdownTimeout):
	}

	return nil
}

// Health reports whether the scheduler is running.
func (s *Service) Health() lifecycle.Health {
	if !s.started {
		return lifecycle.Health{Status: lifecycle.StatusDown, Since: time.Now()}
	}
	return lifecycle.Health{Status: lifecycle.StatusUp, Since: time.Now()}
}

// ---------------------------------------------------------------------------
// Worker
// ---------------------------------------------------------------------------

// workerLoop processes tasks from the queue sequentially.
func (s *Service) workerLoop(ctx context.Context) {
	defer close(s.done)
	defer s.running.Store(false)

	for {
		select {
		case <-ctx.Done():
			return
		case task := <-s.queue:
			s.queueLen.Add(-1)
			s.running.Store(true)

			s.updateTaskState(task.ID, TaskStateRunning)

			// Create a cancelable context for this task.
			taskCtx, taskCancel := context.WithCancel(ctx)
			s.mu.Lock()
			s.taskCancel = taskCancel
			s.mu.Unlock()

			err := s.workFn(taskCtx, task)

			taskCancel()
			s.mu.Lock()
			s.taskCancel = nil
			s.mu.Unlock()

			if err != nil {
				s.updateTaskState(task.ID, TaskStateFailed)
			} else {
				s.updateTaskState(task.ID, TaskStateDone)
			}

			s.running.Store(false)
		}
	}
}

// updateTaskState atomically updates a task's state.
func (s *Service) updateTaskState(id string, state TaskState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if t, ok := s.tasks[id]; ok {
		t.State = state
		s.tasks[id] = t
	}
}
