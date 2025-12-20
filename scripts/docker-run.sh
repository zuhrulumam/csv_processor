#!/bin/bash
set -e

# Script to run CSV processor in Docker

# Default values
INPUT_DIR="${INPUT_DIR:-./examples/input}"
OUTPUT_DIR="${OUTPUT_DIR:-./examples/output}"
WORKERS="${WORKERS:-4}"
INPUT_FILE="${1:-sample.csv}"

# Create output directory if it doesn't exist
mkdir -p "$OUTPUT_DIR"

echo "Processing CSV file with Docker..."
echo "Input: $INPUT_DIR/$INPUT_FILE"
echo "Output: $OUTPUT_DIR/results.csv"
echo "Workers: $WORKERS"
echo ""

docker run --rm \
    -v "$(pwd)/$INPUT_DIR:/data/input:ro" \
    -v "$(pwd)/$OUTPUT_DIR:/data/output" \
    csv-processor:latest \
    -workers "$WORKERS" \
    -progress \
    -output /data/output/results.csv \
    /data/input/"$INPUT_FILE"

echo ""
echo "Processing complete! Results saved to: $OUTPUT_DIR/results.csv"