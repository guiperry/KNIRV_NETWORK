#!/bin/bash

# KNIRVROUTER Integration Test Script
# Tests the revolutionary ErrorContext → KNIRVGRAPH → KNIRVROUTER → SkillNode architecture

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
CONTROLLER_DIR="$PROJECT_ROOT/KNIRVCONTROLLER"
ROUTER_DIR="$PROJECT_ROOT/KNIRVROUTER"
GRAPH_DIR="$PROJECT_ROOT/KNIRVGRAPH"

# Service URLs
CONTROLLER_URL="http://localhost:3000"
ROUTER_URL="http://localhost:5000"
GRAPH_URL="http://localhost:5001"

# Test configuration
TEST_TIMEOUT=60
STARTUP_TIMEOUT=30
VERBOSE=false
CLEANUP_ON_EXIT=true

# Function to print colored output
print_header() {
    echo -e "${PURPLE}╔══════════════════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${PURPLE}║${NC} $1 ${PURPLE}║${NC}"
    echo -e "${PURPLE}╚══════════════════════════════════════════════════════════════════════════════╝${NC}"
}

print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_step() {
    echo -e "${CYAN}[STEP]${NC} $1"
}

# Function to check if service is running
check_service() {
    local url=$1
    local service_name=$2
    local timeout=${3:-10}
    
    print_status "Checking $service_name at $url..."
    
    for i in $(seq 1 $timeout); do
        if curl -s -f "$url/health" >/dev/null 2>&1; then
            print_success "$service_name is running"
            return 0
        fi
        sleep 1
    done
    
    print_error "$service_name is not responding at $url"
    return 1
}

# Function to start KNIRVCONTROLLER
start_controller() {
    print_step "Starting KNIRVCONTROLLER..."
    
    cd "$CONTROLLER_DIR" || exit 1
    
    # Install dependencies if needed
    if [ ! -d "node_modules" ]; then
        print_status "Installing KNIRVCONTROLLER dependencies..."
        npm install
    fi
    
    # Build if needed
    if [ ! -d "dist" ]; then
        print_status "Building KNIRVCONTROLLER..."
        npm run build
    fi
    
    # Start in background
    print_status "Starting KNIRVCONTROLLER server..."
    npm start &
    CONTROLLER_PID=$!
    
    # Wait for startup
    sleep 5
    
    if check_service "$CONTROLLER_URL" "KNIRVCONTROLLER" $STARTUP_TIMEOUT; then
        print_success "KNIRVCONTROLLER started successfully (PID: $CONTROLLER_PID)"
        return 0
    else
        print_error "Failed to start KNIRVCONTROLLER"
        return 1
    fi
}

# Function to start KNIRVROUTER
start_router() {
    print_step "Starting KNIRVROUTER..."
    
    cd "$ROUTER_DIR" || exit 1
    
    # Build if needed
    if [ ! -f "knirv-router" ]; then
        print_status "Building KNIRVROUTER..."
        go build -o knirv-router ./cmd/router
    fi
    
    # Start in background
    print_status "Starting KNIRVROUTER server..."
    ./knirv-router &
    ROUTER_PID=$!
    
    # Wait for startup
    sleep 3
    
    if check_service "$ROUTER_URL" "KNIRVROUTER" $STARTUP_TIMEOUT; then
        print_success "KNIRVROUTER started successfully (PID: $ROUTER_PID)"
        return 0
    else
        print_error "Failed to start KNIRVROUTER"
        return 1
    fi
}

# Function to start KNIRVGRAPH
start_graph() {
    print_step "Starting KNIRVGRAPH..."
    
    cd "$GRAPH_DIR" || exit 1
    
    # Build if needed
    if [ ! -f "knirv-graph" ]; then
        print_status "Building KNIRVGRAPH..."
        go build -o knirv-graph ./cmd/graph
    fi
    
    # Start in background
    print_status "Starting KNIRVGRAPH server..."
    ./knirv-graph &
    GRAPH_PID=$!
    
    # Wait for startup
    sleep 3
    
    if check_service "$GRAPH_URL" "KNIRVGRAPH" $STARTUP_TIMEOUT; then
        print_success "KNIRVGRAPH started successfully (PID: $GRAPH_PID)"
        return 0
    else
        print_error "Failed to start KNIRVGRAPH"
        return 1
    fi
}

