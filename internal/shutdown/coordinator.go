// Package shutdown provides graceful shutdown coordination for services.
package shutdown

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Service represents a service that can be shutdown gracefully.
type Service interface {
	// Name returns the service name for logging.
	Name() string
	// Shutdown performs graceful shutdown. It should return when shutdown is complete.
	Shutdown(ctx context.Context) error
}

// ServiceFunc wraps a function to implement the Service interface.
type ServiceFunc struct {
	ServiceName string
	ShutdownFn  func(ctx context.Context) error
}

func (s ServiceFunc) Name() string                          { return s.ServiceName }
func (s ServiceFunc) Shutdown(ctx context.Context) error    { return s.ShutdownFn(ctx) }

// Phase represents a shutdown phase. Services in the same phase are shutdown concurrently.
type Phase int

const (
	// PhasePreDrain is the first phase - stop accepting new work.
	PhasePreDrain Phase = iota
	// PhaseDrain is for draining in-flight requests.
	PhaseDrain
	// PhaseShutdown is for shutting down background workers.
	PhaseShutdown
	// PhaseCleanup is for final cleanup (close connections, flush buffers).
	PhaseCleanup
)

func (p Phase) String() string {
	switch p {
	case PhasePreDrain:
		return "pre-drain"
	case PhaseDrain:
		return "drain"
	case PhaseShutdown:
		return "shutdown"
	case PhaseCleanup:
		return "cleanup"
	default:
		return "unknown"
	}
}

// Coordinator manages graceful shutdown of multiple services.
type Coordinator struct {
	mu       sync.Mutex
	services map[Phase][]Service
	timeout  time.Duration
	logger   *zap.Logger

	// State
	shutdownCh   chan struct{}
	shutdownOnce sync.Once
	done         chan struct{}
}

// Config holds configuration for the shutdown coordinator.
type Config struct {
	// Timeout is the total time allowed for shutdown.
	Timeout time.Duration
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Timeout: 30 * time.Second,
	}
}

// NewCoordinator creates a new shutdown coordinator.
func NewCoordinator(cfg *Config, logger *zap.Logger) *Coordinator {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	return &Coordinator{
		services:   make(map[Phase][]Service),
		timeout:    cfg.Timeout,
		logger:     logger,
		shutdownCh: make(chan struct{}),
		done:       make(chan struct{}),
	}
}

// Register adds a service to be shutdown in the specified phase.
func (c *Coordinator) Register(phase Phase, svc Service) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.services[phase] = append(c.services[phase], svc)
	c.logger.Debug("registered service for shutdown",
		zap.String("service", svc.Name()),
		zap.String("phase", phase.String()),
	)
}

// RegisterFunc is a convenience method to register a shutdown function.
func (c *Coordinator) RegisterFunc(phase Phase, name string, fn func(ctx context.Context) error) {
	c.Register(phase, ServiceFunc{ServiceName: name, ShutdownFn: fn})
}

