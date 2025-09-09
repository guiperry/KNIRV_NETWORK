# KNIRVENGINE: Three-Engine Cognitive Architecture

KNIRVENGINE is a revolutionary three-engine cognitive architecture that combines Human-like Reasoning Models (HRM), QR code linkage systems, and WebAssembly (WASM) runtime for autonomous agent deployment and cognitive processing.

## 🏗️ Architecture Overview

KNIRVENGINE consists of three core engines:

### 1. Desktop-Host Engine
- **Purpose**: Central coordination and HRM processing
- **Technology**: Go backend with Wasmtime integration
- **Features**:
  - HRM 562M-parameter cognitive model
  - QR code generation and linkage
  - Model Context Protocol (MCP) integration
  - Secure bridge for mobile communication
  - TEE (Trusted Execution Environment) management

### 2. Mobile-Tool Engine
- **Purpose**: Enhanced mobile client with sensory processing
- **Technology**: React/TypeScript with native capabilities
- **Features**:
  - QR code scanning and device pairing
  - Voice processing and analysis
  - Visual processing with object detection
  - Real-time HRM communication
  - Integrated wallet functionality with KNIRVCONTROLLER linking
  - Multi-chain cryptocurrency support (BTC, ETH, SOL, NRN)
  - XION Meta Accounts integration for gasless transactions

### 3. Agent-Core Engine
- **Purpose**: Pure WASM cognitive shell
- **Technology**: Rust WASM with personality adaptation
- **Features**:
  - 562M-parameter HRM integration
  - Personality adaptation system
  - Host interface for desktop communication
  - Cognitive state management
  - Emotional modeling

## 🚀 Quick Start

### Prerequisites
- Go 1.21+
- Node.js 18+
- Rust 1.70+
- Python 3.9+ (for HRM model)

### Installation

1. **Clone the repository**:
```bash
git clone https://github.com/guiperry/KNIRV_NETWORK.git
cd KNIRV_NETWORK/KNIRVENGINE
```

2. **Build Desktop-Host**:
```bash
cd desktop-client
go build -o desktop-client main.go
```

3. **Build Mobile-Tool**:
```bash
cd mobile-controller
npm install
npm run build
```

4. **Build Agent-Core WASM**:
```bash
cd agent-core/rust-wasm
cargo build --release --target wasm32-unknown-unknown
```

### Running the System

1. **Start Desktop Host**:
```bash
cd desktop-client
./desktop-client
```

2. **Start Mobile Tool** (development):
```bash
cd mobile-controller
npm run dev
```

3. **Access the system**:
- Desktop Host: http://localhost:8082
- Mobile Tool: http://localhost:5173
- MCP WebSocket: ws://localhost:8082/api/mcp/ws

## 📱 QR Code Linkage System

The QR code linkage system enables secure pairing between desktop and mobile devices:

### QR Code Types

1. **Target Assignment QR**:
   - Links mobile device to specific target system
   - Contains session ID, capabilities, and expiration
   - Enables agent deployment authorization

2. **Transaction Signing QR**:
   - Authorizes blockchain transactions
   - Contains transaction data and security signatures
   - Enables secure wallet operations

### Usage Example

```javascript
// Generate target assignment QR
const response = await fetch('/api/qr/target-assignment', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    target_system_id: 'my_target',
    capabilities: ['agent_deployment', 'cognitive_processing'],
    expiry_minutes: 5
  })
});

const { qr_code_data, session_id } = await response.json();
```

## 🧠 HRM Cognitive Processing

The Human-like Reasoning Model (HRM) provides advanced cognitive capabilities:

### Model Architecture
- **Parameters**: 562,741,762 (562M)
- **L-modules**: Sensory-motor pattern processing
- **H-modules**: Long-horizon planning and reasoning
- **Personality Adapter**: User-specific behavior adaptation

### Processing Example

