# KNIRV Gaming Cortex Compiler

This is the Rust-based cortex.wasm compiler that has been moved from KNIRVCORTEX to the KNIRVANA gaming sub-module.

## Purpose

This compiler is specifically designed for gaming applications that need high-performance, deterministic cognitive processing with embedded LoRA adapters.

## Features

- **High-Performance WASM**: Optimized for gaming workloads
- **LoRA Adapter Engine**: Embedded neural network skill system
- **Deterministic Processing**: Consistent results for gaming scenarios
- **ProtoBuf ABI**: Efficient binary communication protocol

## Build

```bash
# Build the gaming cortex.wasm
cargo build --target wasm32-unknown-unknown --release

# Copy to dist
cp target/wasm32-unknown-unknown/release/knirv_cortex_wasm.wasm ../../../dist/gaming-cortex.wasm
```

## Integration

This compiler is used by gaming applications that need embedded AI capabilities:

- Real-time strategy games
- Procedural content generation
- Adaptive game mechanics
- Player behavior analysis

## Dependencies

- `shared-types`: ProtoBuf definitions and ABI helpers
- `lora-engine`: Neural network adapter system
- `wasm-bindgen`: WebAssembly bindings

## Migration Note

This compiler was moved from `KNIRVCORTEX/rust-wasm` to support the separation of concerns:
- Gaming applications use this high-performance Rust compiler
- Model building applications use the TypeScript compiler in primary-website
- Agent compilation uses platform-specific compilers in KNIRVCONTROLLER/KNIRVENGINE
