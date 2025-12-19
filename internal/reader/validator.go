package reader

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/zuhrulumam/csv_processor/internal/errors"
	"github.com/zuhrulumam/csv_processor/internal/models"
)

// validateHeaders validates CSV header fields
func validateHeaders(headers []string) error {
	if len(headers) == 0 {
		return errors.ErrInvalidCSV
	}

	seen := make(map[string]bool)

	for i, header := range headers {
		// Check for empty header
		if strings.TrimSpace(header) == "" {
			return fmt.Errorf("empty header at column %d", i+1)
		}

		// Check for duplicate headers
		normalized := strings.ToLower(strings.TrimSpace(header))
		if seen[normalized] {
			return fmt.Errorf("duplicate header: %s", header)
		}
		seen[normalized] = true

		// Check for valid characters
		if !isValidHeaderName(header) {
			return fmt.Errorf("invalid header name: %s (contains invalid characters)", header)
		}
	}

	return nil
}

// isValidHeaderName checks if a header name contains valid characters
func isValidHeaderName(name string) bool {
	for _, r := range name {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '-' && r != ' ' {
			return false
		}
	}
	return true
}

// ValidateRecord validates a CSV record against its headers
func ValidateRecord(record *models.Record) error {
	if record == nil {
		return errors.ErrInvalidRecord
	}

	// Check if record has data
	if len(record.Data) == 0 {
		return fmt.Errorf("record has no data")
	}

	// Check field count matches header count
	if len(record.Headers) > 0 && len(record.Data) != len(record.Headers) {
		return fmt.Errorf(
			"field count mismatch: expected %d fields, got %d (line %d in %s)",
			len(record.Headers),
			len(record.Data),
			record.LineNumber,
			record.FileName,
		)
	}

	return nil
}

// ValidateFieldCount checks if all records have the same number of fields
func ValidateFieldCount(records []*models.Record) error {
	if len(records) == 0 {
		return nil
	}

	expectedCount := records[0].FieldCount()

	for _, record := range records {
		if record.FieldCount() != expectedCount {
			return fmt.Errorf(
				"inconsistent field count: expected %d, got %d at line %d in %s",
				expectedCount,
				record.FieldCount(),
				record.LineNumber,
				record.FileName,
			)
		}
	}

	return nil
}
