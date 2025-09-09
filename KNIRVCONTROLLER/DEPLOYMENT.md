# KNIRVCONTROLLER Web Deployment Guide

## Overview

KNIRVCONTROLLER can be deployed as a web-hosted application with full API access via API keys. This guide covers deployment options and configuration.

## Deployment Options

### 1. Docker Deployment (Recommended)

#### Quick Start
```bash
# Clone and build
git clone <repository>
cd KNIRVCONTROLLER

# Build and run with Docker Compose
docker-compose up -d

# Access the application
# Frontend: http://localhost:3000
# API: http://localhost:3001
```

#### Production Deployment
```bash
# Build for production
docker build -t knirvcontroller:latest .

# Run with environment variables
docker run -d \
  --name knirvcontroller \
  -p 3000:3000 \
  -p 3001:3001 \
  -e NODE_ENV=production \
  -e API_BASE_URL=https://your-domain.com/api \
  -v knirvcontroller_data:/app/data \
  knirvcontroller:latest
```

### 2. Cloud Platform Deployment

#### Vercel
```bash
# Install Vercel CLI
npm i -g vercel

# Deploy
vercel --prod
```

#### Netlify
```bash
# Build command: npm run build
# Publish directory: dist
# Functions directory: netlify/functions (if using serverless functions)
```

#### Railway
```bash
# Connect GitHub repository
# Set build command: npm run build
# Set start command: npm run start:prod
```

### 3. Traditional Server Deployment

#### Prerequisites
- Node.js 18+
- PM2 (for process management)

#### Steps
```bash
# Install dependencies
npm install --production

# Build application
npm run build

# Install PM2
npm install -g pm2

# Start with PM2
pm2 start ecosystem.config.js

# Save PM2 configuration
pm2 save
pm2 startup
```

## Configuration

### Environment Variables

Create a `.env` file in the root directory:

```env
# Application
NODE_ENV=production
PORT=3000
API_PORT=3001

# Database
DATABASE_PATH=/app/data/knirvcontroller.db

# API Configuration
API_BASE_URL=https://your-domain.com/api
CORS_ORIGIN=https://your-domain.com

# Security
JWT_SECRET=your-jwt-secret-here
API_KEY_ENCRYPTION_KEY=your-encryption-key-here

# Rate Limiting
DEFAULT_RATE_LIMIT_PER_MINUTE=60
DEFAULT_RATE_LIMIT_PER_HOUR=1000
DEFAULT_RATE_LIMIT_PER_DAY=10000

# Logging
LOG_LEVEL=info
LOG_FILE=/app/logs/knirvcontroller.log
```

### PM2 Configuration

Create `ecosystem.config.js`:

```javascript
module.exports = {
  apps: [{
    name: 'knirvcontroller',
    script: 'dist/server/api-server.js',
    instances: 'max',
    exec_mode: 'cluster',
    env: {
      NODE_ENV: 'production',
      PORT: 3000,
      API_PORT: 3001
    },
    error_file: './logs/err.log',
    out_file: './logs/out.log',
    log_file: './logs/combined.log',
    time: true
  }]
};
```

## API Access

### Creating API Keys

1. Access the web interface at your deployed URL
2. Open the burger menu (☰)
3. Click "API Keys"
4. Click "Create New API Key"
5. Configure permissions and rate limits
6. Copy the generated key (you won't see it again)

### Using API Keys

Include the API key in requests:

```bash
# Using X-API-Key header
curl -H "X-API-Key: knirv_your_api_key_here" \
     https://your-domain.com/api/status

# Using Authorization header
curl -H "Authorization: Bearer knirv_your_api_key_here" \
     https://your-domain.com/api/agents/deploy
```

### Available Permissions

- `read:agents` - Read agent information
- `write:agents` - Deploy and manage agents
- `read:graph` - Read KNIRVGRAPH data
- `write:graph` - Modify KNIRVGRAPH data
- `read:cortex` - Read CORTEX models
- `write:cortex` - Train and manage CORTEX models
- `read:wallet` - Read wallet information
- `write:wallet` - Perform wallet operations
- `read:skills` - Read skill information
- `write:skills` - Manage skills
- `read:analytics` - Read analytics data
- `admin:all` - Full administrative access

## Progressive Web App (PWA)

KNIRVCONTROLLER includes PWA support for mobile access:

### Features
- Offline functionality
- App-like experience on mobile
- Push notifications (when configured)
- Home screen installation

### Mobile Access
1. Visit your deployed URL on mobile
2. Browser will prompt to "Add to Home Screen"
3. App will function like a native mobile app

### PWA Configuration

The PWA is configured via `public/manifest.json`:
- App name and description
- Icons for various screen sizes
- Theme colors
- Shortcuts and protocol handlers

## Security Considerations

### API Key Security
- Store API keys securely
- Use environment variables for keys
- Implement proper rate limiting
- Monitor API usage regularly

### HTTPS
Always use HTTPS in production:
```nginx
server {
    listen 443 ssl;
    server_name your-domain.com;
    
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
    
    location / {
        proxy_pass http://localhost:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
    
    location /api {
        proxy_pass http://localhost:3001;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

### Firewall Configuration
```bash
# Allow only necessary ports
ufw allow 22    # SSH
ufw allow 80    # HTTP
ufw allow 443   # HTTPS
ufw enable
```

## Monitoring and Maintenance

### Health Checks
- Frontend: `GET /health`
- API: `GET /api/status`

### Logs
```bash
# PM2 logs
pm2 logs knirvcontroller

# Docker logs
docker logs knirvcontroller

# Application logs
tail -f /app/logs/knirvcontroller.log
```

### Database Backup
```bash
# Manual backup
cp /app/data/knirvcontroller.db /backups/backup-$(date +%Y%m%d).db

# Automated backup (cron)
0 2 * * * cp /app/data/knirvcontroller.db /backups/backup-$(date +\%Y\%m\%d).db
```

## Troubleshooting

### Common Issues

1. **Port conflicts**
   - Check if ports 3000/3001 are available
   - Modify PORT environment variables

2. **Database permissions**
   - Ensure write permissions to data directory
   - Check disk space

3. **API key issues**
   - Verify key format (starts with 'knirv_')
   - Check permissions and expiration
   - Monitor rate limits

### Support
For deployment issues, check:
1. Application logs
2. System resource usage
3. Network connectivity
4. Database integrity

## Performance Optimization

### Production Optimizations
- Enable gzip compression
- Use CDN for static assets
- Implement caching headers
- Monitor memory usage
- Scale horizontally with load balancer

### Database Optimization
- Regular database maintenance
- Monitor query performance
- Implement connection pooling
- Consider read replicas for high traffic
