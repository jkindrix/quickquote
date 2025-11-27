// Package clock provides a time abstraction for testable time handling.
//
// This follows the Clock pattern from the software quality taxonomy:
// Instead of calling time.Now() directly, inject a Clock interface.
// This enables deterministic testing of time-dependent code.
//
// Usage:
//
//	// In production code
//	type Service struct {
//	    clock clock.Clock
//	}
//
//	func NewService(c clock.Clock) *Service {
//	    if c == nil {
//	        c = clock.New() // Default to real clock
//	    }
//	    return &Service{clock: c}
//	}
//
//	func (s *Service) IsExpired(expiresAt time.Time) bool {
//	    return s.clock.Now().After(expiresAt)
//	}
//
//	// In tests
//	func TestIsExpired(t *testing.T) {
//	    mock := clock.NewMock(time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC))
//	    svc := NewService(mock)
//
//	    past := time.Date(2024, 1, 14, 12, 0, 0, 0, time.UTC)
//	    if !svc.IsExpired(past) {
//	        t.Error("expected past time to be expired")
//	    }
//
//	    // Advance time
//	    mock.Advance(24 * time.Hour)
//	    // Now is 2024-01-16
//	}
package clock

import (
	"sync"
	"time"
)

// Clock provides time operations that can be mocked for testing.
type Clock interface {
	// Now returns the current time.
	Now() time.Time

	// NowUTC returns the current time in UTC.
	// Preferred over Now() for storage operations.
	NowUTC() time.Time

	// Since returns the time elapsed since t.
	Since(t time.Time) time.Duration

	// Until returns the duration until t.
	Until(t time.Time) time.Duration

	// After waits for the duration to elapse and then sends the current time
	// on the returned channel.
	After(d time.Duration) <-chan time.Time

	// NewTicker returns a new Ticker.
	NewTicker(d time.Duration) Ticker

	// NewTimer returns a new Timer.
	NewTimer(d time.Duration) Timer
}

// Ticker wraps time.Ticker for mockability.
type Ticker interface {
	C() <-chan time.Time
	Stop()
	Reset(d time.Duration)
}

// Timer wraps time.Timer for mockability.
type Timer interface {
	C() <-chan time.Time
	Stop() bool
	Reset(d time.Duration) bool
}

// realClock implements Clock using the standard time package.
type realClock struct{}

// New returns a Clock that uses the real system time.
func New() Clock {
	return &realClock{}
}

func (c *realClock) Now() time.Time {
	return time.Now()
}

func (c *realClock) NowUTC() time.Time {
	return time.Now().UTC()
}

func (c *realClock) Since(t time.Time) time.Duration {
	return time.Since(t)
}

func (c *realClock) Until(t time.Time) time.Duration {
	return time.Until(t)
}

func (c *realClock) After(d time.Duration) <-chan time.Time {
	return time.After(d)
}

func (c *realClock) NewTicker(d time.Duration) Ticker {
	return &realTicker{ticker: time.NewTicker(d)}
}

func (c *realClock) NewTimer(d time.Duration) Timer {
	return &realTimer{timer: time.NewTimer(d)}
}

// realTicker wraps time.Ticker.
type realTicker struct {
	ticker *time.Ticker
}

func (t *realTicker) C() <-chan time.Time {
	return t.ticker.C
}

func (t *realTicker) Stop() {
	t.ticker.Stop()
}

func (t *realTicker) Reset(d time.Duration) {
	t.ticker.Reset(d)
}

// realTimer wraps time.Timer.
type realTimer struct {
	timer *time.Timer
}

func (t *realTimer) C() <-chan time.Time {
	return t.timer.C
}

func (t *realTimer) Stop() bool {
	return t.timer.Stop()
}

func (t *realTimer) Reset(d time.Duration) bool {
	return t.timer.Reset(d)
}

// Mock implements Clock with controllable time for testing.
type Mock struct {
	mu      sync.RWMutex
	current time.Time
}

// NewMock creates a new Mock clock set to the given time.
func NewMock(t time.Time) *Mock {
	return &Mock{current: t}
}

// Now returns the mock's current time.
func (m *Mock) Now() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.current
}

// NowUTC returns the mock's current time in UTC.
func (m *Mock) NowUTC() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.current.UTC()
}

// Since returns the duration since t.
func (m *Mock) Since(t time.Time) time.Duration {
	return m.Now().Sub(t)
}

// Until returns the duration until t.
func (m *Mock) Until(t time.Time) time.Duration {
	return t.Sub(m.Now())
}

// After returns a channel that receives after the duration.
// For mock, this returns immediately with current time + duration.
func (m *Mock) After(d time.Duration) <-chan time.Time {
	ch := make(chan time.Time, 1)
	ch <- m.Now().Add(d)
	return ch
}

// NewTicker returns a mock ticker.
// Note: Mock ticker doesn't actually tick - use for interface compatibility.
func (m *Mock) NewTicker(d time.Duration) Ticker {
	return &mockTicker{
		ch: make(chan time.Time),
	}
}

// NewTimer returns a mock timer.
func (m *Mock) NewTimer(d time.Duration) Timer {
	return &mockTimer{
		ch: make(chan time.Time, 1),
	}
}

// Set sets the mock clock to a specific time.
func (m *Mock) Set(t time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.current = t
}

// Advance moves the mock clock forward by the given duration.
func (m *Mock) Advance(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.current = m.current.Add(d)
}

// mockTicker is a non-ticking ticker for tests.
type mockTicker struct {
	ch chan time.Time
}

func (t *mockTicker) C() <-chan time.Time {
	return t.ch
}

func (t *mockTicker) Stop() {}

func (t *mockTicker) Reset(d time.Duration) {}

// mockTimer is a mock timer for tests.
type mockTimer struct {
	ch chan time.Time
}

func (t *mockTimer) C() <-chan time.Time {
	return t.ch
}

func (t *mockTimer) Stop() bool {
	return true
}

func (t *mockTimer) Reset(d time.Duration) bool {
	return true
}
