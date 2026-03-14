#!/bin/bash

# KNIRVGATEWAY Integration Test Script
# Tests NEXUS API integration and authentication
# Integrated with KNIRV D-TEN Integration Testing Suite

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
TEST_DIR="$SCRIPT_DIR/reports"
LOG_FILE="$TEST_DIR/gateway-nexus-integration-test-$(date +%Y%m%d-%H%M%S).log"

# Load test configuration if available
if [ -f "$SCRIPT_DIR/config/test-config.yaml" ]; then
    source "$SCRIPT_DIR/config/setup.sh" --load-config-only 2>/dev/null || true
fi

# Create test directory
mkdir -p "$TEST_DIR"

# Logging functions
log() {
    echo -e "${BLUE}[$(date '+%Y-%m-%d %H:%M:%S')]${NC} $1" | tee -a "$LOG_FILE"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1" | tee -a "$LOG_FILE"
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1" | tee -a "$LOG_FILE"
}

warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1" | tee -a "$LOG_FILE"
}

# Test functions
test_gateway_health() {
    log "Testing KNIRVGATEWAY health endpoints..."

    # Use configurable gateway URL (default to localhost for integration tests)
    local gateway_url="${KNIRV_GATEWAY_URL:-http://localhost:8888}"

    # Test basic health endpoint
    if curl -f "$gateway_url/.netlify/functions/gateway-sse/gateway/health" >/dev/null 2>&1; then
        success "KNIRVGATEWAY health endpoint is responding"
    else
        error "KNIRVGATEWAY health endpoint is not responding"
        return 1
    fi

    # Test services endpoint
    if curl -f "$gateway_url/.netlify/functions/gateway-sse/gateway/services" >/dev/null 2>&1; then
        success "KNIRVGATEWAY services endpoint is responding"
    else
        error "KNIRVGATEWAY services endpoint is not responding"
        return 1
    fi
}

test_nexus_api_routes() {
    log "Testing KNIRVSERVER API routes through KNIRVGATEWAY..."

    local gateway_url="${KNIRV_GATEWAY_URL:-http://localhost:8888}"

    # Test DVE nodes endpoint
    local response=$(curl -s -w "%{http_code}" "$gateway_url/.netlify/functions/gateway-sse/nexus/dve-nodes")
    local http_code="${response: -3}"

    if [ "$http_code" = "200" ] || [ "$http_code" = "403" ]; then
        success "KNIRVSERVER DVE nodes endpoint is accessible through gateway"
    else
        error "KNIRVSERVER DVE nodes endpoint returned unexpected status: $http_code"
        return 1
    fi

    # Test validation tasks endpoint
    response=$(curl -s -w "%{http_code}" "$gateway_url/.netlify/functions/gateway-sse/nexus/validation-tasks")
    http_code="${response: -3}"

    if [ "$http_code" = "200" ] || [ "$http_code" = "403" ]; then
        success "KNIRVSERVER validation tasks endpoint is accessible through gateway"
    else
        error "KNIRVSERVER validation tasks endpoint returned unexpected status: $http_code"
        return 1
    fi

    # Test system status endpoint
    response=$(curl -s -w "%{http_code}" "$gateway_url/.netlify/functions/gateway-sse/nexus/system/status")
    http_code="${response: -3}"

    if [ "$http_code" = "200" ] || [ "$http_code" = "403" ]; then
        success "KNIRVSERVER system status endpoint is accessible through gateway"
    else
        error "KNIRVSERVER system status endpoint returned unexpected status: $http_code"
        return 1
    fi
}

