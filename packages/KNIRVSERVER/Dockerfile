# KNIRV-SERVER Unified Dockerfile
# Multi-stage build for optimized production image

# Stage 1: Frontend Build
FROM node:18-alpine AS frontend-builder

# Install git for build-log-start.mjs script
RUN apk add --no-cache git

WORKDIR /app/frontend

# Copy package files (relative to current build context `./KNIRVSERVER/`)
COPY frontend/package*.json ./
COPY frontend/tsconfig.json ./
COPY frontend/next.config.ts ./
COPY frontend/tailwind.config.ts ./
COPY frontend/postcss.config.mjs ./
COPY frontend/components.json ./

# Install dependencies
RUN npm ci

# Copy source code (relative to current build context `./KNIRVSERVER/`)
COPY frontend/src/ ./src/
COPY frontend/public/ ./public/
COPY frontend/scripts/ ./scripts/

# Build frontend
RUN npm run build

# Stage 2: Backend Build
FROM golang:1.24-alpine AS backend-builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copy Go modules (relative to current build context `./KNIRVSERVER/`)
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# Copy backend source (relative to current build context `./KNIRVSERVER/`)
COPY backend/ ./

# Copy frontend build output to embed location
COPY --from=frontend-builder /app/frontend/out ./pkg/gui/frontend/dist/ 

# Build flags
ARG VERSION=dev
ARG BUILD_TIME
ARG GIT_COMMIT

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

# Build the unified binary
RUN go build \
    -ldflags="-w -s -X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.GitCommit=${GIT_COMMIT}" \
    -o bin/knirv-server \
    ./cmd/backend_server

# Stage 3: Production Image
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    curl \
    bash \
    podman \
    && rm -rf /var/cache/apk/*

# Create non-root user
RUN addgroup -g 1001 -S server && \
    adduser -S server -u 1001 -G server

# Create application directories
RUN mkdir -p /opt/knirv-server/{bin,config,data,logs} && \
    mkdir -p /var/lib/knirv-server && \
    chown -R server:server /opt/knirv-server /var/lib/knirv-server

WORKDIR /opt/knirv-server

# Copy binary from builder
COPY --from=backend-builder /app/bin/knirv-server ./bin/ 

# Copy configuration files (relative to current build context `./KNIRVSERVER/`)
COPY config/ ./config/ 

# Set permissions
RUN chmod +x ./bin/knirv-server && \
    chown -R server:server /opt/knirv-server

# Switch to non-root user
USER server

# Environment variables
ENV NEXUS_ENV=production
ENV NEXUS_CONFIG=/opt/knirv-server/config/production.yaml
ENV NEXUS_DATA_DIR=/var/lib/knirv-server
ENV NEXUS_LOG_LEVEL=info

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/api/v1/health || exit 1

# Expose ports
EXPOSE 8080 9090 4001

# Volume for persistent data
VOLUME ["/var/lib/knirv-server"]

# Default command
CMD ["./bin/knirv-server", "--config", "/opt/knirv-server/config/production.yaml"]

# Labels
LABEL maintainer="KNIRV Team <team@knirv.com>"
LABEL version="${VERSION}"
LABEL description="KNIRV-SERVER Unified Distributed Virtual Environment Platform"
LABEL org.opencontainers.image.title="KNIRV-SERVER"
LABEL org.opencontainers.image.description="Unified distributed virtual environment platform"
LABEL org.opencontainers.image.vendor="KNIRV"
LABEL org.opencontainers.image.licenses="MIT"