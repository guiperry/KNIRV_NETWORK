# KNIRV Testnet - Podman Migration Guide

This document explains the migration from Docker to Podman for the KNIRVTESTNET deployment.

## 🚀 Overview

The KNIRVTESTNET has been updated to support Podman as the primary container runtime for local development, while maintaining Docker compatibility for cloud deployments where Podman is not supported (like Render.com).

## 📋 What Changed

### 1. Container Configuration
- **docker-compose.yml**: Updated with Podman-specific configurations
  - Added `userns_mode: "keep-id"` for proper rootless container support
  - Added `:Z` flags to volume mounts for SELinux compatibility
  - Changed Dockerfile references to Containerfile (Podman's preferred format)
  - Updated networking comments for Podman's netavark/aardvark-dns

### 2. Container Files
- **Containerfile**: Created as Podman's preferred container definition format
- **Dockerfile**: Kept for backward compatibility and cloud deployments

### 3. Scripts
- **scripts/start-podman.sh**: New script to start services with Podman
- **scripts/stop-podman.sh**: New script to stop Podman services
- **scripts/kill_knirv.sh**: Updated to clean both Podman and Docker resources

### 4. Package.json
Added new npm scripts for Podman operations:
- `npm run podman:start` - Start services with Podman
- `npm run podman:stop` - Stop Podman services
- `npm run podman:restart` - Restart Podman services
- `npm run podman:logs` - View service logs
- `npm run podman:status` - Check container status
- `npm run build:podman` - Install Podman + dependencies

## 🛠️ Installation & Setup

### Prerequisites
1. **Install Podman**:
   ```bash
   # Ubuntu/Debian
   sudo apt-get install podman
   
   # RHEL/CentOS/Fedora
   sudo dnf install podman
   
   # macOS
   brew install podman
   ```

2. **Install podman-compose**:
   ```bash
   pip3 install podman-compose
   ```

### Rootless Configuration
Podman runs rootless by default, which provides better security. No additional configuration is needed for basic usage.

## 🚀 Usage

### Local Development with Podman
```bash
# Start all services
npm run podman:start
# or
./scripts/start-podman.sh

# Check status
npm run podman:status

# View logs
npm run podman:logs

# Stop services
npm run podman:stop
# or
./scripts/stop-podman.sh
```

### Traditional Docker (still supported)
```bash
# Start with Docker
docker-compose up -d

# Stop with Docker
docker-compose down
```

### Cloud Deployment (Render.com)
The Render deployment continues to use Docker since Render.com doesn't support Podman directly.

## 🔧 Key Differences: Podman vs Docker

| Feature | Docker | Podman |
|---------|--------|--------|
| **Root Access** | Requires root/sudo | Rootless by default |
| **Daemon** | Requires Docker daemon | Daemonless |
| **Security** | Runs as root | Runs as user |
| **Systemd** | Limited support | Native systemd support |
| **Compose** | docker-compose | podman-compose |
| **Commands** | docker | podman |

## 🌐 Service URLs

When running with Podman, services are available at:

- **IPFS API**: http://localhost:5001
- **IPFS Gateway**: http://localhost:8080
- **IPFS Swarm**: tcp://localhost:4001
- **KNIRV Oracle**: http://localhost:1317
- **KNIRV Chain**: http://localhost:8090
- **KNIRV Graph**: http://localhost:8082
- **KNIRV Nexus**: http://localhost:8084
- **KNIRV Router**: http://localhost:8086
- **KNIRV Testnet Gateway**: http://localhost:10000

### IPFS Configuration
The IPFS node is automatically configured for the KNIRV network with:
- CORS enabled for web applications
- Server profile for optimal performance
- Custom agent version suffix for network identification
- Proper connection management settings

**Note**: The testnet gateway is the built-in KNIRVTESTNET Express.js server that provides web applications and API endpoints. This is different from the external KNIRVGATEWAY which uses Netlify-CLI.

## 🔍 Troubleshooting

### Common Issues

1. **Permission Denied on Volumes**:
   ```bash
   # Fix SELinux context
   sudo setsebool -P container_manage_cgroup true
   ```

2. **Port Conflicts**:
   ```bash
   # Check what's using ports
   podman ps
   netstat -tulpn | grep :8080
   ```

3. **podman-compose not found**:
   ```bash
   # Install via pip
   pip3 install podman-compose
   ```

### Debugging Commands
```bash
# List all containers
podman ps -a

# View container logs
podman logs <container_name>

# Inspect container
podman inspect <container_name>

# Clean up everything
podman system prune -a
```

## 🔄 Migration Path

### For Existing Docker Users
1. Install Podman and podman-compose
2. Use the new Podman scripts: `npm run podman:start`
3. Verify services are running: `npm run podman:status`
4. Continue development as normal

### For New Users
1. Install Podman (recommended for better security)
2. Clone the repository
3. Run: `npm run podman:start`

## 📝 Notes

- **Render Deployment**: Still uses Docker since Render.com doesn't support Podman
- **Backward Compatibility**: Docker commands still work for local development
- **Security**: Podman provides better security with rootless containers
- **Performance**: Podman typically has lower overhead than Docker

## 🤝 Contributing

When contributing to KNIRVTESTNET:
- Test changes with both Podman and Docker
- Update both Containerfile and Dockerfile if needed
- Ensure scripts work in both environments
- Document any new Podman-specific features
