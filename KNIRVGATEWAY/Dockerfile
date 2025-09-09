# KNIRVGATEWAY Production Dockerfile
# Multi-stage build for KNIRV Gateway with Netlify Functions support
FROM node:20-alpine AS builder

# Install build dependencies
RUN apk add --no-cache python3 make g++ git curl

# Set working directory
WORKDIR /app

# Copy package files
COPY package*.json ./

# Install dependencies
RUN npm ci --legacy-peer-deps

# Copy source code
COPY . .

# Load endpoints and build
RUN npm run load-endpoints && \
    npm run check-function-deps && \
    echo 'KNIRV Gateway built with Netlify Functions support'

# Production stage
FROM node:20-alpine AS production

# Install runtime dependencies
RUN apk add --no-cache curl nginx

# Create non-root user
RUN addgroup -g 1001 -S knirvgateway && \
    adduser -u 1001 -S knirvgateway -G knirvgateway

# Set working directory
WORKDIR /app

# Copy package files and install production dependencies
COPY package*.json ./
RUN npm ci --only=production --legacy-peer-deps && \
    npm cache clean --force

# Copy built application from builder stage
COPY --from=builder /app .

# Create necessary directories
RUN mkdir -p /app/logs && \
    mkdir -p /app/data && \
    chown -R knirvgateway:knirvgateway /app

# Configure nginx for static file serving
COPY --chown=knirvgateway:knirvgateway <<EOF /etc/nginx/nginx.conf
events {
    worker_connections 1024;
}

http {
    include /etc/nginx/mime.types;
    default_type application/octet-stream;
    
    server {
        listen 80;
        server_name localhost;
        root /app;
        index index.html;
        
        # Serve static files
        location / {
            try_files \$uri \$uri/ /index.html;
        }
        
        # Proxy API requests to Node.js
        location /api/ {
            proxy_pass http://localhost:3000;
            proxy_http_version 1.1;
            proxy_set_header Upgrade \$http_upgrade;
            proxy_set_header Connection 'upgrade';
            proxy_set_header Host \$host;
            proxy_cache_bypass \$http_upgrade;
        }
        
        # Health check endpoint
        location /health {
            access_log off;
            return 200 "healthy\n";
            add_header Content-Type text/plain;
        }
    }
}
EOF

# Switch to non-root user
USER knirvgateway

# Expose ports
EXPOSE 80 3000

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost/health || exit 1

# Set environment variables
ENV NODE_ENV=production
ENV PORT=3000

# Volume for persistent data
VOLUME ["/app/data", "/app/logs"]

# Start script
COPY --chown=knirvgateway:knirvgateway <<EOF /app/start.sh
#!/bin/sh
# Start nginx in background
nginx &

# Start Node.js application
npm start
EOF

RUN chmod +x /app/start.sh

# Start command
CMD ["/app/start.sh"]
