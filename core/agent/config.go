package agent

// Config configures the agent runtime.
type Config struct {
	// DefaultMaxIterations is the max LLM-tool cycles when the task does not
	// specify one. Defaults to 10.
	DefaultMaxIterations int

	// DefaultModel is the LLM model to use when none is specified.
	DefaultModel string

	// PublishEvents controls whether the runtime publishes lifecycle events
	// to the bus. Defaults to true.
	PublishEvents bool
}

// DefaultConfig returns a sensible default runtime configuration.
func DefaultConfig() Config {
	return Config{
		DefaultMaxIterations: 10,
		DefaultModel:         "",
		PublishEvents:        true,
	}
}
