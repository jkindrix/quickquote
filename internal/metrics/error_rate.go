// Package metrics provides error rate tracking with time-windowed analysis.
package metrics

import (
	"sync"
	"sync/atomic"
	"time"
)

// ErrorCategory represents different categories of errors for tracking.
type ErrorCategory string

const (
	ErrorCategoryDatabase   ErrorCategory = "database"
	ErrorCategoryHTTP       ErrorCategory = "http"
	ErrorCategoryValidation ErrorCategory = "validation"
	ErrorCategoryExternal   ErrorCategory = "external"
	ErrorCategoryInternal   ErrorCategory = "internal"
	ErrorCategoryAuth       ErrorCategory = "auth"
	ErrorCategoryRateLimit  ErrorCategory = "rate_limit"
)

// ErrorRateConfig configures the error rate tracker.
type ErrorRateConfig struct {
	// WindowDuration is the time window for rate calculation (default: 1 minute)
	WindowDuration time.Duration

	// BucketCount is the number of buckets within the window (default: 60)
	// More buckets = finer granularity but more memory
	BucketCount int

	// AlertThreshold is the error rate (errors/second) that triggers alerts (default: 10)
	AlertThreshold float64

	// AlertCallback is called when error rate exceeds threshold
	AlertCallback func(category ErrorCategory, rate float64)
}

// DefaultErrorRateConfig returns sensible defaults.
func DefaultErrorRateConfig() ErrorRateConfig {
	return ErrorRateConfig{
		WindowDuration: time.Minute,
		BucketCount:    60,
		AlertThreshold: 10.0,
		AlertCallback:  nil,
	}
}

// ErrorRateTracker tracks error rates across different categories.
type ErrorRateTracker struct {
	config   ErrorRateConfig
	counters map[ErrorCategory]*slidingWindow
	mu       sync.RWMutex

	// Aggregate counters for total errors
	totalErrors   atomic.Int64
	totalRequests atomic.Int64
}

// NewErrorRateTracker creates a new error rate tracker.
func NewErrorRateTracker(config ErrorRateConfig) *ErrorRateTracker {
	if config.WindowDuration == 0 {
		config.WindowDuration = time.Minute
	}
	if config.BucketCount == 0 {
		config.BucketCount = 60
	}

	return &ErrorRateTracker{
		config:   config,
		counters: make(map[ErrorCategory]*slidingWindow),
	}
}

// RecordError records an error in the specified category.
func (t *ErrorRateTracker) RecordError(category ErrorCategory) {
	t.totalErrors.Add(1)
	t.getOrCreateWindow(category).increment()

	// Check if we should trigger an alert
	if t.config.AlertCallback != nil {
		rate := t.Rate(category)
		if rate > t.config.AlertThreshold {
			t.config.AlertCallback(category, rate)
		}
	}
}

// RecordRequest records a request (for calculating error percentage).
func (t *ErrorRateTracker) RecordRequest() {
	t.totalRequests.Add(1)
}

// Rate returns the current error rate (errors per second) for a category.
func (t *ErrorRateTracker) Rate(category ErrorCategory) float64 {
	t.mu.RLock()
	window, ok := t.counters[category]
	t.mu.RUnlock()

	if !ok {
		return 0
	}

	count := window.count()
	return float64(count) / t.config.WindowDuration.Seconds()
}

// Count returns the error count in the current window for a category.
func (t *ErrorRateTracker) Count(category ErrorCategory) int64 {
	t.mu.RLock()
	window, ok := t.counters[category]
	t.mu.RUnlock()

	if !ok {
		return 0
	}

	return window.count()
}

// TotalRate returns the aggregate error rate across all categories.
func (t *ErrorRateTracker) TotalRate() float64 {
	var total int64
	t.mu.RLock()
	for _, window := range t.counters {
		total += window.count()
	}
	t.mu.RUnlock()

	return float64(total) / t.config.WindowDuration.Seconds()
}