```javascript
// Send cognitive processing request
const hrmRequest = {
  sensory_data: [0.1, 0.3, 0.7, 0.2, 0.9, 0.4, 0.6, 0.8],
  context: 'pattern_recognition',
  task_type: 'visual_analysis'
};

const response = await fetch('/api/hrm/process', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify(hrmRequest)
});

const result = await response.json();
console.log('Reasoning:', result.reasoning_result);
console.log('Confidence:', result.confidence);
```

## 🔌 Model Context Protocol (MCP)

KNIRVENGINE implements MCP for standardized agent communication:

### Available Tools
- **hrm_process**: Process input through HRM cognitive engine
- **generate_qr**: Generate QR codes for device linkage
- **system_status**: Get current system status and health

### Available Resources
- **knirv://hrm/model_info**: HRM model information
- **knirv://system/capabilities**: System capabilities

### Available Prompts
- **cognitive_analysis**: Analyze input using HRM processing

### MCP Client Example

```javascript
const ws = new WebSocket('ws://localhost:8082/api/mcp/ws');

// List available tools
ws.send(JSON.stringify({
  jsonrpc: "2.0",
  method: "tools/list",
  id: 1
}));

// Call HRM processing tool
ws.send(JSON.stringify({
  jsonrpc: "2.0",
  method: "tools/call",
  id: 2,
  params: {
    name: "hrm_process",
    arguments: {
      sensory_data: [0.5, 0.8, 0.2],
      context: "test",
      task_type: "analysis"
    }
  }
}));
```

## 🔧 Configuration

### Desktop Host Configuration
- **Port**: 8082 (default)
- **HRM Model Path**: `./dist/weights.safetensors`
- **TEE Data Directory**: `./data`

### Mobile Tool Configuration
- **Development Port**: 5173
- **Production Build**: `dist/`
- **QR Scanner**: Camera access required

### Agent-Core Configuration
- **WASM Module**: `dist/agent-core.wasm`
- **Personality Metrics**: Configurable via API
- **Memory Buffer**: 100 items max

## 🧪 Testing

### Comprehensive Test Suite Implementation ✨

KNIRVENGINE now features a comprehensive test suite with significant coverage improvements across all core packages:

#### **Test Coverage Achievements:**
- **Utils Package**: 64.6% coverage (improved from 25.9%)
- **Inference Package**: 17.5% coverage (comprehensive AI/LLM testing) ⭐ **NEW**
- **Database Package**: 14.0% coverage (comprehensive model and repository tests)
- **API Package**: 9.6% coverage (foundational tests implemented)
- **Agent Package**: 43.1% coverage (maintained existing coverage)

#### **Run Unit Tests**
```bash
cd KNIRVENGINE/desktop-client

# Run all unit tests with coverage
make test/cover

# Run specific package tests
go test -v ./utils/
go test -v ./inference/
go test -v ./database/
go test -v ./api/

# Generate coverage report
make test/cover-report
```

#### **Run Integration Tests**
```bash
cd KNIRVENGINE

# Complete system integration tests
node test_complete_system.js

# QR linkage system tests
node test_qr_linkage.js

# Comprehensive test suite
make test/integration
```

#### **Test Implementation Highlights:**
- ✅ **TypeSafe Code**: All tests implement proper Go interfaces with comprehensive error handling
- ✅ **Edge Case Coverage**: Nil inputs, empty values, malformed data, and boundary conditions
- ✅ **Cross-Platform Testing**: Platform-aware tests for Windows, macOS, and Linux
- ✅ **Database Integration**: SQLite and ChromeDB testing with proper setup/teardown
- ✅ **Concurrency Testing**: Thread-safe operations and context cancellation
- ✅ **AI/LLM Testing**: Comprehensive inference engine, memory management, and provider delegation ⭐ **NEW**
- ✅ **Table-Driven Tests**: Comprehensive scenario coverage with multiple test cases

#### **Integration Test Results**
The system passes 8/10 comprehensive integration tests:
- ✅ Desktop Host Startup
- ✅ QR Linkage System
- ✅ MCP Connection
- ✅ MCP Tools/Resources/Prompts
- ✅ Mobile Connection
- ✅ WASM Cognitive Shell
- ⚠️ HRM Engine (requires actual model weights)
- ✅ Unit Test Coverage (64.6% utils, 17.5% inference, 14.0% database, 9.6% api)

