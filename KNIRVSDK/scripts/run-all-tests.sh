#!/bin/bash

# KNIRVSDK Comprehensive Test Runner
# This script runs tests for all SDK languages (Go, Python, TypeScript)

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

echo -e "${BLUE}KNIRVSDK Multi-Language Test Suite${NC}"
echo -e "${BLUE}===================================${NC}"
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
PYTHON_TESTS_PASSED=false
TYPESCRIPT_TESTS_PASSED=false

# Test Go SDK
print_section "Testing Go SDK"
if [ -d "go" ]; then
    cd go
    
    # Check if Go is installed
    if command -v go &> /dev/null; then
        print_success "Go version: $(go version)"
        
        # Install dependencies
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
        
    else
        print_error "Go is not installed"
    fi
    
    cd ..
else
    print_warning "Go SDK directory not found"
fi

# Test Python SDK
print_section "Testing Python SDK"
if [ -d "py" ]; then
    cd py
    
    # Check if Python is installed
    if command -v python3 &> /dev/null; then
        print_success "Python version: $(python3 --version)"
        
        # Create virtual environment if it doesn't exist
        if [ ! -d "venv" ]; then
            echo "Creating Python virtual environment..."
            python3 -m venv venv
        fi
        
        # Activate virtual environment
        source venv/bin/activate
        
        # Install dependencies
        echo "Installing Python dependencies..."
        if pip install -r requirements.txt > /dev/null 2>&1; then
            print_success "Python dependencies installed"
        else
            print_warning "Some Python dependencies may have failed to install"
        fi
        
        # Install test dependencies
        pip install pytest pytest-cov pytest-mock responses > /dev/null 2>&1
        
        # Run Python tests
        echo "Running Python tests..."
        PYTHON_TEST_OUTPUT="$REPORTS_DIR/python_tests_$TIMESTAMP.txt"
        PYTHON_COVERAGE_OUTPUT="$COVERAGE_DIR/python_coverage_$TIMESTAMP.xml"
        
        if pytest --verbose --cov=gateway --cov-report=xml:"$PYTHON_COVERAGE_OUTPUT" --cov-report=html:"$COVERAGE_DIR/python_coverage_$TIMESTAMP" > "$PYTHON_TEST_OUTPUT" 2>&1; then
            print_success "Python tests passed"
            PYTHON_TESTS_PASSED=true
            
            # Extract coverage percentage
            if [ -f "$PYTHON_COVERAGE_OUTPUT" ]; then
                PYTHON_COVERAGE=$(python3 -c "
import xml.etree.ElementTree as ET
tree = ET.parse('$PYTHON_COVERAGE_OUTPUT')
root = tree.getroot()
coverage = root.get('line-rate')
print(f'{float(coverage)*100:.1f}%')
" 2>/dev/null || echo "N/A")
                echo "Python Coverage: $PYTHON_COVERAGE"
            fi
        else
            print_error "Python tests failed"
            echo "Python test output saved to: $PYTHON_TEST_OUTPUT"
            tail -20 "$PYTHON_TEST_OUTPUT"
        fi
        
        # Run Python linting if available
        if command -v flake8 &> /dev/null; then
            echo "Running Python linting..."
            if flake8 gateway/; then
                print_success "Python linting passed"
            else
                print_warning "Python linting found issues"
            fi
        fi
        
        # Deactivate virtual environment
        deactivate
        
    else
        print_error "Python3 is not installed"
    fi
    
    cd ..
else
    print_warning "Python SDK directory not found"
fi

# Test TypeScript SDK
print_section "Testing TypeScript SDK"
if [ -d "ts/gateway" ]; then
    cd ts/gateway
    
    # Check if Node.js is installed
    if command -v node &> /dev/null; then
        print_success "Node.js version: $(node --version)"
        
        # Install dependencies
        echo "Installing TypeScript dependencies..."
        if npm ci > /dev/null 2>&1; then
            print_success "TypeScript dependencies installed"
        else
            print_error "Failed to install TypeScript dependencies"
        fi
        
        # Run TypeScript tests
        echo "Running TypeScript tests..."
        TS_TEST_OUTPUT="$REPORTS_DIR/typescript_tests_$TIMESTAMP.txt"
        
        if npm test > "$TS_TEST_OUTPUT" 2>&1; then
            print_success "TypeScript tests passed"
            TYPESCRIPT_TESTS_PASSED=true
            
            # Run coverage if configured
            if npm run test:coverage > /dev/null 2>&1; then
                print_success "TypeScript coverage report generated"
                
                # Extract coverage percentage if available
                if [ -f "coverage/coverage-summary.json" ]; then
                    TS_COVERAGE=$(node -e "
                        const fs = require('fs');
                        const coverage = JSON.parse(fs.readFileSync('coverage/coverage-summary.json', 'utf8'));
                        console.log(coverage.total.lines.pct + '%');
                    " 2>/dev/null || echo "N/A")
                    echo "TypeScript Coverage: $TS_COVERAGE"
                fi
            fi
        else
            print_error "TypeScript tests failed"
            echo "TypeScript test output saved to: $TS_TEST_OUTPUT"
            tail -20 "$TS_TEST_OUTPUT"
        fi
        
        # Run TypeScript linting
        echo "Running TypeScript linting..."
        if npm run lint > /dev/null 2>&1; then
            print_success "TypeScript linting passed"
        else
            print_warning "TypeScript linting found issues"
        fi
        
        # Type checking
        echo "Running TypeScript type checking..."
        if npx tsc --noEmit > /dev/null 2>&1; then
            print_success "TypeScript type checking passed"
        else
            print_warning "TypeScript type checking found issues"
        fi
        
    else
        print_error "Node.js is not installed"
    fi
    
    cd ../..
else
    print_warning "TypeScript SDK directory not found"
fi

# Generate comprehensive test summary
print_section "Test Summary"
SUMMARY_FILE="$REPORTS_DIR/sdk_test_summary_$TIMESTAMP.txt"

cat > "$SUMMARY_FILE" << EOF
KNIRVSDK Multi-Language Test Summary
====================================
Timestamp: $TIMESTAMP
Project Root: $PROJECT_ROOT

Test Results:
- Go SDK: $([ "$GO_TESTS_PASSED" = true ] && echo "PASSED" || echo "FAILED")
- Python SDK: $([ "$PYTHON_TESTS_PASSED" = true ] && echo "PASSED" || echo "FAILED")
- TypeScript SDK: $([ "$TYPESCRIPT_TESTS_PASSED" = true ] && echo "PASSED" || echo "FAILED")

Coverage Information:
- Go Coverage: ${GO_COVERAGE:-"N/A"}
- Python Coverage: ${PYTHON_COVERAGE:-"N/A"}
- TypeScript Coverage: ${TS_COVERAGE:-"N/A"}

Generated Files:
- Go Test Output: $([ -f "$REPORTS_DIR/go_tests_$TIMESTAMP.txt" ] && echo "✓" || echo "✗")
- Python Test Output: $([ -f "$REPORTS_DIR/python_tests_$TIMESTAMP.txt" ] && echo "✓" || echo "✗")
- TypeScript Test Output: $([ -f "$REPORTS_DIR/typescript_tests_$TIMESTAMP.txt" ] && echo "✓" || echo "✗")

Overall Status: $([ "$GO_TESTS_PASSED" = true ] && [ "$PYTHON_TESTS_PASSED" = true ] && [ "$TYPESCRIPT_TESTS_PASSED" = true ] && echo "SUCCESS" || echo "PARTIAL SUCCESS")
EOF

print_success "Test summary generated: $SUMMARY_FILE"

# Final results
print_section "Final Results"
if [ "$GO_TESTS_PASSED" = true ]; then
    print_success "Go SDK: All tests passed (Coverage: ${GO_COVERAGE:-"N/A"})"
else
    print_error "Go SDK: Tests failed or not run"
fi

if [ "$PYTHON_TESTS_PASSED" = true ]; then
    print_success "Python SDK: All tests passed (Coverage: ${PYTHON_COVERAGE:-"N/A"})"
else
    print_error "Python SDK: Tests failed or not run"
fi

if [ "$TYPESCRIPT_TESTS_PASSED" = true ]; then
    print_success "TypeScript SDK: All tests passed (Coverage: ${TS_COVERAGE:-"N/A"})"
else
    print_error "TypeScript SDK: Tests failed or not run"
fi

echo ""
if [ "$GO_TESTS_PASSED" = true ] && [ "$PYTHON_TESTS_PASSED" = true ] && [ "$TYPESCRIPT_TESTS_PASSED" = true ]; then
    echo -e "${GREEN}🎉 All SDK tests completed successfully! 🎉${NC}"
    exit 0
else
    echo -e "${YELLOW}⚠ Some SDK tests failed or were not run ⚠${NC}"
    exit 1
fi