# Function to run TypeScript integration tests
run_typescript_tests() {
    print_step "Running TypeScript KNIRVROUTER integration tests..."
    
    cd "$CONTROLLER_DIR" || exit 1
    
    if npm test -- --testPathPattern="integration/knirvrouter-integration.test.ts" --verbose; then
        print_success "TypeScript integration tests passed"
        return 0
    else
        print_error "TypeScript integration tests failed"
        return 1
    fi
}

# Function to run Go integration tests
run_go_tests() {
    print_step "Running Go KNIRVROUTER integration tests..."
    
    cd "$PROJECT_ROOT/integration-tests" || exit 1
    
    if go test -v -timeout ${TEST_TIMEOUT}s ./knirvcontroller_router_integration_test.go; then
        print_success "Go integration tests passed"
        return 0
    else
        print_error "Go integration tests failed"
        return 1
    fi
}

# Function to test ErrorContext → KNIRVGRAPH → KNIRVROUTER flow
test_error_context_flow() {
    print_step "Testing ErrorContext → KNIRVGRAPH → KNIRVROUTER flow..."
    
    # Test ErrorContext generation
    local error_context='{
        "errorId": "test-error-001",
        "errorType": "skill_invocation_request",
        "errorMessage": "Test skill invocation via ErrorContext",
        "stackTrace": "test stack trace",
        "userContext": {
            "userAddress": "knirv1test123456789",
            "nrnAmount": "100"
        },
        "agentId": "test-agent-001",
        "timestamp": '$(date +%s000)',
        "severity": "medium"
    }'
    
    # Submit to KNIRVCONTROLLER
    print_status "Submitting ErrorContext to KNIRVCONTROLLER..."
    local response=$(curl -s -X POST "$CONTROLLER_URL/api/process-error-context" \
        -H "Content-Type: application/json" \
        -d "$error_context")
    
    if echo "$response" | grep -q '"status":"SUCCESS"'; then
        print_success "ErrorContext processing successful"
        return 0
    else
        print_error "ErrorContext processing failed: $response"
        return 1
    fi
}

# Function to test skill invocation
test_skill_invocation() {
    print_step "Testing skill invocation via KNIRVROUTER..."
    
    local skill_request='{
        "skillId": "test-skill-001",
        "userAddress": "knirv1test123456789",
        "nrnAmount": "100",
        "parameters": {
            "agentId": "test-agent-001",
            "capabilities": ["text-processing", "analysis"],
            "priority": "high",
            "useP2P": true,
            "useWASM": true
        },
        "priority": "high"
    }'
    
    print_status "Submitting skill invocation request..."
    local response=$(curl -s -X POST "$CONTROLLER_URL/api/invoke-skill" \
        -H "Content-Type: application/json" \
        -d "$skill_request")
    
    if echo "$response" | grep -q '"status":"SUCCESS"'; then
        print_success "Skill invocation successful"
        local request_id=$(echo "$response" | grep -o '"requestId":"[^"]*"' | cut -d'"' -f4)
        print_status "Request ID: $request_id"
        return 0
    else
        print_error "Skill invocation failed: $response"
        return 1
    fi
}

# Function to test LoRA adapter operations
test_lora_operations() {
    print_step "Testing LoRA adapter operations..."
    
    # Test registration
    local lora_adapter='{
        "adapterName": "test-lora-adapter",
        "description": "Test LoRA adapter for integration testing",
        "baseModelCompatibility": "hrm-v1",
        "version": 1,
        "rank": 16,
        "alpha": 0.5,
        "metadata": {
            "test": "true",
            "author": "integration-test"
        }
    }'
    
    print_status "Registering LoRA adapter..."
    local reg_response=$(curl -s -X POST "$CONTROLLER_URL/api/register-lora-adapter" \
        -H "Content-Type: application/json" \
        -d "$lora_adapter")
    
    if echo "$reg_response" | grep -q '"status":"SUCCESS"'; then
        print_success "LoRA adapter registration successful"
        local adapter_id=$(echo "$reg_response" | grep -o '"adapterId":"[^"]*"' | cut -d'"' -f4)
        print_status "Adapter ID: $adapter_id"
    else
        print_error "LoRA adapter registration failed: $reg_response"
        return 1
    fi
    
    # Test retrieval
    print_status "Retrieving LoRA adapters..."
    local get_response=$(curl -s "$CONTROLLER_URL/api/lora-adapters")
    
    if echo "$get_response" | grep -q "test-lora-adapter"; then
        print_success "LoRA adapter retrieval successful"
        return 0
    else
        print_error "LoRA adapter retrieval failed: $get_response"
        return 1
    fi
}

