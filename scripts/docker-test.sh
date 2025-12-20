#!/bin/bash
set -e

# Script to run tests in Docker

echo "Running tests in Docker..."

# Run tests
docker run --rm \
    -v "$(pwd):/build" \
    -w /build \
    golang:1.21-alpine \
    sh -c "go mod download && go test -v -race ./..."

echo ""
echo "All tests passed!"