# Multi-stage build for KNIRV Elixir Backend
FROM elixir:1.15-alpine AS builder

# Install build dependencies
RUN apk add --no-cache gcc musl-dev pkgconfig

WORKDIR /app

# Copy mix files first for better caching
COPY mix.exs mix.lock ./
COPY config ./config

# Install dependencies
RUN mix local.hex --force && \
    mix local.rebar --force && \
    mix deps.get

# Copy source code
COPY lib ./lib
COPY priv ./priv
COPY test ./test

# Compile the application
RUN mix compile

# Build release
RUN mix release

# Production stage
FROM alpine:latest AS production

# Install runtime dependencies
RUN apk add --no-cache ca-certificates curl libstdc++

# Create non-root user
RUN addgroup -g 1001 -S knirvana && \
    adduser -u 1001 -S knirvana -G knirvana

WORKDIR /app

# Copy release from builder
COPY --from=builder /app/_build/prod/rel/knirvarena ./knirvarena
COPY --from=builder /app/_build/prod/rel/knirvarena/bin/start ./start

# Create data directories
RUN mkdir -p /app/data /app/logs && \
    chown -R knirvana:knirvana /app

# Switch to non-root user
USER knirvana

# Expose ports
EXPOSE 4000

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD curl -f http://localhost:4000/health || exit 1

# Environment variables
ENV MIX_ENV=prod
ENV PHX_HOST=localhost

# Volume for persistent data
VOLUME ["/app/data"]

# Default command
CMD ["./start"]
