# KNIRVWALLET

![Banner](browser-wallet/banner.png)

A comprehensive, multi-platform wallet ecosystem designed as the seamless gateway to the KNIRV Decentralized Trusted Execution Network (D-TEN). KNIRVWALLET provides unified access to NRN tokens, KNIRV-CORTEX agents, and the entire KNIRV ecosystem through intuitive, Web2-like experiences powered by XION's Meta Accounts.

## Table of Contents

- [🚀 Overview](#-overview)
- [✨ Key Features](#-key-features)
- [📱 Platform Implementations](#-platform-implementations)
    - [Agentic Wallet (Mobile)](#agentic-wallet-mobile)
    - [Browser Wallet (Web/Extension)](#browser-wallet-webextension)
- [🛠️ Installation & Setup](#-installation--setup)
    - [Prerequisites](#prerequisites)
    - [Quick Start](#quick-start)
    - [Configuration](#configuration)
        - [XION Integration](#xion-integration)
        - [Browser Extension Installation](#browser-extension-installation)
- [🏗️ Architecture](#-architecture)
    - [Unified Wallet Architecture](#unified-wallet-architecture)
    - [Cross-Platform Communication](#cross-platform-communication)
- [📚 Usage Examples](#-usage-examples)
    - [Creating a Wallet](#creating-a-wallet)
    - [NRN Token Operations](#nrn-token-operations)
    - [Agent Management](#agent-management)
- [🔗 KNIRV Ecosystem Integration](#-knirv-ecosystem-integration)
- [🧪 Testing](#-testing)
    - [Test Architecture](#test-architecture)
    - [Test Categories](#test-categories)
        - [Unit Tests](#unit-tests)
        - [Integration Tests](#integration-tests)
        - [End-to-End Tests](#end-to-end-tests)
    - [Test Utilities](#test-utilities)
    - [Running Tests](#running-tests)
    - [Test Configuration](#test-configuration)
    - [Test Data and Mocking](#test-data-and-mocking)
    - [Coverage and Reporting](#coverage-and-reporting)
    - [Continuous Integration](#continuous-integration)
    - [Best Practices](#best-practices)
    - [Debugging Tests](#debugging-tests)
    - [Performance Testing](#performance-testing)
    - [Security Testing](#security-testing)
    - [Troubleshooting](#troubleshooting)
    - [Contributing](#contributing-1)
- [🚀 Deployment](#-deployment)
    - [Mobile App Deployment](#mobile-app-deployment)
    - [Browser Extension Deployment](#browser-extension-deployment)
- [🤝 Contributing](#-contributing)
    - [Development Guidelines](#development-guidelines)
- [📄 License](#-license)
- [🆘 Support](#-support)


## 🚀 Overview

KNIRVWALLET consists of two complementary implementations that provide seamless integration between mobile and browser environments:

- **🔗 Agentic Wallet**: React Native mobile application with Go backend for native mobile experiences. Located at `agentic-wallet/`.
- **🌐 Browser Wallet**: Browser extension and web application for desktop and web-based interactions. Located at `browser-wallet/`.

Both implementations share core functionality while optimizing for their respective platforms, ensuring users can access their KNIRV assets and agents from any device.


## ✨ Key Features

### 🔐 Core Wallet Functionality
- Multi-Chain Support: BTC, ETH, Solana, and KNIRV-ORACLE blockchain
- NRN Token Management: Native support for KNIRV Network's NRN tokens
- XION Meta Accounts: Gasless transactions and Web2-like authentication
- Biometric Authentication: Secure, convenient access across platforms
- Hardware Wallet Support: Ledger integration for enhanced security

### 🤖 KNIRV Ecosystem Integration
- KNIRV-CORTEX Control: Manage and control your AI agents
- User Delegation Certificates (UDCs): Secure agent authorization
- NRV System Integration: Submit ErrorNodes and SkillNodes
- Economics Module: Skill registration and fee management
- Real-time Communication: WebSocket and SSE support

### 🌍 Cross-Platform Features
- QR Code Connectivity: Seamless connection between mobile and browser
- Unified Account Management: Consistent experience across platforms
- Progressive Web App: Browser wallet works offline
- Native Mobile Performance: Optimized React Native implementation


## 📱 Platform Implementations

### Agentic Wallet (Mobile)

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
- QR Code Pairing: Mobile app generates QR codes for browser connection
- WebSocket Bridge: Real-time synchronization between platforms
- Shared State Management: Consistent wallet state across devices
- Universal Deep Links: Seamless handoff between applications


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

- **[KNIRV-ORACLE](../KNIRVORACLE/)**: Foundational blockchain for NRN tokens
- **[KNIRVCHAIN](../KNIRVCHAIN/)**: Smart contract platform for Skills and Base LLMs  
- **[KNIRV-CORTEX](../KNIRVCORTEX/)**: AI agent framework
- **[KNIRV-NEXUS](../KNIRVNEXUS/)**: Distributed verification engine
- **[KNIRVGATEWAY](../KNIRVGATEWAY/)**: Unified API gateway
- **[KNIRVSDK](../KNIRVSDK/)**: Development tools and libraries


## 🧪 Testing

### Test Architecture
```
KNIRVWALLET/
├── tests/
│   ├── unit/                    # Unit tests
│   │   ├── browser-wallet/      # Browser extension tests
│   │   ├── react-native/        # Mobile app tests
│   │   └── go-backend/          # Backend service tests
│   ├── integration/             # Integration tests
│   │   ├── cross-platform-sync.test.go
│   │   └── xion-integration.test.go
│   └── e2e/                     # End-to-end tests
│       ├── wallet-workflow.test.js
│       ├── playwright.config.js
│       ├── global-setup.js
│       └── global-teardown.js
├── test-utils/                  # Shared test utilities
│   ├── test-data.ts
│   ├── wallet-test-utils.ts
│   ├── crypto-test-utils.ts
│   ├── xion-test-utils.ts
│   ├── mock-services.ts
│   └── test-helpers.ts
├── jest.config.js              # Jest configuration
└── scripts/
    └── run-tests.sh            # Test runner script
```

### Test Categories

#### 1. Unit Tests

##### Browser Wallet Module Tests
- **Location**: `tests/unit/browser-wallet/`
- **Framework**: Jest + Testing Library
- **Coverage**: Wallet core functionality, Transaction signing, Keyring management, Cryptographic operations

##### React Native Mobile Tests
- **Location**: `tests/unit/react-native/`
- **Framework**: Jest + React Native Testing Library
- **Coverage**: XION meta accounts, Mobile UI components, Wallet screens, Cross-platform functionality

##### Go Backend Tests
- **Location**: `tests/unit/go-backend/`
- **Framework**: Go testing + Testify
- **Coverage**: Multichain wallet service, XION integration, Wallet synchronization, API handlers

#### 2. Integration Tests

##### Cross-Platform Synchronization
- **File**: `tests/integration/cross-platform-sync.test.go`
- **Coverage**: QR code pairing, WebSocket communication, Wallet data synchronization, Session management

##### XION Blockchain Integration
- **File**: `tests/integration/xion-integration.test.go`
- **Coverage**: Meta account management, NRN token operations, Skill invocation, Gasless transactions, Faucet integration

#### 3. End-to-End Tests

##### Complete Wallet Workflow
- **File**: `tests/e2e/wallet-workflow.test.js`
- **Framework**: Playwright
- **Coverage**: Browser extension workflow, Mobile app workflow, Cross-platform synchronization, Error handling, Performance testing, Security validation

### Test Utilities

- **Shared Test Data**: `test-utils/test-data.ts` (Test mnemonics, private keys, addresses, transactions, XION data, wallet templates)
- **Wallet Test Utilities**: `test-utils/wallet-test-utils.ts` (Wallet factory, transaction validation, account/keyring utilities, mock services)
- **Cryptographic Test Utilities**: `test-utils/crypto-test-utils.ts` (Mnemonic validation, private key testing, encryption/decryption, signature verification, key derivation)
- **XION Test Utilities**: `test-utils/xion-test-utils.ts` (XION meta account mocking, transaction simulation, contract interaction, gasless transaction testing)


### Running Tests

### Prerequisites
```bash
# Install dependencies
cd KNIRVWALLET
npm install

# Install Go dependencies
cd agentic-wallet/go-backend
go mod download

# Install Playwright browsers
npx playwright install
```

### Running All Tests
```bash
# Run comprehensive test suite
./scripts/run-tests.sh

# Run with specific options
./scripts/run-tests.sh --unit-only
./scripts/run-tests.sh --integration-only
./scripts/run-tests.sh --e2e-only
./scripts/run-tests.sh --all --verbose
```

### Running Specific Test Categories

#### Unit Tests Only
```bash
# Browser wallet tests
cd browser-wallet
npm test

# React Native tests
cd agentic-wallet
npm test

# Go backend tests
cd agentic-wallet/go-backend
go test ./...
```

#### Integration Tests Only
```bash
cd integration-tests
go test -v ./...
```

#### End-to-End Tests Only
```bash
# All browsers
npx playwright test

# Specific browser
npx playwright test --project=chromium

# Mobile testing
npx playwright test --project="Mobile Chrome"

# Cross-platform tests
npx playwright test cross-platform
```

### Test Configuration

#### Jest Configuration
- **File**: `jest.config.js`
- **Features**: Multi-project setup, Coverage reporting, Custom matchers, Mock configurations, Parallel execution

#### Playwright Configuration
- **File**: `tests/e2e/playwright.config.js`
- **Features**: Multi-browser testing, Mobile device simulation, Cross-platform test projects, Video/screenshot capture, Test reporting

### Test Data and Mocking

- **Mock Services**: `test-utils/mock-services.ts` (Mock XION client, wallet manager, transaction service, synchronization service)
- **Test Data Management**: Deterministic test data, isolated environments, automatic cleanup, secure credential handling

### Coverage and Reporting

- **Coverage Targets**: Unit Tests (80% line coverage minimum), Integration Tests (critical path coverage), E2E Tests (user workflow coverage)
- **Report Generation**: `npm run test:coverage`, `npm run test:report`

### Continuous Integration

- **GitHub Actions Integration**: Example CI configuration provided in the original document.

### Best Practices

- **Writing Tests**: Descriptive names, Arrange-Act-Assert pattern, test isolation, mock external dependencies, test edge cases.
- **Test Maintenance**: Regular updates, flaky test management, performance monitoring, documentation.
- **Security Testing**: Sensitive data handling, input validation, authentication, authorization.

### Debugging Tests

- **Debug Configuration**: `npm run test:debug`, running specific test files, verbose output, watch mode.
- **Common Issues**: Timing issues, mock configuration, environment setup, data cleanup.

### Performance Testing

- **Load Testing**: Transaction throughput, concurrent user simulation, memory usage, response time.
- **Stress Testing**: High-volume transactions, resource exhaustion, recovery, scalability.

### Security Testing

- **Vulnerability Testing**: Input sanitization, authentication bypass, authorization escalation, data exposure.
- **Cryptographic Testing**: Key generation, signature verification, encryption/decryption, random number generation.

### Troubleshooting

- **Common Test Failures**: Service unavailability, network timeouts, mock failures, data conflicts.
- **Getting Help**: Check test logs, review screenshots/videos, consult documentation, contact development team.

### Contributing

#### Adding New Tests
1. Follow existing test patterns
2. Add appropriate test utilities
3. Update documentation
4. Ensure CI/CD integration

#### Test Review Process
1. Code review for test quality
2. Coverage impact assessment
3. Performance impact evaluation
4. Documentation updates


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

- **Documentation**: Check this README file and the comprehensive KNIRV documentation
- **Issues**: Open an issue on our GitHub repository with detailed reproduction steps
- **Community**: Join the KNIRV community for discussions, support, and collaboration
- **Security**: Report security issues privately to the maintainers following responsible disclosure
- **Development**: Contribute to the project by following our development guidelines

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- **XION Network**: For providing the Meta Accounts infrastructure that enables seamless Web2-like experiences
- **React Native Community**: For the robust mobile development framework
- **Web3 Community**: For the foundational technologies and standards
- **KNIRV Contributors**: For their dedication to building the decentralized future
- **Security Researchers**: For helping us maintain the highest security standards

## 📚 Additional Resources

### Documentation Links
- [KNIRV Network Documentation](../docs/)
- [XION Meta Accounts Documentation](https://docs.xion.burnt.com/)
- [React Native Documentation](https://reactnative.dev/docs/getting-started)
- [Web Extension API Documentation](https://developer.mozilla.org/en-US/docs/Mozilla/Add-ons/WebExtensions)

### Community Resources
- [KNIRV GitHub Organization](https://github.com/knirv-network)
- [KNIRV Discord Community](https://discord.gg/knirv)
- [KNIRV Twitter](https://twitter.com/knirvnetwork)
- [KNIRV Blog](https://blog.knirv.network)

### Developer Resources
- [KNIRV SDK Documentation](../KNIRVSDK/)
- [KNIRV API Reference](../docs/api/)
- [Development Setup Guide](../docs/development/)
- [Contributing Guidelines](../CONTRIBUTING.md)

## 🔮 Future Roadmap

### Upcoming Features
- **Multi-Chain Support**: Expand beyond XION to support additional blockchain networks
- **Advanced Agent Interactions**: Enhanced UI for complex agent workflows and automation
- **Social Features**: Wallet-to-wallet communication and social trading features
- **DeFi Integration**: Built-in DeFi protocols and yield farming opportunities
- **NFT Marketplace**: Integrated NFT trading and collection management
- **Hardware Wallet Support**: Integration with Ledger, Trezor, and other hardware wallets

### Long-term Vision
- **Decentralized Identity**: Self-sovereign identity management with verifiable credentials
- **Cross-Platform Sync**: Seamless synchronization across all devices and platforms
- **AI-Powered Insights**: Intelligent portfolio management and trading recommendations
- **Governance Participation**: Direct participation in KNIRV network governance
- **Enterprise Solutions**: Business-grade wallet solutions for organizations

## 🔐 Security Considerations

### Security Best Practices
- **Private Key Management**: Never share your private keys or seed phrases
- **Phishing Protection**: Always verify URLs and never enter credentials on suspicious sites
- **Regular Updates**: Keep your wallet software updated to the latest version
- **Backup Strategy**: Maintain secure backups of your seed phrases and important data
- **Network Security**: Use secure networks and avoid public WiFi for sensitive operations

### Security Features
- **Biometric Authentication**: Fingerprint and face recognition for mobile access
- **Hardware Security**: Secure enclave storage for sensitive cryptographic operations
- **Multi-Factor Authentication**: Additional security layers for high-value operations
- **Transaction Signing**: Secure transaction signing with user confirmation
- **Audit Trail**: Comprehensive logging of all wallet operations and transactions

## 🌟 Why Choose KNIRVWALLET?

### Unique Value Propositions
1. **Seamless Web2 Experience**: No complex seed phrases or gas fees for basic operations
2. **Unified Ecosystem Access**: Single wallet for all KNIRV network services and features
3. **Cross-Platform Consistency**: Identical experience across mobile and browser platforms
4. **Agent-First Design**: Built specifically for AI agent interactions and automation
5. **Enterprise Ready**: Scalable architecture suitable for both individual and business use
6. **Open Source**: Transparent, auditable, and community-driven development

### Competitive Advantages
- **XION Meta Accounts**: Leverage cutting-edge account abstraction technology
- **Native Agent Support**: First-class support for AI agent interactions
- **Unified Architecture**: Consistent API and experience across all platforms
- **Developer Friendly**: Comprehensive SDK and documentation for easy integration
- **Security First**: Multiple layers of security with regular audits and updates
- **Community Driven**: Open development process with community input and governance

---

**KNIRVWALLET** - *Your Gateway to the Decentralized Future*

*Built with ❤️ by the KNIRV Network Community*

*© 2024 KNIRV Network. All rights reserved.*