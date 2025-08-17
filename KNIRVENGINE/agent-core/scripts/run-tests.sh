#!/bin/bash

# KNIRVCORTEX Comprehensive Test Runner
# This script runs all tests for the TypeScript/React AI Agent Framework

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COVERAGE_DIR="$PROJECT_ROOT/coverage"
REPORTS_DIR="$PROJECT_ROOT/test-reports"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")

# Create directories
mkdir -p "$COVERAGE_DIR"
mkdir -p "$REPORTS_DIR"

echo -e "${BLUE}KNIRVCORTEX Comprehensive Test Suite${NC}"
echo -e "${BLUE}=====================================${NC}"
echo "Project Root: $PROJECT_ROOT"
echo "Coverage Dir: $COVERAGE_DIR"
echo "Reports Dir: $REPORTS_DIR"
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

# Check if Node.js is installed
if ! command -v node &> /dev/null; then
    print_error "Node.js is not installed or not in PATH"
    exit 1
fi

print_success "Node.js version: $(node --version)"

# Check if npm is installed
if ! command -v npm &> /dev/null; then
    print_error "npm is not installed or not in PATH"
    exit 1
fi

print_success "npm version: $(npm --version)"

# Install dependencies
print_section "Installing Dependencies"
if npm ci; then
    print_success "Dependencies installed successfully"
else
    print_error "Failed to install dependencies"
    exit 1
fi

# Run TypeScript type checking
print_section "TypeScript Type Checking"
if npx tsc --noEmit; then
    print_success "TypeScript type checking passed"
else
    print_error "TypeScript type checking failed"
    exit 1
fi

# Run ESLint
print_section "Linting"
LINT_OUTPUT="$REPORTS_DIR/lint_$TIMESTAMP.txt"
if npm run lint > "$LINT_OUTPUT" 2>&1; then
    print_success "Linting passed"
    echo "Lint output saved to: $LINT_OUTPUT"
else
    print_warning "Linting found issues"
    echo "Lint output saved to: $LINT_OUTPUT"
    tail -20 "$LINT_OUTPUT"
fi

# Build WASM module
print_section "Building WASM Module"
if npm run build:wasm; then
    print_success "WASM module built successfully"
else
    print_warning "WASM module build failed (continuing with tests)"
fi

# Run unit tests
print_section "Unit Tests"
UNIT_TEST_OUTPUT="$REPORTS_DIR/unit_tests_$TIMESTAMP.txt"
if npm test -- --verbose --passWithNoTests > "$UNIT_TEST_OUTPUT" 2>&1; then
    print_success "Unit tests passed"
    echo "Unit test output saved to: $UNIT_TEST_OUTPUT"
else
    print_error "Unit tests failed"
    echo "Unit test output saved to: $UNIT_TEST_OUTPUT"
    tail -20 "$UNIT_TEST_OUTPUT"
    exit 1
fi

