# Base stage: common for Multi-stage Builds
FROM golang:1.25-alpine AS base

# Set working directory
WORKDIR /app

# Install dependencies for go mod download
RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY cmd/ ./cmd/
COPY internal/ ./internal/

# dev stage: hot reload with air
FROM base AS dev

# Install build dependencies for dev
RUN apk add --no-cache tzdata gcc musl-dev

# Install air for hot reload
RUN go install github.com/air-verse/air@v1.61.7

# Copy configuration files
COPY config.dev.json* ./
COPY .air.toml* ./

# Create necessary directories
RUN mkdir -p tmp bin

# Expose port
EXPOSE 8080

# Run Air
CMD ["air"]

# builder stage: build binary
FROM base AS builder

# Install build dependencies
RUN apk add --no-cache tzdata gcc musl-dev

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/boilerplate/main.go

# prod stage: build binary for production
FROM alpine:latest AS prod

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/main .

# Change ownership to non-root user
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Run Main
CMD ["./main"]
