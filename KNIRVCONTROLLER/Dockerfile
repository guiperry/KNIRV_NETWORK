# Multi-stage build for KNIRV Controller with NebulaDB
FROM node:20-alpine AS builder

# Install build dependencies
RUN apk add --no-cache python3 make g++ git

WORKDIR /app

# Copy package files
COPY package*.json ./
COPY tsconfig*.json ./

# Install dependencies
RUN npm ci --legacy-peer-deps

# Copy source code
COPY src/ ./src/
COPY public/ ./public/
COPY config/ ./config/
COPY scripts/ ./scripts/
COPY assembly/ ./assembly/
COPY build/ ./build/
COPY *.config.* ./
COPY index.html ./

# Build the application
RUN npm run build

# Production stage
FROM node:20-alpine AS production

# Install runtime dependencies
RUN apk add --no-cache curl

WORKDIR /app

# Create data directory with proper permissions
RUN mkdir -p /app/data && \
    mkdir -p /app/data/backups && \
    chown -R node:node /app/data

# Copy package files and install production dependencies
COPY package*.json ./
RUN npm ci --only=production --legacy-peer-deps && \
    npm cache clean --force

# Copy built application from builder stage
COPY --from=builder /app/dist ./dist
COPY --from=builder /app/build ./build
COPY --from=builder /app/config ./config
COPY --from=builder /app/scripts ./scripts

# Copy migration script
COPY scripts/migrate-to-nebuladb.ts ./scripts/

# Create non-root user
USER node

# Expose ports
EXPOSE 3000 3001

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
    CMD curl -f http://localhost:3000/health || exit 1

# Environment variables
ENV NODE_ENV=production
ENV DATABASE_PATH=/app/data/knirvcontroller.db

# Volume for persistent data
VOLUME ["/app/data"]

# Start command
CMD ["npm", "start"]
