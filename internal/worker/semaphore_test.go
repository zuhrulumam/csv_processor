package worker

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestSemaphore_BasicUsage(t *testing.T) {
	sem := NewSemaphore(3)

	// Should be able to acquire 3 times
	sem.Acquire()
	sem.Acquire()
	sem.Acquire()

	// Check available slots
	if sem.Available() != 0 {
		t.Errorf("expected 0 available slots, got %d", sem.Available())
	}

	// Release one
	sem.Release()

	if sem.Available() != 1 {
		t.Errorf("expected 1 available slot, got %d", sem.Available())
	}

	// Release remaining
	sem.Release()
	sem.Release()

	if sem.Available() != 3 {
		t.Errorf("expected 3 available slots, got %d", sem.Available())
	}
}

func TestSemaphore_TryAcquire(t *testing.T) {
	sem := NewSemaphore(1)

	// First acquire should succeed
	if !sem.TryAcquire() {
		t.Error("expected TryAcquire to succeed")
	}

	// Second should fail (limit reached)
	if sem.TryAcquire() {
		t.Error("expected TryAcquire to fail when limit reached")
	}

	// Release and try again
	sem.Release()

	if !sem.TryAcquire() {
		t.Error("expected TryAcquire to succeed after release")
	}
}

func TestSemaphore_AcquireContext(t *testing.T) {
	sem := NewSemaphore(1)
	sem.Acquire() // Fill the semaphore

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Should timeout
	err := sem.AcquireContext(ctx)
	if err == nil {
		t.Error("expected AcquireContext to return error on timeout")
	}

	if err != context.DeadlineExceeded {
		t.Errorf("expected context.DeadlineExceeded, got %v", err)
	}
}

func TestSemaphore_Concurrency(t *testing.T) {
	const (
		limit      = 5
		goroutines = 20
	)

	sem := NewSemaphore(limit)
	active := make(chan int, goroutines)
	done := make(chan bool)

	maxActive := 0

	// Monitor active count
	go func() {
		for count := range active {
			if count > maxActive {
				maxActive = count
			}
		}
		done <- true
	}()

	// Start goroutines
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			sem.Acquire()
			defer sem.Release()

			// Signal we're active
			active <- len(sem.slots)

			// Simulate work
			time.Sleep(10 * time.Millisecond)
		}()
	}

	wg.Wait()
	close(active)
	<-done

	if maxActive > limit {
		t.Errorf("max active goroutines (%d) exceeded limit (%d)", maxActive, limit)
	}
}

func TestSemaphore_Limit(t *testing.T) {
	tests := []struct {
		name          string
		limit         int
		expectedLimit int
	}{
		{"positive limit", 5, 5},
		{"zero defaults to 1", 0, 1},
		{"negative defaults to 1", -1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sem := NewSemaphore(tt.limit)

			if sem.Limit() != tt.expectedLimit {
				t.Errorf("expected limit %d, got %d", tt.expectedLimit, sem.Limit())
			}
		})
	}
}

func BenchmarkSemaphore_Acquire(b *testing.B) {
	sem := NewSemaphore(100)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		sem.Acquire()
		sem.Release()
	}
}

func BenchmarkSemaphore_Concurrent(b *testing.B) {
	sem := NewSemaphore(10)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			sem.Acquire()
			sem.Release()
		}
	})
}
