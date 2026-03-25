# KNIRVGATEWAY: Unified Web Portal and API Gateway

[![Netlify Status](https://api.netlify.com/api/v1/badges/your-badge-id/deploy-status)](https://app.netlify.com/sites/your-site-name/deploys)
[![Node.js Version](https://img.shields.io/badge/node-%3E%3D18.0.0-brightgreen)](https://nodejs.org/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

## Table of Contents

- [Overview](#overview)
- [Key Features](#key-features)
- [Architecture](#architecture)
- [Quick Start](#quick-start)
- [API Gateway Functionality](#api-oracle-functionality)
- [Server-Sent Events (SSE)](#server-sent-events-sse)
- [WebGUI Development](#webgui-development)
- [WebGUI Authentication Testing](#webgui-authentication-testing)
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
- [Download Configuration](#download-configuration)
- [Netlify CLI Fix](#netlify-cli-fix)
- [Netlify Dependency Fix](#netlify-dependency-fix)
- [Private DHT Implementation](#private-dht-implementation)
- [DHT Control API](#dht-control-api)


## Overview

KNIRVGATEWAY serves as the primary web portal and API oracle for the KNIRV D-TEN (Decentralized Trusted Execution Network). It combines a modern, responsive website with serverless API oracle functionality, providing a unified entry point for users, developers, and services within the KNIRV ecosystem. Migrated from a Go-based WebSocket oracle, it now utilizes Netlify Functions and Server-Sent Events (SSE) for improved performance and browser compatibility.  The system has undergone a major update in 2025, shifting from an "Agent"-centric to a "Model"-centric terminology and featuring a modernized WebGUI.

### Recent Major Updates (2025)

- **Model-Centric Terminology**: Comprehensive update from "Agent" to "Model" training across all interfaces
- **WebGUI Modernization**: Complete overhaul with hierarchical navigation and improved UX
- **Network Management**: Multi-network support with dynamic switching capabilities
- **Payment Gateway**: Updated testnet token configuration (NRV instead of tNRV)
- **Authentication Streamlining**: Simplified auth flow for better developer experience
- **Footer Migration**: Dynamic footer system migrated from static HTML to React components
- **Testing Infrastructure**: Comprehensive Jest testing suite with React Testing Library


## Key Features

- 🌐 **Modern Web Portal**: Responsive website showcasing KNIRV D-TEN capabilities
- 🚀 **Serverless API Gateway**: Netlify Functions-based oracle with SSE support
- 🧪 **KNIRVTESTNET Integration**: Live testnet frontend with AWS backend connectivity
- 👨‍💻 **Developer Portal**: Comprehensive documentation and tools for KNIRV developers
- 📚 **Documentation Hub**: Integrated documentation with Docsify
- 🔄 **Real-time Updates**: Server-Sent Events (SSE) for live data streaming
- 🔐 **Authentication**: Secure API access and user management (JWT-based, configurable)
- 📊 **Health Monitoring**: Real-time service health and metrics
- 🎨 **Glass Morphism UI**: Modern design with balanced blue/purple color scheme
- 🗣️ **Social Media Optimization**: Platform-specific content and meta tags for enhanced sharing
- ⚙️ **Centralized Configuration Management**: YAML-based configuration for links, settings, and feature flags
- 🔄 **Model-Centric Terminology**: Updated from "Agent" to "Model" training throughout the system
- 🏗️ **Hierarchical Navigation**: Expandable/collapsible navigation structure in WebGUI
- 🌐 **Multi-Network Support**: Switch between mainnet, testnet, local, and private networks
- 📱 **KNIRVCONTROLLER Integration**: QR code connection and mobile app integration
- 🔧 **Comprehensive Testing**: Jest-based testing suite with React Testing Library
- ⬇️ **Centralized Download Management**: Configuration-driven download links for all products.
- 🔄 **Private DHT Implementation**: libp2p-based Private DHT with multi-platform support and automated failover.
- 🎛️ **DHT Control API**: REST API endpoints for on-demand DHT initialization and control.


## Architecture

### Web Portal Components

```
KNIRVGATEWAY/
├── index.html                    # Main website entry point
├── assets/                       # Static assets (CSS, JS, images)
├── images/                       # Website images and branding
├── developer-portal/       # Developer documentation and tools
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
├── oracle-sse.js               # Main API oracle with SSE support
└── health-monitor.js            # Service health monitoring
```

### Configuration

```
├── netlify.toml                 # Netlify deployment configuration
├── package.json                 # Node.js dependencies and scripts
├── config/                       # Configuration files (YAML and JSON)
│   └── portal-links.yaml        # Centralized configuration
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
   - API Gateway: http://localhost:8888/oracle/*
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
- `GET /oracle/health` - Gateway health status
- `GET /oracle/services` - Available services list
- `GET /oracle/metrics` - Performance metrics
- `GET /oracle/events` - SSE event stream

#### Health Monitoring
- `GET /health-monitor/status` - Service health status
- `GET /health-monitor/events` - SSE health updates

#### Authentication
- `POST /auth/login` - User authentication
- `POST /auth/logout` - User logout
- `GET /auth/verify` - Token verification

#### Service Proxy
- `/api/*` - Proxy to KNIRVGATEWAY services
- `/economics/*` - Economics service endpoints
- `/tunnel/*` - Tunnel registry endpoints

#### DHT Control (See dedicated section below)
- `GET /health` - Gateway and DHT health check
- `POST /dht/start` - Start DHT network
- `POST /dht/stop` - Stop DHT network
- `GET /dht/restart` - Restart DHT network
- `GET /dht/status` - Get DHT network status


## Server-Sent Events (SSE)

The oracle supports real-time updates via SSE:

```javascript
// Connect to oracle events
const eventSource = new EventSource('/oracle/events');
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


## WebGUI Development

The WebGUI service (`services/webgui`) provides a modern React-based interface:

```bash
cd services/webgui

# Install dependencies
npm install

# Start development server
npm run dev

# Run tests
npm test

# Run tests with coverage
npm test -- --coverage

# Build for production
npm run build
```

#### WebGUI Features

- **Hierarchical Navigation**: Expandable/collapsible menu structure
- **Network Switching**: Dynamic backend configuration for different networks
- **Model Management**: Comprehensive model building, training, and deployment tools
- **Marketplace Integration**: Skills, capabilities, and properties trading
- **Personal Vault**: User asset management (models, wallets, skills)
- **Real-time Monitoring**: Network health and performance metrics
- **KNIRVCONTROLLER Integration**: QR code connection for mobile apps


## WebGUI Authentication Testing

The WebGUI includes comprehensive authentication testing tools:

### Role Switcher (Top-Right Corner)

- Switch between Root, Bootnode, Dev, and General roles instantly.
- Change network contexts (Mainnet, Public Testnet, Private Testnet, Demo).
- View current authentication state.
- Clear authentication to test the login screen.

### Authentication Testing Page ("Auth Testing" in sidebar)

- Real-time authentication status display.
- Page access testing for the current role.
- Role permission comparison matrix.
- Authentication history tracking.
- Comprehensive testing instructions.

### Testing Different Roles

Follow the instructions in the Role Switcher to test Root, Bootnode, Developer, and General roles.  Expected access is detailed for each role.

### Testing Login Screen

Use either the "Clear Auth & Show Login" button in the Role Switcher or manually clear localStorage items (`knirv_user_role`, `knirv_network`, `knirv_auth_token`, `knirv_demo_mode`) to test the login screen.

### Testing Page Access Controls

Use the Auth Testing page or manually navigate to restricted pages to test access controls. A Role Permission Matrix is provided:

| Page | Root | Bootnode | Dev | General |
|------|------|----------|-----|---------|
| Dashboard | ✅ | ✅ | ✅ | ✅ |
| Inventory | ✅ | ✅ | ✅ | ✅ |
| DEX | ✅ | ✅ | ✅ | ✅ |
| NFT Capability Manager | ✅ | ✅ | ✅ | ✅ |
| Capabilities | ✅ | ✅ | ✅ | ✅ |
| Auth Testing | ✅ | ✅ | ✅ | ✅ |
| Vault | ✅ | ✅ | ✅ | ❌ |
| Blockchain | ✅ | ✅ | ✅ | ❌ |
| NFT Vault | ✅ | ✅ | ✅ | ❌ |
| Add Capability | ✅ | ✅ | ✅ | ❌ |
| Explorer | ✅ | ✅ | ✅ | ❌ |
| DAOs | ✅ | ✅ | ❌ | ❌ |
| Settlement | ✅ | ✅ | ❌ | ❌ |
| Peers | ✅ | ✅ | ❌ | ❌ |
| Network Admin | ✅ | ❌ | ❌ | ❌ |

### Testing Network Contexts

Use the Role Switcher to change networks (Demo Mode, Private Testnet, Public Testnet, Mainnet) and observe the effects on available features.

### Troubleshooting

Refer to the troubleshooting section in the Authentication Testing Guide for common issues.


## API Key Testing Guide

This guide shows how to manually provision and test an API key for the Controller from the Gateway/WebGUI.

- Important: API key provisioning is not performed automatically by the QR Connect flow. The QR page stores a controller base URL but does not create or exchange an API key.

### 1) Prerequisites
- Ensure KNIRVCONTROLLER is running and reachable (e.g., http://localhost:3000 or your remote host).
- Have an admin-scoped API key to create additional keys. Admin endpoints require the 'admin:all' permission.

### 2) Create a test API key (admin only)
Use your admin key to create a scoped, read-only key for Vault endpoints.

```bash
# Replace values as needed
CONTROLLER_BASE="http://localhost:3000"
ADMIN_KEY="<your-admin-key>"

curl -sS -X POST "$CONTROLLER_BASE/api/keys" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $ADMIN_KEY" \
  -d '{
    "name": "webgui-readonly",
    "description": "Read-only WebGUI testing key",
    "permissions": ["read:skills","read:capabilities","read:properties"]
  }'
```

- The response includes `apiKey.key`. Copy this value as TEST_KEY.

### 3) Configure the WebGUI client to use the key
Choose one:

- Option A: Environment variable (recommended for local dev)
  - In `services/webgui/.env.local` set:
    ```bash
    NEXT_PUBLIC_BACKEND_URL="http://localhost:3000"
    NEXT_PUBLIC_CONTROLLER_API_KEY="<TEST_KEY>"
    ```

- Option B: Browser session storage (quick test)
  - In the browser DevTools Console on the WebGUI origin:
    ```javascript
    localStorage.setItem('controller_api_key', '<TEST_KEY>');
    ```

Notes:
- The WebGUI axios client sends the key via `X-API-Key` header when either `NEXT_PUBLIC_CONTROLLER_API_KEY` or `localStorage('controller_api_key')` is present.
- To set the controller base URL without rebuilding, use the QR Connect page to save it via `/session/controller`; for axios-based calls, `NEXT_PUBLIC_BACKEND_URL` controls the base URL.

### 4) Test Vault endpoints
- From terminal:
  ```bash
  CONTROLLER_BASE="http://localhost:3000"
  TEST_KEY="<TEST_KEY>"

  curl -sS "$CONTROLLER_BASE/api/skills" -H "X-API-Key: $TEST_KEY" | jq .
  curl -sS "$CONTROLLER_BASE/api/capabilities" -H "X-API-Key: $TEST_KEY" | jq .
  curl -sS "$CONTROLLER_BASE/api/properties" -H "X-API-Key: $TEST_KEY" | jq .
  ```

- From WebGUI: open Vault pages. If the key and URL are set correctly, data loads.

### 5) Troubleshooting
- 401/403: Verify the key is set and has the required permissions.
- Network errors: Confirm `NEXT_PUBLIC_BACKEND_URL` points to the running Controller.
- Health check mismatch: the Controller returns `{ status: "healthy" }`, while some UI checks expect `"ok"`; treat `healthy` as running.

## Development

### Available Scripts

- `npm run dev` - Start development server with hot reload
- `npm run build` - Build for production
- `npm run deploy` - Deploy to Netlify
- `npm run functions:test` - Test oracle functions
- `npm run validate` - Validate Netlify Functions
- `npm run dht:control:start` - Start DHT (for testing)
- `npm run dht:control:stop` - Stop DHT (for testing)
- `npm run dht:control:restart` - Restart DHT (for testing)
- `npm run dht:control:status` - Get DHT status (for testing)
- `npm run ensure-netlify-cli` - Ensure netlify-cli is installed (for Render)
- `npm run check-health` - Check oracle and DHT health
- `npm run check-function-deps` - Check Netlify function dependencies
- `npm run install-function-deps` - Install Netlify function dependencies
- `npm run smart-build-with-apps` - Build with application support (for Netlify)
- `make deploy-render` - Deploy to Render
- `make deploy-netlify` - Deploy to Netlify
- `make deploy-vercel` - Deploy to Vercel
- `make sync-failover-page` - Sync failover page content
- `make update-submodules` - Update all submodules
- `make setup-submodules` - Setup submodules
- `make test-provision` - Test provision endpoints
- `make test-health` - Test health endpoints
- `make status` - Check system status
- `make ci-build` - Run full CI build
- `make clean` - Clean build artifacts
- `make clean-all` - Full clean including node_modules
- `make audit` - Security audit

### Local Development

1. **Start the oracle**:
   ```bash
   npm run dev
   ```

2. **Test API endpoints**:
   ```bash
   curl http://localhost:8888/oracle/health
   ```

3. **Test SSE functionality**:
   ```bash
   curl -N -H "Accept: text/event-stream" http://localhost:8888/oracle/events
   ```


## Environment Variables

Configure these environment variables for production:

```bash
# Service URLs
KNIRVGATEWAY_URL=https://root.knirv.com
KNIRVCHAIN_URL=https://chain.knirv.com
KNIRVGRAPH_URL=https://graph.knirv.com
KNIRVNEXUS_URL=https://nexus.knirv.com

# Authentication
JWT_SECRET=your-jwt-secret
AUTH_ENABLED=true

# Monitoring
HEALTH_CHECK_INTERVAL=30000
METRICS_ENABLED=true

# DHT Control
DISABLE_DHT=false
KNIRV_BOOTSTRAP_PEERS=peer1,peer2,peer3
DHT_PORT=4001
INTERNAL_API_KEY=your_api_key

# Render Deployment
GATEWAY_MODE=persistent
CLOUDFLARE_API_TOKEN=your_cloudflare_token
CLOUDFLARE_ZONE_ID=your_zone_id
INSTANCE_IP=auto_detected_or_manual

# Netlify/Vercel Deployment
GATEWAY_MODE=serverless
RENDER_GATEWAY_INTERNAL_API=https://render-url/internal/peers

# Supabase Database (for Discourse forum)
DB_TYPE=supabase
SUPABASE_URL=your_supabase_url
SUPABASE_ANON_KEY=your_supabase_anon_key
```


## Portal Sections

### Main Website
- **Hero Section**: KNIRV D-TEN overview and value proposition
- **Features**: Seven sovereign layers explanation
- **Technology**: Technical architecture and capabilities
- **Tokenomics**: NRN token information
- **Roadmap**: Development timeline and milestones

### Developer Portal
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
# Test oracle functionality
./scripts/test-oracle-migration.sh

# Test complete system
./scripts/validate-complete-migration.sh
```

### Integration Testing

```bash
# Run integration tests
./integration-tests/run_oracle_migration_tests.sh
```

### WebGUI Testing

The WebGUI uses Jest and React Testing Library.  See the WebGUI Development section for testing commands.  Test coverage details are in the Implementation Summary.


## Support

- **Documentation**: [docs.knirv.com](https://docs.knirv.com)
- **Developer Portal**: [portal.knirv.com](https://portal.knirv.com)
- **Community**: [community.knirv.com](https://community.knirv.com)
- **Issues**: [GitHub Issues](https://github.com/knirv/issues)


## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.


