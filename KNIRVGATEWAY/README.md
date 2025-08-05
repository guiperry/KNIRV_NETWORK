# KNIRV GATEWAY

**The Unified Web Portal and API Gateway for the KNIRV Decentralized Trusted Execution Network**

[![Netlify Status](https://api.netlify.com/api/v1/badges/your-badge-id/deploy-status)](https://app.netlify.com/sites/your-site-name/deploys)
[![Node.js Version](https://img.shields.io/badge/node-%3E%3D18.0.0-brightgreen)](https://nodejs.org/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

## Overview

KNIRVGATEWAY serves as the primary web portal and API gateway for the KNIRV D-TEN (Decentralized Trusted Execution Network). It combines a modern, responsive website with serverless API gateway functionality, providing a unified entry point for users, developers, and services within the KNIRV ecosystem.

### Key Features

- 🌐 **Modern Web Portal**: Responsive website showcasing KNIRV D-TEN capabilities
- 🚀 **Serverless API Gateway**: Netlify Functions-based gateway with SSE support
- 👨‍💻 **Developer Portal**: Comprehensive documentation and tools for KNIRV developers
- 📚 **Documentation Hub**: Integrated documentation with Docsify
- 🔄 **Real-time Updates**: Server-Sent Events (SSE) for live data streaming
- 🔐 **Authentication**: Secure API access and user management
- 📊 **Health Monitoring**: Real-time service health and metrics
- 🎨 **Glass Morphism UI**: Modern design with balanced blue/purple color scheme

## Architecture

### Web Portal Components

```
KNIRVGATEWAY/
├── index.html                    # Main website entry point
├── assets/                       # Static assets (CSS, JS, images)
├── images/                       # Website images and branding
├── agent-developer-portal/       # Developer documentation and tools
├── documentation/                # Docsify-based documentation
└── logo/                        # KNIRV branding components
```

### API Gateway Components

```
netlify/functions/
├── gateway-sse.js               # Main API gateway with SSE support
└── health-monitor.js            # Service health monitoring
```

### Configuration

```
├── netlify.toml                 # Netlify deployment configuration
├── package.json                 # Node.js dependencies and scripts
└── _redirects                   # URL routing rules
```

## Quick Start

### Prerequisites

- Node.js 18.0.0 or higher
- npm or yarn package manager
- Netlify CLI (for local development)

### Installation

1. **Clone and navigate to the directory**:
   ```bash
   cd KNIRVGATEWAY
   ```

2. **Install dependencies**:
   ```bash
   npm install
   ```

3. **Start local development server**:
   ```bash
   npm run dev
   ```

4. **Access the portal**:
   - Website: http://localhost:8888
   - API Gateway: http://localhost:8888/gateway/*
   - Health Monitor: http://localhost:8888/health-monitor/*

### Build and Deploy

1. **Build for production**:
   ```bash
   npm run build
   ```

2. **Deploy to Netlify**:
   ```bash
   npm run deploy
   ```

## API Gateway Functionality

### Endpoints

#### Gateway Management
- `GET /gateway/health` - Gateway health status
- `GET /gateway/services` - Available services list
- `GET /gateway/metrics` - Performance metrics
- `GET /gateway/events` - SSE event stream

#### Health Monitoring
- `GET /health-monitor/status` - Service health status
- `GET /health-monitor/events` - SSE health updates

#### Authentication
- `POST /auth/login` - User authentication
- `POST /auth/logout` - User logout
- `GET /auth/verify` - Token verification

#### Service Proxy
- `/api/*` - Proxy to KNIRVROOT services
- `/economics/*` - Economics service endpoints
- `/tunnel/*` - Tunnel registry endpoints

### Server-Sent Events (SSE)

The gateway supports real-time updates via SSE:

```javascript
// Connect to gateway events
const eventSource = new EventSource('/gateway/events');
eventSource.onmessage = function(event) {
    const data = JSON.parse(event.data);
    console.log('Gateway update:', data);
};

// Connect to health monitoring
const healthSource = new EventSource('/health-monitor/events');
healthSource.onmessage = function(event) {
    const health = JSON.parse(event.data);
    console.log('Health update:', health);
};
```

## Development

### Available Scripts

- `npm run dev` - Start development server with hot reload
- `npm run build` - Build for production
- `npm run deploy` - Deploy to Netlify
- `npm run functions:test` - Test gateway functions
- `npm run validate` - Validate Netlify Functions

### Local Development

1. **Start the gateway**:
   ```bash
   npm run dev
   ```

2. **Test API endpoints**:
   ```bash
   curl http://localhost:8888/gateway/health
   ```

3. **Test SSE functionality**:
   ```bash
   curl -N -H "Accept: text/event-stream" http://localhost:8888/gateway/events
   ```

### Environment Variables

Configure these environment variables for production:

```bash
# Service URLs
KNIRVROOT_URL=https://root.knirv.com
KNIRVCHAIN_URL=https://chain.knirv.com
KNIRVGRAPH_URL=https://graph.knirv.com
KNIRVNEXUS_URL=https://nexus.knirv.com

# Authentication
JWT_SECRET=your-jwt-secret
AUTH_ENABLED=true

# Monitoring
HEALTH_CHECK_INTERVAL=30000
METRICS_ENABLED=true
```

## Portal Sections

### Main Website
- **Hero Section**: KNIRV D-TEN overview and value proposition
- **Features**: Seven sovereign layers explanation
- **Technology**: Technical architecture and capabilities
- **Tokenomics**: NRN token information
- **Roadmap**: Development timeline and milestones

### Agent Developer Portal
- **Getting Started**: Quick start guide for developers
- **API Documentation**: Comprehensive API reference
- **SDK & Tools**: Development tools and libraries
- **Capabilities**: Available agent capabilities
- **Community**: Developer community and support

### Documentation Hub
- **Technical Docs**: Detailed technical documentation
- **Tutorials**: Step-by-step guides
- **API Reference**: Complete API documentation
- **Examples**: Code examples and use cases

## Deployment

### Netlify Configuration

The project is configured for Netlify deployment with:

- **Build Command**: `npm run build`
- **Publish Directory**: `.` (root)
- **Functions Directory**: `netlify/functions`
- **Node.js Version**: 18.x

### Production Checklist

- [ ] Environment variables configured
- [ ] SSL certificate installed
- [ ] Custom domain configured
- [ ] CDN caching optimized
- [ ] Security headers configured
- [ ] Performance monitoring enabled

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/new-feature`
3. Make your changes
4. Test thoroughly: `npm run validate`
5. Commit your changes: `git commit -am 'Add new feature'`
6. Push to the branch: `git push origin feature/new-feature`
7. Submit a pull request

## Testing

### Manual Testing

```bash
# Test gateway functionality
./scripts/test-gateway-migration.sh

# Test complete system
./scripts/validate-complete-migration.sh
```

### Integration Testing

```bash
# Run integration tests
./integration-tests/run_gateway_migration_tests.sh
```

## Support

- **Documentation**: [docs.knirv.com](https://docs.knirv.com)
- **Developer Portal**: [portal.knirv.com](https://portal.knirv.com)
- **Community**: [community.knirv.com](https://community.knirv.com)
- **Issues**: [GitHub Issues](https://github.com/knirv/issues)

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Built with [Netlify Functions](https://www.netlify.com/products/functions/)
- Documentation powered by [Docsify](https://docsify.js.org/)
- UI components from [Bootstrap](https://getbootstrap.com/)
- Icons from [Feather Icons](https://feathericons.com/)

---

**KNIRV Gateway** - Powering the future of decentralized AI and trusted execution networks.

For more information about the KNIRV ecosystem, visit [knirv.com](https://knirv.com).
