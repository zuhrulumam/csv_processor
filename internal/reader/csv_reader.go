package reader

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/zuhrulumam/csv_processor/internal/errors"
	"github.com/zuhrulumam/csv_processor/internal/models"
)

// CSVReader reads CSV files concurrently and sends records to a channel
type CSVReader struct {
	// files is the list of CSV files to read
	files []string

	// hasHeader indicates if CSV files have a header row
	hasHeader bool

	// validateHeader checks if all files have the same header
	validateHeader bool

	// bufferSize is the size of the output channel buffer
	bufferSize int
}

// Config holds configuration for CSVReader
type Config struct {
	Files          []string
	HasHeader      bool
	ValidateHeader bool
	BufferSize     int
}

// NewCSVReader creates a new CSVReader instance
func NewCSVReader(config Config) *CSVReader {
	if config.BufferSize == 0 {
		config.BufferSize = 100 // Default buffer size
	}

	return &CSVReader{
		files:          config.Files,
		hasHeader:      config.HasHeader,
		validateHeader: config.ValidateHeader,
		bufferSize:     config.BufferSize,
	}
}

// Read reads all CSV files concurrently and sends records to the output channel
// Returns a channel of records and a channel of errors
func (r *CSVReader) Read(ctx context.Context) (<-chan *models.Record, <-chan error) {
	recordCh := make(chan *models.Record, r.bufferSize)
	errCh := make(chan error, len(r.files))

	var wg sync.WaitGroup
	var headerMu sync.Mutex
	var commonHeader []string

	// Start a goroutine for each file
	for _, file := range r.files {
		wg.Add(1)

		go func(filename string) {
			defer wg.Done()

			// Read the file and send records
			header, err := r.readFile(ctx, filename, recordCh, &headerMu, &commonHeader)
			if err != nil {
				errCh <- errors.NewProcessingError("read", filename, 0, err)
				return
			}

			// Validate header consistency across files
			if r.validateHeader && r.hasHeader {
				headerMu.Lock()
				if commonHeader == nil {
					commonHeader = header
				} else if !headersMatch(commonHeader, header) {
					headerMu.Unlock()
					errCh <- errors.NewProcessingError(
						"validate_header",
						filename,
						0,
						errors.ErrHeaderMismatch,
					)
					return
				}
				headerMu.Unlock()
			}
		}(file)
	}

	// Close channels when all files are read
	go func() {
		wg.Wait()
		close(recordCh)
		close(errCh)
	}()

	return recordCh, errCh
}

// readFile reads a single CSV file and sends records to the channel
func (r *CSVReader) readFile(
	ctx context.Context,
	filename string,
	recordCh chan<- *models.Record,
	headerMu *sync.Mutex,
	commonHeader *[]string,
) ([]string, error) {
	// Open file
	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.ErrFileNotFound
		}
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	// Check if file is empty
	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat file: %w", err)
	}
	if stat.Size() == 0 {
		return nil, errors.ErrEmptyFile
	}

	// Create CSV reader
	csvReader := csv.NewReader(file)
	csvReader.ReuseRecord = true // Optimize memory allocation

	var headers []string
	lineNumber := 0

	// Read header if present
	if r.hasHeader {
		rawHeaders, err := csvReader.Read()
		if err != nil {
			if err == io.EOF {
				return nil, errors.ErrEmptyFile
			}
			return nil, fmt.Errorf("read header: %w", err)
		}
		lineNumber++
		headers = make([]string, len(rawHeaders))
		copy(headers, rawHeaders)

		// Validate header
		if err := validateHeaders(headers); err != nil {
			return nil, errors.NewProcessingError("validate_header", filename, lineNumber, err)
		}
	}

	// Read records
	for {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return headers, ctx.Err()
		default:
		}

		// Read next record
		data, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return headers, errors.NewProcessingError("read_record", filename, lineNumber+1, err)
		}

		lineNumber++

		// Create a copy of data (since we're using ReuseRecord)
		dataCopy := make([]string, len(data))
		copy(dataCopy, data)

		// Create record
		record := models.NewRecord(
			lineNumber,
			filepath.Base(filename),
			dataCopy,
			headers,
		)

		// Send record to channel (with context cancellation check)
		select {
		case <-ctx.Done():
			return headers, ctx.Err()
		case recordCh <- record:
		}
	}

	return headers, nil
}

// ReadSingle reads a single CSV file (convenience method for non-concurrent use)
func ReadSingle(ctx context.Context, filename string, hasHeader bool) ([]*models.Record, error) {
	reader := NewCSVReader(Config{
		Files:     []string{filename},
		HasHeader: hasHeader,
	})

	recordCh, errCh := reader.Read(ctx)

	var records []*models.Record
	var errs []error

	// Collect records and errors
	for recordCh != nil || errCh != nil {
		select {
		case record, ok := <-recordCh:
			if !ok {
				recordCh = nil
				continue
			}
			records = append(records, record)
		case err, ok := <-errCh:
			if !ok {
				errCh = nil
				continue
			}
			errs = append(errs, err)
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	if len(errs) > 0 {
		return records, errs[0] // Return first error
	}

	return records, nil
}

// headersMatch checks if two header slices are identical
func headersMatch(h1, h2 []string) bool {
	if len(h1) != len(h2) {
		return false
	}
	for i := range h1 {
		if h1[i] != h2[i] {
			return false
		}
	}
	return true
}
