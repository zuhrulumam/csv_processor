package worker

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/zuhrulumam/csv_processor/internal/models"
	"github.com/zuhrulumam/csv_processor/internal/processor"
)

// Worker represents a single worker that processes records
type Worker struct {
	id        int
	processor processor.Processor
	processed uint64
	failed    uint64
}

// NewWorker creates a new worker
func NewWorker(id int, proc processor.Processor) *Worker {
	if proc == nil {
		proc = processor.NewDefaultProcessor()
	}

	return &Worker{
		id:        id,
		processor: proc,
	}
}

// Process processes a single record
func (w *Worker) Process(ctx context.Context, record *models.Record) *models.Result {
	startTime := time.Now()

	result, err := w.processor.Process(ctx, record)
	duration := time.Since(startTime)

	if err != nil || (result != nil && result.IsFailed()) {
		atomic.AddUint64(&w.failed, 1)
		if result == nil {
			result = models.NewFailedResult(record, err, duration)
		}
	} else {
		atomic.AddUint64(&w.processed, 1)
	}

	return result
}

// ID returns the worker ID
func (w *Worker) ID() int {
	return w.id
}

// ProcessedCount returns the number of successfully processed records
func (w *Worker) ProcessedCount() uint64 {
	return atomic.LoadUint64(&w.processed)
}

// FailedCount returns the number of failed records
func (w *Worker) FailedCount() uint64 {
	return atomic.LoadUint64(&w.failed)
}

// Stats returns worker statistics
func (w *Worker) Stats() WorkerStats {
	return WorkerStats{
		ID:        w.id,
		Processed: atomic.LoadUint64(&w.processed),
		Failed:    atomic.LoadUint64(&w.failed),
	}
}

// WorkerStats holds statistics for a worker
type WorkerStats struct {
	ID        int
	Processed uint64
	Failed    uint64
}

// BatchWorker processes multiple records in batches
type BatchWorker struct {
	*Worker
	batchSize int
}

// NewBatchWorker creates a new batch worker
func NewBatchWorker(id int, proc processor.Processor, batchSize int) *BatchWorker {
	if batchSize <= 0 {
		batchSize = 10 // Default batch size
	}

	return &BatchWorker{
		Worker:    NewWorker(id, proc),
		batchSize: batchSize,
	}
}

// ProcessBatch processes multiple records as a batch
func (bw *BatchWorker) ProcessBatch(ctx context.Context, records []*models.Record) []*models.Result {
	results := make([]*models.Result, 0, len(records))

	for _, record := range records {
		// Check context cancellation
		select {
		case <-ctx.Done():
			// Add failed results for remaining records
			for i := len(results); i < len(records); i++ {
				results = append(results, models.NewFailedResult(
					records[i],
					ctx.Err(),
					0,
				))
			}
			return results
		default:
		}

		result := bw.Process(ctx, record)
		results = append(results, result)
	}

	return results
}

// BatchSize returns the configured batch size
func (bw *BatchWorker) BatchSize() int {
	return bw.batchSize
}
