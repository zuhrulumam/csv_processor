package pipeline

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/zuhrulumam/csv_processor/internal/errors"
	"github.com/zuhrulumam/csv_processor/internal/models"
	"github.com/zuhrulumam/csv_processor/internal/processor"
	"github.com/zuhrulumam/csv_processor/internal/reader"
	"github.com/zuhrulumam/csv_processor/internal/tracker"
	"github.com/zuhrulumam/csv_processor/internal/worker"
)

// Pipeline orchestrates the entire CSV processing workflow
type Pipeline struct {
	// Configuration
	config Config

	// Components
	reader     *reader.CSVReader
	workerPool *worker.Pool
	progress   *tracker.ProgressTracker
	errorCol   *errors.Collector

	// Context and cancellation
	ctx    context.Context
	cancel context.CancelFunc

	// Summary
	summary *models.Summary

	// Mutex for summary updates
	mu sync.Mutex
}

// Config holds pipeline configuration
type Config struct {
	// Input files
	Files []string

	// CSV options
	HasHeader      bool
	ValidateHeader bool

	// Processing
	Workers    int
	Processor  processor.Processor
	BufferSize int

	// Error handling
	MaxErrors      int
	ErrorThreshold float64
	AbortOnError   bool

	// Progress tracking
	ShowProgress  bool
	VerboseOutput bool

	// Output
	OutputWriter *os.File
}

// NewPipeline creates a new processing pipeline
func NewPipeline(config Config) (*Pipeline, error) {
	// Validate configuration
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())

	// Create error collector
	errorCollector := errors.NewCollector(errors.CollectorConfig{
		MaxErrors:        config.MaxErrors,
		ErrorThreshold:   config.ErrorThreshold,
		AbortOnThreshold: config.AbortOnError,
	})

	// Create progress tracker
	progressWriter := os.Stdout
	if !config.ShowProgress {
		progressWriter = nil
	}

	progressTracker := tracker.NewProgressTracker(tracker.Config{
		Writer:         progressWriter,
		UpdateInterval: 1 * time.Second,
		Verbose:        config.VerboseOutput,
	})

	pipeline := &Pipeline{
		config:   config,
		errorCol: errorCollector,
		progress: progressTracker,
		ctx:      ctx,
		cancel:   cancel,
		summary:  models.NewSummary(),
	}

	return pipeline, nil
}

// Run executes the pipeline
func (p *Pipeline) Run() error {
	// Setup signal handling for graceful shutdown
	p.setupSignalHandling()

	// Start progress tracker
	if p.config.ShowProgress {
		if err := p.progress.Start(); err != nil {
			return fmt.Errorf("failed to start progress tracker: %w", err)
		}
	}

	// Create CSV reader
	p.reader = reader.NewCSVReader(reader.Config{
		Files:          p.config.Files,
		HasHeader:      p.config.HasHeader,
		ValidateHeader: p.config.ValidateHeader,
		BufferSize:     p.config.BufferSize,
	})

	// Start reading files
	recordCh, readerErrCh := p.reader.Read(p.ctx)

	// Create worker pool
	p.workerPool = worker.NewPool(worker.Config{
		Workers:          p.config.Workers,
		Processor:        p.config.Processor,
		InputChannel:     recordCh,
		OutputBufferSize: p.config.BufferSize,
		ErrorBufferSize:  10,
	})

	// Start worker pool
	if err := p.workerPool.Start(); err != nil {
		return fmt.Errorf("failed to start worker pool: %w", err)
	}

	// Process results and errors concurrently
	var wg sync.WaitGroup
	wg.Add(3)

	// Handle results
	go func() {
		defer wg.Done()
		p.handleResults()
	}()

	// Handle reader errors
	go func() {
		defer wg.Done()
		p.handleReaderErrors(readerErrCh)
	}()

	// Handle worker errors
	go func() {
		defer wg.Done()
		p.handleWorkerErrors()
	}()

	// Wait for all handlers to complete
	wg.Wait()

	// Finalize
	p.finalize()

	// Check if we should return error
	if p.errorCol.HasErrors() && p.config.AbortOnError {
		if p.errorCol.ThresholdExceeded() {
			return fmt.Errorf("processing aborted: error threshold exceeded")
		}
	}

	return nil
}

