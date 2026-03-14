# KNIRV-SERVER Container Deployer

The **container_deployer** is a specialized Go-based orchestration tool that deploys KNIRV-SERVER applications in containerized environments using Kata Containers with custom Kali Linux kernels and rootfs images.

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Deployment Types](#deployment-types)
- [Configuration](#configuration)
- [Usage](#usage)
- [Directory Structure](#directory-structure)
- [Troubleshooting](#troubleshooting)
- [Development](#development)

## Overview

The container_deployer provides two deployment modes:

1. **Local Development**: Runs KNIRV-SERVER in Docker/Podman with Debian + Kali security tools
2. **Cloud Production**: Deploys KNIRV-SERVER to cloud infrastructure (AWS, Azure, GCP) using Kata Containers for VM-level isolation

### Container Runtime Architecture

KNIRV-SERVER uses different container strategies for different environments:

**Development (Docker/Podman + Debian + Kali Tools)**
- **Base Image**: `debian:bookworm-slim` with Kali security tools installed
- **Runtime**: Native Go runtime using installed security tools
- **Isolation**: Filesystem sandboxing + security monitoring
- **Purpose**: Development, testing, security research
- **Security**: Detection-based (strace, tcpdump, forensics)

**Production (Kata Containers + VM Isolation)**
- **Base Image**: Custom Kali Linux kernel + rootfs (built by os_builder)
- **Runtime**: Kata Containers (VM-based isolation)
- **Isolation**: Full VM boundaries + namespace isolation
- **Purpose**: Production, untrusted code execution
- **Security**: Prevention-based (hardware isolation)

### Why Different Runtimes?

**Docker/Podman with Debian + Kali Tools (Development)**:
- ✅ Fast iteration and debugging
- ✅ Access to Kali security tools (strace, radare2, semgrep, gdb, tcpdump)
- ✅ Works in containerized environments (Docker-in-Docker)
- ✅ Native Go runtime for monitored execution
- ⚠️ Limited isolation (monitoring vs prevention)
- 📋 Use Case: Local development, CI/CD, security analysis

**Kata Containers with Custom Kali (Production)**:
- ✅ Hardware-enforced VM isolation
- ✅ Full namespace + cgroup isolation
- ✅ Minimal attack surface
- ⚠️ Requires hardware virtualization
- ⚠️ Higher resource overhead
- 📋 Use Case: Production deployments, multi-tenant environments

### KNIRV-SERVER Runtime Selection

The KNIRV-SERVER backend automatically detects the environment and selects the appropriate runtime:

1. **Detects OS**: Checks `/etc/os-release` for "kali" or "debian"
2. **Checks Tools**: Verifies security tools (strace, radare2, semgrep, gdb) are installed
3. **Selects Runtime**:
   - If Kali or Debian with tools → Native Go Runtime (monitoring-based)
   - If tools unavailable → Attempts Podman (namespace isolation)
   - If Podman unavailable → Disables containerization (development mode)

This allows KNIRV-SERVER to run in development containers while still providing the DVE (Distributed Validation Environment) functionality needed for the network

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Container Deployer                        │
│  ┌────────────────┐              ┌────────────────┐         │
│  │ Local Deploy   │              │  Cloud Deploy  │         │
│  │  (Development) │              │  (Production)  │         │
│  └────────┬───────┘              └────────┬───────┘         │
│           │                               │                 │
│           v                               v                 │
│  ┌─────────────────┐              ┌──────────────────┐      │
│  │  Kata Runtime   │              │ Kata Runtime     │      │
│  │  + Kali Kernel  │              │ + Kali Kernel    │      │
│  └────────┬────────┘              └─────────┬────────┘      │
│           │                                 │                │
│           v                                 v                │
│  ┌─────────────────┐              ┌──────────────────┐      │
│  │ Docker/Podman   │              │ Kubernetes/ECS   │      │
│  │  (Local)        │              │  (Cloud)         │      │
│  └─────────────────┘              └──────────────────┘      │
└─────────────────────────────────────────────────────────────┘
```

### Component Interaction

```
os_builder (builds) → Kali Kernel + Rootfs
                          ↓
                  Kata Configuration
                          ↓
              container_deployer (uses)
                          ↓
                  Kata Container Runtime
                          ↓
              KNIRV-SERVER Application
```

## Prerequisites

### Required Software

1. **Go 1.21+**
   ```bash
   go version
   ```

2. **Docker**
   ```bash
   docker --version
   docker info
   ```

3. **Kata Containers Runtime**
   ```bash
   kata-runtime --version
   ```
   - Installation: https://github.com/kata-containers/kata-containers/blob/main/docs/install/README.md

4. **Ansible** (for deployment automation)
   ```bash
   ansible --version
   ```

### Required Artifacts

Before using container_deployer, you must build the Kali Linux artifacts using `os_builder`:

```bash
# Build os_builder
make os

# Run os_builder and select "Build Kata Container"
cd backend/cmd/os_builder
./os_builder
```

This creates:
- Custom Kali kernel: `~/.local/share/knirvserver/os_builder/artifacts/output-kata-guest/vmlinuz-kali-clean-tee`
- Custom Kali rootfs: `~/.local/share/knirvserver/os_builder/artifacts/output-kata-guest/kata-rootfs-kali-clean-tee.img`

### System Requirements

- **CPU**: x86_64 with virtualization support (Intel VT-x or AMD-V)
- **Memory**: Minimum 4GB RAM (8GB+ recommended)
- **Storage**: 10GB free space for artifacts and images
- **OS**: Linux (Ubuntu 20.04+, Debian 11+, or similar)

## Quick Start

### 1. Automated Setup (Recommended)

Run the automated configuration script to set up your local environment:

```bash
# Check current configuration
./scripts/local-dev-config.sh --check-only

# Automatically configure everything
./scripts/local-dev-config.sh --auto-fix

# Interactive configuration (asks before making changes)
./scripts/local-dev-config.sh
```

The script will:
- ✅ Check Docker and Kata runtime installations
- ✅ Verify os_builder artifacts exist
- ✅ Create custom Kata configuration for Kali Linux
- ✅ Update Docker daemon.json to register kata-runtime
- ✅ Restart Docker daemon if needed

### 2. Manual Setup

If you prefer manual configuration:

#### Step 1: Add Kata Runtime to Docker

Edit `/etc/docker/daemon.json`:

```json
{
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "100m",
    "max-file": "5"
  },
  "runtimes": {
    "kata-runtime": {
      "path": "/usr/bin/kata-runtime"
    }
  }
}
```

Restart Docker:
```bash
sudo systemctl restart docker
```

#### Step 2: Verify Setup

```bash
# Check Docker recognizes Kata
docker info | grep kata-runtime

# Check Kata configuration
kata-runtime kata-env
```

### 3. Build and Run

```bash
# Build container_deployer
cd /path/to/KNIRVSERVER
make deployer

# Run container_deployer
cd backend/cmd/container_deployer
./container_deployer

# Follow interactive prompts:
# 1. Select deployment type (Local/Cloud)
# 2. Select action (Deploy new container, etc.)
```

## Deployment Types

### Local Deployment

**Use Case**: Development, testing, debugging

**Features**:
- Runs on your local machine using standard Docker
- Uses Kali Linux base image with required security tools
- Includes: strace, gdb, tcpdump, iptables, AppArmor
- Fast build and deployment (no Kata/QEMU overhead)
- Simple, reliable, works on any Docker-capable system

**Requirements**:
- Docker (standard installation, no special runtime needed)
- No Kata Containers required
- No custom kernel/rootfs needed

**Network**: Uses custom bridge network `knirvserver-local`

**Ports Exposed**:
- `8082`: Backend API
- `8089`: Frontend UI

**Example**:
```bash
./container_deployer
# Select: 1 (Local deployment)
# Select: 1 (Deploy new container)
```

**Note**: For deployments requiring Kata Containers with custom Kali kernel, see [Kata_Issue.md](./Kata_Issue.md) for current status and workarounds.

### Cloud Deployment

**Use Case**: Production, staging, distributed deployments

**Features**:
- Deploys to cloud infrastructure (AWS, Azure, GCP)
- Uses Kata Containers for secure multi-tenant isolation
- Terraform-based infrastructure provisioning
- Auto-scaling and load balancing

**Requirements**:
- Cloud provider account (AWS/Azure/GCP)
- Terraform installed
- Cloud CLI tools (aws-cli, azure-cli, or gcloud)
- Custom Kali artifacts from os_builder

**Example**:
```bash
./container_deployer
# Select: 2 (Cloud deployment)
# Select cloud provider and follow prompts
```

## Configuration

### Kata Configuration

Custom Kata configuration is stored at:
```
~/.config/kata-containers/configuration-kali.toml
```

Key settings:
```toml
[hypervisor.qemu]
kernel = "/home/USER/.local/share/knirvserver/os_builder/artifacts/output-kata-guest/vmlinuz-kali-clean-tee"
image = "/home/USER/.local/share/knirvserver/os_builder/artifacts/output-kata-guest/kata-rootfs-kali-clean-tee.img"
default_vcpus = 1
default_memory = 2048  # MB
machine_type = "q35"
shared_fs = "virtio-9p"
```

### Environment Variables

The deployer respects these environment variables:

- `KATA_CONF_FILE`: Override Kata configuration file path
- `DOCKER_HOST`: Override Docker daemon socket
- `ANSIBLE_VERBOSITY`: Set Ansible output verbosity (0-4)

### Application Data Directory

Runtime data is stored in:
```
~/.local/share/knirvserver/container_deployer/
├── artifacts/
│   └── golang-app-source/    # KNIRV-SERVER application binary
└── resources/
    ├── ansible/
    │   ├── cloud-deploy/     # Cloud deployment playbooks
    │   └── local-deploy/     # Local deployment playbooks
    └── scripts/              # Helper scripts
```

## Usage

### Interactive Mode

Run without arguments for interactive menu:

```bash
./container_deployer
```

Menu structure:
```
KNIRV-SERVER Container Deployment
════════════════════════════════
1. Local deployment (Kata + Docker)
2. Cloud deployment (Kata + Kubernetes)
3. Exit

→ Select deployment type: 1

Local Deployment Options
════════════════════════
1. Deploy new container
2. Stop existing container
3. View container logs
4. Container status
5. Back to main menu

→ Select action:
```

### Command-Line Flags

Future versions will support CLI flags for automation:

```bash
# Coming soon
./container_deployer --type local --action deploy
./container_deployer --type cloud --provider aws --region us-east-1
```

### Deployment Workflow

1. **Pre-deployment checks**:
   - Verify Docker/Kata installation
   - Check for os_builder artifacts
   - Validate Kata configuration

2. **Build phase**:
   - Build container image with KNIRV-SERVER binary
   - Tag image appropriately

3. **Deploy phase**:
   - Stop existing container (if any)
   - Create/verify network configuration
   - Launch Kata container with Kali kernel/rootfs
   - Wait for container to be healthy

4. **Post-deployment**:
   - Display container logs
   - Show access information (URLs, ports)
   - Provide management commands

### Viewing Logs

```bash
# Real-time logs
docker logs -f knirvserver-go-local

# Last 100 lines
docker logs --tail 100 knirvserver-go-local

# Logs since timestamp
docker logs --since "2025-01-01T00:00:00" knirvserver-go-local
```

### Managing Containers

```bash
# List running containers
docker ps | grep knirvserver

# Stop container
docker stop knirvserver-go-local

# Remove container
docker rm knirvserver-go-local

# Inspect container
docker inspect knirvserver-go-local

# Execute command in container
docker exec -it knirvserver-go-local /bin/bash
```

## Directory Structure

```
container_deployer/
├── main.go                      # Main application entry point
├── README.md                    # This file
├── container_deployer           # Compiled binary (after build)
│
├── ansible/                     # Ansible playbooks (embedded)
│   ├── cloud-deploy/
│   │   ├── deploy-kata-app.yml
│   │   ├── inventory.ini
│   │   └── containerfile.j2
│   └── local-deploy/
│       ├── deploy-docker-app.yml    # Local deployment with Kata
│       ├── inventory.ini            # Localhost inventory
│       └── containerfile.j2         # Container image definition
│
├── scripts/                         # Utility scripts (embedded)
│   ├── local-dev-config.sh         # Environment setup script
│   └── cleanup.sh                  # Cleanup helper
│
├── golang-app-source/              # Application binaries (embedded)
│   └── knirv-server                 # Main application binary
│
├── container.yaml                  # Container spec (embedded)
└── pod.yaml                        # Pod spec for cloud (embedded)
```

### Embedded Files

The deployer uses Go's `embed` package to include required files in the binary:

```go
//go:embed all:ansible/*
//go:embed all:scripts/*
//go:embed golang-app-source/knirv-server
//go:embed container.yaml
//go:embed pod.yaml
var embeddedFiles embed.FS
```

This means the compiled binary is self-contained and portable.

## Troubleshooting

### Common Issues

#### 1. Docker Bridge Network Error

**Error**: `adding interface to bridge docker0 failed: Device does not exist`

**Solution**:
```bash
# Restart Docker daemon
sudo systemctl restart docker

# Verify docker0 exists
ip link show docker0

# If still failing, use custom network (deployer does this automatically)
docker network create knirvserver-local
```

#### 2. Permission Denied Running Binary

**Error**: `exec /app/knirv-server: permission denied`

**Cause**: Binary doesn't have execute permissions

**Solution**: The deployer automatically fixes this, but if you see this error:
```bash
# Check permissions
ls -l ~/.local/share/knirvserver/container_deployer/artifacts/golang-app-source/knirv-server

# Fix permissions
chmod +x ~/.local/share/knirvserver/container_deployer/artifacts/golang-app-source/knirv-server
```

#### 3. Kata Runtime Not Found

**Error**: `kata-runtime: command not found`

**Solution**:
```bash
# Install Kata Containers
# Ubuntu/Debian:
bash -c "$(curl -fsSL https://raw.githubusercontent.com/kata-containers/kata-containers/main/utils/kata-manager.sh)"

# Verify installation
kata-runtime --version
```

#### 4. Missing Kali Artifacts

**Error**: `golang app source not found` or `Kali kernel not found`

**Solution**:
```bash
# Build artifacts using os_builder
make os
cd backend/cmd/os_builder
./os_builder

# Select: "Build Kata Container (Terraform)"
# Wait for build to complete

# Verify artifacts exist
ls -lh ~/.local/share/knirvserver/os_builder/artifacts/output-kata-guest/
```

#### 5. Container Exits Immediately

**Error**: Container starts but exits with code 255

**Check**:
```bash
# View container logs
docker logs knirvserver-go-local

# Common causes:
# - Binary not found (check golang-app-source/)
# - Missing dependencies in Kali rootfs
# - Configuration errors in knirv-server
```

**Debug**:
```bash
# Run container interactively
docker run -it --rm --runtime kata-runtime \
  --network knirvserver-local \
  -p 8082:8082 -p 8089:8089 \
  knirvserver-go-app:latest /bin/bash

# Test binary manually
/app/knirv-server --help
```

#### 6. Port Already in Use

**Error**: `port is already allocated`

**Solution**:
```bash
# Find process using port
sudo lsof -i :8082
sudo lsof -i :8089

# Kill existing container
docker stop knirvserver-go-local
docker rm knirvserver-go-local

# Or use different ports (modify playbook)
```

### Debug Mode

Enable verbose logging:

```bash
# Set environment variable before running
export ANSIBLE_VERBOSITY=3
./container_deployer

# Or edit main.go temporarily:
// In runCmd function, add:
log.Printf("Executing: %s %v", name, args)
```

### Health Checks

```bash
# Check if container is running
docker ps | grep knirvserver

# Check container resource usage
docker stats knirvserver-go-local

# Check Kata VM info
kata-runtime kata-env

# Verify network connectivity
curl http://localhost:8082/health
curl http://localhost:8089
```

### Getting Help

1. **Check logs**: `docker logs knirvserver-go-local`
2. **Run configuration check**: `./scripts/local-dev-config.sh --check-only`
3. **Verify prerequisites**: Ensure all required software is installed
4. **Check GitHub issues**: https://github.com/KNIRV/KNIRV_NETWORK/issues
5. **Community support**: Join KNIRV Discord/Slack

## Development

### Building from Source

```bash
# From project root
make deployer

# Or manually
cd backend/cmd/container_deployer
go build -o container_deployer main.go
```

### Adding New Features

The deployer is structured around:

1. **Interactive menu system** (`showDeploymentTypeMenu`, `showLocalDeploymentMenu`)
2. **Deployment executors** (`runDeployNewContainer`, `runInstallGoAppOnly`)
3. **Helper functions** (`runCmd`, `ensureGoAppSourceInArtifacts`)
4. **Embedded resources** (Ansible playbooks, scripts, binaries)

To add a new deployment type:

1. Create new Ansible playbooks in `ansible/your-type/`
2. Add menu option in `showDeploymentTypeMenu()`
3. Create executor function (e.g., `runYourTypeDeployment()`)
4. Update embedded files list if needed

### Testing

```bash
# Unit tests (when available)
go test ./...

# Integration test
./container_deployer  # Run through full deployment

# Cleanup after testing
docker stop knirvserver-go-local
docker rm knirvserver-go-local
docker rmi knirvserver-go-app:latest
```

### Code Structure

Key functions:

- `main()`: Entry point, initializes application
- `extractEmbeddedFiles()`: Extracts embedded resources to filesystem
- `showDeploymentTypeMenu()`: Displays deployment type selection
- `executeActionWithDeployType()`: Routes to appropriate deployment handler
- `runDeployNewContainer()`: Handles new container deployment
- `ensureGoAppSourceInArtifacts()`: Manages application binary
- `runCmd()`: Executes external commands (Ansible, Docker, etc.)

### Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test thoroughly
5. Submit a pull request

## License

See project root LICENSE file.

## Related Documentation

- [os_builder README](../os_builder/README.md) - Building Kali artifacts
- [KNIRVSERVER Architecture](../../../docs/architecture.md) - Overall system design
- [Kata Containers Documentation](https://github.com/kata-containers/kata-containers/tree/main/docs)
- [Docker Documentation](https://docs.docker.com/)
- [Ansible Documentation](https://docs.ansible.com/)

---

**Last Updated**: 2025-12-31
**Version**: 1.0.0
**Maintainer**: KNIRV Network Team
