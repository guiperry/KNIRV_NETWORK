#!/bin/bash

# Run Tunnel Registry Tests
# This script runs all the tests for the tunnel registry system

set -e

# Change to the project root directory
cd "$(dirname "$0")/.."

echo "Running Node.js tests..."
cd agent-tunnel-registry
npm install --no-save jest axios axios-mock-adapter
npx jest tests/registry-manager.test.js tests/uri-routes.test.js
cd ..

echo "Running Go tests..."
go test ./uri -v
go test ./tests -v -run TestTunnelRegistryUnitTests
go test . -v -run TestHandleInternalDHT

echo "Running integration tests..."
if [ "$1" == "--integration" ]; then
  go test ./tests -v -run TestTunnelRegistryIntegration
else
  echo "Skipping integration tests. Use --integration flag to run them."
fi

echo "All tests completed successfully!"