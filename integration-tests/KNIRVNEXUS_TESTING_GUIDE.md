# KNIRVNEXUS Integration Testing Guide

## Overview

This guide covers the comprehensive testing approach for the new KNIRVNEXUS architecture, which has been completely rewritten as a modern Next.js frontend with a distributed Go backend.

## Architecture Changes

### Previous Architecture
- Single Go application on port 8082
- Monolithic structure
- Basic API endpoints

### New Architecture
- **Frontend**: Next.js application with Socket.IO (port 3000)
- **Backend Services**:
  - API Gateway (port 8080)
  - DVE Manager (port 8081) 
  - Validation Core (port 8082)
- **Database**: BuntDB for data persistence
- **P2P Networking**: libp2p for DVE node communication
- **Real-time Updates**: Server-Sent Events (SSE) and Socket.IO

## Test Suites

### 1. Backend Integration Tests (`knirvnexus_backend_integration_test.go`)

**Purpose**: Validate all backend services and their interactions

**Test Coverage**:
- API Gateway health and routing
- DVE Manager node registration and management
- Validation Core task creation and execution
- System health metrics and monitoring
- P2P networking functionality

**Key Test Cases**:
```go
// API Gateway Health Check
func (suite *KNIRVNEXUSBackendTestSuite) TestAPIGatewayHealth()

// DVE Node Operations
func (suite *KNIRVNEXUSBackendTestSuite) TestDVEManagerNodeOperations()

// Validation Task Management
func (suite *KNIRVNEXUSBackendTestSuite) TestValidationCoreOperations()

// System Health Metrics
func (suite *KNIRVNEXUSBackendTestSuite) TestSystemHealthMetrics()
```

**Running Backend Tests**:
```bash
# Run all backend tests
./config/run-tests.sh knirvnexus-backend

# Run specific backend test
go test -v -run "TestKNIRVNEXUSBackendIntegration" ./...
```

### 2. Frontend Integration Tests (`knirvnexus_frontend_integration_test.js`)

**Purpose**: Validate Next.js frontend functionality and connectivity

**Test Coverage**:
- Frontend health endpoints
- Page accessibility and routing
- Static asset loading
- API endpoint connectivity
- Socket.IO real-time communication
- Build artifact integrity
- Environment configuration

**Key Test Cases**:
```javascript
// Frontend Health Check
async testFrontendHealth()

// Page Accessibility
async testFrontendPages()

// Static Assets
async testStaticAssets()

// API Connectivity
async testAPIEndpoints()

// Socket.IO Connectivity
async testSocketIOConnectivity()

// Build Artifacts
async testBuildArtifacts()
```

**Running Frontend Tests**:
```bash
# Run all frontend tests
./config/run-tests.sh knirvnexus-frontend

# Run frontend tests directly
node knirvnexus_frontend_integration_test.js
```

### 3. Combined Integration Tests

**Purpose**: Test frontend-backend integration and complete workflows

**Running Combined Tests**:
```bash
# Run both frontend and backend tests
./config/run-tests.sh knirvnexus
```

## Service Configuration

### Port Allocation
- **Frontend (Next.js)**: 3000
- **API Gateway**: 8080
- **DVE Manager**: 8081
- **Validation Core**: 8082

### Health Endpoints
- Frontend: `http://localhost:3000/api/health`
- API Gateway: `http://localhost:8080/health`
- DVE Manager: `http://localhost:8081/health`
- Validation Core: `http://localhost:8082/health`

### API Endpoints
- DVE Nodes: `http://localhost:8081/api/v1/dve-nodes`
- Validation Tasks: `http://localhost:8082/api/v1/validation-tasks`
- System Health: `http://localhost:8081/api/v1/system/health`

## Setup and Teardown

### Prerequisites
```bash
# Install Node.js dependencies
cd KNIRVNEXUS
npm install

# Build frontend
npm run build

# Build backend services
cd backend
go build -o bin/api-gateway ./cmd/api-gateway
go build -o bin/dve-manager ./cmd/dve-manager
go build -o bin/validation-core ./cmd/validation-core
```

