package errors

import (
	"errors"
	"testing"
)

func TestProcessingError(t *testing.T) {
	tests := []struct {
		name       string
		op         string
		fileName   string
		lineNumber int
		err        error
		want       string
	}{
		{
			name:       "full error",
			op:         "read",
			fileName:   "data.csv",
			lineNumber: 42,
			err:        errors.New("invalid format"),
			want:       "read: data.csv:42: invalid format",
		},
		{
			name:       "no line number",
			op:         "parse",
			fileName:   "data.csv",
			lineNumber: 0,
			err:        errors.New("parse error"),
			want:       "parse: data.csv: parse error",
		},
		{
			name:       "no file name",
			op:         "validate",
			fileName:   "",
			lineNumber: 0,
			err:        errors.New("validation failed"),
			want:       "validate: validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewProcessingError(tt.op, tt.fileName, tt.lineNumber, tt.err)
			if got := err.Error(); got != tt.want {
				t.Errorf("Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestErrorCollector(t *testing.T) {
	t.Run("add errors", func(t *testing.T) {
		ec := NewErrorCollector(0) // No limit

		_ = ec.Add(errors.New("error 1"))
		_ = ec.Add(errors.New("error 2"))

		if !ec.HasErrors() {
			t.Error("HasErrors() = false, want true")
		}

		if got := ec.Count(); got != 2 {
			t.Errorf("Count() = %d, want 2", got)
		}
	})

	t.Run("error limit", func(t *testing.T) {
		ec := NewErrorCollector(2)

		_ = ec.Add(errors.New("error 1"))
		err := ec.Add(errors.New("error 2"))

		if err != ErrMaxErrorsExceeded {
			t.Errorf("Add() error = %v, want %v", err, ErrMaxErrorsExceeded)
		}
	})

	t.Run("clear errors", func(t *testing.T) {
		ec := NewErrorCollector(0)
		_ = ec.Add(errors.New("error 1"))

		ec.Clear()

		if ec.HasErrors() {
			t.Error("HasErrors() = true after Clear(), want false")
		}
	})
}

func TestValidationError(t *testing.T) {
	err := NewValidationError("age", "invalid", "must be a number")
	want := "validation error: field=age, value=invalid, message=must be a number"

	if got := err.Error(); got != want {
		t.Errorf("Error() = %v, want %v", got, want)
	}
}
