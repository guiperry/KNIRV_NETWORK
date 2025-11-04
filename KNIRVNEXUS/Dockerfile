# KNIRV-NEXUS Unified Dockerfile
# Multi-stage build for optimized production image

# Stage 1: Frontend Build
FROM node:18-alpine AS frontend-builder

WORKDIR /app

# Copy package files
COPY package*.json ./
COPY tsconfig.json ./
COPY next.config.ts ./
COPY tailwind.config.ts ./
COPY postcss.config.mjs ./
COPY components.json ./

# Install dependencies
RUN npm ci --only=production

# Copy source code
COPY src/ ./src/
COPY public/ ./public/

# Build frontend
RUN npm run build

# Stage 2: Backend Build
FROM golang:1.21-alpine AS backend-builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copy Go modules
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# Copy backend source
COPY backend/ ./

# Copy frontend build output to embed location
COPY --from=frontend-builder /app/out ./pkg/gui/frontend/dist/

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
    -o bin/knirv-nexus \
    ./knirv-nexus

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
RUN addgroup -g 1001 -S nexus && \
    adduser -S nexus -u 1001 -G nexus

# Create application directories
RUN mkdir -p /opt/knirv-nexus/{bin,config,data,logs} && \
    mkdir -p /var/lib/knirv-nexus && \
    chown -R nexus:nexus /opt/knirv-nexus /var/lib/knirv-nexus

WORKDIR /opt/knirv-nexus

# Copy binary from builder
COPY --from=backend-builder /app/bin/knirv-nexus ./bin/

# Copy configuration files
COPY backend/config/ ./config/

# Set permissions
RUN chmod +x ./bin/knirv-nexus && \
    chown -R nexus:nexus /opt/knirv-nexus

# Switch to non-root user
USER nexus

# Environment variables
ENV NEXUS_ENV=production
ENV NEXUS_CONFIG=/opt/knirv-nexus/config/production.yaml
ENV NEXUS_DATA_DIR=/var/lib/knirv-nexus
ENV NEXUS_LOG_LEVEL=info

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/api/v1/health || exit 1

# Expose ports
EXPOSE 8080 9090 4001

# Volume for persistent data
VOLUME ["/var/lib/knirv-nexus"]

# Default command
CMD ["./bin/knirv-nexus", "--config", "/opt/knirv-nexus/config/production.yaml"]

# Labels
LABEL maintainer="KNIRV Team <team@knirv.com>"
LABEL version="${VERSION}"
LABEL description="KNIRV-NEXUS Unified Distributed Virtual Environment Platform"
LABEL org.opencontainers.image.title="KNIRV-NEXUS"
LABEL org.opencontainers.image.description="Unified distributed virtual environment platform"
LABEL org.opencontainers.image.vendor="KNIRV"
LABEL org.opencontainers.image.licenses="MIT"