// Shutdown initiates graceful shutdown of all registered services.
// It runs phases sequentially, but services within each phase run concurrently.
func (c *Coordinator) Shutdown(ctx context.Context) error {
	c.shutdownOnce.Do(func() {
		close(c.shutdownCh)
		go c.runShutdown(ctx)
	})

	select {
	case <-c.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// ShutdownCh returns a channel that's closed when shutdown is initiated.
func (c *Coordinator) ShutdownCh() <-chan struct{} {
	return c.shutdownCh
}

// runShutdown executes the shutdown sequence.
func (c *Coordinator) runShutdown(_ context.Context) {
	defer close(c.done)

	// Use background context for shutdown to ensure full timeout regardless of
	// caller's context state. Shutdown should get its full timeout to complete gracefully.
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	c.logger.Info("starting graceful shutdown",
		zap.Duration("timeout", c.timeout),
	)

	phases := []Phase{PhasePreDrain, PhaseDrain, PhaseShutdown, PhaseCleanup}
	var errors []error

	for _, phase := range phases {
		c.mu.Lock()
		services := c.services[phase]
		c.mu.Unlock()

		if len(services) == 0 {
			continue
		}

		c.logger.Info("executing shutdown phase",
			zap.String("phase", phase.String()),
			zap.Int("services", len(services)),
		)

		phaseErrors := c.shutdownPhase(ctx, phase, services)
		errors = append(errors, phaseErrors...)

		// Check if context is done
		if ctx.Err() != nil {
			c.logger.Error("shutdown timeout exceeded",
				zap.String("phase", phase.String()),
				zap.Error(ctx.Err()),
			)
			break
		}
	}

	if len(errors) > 0 {
		c.logger.Error("shutdown completed with errors",
			zap.Int("error_count", len(errors)),
		)
	} else {
		c.logger.Info("graceful shutdown complete")
	}
}

// shutdownPhase shuts down all services in a phase concurrently.
func (c *Coordinator) shutdownPhase(ctx context.Context, phase Phase, services []Service) []error {
	var wg sync.WaitGroup
	errCh := make(chan error, len(services))

	for _, svc := range services {
		wg.Add(1)
		go func(s Service) {
			defer wg.Done()

			start := time.Now()
			c.logger.Debug("shutting down service",
				zap.String("service", s.Name()),
				zap.String("phase", phase.String()),
			)

			if err := s.Shutdown(ctx); err != nil {
				c.logger.Error("service shutdown failed",
					zap.String("service", s.Name()),
					zap.String("phase", phase.String()),
					zap.Duration("duration", time.Since(start)),
					zap.Error(err),
				)
				errCh <- fmt.Errorf("%s: %w", s.Name(), err)
				return
			}

			c.logger.Debug("service shutdown complete",
				zap.String("service", s.Name()),
				zap.String("phase", phase.String()),
				zap.Duration("duration", time.Since(start)),
			)
		}(svc)
	}

	wg.Wait()
	close(errCh)

	errors := make([]error, 0, len(services))
	for err := range errCh {
		errors = append(errors, err)
	}

	return errors
}

// HealthState represents the shutdown health state.
type HealthState int

const (
	HealthStateHealthy HealthState = iota
	HealthStateDraining
	HealthStateShuttingDown
)

func (h HealthState) String() string {
	switch h {
	case HealthStateHealthy:
		return "healthy"
	case HealthStateDraining:
		return "draining"
	case HealthStateShuttingDown:
		return "shutting_down"
	default:
		return "unknown"
	}
}

// ReadinessProbe provides a readiness probe that respects shutdown state.
type ReadinessProbe struct {
	coordinator *Coordinator
	state       HealthState
	mu          sync.RWMutex
}

// NewReadinessProbe creates a new readiness probe.
func NewReadinessProbe(coordinator *Coordinator) *ReadinessProbe {
	rp := &ReadinessProbe{
		coordinator: coordinator,
		state:       HealthStateHealthy,
	}

	// Start watching for shutdown
	go rp.watchShutdown()

	return rp
}

func (rp *ReadinessProbe) watchShutdown() {
	<-rp.coordinator.ShutdownCh()
	rp.mu.Lock()
	rp.state = HealthStateDraining
	rp.mu.Unlock()
}

// SetState sets the health state.
func (rp *ReadinessProbe) SetState(state HealthState) {
	rp.mu.Lock()
	rp.state = state
	rp.mu.Unlock()
}

// IsReady returns true if the service is ready to accept traffic.
func (rp *ReadinessProbe) IsReady() bool {
	rp.mu.RLock()
	defer rp.mu.RUnlock()
	return rp.state == HealthStateHealthy
}

// State returns the current health state.
func (rp *ReadinessProbe) State() HealthState {
	rp.mu.RLock()
	defer rp.mu.RUnlock()
	return rp.state
}
