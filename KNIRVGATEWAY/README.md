# Consolidated Documentation

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

## BUILD SYSTEM README

# KNIRVGATEWAY Build System

## Overview

This document describes the automated build system for KNIRVGATEWAY that ensures Netlify functions work correctly by automatically patching missing dependencies during the build process.

## Problem Solved

During Netlify deployment, certain npm packages have missing `dist` files that cause build failures:

1. **bcryptjs** - Missing `dist/bcrypt.js` file that the main index.js requires
2. **formidable** - Missing `dist/*` files that the package.json exports point to

These issues cause Netlify's esbuild bundler to fail with "Could not resolve" errors.

## Solution Architecture

### 1. Automated Patch System

The build system includes an automated patcher that:
- Detects missing dist files in problematic packages
- Copies source files to the expected dist locations
- Verifies patches are applied correctly
- Prevents infinite loops during postinstall hooks

### 2. Build Integration

The patches are automatically applied during:
- **Netlify deployment** via `netlify.toml` build command
- **Local development** via npm scripts
- **CI/CD pipelines** via package.json build scripts

## Files and Scripts

### Core Patcher Script
- **`scripts/patch-netlify-deps.js`** - Main patcher that fixes missing dist files
  - Patches bcryptjs by copying `src/bcrypt.js` to `dist/bcrypt.js`
  - Patches formidable by copying entire `src/` directory to `dist/`
  - Includes infinite loop prevention for postinstall contexts
  - Provides detailed logging and error reporting

### Test and Verification
- **`scripts/test-netlify-patches.js`** - Verification script that tests patches
  - Verifies all required files exist
  - Tests that require paths will work
  - Simulates Netlify build process
  - Provides comprehensive test reporting

### Package Configuration
- **`netlify/functions/package.json`** - Dependencies for Netlify functions
  - Includes postinstall hook that runs the patcher
  - Contains all required dependencies (bcryptjs, formidable, axios, node-cache)
  - Prevents infinite loops with lifecycle event detection

### Build Scripts (package.json)
- **`build:netlify-functions`** - Builds and patches Netlify functions
- **`patch-netlify-deps`** - Runs patcher manually
- **`build`** - Main build script that includes function patching

### Netlify Configuration
- **`netlify.toml`** - Updated to include function patching in build process

## Usage

### Automatic (Recommended)

The patches are applied automatically during:

```bash
# Netlify deployment (automatic)
npm run build

# Local development
npm run dev
npx netlify dev

# Manual function build
npm run build:netlify-functions
```

### Manual Patching

If you need to apply patches manually:

```bash
# Run patcher directly
npm run patch-netlify-deps

# Or run the script directly
node scripts/patch-netlify-deps.js

# Test patches
node scripts/test-netlify-patches.js
```

### Verification

To verify patches are working:

```bash
# Run test suite
npm run test-netlify-patches

# Check Netlify dev works without errors
npx netlify dev
```

## Build Process Flow

1. **Install Dependencies**
   ```bash
   cd netlify/functions && npm install
   ```

2. **Automatic Patch Application** (via postinstall hook)
   ```bash
   node ../../scripts/patch-netlify-deps.js
   ```

3. **Patch Verification**
   - bcryptjs: `dist/bcrypt.js` exists and has content
   - formidable: `dist/index.js`, `dist/Formidable.js`, etc. exist
   - All require paths resolve correctly

4. **Function Loading**
   - Netlify CLI loads all functions without errors
   - esbuild bundling succeeds
   - Functions are ready for deployment

## Error Prevention

### Infinite Loop Protection
The patcher detects when it's running in a postinstall context and skips npm install to prevent infinite loops.

### Idempotent Operations
The patcher checks if patches are already applied and skips them, making it safe to run multiple times.

### Comprehensive Logging
All operations are logged with timestamps and status indicators for easy debugging.

## Troubleshooting

### Common Issues

1. **"Could not resolve ./dist/bcrypt.js"**
   - Run: `npm run patch-netlify-deps`
   - Verify: `node scripts/test-netlify-patches.js`

2. **"Could not resolve formidable"**
   - Run: `npm run build:netlify-functions`
   - Check: `ls netlify/functions/node_modules/formidable/dist/`

