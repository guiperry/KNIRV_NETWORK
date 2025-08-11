# KNIRV-TESTNET: Complete Implementation

A minimal-viable representation of the full KNIRV Decentralized Trusted Execution Network (D-TEN) designed for testing, development, and validation.

## 🚀 Quick Start

### Prerequisites
- Docker & Docker Compose
- Go 1.23+
- Rust 1.70+
- Node.js 18+
- Python 3.9+

### 1. Build All Components
```bash
cd KNIRVTESTNET
chmod +x scripts/*.sh
./scripts/build-all.sh
```

### 2. Start the Testnet
```bash
./scripts/start-testnet.sh
```

### 3. Verify Health
```bash
./scripts/health-check.sh
```

### 4. Run Tests
```bash
./scripts/run-tests.sh
```

### 5. Stop When Done
```bash
# Graceful shutdown
./scripts/stop-testnet.sh

# Emergency shutdown (if stop script fails)
./scripts/kill_knirv.sh
```

## 🏗️ Architecture

The KNIRV-TESTNET implements all 12 sovereign layers of the D-TEN:

- **KNIRV-ROOT**: 3-validator PoA blockchain for NRN management
- **KNIRVCHAIN**: Single-shard Rust blockchain with CodeT5 base
- **KNIRVGRAPH**: Small-scale graphchain with pre-populated data
- **KNIRV-NEXUS**: 2 DVE nodes with simulated TEE
- **KNIRV-ROUTER**: Single router for connectivity testing
- **KNIRV-GATEWAY**: Complete API gateway implementation
- **KNIRV-WALLET**: Full XION integration
- **KNIRV-SDK**: Multi-language SDK
- **KNIRV-SHELL**: Complete CLI
- **KNIRV-CORTEX**: Simplified agents with SEAL loop
- **KNIRVANA**: Game client for testing
- **IPFS**: Decentralized storage

## 🌐 Service Endpoints

| Service | Endpoint | Purpose |
|---------|----------|---------|
| KNIRV-ROOT | http://localhost:1317 | NRN blockchain & oracle |
| KNIRVCHAIN | http://localhost:8090 | Base LLM & skill registry |
| KNIRVGRAPH | http://localhost:8082 | Knowledge graphchain |
| KNIRV-NEXUS-DVE | http://localhost:8084 | DVE Manager |
| KNIRV-NEXUS-VAL | http://localhost:8085 | Validation Core |
| KNIRV-ROUTER | http://localhost:8086 | Network routing |
| KNIRV-GATEWAY | http://localhost:8087 | Unified API gateway |
| IPFS API | http://localhost:5001 | Decentralized storage |

## 🧪 Testing

### Integration Tests
```bash
./scripts/run-tests.sh
```

### Economic Loop Testing
```bash
./scripts/test-economics.sh
```

### Agent Simulation
```bash
./scripts/simulate-agents.sh
```

### Resource Monitoring
```bash
./scripts/monitor-resources.sh
```

## 💾 Data Management

### Backup Data
```bash
./scripts/backup-data.sh
```

### View Logs
```bash
tail -f logs/*.log
```

### Check Service Status
```bash
./scripts/health-check.sh
```

## 🐳 Docker Deployment

Alternative deployment using Docker Compose:

```bash
docker-compose up -d
docker-compose logs -f
docker-compose down
```

## 📁 Directory Structure

```
KNIRVTESTNET/
├── bin/                 # Compiled binaries
├── config/              # Configuration files
├── data/                # Blockchain & service data
├── logs/                # Service logs
├── scripts/             # Management scripts
├── backups/             # Data backups
├── docker-compose.yml   # Docker deployment
└── README.md           # This file
```

## 🔧 Development Workflow

### Testing New Features
1. Make changes to components
2. Rebuild specific component
3. Restart specific service
4. Test changes

### Adding New Skills
1. Create SkillNode via API
2. Validate via DVE
3. Register on KNIRVCHAIN

### Debugging Issues
1. Check service logs in `./logs/`
2. Verify service health
3. Monitor resource usage
4. Check configuration files

## 📊 Monitoring

- **Health Checks**: `./scripts/health-check.sh`
- **Resource Monitor**: `./scripts/monitor-resources.sh`
- **Service Logs**: `tail -f logs/*.log`
- **Process Status**: Check `./data/*.pid` files

## 🔒 Security

- All services run in testnet mode
- TEE simulation enabled
- Test keyring backend
- Debug logging enabled
- CORS enabled for development

## 🚦 Production Migration

To migrate to production:
1. Scale validators (3 → production level)
2. Enable real TEE integration
3. Implement production security
4. Add monitoring/alerting
5. Configure load balancing
6. Setup disaster recovery

## 📝 Notes

- This is a simplified testnet implementation
- Some features are simulated for testing
- Economic operations may be mocked
- Real TEE is replaced with simulation
- All tokens are valueless test tokens

## 🆘 Troubleshooting

### Common Issues

**Port conflicts**: Check if ports are in use
```bash
netstat -tulpn | grep :8080
```

**Service startup failures**: Check logs
```bash
tail -f logs/service-name.log
```

**Database corruption**: Reset data
```bash
rm -rf data/service-name/
```

### Getting Help

1. Check service logs in `./logs/`
2. Run health check: `./scripts/health-check.sh`
3. Verify configuration files
4. Check process status: `ps aux | grep knirv`

## 🛠 New Management Scripts (Updated)

The testnet now includes comprehensive management and monitoring tools:

### Unified Control Scripts
- `./start-testnet.sh` - Start all services with dependency checking and health monitoring
- `./stop-testnet.sh` - Gracefully stop all services in reverse order
- `./health-check.sh` - Monitor service health (supports --watch mode for continuous monitoring)
- `./test-integration.sh` - Run comprehensive integration tests across all services
- `./validate-config.sh` - Validate testnet configuration and dependencies

### Enhanced Features
- **Dependency checking** - Verifies Go, Rust, Node.js, Python before starting
- **Port conflict detection** - Checks for port availability before service startup
- **Health monitoring** - Real-time service status with response time metrics
- **Integration testing** - Tests service communication and testnet-specific endpoints
- **Configuration validation** - Ensures all services are properly configured for testnet

### Usage Examples
```bash
# Start everything with full validation
./start-testnet.sh

# Monitor continuously with detailed info
./health-check.sh --watch --detailed

# Run comprehensive integration tests
./test-integration.sh --verbose

# Validate entire setup
./validate-config.sh --verbose

# Stop everything gracefully
./stop-testnet.sh
```

### Testnet-Specific Endpoints
- **Gateway Health**: http://localhost:8888/gateway/health
- **Service Discovery**: http://localhost:8888/gateway/services
- **Testnet Status**: http://localhost:8888/gateway/testnet/status
- **Auth Tokens**: http://localhost:8888/auth/testnet-tokens
- **Mock LLM**: http://localhost:8080/testnet/llm/validate
- **TEE Simulation**: http://localhost:8182/testnet/validate/skill

## 📄 License

This implementation is part of the KNIRV Network project.
