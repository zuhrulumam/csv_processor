package worker

import (
	"context"
	"fmt"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zuhrulumam/csv_processor/internal/models"
)

// mockProcessor is a test processor
type mockProcessor struct {
	processFunc func(ctx context.Context, record *models.Record) (*models.Result, error)
	callCount   uint64
}

func (m *mockProcessor) Process(ctx context.Context, record *models.Record) (*models.Result, error) {
	atomic.AddUint64(&m.callCount, 1)

	if m.processFunc != nil {
		return m.processFunc(ctx, record)
	}

	return models.NewSuccessResult(record, record.Data, 0), nil
}

func (m *mockProcessor) CallCount() uint64 {
	return atomic.LoadUint64(&m.callCount)
}

func TestPool_BasicProcessing(t *testing.T) {
	// Create input channel
	inputCh := make(chan *models.Record, 10)

	// Create test records
	for i := 0; i < 10; i++ {
		record := models.NewRecord(i+1, "test.csv", []string{"data"}, nil)
		inputCh <- record
	}
	close(inputCh)

	// Create mock processor
	mock := &mockProcessor{}

	// Create pool
	pool := NewPool(Config{
		Workers:      2,
		Processor:    mock,
		InputChannel: inputCh,
	})

	// Start pool
	if err := pool.Start(); err != nil {
		t.Fatalf("failed to start pool: %v", err)
	}

	// Collect results
	var successCount int
	for result := range pool.Results() {
		if result.IsSuccess() {
			successCount++
		}
	}

	// Verify all records were processed
	if successCount != 10 {
		t.Errorf("expected 10 successful results, got %d", successCount)
	}

	if mock.CallCount() != 10 {
		t.Errorf("expected processor to be called 10 times, got %d", mock.CallCount())
	}
}

func TestPool_ErrorHandling(t *testing.T) {
	inputCh := make(chan *models.Record, 5)

	// Create records
	for i := 0; i < 5; i++ {
		record := models.NewRecord(i+1, "test.csv", []string{"data"}, nil)
		inputCh <- record
	}
	close(inputCh)

	// Mock processor that fails on odd records
	mock := &mockProcessor{
		processFunc: func(ctx context.Context, record *models.Record) (*models.Result, error) {
			if record.LineNumber%2 == 1 {
				return models.NewFailedResult(record, fmt.Errorf("odd line error"), 0), nil
			}
			return models.NewSuccessResult(record, record.Data, 0), nil
		},
	}

	pool := NewPool(Config{
		Workers:      2,
		Processor:    mock,
		InputChannel: inputCh,
	})

	if err := pool.Start(); err != nil {
		t.Fatalf("failed to start pool: %v", err)
	}

	var successCount, failCount int
	for result := range pool.Results() {
		if result.IsSuccess() {
			successCount++
		} else {
			failCount++
		}
	}

	if successCount != 2 {
		t.Errorf("expected 2 successful results, got %d", successCount)
	}

	if failCount != 3 {
		t.Errorf("expected 3 failed results, got %d", failCount)
	}
}

func TestPool_ContextCancellation(t *testing.T) {
	inputCh := make(chan *models.Record, 100)

	// Create many records
	for i := 0; i < 100; i++ {
		record := models.NewRecord(i+1, "test.csv", []string{"data"}, nil)
		inputCh <- record
	}
	close(inputCh)

	// Slow processor
	mock := &mockProcessor{
		processFunc: func(ctx context.Context, record *models.Record) (*models.Result, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(10 * time.Millisecond):
				return models.NewSuccessResult(record, record.Data, 0), nil
			}
		},
	}

	pool := NewPool(Config{
		Workers:      4,
		Processor:    mock,
		InputChannel: inputCh,
	})

	if err := pool.Start(); err != nil {
		t.Fatalf("failed to start pool: %v", err)
	}

	// Cancel after processing a few records
	time.AfterFunc(50*time.Millisecond, func() {
		pool.Stop()
	})

	var processedCount int
	for range pool.Results() {
		processedCount++
	}

	// Should have processed fewer than all records
	if processedCount >= 100 {
		t.Error("expected context cancellation to stop processing")
	}

	t.Logf("Processed %d/100 records before cancellation", processedCount)
}

