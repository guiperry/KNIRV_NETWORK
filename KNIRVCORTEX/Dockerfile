# KNIRVCORTEX Multi-Component Production Dockerfile
# Includes data-engine (Go), inference-engine (Go), manager (Node.js), and Rust WASM components
FROM golang:1.21-alpine AS go-builder

# Install build dependencies
RUN apk add --no-cache git musl-dev

# Set working directory
WORKDIR /app

# Copy Go modules (data-engine and inference-engine)
COPY data-engine ./data-engine
COPY inference-engine ./inference-engine

# Build data-engine
WORKDIR /app/data-engine
RUN go mod init knirvcortex/data-engine && \
    go mod tidy && \
    go build -o ../bin/data-engine .

# Build inference-engine
WORKDIR /app/inference-engine
RUN go mod init knirvcortex/inference-engine && \
    go mod tidy && \
    go build -o ../bin/inference-engine .

# Rust builder for WASM components
FROM rust:1.75-alpine AS rust-builder

# Install build dependencies
RUN apk add --no-cache musl-dev wasm-pack

# Set working directory
WORKDIR /app/rust-wasm

# Copy Rust WASM source
COPY rust-wasm/Cargo.toml rust-wasm/Cargo.lock ./
COPY rust-wasm/src ./src

# Build WASM module
RUN wasm-pack build --target web --out-dir ../wasm-dist

# Node.js builder for manager
FROM node:20-alpine AS node-builder

# Install build dependencies
RUN apk add --no-cache python3 make g++

# Set working directory
WORKDIR /app/manager

# Copy manager package files
COPY manager/package*.json ./
COPY manager/tsconfig.json ./
COPY manager/vite.config.ts ./
COPY manager/tailwind.config.js ./
COPY manager/postcss.config.js ./

# Install dependencies
RUN npm ci

# Copy manager source
COPY manager/src ./src
COPY manager/public ./public
COPY manager/index.html ./

# Build manager
RUN npm run build

# Production stage
FROM node:20-alpine AS production

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata curl

# Create non-root user
RUN addgroup -g 1001 -S knirvcortex && \
    adduser -u 1001 -S knirvcortex -G knirvcortex

# Set working directory
WORKDIR /app

# Copy built Go binaries
COPY --from=go-builder /app/bin ./bin

# Copy built WASM modules
COPY --from=rust-builder /app/wasm-dist ./wasm

# Copy built Node.js manager
COPY --from=node-builder /app/manager/dist ./manager
COPY --from=node-builder /app/manager/package*.json ./

# Install production dependencies
RUN npm ci --only=production && \
    npm cache clean --force

# Copy additional files
COPY agent-core ./agent-core

# Create data directories
RUN mkdir -p /app/data /app/logs && \
    chown -R knirvcortex:knirvcortex /app

# Switch to non-root user
USER knirvcortex

# Expose ports
EXPOSE 8080 8081 8082

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Set environment variables
ENV NODE_ENV=production
ENV CORTEX_MODE=production

# Volume for persistent data
VOLUME ["/app/data"]

# Start script
COPY --chown=knirvcortex:knirvcortex <<EOF /app/start.sh
#!/bin/sh
# Start data-engine
./bin/data-engine &

# Start inference-engine
./bin/inference-engine &

# Start manager (Node.js frontend)
npm start
EOF

RUN chmod +x /app/start.sh

# Start command
CMD ["/app/start.sh"]
