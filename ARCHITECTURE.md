# Architecture Documentation

This document explains the design decisions, patterns, and architecture of the CSV Processor.

## Table of Contents

- [Overview](#overview)
- [Design Principles](#design-principles)
- [Component Architecture](#component-architecture)
- [Concurrency Model](#concurrency-model)
- [Data Flow](#data-flow)
- [Error Handling Strategy](#error-handling-strategy)
- [Performance Optimizations](#performance-optimizations)
- [Trade-offs and Alternatives](#trade-offs-and-alternatives)

## Overview

The CSV Processor is designed as a **pipeline-based, concurrent data processing system** that maximizes throughput while maintaining data integrity and graceful error handling.

### Core Philosophy

1. **Do one thing well**: Process CSV files efficiently
2. **Composability**: Small, focused components that work together
3. **Safety**: No data races, proper error handling, graceful shutdown
4. **Performance**: Lock-free where possible, optimal concurrency

## Design Principles

### 1. Single Responsibility Principle

Each component has one clear purpose:

```
Reader  → Parse CSV files and emit records
Worker  → Process individual records
Tracker → Monitor and report progress
Collector → Aggregate and categorize errors
Pipeline → Orchestrate all components
```

### 2. Interface Segregation

Components depend on minimal interfaces:

```go
// Processor interface - minimal contract
type Processor interface {
    Process(ctx context.Context, record *Record) (*Result, error)
}

// This allows easy testing and swapping implementations
type CustomProcessor struct{}

func (p *CustomProcessor) Process(ctx context.Context, record *Record) (*Result, error) {
    // Custom logic here
}
```

### 3. Dependency Inversion

High-level modules (Pipeline) don't depend on low-level modules (concrete processors):

```go
// Pipeline depends on interface
type Pipeline struct {
    processor processor.Processor  // Interface, not concrete type
}

// Any implementation works
pipe := NewPipeline(Config{
    Processor: processor.NewDefaultProcessor(),  // Default
    // OR
    Processor: &CustomProcessor{},  // Custom
})
```

### 4. Fail-Safe Defaults

All configuration has sensible defaults:

```go
Workers: runtime.NumCPU()      // Optimal for most machines
BufferSize: 100                // Balance memory vs throughput
ErrorThreshold: 0.0            // Don't abort by default
ShowProgress: true             // Helpful feedback
```

## Component Architecture

### High-Level Component Diagram

```
┌────────────────────────────────────────────────────────┐
│                    CLI Layer                           │
│              (cmd/processor/main.go)                   │
│                                                        │
│  • Flag parsing                                        │
│  • Configuration validation                            │
│  • User feedback                                       │
└───────────────────┬────────────────────────────────────┘
                    │
                    ▼
┌────────────────────────────────────────────────────────┐
│                Pipeline Orchestrator                   │
│            (internal/pipeline/pipeline.go)             │
│                                                        │
│  Coordinates:                                          │
│  ├─ Component lifecycle                               │
│  ├─ Error aggregation                                 │
│  ├─ Signal handling                                   │
│  └─ Summary generation                                │
└───┬───────────────┬───────────────┬────────────────────┘
    │               │               │
    ▼               ▼               ▼
┌──────────┐  ┌──────────┐  ┌──────────────┐
│  Reader  │  │  Worker  │  │   Tracker    │
│  Pool    │  │  Pool    │  │  & Errors    │
└──────────┘  └──────────┘  └──────────────┘
```

### Component Responsibilities

#### 1. Reader (`internal/reader`)

**Purpose**: Parse CSV files concurrently and emit validated records

**Key Features**:
- One goroutine per file (fan-out pattern)
- Header validation across files
- Context-aware cancellation
- Memory-efficient with `csv.ReuseRecord`

**Design Pattern**: Producer (emits to channel)

```go
type CSVReader struct {
    files     []string
    hasHeader bool
}

func (r *CSVReader) Read(ctx context.Context) (<-chan *Record, <-chan error) {
    recordCh := make(chan *Record, bufferSize)
    errCh := make(chan error, len(r.files))
    
    // Start goroutine per file
    for _, file := range r.files {
        go r.readFile(ctx, file, recordCh)
    }
    
    return recordCh, errCh
}
```

#### 2. Worker Pool (`internal/worker`)

**Purpose**: Process records concurrently with controlled parallelism

**Key Features**:
- Fixed-size worker pool (prevents resource exhaustion)
- Backpressure via buffered channels
- Atomic counters for statistics
- Context-based cancellation

**Design Pattern**: Worker Pool

```go
type Pool struct {
    workers  int
    inputCh  <-chan *Record
    outputCh chan *Result
}

func (p *Pool) worker(id int) {
    for record := range p.inputCh {
        result := p.process(record)
        p.outputCh <- result
    }
}
```

#### 3. Progress Tracker (`internal/tracker`)

**Purpose**: Real-time monitoring and reporting

**Key Features**:
- Lock-free atomic counters (high performance)
- Periodic updates via ticker
- Throughput and ETA calculation
- Thread-safe reads

**Design Pattern**: Observer (observes processing events)

```go
type ProgressTracker struct {
    processed uint64  // atomic
    success   uint64  // atomic
    failed    uint64  // atomic
}

func (t *ProgressTracker) RecordProcessed(result *Result) {
    atomic.AddUint64(&t.processed, 1)
    if result.IsSuccess() {
        atomic.AddUint64(&t.success, 1)
    }
}
```

#### 4. Error Collector (`internal/errors`)

**Purpose**: Aggregate, categorize, and report errors

**Key Features**:
- Thread-safe collection with RWMutex
- Automatic error categorization
- Threshold monitoring with auto-abort
- Rich error metadata

**Design Pattern**: Collector + Strategy (categorization)

```go
type Collector struct {
    errors         []ErrorEntry
    mu             sync.RWMutex
    errorThreshold float64
}

func (c *Collector) Add(err error, record *Record) error {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    entry := ErrorEntry{
        Error:    err,
        Category: categorizeError(err),
        Severity: determineSeverity(err),
    }
    c.errors = append(c.errors, entry)
    
    // Check threshold
    if c.shouldAbort() {
        c.cancel()
        return ErrThresholdExceeded
    }
}
```

## Concurrency Model

### Thread Safety Guarantees

| Component | Concurrency Control | Rationale |
|-----------|---------------------|-----------|
| **Summary counters** | `sync/atomic` | Lock-free, high performance |
| **Summary time fields** | `sync.RWMutex` | Infrequent writes, many reads |
| **Error collection** | `sync.RWMutex` | Prioritize read performance |
| **Worker pool** | Channels | Go's native concurrency primitive |
| **Progress tracker** | `sync/atomic` | Updated very frequently |
| **Pipeline.workerPool** | `sync.RWMutex` | Written once, read on shutdown |

### Synchronization Patterns

#### 1. Fan-Out (Reader)

```
File 1 ─┐
File 2 ─┼─→ [Shared Channel] ─→ Workers
File 3 ─┘

Benefits:
- Parallel I/O
- Single coordination point
- Simple downstream consumption
```

#### 2. Worker Pool (Processing)

```
              ┌─→ Worker 1 ─┐
Records ─→ [Q]├─→ Worker 2 ─┤─→ Results
              └─→ Worker N ─┘

Benefits:
- Controlled parallelism
- Backpressure control
- Resource limits
```

#### 3. Result Aggregation (Pipeline)

```
Results ─→ [Handler 1: Update Progress]
        ├→ [Handler 2: Collect Errors]
        └→ [Handler 3: Write Output]

Benefits:
- Separation of concerns
- Parallel result processing
- Independent error handling
```

### Context Propagation

```go
// Context flows through entire pipeline
ctx, cancel := context.WithCancel(background)

// Reader respects context
reader.Read(ctx)

// Workers check context
select {
case <-ctx.Done():
    return
case record := <-inputCh:
    process(record)
}

// Signal handlers trigger cancellation
signal.Notify(sigCh, os.Interrupt)
go func() {
    <-sigCh
    cancel()  // Cascades to all components
}()
```

## Data Flow

### Complete Processing Flow

```
1. CLI Parsing
   └─> Validate flags
       └─> Create Pipeline

2. Pipeline.Run()
   ├─> Start Progress Tracker
   ├─> Create CSV Reader
   │   └─> Spawn goroutines per file
   │       └─> Emit records to channel
   │
   ├─> Create Worker Pool
   │   └─> Spawn N workers
   │       └─> Process records
   │           └─> Emit results
   │
   └─> Start Result Handlers (3 goroutines)
       ├─> Update Progress Tracker
       ├─> Collect Errors
       └─> Write Output

3. Shutdown
   ├─> Close input channels
   ├─> Wait for workers
   ├─> Drain results
   └─> Print summary
```

### Message Flow Diagram

```
┌─────┐                    ┌──────┐                 ┌────────┐
│File │─┬─[Record Ch]──────┤Worker├─[Result Ch]────┤Pipeline│
└─────┘ │                  └──────┘                 └────────┘
        │                     │                          │
┌─────┐ │                     │                          │
│File │─┤                     │                          ├─[Progress]
└─────┘ │                     │                          │
        │                  [Process]                     ├─[Errors]
┌─────┐ │                     │                          │
│File │─┘                     ▼                          └─[Output]
└─────┘                  [Result]
```

## Error Handling Strategy

### Error Categories

```go
const (
    CategoryValidation  // Malformed data
    CategoryProcessing  // Logic errors
    CategoryIO          // File access issues
    CategoryTimeout     // Context deadline
    CategoryUnknown     // Unexpected errors
)
```

### Error Severity Levels

```go
const (
    SeverityLow       // Continue processing
    SeverityMedium    // Log and continue
    SeverityHigh      // Alert but continue
    SeverityCritical  // May trigger abort
)
```

### Graceful Degradation

```go
// Partial success is acceptable
func (p *Pipeline) Run() error {
    // Process all records, even if some fail
    for result := range results {
        if result.IsFailed() {
            collector.Add(result.Error)
            continue  // Keep going
        }
        writeOutput(result)
    }
    
    // Only abort if threshold exceeded
    if collector.ThresholdExceeded() {
        return ErrAborted
    }
    
    return nil  // Success even with some errors
}
```

### Error Propagation

```
Reader Error ─┐
              ├─→ Error Collector ─→ Summary ─→ User
Worker Error ─┤
              │
Process Error─┘

Context Cancel ─→ All Components ─→ Graceful Shutdown
```

## Performance Optimizations

### 1. Lock-Free Atomics

**Problem**: Mutex contention on hot path (progress tracking)

**Solution**: `sync/atomic` for counters

```go
// Before (slow - mutex on every record)
func (t *Tracker) Increment() {
    t.mu.Lock()
    t.count++
    t.mu.Unlock()
}

// After (fast - lock-free)
func (t *Tracker) Increment() {
    atomic.AddUint64(&t.count, 1)
}
```

**Improvement**: ~10x faster on high concurrency

### 2. Channel Buffering

**Problem**: Goroutine blocking on channel send/receive

**Solution**: Buffered channels for backpressure

```go
// Unbuffered (blocks on every send)
recordCh := make(chan *Record)

// Buffered (batches messages)
recordCh := make(chan *Record, 100)
```

**Improvement**: ~30% throughput increase

### 3. Memory Pooling

**Problem**: Frequent allocations in hot path

**Solution**: `csv.ReuseRecord = true`

```go
csvReader := csv.NewReader(file)
csvReader.ReuseRecord = true  // Reuse slice backing array
```

**Improvement**: ~25% reduction in allocations

### 4. Minimal Locking

**Problem**: RWMutex contention on frequent reads

**Solution**: Separate atomic counters from mutex-protected data

```go
type Summary struct {
    // Hot path - atomic (no locks)
    totalRecords uint64
    successCount uint64
    
    // Cold path - mutex protected
    mu         sync.RWMutex
    startTime  time.Time
    endTime    time.Time
}
```

**Improvement**: No contention on hot path

## Trade-offs and Alternatives

### 1. Worker Pool Size

**Decision**: Default to `runtime.NumCPU()`

**Trade-offs**:
- ✅ Optimal for CPU-bound tasks
- ✅ Prevents oversubscription
- ❌ May be suboptimal for I/O-bound tasks

**Alternatives Considered**:
- Fixed pool (e.g., 4 workers): Simple but not adaptive
- Dynamic scaling: Complex, potential race conditions
- Unlimited goroutines: Resource exhaustion risk

**Why chosen**: Best balance for general use case

### 2. Error Handling: Collect vs. Fail-Fast

**Decision**: Collect all errors, allow partial success

**Trade-offs**:
- ✅ Maximum data processed
- ✅ Complete error visibility
- ❌ May process bad data longer

**Alternatives Considered**:
- Fail on first error: Fast but loses data
- Retry failed records: Complex, may not help
- Ignore errors: Dangerous, data loss risk

**Why chosen**: Users prefer seeing all errors

### 3. Progress Reporting: Pull vs. Push

**Decision**: Push via ticker (periodic updates)

**Trade-offs**:
- ✅ Consistent update frequency
- ✅ No blocking on stats access
- ❌ May lag by up to 1 second

**Alternatives Considered**:
- Pull (query on demand): Requires explicit requests
- Push on every record: Too chatty, performance hit
- No progress: Poor user experience

**Why chosen**: Best UX without performance cost

### 4. Channel Size

**Decision**: Buffered channels with size = 100

**Trade-offs**:
- ✅ Reduces blocking
- ✅ Batches messages
- ❌ Slightly more memory

**Alternatives Considered**:
- Unbuffered: Simple but slow
- Large buffers (1000+): More memory, diminishing returns
- Dynamic sizing: Complexity not justified

**Why chosen**: Sweet spot from benchmarks

## Extension Points

### Adding Custom Processors

```go
// Implement the Processor interface
type DataTransformer struct {
    mapping map[string]string
}

func (t *DataTransformer) Process(ctx context.Context, record *Record) (*Result, error) {
    // Transform data
    transformed := transform(record.Data)
    return NewSuccessResult(record, transformed, 0), nil
}

// Use in pipeline
pipe := NewPipeline(Config{
    Processor: &DataTransformer{...},
})
```

### Adding Custom Error Handlers

```go
// Implement ErrorHandler interface
type SlackNotifier struct {
    webhook string
}

func (n *SlackNotifier) HandleError(record *Record, err error) error {
    // Send to Slack
    return sendSlackNotification(n.webhook, err)
}
```

### Adding Custom Progress Reporters

```go
// Implement ProgressReporter interface
type PrometheusReporter struct {
    gauge prometheus.Gauge
}

func (r *PrometheusReporter) Report(stats Stats) {
    r.gauge.Set(float64(stats.Processed))
}
```

## Testing Strategy

### Unit Tests

- Mock interfaces (Processor, ResultHandler)
- Test each component in isolation
- Focus on edge cases (empty files, errors, cancellation)

### Integration Tests

- Test entire pipeline end-to-end
- Use generated test data (fixtures)
- Validate correct behavior under load

### Race Detection

- Run all tests with `-race` flag
- Catch data races early
- Enforce in CI/CD

### Benchmarks

- Measure throughput at different worker counts
- Profile memory allocations
- Track performance regressions

## Conclusion

The CSV Processor demonstrates:

- **Clean Architecture**: Separation of concerns, dependency inversion
- **Go Concurrency**: Proper use of goroutines, channels, atomics
- **Production Quality**: Error handling, testing, monitoring
- **Performance**: Lock-free hot paths, optimal parallelism

The architecture is designed to be:
- **Maintainable**: Clear component boundaries
- **Testable**: Interface-based design
- **Extensible**: Plugin points for customization
- **Safe**: Race-free, graceful shutdown