func TestPool_ConcurrentProcessing(t *testing.T) {
	const numRecords = 1000

	inputCh := make(chan *models.Record, numRecords)

	for i := 0; i < numRecords; i++ {
		record := models.NewRecord(i+1, "test.csv", []string{"data"}, nil)
		inputCh <- record
	}
	close(inputCh)

	// Processor with artificial delay
	mock := &mockProcessor{
		processFunc: func(ctx context.Context, record *models.Record) (*models.Result, error) {
			time.Sleep(1 * time.Millisecond)
			return models.NewSuccessResult(record, record.Data, 0), nil
		},
	}

	// Test different worker counts
	workerCounts := []int{1, 2, 4, 8}

	for _, workers := range workerCounts {
		t.Run(fmt.Sprintf("workers=%d", workers), func(t *testing.T) {
			// Recreate input channel
			inputCh := make(chan *models.Record, numRecords)
			for i := 0; i < numRecords; i++ {
				record := models.NewRecord(i+1, "test.csv", []string{"data"}, nil)
				inputCh <- record
			}
			close(inputCh)

			pool := NewPool(Config{
				Workers:      workers,
				Processor:    mock,
				InputChannel: inputCh,
			})

			startTime := time.Now()

			if err := pool.Start(); err != nil {
				t.Fatalf("failed to start pool: %v", err)
			}

			count := 0
			for range pool.Results() {
				count++
			}

			duration := time.Since(startTime)

			if count != numRecords {
				t.Errorf("expected %d results, got %d", numRecords, count)
			}

			t.Logf("Processed %d records with %d workers in %v", count, workers, duration)
		})
	}
}

func TestPool_Backpressure(t *testing.T) {
	const numRecords = 100

	inputCh := make(chan *models.Record, numRecords)

	for i := 0; i < numRecords; i++ {
		record := models.NewRecord(i+1, "test.csv", []string{"data"}, nil)
		inputCh <- record
	}
	close(inputCh)

	mock := &mockProcessor{}

	// Small output buffer to test backpressure
	pool := NewPool(Config{
		Workers:          4,
		Processor:        mock,
		InputChannel:     inputCh,
		OutputBufferSize: 5, // Small buffer
	})

	if err := pool.Start(); err != nil {
		t.Fatalf("failed to start pool: %v", err)
	}

	// Slow consumer to create backpressure
	time.Sleep(100 * time.Millisecond)

	count := 0
	for range pool.Results() {
		count++
		time.Sleep(1 * time.Millisecond) // Slow consumption
	}

	if count != numRecords {
		t.Errorf("expected %d results, got %d", numRecords, count)
	}
}

func BenchmarkPool(b *testing.B) {
	mock := &mockProcessor{}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		inputCh := make(chan *models.Record, 1000)

		for j := 0; j < 1000; j++ {
			record := models.NewRecord(j+1, "test.csv", []string{"data"}, nil)
			inputCh <- record
		}
		close(inputCh)

		pool := NewPool(Config{
			Workers:      4,
			Processor:    mock,
			InputChannel: inputCh,
		})

		pool.Start()

		for range pool.Results() {
		}
	}
}

func TestPool_WorkerCount(t *testing.T) {
	tests := []struct {
		name            string
		configWorkers   int
		expectedWorkers int
	}{
		{
			name:            "explicit worker count",
			configWorkers:   4,
			expectedWorkers: 4,
		},
		{
			name:            "default to NumCPU",
			configWorkers:   0,
			expectedWorkers: runtime.NumCPU(),
		},
		{
			name:            "negative defaults to NumCPU",
			configWorkers:   -1,
			expectedWorkers: runtime.NumCPU(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputCh := make(chan *models.Record)
			close(inputCh)

			pool := NewPool(Config{
				Workers:      tt.configWorkers,
				InputChannel: inputCh,
			})

			if pool.WorkerCount() != tt.expectedWorkers {
				t.Errorf("expected %d workers, got %d", tt.expectedWorkers, pool.WorkerCount())
			}
		})
	}
}
