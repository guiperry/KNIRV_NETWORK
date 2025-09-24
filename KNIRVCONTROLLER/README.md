# KNIRV-CONTROLLER: The Unified Neural Intelligence Model Management Platform

[![Node.js Version](https://img.shields.io/badge/node-%3E%3D20.0.0-brightgreen)](https://nodejs.org/)
[![Rust Version](https://img.shields.io/badge/rust-%3E%3D1.70.0-orange)](https://www.rust-lang.org/)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.8.3-blue)](https://www.typescriptlang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## Overview

The **KNIRV-CONTROLLER** serves as the comprehensive Neural Intelligence Model (NIM) management platform within the KNIRV D-TEN ecosystem. It unifies separate tools into a cohesive application that provides seamless NIM management, skill development, wallet functionality, and network interaction capabilities.

## 🆕 Recent Updates

### ✅ External AI Integration (Beta Phase)
- **Multi-Provider Support**: Integrated Google Gemini, Anthropic Claude, OpenAI ChatGPT-5, and Deepseek for inference during beta
- **Cognitive Engine Enhancement**: Updated cognitive shell orchestrator to route inference through external API channels
- **Model Creation Workflow**: Complete 5-step model creation and training page with external API configuration
- **Onboarding Sequence**: Guided setup for new users with cortex.wasm compilation and API key configuration
- **Neural Intelligence Model Management**: Enhanced NIM management page with sample starter NIM and navigation to model creation

### 🔧 Cognitive Engine Improvements
- **Real-time Status Updates**: Cognitive engine start/stop now properly updates UI state and status indicators
- **Visual Feedback**: Lightning icon fills when cognitive engine is active, status changes from "idle" to "monitoring/processing"
- **Error Monitoring**: Active error monitoring system with skill-based status display
- **State Management**: Improved state synchronization between cognitive engine and UI components

## Architecture

### Unified Component Structure

The CONTROLLER integrates four core components into a unified platform:

#### 1. **Receiver** (Primary Interface)
- **Location**: `src/components/KnirvShell.tsx`
- **Purpose**: Primary user interface with cognitive shell integration
- **Features**:
  - Voice command processing
  - Screenshot capture and analysis
  - NRV (Neural Response Vector) visualization
  - Real-time cognitive state management
  - Error, Context, and Idea submission workflows
  - **NEW**: Visual cognitive mode indicator (lightning icon fills when active)
  - **NEW**: Dynamic status updates (idle → monitoring → processing)

#### 2. **Manager** (Neural Intelligence Model Lifecycle Management)
- **Location**: `src/pages/` (Skills, UDC, Wallet, Badges)
- **Purpose**: Evolved mobile-controller for comprehensive NIM management
- **Features**:
  - Neural Intelligence Model registration and deployment
  - LoRA adapter skill management
  - UDC (User Delegation Certificate) management
  - Network connectivity monitoring
  - Performance analytics
  - **NEW**: Model creation and training with external AI integration
  - **NEW**: Sample starter NIM with default cortex.wasm
  - **NEW**: External API configuration and management

#### 3. **CLI** (Terminal Interface)
- **Location**: Integrated via sliding panels and terminal services
- **Purpose**: Command-line interface for advanced operations
- **Features**:
  - Neural Intelligence Model minting on the oracle
  - Network diagnostics
  - Direct blockchain interactions
  - Terminal command execution

#### 4. **Wallet** (Neural Intelligence Model's Treasury)
- **Location**: `src/services/KnirvWalletService.ts`
- **Purpose**: XION Meta Account-based wallet for autonomous NIM operations
- **Features**:
  - Gasless transactions via XION
  - NRN token management
  - UDC issuance and validation
  - Secure key management

### WASM Neural Intelligence Model Core System

#### Neural Intelligence Model Compilation Pipeline
- **TypeScript LoRA Compilation**: `src/core/wasm/WASMCompiler.ts`
- **AssemblyScript Integration**: `assembly/index.ts`
- **Rust WASM Support**: `rust-wasm/` directory
- **Cognitive Shell Orchestration**: `src/sensory-shell/`

#### Key Capabilities
- **Neural Intelligence Model Core Upload**: Upload and compile WASM files with LoRA adapters
- **LoRA-Enhanced Export**: Export NIMs with embedded neural network modifications
- **Primary Neural Intelligence Model Management**: Dynamic skill loading and cluster competition participation

## Core Features

### Neural Intelligence Model Management & LoRA Development
- **Complete Neural Intelligence Model Lifecycle**: Creation, training, deployment, and management
- **LoRA Adapter Skills**: Skills ARE LoRA adapters containing weights and biases
- **Cluster Competition**: Participate in KNIRVGRAPH error cluster competitions
- **UDC Management**: Precise NIM permission control

### Network Integration
- **Universal Connectivity**: Connect with all KNIRV network services
- **QR Code Scanning**: Seamless integration with KNIRVENGINE and KNIRVNEXUS
- **Primary Neural Intelligence Model Cloning**: Consistent behavior across platforms
- **Cross-Platform Synchronization**: Real-time NIM configuration sync

### Cognitive Processing
- **Factuality Slicing**: Evidence-based validation for Error submissions
- **Feasibility Slicing**: Market analysis for Idea submissions
- **HRM Reasoning**: Advanced cognitive processing with configurable depth
- **Adaptive Learning**: Continuous improvement through experience

### Economic Model
- **Intelligent NRN Optimization**: Dynamic token consumption based on task complexity
- **Inference-Time Scaling**: Adjustable reasoning depth with corresponding NRN costs
- **Gasless Transactions**: Seamless blockchain interactions without gas fees

## Technology Stack

### Frontend
- **Framework**: React 19.1.0 with TypeScript 5.8.3
- **Build Tool**: Vite 6.3.2
- **Styling**: Tailwind CSS 3.4.17
- **State Management**: RxDB for local persistence
- **UI Components**: Custom components with Lucide React icons

### Backend
- **Runtime**: Node.js 20.0.0+
- **Framework**: Express.js with TypeScript
- **Database**: RxDB (RxDB) with LokiJS adapter
- **WebSocket**: Native WebSocket for real-time communication
- **Authentication**: API key-based with rate limiting

### WASM & Compilation
- **AssemblyScript**: For high-performance WASM modules
- **Rust**: Alternative WASM compilation target
- **TypeScript Compiler**: Custom NIM compilation pipeline

### Blockchain Integration
- **XION Meta Accounts**: Gasless transaction support
- **CosmJS**: Cosmos SDK integration
- **QR Payment Service**: Cross-platform payment handling

### Development & Testing
- **Testing**: Jest with React Testing Library
- **E2E Testing**: Playwright
- **Linting**: ESLint with TypeScript support
- **Build Tools**: npm scripts with custom WASM compilation

## Installation & Setup

### Prerequisites
```bash
Node.js >= 20.0.0
Rust >= 1.70.0 (for WASM compilation)
npm or yarn package manager
```

### Installation
```bash
# Clone the repository
git clone https://github.com/guiperry/KNIRVCONTROLLER.git
cd KNIRVCONTROLLER

# Install dependencies
npm install

# Build WASM modules
npm run build:wasm

# Setup database with default accounts
npm run db:setup

# Start development server
npm run dev
```

**🔐 Default Login Credentials:**
- **Admin**: `admin@knirv.com` / `admin123`
- **Demo**: `demo@knirv.com` / `demo123`
- **Developer**: `dev@knirv.com` / `dev123`
- **Test User**: `test@example.com` / `test123`

### Development Commands
```bash
# Development with hot reload
npm run dev

# Build for production
npm run build

# Run tests
npm test

# Run E2E tests
npm run test:e2e

# Start backend server
npm run server

# Full development stack
npm run dev:full
```

## Project Structure

```
KNIRVCONTROLLER/
├── src/
│   ├── components/          # React components
│   │   ├── KnirvShell.tsx   # Main interface component
│   │   ├── CognitiveShellInterface.tsx
│   │   └── ...
│   ├── pages/               # Route components (Manager interface)
│   │   ├── Skills.tsx
│   │   ├── Wallet.tsx
│   │   └── ...
│   ├── services/            # Business logic services
│   │   ├── ApiKeyService.ts
│   │   ├── KnirvanaBridgeService.ts
│   │   └── ...
│   ├── sensory-shell/       # WASM and cognitive processing
│   │   ├── CognitiveEngine.ts
│   │   ├── WASMOrchestrator.ts
│   │   └── ...
│   ├── core/                # Core system components
│   │   ├── wasm/           # WASM compilation system
│   │   ├── knirvgraph/     # Graph management
│   │   └── ...
│   ├── slices/             # Data processing pipelines
│   │   ├── factualitySlice.ts
│   │   └── feasibilitySlice.ts
│   ├── server/             # Backend API server
│   │   └── api-server.ts
│   └── types/              # TypeScript type definitions
├── tests/                  # Test suites
├── rust-wasm/             # Rust WASM modules
├── assembly/              # AssemblyScript WASM
└── docs/                  # Documentation
```

## API Endpoints

### Core Endpoints
- `GET /health` - Health check
- `POST /api/graph/error` - Submit error for processing
- `POST /api/graph/context` - Submit context/server info
- `POST /api/graph/idea` - Submit new idea/concept
- `GET /api/nims` - List available NIMs
- `POST /api/nims/:id/deploy` - Deploy NIM
- `GET /api/wallet/balance` - Get wallet balance

### WebSocket Events
- `cognitive_state` - Real-time cognitive processing updates
- `nim_status` - Neural Intelligence Model deployment and execution status
- `network_status` - Network connectivity updates

## Configuration

### Environment Variables
```bash
# .env
VITE_API_BASE_URL=http://gateway-testnet.knirv.network
VITE_ORACLE_ENDPOINT=http://oracle-testnet.knirv.network
VITE_WALLET_CONNECT_PROJECT_ID=your_project_id
VITE_WALLET_CONNECT_RPC_URL=http://localhost:8545
VITE_XION_CHAIN_ID=local-1
VITE_KNIRV_GRAPH_ENDPOINT=https://graph-testnet.knirv.network
```

### Network Configuration
The app supports multiple network environments:
- **Local Development**: Local testnet with mock services
- **Testnet**: KNIRV testnet deployment
- **Mainnet**: Production KNIRV network

## Authentication & User Management

### Database Setup & Seeding

The KNIRVCONTROLLER includes a comprehensive authentication system with pre-configured user accounts for testing and development.

#### Initial Database Setup
```bash
# Setup database and seed with default accounts
npm run db:setup

# Or run individually:
npm run db:migrate  # Migrate to NebulaDB
npm run db:seed     # Seed with default accounts
```

#### Default User Accounts

The seeding process creates the following accounts:

| Account Type | Email | Password | Roles | Description |
|-------------|-------|----------|-------|-------------|
| **Admin** | `admin@knirv.com` | `admin123` | `admin`, `user` | Full system administration access |
| **Demo** | `demo@knirv.com` | `demo123` | `user` | Demo account for testing user features |
| **Developer** | `dev@knirv.com` | `dev123` | `developer`, `user` | Development and API testing account |
| **Test User** | `test@example.com` | `test123` | `user` | Basic user account for testing |

#### Authentication Features

- **Multi-Role System**: Users can have multiple roles (admin, developer, user)
- **Permission-Based Access**: Granular permissions for different system features
- **Device-Specific Storage**: User data stored locally on device
- **XION Wallet Integration**: Seamless blockchain wallet connection
- **Session Management**: Secure session handling with refresh tokens
- **Biometric Authentication**: Support for device biometric authentication (PWA)

#### API Key Management

Each user account automatically receives API keys for system integration:

```bash
# View API keys after seeding
npm run db:seed
# API keys will be displayed in the seeding output
```

#### User Permissions

**Admin Permissions:**
- `admin:all` - Full administrative access
- `user:manage` - User account management
- `system:configure` - System configuration
- `deployment:manage` - Deployment management

**Developer Permissions:**
- `api:create` - Create new API keys
- `api:manage` - Manage API configurations
- `deployment:test` - Test deployment access

**User Permissions:**
- `profile:read` - Read user profile
- `profile:update` - Update user profile
- `wallet:access` - Access wallet functionality

### PWA Authentication

For Progressive Web App deployments, authentication includes:

- **Offline Capability**: Authentication works offline with cached credentials
- **Device Registration**: Each device gets a unique identifier
- **Secure Storage**: Credentials stored in device secure storage
- **Auto-Login**: Remember user sessions across app launches
- **Wallet Integration**: Seamless XION Meta Account connection

### Security Features

- **Password Hashing**: PBKDF2 with salt for secure password storage
- **Session Tokens**: JWT-based session management
- **Device Fingerprinting**: Device-specific security measures
- **Rate Limiting**: API rate limiting for security
- **Audit Logging**: Complete authentication audit trail

## Deployment

### Docker Deployment
```bash
# Build Docker image
docker build -t knirv-controller .

# Run with docker-compose
docker-compose up -d
```

### Production Build
```bash
# Build optimized production bundle
npm run build

# Start production server
npm run start:production
```

### Kubernetes Deployment
See the main KNIRV_NETWORK repository for Kubernetes manifests and deployment scripts.

## Testing

### Unit Tests
```bash
npm run test:unit
```

### Integration Tests
```bash
npm run test:integration
```

### E2E Tests
```bash
npm run test:e2e
```

### Test Coverage
```bash
npm run test:coverage
```

## Contributing

### Development Workflow
1. Fork the repository
2. Create a feature branch
3. Make changes with tests
4. Run the full test suite
5. Submit a pull request

### Code Standards
- TypeScript strict mode enabled
- ESLint configuration enforced
- Prettier for code formatting
- Comprehensive test coverage required

### WASM Development
- AssemblyScript for performance-critical modules
- Rust for complex cryptographic operations
- Automated build pipeline with CI/CD

## Security

### Key Security Features
- **API Key Authentication**: Secure API access with rate limiting
- **UDC Validation**: Cryptographic delegation certificates
- **Secure Key Storage**: Hardware-backed wallet key management
- **Input Validation**: Comprehensive request validation
- **Audit Logging**: Complete transaction and operation logging

### Wallet Security
- Non-custodial design
- XION Meta Accounts for Web2-like UX
- Gasless transactions
- Secure delegation via UDCs

## Performance

### Optimization Features
- **WASM Compilation**: High-performance NIM execution
- **Lazy Loading**: Component and route lazy loading
- **Caching**: Intelligent caching with RxDB
- **Memory Management**: Automatic memory optimization
- **Network Optimization**: Efficient API calls and WebSocket usage

### Monitoring
- **Performance Metrics**: Real-time performance monitoring
- **Error Tracking**: Sentry integration for error reporting
- **Analytics**: Usage analytics and behavioral insights

## Troubleshooting

### Common Issues
1. **WASM Build Failures**: Ensure Rust and AssemblyScript are properly installed
2. **Network Connectivity**: Check network configuration and API endpoints
3. **Wallet Connection**: Verify XION Meta Account setup
4. **Memory Issues**: Monitor WASM module memory usage

### Debug Mode
```bash
# Enable debug logging
DEBUG=* npm run dev

# View WASM compilation logs
npm run build:wasm -- --verbose
```

## Documentation

- **[Whitepaper](https://knirv.network/documentation/static/whitepapers/)**: Comprehensive technical specification
- **[API Documentation](https://knirv.network/documentation/static/knirvsdk/README)**: Detailed API endpoint documentation
- **[Gap Analysis](https://knirv.network/documentation/knirvcontroller/README)**: Current implementation status and roadmap

## License

MIT License - see [LICENSE](https://knirv.network/documentation/static/legal/TERMS_AND_CONDITIONS) file for details.

## Support

- **Issues**: [GitHub Issues](https://github.com/guiperry/KNIRVCONTROLLER/issues)
- **Discussions**: [GitHub Discussions](https://github.com/guiperry/KNIRVCONTROLLER/discussions)
- **Documentation**: [KNIRV Network Docs](https://docs.knirv.network)

---

**Built with ❤️ for the KNIRV D-TEN ecosystem**