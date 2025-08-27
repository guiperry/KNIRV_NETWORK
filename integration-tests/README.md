# KNIRV D-TEN Month 6 Integration Testing Suite

## Overview

This directory contains the complete integration testing suite for the KNIRV D-TEN ecosystem, implementing all Month 6 requirements from the Comprehensive Implementation Plan. The suite provides comprehensive validation of all KNIRV components and their interactions.

## Quick Start

### Run All Tests
```bash
# Complete test execution with setup and teardown
./config/run-tests.sh

# Run specific test suite
./config/run-tests.sh performance

# Run KNIRVNEXUS tests (both backend and frontend)
./config/run-tests.sh knirvnexus

# Run KNIRVNEXUS backend tests only
./config/run-tests.sh knirvnexus-backend

# Run KNIRVNEXUS frontend tests only
./config/run-tests.sh knirvnexus-frontend

# Run JavaScript tests only
./config/run-tests.sh javascript

# Run KNIRV GraphChain Explorer tests only
./config/run-tests.sh graphchain-explorer

# Run KNIRVGATEWAY NEXUS Integration tests only
./config/run-tests.sh gateway-nexus

# Run KNIRVCONTROLLER real network integration tests
./config/run-tests.sh knirvcontroller

# Run KNIRVCONTROLLER demo workflows
./config/run-tests.sh knirvcontroller-demo

# Run with verbose output
./config/run-tests.sh --verbose
```

### Manual Setup
```bash
# Setup test environment
./config/setup.sh

# Run all tests
go test -v ./...

# Run specific test suites
go test -v -run TestBasicIntegration
go test -v -run TestCrossComponentValidation
go test -v -run TestPerformance
go test -v -run TestE2EWorkflow
go test -v -run TestPoAuDIntegrationSuite  # ⭐ NEW PoAu-D tests

# Run PoAu-D tests only
go test -v -run TestPoAuD

# Run Developer Portal tests
node portal-integration.test.js
node validate-portal.js

# Cleanup
./config/teardown.sh
```

## Test Suites

### 1. Basic Integration Tests (`xion-integration-test.go`)
- **Purpose**: Validate core functionality of each KNIRV component
- **Coverage**: API endpoints, health checks, basic operations
- **Components**: KNIRVCHAIN, KNIRVGRAPH, KNIRVNEXUS (Frontend + Backend), KNIRVORACLE, KNIRVROUTER

### 2. Cross-Component Validation (`cross-component-validation.go`)
- **Purpose**: Verify inter-component communication and data flow
- **Coverage**: Integration points, data propagation, consistency
- **Key Tests**: LLM registration propagation, agent blockchain recording, connectivity rewards

### 3. Performance Testing (`performance-test.go`)
- **Purpose**: Validate system performance under load
- **Coverage**: Throughput, latency, concurrent users, resource utilization
- **Metrics**: Requests/second, error rates, response times

### 4. End-to-End Workflows (`e2e-workflow-test.go`)
- **Purpose**: Test complete user workflows and scenarios
- **Coverage**: Developer workflows, agent lifecycle, bridge transfers, P2P networking
- **Validation**: Real-world usage patterns and user journeys

### 5. PoAu-D Consensus Testing (`poaud_integration_test.go`) ⭐ NEW
- **Purpose**: Validate PoAu-D consensus mechanism and delegation functionality
- **Coverage**: Consensus control, Network Authors management, transaction delegation
- **Key Tests**: Enable/disable PoAu-D, NAP management, PAP delegation, gateway integration
- **Validation**: Hybrid mining, delegation statistics, error handling

### 6. KNIRVNEXUS Backend Integration Testing (`knirvnexus_backend_integration_test.go`) ⭐ NEW
- **Purpose**: Validate KNIRVNEXUS backend services (API Gateway, DVE Manager, Validation Core)
- **Coverage**: Service health, DVE node management, validation task creation, system metrics
- **Key Tests**: API Gateway endpoints, DVE node registration, validation task lifecycle, P2P networking
- **Validation**: Service communication, data persistence, error handling, performance metrics

### 7. KNIRVNEXUS Frontend Integration Testing (`knirvnexus_frontend_integration_test.js`) ⭐ NEW
- **Purpose**: Validate KNIRVNEXUS Next.js frontend functionality and Socket.IO connectivity
- **Coverage**: Frontend health, page accessibility, static assets, API endpoints, Socket.IO
- **Key Tests**: Next.js build artifacts, package integrity, environment configuration, real-time updates
- **Validation**: Frontend-backend integration, responsive design, WebSocket connectivity

