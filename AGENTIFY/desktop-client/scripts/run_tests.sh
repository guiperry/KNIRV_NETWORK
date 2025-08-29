#!/bin/bash

# Script to run implementation tests

set -e  # Exit on any error

echo "Running implementation tests..."

# Set demo mode for testing
export AGENTIC_ENGINE_DEMO_MODE=true

# Build the test program
echo "Building test program..."
cd "$(dirname "$0")/.."
go build -o test_implementations ./scripts/test_implementations.go

# Run the tests
echo "Running tests..."
./test_implementations

# Clean up
rm -f test_implementations

echo "Tests completed."