// ErrorPercentage returns the percentage of requests that resulted in errors.
// Returns 0 if no requests have been recorded.
func (t *ErrorRateTracker) ErrorPercentage() float64 {
	requests := t.totalRequests.Load()
	if requests == 0 {
		return 0
	}
	errors := t.totalErrors.Load()
	return (float64(errors) / float64(requests)) * 100
}

// Snapshot returns a point-in-time snapshot of all error rates.
func (t *ErrorRateTracker) Snapshot() map[ErrorCategory]ErrorRateSnapshot {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make(map[ErrorCategory]ErrorRateSnapshot, len(t.counters))
	for category, window := range t.counters {
		count := window.count()
		result[category] = ErrorRateSnapshot{
			Category: category,
			Count:    count,
			Rate:     float64(count) / t.config.WindowDuration.Seconds(),
		}
	}

	return result
}

// Categories returns all categories with recorded errors.
func (t *ErrorRateTracker) Categories() []ErrorCategory {
	t.mu.RLock()
	defer t.mu.RUnlock()

	categories := make([]ErrorCategory, 0, len(t.counters))
	for category := range t.counters {
		categories = append(categories, category)
	}
	return categories
}

// Reset clears all error counters (useful for testing).
func (t *ErrorRateTracker) Reset() {
	t.mu.Lock()
	t.counters = make(map[ErrorCategory]*slidingWindow)
	t.mu.Unlock()

	t.totalErrors.Store(0)
	t.totalRequests.Store(0)
}

// ErrorRateSnapshot represents a point-in-time error rate for a category.
type ErrorRateSnapshot struct {
	Category ErrorCategory
	Count    int64
	Rate     float64
}

// getOrCreateWindow gets or creates a sliding window for a category.
func (t *ErrorRateTracker) getOrCreateWindow(category ErrorCategory) *slidingWindow {
	t.mu.RLock()
	window, ok := t.counters[category]
	t.mu.RUnlock()

	if ok {
		return window
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	// Double-check after acquiring write lock
	if window, ok = t.counters[category]; ok {
		return window
	}

	window = newSlidingWindow(t.config.WindowDuration, t.config.BucketCount)
	t.counters[category] = window
	return window
}

// slidingWindow implements a time-based sliding window counter.
type slidingWindow struct {
	mu           sync.Mutex
	buckets      []int64
	bucketDur    time.Duration
	windowDur    time.Duration
	currentIndex int
	lastUpdate   time.Time
}

// newSlidingWindow creates a new sliding window.
func newSlidingWindow(windowDur time.Duration, bucketCount int) *slidingWindow {
	return &slidingWindow{
		buckets:    make([]int64, bucketCount),
		bucketDur:  windowDur / time.Duration(bucketCount),
		windowDur:  windowDur,
		lastUpdate: time.Now(),
	}
}

// increment adds one to the current bucket.
func (w *slidingWindow) increment() {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.rotate()
	w.buckets[w.currentIndex]++
}

// count returns the total count across all buckets.
func (w *slidingWindow) count() int64 {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.rotate()

	var total int64
	for _, count := range w.buckets {
		total += count
	}
	return total
}

// rotate advances the window if needed, clearing old buckets.
func (w *slidingWindow) rotate() {
	now := time.Now()
	elapsed := now.Sub(w.lastUpdate)

	// How many buckets have passed
	bucketsPassed := int(elapsed / w.bucketDur)
	if bucketsPassed == 0 {
		return
	}

	// Cap at window size to avoid unnecessary iterations
	if bucketsPassed > len(w.buckets) {
		bucketsPassed = len(w.buckets)
	}

	// Clear buckets that have expired
	for i := 0; i < bucketsPassed; i++ {
		w.currentIndex = (w.currentIndex + 1) % len(w.buckets)
		w.buckets[w.currentIndex] = 0
	}

	w.lastUpdate = now
}
