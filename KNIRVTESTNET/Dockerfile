# KNIRV Wizened Environment Dockerfile
# Implements the WizenedEnvironmentGuide approach for Render deployment

FROM ubuntu:22.04

# Install system dependencies
RUN apt-get update && apt-get install -y \
    curl \
    wget \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Set working directory
WORKDIR /app

# Copy pre-built artifacts (built locally with build-local-release.sh)
COPY bin/ ./bin/
COPY config/ ./config/
COPY assets/ ./assets/
COPY agent-developer-portal/ ./agent-developer-portal/
COPY graphchain-explorer/ ./graphchain-explorer/
COPY index.html ./
COPY package*.json ./

# Install Node.js (for npm operations on host)
ENV NODE_VERSION=v18.18.0
RUN curl -L "https://nodejs.org/dist/${NODE_VERSION}/node-${NODE_VERSION}-linux-x64.tar.xz" | tar -xJ \
    && mv node-${NODE_VERSION}-linux-x64 /usr/local/node \
    && ln -s /usr/local/node/bin/node /usr/local/bin/node \
    && ln -s /usr/local/node/bin/npm /usr/local/bin/npm

# Install npm dependencies
RUN npm install

# Download Wasmtime runtime
ENV WASMTIME_VERSION=v22.0.0
RUN curl -L "https://github.com/bytecodealliance/wasmtime/releases/download/${WASMTIME_VERSION}/wasmtime-${WASMTIME_VERSION}-x86_64-linux.tar.xz" | tar -xJ --strip-components=1 -C ./bin

# Make all binaries executable
RUN chmod +x ./bin/*

# Create necessary directories
RUN mkdir -p logs data

# Expose the port
EXPOSE 10000

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:10000/health || exit 1

# Start the native KNIRV orchestrator
# VFS toolchain available separately for development tools
CMD ["./bin/knirv-orchestrator"]
