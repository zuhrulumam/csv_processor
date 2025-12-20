# CSV Processor

A high-performance, concurrent CSV file processor built in Go with support for multi-file processing, error handling, and real-time progress tracking.

## Features

✨ **High Performance**
- Concurrent processing with configurable worker pools
- Lock-free atomic operations for statistics
- Channel-based backpressure control
- Optimized for large files (tested with 100k+ records)

🔧 **Flexible Processing**
- Process single or multiple CSV files
- Custom processor interface
- Configurable buffer sizes
- Header validation across files

📊 **Rich Monitoring**
- Real-time progress tracking
- Throughput metrics (records/second)
- ETA calculation
- Detailed error reporting

🛡️ **Robust Error Handling**
- Error rate thresholds with auto-abort
- Categorized errors (validation, I/O, processing)
- Graceful shutdown on signals (SIGINT, SIGTERM)
- Partial success support

🐳 **Production Ready**
- Docker support with multi-stage builds
- Comprehensive test suite (>80% coverage)
- Race condition free (tested with `-race`)
- CI/CD ready with GitHub Actions

## Quick Start

### Installation

```bash
# Clone repository
git clone https://github.com/zuhrulumam/csv_processor.git
cd csv_processor

# Build
make build

# Or install directly
go install github.com/zuhrulumam/csv_processor/cmd/processor@latest
```

### Basic Usage

```bash
# Process a single file
./processor data.csv

# Process multiple files with 8 workers
./processor -workers 8 file1.csv file2.csv file3.csv

# With output file
./processor -output results.csv data.csv

# Show version
./processor -version
```

## Usage Examples

### Process Large File

```bash
./processor -workers 8 -buffer 1000 -progress large-dataset.csv
```

Output:
```
========================================
CSV Processor Starting
========================================
Files:          1
Workers:        8
Buffer Size:    1000
Has Header:     true
========================================

[5s] Progress: 45823/100000 (45.8%) | Success: 45821 | Failed: 2 | 9164 rec/s | ETA: 5s

========================================
Processing Complete
========================================
Total Time:       10.923s
Total Processed:  100000
Successful:       99998 (100.0%)
Failed:           2 (0.0%)
Avg Throughput:   9155 records/sec
========================================
```

### Error Threshold

```bash
# Abort if error rate exceeds 5%
./processor -error-threshold 0.05 -abort-on-error data.csv
```

### Quiet Mode (for automation)

```bash
./processor -quiet -output results.csv data.csv
echo $?  # 0 = success, 1 = failure
```

### Docker

```bash
# Build image
docker build -t csv_processor:latest .

# Run
docker run --rm \
  -v $(pwd)/data:/data \
  csv_processor:latest \
  -workers 4 /data/input.csv

# Or use docker-compose
docker-compose up processor
```

## Command Line Options

```
Usage:
  processor [options] <file1.csv> [file2.csv ...]

Options:
  -header             CSV files have header row (default: true)
  -validate-header    Validate header consistency (default: true)
  -workers N          Number of worker goroutines (default: NumCPU)
  -buffer N           Channel buffer size (default: 100)
  -max-errors N       Maximum errors to collect (default: 0 = unlimited)
  -error-threshold F  Error rate threshold 0.0-1.0 (default: 0.0 = disabled)
  -abort-on-error     Abort when error threshold exceeded (default: false)
  -output FILE        Output file path (default: none)
  -progress           Show progress updates (default: true)
  -verbose            Verbose output (default: false)
  -quiet              Suppress all output except errors (default: false)
  -version            Show version information

Examples:
  # Process a single file
  processor data.csv

  # Process multiple files with 8 workers
  processor -workers 8 file1.csv file2.csv file3.csv

  # Abort if error rate exceeds 10%
  processor -error-threshold 0.1 -abort-on-error data.csv

  # Quiet mode with output file
  processor -quiet -output results.csv data.csv
```

## Architecture

The processor uses a pipeline architecture with the following components:

