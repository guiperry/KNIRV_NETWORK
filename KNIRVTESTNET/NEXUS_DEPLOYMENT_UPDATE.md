# KNIRV-NEXUS Deployment Model Update

## Overview

KNIRVTESTNET has been updated to use the new unified KNIRV-NEXUS deployment model. This change simplifies the architecture and aligns the testnet with the production deployment approach.

## Changes Made

### 1. Unified Binary Architecture

**Before:**
- Separate `knirvnexus-dve-manager` binary (port 8084)
- Separate `knirvnexus-validation-core` binary (port 8085)
- Separate frontend service (port 8083)

**After:**
- Single `knirvnexus` unified binary (port 8084)
- Embedded frontend served from the same port
- Embedded backend API accessible via `/api/*` routes

### 2. Updated Scripts

#### Build Script (`scripts/build-knirvnexus.sh`)
- Now builds the unified binary with embedded frontend and backend
- Installs frontend dependencies and builds Next.js output
- Builds unified backend binary
- Creates single `knirvnexus` executable

#### Startup Script (`scripts/start-knirvnexus.sh`)
- Starts single unified service instead of multiple services
- Uses testnet configuration file
- Serves both frontend and API from port 8084

#### Main Testnet Script (`scripts/start-testnet.sh`)
- Removed separate frontend startup step
- Updated service health checks for unified service

### 3. Configuration Updates

#### New Configuration File
- `KNIRVNEXUS/config/nexus-testnet.yaml` - Optimized for testnet deployment
- Includes testnet-specific settings and resource limits

#### Environment Variables (KNIRVGATEWAY)
- `KNIRVNEXUS_URL=http://localhost:8084` - Unified service URL
- `KNIRVNEXUS_API_URL=http://localhost:8084/api` - API endpoint
- Removed separate DVE and Validation URLs

### 4. Admin Gateway

#### Standalone Portal Refactoring
- KNIRVGATEWAY nexus-portal now serves as admin gateway
- Added `admin-gateway.config.js` for admin interface configuration
- Provides administrative access to unified NEXUS service

## Usage

### Building KNIRV-NEXUS for Testnet

```bash
cd KNIRVTESTNET
./scripts/build-knirvnexus.sh
```

### Starting the Testnet

```bash
cd KNIRVTESTNET
./scripts/start-testnet.sh
```

### Accessing Services

- **NEXUS Frontend**: http://localhost:8084
- **NEXUS API**: http://localhost:8084/api
- **Admin Gateway**: http://localhost:8888/nexus-portal (via KNIRVGATEWAY)

### Configuration

The unified service uses the testnet configuration file:
- `KNIRVNEXUS/config/nexus-testnet.yaml`

Command line options:
- `-testnet` - Enable testnet mode
- `-port 8084` - Set service port
- `-config config/nexus-testnet.yaml` - Use specific config file

## Benefits

1. **Simplified Architecture**: Single binary reduces complexity
2. **Production Alignment**: Testnet now matches production deployment model
3. **Resource Efficiency**: Reduced memory and CPU usage
4. **Easier Management**: Single service to start, stop, and monitor
5. **Embedded Frontend**: No separate frontend deployment needed

## Migration Notes

- Old PID files (`knirvnexus-dve-manager.pid`, `knirvnexus-validation-core.pid`, `knirvnexus-frontend.pid`) are no longer used
- New PID file: `knirvnexus.pid`
- Port 8083 (old frontend) is no longer used
- Port 8085 (old validation core) is no longer used
- All functionality now available on port 8084

## Troubleshooting

### Service Not Starting
Check logs: `tail -f logs/knirvnexus.log`

### Frontend Not Loading
Ensure the unified binary was built with frontend embedded:
```bash
./scripts/build-knirvnexus.sh
```

### API Not Responding
Verify the service is running and accessible:
```bash
curl http://localhost:8084/api/health
```

## Future Enhancements

1. **Enhanced Admin Gateway**: Expand admin interface capabilities
2. **Configuration Management**: Web-based configuration editing
3. **Real-time Monitoring**: Enhanced metrics and logging
4. **Service Discovery**: Automatic service registration and discovery
