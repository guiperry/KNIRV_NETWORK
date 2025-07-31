# KNIRV Scripts Directory

This directory contains utility scripts for managing the KNIRV D-TEN ecosystem.

## Available Scripts

### `manage-knirv.sh` ⭐ NEW

**Purpose**: Unified management script for all KNIRV network components including the new KNIRVGATEWAY services.

**Location**: `./scripts/manage-knirv.sh`

**Features**:
- ✅ Complete KNIRV network lifecycle management
- ✅ Individual component control (start/stop/restart/status)
- ✅ Health monitoring for all services
- ✅ Integrated testing capabilities
- ✅ Development and production deployment support
- ✅ Cross-component dependency management

### `run-gateway.sh` ⭐ NEW

**Purpose**: Dedicated management for KNIRVGATEWAY services (Economics Service + API Gateway).

**Location**: `./scripts/run-gateway.sh`

**Features**:
- ✅ Economics Service management (Month 11 implementation)
- ✅ API Gateway routing and health checks
- ✅ Dependency validation and startup coordination
- ✅ Comprehensive testing and verification
- ✅ Service-specific configuration management
- ✅ Integration with existing KNIRV components

### `test-gateway-integration.sh` ⭐ NEW

**Purpose**: Comprehensive integration testing for KNIRVGATEWAY services with the existing test suite.

**Location**: `./scripts/test-gateway-integration.sh`

**Features**:
- ✅ Economics API endpoint testing
- ✅ Gateway routing validation
- ✅ Cross-component integration verification
- ✅ Automated service startup and cleanup
- ✅ HTML report generation
- ✅ Integration with existing test framework

### `run-integration-tests.sh`

**Purpose**: Complete integration test lifecycle management script that handles setup, execution, and teardown of the KNIRV integration test suite.

**Location**: `./scripts/run-integration-tests.sh`

**Features**:
- ✅ Automated test environment setup
- ✅ Integration test execution with configurable options
- ✅ Automatic teardown and cleanup
- ✅ Comprehensive error handling and reporting
- ✅ Colored output for better readability
- ✅ Flexible configuration options
- ✅ Enhanced with gateway testing capabilities

## Quick Start

### KNIRV Network Management

#### Start All KNIRV Components
```bash
# Start complete KNIRV network including gateway services
./scripts/manage-knirv.sh start all
```

#### Start Gateway Services Only
```bash
# Start KNIRVGATEWAY (Economics + API Gateway)
./scripts/run-gateway.sh start

# Or use the unified manager
./scripts/manage-knirv.sh start gateway
```

#### Check Network Health
```bash
# Check health of all KNIRV services
./scripts/manage-knirv.sh health

# Check gateway services specifically
./scripts/run-gateway.sh status
```

### Gateway Testing

#### Run Gateway Integration Tests
```bash
# Complete gateway integration testing
./scripts/test-gateway-integration.sh

# Economics service tests only
./scripts/test-gateway-integration.sh --economics-only

# Generate HTML report
./scripts/test-gateway-integration.sh --report
```

#### Verify Month 11 Implementation
```bash
# Verify complete Month 11 economics implementation
./scripts/run-gateway.sh economics verify
```

### Integration Testing

#### Run All Tests (Default)
```bash
# Run complete integration test suite with setup and teardown
./scripts/run-integration-tests.sh
```

#### Run Gateway-Specific Integration Tests
```bash
# Run economics integration tests
cd integration-tests
./config/run-tests.sh economics

# Run gateway integration tests
./config/run-tests.sh gateway
```

### Common Usage Examples
```bash
# Run with verbose output
./scripts/run-integration-tests.sh --verbose

# Run specific test pattern
./scripts/run-integration-tests.sh TestPerformance

# Run in parallel with extended timeout
./scripts/run-integration-tests.sh --parallel --timeout 900s

# Run without automatic teardown (for debugging)
./scripts/run-integration-tests.sh --no-teardown

# Force cleanup existing processes
./scripts/run-integration-tests.sh --force-cleanup
```

## Script Options

### Basic Options
- `--verbose`: Enable detailed output during execution
- `--parallel`: Run tests in parallel for faster execution
- `--timeout DURATION`: Set test timeout (default: 600s)

### Environment Control
- `--skip-setup`: Skip test environment setup (use existing environment)
- `--no-teardown`: Skip automatic teardown after tests
- `--force-cleanup`: Force kill existing processes during setup

### Output Control
- `--no-report`: Skip generating test reports
- `--no-preserve-logs`: Remove logs during teardown

### Test Selection
- `TEST_PATTERN`: Regex pattern to match specific tests (e.g., `TestLLM`, `TestPerformance`)

## Environment Variables

### Gateway Services Configuration
```bash
# Economics Service
export ECONOMICS_PORT=8090
export NRN_CONTRACT=your_nrn_contract_address
export XION_RPC=https://rpc.xion-testnet-1.burnt.com:443

# API Gateway
export GATEWAY_PORT=8000

# KNIRV Component URLs
export KNIRVCHAIN_URL=http://localhost:8080
export KNIRVNEXUS_URL=http://localhost:8081
export KNIRVROOT_URL=http://localhost:8082
export KNIRVGRAPH_URL=http://localhost:8083

# Integration Testing
export ECONOMICS_SERVICE_URL=http://localhost:8090
export GATEWAY_SERVICE_URL=http://localhost:8000
```

