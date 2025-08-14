# KNIRV TESTNET Test Suite

## 🎯 Overview

Comprehensive test suite for the KNIRV TESTNET including automated CORTEX demonstrations, end-to-end ecosystem validation, and performance benchmarking.

## 📁 Directory Structure

```
tests/
├── e2e/                             # End-to-end test suites
│   ├── cortex-demo-suite/           # Automated CORTEX demonstrations
│   ├── user-journey-tests/          # Complete user workflows
│   ├── economic-loop-tests/         # NRN token flow validation
│   └── cross-service-integration/   # Service interaction validation
├── performance/                     # Performance testing
│   ├── load-testing/                # Testnet load testing
│   ├── stress-testing/              # Service stress testing
│   └── benchmarking/                # Performance baselines
├── security/                        # Security validation
│   ├── auth-testing/                # Authentication flow testing
│   ├── permission-testing/          # Access control validation
│   └── vulnerability-scanning/      # Security validation
├── automation/                      # Test automation framework
│   ├── demo-orchestrator/           # Automated demo coordination
│   ├── test-data-generator/         # Test data creation
│   └── reporting/                   # Test result aggregation
├── config/                          # Test configurations
├── utils/                           # Shared utilities
└── scripts/                         # Test execution scripts
```

## 🚀 Quick Start

### Run All Tests
```bash
# Complete test suite execution
./scripts/run-all-tests.sh

# Run specific test category
./scripts/run-tests.sh --category e2e
./scripts/run-tests.sh --category performance
./scripts/run-tests.sh --category security
```

### Run CORTEX Demos
```bash
# Run all CORTEX demos
./scripts/run-cortex-demos.sh

# Run specific demo
./scripts/run-demo.sh --scenario skill-development
./scripts/run-demo.sh --scenario multi-agent-collaboration
./scripts/run-demo.sh --scenario learning-adaptation

# Continuous demo execution
./scripts/run-demo.sh --continuous --interval 30m
```

### Performance Testing
```bash
# Load testing
./scripts/run-load-tests.sh

# Stress testing
./scripts/run-stress-tests.sh

# Benchmarking
./scripts/run-benchmarks.sh
```

## 📊 Test Categories

### 1. End-to-End Tests
- **CORTEX Demo Suite**: Automated demonstrations of CORTEX capabilities
- **User Journey Tests**: Complete user experience validation
- **Economic Loop Tests**: NRN token flow and economic incentive validation
- **Cross-Service Integration**: Service interaction and communication testing

### 2. Performance Tests
- **Load Testing**: Validate system performance under expected load
- **Stress Testing**: Test system limits and failure modes
- **Benchmarking**: Establish performance baselines and regression detection

### 3. Security Tests
- **Authentication Testing**: Validate authentication flows and security
- **Permission Testing**: Test access control and authorization
- **Vulnerability Scanning**: Security validation and penetration testing

## 🔧 Configuration

### Test Environment Configuration
```yaml
# config/testnet-config.yaml
testnet:
  environment: "local"
  services:
    - knirv-root
    - knirvchain
    - knirvgraph
    - knirv-nexus
    - knirv-router
    - knirv-gateway
  
  cortex:
    agents: 3
    scenarios: ["skill-development", "collaboration", "learning"]
  
  performance:
    load_users: 100
    stress_users: 500
    duration: "10m"
```

### Demo Configuration
```yaml
# config/demo-config.yaml
demos:
  skill-development:
    duration: "10m"
    agents: 1
    services: ["knirvchain", "knirvgraph", "knirv-nexus"]
  
  multi-agent-collaboration:
    duration: "15m"
    agents: 3
    services: ["knirv-router", "knirvgraph", "knirv-root"]
  
  learning-adaptation:
    duration: "20m"
    agents: 1
    components: ["seal", "hrm", "lora"]
```

## 📈 Reporting

### Test Reports
- **HTML Reports**: Comprehensive test results with visualizations
- **JSON Reports**: Machine-readable test data for CI/CD integration
- **Performance Reports**: Detailed performance metrics and trends
- **Demo Reports**: CORTEX demonstration results and recordings

### Metrics Collection
- Service health and performance metrics
- CORTEX agent performance and learning metrics
- Economic flow and transaction metrics
- Security validation results

## 🔄 Integration

### CI/CD Integration
- Automated test execution on code changes
- Performance regression detection
- Security validation in deployment pipeline
- Demo execution for stakeholder presentations

### Monitoring Integration
- Real-time test execution monitoring
- Alert integration for test failures
- Performance trend analysis
- Demo success rate tracking

## 📋 Test Execution Workflow

1. **Pre-Test Setup**: Environment validation and service startup
2. **Test Execution**: Parallel execution of test suites
3. **Demo Orchestration**: Automated CORTEX demonstration execution
4. **Results Collection**: Metrics aggregation and report generation
5. **Cleanup**: Environment teardown and resource cleanup

## 🎯 Success Criteria

- **Test Coverage**: >95% of testnet functionality
- **Demo Success Rate**: >90% automated demo completion
- **Performance**: <200ms API response times
- **Reliability**: >99% service uptime during tests
- **Security**: 100% security validation pass rate

## 🛠️ Development

### Adding New Tests
1. Create test files in appropriate category directory
2. Update configuration files as needed
3. Add test execution to relevant scripts
4. Update documentation and reporting

### Adding New Demos
1. Define demo scenario in `config/demo-scenarios.yaml`
2. Implement demo logic in `e2e/cortex-demo-suite/`
3. Add demo orchestration in `automation/demo-orchestrator/`
4. Update demo execution scripts

For detailed implementation guides, see individual component README files.
