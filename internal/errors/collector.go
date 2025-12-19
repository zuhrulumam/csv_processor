package errors

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/zuhrulumam/csv_processor/internal/models"
)

// Collector collects and aggregates errors during processing
type Collector struct {
	// errors stores all collected errors
	errors []ErrorEntry

	// mu protects the errors slice
	mu sync.RWMutex

	// maxErrors is the maximum number of errors to collect (0 = unlimited)
	maxErrors int

	// errorThreshold is the max error rate before aborting (0.0-1.0)
	errorThreshold float64

	// totalProcessed tracks total records processed
	totalProcessed uint64

	// abortOnThreshold indicates whether to abort when threshold is exceeded
	abortOnThreshold bool

	// ctx for cancellation
	ctx    context.Context
	cancel context.CancelFunc
}

// ErrorEntry represents a single error with context
type ErrorEntry struct {
	Error     error
	Record    *models.Record
	Timestamp time.Time
	Category  ErrorCategory
	Severity  ErrorSeverity
	Retryable bool
}

// ErrorCategory categorizes error types
type ErrorCategory string

const (
	CategoryValidation ErrorCategory = "VALIDATION"
	CategoryProcessing ErrorCategory = "PROCESSING"
	CategoryIO         ErrorCategory = "IO"
	CategoryTimeout    ErrorCategory = "TIMEOUT"
	CategoryUnknown    ErrorCategory = "UNKNOWN"
)

// ErrorSeverity indicates error severity
type ErrorSeverity string

const (
	SeverityLow      ErrorSeverity = "LOW"
	SeverityMedium   ErrorSeverity = "MEDIUM"
	SeverityHigh     ErrorSeverity = "HIGH"
	SeverityCritical ErrorSeverity = "CRITICAL"
)

// CollectorConfig holds configuration for error collector
type CollectorConfig struct {
	// MaxErrors is the maximum number of errors to store (0 = unlimited)
	MaxErrors int

	// ErrorThreshold is the max error rate (0.0-1.0) before aborting
	// Example: 0.1 = abort if >10% of records fail
	ErrorThreshold float64

	// AbortOnThreshold indicates whether to abort when threshold exceeded
	AbortOnThreshold bool
}

// NewCollector creates a new error collector
func NewCollector(config CollectorConfig) *Collector {
	ctx, cancel := context.WithCancel(context.Background())

	return &Collector{
		errors:           make([]ErrorEntry, 0),
		maxErrors:        config.MaxErrors,
		errorThreshold:   config.ErrorThreshold,
		abortOnThreshold: config.AbortOnThreshold,
		ctx:              ctx,
		cancel:           cancel,
	}
}

// Add adds an error to the collector
func (c *Collector) Add(err error, record *models.Record) error {
	if err == nil {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if we've reached max errors
	if c.maxErrors > 0 && len(c.errors) >= c.maxErrors {
		return fmt.Errorf("maximum error limit reached (%d errors)", c.maxErrors)
	}

	// Create error entry
	entry := ErrorEntry{
		Error:     err,
		Record:    record,
		Timestamp: time.Now(),
		Category:  categorizeError(err),
		Severity:  determineSeverity(err),
		Retryable: isRetryable(err),
	}

	c.errors = append(c.errors, entry)

	// Check error threshold
	if c.abortOnThreshold && c.errorThreshold > 0 {
		errorRate := c.calculateErrorRate()
		if errorRate > c.errorThreshold {
			c.cancel() // Signal abort
			return fmt.Errorf("error threshold exceeded: %.1f%% > %.1f%%",
				errorRate*100, c.errorThreshold*100)
		}
	}

	return nil
}

// AddWithCategory adds an error with explicit category
func (c *Collector) AddWithCategory(err error, record *models.Record, category ErrorCategory) error {
	if err == nil {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.maxErrors > 0 && len(c.errors) >= c.maxErrors {
		return fmt.Errorf("maximum error limit reached (%d errors)", c.maxErrors)
	}

	entry := ErrorEntry{
		Error:     err,
		Record:    record,
		Timestamp: time.Now(),
		Category:  category,
		Severity:  determineSeverity(err),
		Retryable: isRetryable(err),
	}

	c.errors = append(c.errors, entry)

	return nil
}

// IncrementProcessed increments the total processed count
func (c *Collector) IncrementProcessed() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.totalProcessed++
}

// calculateErrorRate calculates the current error rate
func (c *Collector) calculateErrorRate() float64 {
	if c.totalProcessed == 0 {
		return 0
	}
	return float64(len(c.errors)) / float64(c.totalProcessed)
}

// Errors returns all collected errors
func (c *Collector) Errors() []ErrorEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Return a copy to prevent external modification
	errorsCopy := make([]ErrorEntry, len(c.errors))
	copy(errorsCopy, c.errors)

	return errorsCopy
}

