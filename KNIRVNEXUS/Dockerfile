# KNIRVNEXUS Production Dockerfile
# Multi-stage build for Go backend and React frontend

# Frontend build stage
FROM node:18-alpine AS frontend-builder

WORKDIR /app/frontend

# Copy package files
COPY gui/package*.json ./

# Install dependencies
RUN npm ci --only=production

# Copy frontend source
COPY gui/src ./src
COPY gui/public ./public
COPY gui/index.html ./
COPY gui/vite.config.ts ./
COPY gui/tsconfig*.json ./
COPY gui/tailwind.config.js ./
COPY gui/postcss.config.cjs ./

# Build frontend
RUN npm run build

# Backend build stage
FROM golang:1.21-alpine AS backend-builder

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o knirvnexus .

# Production stage
FROM alpine:3.18

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata curl python3 py3-pip

# Create non-root user
RUN addgroup -g 1001 -S knirvnexus && \
    adduser -u 1001 -S knirvnexus -G knirvnexus

# Set working directory
WORKDIR /app

# Copy backend binary
COPY --from=backend-builder /app/knirvnexus .

# Copy frontend build
COPY --from=frontend-builder /app/frontend/dist ./static

# Copy static assets and configurations
COPY --from=backend-builder /app/static ./static
COPY --from=backend-builder /app/utils/default.env ./utils/

# Create necessary directories
RUN mkdir -p /app/data /app/logs /app/plugins && \
    chown -R knirvnexus:knirvnexus /app

# Switch to non-root user
USER knirvnexus

# Expose port
EXPOSE 8082

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8082/health || exit 1

# Set environment variables
ENV KNIRV_ENV=production
ENV KNIRV_LOG_LEVEL=info
ENV KNIRV_DATA_DIR=/app/data

# Run the application
CMD ["./knirvnexus"]
