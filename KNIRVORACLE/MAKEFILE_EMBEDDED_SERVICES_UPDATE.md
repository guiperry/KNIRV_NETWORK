# KNIRVORACLE Makefile - Embedded Services Update

## Overview

The KNIRVORACLE Makefile has been updated to recognize and manage the new locations of all Node.js services and embedded binary services in the `embedded/` directory structure.

## New Directory Structure

### Node.js Services
- **Tunnel Registry**: `embedded/nodejs/tunnel/agent-tunnel-registry/`
- **Payment Gateway**: `embedded/nodejs/payment/agent-payment-gateway/`
- **Operator Registry**: `embedded/nodejs/operator/operator-registry/`
- **WebGUI**: `embedded/nodejs/webgui/webGUI/` (Next.js application)
- **Docsify**: `embedded/nodejs/docsify/docsify/`

### Binary Services
- **Economics Service**: `embedded/binaries/economics/economics-service`
- **Network Monitor**: `embedded/binaries/network/knirv-network-monitor`

## New Makefile Targets

### Main Targets

- **`build-embedded-binaries`**: Build all embedded services (Go binaries + Node.js services)
- **`build-embedded-go-binaries`**: Build only Go binaries (network-monitor, economics)
- **`build-embedded-nodejs-services`**: Build and prepare all Node.js services
- **`clean-embedded`**: Clean all embedded services (remove node_modules, builds, binaries)

### Node.js Service Management

- **`install-embedded-nodejs-deps`**: Install Node.js dependencies for all embedded services
- **`build-embedded-webgui`**: Build Next.js WebGUI application specifically

### Individual Service Targets

- **`build-tunnel-registry`**: Build tunnel registry service individually
- **`build-payment-gateway`**: Build payment gateway service individually
- **`build-operator-registry`**: Build operator registry service individually
- **`build-webgui`**: Build WebGUI service individually
- **`build-docsify`**: Build docsify service individually

## Key Features

### 1. **Comprehensive Service Management**
- Handles both Go binaries and Node.js services
- Automatic dependency installation for Node.js services
- Special handling for Next.js applications that need building

### 2. **Error Handling**
- Graceful fallback when services are not found
- Silent npm operations with error reporting
- Continues building other services if one fails

### 3. **Directory Creation**
- Automatically creates necessary directories for binaries
- Ensures proper structure for embedded services

### 4. **Backward Compatibility**
- Maintains existing `build-embedded-binaries` target
- Integrates with existing build pipeline
- Works with cross-compilation scripts

## Usage Examples

```bash
# Build all embedded services
make build-embedded-binaries

# Build only Go binaries
make build-embedded-go-binaries

# Install Node.js dependencies only
make install-embedded-nodejs-deps

# Build individual services
make build-tunnel-registry
make build-webgui

# Clean all embedded services
make clean-embedded

# Full clean (includes embedded services)
make clean
```

## Integration with Existing Workflow

The updated Makefile integrates seamlessly with the existing build workflow:

1. **Development builds** (`make build`, `make build/root`, etc.) now include embedded services
2. **Cross-compilation** (`make build-binaries`) includes embedded services preparation
3. **Clean operations** (`make clean`) now clean embedded services as well

## Testing Results

All new targets have been tested and are working correctly:

✅ `make build-embedded-go-binaries` - Builds Go binaries
✅ `make install-embedded-nodejs-deps` - Installs Node.js dependencies
✅ `make build-tunnel-registry` - Builds individual services
✅ `make clean-embedded` - Cleans embedded services
✅ `make build-embedded-nodejs-services` - Builds all Node.js services

## Notes

- **WebGUI Build**: The Next.js WebGUI may require additional configuration for production builds
- **Service Detection**: Services are automatically detected based on directory and package.json presence
- **Silent Operations**: npm operations run silently to reduce output noise
- **Error Tolerance**: Build continues even if individual services fail to build

This update ensures that the KNIRVORACLE build system properly recognizes and manages all embedded services in their new locations, providing a comprehensive and maintainable build pipeline.
