# KNIRVORACLE Production Dockerfile
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

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o knirvoracle .

# Production stage
FROM alpine:3.18

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata curl

# Create non-root user
RUN addgroup -g 1001 -S knirvoracle && \
    adduser -u 1001 -S knirvoracle -G knirvoracle

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/knirvoracle .

# Create data directory for LevelDB
RUN mkdir -p /app/data && \
    chown -R knirvoracle:knirvoracle /app

# Switch to non-root user
USER knirvoracle

# Expose port
EXPOSE 8083

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8083/health || exit 1

# Set environment variables
ENV KNIRV_ENV=production
ENV KNIRV_LOG_LEVEL=info
ENV KNIRV_DATA_DIR=/app/data

# Run the application
CMD ["./knirvoracle"]
