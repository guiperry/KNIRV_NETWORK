# KNIRV-NEXUS: Decentralized Validation Environment

KNIRV-NEXUS is a production-ready implementation of the Decentralized Validation Environment (DVE) for the KNIRV Network. It provides a secure, scalable, and distributed platform for validating SkillNodes and Base LLMs using Trusted Execution Environments (TEE) and P2P networking.

## 🏗️ Architecture Overview

KNIRV-NEXUS implements a microservices architecture running on Kubernetes with the following core components:

### Core Services

- **DVE Manager**: Orchestrates DVE nodes, manages task allocation, and monitors system health
- **Validation Core**: Executes validation tasks with TEE support and cryptographic proofs

> **Note**: KNIRV-NEXUS integrates with the primary KNIRVGATEWAY for API routing and frontend hosting. See `NEXUS_GATEWAY_MIGRATION.md` for integration details.

### Frontend Technology Stack

- **Framework**: Next.js 15 with App Router
- **UI Components**: shadcn/ui built on Radix UI primitives
- **Styling**: Tailwind CSS 4 with custom KNIRV theme
- **Real-time**: Socket.io for live updates and notifications
- **State Management**: React hooks and context
- **Type Safety**: TypeScript with strict configuration
- **Database**: Prisma ORM for data modeling
- **Authentication**: NextAuth.js integration ready

### Backend Infrastructure

- **Base OS**: Kali Linux (as specified in KALI_LINUX_FOUNDATION.md)
- **Container Runtime**: Podman (rootless containers)
- **Orchestration**: Kubernetes with production-ready configurations
- **Database**: BuntDB (embedded key-value store with custom indexes)
- **Networking**: libp2p (aligned with KNIRV-ORACLE protocols)
- **Configuration**: Viper for professional configuration management

## 🚀 Features

### Frontend (Next.js with shadcn/ui)
- **Modern UI Framework**: Next.js 15 with App Router
- **Component Library**: shadcn/ui components built on Radix UI
- **Real-time Updates**: Socket.io integration for live data
- **Responsive Design**: Mobile-first design with Tailwind CSS
- **Type Safety**: Full TypeScript implementation
- **Role-based Access**: Dynamic UI based on user permissions

### DVE Node Management
- Node registration with TEE type and capabilities
- Health monitoring with heartbeat tracking
- Load balancing with multiple algorithms (reputation, resource, geographic)
- Geographic distribution support

### Validation Engine
- SkillNode validation with test case execution
- Base LLM validation framework
- Custom validation types support
- Cryptographic proof generation

### P2P Networking
- KNIRV-ORACLE aligned libp2p implementation
- DHT-based node discovery
- GossipSub message distribution
- Network topology monitoring

### Security & TEE
- Software TEE simulation for development
- Hardware TEE support framework (SGX, SEV-SNP, TDX)
- Attestation and proof verification
- Secure key management

### API & Real-time Updates
- RESTful API with JWT authentication
- Server-Sent Events for real-time updates
- Role-based access control
- Report generation and sharing

### Operational Modes
- **Headless Mode**: Production deployment without GUI (default)
- **GUI Mode**: Local admin interface with built-in web dashboard (`-gui` flag)
- **Configuration Management**: Viper-based configuration with YAML files
- **Role-based Permissions**: Configurable user roles and access control
## 🚀 Quick Start

### Prerequisites

- Kubernetes cluster (v1.25+) for production deployment
- kubectl configured
- Docker or Podman for containerization
- Go 1.21+ (required for backend development)
- Node.js 18+ and npm (for frontend development)

### Quick Backend Development Setup

**Get started with backend development in 5 minutes:**

```bash
# 1. Clone and navigate to backend
git clone https://github.com/knirv/KNIRV_NETWORK.git
cd KNIRV_NETWORK/KNIRVNEXUS

# 2. Set up development environment
export $(cat .env.development | xargs)

# 3. Build and run DVE Manager
cd backend
mkdir -p data
go mod tidy
go build -o bin/dve-manager ./cmd/dve-manager/
./bin/dve-manager --gui

# 4. Access GUI at http://localhost:9080
```

