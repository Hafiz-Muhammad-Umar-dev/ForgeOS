package ingress

import (
	"time"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/bus"
)

// RESTConfig configures the REST HTTP adapter.
type RESTConfig struct {
	// ListenAddr is the TCP address for the HTTP server (e.g., ":8080").
	ListenAddr string

	// Bus is the message bus for publishing intent.created events.
	Bus bus.BusPort

	// MaxTextLength is the maximum allowed intent text length. Longer
	// texts are rejected with ErrInvalidRequest. Zero means 10000.
	MaxTextLength int

	// ReadTimeout is the HTTP request read timeout.
	ReadTimeout time.Duration

	// WriteTimeout is the HTTP response write timeout.
	WriteTimeout time.Duration

	// ShutdownTimeout is the maximum time to wait for in-flight requests
	// to complete during graceful shutdown.
	ShutdownTimeout time.Duration
}

// DefaultRESTConfig returns a sensible default configuration.
func DefaultRESTConfig() RESTConfig {
	return RESTConfig{
		ListenAddr:      ":8080",
		MaxTextLength:   10000,
		ReadTimeout:     10 * time.Second,
		WriteTimeout:    10 * time.Second,
		ShutdownTimeout: 5 * time.Second,
	}
}

// RESTOption configures the REST adapter.
type RESTOption func(*RESTConfig)

// WithListenAddr sets the HTTP listen address.
func WithListenAddr(addr string) RESTOption {
	return func(c *RESTConfig) { c.ListenAddr = addr }
}

// WithIngressBus sets the message bus for the REST adapter.
func WithIngressBus(b bus.BusPort) RESTOption {
	return func(c *RESTConfig) { c.Bus = b }
}

// WithMaxTextLength sets the maximum allowed intent text length.
func WithMaxTextLength(n int) RESTOption {
	return func(c *RESTConfig) { c.MaxTextLength = n }
}
