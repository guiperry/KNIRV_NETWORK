# ERGO: Error Resolution Gamming Operation

[![Node.js Version](https://img.shields.io/badge/node-%3E%3D20.0.0-brightgreen)](https://nodejs.org/)
[![Rust Version](https://img.shields.io/badge/rust-%3E%3D1.70.0-orange)](https://www.rust-lang.org/)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.8.3-blue)](https://www.typescriptlang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## Overview

The **ERGO** serves as a comprehensive error resolution management platform. It unifies separate tools into a cohesive application that provides seamless error management, skill development, wallet functionality, and network interaction capabilities.

## 🆕 Recent Updates

### 🔮 Adaline Cognitive Engine Integration (Latest)

The ERGO now features advanced cognitive processing through the Adaline Gateway architecture:

#### **Multi-Model LLM Gateway**
- **Provider Agnostic**: Unified interface for Gemini, OpenAI, DeepSeek, and other LLM providers
- **Fallback Routing**: Automatic failover between providers for reliability
- **Confidence-Based Routing**: Intelligent routing based on model confidence scores

#### **Advanced Agent Behaviors**
- **Anchor Datasets**: Behavioral consistency through few-shot context injection
  - Pre-defined templates for error resolution, combat, exploration, dialogue, crafting, and NPC interactions
  - Automatic context population with historical data derivation
  - Template matching and scoring for optimal behavior selection
- **Field Noise Remediation**: Sabotage detection and cleanup
  - Noise injection detection via entropy analysis
  - Prompt injection pattern matching
  - Context poisoning detection
  - Adversarial drift identification
  - Language model-based filtering

#### **DVE/CDE Validation Pipeline**
- **DVE (Deterministic Validation Environment)**: Output validation through simulation
  - Skill code execution with failure context
  - Validation score calculation
  - Configurable thresholds via environment variables
- **CDE (Code/Document Execution)**: Sandboxed solution validation
  - Multi-language code validation (JavaScript, Python, Go, Rust, C++)
  - Constraint satisfaction checking
  - Severity-based violation handling
  - Safe sandboxed execution

#### **KNIRVSERVER Integration**
- HTTP client for partner KNIRVSERVER backend services
- DVE task management and node querying
- CDE environment lifecycle management
- Cognitive metrics reporting
- Guardrail violation tracking

#### **Key Files**
- `src/sensory-shell/AdalineBridge.ts` - Multi-model LLM gateway bridge
- `src/sensory-shell/AnchorDatasetManager.ts` - Behavioral template management
- `src/sensory-shell/DenoisingService.ts` - Field noise remediation
- `src/services/KNIRVSERVERClient.ts` - Backend service client

### 🧠 Chat-Brain: Personal Memory AI
- **Universal LLM Chat Interface**: Unified chat interface for all major LLMs (Gemini, OpenAI, DeepSeek, Adaline)
- **Persistent Memory Layer**: KNIRVGRAPH-powered knowledge graph that captures and synthesizes every conversation
- **Multi-LLM Support**: Seamlessly switch between different AI providers while maintaining conversation context
- **Memory Graph Visualization**: Interactive Cytoscape-based graph showing relationships between concepts, entities, and conversations
- **Notes & Knowledge Management**: Markdown-powered note-taking with automatic indexing and retrieval
- **Contextual Intelligence**: Builds deep understanding of user knowledge, personality, and preferences over time
- **Digital Memory Clone**: Creates an evolving representation of your thinking patterns and information needs

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

The inferface integrates four core components into a unified platform:

#### 1. **Receiver** (Primary Interface)
- **Location**: `src/components/KnirvShell.tsx`
- **Purpose**: Primary user interface with cognitive shell integration
- **Features**:
  - Voice command processing
  - Screenshot capture and analysis
  - NRV (Neural Response Vector) visualization
  - Real-time cognitive state management
  - Error, Context, and Idea submission workflows
  - Visual cognitive mode indicator (lightning icon fills when active)
  - Dynamic status updates (idle → monitoring → processing)

#### 2. **Manager** (Neural Intelligence Model Lifecycle Management)
- **Location**: `src/pages/` (Skills, UDC, Wallet)
- **Purpose**: Evolved mobile-controller for comprehensive NIM management
- **Features**:
  - Neural Intelligence Model registration and deployment
  - LoRA adapter skill management
  - UDC (User Delegation Certificate) management
  - Network connectivity monitoring
  - Performance analytics
  - Model creation and training with external AI integration
  - Sample starter NIM with default cortex.wasm
  - External API configuration and management
  - Chat-Brain Personal Memory AI interface

#### 2.1 **Chat-Brain** (Personal Memory AI)
- **Location**: `src/pages/ChatBrain.tsx`
- **Purpose**: Universal LLM interface with persistent memory and knowledge graph
- **Features**:
  - **Multi-LLM Chat**: Unified interface for Gemini, OpenAI, DeepSeek, Adaline
  - **Memory Graph**: KNIRVGRAPH-powered knowledge graph with Cytoscape visualization
  - **Persistent Context**: Conversations stored and synthesized across sessions
  - **Smart Notes**: Markdown note-taking with automatic knowledge extraction
  - **Contextual Learning**: Builds deep understanding of user patterns and preferences
  - **LLM Provider Switching**: Seamlessly switch models while preserving context
  - **Semantic Search**: Vector-based memory search for relevant past conversations

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
- **QR Code Scanning**: Seamless integration with KNIRVHUB and KNIRVSERVER
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
- **Database**: NebulaDB
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
git clone https://github.com/guiperry/ERGO.git
cd ERGO

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
ERGO/
├── src/
│   ├── components/          # React components
│   │   ├── KnirvShell.tsx   # Main interface component
│   │   ├── CognitiveShellInterface.tsx
│   │   ├── chat-brain/      # Chat-Brain components
│   │   │   ├── ChatInterface.tsx
│   │   │   ├── MemoryGraphView.tsx
│   │   │   ├── NotesPanel.tsx
│   │   │   └── LLMSelector.tsx
│   │   └── ...
│   ├── pages/               # Route components (Manager interface)
│   │   ├── Skills.tsx
│   │   ├── Wallet.tsx
│   │   ├── ChatBrain.tsx    # Personal Memory AI page
│   │   └── ...
│   ├── services/            # Business logic services
│   │   ├── ApiKeyService.ts
│   │   ├── KnirvanaBridgeService.ts
│   │   ├── knirvGraphService.ts    # KNIRVGRAPH integration
│   │   ├── llmProviderService.ts   # Multi-LLM support
│   │   ├── chatBrainService.ts     # Chat-Brain logic
│   │   ├── KNIRVSERVERClient.ts    # DVE/CDE backend client
│   │   └── ...
│   ├── contexts/            # React Context providers
│   │   ├── ChatBrainContext.tsx    # Chat-Brain state
│   │   └── ...
│   ├── sensory-shell/       # WASM and cognitive processing
│   │   ├── CognitiveEngine.ts
│   │   ├── WASMOrchestrator.ts
│   │   ├── AdalineBridge.ts         # Multi-model LLM gateway
│   │   ├── AnchorDatasetManager.ts   # Behavioral templates
│   │   ├── DenoisingService.ts      # Field noise remediation
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
│       ├── chatBrain.ts    # Chat-Brain types
│       └── ...
├── tests/                  # Test suites
├── rust-wasm/             # Rust WASM modules
├── assembly/              # AssemblyScript WASM
└── docs/                  # Documentation
    └── SDD.md             # Software Design Document
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

# Core Network Configuration
VITE_API_BASE_URL=http://gateway-testnet.knirv.network
VITE_ORACLE_ENDPOINT=http://oracle-testnet.knirv.network
VITE_WALLET_CONNECT_PROJECT_ID=your_project_id
VITE_WALLET_CONNECT_RPC_URL=http://localhost:8545
VITE_XION_CHAIN_ID=local-1
VITE_KNIRV_GRAPH_ENDPOINT=https://graph-testnet.knirv.network

# Chat-Brain LLM Providers
VITE_GOOGLE_API_KEY=your-gemini-api-key
VITE_OPENAI_API_KEY=your-openai-api-key
VITE_DEEPSEEK_API_KEY=your-deepseek-api-key
VITE_ADALINE_KEY=your-adaline-api-key

# KNIRVGRAPH Configuration (for Chat-Brain memory)
VITE_KNIRVGRAPH_ENDPOINT=http://localhost:26657
VITE_KNIRVGRAPH_CHAIN_ID=knirvgraph-1
VITE_KNIRVGRAPH_API_KEY=your-api-key

# Adaline Cognitive Engine Configuration
VITE_KNIRVSERVER_URL=http://localhost:8082
VITE_DVE_VALIDATION_THRESHOLD=0.7
VITE_CDE_VALIDATION_ENABLED=true
```

### Network Configuration
The app supports multiple network environments:
- **Local Development**: Local testnet with mock services
- **Testnet**: KNIRV testnet deployment
- **Mainnet**: Production KNIRV network

## Authentication & User Management

### Database Setup & Seeding

The ERGO includes a comprehensive authentication system with pre-configured user accounts for testing and development.

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
docker build -t ERGO .

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
# KNIRV Controller - Android Installation

## Quick Install
1. Open this link on your Android device
2. Tap "Add to Home Screen" when prompted
3. The app will be installed like a native app

## Manual Installation
1. Open Chrome on your Android device
2. Navigate to the app URL
3. Tap the menu (⋮) and select "Add to Home Screen"
4. Confirm the installation

## Features
- Works offline
- Push notifications
- Native app experience
- Secure authentication
- Local data storage

## System Requirements
- Android 7.0 or later
- Chrome 70+ or compatible browser
- 50MB free storage space

## Technical Details
- Uses Vite for fast builds
- RxDB for reactive data management
- TailwindCSS for styling
- React Router v6 for routing
- Redux Toolkit Query for API requests

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

- **Issues**: [GitHub Issues](https://github.com/guiperry/ERGO/issues)
- **Discussions**: [GitHub Discussions](https://github.com/guiperry/ERGO/discussions)
- **Documentation**: [KNIRV Network Docs](https://docs.knirv.network)

---

**Built with ❤️ for the KNIRV D-TEN ecosystem**
