#!/bin/bash

# Comprehensive Test Runner for Agentic-Engine
# This script runs the complete test suite with proper setup and cleanup

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TEST_MODE="${1:-all}"
VERBOSE="${VERBOSE:-false}"
SKIP_SETUP="${SKIP_SETUP:-false}"
PARALLEL="${PARALLEL:-true}"

# Test results tracking
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0
SKIPPED_TESTS=0

echo -e "${BLUE}🚀 Agentic-Engine Comprehensive Test Suite${NC}"
echo -e "${BLUE}===========================================${NC}"
echo -e "Project Root: ${PROJECT_ROOT}"
echo -e "Test Mode: ${TEST_MODE}"
echo -e "Verbose: ${VERBOSE}"
echo -e "Skip Setup: ${SKIP_SETUP}"
echo ""

# Function to log test results
log_test_result() {
    local test_name="$1"
    local result="$2"
    local duration="$3"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    case "$result" in
        "PASS")
            PASSED_TESTS=$((PASSED_TESTS + 1))
            echo -e "${GREEN}✅ PASS${NC} $test_name ${CYAN}(${duration}s)${NC}"
            ;;
        "FAIL")
            FAILED_TESTS=$((FAILED_TESTS + 1))
            echo -e "${RED}❌ FAIL${NC} $test_name ${CYAN}(${duration}s)${NC}"
            ;;
        "SKIP")
            SKIPPED_TESTS=$((SKIPPED_TESTS + 1))
            echo -e "${YELLOW}⏭️ SKIP${NC} $test_name"
            ;;
    esac
}

# Function to run a test with timing
run_test() {
    local test_name="$1"
    local test_command="$2"
    local required="${3:-false}"
    
    echo -e "\n${PURPLE}🧪 Running: $test_name${NC}"
    echo -e "${CYAN}Command: $test_command${NC}"
    
    start_time=$(date +%s)
    
    if [ "$VERBOSE" = "true" ]; then
        if eval "$test_command"; then
            end_time=$(date +%s)
            duration=$((end_time - start_time))
            log_test_result "$test_name" "PASS" "$duration"
            return 0
        else
            end_time=$(date +%s)
            duration=$((end_time - start_time))
            log_test_result "$test_name" "FAIL" "$duration"
            if [ "$required" = "true" ]; then
                echo -e "${RED}❌ Required test failed, aborting${NC}"
                exit 1
            fi
            return 1
        fi
    else
        if eval "$test_command" >/dev/null 2>&1; then
            end_time=$(date +%s)
            duration=$((end_time - start_time))
            log_test_result "$test_name" "PASS" "$duration"
            return 0
        else
            end_time=$(date +%s)
            duration=$((end_time - start_time))
            log_test_result "$test_name" "FAIL" "$duration"
            if [ "$required" = "true" ]; then
                echo -e "${RED}❌ Required test failed, aborting${NC}"
                exit 1
            fi
            return 1
        fi
    fi
}

# Function to setup test environment
setup_test_environment() {
    if [ "$SKIP_SETUP" = "true" ]; then
        echo -e "${YELLOW}⏭️ Skipping test environment setup${NC}"
        return 0
    fi
    
    echo -e "\n${BLUE}🔧 Setting up test environment...${NC}"
    
    # Set test environment variables
    export AGENTIC_ENGINE_DEMO_MODE=true
    export AGENTIC_ENGINE_TEST_MODE=true
    export CI=true
    
    # Create test directories
    mkdir -p /tmp/knirv-engine-test
    
    # Install frontend dependencies if needed
    if [ -d "$PROJECT_ROOT/gui" ] && [ -f "$PROJECT_ROOT/gui/package.json" ]; then
        echo -e "${CYAN}📦 Installing frontend dependencies...${NC}"
        cd "$PROJECT_ROOT/gui"
        npm install >/dev/null 2>&1 || echo -e "${YELLOW}⚠️ Frontend dependency installation failed${NC}"
        cd "$PROJECT_ROOT"
    fi
    
    echo -e "${GREEN}✅ Test environment setup completed${NC}"
}

# Function to cleanup test environment
cleanup_test_environment() {
    echo -e "\n${BLUE}🧹 Cleaning up test environment...${NC}"
    
    # Kill any running test servers
    pkill -f "knirv-engine" >/dev/null 2>&1 || true
    pkill -f "npm.*dev" >/dev/null 2>&1 || true
    
    # Clean up test files
    rm -rf /tmp/knirv-engine-test >/dev/null 2>&1 || true
    
    echo -e "${GREEN}✅ Cleanup completed${NC}"
}

