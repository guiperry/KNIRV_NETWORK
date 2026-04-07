#!/bin/bash

# verify-network-model.sh - Pre-deployment Formal Verification Gate
# Runs formal verification before starting testnet services

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TESTNET_ROOT="$(dirname "$SCRIPT_DIR")"
# MODP_DIR: honour env var, else resolve to KNIRV_NETWORK/modp
# TESTNET_ROOT = .../packages/KNIRVTESTNET → two dirname calls reach repo root
REPO_ROOT="$(dirname "$(dirname "$TESTNET_ROOT")")"
MODP_DIR="${MODP_DIR:-$REPO_ROOT/modp}"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
CYAN='\033[0;36m'
NC='\033[0m'

print_status() {
    echo -e "${BLUE}[VERIFY]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[FAIL]${NC} $1"
}

print_header() {
    echo -e "${CYAN}=== $1 ===${NC}"
}

# Configuration
VERIFICATION_LEVEL="${VERIFICATION_LEVEL:-standard}" # minimal, standard, thorough
MAX_DURATION="${MAX_DURATION:-300}" # 5 minutes max

print_header "KNIRV Network Formal Verification Gate"
print_status "Verification Level: $VERIFICATION_LEVEL"
print_status "Max Duration: ${MAX_DURATION}s"
echo ""

# Check if modp directory exists
if [ ! -d "$MODP_DIR" ]; then
    print_error "modp directory not found at $MODP_DIR"
    exit 1
fi

cd "$MODP_DIR"

# Function to run verification with timeout
run_verification() {
    local test_name="$1"
    local timeout_duration="$2"
    
    print_status "Running: $test_name"
    
    # Use timeout command to limit execution time
    if timeout "$timeout_duration" bash scripts/run-tests.sh "$test_name" > verification-temp.log 2>&1; then
        print_success "$test_name - PASSED"
        return 0
    else
        local exit_code=$?
        if [ $exit_code -eq 124 ]; then
            print_error "$test_name - TIMEOUT (${timeout_duration}s)"
        else
            print_error "$test_name - FAILED"
        fi
        
        # Show last few lines of output for debugging
        if [ -f "verification-temp.log" ]; then
            echo "Last 5 lines of output:"
            tail -5 verification-temp.log | sed 's/^/  /'
        fi
        return 1
    fi
}

# Function to check P compiler availability
check_p_compiler() {
    print_header "P Compiler Check"
    
    local p_dll="$HOME/.p-lang/Bld/Drops/Release/Binaries/net8.0/p.dll"
    if [ -f "$p_dll" ]; then
        print_success "P compiler found"
        return 0
    else
        print_warning "P compiler not found - running in syntax validation mode"
        return 1
    fi
}

# Function to validate P syntax
validate_syntax() {
    print_header "P Syntax Validation"
    
    local errors=0
    for pfile in $(find . -name "*.p" -type f | head -10); do
        if ! grep -q "^machine\|^spec\|^type\|^event\|^enum" "$pfile" 2>/dev/null; then
            if [ "$(wc -l < "$pfile")" -gt 5 ]; then
                print_warning "Syntax check: $pfile may not be valid P syntax"
                errors=$((errors + 1))
            fi
        fi
    done
    
    if [ $errors -eq 0 ]; then
        print_success "P syntax validation passed"
        return 0
    else
        print_warning "P syntax validation found $errors potential issues"
        return 1
    fi
}

# Function to run minimal verification tests
run_minimal_tests() {
    print_header "Minimal Verification Tests"
    
    local tests_passed=0
    local tests_total=2
    
    # Test 1: P model compilation
    if check_p_compiler; then
        if run_verification "compile" 60; then
            tests_passed=$((tests_passed + 1))
        fi
    else
        if validate_syntax; then
            tests_passed=$((tests_passed + 1))
        fi
    fi
    
    # Test 2: Basic component test
    if run_verification "test TokenTransferConsistency" 60; then
        tests_passed=$((tests_passed + 1))
    fi
    
    print_status "Minimal tests: $tests_passed/$tests_total passed"
    return $((tests_total - tests_passed))
}