3. **Infinite loop during build**
   - Check if postinstall hook is calling npm install
   - Verify lifecycle event detection in patcher

### Debug Commands

```bash
# Check patch status
node scripts/test-netlify-patches.js

# Manually apply patches
node scripts/patch-netlify-deps.js

# Test Netlify functions
npx netlify dev

# Check function loading
curl http://localhost:8888/.netlify/functions/provision
```

## Production Deployment

### Netlify
The build system is fully integrated with Netlify deployment:

1. Netlify runs the build command from `netlify.toml`
2. Build command includes `npm run build:netlify-functions`
3. Patches are applied automatically during function dependency installation
4. All functions load without errors

### Other Platforms
For deployment to other platforms (Vercel, AWS Lambda, etc.), run:

```bash
npm run build:netlify-functions
```

This ensures all patches are applied before deployment.

## Maintenance

### Adding New Patches
To add patches for additional packages:

1. Add patch logic to `scripts/patch-netlify-deps.js`
2. Add verification to `scripts/test-netlify-patches.js`
3. Test with `npm run build:netlify-functions`

### Updating Dependencies
When updating bcryptjs or formidable versions:

1. Test that patches still work: `npm run test-netlify-patches`
2. Update patch logic if package structure changes
3. Verify Netlify dev works: `npx netlify dev`

## Success Indicators

✅ **All functions load without errors in `npx netlify dev`**
✅ **No "Could not resolve" errors in build output**
✅ **Test script passes: `node scripts/test-netlify-patches.js`**
✅ **Provision endpoint responds: `curl http://localhost:8888/.netlify/functions/provision`**

The build system ensures reliable, automated deployment of KNIRVGATEWAY with full Netlify Functions support.


## CONFIGURATION MANAGEMENT

# KNIRV Gateway Configuration Management System

## Overview

The KNIRV Gateway now includes a centralized configuration management system that allows for easy maintenance and updates of links, features, and settings across all portal components. This system uses YAML configuration files and JavaScript loaders to provide dynamic, maintainable configuration management.

## Architecture

### Configuration Files

#### 1. YAML Configuration (`config/portal-links.yaml`)
The master configuration file containing all links, settings, and feature flags:

```yaml
# Main Navigation Links
navigation:
  main_site: "https://knirv.com"
  documentation: "../documentation/docsify/"
  graphchain_explorer: "../graphchain-explorer/"
  nexus_portal: "../nexus-portal/"

# External Services
external_services:
  payment_gateway: "https://pay.knirv.com/add-funds"
  testnet_access: "https://testnet.knirv.com"

# Feature Flags
features:
  authentication_enabled: true
  payment_gateway_enabled: true
  nexus_integration_enabled: true
```

#### 2. JSON Configuration (`config/portal-config.json`)
Browser-compatible JSON version of the YAML configuration for direct loading.

### JavaScript Components

#### 1. Configuration Loader (`js/config-loader.js`)
The main configuration management class that:
- Loads configuration from JSON files
- Provides fallback configurations
- Updates UI elements dynamically
- Manages feature visibility

#### 2. Universal Footer (`js/universal-footer.js`)
Shared footer component that:
- Uses configuration for all links
- Maintains consistent branding
- Provides responsive design
- Integrates with all portal pages

## Usage Guide

### Basic Configuration Loading

```javascript
// Configuration is automatically loaded on page load
document.addEventListener('DOMContentLoaded', async () => {
    await window.knirvConfig.loadConfig();
    console.log('Configuration loaded:', window.knirvConfig.config);
});
```

### Accessing Configuration Values

```javascript
// Get navigation links
const mainSiteUrl = window.knirvConfig.getNavigationLink('main_site');
const docsUrl = window.knirvConfig.getNavigationLink('documentation');

// Get external service URLs
const paymentUrl = window.knirvConfig.getExternalService('payment_gateway');

// Check feature flags
if (window.knirvConfig.isFeatureEnabled('authentication_enabled')) {
    // Enable authentication features
}

// Get nested configuration values
const primaryColor = window.knirvConfig.getConfigValue('ui.theme.primary_color');
```

