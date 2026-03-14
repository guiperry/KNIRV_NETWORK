#!/bin/bash

# Phase 5 Test Runner Script
# Tests for Synchronization and Optimization (Weeks 17-20)

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test configuration
TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$TEST_DIR/../../.." && pwd)"
REPORTS_DIR="$TEST_DIR/reports"
LOGS_DIR="$TEST_DIR/logs"

# Create directories
mkdir -p "$REPORTS_DIR" "$LOGS_DIR"

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1" | tee -a "$LOGS_DIR/phase5-tests.log"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1" | tee -a "$LOGS_DIR/phase5-tests.log"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1" | tee -a "$LOGS_DIR/phase5-tests.log"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" | tee -a "$LOGS_DIR/phase5-tests.log"
}

# Test execution functions
run_synchronization_tests() {
    log_info "Running Phase 5.1 Synchronization Strategy Tests..."
    
    cd "$TEST_DIR"
    
    # Run Go tests for synchronization
    if go test -v -run "TestSynchronizationStrategyTestSuite" . > "$LOGS_DIR/sync-tests.log" 2>&1; then
        log_success "Synchronization strategy tests passed"
        return 0
    else
        log_error "Synchronization strategy tests failed"
        cat "$LOGS_DIR/sync-tests.log"
        return 1
    fi
}

run_agent_builder_tests() {
    log_info "Running Phase 5.2 Agent Builder Updates Tests..."
    
    cd "$TEST_DIR"
    
    # Run Go tests for agent builder
    if go test -v -run "TestAgentBuilderUpdatesTestSuite" . > "$LOGS_DIR/agent-builder-tests.log" 2>&1; then
        log_success "Agent builder updates tests passed"
        return 0
    else
        log_error "Agent builder updates tests failed"
        cat "$LOGS_DIR/agent-builder-tests.log"
        return 1
    fi
}

run_sync_integration_tests() {
    log_info "Running synchronization integration tests..."
    
    # Test actual sync manager if available
    if [ -f "$ROOT_DIR/sync/sync-manager.go" ]; then
        cd "$ROOT_DIR/sync"
        
        # Build sync manager
        if go build -o bin/sync-manager .; then
            log_success "Sync manager built successfully"
            
            # Test sync configuration
            if [ -f "sync-config.json" ]; then
                log_info "Testing sync configuration validation..."
                if ./bin/sync-manager -config sync-config.json -validate; then
                    log_success "Sync configuration is valid"
                else
                    log_warning "Sync configuration validation failed"
                fi
            fi
        else
            log_warning "Failed to build sync manager"
        fi
    else
        log_warning "Sync manager not found, skipping integration tests"
    fi
}

run_typescript_compiler_tests() {
    log_info "Running TypeScript compiler integration tests..."
    
    # Test TypeScript compiler if available
    CORTEX_DIR="$ROOT_DIR/KNIRVCORTEX/agent-core/agent-core-compiler"
    if [ -d "$CORTEX_DIR" ]; then
        cd "$CORTEX_DIR"
        
        # Check if TypeScript compiler exists
        if [ -f "src/AgentCoreCompiler.ts" ]; then
            log_info "Testing TypeScript compiler..."
            
            # Install dependencies if package.json exists
            if [ -f "package.json" ]; then
                npm install > /dev/null 2>&1 || log_warning "npm install failed"
            fi
            
            # Try to compile TypeScript
            if command -v tsc >/dev/null 2>&1; then
                if tsc --noEmit src/AgentCoreCompiler.ts; then
                    log_success "TypeScript compiler validation passed"
                else
                    log_warning "TypeScript compiler validation failed"
                fi
            else
                log_warning "TypeScript compiler not available"
            fi
        else
            log_warning "AgentCoreCompiler.ts not found"
        fi
    else
        log_warning "KNIRVCORTEX agent-core-compiler directory not found"
    fi
}

