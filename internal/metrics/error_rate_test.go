package metrics

import (
	"sync"
	"testing"
	"time"
)

func TestNewErrorRateTracker(t *testing.T) {
	t.Run("with default config", func(t *testing.T) {
		config := DefaultErrorRateConfig()
		tracker := NewErrorRateTracker(config)

		if tracker == nil {
			t.Fatal("expected non-nil tracker")
		}
		if tracker.config.WindowDuration != time.Minute {
			t.Errorf("expected 1 minute window, got %v", tracker.config.WindowDuration)
		}
		if tracker.config.BucketCount != 60 {
			t.Errorf("expected 60 buckets, got %d", tracker.config.BucketCount)
		}
	})

	t.Run("with zero values uses defaults", func(t *testing.T) {
		tracker := NewErrorRateTracker(ErrorRateConfig{})

		if tracker.config.WindowDuration != time.Minute {
			t.Errorf("expected default 1 minute window, got %v", tracker.config.WindowDuration)
		}
		if tracker.config.BucketCount != 60 {
			t.Errorf("expected default 60 buckets, got %d", tracker.config.BucketCount)
		}
	})
}

func TestErrorRateTracker_RecordError(t *testing.T) {
	tracker := NewErrorRateTracker(ErrorRateConfig{
		WindowDuration: time.Second,
		BucketCount:    10,
	})

	// Record some errors
	tracker.RecordError(ErrorCategoryDatabase)
	tracker.RecordError(ErrorCategoryDatabase)
	tracker.RecordError(ErrorCategoryHTTP)

	// Check counts
	if count := tracker.Count(ErrorCategoryDatabase); count != 2 {
		t.Errorf("expected 2 database errors, got %d", count)
	}
	if count := tracker.Count(ErrorCategoryHTTP); count != 1 {
		t.Errorf("expected 1 HTTP error, got %d", count)
	}
	if count := tracker.Count(ErrorCategoryValidation); count != 0 {
		t.Errorf("expected 0 validation errors, got %d", count)
	}
}

func TestErrorRateTracker_Rate(t *testing.T) {
	tracker := NewErrorRateTracker(ErrorRateConfig{
		WindowDuration: time.Second,
		BucketCount:    10,
	})

	// Record 5 errors
	for i := 0; i < 5; i++ {
		tracker.RecordError(ErrorCategoryDatabase)
	}

	// Rate should be 5 errors per second
	rate := tracker.Rate(ErrorCategoryDatabase)
	if rate != 5.0 {
		t.Errorf("expected rate of 5.0, got %f", rate)
	}

	// Non-existent category should return 0
	rate = tracker.Rate(ErrorCategoryInternal)
	if rate != 0 {
		t.Errorf("expected rate of 0 for non-existent category, got %f", rate)
	}
}

func TestErrorRateTracker_TotalRate(t *testing.T) {
	tracker := NewErrorRateTracker(ErrorRateConfig{
		WindowDuration: time.Second,
		BucketCount:    10,
	})

	tracker.RecordError(ErrorCategoryDatabase)
	tracker.RecordError(ErrorCategoryDatabase)
	tracker.RecordError(ErrorCategoryHTTP)
	tracker.RecordError(ErrorCategoryValidation)
	tracker.RecordError(ErrorCategoryExternal)

	rate := tracker.TotalRate()
	if rate != 5.0 {
		t.Errorf("expected total rate of 5.0, got %f", rate)
	}
}

func TestErrorRateTracker_ErrorPercentage(t *testing.T) {
	tracker := NewErrorRateTracker(ErrorRateConfig{
		WindowDuration: time.Second,
		BucketCount:    10,
	})

	// No requests yet
	if pct := tracker.ErrorPercentage(); pct != 0 {
		t.Errorf("expected 0%% with no requests, got %f%%", pct)
	}

	// Record 100 requests
	for i := 0; i < 100; i++ {
		tracker.RecordRequest()
	}

	// Record 5 errors
	for i := 0; i < 5; i++ {
		tracker.RecordError(ErrorCategoryHTTP)
	}

	// Should be 5%
	if pct := tracker.ErrorPercentage(); pct != 5.0 {
		t.Errorf("expected 5%% error rate, got %f%%", pct)
	}
}

func TestErrorRateTracker_Snapshot(t *testing.T) {
	tracker := NewErrorRateTracker(ErrorRateConfig{
		WindowDuration: time.Second,
		BucketCount:    10,
	})

	tracker.RecordError(ErrorCategoryDatabase)
	tracker.RecordError(ErrorCategoryDatabase)
	tracker.RecordError(ErrorCategoryHTTP)

	snapshot := tracker.Snapshot()

	if len(snapshot) != 2 {
		t.Errorf("expected 2 categories in snapshot, got %d", len(snapshot))
	}

	if snapshot[ErrorCategoryDatabase].Count != 2 {
		t.Errorf("expected 2 database errors, got %d", snapshot[ErrorCategoryDatabase].Count)
	}

	if snapshot[ErrorCategoryHTTP].Count != 1 {
		t.Errorf("expected 1 HTTP error, got %d", snapshot[ErrorCategoryHTTP].Count)
	}
}

