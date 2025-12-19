package errors

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/zuhrulumam/csv_processor/internal/models"
)

func TestCollector_Add(t *testing.T) {
	collector := NewCollector(CollectorConfig{})

	record := models.NewRecord(1, "test.csv", []string{"data"}, nil)
	err := errors.New("test error")

	if err := collector.Add(err, record); err != nil {
		t.Errorf("Add() returned error: %v", err)
	}

	if collector.Count() != 1 {
		t.Errorf("expected 1 error, got %d", collector.Count())
	}

	if !collector.HasErrors() {
		t.Error("HasErrors() returned false, want true")
	}
}

func TestCollector_MaxErrors(t *testing.T) {
	maxErrors := 5
	collector := NewCollector(CollectorConfig{
		MaxErrors: maxErrors,
	})

	record := models.NewRecord(1, "test.csv", []string{"data"}, nil)

	// Add errors up to limit
	for i := 0; i < maxErrors; i++ {
		err := fmt.Errorf("error %d", i)
		if err := collector.Add(err, record); err != nil {
			t.Errorf("Add() returned error at %d: %v", i, err)
		}
	}

	// Next add should fail
	err := collector.Add(errors.New("overflow"), record)
	if err == nil {
		t.Error("expected error when exceeding max errors")
	}
}

func TestCollector_ErrorThreshold(t *testing.T) {
	collector := NewCollector(CollectorConfig{
		ErrorThreshold:   0.1, // 10%
		AbortOnThreshold: true,
	})

	record := models.NewRecord(1, "test.csv", []string{"data"}, nil)

	// Process 10 records, fail on 2nd (20% error rate)
	collector.IncrementProcessed()
	collector.IncrementProcessed()
	collector.IncrementProcessed()
	collector.IncrementProcessed()
	collector.IncrementProcessed()
	collector.IncrementProcessed()
	collector.IncrementProcessed()
	collector.IncrementProcessed()
	collector.IncrementProcessed()
	collector.IncrementProcessed()

	err := collector.Add(errors.New("error 1"), record)
	if err != nil {
		t.Errorf("Add() returned error: %v", err)
	}

	// This should exceed threshold (2 error in 10 processed = 20% > 10%)
	err = collector.Add(errors.New("error 2"), record)
	if err == nil {
		t.Error("expected error when threshold exceeded")
	}

	// Check context was canceled
	select {
	case <-collector.Context().Done():
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Error("context was not canceled")
	}
}

func TestCollector_ErrorsByCategory(t *testing.T) {
	collector := NewCollector(CollectorConfig{})

	record := models.NewRecord(1, "test.csv", []string{"data"}, nil)

	// Add errors of different categories
	_ = collector.AddWithCategory(ErrInvalidRecord, record, CategoryValidation)
	_ = collector.AddWithCategory(ErrFileNotFound, record, CategoryIO)
	_ = collector.AddWithCategory(ErrProcessingFailed, record, CategoryProcessing)

	byCategory := collector.ErrorsByCategory()

	if len(byCategory[CategoryValidation]) != 1 {
		t.Errorf("expected 1 validation error, got %d", len(byCategory[CategoryValidation]))
	}

	if len(byCategory[CategoryIO]) != 1 {
		t.Errorf("expected 1 IO error, got %d", len(byCategory[CategoryIO]))
	}

	if len(byCategory[CategoryProcessing]) != 1 {
		t.Errorf("expected 1 processing error, got %d", len(byCategory[CategoryProcessing]))
	}
}

func TestCollector_ErrorsBySeverity(t *testing.T) {
	collector := NewCollector(CollectorConfig{})

	record := models.NewRecord(1, "test.csv", []string{"data"}, nil)

	// Add errors with different severities
	_ = collector.Add(ErrInvalidRecord, record)     // Low
	_ = collector.Add(ErrFileNotFound, record)      // Medium
	_ = collector.Add(ErrContextCanceled, record)   // High
	_ = collector.Add(ErrMaxErrorsExceeded, record) // Critical

	bySeverity := collector.ErrorsBySeverity()

	if len(bySeverity[SeverityLow]) != 1 {
		t.Errorf("expected 1 low severity error, got %d", len(bySeverity[SeverityLow]))
	}

	if len(bySeverity[SeverityMedium]) != 1 {
		t.Errorf("expected 1 medium severity error, got %d", len(bySeverity[SeverityMedium]))
	}

	if len(bySeverity[SeverityHigh]) != 1 {
		t.Errorf("expected 1 high severity error, got %d", len(bySeverity[SeverityHigh]))
	}

	if len(bySeverity[SeverityCritical]) != 1 {
		t.Errorf("expected 1 critical severity error, got %d", len(bySeverity[SeverityCritical]))
	}
}

