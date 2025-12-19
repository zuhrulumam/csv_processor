package tracker

import (
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zuhrulumam/csv_processor/internal/models"
)

// ProgressTracker tracks processing progress with atomic operations
type ProgressTracker struct {
	// Atomic counters (must be 64-bit aligned for 32-bit systems)
	totalRecords   uint64
	processedCount uint64
	successCount   uint64
	failedCount    uint64
	skippedCount   uint64

	// Start time
	startTime time.Time

	// Ticker for periodic updates
	ticker *time.Ticker

	// Output writer
	writer io.Writer

	// Update interval
	interval time.Duration

	// Context for cancellation
	ctx    context.Context
	cancel context.CancelFunc

	// WaitGroup for cleanup
	wg sync.WaitGroup

	// Mutex for non-atomic operations
	mu sync.RWMutex

	// Started flag
	started bool

	// Verbose mode
	verbose bool
}

// Config holds configuration for progress tracker
type Config struct {
	// Writer is where progress updates are written (default: os.Stdout)
	Writer io.Writer

	// UpdateInterval is how often to print updates (default: 1 second)
	UpdateInterval time.Duration

	// Verbose enables detailed progress information
	Verbose bool

	// TotalRecords is the expected total (0 = unknown)
	TotalRecords uint64
}

// NewProgressTracker creates a new progress tracker
func NewProgressTracker(config Config) *ProgressTracker {
	if config.Writer == nil {
		config.Writer = io.Discard // Default to no output
	}

	if config.UpdateInterval <= 0 {
		config.UpdateInterval = 1 * time.Second
	}

	ctx, cancel := context.WithCancel(context.Background())

	tracker := &ProgressTracker{
		totalRecords: config.TotalRecords,
		startTime:    time.Now(),
		writer:       config.Writer,
		interval:     config.UpdateInterval,
		ctx:          ctx,
		cancel:       cancel,
		verbose:      config.Verbose,
	}

	return tracker
}

// Start starts the progress tracker
func (pt *ProgressTracker) Start() error {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if pt.started {
		return fmt.Errorf("progress tracker already started")
	}

	pt.started = true
	pt.startTime = time.Now()
	pt.ticker = time.NewTicker(pt.interval)

	// Start update loop
	pt.wg.Add(1)
	go pt.updateLoop()

	return nil
}

// updateLoop periodically prints progress updates
func (pt *ProgressTracker) updateLoop() {
	defer pt.wg.Done()

	for {
		select {
		case <-pt.ctx.Done():
			return
		case <-pt.ticker.C:
			pt.printProgress()
		}
	}
}

// RecordProcessed increments the processed counter
func (pt *ProgressTracker) RecordProcessed(result *models.Result) {
	atomic.AddUint64(&pt.processedCount, 1)

	if result == nil {
		return
	}

	switch result.Status {
	case models.StatusSuccess:
		atomic.AddUint64(&pt.successCount, 1)
	case models.StatusFailed:
		atomic.AddUint64(&pt.failedCount, 1)
	case models.StatusSkipped:
		atomic.AddUint64(&pt.skippedCount, 1)
	}
}

// IncrementProcessed increments the processed counter
func (pt *ProgressTracker) IncrementProcessed() {
	atomic.AddUint64(&pt.processedCount, 1)
}

// IncrementSuccess increments the success counter
func (pt *ProgressTracker) IncrementSuccess() {
	atomic.AddUint64(&pt.processedCount, 1)
	atomic.AddUint64(&pt.successCount, 1)
}

// IncrementFailed increments the failed counter
func (pt *ProgressTracker) IncrementFailed() {
	atomic.AddUint64(&pt.processedCount, 1)
	atomic.AddUint64(&pt.failedCount, 1)
}

// IncrementSkipped increments the skipped counter
func (pt *ProgressTracker) IncrementSkipped() {
	atomic.AddUint64(&pt.processedCount, 1)
	atomic.AddUint64(&pt.skippedCount, 1)
}

// SetTotal sets the total expected records
func (pt *ProgressTracker) SetTotal(total uint64) {
	atomic.StoreUint64(&pt.totalRecords, total)
}