// handleResults processes results from workers
func (p *Pipeline) handleResults() {
	for result := range p.workerPool.Results() {
		// Update progress
		if p.config.ShowProgress {
			p.progress.RecordProcessed(result)
		}

		// Update summary
		p.mu.Lock()
		p.summary.AddResult(result)
		p.mu.Unlock()

		// Collect errors
		if result.IsFailed() && result.Error != nil {
			_ = p.errorCol.Add(result.Error, result.Record)
		}

		// Update error collector processed count
		p.errorCol.IncrementProcessed()

		// Check if we should abort
		select {
		case <-p.errorCol.Context().Done():
			// Error threshold exceeded, initiate shutdown
			p.cancel()
			return
		default:
		}

		// Write output if configured
		if p.config.OutputWriter != nil && result.IsSuccess() {
			p.writeOutput(result)
		}
	}
}

// handleReaderErrors handles errors from the CSV reader
func (p *Pipeline) handleReaderErrors(errCh <-chan error) {
	for err := range errCh {
		p.errorCol.Add(err, nil)

		// For critical reader errors, we might want to abort
		if errors.IsIOError(err) {
			fmt.Fprintf(os.Stderr, "Reader error: %v\n", err)
		}
	}
}

// handleWorkerErrors handles errors from the worker pool
func (p *Pipeline) handleWorkerErrors() {
	for err := range p.workerPool.Errors() {
		p.errorCol.Add(err, nil)
	}
}

// writeOutput writes successful result to output file
func (p *Pipeline) writeOutput(result *models.Result) {
	// This is a simple implementation
	// In a real scenario, you'd format the output appropriately
	if result.Record != nil {
		fmt.Fprintf(p.config.OutputWriter, "%s\n", result.Record.Data)
	}
}

// setupSignalHandling sets up signal handlers for graceful shutdown
func (p *Pipeline) setupSignalHandling() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		fmt.Fprintf(os.Stderr, "\nReceived signal: %v\n", sig)
		fmt.Fprintf(os.Stderr, "Initiating graceful shutdown...\n")
		p.cancel()
	}()
}

// finalize completes the pipeline execution
func (p *Pipeline) finalize() {
	// Stop progress tracker
	if p.config.ShowProgress {
		p.progress.StopAndPrintFinal()
	}

	// Finalize summary
	p.summary.Finalize()

	// Print error summary if there are errors
	if p.errorCol.HasErrors() {
		reporter := errors.NewReporter(p.errorCol, os.Stderr)
		reporter.PrintSummary()

		// Print top errors
		if p.errorCol.Count() > 5 {
			reporter.PrintTopErrors(5)
		} else {
			reporter.PrintDetailed(10)
		}
	}
}

// Summary returns the processing summary
func (p *Pipeline) Summary() *models.Summary {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.summary
}

// Errors returns the error collector
func (p *Pipeline) Errors() *errors.Collector {
	return p.errorCol
}

// Stop gracefully stops the pipeline
func (p *Pipeline) Stop() {
	p.cancel()

	if p.workerPool != nil {
		p.workerPool.StopAndWait()
	}
}

// validateConfig validates pipeline configuration
func validateConfig(config Config) error {
	if len(config.Files) == 0 {
		return fmt.Errorf("no input files specified")
	}

	// Check if files exist
	for _, file := range config.Files {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			return fmt.Errorf("file does not exist: %s", file)
		}
	}

	if config.Workers < 0 {
		return fmt.Errorf("workers must be non-negative")
	}

	if config.BufferSize < 0 {
		return fmt.Errorf("buffer size must be non-negative")
	}

	if config.ErrorThreshold < 0 || config.ErrorThreshold > 1 {
		return fmt.Errorf("error threshold must be between 0.0 and 1.0")
	}

	return nil
}