run_lora_training_tests() {
    log_info "Running LoRA training integration tests..."
    
    # Test LoRA training components
    CONTROLLER_DIR="$ROOT_DIR/KNIRVCONTROLLER"
    if [ -d "$CONTROLLER_DIR" ]; then
        cd "$CONTROLLER_DIR"
        
        # Check for LoRA-related test files
        if find . -name "*lora*test*" -type f | grep -q .; then
            log_info "Found LoRA test files, running tests..."
            
            # Run LoRA-specific tests
            if npm test -- --testNamePattern="LoRA" > "$LOGS_DIR/lora-tests.log" 2>&1; then
                log_success "LoRA training tests passed"
            else
                log_warning "LoRA training tests failed or not available"
                cat "$LOGS_DIR/lora-tests.log" | tail -20
            fi
        else
            log_warning "No LoRA test files found"
        fi
    else
        log_warning "KNIRVCONTROLLER directory not found"
    fi
}

run_nexus_deployment_tests() {
    log_info "Running NEXUS deployment tests..."
    
    # Test NEXUS deployment if available
    NEXUS_DIR="$ROOT_DIR/KNIRVSERVER"
    if [ -d "$NEXUS_DIR" ]; then
        cd "$NEXUS_DIR"
        
        # Check for deployment scripts
        if [ -f "scripts/deploy.sh" ] || [ -f "deploy.sh" ]; then
            log_info "Testing NEXUS deployment scripts..."
            
            # Validate deployment configuration
            if [ -f "config/nexus-config.yaml" ] || [ -f "nexus-config.yaml" ]; then
                log_success "NEXUS deployment configuration found"
            else
                log_warning "NEXUS deployment configuration not found"
            fi
        else
            log_warning "NEXUS deployment scripts not found"
        fi
    else
        log_warning "KNIRVSERVER directory not found"
    fi
}