```
┌─────────────┐
│   Reader    │  Concurrent CSV reading
│  (goroutine │  Multiple files → Single channel
│   per file) │
└──────┬──────┘
       │
       ▼
┌──────────────┐
│ Work Queue   │  Buffered channel
│ (records)    │  Backpressure control
└──────┬───────┘
       │
  ┌────┴────┬────────┬────────┐
  ▼         ▼        ▼        ▼
┌────────┐┌────────┐┌────────┐
│Worker 1││Worker 2││Worker N│  Configurable pool
│        ││        ││        │  Context cancellation
└────┬───┘└────┬───┘└────┬───┘
     │         │         │
     └─────────┴─────────┘
              │
              ▼
       ┌──────────────┐
       │   Results    │  Aggregation
       │   Tracker    │  Progress
       │   Errors     │  Error handling
       └──────────────┘
```

**Key Design Decisions:**

- **Concurrency**: Fan-out reader → Worker pool → Result aggregation
- **Atomics**: Lock-free counters for high-performance statistics
- **Channels**: Buffered channels for backpressure control
- **Context**: Graceful cancellation propagation
- **Mutex**: RWMutex for protecting shared state (workerPool, time fields)

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for detailed design documentation.

## Performance

Benchmarks on Apple M1 (8 cores):

| Records | Workers | Throughput | Duration |
|---------|---------|------------|----------|
| 1,000 | 4 | 76,020 rec/s | 13ms |
| 10,000 | 4 | 125,000 rec/s | 80ms |
| 100,000 | 8 | 149,241 rec/s | 670ms |
| 1,000,000 | 16 | 185,000 rec/s | 5.4s |

See [docs/PERFORMANCE.md](docs/PERFORMANCE.md) for detailed benchmarks and optimization guide.

## Testing

### Run Tests

```bash
# All tests
make test

# With race detector
go test -race ./...

# Coverage report
make test-coverage
open coverage.html

# Benchmarks
make bench
```

### Test Coverage

Current coverage: **85%**

```
internal/models/      90%
internal/reader/      88%
internal/worker/      92%
internal/tracker/     87%
internal/errors/      85%
internal/pipeline/    82%
```

### CI/CD

Tests run automatically on:
- Push to main/develop branches
- Pull requests
- Includes: unit tests, integration tests, race detection, benchmarks

## Development

### Prerequisites

- Go 1.21 or higher
- Make (optional, for Makefile targets)
- Docker (optional, for containerization)

### Setup

```bash
# Clone repository
git clone https://github.com/zuhrulumam/csv_processor.git
cd csv_processor

# Install dependencies
make deps

# Run tests
make test

# Build
make build
```

### Project Structure

```
csv_processor/
├── cmd/
│   └── processor/          # CLI entry point
│       └── main.go
├── internal/
│   ├── models/            # Domain models (Record, Result, Summary)
│   ├── reader/            # CSV reader with concurrency
│   ├── worker/            # Worker pool implementation
│   ├── tracker/           # Progress tracking
│   ├── errors/            # Error collection and reporting
│   ├── processor/         # Processor interface
│   └── pipeline/          # Pipeline orchestration
├── test/
│   ├── fixtures/          # Test data generation
│   └── integration/       # Integration tests
├── examples/              # Example CSV files
├── docs/                  # Documentation
├── scripts/               # Helper scripts
├── Dockerfile            # Production Docker image
├── docker-compose.yml    # Docker orchestration
└── Makefile              # Build automation
```

## AI Usage Transparency

This project was developed with AI assistance. See [AI_USAGE.md](AI_USAGE.md) for details on:
- Where AI tools were used (boilerplate, tests, Docker configs)
- What was written manually (core concurrency logic, algorithms)
- How AI assistance is documented in commits

## Contributing

Contributions are welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### Development Workflow

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Write tests for your changes
4. Ensure tests pass (`make test`)
5. Ensure no race conditions (`go test -race ./...`)
6. Commit with descriptive messages
7. Push to your fork
8. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Built with [Go](https://golang.org/)
- Inspired by Unix philosophy: Do one thing well
- Developed as part of technical interview assessment

## Contact

- GitHub: [@zuhrulumam](https://github.com/zuhrulumam)
- Email: zuhrulu@gmail.com

---

**Note**: This is a demonstration project showcasing Go concurrency patterns, clean architecture, and production-ready practices.