### 8. KNIRV GraphChain Explorer Integration Testing (`knirv-graphchain-explorer.test.js`) ⭐ NEW
- **Purpose**: Validate KNIRV GraphChain Explorer frontend functionality and integration
- **Coverage**: File structure, resource loading, UI components, API integration, SSE connectivity
- **Key Tests**: HTML pages, CSS styling, JavaScript components, navigation, mock data, accessibility
- **Validation**: Real-time updates, responsive design, branding consistency, documentation completeness

### 9. Developer Portal Integration Testing (`portal-integration.test.js`) ⭐ NEW
- **Purpose**: Validate KNIRV Developer Portal functionality and integration
- **Coverage**: Portal structure, navigation, branding, website integration
- **Key Tests**: File structure, HTML validation, responsive design, accessibility
- **Validation**: Portal readiness, user experience, deployment configuration

### 10. KNIRVGATEWAY NEXUS Integration Testing (`gateway_nexus_integration_test.sh`) ⭐ NEW
- **Purpose**: Validate KNIRVGATEWAY integration with KNIRVNEXUS services and role-based authentication
- **Coverage**: Gateway health, NEXUS API routing, authentication, CORS, SSE endpoints, NEXUS Portal
- **Key Tests**: Gateway endpoints, role-based access control, real-time features, cross-origin requests
- **Validation**: Service integration, authentication flows, portal accessibility, API gateway functionality

### 11. KNIRVTestnet Integration Testing (`knirvtestnet_integration_test.go`) ⭐ NEW
- **Purpose**: Validate KNIRVTestnet standalone integration and service discovery
- **Coverage**: Testnet service health, cross-service communication, authentication, gateway proxy
- **Key Tests**: Service endpoints, testnet-specific functionality, service discovery
- **Validation**: Testnet readiness, standalone operation, mirrored production behavior
- **Execution**: Can be run independently or as part of full test suite via `run-integration-tests.sh`

### 12. KNIRVCONTROLLER Real Network Integration Testing (`knirvcontroller_real_network_test.go`) ⭐ NEW
- **Purpose**: Validate KNIRVCONTROLLER integration with real KNIRV network services for demos
- **Coverage**: Unified server health, skill invocation, LoRA adapters, WASM compilation, network integration
- **Key Tests**: Health checks, ErrorContext processing, cross-service communication, real-time monitoring
- **Validation**: Real network connectivity, service integration, demo readiness
- **Execution**: Connects to actual running services (no mocks)

### 13. KNIRVCONTROLLER Demo Workflows (`knirvcontroller_demo_workflows_test.go`) ⭐ NEW
- **Purpose**: Demonstrate real-world KNIRVCONTROLLER usage scenarios for live presentations
- **Coverage**: Agent development, skill invocation, error fixing, LoRA workflows, network monitoring
- **Key Tests**: 6 comprehensive demo workflows showcasing all major features
- **Validation**: End-to-end functionality, real-world scenarios, presentation readiness
- **Execution**: Designed for live demonstrations and real-world testing

## Architecture

### Component Integration
```
┌─────────────┐    ┌─────────────┐    ┌─────────────────────────────┐
│ KNIRVCHAIN  │◄──►│ KNIRVGRAPH  │◄──►│      KNIRVNEXUS             │
│   (8080)    │    │   (8081)    │    │  ┌─────────────────────────┐ │
└─────────────┘    └─────────────┘    │  │ Frontend (Next.js)      │ │
       │                   │          │  │      (3000)             │ │
       ▼                   ▼          │  └─────────────────────────┘ │
┌─────────────┐    ┌─────────────┐    │  ┌─────────────────────────┐ │
│ KNIRVORACLE   │◄──►│ KNIRVROUTER │◄──►│  │ API Gateway    (8080)   │ │
│   (8086)    │    │   (8085)    │    │  │ DVE Manager    (8081)   │ │
└─────────────┘    └─────────────┘    │  │ Validation Core(8082)   │ │
       │                              │  └─────────────────────────┘ │
       ▼                              └─────────────────────────────┘
┌─────────────┐    ┌─────────────┐
│ KNIRVWALLET │    │ KNIRVCORTEX │
│   (8083)    │    │   (8084)        │
└─────────────┘    └─────────────────┘
```

### Test Data Flow
1. **Setup**: Initialize test environment and services
2. **Validation**: Execute test suites with comprehensive assertions
3. **Monitoring**: Collect performance metrics and logs
4. **Reporting**: Generate detailed test reports
5. **Cleanup**: Clean shutdown and environment reset

## Configuration

### Test Configuration (`config/test-config.yaml`)
- Service endpoints and timeouts
- Test data parameters
- Performance thresholds
- Cross-component settings
- E2E workflow configuration

### Environment Variables
```bash
KNIRV_TEST_MODE=true                    # Enable test mode
KNIRV_TEST_CONFIG=config/test-config.yaml  # Configuration file
KNIRV_TEST_DATA_DIR=./data             # Test data directory
KNIRV_TEST_LOGS_DIR=./logs             # Test logs directory
```

