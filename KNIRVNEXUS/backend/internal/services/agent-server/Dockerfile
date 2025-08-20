# Multi-stage build for AGENTCHAIN Plugin Agent Server

# Build stage
FROM golang:1.21-alpine AS builder

# Set working directory
WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o plugin-server .

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates curl

# Create non-root user
RUN addgroup -g 1001 -S agentchain && \
    adduser -u 1001 -S agentchain -G agentchain

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/plugin-server .

# Create agents directory
RUN mkdir -p /app/agents && \
    chown -R agentchain:agentchain /app

# Switch to non-root user
USER agentchain

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/info || exit 1

# Default command
CMD ["./plugin-server", "--port", "8080", "--agents", "/app/agents"]
