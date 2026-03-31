# Multi-stage build for KNIRV Controller with NebulaDB
FROM node:20-alpine AS builder

# Install build dependencies
RUN apk add --no-cache python3 make g++ git curl bash

WORKDIR /app

# Copy package files first for better caching
COPY package*.json ./
COPY package-lock.json ./
COPY tsconfig*.json ./

# Install dependencies including dev dependencies for build
RUN npm ci --legacy-peer-deps

# Copy source code selectively
COPY src/ ./src/
COPY public/ ./public/
COPY config/ ./config/
COPY scripts/ ./scripts/
COPY assembly/ ./assembly/
COPY *.config.* ./
COPY index.html ./

# Make build script executable
RUN chmod +x scripts/build-simple.sh

# Build the application with robust error handling
RUN echo "Starting build process..." && \
    (npm run build:simple || \
     (echo "Simple build failed, attempting basic vite build..." && \
      npx vite build || \
      (echo "All build attempts failed" && exit 1)))

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
COPY package-lock.json ./
RUN npm ci --only=production --legacy-peer-deps && \
    npm cache clean --force

# Copy built application and scripts from builder stage
COPY --from=builder /app/dist ./dist
COPY --from=builder /app/config ./config
COPY --from=builder /app/scripts ./scripts

# Create non-root user
USER node

# Expose ports
EXPOSE 3000 3001

# Health check using the health endpoint
HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
    CMD curl -f http://localhost:3000/health || exit 1

# Environment variables
ENV NODE_ENV=production
ENV DATABASE_PATH=/app/data/knirvcontroller.db

# Volume for persistent data
VOLUME ["/app/data"]

# Start command using npm start (now points to static server)
CMD ["npm", "start"]