func TestErrorRateTracker_Categories(t *testing.T) {
	tracker := NewErrorRateTracker(ErrorRateConfig{
		WindowDuration: time.Second,
		BucketCount:    10,
	})

	tracker.RecordError(ErrorCategoryDatabase)
	tracker.RecordError(ErrorCategoryHTTP)

	categories := tracker.Categories()

	if len(categories) != 2 {
		t.Errorf("expected 2 categories, got %d", len(categories))
	}

	found := make(map[ErrorCategory]bool)
	for _, cat := range categories {
		found[cat] = true
	}

	if !found[ErrorCategoryDatabase] {
		t.Error("expected database category")
	}
	if !found[ErrorCategoryHTTP] {
		t.Error("expected HTTP category")
	}
}

func TestErrorRateTracker_Reset(t *testing.T) {
	tracker := NewErrorRateTracker(ErrorRateConfig{
		WindowDuration: time.Second,
		BucketCount:    10,
	})

	tracker.RecordError(ErrorCategoryDatabase)
	tracker.RecordRequest()

	tracker.Reset()

	if count := tracker.Count(ErrorCategoryDatabase); count != 0 {
		t.Errorf("expected 0 errors after reset, got %d", count)
	}
	if pct := tracker.ErrorPercentage(); pct != 0 {
		t.Errorf("expected 0%% after reset, got %f%%", pct)
	}
}

func TestErrorRateTracker_AlertCallback(t *testing.T) {
	var alertCalled bool
	var alertCategory ErrorCategory
	var alertRate float64

	tracker := NewErrorRateTracker(ErrorRateConfig{
		WindowDuration: time.Second,
		BucketCount:    10,
		AlertThreshold: 2.0, // 2 errors per second
		AlertCallback: func(category ErrorCategory, rate float64) {
			alertCalled = true
			alertCategory = category
			alertRate = rate
		},
	})

	// First error - rate is 1/s, below threshold
	tracker.RecordError(ErrorCategoryDatabase)
	if alertCalled {
		t.Error("alert should not be called below threshold")
	}

	// Second error - rate is 2/s, at threshold
	tracker.RecordError(ErrorCategoryDatabase)
	if alertCalled {
		t.Error("alert should not be called at threshold")
	}

	// Third error - rate is 3/s, above threshold
	tracker.RecordError(ErrorCategoryDatabase)
	if !alertCalled {
		t.Error("alert should be called above threshold")
	}
	if alertCategory != ErrorCategoryDatabase {
		t.Errorf("expected database category in alert, got %s", alertCategory)
	}
	if alertRate != 3.0 {
		t.Errorf("expected rate of 3.0 in alert, got %f", alertRate)
	}
}

func TestErrorRateTracker_Concurrent(t *testing.T) {
	tracker := NewErrorRateTracker(ErrorRateConfig{
		WindowDuration: time.Second,
		BucketCount:    10,
	})

	var wg sync.WaitGroup
	goroutines := 10
	errorsPerGoroutine := 100

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < errorsPerGoroutine; j++ {
				tracker.RecordError(ErrorCategoryHTTP)
				tracker.RecordRequest()
			}
		}()
	}

	wg.Wait()

	expectedCount := int64(goroutines * errorsPerGoroutine)
	actualCount := tracker.Count(ErrorCategoryHTTP)
	if actualCount != expectedCount {
		t.Errorf("expected %d errors, got %d", expectedCount, actualCount)
	}
}

func TestSlidingWindow(t *testing.T) {
	t.Run("basic increment and count", func(t *testing.T) {
		w := newSlidingWindow(time.Second, 10)

		w.increment()
		w.increment()
		w.increment()

		if count := w.count(); count != 3 {
			t.Errorf("expected count of 3, got %d", count)
		}
	})

	t.Run("bucket rotation", func(t *testing.T) {
		// Use a very short window for testing
		w := newSlidingWindow(100*time.Millisecond, 10)

		w.increment()
		w.increment()

		// Wait for window to expire
		time.Sleep(150 * time.Millisecond)

		// Trigger rotation by counting
		if count := w.count(); count != 0 {
			t.Errorf("expected count of 0 after window expired, got %d", count)
		}
	})
}

func TestErrorCategories(t *testing.T) {
	// Verify all categories are distinct
	categories := []ErrorCategory{
		ErrorCategoryDatabase,
		ErrorCategoryHTTP,
		ErrorCategoryValidation,
		ErrorCategoryExternal,
		ErrorCategoryInternal,
		ErrorCategoryAuth,
		ErrorCategoryRateLimit,
	}

	seen := make(map[ErrorCategory]bool)
	for _, cat := range categories {
		if seen[cat] {
			t.Errorf("duplicate category: %s", cat)
		}
		seen[cat] = true

		if cat == "" {
			t.Error("category should not be empty")
		}
	}
}