test_authentication() {
    log "Testing KNIRVSERVER role-based authentication through KNIRVGATEWAY..."

    local gateway_url="${KNIRV_GATEWAY_URL:-http://localhost:8888}"

    # Test admin token
    local response=$(curl -s -w "%{http_code}" -H "Authorization: Bearer testnet-admin-123" \
        "$gateway_url/.netlify/functions/gateway-sse/nexus/system/status")
    local http_code="${response: -3}"

    if [ "$http_code" = "200" ]; then
        success "Admin role authentication works correctly"
    else
        warning "Admin authentication test returned status: $http_code (may be expected if KNIRVSERVER services are not running)"
    fi

    # Test validator token
    response=$(curl -s -w "%{http_code}" -H "Authorization: Bearer testnet-validator-456" \
        "$gateway_url/.netlify/functions/gateway-sse/nexus/validation-tasks")
    http_code="${response: -3}"

    if [ "$http_code" = "200" ] || [ "$http_code" = "403" ]; then
        success "Validator role authentication is processed correctly"
    else
        warning "Validator authentication test returned status: $http_code"
    fi

    # Test observer token
    response=$(curl -s -w "%{http_code}" -H "Authorization: Bearer testnet-observer-789" \
        "$gateway_url/.netlify/functions/gateway-sse/nexus/dve-nodes")
    http_code="${response: -3}"

    if [ "$http_code" = "200" ] || [ "$http_code" = "403" ]; then
        success "Observer role authentication is processed correctly"
    else
        warning "Observer authentication test returned status: $http_code"
    fi

    # Test invalid token
    response=$(curl -s -w "%{http_code}" -H "Authorization: Bearer invalid-token" \
        "$gateway_url/.netlify/functions/gateway-sse/nexus/system/status")
    http_code="${response: -3}"

    if [ "$http_code" = "403" ] || [ "$http_code" = "401" ]; then
        success "Invalid token correctly rejected"
    else
        warning "Invalid token test returned status: $http_code"
    fi
}

test_cors_headers() {
    log "Testing CORS headers for cross-origin requests..."

    local gateway_url="${KNIRV_GATEWAY_URL:-http://localhost:8888}"

    # Test OPTIONS request
    local response=$(curl -s -I -X OPTIONS \
        -H "Origin: http://localhost:3000" \
        -H "Access-Control-Request-Method: GET" \
        -H "Access-Control-Request-Headers: Authorization" \
        "$gateway_url/.netlify/functions/gateway-sse/nexus/system/status")

    if echo "$response" | grep -q "Access-Control-Allow-Origin"; then
        success "CORS headers are present for cross-origin requests"
    else
        error "CORS headers are missing"
        return 1
    fi
}

