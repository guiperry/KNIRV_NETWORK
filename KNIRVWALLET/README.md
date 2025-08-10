# KNIRVWALLET

![Banner](browser-wallet/banner.png)

A comprehensive, multi-platform wallet ecosystem designed as the seamless gateway to the KNIRV Decentralized Trusted Execution Network (D-TEN). KNIRVWALLET provides unified access to NRN tokens, KNIRV-CORTEX agents, and the entire KNIRV ecosystem through intuitive, Web2-like experiences powered by XION's Meta Accounts.

## 🚀 Overview

KNIRVWALLET consists of two complementary implementations that provide seamless integration between mobile and browser environments:

- **🔗 Agentic Wallet**: React Native mobile application with Go backend for native mobile experiences
- **🌐 Browser Wallet**: Browser extension and web application for desktop and web-based interactions

Both implementations share core functionality while optimizing for their respective platforms, ensuring users can access their KNIRV assets and agents from any device.

## ✨ Key Features

### 🔐 Core Wallet Functionality
- **Multi-Chain Support**: BTC, ETH, Solana, and KNIRV-ROOT blockchain
- **NRN Token Management**: Native support for KNIRV Network's NRN tokens
- **XION Meta Accounts**: Gasless transactions and Web2-like authentication
- **Biometric Authentication**: Secure, convenient access across platforms
- **Hardware Wallet Support**: Ledger integration for enhanced security

### 🤖 KNIRV Ecosystem Integration
- **KNIRV-CORTEX Control**: Manage and control your AI agents
- **User Delegation Certificates (UDCs)**: Secure agent authorization
- **NRV System Integration**: Submit ErrorNodes and SkillNodes
- **Economics Module**: Skill registration and fee management
- **Real-time Communication**: WebSocket and SSE support

### 🌍 Cross-Platform Features
- **QR Code Connectivity**: Seamless connection between mobile and browser
- **Unified Account Management**: Consistent experience across platforms
- **Progressive Web App**: Browser wallet works offline
- **Native Mobile Performance**: Optimized React Native implementation

## 📱 Platform Implementations

### Agentic Wallet (Mobile)
**Location**: `agentic-wallet/`

A React Native application with Go backend providing:
- Native mobile performance and UX
- Expo-based development for rapid iteration
- XION Meta Account integration
- Go backend with WASM-based AI agent execution
- Android APK compilation support
- Biometric authentication
- Camera integration for QR codes

**Tech Stack**:
- React Native 0.79.1 with Expo 53.0.0
- Go 1.21+ backend with WASM runtime
- CosmJS for blockchain interactions
- PostgreSQL and Redis for data storage

### Browser Wallet (Web/Extension)
**Location**: `browser-wallet/`

A comprehensive browser extension and web application featuring:
- Chrome/Firefox extension support
- Progressive Web App capabilities
- Multi-workspace architecture
- Torus signin integration
- Advanced transaction management
- Web3 login support
- Airgap account support

**Tech Stack**:
- TypeScript 4.9.5 with React 18.2.0
- Webpack 5.90.3 build system
- Yarn workspaces for modular architecture
- Jest testing framework

## 🛠️ Installation & Setup

### Prerequisites
- Node.js 18.14.2+
- Go 1.21+ (for agentic wallet backend)
- PostgreSQL 13+ and Redis 6+ (for agentic wallet)
- Yarn package manager

### Quick Start

#### 1. Clone and Setup
```bash
git clone <repository-url>
cd KNIRVWALLET
```

#### 2. Browser Wallet Setup
```bash
cd browser-wallet
yarn install
yarn build
```

#### 3. Agentic Wallet Setup
```bash
cd agentic-wallet

# Install dependencies
yarn install

# Setup Go backend
cd go-backend
make deps
cp .env.example .env
# Edit .env with your configuration
make migrate-up

# Start development
cd ..
yarn dev
```

### Configuration

#### XION Integration
1. Copy environment configuration:
```bash
cp agentic-wallet/.env.example agentic-wallet/.env
```

2. Configure XION contract addresses:
```env
EXPO_PUBLIC_NRN_CONTRACT=xion1your_nrn_contract_address
EXPO_PUBLIC_FAUCET_CONTRACT=xion1your_faucet_contract_address
```

#### Browser Extension Installation
1. Build the extension:
```bash
cd browser-wallet
yarn build
```