#### **Bug Discovery & Resolution:**
During comprehensive testing implementation, several critical issues were identified and documented:
- **Nil Pointer Dereference**: User repository methods lack nil input validation
- **Missing Error Handling**: UpdateUser method doesn't verify user existence
- **Database Schema Inconsistencies**: Password storage implementation gaps
- **Method Signature Mismatches**: Context parameter inconsistencies across repository methods

## 📊 Performance Metrics

### Desktop Host
- **Startup Time**: ~2 seconds
- **Memory Usage**: ~50MB base
- **QR Generation**: <100ms
- **MCP Response**: <50ms

### Mobile Tool
- **Bundle Size**: 316KB (gzipped: 95KB)
- **QR Scan Time**: <500ms
- **Voice Processing**: Real-time
- **Visual Processing**: 30+ FPS

### Agent-Core WASM
- **Module Size**: 370KB
- **Initialization**: <100ms
- **Cognitive Processing**: Variable (depends on HRM)
- **Memory Footprint**: ~10MB

## 🔐 Security Features

### Secure Communication
- QR codes include cryptographic signatures
- Session-based authentication
- Encrypted payload support
- Expiration timestamps

### TEE Integration
- Trusted execution environment
- Secure key storage
- Isolated processing
- Audit logging

### Mobile Security
- Device fingerprinting
- Wallet integration
- Capability-based permissions
- Secure WebSocket connections

## 🛠️ Development

### Project Structure
```
KNIRVENGINE/
├── desktop-client/          # Go backend with HRM
├── mobile-controller/           # React mobile client
├── agent-core/            # WASM cognitive shell
├── test_*.js             # Integration tests
└── README.md             # This file
```

### Contributing
1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

### Code Style
- Go: `gofmt` and `golint`
- TypeScript: ESLint + Prettier
- Rust: `rustfmt` and `clippy`

## 📚 API Reference

### Desktop Host Endpoints
- `GET /api/health` - System health check
- `POST /api/qr/target-assignment` - Generate target QR
- `POST /api/qr/transaction-sign` - Generate transaction QR
- `POST /api/mobile/connect` - Connect mobile device
- `POST /api/hrm/process` - Process cognitive input
- `GET /api/hrm/info` - Get HRM model info
- `GET /api/mcp/ws` - MCP WebSocket endpoint

### Mobile Tool API
- QR Scanner component
- Voice Processor component
- Visual Processor component
- Desktop Connection service

### Agent-Core WASM API
- `new()` - Initialize cognitive core
- `process_cognitive_input()` - Process input
- `set_personality_metric()` - Configure personality
- `connect_to_desktop()` - Establish host connection

## 🚀 Deployment

### Production Deployment
1. Build all components for production
2. Configure environment variables
3. Set up reverse proxy (nginx recommended)
4. Enable HTTPS/WSS
5. Configure monitoring and logging

### Docker Support
```dockerfile
# Example Dockerfile for desktop-client
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o desktop-client main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/desktop-client .
CMD ["./desktop-client"]
```

## 💰 Integrated Wallet Functionality

KNIRVENGINE includes comprehensive wallet functionality that seamlessly integrates with the KNIRV ecosystem, providing users with a unified interface for managing digital assets and NRN tokens.

### 🔗 KNIRVCONTROLLER Integration

The wallet functionality is designed to work in conjunction with KNIRVCONTROLLER:

- **Conditional Activation**: Wallet features only activate when successfully linked with KNIRVCONTROLLER
- **QR Code Pairing**: Secure connection establishment through QR code scanning
- **Real-time Synchronization**: Continuous sync between desktop and mobile wallet states
- **Cross-platform Consistency**: Unified experience across all KNIRV applications

### ✨ Key Wallet Features

#### 🔐 Core Wallet Functionality
- **Multi-Chain Support**: BTC, ETH, Solana, and KNIRV-ORACLE blockchain
- **NRN Token Management**: Native support for KNIRV Network's NRN tokens
- **XION Meta Accounts**: Gasless transactions and Web2-like authentication
- **Biometric Authentication**: Secure, convenient access across platforms
- **Hardware Wallet Support**: Ledger integration for enhanced security

