package clock

import (
	"testing"
	"time"
)

func TestRealClock_Now(t *testing.T) {
	c := New()

	before := time.Now()
	got := c.Now()
	after := time.Now()

	if got.Before(before) || got.After(after) {
		t.Errorf("Now() = %v, want between %v and %v", got, before, after)
	}
}

func TestRealClock_NowUTC(t *testing.T) {
	c := New()

	got := c.NowUTC()

	if got.Location() != time.UTC {
		t.Errorf("NowUTC() location = %v, want UTC", got.Location())
	}
}

func TestRealClock_Since(t *testing.T) {
	c := New()
	past := time.Now().Add(-time.Second)

	got := c.Since(past)

	if got < time.Second {
		t.Errorf("Since() = %v, want >= 1s", got)
	}
}

func TestRealClock_Until(t *testing.T) {
	c := New()
	future := time.Now().Add(time.Hour)

	got := c.Until(future)

	if got < 59*time.Minute {
		t.Errorf("Until() = %v, want >= 59m", got)
	}
}

func TestMock_Now(t *testing.T) {
	fixed := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	m := NewMock(fixed)

	got := m.Now()

	if !got.Equal(fixed) {
		t.Errorf("Now() = %v, want %v", got, fixed)
	}
}

func TestMock_NowUTC(t *testing.T) {
	// Create time in non-UTC timezone
	loc, _ := time.LoadLocation("America/New_York")
	fixed := time.Date(2024, 1, 15, 12, 0, 0, 0, loc)
	m := NewMock(fixed)

	got := m.NowUTC()

	if got.Location() != time.UTC {
		t.Errorf("NowUTC() location = %v, want UTC", got.Location())
	}
}

func TestMock_Set(t *testing.T) {
	initial := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	newTime := time.Date(2024, 6, 20, 18, 30, 0, 0, time.UTC)
	m := NewMock(initial)

	m.Set(newTime)
	got := m.Now()

	if !got.Equal(newTime) {
		t.Errorf("Now() after Set() = %v, want %v", got, newTime)
	}
}

func TestMock_Advance(t *testing.T) {
	initial := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	m := NewMock(initial)

	m.Advance(24 * time.Hour)
	got := m.Now()

	expected := time.Date(2024, 1, 16, 12, 0, 0, 0, time.UTC)
	if !got.Equal(expected) {
		t.Errorf("Now() after Advance(24h) = %v, want %v", got, expected)
	}
}

func TestMock_Since(t *testing.T) {
	now := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	past := time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC)
	m := NewMock(now)

	got := m.Since(past)

	if got != time.Hour {
		t.Errorf("Since() = %v, want 1h", got)
	}
}

func TestMock_Until(t *testing.T) {
	now := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	future := time.Date(2024, 1, 15, 14, 0, 0, 0, time.UTC)
	m := NewMock(now)

	got := m.Until(future)

	if got != 2*time.Hour {
		t.Errorf("Until() = %v, want 2h", got)
	}
}

func TestMock_After(t *testing.T) {
	now := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	m := NewMock(now)

	ch := m.After(time.Hour)
	got := <-ch

	expected := now.Add(time.Hour)
	if !got.Equal(expected) {
		t.Errorf("After(1h) = %v, want %v", got, expected)
	}
}

func TestMock_Concurrent(t *testing.T) {
	m := NewMock(time.Now())

	done := make(chan struct{})

	// Reader goroutine
	go func() {
		for i := 0; i < 1000; i++ {
			_ = m.Now()
		}
		done <- struct{}{}
	}()

	// Writer goroutine
	go func() {
		for i := 0; i < 1000; i++ {
			m.Advance(time.Millisecond)
		}
		done <- struct{}{}
	}()

	<-done
	<-done
}

// Example of using Mock in a test
func TestExampleUsage(t *testing.T) {
	// Create a mock clock at a fixed time
	fixedTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	clock := NewMock(fixedTime)

	// Check if something is expired
	expiresAt := time.Date(2024, 1, 14, 12, 0, 0, 0, time.UTC) // Yesterday

	if !clock.Now().After(expiresAt) {
		t.Error("expected past time to be expired")
	}

	// Advance time by 1 day
	clock.Advance(24 * time.Hour)

	// Now is 2024-01-16
	if !clock.Now().Equal(time.Date(2024, 1, 16, 12, 0, 0, 0, time.UTC)) {
		t.Errorf("unexpected time after advance: %v", clock.Now())
	}
}
