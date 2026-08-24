# KNIRV-Engine Comprehensive Test Suite

[TOC]

## Overview

This document outlines the comprehensive test suite for the KNIRV-Engine application, covering both cloud and desktop deployments.  The suite aims for full coverage across all components, including unit, integration, and end-to-end tests.  This README consolidates information from previous documentation files.

## Test Categories

### 1. Unit Tests

* **Backend Services:** API endpoints, inference services, agent management, database operations.
* **Frontend Components:** React components, utilities, hooks, state management.
* **Agent System:** Plugin building, WASM agents, sub-agents, orchestration patterns.
* **MCP Integration:** Server discovery, installation, capability management.

### 2. Integration Tests

* **API Integration:** End-to-end API workflows.  (See detailed description below in "Agent Builder Integration Test" section).
* **Frontend-Backend Integration:** Complete user workflows.
* **Agent-MCP Integration:** Agent-capability interactions.
* **Database Integration:** Data persistence and migration.  (See detailed description below in "Agent Builder Integration Test" section).

### 3. Cloud Deployment Tests

* **Cross-Platform Builds:** Linux, Windows, macOS builds.
* **Cloud-Specific Features:** Server deployment, scaling, cloud storage.
* **API Performance:** Load testing, concurrent users.
* **Security:** Authentication, authorization, data protection.

### 4. Desktop Application Tests

* **Electron Integration:** Desktop-specific features.
* **Cross-Platform Desktop:** Windows, macOS, Linux desktop apps.
* **Desktop-Specific Features:** System tray, file system access, native dialogs.
* **Performance:** Memory usage, startup time, responsiveness.

### 5. End-to-End Tests

* **Complete User Workflows:** Agent creation to deployment.
* **Multi-Agent Orchestration:** Complex agent interactions.
* **Real-World Scenarios:** Actual use cases with real APIs.


## Integration Tests: Agent Builder Integration Test

**File:** `test/test_agent_builder_integration.go`

This integration test validates the complete agent building pipeline, including:

1. **Template Management:** Tests the `/templates` endpoint to retrieve available agent templates.
2. **Agent Creation:** Creates a new agent via the `/agents` endpoint.
3. **Plugin Building:** Builds an agent plugin using the `/agents/{id}/build` endpoint.
4. **Build Status:** Checks build status and progress.
5. **Sub-Agent Functionality:** Tests spawning and managing sub-agents.
6. **Plugin Retrieval:** Lists compiled plugins via the `/plugins` endpoint.

**Running the Integration Test:**

1. Start the KNIRVENGINE server:
   ```bash
   make run
   # or
   go run .
   ```

2. In a separate terminal, run the integration test:
   ```bash
   cd test
   go run test_agent_builder_integration.go
   ```

**Expected Output:**

```
Testing Agent Builder Integration...

1. Testing GET /templates
Status: 200
Response: {"templates":[...]}

2. Testing agent creation and plugin building
Create Agent Status: 201
Create Agent Response: {"agent":{"id":"...","name":"Test Agent",...}}
Build Agent Status: 202
Build Agent Response: {"build_id":"...","status":"started"}
Build Status: 200
Build Status Response: {"status":"completed","progress":100}

3. Testing sub-agent functionality
Spawn Sub-Agent Status: 201
Spawn Sub-Agent Response: {"sub_agent":{"id":"...","template":"python"}}
Get Sub-Agents Status: 200
Get Sub-Agents Response: {"sub_agents":[...]}

4. Testing GET /plugins
Status: 200
Response: {"plugins":[...]}

Agent Builder Integration Test Complete!
```

**Test Configuration:**

* **Base URL:** `http://localhost:8081/api/v1`
* **Default Agent:** Creates a "Test Agent" with standard template.
* **Sub-Agent Template:** Uses Python template for sub-agent testing.

**Troubleshooting:**

* **Connection Refused:** Ensure the KNIRVENGINE server is running on port 8081.
* **404 Errors:** Verify the API endpoints are properly implemented.
* **Build Failures:** Check that the AgentBuilder service is properly configured.
* **Timeout Issues:** The test includes small delays between operations; increase if needed.


## Test Infrastructure

### Existing Components

* `test/builder_test.go` - Agent builder integration tests
* `test/enhanced_agent_management_test.go` - Advanced agent management
* `test/frontend_test.go` - Frontend test runner
* `test/orchestration_integration_test.go` - Orchestration patterns
* `test/performance_load_test.go` - Performance testing
* `test/security_test.go` - Security and TEE testing
* `test/wasm/` - WASM agent testing
* `scripts/test_mcp_integration.sh` - MCP integration tests
* `scripts/test_frontend_mcp_integration.sh` - Frontend MCP tests

