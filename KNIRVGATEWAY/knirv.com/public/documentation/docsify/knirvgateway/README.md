# KNIRV GATEWAY

**The Unified Web Portal and API Gateway for the KNIRV Decentralized Trusted Execution Network**

[![Netlify Status](https://api.netlify.com/api/v1/badges/your-badge-id/deploy-status)](https://app.netlify.com/sites/your-site-name/deploys)
[![Node.js Version](https://img.shields.io/badge/node-%3E%3D18.0.0-brightgreen)](https://nodejs.org/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

## Table of Contents

- [Overview](#overview)
- [Key Features](#key-features)
- [Architecture](#architecture)
- [Quick Start](#quick-start)
- [API Gateway Functionality](#api-gateway-functionality)
- [Server-Sent Events (SSE)](#server-sent-events-sse)
- [Development](#development)
- [Environment Variables](#environment-variables)
- [Portal Sections](#portal-sections)
- [Deployment](#deployment)
- [Contributing](#contributing)
- [Testing](#testing)
- [Support](#support)
- [License](#license)
- [Acknowledgments](#acknowledgments)
- [Changelog](#changelog)
- [Social Media Sharing Guide](#social-media-sharing-guide)


## Overview

KNIRVGATEWAY serves as the primary web portal and API gateway for the KNIRV D-TEN (Decentralized Trusted Execution Network). It combines a modern, responsive website with serverless API gateway functionality, providing a unified entry point for users, developers, and services within the KNIRV ecosystem.  Migrated from a Go-based WebSocket gateway (KNIRVGATEWAY), it now utilizes Netlify Functions and Server-Sent Events (SSE) for improved performance and browser compatibility.


## Key Features

- 🌐 **Modern Web Portal**: Responsive website showcasing KNIRV D-TEN capabilities
- 🚀 **Serverless API Gateway**: Netlify Functions-based gateway with SSE support
- 🧪 **KNIRVTESTNET Integration**: Live testnet frontend with AWS backend connectivity
- 👨‍💻 **Developer Portal**: Comprehensive documentation and tools for KNIRV developers
- 📚 **Documentation Hub**: Integrated documentation with Docsify
- 🔄 **Real-time Updates**: Server-Sent Events (SSE) for live data streaming
- 🔐 **Authentication**: Secure API access and user management (JWT-based, configurable)
- 📊 **Health Monitoring**: Real-time service health and metrics
- 🎨 **Glass Morphism UI**: Modern design with balanced blue/purple color scheme
- 🗣️ **Social Media Optimization**: Platform-specific content and meta tags for enhanced sharing.


## Architecture

### Web Portal Components

```
KNIRVGATEWAY/
├── index.html                    # Main website entry point
├── assets/                       # Static assets (CSS, JS, images)
├── images/                       # Website images and branding
├── agent-developer-portal/       # Developer documentation and tools
├── documentation/                # Docsify-based documentation
├── knirvtestnet/                 # KNIRVTESTNET frontend integration
│   ├── index.html               # Testnet dashboard
│   ├── testnet-config.js        # Testnet API configuration
│   └── README.md                # Testnet documentation
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
- `/api/*` - Proxy to KNIRVORACLE services
- `/economics/*` - Economics service endpoints
- `/tunnel/*` - Tunnel registry endpoints


## Server-Sent Events (SSE)

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

## Environment Variables

Configure these environment variables for production:

```bash
# Service URLs
KNIRVORACLE_URL=https://root.knirv.com
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


## Changelog

All notable changes to the KNIRV Gateway project will be documented in this section.  The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2025-01-08

### Added
- **Initial Release**: KNIRV Gateway unified web portal and API gateway
- **Netlify Functions**: Serverless API gateway with SSE support
- **Web Portal**: Modern responsive website for KNIRV D-TEN
- **Developer Portal**: Comprehensive documentation and tools
- **Documentation Hub**: Integrated Docsify-based documentation
- **Real-time Updates**: Server-Sent Events (SSE) for live data streaming
- **Authentication System**: Secure API access and user management
- **Health Monitoring**: Real-time service health and metrics
- **Glass Morphism UI**: Modern design with balanced blue/purple color scheme

### Gateway Functions
- `gateway-sse.js`: Main API gateway with SSE support
  - Service discovery and routing
  - Authentication and authorization
  - Real-time event streaming
  - Proxy functionality for KNIRV services
- `health-monitor.js`: Service health monitoring
  - Real-time health status
  - Performance metrics
  - SSE health updates

### API Endpoints
- **Gateway Management**: `/gateway/*` endpoints
- **Health Monitoring**: `/health-monitor/*` endpoints  
- **Authentication**: `/auth/*` endpoints
- **Service Proxy**: `/api/*`, `/economics/*`, `/tunnel/*` endpoints

### Infrastructure
- **Netlify Deployment**: Production-ready serverless deployment
- **Build System**: Proper build configuration for static site + functions
- **Environment Configuration**: Production environment variable support
- **SSL/HTTPS**: Secure connections and API access
- **CDN**: Global content delivery network

### Documentation
- **README.md**: Comprehensive project documentation
- **API Documentation**: Complete API reference
- **Developer Guide**: Getting started and development instructions
- **Deployment Guide**: Production deployment instructions

### Development Tools
- **Local Development**: `npm run dev` with hot reload
- **Testing Scripts**: Gateway function validation
- **Build Scripts**: Production build process
- **Deployment Scripts**: Automated deployment to Netlify

## [0.9.0] - 2025-01-07 (Pre-release)

### Added
- **Gateway Migration**: Migrated from KNIRVGATEWAY Go-based WebSocket gateway
- **SSE Conversion**: Converted WebSocket functionality to Server-Sent Events
- **Netlify Functions**: Implemented serverless functions architecture
- **Economics Integration**: Integrated economics module into KNIRVORACLE
- **Tunnel Registry Migration**: Moved tunnel registry to KNIRVORACLE

### Changed
- **Architecture**: Moved from centralized gateway to distributed serverless functions
- **Communication**: Replaced WebSockets with Server-Sent Events for better browser compatibility
- **Service Location**: Moved economics and tunnel registry to KNIRVORACLE for better architecture
- **Build Process**: Updated from static-only to static + functions build

### Removed
- **Go WebSocket Gateway**: Replaced with Netlify Functions
- **Centralized Gateway Server**: Distributed functionality across serverless functions

## Migration History

### From KNIRVGATEWAY to KNIRVGATEWAY
- **Renamed**: Project renamed from KNIRVGATEWAY to KNIRVGATEWAY to better reflect its role
- **Enhanced**: Added comprehensive API gateway functionality
- **Unified**: Combined web portal and API gateway into single project
- **Modernized**: Updated architecture for serverless deployment

### Architecture Evolution
1. **Static Website** (Original): Pure static HTML/CSS/JS website
2. **Website + Functions** (v0.9): Added Netlify Functions for API gateway
3. **Unified Gateway** (v1.0): Full integration as KNIRV Gateway portal

## Technical Debt Addressed

### Build Configuration
- ✅ Fixed build command from static-only to proper dependency installation
- ✅ Updated Netlify configuration for functions support
- ✅ Proper environment variable handling

### Security
- ✅ Added proper authentication system
- ✅ Implemented secure API access
- ✅ Added security headers and HTTPS enforcement

### Performance
- ✅ Optimized for global CDN delivery
- ✅ Implemented efficient SSE streaming
- ✅ Added proper caching strategies

### Maintainability
- ✅ Comprehensive documentation
- ✅ Proper project structure
- ✅ Development and testing tools
- ✅ Clear deployment procedures

## Future Roadmap

### Planned Features
- [ ] **Enhanced Authentication**: OAuth2/OIDC integration
- [ ] **Advanced Monitoring**: Detailed analytics and logging
- [ ] **API Rate Limiting**: Request throttling and quotas
- [ ] **WebSocket Support**: Optional WebSocket fallback for SSE
- [ ] **Multi-language Support**: Internationalization
- [ ] **Advanced Caching**: Redis-based caching layer
- [ ] **Load Balancing**: Advanced traffic distribution
- [ ] **API Versioning**: Versioned API endpoints

### Technical Improvements
- [ ] **TypeScript Migration**: Convert to TypeScript for better type safety
- [ ] **Testing Suite**: Comprehensive unit and integration tests
- [ ] **CI/CD Pipeline**: Automated testing and deployment
- [ ] **Performance Monitoring**: Real-time performance analytics
- [ ] **Error Tracking**: Advanced error monitoring and alerting


## Social Media Sharing Guide

This guide explains the comprehensive social media sharing optimization implemented for the KNIRV D-TEN website. Each platform has been customized with platform-specific content, images, and metadata to maximize engagement and reach.

## Platform-Specific Optimizations

### 🔵 Facebook
- **Card Image**: `facebook-card.png` (1200x630px)
- **Content Style**: Professional, business-focused
- **Sample Share Text**:
```
🚀 Experience the world's first self-improving AI network with 12 sovereign layers. Transform AI failures into collective knowledge with verifiable execution and NRN token economics. Join the future of AI! 🤖
```

### 🐦 Twitter/X
- **Card Image**: `twitter-card.png` (1200x675px)
- **Content Style**: Concise, hashtag-rich, engaging
- **Sample Tweet**:
```
🤖 World's first self-improving AI network with 12 sovereign layers
🔗 Transform AI failures into collective knowledge  
⚡ Verifiable execution & NRN token economics
🌐 Join the future of decentralized AI!

#AI #Blockchain #DeFi #Web3
```

### 💼 LinkedIn
- **Card Image**: `linkedin-card.png` (1200x627px)
- **Content Style**: Professional, thought leadership
- **Sample LinkedIn Post**:
```
Experience the world's first Decentralized Trusted Execution Network (D-TEN) that transforms AI failures into collective knowledge through twelve sovereign layers. Our revolutionary framework enables self-improving AI systems with verifiable execution and NRN token economics.
```

### 📺 YouTube
- **Thumbnail**: `youtube-thumbnail.png` (1280x720px)
- **Content Style**: Video-focused, educational
- **Sample Description**:
```
🎥 Watch how KNIRV D-TEN transforms AI failures into collective knowledge! 

🔥 Features:
• 12 sovereign layers working in harmony
• Self-healing AI network
• Verifiable execution environments  
• NRN token economics
• Collective intelligence

Subscribe for updates on the future of AI! 🚀
```

### 🐙 GitHub
- **Card Image**: `github-card.png` (1280x640px)
- **Content Style**: Developer-focused, technical
- **Sample README Description**:
```
🚀 Open-source implementation of the world's first self-improving AI network

## Key Features
- 12 sovereign layers (KNIRV-ORACLE, KNIRVCHAIN, KNIRVGRAPH, etc.)
- Self-healing AI through ErrorNode → SkillNode transformation
- Verifiable execution with KNIRV-NEXUS DVE
- NRN token economics and