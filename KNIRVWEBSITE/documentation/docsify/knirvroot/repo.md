

---

**Source**: KNIRVROOT/.zencoder/docs/repo.md

# Repository Information Overview

## Repository Summary
KNIRVCHAIN is a blockchain-based platform designed to facilitate a transparent, verifiable, and monetizable ecosystem for AI capabilities. It implements the Model Context Protocol (MCP), allowing AI-related functionalities (Tools, Prompts, Plugins, Memory Services) to be registered, discovered, invoked, and audited on-chain.

## Repository Structure
The repository is organized as a multi-project codebase with a Go-based core blockchain implementation and several Node.js-based supporting services:

- **Core Blockchain**: Go-based implementation of the KNIRVCHAIN blockchain and MCP protocol
- **Agent Services**: Multiple Node.js services supporting the blockchain network
- **UI Components**: React-based user interfaces for interacting with the network

### Main Repository Components
- **Core Blockchain**: Go implementation of the blockchain, P2P network, and MCP protocol
- **agent-tunnel-registry**: Node.js service for NAT traversal, node registration, and URI resolution
- **agent-bootnode-registry**: Node.js service for bootnode registration
- **agent-payment-gateway**: Node.js service for payment processing
- **agent-developer-portal**: Node.js service for developer tools and documentation
- **altgui**: React-based alternative GUI for interacting with the KNIRVCHAIN network

## Projects

### Core Blockchain
**Configuration File**: go.mod

#### Language & Runtime
**Language**: Go
**Version**: 1.23.3
**Build System**: Go modules with Makefile
**Package Manager**: Go modules

#### Dependencies
**Main Dependencies**:
- github.com/libp2p/go-libp2p v0.39.1 (P2P networking)
- github.com/libp2p/go-libp2p-kad-dht v0.25.2 (DHT implementation)
- github.com/syndtr/goleveldb v1.0.0 (Database)
- github.com/gin-gonic/gin v1.9.1 (Web framework)
- github.com/spf13/viper v1.20.1 (Configuration)
- github.com/stretchr/testify v1.10.0 (Testing)

#### Build & Installation
```bash
# Build client role
make build

# Build specific roles
make build/root
make build/bootnode
make build/dev

# Build for all roles
make build/all-roles

# Run tests
make test
```

#### Testing
**Framework**: Go testing with testify
**Test Location**: *_test.go files throughout the codebase
**Run Command**:
```bash
make test
```

### agent-tunnel-registry
**Configuration File**: package.json

#### Language & Runtime
**Language**: JavaScript (Node.js)
**Version**: Node.js >=16.0.0
**Package Manager**: npm

#### Dependencies
**Main Dependencies**:
- express: ^4.18.2 (Web framework)
- node-stun: ^0.1.2 (STUN server implementation)
- uuid: ^9.0.0 (UUID generation)
- winston: ^3.10.0 (Logging)

#### Build & Installation
```bash
cd agent-tunnel-registry
npm install
npm start
```

#### Testing
**Framework**: Jest
**Test Location**: agent-tunnel-registry/tests
**Run Command**:
```bash
cd agent-tunnel-registry
npm test
```

### altgui
**Configuration File**: package.json

#### Language & Runtime
**Language**: JavaScript (React)
**Version**: Node.js (unspecified)
**Framework**: Next.js 14.0.4
**Package Manager**: npm

#### Dependencies
**Main Dependencies**:
- react: ^18.2.0
- next: ^14.0.4
- react-bootstrap: ^2.10.1
- axios: ^1.9.0

#### Build & Installation
```bash
cd altgui
npm install
npm run build
npm start
```

#### Testing
**Framework**: Jest with React Testing Library
**Test Location**: Tests alongside components
**Run Command**:
```bash
cd altgui
npm test
```

### agent-developer-portal
**Configuration File**: package.json

#### Language & Runtime
**Language**: JavaScript (Node.js)
**Package Manager**: npm

#### Build & Installation
```bash
cd agent-developer-portal
npm install
npm start
```

### agent-payment-gateway
**Configuration File**: package.json

#### Language & Runtime
**Language**: JavaScript (Node.js)
**Package Manager**: npm

#### Build & Installation
```bash
cd agent-payment-gateway
npm install
npm start
```

### agent-bootnode-registry
**Configuration File**: package.json

#### Language & Runtime
**Language**: JavaScript (Node.js)
**Package Manager**: npm

#### Build & Installation
```bash
cd agent-bootnode-registry
npm install
npm start
```

---

<div class="footer-links">


© 2025 KNIRV Network
</div>
