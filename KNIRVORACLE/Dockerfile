# KNIRVORACLE Production Dockerfile
FROM rust:1.75-alpine AS builder

# Install build dependencies
RUN apk add --no-cache musl-dev pkgconfig openssl-dev

# Set working directory
WORKDIR /app

# Copy Cargo files
COPY Cargo.toml Cargo.lock ./

# Create src directory and dummy main.rs for dependency caching
RUN mkdir src && echo "fn main() {}" > src/main.rs

# Build dependencies (this layer will be cached)
RUN cargo build --release && rm -rf src

# Copy source code
COPY src ./src
COPY config ./config

# Build the application
RUN cargo build --release

# Production stage
FROM alpine:3.18

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata libgcc curl

# Create non-root user
RUN addgroup -g 1001 -S knirvchain && \
    adduser -u 1001 -S knirvchain -G knirvchain

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/target/release/knirvchain .

# Copy configuration
COPY --from=builder /app/config ./config

# Create data directory
RUN mkdir -p /app/data && \
    chown -R knirvchain:knirvchain /app

# Switch to non-root user
USER knirvchain

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Set environment variables
ENV KNIRV_ENV=production
ENV RUST_LOG=info

# Run the application
CMD ["./knirvchain"]
