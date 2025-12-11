# Multi-stage Dockerfile for KNIRVGATEWAY Go version

# Stage 1: Build the Go binary
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -a -installsuffix cgo \
    -ldflags="-s -w" \
    -o /app/bin/knirvgateway \
    ./cmd/gateway

# Stage 2: Create minimal runtime image
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    nodejs \
    npm \
    && rm -rf /var/cache/apk/*

# Create non-root user
RUN addgroup -g 1000 knirvgateway && \
    adduser -D -u 1000 -G knirvgateway knirvgateway

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/bin/knirvgateway /app/knirvgateway

# Copy services directory
COPY --chown=knirvgateway:knirvgateway services ./services

# Install Node.js service dependencies
RUN cd services/payment-gateway && npm install --production && \
    cd ../tunnel-registry && npm install --production && \
    cd ../webgui && npm install --production || true

# Create data and logs directories
RUN mkdir -p /app/data /app/logs && \
    chown -R knirvgateway:knirvgateway /app

# Switch to non-root user
USER knirvgateway

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Set environment variables
ENV PORT=8080 \
    GATEWAY_MODE=persistent \
    KNIRV_CHAIN_ID=testnet

# Run the binary
CMD ["/app/knirvgateway"]
