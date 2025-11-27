// Package circuitbreaker provides circuit breaker pattern implementation for fault tolerance.
package circuitbreaker

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.uber.org/zap"
)

// State represents the circuit breaker state.
type State int

const (
	StateClosed State = iota // Normal operation, requests go through
	StateOpen                // Circuit is open, requests fail fast
	StateHalfOpen            // Testing if the service has recovered
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// Errors returned by the circuit breaker.
var (
	ErrCircuitOpen    = errors.New("circuit breaker is open")
	ErrTooManyRequests = errors.New("too many requests in half-open state")
)

// Config holds circuit breaker configuration.
type Config struct {
	// FailureThreshold is the number of consecutive failures before opening the circuit.
	FailureThreshold int
	// SuccessThreshold is the number of consecutive successes needed in half-open to close.
	SuccessThreshold int
	// OpenTimeout is how long the circuit stays open before testing recovery.
	OpenTimeout time.Duration
	// HalfOpenMaxRequests is the maximum number of requests allowed in half-open state.
	HalfOpenMaxRequests int
}

// DefaultConfig returns sensible defaults for a circuit breaker.
func DefaultConfig() *Config {
	return &Config{
		FailureThreshold:    5,
		SuccessThreshold:    3,
		OpenTimeout:         30 * time.Second,
		HalfOpenMaxRequests: 3,
	}
}

// CircuitBreaker implements the circuit breaker pattern.
type CircuitBreaker struct {
	mu sync.RWMutex

	// Configuration
	config *Config

	// State
	state             State
	consecutiveFailures int
	consecutiveSuccesses int
	halfOpenRequests    int
	lastFailure        time.Time
	lastStateChange    time.Time

	// Metrics
	totalRequests      int64
	totalSuccesses     int64
	totalFailures      int64
	totalRejected      int64
	lastError          error

	logger *zap.Logger
	name   string
}

// New creates a new circuit breaker.
func New(name string, config *Config, logger *zap.Logger) *CircuitBreaker {
	if config == nil {
		config = DefaultConfig()
	}

	return &CircuitBreaker{
		name:            name,
		config:          config,
		state:           StateClosed,
		lastStateChange: time.Now(),
		logger:          logger,
	}
}

// Execute runs the given function within the circuit breaker's protection.
// Returns ErrCircuitOpen if the circuit is open.
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func(context.Context) error) error {
	if err := cb.beforeRequest(); err != nil {
		return err
	}

	// Execute the function
	err := fn(ctx)

	cb.afterRequest(err)
	return err
}

// beforeRequest checks if the request should be allowed.
func (cb *CircuitBreaker) beforeRequest() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.totalRequests++
	now := time.Now()

	switch cb.state {
	case StateClosed:
		return nil

	case StateOpen:
		// Check if we should transition to half-open
		if now.Sub(cb.lastFailure) >= cb.config.OpenTimeout {
			cb.setState(StateHalfOpen)
			cb.halfOpenRequests = 1
			cb.logger.Info("circuit breaker transitioning to half-open",
				zap.String("name", cb.name),
				zap.Duration("after", now.Sub(cb.lastFailure)),
			)
			return nil
		}
		cb.totalRejected++
		return ErrCircuitOpen

	case StateHalfOpen:
		if cb.halfOpenRequests >= cb.config.HalfOpenMaxRequests {
			cb.totalRejected++
			return ErrTooManyRequests
		}
		cb.halfOpenRequests++
		return nil
	}

	return nil
}

// afterRequest records the result of a request.
func (cb *CircuitBreaker) afterRequest(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.recordFailure(err)
	} else {
		cb.recordSuccess()
	}
}