### Starting Services
```bash
# Use the setup script
./config/setup.sh

# Or start manually:
# Backend services
cd KNIRVNEXUS/backend
./bin/api-gateway &
./bin/dve-manager &
./bin/validation-core &

# Frontend
cd KNIRVNEXUS
npm run start &
```

### Stopping Services
```bash
# Use the teardown script
./config/teardown.sh

# Or stop manually by killing processes
```

## Test Data and Fixtures

### DVE Node Test Data
```go
DVENode{
    Name:         "test-node-1",
    TEEType:      "software",
    StakeAmount:  1000000,
    Location:     "us-east-1",
    IPAddress:    "192.168.1.100",
    PublicKey:    "test-public-key",
    Capabilities: []string{"skillnode", "base_llm"},
    Latitude:     40.7128,
    Longitude:    -74.0060,
}
```

### Validation Task Test Data
```go
ValidationTask{
    Type:            "skillnode",
    Priority:        5,
    SkillCode:       "def hello_world():\n    return 'Hello! How can I help you?'",
    RequiredTEEType: "software",
    RequestedBy:     "test-user",
    TestCases: []TestCase{
        {
            ID:          "test-case-1",
            Name:        "Basic functionality test",
            Description: "Tests basic SkillNode functionality",
            Input:       map[string]interface{}{"prompt": "Hello, world!"},
            Expected:    map[string]interface{}{"response": "Hello! How can I help you?"},
            Weight:      1.0,
        },
    },
}
```

## Troubleshooting

### Common Issues

1. **Port Conflicts**
   ```bash
   # Check for processes using ports
   lsof -i :3000
   lsof -i :8080
   lsof -i :8081
   lsof -i :8082
   
   # Kill conflicting processes
   ./config/teardown.sh --force
   ```

2. **Frontend Build Issues**
   ```bash
   # Clean and rebuild
   cd KNIRVNEXUS
   rm -rf .next node_modules
   npm install
   npm run build
   ```

3. **Backend Service Issues**
   ```bash
   # Check logs
   tail -f integration-tests/logs/knirvnexus-*.log
   
   # Rebuild backend
   cd KNIRVNEXUS/backend
   go clean
   go build -o bin/api-gateway ./cmd/api-gateway
   ```

4. **Database Issues**
   ```bash
   # Clean database
   rm -rf KNIRVNEXUS/db/*
   ```

### Debug Mode
```bash
# Run tests with verbose output
./config/run-tests.sh knirvnexus --verbose

# Run individual test with debug
go test -v -run "TestAPIGatewayHealth" ./...
```

## Performance Considerations

### Expected Response Times
- Health endpoints: < 100ms
- DVE node registration: < 500ms
- Validation task creation: < 1s
- Frontend page load: < 2s

### Resource Usage
- Memory: ~100MB per backend service
- CPU: Low during idle, moderate during validation
- Disk: Minimal (database files)

## Integration with Other Components

### KNIRVCHAIN Integration
- DVE nodes register with blockchain
- Validation results recorded on-chain

### KNIRVGRAPH Integration
- SkillNode validation requests
- Validation results feedback

### KNIRVROOT Integration
- Economic transactions
- Staking and rewards

## Continuous Integration

### GitHub Actions Integration
```yaml
name: KNIRVNEXUS Integration Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-go@v2
        with:
          go-version: 1.21
      - uses: actions/setup-node@v2
        with:
          node-version: '18'
      - name: Run KNIRVNEXUS Tests
        run: ./integration-tests/config/run-tests.sh knirvnexus
```

## Future Enhancements

### Planned Test Additions
- Load testing for concurrent validation tasks
- Security testing for TEE simulation
- Performance benchmarking
- Chaos engineering tests
- End-to-end user workflow tests

### Monitoring Integration
- Prometheus metrics collection
- Grafana dashboard validation
- Alert testing

---

**Last Updated**: August 2025
**Version**: 2.0 (New Architecture)
**Maintained By**: KNIRV Development Team
