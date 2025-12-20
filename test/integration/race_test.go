package integration

import (
	"sync"
	"testing"
	"time"

	"github.com/zuhrulumam/csv_processor/internal/pipeline"
	"github.com/zuhrulumam/csv_processor/internal/processor"
	"github.com/zuhrulumam/csv_processor/test/fixtures"
)

// TestRace_ConcurrentPipelines tests running multiple pipelines concurrently
func TestRace_ConcurrentPipelines(t *testing.T) {
	tmpDir := t.TempDir()

	// Generate test files
	gen := fixtures.NewGenerator(tmpDir)
	files, err := gen.GenerateMultiple("race", 5, 100)
	if err != nil {
		t.Fatalf("failed to generate test files: %v", err)
	}

	const numPipelines = 10
	var wg sync.WaitGroup
	errors := make(chan error, numPipelines)

	// Run multiple pipelines concurrently
	for i := 0; i < numPipelines; i++ {
		wg.Add(1)

		go func(id int) {
			defer wg.Done()

			pipe, err := pipeline.NewPipeline(pipeline.Config{
				Files:        files,
				HasHeader:    true,
				Workers:      4,
				Processor:    processor.NewDefaultProcessor(),
				ShowProgress: false,
			})

			if err != nil {
				errors <- err
				return
			}

			if err := pipe.Run(); err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("pipeline error: %v", err)
	}
}

// TestRace_SharedResources tests concurrent access to shared resources
func TestRace_SharedResources(t *testing.T) {
	tmpDir := t.TempDir()

	gen := fixtures.NewGenerator(tmpDir)
	file, err := gen.GenerateSimple("shared.csv", 1000)
	if err != nil {
		t.Fatalf("failed to generate test file: %v", err)
	}

	// Create pipeline
	pipe, err := pipeline.NewPipeline(pipeline.Config{
		Files:        []string{file},
		HasHeader:    true,
		Workers:      8, // High concurrency
		Processor:    processor.NewDefaultProcessor(),
		ShowProgress: false,
	})

	if err != nil {
		t.Fatalf("failed to create pipeline: %v", err)
	}

	// Run pipeline (will be tested with -race flag)
	if err := pipe.Run(); err != nil {
		t.Fatalf("pipeline execution failed: %v", err)
	}

	// Verify no race conditions occurred
	summary := pipe.Summary()
	if summary.TotalRecords() != 1000 {
		t.Errorf("expected 1000 records, got %d", summary.TotalRecords())
	}
}

// TestRace_ProgressTracker tests concurrent updates to progress tracker
func TestRace_ProgressTracker(t *testing.T) {
	tmpDir := t.TempDir()

	gen := fixtures.NewGenerator(tmpDir)
	files, err := gen.GenerateMultiple("progress", 3, 500)
	if err != nil {
		t.Fatalf("failed to generate test files: %v", err)
	}

	pipe, err := pipeline.NewPipeline(pipeline.Config{
		Files:        files,
		HasHeader:    true,
		Workers:      16, // Very high concurrency
		Processor:    processor.NewDefaultProcessor(),
		ShowProgress: true,
	})

	if err != nil {
		t.Fatalf("failed to create pipeline: %v", err)
	}

	if err := pipe.Run(); err != nil {
		t.Fatalf("pipeline execution failed: %v", err)
	}
}

// TestRace_ErrorCollector tests concurrent error collection
func TestRace_ErrorCollector(t *testing.T) {
	tmpDir := t.TempDir()

	gen := fixtures.NewGenerator(tmpDir)
	file, err := gen.GenerateWithErrors("errors.csv", 1000, 0.3)
	if err != nil {
		t.Fatalf("failed to generate test file: %v", err)
	}

	pipe, err := pipeline.NewPipeline(pipeline.Config{
		Files:        []string{file},
		HasHeader:    true,
		Workers:      16,
		Processor:    processor.NewDefaultProcessor(),
		ShowProgress: false,
	})

	if err != nil {
		t.Fatalf("failed to create pipeline: %v", err)
	}

	if err := pipe.Run(); err != nil {
		t.Fatalf("pipeline execution failed: %v", err)
	}

	// Verify error collection worked correctly
	if !pipe.Errors().HasErrors() {
		t.Error("expected errors to be collected")
	}
}

// TestRace_StopDuringExecution tests stopping pipeline during execution
func TestRace_StopDuringExecution(t *testing.T) {
	tmpDir := t.TempDir()

	gen := fixtures.NewGenerator(tmpDir)
	file, err := gen.GenerateLarge("stop.csv", 100000)
	if err != nil {
		t.Fatalf("failed to generate test file: %v", err)
	}

	const iterations = 5

	for i := 0; i < iterations; i++ {
		pipe, err := pipeline.NewPipeline(pipeline.Config{
			Files:        []string{file},
			HasHeader:    true,
			Workers:      8,
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

		// Stop at random time
		stopAfter := time.Duration(10+i*10) * time.Millisecond
		time.Sleep(stopAfter)
		pipe.Stop()

		// Wait for completion
		select {
		case <-done:
			// OK
		case <-time.After(5 * time.Second):
			t.Fatal("pipeline did not stop within timeout")
		}
	}
}
