# Quick Start Guide - KNIRVORACLE Go Version

Get up and running with the Go implementation in under 5 minutes.

## Prerequisites

- Go 1.21 or higher
- Node.js 18+ (for embedded services)
- npm

## 3-Step Quick Start

### 1. Build

```bash
cd KNIRVORACLE
make -f Makefile.go go-build
```

Expected output:
```
Formatting Go code...
Vetting Go code...
Building Go application...
Build complete: bin/knirvoracle
```

### 2. Run

```bash
./bin/knirvoracle
```

Expected output:
```
{"level":"info","msg":"KNIRVORACLE starting","mode":"persistent","port":8080,"chainID":"testnet"}
{"level":"info","msg":"Auto-starting Node.js services"}
{"level":"info","msg":"Starting HTTP server","address":":8080"}
```

### 3. Test

Open your browser:
```
http://localhost:8080
```

Click the **"Login to Dashboard"** button to access the WebGUI.

## What You Get

✅ **Landing Page** at `http://localhost:8080`
- Modern UI with login button
- Service information
- Documentation links

✅ **Dashboard** at `http://localhost:8080/dashboard`
- Full WebGUI interface
- Network management
- Service controls

✅ **Auto-Started Services**
- Payment Gateway (port 3001)
- Tunnel Registry (port 3002)
- WebGUI (port 3007)

## Common Commands

```bash
# Health check
curl http://localhost:8080/health

# Service status
curl http://localhost:8080/services/status

# Stop all services
curl -X POST http://localhost:8080/services/stop

# Start all services
curl -X POST http://localhost:8080/services/start
```

## Configuration

Create a `.env` file:

```bash
# Server
PORT=8080
GATEWAY_MODE=persistent
KNIRV_CHAIN_ID=testnet

# Services
NODEJS_SERVICES_AUTOSTART=true
PAYMENT_GATEWAY_PORT=3001
TUNNEL_REGISTRY_HTTP_PORT=3002
WEBGUI_PORT=3007
```

Then run:
```bash
./bin/knirvoracle
```

## Development Mode

For development with hot-reload:

```bash
# Install air
go install github.com/cosmtrek/air@latest

# Run with auto-reload
make -f Makefile.go go-watch
```

Now any changes to `.go` files will automatically rebuild and restart the server.

## Docker Quick Start

```bash
# Build Docker image
docker build -t knirvoracle:latest -f Dockerfile.go .

# Run container
docker run -p 8080:8080 knirvoracle:latest

# Open browser
open http://localhost:8080
```

## Troubleshooting

### Port Already in Use

```bash
# Use a different port
PORT=9000 ./bin/knirvoracle
```

### Services Not Starting

```bash
# Install service dependencies
cd services/payment-oracle && npm install
cd ../tunnel-registry && npm install
cd ../webgui && npm install
cd ../..

# Restart the oracle
./bin/knirvoracle
```

### Build Errors

```bash
# Clean and rebuild
make -f Makefile.go go-clean
make -f Makefile.go go-build
```

## Next Steps

1. **Read the docs**: Check `README.go.md` for detailed documentation
2. **Explore the API**: Try the various endpoints
3. **Customize**: Modify configuration for your needs
4. **Deploy**: Follow `MIGRATION.md` for deployment options

## Getting Help

- 📖 Full Documentation: `README.go.md`
- 🔄 Migration Guide: `MIGRATION.md`
- 📋 Implementation Details: `GO_IMPLEMENTATION.md`
- 💬 Community: KNIRV Discord

That's it! You now have a fully functional KNIRVORACLE running in Go! 🚀