func TestCollector_Summary(t *testing.T) {
	collector := NewCollector(CollectorConfig{})

	record := models.NewRecord(1, "test.csv", []string{"data"}, nil)

	// Add various errors
	for i := 0; i < 10; i++ {
		collector.IncrementProcessed()
	}

	_ = collector.Add(ErrInvalidRecord, record)
	_ = collector.Add(ErrFileNotFound, record)
	_ = collector.Add(ErrProcessingFailed, record)

	summary := collector.Summary()

	if summary.TotalErrors != 3 {
		t.Errorf("expected 3 total errors, got %d", summary.TotalErrors)
	}

	if summary.TotalProcessed != 10 {
		t.Errorf("expected 10 processed, got %d", summary.TotalProcessed)
	}

	expectedRate := 3.0 / 10.0
	if summary.ErrorRate != expectedRate {
		t.Errorf("expected error rate %.2f, got %.2f", expectedRate, summary.ErrorRate)
	}

	t.Logf("Summary: %s", summary.String())
}

func TestCollector_Clear(t *testing.T) {
	collector := NewCollector(CollectorConfig{})

	record := models.NewRecord(1, "test.csv", []string{"data"}, nil)

	// Add some errors
	_ = collector.Add(errors.New("error 1"), record)
	_ = collector.Add(errors.New("error 2"), record)

	if collector.Count() != 2 {
		t.Errorf("expected 2 errors before clear, got %d", collector.Count())
	}

	collector.Clear()

	if collector.Count() != 0 {
		t.Errorf("expected 0 errors after clear, got %d", collector.Count())
	}

	if collector.HasErrors() {
		t.Error("HasErrors() returned true after clear")
	}
}

func TestCollector_ConcurrentAdd(t *testing.T) {
	collector := NewCollector(CollectorConfig{})

	const goroutines = 10
	const errorsPerGoroutine = 100

	done := make(chan bool)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			record := models.NewRecord(id, "test.csv", []string{"data"}, nil)

			for j := 0; j < errorsPerGoroutine; j++ {
				err := fmt.Errorf("error %d-%d", id, j)
				_ = collector.Add(err, record)
				collector.IncrementProcessed()
			}

			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < goroutines; i++ {
		<-done
	}

	expectedErrors := goroutines * errorsPerGoroutine

	if collector.Count() != expectedErrors {
		t.Errorf("expected %d errors, got %d", expectedErrors, collector.Count())
	}
}

func TestCategorizeError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected ErrorCategory
	}{
		{"validation error", ErrInvalidRecord, CategoryValidation},
		{"IO error", ErrFileNotFound, CategoryIO},
		{"timeout error", context.DeadlineExceeded, CategoryTimeout},
		{"processing error", ErrProcessingFailed, CategoryProcessing},
		{"unknown error", errors.New("unknown"), CategoryUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := categorizeError(tt.err)
			if got != tt.expected {
				t.Errorf("categorizeError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"validation error not retryable", ErrInvalidRecord, false},
		{"IO error retryable", ErrFileNotFound, true},
		{"timeout retryable", context.DeadlineExceeded, true},
		{"nil not retryable", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isRetryable(tt.err)
			if got != tt.expected {
				t.Errorf("isRetryable() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func BenchmarkCollector_Add(b *testing.B) {
	collector := NewCollector(CollectorConfig{})
	record := models.NewRecord(1, "test.csv", []string{"data"}, nil)
	err := errors.New("benchmark error")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = collector.Add(err, record)
	}
}

func BenchmarkCollector_Concurrent(b *testing.B) {
	collector := NewCollector(CollectorConfig{})
	record := models.NewRecord(1, "test.csv", []string{"data"}, nil)
	err := errors.New("benchmark error")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = collector.Add(err, record)
		}
	})
}
