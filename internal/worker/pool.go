package worker

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/zuhrulumam/csv_processor/internal/models"
	"github.com/zuhrulumam/csv_processor/internal/processor"
)

// Pool manages a pool of workers that process records concurrently
type Pool struct {
	// workers is the number of concurrent workers
	workers int

	// processor processes individual records
	processor processor.Processor

	// inputCh receives records to process
	inputCh <-chan *models.Record

	// outputCh sends processing results
	outputCh chan *models.Result

	// errorCh sends errors that occur during processing
	errorCh chan error

	// Mutex protects ctx and cancel
	ctxMu sync.RWMutex

	// wg waits for all workers to complete
	wg sync.WaitGroup

	// ctx is the context for cancellation
	ctx context.Context

	// cancel cancels the context
	cancel context.CancelFunc

	// started indicates if the pool has been started
	started bool

	// mu protects started flag
	mu sync.Mutex
}

// Config holds configuration for the worker pool
type Config struct {
	// Workers is the number of concurrent workers (0 = NumCPU)
	Workers int

	// Processor processes individual records
	Processor processor.Processor

	// InputChannel receives records to process
	InputChannel <-chan *models.Record

	// OutputBufferSize is the size of the output channel buffer
	OutputBufferSize int

	// ErrorBufferSize is the size of the error channel buffer
	ErrorBufferSize int
}

// NewPool creates a new worker pool
func NewPool(config Config) *Pool {
	// Default to NumCPU workers if not specified
	if config.Workers <= 0 {
		config.Workers = runtime.NumCPU()
	}

	// Default buffer sizes
	if config.OutputBufferSize <= 0 {
		config.OutputBufferSize = config.Workers * 2
	}
	if config.ErrorBufferSize <= 0 {
		config.ErrorBufferSize = 10
	}

	// Default processor
	if config.Processor == nil {
		config.Processor = processor.NewDefaultProcessor()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Pool{
		workers:   config.Workers,
		processor: config.Processor,
		inputCh:   config.InputChannel,
		outputCh:  make(chan *models.Result, config.OutputBufferSize),
		errorCh:   make(chan error, config.ErrorBufferSize),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Start starts the worker pool
func (p *Pool) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.started {
		return fmt.Errorf("pool already started")
	}

	p.started = true

	// Start workers
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}

	// Start result collector (closes output channels when done)
	go func() {
		p.wg.Wait()
		close(p.outputCh)
		close(p.errorCh)
	}()

	return nil
}

// worker processes records from the input channel
func (p *Pool) worker(id int) {
	defer p.wg.Done()

	for {
		p.ctxMu.RLock()
		ctx := p.ctx
		p.ctxMu.RUnlock()

		select {
		case <-ctx.Done():
			// Context canceled, stop processing
			return

		case record, ok := <-p.inputCh:
			if !ok {
				// Input channel closed, stop processing
				return
			}

			// Process the record
			result := p.processRecord(record)

			// Send result to output channel (non-blocking)
			select {
			case p.outputCh <- result:
			case <-ctx.Done():
				return
			}
		}
	}
}

// processRecord processes a single record and measures duration
func (p *Pool) processRecord(record *models.Record) *models.Result {
	startTime := time.Now()

	p.ctxMu.RLock()
	ctx := p.ctx
	p.ctxMu.RUnlock()

	// Process with context
	result, err := p.processor.Process(ctx, record)

	duration := time.Since(startTime)

	// Handle processing error
	if err != nil {
		// Send error to error channel (non-blocking)
		select {
		case p.errorCh <- err:
		default:
			// Error channel full, skip
		}

		return models.NewFailedResult(record, err, duration)
	}

	// Set duration if not already set
	if result.Duration == 0 {
		result.Duration = duration
	}

	return result
}

// Results returns the output channel for processing results
func (p *Pool) Results() <-chan *models.Result {
	return p.outputCh
}

// Errors returns the error channel
func (p *Pool) Errors() <-chan error {
	return p.errorCh
}

// Stop gracefully stops the pool and waits for workers to finish
func (p *Pool) Stop() {
	p.ctxMu.Lock()
	cancel := p.cancel
	p.ctxMu.Unlock()

	if cancel != nil {
		cancel()
	}
}

// Wait waits for all workers to complete
func (p *Pool) Wait() {
	p.wg.Wait()
}

// StopAndWait stops the pool and waits for completion
func (p *Pool) StopAndWait() {
	p.Stop()
	p.Wait()
}

// WorkerCount returns the number of workers in the pool
func (p *Pool) WorkerCount() int {
	return p.workers
}