// Processed returns the number of processed records
func (pt *ProgressTracker) Processed() uint64 {
	return atomic.LoadUint64(&pt.processedCount)
}

// Success returns the number of successful records
func (pt *ProgressTracker) Success() uint64 {
	return atomic.LoadUint64(&pt.successCount)
}

// Failed returns the number of failed records
func (pt *ProgressTracker) Failed() uint64 {
	return atomic.LoadUint64(&pt.failedCount)
}

// Skipped returns the number of skipped records
func (pt *ProgressTracker) Skipped() uint64 {
	return atomic.LoadUint64(&pt.skippedCount)
}

// Total returns the total expected records
func (pt *ProgressTracker) Total() uint64 {
	return atomic.LoadUint64(&pt.totalRecords)
}

// Elapsed returns the time elapsed since start
func (pt *ProgressTracker) Elapsed() time.Duration {
	return time.Since(pt.startTime)
}

// Throughput returns records processed per second
func (pt *ProgressTracker) Throughput() float64 {
	elapsed := pt.Elapsed().Seconds()
	if elapsed == 0 {
		return 0
	}
	return float64(pt.Processed()) / elapsed
}

// SuccessRate returns the success rate as a percentage
func (pt *ProgressTracker) SuccessRate() float64 {
	processed := pt.Processed()
	if processed == 0 {
		return 0
	}
	return float64(pt.Success()) / float64(processed) * 100
}

// FailureRate returns the failure rate as a percentage
func (pt *ProgressTracker) FailureRate() float64 {
	processed := pt.Processed()
	if processed == 0 {
		return 0
	}
	return float64(pt.Failed()) / float64(processed) * 100
}

// PercentComplete returns the completion percentage
func (pt *ProgressTracker) PercentComplete() float64 {
	total := pt.Total()
	if total == 0 {
		return 0
	}
	return float64(pt.Processed()) / float64(total) * 100
}

// ETA returns the estimated time to completion
func (pt *ProgressTracker) ETA() time.Duration {
	total := pt.Total()
	processed := pt.Processed()

	if total == 0 || processed == 0 {
		return 0
	}

	if processed >= total {
		return 0
	}

	elapsed := pt.Elapsed()
	remaining := total - processed
	avgTimePerRecord := elapsed / time.Duration(processed)

	return avgTimePerRecord * time.Duration(remaining)
}

// printProgress prints current progress to the writer
func (pt *ProgressTracker) printProgress() {
	processed := pt.Processed()
	success := pt.Success()
	failed := pt.Failed()
	skipped := pt.Skipped()
	total := pt.Total()
	elapsed := pt.Elapsed()
	throughput := pt.Throughput()

	if pt.verbose {
		pt.printVerboseProgress(processed, success, failed, skipped, total, elapsed, throughput)
	} else {
		pt.printCompactProgress(processed, success, failed, total, elapsed, throughput)
	}
}

// printCompactProgress prints compact progress information
func (pt *ProgressTracker) printCompactProgress(processed, success, failed, total uint64, elapsed time.Duration, throughput float64) {
	if total > 0 {
		percent := pt.PercentComplete()
		eta := pt.ETA()

		fmt.Fprintf(pt.writer,
			"\r[%s] Progress: %d/%d (%.1f%%) | Success: %d | Failed: %d | %.0f rec/s | ETA: %s",
			elapsed.Round(time.Second),
			processed,
			total,
			percent,
			success,
			failed,
			throughput,
			eta.Round(time.Second),
		)
	} else {
		fmt.Fprintf(pt.writer,
			"\r[%s] Processed: %d | Success: %d | Failed: %d | %.0f rec/s",
			elapsed.Round(time.Second),
			processed,
			success,
			failed,
			throughput,
		)
	}
}

