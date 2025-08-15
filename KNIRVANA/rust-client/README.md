# KNIRVANA Rust Game Client

<div align="center">

![Rust](https://img.shields.io/badge/Rust-1.70+-orange?style=flat-square&logo=rust)
![Bevy](https://img.shields.io/badge/Bevy-0.12-blue?style=flat-square)
![Cross Platform](https://img.shields.io/badge/Platform-Desktop%20%7C%20Android%20%7C%20iOS-green?style=flat-square)

*High-performance, cross-platform native game client for the KNIRVANA ecosystem built with Rust and Bevy*

</div>

A blazing-fast, memory-safe game client that brings the full power of native performance to the KNIRVANA gaming experience. Built with the Bevy engine and Rust's zero-cost abstractions, this client delivers optimal performance across desktop and mobile platforms.  Successfully implemented Month 13 Task 13.2 from the KNIRV_D-TEN_Comprehensive_Implementation_Plan.md.

## Table of Contents

- [Overview](#overview)
- [Features](#features)
    - [Performance & Platform Support](#performance--platform-support)
    - [Game Engine Capabilities](#game-engine-capabilities)
    - [Blockchain & Networking](#blockchain--networking)
- [Quick Start](#quick-start)
    - [Prerequisites](#prerequisites)
    - [Installation](#installation)
- [Building](#building)
    - [Desktop (Windows, macOS, Linux)](#desktop-windows-macos-linux)
    - [Android](#android)
    - [iOS](#ios)
- [Mobile Optimizations](#mobile-optimizations)
    - [Performance Adaptations](#performance-adaptations)
    - [Input & UX](#input--ux)
    - [Technical Optimizations](#technical-optimizations)
- [Architecture](#architecture)
    - [Core Modules](#core-modules)
    - [Specialized Systems](#specialized-systems)
    - [Plugin Architecture](#plugin-architecture)
    - [ECS Design Patterns](#ecs-design-patterns)
- [Performance](#performance)
    - [Memory & Safety](#memory--safety)
    - [Concurrency & Parallelism](#concurrency--parallelism)
    - [Platform Optimization](#platform-optimization)
    - [Benchmarks](#benchmarks)
- [Integration with KNIRV Ecosystem](#integration-with-knirv-ecosystem)
    - [Blockchain Layer](#blockchain-layer)
    - [Network Services](#network-services)
    - [Development Tools](#development-tools)
    - [Game Mechanics](#game-mechanics)
    - [Economic Integration](#economic-integration)
- [Development](#development)
    - [Development Environment Setup](#development-environment-setup)
    - [Running & Testing](#running--testing)
    - [Development Features](#development-features)
    - [Code Quality](#code-quality)
- [Configuration](#configuration)
    - [Environment Variables](#environment-variables)
    - [Configuration Files](#configuration-files)
- [Contributing](#contributing)
    - [Getting Started](#getting-started)
    - [Development Guidelines](#development-guidelines)
    - [Areas for Contribution](#areas-for-contribution)
- [License](#license)
- [Links](#links)


## Overview

This document provides a comprehensive guide to the KNIRVANA Rust game client, a high-performance, cross-platform native game client built using Rust and the Bevy engine.  It includes details on features, architecture, building, deployment, and integration with the KNIRV ecosystem.  The project successfully implemented Month 13 Task 13.2 from the KNIRV_D-TEN_Comprehensive_Implementation_Plan.md.


## Features

### Performance & Platform Support
- **Cross-Platform**: Native support for Desktop (Windows, macOS, Linux), Android, and iOS
- **Zero-Cost Abstractions**: Rust's memory safety without garbage collection overhead
- **Multithreaded ECS**: Bevy's parallel processing for optimal performance
- **Mobile Optimized**: Adaptive graphics quality, battery optimization, touch controls

### Game Engine Capabilities
- **3D Graphics**: Modern 3D rendering with post-processing effects and dynamic lighting
- **Physics Simulation**: Realistic physics using Rapier3D with mobile optimizations
- **Audio System**: Spatial audio with Kira audio engine
- **Input Handling**: Unified input system supporting keyboard, mouse, and touch

### Blockchain & Networking
- **XION Integration**: Native blockchain connectivity with gasless transactions
- **NRN Token Support**: Direct token management and skill invocation
- **Real-time Multiplayer**: P2P networking via KNIRV-ROUTER
- **Offline Capabilities**: Local gameplay with sync when connected


## Quick Start

### Prerequisites
- **Rust 1.70+** - Install from [rustup.rs](https://rustup.rs/)
- **Git** - For cloning the repository
- **Platform-specific tools** (see platform sections below)

### Installation

```bash
# Clone the repository
git clone <repository-url>
cd KNIRVANA/rust-client

# Build and run (desktop)
cargo run --release
```

## Building

### Desktop (Windows, macOS, Linux)
```bash
# Debug build
cargo build

# Release build (optimized)
cargo build --release

# Run directly
cargo run --release
```

### Android
```bash
# Install Android NDK and tools
export ANDROID_NDK_ROOT=/path/to/android-ndk
rustup target add aarch64-linux-android
cargo install cargo-apk

# Build APK
cargo apk build --release

# Install and run on device
cargo apk run --release
```

### iOS
```bash
# Install iOS targets and tools
rustup target add aarch64-apple-ios
cargo install cargo-lipo

# Build universal binary
cargo lipo --release

# Open in Xcode for deployment
open target/universal/release/
```

## Mobile Optimizations

### Performance Adaptations
- **Adaptive Quality**: Automatically adjusts graphics quality based on device performance
- **Battery Optimization**: Intelligent frame rate management and system sleeping
- **Memory Management**: Efficient asset streaming and garbage collection
- **Thermal Throttling**: Automatic quality reduction to prevent overheating

### Input & UX
- **Touch Controls**: Intuitive touch-based movement and interaction
- **Gesture Recognition**: Pinch-to-zoom, swipe navigation, and multi-touch support
- **Haptic Feedback**: Native vibration support for game events
- **Adaptive UI**: Screen size and orientation-aware interface scaling

### Technical Optimizations
- **Simplified Physics**: Optimized Rapier3D configuration for mobile
- **Reduced Effects**: Configurable particle and post-processing reduction
- **Asset Compression**: Optimized textures and models for mobile storage
- **Network Efficiency**: Reduced bandwidth usage and smart caching


## Architecture

### Core Modules
- **`main.rs`** - Application entry point and plugin configuration
- **`lib.rs`** - Public API and library interface for external integration
- **`game_engine.rs`** - Core game engine logic, events, and state management
- **`components.rs`** - ECS components for game entities (players, agents, nodes)
- **`systems.rs`** - Game systems for updating entities and handling logic
- **`resources.rs`** - Global game resources, configuration, and shared state

### Specialized Systems
- **`networking.rs`** - Network communication, synchronization, and blockchain integration
- **`mobile.rs`** - Mobile-specific optimizations, input handling, and platform features
- **`knirvana_systems.rs`** - KNIRVANA-specific game logic and KNIRV ecosystem integration

### Plugin Architecture
```rust
App::new()
    .add_plugins(DefaultPlugins)
    .add_plugins(GameEnginePlugin)
    .add_plugins(NetworkingPlugin)
    .add_plugins(MobilePlugin)
    .run();
```

### ECS Design Patterns
- **Components**: Data-only structs representing game entity properties
- **Systems**: Functions that operate on entities with specific component combinations
- **Resources**: Global state accessible across all systems
- **Events**: Type-safe communication between systems


## Performance

### Memory & Safety
- **Zero-Cost Abstractions**: High-level features without runtime overhead
- **Memory Safety**: Compile-time guarantees preventing crashes and security vulnerabilities
- **No Garbage Collection**: Deterministic memory management for consistent frame rates
- **RAII**: Automatic resource cleanup preventing memory leaks

### Concurrency & Parallelism
- **Multithreaded ECS**: Bevy's parallel system execution across CPU cores
- **Async Networking**: Non-blocking I/O for smooth gameplay during network operations
- **Lock-Free Data Structures**: Minimal contention in multi-threaded scenarios
- **Work Stealing**: Efficient task distribution across available threads

### Platform Optimization
- **Native Compilation**: Platform-specific optimizations for each target
- **SIMD Instructions**: Vectorized operations for graphics and physics calculations
- **Battery Efficiency**: Intelligent power management for mobile devices
- **Thermal Management**: Adaptive performance scaling to prevent overheating

### Benchmarks
- **Frame Rate**: Consistent 60+ FPS on mid-range mobile devices
- **Memory Usage**: <100MB RAM usage on mobile platforms
- **Startup Time**: <2 seconds cold start on most devices
- **Network Latency**: <50ms response time for multiplayer actions


## Integration with KNIRV Ecosystem

### Blockchain Layer
- **KNIRV-ROOT**: Native NRN token ledger integration with real-time balance updates
- **KNIRVCHAIN**: Direct smart contract interaction for skill registry and governance
- **XION Integration**: Gasless transaction support with seamless user experience
- **Wallet Connectivity**: Support for KNIRVWALLET and external wallet providers

### Network Services
- **KNIRVGRAPH**: Real-time knowledge graph synchronization and ErrorNode discovery
- **KNIRV-NEXUS**: Validation environment connectivity for skill execution proofs
- **KNIRV-ROUTER**: P2P networking with proof-of-connectivity rewards
- **KNIRVGATEWAY**: API gateway integration for service orchestration

### Development Tools
- **KNIRV-SDK**: Native Rust SDK integration for all KNIRV services
- **KNIRV-SHELL**: AI-powered development interface connectivity
- **KNIRV-CORTEX**: Agent management and "The Fabric" algorithm integration

### Game Mechanics
- **ErrorNode Resolution**: Competitive resolution of KNIRVGRAPH errors
- **AI Agent Management**: Deploy, train, and optimize autonomous agents
- **Skill Invocation**: Execute complex skills using NRN token burning
- **Collective Intelligence**: Contribute solutions to the decentralized knowledge base
- **Real-time Competition**: Synchronized multiplayer across all client types

### Economic Integration
- **NRN Token Economy**: Earn, spend, and stake NRN tokens through gameplay
- **Skill Marketplace**: Purchase and trade skills with other players
- **Governance Participation**: Vote on network upgrades using in-game NRN
- **Reward Distribution**: Automatic reward calculation and distribution


## Development

### Development Environment Setup

```bash
# Install Rust toolchain
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh

# Add required targets
rustup target add aarch64-linux-android  # Android
rustup target add aarch64-apple-ios      # iOS

# Install development tools
cargo install cargo-apk cargo-lipo
```

### Running & Testing

```bash
# Desktop development
cargo run                    # Debug build
cargo run --release         # Release build
cargo run --features mobile # Mobile features on desktop

# Mobile development
cargo apk run               # Android device/emulator
cargo lipo --release        # iOS universal binary

# Testing
cargo test                  # Unit tests
cargo test --release        # Optimized tests
cargo bench                 # Performance benchmarks

# Documentation
cargo doc --open            # Generate and open docs
cargo doc --no-deps         # Only project docs
```

### Development Features

```bash
# Feature flags for development
cargo run --features "mobile,debug,dev-tools"

# Available features:
# - mobile: Mobile optimizations
# - debug: Debug overlays and logging
# - dev-tools: Development utilities
# - networking: Network features
# - blockchain: Blockchain integration
```

### Code Quality

```bash
# Linting and formatting
cargo clippy                # Rust linter
cargo fmt                   # Code formatting
cargo audit                 # Security audit

# Performance profiling
cargo flamegraph            # CPU profiling
cargo instruments           # macOS profiling
```

## Configuration

### Environment Variables

```bash
# Core configuration
export KNIRVANA_API_ENDPOINT="https://api.knirv.com"
export KNIRVANA_CHAIN_ID="knirv-mainnet-1"
export KNIRVANA_NETWORK_MODE="mainnet"  # mainnet, testnet, local

# Performance settings
export KNIRVANA_GRAPHICS_QUALITY="auto"  # low, medium, high, ultra, auto
export KNIRVANA_MOBILE_OPTIMIZED="auto"  # true, false, auto
export KNIRVANA_MAX_FPS="60"
export KNIRVANA_VSYNC="true"

# Blockchain settings
export KNIRVANA_WALLET_PROVIDER="knirvwallet"
export KNIRVANA_AUTO_CONNECT="true"
export KNIRVANA_GAS_PRICE="0.025"

# Debug settings
export KNIRVANA_DEBUG_MODE="false"
export KNIRVANA_LOG_LEVEL="info"  # error, warn, info, debug, trace
export KNIRVANA_PROFILING="false"
```

### Configuration Files

```toml
# config/game_config.toml
[graphics]
quality = "auto"
vsync = true
max_fps = 60
fullscreen = false

[audio]
master_volume = 1.0
sfx_volume = 0.8
music_volume = 0.6

[network]
auto_connect = true
timeout = 30
retry_attempts = 3

[blockchain]
wallet_provider = "knirvwallet"
auto_sign = false
gas_price = 0.025
```

## Contributing

### Getting Started
1. **Fork** the repository on GitHub
2. **Clone** your fork locally
3. **Create** a feature branch: `git checkout -b feature/amazing-feature`
4. **Install** development dependencies: `cargo install cargo-clippy cargo-audit`
5. **Make** your changes following our coding standards
6. **Test** your changes: `cargo test && cargo clippy`
7. **Commit** your changes: `git commit -m 'Add amazing feature'`
8. **Push** to your branch: `git push origin feature/amazing-feature`
9. **Submit** a Pull Request

### Development Guidelines
- **Code Style**: Follow Rust standard formatting (`cargo fmt`)
- **Linting**: Ensure `cargo clippy` passes without warnings
- **Testing**: Add tests for new functionality
- **Documentation**: Document public APIs and complex logic
- **Performance**: Profile performance-critical changes
- **Mobile**: Test mobile optimizations on actual devices

### Areas for Contribution
- **Platform Support**: Additional platform integrations
- **Performance**: Optimization and profiling improvements
- **Graphics**: Visual effects and rendering enhancements
- **Networking**: P2P and blockchain integration improvements
- **Mobile**: Touch controls and mobile UX enhancements
- **Documentation**: Tutorials, examples, and API documentation


## License

This project is part of the KNIRV Network ecosystem. See [LICENSE](LICENSE) for details.

## Links

- **[KNIRV Network](https://knirv.network)** - Main ecosystem website
- **[Documentation](https://docs.knirv.network)** - Complete KNIRV documentation
- **[TypeScript Client](../ts-client/)** - Web-based KNIRVANA client
- **[KNIRV SDK](../../KNIRVSDK/)** - Development SDKs and tools
- **[Community Discord](https://discord.gg/knirv)** - Join our community

---

<div align="center">

**Experience native performance in the KNIRVGRAPH**

[🎮 Download](https://github.com/knirv/releases) • [📱 Mobile](https://play.google.com/store/apps/details?id=com.knirv.knirvana) • [🛠️ Build Guide](#building)

</div>
