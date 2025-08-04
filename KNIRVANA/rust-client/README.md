# KNIRVANA Rust Game Client

A high-performance, cross-platform game client for the KNIRVANA ecosystem built with Rust and Bevy.

## Features

- **Cross-Platform**: Supports Desktop (Windows, macOS, Linux), Android, and iOS
- **Mobile Optimized**: Adaptive graphics quality, battery optimization, touch controls
- **Blockchain Integration**: Native integration with XION blockchain and NRN token
- **Real-time Networking**: Multiplayer support with server synchronization
- **3D Graphics**: Modern 3D rendering with post-processing effects
- **Physics Simulation**: Realistic physics using Rapier3D
- **Modular Architecture**: Clean, extensible codebase

## Building

### Desktop
```bash
cargo build --release
```

### Android
```bash
# Install Android NDK and set environment variables
export ANDROID_NDK_ROOT=/path/to/android-ndk
cargo install cargo-apk
cargo apk build --release
```

### iOS
```bash
# Requires Xcode and iOS development setup
cargo install cargo-lipo
cargo lipo --release
```

## Mobile Optimizations

- **Adaptive Quality**: Automatically adjusts graphics quality based on performance
- **Battery Optimization**: Reduces update frequency for non-critical systems
- **Touch Controls**: Intuitive touch-based movement and interaction
- **Simplified Physics**: Optimized physics simulation for mobile devices
- **Reduced Effects**: Configurable particle and effect reduction

## Architecture

- `main.rs`: Application entry point and plugin configuration
- `game_engine.rs`: Core game engine logic and event handling
- `components.rs`: ECS components for game entities
- `systems.rs`: Game systems for updating entities
- `resources.rs`: Global game resources and state
- `networking.rs`: Network communication and synchronization
- `mobile.rs`: Mobile-specific optimizations and input handling

## Performance

The Rust implementation provides significant performance benefits:
- **Memory Safety**: Zero-cost abstractions with compile-time guarantees
- **Multithreading**: Efficient parallel processing with Bevy's ECS
- **Low Latency**: Minimal garbage collection overhead
- **Battery Efficiency**: Optimized for mobile battery life

## Integration with KNIRV Ecosystem

### Blockchain Integration
- **NRN Token**: Native support for NRN token transactions
- **XION Blockchain**: Direct integration with XION for gasless transactions
- **Smart Contracts**: Interaction with KNIRVCHAIN smart contracts

### Network Components
- **KNIRVROOT**: Oracle and orchestration services
- **KNIRVGRAPH**: Knowledge graph integration
- **KNIRVNEXUS**: Validation engine connectivity
- **KNIRVROUTER**: P2P networking and routing

### Game Features
- **Agent Units**: Control and manage AI agent units
- **Skill System**: Invoke skills using NRN tokens
- **Challenge System**: Complete challenges for rewards
- **Real-time Multiplayer**: Synchronized gameplay across devices

## Development

### Prerequisites
- Rust 1.70+
- Bevy 0.12+
- Android NDK (for Android builds)
- Xcode (for iOS builds)

### Running
```bash
# Desktop
cargo run

# Mobile (with features)
cargo run --features mobile

# Android
cargo apk run

# iOS
cargo lipo --release && open target/universal/release/
```

### Testing
```bash
cargo test
```

### Documentation
```bash
cargo doc --open
```

## Configuration

The game client can be configured through environment variables:

```bash
export KNIRVANA_API_ENDPOINT="https://api.knirv.com"
export KNIRVANA_CHAIN_ID="knirv-mainnet-1"
export KNIRVANA_MOBILE_OPTIMIZED="true"
export KNIRVANA_GRAPHICS_QUALITY="medium"
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

This project is part of the KNIRV Network ecosystem.
