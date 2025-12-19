package tracker

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/zuhrulumam/csv_processor/internal/models"
)

func TestProgressTracker_BasicTracking(t *testing.T) {
	buf := &bytes.Buffer{}

	tracker := NewProgressTracker(Config{
		Writer:         buf,
		UpdateInterval: 100 * time.Millisecond,
		TotalRecords:   100,
	})

	if err := tracker.Start(); err != nil {
		t.Fatalf("failed to start tracker: %v", err)
	}
	defer tracker.Stop()

	// Simulate processing
	for i := 0; i < 10; i++ {
		tracker.IncrementSuccess()
	}

	for i := 0; i < 2; i++ {
		tracker.IncrementFailed()
	}

	// Wait for update
	time.Sleep(150 * time.Millisecond)

	if tracker.Processed() != 12 {
		t.Errorf("expected 12 processed, got %d", tracker.Processed())
	}

	if tracker.Success() != 10 {
		t.Errorf("expected 10 success, got %d", tracker.Success())
	}

	if tracker.Failed() != 2 {
		t.Errorf("expected 2 failed, got %d", tracker.Failed())
	}
}

func TestProgressTracker_Rates(t *testing.T) {
	tracker := NewProgressTracker(Config{
		UpdateInterval: 100 * time.Millisecond,
	})

	// Add some data
	for i := 0; i < 80; i++ {
		tracker.IncrementSuccess()
	}

	for i := 0; i < 20; i++ {
		tracker.IncrementFailed()
	}

	successRate := tracker.SuccessRate()
	if successRate != 80.0 {
		t.Errorf("expected success rate 80%%, got %.1f%%", successRate)
	}

	failureRate := tracker.FailureRate()
	if failureRate != 20.0 {
		t.Errorf("expected failure rate 20%%, got %.1f%%", failureRate)
	}
}

func TestProgressTracker_PercentComplete(t *testing.T) {
	tracker := NewProgressTracker(Config{
		TotalRecords: 200,
	})

	// Process 50 records
	for i := 0; i < 50; i++ {
		tracker.IncrementSuccess()
	}

	percent := tracker.PercentComplete()
	if percent != 25.0 {
		t.Errorf("expected 25%% complete, got %.1f%%", percent)
	}
}

func TestProgressTracker_Throughput(t *testing.T) {
	tracker := NewProgressTracker(Config{})

	if err := tracker.Start(); err != nil {
		t.Fatalf("failed to start tracker: %v", err)
	}
	defer tracker.Stop()

	// Process some records
	for i := 0; i < 100; i++ {
		tracker.IncrementSuccess()
	}

	// Wait a bit for elapsed time
	time.Sleep(100 * time.Millisecond)

	throughput := tracker.Throughput()

	// Should have some throughput > 0
	if throughput <= 0 {
		t.Error("expected positive throughput")
	}

	t.Logf("Throughput: %.0f records/sec", throughput)
}

func TestProgressTracker_ETA(t *testing.T) {
	tracker := NewProgressTracker(Config{
		TotalRecords: 1000,
	})

	if err := tracker.Start(); err != nil {
		t.Fatalf("failed to start tracker: %v", err)
	}
	defer tracker.Stop()

	// Process 100 records
	for i := 0; i < 100; i++ {
		tracker.IncrementSuccess()
		time.Sleep(1 * time.Millisecond) // Simulate work
	}

	eta := tracker.ETA()

	// ETA should be positive
	if eta <= 0 {
		t.Error("expected positive ETA")
	}

	t.Logf("ETA: %s", eta)
}

func TestProgressTracker_RecordProcessed(t *testing.T) {
	tracker := NewProgressTracker(Config{})

	// Create test records with different statuses
	record1 := models.NewRecord(1, "test.csv", []string{"data"}, nil)
	result1 := models.NewSuccessResult(record1, nil, 0)

	record2 := models.NewRecord(2, "test.csv", []string{"data"}, nil)
	result2 := models.NewFailedResult(record2, fmt.Errorf("error"), 0)

	record3 := models.NewRecord(3, "test.csv", []string{"data"}, nil)
	result3 := &models.Result{
		Record: record3,
		Status: models.StatusSkipped,
	}

	tracker.RecordProcessed(result1)
	tracker.RecordProcessed(result2)
	tracker.RecordProcessed(result3)

	if tracker.Processed() != 3 {
		t.Errorf("expected 3 processed, got %d", tracker.Processed())
	}

	if tracker.Success() != 1 {
		t.Errorf("expected 1 success, got %d", tracker.Success())
	}

	if tracker.Failed() != 1 {
		t.Errorf("expected 1 failed, got %d", tracker.Failed())
	}

	if tracker.Skipped() != 1 {
		t.Errorf("expected 1 skipped, got %d", tracker.Skipped())
	}
}