// printVerboseProgress prints detailed progress information
func (pt *ProgressTracker) printVerboseProgress(processed, success, failed, skipped, total uint64, elapsed time.Duration, throughput float64) {
	fmt.Fprintf(pt.writer, "\n========================================\n")
	fmt.Fprintf(pt.writer, "Progress Update\n")
	fmt.Fprintf(pt.writer, "========================================\n")
	fmt.Fprintf(pt.writer, "Elapsed:     %s\n", elapsed.Round(time.Second))
	fmt.Fprintf(pt.writer, "Processed:   %d\n", processed)

	if total > 0 {
		fmt.Fprintf(pt.writer, "Total:       %d\n", total)
		fmt.Fprintf(pt.writer, "Complete:    %.1f%%\n", pt.PercentComplete())
		fmt.Fprintf(pt.writer, "ETA:         %s\n", pt.ETA().Round(time.Second))
	}

	fmt.Fprintf(pt.writer, "Success:     %d (%.1f%%)\n", success, pt.SuccessRate())
	fmt.Fprintf(pt.writer, "Failed:      %d (%.1f%%)\n", failed, pt.FailureRate())

	if skipped > 0 {
		fmt.Fprintf(pt.writer, "Skipped:     %d\n", skipped)
	}

	fmt.Fprintf(pt.writer, "Throughput:  %.0f records/sec\n", throughput)
	fmt.Fprintf(pt.writer, "========================================\n")
}

// PrintFinal prints the final summary
func (pt *ProgressTracker) PrintFinal() {
	processed := pt.Processed()
	success := pt.Success()
	failed := pt.Failed()
	skipped := pt.Skipped()
	elapsed := pt.Elapsed()
	throughput := pt.Throughput()

	fmt.Fprintf(pt.writer, "\n")
	fmt.Fprintf(pt.writer, "========================================\n")
	fmt.Fprintf(pt.writer, "Processing Complete\n")
	fmt.Fprintf(pt.writer, "========================================\n")
	fmt.Fprintf(pt.writer, "Total Time:       %s\n", elapsed.Round(time.Millisecond))
	fmt.Fprintf(pt.writer, "Total Processed:  %d\n", processed)
	fmt.Fprintf(pt.writer, "Successful:       %d (%.1f%%)\n", success, pt.SuccessRate())
	fmt.Fprintf(pt.writer, "Failed:           %d (%.1f%%)\n", failed, pt.FailureRate())

	if skipped > 0 {
		fmt.Fprintf(pt.writer, "Skipped:          %d\n", skipped)
	}

	fmt.Fprintf(pt.writer, "Avg Throughput:   %.0f records/sec\n", throughput)
	fmt.Fprintf(pt.writer, "========================================\n")
}

// Stats returns current statistics
func (pt *ProgressTracker) Stats() Stats {
	return Stats{
		Processed:       pt.Processed(),
		Success:         pt.Success(),
		Failed:          pt.Failed(),
		Skipped:         pt.Skipped(),
		Total:           pt.Total(),
		Elapsed:         pt.Elapsed(),
		Throughput:      pt.Throughput(),
		SuccessRate:     pt.SuccessRate(),
		FailureRate:     pt.FailureRate(),
		PercentComplete: pt.PercentComplete(),
		ETA:             pt.ETA(),
	}
}

// Stop stops the progress tracker
func (pt *ProgressTracker) Stop() {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if !pt.started {
		return
	}

	if pt.ticker != nil {
		pt.ticker.Stop()
	}

	pt.cancel()
	pt.wg.Wait()

	// Print final progress
	pt.printProgress()
}

// StopAndPrintFinal stops the tracker and prints final summary
func (pt *ProgressTracker) StopAndPrintFinal() {
	pt.Stop()
	pt.PrintFinal()
}

// Stats holds statistics snapshot
type Stats struct {
	Processed       uint64
	Success         uint64
	Failed          uint64
	Skipped         uint64
	Total           uint64
	Elapsed         time.Duration
	Throughput      float64
	SuccessRate     float64
	FailureRate     float64
	PercentComplete float64
	ETA             time.Duration
}

// String returns a string representation of stats
func (s Stats) String() string {
	return fmt.Sprintf(
		"Processed: %d, Success: %d (%.1f%%), Failed: %d (%.1f%%), Throughput: %.0f rec/s, Elapsed: %s",
		s.Processed,
		s.Success,
		s.SuccessRate,
		s.Failed,
		s.FailureRate,
		s.Throughput,
		s.Elapsed.Round(time.Second),
	)
}
