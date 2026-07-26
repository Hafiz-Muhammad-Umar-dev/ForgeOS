package scheduler

import "time"

// Config configures the scheduler service.
type Config struct {
	// QueueSize is the maximum number of pending tasks.
	// Zero means a default of 100.
	QueueSize int

	// WorkerCount is the number of concurrent workers.
	// Sprint 0 only supports 1.
	WorkerCount int

	// ShutdownTimeout is how long to wait for in-flight tasks
	// to complete during Stop().
	ShutdownTimeout time.Duration
}

// DefaultConfig returns a sensible default configuration.
func DefaultConfig() Config {
	return Config{
		QueueSize:       100,
		WorkerCount:     1,
		ShutdownTimeout: 5 * time.Second,
	}
}