// ErrorsByCategory returns errors grouped by category
func (c *Collector) ErrorsByCategory() map[ErrorCategory][]ErrorEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()

	grouped := make(map[ErrorCategory][]ErrorEntry)

	for _, entry := range c.errors {
		grouped[entry.Category] = append(grouped[entry.Category], entry)
	}

	return grouped
}

// ErrorsBySeverity returns errors grouped by severity
func (c *Collector) ErrorsBySeverity() map[ErrorSeverity][]ErrorEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()

	grouped := make(map[ErrorSeverity][]ErrorEntry)

	for _, entry := range c.errors {
		grouped[entry.Severity] = append(grouped[entry.Severity], entry)
	}

	return grouped
}

// HasErrors returns true if any errors were collected
func (c *Collector) HasErrors() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.errors) > 0
}

// Count returns the number of errors collected
func (c *Collector) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.errors)
}

// ErrorRate returns the current error rate (0.0-1.0)
func (c *Collector) ErrorRate() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.calculateErrorRate()
}

// ThresholdExceeded returns true if error threshold was exceeded
func (c *Collector) ThresholdExceeded() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.errorThreshold == 0 {
		return false
	}

	return c.calculateErrorRate() > c.errorThreshold
}

// Context returns the collector's context
func (c *Collector) Context() context.Context {
	return c.ctx
}

// Clear clears all collected errors
func (c *Collector) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.errors = make([]ErrorEntry, 0)
	c.totalProcessed = 0
}

// Summary returns an error summary
func (c *Collector) Summary() ErrorSummary {
	c.mu.RLock()
	defer c.mu.RUnlock()

	summary := ErrorSummary{
		TotalErrors:    len(c.errors),
		TotalProcessed: c.totalProcessed,
		ErrorRate:      c.calculateErrorRate(),
		ByCategory:     make(map[ErrorCategory]int),
		BySeverity:     make(map[ErrorSeverity]int),
	}

	for _, entry := range c.errors {
		summary.ByCategory[entry.Category]++
		summary.BySeverity[entry.Severity]++

		if entry.Retryable {
			summary.RetryableErrors++
		}
	}

	return summary
}

// ErrorSummary provides aggregated error statistics
type ErrorSummary struct {
	TotalErrors     int
	TotalProcessed  uint64
	ErrorRate       float64
	RetryableErrors int
	ByCategory      map[ErrorCategory]int
	BySeverity      map[ErrorSeverity]int
}

// String returns a string representation of the summary
func (s ErrorSummary) String() string {
	return fmt.Sprintf(
		"Total Errors: %d/%d (%.1f%%), Retryable: %d",
		s.TotalErrors,
		s.TotalProcessed,
		s.ErrorRate*100,
		s.RetryableErrors,
	)
}

// categorizeError attempts to categorize an error
func categorizeError(err error) ErrorCategory {
	if err == nil {
		return CategoryUnknown
	}

	// Check for known error types
	switch {
	case IsValidationError(err):
		return CategoryValidation
	case IsIOError(err):
		return CategoryIO
	case IsTimeoutError(err):
		return CategoryTimeout
	case IsProcessingError(err):
		return CategoryProcessing
	default:
		return CategoryUnknown
	}
}

// determineSeverity determines error severity
func determineSeverity(err error) ErrorSeverity {
	if err == nil {
		return SeverityLow
	}

	// Check for critical errors
	switch {
	case err == ErrMaxErrorsExceeded:
		return SeverityCritical
	case err == ErrContextCanceled:
		return SeverityHigh
	case IsValidationError(err):
		return SeverityLow
	case IsIOError(err):
		return SeverityMedium
	default:
		return SeverityMedium
	}
}

// isRetryable determines if an error is retryable
func isRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Validation errors are not retryable
	if IsValidationError(err) {
		return false
	}

	// IO errors might be retryable
	if IsIOError(err) {
		return true
	}

	// Timeout errors are retryable
	if IsTimeoutError(err) {
		return true
	}

	return false
}

// IsValidationError checks if error is a validation error
func IsValidationError(err error) bool {
	if err == nil {
		return false
	}

	_, ok := err.(*ValidationError)
	return ok || err == ErrInvalidRecord || err == ErrHeaderMismatch
}

// IsIOError checks if error is an I/O error
func IsIOError(err error) bool {
	if err == nil {
		return false
	}

	return err == ErrFileNotFound || err == ErrEmptyFile
}

// IsTimeoutError checks if error is a timeout error
func IsTimeoutError(err error) bool {
	if err == nil {
		return false
	}

	return err == context.DeadlineExceeded
}

// IsProcessingError checks if error is a processing error
func IsProcessingError(err error) bool {
	if err == nil {
		return false
	}

	_, ok := err.(*ProcessingError)
	return ok || err == ErrProcessingFailed
}
