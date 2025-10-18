# KNIRV-NEXUS: Decentralized Validation Environment

KNIRV-NEXUS is a unified implementation of the Decentralized Validation Environment (DVE) for the KNIRV Network. It provides a secure, scalable platform for validating SkillNodes and Base LLMs using a modern architecture that combines Next.js frontend with Go backend services in a single deployable binary.

## 🏗️ Architecture Overview

KNIRV-NEXUS implements a **unified architecture** with embedded frontend and backend services:

### Unified Binary Architecture

- **Main Wrapper** (`main.go`): Embeds both frontend and backend into a single executable
- **Frontend Embedding**: Next.js build output embedded via `//go:embed all:out`
- **Backend Embedding**: Unified backend binary embedded via `//go:embed bin/backend-server`
- **API Proxy**: Gin-based proxy routing `/api/*` requests to embedded backend
- **Static Serving**: Embedded filesystem serving Next.js static assets

### Core Backend Services

- **DVE Manager**: Orchestrates DVE nodes, manages task allocation, and monitors system health
- **Validation Core**: Executes validation tasks with TEE support and cryptographic proofs
- **Model Server**: Manages WASM plugin models and runtime execution
- **Data Engine**: BuntDB-based data processing, metrics aggregation, and alerting
- **CDE Service**: Cloud Development Environments for isolated execution
- **DNS Service**: Dynamic DNS management for distributed nodes
- **P2P Manager**: libp2p-based networking for node discovery and communication

> **Note**: KNIRV-NEXUS integrates with KNIRVGATEWAY for production routing. The unified architecture enables both standalone deployment and gateway integration.

### Frontend Technology Stack

- **Framework**: Next.js 15 with App Router
- **UI Components**: shadcn/ui built on Radix UI primitives
- **Styling**: Tailwind CSS 4 with custom KNIRV theme
- **Real-time**: Socket.io for live updates and notifications
- **State Management**: React hooks and context
- **Type Safety**: TypeScript with strict configuration
- **Authentication**: Role-based access control with JWT
- **Embedding**: Static build output embedded in Go binary

### Backend Infrastructure

- **Language**: Go 1.21+ with modern concurrency patterns
- **Database**: BuntDB (embedded key-value store with custom indexes)
- **Networking**: libp2p for P2P communication and node discovery
- **Configuration**: Viper for hierarchical configuration management
- **Web Framework**: Gorilla Mux for HTTP routing and middleware
- **Real-time**: WebSocket support for live data streaming
- **Container Runtime**: Podman (rootless containers) for production
- **Orchestration**: Kubernetes with production-ready configurations

## 🧪 Phase 7 Testing Infrastructure

### Testing Architecture
KNIRVNEXUS implements comprehensive testing across multiple layers:

#### Backend Testing (Go)
```bash
# Run backend unit tests
cd backend && go test -v ./tests/...

# Run specific test suites
go test -v ./tests/phase6_comprehensive_unit_test.go
go test -v ./tests/integration_test.go
go test -v ./tests/architecture_test.go
```

#### Frontend Testing (Next.js)
```bash
# Run frontend tests
npm test

# Type checking
npm run type-check

# Linting
npm run lint
```

#### Integration with Network Tests
```bash
# From project root - run KNIRVNEXUS network integration tests
cd integration-tests && go test -v -run TestKNIRVNEXUS

# Run comprehensive KNIRVNEXUS test suite
make test-nexus

# Run specific test categories
make test-nexus-unit          # Unit tests only
make test-nexus-integration   # Integration tests only
make test-nexus-e2e          # End-to-end tests only
make test-nexus-performance  # Performance tests only
make test-nexus-security     # Security tests only
```

## 🛠️ Build System

KNIRV-NEXUS uses a unified build process that creates a single deployable binary containing both frontend and backend:

### Build Process Overview

1. **Frontend Build**: Next.js builds static output to `out/` directory
2. **Backend Build**: Go builds unified backend binary to `bin/backend-server`
3. **Embedding**: Main wrapper embeds both frontend and backend using `go:embed`
4. **Final Binary**: Single executable containing complete application

### Core Build Commands

```bash
# Quick Development Setup
npm install              # Install frontend dependencies
npm run build           # Build Next.js frontend
cd backend && go build -o ../bin/backend-server ./main.go  # Build backend
go build -o knirv-nexus main.go  # Build unified binary

# Development Mode
npm run dev             # Start Next.js dev server (port 3000)
cd backend && go run main.go --config config/development.yaml  # Backend only

# Production Build
npm run build           # Build optimized frontend
cd backend && CGO_ENABLED=1 go build -ldflags="-s -w" -o ../bin/backend-server ./main.go
go build -ldflags="-s -w -X main.Version=v1.0.0" -o knirv-nexus main.go

# Testing
npm test                # Frontend tests
cd backend && go test ./...  # Backend tests
```

