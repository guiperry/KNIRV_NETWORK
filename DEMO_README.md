# KNIRV Network Full Demo

This repository includes a comprehensive demonstration script that showcases the complete KNIRV Network infrastructure, including the testnet deployment and KNIRVCONTROLLER integration.

## 🚀 Quick Start

### Using Makefile (Recommended)

```bash
# Run the full demo with default settings
make demo

# Quick 5-minute demo (non-interactive)
make demo-quick

# Interactive demo with user prompts
make demo-interactive

# Extended demo with full testing (15 minutes)
make demo-extended

# Development demo (no cleanup)
make demo-development

# Show all demo options
make demo-help
```

### Direct Script Usage

```bash
# Make script executable (if not already)
chmod +x run-full-demo.sh

# Run with default settings
./run-full-demo.sh

# Run with custom options
./run-full-demo.sh --duration 600 --non-interactive --skip-tests

# Show help
./run-full-demo.sh --help
```

## 📋 Demo Phases

The demo consists of 9 comprehensive phases:

### Phase 1: Introduction
- Welcome message and configuration overview
- Demo objectives and timeline

### Phase 2: Infrastructure Setup
- Project structure validation using Makefile
- System health checks
- Configuration validation

### Phase 3: Testnet Deployment
- Complete KNIRVTESTNET infrastructure deployment
- Service initialization and health verification
- Network connectivity testing

### Phase 4: KNIRVCONTROLLER Integration
- KNIRVCONTROLLER startup in testnet mode
- Cognitive interface initialization
- Service endpoint verification

### Phase 5: Service Demonstration
- Individual service endpoint testing
- Integration verification
- Real-time communication testing

### Phase 6: Testing Suite
- Comprehensive test execution using Makefile
- Quick test suite for rapid validation
- Test report generation

### Phase 7: Monitoring & Analytics
- Real-time service status monitoring
- Log file analysis
- Performance metrics display

### Phase 8: Interactive Demonstration
- Browser-based interface exploration
- Live service interaction
- Extended runtime for manual testing

### Phase 9: Cleanup & Summary
- Service shutdown and cleanup
- Demo summary and results
- Resource cleanup

## 🎛️ Configuration Options

### Command Line Arguments

| Option | Description | Default |
|--------|-------------|---------|
| `--duration SECONDS` | Demo runtime duration | 300 (5 minutes) |
| `--non-interactive` | Run without user prompts | false |
| `--skip-tests` | Skip testing phase | false |
| `--no-cleanup` | Don't cleanup after demo | false |
| `--help` | Show help message | - |

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DEMO_DURATION` | Demo duration in seconds | 300 |
| `INTERACTIVE_MODE` | Enable interactive prompts | true |
| `SKIP_TESTS` | Skip testing phase | false |
| `CLEANUP_AFTER` | Cleanup after demo | true |

## 🌐 Service Endpoints

During the demo, the following services will be available:

| Service | Port | URL | Description |
|---------|------|-----|-------------|
| KNIRVCONTROLLER | 3000 | http://localhost:3000 | Cognitive interface |
| KNIRV-NEXUS | 8084 | http://localhost:8084 | Validation engine |
| KNIRV-GATEWAY | 8888 | http://localhost:8888 | API gateway |
| KNIRVCHAIN | 8090 | http://localhost:8090 | Blockchain layer |
| KNIRVGRAPH | 8082 | http://localhost:8082 | Graph storage |
| KNIRV-ORACLE | 1317 | http://localhost:1317 | Oracle services |
| KNIRV-ROUTER | 8086 | http://localhost:8086 | Network routing |

## 📊 Testing Integration

The demo integrates with the Makefile testing suite:

```bash
# Tests executed during demo
make test-quick          # Quick validation tests
make testnet-tests       # Testnet-specific tests
make test-reports        # Generate test reports
make health-check        # Service health verification
```

## 🔧 Prerequisites

### Required Tools
- Node.js (v18+)
- npm
- make
- curl
- Git

### Required Directories
- `KNIRVTESTNET/` - Testnet infrastructure
- `KNIRVCONTROLLER/` - Controller application
- `Makefile` - Build and deployment automation

### System Requirements
- Linux/macOS/WSL
- 8GB+ RAM recommended
- 10GB+ free disk space
- Network connectivity for package downloads

## 🎯 Demo Scenarios

### Scenario 1: Quick Validation
```bash
make demo-quick
```
- 5-minute runtime
- Non-interactive
- Skip comprehensive tests
- Focus on core functionality

### Scenario 2: Full Demonstration
```bash
make demo-interactive
```
- 10-minute runtime
- Interactive prompts
- Complete testing suite
- Manual exploration time

### Scenario 3: Development Testing
```bash
make demo-development
```
- No automatic cleanup
- Services remain running
- Extended testing capabilities
- Development-friendly

### Scenario 4: Extended Showcase
```bash
make demo-extended
```
- 15-minute runtime
- Full testing suite
- Comprehensive validation
- Complete feature demonstration

## 🛠️ Troubleshooting

### Common Issues

**Port Conflicts**
```bash
# Check for running services
lsof -i :3000,8084,8888,8090,8082,1317,8086

# Stop conflicting services
make testnet-tests  # This includes cleanup
```

**Permission Issues**
```bash
# Make script executable
chmod +x run-full-demo.sh

# Check file permissions
ls -la run-full-demo.sh
```

**Missing Dependencies**
```bash
# Install Node.js dependencies
cd KNIRVTESTNET && npm install
cd KNIRVCONTROLLER && npm install

# Check system prerequisites
make check-prereqs
```

### Manual Cleanup

If the demo fails or is interrupted:

```bash
# Stop testnet services
cd KNIRVTESTNET && npm run testnet:stop

# Kill KNIRVCONTROLLER
pkill -f "vite.*3000"

# Clean test artifacts
make test-clean

# Check for remaining processes
ps aux | grep -E "(knirv|vite|node)" | grep -v grep
```

## 📈 Performance Monitoring

During the demo, monitor system resources:

```bash
# CPU and memory usage
htop

# Network connections
netstat -tulpn | grep -E "(3000|8084|8888|8090|8082|1317|8086)"

# Service logs
tail -f KNIRVTESTNET/logs/*.log
```

## 🎉 Success Indicators

A successful demo will show:

- ✅ All 7 services running and responding
- ✅ KNIRVCONTROLLER accessible at http://localhost:3000
- ✅ Service health checks passing
- ✅ Test suite completion (if not skipped)
- ✅ Browser interfaces loading correctly
- ✅ Real-time service communication

## 📞 Support

For issues or questions:

1. Check the troubleshooting section above
2. Review service logs in `KNIRVTESTNET/logs/`
3. Verify prerequisites with `make check-prereqs`
4. Run `make demo-help` for additional options

## 🔄 Integration with CI/CD

The demo script can be integrated into CI/CD pipelines:

```yaml
# Example GitHub Actions integration
- name: Run KNIRV Demo
  run: |
    make demo-quick
    
# Example with custom settings
- name: Run Extended Demo
  run: |
    ./run-full-demo.sh --duration 180 --non-interactive --skip-tests
```

This demo provides a comprehensive showcase of the KNIRV Network's capabilities, from infrastructure deployment to real-time service interaction.