2. Load in browser:
   - Chrome: Go to `chrome://extensions/`, enable Developer mode, click "Load unpacked", select `packages/knirvwallet-extension/dist`
   - Firefox: Go to `about:debugging`, click "This Firefox", click "Load Temporary Add-on", select manifest file

## 🏗️ Architecture

### Unified Wallet Architecture
```
KNIRVWALLET/
├── agentic-wallet/          # Mobile React Native App
│   ├── src/                 # React Native components
│   ├── go-backend/          # Go API server
│   ├── services/            # Blockchain services
│   └── components/          # UI components
├── browser-wallet/          # Browser Extension & Web App
│   ├── packages/
│   │   ├── knirvwallet-extension/    # Browser extension
│   │   ├── knirvwallet-module/       # Core wallet module
│   │   └── knirvwallet-torus-signin/ # Torus integration
│   └── scripts/             # Build and deployment scripts
└── shared/                  # Shared utilities (future)
```

### Cross-Platform Communication
- **QR Code Pairing**: Mobile app generates QR codes for browser connection
- **WebSocket Bridge**: Real-time synchronization between platforms
- **Shared State Management**: Consistent wallet state across devices
- **Universal Deep Links**: Seamless handoff between applications

## 📚 Usage Examples

### Creating a Wallet
```typescript
// Agentic Wallet (React Native)
import { WalletManager, getXionConfig } from './src/xion-meta-accounts';

const config = getXionConfig('testnet');
const walletManager = new WalletManager(config);
const wallet = await walletManager.createWallet('my-wallet');
```

### NRN Token Operations
```typescript
// Get balance
const balance = await wallet.getNRNBalance();

// Transfer tokens
const txHash = await wallet.transferNRN(recipientAddress, amount);

// Burn for skills
const txHash = await wallet.burnNRNForSkill(skillId, amount);
```

### Agent Management
```typescript
// Execute AI agent
const result = await agentService.executeAgent(agentId, parameters);

// Submit to NRV system
await nrvService.submitErrorNode(errorType, description);
await nrvService.submitSkillNode(skillType, capabilities);
```

## 🔗 KNIRV Ecosystem Integration

KNIRVWALLET seamlessly integrates with the broader KNIRV ecosystem:

- **[KNIRV-ROOT](../KNIRVROOT/)**: Foundational blockchain for NRN tokens
- **[KNIRVCHAIN](../KNIRVCHAIN/)**: Smart contract platform for Skills and Base LLMs  
- **[KNIRV-CORTEX](../KNIRVCORTEX/)**: AI agent framework
- **[KNIRV-NEXUS](../KNIRVNEXUS/)**: Distributed verification engine
- **[KNIRVGATEWAY](../KNIRVGATEWAY/)**: Unified API gateway
- **[KNIRVSDK](../KNIRVSDK/)**: Development tools and libraries

## 🧪 Testing

### Browser Wallet
```bash
cd browser-wallet
yarn test
```

### Agentic Wallet
```bash
cd agentic-wallet
yarn test

# Go backend tests
cd go-backend
make test
```

### Integration Testing
```bash
# Test cross-platform connectivity
yarn test:integration

# Test XION integration
yarn test:xion
```

## 🚀 Deployment

### Mobile App Deployment
```bash
cd agentic-wallet

# Build for iOS
expo build:ios

# Build for Android
expo build:android

# Or build Android APK with Go backend
cd go-backend
make android-build
```

### Browser Extension Deployment
```bash
cd browser-wallet
yarn build
# Upload to Chrome Web Store / Firefox Add-ons
```

## 🤝 Contributing

We welcome contributions to KNIRVWALLET! Please see our contributing guidelines:

1. **Fork the repository**
2. **Create a feature branch**: `git checkout -b feature/amazing-feature`
3. **Make your changes** with appropriate tests
4. **Follow code style**: Run `yarn lint` and `yarn fmt`
5. **Submit a pull request**

### Development Guidelines
- Follow TypeScript best practices
- Write comprehensive tests
- Document new features
- Ensure cross-platform compatibility
- Test on both mobile and browser implementations

## 📄 License

Copyright (c) 2025 KNIRV Network

This project is part of the KNIRV Network ecosystem. See LICENSE file for details.

## 🆘 Support

- **Documentation**: Check the individual README files in each implementation
- **Issues**: Open an issue on our GitHub repository
- **Community**: Join the KNIRV community for discussions and support
- **Security**: Report security issues privately to the maintainers

---

*KNIRVWALLET - Your gateway to the decentralized AI economy*