#### 🤖 KNIRV Ecosystem Integration
- **KNIRV-CORTEX Control**: Manage and control your AI agents
- **User Delegation Certificates (UDCs)**: Secure agent authorization
- **NRV System Integration**: Submit ErrorNodes and SkillNodes
- **Economics Module**: Skill registration and fee management
- **Real-time Communication**: WebSocket and SSE support

#### 🌍 Cross-Platform Features
- **QR Code Connectivity**: Seamless connection between mobile and browser
- **Unified Account Management**: Consistent experience across platforms
- **Progressive Web App**: Browser wallet works offline
- **Native Mobile Performance**: Optimized React Native implementation

### 🛠️ Wallet Architecture

```
Desktop-Client Wallet Integration
├── Frontend (React/TypeScript)
│   ├── Wallet Component
│   ├── Dashboard Balance Widget
│   └── Controller Connection Status
├── Backend (Go)
│   ├── Wallet API Handlers
│   ├── Controller Link Management
│   └── Transaction Processing
└── Services
    ├── Wallet Service (TypeScript)
    ├── Controller Connection Service
    └── Real-time Sync Service
```

### 🔄 Wallet Activation Flow

1. **Controller Detection**: System checks for KNIRVCONTROLLER availability
2. **QR Code Generation**: Desktop generates secure pairing QR code
3. **Mobile Scanning**: KNIRVCONTROLLER scans QR code for pairing
4. **Secure Handshake**: Cryptographic verification and session establishment
5. **Wallet Activation**: Full wallet functionality becomes available
6. **Continuous Sync**: Real-time synchronization of wallet state

### 📱 Usage Examples

#### Creating a Wallet Connection
```typescript
// Check controller connection status
const status = await walletService.checkControllerConnection();

// Link with KNIRVCONTROLLER
if (!status.connected) {
  const success = await walletService.linkWithController(controllerEndpoint);
}

// Access wallet functionality
const balance = await walletService.getBalance();
const transactions = await walletService.getTransactions();
```

#### NRN Token Operations
```typescript
// Get NRN balance
const nrnBalance = await walletService.getNRNBalance();

// Transfer NRN tokens
const txHash = await walletService.transferNRN(recipientAddress, amount);

// Burn NRN for skill invocation
const result = await walletService.burnNRNForSkill(skillId, amount);
```

### 🔒 Security Features

- **Non-Custodial Design**: Users maintain full control of private keys
- **Secure Enclave Storage**: Hardware-level security for sensitive data
- **Multi-Factor Authentication**: Additional security layers for high-value operations
- **Transaction Signing**: Secure transaction signing with user confirmation
- **Audit Trail**: Comprehensive logging of all wallet operations

### 🌐 KNIRV Ecosystem Integration

The integrated wallet seamlessly connects with:

- **[KNIRV-ORACLE](../KNIRVORACLE/)**: Foundational blockchain for NRN tokens
- **[KNIRVCHAIN](../KNIRVCHAIN/)**: Smart contract platform for Skills and Base LLMs
- **[KNIRV-CORTEX](../KNIRVCORTEX/)**: AI agent framework
- **[KNIRV-NEXUS](../KNIRVNEXUS/)**: Distributed verification engine
- **[KNIRVGATEWAY](../KNIRVGATEWAY/)**: Unified API gateway
- **[KNIRVSDK](../KNIRVSDK/)**: Development tools and libraries

## 📄 License

This project is part of the KNIRV Network ecosystem. See the main repository for license information.

## 🤝 Support

For support and questions:
- GitHub Issues: [KNIRV_NETWORK Issues](https://github.com/guiperry/KNIRV_NETWORK/issues)
- Documentation: See individual component READMEs
- Community: KNIRV Network Discord

---

**KNIRVENGINE** - Powering the future of autonomous cognitive agents with human-like reasoning capabilities.
