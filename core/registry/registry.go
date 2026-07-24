// Package registry defines the service-registry port and supporting types
// for DevOS. It is the contract only; the NATS-KV-backed implementation is
// delivered in a later milestone (see SDD §09 and ADR-003).
//
// Tenant services depend on the Registry interface, never on a concrete
// implementation, so the backing store can be swapped without touching
// callers.
package registry

import (
	"context"
	"time"
)

// Capability expresses a feature a registered service provides.
type Capability string

// ServiceInfo describes a registered service, agent, or provider.
type ServiceInfo struct {
	ID           string
	Name         string
	Kind         string // "agent" | "provider" | "service" | "channel" | "tool"
	Capabilities []Capability
	Endpoint     string
	Metadata     map[string]string
	RegisteredAt time.Time
}

// Has reports whether info provides the given capability.
func (s ServiceInfo) Has(c Capability) bool {
	for _, c2 := range s.Capabilities {
		if c2 == c {
			return true
		}
	}
	return false
}

// CapabilitySet is a uniqueness-preserving collection of capabilities.
type CapabilitySet map[Capability]struct{}

// NewCapabilitySet builds a set from the given capabilities.
func NewCapabilitySet(caps ...Capability) CapabilitySet {
	s := make(CapabilitySet, len(caps))
	for _, c := range caps {
		s.Add(c)
	}
	return s
}

// Add inserts a capability.
func (s CapabilitySet) Add(c Capability) {
	s[c] = struct{}{}
}

// Contains reports whether c is in the set.
func (s CapabilitySet) Contains(c Capability) bool {
	_, ok := s[c]
	return ok
}

// Slice returns the capabilities as a slice (order is non-deterministic).
func (s CapabilitySet) Slice() []Capability {
	out := make([]Capability, 0, len(s))
	for c := range s {
		out = append(out, c)
	}
	return out
}

// RegistrationKind constants for ServiceInfo.Kind.
const (
	KindAgent    = "agent"
	KindTool     = "tool"
	KindProvider = "provider"
	KindService  = "service"
	KindChannel  = "channel"
)

// Registry is the service-discovery and capability-index port.
type Registry interface {
	// --- Generic methods ---

	// Register adds a service to the registry. Returns ErrAlreadyRegistered
	// when an entry with the same ID already exists.
	Register(ctx context.Context, info ServiceInfo) error

	// Deregister removes a service by ID.
	Deregister(ctx context.Context, id string) error

	// Discover returns all services of the given kind.
	Discover(ctx context.Context, kind string) ([]ServiceInfo, error)

	// Resolve returns the first service that provides the given capability.
	Resolve(ctx context.Context, capability Capability) (ServiceInfo, error)

	// ResolveByID returns the service with the given ID.
	ResolveByID(ctx context.Context, id string) (ServiceInfo, error)

	// Watch returns a channel that receives updates when services matching
	// the capability are registered or deregistered.
	Watch(ctx context.Context, capability Capability) (<-chan ServiceInfo, error)

	// --- Typed convenience methods ---

	// RegisterAgent registers a service with KindAgent.
	RegisterAgent(ctx context.Context, info ServiceInfo) error

	// RegisterTool registers a service with KindTool.
	RegisterTool(ctx context.Context, info ServiceInfo) error

	// RegisterProvider registers a service with KindProvider.
	RegisterProvider(ctx context.Context, info ServiceInfo) error

	// GetAgent returns a registered agent by ID.
	GetAgent(ctx context.Context, id string) (ServiceInfo, error)

	// GetTool returns a registered tool by ID.
	GetTool(ctx context.Context, id string) (ServiceInfo, error)

	// GetProvider returns a registered provider by ID.
	GetProvider(ctx context.Context, id string) (ServiceInfo, error)

	// ListAgents returns all registered agents.
	ListAgents(ctx context.Context) ([]ServiceInfo, error)

	// ListTools returns all registered tools.
	ListTools(ctx context.Context) ([]ServiceInfo, error)

	// ListProviders returns all registered providers.
	ListProviders(ctx context.Context) ([]ServiceInfo, error)
}
