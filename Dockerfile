# Multi-stage build for optimal image size
# Stage 1: Build
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s" -o processor ./cmd/processor

# Stage 2: Runtime
FROM alpine:3.19

# Install CA certificates for HTTPS
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/processor /app/processor

# Create directories for data
RUN mkdir -p /data/input /data/output && \
    chown -R appuser:appuser /app /data

# Switch to non-root user
USER appuser

# Set environment variables
ENV PATH="/app:${PATH}"

# Default command
ENTRYPOINT ["/app/processor"]
CMD ["--help"]

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/app/processor", "--version"] || exit 1

# Labels
LABEL org.opencontainers.image.title="CSV Processor" \
      org.opencontainers.image.description="Concurrent CSV file processor" \
      org.opencontainers.image.version="1.0.0" \
      org.opencontainers.image.authors="Your Name <your.email@example.com>"