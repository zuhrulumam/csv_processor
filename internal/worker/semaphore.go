package worker

import (
	"context"
	"fmt"
)

// Semaphore provides a way to limit concurrent operations
type Semaphore struct {
	slots chan struct{}
}

// NewSemaphore creates a new semaphore with the given limit
func NewSemaphore(limit int) *Semaphore {
	if limit <= 0 {
		limit = 1
	}

	return &Semaphore{
		slots: make(chan struct{}, limit),
	}
}

// Acquire acquires a slot, blocking if necessary
func (s *Semaphore) Acquire() {
	s.slots <- struct{}{}
}

// TryAcquire attempts to acquire a slot without blocking
// Returns true if successful, false otherwise
func (s *Semaphore) TryAcquire() bool {
	select {
	case s.slots <- struct{}{}:
		return true
	default:
		return false
	}
}

// AcquireContext acquires a slot with context support
// Returns an error if context is canceled before acquiring
func (s *Semaphore) AcquireContext(ctx context.Context) error {
	select {
	case s.slots <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Release releases a slot
func (s *Semaphore) Release() {
	select {
	case <-s.slots:
	default:
		panic("semaphore: release without acquire")
	}
}

// Available returns the number of available slots
func (s *Semaphore) Available() int {
	return cap(s.slots) - len(s.slots)
}

// Limit returns the maximum number of concurrent operations
func (s *Semaphore) Limit() int {
	return cap(s.slots)
}

// PoolWithSemaphore wraps a pool with semaphore-based backpressure
type PoolWithSemaphore struct {
	*Pool
	semaphore *Semaphore
}

// NewPoolWithSemaphore creates a pool with semaphore-based concurrency control
func NewPoolWithSemaphore(config Config, maxConcurrent int) (*PoolWithSemaphore, error) {
	if maxConcurrent <= 0 {
		return nil, fmt.Errorf("maxConcurrent must be positive")
	}

	pool := NewPool(config)

	return &PoolWithSemaphore{
		Pool:      pool,
		semaphore: NewSemaphore(maxConcurrent),
	}, nil
}

// StartWithBackpressure starts the pool with semaphore backpressure
func (p *PoolWithSemaphore) StartWithBackpressure() error {
	if err := p.Start(); err != nil {
		return err
	}

	// Override workers to use semaphore
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.workerWithSemaphore(i)
	}

	return nil
}

// workerWithSemaphore processes records with semaphore-based rate limiting
func (p *PoolWithSemaphore) workerWithSemaphore(id int) {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			return

		case record, ok := <-p.inputCh:
			if !ok {
				return
			}

			// Acquire semaphore slot
			if err := p.semaphore.AcquireContext(p.ctx); err != nil {
				// Context canceled
				return
			}

			// Process record
			result := p.processRecord(record)

			// Release semaphore slot
			p.semaphore.Release()

			// Send result
			select {
			case p.outputCh <- result:
			case <-p.ctx.Done():
				return
			}
		}
	}
}
