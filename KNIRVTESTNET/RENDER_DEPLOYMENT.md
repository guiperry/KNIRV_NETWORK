# KNIRV Testnet Deployment on Render.com

This guide explains how to deploy the KNIRV Testnet on Render.com with automatic toolchain installation.

## 🚀 Quick Deployment

### 1. Render Service Configuration

**Build Command:**
```bash
npm run build:render
```

**Start Command:**
```bash
npm start
```

> **Note**: The start process now includes automatic toolchain detection and installation. If Go/Rust toolchains are missing, they will be installed automatically during startup.

### 2. Environment Variables

Set these environment variables in your Render service:

```bash
NODE_ENV=production
DEPLOYMENT_ENV=testnet
PORT=10000
```

### 3. Automatic Toolchain Installation

The `npm run build:render` command will automatically:

- ✅ Install Go toolchain (v1.23.4)
- ✅ Install Rust toolchain (latest stable)
- ✅ Install build essentials (if available)
- ✅ Run `go mod tidy` for all Go applications
- ✅ Build all KNIRV components
- ✅ Build frontend applications
- ✅ Load production endpoints

## 🔧 Manual Toolchain Installation

If you need to install toolchains separately:

```bash
# Install Go and Rust toolchains
npm run install:toolchains

# Install Node.js dependencies
npm run install:deps

# Build all components
npm run build:all
```

## 📋 Build Process

The Render build process follows these steps:

1. **Install Toolchains** - Go, Rust, build tools
2. **Install Dependencies** - npm packages for all sub-projects
3. **Pre-Start Check** - Verify toolchains are available, install if missing
4. **Build Backend Components**:
   - KNIRV-ORACLE (Go)
   - KNIRVCHAIN (Rust)
   - KNIRVGRAPH (Go)
   - KNIRV-NEXUS (Go)
   - KNIRV-ROUTER (Go)
   - KNIRV-GATEWAY (Node.js)
4. **Build Frontend Components**:
   - NEXUS Portal (Next.js)

5. **Load Configuration** - Testnet endpoints and settings

## 🏗️ Architecture

```
KNIRVTESTNET (Port 10000)
├── KNIRV-ORACLE (Port 1317) - Blockchain foundation
├── KNIRVCHAIN (Port 8090) - Smart contracts & LLM validation  
├── KNIRVGRAPH (Port 8082) - Graph storage & DHT
├── KNIRV-NEXUS (Port 8084) - TEE simulation
├── KNIRV-ROUTER (Port 8086) - Network routing
├── KNIRV-GATEWAY (Port 8888) - API gateway

└── Health Monitor (Port 10001) - Service monitoring
```

## 🔍 Health Checks

The testnet includes comprehensive health monitoring:

- **Service Health**: `/health` endpoints for all services
- **Dynamic Port Discovery**: Automatic port detection
- **Status Dashboard**: `npm run testnet:status`
- **Health Monitor**: http://your-app.onrender.com:10001/health-monitor

## 🛠️ Troubleshooting

### Build Failures

If the build fails due to missing toolchains:

1. Check the build logs for specific errors
2. Ensure the `install-deps.sh` script ran successfully
3. Verify Go and Rust are in the PATH

### Runtime Issues

If services fail to start:

1. Check service logs in the `logs/` directory
2. Verify all required ports are available
3. Run health checks: `npm run testnet:status`

### Common Issues

**"go: command not found"**
- Solution: Run `npm run install:toolchains` before building

**"rustc: command not found"**  
- Solution: Run `npm run install:toolchains` before building

**"Text file busy" errors**
- Solution: The build scripts now handle this automatically

## 📊 Monitoring

Access these URLs once deployed:

- **Main Portal**: `https://your-app.onrender.com`
- **Health Monitor**: `https://your-app.onrender.com:10001/health-monitor`
- **API Gateway**: `https://your-app.onrender.com:8888`
- **Agent Registry**: `https://your-app.onrender.com:9002`

## 🔄 Updates

To update the deployment:

1. Push changes to your repository
2. Render will automatically rebuild using `npm run build:render`
3. The build process will update all components
4. Services will restart with new binaries

## 📝 Notes

- The testnet runs in **local network mode** for testing
- All services use **mock/simulation** modes for development
- **Simplified authentication** is enabled for easy testing
- **In-memory storage** is used (data doesn't persist between restarts)

For production deployments, additional configuration may be required.
