package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/zuhrulumam/csv_processor/internal/pipeline"
	"github.com/zuhrulumam/csv_processor/internal/processor"
	"github.com/zuhrulumam/csv_processor/test/fixtures"
)

func TestIntegration_EndToEnd(t *testing.T) {
	tmpDir := t.TempDir()

	// Generate test data
	gen := fixtures.NewGenerator(tmpDir)
	file, err := gen.GenerateSimple("test.csv", 1000)
	if err != nil {
		t.Fatalf("failed to generate test file: %v", err)
	}

	// Create output file
	outputFile := filepath.Join(tmpDir, "output.csv")
	outFile, err := os.Create(outputFile)
	if err != nil {
		t.Fatalf("failed to create output file: %v", err)
	}
	defer outFile.Close()

	// Create and run pipeline
	pipe, err := pipeline.NewPipeline(pipeline.Config{
		Files:        []string{file},
		HasHeader:    true,
		Workers:      4,
		Processor:    processor.NewDefaultProcessor(),
		OutputWriter: outFile,
		ShowProgress: false,
	})

	if err != nil {
		t.Fatalf("failed to create pipeline: %v", err)
	}

	startTime := time.Now()

	if err := pipe.Run(); err != nil {
		t.Fatalf("pipeline execution failed: %v", err)
	}

	duration := time.Since(startTime)

	// Verify results
	summary := pipe.Summary()

	if summary.TotalRecords() != 1000 {
		t.Errorf("expected 1000 records, got %d", summary.TotalRecords())
	}

	if summary.SuccessCount() != 1000 {
		t.Errorf("expected 1000 successful records, got %d", summary.SuccessCount())
	}

	if summary.FailedCount() != 0 {
		t.Errorf("expected 0 failed records, got %d", summary.FailedCount())
	}

	// Verify output file
	outFile.Close()
	stat, err := os.Stat(outputFile)
	if err != nil {
		t.Fatalf("output file not found: %v", err)
	}

	if stat.Size() == 0 {
		t.Error("output file is empty")
	}

	t.Logf("Processed 1000 records in %v (%.0f rec/s)", duration, summary.Throughput())
}

func TestIntegration_MultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Generate multiple test files
	gen := fixtures.NewGenerator(tmpDir)
	files, err := gen.GenerateMultiple("data", 5, 200)
	if err != nil {
		t.Fatalf("failed to generate test files: %v", err)
	}

	// Create and run pipeline
	pipe, err := pipeline.NewPipeline(pipeline.Config{
		Files:        files,
		HasHeader:    true,
		Workers:      8,
		Processor:    processor.NewDefaultProcessor(),
		ShowProgress: false,
	})

	if err != nil {
		t.Fatalf("failed to create pipeline: %v", err)
	}

	if err := pipe.Run(); err != nil {
		t.Fatalf("pipeline execution failed: %v", err)
	}

	summary := pipe.Summary()

	expectedRecords := 5 * 200
	if summary.TotalRecords() != expectedRecords {
		t.Errorf("expected %d records, got %d", expectedRecords, summary.TotalRecords())
	}
}

func TestIntegration_ErrorHandling(t *testing.T) {
	tmpDir := t.TempDir()

	// Generate file with errors
	gen := fixtures.NewGenerator(tmpDir)
	file, err := gen.GenerateWithErrors("errors.csv", 100, 0.2) // 20% error rate
	if err != nil {
		t.Fatalf("failed to generate test file: %v", err)
	}

	// Create and run pipeline
	pipe, err := pipeline.NewPipeline(pipeline.Config{
		Files:        []string{file},
		HasHeader:    true,
		Workers:      2,
		Processor:    processor.NewDefaultProcessor(),
		ShowProgress: false,
	})

	if err != nil {
		t.Fatalf("failed to create pipeline: %v", err)
	}

	if err := pipe.Run(); err != nil {
		t.Fatalf("pipeline execution failed: %v", err)
	}

	// Verify errors were collected
	if !pipe.Errors().HasErrors() {
		t.Error("expected errors to be collected")
	}

	errorCount := pipe.Errors().Count()
	t.Logf("Collected %d errors", errorCount)

	if errorCount == 0 {
		t.Error("expected some errors due to malformed records")
	}
}

func TestIntegration_LargeFile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping large file test in short mode")
	}

	tmpDir := t.TempDir()

	// Generate large file
	gen := fixtures.NewGenerator(tmpDir)
	file, err := gen.GenerateLarge("large.csv", 100000)
	if err != nil {
		t.Fatalf("failed to generate large file: %v", err)
	}

	// Create and run pipeline
	pipe, err := pipeline.NewPipeline(pipeline.Config{
		Files:        []string{file},
		HasHeader:    true,
		Workers:      8,
		Processor:    processor.NewDefaultProcessor(),
		BufferSize:   1000,
		ShowProgress: false,
	})

	if err != nil {
		t.Fatalf("failed to create pipeline: %v", err)
	}

	startTime := time.Now()

	if err := pipe.Run(); err != nil {
		t.Fatalf("pipeline execution failed: %v", err)
	}

	duration := time.Since(startTime)
	summary := pipe.Summary()

	if summary.TotalRecords() != 100000 {
		t.Errorf("expected 100000 records, got %d", summary.TotalRecords())
	}

	t.Logf("Processed 100k records in %v (%.0f rec/s)", duration, summary.Throughput())

	// Performance check
	if summary.Throughput() < 1000 {
		t.Logf("Warning: throughput is low (%.0f rec/s)", summary.Throughput())
	}
}