## Performance Benchmarks

### Target Metrics
- **Error Rate**: < 5%
- **Throughput**: > 1 request/second
- **Latency**: < 5 seconds average
- **Availability**: > 99% uptime

### Load Testing
- **Concurrent Users**: 10-50
- **Test Duration**: 30-60 seconds
- **Request Volume**: 50-100 per user
- **Ramp-up Time**: 5-10 seconds

## Test Coverage

### Functional Coverage
- ✅ All API endpoints tested
- ✅ Component health validation
- ✅ Data persistence verification
- ✅ Error handling and recovery
- ✅ Authentication and authorization

### Integration Coverage
- ✅ KNIRVCHAIN ↔ KNIRVGRAPH: LLM registration and skill invocation
- ✅ KNIRVNEXUS ↔ KNIRVORACLE: Agent creation and execution payments
- ✅ KNIRVROUTER ↔ KNIRVORACLE: Connectivity proofs and rewards
- ✅ Cross-chain bridge functionality

### Workflow Coverage
- ✅ Developer code fixing workflow
- ✅ Agent development lifecycle
- ✅ Cross-chain token transfers
- ✅ P2P network participation

### Developer Portal Coverage
- ✅ Portal file structure and assets
- ✅ HTML structure and navigation consistency
- ✅ Responsive design and accessibility
- ✅ KNIRV branding and terminology
- ✅ Main website integration
- ✅ Netlify deployment configuration

## Reports and Documentation

### Generated Reports
- **Test Results**: JSON format with detailed execution data
- **Performance Reports**: Metrics and benchmarking data
- **HTML Reports**: Human-readable summaries with visualizations
- **Cleanup Reports**: Environment status and verification

### Documentation
- **README.md**: This overview document
- **TEST_DOCUMENTATION.md**: Comprehensive testing documentation
- **reports/README.md**: Report format and analysis guide
- **validation-report-template.md**: Standardized report template

## Troubleshooting

### Common Issues

1. **Port Conflicts**
   ```bash
   ./config/setup.sh --clean-start
   ```

2. **Service Health Issues**
   ```bash
   # Check logs
   tail -f logs/*.log
   
   # Verify endpoints
   curl http://localhost:8080/health
   ```

3. **Test Timeouts**
   ```bash
   ./config/run-tests.sh --timeout 900s
   ```

4. **Database Cleanup Issues**
   ```bash
   # Manual cleanup of test databases if needed
   rm -rf ./database ./database_reflection ./data/testnet ./testdata/node1/db
   rm -rf integration-tests/data KNIRVCHAIN/sledchain.db KNIRVGRAPH/data
   rm -rf KNIRVNEXUS/db KNIRVORACLE/data KNIRVROUTER/data
   ```

### Debug Mode
```bash
# Verbose execution
./config/run-tests.sh --verbose

# Manual debugging
./config/setup.sh --verbose
```

## CI/CD Integration

### GitHub Actions Example
```yaml
name: Integration Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-go@v2
        with:
          go-version: 1.21
      - name: Run Tests
        run: ./integration-tests/config/run-tests.sh
```

## Month 6 Compliance

### Completed Tasks
- ✅ **Task 6.1**: Comprehensive Integration Test Suite
- ✅ **Task 6.2**: Cross-Component Validation Tests
- ✅ **Task 6.3**: Performance and Load Testing
- ✅ **Task 6.4**: End-to-End Workflow Tests
- ✅ **Task 6.5**: Test Configuration and Setup
- ✅ **Task 6.6**: Validation Reports and Documentation

### Validation Criteria Met
- All KNIRV components tested and validated
- Cross-component integration verified
- Performance benchmarks established
- End-to-end workflows validated
- Automated testing infrastructure deployed
- Comprehensive documentation provided

## Future Enhancements

### Phase 2 Preparation
- Expand coverage for new components
- Add chaos engineering tests
- Implement continuous monitoring
- Enhance security testing

### Automation Improvements
- Parallel test execution
- Dynamic test data generation
- Performance regression detection
- Enhanced reporting and visualization

## Support

For issues with the integration test suite:
1. Check the troubleshooting section
2. Review service logs in `logs/` directory
3. Verify configuration in `config/test-config.yaml`
4. Contact the KNIRV development team

---

**Version**: 1.0 (Month 6 Implementation)
**Last Updated**: July 2025
**Maintained By**: KNIRV Development Team

## Dependencies

- Go 1.21+
- Docker (optional)
- curl, lsof (for setup scripts)
- All KNIRV components built and available

## License

This testing suite is part of the KNIRV D-TEN project and follows the same licensing terms.
