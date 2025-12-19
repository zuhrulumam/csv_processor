package errors

import (
	"errors"
	"fmt"
)

// Sentinel errors for common error conditions
var (
	// ErrInvalidCSV indicates the CSV file format is invalid
	ErrInvalidCSV = errors.New("invalid CSV format")

	// ErrInvalidRecord indicates a record is malformed
	ErrInvalidRecord = errors.New("invalid record")

	// ErrProcessingFailed indicates processing failed
	ErrProcessingFailed = errors.New("processing failed")

	// ErrFileNotFound indicates the file doesn't exist
	ErrFileNotFound = errors.New("file not found")

	// ErrEmptyFile indicates the file is empty
	ErrEmptyFile = errors.New("empty file")

	// ErrHeaderMismatch indicates header columns don't match data
	ErrHeaderMismatch = errors.New("header column count mismatch")

	// ErrContextCanceled indicates the context was canceled
	ErrContextCanceled = errors.New("context canceled")

	// ErrWorkerPoolClosed indicates the worker pool is closed
	ErrWorkerPoolClosed = errors.New("worker pool is closed")

	// ErrMaxErrorsExceeded indicates too many errors occurred
	ErrMaxErrorsExceeded = errors.New("maximum error threshold exceeded")
)

// ProcessingError wraps errors with additional context
type ProcessingError struct {
	// Op is the operation that failed
	Op string

	// FileName is the file being processed
	FileName string

	// LineNumber is the line where the error occurred
	LineNumber int

	// Err is the underlying error
	Err error
}

// Error implements the error interface
func (e *ProcessingError) Error() string {
	if e.LineNumber > 0 {
		return fmt.Sprintf("%s: %s:%d: %v", e.Op, e.FileName, e.LineNumber, e.Err)
	}
	if e.FileName != "" {
		return fmt.Sprintf("%s: %s: %v", e.Op, e.FileName, e.Err)
	}
	return fmt.Sprintf("%s: %v", e.Op, e.Err)
}

// Unwrap returns the underlying error
func (e *ProcessingError) Unwrap() error {
	return e.Err
}

// NewProcessingError creates a new ProcessingError
func NewProcessingError(op, fileName string, lineNumber int, err error) *ProcessingError {
	return &ProcessingError{
		Op:         op,
		FileName:   fileName,
		LineNumber: lineNumber,
		Err:        err,
	}
}

// ValidationError represents a data validation error
type ValidationError struct {
	Field   string
	Value   string
	Message string
}

// Error implements the error interface
func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error: field=%s, value=%s, message=%s", e.Field, e.Value, e.Message)
}

// NewValidationError creates a new ValidationError
func NewValidationError(field, value, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Value:   value,
		Message: message,
	}
}

// ErrorCollector collects multiple errors during processing
type ErrorCollector struct {
	errors []error
	limit  int
}

// NewErrorCollector creates a new ErrorCollector
func NewErrorCollector(limit int) *ErrorCollector {
	return &ErrorCollector{
		errors: make([]error, 0),
		limit:  limit,
	}
}

// Add adds an error to the collector
func (ec *ErrorCollector) Add(err error) error {
	if err == nil {
		return nil
	}

	ec.errors = append(ec.errors, err)

	// Check if we've exceeded the error limit
	if ec.limit > 0 && len(ec.errors) >= ec.limit {
		return ErrMaxErrorsExceeded
	}

	return nil
}

// Errors returns all collected errors
func (ec *ErrorCollector) Errors() []error {
	return ec.errors
}

// HasErrors returns true if any errors were collected
func (ec *ErrorCollector) HasErrors() bool {
	return len(ec.errors) > 0
}

// Count returns the number of errors collected
func (ec *ErrorCollector) Count() int {
	return len(ec.errors)
}

// Clear clears all collected errors
func (ec *ErrorCollector) Clear() {
	ec.errors = make([]error, 0)
}