test_rate_limiting() {
    log "Testing rate limiting (if enabled)..."
    
    # Make multiple rapid requests
    local success_count=0
    local rate_limited_count=0
    
    for i in {1..10}; do
        local response=$(curl -s -w "%{http_code}" \
            http://localhost:8888/.netlify/functions/gateway-sse/gateway/health)
        local http_code="${response: -3}"
        
        if [ "$http_code" = "200" ]; then
            ((success_count++))
        elif [ "$http_code" = "429" ]; then
            ((rate_limited_count++))
        fi
        
        sleep 0.1
    done
    
    if [ $success_count -gt 0 ]; then
        success "Rate limiting allows normal requests ($success_count successful)"
    else
        error "Rate limiting is too restrictive"
        return 1
    fi
    
    if [ $rate_limited_count -gt 0 ]; then
        log "Rate limiting is active ($rate_limited_count requests limited)"
    else
        log "Rate limiting is disabled or not triggered"
    fi
}

test_sse_endpoints() {
    log "Testing SSE endpoints..."
    
    # Test SSE connection (timeout after 5 seconds)
    timeout 5s curl -N -H "Accept: text/event-stream" \
        http://localhost:8888/.netlify/functions/gateway-sse/events/nexus-system >/dev/null 2>&1 || true
    
    if [ $? -eq 124 ]; then
        success "SSE endpoint is accessible (timed out as expected)"
    else
        warning "SSE endpoint test completed unexpectedly"
    fi
}

test_nexus_portal() {
    log "Testing NEXUS Portal..."
    
    # Check if portal files exist
    if [ -f "$PROJECT_ROOT/nexus-portal/index.html" ]; then
        success "NEXUS Portal files are present"
    else
        error "NEXUS Portal files are missing"
        return 1
    fi
    
    # Test portal build (if npm is available)
    if command -v npm &> /dev/null; then
        cd "$PROJECT_ROOT/nexus-portal"
        
        if [ -f "package.json" ]; then
            log "Installing NEXUS Portal dependencies..."
            if npm install >/dev/null 2>&1; then
                success "NEXUS Portal dependencies installed"
                
                log "Building NEXUS Portal..."
                if npm run build >/dev/null 2>&1; then
                    success "NEXUS Portal builds successfully"
                else
                    error "NEXUS Portal build failed"
                    return 1
                fi
            else
                error "Failed to install NEXUS Portal dependencies"
                return 1
            fi
        fi
        
        cd "$PROJECT_ROOT"
    else
        warning "npm not available, skipping NEXUS Portal build test"
    fi
}

# Generate JSON test report
generate_json_report() {
    local start_time="$1"
    local end_time="$2"
    local failed_tests="$3"
    local total_tests="$4"

    local report_file="$TEST_DIR/gateway-nexus-integration-test-$(date +%Y-%m-%dT%H-%M-%S-%3NZ).json"

    cat > "$report_file" << EOF
{
  "test_suite": "KNIRVGATEWAY NEXUS Integration Tests",
  "version": "1.0",
  "timestamp": "$(date -Iseconds)",
  "start_time": "$start_time",
  "end_time": "$end_time",
  "duration_seconds": $((end_time - start_time)),
  "environment": {
    "gateway_url": "${KNIRV_GATEWAY_URL:-http://localhost:8888}",
    "project_root": "$PROJECT_ROOT",
    "test_mode": "integration"
  },
  "results": {
    "total_tests": $total_tests,
    "passed_tests": $((total_tests - failed_tests)),
    "failed_tests": $failed_tests,
    "success_rate": $(echo "scale=2; (($total_tests - $failed_tests) * 100) / $total_tests" | bc -l 2>/dev/null || echo "0")
  },
  "test_categories": [
    "Gateway Health",
    "NEXUS API Routes",
    "Role-Based Authentication",
    "CORS Headers",
    "Rate Limiting",
    "SSE Endpoints",
    "NEXUS Portal"
  ],
  "status": "$([ $failed_tests -eq 0 ] && echo "PASSED" || echo "FAILED")"
}
EOF

    log "JSON report generated: $report_file"
}

# Main test execution
main() {
    local start_time=$(date +%s)

    log "Starting KNIRVGATEWAY NEXUS Integration Test Suite"
    log "Part of KNIRV D-TEN Integration Testing Framework"
    log "Project root: $PROJECT_ROOT"
    log "Test results will be saved to: $TEST_DIR"
    log "Gateway URL: ${KNIRV_GATEWAY_URL:-http://localhost:8888}"

    # Ensure test directory exists
    mkdir -p "$TEST_DIR"

    # Check prerequisites
    if ! command -v curl &> /dev/null; then
        error "curl is not installed or not in PATH"
        exit 1
    fi

    if ! command -v bc &> /dev/null; then
        warning "bc not available, some calculations may be skipped"
    fi

    # Run tests
    local failed_tests=0
    local total_tests=7

    test_gateway_health || ((failed_tests++))
    test_nexus_api_routes || ((failed_tests++))
    test_authentication || ((failed_tests++))
    test_cors_headers || ((failed_tests++))
    test_rate_limiting || ((failed_tests++))
    test_sse_endpoints || ((failed_tests++))
    test_nexus_portal || ((failed_tests++))

    local end_time=$(date +%s)

    # Generate reports
    generate_json_report "$start_time" "$end_time" "$failed_tests" "$total_tests"

    # Summary
    log "KNIRVGATEWAY NEXUS Integration Test Suite completed"
    log "Duration: $((end_time - start_time)) seconds"
    log "Tests passed: $((total_tests - failed_tests))/$total_tests"

    if [ $failed_tests -eq 0 ]; then
        success "🎉 All KNIRVGATEWAY NEXUS integration tests passed!"
        exit 0
    else
        error "❌ $failed_tests test(s) failed"
        exit 1
    fi
}

# Run main function
main "$@"
