package reader

import (
	"testing"

	"github.com/zuhrulumam/csv_processor/internal/models"
)

func TestValidateHeaders(t *testing.T) {
	tests := []struct {
		name        string
		headers     []string
		expectError bool
	}{
		{
			name:        "valid headers",
			headers:     []string{"name", "age", "city"},
			expectError: false,
		},
		{
			name:        "empty headers",
			headers:     []string{},
			expectError: true,
		},
		{
			name:        "duplicate headers",
			headers:     []string{"name", "age", "name"},
			expectError: true,
		},
		{
			name:        "empty header field",
			headers:     []string{"name", "", "city"},
			expectError: true,
		},
		{
			name:        "headers with special characters",
			headers:     []string{"user_id", "first-name", "last name"},
			expectError: false,
		},
		{
			name:        "headers with invalid characters",
			headers:     []string{"name", "age@", "city"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateHeaders(tt.headers)
			hasError := err != nil

			if hasError != tt.expectError {
				t.Errorf("validateHeaders() error = %v, expectError = %v", err, tt.expectError)
			}
		})
	}
}

func TestValidateRecord(t *testing.T) {
	tests := []struct {
		name        string
		record      *models.Record
		expectError bool
	}{
		{
			name: "valid record",
			record: models.NewRecord(
				1,
				"test.csv",
				[]string{"Alice", "30", "NYC"},
				[]string{"name", "age", "city"},
			),
			expectError: false,
		},
		{
			name:        "nil record",
			record:      nil,
			expectError: true,
		},
		{
			name: "empty data",
			record: models.NewRecord(
				1,
				"test.csv",
				[]string{},
				[]string{"name", "age"},
			),
			expectError: true,
		},
		{
			name: "field count mismatch",
			record: models.NewRecord(
				1,
				"test.csv",
				[]string{"Alice", "30"},
				[]string{"name", "age", "city"},
			),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRecord(tt.record)
			hasError := err != nil

			if hasError != tt.expectError {
				t.Errorf("ValidateRecord() error = %v, expectError = %v", err, tt.expectError)
			}
		})
	}
}

func TestIsValidHeaderName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"letters only", "name", true},
		{"with underscore", "user_id", true},
		{"with hyphen", "first-name", true},
		{"with space", "first name", true},
		{"with digits", "col1", true},
		{"with special char", "name@domain", false},
		{"with brackets", "name[0]", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidHeaderName(tt.input)
			if got != tt.want {
				t.Errorf("isValidHeaderName(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
