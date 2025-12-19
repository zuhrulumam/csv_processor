package processor

import (
	"context"

	"github.com/zuhrulumam/csv_processor/internal/errors"
	"github.com/zuhrulumam/csv_processor/internal/models"
)

// Processor defines the contract for processing CSV records
type Processor interface {
	// Process processes a single record and returns the result
	Process(ctx context.Context, record *models.Record) (*models.Result, error)
}

// ProcessorFunc is a function type that implements the Processor interface
type ProcessorFunc func(ctx context.Context, record *models.Record) (*models.Result, error)

// Process calls the function itself
func (f ProcessorFunc) Process(ctx context.Context, record *models.Record) (*models.Result, error) {
	return f(ctx, record)
}

// DefaultProcessor is a simple processor that just validates records
type DefaultProcessor struct{}

// NewDefaultProcessor creates a new DefaultProcessor
func NewDefaultProcessor() *DefaultProcessor {
	return &DefaultProcessor{}
}

// Process implements the Processor interface
func (p *DefaultProcessor) Process(ctx context.Context, record *models.Record) (*models.Result, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Simple validation
	if !record.IsValid() {
		return models.NewFailedResult(record, errors.ErrInvalidRecord, 0), nil
	}

	// For this default processor, just return success
	// In real use, this would transform the data
	return models.NewSuccessResult(record, record.Data, 0), nil
}

// BatchProcessor processes multiple records
type BatchProcessor interface {
	// ProcessBatch processes multiple records at once
	ProcessBatch(ctx context.Context, records []*models.Record) ([]*models.Result, error)
}

// ResultHandler handles processing results
type ResultHandler interface {
	// Handle processes a result (e.g., write to output, log errors)
	Handle(result *models.Result) error
}

// ErrorHandler handles processing errors
type ErrorHandler interface {
	// HandleError handles a processing error
	HandleError(record *models.Record, err error) error
}
