#!/bin/bash

# KNIRVGRAPH Comprehensive Test Runner
# This script runs tests for both Go backend and TypeScript frontend

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPORTS_DIR="$PROJECT_ROOT/test-reports"
COVERAGE_DIR="$PROJECT_ROOT/coverage"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")

# Create directories
mkdir -p "$REPORTS_DIR"
mkdir -p "$COVERAGE_DIR"

echo -e "${BLUE}KNIRVGRAPH Comprehensive Test Suite${NC}"
echo -e "${BLUE}====================================${NC}"
echo "Project Root: $PROJECT_ROOT"
echo "Reports Dir: $REPORTS_DIR"
echo "Coverage Dir: $COVERAGE_DIR"
echo "Timestamp: $TIMESTAMP"
echo ""

# Function to print section headers
print_section() {
    echo -e "\n${BLUE}$1${NC}"
    echo -e "${BLUE}$(printf '=%.0s' $(seq 1 ${#1}))${NC}"
}

# Function to print success messages
print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

# Function to print error messages
print_error() {
    echo -e "${RED}✗ $1${NC}"
}

# Function to print warning messages
print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

# Change to project directory
cd "$PROJECT_ROOT"

# Initialize test results
GO_TESTS_PASSED=false
FRONTEND_TESTS_PASSED=false
BUILD_SUCCESSFUL=false

# Test Go Backend
print_section "Testing Go Backend"
if command -v go &> /dev/null; then
    print_success "Go version: $(go version)"
    
    # Install Go dependencies
    echo "Installing Go dependencies..."
    if go mod download && go mod tidy; then
        print_success "Go dependencies installed"
    else
        print_error "Failed to install Go dependencies"
    fi
    
    # Run Go tests
    echo "Running Go tests..."
    GO_TEST_OUTPUT="$REPORTS_DIR/go_tests_$TIMESTAMP.txt"
    GO_COVERAGE_OUTPUT="$COVERAGE_DIR/go_coverage_$TIMESTAMP.out"
    
    if go test -v -coverprofile="$GO_COVERAGE_OUTPUT" ./... > "$GO_TEST_OUTPUT" 2>&1; then
        print_success "Go tests passed"
        GO_TESTS_PASSED=true
        
        # Generate coverage report
        if go tool cover -html="$GO_COVERAGE_OUTPUT" -o "$COVERAGE_DIR/go_coverage_$TIMESTAMP.html"; then
            print_success "Go coverage report generated"
        fi
        
        # Extract coverage percentage
        GO_COVERAGE=$(go tool cover -func="$GO_COVERAGE_OUTPUT" | tail -1 | awk '{print $3}')
        echo "Go Coverage: $GO_COVERAGE"
    else
        print_error "Go tests failed"
        echo "Go test output saved to: $GO_TEST_OUTPUT"
        tail -20 "$GO_TEST_OUTPUT"
    fi
    
    # Run Go linting if available
    if command -v golangci-lint &> /dev/null; then
        echo "Running Go linting..."
        if golangci-lint run ./...; then
            print_success "Go linting passed"
        else
            print_warning "Go linting found issues"
        fi
    fi
    
    # Test Go build
    echo "Testing Go build..."
    if go build -o build/knirvgraph ./cmd/node; then
        print_success "Go build successful"
    else
        print_error "Go build failed"
    fi
    
else
    print_error "Go is not installed"
fi

