

---

**Source**: KNIRVROOT/docs/tunnel_registry_tests.md

# Tunnel Registry Tests

This document describes the test suite for the Tunnel Registry system.

## Overview

The test suite consists of:

1. **Node.js Unit Tests**: Tests for the Node.js components of the tunnel registry
2. **Go Unit Tests**: Tests for the Go components of the tunnel registry
3. **Integration Tests**: Tests that verify the interaction between the Node.js and Go components

## Test Files

### Node.js Tests

- `agent-tunnel-registry/tests/registry-manager.test.js`: Tests for the registry manager
- `agent-tunnel-registry/tests/uri-routes.test.js`: Tests for the URI routes

### Go Tests

- `uri/uri_resolver_test.go`: Tests for the URI resolver
- `internal_api_test.go`: Tests for the internal API endpoints
- `tests/tunnel_registry_integration_test.go`: Integration tests for the tunnel registry system

## Running the Tests

A script is provided to run all the tests:

```bash
./scripts/run_tunnel_registry_tests.sh
```

To run the integration tests as well:

```bash
./scripts/run_tunnel_registry_tests.sh --integration
```

## Test Coverage

### Registry Manager Tests

The registry manager tests cover:

- Node registration via API
- Node registration via control socket
- Tunneled resource mapping
- Node lookup by peer ID and chain ID
- Node deregistration

### URI Routes Tests

The URI routes tests cover:

- URI generation for tunneled and direct nodes
- URI resolution for various types of URIs:
  - Tunneled resources
  - Direct nodes by peer ID
  - Direct nodes by chain ID
  - Resources via DHT lookup
  - Capabilities via blockchain lookup

### URI Resolver Tests

The URI resolver tests cover:

- URI parsing
- URI resolution
- Connection establishment

### Internal API Tests

The internal API tests cover:

- DHT resource lookup
- DHT resource announcement
- Capability lookup
- ID existence check

### Integration Tests

The integration tests cover:

- Node registration
- URI generation and resolution
- Cross-component communication

## Test Dependencies

- Node.js tests require Jest, axios, and axios-mock-adapter
- Go tests require the standard Go testing package and testify
- Integration tests require both Node.js and Go servers to be running

## Mocking

- Node.js tests use axios-mock-adapter to mock HTTP requests
- Go tests use httptest to mock HTTP servers and custom mock implementations for interfaces

## Continuous Integration

The tests can be run in CI environments by setting the appropriate environment variables:

- `SKIP_INTEGRATION_TESTS=true` to skip integration tests

---

<div class="footer-links">


© 2025 KNIRV Network
</div>