func TestProgressTracker_ProgressOutput(t *testing.T) {
	buf := &bytes.Buffer{}

	tracker := NewProgressTracker(Config{
		Writer:         buf,
		UpdateInterval: 50 * time.Millisecond,
		TotalRecords:   100,
	})

	if err := tracker.Start(); err != nil {
		t.Fatalf("failed to start tracker: %v", err)
	}

	// Process some records
	for i := 0; i < 50; i++ {
		tracker.IncrementSuccess()
	}

	// Wait for update
	time.Sleep(100 * time.Millisecond)

	tracker.Stop()

	output := buf.String()

	// Check that output contains expected information
	if !strings.Contains(output, "Progress:") {
		t.Error("expected output to contain 'Progress:'")
	}

	if !strings.Contains(output, "50/100") {
		t.Error("expected output to contain '50/100'")
	}

	t.Logf("Progress output:\n%s", output)
}

func TestProgressTracker_VerboseMode(t *testing.T) {
	buf := &bytes.Buffer{}

	tracker := NewProgressTracker(Config{
		Writer:         buf,
		UpdateInterval: 50 * time.Millisecond,
		Verbose:        true,
	})

	if err := tracker.Start(); err != nil {
		t.Fatalf("failed to start tracker: %v", err)
	}

	for i := 0; i < 10; i++ {
		tracker.IncrementSuccess()
	}

	time.Sleep(100 * time.Millisecond)

	tracker.Stop()

	output := buf.String()

	// Verbose mode should include more details
	if !strings.Contains(output, "Progress Update") {
		t.Error("expected verbose output to contain 'Progress Update'")
	}

	if !strings.Contains(output, "Elapsed:") {
		t.Error("expected verbose output to contain 'Elapsed:'")
	}

	t.Logf("Verbose output:\n%s", output)
}

func TestProgressTracker_ConcurrentUpdates(t *testing.T) {
	tracker := NewProgressTracker(Config{
		UpdateInterval: 50 * time.Millisecond,
	})

	if err := tracker.Start(); err != nil {
		t.Fatalf("failed to start tracker: %v", err)
	}
	defer tracker.Stop()

	// Simulate concurrent updates from multiple goroutines
	const goroutines = 10
	const recordsPerGoroutine = 100

	done := make(chan bool)

	for i := 0; i < goroutines; i++ {
		go func() {
			for j := 0; j < recordsPerGoroutine; j++ {
				if j%2 == 0 {
					tracker.IncrementSuccess()
				} else {
					tracker.IncrementFailed()
				}
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < goroutines; i++ {
		<-done
	}

	expectedTotal := goroutines * recordsPerGoroutine

	if tracker.Processed() != uint64(expectedTotal) {
		t.Errorf("expected %d processed, got %d", expectedTotal, tracker.Processed())
	}

	expectedSuccess := expectedTotal / 2
	if tracker.Success() != uint64(expectedSuccess) {
		t.Errorf("expected %d success, got %d", expectedSuccess, tracker.Success())
	}
}

func TestProgressTracker_Stats(t *testing.T) {
	tracker := NewProgressTracker(Config{
		TotalRecords: 100,
	})

	tracker.Start()
	defer tracker.Stop()

	for i := 0; i < 75; i++ {
		tracker.IncrementSuccess()
	}

	for i := 0; i < 25; i++ {
		tracker.IncrementFailed()
	}

	time.Sleep(100 * time.Millisecond) // Let some time pass

	stats := tracker.Stats()

	if stats.Processed != 100 {
		t.Errorf("expected 100 processed, got %d", stats.Processed)
	}

	if stats.Success != 75 {
		t.Errorf("expected 75 success, got %d", stats.Success)
	}

	if stats.Failed != 25 {
		t.Errorf("expected 25 failed, got %d", stats.Failed)
	}

	if stats.SuccessRate != 75.0 {
		t.Errorf("expected 75%% success rate, got %.1f%%", stats.SuccessRate)
	}

	if stats.PercentComplete != 100.0 {
		t.Errorf("expected 100%% complete, got %.1f%%", stats.PercentComplete)
	}

	t.Logf("Stats: %s", stats.String())
}

func BenchmarkProgressTracker_IncrementSuccess(b *testing.B) {
	tracker := NewProgressTracker(Config{})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		tracker.IncrementSuccess()
	}
}

func BenchmarkProgressTracker_Concurrent(b *testing.B) {
	tracker := NewProgressTracker(Config{})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			tracker.IncrementSuccess()
		}
	})
}
