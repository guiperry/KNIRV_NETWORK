# KNIRVANA Multi-Client Production Dockerfile
# Supports both Rust game client and TypeScript web client
FROM rust:1.75-alpine AS rust-builder

# Install build dependencies for Rust
RUN apk add --no-cache musl-dev pkgconfig openssl-dev

# Set working directory for Rust client
WORKDIR /app/rust-client

# Copy Rust client files
COPY rust-client/Cargo.toml rust-client/Cargo.lock ./
COPY rust-client/src ./src
COPY rust-client/examples ./examples
COPY rust-client/benches ./benches
COPY rust-client/build.rs ./

# Build Rust client
RUN cargo build --release --features desktop

# Node.js builder stage for TypeScript client
FROM node:20-alpine AS ts-builder

# Install build dependencies
RUN apk add --no-cache python3 make g++ git

# Set working directory for TypeScript client
WORKDIR /app/ts-client

# Copy TypeScript client package files
COPY ts-client/package*.json ./
COPY ts-client/tsconfig.json ./
COPY ts-client/vite.config.ts ./
COPY ts-client/tailwind.config.ts ./
COPY ts-client/postcss.config.js ./
COPY ts-client/drizzle.config.ts ./

# Install dependencies
RUN npm ci

# Copy TypeScript client source
COPY ts-client/src ./src
COPY ts-client/server ./server
COPY ts-client/shared ./shared
COPY ts-client/client ./client

# Build TypeScript client
RUN npm run build

# Production stage
FROM node:20-alpine AS production

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata curl

# Create non-root user
RUN addgroup -g 1001 -S knirvana && \
    adduser -u 1001 -S knirvana -G knirvana

# Set working directory
WORKDIR /app

# Copy built Rust binary
COPY --from=rust-builder /app/rust-client/target/release/knirvana_game ./bin/knirvana-game

# Copy built TypeScript application
COPY --from=ts-builder /app/ts-client/dist ./web
COPY --from=ts-builder /app/ts-client/package*.json ./

# Install production dependencies for TypeScript client
RUN npm ci --only=production && \
    npm cache clean --force

# Create data directories
RUN mkdir -p /app/data /app/logs && \
    chown -R knirvana:knirvana /app

# Switch to non-root user
USER knirvana

# Expose ports
EXPOSE 3000 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:3000/health || exit 1

# Set environment variables
ENV NODE_ENV=production
ENV RUST_LOG=info
ENV KNIRVANA_MODE=web

# Volume for persistent data
VOLUME ["/app/data"]

# Default command (starts TypeScript web client)
CMD ["npm", "start"]
