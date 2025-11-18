# KNIRVORACLE Command Line Interface

This directory contains the main application entry points for KNIRVORACLE.

## Entry Points

### main.go
The primary entry point for the KNIRVORACLE application. This is the standard mode that includes all services and functionality.

### main_bootnode.go
Entry point for running KNIRVORACLE as a bootnode. This mode focuses on P2P discovery and network bootstrapping.

### main_client.go
Entry point for running KNIRVORACLE in client mode. This mode connects to existing Oracle nodes without providing full Oracle services.

### main_developer.go
Entry point for running KNIRVORACLE in developer mode. This mode includes additional debugging tools and development features.

### main_root.go
Entry point for running KNIRVORACLE with root/administrative privileges. This mode includes system-level operations and privileged functionality.

## Usage

Each entry point can be built and run independently:

```bash
# Build the main application
go build -o knirvoracle main.go

# Build the bootnode
go build -o knirvoracle-bootnode main_bootnode.go

# Build the client
go build -o knirvoracle-client main_client.go

# Build the developer version
go build -o knirvoracle-dev main_developer.go

# Build the root version
go build -o knirvoracle-root main_root.go
```

## Configuration

All entry points use the unified configuration system located in the `config/` directory. The configuration is loaded based on environment variables with the `KNIRV_` prefix.

## Package Structure

The entry points import and use the following packages:

- `pkg/services/` - Unified service management
- `pkg/api/` - Unified API endpoints
- `pkg/blockchain/` - Blockchain operations
- `pkg/p2p/` - P2P networking
- `pkg/agent/` - Agent management
- `pkg/wallet/` - Wallet operations
- `pkg/database/` - Database operations
- `pkg/crypto/` - Cryptographic functions
- `pkg/mcp/` - Model Context Protocol
- `pkg/protocol/` - Protocol operations
- `pkg/network/` - Network utilities
- `pkg/integrations/` - External integrations
- `config/` - Configuration management

## Build Tags

Different build tags can be used to include or exclude functionality:

```bash
# Build with all features
go build -tags "full" main.go

# Build without GUI
go build -tags "nogui" main.go

# Build for development
go build -tags "dev,debug" main_developer.go
```

## Environment Variables

Key environment variables (all prefixed with `KNIRV_`):

- `KNIRV_CONFIG_PATH` - Path to configuration file
- `KNIRV_DATA_PATH` - Path to data directory
- `KNIRV_LOG_LEVEL` - Logging level (debug, info, warn, error)
- `KNIRV_HTTP_PORT` - HTTP API port
- `KNIRV_P2P_PORT` - P2P networking port
- `KNIRV_ENABLE_GUI` - Enable web GUI (true/false)

See the configuration documentation for a complete list of environment variables.
