package bus

import (
	"time"
)

// NatsConfig holds configuration for the NATS JetStream bus adapter.
type NatsConfig struct {
	// URL is the NATS server connection URL(s), comma-separated.
	URL string

	// Name is the NATS client name (logged on the server).
	Name string

	// Timeout for connection establishment.
	Timeout time.Duration

	// ReconnectWait is the wait time between reconnection attempts.
	ReconnectWait time.Duration

	// MaxReconnects is the maximum number of reconnection attempts.
	// -1 means unlimited (NATS default).
	MaxReconnects int

	// StreamConfigs defines the JetStream streams to auto-create on startup.
	// If nil, default streams are created based on subject domains.
	StreamConfigs []StreamConfig
}

// StreamConfig defines a JetStream stream for auto-creation.
type StreamConfig struct {
	// Name is the stream name (e.g., "INTENTS").
	Name string

	// Subjects are the subject patterns this stream captures (e.g., ["devos.intent.>"]).
	Subjects []string

	// MaxAge is how long messages are kept (e.g., 7d for INTENTS).
	MaxAge time.Duration

	// Storage is the storage type: "file" (default) or "memory".
	Storage string

	// Retention is the retention policy: "limits" (default), "workQueue", or "interest".
	Retention string

	// Replicas is the number of stream replicas. Default 1 (Sprint 0; production RF=3).
	Replicas int
}

// DefaultNatsConfig returns a sensible default configuration for development.
func DefaultNatsConfig() NatsConfig {
	return NatsConfig{
		URL:           "nats://localhost:4222",
		Name:          "devos",
		Timeout:       5 * time.Second,
		ReconnectWait: 2 * time.Second,
		MaxReconnects: -1,
		StreamConfigs: DefaultStreamConfigs(),
	}
}

// DefaultStreamConfigs returns the canonical stream definitions from the spec.
func DefaultStreamConfigs() []StreamConfig {
	return []StreamConfig{
		{Name: "INTENTS", Subjects: []string{"devos.intent.>"}, MaxAge: 7 * 24 * time.Hour, Storage: "file", Retention: "workQueue", Replicas: 1},
		{Name: "PLANS", Subjects: []string{"devos.plan.>"}, MaxAge: 30 * 24 * time.Hour, Storage: "file", Retention: "limits", Replicas: 1},
		{Name: "TASKS", Subjects: []string{"devos.task.>"}, MaxAge: 30 * 24 * time.Hour, Storage: "file", Retention: "limits", Replicas: 1},
		{Name: "TOKENS", Subjects: []string{"devos.agent.token"}, MaxAge: 24 * time.Hour, Storage: "file", Retention: "limits", Replicas: 1},
		{Name: "ARTIFACTS", Subjects: []string{"devos.artifact.>"}, MaxAge: 90 * 24 * time.Hour, Storage: "file", Retention: "limits", Replicas: 1},
		{Name: "DEPLOYS", Subjects: []string{"devos.deploy.>"}, MaxAge: 90 * 24 * time.Hour, Storage: "file", Retention: "limits", Replicas: 1},
		{Name: "OBS", Subjects: []string{"devos.obs.>"}, MaxAge: 14 * 24 * time.Hour, Storage: "file", Retention: "limits", Replicas: 1},
	}
}

// NatsOption configures the NATS bus adapter.
type NatsOption func(*natsOptions)

type natsOptions struct {
	config NatsConfig
}

// WithNatsConfig sets the complete NATS configuration.
func WithNatsConfig(cfg NatsConfig) NatsOption {
	return func(o *natsOptions) { o.config = cfg }
}

// WithNatsURL sets the NATS server URL.
func WithNatsURL(url string) NatsOption {
	return func(o *natsOptions) { o.config.URL = url }
}

// WithNatsName sets the NATS client name.
func WithNatsName(name string) NatsOption {
	return func(o *natsOptions) { o.config.Name = name }
}