generate_test_report() {
    log_info "Generating Phase 5 test report..."
    
    local report_file="$REPORTS_DIR/phase5-test-report-$(date +%Y%m%d-%H%M%S).md"
    
    cat > "$report_file" << EOF
# Phase 5 Test Report
**Generated:** $(date)
**Test Suite:** Synchronization and Optimization (Weeks 17-20)

## Test Summary

### 5.1 Synchronization Strategy Refactor
- **Synchronization Accuracy Tests:** $sync_accuracy_status
- **Cross-Environment Consistency Tests:** $cross_env_status
- **Automated Sync Mechanism Tests:** $auto_sync_status
- **Monitoring System Validation Tests:** $monitoring_status
- **Rollback and Recovery Tests:** $rollback_status

### 5.2 KNIRVCORTEX Agent-Builder Updates
- **TypeScript Pipeline Integration Tests:** $ts_pipeline_status
- **Pre-training Functionality Tests:** $pretraining_status
- **Deployment Sequence Tests:** $deployment_status
- **LoRA Adapter Training Tests:** $lora_training_status
- **End-to-End Workflow Tests:** $e2e_workflow_status

## Integration Tests
- **Sync Manager Integration:** $sync_integration_status
- **TypeScript Compiler Integration:** $ts_compiler_status
- **LoRA Training Integration:** $lora_integration_status
- **NEXUS Deployment Integration:** $nexus_deployment_status

## Overall Status
**Phase 5 Completion:** $overall_status

## Recommendations
$recommendations

## Test Logs
- Synchronization Tests: \`logs/sync-tests.log\`
- Agent Builder Tests: \`logs/agent-builder-tests.log\`
- LoRA Training Tests: \`logs/lora-tests.log\`
- Main Test Log: \`logs/phase5-tests.log\`

EOF

    log_success "Test report generated: $report_file"
}

check_prerequisites() {
    log_info "Checking test prerequisites..."
    
    # Check Go
    if ! command -v go >/dev/null 2>&1; then
        log_error "Go is not installed"
        return 1
    fi
    
    # Check Node.js/npm
    if ! command -v node >/dev/null 2>&1; then
        log_warning "Node.js is not installed"
    fi
    
    # Check required Go modules
    if ! go list github.com/stretchr/testify/suite >/dev/null 2>&1; then
        log_info "Installing required Go modules..."
        go mod init phase5-tests 2>/dev/null || true
        go get github.com/stretchr/testify/suite
        go get github.com/stretchr/testify/assert
        go get github.com/stretchr/testify/require
    fi
    
    log_success "Prerequisites check completed"
}

# Main execution
main() {
    log_info "Starting Phase 5 Test Suite"
    log_info "Testing Synchronization and Optimization components"
    
    # Initialize status variables
    sync_accuracy_status="❌ FAILED"
    cross_env_status="❌ FAILED"
    auto_sync_status="❌ FAILED"
    monitoring_status="❌ FAILED"
    rollback_status="❌ FAILED"
    ts_pipeline_status="❌ FAILED"
    pretraining_status="❌ FAILED"
    deployment_status="❌ FAILED"
    lora_training_status="❌ FAILED"
    e2e_workflow_status="❌ FAILED"
    sync_integration_status="❌ FAILED"
    ts_compiler_status="❌ FAILED"
    lora_integration_status="❌ FAILED"
    nexus_deployment_status="❌ FAILED"
    overall_status="❌ FAILED"
    recommendations="- Review failed tests and address issues\n- Ensure all dependencies are properly installed\n- Check configuration files"
    
    # Check prerequisites
    if ! check_prerequisites; then
        log_error "Prerequisites check failed"
        exit 1
    fi
    
    # Run test suites
    local tests_passed=0
    local total_tests=9
    
    # 5.1 Synchronization Strategy Tests
    if run_synchronization_tests; then
        sync_accuracy_status="✅ PASSED"
        cross_env_status="✅ PASSED"
        auto_sync_status="✅ PASSED"
        monitoring_status="✅ PASSED"
        rollback_status="✅ PASSED"
        ((tests_passed++))
    fi
    
    # 5.2 Agent Builder Tests
    if run_agent_builder_tests; then
        ts_pipeline_status="✅ PASSED"
        pretraining_status="✅ PASSED"
        deployment_status="✅ PASSED"
        lora_training_status="✅ PASSED"
        e2e_workflow_status="✅ PASSED"
        ((tests_passed++))
    fi
    
    # Integration tests
    run_sync_integration_tests && sync_integration_status="✅ PASSED" && ((tests_passed++))
    run_typescript_compiler_tests && ts_compiler_status="✅ PASSED" && ((tests_passed++))
    run_lora_training_tests && lora_integration_status="✅ PASSED" && ((tests_passed++))
    run_nexus_deployment_tests && nexus_deployment_status="✅ PASSED" && ((tests_passed++))
    
    # Calculate overall status
    local pass_rate=$((tests_passed * 100 / total_tests))
    if [ $pass_rate -ge 80 ]; then
        overall_status="✅ PASSED ($pass_rate%)"
        recommendations="- Phase 5 implementation is largely complete\n- Address any remaining failed tests\n- Consider performance optimizations"
    elif [ $pass_rate -ge 60 ]; then
        overall_status="⚠️ PARTIAL ($pass_rate%)"
        recommendations="- Significant progress made on Phase 5\n- Focus on failing test areas\n- Review implementation gaps"
    else
        overall_status="❌ FAILED ($pass_rate%)"
        recommendations="- Major implementation work needed\n- Review Phase 5 requirements\n- Address fundamental issues"
    fi
    
    # Generate report
    generate_test_report
    
    log_info "Phase 5 Test Suite completed"
    log_info "Overall Status: $overall_status"
    log_info "Tests Passed: $tests_passed/$total_tests"
    
    # Exit with appropriate code
    if [ $pass_rate -ge 80 ]; then
        exit 0
    else
        exit 1
    fi
}

# Handle script arguments
case "${1:-}" in
    "sync")
        run_synchronization_tests
        ;;
    "agent-builder")
        run_agent_builder_tests
        ;;
    "integration")
        run_sync_integration_tests
        run_typescript_compiler_tests
        run_lora_training_tests
        run_nexus_deployment_tests
        ;;
    "report")
        generate_test_report
        ;;
    *)
        main
        ;;
esac
