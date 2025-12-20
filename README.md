## Architecture

### Core Models
- **Record**: Represents a CSV row with metadata (line number, filename)
- **Result**: Processing outcome with status, error, and timing
- **Summary**: Aggregated statistics across all records

### Interfaces
- **Processor**: Contract for record transformation logic
- **ResultHandler**: Contract for handling processed results

## Docker Usage

### Quick Start
```bash
# Build image
make docker-build

# Run processor
make docker-run

# Or use docker-compose
docker-compose up processor
```

### Docker Commands
```bash
# Build
docker build -t csv-processor:latest .

# Run
docker run --rm \
  -v $(pwd)/examples/input:/data/input:ro \
  -v $(pwd)/examples/output:/data/output \
  csv-processor:latest \
  -workers 4 /data/input/sample.csv

# Run tests
make docker-test
```

See [Docker Guide](docs/DOCKER.md) for detailed documentation.