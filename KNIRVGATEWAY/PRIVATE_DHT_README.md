# KNIRVGATEWAY - Private DHT Implementation

**Complete Private DHT deployment system with multi-platform support and automated failover**

## 🚀 Overview

This implementation provides a complete Private DHT deployment system as specified in `PrivateDHTDeploymentPlan.md`. It includes:

- **libp2p-based Private DHT** - Secure, decentralized peer discovery
- **Multi-platform Deployment** - Render (persistent), Netlify & Vercel (serverless)
- **Automated DNS Failover** - CloudFlare integration with leader election
- **Frontend Failover Logic** - Automatic health checks and redirection
- **Dynamic Peer Discovery** - `/provision` endpoint for real-time peer lists
- **Content Synchronization** - Automated failover page updates

## 🏗️ Architecture

### Deployment Modes

1. **Persistent Mode (Render)** - Full DHT node with bootstrap capabilities
2. **Serverless Mode (Netlify/Vercel)** - Lightweight proxy to persistent gateway

### Key Components

- `server.js` - Unified server supporting all deployment modes
- `lib/p2p/private_dht_manager.js` - libp2p-based private DHT implementation
- `lib/dns/cloudflare_manager.js` - Automated DNS failover management
- `netlify/functions/provision.js` - Netlify serverless provision endpoint
- `api/provision.js` - Vercel serverless provision endpoint
- `index.html` - Frontend failover logic with health checks
- `home.html` - Fallback content (copy of original site)

## 🚀 Quick Start

### Local Development

```bash
# Install dependencies (includes new libp2p packages)
npm install

# Start in persistent mode (full DHT)
npm run start:render

# Start in Netlify development mode
npm run dev

# Test provision endpoint
curl http://localhost:8080/provision
```

### Deployment Scripts

```bash
# Deploy to Render
make deploy-render

# Deploy to Netlify
make deploy-netlify

# Deploy to Vercel
make deploy-vercel

# Sync failover content
make sync-failover-page
```

## 📋 Environment Variables

### Render (Persistent Mode)
```
GATEWAY_MODE=persistent
KNIRV_CHAIN_ID=testnet
CLOUDFLARE_API_TOKEN=your_cloudflare_token
CLOUDFLARE_ZONE_ID=your_zone_id
INTERNAL_API_KEY=secure_random_key
INSTANCE_IP=auto_detected_or_manual
KNIRV_BOOTSTRAP_PEERS=peer1,peer2,peer3
```

### Netlify/Vercel (Serverless Mode)
```
GATEWAY_MODE=serverless
RENDER_GATEWAY_INTERNAL_API=https://render-url/internal/peers
INTERNAL_API_KEY=same_as_render_key
```

## 🔗 API Endpoints

### Core Endpoints

- `GET /provision` - Get list of available DHT peers
- `GET /health` - Health check and status
- `GET /dht/status` - DHT network status
- `GET /services` - Discovered services

### Internal Endpoints (Persistent Mode Only)

- `GET /internal/peers` - Internal peer list (requires API key)

## 🌐 Frontend Failover

The new `index.html` implements intelligent failover:

1. **Health Check** - Tests knirv.com availability
2. **Primary Redirect** - Redirects to knirv.com if healthy
3. **Fallback** - Serves local `home.html` if primary fails
4. **Retry Logic** - Multiple attempts with exponential backoff

## 🔧 Content Synchronization

Use the Makefile for content management:

```bash
# Sync content from knirvcom-repo submodule
make sync-failover-page

# Update all submodules
make update-submodules

# Setup submodules (first time)
make setup-submodules
```

## 🏥 Health Monitoring

### Health Check Response
```json
{
  "status": "healthy",
  "mode": "persistent",
  "timestamp": 1640995200000,
  "chainId": "testnet",
  "dht": {
    "isStarted": true,
    "connectedPeers": 5,
    "discoveredServices": ["knirvgraph", "knirvchain"]
  }
}
```

### Provision Response
```json
[
  "/ip4/192.168.1.100/tcp/4001/p2p/12D3KooWExample1",
  "/ip4/192.168.1.101/tcp/4001/p2p/12D3KooWExample2"
]
```

## 🔄 DNS Failover

Automated CloudFlare DNS management:

- **Health Monitoring** - Continuous health checks of primary gateway
- **Leader Election** - Prevents multiple instances updating DNS
- **Automatic Failover** - Updates DNS records on failure detection
- **Fast Recovery** - 60-second TTL for rapid failover

