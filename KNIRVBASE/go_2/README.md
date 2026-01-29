# KNIRVCHAIN

**KNIRVCHAIN** is a specialized, private blockchain designed to function as a persistent "Long-Term Memory" (LTM) for Large Language Models. By leveraging the **Model Context Protocol (MCP)**, it allows any compatible LLM to read from and write to a secure, immutable ledger of user experiences, facts, and insights.

## 🚀 Key Features

- **Memory-Optimized Blockchain**: Purpose-built for AI memory storage with specialized indexing and retrieval mechanisms
- **GLB-Native Storage**: Leverages the GLB format for rich, multi-dimensional memory representation
- **Token-Gated Intelligence**: Uses NRN tokens to create a sustainable economy around AI memory operations
- **High-Performance Go Architecture**: Built with Go for superior concurrency, performance, and reliability
- **Semantic Search**: Advanced vector similarity search with HNSW indexing
- **Multi-Category Classification**: Automatic categorization of memories (ERROR, CONTEXT, IDEA, TASK, GENERAL)
- **KNIRVGRAPH Integration**: Seamless bridging to relational knowledge graphs

## 📋 Table of Contents

- [Architecture](#architecture)
- [Installation](#installation)
- [Configuration](#configuration)
- [Building](#building)
- [Running](#running)
- [API Reference](#api-reference)
- [Deployment](#deployment)
- [Development](#development)
- [Contributing](#contributing)
- [License](#license)

## 🏗️ Architecture

KNIRVCHAIN consists of three primary layers:

### Interface Layer (MCP Server)
- REST API endpoints for memory operations
- JWT-based authentication
- Rate limiting and token validation

### Core Logic Layer (Blockchain)
- Proof-of-Authority (PoA) consensus
- GLB-formatted memory storage
- Multi-index search system (semantic, temporal, categorical)

### Integration Layer (KNIRVGRAPH Bridge)
- Automatic bridging of categorized memories
- Event-driven synchronization
- RESTful communication with KNIRVGRAPH API

## 📦 Installation

### Prerequisites

- Go 1.21+
- Docker & Docker Compose (for containerized deployment)
- Redis (optional, for caching)
- PostgreSQL (optional, for metadata storage)

### Clone the Repository

```bash
git clone https://github.com/KNIRV_NETWORK/knirvchain.git
cd knirvchain
```

### Install Dependencies

```bash
make deps
```

## ⚙️ Configuration

KNIRVCHAIN uses YAML configuration files. Copy and modify the example config:

```bash
cp config.yaml config.local.yaml
```

### Key Configuration Options

```yaml
node:
  id: "node-001"
  role: "VALIDATOR"
  listen_addr: ":8080"

blockchain:
  data_dir: "/var/lib/knirvchain"
  consensus: "PoA"

wallet:
  private_key_path: "/etc/knirvchain/wallet.key"
  contract_address: "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb"

indexing:
  semantic:
    enabled: true
    dimension: 768
  temporal:
    enabled: true
  category:
    enabled: true

security:
  encryption:
    enabled: true
  jwt:
    secret: "your-secret-here"
```

## 🔨 Building

### Using Make

```bash
# Build the binary to bin/
make build

# Clean build artifacts
make clean

# Run tests
make test
```

### Manual Build

```bash
go build -o bin/knirvchain ./cmd/node
```

### Docker Build

```bash
make docker
```

## ▶️ Running

### Local Development

```bash
# Build and run
make run

# Or manually
./bin/knirvchain --config config.yaml
```

### Docker Compose

```bash
# Start all services
make docker-compose

# Stop services
make docker-compose-stop
```

### Production Deployment

```bash
# Using Docker Compose
docker-compose -f deployment/docker-compose.yml up -d

# Or Kubernetes
kubectl apply -f deployment/kubernetes/
```

## 📡 API Reference

### Memory Operations

#### Store Memory
```http
POST /tools/store_memory
Content-Type: application/json
Authorization: Bearer <token>

{
  "content": "User prefers dark mode interface",
  "memory_type": "CONTEXT",
  "tags": ["preference", "ui"]
}
```

#### Retrieve Memory
```http
POST /tools/retrieve_memory
Content-Type: application/json
Authorization: Bearer <token>

{
  "query": "user interface preferences",
  "limit": 10,
  "category": "CONTEXT"
}
```

#### Query Balance
```http
GET /tools/query_balance
Authorization: Bearer <token>
```

#### Estimate Cost
```http
POST /tools/estimate_cost
Content-Type: application/json
Authorization: Bearer <token>

{
  "operation": "store",
  "params": {
    "size": 1024,
    "category": "GENERAL"
  }
}
```

### Health & Status

#### Health Check
```http
GET /health
```

#### Node Status
```http
GET /status
```

#### Metrics
```http
GET /metrics
```

## 🚀 Deployment

### Docker

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o knirvchain ./cmd/node

FROM alpine:latest
COPY --from=builder /app/knirvchain .
CMD ["./knirvchain"]
```

### Docker Compose

```yaml
version: '3.8'
services:
  knirvchain-node:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - ./config.yaml:/root/config.yaml
```

### Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: knirvchain-node
spec:
  replicas: 3
  selector:
    matchLabels:
      app: knirvchain-node
  template:
    spec:
      containers:
      - name: knirvchain
        image: knirvchain/node:latest
        ports:
        - containerPort: 8080
```

## 💻 Development

### Project Structure

```
KNIRVCHAIN/
├── cmd/node/           # Application entry points
├── internal/           # Private application code
│   ├── blockchain/     # Core blockchain logic
│   ├── mcp/            # MCP server implementation
│   ├── classifier/     # Memory classification
│   ├── bridge/         # KNIRVGRAPH integration
│   ├── pricing/        # Token economics
│   ├── wallet/         # NRN wallet management
│   ├── security/       # Encryption & auth
│   ├── indexing/       # Search indexing
│   ├── cache/          # Caching layer
│   └── query/          # Query optimization
├── pkg/                # Public library code
│   ├── glb/            # GLB format handling
│   └── config/         # Configuration management
├── deployment/         # Deployment configurations
├── docs/               # Documentation
├── config.yaml         # Default configuration
├── go.mod              # Go module file
├── Makefile            # Build automation
└── README.md           # This file
```

### Testing

```bash
# Run all tests
make test

# Run specific package tests
go test ./internal/blockchain/

# Run benchmarks
go test -bench=. ./...

# Run integration tests
go test -tags=integration ./...
```

### Code Quality

```bash
# Format code
make fmt

# Lint code
make lint

# Generate mocks
make mocks
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Guidelines

- Follow Go best practices and conventions
- Write comprehensive tests for new features
- Update documentation for API changes
- Ensure all tests pass before submitting PRs
- Use conventional commit messages

## 📊 Performance

### Benchmarks

- **Block Commit**: <100ms (p99)
- **Semantic Search**: <50ms for 10k blocks
- **Memory Storage**: 10,000+ TPS
- **Concurrent Connections**: 100,000+

### System Requirements

**Minimum:**
- CPU: 4 cores
- RAM: 8 GB
- Storage: 100 GB SSD

**Recommended:**
- CPU: 16+ cores
- RAM: 32+ GB
- Storage: 500 GB NVMe SSD

## 🔒 Security

- AES-256-GCM encryption for data at rest
- TLS 1.3 for data in transit
- JWT-based authentication
- Role-based access control
- Regular security audits

## 📈 Monitoring

### Metrics

KNIRVCHAIN exposes Prometheus metrics at `/metrics`:

- `knirvchain_blocks_committed_total`
- `knirvchain_memory_store_ops_total`
- `knirvchain_query_latency_seconds`
- `knirvchain_cache_hits_total`

### Logging

Structured JSON logging with configurable levels:
- ERROR, WARN, INFO, DEBUG

### Tracing

OpenTelemetry integration with Jaeger support for distributed tracing.

## 🧪 Token Economics

### NRN Token

- **Total Supply**: 1,000,000,000 NRN
- **Utility**: Memory storage, retrieval, and computational resources
- **Governance**: Community voting on network parameters

### Pricing Examples

| Operation | Typical Cost |
|-----------|-------------|
| Store 1KB memory (General) | 15 NRN |
| Store 5KB memory (Idea) | 23 NRN |
| Retrieve 10 results | 25 NRN |
| Priority storage | 5x normal |

## 🌐 Ecosystem

- **KNIRVANA**: Gaming platform integration
- **KNIRVGRAPH**: Knowledge graph visualization
- **KNIRVCONTROLLER**: Web interface
- **KNIRVCLI**: Command-line tools

## 📚 Documentation

- [Solution Design Document](./docs/KNIRVCHAIN_SDD.md)
- [API Reference](./docs/api.md)
- [Deployment Guide](./docs/deployment.md)
- [Contributing Guide](./CONTRIBUTING.md)

## 🐛 Troubleshooting

### Common Issues

1. **Build fails**: Ensure Go 1.21+ is installed
2. **Port conflicts**: Check if ports 8080, 8081, 9090 are available
3. **Memory issues**: Increase RAM allocation for large datasets
4. **Search performance**: Ensure semantic indexing is enabled

### Logs

Check logs for detailed error information:
```bash
tail -f /var/log/knirvchain/node.log
```

## 📞 Support

- **Documentation**: https://docs.knirv.network
- **Issues**: https://github.com/KNIRV_NETWORK/knirvchain/issues
- **Discussions**: https://github.com/KNIRV_NETWORK/knirvchain/discussions
- **Discord**: https://discord.gg/knirvnetwork

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](../LICENSE) file for details.

## 🙏 Acknowledgments

- KNIRV Network Core Team
- Open source contributors
- The Go community
- Blockchain and AI research communities

---

**Version**: 1.0.0
**Last Updated**: December 2024
**Authors**: KNIRV Network Core Team