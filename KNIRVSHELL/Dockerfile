FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build arguments
ARG VERSION=dev
ARG BUILD_TIME=unknown
ARG GIT_COMMIT=unknown

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.GitCommit=${GIT_COMMIT}\" \
    -o knirvchain-cli .

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1001 -S knirvchain && \
    adduser -u 1001 -S knirvchain -G knirvchain

# Set working directory
WORKDIR /home/knirvchain

# Copy binary from builder
COPY --from=builder /app/knirvchain-cli /usr/local/bin/knirv

# Copy configuration files
COPY --from=builder /app/README.md /app/LICENSE /app/CHANGELOG.md ./

# Create directories
RUN mkdir -p .knirv/wallets .knirv/logs && \
    chown -R knirvchain:knirvchain .knirv

# Switch to non-root user
USER knirvchain

# Set environment variables
ENV KNIRV_WALLET_DIRECTORY=/home/knirvchain/.knirv/wallets
ENV KNIRV_LOG_LEVEL=info

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD knirv version || exit 1

# Default command
ENTRYPOINT ["knirv"]
CMD ["--help"]

# Labels
LABEL org.opencontainers.image.title="KNIRVCHAIN CLI"
LABEL org.opencontainers.image.description="Command-line interface for KNIRVCHAIN Multi-Capability Protocol"
LABEL org.opencontainers.image.vendor="Cloud Equities, Inc."
LABEL org.opencontainers.image.licenses="MIT"
LABEL org.opencontainers.image.source="https://github.com/guiperry/KNIRVCHAIN-CLI"
LABEL org.opencontainers.image.documentation="https://github.com/guiperry/KNIRVCHAIN-CLI/blob/main/docs/user_guide.md"