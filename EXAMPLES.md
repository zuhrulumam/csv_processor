# Usage Examples

This document provides practical examples of using the CSV Processor in various scenarios.

## Table of Contents

- [Basic Usage](#basic-usage)
- [Advanced Options](#advanced-options)
- [Real-World Scenarios](#real-world-scenarios)
- [Docker Examples](#docker-examples)
- [Automation Scripts](#automation-scripts)
- [Error Handling](#error-handling)
- [Performance Tuning](#performance-tuning)

## Basic Usage

### Process Single File

```bash
# Simple processing
./processor data.csv

# Output:
# ========================================
# CSV Processor Starting
# ========================================
# Files:          1
# Workers:        8
# Buffer Size:    100
# Has Header:     true
# ========================================
# 
# [2s] Progress: 10000/10000 (100.0%) | Success: 9998 | Failed: 2 | 5000 rec/s | ETA: 0s
#
# ========================================
# Processing Complete
# ========================================
# Total Time:       2.0s
# Total Processed:  10000
# Successful:       9998 (99.98%)
# Failed:           2 (0.02%)
# Avg Throughput:   5000 records/sec
# ========================================
```

### Process Multiple Files

```bash
# Process all CSV files in directory
./processor data/*.csv

# Process specific files
./processor sales.csv customers.csv products.csv
```

### Save Output

```bash
# Save processed records to file
./processor -output results.csv data.csv

# Verify output
head results.csv
```

## Advanced Options

### Configure Worker Count

```bash
# Use 4 workers
./processor -workers 4 data.csv

# Use all CPU cores (default)
./processor -workers $(nproc) data.csv

# Use 2x CPU cores for I/O-bound tasks
./processor -workers $(($(nproc) * 2)) data.csv
```

### Adjust Buffer Size

```bash
# Small buffer for memory-constrained environments
./processor -buffer 50 data.csv

# Large buffer for high throughput
./processor -buffer 1000 data.csv

# Default (100) works well for most cases
./processor data.csv
```

### Progress Reporting

```bash
# Show detailed progress (default)
./processor -progress data.csv

# Verbose mode with extra details
./processor -verbose data.csv

# Quiet mode (only errors)
./processor -quiet data.csv
```

### Header Handling

```bash
# Process files with headers (default)
./processor -header data.csv

# Process files without headers
./processor -header=false data.csv

# Validate header consistency across files
./processor -validate-header sales-2023.csv sales-2024.csv
```

## Real-World Scenarios

### Scenario 1: Data Migration

**Task**: Migrate 500k customer records from CSV to database

```bash
#!/bin/bash
# migrate-customers.sh

INPUT_DIR="/data/exports"
OUTPUT_DIR="/data/processed"
LOG_FILE="/var/log/migration.log"

echo "Starting migration at $(date)" | tee -a "$LOG_FILE"

./processor \
  -workers 16 \
  -buffer 1000 \
  -error-threshold 0.01 \
  -abort-on-error \
  -output "$OUTPUT_DIR/customers.csv" \
  "$INPUT_DIR"/customers-*.csv \
  2>&1 | tee -a "$LOG_FILE"

if [ $? -eq 0 ]; then
    echo "Migration successful" | tee -a "$LOG_FILE"
    # Load to database
    psql -d mydb -c "\COPY customers FROM '$OUTPUT_DIR/customers.csv' CSV HEADER"
else
    echo "Migration failed" | tee -a "$LOG_FILE"
    exit 1
fi
```

### Scenario 2: ETL Pipeline

**Task**: Extract, transform, and load sales data daily

```bash
#!/bin/bash
# daily-etl.sh

DATE=$(date +%Y-%m-%d)
SOURCE="/data/raw/sales-$DATE.csv"
STAGING="/data/staging/sales-processed-$DATE.csv"
ARCHIVE="/data/archive/"

# Check if source file exists
if [ ! -f "$SOURCE" ]; then
    echo "Error: Source file not found: $SOURCE"
    exit 1
fi

# Process with strict error handling
./processor \
  -workers 8 \
  -error-threshold 0.05 \
  -abort-on-error \
  -max-errors 100 \
  -output "$STAGING" \
  -quiet \
  "$SOURCE"

EXIT_CODE=$?

if [ $EXIT_CODE -eq 0 ]; then
    echo "Processing successful: $STAGING"
    
    # Archive original
    mv "$SOURCE" "$ARCHIVE"
    
    # Trigger downstream processes
    ./load-to-warehouse.sh "$STAGING"
else
    echo "Processing failed with exit code: $EXIT_CODE"
    
    # Send alert
    curl -X POST \
      -H 'Content-Type: application/json' \
      -d "{\"text\":\"ETL failed for $DATE\"}" \
      "$SLACK_WEBHOOK_URL"
    
    exit $EXIT_CODE
fi
```

### Scenario 3: Data Validation

**Task**: Validate customer data before importing

```bash
#!/bin/bash
# validate-data.sh

INPUT_FILE="$1"
REPORT_FILE="validation-report-$(date +%Y%m%d).txt"

echo "Validating: $INPUT_FILE" | tee "$REPORT_FILE"
echo "====================================" | tee -a "$REPORT_FILE"

# Run processor in validation mode
./processor \
  -workers 4 \
  -verbose \
  -max-errors 0 \
  "$INPUT_FILE" \
  2>&1 | tee -a "$REPORT_FILE"

EXIT_CODE=$?

# Check results
if [ $EXIT_CODE -eq 0 ]; then
    echo "✓ Validation passed" | tee -a "$REPORT_FILE"
    exit 0
else
    echo "✗ Validation failed - see report: $REPORT_FILE" | tee -a "$REPORT_FILE"
    exit 1
fi
```

### Scenario 4: Batch Processing

**Task**: Process multiple datasets overnight

```bash
#!/bin/bash
# batch-process.sh

LOG_DIR="/var/log/batch"
mkdir -p "$LOG_DIR"

DATASETS=(
    "/data/sales/2024-01.csv:sales-processed-01.csv"
    "/data/sales/2024-02.csv:sales-processed-02.csv"
    "/data/sales/2024-03.csv:sales-processed-03.csv"
)

TOTAL=${#DATASETS[@]}
SUCCESS=0
FAILED=0

for dataset in "${DATASETS[@]}"; do
    IFS=':' read -r INPUT OUTPUT <<< "$dataset"
    
    echo "Processing: $INPUT"
    
    ./processor \
      -workers 8 \
      -buffer 500 \
      -output "/data/processed/$OUTPUT" \
      -quiet \
      "$INPUT" \
      > "$LOG_DIR/$(basename $INPUT).log" 2>&1
    
    if [ $? -eq 0 ]; then
        ((SUCCESS++))
        echo "✓ Success: $INPUT"
    else
        ((FAILED++))
        echo "✗ Failed: $INPUT"
    fi
done

echo ""
echo "Batch processing complete"
echo "Total: $TOTAL, Success: $SUCCESS, Failed: $FAILED"

[ $FAILED -eq 0 ]
```

## Docker Examples

### Basic Docker Usage

```bash
# Build image
docker build -t csv-processor:latest .

# Run with volume mounts
docker run --rm \
  -v $(pwd)/input:/data/input:ro \
  -v $(pwd)/output:/data/output \
  csv-processor:latest \
  -workers 4 \
  -output /data/output/results.csv \
  /data/input/data.csv
```

### Docker Compose

```bash
# Process with docker-compose
docker-compose up processor

# Run tests
docker-compose up test

# Run benchmarks
docker-compose up bench
```

### Production Docker Setup

```bash
# Production configuration
docker run -d \
  --name csv-processor \
  --cpus="8.0" \
  --memory="4g" \
  --restart=unless-stopped \
  -v /data/input:/data/input:ro \
  -v /data/output:/data/output \
  -e TZ=America/New_York \
  csv-processor:latest \
  -workers 16 \
  -buffer 1000 \
  -quiet \
  -output /data/output/processed.csv \
  /data/input/*.csv

# Check logs
docker logs -f csv-processor

# Stop container
docker stop csv-processor
```

## Automation Scripts

### Cron Job

```bash
# Add to crontab (crontab -e)

# Process daily at 2 AM
0 2 * * * /usr/local/bin/csv-processor -workers 8 -quiet /data/daily/*.csv >> /var/log/processor.log 2>&1

# Process hourly
0 * * * * /usr/local/bin/csv-processor -workers 4 -quiet /data/hourly/*.csv

# Weekly cleanup
0 0 * * 0 find /data/processed -name "*.csv" -mtime +30 -delete
```

### Systemd Service

```ini
# /etc/systemd/system/csv-processor.service

[Unit]
Description=CSV Processor Service
After=network.target

[Service]
Type=oneshot
User=dataprocessor
Group=dataprocessor
WorkingDirectory=/opt/csv-processor
ExecStart=/usr/local/bin/csv-processor \
  -workers 8 \
  -buffer 500 \
  -output /data/processed/output.csv \
  /data/input/*.csv
StandardOutput=append:/var/log/csv-processor/stdout.log
StandardError=append:/var/log/csv-processor/stderr.log

[Install]
WantedBy=multi-user.target
```

```bash
# Enable and start
sudo systemctl enable csv-processor.service
sudo systemctl start csv-processor.service

# Check status
sudo systemctl status csv-processor.service

# View logs
sudo journalctl -u csv-processor.service -f
```

### Monitoring Script

```bash
#!/bin/bash
# monitor-processing.sh

THRESHOLD=1000  # Alert if throughput < 1000 rec/s

./processor -workers 8 -verbose data.csv 2>&1 | while read line; do
    echo "$line"
    
    # Extract throughput
    if [[ $line =~ ([0-9]+)\ rec/s ]]; then
        THROUGHPUT="${BASH_REMATCH[1]}"
        
        if [ "$THROUGHPUT" -lt "$THRESHOLD" ]; then
            echo "WARNING: Low throughput: $THROUGHPUT rec/s"
            
            # Send alert
            curl -X POST \
              -H 'Content-Type: application/json' \
              -d "{\"text\":\"Low throughput: $THROUGHPUT rec/s\"}" \
              "$SLACK_WEBHOOK_URL"
        fi
    fi
done
```

## Error Handling

### Abort on High Error Rate

```bash
# Abort if error rate exceeds 5%
./processor \
  -error-threshold 0.05 \
  -abort-on-error \
  data.csv

# Exit code: 0 = success, 1 = aborted
echo "Exit code: $?"
```

### Collect Limited Errors

```bash
# Collect maximum 100 errors, then continue
./processor \
  -max-errors 100 \
  data.csv
```

### Save Error Report

```bash
# Redirect error output to file
./processor data.csv 2> errors.log

# Or use tee to see and save
./processor data.csv 2>&1 | tee processing.log
```

## Performance Tuning

### Benchmark Different Configurations

```bash
#!/bin/bash
# benchmark.sh

FILE="large-dataset.csv"

echo "Benchmarking different configurations..."
echo ""

for WORKERS in 1 2 4 8 16; do
    echo "Testing with $WORKERS workers..."
    
    time ./processor \
      -workers $WORKERS \
      -quiet \
      "$FILE"
    
    echo ""
done
```

### Memory Profiling

```bash
# Run with memory profiling
GODEBUG=gctrace=1 ./processor -workers 8 large.csv

# Monitor memory usage
watch -n 1 'ps aux | grep processor'
```

### CPU Profiling

```bash
# Build with profiling
go build -o processor ./cmd/processor

# Run and profile
./processor -workers 8 data.csv &
PID=$!

# Collect profile
go tool pprof -http=:8080 http://localhost:6060/debug/pprof/profile?seconds=30

# Kill process
kill $PID
```

## Tips and Tricks

### Process Compressed Files

```bash
# Decompress and process
gunzip -c data.csv.gz | ./processor -

# Or use process substitution
./processor <(gunzip -c data.csv.gz)
```

### Process Remote Files

```bash
# Download and process
curl -s https://example.com/data.csv | ./processor -

# Or with wget
wget -qO- https://example.com/data.csv | ./processor -
```

### Chain with Other Tools

```bash
# Filter then process
cat data.csv | grep "2024" | ./processor -

# Process then analyze
./processor data.csv -output - | awk -F',' '{sum+=$3} END {print sum}'

# Parallel processing
find /data -name "*.csv" | xargs -P 4 -I {} ./processor -quiet {}
```

### Environment Variables

```bash
# Set defaults via environment
export CSV_WORKERS=8
export CSV_BUFFER=500

# Use in scripts
./processor -workers ${CSV_WORKERS:-4} data.csv
```

## Troubleshooting

### Debug Mode

```bash
# Enable verbose logging
./processor -verbose data.csv

# Show all processing details
./processor -verbose -progress data.csv 2>&1 | tee debug.log
```

### Check Resource Usage

```bash
# Monitor during execution
./processor data.csv &
PID=$!

# Watch resources
top -pid $PID

# Or use htop
htop -p $PID
```

### Validate Input

```bash
# Check CSV format
head -n 10 data.csv

# Count records
wc -l data.csv

# Validate headers
head -n 1 data.csv | tr ',' '\n'
```