**For production deployment:**

### Deployment

1. **Clone the repository**:
   ```bash
   git clone https://github.com/knirv/KNIRV_NETWORK.git
   cd KNIRV_NETWORK/KNIRVNEXUS
   ```

2. **Build the components**:
   ```bash
   chmod +x scripts/build.sh
   ./scripts/build.sh
   ```

3. **Deploy to Kubernetes**:
   ```bash
   chmod +x scripts/deploy.sh
   ./scripts/deploy.sh
   ```

4. **Verify deployment**:
   ```bash
   kubectl get pods -n knirv-nexus
   ```

### Development Setup

#### Frontend Development
1. **Install frontend dependencies**:
   ```bash
   npm install
   ```

2. **Start development server**:
   ```bash
   npm run dev
   ```

3. **Build frontend**:
   ```bash
   npm run build
   ```

4. **Start custom server with Socket.io**:
   ```bash
   npm run start
   ```

#### Backend Development

##### Prerequisites
- Go 1.21+ installed
- Git for version control
- Optional: Docker for containerized builds

##### Environment Setup

1. **Set up environment variables**:
   ```bash
   # For development (GUI mode, no auth required)
   cp .env.development .env

   # OR for production (headless mode, auth required)
   cp .env.production .env
   # Then set JWT_SECRET environment variable:
   export JWT_SECRET="your-secure-jwt-secret-here"
   ```

2. **Install backend dependencies**:
   ```bash
   cd backend
   go mod tidy
   ```

3. **Create required directories**:
   ```bash
   mkdir -p data logs reports keys
   ```

##### Building the Backend

1. **Quick development build**:
   ```bash
   cd backend

   # Build DVE Manager
   go build -o bin/dve-manager ./cmd/dve-manager/

   # Build Validation Core
   go build -o bin/validation-core ./cmd/validation-core/

   # Build API Gateway (if available)
   go build -o bin/api-gateway ./cmd/api-gateway/
   ```

2. **Production build (static linking)**:
   ```bash
   cd backend

   # Build with static linking for containers
   CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo \
       -ldflags '-extldflags "-static" -s -w' \
       -o bin/dve-manager ./cmd/dve-manager/

   CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo \
       -ldflags '-extldflags "-static" -s -w' \
       -o bin/validation-core ./cmd/validation-core/
   ```

3. **Using the build script**:
   ```bash
   # Full build with Docker images
   ./scripts/build.sh

   # Build binaries only (skip Docker)
   ./scripts/build.sh --skip-docker

   # Skip tests and security checks
   ./scripts/build.sh --skip-tests --skip-security
   ```

##### Running the Services

1. **Development mode (with GUI)**:
   ```bash
   cd backend

   # Load development environment
   export $(cat ../.env.development | xargs)

   # Run DVE Manager with GUI
   ./bin/dve-manager --gui

   # In another terminal, run Validation Core with GUI
   ./bin/validation-core --gui
   ```

2. **Production mode (headless)**:
   ```bash
   cd backend

   # Load production environment
   export $(cat ../.env.production | xargs)
   export JWT_SECRET="your-secure-jwt-secret"

   # Run services
   ./bin/dve-manager
   ./bin/validation-core
   ```

3. **With custom configuration**:
   ```bash
   ./bin/dve-manager --config ./config/custom.yaml --port 8080
   ./bin/validation-core --config ./config/custom.yaml --port 8081
   ```

##### Testing

1. **Run unit tests**:
   ```bash
   cd backend
   go test ./tests/... -v
   ```

2. **Run integration tests**:
   ```bash
   cd backend
   go test ./tests/integration_test.go -v
   ```

