# KNIRVENGINE Desktop Client Production Dockerfile
# Native Go binary with webview support
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git musl-dev pkgconfig webkit2gtk-dev gtk+3.0-dev

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum
COPY desktop-client/go.mod desktop-client/go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY desktop-client .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o knirv-engine ./main.go

# Production stage
FROM alpine:3.18

# Install runtime dependencies for webview
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    webkit2gtk \
    gtk+3.0 \
    curl \
    libgcc \
    util-linux \
    libcap

# Keep the container process non-root while allowing namespace entry and eBPF
# tracing. The container runtime must also permit these capabilities.
RUN setcap cap_sys_admin,cap_sys_ptrace,cap_sys_chroot+ep /usr/bin/nsenter && \
    setcap cap_bpf,cap_perfmon,cap_sys_admin+ep /usr/bin/bpftrace

# Create non-root user
RUN addgroup -g 1001 -S knirvengine && \
    adduser -u 1001 -S knirvengine -G knirvengine

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/knirv-engine .

# Copy static files and configuration
COPY --from=builder /app/static ./static
COPY --from=builder /app/gui ./gui
COPY --from=builder /app/public ./public

# Create data directories
RUN mkdir -p /app/data /app/logs && \
    chown -R knirvengine:knirvengine /app

# Switch to non-root user
USER knirvengine

# Expose ports
EXPOSE 8080 8081

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Set environment variables
ENV KNIRV_ENV=production
ENV KNIRV_LOG_LEVEL=info
ENV DISPLAY=:0

# Volume for persistent data
VOLUME ["/app/data"]

# Run the application
CMD ["./knirv-engine"]
