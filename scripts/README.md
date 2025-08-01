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

### `kill_knirv.sh` ⭐ ENHANCED

**Purpose**: Comprehensive network-wide termination of all KNIRV-related processes with advanced management features.

**Location**: `./scripts/kill_knirv.sh`

**Features**:
- ✅ **Complete Network Coverage**: Terminates all KNIRV services (KNIRVCHAIN, KNIRVNEXUS, KNIRVGRAPH, KNIRVROOT, KNIRVROUTER, KNIRVGATEWAY)
- ✅ **Multi-Service Detection**: Economics Service, API Gateway, frontend processes (Node.js/Vite)
- ✅ **Advanced Process Discovery**: By name, port, working directory, and child processes
- ✅ **Graceful Shutdown**: SIGTERM first, then SIGKILL if needed with configurable timeouts
- ✅ **Multiple Operation Modes**: Normal, force, dry-run, verbose, emergency
- ✅ **Comprehensive Cleanup**: Temp files, logs, databases, Node.js artifacts, Docker resources
- ✅ **Network Status Monitoring**: Real-time service status and port monitoring
- ✅ **Safety Features**: Root warnings, confirmation prompts, final verification

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

# Check current KNIRV network status
./scripts/kill_knirv.sh --status
```

#### Stop All KNIRV Services
```bash
# Graceful shutdown of all KNIRV network processes
./scripts/kill_knirv.sh

# Force kill all KNIRV processes immediately
./scripts/kill_knirv.sh --force

# Preview what would be terminated (dry run)
./scripts/kill_knirv.sh --dry-run

# Verbose output with detailed process information
./scripts/kill_knirv.sh --verbose

# Emergency kill mode for stubborn processes
./scripts/kill_knirv.sh --emergency
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

### KNIRV Network Process Management

#### Kill Script Usage Examples
```bash
# Normal graceful shutdown of all KNIRV services
./scripts/kill_knirv.sh

# Force kill immediately (skip graceful shutdown)
./scripts/kill_knirv.sh --force

# Preview what would be killed without actually killing
./scripts/kill_knirv.sh --dry-run

# Detailed verbose output showing all processes
./scripts/kill_knirv.sh --verbose

# Check current network status without killing
./scripts/kill_knirv.sh --status

# Emergency kill mode for stubborn processes (with confirmation)
./scripts/kill_knirv.sh --emergency

# Skip cleanup of temp files and logs
./scripts/kill_knirv.sh --no-cleanup

# Show help and usage information
./scripts/kill_knirv.sh --help
```

### Common Integration Test Usage Examples
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

### Kill Script Options (`kill_knirv.sh`)

#### Basic Options
- `-h, --help`: Show comprehensive help message and usage examples
- `-v, --verbose`: Show detailed process information and termination progress
- `-n, --dry-run`: Preview what would be killed without actually terminating processes
- `-f, --force`: Skip graceful shutdown, force kill immediately with SIGKILL

#### Advanced Options
- `--emergency`: Emergency kill mode for stubborn processes (requires confirmation)
- `--status`: Check and display current KNIRV network status
- `--no-cleanup`: Skip cleanup of temporary files, logs, and build artifacts

#### Process Detection Features
- **Service Pattern Matching**: Detects KNIRVCHAIN, KNIRVNEXUS, KNIRVGRAPH, KNIRVROOT, KNIRVROUTER, KNIRVGATEWAY
- **Port-based Detection**: Monitors ports 8000-8091 and legacy ports (5000-6001)
- **Directory-based Search**: Finds Go processes in KNIRV directories
- **Child Process Detection**: Automatically finds and terminates child processes
- **Frontend Process Detection**: Detects Node.js, Vite, npm development servers

#### Cleanup Features
- **Temporary Files**: Go build cache, KNIRV-specific temp files
- **Lock/PID Files**: Service lock files, PID files, gateway.pid, economics.pid
- **Database Files**: Service databases (knirvnexus.db, knirvchain.db, etc.)
- **Log Files**: Large log files (>100MB), service logs
- **Node.js Artifacts**: node_modules/.cache, dist/, .vite/ directories
- **Docker Resources**: KNIRV-related containers and volumes

### Integration Test Options (`run-integration-tests.sh`)

#### Basic Options
- `--verbose`: Enable detailed output during execution
- `--parallel`: Run tests in parallel for faster execution
- `--timeout DURATION`: Set test timeout (default: 600s)

#### Environment Control
- `--skip-setup`: Skip test environment setup (use existing environment)
- `--no-teardown`: Skip automatic teardown after tests
- `--force-cleanup`: Force kill existing processes during setup

#### Output Control
- `--no-report`: Skip generating test reports
- `--no-preserve-logs`: Remove logs during teardown

#### Test Selection
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

## Script Workflows

### Kill Script Workflow (`kill_knirv.sh`)

#### 1. Process Discovery Phase
- **Pattern Matching**: Searches for processes by KNIRV service names
- **Port Scanning**: Checks all KNIRV service ports (8000-8091, legacy ports)
- **Directory Search**: Finds Go processes in KNIRV directories
- **Child Process Detection**: Identifies child processes of found parents
- **Frontend Detection**: Locates Node.js/Vite development servers

