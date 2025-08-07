# KNIRV-NEXUS: Decentralized Validation Environment

KNIRV-NEXUS is a production-ready implementation of the Decentralized Validation Environment (DVE) for the KNIRV Network. It provides a secure, scalable, and distributed platform for validating SkillNodes and Base LLMs using Trusted Execution Environments (TEE) and P2P networking.

## 🏗️ Architecture Overview

KNIRV-NEXUS implements a microservices architecture running on Kubernetes with the following core components:

### Core Services

- **DVE Manager**: Orchestrates DVE nodes, manages task allocation, and monitors system health
- **Validation Core**: Executes validation tasks with TEE support and cryptographic proofs
- **API Gateway**: Provides RESTful APIs and Server-Sent Events for real-time updates

### Infrastructure

- **Base OS**: Kali Linux (as specified in KALI_LINUX_FOUNDATION.md)
- **Container Runtime**: Podman (rootless containers)
- **Orchestration**: Kubernetes with production-ready configurations
- **Database**: BuntDB (embedded key-value store with custom indexes)
- **Networking**: libp2p (aligned with KNIRV-ROOT protocols)

## 🚀 Features

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
- KNIRV-ROOT aligned libp2p implementation
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
## � Quick Start

### Prerequisites

- Kubernetes cluster (v1.25+)
- kubectl configured
- Docker or Podman
- Go 1.21+ (for development)

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

1. **Install dependencies**:
   ```bash
   cd backend
   go mod tidy
   ```

2. **Run tests**:
   ```bash
   go test ./tests/... -v
   ```

3. **Build locally**:
   ```bash
   go build -o bin/dve-manager ./cmd/dve-manager/
   go build -o bin/validation-core ./cmd/validation-core/
   go build -o bin/api-gateway ./cmd/api-gateway/
   ```

## 📁 Project Structure

```
KNIRVNEXUS/
├── backend/                    # Go backend services
│   ├── cmd/                   # Service entry points
│   │   ├── dve-manager/       # DVE Manager service
│   │   ├── validation-core/   # Validation Core service
│   │   └── api-gateway/       # API Gateway service
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
└── README.md                # This file
```

## 🔧 API Documentation

### Authentication

All API endpoints (except `/health`) require authentication via JWT tokens.

```bash
# Login
curl -X POST http://api.knirv-nexus.local/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password"}'
```

### DVE Node Management

```bash
# Register a new DVE node
curl -X POST http://api.knirv-nexus.local/api/v1/dve-nodes \
  -H "Authorization: Bearer $TOKEN" \
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
curl -X GET http://api.knirv-nexus.local/api/v1/dve-nodes \
  -H "Authorization: Bearer $TOKEN"
```

### Validation Tasks

```bash
# Create validation task
curl -X POST http://api.knirv-nexus.local/api/v1/validation-tasks \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "skillnode",
    "priority": 5,
    "skill_code": "def hello(): return \"Hello, World!\"",
    "test_cases": [...],
    "required_tee_type": "sgx"
  }'

# Get task status
curl -X GET http://api.knirv-nexus.local/api/v1/validation-tasks/{id} \
  -H "Authorization: Bearer $TOKEN"
```

### Real-time Updates

Connect to Server-Sent Events for real-time updates:

```javascript
const eventSource = new EventSource('http://api.knirv-nexus.local/api/v1/sse?user_id=123');
eventSource.onmessage = function(event) {
  const data = JSON.parse(event.data);
  console.log('Update:', data);
};
```

## ⚙️ Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `KNIRV_CHAIN_ID` | Blockchain network ID | `knirv-nexus-mainnet` |
| `KNIRV_NODE_ROLE` | Node role (dve-manager, dve-validator) | `dve-manager` |
| `KNIRV_DATABASE_PATH` | Database file path | `/app/data/nexus.db` |
| `KNIRV_API_PORT` | API server port | `8080` |
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

### Common Issues

1. **Pod startup failures**: Check resource limits and node capacity
2. **P2P connectivity issues**: Verify firewall rules and port accessibility
3. **Database errors**: Check persistent volume availability
4. **Authentication failures**: Verify JWT secret configuration

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