## Available Test Suites

### 1. Economics Integration Tests ⭐ NEW
**Pattern**: `TestEconomics`
**Description**: Month 11 economics service integration validation
```bash
./scripts/test-gateway-integration.sh --economics-only
# Or via integration test runner
cd integration-tests && ./config/run-tests.sh economics
```

### 2. Gateway Integration Tests ⭐ NEW
**Pattern**: `TestGateway`
**Description**: API Gateway routing and service integration validation
```bash
./scripts/test-gateway-integration.sh --gateway-only
# Or via integration test runner
cd integration-tests && ./config/run-tests.sh gateway
```

### 3. Basic Integration Tests
**Pattern**: `TestIntegrationSuite`
**Description**: Core functionality validation for all KNIRV components
```bash
./scripts/run-integration-tests.sh TestIntegrationSuite
```

### 2. Cross-Component Validation
**Pattern**: `TestCrossComponentValidation`
**Description**: Inter-component communication and data flow validation
```bash
./scripts/run-integration-tests.sh TestCrossComponentValidation
```

### 3. Performance and Load Tests
**Pattern**: `TestPerformanceAndLoad`
**Description**: System performance under load with metrics collection
```bash
./scripts/run-integration-tests.sh TestPerformanceAndLoad
```

### 4. End-to-End Workflows
**Pattern**: `TestE2EWorkflows`
**Description**: Complete user workflow validation
```bash
./scripts/run-integration-tests.sh TestE2EWorkflows
```

## Script Workflow

### 1. Prerequisites Check
- Verifies integration test directory exists
- Checks required scripts are present and executable
- Validates Go installation

### 2. Environment Setup
- Calls `integration-tests/config/setup.sh`
- Starts all KNIRV services
- Waits for service health confirmation

### 3. Test Execution
- Calls `integration-tests/config/run-tests.sh`
- Runs tests with specified parameters
- Collects test results and metrics

### 4. Teardown and Cleanup
- Calls `integration-tests/config/teardown.sh`
- Stops all services gracefully
- Cleans up test data (optional)
- Preserves logs and reports

### 5. Summary Report
- Displays test configuration
- Lists generated reports
- Shows available logs
- Provides final status

## Error Handling

### Automatic Recovery
- Emergency teardown on script failure
- Graceful service shutdown
- Process cleanup on exit

### Debug Mode
```bash
# Run with verbose output for debugging
./scripts/run-integration-tests.sh --verbose --no-teardown

# Check logs after failure
tail -f integration-tests/logs/*.log
```

### Common Issues

1. **Port Conflicts**
   ```bash
   ./scripts/run-integration-tests.sh --force-cleanup
   ```

2. **Service Startup Issues**
   ```bash
   # Check individual service logs
   ./scripts/run-integration-tests.sh --verbose
   ```

3. **Test Timeouts**
   ```bash
   ./scripts/run-integration-tests.sh --timeout 900s
   ```

## Integration with CI/CD

### GitHub Actions Example
```yaml
name: Integration Tests
on: [push, pull_request]
jobs:
  integration-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-go@v2
        with:
          go-version: 1.21
      - name: Run Integration Tests
        run: ./scripts/run-integration-tests.sh --verbose
      - name: Upload Test Reports
        uses: actions/upload-artifact@v2
        with:
          name: integration-test-reports
          path: integration-tests/reports/
```

### Jenkins Pipeline Example
```groovy
pipeline {
    agent any
    stages {
        stage('Integration Tests') {
            steps {
                sh './scripts/run-integration-tests.sh --parallel --timeout 900s'
            }
            post {
                always {
                    archiveArtifacts artifacts: 'integration-tests/reports/**/*'
                    publishHTML([
                        allowMissing: false,
                        alwaysLinkToLastBuild: true,
                        keepAll: true,
                        reportDir: 'integration-tests/reports',
                        reportFiles: '*.html',
                        reportName: 'Integration Test Report'
                    ])
                }
            }
        }
    }
}
```

## Output and Reports

### Generated Files
- **Test Reports**: `integration-tests/reports/test-results-*.json`
- **HTML Reports**: `integration-tests/reports/test-report-*.html`
- **Service Logs**: `integration-tests/logs/*.log`
- **Cleanup Reports**: `integration-tests/reports/cleanup-report-*.txt`

### Log Locations
- **Script Logs**: Console output with colored formatting
- **Service Logs**: `integration-tests/logs/` directory
- **Test Output**: Captured in JSON and HTML reports

## Contributing

### Adding New Scripts
1. Create script in `scripts/` directory
2. Make executable with `chmod +x`
3. Follow naming convention: `action-description.sh`
4. Add documentation to this README
5. Include usage examples

### Script Standards
- Use bash shebang: `#!/bin/bash`
- Include error handling: `set -e`
- Provide colored output for readability
- Include comprehensive help text
- Support common options (verbose, help, etc.)

---

**Last Updated**: Month 6 Implementation (July 2025)
**Maintained By**: KNIRV Development Team
