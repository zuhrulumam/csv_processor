#!/bin/bash
set -e

# Script to build Docker images

VERSION=${VERSION:-"1.0.0"}
IMAGE_NAME="csv-processor"

echo "Building Docker images..."

# Build production image
echo "Building production image: ${IMAGE_NAME}:${VERSION}"
docker build -t ${IMAGE_NAME}:${VERSION} -t ${IMAGE_NAME}:latest .

# Build development image
echo "Building development image: ${IMAGE_NAME}:dev"
docker build -f Dockerfile.dev -t ${IMAGE_NAME}:dev .

echo "Docker images built successfully!"
echo ""
echo "Available images:"
docker images | grep ${IMAGE_NAME}