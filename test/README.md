# Test Suite

This directory contains comprehensive tests for the CSV processor.

## Structure
```
test/
├── fixtures/           # Test data generation
│   └── generator.go   # CSV file generators
├── integration/       # End-to-end integration tests
│   ├── integration_test.go
│   └── race_test.go  # Race condition tests
└── README.md         # This file
```

## Running Tests

### All tests
```bash
make test
```

### With race detector
```bash
go test -race ./...
```

### Integration tests only
```bash
go test -v ./test/integration/...
```

### Skip long-running tests
```bash
go test -short ./...
```

### With coverage
```bash
make test-coverage
open coverage.html
```

## Benchmarks

### Run all benchmarks
```bash
make bench
```

### Specific benchmark
```bash
go test -bench=BenchmarkIntegration_1000Records -benchmem ./test/integration/
```

### Profile CPU
```bash
go test -bench=. -cpuprofile=cpu.prof ./test/integration/
go tool pprof cpu.prof
```

### Profile Memory
```bash
go test -bench=. -memprofile=mem.prof ./test/integration/
go tool pprof mem.prof
```

## Test Data

Test data is generated on-the-fly using `fixtures.Generator`:

- **Simple**: Standard CSV with consistent format
- **WithErrors**: CSV with intentional format errors
- **Large**: Large files for performance testing
- **Multiple**: Multiple files for concurrent processing

## CI/CD

Tests run automatically on:
- Push to main/develop branches
- Pull requests

See `.github/workflows/test.yml` for configuration.

## Coverage Goals

- Overall: > 80%
- Critical paths: > 90%
- Error handling: > 95%

Current coverage: Run `make test-coverage` to see latest.