### Dynamic UI Updates

The configuration system automatically updates UI elements with `data-config` attributes:

```html
<!-- Navigation links -->
<a href="#" data-config="main-site">Back to KNIRV.com</a>
<a href="#" data-config-nav="documentation">Documentation</a>

<!-- Footer links -->
<a href="#" data-config-footer="social.github">GitHub</a>
<a href="#" data-config-footer="legal.terms">Terms of Service</a>

<!-- Feature-based visibility -->
<div data-feature="authentication_enabled">
    <!-- This div will be hidden if authentication is disabled -->
</div>

<!-- Payment gateway links -->
<button data-config="payment-gateway" onclick="addFunds()">Add Funds</button>

<!-- iFrame configurations -->
<iframe data-config-iframe="graphchain_explorer"></iframe>
```

## Configuration Reference

### Navigation Section
```yaml
navigation:
  main_site: "https://knirv.com"                    # Main KNIRV website
  documentation: "../documentation/docsify/"        # Documentation portal
  graphchain_explorer: "../graphchain-explorer/"    # GraphChain explorer
  nexus_portal: "../nexus-portal/"                  # KNIRV-NEXUS portal
  support_desk: "../support-desk/"                  # Support system
  nanda_ans: "../nanda_ans/"                        # NANDA+ANS registry
```

### External Services
```yaml
external_services:
  payment_gateway: "https://pay.knirv.com/add-funds"  # Payment processing
  knirv_website: "https://knirv.com"                  # Main website
  testnet_access: "https://testnet.knirv.com"         # Testnet access
```

### Documentation Links
```yaml
documentation:
  whitepapers:
    knirv_oracle: "../documentation/static/whitepapers/KNIRVROOT_Whitepaper.md"
    knirv_router: "../documentation/static/whitepapers/KNIRV-ROUTER_Whitepaper.md"
    # ... additional whitepapers
  guides:
    getting_started: "../documentation/static/guides/getting-started.md"
    api_reference: "../documentation/static/api/reference.md"
```

### Footer Configuration
```yaml
footer:
  legal:
    terms: "../documentation/static/legal/terms-of-service.md"
    privacy: "../documentation/static/legal/privacy-policy.md"
    contribution: "../documentation/static/contributing/contribution-guidelines.md"
  social:
    github: "https://github.com/knirv-network"
    discord: "https://discord.gg/knirv"
    twitter: "https://twitter.com/knirvnetwork"
    telegram: "https://t.me/knirvnetwork"
  resources:
    documentation: "../documentation/docsify/"
    support: "../support-desk/"
    forum: "../forum/"
    blog: "https://blog.knirv.com"
```

### Feature Flags
```yaml
features:
  authentication_enabled: true          # Enable/disable authentication
  payment_gateway_enabled: true         # Enable/disable payment features
  nexus_integration_enabled: true       # Enable/disable NEXUS integration
  graphchain_explorer_enabled: true     # Enable/disable GraphChain explorer
  nanda_ans_enabled: true              # Enable/disable NANDA+ANS features
  support_desk_enabled: true           # Enable/disable support desk
```

### UI Configuration
```yaml
ui:
  theme:
    primary_color: "#00c0fa"           # Primary brand color
    secondary_color: "#2b56f5"         # Secondary brand color
    accent_color: "#8b5cf6"            # Accent color
  branding:
    logo_url: "../logo/knirv-logo.png"
    favicon_url: "favicon.png"
    site_title: "KNIRV Developer Portal"
    site_description: "Build on the Decentralized Trusted Execution Network"
```

### iFrame Configuration
```yaml
iframes:
  graphchain_explorer:
    url: "../graphchain-explorer/"
    title: "KNIRV Graphchain Explorer"
    height: "800px"
  documentation:
    url: "../documentation/docsify/"
    title: "KNIRV Documentation"
    height: "800px"
```

## API Reference

### KNIRVConfigLoader Class

#### Methods

##### `loadConfig()`
Loads configuration from JSON file with fallback support.

```javascript
await window.knirvConfig.loadConfig();
```

##### `getNavigationLink(key)`
Returns a navigation URL by key.

