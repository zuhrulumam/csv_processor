package models

import (
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
	// TotalRecords is the total number of records processed
	TotalRecords int

	// SuccessCount is the number of successfully processed records
	SuccessCount int

	// FailedCount is the number of failed records
	FailedCount int

	// SkippedCount is the number of skipped records
	SkippedCount int

	// StartTime is when processing started
	StartTime time.Time

	// EndTime is when processing completed
	EndTime time.Time

	// Duration is the total processing time
	Duration time.Duration

	// Throughput is records processed per second
	Throughput float64
}

// NewSummary creates a new Summary instance
func NewSummary() *Summary {
	return &Summary{
		StartTime: time.Now(),
	}
}

// AddResult updates the summary with a new result
func (s *Summary) AddResult(result *Result) {
	s.TotalRecords++

	switch result.Status {
	case StatusSuccess:
		s.SuccessCount++
	case StatusFailed:
		s.FailedCount++
	case StatusSkipped:
		s.SkippedCount++
	}
}

// Finalize completes the summary calculation
func (s *Summary) Finalize() {
	s.EndTime = time.Now()
	s.Duration = s.EndTime.Sub(s.StartTime)

	if s.Duration.Seconds() > 0 {
		s.Throughput = float64(s.TotalRecords) / s.Duration.Seconds()
	}
}

// SuccessRate returns the percentage of successful records
func (s *Summary) SuccessRate() float64 {
	if s.TotalRecords == 0 {
		return 0
	}
	return float64(s.SuccessCount) / float64(s.TotalRecords) * 100
}

// FailureRate returns the percentage of failed records
func (s *Summary) FailureRate() float64 {
	if s.TotalRecords == 0 {
		return 0
	}
	return float64(s.FailedCount) / float64(s.TotalRecords) * 100
}