3. **Run with coverage**:
   ```bash
   cd backend
   go test ./tests/... -v -coverprofile=coverage.out
   go tool cover -html=coverage.out -o coverage.html
   ```

#### Full Stack Development
1. **Build frontend first**:
   ```bash
   npm run build
   ```

2. **Start backend with GUI mode**:
   ```bash
   cd backend
   ./dve-manager -gui
   ./validation-core -gui
   ```

3. **Access the application**:
   - Frontend: http://localhost:3000 (development)
   - GUI Mode: http://localhost:9080 (DVE Manager), http://localhost:9081 (Validation Core)
   - API: http://localhost:8080 (DVE Manager), http://localhost:8081 (Validation Core)

## 📁 Project Structure

```
KNIRVNEXUS/
├── src/                        # Next.js Frontend
│   ├── app/                   # Next.js App Router
│   │   ├── layout.tsx         # Root layout
│   │   ├── page.tsx           # Home page
│   │   ├── globals.css        # Global styles
│   │   └── api/               # API routes
│   ├── components/            # React components
│   │   └── ui/                # shadcn/ui components
│   ├── hooks/                 # Custom React hooks
│   │   ├── use-knirv-socket.ts # Socket.io integration
│   │   ├── use-mobile.ts      # Mobile detection
│   │   └── use-toast.ts       # Toast notifications
│   └── lib/                   # Utility libraries
│       ├── db.ts              # Database utilities
│       ├── socket.ts          # Socket.io client
│       └── utils.ts           # General utilities
├── backend/                    # Go backend services
│   ├── cmd/                   # Service entry points
│   │   ├── dve-manager/       # DVE Manager service
│   │   └── validation-core/   # Validation Core service
│   ├── internal/              # Internal packages
│   │   ├── config/           # Configuration management
│   │   ├── database/         # BuntDB wrapper
│   │   ├── models/           # Data models
│   │   └── services/         # Business logic services
│   ├── pkg/                  # Public packages
│   │   ├── p2p/             # P2P networking
│   │   └── sse/             # Server-Sent Events
│   ├── tests/               # Test suites
│   └── Dockerfile.*         # Container definitions
├── k8s/                     # Kubernetes manifests
│   ├── namespace.yaml       # Namespace and quotas
│   ├── configmap.yaml       # Configuration
│   ├── secrets.yaml         # Secrets management
│   └── *-deployment.yaml    # Service deployments
├── scripts/                 # Automation scripts
│   ├── build.sh            # Build automation
│   └── deploy.sh           # Deployment automation
├── public/                  # Static assets
│   ├── logo.svg            # KNIRV logo
│   └── robots.txt          # SEO configuration
├── prisma/                  # Database schema
│   └── schema.prisma       # Prisma schema definition
├── config/                  # Configuration files
│   ├── knirv-nexus.yaml    # Main configuration
│   ├── development.yaml    # Development config
│   └── production.yaml     # Production config
├── package.json            # Frontend dependencies
├── next.config.ts          # Next.js configuration
├── tailwind.config.ts      # Tailwind CSS configuration
├── components.json         # shadcn/ui configuration
├── server.ts              # Custom server with Socket.io
├── tsconfig.json          # TypeScript configuration
└── README.md              # This file
```

## 🔧 API Documentation

> **Note**: KNIRV-NEXUS APIs are accessed through the primary KNIRVGATEWAY at `/api/nexus/*` endpoints. Direct service access is available for development and internal communication.

### Direct Service APIs

KNIRV-NEXUS exposes two main services with direct API access:

#### DVE Manager Service (Port 8080)

```bash
# Health check
curl -X GET http://dve-manager:8080/health

# Register a new DVE node
curl -X POST http://dve-manager:8080/api/v1/nodes \
  -H "Content-Type: application/json" \
  -d '{
    "name": "validator-1",
    "tee_type": "sgx",
    "stake_amount": 1000000,
    "location": "us-east-1",
    "ip_address": "192.168.1.100",
    "public_key": "...",
    "capabilities": ["skillnode", "base_llm"]
  }'

# List DVE nodes
curl -X GET http://dve-manager:8080/api/v1/nodes

# System health
curl -X GET http://dve-manager:8080/api/v1/system/health
```