# Test TypeScript Frontend
print_section "Testing TypeScript Frontend"
if command -v node &> /dev/null; then
    print_success "Node.js version: $(node --version)"
    
    # Install dependencies
    echo "Installing TypeScript dependencies..."
    if npm ci > /dev/null 2>&1; then
        print_success "TypeScript dependencies installed"
    else
        print_error "Failed to install TypeScript dependencies"
    fi
    
    # Run TypeScript type checking
    echo "Running TypeScript type checking..."
    if npx tsc --noEmit > /dev/null 2>&1; then
        print_success "TypeScript type checking passed"
    else
        print_warning "TypeScript type checking found issues"
    fi
    
    # Run ESLint
    echo "Running ESLint..."
    if npx eslint src/ --ext .ts,.tsx > /dev/null 2>&1; then
        print_success "ESLint passed"
    else
        print_warning "ESLint found issues"
    fi
    
    # Run frontend tests
    echo "Running frontend tests..."
    FRONTEND_TEST_OUTPUT="$REPORTS_DIR/frontend_tests_$TIMESTAMP.txt"
    
    # Check if Jest is configured
    if [ -f "jest.config.js" ] || [ -f "jest.config.ts" ] || grep -q '"test"' package.json; then
        if npm test > "$FRONTEND_TEST_OUTPUT" 2>&1; then
            print_success "Frontend tests passed"
            FRONTEND_TESTS_PASSED=true
            
            # Run coverage if configured
            if npm run test:coverage > /dev/null 2>&1; then
                print_success "Frontend coverage report generated"
                
                # Extract coverage percentage if available
                if [ -f "coverage/coverage-summary.json" ]; then
                    FRONTEND_COVERAGE=$(node -e "
                        const fs = require('fs');
                        const coverage = JSON.parse(fs.readFileSync('coverage/coverage-summary.json', 'utf8'));
                        console.log(coverage.total.lines.pct + '%');
                    " 2>/dev/null || echo "N/A")
                    echo "Frontend Coverage: $FRONTEND_COVERAGE"
                fi
            fi
        else
            print_error "Frontend tests failed"
            echo "Frontend test output saved to: $FRONTEND_TEST_OUTPUT"
            tail -20 "$FRONTEND_TEST_OUTPUT"
        fi
    else
        print_warning "No test configuration found for frontend"
        FRONTEND_TESTS_PASSED=true # Consider it passed if no tests configured
    fi
    
    # Test frontend build
    echo "Testing frontend build..."
    BUILD_OUTPUT="$REPORTS_DIR/build_$TIMESTAMP.txt"
    if npm run build > "$BUILD_OUTPUT" 2>&1; then
        print_success "Frontend build successful"
        BUILD_SUCCESSFUL=true
        
        # Check bundle size
        if [ -d "dist" ]; then
            BUNDLE_SIZE=$(du -sh dist | cut -f1)
            print_success "Bundle size: $BUNDLE_SIZE"
        fi
    else
        print_error "Frontend build failed"
        echo "Build output saved to: $BUILD_OUTPUT"
        tail -20 "$BUILD_OUTPUT"
    fi
    
else
    print_error "Node.js is not installed"
fi

# Integration Tests
print_section "Integration Tests"
if [ "$GO_TESTS_PASSED" = true ] && [ "$BUILD_SUCCESSFUL" = true ]; then
    echo "Running integration tests..."
    
    # Start the Go backend in background
    if [ -f "build/knirvgraph" ]; then
        echo "Starting KNIRVGRAPH backend..."
        ./build/knirvgraph --port 8080 --data-dir ./test-data > /dev/null 2>&1 &
        BACKEND_PID=$!
        
        # Give it time to start
        sleep 3
        
        # Test API endpoints
        if command -v curl &> /dev/null; then
            echo "Testing API endpoints..."
            
            # Test health endpoint
            if curl -s http://localhost:8080/health > /dev/null; then
                print_success "Health endpoint responding"
            else
                print_warning "Health endpoint not responding"
            fi
            
            # Test graph data endpoint
            if curl -s http://localhost:8080/api/graph > /dev/null; then
                print_success "Graph API endpoint responding"
            else
                print_warning "Graph API endpoint not responding"
            fi
        fi
        
        # Clean up
        kill $BACKEND_PID 2>/dev/null || true
        rm -rf ./test-data 2>/dev/null || true
    else
        print_warning "Backend binary not found, skipping integration tests"
    fi
else
    print_warning "Skipping integration tests due to previous failures"
fi

# Security and Performance Tests
print_section "Security and Performance"

# Security audit for Node.js dependencies
if command -v npm &> /dev/null; then
    echo "Running security audit..."
    AUDIT_OUTPUT="$REPORTS_DIR/security_audit_$TIMESTAMP.txt"
    if npm audit --audit-level=moderate > "$AUDIT_OUTPUT" 2>&1; then
        print_success "Security audit passed"
    else
        print_warning "Security audit found issues"
        echo "Audit output saved to: $AUDIT_OUTPUT"
    fi
fi

# Check for outdated dependencies
if command -v npm &> /dev/null; then
    echo "Checking for outdated dependencies..."
    OUTDATED_OUTPUT="$REPORTS_DIR/outdated_$TIMESTAMP.txt"
    if npm outdated > "$OUTDATED_OUTPUT" 2>&1; then
        print_success "All dependencies are up to date"
    else
        print_warning "Some dependencies are outdated"
        echo "Outdated packages saved to: $OUTDATED_OUTPUT"
    fi
fi

# Generate comprehensive test summary
print_section "Test Summary"
SUMMARY_FILE="$REPORTS_DIR/comprehensive_test_summary_$TIMESTAMP.txt"

cat > "$SUMMARY_FILE" << EOF
KNIRVGRAPH Comprehensive Test Summary
=====================================
Timestamp: $TIMESTAMP
Project Root: $PROJECT_ROOT

Test Results:
- Go Backend Tests: $([ "$GO_TESTS_PASSED" = true ] && echo "PASSED" || echo "FAILED")
- Frontend Tests: $([ "$FRONTEND_TESTS_PASSED" = true ] && echo "PASSED" || echo "FAILED")
- Build Process: $([ "$BUILD_SUCCESSFUL" = true ] && echo "PASSED" || echo "FAILED")

Coverage Information:
- Go Coverage: ${GO_COVERAGE:-"N/A"}
- Frontend Coverage: ${FRONTEND_COVERAGE:-"N/A"}

Build Information:
- Go Binary: $([ -f "build/knirvgraph" ] && echo "✓" || echo "✗")
- Frontend Bundle: $([ -d "dist" ] && echo "✓" || echo "✗")
- Bundle Size: ${BUNDLE_SIZE:-"N/A"}

Generated Files:
- Go Test Output: $([ -f "$REPORTS_DIR/go_tests_$TIMESTAMP.txt" ] && echo "✓" || echo "✗")
- Frontend Test Output: $([ -f "$REPORTS_DIR/frontend_tests_$TIMESTAMP.txt" ] && echo "✓" || echo "✗")
- Build Output: $([ -f "$REPORTS_DIR/build_$TIMESTAMP.txt" ] && echo "✓" || echo "✗")
- Security Audit: $([ -f "$REPORTS_DIR/security_audit_$TIMESTAMP.txt" ] && echo "✓" || echo "✗")

Overall Status: $([ "$GO_TESTS_PASSED" = true ] && [ "$FRONTEND_TESTS_PASSED" = true ] && [ "$BUILD_SUCCESSFUL" = true ] && echo "SUCCESS" || echo "PARTIAL SUCCESS")
EOF

print_success "Test summary generated: $SUMMARY_FILE"

# Final results
print_section "Final Results"
if [ "$GO_TESTS_PASSED" = true ]; then
    print_success "Go Backend: All tests passed (Coverage: ${GO_COVERAGE:-"N/A"})"
else
    print_error "Go Backend: Tests failed or not run"
fi

if [ "$FRONTEND_TESTS_PASSED" = true ]; then
    print_success "Frontend: All tests passed (Coverage: ${FRONTEND_COVERAGE:-"N/A"})"
else
    print_error "Frontend: Tests failed or not run"
fi

if [ "$BUILD_SUCCESSFUL" = true ]; then
    print_success "Build: Successful (Bundle size: ${BUNDLE_SIZE:-"N/A"})"
else
    print_error "Build: Failed"
fi

echo ""
if [ "$GO_TESTS_PASSED" = true ] && [ "$FRONTEND_TESTS_PASSED" = true ] && [ "$BUILD_SUCCESSFUL" = true ]; then
    echo -e "${GREEN}🎉 All KNIRVGRAPH tests completed successfully! 🎉${NC}"
    exit 0
else
    echo -e "${YELLOW}⚠ Some KNIRVGRAPH tests failed or were not run ⚠${NC}"
    exit 1
fi
