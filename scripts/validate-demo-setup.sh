#!/bin/bash

# KNIRV Network Demo Setup Validation Script
# This script validates that all prerequisites are met for running the full demo

# Note: Don't use 'set -e' here as we want to continue checking even if some checks fail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Counters
CHECKS_PASSED=0
CHECKS_FAILED=0
WARNINGS=0

print_header() {
    echo ""
    echo -e "${BLUE}================================================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}================================================================${NC}"
    echo ""
}

print_check() {
    echo -n "Checking $1... "
}

print_pass() {
    echo -e "${GREEN}✓ PASS${NC}"
    ((CHECKS_PASSED++))
}

print_fail() {
    echo -e "${RED}✗ FAIL${NC}"
    echo -e "${RED}  $1${NC}"
    ((CHECKS_FAILED++))
}

print_warning() {
    echo -e "${YELLOW}⚠ WARNING${NC}"
    echo -e "${YELLOW}  $1${NC}"
    ((WARNINGS++))
}

print_info() {
    echo -e "${BLUE}ℹ INFO${NC} $1"
}

# Project root
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

print_header "KNIRV NETWORK DEMO SETUP VALIDATION"

echo "Validating demo prerequisites and environment setup..."
echo "Project Root: $PROJECT_ROOT"
echo ""

# Check 1: Required files
print_check "demo script existence"
if [ -f "$PROJECT_ROOT/run-full-demo.sh" ]; then
    if [ -x "$PROJECT_ROOT/run-full-demo.sh" ]; then
        print_pass
    else
        print_fail "Demo script exists but is not executable. Run: chmod +x run-full-demo.sh"
    fi
else
    print_fail "Demo script not found at $PROJECT_ROOT/run-full-demo.sh"
fi

print_check "Makefile existence"
if [ -f "$PROJECT_ROOT/Makefile" ]; then
    print_pass
else
    print_fail "Makefile not found in project root"
fi

print_check "DEMO_README.md existence"
if [ -f "$PROJECT_ROOT/DEMO_README.md" ]; then
    print_pass
else
    print_warning "DEMO_README.md not found - documentation may be incomplete"
fi

# Check 2: Required directories
print_check "KNIRVTESTNET directory"
if [ -d "$PROJECT_ROOT/packages/KNIRVTESTNET" ]; then
    print_pass
else
    print_fail "KNIRVTESTNET directory not found"
fi

print_check "KNIRVWALLET directory"
if [ -d "$PROJECT_ROOT/packages/KNIRVWALLET" ]; then
    print_pass
else
    print_fail "KNIRVWALLET directory not found"
fi

# Check 3: System tools
print_check "make command"
if command -v make >/dev/null 2>&1; then
    print_pass
else
    print_fail "make command not found. Install build-essential or equivalent"
fi

print_check "node command"
if command -v node >/dev/null 2>&1; then
    NODE_VERSION=$(node --version)
    print_pass
    print_info "Node.js version: $NODE_VERSION"
else
    print_fail "node command not found. Install Node.js"
fi

print_check "npm command"
if command -v npm >/dev/null 2>&1; then
    NPM_VERSION=$(npm --version)
    print_pass
    print_info "npm version: $NPM_VERSION"
else
    print_fail "npm command not found. Install npm"
fi

print_check "curl command"
if command -v curl >/dev/null 2>&1; then
    print_pass
else
    print_fail "curl command not found. Install curl"
fi

print_check "git command"
if command -v git >/dev/null 2>&1; then
    print_pass
else
    print_warning "git command not found. Some features may not work"
fi

# Check 4: KNIRVTESTNET setup
if [ -d "$PROJECT_ROOT/packages/KNIRVTESTNET" ]; then
    print_check "KNIRVTESTNET package.json"
    if [ -f "$PROJECT_ROOT/packages/KNIRVTESTNET/package.json" ]; then
        print_pass
    else
        print_fail "KNIRVTESTNET/package.json not found"
    fi
    
    print_check "KNIRVTESTNET node_modules"
    if [ -d "$PROJECT_ROOT/packages/KNIRVTESTNET/node_modules" ]; then
        print_pass
    else
        print_warning "KNIRVTESTNET dependencies not installed. Run: cd packages/KNIRVTESTNET && npm install"
    fi
    
    print_check "KNIRVTESTNET scripts directory"
    if [ -d "$PROJECT_ROOT/packages/KNIRVTESTNET/scripts" ]; then
        print_pass
    else
        print_fail "KNIRVTESTNET/scripts directory not found"
    fi
    
    print_check "KNIRVTESTNET binaries"
    if [ -d "$PROJECT_ROOT/packages/KNIRVTESTNET/bin" ] && [ "$(ls -A $PROJECT_ROOT/packages/KNIRVTESTNET/bin 2>/dev/null)" ]; then
        print_pass
        print_info "Found $(ls $PROJECT_ROOT/packages/KNIRVTESTNET/bin | wc -l) binaries"
    else
        print_warning "KNIRVTESTNET binaries not found or empty. Some services may not start"
    fi