# Function to cleanup processes
cleanup() {
    if [ "$CLEANUP_ON_EXIT" = true ]; then
        print_step "Cleaning up processes..."
        
        if [ ! -z "$CONTROLLER_PID" ]; then
            print_status "Stopping KNIRVCONTROLLER (PID: $CONTROLLER_PID)..."
            kill $CONTROLLER_PID 2>/dev/null || true
        fi
        
        if [ ! -z "$ROUTER_PID" ]; then
            print_status "Stopping KNIRVROUTER (PID: $ROUTER_PID)..."
            kill $ROUTER_PID 2>/dev/null || true
        fi
        
        if [ ! -z "$GRAPH_PID" ]; then
            print_status "Stopping KNIRVGRAPH (PID: $GRAPH_PID)..."
            kill $GRAPH_PID 2>/dev/null || true
        fi
        
        # Wait a moment for graceful shutdown
        sleep 2
        
        # Force kill if still running
        pkill -f "knirv-router" 2>/dev/null || true
        pkill -f "knirv-graph" 2>/dev/null || true
        pkill -f "npm.*start" 2>/dev/null || true
        
        print_success "Cleanup completed"
    fi
}

# Function to display usage
usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Test the KNIRVROUTER integration with KNIRVCONTROLLER"
    echo ""
    echo "Options:"
    echo "  --verbose            Enable verbose output"
    echo "  --no-cleanup         Skip cleanup on exit"
    echo "  --timeout SECONDS    Test timeout (default: 60)"
    echo "  --help               Show this help message"
    echo ""
    echo "This script will:"
    echo "  1. Start KNIRVCONTROLLER, KNIRVROUTER, and KNIRVGRAPH services"
    echo "  2. Run TypeScript integration tests"
    echo "  3. Run Go integration tests"
    echo "  4. Test ErrorContext → KNIRVGRAPH → KNIRVROUTER flow"
    echo "  5. Test skill invocation and LoRA adapter operations"
    echo "  6. Cleanup all processes"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --verbose)
            VERBOSE=true
            shift
            ;;
        --no-cleanup)
            CLEANUP_ON_EXIT=false
            shift
            ;;
        --timeout)
            TEST_TIMEOUT="$2"
            shift 2
            ;;
        --help)
            usage
            exit 0
            ;;
        -*)
            print_error "Unknown option: $1"
            usage
            exit 1
            ;;
        *)
            print_error "Unexpected argument: $1"
            usage
            exit 1
            ;;
    esac
done

# Set up trap for cleanup on exit
trap cleanup EXIT

# Main execution
main() {
    print_header "KNIRVROUTER Integration Test Suite"
    
    print_status "Starting KNIRVROUTER integration tests..."
    print_status "Project Root: $PROJECT_ROOT"
    print_status "Test Timeout: ${TEST_TIMEOUT}s"
    print_status "Cleanup on Exit: $CLEANUP_ON_EXIT"
    
    local test_result=0
    
    # Start services
    start_graph || exit 1
    start_router || exit 1
    start_controller || exit 1
    
    # Run tests
    run_typescript_tests || test_result=$?
    
    if [ $test_result -eq 0 ]; then
        run_go_tests || test_result=$?
    fi
    
    if [ $test_result -eq 0 ]; then
        test_error_context_flow || test_result=$?
    fi
    
    if [ $test_result -eq 0 ]; then
        test_skill_invocation || test_result=$?
    fi
    
    if [ $test_result -eq 0 ]; then
        test_lora_operations || test_result=$?
    fi
    
    # Display results
    if [ $test_result -eq 0 ]; then
        print_header "🎉 ALL KNIRVROUTER INTEGRATION TESTS PASSED! 🎉"
        print_success "Revolutionary ErrorContext → KNIRVGRAPH → KNIRVROUTER architecture is working!"
        exit 0
    else
        print_header "❌ KNIRVROUTER INTEGRATION TESTS FAILED"
        print_error "Some tests failed with exit code $test_result"
        exit $test_result
    fi
}

# Run main function
main "$@"
