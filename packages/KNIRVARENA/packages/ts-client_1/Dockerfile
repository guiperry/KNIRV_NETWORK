# Multi-stage build for KNIRV TypeScript Client 1
FROM node:20-alpine AS builder

# Install build dependencies
RUN apk add --no-cache python3 make g++ git curl bash

WORKDIR /app

# Copy package files first for better caching
COPY package*.json ./
COPY package-lock.json ./
COPY tsconfig*.json ./

# Install dependencies
RUN npm ci

# Copy source code
COPY src/ ./src/
COPY client/ ./client/
COPY server/ ./server/
COPY shared/ ./shared/
COPY config/ ./config/
COPY public/ ./public/
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
    chown -R node:node /app/data

# Copy package files and install production dependencies
COPY package*.json ./
COPY package-lock.json ./
RUN npm ci --only=production && \
    npm cache clean --force

# Copy built application from builder
COPY --from=builder /app/dist ./dist
COPY --from=builder /app/public ./public

# Copy server for production
COPY --from=builder /app/server ./server

# Switch to non-root user
USER node

# Expose ports
EXPOSE 3000 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD curl -f http://localhost:3000/health || exit 1

# Environment variables
ENV NODE_ENV=production
ENV PORT=3000

# Volume for persistent data
VOLUME ["/app/data"]

# Start command
CMD ["npm", "start"]