#### Validation Core Service (Port 8081)

```bash
# Health check
curl -X GET http://validation-core:8081/health

# Create validation task
curl -X POST http://validation-core:8081/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "type": "skillnode",
    "priority": 5,
    "skill_code": "def hello(): return \"Hello, World!\"",
    "test_cases": [...],
    "required_tee_type": "sgx"
  }'

# List validation tasks
curl -X GET http://validation-core:8081/api/v1/tasks

# Get validation results
curl -X GET http://validation-core:8081/api/v1/results
```

### Gateway Integration

For production use, access KNIRV-NEXUS through KNIRVGATEWAY:

```bash
# Via KNIRVGATEWAY (Production)
curl -X GET https://gateway.knirv.network/api/nexus/nodes
curl -X POST https://gateway.knirv.network/api/nexus/tasks

# Real-time updates via SSE
const eventSource = new EventSource('https://gateway.knirv.network/api/nexus/sse');
eventSource.addEventListener('nexus-nodes', function(event) {
  const data = JSON.parse(event.data);
  console.log('Node update:', data);
});
```

## 🔧 Operational Modes

KNIRV-NEXUS supports two operational modes for different deployment scenarios:

### Headless Mode (Default - Production)

**Use Case**: Production deployments, Kubernetes clusters, cloud environments

```bash
# Default headless mode
./dve-manager
./validation-core

# With configuration file
./dve-manager --config ./config/production.yaml
```

**Characteristics**:
- API-only access (no web interface)
- Full JWT authentication required
- Binds to all network interfaces (0.0.0.0)
- Production-optimized resource usage
- Comprehensive audit logging
- Kubernetes-ready with health checks

### GUI Mode (Local Administration)

**Use Case**: Local development, system administration, debugging

```bash
# Enable GUI mode
./dve-manager -gui
./validation-core -gui

# With custom configuration
./dve-manager -gui --config ./config/development.yaml
```

**Characteristics**:
- Built-in web interface using existing Next.js frontend
- No authentication required (admin environment assumed)
- Localhost-only access (127.0.0.1)
- Real-time updates via WebSocket
- Extended debugging and diagnostic tools
- Direct access to configuration management

**GUI Access**:
- DVE Manager GUI: http://localhost:9080
- Validation Core GUI: http://localhost:9081
- API Access: http://localhost:8080, http://localhost:8081

### Mode Comparison

| Feature | Headless Mode | GUI Mode |
|---------|---------------|----------|
| **Target Use** | Production | Local admin |
| **Authentication** | JWT required | None |
| **Web Interface** | None | Built-in Next.js frontend |
| **Network Access** | All interfaces | Localhost only |
| **Resource Usage** | Minimal | Higher (includes web server) |
| **Security** | Full RBAC + audit | Local access only |

## ⚙️ Configuration

### Configuration Management (Viper)