# Function to run standard verification tests
run_standard_tests() {
    print_header "Standard Verification Tests"
    
    local tests_passed=0
    local tests_total=4
    
    # Test 1: Compilation
    if check_p_compiler; then
        if run_verification "compile" 60; then
            tests_passed=$((tests_passed + 1))
        fi
    else
        if validate_syntax; then
            tests_passed=$((tests_passed + 1))
        fi
    fi
    
    # Test 2: Token economics
    if run_verification "test TokenTransferConsistency" 60; then
        tests_passed=$((tests_passed + 1))
    fi
    
    # Test 3: Governance
    if run_verification "test GovernanceLifecycle" 60; then
        tests_passed=$((tests_passed + 1))
    fi
    
    # Test 4: Network composition
    if run_verification "test FullNetworkComposition" 120; then
        tests_passed=$((tests_passed + 1))
    fi
    
    print_status "Standard tests: $tests_passed/$tests_total passed"
    return $((tests_total - tests_passed))
}

# Function to run thorough verification tests
run_thorough_tests() {
    print_header "Thorough Verification Tests"
    
    local tests_passed=0
    local tests_total=8
    
    # Test 1: Compilation
    if check_p_compiler; then
        if run_verification "compile" 60; then
            tests_passed=$((tests_passed + 1))
        fi
    else
        if validate_syntax; then
            tests_passed=$((tests_passed + 1))
        fi
    fi
    
    # Test 2: All component tests
    local component_tests=(
        "TokenTransferConsistency"
        "GovernanceLifecycle"
        "SkillRegistration"
        "NodeTransformation"
        "KnowledgeGraphConsistency"
    )
    
    for test in "${component_tests[@]}"; do
        if run_verification "test $test" 60; then
            tests_passed=$((tests_passed + 1))
        fi
    done
    
    # Test 3: Network-wide tests
    local network_tests=(
        "FullNetworkComposition"
        "NetworkMaliciousBehaviorRejection"
    )
    
    for test in "${network_tests[@]}"; do
        if run_verification "test $test" 120; then
            tests_passed=$((tests_passed + 1))
        fi
    done
    
    print_status "Thorough tests: $tests_passed/$tests_total passed"
    return $((tests_total - tests_passed))
}

# Main verification logic
main() {
    local overall_result=0
    
    # Check prerequisites
    if [ ! -f "scripts/run-tests.sh" ]; then
        print_error "run-tests.sh not found - modp system not properly installed"
        exit 1
    fi
    
    # Run tests based on verification level
    case "$VERIFICATION_LEVEL" in
        "minimal")
            run_minimal_tests || overall_result=$?
            ;;
        "standard")
            run_standard_tests || overall_result=$?
            ;;
        "thorough")
            run_thorough_tests || overall_result=$?
            ;;
        *)
            print_error "Unknown verification level: $VERIFICATION_LEVEL"
            print_status "Valid levels: minimal, standard, thorough"
            exit 1
            ;;
    esac
    
    # Generate verification report
    print_header "Verification Summary"
    
    if [ $overall_result -eq 0 ]; then
        print_success "All verification tests passed!"
        print_status "Network model is formally verified ✅"
        
        # Create verification success marker
        echo "verification_passed_$(date +%Y%m%d_%H%M%S)" > data/verification_success.txt
        
    else
        print_error "$overall_result verification test(s) failed"
        print_status "Network model has verification issues ❌"
        print_status "Check logs in modp/results/ for details"
        print_warning "Deployment may be unsafe"
        
        # Create verification failure marker
        echo "verification_failed_$(date +%Y%m%d_%H%M%S)" > data/verification_failure.txt
        
        exit 1
    fi
    
    # Cleanup temporary files
    rm -f verification-temp.log
    
    print_success "Formal verification gate completed successfully"
}

# Run main function
main "$@"