package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/zuhrulumam/csv_processor/internal/processor"
)

func TestPipeline_BasicExecution(t *testing.T) {
	// Create temporary test file
	tmpDir := t.TempDir()

	file := filepath.Join(tmpDir, "test.csv")
	content := "name,age,city\nAlice,30,NYC\nBob,25,LA\nCharlie,35,SF\n"

	if err := os.WriteFile(file, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create pipeline
	pipe, err := NewPipeline(Config{
		Files:        []string{file},
		HasHeader:    true,
		Workers:      2,
		Processor:    processor.NewDefaultProcessor(),
		ShowProgress: false,
	})

	if err != nil {
		t.Fatalf("failed to create pipeline: %v", err)
	}

	// Run pipeline
	if err := pipe.Run(); err != nil {
		t.Fatalf("pipeline execution failed: %v", err)
	}

	// Verify summary
	summary := pipe.Summary()

	if summary.TotalRecords != 3 {
		t.Errorf("expected 3 records, got %d", summary.TotalRecords)
	}

	if summary.SuccessCount != 3 {
		t.Errorf("expected 3 successful, got %d", summary.SuccessCount)
	}
}

func TestPipeline_MultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple test files
	files := make([]string, 3)
	for i := 0; i < 3; i++ {
		file := filepath.Join(tmpDir, fmt.Sprintf("test%d.csv", i))
		content := "id,value\n1,100\n2,200\n3,300\n"

		if err := os.WriteFile(file, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		files[i] = file
	}

	// Create and run pipeline
	pipe, err := NewPipeline(Config{
		Files:        files,
		HasHeader:    true,
		Workers:      4,
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

	expectedRecords := 9 // 3 files × 3 records each
	if summary.TotalRecords != expectedRecords {
		t.Errorf("expected %d records, got %d", expectedRecords, summary.TotalRecords)
	}
}

func TestPipeline_ErrorThreshold(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file that will cause errors
	file := filepath.Join(tmpDir, "invalid.csv")
	content := `name,age\nAlice,30\nBob,invalid\nCharlie,35\nDavid,40\nEve,invalid\nFrank,28\nGrace,33\nHannah,29\nIvan,31\nJack,27`

	if err := os.WriteFile(file, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create pipeline with low error threshold
	pipe, err := NewPipeline(Config{
		Files:          []string{file},
		HasHeader:      true,
		Workers:        2,
		Processor:      processor.NewDefaultProcessor(),
		ErrorThreshold: 0.1, // 10%
		AbortOnError:   true,
		ShowProgress:   false,
	})

	if err != nil {
		t.Fatalf("failed to create pipeline: %v", err)
	}

	// Run pipeline - should abort if threshold exceeded
	err = pipe.Run()

	// Check if errors were collected
	if !pipe.Errors().HasErrors() {
		t.Error("expected errors to be collected")
	}
}

func TestPipeline_GracefulShutdown(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a large file
	file := filepath.Join(tmpDir, "large.csv")
	content := "id,value\n"
	for i := 0; i < 10000; i++ {
		content += fmt.Sprintf("%d,%d\n", i, i*10)
	}

	if err := os.WriteFile(file, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create pipeline
	pipe, err := NewPipeline(Config{
		Files:        []string{file},
		HasHeader:    true,
		Workers:      2,
		Processor:    processor.NewDefaultProcessor(),
		ShowProgress: false,
	})

	if err != nil {
		t.Fatalf("failed to create pipeline: %v", err)
	}

	// Run pipeline in background
	done := make(chan error)
	go func() {
		done <- pipe.Run()
	}()

	// Wait until some work has started
	deadline := time.After(2 * time.Second)
	for {
		if pipe.Summary().TotalRecords > 0 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("pipeline never started processing")
		default:
			time.Sleep(1 * time.Millisecond)
		}
	}
	pipe.Stop()

	// Wait for completion
	select {
	case <-done:
		// Pipeline stopped
	case <-time.After(5 * time.Second):
		t.Error("pipeline did not stop within timeout")
	}

	summary := pipe.Summary()

	// Should have processed some records but not all
	if summary.TotalRecords == 0 {
		t.Error("expected some records to be processed")
	}

	if summary.TotalRecords >= 10000 {
		t.Error("pipeline did not stop gracefully")
	}

	t.Logf("Processed %d records before shutdown", summary.TotalRecords)
}

func TestPipeline_OutputFile(t *testing.T) {
	tmpDir := t.TempDir()

	inputFile := filepath.Join(tmpDir, "input.csv")
	outputFile := filepath.Join(tmpDir, "output.csv")

	content := "name,value\ntest1,100\ntest2,200\n"
	if err := os.WriteFile(inputFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create input file: %v", err)
	}

	// Open output file
	outFile, err := os.Create(outputFile)
	if err != nil {
		t.Fatalf("failed to create output file: %v", err)
	}
	defer outFile.Close()

	// Create and run pipeline
	pipe, err := NewPipeline(Config{
		Files:        []string{inputFile},
		HasHeader:    true,
		Workers:      2,
		Processor:    processor.NewDefaultProcessor(),
		OutputWriter: outFile,
		ShowProgress: false,
	})

	if err != nil {
		t.Fatalf("failed to create pipeline: %v", err)
	}

	if err := pipe.Run(); err != nil {
		t.Fatalf("pipeline execution failed: %v", err)
	}

	// Verify output file was written
	outFile.Close()

	stat, err := os.Stat(outputFile)
	if err != nil {
		t.Fatalf("output file not found: %v", err)
	}

	if stat.Size() == 0 {
		t.Error("output file is empty")
	}
}

func TestValidateConfig(t *testing.T) {
	tmpDir := t.TempDir()

	validFile := filepath.Join(tmpDir, "test.csv")
	if err := os.WriteFile(validFile, []byte("a,b,c\n1,2,3\n"), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	tests := []struct {
		name        string
		config      Config
		expectError bool
	}{
		{
			name: "valid config",
			config: Config{
				Files:   []string{validFile},
				Workers: 2,
			},
			expectError: false,
		},
		{
			name: "no files",
			config: Config{
				Workers: 2,
			},
			expectError: true,
		},
		{
			name: "negative workers",
			config: Config{
				Files:   []string{validFile},
				Workers: -1,
			},
			expectError: true,
		},
		{
			name: "invalid error threshold",
			config: Config{
				Files:          []string{validFile},
				Workers:        2,
				ErrorThreshold: 1.5,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)
			hasError := err != nil

			if hasError != tt.expectError {
				t.Errorf("validateConfig() error = %v, expectError = %v", err, tt.expectError)
			}
		})
	}
}

func BenchmarkPipeline(b *testing.B) {
	tmpDir := b.TempDir()

	// Create test file
	file := filepath.Join(tmpDir, "bench.csv")
	content := "id,name,value\n"
	for i := 0; i < 1000; i++ {
		content += fmt.Sprintf("%d,name%d,%d\n", i, i, i*10)
	}

	if err := os.WriteFile(file, []byte(content), 0644); err != nil {
		b.Fatalf("failed to create test file: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		pipe, _ := NewPipeline(Config{
			Files:        []string{file},
			HasHeader:    true,
			Workers:      4,
			Processor:    processor.NewDefaultProcessor(),
			ShowProgress: false,
		})

		pipe.Run()
	}
}