// recordFailure handles a failed request.
func (cb *CircuitBreaker) recordFailure(err error) {
	cb.totalFailures++
	cb.consecutiveFailures++
	cb.consecutiveSuccesses = 0
	cb.lastFailure = time.Now()
	cb.lastError = err

	switch cb.state {
	case StateClosed:
		if cb.consecutiveFailures >= cb.config.FailureThreshold {
			cb.setState(StateOpen)
			cb.logger.Warn("circuit breaker opened",
				zap.String("name", cb.name),
				zap.Int("consecutive_failures", cb.consecutiveFailures),
				zap.Error(err),
			)
		}

	case StateHalfOpen:
		// Single failure in half-open reopens the circuit
		cb.setState(StateOpen)
		cb.logger.Warn("circuit breaker reopened from half-open",
			zap.String("name", cb.name),
			zap.Error(err),
		)
	}
}

// recordSuccess handles a successful request.
func (cb *CircuitBreaker) recordSuccess() {
	cb.totalSuccesses++
	cb.consecutiveSuccesses++
	cb.consecutiveFailures = 0

	if cb.state == StateHalfOpen {
		if cb.consecutiveSuccesses >= cb.config.SuccessThreshold {
			cb.setState(StateClosed)
			cb.logger.Info("circuit breaker closed",
				zap.String("name", cb.name),
				zap.Int("consecutive_successes", cb.consecutiveSuccesses),
			)
		}
	}
}

// setState changes the circuit breaker state.
func (cb *CircuitBreaker) setState(newState State) {
	cb.state = newState
	cb.lastStateChange = time.Now()
	cb.consecutiveFailures = 0
	cb.consecutiveSuccesses = 0
	cb.halfOpenRequests = 0
}

// State returns the current state.
func (cb *CircuitBreaker) State() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// IsOpen returns true if the circuit is open.
func (cb *CircuitBreaker) IsOpen() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state == StateOpen
}

// Stats returns current circuit breaker statistics.
func (cb *CircuitBreaker) Stats() Stats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	var lastError string
	if cb.lastError != nil {
		lastError = cb.lastError.Error()
	}

	return Stats{
		Name:                 cb.name,
		State:                cb.state.String(),
		TotalRequests:        cb.totalRequests,
		TotalSuccesses:       cb.totalSuccesses,
		TotalFailures:        cb.totalFailures,
		TotalRejected:        cb.totalRejected,
		ConsecutiveFailures:  cb.consecutiveFailures,
		ConsecutiveSuccesses: cb.consecutiveSuccesses,
		LastFailure:          cb.lastFailure,
		LastStateChange:      cb.lastStateChange,
		LastError:            lastError,
	}
}

// Stats holds circuit breaker statistics.
type Stats struct {
	Name                 string    `json:"name"`
	State                string    `json:"state"`
	TotalRequests        int64     `json:"total_requests"`
	TotalSuccesses       int64     `json:"total_successes"`
	TotalFailures        int64     `json:"total_failures"`
	TotalRejected        int64     `json:"total_rejected"`
	ConsecutiveFailures  int       `json:"consecutive_failures"`
	ConsecutiveSuccesses int       `json:"consecutive_successes"`
	LastFailure          time.Time `json:"last_failure,omitempty"`
	LastStateChange      time.Time `json:"last_state_change"`
	LastError            string    `json:"last_error,omitempty"`
}

// Reset forces the circuit breaker to the closed state.
// Use with caution - this should only be called for administrative purposes.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	oldState := cb.state
	cb.setState(StateClosed)
	cb.totalRejected = 0
	cb.lastError = nil

	cb.logger.Info("circuit breaker reset",
		zap.String("name", cb.name),
		zap.String("from_state", oldState.String()),
	)
}

// ShouldRetry determines if an error should trigger circuit breaker failure tracking.
// Some errors (like context cancellation) shouldn't count against the circuit.
func ShouldRetry(err error) bool {
	if err == nil {
		return false
	}

	// Don't count cancellation/timeout from client side
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// Don't count circuit breaker's own errors
	if errors.Is(err, ErrCircuitOpen) || errors.Is(err, ErrTooManyRequests) {
		return false
	}

	return true
}
