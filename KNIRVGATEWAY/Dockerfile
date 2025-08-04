# KNIRVGATEWAY Production Dockerfile
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the gateway application
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o gateway ./api-gateway

# Build the economics service
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o economics ./economics/cmd

# Production stage
FROM alpine:3.18

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata curl

# Create non-root user
RUN addgroup -g 1001 -S knirvgateway && \
    adduser -u 1001 -S knirvgateway -G knirvgateway

# Set working directory
WORKDIR /app

# Copy binaries from builder
COPY --from=builder /app/gateway .
COPY --from=builder /app/economics .

# Copy configuration files
COPY --from=builder /app/api-gateway/config.yaml ./config/

# Create data directory
RUN mkdir -p /app/data /app/logs && \
    chown -R knirvgateway:knirvgateway /app

# Switch to non-root user
USER knirvgateway

# Expose port
EXPOSE 8000

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8000/health || exit 1

# Set environment variables
ENV KNIRV_ENV=production
ENV KNIRV_LOG_LEVEL=info

# Run the gateway application
CMD ["./gateway"]
