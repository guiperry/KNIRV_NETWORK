#!/bin/bash

# Script to run all tests for the KNIRVCHAIN CLI

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Print header
print_header() {
    echo -e "\n${YELLOW}=======================================${NC}"
    echo -e "${YELLOW}$1${NC}"
    echo -e "${YELLOW}=======================================${NC}\n"
}

# Run tests with proper output
run_test() {
    local test_type=$1
    local test_path=$2
    local test_flags=$3

    print_header "Running $test_type tests"
    
    if go test $test_flags $test_path; then
        echo -e "\n${GREEN}✓ $test_type tests passed${NC}\n"
    else
        echo -e "\n${RED}✗ $test_type tests failed${NC}\n"
        exit 1
    fi
}

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo -e "${RED}Error: Go is not installed or not in PATH${NC}"
    exit 1
fi

# Get the project root directory
PROJECT_ROOT=$(git rev-parse --show-toplevel 2>/dev/null || pwd)
cd "$PROJECT_ROOT"

# Run unit tests
run_test "Unit" "./test/unit/..." "-v -race"

# Run integration tests
run_test "Integration" "./test/integration/..." "-v -race"

# Run end-to-end tests
run_test "End-to-End" "./test/e2e/..." "-v -race"

# Run benchmark tests
print_header "Running benchmark tests"
go test -bench=. ./test/benchmark/...

# Run coverage tests
print_header "Running tests with coverage"
go test -v -race -coverprofile=/tmp/coverage.out ./...
go tool cover -func=/tmp/coverage.out

# Optional: Generate HTML coverage report
if [ "$1" == "--html" ]; then
    go tool cover -html=/tmp/coverage.out -o coverage.html
    echo -e "\n${GREEN}Coverage report generated: coverage.html${NC}\n"
fi

print_header "All tests completed successfully!"