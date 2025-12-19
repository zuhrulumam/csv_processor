package tracker

import (
	"io"
	"sync"
	"time"

	"github.com/zuhrulumam/csv_processor/internal/models"
)

// MultiTracker tracks progress for multiple files
type MultiTracker struct {
	trackers map[string]*ProgressTracker
	global   *ProgressTracker
	mu       sync.RWMutex
}

// NewMultiTracker creates a new multi-file progress tracker
func NewMultiTracker(writer io.Writer, verbose bool) *MultiTracker {
	return &MultiTracker{
		trackers: make(map[string]*ProgressTracker),
		global: NewProgressTracker(Config{
			Writer:         writer,
			UpdateInterval: 1 * time.Second,
			Verbose:        verbose,
		}),
	}
}

// AddFile adds a tracker for a specific file
func (mt *MultiTracker) AddFile(filename string, expectedRecords uint64) *ProgressTracker {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	tracker := NewProgressTracker(Config{
		Writer:         io.Discard, // Individual files don't print
		UpdateInterval: 1 * time.Second,
		TotalRecords:   expectedRecords,
	})

	mt.trackers[filename] = tracker

	return tracker
}

// GetFileTracker returns the tracker for a specific file
func (mt *MultiTracker) GetFileTracker(filename string) *ProgressTracker {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	return mt.trackers[filename]
}

// RecordProcessed records a processed record for both file and global trackers
func (mt *MultiTracker) RecordProcessed(filename string, result *models.Result) {
	mt.mu.RLock()
	if tracker, exists := mt.trackers[filename]; exists {
		tracker.RecordProcessed(result)
	}
	mt.mu.RUnlock()

	mt.global.RecordProcessed(result)
}

// Start starts all trackers
func (mt *MultiTracker) Start() error {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	for _, tracker := range mt.trackers {
		if err := tracker.Start(); err != nil {
			return err
		}
	}

	return mt.global.Start()
}

// Stop stops all trackers
func (mt *MultiTracker) Stop() {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	for _, tracker := range mt.trackers {
		tracker.Stop()
	}

	mt.global.StopAndPrintFinal()
}

// GlobalStats returns global statistics
func (mt *MultiTracker) GlobalStats() Stats {
	return mt.global.Stats()
}

// FileStats returns statistics for a specific file
func (mt *MultiTracker) FileStats(filename string) Stats {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	if tracker, exists := mt.trackers[filename]; exists {
		return tracker.Stats()
	}

	return Stats{}
}

// AllFileStats returns statistics for all files
func (mt *MultiTracker) AllFileStats() map[string]Stats {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	stats := make(map[string]Stats)

	for filename, tracker := range mt.trackers {
		stats[filename] = tracker.Stats()
	}

	return stats
}