fi

# Check 5: KNIRVWALLET setup
if [ -d "$PROJECT_ROOT/packages/KNIRVWALLET" ]; then
    print_check "KNIRVWALLET package.json"
    if [ -f "$PROJECT_ROOT/packages/KNIRVWALLET/package.json" ]; then
        print_pass
    else
        print_fail "KNIRVWALLET/package.json not found"
    fi
    
    print_check "KNIRVWALLET node_modules"
    if [ -d "$PROJECT_ROOT/packages/KNIRVWALLET/node_modules" ]; then
        print_pass
    else
        print_warning "KNIRVWALLET dependencies not installed. Run: cd packages/KNIRVWALLET && npm install"
    fi
fi

# Check 6: Port availability
print_check "port availability"
REQUIRED_PORTS=(3000 1317 8082 8084 8086 8088 8090)
PORTS_IN_USE=()

for port in "${REQUIRED_PORTS[@]}"; do
    if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
        PORTS_IN_USE+=($port)
    fi
done

if [ ${#PORTS_IN_USE[@]} -eq 0 ]; then
    print_pass
else
    print_warning "Ports in use: ${PORTS_IN_USE[*]}. Demo may conflict with existing services"
fi

# Check 7: System resources
print_check "available memory"
if command -v free >/dev/null 2>&1; then
    AVAILABLE_MEM=$(free -m | awk 'NR==2{printf "%.0f", $7}')
    if [ "$AVAILABLE_MEM" -gt 4096 ]; then
        print_pass
        print_info "Available memory: ${AVAILABLE_MEM}MB"
    else
        print_warning "Low available memory: ${AVAILABLE_MEM}MB. Demo may be slow"
    fi
elif command -v vm_stat >/dev/null 2>&1; then
    # macOS
    print_pass
    print_info "Memory check completed (macOS)"
else
    print_warning "Cannot check available memory"
fi

print_check "disk space"
AVAILABLE_SPACE=$(df . | awk 'NR==2 {print $4}')
if [ "$AVAILABLE_SPACE" -gt 10485760 ]; then  # 10GB in KB
    print_pass
else
    print_warning "Low disk space. Demo may fail if space is insufficient"
fi

# Check 8: Makefile targets
print_check "Makefile demo targets"
if make -n demo >/dev/null 2>&1; then
    print_pass
else
    print_fail "Makefile demo targets not properly configured"
fi

# Summary
print_header "VALIDATION SUMMARY"

echo "Validation Results:"
echo -e "  ${GREEN}Checks Passed: $CHECKS_PASSED${NC}"
echo -e "  ${RED}Checks Failed: $CHECKS_FAILED${NC}"
echo -e "  ${YELLOW}Warnings: $WARNINGS${NC}"
echo ""

if [ $CHECKS_FAILED -eq 0 ]; then
    echo -e "${GREEN}✅ Demo environment is ready!${NC}"
    echo ""
    echo "You can now run the demo using:"
    echo "  make demo                 # Full demo"
    echo "  make demo-quick          # Quick demo"
    echo "  make demo-interactive    # Interactive demo"
    echo "  ./run-full-demo.sh --help # See all options"
    echo ""
    
    if [ $WARNINGS -gt 0 ]; then
        echo -e "${YELLOW}Note: There are $WARNINGS warnings that may affect demo performance.${NC}"
    fi
    
    exit 0
else
    echo -e "${RED}❌ Demo environment has issues that must be resolved.${NC}"
    echo ""
    echo "Please fix the failed checks above before running the demo."
    echo ""
    echo "Common fixes:"
    echo "  chmod +x run-full-demo.sh"
    echo "  cd packages/KNIRVTESTNET && npm install"
    echo "  cd packages/KNIRVWALLET && npm install"
    echo "  sudo apt-get install build-essential curl"
    echo ""
    exit 1
fi