### Deployment Options

```bash
# Standalone Binary
./knirv-nexus           # Runs on port 8090 (configurable)

# With Custom Configuration
./knirv-nexus --config config/production.yaml

# Environment Variables
NEXUS_PORT=8080 NEXUS_BACKEND_PORT=8081 ./knirv-nexus

# Docker Deployment
docker build -t knirv/nexus:latest .
docker run -p 8090:8090 knirv/nexus:latest
```

## 🚀 Current Implementation Status

### ✅ Fully Implemented Features

#### Unified Architecture
- **Single Binary Deployment**: Frontend and backend embedded in one executable
- **Embedded Frontend**: Next.js build output served via Go's embed filesystem
- **API Proxy**: Gin-based routing of `/api/*` requests to embedded backend
- **Configuration Management**: Viper-based hierarchical configuration with YAML support

#### Frontend (Next.js with shadcn/ui)
- **Modern UI Framework**: Next.js 15 with App Router and TypeScript
- **Component Library**: Complete shadcn/ui component set built on Radix UI
- **Dashboard Interface**: Comprehensive dashboard with tabs for all major functions
- **Responsive Design**: Mobile-first design with Tailwind CSS and KNIRV theme
- **Authentication UI**: Role-based components and user profile management

#### Backend Services
- **DVE Manager**: Node orchestration and task allocation service
- **Validation Core**: Task queue and validation execution framework
- **Model Server**: WASM plugin model management and runtime execution
- **Data Engine**: BuntDB-based metrics, alerts, and event processing
- **CDE Service**: Cloud Development Environment management
- **DNS Service**: Dynamic DNS management for distributed nodes

#### Database & Storage
- **BuntDB Integration**: Embedded key-value store with custom indexes
- **Data Engine**: Comprehensive metrics aggregation and alerting system
- **Event Processing**: Real-time event ingestion and processing pipeline
- **Report Generation**: User and system report storage and retrieval

### ⚠️ Partially Implemented Features

#### P2P Networking (30% Complete)
- **Basic Structure**: libp2p manager framework in place
- **Missing**: DHT integration, GossipSub messaging, active node discovery
- **Status**: Foundation exists but not operational

#### Validation Engine (40% Complete)
- **Task Management**: Queue system and basic task structures
- **Missing**: Actual validation logic, cryptographic proof generation
- **Status**: Framework ready but core functionality incomplete

#### API Endpoints (50% Complete)
- **Route Structure**: All service routes defined with proper handlers
- **Missing**: Complete CRUD operations, many endpoints return placeholder data
- **Status**: Infrastructure ready but implementations incomplete

### ❌ Missing Critical Features

#### TEE (Trusted Execution Environment)
- **Status**: Only basic type definitions exist
- **Missing**: SGX/SEV-SNP/TDX integration, attestation, secure execution
- **Impact**: Core security guarantees not implemented

#### JWT Authentication System
- **Status**: Frontend auth components exist but no backend implementation
- **Missing**: JWT middleware, user management, role-based access control
- **Impact**: Security model incomplete

#### Real-time Updates
- **Status**: Frontend expects Socket.io, backend has basic WebSocket stubs
- **Missing**: Functional Socket.io server, SSE implementation
- **Impact**: Dashboard real-time features non-functional

#### Operational Modes (GUI/Headless)
- **Status**: Documented but not implemented
- **Missing**: Mode switching logic, GUI-specific configurations
- **Impact**: Deployment flexibility limited

> **Note**: See `Nexus_Gap_Analysis.md` for detailed analysis of implementation gaps and recommended actions.
## 🚀 Quick Start

### Prerequisites

- **Go 1.21+**: Required for backend compilation
- **Node.js 18+**: Required for frontend development and building
- **npm**: Package manager for frontend dependencies
- **Git**: Version control for cloning repository

### Quick Development Setup

**Get started in 5 minutes:**

```bash
# 1. Clone repository
git clone https://github.com/knirv/KNIRV_NETWORK.git
cd KNIRV_NETWORK/KNIRVNEXUS

# 2. Install frontend dependencies
npm install

# 3. Build frontend
npm run build

# 4. Build backend
cd backend
go mod tidy
go build -o ../bin/backend-server ./main.go
cd ..

# 5. Build unified binary
go build -o knirv-nexus main.go

# 6. Run application
./knirv-nexus

# 7. Access application at http://localhost:8090
```

**For development with hot reload:**

```bash
# Terminal 1: Frontend development server
npm run dev  # Runs on http://localhost:3000

# Terminal 2: Backend development server
cd backend
go run main.go --config config/development.yaml  # Runs on http://localhost:8080
```

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