```javascript
const url = window.knirvConfig.getNavigationLink('main_site');
```

##### `getDocumentationLink(category, key)`
Returns a documentation URL by category and key.

```javascript
const url = window.knirvConfig.getDocumentationLink('whitepapers', 'knirv_oracle');
```

##### `getFooterLink(category, key)`
Returns a footer URL by category and key.

```javascript
const url = window.knirvConfig.getFooterLink('social', 'github');
```

##### `getExternalService(key)`
Returns an external service URL by key.

```javascript
const url = window.knirvConfig.getExternalService('payment_gateway');
```

##### `isFeatureEnabled(feature)`
Checks if a feature is enabled.

```javascript
if (window.knirvConfig.isFeatureEnabled('authentication_enabled')) {
    // Feature is enabled
}
```

##### `getIframeConfig(key)`
Returns iFrame configuration by key.

```javascript
const config = window.knirvConfig.getIframeConfig('graphchain_explorer');
// Returns: { url: "...", title: "...", height: "..." }
```

##### `createLink(category, key, text, className)`
Creates a configured link element.

```javascript
const link = window.knirvConfig.createLink('navigation', 'main_site', 'Home', 'nav-link');
```

##### `createConfiguredButton(text, action, configKey, className)`
Creates a configured button element.

```javascript
const button = window.knirvConfig.createConfiguredButton(
    'Add Funds', 
    'payment_gateway', 
    'payment_gateway',
    'btn-primary'
);
```

## Universal Footer System

### Features
- **Consistent Branding**: Unified footer across all portal pages
- **Dynamic Links**: All links sourced from configuration
- **Responsive Design**: Mobile-friendly layout
- **Easy Integration**: Single script inclusion

### Implementation
```html
<!-- Include in any page -->
<script src="../js/universal-footer.js"></script>
```

### Customization
```javascript
// Add custom footer section
window.knirvFooter.addCustomSection('Custom Links', [
    { text: 'Custom Link 1', url: '/custom1', external: false },
    { text: 'External Link', url: 'https://example.com', external: true }
]);

// Update footer with new configuration
window.knirvFooter.updateFooterLinks(newConfig);
```

## Maintenance

### Updating Configuration

1. **Edit YAML File**: Modify `config/portal-links.yaml`
2. **Convert to JSON**: Update `config/portal-config.json` with the same changes
3. **Test Changes**: Verify all links and features work correctly
4. **Deploy**: Push changes to production

### Adding New Features

1. **Add Feature Flag**: Include in `features` section
2. **Update UI Elements**: Add `data-feature` attributes
3. **Test Feature Toggle**: Verify feature shows/hides correctly

### Adding New Links

1. **Choose Section**: Add to appropriate configuration section
2. **Update UI**: Add `data-config` attributes to HTML elements
3. **Test Links**: Verify all links resolve correctly

## Best Practices

1. **Use Relative Paths**: For internal links, use relative paths for portability
2. **Feature Flags**: Use feature flags for gradual rollouts
3. **Fallback URLs**: Always provide fallback URLs for critical links
4. **Testing**: Test configuration changes in development before production
5. **Documentation**: Update this documentation when adding new configuration options

## Troubleshooting

### Common Issues

1. **Configuration Not Loading**: Check browser console for fetch errors
2. **Links Not Updating**: Verify `data-config` attributes are correct
3. **Features Not Hiding**: Check feature flag names and values
4. **Footer Not Appearing**: Ensure universal-footer.js is loaded

### Debug Mode
```javascript
// Enable debug logging
window.knirvConfig.debugMode = true;
```

---

*This configuration system is designed to make KNIRV Gateway maintenance easier and more reliable across all portal components.*


## DOWNLOAD CONFIGURATION

# KNIRV Gateway Download Configuration System

## Overview

All download buttons across KNIRVGATEWAY product pages now use a centralized configuration system that loads download links from `config/portal-links.yaml`. This eliminates hardcoded URLs and provides a single source of truth for all download links.

## Configuration File

The download configuration is stored in `KNIRVGATEWAY/config/portal-links.yaml` under the `downloads` section:

