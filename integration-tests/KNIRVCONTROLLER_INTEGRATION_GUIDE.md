# KNIRVCONTROLLER Real Network Integration Tests

## Overview

This guide documents the comprehensive integration tests for KNIRVCONTROLLER that connect to real KNIRV network services for live demonstrations and real-world testing scenarios.

## Architecture

KNIRVCONTROLLER serves as the unified frontend and backend interface for the KNIRV ecosystem, integrating with all network services:

```
┌─────────────────────────────────────────────────────────────┐
│                    KNIRVCONTROLLER                          │
│                   (localhost:3000)                         │
│  ┌─────────────────┐  ┌─────────────────┐                 │
│  │   Frontend      │  │    Backend      │                 │
│  │   (React)       │  │   (Express)     │                 │
│  └─────────────────┘  └─────────────────┘                 │
│  ┌─────────────────┐  ┌─────────────────┐                 │
│  │  WASM Compiler  │  │  LoRA Engine    │                 │
│  └─────────────────┘  └─────────────────┘                 │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    KNIRV Network                            │
│                                                             │
│  KNIRVCHAIN    KNIRVGRAPH    KNIRVROUTER    KNIRVORACLE    │
│  (8080)        (8081)        (8085)        (8086)         │
│                                                             │
│                    KNIRVNEXUS                               │
│                    (8084)                                   │
└─────────────────────────────────────────────────────────────┘
```

## Test Suites

### 1. Real Network Integration Tests (`knirvcontroller_real_network_test.go`)

**Purpose**: Validate KNIRVCONTROLLER integration with actual network services

**Test Coverage**:
- ✅ Health check and component status
- ✅ Skill invocation via ErrorContext → KNIRVGRAPH → KNIRVROUTER
- ✅ LoRA adapter registration and management
- ✅ WASM compilation capabilities
- ✅ Error context processing for skill resolution
- ✅ Network service integration validation

**Key Features**:
- **No Mocks**: Connects to real running services
- **Real Network Calls**: Actual HTTP requests to network endpoints
- **Graceful Failure Handling**: Accepts both success and expected failure modes
- **Comprehensive Logging**: Detailed output for debugging and demos

### 2. Demo Workflow Tests (`knirvcontroller_demo_workflows_test.go`)

**Purpose**: Demonstrate real-world usage scenarios for live presentations

**Demo Workflows**:

#### Demo 1: Agent Development Workflow
- Agent creation with capabilities and templates
- Template compilation and export
- WASM generation for deployment

#### Demo 2: Skill Invocation Workflow
- ErrorContext creation for skill resolution
- KNIRVGRAPH integration for skill discovery
- KNIRVROUTER integration for skill execution

#### Demo 3: Error Fixing Workflow
- Code error analysis and detection
- LoRA adapter compilation for error fixing
- Automated code repair demonstration

#### Demo 4: LoRA Adapter Workflow
- Specialized adapter creation
- Adapter registration and management
- Real-time adapter invocation

#### Demo 5: Network Integration Workflow
- Cross-service communication testing
- Network status monitoring
- Service health validation

#### Demo 6: Real-Time Monitoring Workflow
- System metrics collection
- Performance monitoring
- Component health tracking

## Configuration

### Service Endpoints

```yaml
# integration-tests/config/test-config.yaml
endpoints:
  knirvcontroller:
    url: "http://localhost:3000"
    health_endpoint: "/health"
    api_base: "/api"
    timeout: "30s"
    architecture: "unified"
    services: ["frontend", "backend", "wasm-compiler", "lora-engine", "protobuf-handler"]
```

### Test Data

```yaml
knirvcontroller:
  test_agent_id: "integration-test-agent"
  test_skill_id: "integration-test-skill"
  lora_adapter:
    name: "test-integration-adapter"
    rank: 16
    alpha: 0.5
    base_model: "hrm-v1"
  wasm_compilation:
    timeout: "60s"
    optimization_level: "O2"
  error_context:
    test_error_type: "skill_invocation_request"
    severity: "medium"
```

## Running the Tests

### Prerequisites

1. **All KNIRV Services Running**:
   ```bash
   # Start all services
   ./integration-tests/config/setup.sh
   ```

2. **KNIRVCONTROLLER Built and Ready**:
   ```bash
   cd KNIRVCONTROLLER
   npm install
   npm run build
   npm run start
   ```

### Execute Tests

```bash
# Run all KNIRVCONTROLLER integration tests
cd integration-tests
go test -v -run TestKNIRVControllerRealNetworkIntegration

# Run demo workflow tests
go test -v -run TestKNIRVControllerDemoWorkflows

# Run specific demo
go test -v -run TestKNIRVControllerDemoWorkflows/Demo1_AgentDevelopmentWorkflow

# Run with timeout for real network
go test -v -timeout 300s -run TestKNIRVController
```

### Using the Test Runner

```bash
# Run KNIRVCONTROLLER tests via test runner
./config/run-tests.sh knirvcontroller

# Run with verbose output for demos
./config/run-tests.sh knirvcontroller --verbose
```

