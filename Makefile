# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

BINARY_NAME=processor
BINARY_UNIX=$(BINARY_NAME)_unix

# Build information
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.gitCommit=$(GIT_COMMIT)"

all: test build

build:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) -v ./cmd/processor

build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_UNIX) -v ./cmd/processor

test:
	$(GOTEST) -v -race -coverprofile=coverage.out ./...

test-coverage: test
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

bench:
	$(GOTEST) -bench=. -benchmem ./...

clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)
	rm -f coverage.out coverage.html

run:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) -v ./cmd/processor
	./$(BINARY_NAME)

deps:
	$(GOMOD) download
	$(GOMOD) tidy

lint:
	golangci-lint run

# Docker targets
docker-build:
	docker build -t csv-processor:latest .

docker-build-dev:
	docker build -f Dockerfile.dev -t csv-processor:dev .

docker-run:
	./scripts/docker-run.sh

docker-test:
	docker run --rm -v $(PWD):/build -w /build golang:1.21-alpine sh -c "go mod download && go test -v -race ./..."

docker-clean:
	docker rmi csv-processor:latest csv-processor:dev || true

docker-compose-up:
	docker-compose up processor

docker-compose-test:
	docker-compose up test

docker-compose-down:
	docker-compose down

# Make scripts executable
setup-scripts:
	chmod +x scripts/*.sh

.PHONY: docker-build docker-build-dev docker-run docker-test docker-clean \
        docker-compose-up docker-compose-test docker-compose-down setup-scripts