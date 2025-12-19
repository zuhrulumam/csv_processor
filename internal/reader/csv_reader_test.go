package reader

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCSVReader_Read(t *testing.T) {
	// Create temporary test files
	tmpDir := t.TempDir()

	// Test file 1
	file1 := filepath.Join(tmpDir, "test1.csv")
	content1 := "name,age,city\nAlice,30,NYC\nBob,25,LA\n"
	if err := os.WriteFile(file1, []byte(content1), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Test file 2
	file2 := filepath.Join(tmpDir, "test2.csv")
	content2 := "name,age,city\nCharlie,35,SF\nDiana,28,Seattle\n"
	if err := os.WriteFile(file2, []byte(content2), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	tests := []struct {
		name          string
		files         []string
		hasHeader     bool
		expectedCount int
		expectError   bool
	}{
		{
			name:          "read single file with header",
			files:         []string{file1},
			hasHeader:     true,
			expectedCount: 2,
			expectError:   false,
		},
		{
			name:          "read multiple files with header",
			files:         []string{file1, file2},
			hasHeader:     true,
			expectedCount: 4,
			expectError:   false,
		},
		{
			name:          "read non-existent file",
			files:         []string{"nonexistent.csv"},
			hasHeader:     true,
			expectedCount: 0,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			reader := NewCSVReader(Config{
				Files:     tt.files,
				HasHeader: tt.hasHeader,
			})

			recordCh, errCh := reader.Read(ctx)

			var count int
			var hasError bool

			// Collect records and errors
			for recordCh != nil || errCh != nil {
				select {
				case record, ok := <-recordCh:
					if !ok {
						recordCh = nil
						continue
					}
					count++

					// Validate record structure
					if tt.hasHeader && len(record.Headers) == 0 {
						t.Error("expected headers but got none")
					}
					if !record.IsValid() {
						t.Errorf("invalid record at line %d", record.LineNumber)
					}

				case err, ok := <-errCh:
					if !ok {
						errCh = nil
						continue
					}
					hasError = true
					t.Logf("got error: %v", err)
				}
			}

			if count != tt.expectedCount {
				t.Errorf("got %d records, want %d", count, tt.expectedCount)
			}

			if hasError != tt.expectError {
				t.Errorf("hasError = %v, want %v", hasError, tt.expectError)
			}
		})
	}
}

func TestCSVReader_ConcurrentRead(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple test files
	numFiles := 10
	recordsPerFile := 100
	files := make([]string, numFiles)

	for i := 0; i < numFiles; i++ {
		filename := filepath.Join(tmpDir, fmt.Sprintf("test%d.csv", i))
		files[i] = filename

		// Generate content
		content := "id,value\n"
		for j := 0; j < recordsPerFile; j++ {
			content += fmt.Sprintf("%d,%d\n", j, j*10)
		}

		if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
	}

	ctx := context.Background()

	reader := NewCSVReader(Config{
		Files:     files,
		HasHeader: true,
	})

	startTime := time.Now()
	recordCh, errCh := reader.Read(ctx)

	var count int

	for recordCh != nil || errCh != nil {
		select {
		case record, ok := <-recordCh:
			if !ok {
				recordCh = nil
				continue
			}
			count++

			if len(record.Headers) != 2 {
				t.Errorf("expected 2 headers, got %d", len(record.Headers))
			}

		case err, ok := <-errCh:
			if !ok {
				errCh = nil
				continue
			}
			t.Errorf("unexpected error: %v", err)
		}
	}

	duration := time.Since(startTime)

	expectedCount := numFiles * recordsPerFile
	if count != expectedCount {
		t.Errorf("got %d records, want %d", count, expectedCount)
	}

	t.Logf("Read %d records from %d files in %v", count, numFiles, duration)
}

func TestCSVReader_ContextCancellation(t *testing.T) {
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

	ctx, cancel := context.WithCancel(context.Background())

	reader := NewCSVReader(Config{
		Files:     []string{file},
		HasHeader: true,
	})

	recordCh, errCh := reader.Read(ctx)

	// Cancel after reading a few records
	readCount := 0
	for record := range recordCh {
		readCount++
		if readCount >= 10 {
			cancel()
			break
		}
		_ = record
	}

	// Drain error channel
	for range errCh {
	}

	if readCount < 10 {
		t.Errorf("expected to read at least 10 records, got %d", readCount)
	}

	// Should have read fewer than all records due to cancellation
	if readCount >= 10000 {
		t.Error("context cancellation did not stop reading")
	}
}

func TestReadSingle(t *testing.T) {
	tmpDir := t.TempDir()

	file := filepath.Join(tmpDir, "test.csv")
	content := "name,value\ntest1,100\ntest2,200\n"
	if err := os.WriteFile(file, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	ctx := context.Background()
	records, err := ReadSingle(ctx, file, true)

	if err != nil {
		t.Fatalf("ReadSingle() error = %v", err)
	}

	if len(records) != 2 {
		t.Errorf("got %d records, want 2", len(records))
	}

	// Verify headers
	if len(records[0].Headers) != 2 {
		t.Errorf("got %d headers, want 2", len(records[0].Headers))
	}

	if records[0].Headers[0] != "name" || records[0].Headers[1] != "value" {
		t.Errorf("unexpected headers: %v", records[0].Headers)
	}
}

func BenchmarkCSVReader(b *testing.B) {
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
		ctx := context.Background()

		reader := NewCSVReader(Config{
			Files:     []string{file},
			HasHeader: true,
		})

		recordCh, errCh := reader.Read(ctx)

		count := 0
		for recordCh != nil || errCh != nil {
			select {
			case _, ok := <-recordCh:
				if !ok {
					recordCh = nil
					continue
				}
				count++
			case _, ok := <-errCh:
				if !ok {
					errCh = nil
				}
			}
		}
	}
}