## API Endpoints Tested

### Core Endpoints
- `GET /health` - Health check and component status
- `GET /api/status` - Operational status and capabilities
- `GET /api/templates/info` - Template information

### Agent & Skill Management
- `POST /api/compile-agent` - Agent compilation from templates
- `POST /api/invoke-skill` - Skill invocation via network
- `POST /api/process-error-context` - ErrorContext processing

### LoRA Adapter Operations
- `POST /lora/compile` - LoRA adapter compilation
- `POST /lora/invoke` - LoRA adapter invocation

### WASM Operations
- `POST /wasm/compile` - WASM compilation from TypeScript

### Network Integration
- `GET /api/network-status` - Network service status
- `POST /api/graph-query` - KNIRVGRAPH integration
- `GET /api/metrics` - System metrics
- `GET /api/performance` - Performance monitoring

## Demo Scenarios

### Live Demonstration Flow

1. **Setup Phase** (2 minutes):
   ```bash
   ./integration-tests/config/setup.sh
   # Wait for all services to be ready
   ```

2. **Health Check Demo** (1 minute):
   ```bash
   go test -v -run TestKNIRVControllerRealNetworkIntegration/TestKNIRVControllerHealthCheck
   ```

3. **Agent Development Demo** (3 minutes):
   ```bash
   go test -v -run TestKNIRVControllerDemoWorkflows/Demo1_AgentDevelopmentWorkflow
   ```

4. **Skill Invocation Demo** (3 minutes):
   ```bash
   go test -v -run TestKNIRVControllerDemoWorkflows/Demo2_SkillInvocationWorkflow
   ```

5. **Error Fixing Demo** (2 minutes):
   ```bash
   go test -v -run TestKNIRVControllerDemoWorkflows/Demo3_ErrorFixingWorkflow
   ```

6. **Network Integration Demo** (2 minutes):
   ```bash
   go test -v -run TestKNIRVControllerDemoWorkflows/Demo5_NetworkIntegrationWorkflow
   ```

### Expected Outputs

**Successful Integration**:
```
✅ KNIRVCONTROLLER health check successful
✅ Agent created successfully
✅ Skill invoked successfully: RequestID=abc123, ExecutionTime=150ms
✅ LoRA adapter compiled: AdapterID=lora-456
✅ Network status: operational
```

**Demo Mode (Expected in Test Environment)**:
```
ℹ️  Agent creation returned status 404 (expected in demo environment)
ℹ️  Skill invocation returned status 503 (service not fully configured)
ℹ️  LoRA compilation returned status 500 (WASM compiler not ready)
```

## Troubleshooting

### Common Issues

1. **Service Not Ready**:
   ```bash
   # Check if KNIRVCONTROLLER is running
   curl http://localhost:3000/health
   
   # Restart if needed
   cd KNIRVCONTROLLER && npm run start
   ```

2. **Port Conflicts**:
   ```bash
   # Clean up ports
   ./integration-tests/config/setup.sh --clean-start
   ```

3. **Test Timeouts**:
   ```bash
   # Increase timeout for real network
   go test -v -timeout 600s -run TestKNIRVController
   ```

### Debug Mode

```bash
# Enable verbose logging
export KNIRV_TEST_VERBOSE=true
go test -v -run TestKNIRVController

# Check service logs
tail -f integration-tests/logs/knirvcontroller.log
```

## Integration with CI/CD

### GitHub Actions Example

```yaml
name: KNIRVCONTROLLER Integration Tests
on: [push, pull_request]
jobs:
  integration-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-go@v2
        with:
          go-version: 1.21
      - uses: actions/setup-node@v2
        with:
          node-version: 18
      - name: Setup Services
        run: ./integration-tests/config/setup.sh
      - name: Run KNIRVCONTROLLER Tests
        run: |
          cd integration-tests
          go test -v -timeout 300s -run TestKNIRVController
      - name: Cleanup
        run: ./integration-tests/config/teardown.sh
```

## Future Enhancements

### Planned Improvements

1. **WebSocket Testing**: Real-time communication validation
2. **Performance Benchmarks**: Load testing with real network
3. **Security Testing**: Authentication and authorization validation
4. **Chaos Engineering**: Network failure simulation
5. **Multi-Environment**: Testing across different network configurations

### Demo Enhancements

1. **Interactive Demos**: Web-based demo interface
2. **Video Recording**: Automated demo recording
3. **Metrics Dashboard**: Real-time demo metrics
4. **Custom Scenarios**: User-defined demo workflows

## Support

For issues with KNIRVCONTROLLER integration tests:

1. Check service logs in `integration-tests/logs/`
2. Verify configuration in `integration-tests/config/test-config.yaml`
3. Review the troubleshooting section above
4. Contact the KNIRV development team

---

**Version**: 1.0  
**Last Updated**: August 2025  
**Maintained By**: KNIRV Development Team
