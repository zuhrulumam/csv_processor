package errors

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

// Reporter formats and reports error information
type Reporter struct {
	collector *Collector
	writer    io.Writer
}

// NewReporter creates a new error reporter
func NewReporter(collector *Collector, writer io.Writer) *Reporter {
	return &Reporter{
		collector: collector,
		writer:    writer,
	}
}

// PrintSummary prints a summary of all errors
func (r *Reporter) PrintSummary() {
	summary := r.collector.Summary()

	fmt.Fprintf(r.writer, "\n")
	fmt.Fprintf(r.writer, "========================================\n")
	fmt.Fprintf(r.writer, "Error Summary\n")
	fmt.Fprintf(r.writer, "========================================\n")
	fmt.Fprintf(r.writer, "Total Errors:      %d\n", summary.TotalErrors)
	fmt.Fprintf(r.writer, "Total Processed:   %d\n", summary.TotalProcessed)
	fmt.Fprintf(r.writer, "Error Rate:        %.2f%%\n", summary.ErrorRate*100)
	fmt.Fprintf(r.writer, "Retryable Errors:  %d\n", summary.RetryableErrors)
	fmt.Fprintf(r.writer, "\n")

	// Print by category
	if len(summary.ByCategory) > 0 {
		fmt.Fprintf(r.writer, "Errors by Category:\n")
		for category, count := range summary.ByCategory {
			fmt.Fprintf(r.writer, "  %-15s: %d\n", category, count)
		}
		fmt.Fprintf(r.writer, "\n")
	}

	// Print by severity
	if len(summary.BySeverity) > 0 {
		fmt.Fprintf(r.writer, "Errors by Severity:\n")
		for severity, count := range summary.BySeverity {
			fmt.Fprintf(r.writer, "  %-15s: %d\n", severity, count)
		}
		fmt.Fprintf(r.writer, "\n")
	}

	fmt.Fprintf(r.writer, "========================================\n")
}

// PrintDetailed prints detailed error information
func (r *Reporter) PrintDetailed(maxErrors int) {
	errors := r.collector.Errors()

	if len(errors) == 0 {
		fmt.Fprintf(r.writer, "No errors to report.\n")
		return
	}

	fmt.Fprintf(r.writer, "\n")
	fmt.Fprintf(r.writer, "========================================\n")
	fmt.Fprintf(r.writer, "Detailed Error Report\n")
	fmt.Fprintf(r.writer, "========================================\n")

	// Limit number of errors shown
	count := len(errors)
	if maxErrors > 0 && maxErrors < count {
		count = maxErrors
	}

	for i := 0; i < count; i++ {
		entry := errors[i]

		fmt.Fprintf(r.writer, "\nError #%d:\n", i+1)
		fmt.Fprintf(r.writer, "  Time:      %s\n", entry.Timestamp.Format(time.RFC3339))
		fmt.Fprintf(r.writer, "  Category:  %s\n", entry.Category)
		fmt.Fprintf(r.writer, "  Severity:  %s\n", entry.Severity)
		fmt.Fprintf(r.writer, "  Retryable: %v\n", entry.Retryable)

		if entry.Record != nil {
			fmt.Fprintf(r.writer, "  File:      %s\n", entry.Record.FileName)
			fmt.Fprintf(r.writer, "  Line:      %d\n", entry.Record.LineNumber)
		}

		fmt.Fprintf(r.writer, "  Error:     %v\n", entry.Error)
	}

	if maxErrors > 0 && len(errors) > maxErrors {
		fmt.Fprintf(r.writer, "\n... and %d more errors\n", len(errors)-maxErrors)
	}

	fmt.Fprintf(r.writer, "\n========================================\n")
}

// PrintTopErrors prints the most common errors
func (r *Reporter) PrintTopErrors(topN int) {
	errors := r.collector.Errors()

	if len(errors) == 0 {
		return
	}

	// Group errors by message
	errorCounts := make(map[string]int)
	errorExamples := make(map[string]ErrorEntry)

	for _, entry := range errors {
		msg := entry.Error.Error()
		errorCounts[msg]++
		if _, exists := errorExamples[msg]; !exists {
			errorExamples[msg] = entry
		}
	}

	// Sort by count
	type errorCount struct {
		message string
		count   int
	}

	var sorted []errorCount
	for msg, count := range errorCounts {
		sorted = append(sorted, errorCount{msg, count})
	}

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].count > sorted[j].count
	})

	// Print top N
	if topN > len(sorted) {
		topN = len(sorted)
	}

	fmt.Fprintf(r.writer, "\n")
	fmt.Fprintf(r.writer, "========================================\n")
	fmt.Fprintf(r.writer, "Top %d Most Common Errors\n", topN)
	fmt.Fprintf(r.writer, "========================================\n")

	for i := 0; i < topN; i++ {
		item := sorted[i]
		example := errorExamples[item.message]

		fmt.Fprintf(r.writer, "\n%d. (%d occurrences)\n", i+1, item.count)
		fmt.Fprintf(r.writer, "   Category: %s\n", example.Category)
		fmt.Fprintf(r.writer, "   Message:  %s\n", truncateString(item.message, 100))
	}

	fmt.Fprintf(r.writer, "\n========================================\n")
}

// ExportToFile exports errors to a file
func (r *Reporter) ExportToFile(filename string) error {
	// This would write errors to a CSV or JSON file
	// Implementation omitted for brevity
	return nil
}

// truncateString truncates a string to maxLen
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// FormatError formats a single error entry
func FormatError(entry ErrorEntry) string {
	var parts []string

	if entry.Record != nil {
		parts = append(parts, fmt.Sprintf("%s:%d", entry.Record.FileName, entry.Record.LineNumber))
	}

	parts = append(parts, fmt.Sprintf("[%s]", entry.Category))
	parts = append(parts, entry.Error.Error())

	return strings.Join(parts, " ")
}
