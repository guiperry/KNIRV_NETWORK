# Multi-stage build for KNIRV Rust Client
FROM rust:1.75-alpine AS builder

# Install build dependencies
RUN apk add --no-cache musl-dev pkgconfig openssl-dev

WORKDIR /app

# Copy Cargo files first for better caching
COPY Cargo.toml Cargo.lock ./
COPY src ./src
COPY examples ./examples
COPY benches ./benches
COPY build.rs ./

# Build release binary
RUN cargo build --release

# Production stage
FROM alpine:latest AS production

# Install runtime dependencies
RUN apk add --no-cache ca-certificates curl

# Create non-root user
RUN addgroup -g 1001 -S knirvana && \
    adduser -u 1001 -S knirvana -G knirvana

WORKDIR /app

# Copy built binary
COPY --from=builder /app/target/release/knirvana_game ./knirvana-game

# Create data directories
RUN mkdir -p /app/data /app/logs && \
    chown -R knirvana:knirvana /app

# Switch to non-root user
USER knirvana

# Expose ports
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Environment variables
ENV RUST_LOG=info

# Volume for persistent data
VOLUME ["/app/data"]

# Default command
CMD ["./knirvana-game"]
