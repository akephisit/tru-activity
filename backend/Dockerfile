# Development Dockerfile for Go Backend
FROM golang:1.24.5-alpine AS development

# Install git and other dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Install air for hot reloading
RUN go install github.com/cosmtrek/air@latest

# Copy source code
COPY . .

# Expose port
EXPOSE 8080

# Run with hot reload
CMD ["air", "-c", ".air.toml"]

# Production build stage
FROM golang:1.24.5-alpine AS builder

# Install dependencies for building
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the application with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -a -installsuffix cgo \
    -ldflags='-w -s -extldflags "-static"' \
    -o main ./cmd/server/main.go

# Verify the binary was created
RUN ls -la main && file main

# Production stage
FROM alpine:latest AS production

# Install ca-certificates for HTTPS requests and curl for health checks
RUN apk --no-cache add ca-certificates curl tzdata && \
    update-ca-certificates

# Create non-root user
RUN addgroup -g 1001 -S app && \
    adduser -S -D -H -u 1001 -h /app -s /sbin/nologin -G app -g app app

WORKDIR /app

# Copy timezone data from builder
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy binary from builder stage
COPY --from=builder /app/main .

# Copy migrations if they exist
COPY --from=builder /app/migrations/ ./migrations/ 2>/dev/null || true

# Make binary executable and change ownership
RUN chmod +x ./main && \
    chown -R app:app /app

# Switch to non-root user
USER app

EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Run the binary
CMD ["./main"]