# Function to print test summary
print_test_summary() {
    echo -e "\n${BLUE}📊 Test Summary${NC}"
    echo -e "${BLUE}===============${NC}"
    echo -e "Total Tests: ${TOTAL_TESTS}"
    echo -e "${GREEN}Passed: ${PASSED_TESTS}${NC}"
    echo -e "${RED}Failed: ${FAILED_TESTS}${NC}"
    echo -e "${YELLOW}Skipped: ${SKIPPED_TESTS}${NC}"
    
    if [ $FAILED_TESTS -eq 0 ]; then
        echo -e "\n${GREEN}🎉 All tests passed!${NC}"
        return 0
    else
        echo -e "\n${RED}❌ Some tests failed${NC}"
        return 1
    fi
}

# Trap to ensure cleanup on exit
trap cleanup_test_environment EXIT

# Change to project root
cd "$PROJECT_ROOT"

# Setup test environment
setup_test_environment

# Run tests based on mode
case "$TEST_MODE" in
    "unit")
        echo -e "\n${PURPLE}🧪 Running Unit Tests${NC}"
        run_test "Go Unit Tests" "make test-unit" true
        ;;
        
    "integration")
        echo -e "\n${PURPLE}🔗 Running Integration Tests${NC}"
        run_test "Integration Tests" "make test-integration" true
        ;;
        
    "frontend")
        echo -e "\n${PURPLE}⚛️ Running Frontend Tests${NC}"
        run_test "Frontend Tests" "make test-frontend"
        run_test "Frontend Linting" "make frontend/lint"
        run_test "TypeScript Check" "make frontend/type-check"
        ;;
        
    "api")
        echo -e "\n${PURPLE}🌐 Running API Tests${NC}"
        run_test "API Tests" "make test-api" true
        run_test "API Endpoint Tests" "make test-api-endpoints"
        run_test "Simple API Tests" "make test-api-simple"
        ;;
        
    "mcp")
        echo -e "\n${PURPLE}🔌 Running MCP Tests${NC}"
        run_test "MCP Integration Tests" "make test-mcp"
        ;;
        
    "cloud")
        echo -e "\n${PURPLE}☁️ Running Cloud Tests${NC}"
        run_test "Cloud Deployment Tests" "make test-cloud"
        ;;
        
    "desktop")
        echo -e "\n${PURPLE}🖥️ Running Desktop Tests${NC}"
        run_test "Desktop Application Tests" "make test-desktop"
        ;;
        
    "security")
        echo -e "\n${PURPLE}🔒 Running Security Tests${NC}"
        run_test "Security Tests" "make test-security"
        ;;
        
    "performance")
        echo -e "\n${PURPLE}⚡ Running Performance Tests${NC}"
        run_test "Performance Tests" "make test-performance"
        ;;
        
    "ci")
        echo -e "\n${PURPLE}🤖 Running CI Test Suite${NC}"
        run_test "Unit Tests" "make test-unit" true
        run_test "Integration Tests" "make test-integration" true
        run_test "API Tests" "make test-api" true
        run_test "Frontend Tests" "make test-frontend"
        ;;
        
    "all")
        echo -e "\n${PURPLE}🎯 Running All Tests${NC}"
        run_test "Unit Tests" "make test-unit" true
        run_test "Integration Tests" "make test-integration" true
        run_test "Frontend Tests" "make test-frontend"
        run_test "API Tests" "make test-api" true
        run_test "MCP Tests" "make test-mcp"
        run_test "Security Tests" "make test-security"
        run_test "WASM Tests" "make test-wasm"
        run_test "Connectivity Tests" "make test-connectivity"
        run_test "Chat Tests" "make test-chat"
        ;;

    "connectivity")
        echo -e "\n${PURPLE}🔗 Running Connectivity Tests${NC}"
        run_test "Agent Connectivity Tests" "make test-connectivity"
        ;;

    "chat")
        echo -e "\n${PURPLE}💬 Running Chat Tests${NC}"
        run_test "Agent Chat Tests" "make test-chat"
        ;;

    "full")
        echo -e "\n${PURPLE}🚀 Running Full Test Suite${NC}"
        run_test "Unit Tests" "make test-unit" true
        run_test "Integration Tests" "make test-integration" true
        run_test "Frontend Tests" "make test-frontend"
        run_test "API Tests" "make test-api" true
        run_test "MCP Tests" "make test-mcp"
        run_test "Security Tests" "make test-security"
        run_test "WASM Tests" "make test-wasm"
        run_test "Connectivity Tests" "make test-connectivity"
        run_test "Chat Tests" "make test-chat"
        run_test "Cloud Tests" "make test-cloud"
        run_test "Desktop Tests" "make test-desktop"
        run_test "Performance Tests" "make test-performance"
        ;;
        
    *)
        echo -e "${RED}❌ Unknown test mode: $TEST_MODE${NC}"
        echo -e "Available modes: unit, integration, frontend, api, mcp, cloud, desktop, security, performance, wasm, connectivity, chat, ci, all, full"
        exit 1
        ;;
esac

# Print summary and exit with appropriate code
print_test_summary
