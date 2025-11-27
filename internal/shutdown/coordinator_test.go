package shutdown

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"
)

type mockService struct {
	name         string
	shutdownFn   func(ctx context.Context) error
	shutdownTime time.Duration
	called       bool
	mu           sync.Mutex
}

func newMockService(name string) *mockService {
	return &mockService{name: name}
}

func (m *mockService) Name() string { return m.name }

func (m *mockService) Shutdown(ctx context.Context) error {
	m.mu.Lock()
	m.called = true
	m.mu.Unlock()

	if m.shutdownTime > 0 {
		select {
		case <-time.After(m.shutdownTime):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	if m.shutdownFn != nil {
		return m.shutdownFn(ctx)
	}
	return nil
}

func (m *mockService) WasCalled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.called
}

func TestCoordinator_Register(t *testing.T) {
	logger := zap.NewNop()
	coord := NewCoordinator(nil, logger)

	svc1 := newMockService("svc1")
	svc2 := newMockService("svc2")

	coord.Register(PhaseShutdown, svc1)
	coord.Register(PhaseShutdown, svc2)

	if len(coord.services[PhaseShutdown]) != 2 {
		t.Errorf("expected 2 services, got %d", len(coord.services[PhaseShutdown]))
	}
}

func TestCoordinator_RegisterFunc(t *testing.T) {
	logger := zap.NewNop()
	coord := NewCoordinator(nil, logger)

	called := false
	coord.RegisterFunc(PhaseShutdown, "test", func(ctx context.Context) error {
		called = true
		return nil
	})

	ctx := context.Background()
	coord.Shutdown(ctx)

	// Wait for shutdown to complete
	time.Sleep(100 * time.Millisecond)

	if !called {
		t.Error("shutdown function was not called")
	}
}

func TestCoordinator_Shutdown_PhasesRunInOrder(t *testing.T) {
	logger := zap.NewNop()
	coord := NewCoordinator(&Config{Timeout: 5 * time.Second}, logger)

	var order []Phase
	var mu sync.Mutex

	for _, phase := range []Phase{PhasePreDrain, PhaseDrain, PhaseShutdown, PhaseCleanup} {
		p := phase // capture
		coord.RegisterFunc(p, p.String(), func(ctx context.Context) error {
			mu.Lock()
			order = append(order, p)
			mu.Unlock()
			return nil
		})
	}

	ctx := context.Background()
	if err := coord.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}

	expected := []Phase{PhasePreDrain, PhaseDrain, PhaseShutdown, PhaseCleanup}
	if len(order) != len(expected) {
		t.Fatalf("expected %d phases, got %d", len(expected), len(order))
	}

	for i, p := range expected {
		if order[i] != p {
			t.Errorf("phase %d: expected %v, got %v", i, p, order[i])
		}
	}
}

func TestCoordinator_Shutdown_ServicesInPhaseRunConcurrently(t *testing.T) {
	logger := zap.NewNop()
	coord := NewCoordinator(&Config{Timeout: 5 * time.Second}, logger)

	var concurrent int32
	var maxConcurrent int32
	var mu sync.Mutex

	for i := 0; i < 3; i++ {
		coord.RegisterFunc(PhaseShutdown, "svc", func(ctx context.Context) error {
			current := atomic.AddInt32(&concurrent, 1)
			mu.Lock()
			if current > maxConcurrent {
				maxConcurrent = current
			}
			mu.Unlock()

			time.Sleep(50 * time.Millisecond)
			atomic.AddInt32(&concurrent, -1)
			return nil
		})
	}

	ctx := context.Background()
	if err := coord.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}

	if maxConcurrent < 2 {
		t.Errorf("expected concurrent execution, maxConcurrent = %d", maxConcurrent)
	}
}

func TestCoordinator_Shutdown_CollectsErrors(t *testing.T) {
	logger := zap.NewNop()
	coord := NewCoordinator(&Config{Timeout: 5 * time.Second}, logger)

	coord.RegisterFunc(PhaseShutdown, "failing-svc", func(ctx context.Context) error {
		return errors.New("shutdown failed")
	})

	ctx := context.Background()
	// Shutdown should complete even with errors
	err := coord.Shutdown(ctx)
	if err != nil {
		t.Errorf("Shutdown() should not return error, got %v", err)
	}
}

func TestCoordinator_Shutdown_RespectsTimeout(t *testing.T) {
	logger := zap.NewNop()
	coord := NewCoordinator(&Config{Timeout: 100 * time.Millisecond}, logger)

	coord.RegisterFunc(PhaseShutdown, "slow-svc", func(ctx context.Context) error {
		select {
		case <-time.After(1 * time.Second):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})

	ctx := context.Background()
	start := time.Now()
	coord.Shutdown(ctx)

	if time.Since(start) > 500*time.Millisecond {
		t.Error("shutdown should have timed out quickly")
	}
}

func TestCoordinator_ShutdownOnlyOnce(t *testing.T) {
	logger := zap.NewNop()
	coord := NewCoordinator(nil, logger)

	var callCount int32
	coord.RegisterFunc(PhaseShutdown, "svc", func(ctx context.Context) error {
		atomic.AddInt32(&callCount, 1)
		return nil
	})

	ctx := context.Background()

	// Call shutdown multiple times
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			coord.Shutdown(ctx)
		}()
	}
	wg.Wait()

	if callCount != 1 {
		t.Errorf("expected shutdown called once, got %d", callCount)
	}
}

func TestCoordinator_ShutdownCh(t *testing.T) {
	logger := zap.NewNop()
	coord := NewCoordinator(nil, logger)

	select {
	case <-coord.ShutdownCh():
		t.Error("shutdown channel should not be closed initially")
	default:
		// Expected
	}

	go coord.Shutdown(context.Background())

	select {
	case <-coord.ShutdownCh():
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Error("shutdown channel should be closed after Shutdown()")
	}
}

func TestReadinessProbe(t *testing.T) {
	logger := zap.NewNop()
	coord := NewCoordinator(nil, logger)
	probe := NewReadinessProbe(coord)

	// Initially ready
	if !probe.IsReady() {
		t.Error("probe should be ready initially")
	}
	if probe.State() != HealthStateHealthy {
		t.Errorf("expected healthy state, got %v", probe.State())
	}

	// Trigger shutdown
	go coord.Shutdown(context.Background())
	time.Sleep(50 * time.Millisecond)

	// Should be draining
	if probe.IsReady() {
		t.Error("probe should not be ready after shutdown initiated")
	}
	if probe.State() != HealthStateDraining {
		t.Errorf("expected draining state, got %v", probe.State())
	}
}

func TestPhase_String(t *testing.T) {
	tests := []struct {
		phase    Phase
		expected string
	}{
		{PhasePreDrain, "pre-drain"},
		{PhaseDrain, "drain"},
		{PhaseShutdown, "shutdown"},
		{PhaseCleanup, "cleanup"},
		{Phase(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.phase.String(); got != tt.expected {
				t.Errorf("Phase.String() = %q, expected %q", got, tt.expected)
			}
		})
	}
}

func TestHealthState_String(t *testing.T) {
	tests := []struct {
		state    HealthState
		expected string
	}{
		{HealthStateHealthy, "healthy"},
		{HealthStateDraining, "draining"},
		{HealthStateShuttingDown, "shutting_down"},
		{HealthState(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.state.String(); got != tt.expected {
				t.Errorf("HealthState.String() = %q, expected %q", got, tt.expected)
			}
		})
	}
}
