# KNIRV Wizened Environment Implementation

This document describes the implementation of the WizenedEnvironmentGuide.md in KNIRVTESTNET.

## Overview

The KNIRV Testnet now uses a **wizened WASM module** approach for instant startup and pre-configured environments. This implementation follows the WizenedEnvironmentGuide.md exactly, providing:

- **Instant Startup**: Pre-initialized WASM module with Wizer snapshots
- **Hybrid Architecture**: WASM module for orchestration, native binaries for performance
- **Local Build**: All binaries built locally, not in the cloud
- **Render Deployment**: Optimized for Render.com with Docker runtime

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Render.com Host                         │
│  ┌─────────────────────────────────────────────────────┐    │
│  │              Docker Container                       │    │
│  │  ┌─────────────────────────────────────────────┐    │    │
│  │  │         Wasmtime Runtime                    │    │    │
│  │  │  ┌─────────────────────────────────────┐    │    │    │
│  │  │  │    knirv-server.wasm                │    │    │    │
│  │  │  │  (Wizened with Wizer)               │    │    │    │
│  │  │  │                                     │    │    │    │
│  │  │  │  • Pre-configured environment       │    │    │    │
│  │  │  │  • Python runtime                   │    │    │    │
│  │  │  │  • Bash shell                       │    │    │    │
│  │  │  │  • KNIRV orchestrator               │    │    │    │
│  │  │  │  • Instant startup                  │    │    │    │
│  │  │  └─────────────────────────────────────┘    │    │    │
│  │  └─────────────────────────────────────────────┘    │    │
│  │                                                      │    │
│  │  Native KNIRV Services:                              │    │
│  │  • bin/knirvoracle                                   │    │
│  │  • bin/knirvchain                                    │    │
│  │  • bin/knirvgraph                                    │    │
│  │  • bin/knirvnexus                                    │    │
│  │  • bin/knirvrouter                                   │    │
│  └─────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
```

## Key Components

### 1. Wizened WASM Module (`bin/knirv-server.wasm`)
- **Pre-initialized** with Wizer at build time
- Contains the KNIRV orchestrator written in Go
- Includes Python runtime and bash shell
- Starts instantly without setup overhead

### 2. Virtual File System (VFS)
- `wasi-vfs-root/` contains the filesystem for the WASM module
- `wasi-vfs-root/bin/` - WASI-compatible binaries (python, bash)
- `wasi-vfs-root/scripts/setup.sh` - Environment configuration script
- `wasi-vfs-root/lib/` - Libraries and dependencies
- `wasi-vfs-root/go-workspace/` - Go workspace

### 3. Build Process
- **Local Build**: `scripts/build-local-release.sh`
- Downloads WASI SDK, Wizer, and wasi-vfs tools
- Compiles `launcher.c` and `wasm-main.go`
- Uses Wizer to pre-initialize the WASM module
- Creates `bin/knirv-server.wasm` with instant startup

### 4. Deployment
- **Docker-based**: Uses `Dockerfile` for Render deployment
- **Pre-built artifacts**: All binaries built locally and committed
- **Wasmtime runtime**: Downloads and runs the wizened module

## Build Instructions

### Prerequisites
- Go 1.21+
- Rust with `x86_64-unknown-linux-gnu` target
- Internet connection (for downloading tools)

### Local Build
```bash
# Build all components locally
cd KNIRVTESTNET
./scripts/build-local-release.sh

# This creates:
# - bin/knirv-server.wasm (wizened module)
# - bin/knirv* (native KNIRV services)
# - tools/ (build tools)
```

### Deployment to Render
```bash
# 1. Build locally first
./scripts/build-local-release.sh

# 2. Commit artifacts
git add bin/ tools/
git commit -m "Add wizened artifacts"

# 3. Push to Render
git push origin main
```

## File Structure

```
KNIRVTESTNET/
├── launcher.c                 # C entry point for Wizer
├── wasm-main.go              # Go application for WASM module
├── main.go                   # Original orchestrator (deprecated)
├── Dockerfile                # Docker configuration for Render
├── render.yaml               # Render deployment configuration
├── wasi-vfs-root/           # Virtual filesystem for WASM
│   ├── bin/                 # WASI-compatible binaries
│   ├── scripts/setup.sh     # Environment setup script
│   ├── lib/                 # Libraries
│   └── go-workspace/        # Go workspace
├── bin/                     # Built artifacts
│   ├── knirv-server.wasm    # Wizened WASM module
│   ├── knirv*               # Native KNIRV services
│   └── wasmtime             # WASM runtime
├── tools/                   # Build tools
│   ├── wasi-sdk/            # WASI SDK
│   ├── wizer                # Wizer tool
│   └── wasi-vfs             # VFS packing tool
└── scripts/
    ├── build-local-release.sh  # Main build script
    └── start-render.sh         # Render startup script
```

## Environment Variables

The wizened module sets these environment variables:

```bash
KNIRV_ENV=wizened-wasm
KNIRV_TOOLCHAIN_PATH=/bin
KNIRV_WORKSPACE=/go-workspace
PYTHON_EXECUTABLE=/bin/python
GOROOT=/usr/lib/go
GOPATH=/go-workspace
PATH=/bin:/usr/lib/go/bin:/usr/lib/cargo/bin:$PATH
```

## Benefits

1. **Instant Startup**: No installation or setup time
2. **Consistent Environment**: Pre-configured and tested
3. **Hybrid Performance**: WASM for orchestration, native for compute
4. **Local Build**: No cloud compilation dependencies
5. **Render Optimized**: Docker-based deployment

## Troubleshooting

### Build Issues
- Ensure all prerequisites are installed
- Check internet connection for tool downloads
- Verify Go and Rust toolchains are working

### Deployment Issues
- Ensure `bin/knirv-server.wasm` exists and is committed
- Check Render logs for wasmtime execution errors
- Verify Docker build completes successfully

### Runtime Issues
- Check health endpoint: `/health`
- Review container logs in Render dashboard
- Verify WASM module permissions and file mappings

## Migration from Previous Implementation

The wizened implementation replaces:
- Node.js runtime → Docker with wasmtime
- Mock services → Real services in WASM
- Runtime installation → Pre-built artifacts
- Express.js server → Go web server in WASM

All existing endpoints and functionality are preserved.
