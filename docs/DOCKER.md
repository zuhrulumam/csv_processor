# Docker Guide

This guide explains how to use the CSV Processor with Docker.

## Quick Start

### Build the image
```bash
docker build -t csv-processor:latest .
```

Or use the build script:
```bash
./scripts/docker-build.sh
```

### Run with Docker
```bash
# Process a single file
docker run --rm \
  -v $(pwd)/examples/input:/data/input:ro \
  -v $(pwd)/examples/output:/data/output \
  csv-processor:latest \
  -workers 4 \
  -output /data/output/results.csv \
  /data/input/sample.csv
```

Or use the run script:
```bash
./scripts/docker-run.sh sample.csv
```

### Run with Docker Compose
```bash
# Process files
docker-compose up processor

# Run tests
docker-compose up test

# Run benchmarks
docker-compose up bench
```

## Docker Images

### Production Image

Optimized for minimal size and security:

- Multi-stage build
- Alpine Linux base (< 20MB)
- Non-root user
- No shell or unnecessary tools

**Build:**
```bash
docker build -t csv-processor:latest .
```

**Size:** ~15-20MB

### Development Image

Includes development tools:

- Full Go toolchain
- git, make, gcc
- Hot reload with air (optional)

**Build:**
```bash
docker build -f Dockerfile.dev -t csv-processor:dev .
```

**Size:** ~400MB

## Usage Examples

### Basic Processing
```bash
docker run --rm \
  -v $(pwd)/data:/data \
  csv-processor:latest \
  /data/input.csv
```

### Multiple Files
```bash
docker run --rm \
  -v $(pwd)/data:/data \
  csv-processor:latest \
  /data/file1.csv /data/file2.csv /data/file3.csv
```

### With All Options
```bash
docker run --rm \
  -v $(pwd)/input:/data/input:ro \
  -v $(pwd)/output:/data/output \
  csv-processor:latest \
  -workers 8 \
  -buffer 500 \
  -error-threshold 0.05 \
  -abort-on-error \
  -progress \
  -verbose \
  -output /data/output/results.csv \
  /data/input/data.csv
```

### Environment Variables
```bash
docker run --rm \
  -e TZ=America/New_York \
  -v $(pwd)/data:/data \
  csv-processor:latest \
  /data/input.csv
```

## Volume Mounts

### Input Files (Read-Only)
```bash
-v $(pwd)/input:/data/input:ro
```

The `:ro` flag mounts the directory as read-only for security.

### Output Files (Read-Write)
```bash
-v $(pwd)/output:/data/output
```

Writable mount for saving processed results.

## Docker Compose Workflows

### Development Workflow
```bash
# Start development container
docker-compose up processor-dev

# Run tests
docker-compose up test

# Run benchmarks
docker-compose up bench
```

### Production Workflow
```bash
# Process files
docker-compose up processor

# Check results
ls -lh examples/output/
```

### Batch Processing
```bash
# Process multiple datasets
docker-compose -f docker-compose.example.yml up process-batch
```

## Performance Tuning

### CPU Limits
```bash
docker run --rm \
  --cpus="4.0" \
  -v $(pwd)/data:/data \
  csv-processor:latest \
  -workers 4 \
  /data/input.csv
```

### Memory Limits
```bash
docker run --rm \
  --memory="2g" \
  --memory-swap="2g" \
  -v $(pwd)/data:/data \
  csv-processor:latest \
  -workers 4 \
  /data/input.csv
```

### Resource Limits in Compose
```yaml
services:
  processor:
    image: csv-processor:latest
    deploy:
      resources:
        limits:
          cpus: '4.0'
          memory: 2G
        reservations:
          cpus: '2.0'
          memory: 1G
```

## Troubleshooting

### Permission Issues

If you get permission errors:
```bash
# Fix ownership
docker run --rm \
  -v $(pwd)/output:/data/output \
  csv-processor:latest \
  sh -c "chown -R $(id -u):$(id -g) /data/output"
```

### See Container Logs
```bash
docker logs csv-processor
```

### Interactive Debugging
```bash
docker run --rm -it \
  --entrypoint /bin/sh \
  -v $(pwd)/data:/data \
  csv-processor:latest
```

### Test Inside Container
```bash
docker run --rm \
  -v $(pwd):/build \
  -w /build \
  golang:1.21-alpine \
  go test -v ./...
```

## Security Best Practices

1. **Run as non-root user** ✅ (Already configured)
2. **Use read-only mounts** for input files
3. **Limit resources** with `--cpus` and `--memory`
4. **Keep base image updated** (`docker pull alpine:3.19`)
5. **Scan for vulnerabilities**:
```bash
   docker scan csv-processor:latest
```

## CI/CD Integration

### GitHub Actions
```yaml
- name: Build Docker image
  run: docker build -t csv-processor:${{ github.sha }} .

- name: Run tests in Docker
  run: docker run csv-processor:${{ github.sha }} go test -v ./...

- name: Push to registry
  run: |
    docker tag csv-processor:${{ github.sha }} myregistry/csv-processor:latest
    docker push myregistry/csv-processor:latest
```

### GitLab CI
```yaml
build:
  script:
    - docker build -t csv-processor:$CI_COMMIT_SHA .
    - docker run csv-processor:$CI_COMMIT_SHA go test -v ./...
```

## Registry Publishing

### Docker Hub
```bash
docker tag csv-processor:latest username/csv-processor:latest
docker push username/csv-processor:latest
```

### GitHub Container Registry
```bash
docker tag csv-processor:latest ghcr.io/username/csv-processor:latest
docker push ghcr.io/username/csv-processor:latest
```

## Cleanup
```bash
# Remove containers
docker-compose down

# Remove images
docker rmi csv-processor:latest csv-processor:dev

# Remove volumes (careful!)
docker volume prune

# Full cleanup
docker system prune -a
```