```yaml
downloads:
  knirvrouter:
    windows: "https://releases.knirv.network/knirvrouter/windows/knirvrouter-setup.exe"
    mac: "https://releases.knirv.network/knirvrouter/mac/knirvrouter.dmg"
    linux: "https://releases.knirv.network/knirvrouter/linux/knirvrouter.AppImage"
    requirements:
      windows: "Windows 10/11, 8GB RAM, 100GB Storage"
      mac: "macOS 10.15+, 8GB RAM, 100GB Storage"
      linux: "Ubuntu 20.04+/CentOS 8+, 8GB RAM, 100GB Storage"
    note: "You'll need 1000 NRN tokens to start routing."
```

## Supported Products

The system supports the following KNIRV products:

1. **KNIRVROUTER** - Network routing software
2. **KNIRVANA** - Gaming platform
3. **KNIRVORACLE** - Oracle bootnode software
4. **KNIRVSDK** - Software development kits
5. **KNIRVWALLET** - Wallet application
6. **KNIRVNEXUS** - DVE (Development Virtual Environment) services

## Implementation

### Config Loader (js/config-loader.js)

The `KNIRVConfigLoader` class handles:
- Loading configuration from `portal-links.yaml`
- Setting up global download functions
- Managing download link retrieval
- Handling analytics tracking

### Global Functions

The following global functions are automatically created:
- `downloadRouter(platform)` - KNIRVROUTER downloads
- `downloadGame(platform)` - KNIRVANA downloads
- `downloadBootnode(type)` - KNIRVORACLE downloads
- `downloadSDK(language)` - KNIRV SDK downloads
- `downloadWallet(platform)` - KNIRVWALLET downloads
- `rentDVE(plan)` - KNIRVNEXUS DVE rental
- `accessDVE(method)` - KNIRVNEXUS DVE access

### Product Pages

All product pages have been updated to remove hardcoded download functions:
- `knirvrouter.html`
- `knirvana.html`
- `knirvoracle.html`
- `knirvsdk.html`
- `knirvwallet.html`
- `knirvnexus.html`

## Configuration Structure

### Standard Downloads
```yaml
product_name:
  platform: "download_url"
  requirements:
    platform: "system requirements"
  note: "additional information"
```

### SDK Downloads
```yaml
knirvsdk:
  language: "install_command"
  instructions:
    language: "detailed installation instructions"
  documentation:
    language: "documentation_url"
```

## Benefits

1. **Centralized Management** - All download links in one place
2. **Easy Updates** - Change URLs without touching HTML files
3. **Consistency** - Uniform download experience across all products
4. **Maintainability** - Reduced code duplication
5. **Analytics** - Consistent tracking across all downloads

## Testing

Use `test-download-config.html` to verify the configuration system:
1. Open the test page in a browser
2. Check that configuration loads successfully
3. Test download buttons for all products
4. Verify that alerts show correct information from config

## Updating Download Links

To update download links:
1. Edit `config/portal-links.yaml`
2. Update the relevant product section
3. No changes needed to HTML files
4. Test using the test page

## Error Handling

The system includes fallback behavior:
- If configuration fails to load, uses fallback config
- If specific download not found, shows appropriate message
- Graceful degradation for missing platforms/languages

## Analytics

All downloads are tracked with Google Analytics:
- Event category: Product name (e.g., "KNIRVROUTER")
- Event action: "download" or "sdk_download"
- Event label: Platform or language

## Future Enhancements

Potential improvements:
- Version management for downloads
- Platform detection and auto-selection
- Download progress tracking
- Mirror URL support
- Checksum verification links


## NETLIFY CLI FIX

# Netlify CLI Dependency Fix for Render Deployment

This document explains the netlify-cli dependency issues and fixes applied to KNIRVGATEWAY for Render deployment.

## Problem

The KNIRVGATEWAY application uses netlify/functions routes for serverless functionality, even when deployed in persistent mode on Render. However, netlify-cli was frequently becoming corrupted or missing during builds, causing deployment failures with errors like:

```
❌ Found 2 critical issues:
❌   - netlify-cli not found in node_modules
❌   - netlify-cli binary not found in node_modules
❌ Health check failed - automatic fix may be needed
```

## Root Cause

