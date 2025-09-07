# KNIRVANA - The Experiential Gateway

<div align="center">

![KNIRVANA Logo](https://img.shields.io/badge/KNIRVANA-Gaming%20Gateway-blue?style=for-the-badge)
[![Rust](https://img.shields.io/badge/Rust-Native%20Client-orange?style=flat-square&logo=rust)](rust-client/)
[![TypeScript](https://img.shields.io/badge/TypeScript-Web%20Client-blue?style=flat-square&logo=typescript)](ts-client/)
[![KNIRV Ecosystem](https://img.shields.io/badge/KNIRV-D--TEN%20Ecosystem-purple?style=flat-square)](https://knirv.network)

*A high-performance, cross-platform Real-Time Strategy game serving as the experiential gateway to the KNIRV Decentralized Trusted Execution Network (D-TEN)*

</div>

## 🎮 Overview

KNIRVANA is a revolutionary 3D TRON-style Real-Time Strategy (RTS) game where players command AI agents to competitively resolve ErrorNodes within the living KNIRVGRAPH. As the primary experiential interface for the KNIRV D-TEN ecosystem, KNIRVANA transforms complex AI network operations into an engaging strategy experience.

### Key Concepts

- **ErrorNodes**: Problems within the KNIRVGRAPH that need resolution
- **AI Agents**: Autonomous units controlled by players to solve ErrorNodes
- **SkillNodes**: Capabilities that agents can learn and execute
- **NRN Tokens**: The native currency earned through successful problem resolution
- **KNIRVGRAPH**: The decentralized knowledge fabric underlying the game world

## 🏗️ Architecture

KNIRVANA offers two complementary client implementations:

### 🦀 Rust Client (Native/Mobile)
High-performance, cross-platform game client built with Bevy engine.

**Platforms**: Desktop (Windows, macOS, Linux), Android, iOS  
**Engine**: Bevy 0.12 with ECS architecture  
**Features**: Native performance, mobile optimizations, offline capabilities

[📖 Rust Client Documentation →](rust-client/)

### 🌐 TypeScript Client (Web)
Full-stack web application with immersive 3D graphics and real-time multiplayer.

**Platform**: Web browsers (Desktop & Mobile)  
**Stack**: React + Three.js + Express.js  
**Features**: WebGL rendering, real-time collaboration, cross-device sync

[📖 TypeScript Client Documentation →](ts-client/)

## 🚀 Quick Start

### Choose Your Client

#### For Native Performance & Mobile
```bash
cd rust-client
cargo run --release
```

#### For Web & Cross-Platform Access
```bash
cd ts-client
npm install
npm run dev
```

## 🎯 Game Mechanics

### Core Gameplay Loop

1. **Explore** the KNIRVGRAPH to discover ErrorNodes
2. **Deploy** AI agents to investigate and resolve errors
3. **Invoke Skills** using NRN tokens to enhance agent capabilities
4. **Compete** with other players in real-time resolution challenges
5. **Earn Rewards** through successful ErrorNode resolution

### Strategic Elements

- **Resource Management**: Balance NRN token spending with skill acquisition
- **Agent Optimization**: Train and specialize agents for different error types
- **Competitive Resolution**: Race against other players to solve high-value ErrorNodes
- **Collective Intelligence**: Contribute solutions to the broader KNIRV knowledge base

## 🔗 KNIRV Ecosystem Integration

KNIRVANA seamlessly integrates with the entire KNIRV D-TEN ecosystem:

### Blockchain Components
- **KNIRV-ORACLE**: NRN token ledger and network oracle
- **KNIRVCHAIN**: Smart contracts and SkillRegistry
- **XION Integration**: Gasless transactions and user experience

### Network Services
- **KNIRVGRAPH**: Decentralized knowledge fabric
- **KNIRV-NEXUS**: Validation environments for skill execution
- **KNIRV-ROUTER**: P2P networking and connectivity proofs
- **KNIRVGATEWAY**: API gateway and service orchestration

### Development Tools
- **KNIRV-SDK**: Integration libraries for both clients
- **KNIRV-CLI**: AI-powered development interface
- **KNIRV-CORTEX**: Agent management and "The Fabric" algorithm

## 🛠️ Development

### Prerequisites

**For Rust Client:**
- Rust 1.70+
- Platform-specific tools (Android NDK, Xcode)

**For TypeScript Client:**
- Node.js 18+
- Modern web browser with WebGL support

### Testing

KNIRVANA includes a comprehensive test suite for both implementations:

```bash
# Run all tests
make test

# Run implementation-specific tests
make test-rust    # Rust client tests
make test-ts      # TypeScript client tests

# Run specific test categories
make test-unit           # Unit tests
make test-integration    # Integration tests
make test-performance    # Performance tests
make test-e2e           # End-to-end tests
```

[📖 Complete Testing Guide →](TESTING.md)

### Project Structure

```
KNIRVANA/
├── rust-client/          # Native Bevy-based client
│   ├── src/             # Rust source code
│   ├── assets/          # Game assets
│   ├── config/          # Configuration files
│   └── examples/        # Example implementations
├── ts-client/           # Web-based client
│   ├── client/          # React frontend
│   ├── server/          # Express.js backend
│   └── shared/          # Shared types and schemas
└── README.md           # This file
```

### Building Both Clients

```bash
# Build Rust client
cd rust-client && cargo build --release

# Build TypeScript client
cd ts-client && npm run build
```

## 🎨 Visual Design

KNIRVANA features a distinctive TRON-style aesthetic with:

- **Neon Grid Environments**: Glowing wireframe landscapes
- **Particle Effects**: Dynamic visual feedback for agent actions
- **Holographic UI**: Immersive interface elements
- **Adaptive Lighting**: Dynamic lighting based on game state
- **Cross-Platform Consistency**: Unified visual experience across clients

## 🌐 Multiplayer & Networking

### Real-Time Strategy Features
- **Synchronized Gameplay**: Real-time competition across all platforms
- **Decentralized Architecture**: P2P networking via KNIRV-ROUTER
- **Cross-Client Compatibility**: Rust and TypeScript clients can play together
- **Persistent World State**: Shared KNIRVGRAPH updates across sessions

## 📊 Economics & Tokenomics

### NRN Token Integration
- **Skill Invocation**: Spend NRN to execute complex agent skills
- **Reward Distribution**: Earn NRN through successful ErrorNode resolution
- **Staking Mechanisms**: Stake NRN for enhanced agent capabilities
- **Governance Participation**: Use NRN for ecosystem governance voting

## 🤝 Contributing

We welcome contributions to both client implementations!

### Getting Started
1. Fork the repository
2. Choose your preferred client (Rust or TypeScript)
3. Follow the client-specific development guides
4. Submit pull requests with comprehensive tests

### Development Guidelines
- Follow the established code style for each client
- Add tests for new features
- Update documentation for API changes
- Ensure cross-platform compatibility

## 📄 License

This project is part of the KNIRV Network ecosystem. See individual client directories for specific licensing information.

## 🔗 Links

- [KNIRV Network](https://knirv.network)
- [KNIRV Documentation](https://docs.knirv.network)
- [KNIRV Gateway](https://gateway.knirv.network)
- [Community Discord](https://discord.gg/knirv)

---

<div align="center">

**Experience the future of decentralized AI through gaming**

[🎮 Start Playing](ts-client/) • [📱 Download Mobile](rust-client/) • [🛠️ Developer Docs](https://docs.knirv.network)

</div>