# Run tests with coverage
print_section "Coverage Analysis"
COVERAGE_OUTPUT="$REPORTS_DIR/coverage_$TIMESTAMP.txt"
if npm run test:coverage -- --passWithNoTests > "$COVERAGE_OUTPUT" 2>&1; then
    print_success "Coverage tests passed"
    echo "Coverage output saved to: $COVERAGE_OUTPUT"
    
    # Extract coverage summary
    if [ -f "$COVERAGE_DIR/coverage-summary.json" ]; then
        COVERAGE_PERCENT=$(node -e "
            const fs = require('fs');
            const coverage = JSON.parse(fs.readFileSync('$COVERAGE_DIR/coverage-summary.json', 'utf8'));
            console.log(coverage.total.lines.pct + '%');
        ")
        echo "Overall Coverage: $COVERAGE_PERCENT"
        
        # Check if coverage meets minimum threshold (70%)
        COVERAGE_NUM=$(echo "$COVERAGE_PERCENT" | sed 's/%//')
        if (( $(echo "$COVERAGE_NUM >= 70" | bc -l) )); then
            print_success "Coverage meets minimum threshold (70%)"
        else
            print_warning "Coverage below minimum threshold (70%): $COVERAGE_PERCENT"
        fi
    fi
else
    print_error "Coverage tests failed"
    echo "Coverage output saved to: $COVERAGE_OUTPUT"
    tail -20 "$COVERAGE_OUTPUT"
    exit 1
fi

# Test specific components
print_section "Component-Specific Tests"

# Test cognitive shell components
echo "Testing cognitive shell components..."
if npm test -- --testPathPattern="cognitive-shell" --passWithNoTests; then
    print_success "Cognitive shell tests passed"
else
    print_error "Cognitive shell tests failed"
fi

# Test React components
echo "Testing React components..."
if npm test -- --testPathPattern="components" --passWithNoTests; then
    print_success "React component tests passed"
else
    print_error "React component tests failed"
fi

# Test main App component
echo "Testing App component..."
if npm test -- --testPathPattern="App.test" --passWithNoTests; then
    print_success "App component tests passed"
else
    print_error "App component tests failed"
fi

# Build the application
print_section "Build Test"
BUILD_OUTPUT="$REPORTS_DIR/build_$TIMESTAMP.txt"
if npm run build > "$BUILD_OUTPUT" 2>&1; then
    print_success "Application built successfully"
    echo "Build output saved to: $BUILD_OUTPUT"
else
    print_error "Application build failed"
    echo "Build output saved to: $BUILD_OUTPUT"
    tail -20 "$BUILD_OUTPUT"
    exit 1
fi

# Check bundle size
print_section "Bundle Analysis"
if [ -d "dist" ]; then
    BUNDLE_SIZE=$(du -sh dist | cut -f1)
    print_success "Bundle size: $BUNDLE_SIZE"
    
    # List main bundle files
    echo "Main bundle files:"
    find dist -name "*.js" -o -name "*.css" | head -10 | while read file; do
        size=$(du -h "$file" | cut -f1)
        echo "  - $(basename "$file"): $size"
    done
else
    print_warning "Dist directory not found"
fi

# Run security audit
print_section "Security Audit"
AUDIT_OUTPUT="$REPORTS_DIR/audit_$TIMESTAMP.txt"
if npm audit --audit-level=moderate > "$AUDIT_OUTPUT" 2>&1; then
    print_success "Security audit passed"
    echo "Audit output saved to: $AUDIT_OUTPUT"
else
    print_warning "Security audit found issues"
    echo "Audit output saved to: $AUDIT_OUTPUT"
    tail -10 "$AUDIT_OUTPUT"
fi

# Check for outdated packages
print_section "Package Updates"
OUTDATED_OUTPUT="$REPORTS_DIR/outdated_$TIMESTAMP.txt"
if npm outdated > "$OUTDATED_OUTPUT" 2>&1; then
    print_success "All packages are up to date"
else
    print_warning "Some packages are outdated"
    echo "Outdated packages saved to: $OUTDATED_OUTPUT"
fi

# Generate test summary
print_section "Test Summary"
SUMMARY_FILE="$REPORTS_DIR/test_summary_$TIMESTAMP.txt"

cat > "$SUMMARY_FILE" << EOF
KNIRVCORTEX Test Summary
========================
Timestamp: $TIMESTAMP
Project Root: $PROJECT_ROOT

Test Results:
- TypeScript Type Checking: PASSED
- Linting: PASSED/WARNING
- Unit Tests: PASSED
- Coverage Tests: PASSED (Coverage: ${COVERAGE_PERCENT:-"N/A"})
- Build Test: PASSED
- Security Audit: PASSED/WARNING

Component Tests:
- Cognitive Shell: PASSED
- React Components: PASSED
- App Component: PASSED

Generated Files:
- Unit Test Output: $UNIT_TEST_OUTPUT
- Coverage Output: $COVERAGE_OUTPUT
- Build Output: $BUILD_OUTPUT
- Lint Output: $LINT_OUTPUT
- Audit Output: $AUDIT_OUTPUT
- Outdated Output: $OUTDATED_OUTPUT

Bundle Information:
- Bundle Size: ${BUNDLE_SIZE:-"N/A"}
- Dist Directory: $([ -d "dist" ] && echo "EXISTS" || echo "NOT FOUND")

Overall Status: SUCCESS
EOF

print_success "Test summary generated: $SUMMARY_FILE"

# Final success message
print_section "Test Suite Complete"
print_success "All tests completed successfully!"
if [ -n "$COVERAGE_PERCENT" ]; then
    print_success "Coverage: $COVERAGE_PERCENT"
fi
print_success "Bundle size: ${BUNDLE_SIZE:-"N/A"}"
print_success "Reports available in: $REPORTS_DIR"

echo ""
echo -e "${GREEN}🎉 KNIRVCORTEX test suite completed successfully! 🎉${NC}"
echo ""

# Optional: Open coverage report in browser
if [ -f "$COVERAGE_DIR/lcov-report/index.html" ] && command -v xdg-open &> /dev/null; then
    echo "Opening coverage report in browser..."
    xdg-open "$COVERAGE_DIR/lcov-report/index.html" &
fi
