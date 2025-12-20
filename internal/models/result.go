package models

import (
	"sync"
	"sync/atomic"
	"time"
)

// ProcessingStatus represents the outcome of processing a record
type ProcessingStatus string

const (
	StatusSuccess ProcessingStatus = "SUCCESS"
	StatusFailed  ProcessingStatus = "FAILED"
	StatusSkipped ProcessingStatus = "SKIPPED"
)

// Result represents the outcome of processing a single record
type Result struct {
	// Record is the original record that was processed
	Record *Record

	// Status indicates the processing outcome
	Status ProcessingStatus

	// Error contains any error that occurred during processing
	Error error

	// ProcessedData contains the transformed/processed data
	ProcessedData interface{}

	// ProcessedAt is when this record was processed
	ProcessedAt time.Time

	// Duration is how long processing took
	Duration time.Duration
}

// NewResult creates a new Result instance
func NewResult(record *Record, status ProcessingStatus, err error) *Result {
	return &Result{
		Record:      record,
		Status:      status,
		Error:       err,
		ProcessedAt: time.Now(),
	}
}

// NewSuccessResult creates a successful result
func NewSuccessResult(record *Record, processedData interface{}, duration time.Duration) *Result {
	return &Result{
		Record:        record,
		Status:        StatusSuccess,
		ProcessedData: processedData,
		ProcessedAt:   time.Now(),
		Duration:      duration,
	}
}

// NewFailedResult creates a failed result
func NewFailedResult(record *Record, err error, duration time.Duration) *Result {
	return &Result{
		Record:      record,
		Status:      StatusFailed,
		Error:       err,
		ProcessedAt: time.Now(),
		Duration:    duration,
	}
}

// IsSuccess returns true if the result is successful
func (r *Result) IsSuccess() bool {
	return r.Status == StatusSuccess
}

// IsFailed returns true if the result is failed
func (r *Result) IsFailed() bool {
	return r.Status == StatusFailed
}

// Summary represents aggregated processing results
type Summary struct {
	// Atomic counters
	totalRecords uint64
	successCount uint64
	failedCount  uint64
	skippedCount uint64

	// Mutex protects time-related fields
	mu         sync.RWMutex
	startTime  time.Time
	endTime    time.Time
	duration   time.Duration
	throughput float64
}

// NewSummary creates a new Summary instance
func NewSummary() *Summary {
	return &Summary{
		startTime: time.Now(),
	}
}

// AddResult updates the summary with a new result (thread-safe)
func (s *Summary) AddResult(result *Result) {
	atomic.AddUint64(&s.totalRecords, 1)

	switch result.Status {
	case StatusSuccess:
		atomic.AddUint64(&s.successCount, 1)
	case StatusFailed:
		atomic.AddUint64(&s.failedCount, 1)
	case StatusSkipped:
		atomic.AddUint64(&s.skippedCount, 1)
	}
}

// Finalize completes the summary calculation
func (s *Summary) Finalize() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.endTime = time.Now()
	s.duration = s.endTime.Sub(s.startTime)

	if s.duration.Seconds() > 0 {
		total := atomic.LoadUint64(&s.totalRecords)
		s.throughput = float64(total) / s.duration.Seconds()
	}
}

// TotalRecords returns total records processed
func (s *Summary) TotalRecords() int {
	return int(atomic.LoadUint64(&s.totalRecords))
}

// SuccessCount returns successful records
func (s *Summary) SuccessCount() int {
	return int(atomic.LoadUint64(&s.successCount))
}

// FailedCount returns failed records
func (s *Summary) FailedCount() int {
	return int(atomic.LoadUint64(&s.failedCount))
}

// SkippedCount returns skipped records
func (s *Summary) SkippedCount() int {
	return int(atomic.LoadUint64(&s.skippedCount))
}

// StartTime returns the start time
func (s *Summary) StartTime() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.startTime
}

// EndTime returns the end time
func (s *Summary) EndTime() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.endTime
}

// Duration returns the total duration
func (s *Summary) Duration() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.duration
}

// Throughput returns records per second
func (s *Summary) Throughput() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.throughput
}

// SuccessRate returns the percentage of successful records
func (s *Summary) SuccessRate() float64 {
	total := atomic.LoadUint64(&s.totalRecords)
	if total == 0 {
		return 0
	}
	success := atomic.LoadUint64(&s.successCount)
	return float64(success) / float64(total) * 100
}

// FailureRate returns the percentage of failed records
func (s *Summary) FailureRate() float64 {
	total := atomic.LoadUint64(&s.totalRecords)
	if total == 0 {
		return 0
	}
	failed := atomic.LoadUint64(&s.failedCount)
	return float64(failed) / float64(total) * 100
}
