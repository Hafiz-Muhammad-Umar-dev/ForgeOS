// Package kernel is the DevOS kernel composition root. It loads configuration,
// builds the dependency-injection container, registers core services, and
// constructs the lifecycle manager.
//
// This is the skeleton: it wires DI + lifecycle + config and exposes the
// runtime facade. Bus/registry/scheduler client implementations (NATS, etc.)
// and tenant services are attached in later milestones (SDD §11, build order
// Step 1+). No business logic, network I/O, or authentication is performed
// here beyond configuration loading.
package kernel

import (
	"context"
	"fmt"
	"reflect"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/bus"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/config"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/di"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/lifecycle"
)

// Kernel is the composed runtime. It owns the configuration, the
// dependency-injection container, and the lifecycle manager.
type Kernel struct {
	Config    *config.Config
	Container *di.Container
	Lifecycle *lifecycle.Manager
	Bus       bus.BusPort // optional; nil when not configured
}

// Option configures kernel construction.
type Option func(*options)

type options struct {
	configOpts []config.Option
	disableBus bool
	busURL     string
}

// WithConfigOpts forwards configuration options to config.Load.
func WithConfigOpts(opts ...config.Option) Option {
	return func(o *options) { o.configOpts = append(o.configOpts, opts...) }
}

// WithBusURL overrides the NATS URL for the message bus. When set, a NATS bus
// adapter is created and registered with the lifecycle manager.
func WithBusURL(url string) Option {
	return func(o *options) { o.busURL = url }
}

// WithoutBus disables bus creation even when a NATS URL is configured.
func WithoutBus() Option {
	return func(o *options) { o.disableBus = true }
}

// New loads configuration and wires the core services into a DI container and
// lifecycle manager. It returns a ready-to-run kernel.
//
// When a bus URL is provided via WithBusURL or the DEVOS_NATS_URL environment
// variable, a NATS bus adapter is created and registered as a lifecycle
// component. Pass WithoutBus() to explicitly suppress bus creation (e.g. in
// unit tests that do not require a NATS server).
func New(opts ...Option) (*Kernel, error) {
	o := options{}
	for _, fn := range opts {
		fn(&o)
	}

	cfg, err := config.Load(o.configOpts...)
	if err != nil {
		return nil, fmt.Errorf("kernel: load config: %w", err)
	}

	container := di.New()
	container.MustRegister(reflect.TypeOf(&config.Config{}), func(*di.Container) (any, error) {
		return cfg, nil
	}, di.Singleton)
	container.MustRegister(reflect.TypeOf(&di.Container{}), func(*di.Container) (any, error) {
		return container, nil
	}, di.Singleton)

	lm := lifecycle.NewManager()

	k := &Kernel{
		Config:    cfg,
		Container: container,
		Lifecycle: lm,
	}

	// Wire the message bus when a URL is explicitly provided.
	if o.busURL != "" {
		nb := bus.NewNatsBus(bus.WithNatsURL(o.busURL))
		container.MustRegister(reflect.TypeOf((*bus.BusPort)(nil)).Elem(), func(*di.Container) (any, error) {
			return nb, nil
		}, di.Singleton)
		lm.Register(nb)
		k.Bus = nb

		// Connect the bus immediately so it is ready for use. The lifecycle
		// manager will also Start/Stop it as part of Run().
		initCtx := context.Background()
		if err := nb.Init(initCtx); err != nil {
			return nil, fmt.Errorf("kernel: bus init: %w", err)
		}
		if err := nb.Start(initCtx); err != nil {
			return nil, fmt.Errorf("kernel: bus start: %w", err)
		}
	}

	return k, nil
}

// Run starts the kernel and blocks until ctx is cancelled, then stops all
// registered components gracefully.
func (k *Kernel) Run(ctx context.Context) error {
	return k.Lifecycle.Run(ctx)
}
