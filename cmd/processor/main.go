package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/zuhrulumam/csv_processor/internal/pipeline"
	"github.com/zuhrulumam/csv_processor/internal/processor"
)

var (
	// Version information
	version   = "1.0.0"
	buildTime = "unknown"
	gitCommit = "unknown"
)

func main() {
	// Parse command line flags
	config := parseFlags()

	// Show version and exit
	if config.showVersion {
		printVersion()
		os.Exit(0)
	}

	// Validate configuration
	if err := config.validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		os.Exit(1)
	}

	// Create pipeline configuration
	pipelineConfig := pipeline.Config{
		Files:          config.inputFiles,
		HasHeader:      config.hasHeader,
		ValidateHeader: config.validateHeader,
		Workers:        config.workers,
		Processor:      processor.NewDefaultProcessor(),
		BufferSize:     config.bufferSize,
		MaxErrors:      config.maxErrors,
		ErrorThreshold: config.errorThreshold,
		AbortOnError:   config.abortOnError,
		ShowProgress:   config.showProgress,
		VerboseOutput:  config.verbose,
	}

	// Open output file if specified
	if config.outputFile != "" {
		file, err := os.Create(config.outputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create output file: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()

		pipelineConfig.OutputWriter = file
	}

	// Create and run pipeline
	pipe, err := pipeline.NewPipeline(pipelineConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create pipeline: %v\n", err)
		os.Exit(1)
	}

	// Print startup info
	if !config.quiet {
		printStartupInfo(config)
	}

	// Run pipeline
	if err := pipe.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Pipeline execution failed: %v\n", err)
		os.Exit(1)
	}

	// Print final summary
	if !config.quiet {
		printFinalSummary(pipe)
	}
}

// Config holds command line configuration
type Config struct {
	// Input
	inputFiles     []string
	hasHeader      bool
	validateHeader bool

	// Processing
	workers    int
	bufferSize int

	// Error handling
	maxErrors      int
	errorThreshold float64
	abortOnError   bool

	// Output
	outputFile   string
	showProgress bool
	verbose      bool
	quiet        bool

	// Meta
	showVersion bool
}

// parseFlags parses command line flags
func parseFlags() *Config {
	config := &Config{}

	// Input options
	flag.BoolVar(&config.hasHeader, "header", true, "CSV files have header row")
	flag.BoolVar(&config.validateHeader, "validate-header", true, "Validate header consistency across files")

	// Processing options
	flag.IntVar(&config.workers, "workers", runtime.NumCPU(), "Number of worker goroutines")
	flag.IntVar(&config.bufferSize, "buffer", 100, "Channel buffer size")

	// Error handling
	flag.IntVar(&config.maxErrors, "max-errors", 0, "Maximum errors to collect (0 = unlimited)")
	flag.Float64Var(&config.errorThreshold, "error-threshold", 0.0, "Error rate threshold (0.0-1.0, 0 = disabled)")
	flag.BoolVar(&config.abortOnError, "abort-on-error", false, "Abort when error threshold is exceeded")

	// Output options
	flag.StringVar(&config.outputFile, "output", "", "Output file path (default: none)")
	flag.BoolVar(&config.showProgress, "progress", true, "Show progress updates")
	flag.BoolVar(&config.verbose, "verbose", false, "Verbose output")
	flag.BoolVar(&config.quiet, "quiet", false, "Suppress all output except errors")

	// Meta
	flag.BoolVar(&config.showVersion, "version", false, "Show version information")

	flag.Usage = printUsage
	flag.Parse()

	// Remaining arguments are input files
	config.inputFiles = flag.Args()

	// Quiet mode overrides other output options
	if config.quiet {
		config.showProgress = false
		config.verbose = false
	}

	return config
}

// validate validates the configuration
func (c *Config) validate() error {
	if len(c.inputFiles) == 0 && !c.showVersion {
		return fmt.Errorf("no input files specified")
	}

	if c.workers < 1 {
		return fmt.Errorf("workers must be at least 1")
	}

	if c.errorThreshold < 0 || c.errorThreshold > 1 {
		return fmt.Errorf("error threshold must be between 0.0 and 1.0")
	}

	return nil
}

// printUsage prints usage information
func printUsage() {
	fmt.Fprintf(os.Stderr, `CSV Processor - Concurrent CSV file processor

Usage:
  processor [options] <file1.csv> [file2.csv ...]

Options:
  -header             CSV files have header row (default: true)
  -validate-header    Validate header consistency (default: true)
  -workers N          Number of worker goroutines (default: NumCPU)
  -buffer N           Channel buffer size (default: 100)
  -max-errors N       Maximum errors to collect (default: 0 = unlimited)
  -error-threshold F  Error rate threshold 0.0-1.0 (default: 0.0 = disabled)
  -abort-on-error     Abort when error threshold exceeded (default: false)
  -output FILE        Output file path (default: none)
  -progress           Show progress updates (default: true)
  -verbose            Verbose output (default: false)
  -quiet              Suppress all output except errors (default: false)
  -version            Show version information

Examples:
  # Process a single file
  processor data.csv

  # Process multiple files with 8 workers
  processor -workers 8 file1.csv file2.csv file3.csv

  # Abort if error rate exceeds 10%%
  processor -error-threshold 0.1 -abort-on-error data.csv

  # Quiet mode with output file
  processor -quiet -output results.csv data.csv

For more information, visit: https://github.com/zuhrulumam/csv_processor
`)
}

// printVersion prints version information
func printVersion() {
	fmt.Printf("CSV Processor version %s\n", version)
	fmt.Printf("Build time: %s\n", buildTime)
	fmt.Printf("Git commit: %s\n", gitCommit)
	fmt.Printf("Go version: %s\n", runtime.Version())
	fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
}

// printStartupInfo prints startup information
func printStartupInfo(config *Config) {
	fmt.Println("========================================")
	fmt.Println("CSV Processor Starting")
	fmt.Println("========================================")
	fmt.Printf("Files:          %d\n", len(config.inputFiles))
	fmt.Printf("Workers:        %d\n", config.workers)
	fmt.Printf("Buffer Size:    %d\n", config.bufferSize)
	fmt.Printf("Has Header:     %v\n", config.hasHeader)

	if config.errorThreshold > 0 {
		fmt.Printf("Error Threshold: %.1f%%\n", config.errorThreshold*100)
	}

	if config.outputFile != "" {
		fmt.Printf("Output File:    %s\n", config.outputFile)
	}

	fmt.Println("========================================")
	fmt.Println()
}

// printFinalSummary prints final processing summary
func printFinalSummary(pipe *pipeline.Pipeline) {
	summary := pipe.Summary()

	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("Processing Summary")
	fmt.Println("========================================")
	fmt.Printf("Total Records:    %d\n", summary.TotalRecords)
	fmt.Printf("Successful:       %d (%.1f%%)\n", summary.SuccessCount, summary.SuccessRate())
	fmt.Printf("Failed:           %d (%.1f%%)\n", summary.FailedCount, summary.FailureRate())
	fmt.Printf("Duration:         %s\n", summary.Duration.Round(time.Millisecond))
	fmt.Printf("Throughput:       %.0f records/sec\n", summary.Throughput)
	fmt.Println("========================================")
}
