package models

import (
	"time"
)

// Record represents a single CSV record with its metadata
type Record struct {
	// LineNumber is the original line number in the CSV file (1-indexed)
	LineNumber int

	// FileName is the source CSV file name
	FileName string

	// Data contains the parsed CSV fields
	Data []string

	// Headers contains the column names (if available)
	Headers []string

	// ReadAt is when this record was read
	ReadAt time.Time
}

// NewRecord creates a new Record instance
func NewRecord(lineNumber int, fileName string, data []string, headers []string) *Record {
	return &Record{
		LineNumber: lineNumber,
		FileName:   fileName,
		Data:       data,
		Headers:    headers,
		ReadAt:     time.Now(),
	}
}

// GetField returns the value at the specified column index
// Returns empty string if index is out of bounds
func (r *Record) GetField(index int) string {
	if index < 0 || index >= len(r.Data) {
		return ""
	}
	return r.Data[index]
}

// GetFieldByName returns the value for the specified column name
// Returns empty string if column name not found
func (r *Record) GetFieldByName(columnName string) string {
	for i, header := range r.Headers {
		if header == columnName {
			return r.GetField(i)
		}
	}
	return ""
}

// FieldCount returns the number of fields in this record
func (r *Record) FieldCount() int {
	return len(r.Data)
}

// IsValid performs basic validation on the record
func (r *Record) IsValid() bool {
	// Record must have at least one field
	if len(r.Data) == 0 {
		return false
	}

	// If headers exist, data length should match
	if len(r.Headers) > 0 && len(r.Data) != len(r.Headers) {
		return false
	}

	return true
}