1. **Build Process Issues**: The build process wasn't consistently installing devDependencies on Render
2. **Corruption**: netlify-cli installation was prone to corruption during npm operations
3. **Version Conflicts**: Different versions of netlify-cli had compatibility issues
4. **Render Environment**: Render's build environment sometimes skipped devDependencies

## Solution

### 1. Automated Netlify CLI Ensurer

Created `scripts/ensure-netlify-cli.js` that:
- Checks if netlify-cli is properly installed
- Tests if the netlify command works
- Automatically installs or reinstalls netlify-cli if needed
- Uses a specific stable version (21.6.0)
- Handles timeouts and corruption gracefully

### 2. Updated Build Process

Modified package.json scripts:
- `ensure-netlify-cli`: Runs the automated ensurer
- `build`: Now includes netlify-cli check before building
- `build:persistent`: Optimized for Render deployment with netlify-cli
- `auto-fix`: Uses the new ensurer instead of shell script

### 3. Render Configuration

Updated `render.yaml`:
- Uses `npm install --include=dev` to ensure devDependencies are installed
- Uses `build:persistent` script optimized for Render

## Why Netlify CLI is Required on Render

Even in persistent mode on Render, KNIRVGATEWAY needs netlify-cli because:

1. **Netlify Functions**: The application uses `netlify/functions/` routes for serverless functionality
2. **SSE Support**: Server-Sent Events are implemented using netlify-cli for compatibility
3. **API Gateway**: Many API endpoints are routed through netlify functions
4. **Development Consistency**: Maintains compatibility between local development and deployment

## Usage

### Manual Fix
```bash
npm run ensure-netlify-cli
```

### Automatic Fix (during build)
The build process now automatically ensures netlify-cli is available:
```bash
npm run build
# or for Render deployment
npm run build:persistent
```

### Health Check
The health check script validates netlify-cli installation:
```bash
npm run check-health
```

## Troubleshooting

If netlify-cli issues persist on Render:

1. **Check Installation**:
   ```bash
   npm run ensure-netlify-cli
   ```

2. **Manual Clean Install**:
   ```bash
   npm uninstall netlify-cli
   npm cache clean --force
   npm install netlify-cli@21.6.0 --save-dev
   ```

3. **Verify Working**:
   ```bash
   npx netlify --version
   ```

4. **Check Health**:
   ```bash
   npm run check-health
   ```

## Files Modified

- `scripts/ensure-netlify-cli.js` - New automated ensurer script
- `scripts/check-health.js` - Always checks netlify-cli (no skipping in persistent mode)
- `package.json` - Updated build scripts and dependencies
- `render.yaml` - Updated build command for Render deployment

## Version History

- **v1.0**: Initial netlify-cli dependency management
- **v1.1**: Added automated ensurer script
- **v1.2**: Integrated with build process and Render deployment
- **v1.3**: Fixed persistent mode to still require netlify-cli (2025-01-09)


## NETLIFY DEPENDENCY FIX

# Netlify Function Dependencies Fix

## Problem

The Netlify deploy was failing with the error:
```
A Netlify Function is using "@supabase/supabase-js" but that dependency has not been installed yet.
```

This occurred because Netlify Functions with their own `package.json` files don't automatically install dependencies during the build process.

## Root Cause

1. **Function-specific dependencies**: The `netlify/functions/package.json` file contains dependencies like `@supabase/supabase-js` that are needed by the Discourse forum functions.

2. **Netlify build process**: By default, Netlify doesn't automatically install dependencies for functions that have their own `package.json` files.

3. **Conditional imports**: Even though the Supabase import was conditional (only when `DB_TYPE=supabase`), Netlify's bundling process still tried to resolve the dependency.

## Solutions Implemented

### 1. Added Netlify Plugin

**File**: `netlify.toml`
```toml
[[plugins]]
  package = "@netlify/plugin-functions-install-core"
```

This plugin automatically installs dependencies for Netlify Functions that have their own `package.json` files.

### 2. Added Dependencies to Main Package.json