func TestIntegration_GracefulShutdown(t *testing.T) {
	tmpDir := t.TempDir()

	// Generate large file
	gen := fixtures.NewGenerator(tmpDir)
	file, err := gen.GenerateLarge("shutdown.csv", 50000)
	if err != nil {
		t.Fatalf("failed to generate test file: %v", err)
	}

	// Create pipeline
	pipe, err := pipeline.NewPipeline(pipeline.Config{
		Files:        []string{file},
		HasHeader:    true,
		Workers:      4,
		Processor:    processor.NewDefaultProcessor(),
		ShowProgress: false,
	})

	if err != nil {
		t.Fatalf("failed to create pipeline: %v", err)
	}

	// Run in background
	done := make(chan error)
	go func() {
		done <- pipe.Run()
	}()

	// Wait a bit then stop
	time.Sleep(200 * time.Millisecond)
	pipe.Stop()

	// Wait for completion
	select {
	case err := <-done:
		if err != nil {
			t.Logf("Pipeline stopped with error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("pipeline did not stop within timeout")
	}

	summary := pipe.Summary()

	// Should have processed some but not all
	if summary.TotalRecords() == 0 {
		t.Error("expected some records to be processed")
	}

	if summary.TotalRecords() >= 50000 {
		t.Error("pipeline processed all records (should have been interrupted)")
	}

	t.Logf("Processed %d/%d records before shutdown", summary.TotalRecords(), 50000)
}

func TestIntegration_ErrorThreshold(t *testing.T) {
	tmpDir := t.TempDir()

	// Generate file with high error rate
	gen := fixtures.NewGenerator(tmpDir)
	file, err := gen.GenerateWithErrors("high_errors.csv", 100, 0.5) // 50% errors
	if err != nil {
		t.Fatalf("failed to generate test file: %v", err)
	}

	// Create pipeline with low threshold
	pipe, err := pipeline.NewPipeline(pipeline.Config{
		Files:          []string{file},
		HasHeader:      true,
		Workers:        2,
		Processor:      processor.NewDefaultProcessor(),
		ErrorThreshold: 0.1, // 10% threshold
		AbortOnError:   true,
		ShowProgress:   false,
	})

	if err != nil {
		t.Fatalf("failed to create pipeline: %v", err)
	}

	// Run pipeline - should abort
	err = pipe.Run()

	if err == nil {
		t.Error("expected error when threshold exceeded")
	}

	if !pipe.Errors().ThresholdExceeded() {
		t.Error("expected threshold exceeded flag to be set")
	}

	t.Logf("Pipeline aborted with error: %v", err)
}

func TestIntegration_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Generate empty file
	gen := fixtures.NewGenerator(tmpDir)
	file, err := gen.GenerateEmpty("empty.csv")
	if err != nil {
		t.Fatalf("failed to generate empty file: %v", err)
	}

	// Create and run pipeline
	pipe, err := pipeline.NewPipeline(pipeline.Config{
		Files:        []string{file},
		HasHeader:    true,
		Workers:      2,
		Processor:    processor.NewDefaultProcessor(),
		ShowProgress: false,
	})

	if err != nil {
		t.Fatalf("failed to create pipeline: %v", err)
	}

	err = pipe.Run()

	// Should handle empty file gracefully
	if pipe.Errors().HasErrors() {
		t.Logf("Errors collected: %d", pipe.Errors().Count())
	}
}

func TestIntegration_HeaderOnly(t *testing.T) {
	tmpDir := t.TempDir()

	// Generate file with only header
	gen := fixtures.NewGenerator(tmpDir)
	file, err := gen.GenerateHeaderOnly("header_only.csv")
	if err != nil {
		t.Fatalf("failed to generate test file: %v", err)
	}

	// Create and run pipeline
	pipe, err := pipeline.NewPipeline(pipeline.Config{
		Files:        []string{file},
		HasHeader:    true,
		Workers:      2,
		Processor:    processor.NewDefaultProcessor(),
		ShowProgress: false,
	})

	if err != nil {
		t.Fatalf("failed to create pipeline: %v", err)
	}

	if err := pipe.Run(); err != nil {
		t.Fatalf("pipeline execution failed: %v", err)
	}

	summary := pipe.Summary()

	if summary.TotalRecords() != 0 {
		t.Errorf("expected 0 records, got %d", summary.TotalRecords())
	}
}

func BenchmarkIntegration_1000Records(b *testing.B) {
	tmpDir := b.TempDir()

	gen := fixtures.NewGenerator(tmpDir)
	file, err := gen.GenerateSimple("bench.csv", 1000)
	if err != nil {
		b.Fatalf("failed to generate test file: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		pipe, _ := pipeline.NewPipeline(pipeline.Config{
			Files:        []string{file},
			HasHeader:    true,
			Workers:      4,
			Processor:    processor.NewDefaultProcessor(),
			ShowProgress: false,
		})

		pipe.Run()
	}
}

func BenchmarkIntegration_ConcurrentWorkers(b *testing.B) {
	tmpDir := b.TempDir()

	gen := fixtures.NewGenerator(tmpDir)
	file, err := gen.GenerateSimple("bench.csv", 10000)
	if err != nil {
		b.Fatalf("failed to generate test file: %v", err)
	}

	workerCounts := []int{1, 2, 4, 8, 16}

	for _, workers := range workerCounts {
		b.Run(fmt.Sprintf("workers=%d", workers), func(b *testing.B) {
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				pipe, _ := pipeline.NewPipeline(pipeline.Config{
					Files:        []string{file},
					HasHeader:    true,
					Workers:      workers,
					Processor:    processor.NewDefaultProcessor(),
					ShowProgress: false,
				})

				pipe.Run()
			}
		})
	}
}