## 🧪 Testing

```bash
# Test all provision endpoints
make test-provision

# Test health endpoints
make test-health

# Check system status
make status

# Run full CI build
make ci-build
```

## 📁 New File Structure

```
KNIRVGATEWAY/
├── server.js                          # NEW: Unified server
├── index.html                         # MODIFIED: Failover frontend
├── home.html                          # NEW: Fallback content
├── lib/
│   ├── p2p/private_dht_manager.js     # NEW: Private DHT implementation
│   └── dns/cloudflare_manager.js      # NEW: DNS failover management
├── netlify/functions/provision.js     # NEW: Netlify serverless function
├── api/
│   ├── provision.js                   # NEW: Vercel serverless function
│   ├── health.js                      # NEW: Vercel health endpoint
│   └── dht-status.js                  # NEW: Vercel DHT status
├── render.yaml                        # NEW: Render deployment config
├── vercel.json                        # NEW: Vercel deployment config
├── Makefile                           # NEW: Build automation
└── DHT_Next_Steps_Plan.md             # NEW: Deployment guide
```

## 🚀 Deployment Guide

**See `DHT_Next_Steps_Plan.md` for complete step-by-step deployment instructions.**

The deployment process includes:

1. **Phase 1** - Install dependencies and test locally
2. **Phase 2** - Configure CloudFlare DNS
3. **Phase 3** - Deploy to Render, Netlify, and Vercel
4. **Phase 4** - Setup A2 Hosting for knirv.com
5. **Phase 5** - Configure and test all systems
6. **Phase 6** - Set up monitoring and maintenance

## 🔐 Security Features

- **API Key Management** - Secure environment variable storage
- **CORS Configuration** - Proper cross-origin resource sharing
- **Rate Limiting** - Built-in protection against abuse
- **Encrypted Communication** - TLS for all DHT traffic
- **Leader Election** - Prevents DNS update conflicts

## 📊 Monitoring & Observability

- **Health Checks** - Automated endpoint monitoring
- **DNS Analytics** - CloudFlare traffic analysis
- **Performance Metrics** - Platform-specific monitoring
- **Error Tracking** - Comprehensive logging
- **Failover Events** - Real-time failover notifications

## 🔧 Development Commands

```bash
# Development modes
make dev-render      # Start in persistent mode
make dev-netlify     # Start in Netlify mode
make dev-vercel      # Start in Vercel mode

# Content management
make sync-failover-page    # Sync content from submodule
make update-submodules     # Update all submodules

# Testing
make test-provision        # Test provision endpoints
make test-health          # Test health endpoints

# Deployment
make deploy-render        # Deploy to Render
make deploy-netlify       # Deploy to Netlify
make deploy-vercel        # Deploy to Vercel

# Maintenance
make clean               # Clean build artifacts
make clean-all          # Full clean including node_modules
make audit              # Security audit
```

## 🚨 Troubleshooting

### Common Issues

1. **Provision endpoint returns empty array:**
   - Check `RENDER_GATEWAY_INTERNAL_API` configuration
   - Verify `INTERNAL_API_KEY` matches across deployments

2. **DNS failover not working:**
   - Verify CloudFlare API credentials
   - Check `INSTANCE_IP` configuration

3. **Frontend not redirecting:**
   - Check knirv.com availability
   - Verify CORS configuration

### Debug Commands

```bash
# Check DHT status
curl https://[render-url]/dht/status

# Test internal API
curl -H "Authorization: Bearer [INTERNAL_API_KEY]" \
     https://[render-url]/internal/peers

# Check DNS
dig gateway.knirv.network
```

## 📈 Implementation Status

✅ **Completed:**
- Private DHT with libp2p
- Unified server architecture
- Provision endpoint (all modes)
- Frontend failover logic
- Content synchronization system
- CloudFlare DNS management
- Deployment configurations

🚀 **Ready for Deployment:**
- All code implemented and tested
- Configuration files ready
- Documentation complete
- Follow `DHT_Next_Steps_Plan.md` for deployment

## 📞 Support

For deployment assistance:

1. Review `DHT_Next_Steps_Plan.md` for detailed instructions
2. Check platform-specific logs for errors
3. Verify all environment variables are configured
4. Test each component individually before integration

The implementation is complete and production-ready!