KNIRV-NEXUS uses [Viper](https://github.com/spf13/viper) for professional configuration management with the following hierarchy (highest to lowest precedence):

1. **CLI Flags**: `--gui`, `--port`, `--config`
2. **Environment Variables**: `KNIRV_GUI_ENABLED`, `KNIRV_SERVICE_PORT`
3. **Configuration File**: `config/knirv-nexus.yaml`
4. **Default Values**: Hardcoded sensible defaults

### Configuration File Structure

```yaml
# config/knirv-nexus.yaml
mode: headless  # headless | gui

service:
  port: 8080
  bind_address: "0.0.0.0"
  name: "dve-manager"

gui:
  enabled: false
  port: 9080
  frontend_path: "./dist"

security:
  auth_required: true
  tls_enabled: true
  audit_logging: true
  jwt_secret: "${KNIRV_JWT_SECRET}"

roles:
  validator:
    permissions: ["node:read", "node:update", "tasks:read", "results:read"]
    scoped_access: true
  admin:
    permissions: ["*:*"]
    scoped_access: false
  observer:
    permissions: ["*:read"]
    scoped_access: false

database:
  type: "buntdb"
  path: "./data/nexus.db"

network:
  chain_id: "knirv-nexus-mainnet"
  p2p_port: 4001
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `KNIRV_MODE` | Operational mode | `headless` |
| `KNIRV_GUI_ENABLED` | Enable GUI mode | `false` |
| `KNIRV_SERVICE_PORT` | API server port | `8080` |
| `KNIRV_GUI_PORT` | GUI server port | `9080` |
| `KNIRV_CHAIN_ID` | Blockchain network ID | `knirv-nexus-mainnet` |
| `KNIRV_DATABASE_PATH` | Database file path | `./data/nexus.db` |
| `KNIRV_P2P_PORT` | P2P networking port | `4001` |
| `KNIRV_LOG_LEVEL` | Logging level | `info` |
| `KNIRV_JWT_SECRET` | JWT signing secret | Required |

### Kubernetes Configuration

Key configuration files:

- `k8s/namespace.yaml`: Namespace and resource quotas
- `k8s/configmap.yaml`: Application configuration
- `k8s/secrets.yaml`: Sensitive configuration (JWT secrets, TLS certs)
- `k8s/*-deployment.yaml`: Service deployments

## 🔒 Security

### TEE Support

KNIRV-NEXUS supports multiple TEE technologies:

- **Intel SGX**: Hardware-based secure enclaves
- **AMD SEV-SNP**: Secure encrypted virtualization
- **Intel TDX**: Trust domain extensions
- **Software TEE**: Simulation for development/testing

### Network Security

- **P2P Encryption**: All P2P communications are encrypted
- **TLS Termination**: HTTPS/TLS for all API endpoints
- **JWT Authentication**: Secure token-based authentication
- **RBAC**: Role-based access control

### Container Security

- **Rootless Containers**: Podman-based rootless execution
- **Minimal Base Images**: Hardened Kali Linux base
- **Security Scanning**: Automated vulnerability scanning
- **Network Policies**: Kubernetes network isolation

## 📊 Monitoring and Observability

### Metrics

KNIRV-NEXUS exposes Prometheus metrics on `/metrics` endpoints:

- System health and performance metrics
- DVE node status and resource utilization
- Validation task throughput and success rates
- P2P network topology and latency

### Logging

Structured JSON logging with configurable levels:

```bash
# View logs
kubectl logs -f deployment/api-gateway -n knirv-nexus
kubectl logs -f deployment/dve-manager -n knirv-nexus
kubectl logs -f deployment/validation-core -n knirv-nexus
```

### Health Checks

All services provide health check endpoints:

- `GET /health`: Basic health status
- Kubernetes liveness and readiness probes
- Automatic service recovery and scaling

## 🧪 Testing

The implementation includes comprehensive testing:

- **Unit Tests**: Individual component testing
- **Integration Tests**: End-to-end service testing
- **Performance Tests**: Validation throughput benchmarks
- **Security Tests**: Vulnerability scanning

Run tests with:
```bash
cd backend
go test ./tests/... -v
```

## 🛠️ Troubleshooting

### Common Backend Issues

#### 1. JWT Secret Configuration Error
**Error**: `Failed to load configuration: invalid configuration: security.jwt_secret is required in headless mode with authentication`

**Causes & Solutions**:
- **Environment variable not set**:
  ```bash
  # Set JWT_SECRET environment variable
  export JWT_SECRET="your-secure-jwt-secret-here"

  # Or use development environment
  cp .env.development .env
  export $(cat .env | xargs)
  ```

- **Using GUI mode but auth still required**:
  ```bash
  # Use development environment for GUI mode
  export KNIRV_MODE=gui
  export KNIRV_AUTH_REQUIRED=false

  # Or run with GUI flag
  ./bin/dve-manager --gui
  ```

- **Shell variable expansion issue in .env file**:
  ```bash
  # Instead of KNIRV_JWT_SECRET=${JWT_SECRET}
  # Use direct value: KNIRV_JWT_SECRET=your-actual-secret
  ```

#### 2. Build Failures
**Error**: `go build` fails with dependency issues

**Solutions**:
```bash
cd backend

# Clean module cache
go clean -modcache

# Update dependencies
go mod tidy

# Verify dependencies
go mod verify

# Build with verbose output
go build -v ./cmd/dve-manager/
```

#### 3. Database Connection Issues
**Error**: Database path not accessible

**Solutions**:
```bash
# Create data directory
mkdir -p data

# Check permissions
chmod 755 data

# Use absolute path in config
export KNIRV_DATABASE_PATH="$(pwd)/data/nexus.db"
```

#### 4. Port Already in Use
**Error**: `bind: address already in use`

**Solutions**:
```bash
# Check what's using the port
lsof -i :8080

# Kill process using port
kill -9 $(lsof -t -i:8080)

# Use different port
./bin/dve-manager --port 8081
```

#### 5. P2P Network Issues
**Error**: P2P connection failures

**Solutions**:
```bash
# Check firewall
sudo ufw status

# Allow P2P port
sudo ufw allow 4001

# Check if port is accessible
nc -zv localhost 4001
```

### Configuration Troubleshooting

#### Environment Variable Priority
Configuration is loaded in this order (highest to lowest priority):
1. CLI flags: `--gui`, `--port`, `--config`
2. Environment variables: `KNIRV_*`
3. Configuration file: `config/knirv-nexus.yaml`
4. Default values

#### Debug Configuration Loading
```bash
# Enable debug logging
export KNIRV_LOG_LEVEL=debug

# Run with verbose output
./bin/dve-manager --gui 2>&1 | grep -i config
```

#### Validate Configuration
```bash
# Test configuration without starting service
./bin/dve-manager --config ./config/development.yaml --help
```

### Development Environment Issues

#### 1. Go Version Compatibility
**Error**: Build fails with Go version issues

**Solutions**:
```bash
# Check Go version
go version

# Should be 1.21+
# Update Go if needed
sudo rm -rf /usr/local/go
wget https://golang.org/dl/go1.21.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
```

#### 2. CGO Dependencies
**Error**: CGO compilation fails

**Solutions**:
```bash
# Install build essentials
sudo apt-get update
sudo apt-get install build-essential

# For static builds
sudo apt-get install musl-dev
```

### Kubernetes Deployment Issues

1. **Pod startup failures**: Check resource limits and node capacity
2. **P2P connectivity issues**: Verify firewall rules and port accessibility
3. **Database errors**: Check persistent volume availability
4. **Authentication failures**: Verify JWT secret configuration in secrets

### Debug Commands

```bash
# Check pod status
kubectl get pods -n knirv-nexus -o wide

# View pod logs
kubectl logs -f pod/<pod-name> -n knirv-nexus

# Execute into pod
kubectl exec -it pod/<pod-name> -n knirv-nexus -- /bin/bash

# Check service endpoints
kubectl get endpoints -n knirv-nexus

# View events
kubectl get events -n knirv-nexus --sort-by='.lastTimestamp'
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Run the test suite: `go test ./tests/... -v`
6. Submit a pull request

## 📄 License

This project is part of the KNIRV Network and follows the project's licensing terms.

## 🆘 Support

For support and questions:

- Create an issue in the GitHub repository
- Join the KNIRV Network community discussions
- Review the technical documentation in the codebase

Built with ❤️ for the developer community. Supercharged by [Z.ai](https://chat.z.ai) 🚀