**File**: `package.json`
```json
{
  "dependencies": {
    "@supabase/supabase-js": "^2.39.0",
    "formidable": "^3.5.1",
    "jsonwebtoken": "^9.0.2",
    "nodemailer": "^6.9.8",
    "uuid": "^9.0.1",
    "mime-types": "^2.1.35",
    "sharp": "^0.33.0",
    "markdown-it": "^14.0.0",
    "dompurify": "^3.0.7",
    "jsdom": "^23.0.1"
  }
}
```

This ensures that all function dependencies are available at the project root level as a fallback.

### 3. Enhanced Build Process

**File**: `package.json`
```json
{
  "scripts": {
    "build": "npm run check-health && npm install && npm run check-function-deps --fix && npm run build:nexus && echo 'KNIRV Gateway built with Netlify Functions support'",
    "install-function-deps": "cd netlify/functions && npm install",
    "check-function-deps": "node scripts/check-function-deps.js"
  }
}
```

**File**: `netlify.toml`
```toml
[build]
  command = "npm run smart-build-with-apps && cd netlify/functions && npm install"
```

### 4. Improved Error Handling

**File**: `netlify/functions/discourse-utils.js`
```javascript
if (this.dbType === 'supabase') {
    try {
        const { createClient } = require('@supabase/supabase-js');
        this.supabaseClient = createClient(
            config.get('SUPABASE_URL'),
            config.get('SUPABASE_ANON_KEY')
        );
    } catch (error) {
        console.warn('Supabase client not available, falling back to JSON database');
        this.dbType = 'json';
        this.supabaseClient = null;
    }
}
```

This gracefully handles cases where Supabase dependencies aren't available and falls back to JSON database mode.

### 5. Dependency Checker Script

**File**: `scripts/check-function-deps.js`

A comprehensive script that:
- Verifies function `package.json` exists and has required dependencies
- Checks that `node_modules` contains required packages
- Tests that Supabase can be imported successfully
- Provides auto-fix functionality
- Gives detailed error messages and suggestions

## Testing the Fix

### Local Testing
```bash
# Check function dependencies
npm run check-function-deps

# Install function dependencies manually
npm run install-function-deps

# Run full build process
npm run build
```

### Netlify Deploy Testing
The build process now includes:
1. Health checks
2. Main dependency installation
3. Function dependency verification and installation
4. NEXUS portal build
5. Comprehensive error reporting

## Verification Steps

1. **Check Plugin Installation**:
   ```bash
   npm list @netlify/plugin-functions-install-core
   ```

2. **Verify Function Dependencies**:
   ```bash
   cd netlify/functions
   npm list @supabase/supabase-js
   ```

3. **Test Function Import**:
   ```bash
   node -e "const { createClient } = require('./netlify/functions/node_modules/@supabase/supabase-js'); console.log('Success');"
   ```

## Fallback Options

If the primary solutions don't work, these alternatives are available:

### Option A: Remove Function Package.json
Remove `netlify/functions/package.json` and rely entirely on main `package.json`.

### Option B: Manual Installation in Build
Add explicit installation commands to the build process:
```bash
cd netlify/functions && npm install && cd ../..
```

### Option C: Environment Variable Override
Set `DB_TYPE=json` to force JSON database mode and avoid Supabase entirely.

## Monitoring

The build process now includes comprehensive logging:
- Dependency installation status
- Function health checks
- Error reporting with suggested fixes
- Performance metrics

## Future Improvements

1. **Dependency Optimization**: Consider consolidating dependencies to reduce bundle size
2. **Caching**: Implement dependency caching for faster builds
3. **Testing**: Add automated tests for function dependencies
4. **Documentation**: Maintain up-to-date dependency documentation

## Related Files

- `netlify.toml` - Netlify configuration with plugin
- `package.json` - Main project dependencies and scripts
- `netlify/functions/package.json` - Function-specific dependencies
- `netlify/functions/discourse-utils.js` - Database abstraction with error handling
- `scripts/check-function-deps.js` - Dependency verification script

## Support

If you encounter dependency issues:

1. Run `npm run check-function-deps` for diagnostics
2. Check the build logs for specific error messages
3. Verify that all required dependencies are listed in both package.json files
4. Ensure the Netlify plugin is properly configured
5. Test locally with `netlify dev` before deploying


## PRIVATE DHT README

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