#### 2. Safety Checks
- **Root Warning**: Warns if running as root user
- **Process Validation**: Verifies found processes are actually KNIRV-related
- **Confirmation Prompts**: Requires confirmation for emergency operations
- **Dry Run Support**: Shows what would be killed without actually doing it

#### 3. Termination Phase
- **Graceful Shutdown**: Sends SIGTERM signal first (configurable timeout: 15s)
- **Progress Monitoring**: Tracks which processes shut down gracefully
- **Force Kill**: Sends SIGKILL to remaining processes if needed
- **Final Verification**: Confirms all processes are terminated

#### 4. Cleanup Phase
- **Temporary Files**: Removes Go build cache and KNIRV temp files
- **Lock/PID Files**: Cleans service lock and PID files
- **Database Files**: Removes service-specific database files
- **Log Cleanup**: Removes large log files (>100MB)
- **Node.js Artifacts**: Cleans frontend build artifacts and cache
- **Docker Resources**: Removes KNIRV-related containers and volumes

#### 5. Status Reporting
- **Process Summary**: Shows what was terminated
- **Cleanup Summary**: Reports what was cleaned up
- **Final Status**: Confirms successful termination or reports remaining issues
- **Exit Codes**: Returns 0 for success, 1 for partial failure

### Integration Test Workflow (`run-integration-tests.sh`)

#### 1. Prerequisites Check
- Verifies integration test directory exists
- Checks required scripts are present and executable
- Validates Go installation

#### 2. Environment Setup
- Calls `integration-tests/config/setup.sh`
- Starts all KNIRV services
- Waits for service health confirmation

#### 3. Test Execution
- Calls `integration-tests/config/run-tests.sh`
- Runs tests with specified parameters
- Collects test results and metrics

#### 4. Teardown and Cleanup
- Calls `integration-tests/config/teardown.sh`
- Stops all services gracefully
- Cleans up test data (optional)
- Preserves logs and reports

#### 5. Summary Report
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

#### KNIRV Process Management Issues

1. **Processes Won't Terminate**
   ```bash
   # Try force kill mode
   ./scripts/kill_knirv.sh --force

   # Use emergency mode for stubborn processes
   ./scripts/kill_knirv.sh --emergency

   # Check what's still running
   ./scripts/kill_knirv.sh --status
   ```

2. **Permission Issues**
   ```bash
   # If running as root (not recommended)
   sudo ./scripts/kill_knirv.sh --verbose

   # Check process ownership
   ps aux | grep -E "(knirv|economics|gateway)"
   ```

3. **Incomplete Cleanup**
   ```bash
   # Run cleanup manually
   ./scripts/kill_knirv.sh --no-cleanup  # Kill processes only
   # Then clean manually or run again without --no-cleanup
   ./scripts/kill_knirv.sh --dry-run     # Check what would be cleaned
   ```

4. **Service Detection Issues**
   ```bash
   # Use verbose mode to see detection process
   ./scripts/kill_knirv.sh --verbose --dry-run

   # Check specific ports manually
   lsof -i :8000 -i :8081 -i :8090
   ```

#### Integration Test Issues

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

## Quick Reference

### Kill Script Quick Commands
```bash
# Most common usage patterns
./scripts/kill_knirv.sh                    # Normal graceful shutdown
./scripts/kill_knirv.sh --status           # Check what's running
./scripts/kill_knirv.sh --dry-run          # Preview without killing
./scripts/kill_knirv.sh --force            # Immediate force kill
./scripts/kill_knirv.sh --help             # Show all options
```

### Detected Services and Ports
| Service | Port | Description |
|---------|------|-------------|
| KNIRVGATEWAY | 8000 | API Gateway |
| KNIRVCHAIN | 8080 | Blockchain Frontend |
| KNIRVNEXUS | 8081 | Inference Engine API |
| KNIRVROOT | 8082 | Root Service |
| KNIRVGRAPH | 8083 | Graph Service |
| Economics | 8090 | Economics Service |
| KNIRVROUTER | 8091 | Router Service |
| Legacy Ports | 5000-6001 | Legacy KNIRVROOT |
| Dev Servers | 3000-4001 | Development |

### Exit Codes
- `0`: All processes terminated successfully
- `1`: Some processes could not be terminated
- `2`: Script error or invalid arguments

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

## Recent Updates

### Month 12 Enhancements (August 2025)
- ✅ **Enhanced `kill_knirv.sh`**: Complete network-wide process termination with advanced features
- ✅ **Comprehensive Process Detection**: Multi-method process discovery across all KNIRV services
- ✅ **Advanced Operation Modes**: Dry-run, verbose, force, emergency modes
- ✅ **Safety Features**: Root warnings, confirmation prompts, graceful shutdown
- ✅ **Complete Cleanup**: Temp files, databases, logs, Node.js artifacts, Docker resources
- ✅ **Network Status Monitoring**: Real-time service status and port monitoring

### Month 11 Enhancements (July 2025)
- ✅ **KNIRVGATEWAY Integration**: Economics Service and API Gateway management
- ✅ **Gateway Testing Framework**: Comprehensive integration testing
- ✅ **Enhanced Service Management**: Unified management across all components

---

**Last Updated**: Month 12 Implementation (August 2025)
**Maintained By**: KNIRV Development Team