### New Components to Implement (from Comprehensive Test Plan)

* **API Test Suite:** Comprehensive API endpoint testing.
* **Frontend Component Tests:** Individual React component tests.
* **Cloud Deployment Tests:** Cloud-specific testing scenarios.
* **Desktop Integration Tests:** Electron-specific testing.
* **Performance Benchmarks:** Standardized performance metrics.
* **Security Audit Tests:** Comprehensive security testing.


## Test Execution Strategy

### Local Development

```bash
make test-unit          # Run unit tests only
make test-integration   # Run integration tests
make test-frontend      # Run frontend tests
make test-all          # Run all tests
```

### Cloud Testing

```bash
make test-cloud        # Test cloud deployment scenarios
make test-performance  # Run performance tests
make test-security     # Run security audit
```

### Desktop Testing

```bash
make test-desktop      # Test desktop application
make test-cross-platform # Test all desktop platforms
```

### Continuous Integration

```bash
make test-ci           # Run CI-appropriate test subset
make test-full         # Run complete test suite
```

## Test Data Management

### Test Fixtures

* **Agent Templates:** Standard test agent configurations.
* **MCP Servers:** Mock MCP server configurations.
* **User Data:** Test user accounts and permissions.
* **API Responses:** Mock API response data.

### Test Databases

* **SQLite Test DB:** Isolated test database instances.
* **Memory Databases:** Fast in-memory testing.
* **Migration Testing:** Database schema migration tests.


## Coverage Requirements

### Minimum Coverage Targets

* **Backend Code:** 80% line coverage.
* **Frontend Components:** 75% component coverage.
* **API Endpoints:** 100% endpoint coverage.
* **Critical Paths:** 95% coverage for security and data integrity.

### Coverage Reporting

* **Go Coverage:** Built-in Go coverage tools.
* **Frontend Coverage:** Jest/React Testing Library coverage.
* **Integration Coverage:** Custom coverage tracking.
* **Combined Reports:** Unified coverage reporting.


## Test Environment Setup

### Prerequisites

* Go 1.21+ for backend tests.
* Node.js 18+ for frontend tests.
* Docker for containerized testing.
* Electron for desktop testing.

### Environment Variables

```bash
AGENTIC_ENGINE_DEMO_MODE=true
AGENTIC_ENGINE_TEST_MODE=true
TEST_DATABASE_URL=sqlite://test.db
TEST_API_BASE_URL=http://localhost:8081
```

## Continuous Integration Integration

### GitHub Actions

* **Pull Request Tests:** Run subset of tests on PRs.
* **Main Branch Tests:** Full test suite on main branch.
* **Release Tests:** Complete validation before releases.
* **Performance Regression:** Track performance over time.

### Test Reporting

* **Test Results:** JUnit XML format for CI integration.
* **Coverage Reports:** Codecov integration.
* **Performance Metrics:** Benchmark result tracking.
* **Security Scan Results:** Security audit reporting.


## Troubleshooting and Debugging

### Test Debugging

* **Verbose Output:** Detailed test execution logs.
* **Test Isolation:** Run individual test suites.
* **Debug Mode:** Step-through debugging for complex tests.
* **Log Analysis:** Structured logging for test failures.

### Common Issues

* **Port Conflicts:** Automated port management.
* **Database Locks:** Proper test isolation.
* **Timing Issues:** Robust retry mechanisms.
* **Resource Cleanup:** Automatic cleanup after tests.


## Adding New Tests

1. Create a new `.go` file in the `test` directory.
2. Use `package main` and include a `main()` function.
3. Follow the naming convention: `test_<feature>_integration.go`.
4. Document the test purpose and usage in this README.
5. Include proper error handling and status reporting.


## Test Data

Test data and fixtures should be placed in subdirectories:

* `fixtures/` - Static test data files.
* `mocks/` - Mock data for testing.
* `configs/` - Test-specific configuration files.


## Best Practices

* Always check HTTP status codes.
* Include meaningful error messages.
* Test both success and failure scenarios.
* Clean up any test data created during testing.
* Use timeouts for operations that might hang.
* Document expected behavior and outputs.

## Next Steps Recommended (from Comprehensive Test Plan)

* Fix Template Issues: Add missing agent template files to resolve build test failures.
* Address Race Conditions: Fix the CustomTEE race conditions in the agentify package.
* Implement Missing Endpoints: Complete implementation of API endpoints that return 404.
* Enhance Coverage: Add more specific test cases for edge cases and error scenarios.
* Performance Optimization: Optimize test execution speed and add parallel execution.

