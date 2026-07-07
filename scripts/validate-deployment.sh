#!/bin/bash

# KNIRV Network Phase 7 Deployment Validation Script
# Comprehensive validation procedures and health checks

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
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
VALIDATION_RESULTS_DIR="$PROJECT_ROOT/validation-results"

# Validation configuration
VALIDATION_MODE="comprehensive"  # quick, comprehensive, production
TIMEOUT=300
RETRY_COUNT=3
RETRY_DELAY=10
GENERATE_REPORT=true
RUN_INTEGRATION_TESTS=true
CHECK_PERFORMANCE=true
VALIDATE_SECURITY=true

# Service endpoints
declare -A SERVICES=(
    ["knirv-oracle"]="8083"
    ["knirvchain"]="8080"
    ["knirvgraph"]="8081"
    ["knirv-nexus"]="8082"
    ["knirvrouter"]="3478"
    ["knirvcontroller"]="3000"
    ["knirv-gateway"]="8000"
)

declare -A INFRASTRUCTURE=(
    ["postgres"]="5432"
    ["redis"]="6379"
    ["prometheus"]="9090"
    ["grafana"]="3000"
    ["alertmanager"]="9093"
)

# Validation results
declare -A VALIDATION_RESULTS
TOTAL_CHECKS=0
PASSED_CHECKS=0
FAILED_CHECKS=0
WARNING_CHECKS=0

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_step() {
    echo -e "${BLUE}[STEP]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

# Utility functions
create_validation_directories() {
    mkdir -p "$VALIDATION_RESULTS_DIR"/{reports,logs,screenshots}
}

record_result() {
    local check_name="$1"
    local status="$2"
    local message="$3"
    
    VALIDATION_RESULTS["$check_name"]="$status:$message"
    TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
    
    case "$status" in
        "PASS")
            PASSED_CHECKS=$((PASSED_CHECKS + 1))
            log_success "$check_name: $message"
            ;;
        "FAIL")
            FAILED_CHECKS=$((FAILED_CHECKS + 1))
            log_error "$check_name: $message"
            ;;
        "WARN")
            WARNING_CHECKS=$((WARNING_CHECKS + 1))
            log_warn "$check_name: $message"
            ;;
    esac
}

# Basic connectivity checks
check_service_connectivity() {
    log_step "Checking service connectivity"
    
    for service in "${!SERVICES[@]}"; do
        local port="${SERVICES[$service]}"
        local check_name="connectivity_${service}"
        
        if nc -z localhost "$port" 2>/dev/null; then
            record_result "$check_name" "PASS" "Service responding on port $port"
        else
            record_result "$check_name" "FAIL" "Service not responding on port $port"
        fi
    done
}

check_infrastructure_connectivity() {
    log_step "Checking infrastructure connectivity"
    
    for service in "${!INFRASTRUCTURE[@]}"; do
        local port="${INFRASTRUCTURE[$service]}"
        local check_name="infrastructure_${service}"
        
        if nc -z localhost "$port" 2>/dev/null; then
            record_result "$check_name" "PASS" "Infrastructure service responding on port $port"
        else
            record_result "$check_name" "FAIL" "Infrastructure service not responding on port $port"
        fi
    done
}

# Health endpoint checks
check_health_endpoints() {
    log_step "Checking health endpoints"
    
    for service in "${!SERVICES[@]}"; do
        local port="${SERVICES[$service]}"
        local health_url="http://localhost:$port/health"
        local check_name="health_${service}"
        
        local response=$(curl -s -w "%{http_code}" -o /tmp/health_response "$health_url" 2>/dev/null || echo "000")
        
        if [ "$response" = "200" ]; then
            local health_status=$(jq -r '.status' /tmp/health_response 2>/dev/null || echo "unknown")
            if [ "$health_status" = "healthy" ] || [ "$health_status" = "ok" ]; then
                record_result "$check_name" "PASS" "Health endpoint reports healthy"
            else
                record_result "$check_name" "WARN" "Health endpoint reports: $health_status"
            fi
        elif [ "$response" = "404" ]; then
            record_result "$check_name" "WARN" "Health endpoint not available"
        else
            record_result "$check_name" "FAIL" "Health endpoint returned HTTP $response"
        fi
    done
}

# API functionality checks
check_api_functionality() {
    log_step "Checking API functionality"
    
    # Gateway API check
    local gateway_response=$(curl -s -w "%{http_code}" -o /tmp/gateway_response "http://localhost:8000/gateway/health" 2>/dev/null || echo "000")
    if [ "$gateway_response" = "200" ]; then
        record_result "api_gateway" "PASS" "Gateway API responding correctly"
    else
        record_result "api_gateway" "FAIL" "Gateway API returned HTTP $gateway_response"
    fi
    
    # KNIRV-ORACLE API check
    local oracle_response=$(curl -s -w "%{http_code}" -o /tmp/oracle_response "http://localhost:8083/api/agents" 2>/dev/null || echo "000")
    if [ "$oracle_response" = "200" ] || [ "$oracle_response" = "401" ]; then
        record_result "api_oracle" "PASS" "Oracle API responding"
    else
        record_result "api_oracle" "FAIL" "Oracle API returned HTTP $oracle_response"
    fi
    
    # KNIRVCHAIN API check
    local chain_response=$(curl -s -w "%{http_code}" -o /tmp/chain_response "http://localhost:8080/api/skills" 2>/dev/null || echo "000")
    if [ "$chain_response" = "200" ] || [ "$chain_response" = "401" ]; then
        record_result "api_chain" "PASS" "Chain API responding"
    else
        record_result "api_chain" "FAIL" "Chain API returned HTTP $chain_response"
    fi
}

# Database connectivity checks
check_database_connectivity() {
    log_step "Checking database connectivity"
    
    # PostgreSQL check
    if docker-compose exec -T postgres psql -U postgres -c "SELECT 1;" &>/dev/null; then
        record_result "database_postgres" "PASS" "PostgreSQL connection successful"
        
        # Check database exists
        if docker-compose exec -T postgres psql -U postgres -lqt | cut -d \| -f 1 | grep -qw knirv; then
            record_result "database_knirv_exists" "PASS" "KNIRV database exists"
        else
            record_result "database_knirv_exists" "FAIL" "KNIRV database not found"
        fi
    else
        record_result "database_postgres" "FAIL" "PostgreSQL connection failed"
    fi
    
    # Redis check
    if docker-compose exec -T redis redis-cli ping | grep -q "PONG"; then
        record_result "database_redis" "PASS" "Redis connection successful"
    else
        record_result "database_redis" "FAIL" "Redis connection failed"
    fi
}

# Performance checks
check_performance() {
    if [ "$CHECK_PERFORMANCE" = true ]; then
        log_step "Checking performance metrics"
        
        # API response time check
        for service in "${!SERVICES[@]}"; do
            local port="${SERVICES[$service]}"
            local url="http://localhost:$port/health"
            local check_name="performance_${service}"
            
            local response_time=$(curl -s -w "%{time_total}" -o /dev/null "$url" 2>/dev/null || echo "999")
            local response_time_ms=$(echo "$response_time * 1000" | bc 2>/dev/null || echo "999")
            
            if (( $(echo "$response_time < 2.0" | bc -l) )); then
                record_result "$check_name" "PASS" "Response time: ${response_time_ms}ms"
            elif (( $(echo "$response_time < 5.0" | bc -l) )); then
                record_result "$check_name" "WARN" "Slow response time: ${response_time_ms}ms"
            else
                record_result "$check_name" "FAIL" "Very slow response time: ${response_time_ms}ms"
            fi
        done
        
        # Resource usage check
        local cpu_usage=$(docker stats --no-stream --format "table {{.CPUPerc}}" | tail -n +2 | sed 's/%//' | awk '{sum+=$1} END {print sum/NR}' 2>/dev/null || echo "0")
        if (( $(echo "$cpu_usage < 80" | bc -l) )); then
            record_result "performance_cpu" "PASS" "Average CPU usage: ${cpu_usage}%"
        else
            record_result "performance_cpu" "WARN" "High CPU usage: ${cpu_usage}%"
        fi
    fi
}

# Security checks
check_security() {
    if [ "$VALIDATE_SECURITY" = true ]; then
        log_step "Checking security configuration"
        
        # Check for default passwords
        if curl -s -u admin:admin123 "http://localhost:3000/api/health" &>/dev/null; then
            record_result "security_grafana_default_password" "WARN" "Grafana using default password"
        else
            record_result "security_grafana_default_password" "PASS" "Grafana password changed from default"
        fi
        
        # Check for exposed services
        local exposed_services=$(netstat -tulpn 2>/dev/null | grep -E ":(5432|6379|9090|9093)" | grep "0.0.0.0" | wc -l)
        if [ "$exposed_services" -gt 0 ]; then
            record_result "security_exposed_services" "WARN" "$exposed_services infrastructure services exposed"
        else
            record_result "security_exposed_services" "PASS" "Infrastructure services not exposed"
        fi
        
        # Check SSL/TLS configuration
        if curl -k -s "https://localhost:8000/gateway/health" &>/dev/null; then
            record_result "security_ssl" "PASS" "SSL/TLS configured"
        else
            record_result "security_ssl" "WARN" "SSL/TLS not configured"
        fi
    fi
}

# Integration tests
run_integration_tests() {
    if [ "$RUN_INTEGRATION_TESTS" = true ]; then
        log_step "Running integration tests"
        
        if [ -f "$PROJECT_ROOT/integration-tests/run-tests.sh" ]; then
            cd "$PROJECT_ROOT/integration-tests"
            if timeout $TIMEOUT ./run-tests.sh --quick &>/dev/null; then
                record_result "integration_tests" "PASS" "Integration tests passed"
            else
                record_result "integration_tests" "FAIL" "Integration tests failed"
            fi
            cd "$PROJECT_ROOT"
        else
            record_result "integration_tests" "WARN" "Integration tests not found"
        fi
    fi
}

# Monitoring checks
check_monitoring() {
    log_step "Checking monitoring stack"
    
    # Prometheus check
    if curl -s "http://localhost:9090/api/v1/status/config" &>/dev/null; then
        record_result "monitoring_prometheus" "PASS" "Prometheus is running"
        
        # Check targets
        local targets_up=$(curl -s "http://localhost:9090/api/v1/targets" | jq '.data.activeTargets | map(select(.health == "up")) | length' 2>/dev/null || echo "0")
        local targets_total=$(curl -s "http://localhost:9090/api/v1/targets" | jq '.data.activeTargets | length' 2>/dev/null || echo "0")
        
        if [ "$targets_up" -gt 0 ] && [ "$targets_up" -eq "$targets_total" ]; then
            record_result "monitoring_targets" "PASS" "All $targets_total targets are up"
        else
            record_result "monitoring_targets" "WARN" "$targets_up/$targets_total targets are up"
        fi
    else
        record_result "monitoring_prometheus" "FAIL" "Prometheus not responding"
    fi
    
    # Grafana check
    if curl -s "http://localhost:3000/api/health" &>/dev/null; then
        record_result "monitoring_grafana" "PASS" "Grafana is running"
    else
        record_result "monitoring_grafana" "FAIL" "Grafana not responding"
    fi
    
    # Alertmanager check
    if curl -s "http://localhost:9093/api/v1/status" &>/dev/null; then
        record_result "monitoring_alertmanager" "PASS" "Alertmanager is running"
    else
        record_result "monitoring_alertmanager" "FAIL" "Alertmanager not responding"
    fi
}

# Generate validation report
generate_report() {
    if [ "$GENERATE_REPORT" = true ]; then
        log_step "Generating validation report"
        
        local report_file="$VALIDATION_RESULTS_DIR/reports/validation_report_$(date '+%Y%m%d_%H%M%S').md"
        
        cat > "$report_file" << EOF
# KNIRV Network Deployment Validation Report

**Date:** $(date '+%Y-%m-%d %H:%M:%S')
**Validation Mode:** $VALIDATION_MODE
**Total Checks:** $TOTAL_CHECKS
**Passed:** $PASSED_CHECKS
**Failed:** $FAILED_CHECKS
**Warnings:** $WARNING_CHECKS

## Summary

$(if [ $FAILED_CHECKS -eq 0 ]; then echo "✅ **DEPLOYMENT VALIDATION PASSED**"; else echo "❌ **DEPLOYMENT VALIDATION FAILED**"; fi)

**Success Rate:** $(echo "scale=2; $PASSED_CHECKS * 100 / $TOTAL_CHECKS" | bc)%

## Detailed Results

EOF

        # Add detailed results
        for check in "${!VALIDATION_RESULTS[@]}"; do
            local result="${VALIDATION_RESULTS[$check]}"
            local status=$(echo "$result" | cut -d':' -f1)
            local message=$(echo "$result" | cut -d':' -f2-)
            
            case "$status" in
                "PASS") echo "✅ **$check:** $message" >> "$report_file" ;;
                "FAIL") echo "❌ **$check:** $message" >> "$report_file" ;;
                "WARN") echo "⚠️ **$check:** $message" >> "$report_file" ;;
            esac
        done
        
        cat >> "$report_file" << EOF

## Recommendations

EOF

        if [ $FAILED_CHECKS -gt 0 ]; then
            echo "- Address failed checks before proceeding to production" >> "$report_file"
        fi
        
        if [ $WARNING_CHECKS -gt 0 ]; then
            echo "- Review warnings and consider improvements" >> "$report_file"
        fi
        
        echo "- Monitor system performance and health continuously" >> "$report_file"
        echo "- Set up automated health checks and alerting" >> "$report_file"
        
        log_success "Validation report generated: $report_file"
    fi
}

# Show validation summary
show_validation_summary() {
    log_step "Validation Summary"
    
    echo -e "${CYAN}================================${NC}"
    echo -e "${CYAN}  KNIRV Deployment Validation${NC}"
    echo -e "${CYAN}================================${NC}"
    echo ""
    echo -e "${GREEN}Results:${NC}"
    echo -e "  • Total Checks: $TOTAL_CHECKS"
    echo -e "  • Passed: ${GREEN}$PASSED_CHECKS${NC}"
    echo -e "  • Failed: ${RED}$FAILED_CHECKS${NC}"
    echo -e "  • Warnings: ${YELLOW}$WARNING_CHECKS${NC}"
    echo ""
    
    local success_rate=$(echo "scale=2; $PASSED_CHECKS * 100 / $TOTAL_CHECKS" | bc)
    echo -e "${GREEN}Success Rate: $success_rate%${NC}"
    echo ""
    
    if [ $FAILED_CHECKS -eq 0 ]; then
        echo -e "${GREEN}✅ DEPLOYMENT VALIDATION PASSED${NC}"
        echo -e "${GREEN}The KNIRV Network is ready for operation!${NC}"
    else
        echo -e "${RED}❌ DEPLOYMENT VALIDATION FAILED${NC}"
        echo -e "${RED}Please address the failed checks before proceeding.${NC}"
    fi
    
    echo ""
    echo -e "${GREEN}Next Steps:${NC}"
    if [ $FAILED_CHECKS -eq 0 ]; then
        echo -e "  1. Monitor system health: http://localhost:3000"
        echo -e "  2. Run load tests if needed"
        echo -e "  3. Configure production monitoring"
    else
        echo -e "  1. Review failed checks in the validation report"
        echo -e "  2. Fix issues and re-run validation"
        echo -e "  3. Check logs for detailed error information"
    fi
    echo ""
}

# Main validation function
main() {
    log_info "Starting KNIRV Network deployment validation"
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --mode)
                VALIDATION_MODE="$2"
                shift 2
                ;;
            --no-integration-tests)
                RUN_INTEGRATION_TESTS=false
                shift
                ;;
            --no-performance)
                CHECK_PERFORMANCE=false
                shift
                ;;
            --no-security)
                VALIDATE_SECURITY=false
                shift
                ;;
            --no-report)
                GENERATE_REPORT=false
                shift
                ;;
            --timeout)
                TIMEOUT="$2"
                shift 2
                ;;
            --help)
                echo "Usage: $0 [OPTIONS]"
                echo "Options:"
                echo "  --mode MODE              Validation mode (quick, comprehensive, production)"
                echo "  --no-integration-tests   Skip integration tests"
                echo "  --no-performance         Skip performance checks"
                echo "  --no-security           Skip security checks"
                echo "  --no-report             Don't generate report"
                echo "  --timeout SECONDS       Timeout for tests"
                echo "  --help                  Show this help message"
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                exit 1
                ;;
        esac
    done
    
    # Adjust checks based on mode
    case "$VALIDATION_MODE" in
        "quick")
            RUN_INTEGRATION_TESTS=false
            CHECK_PERFORMANCE=false
            VALIDATE_SECURITY=false
            ;;
        "production")
            RUN_INTEGRATION_TESTS=true
            CHECK_PERFORMANCE=true
            VALIDATE_SECURITY=true
            ;;
    esac
    
    # Execute validation steps
    create_validation_directories
    check_service_connectivity
    check_infrastructure_connectivity
    check_health_endpoints
    check_api_functionality
    check_database_connectivity
    check_performance
    check_security
    check_monitoring
    run_integration_tests
    generate_report
    show_validation_summary
    
    # Exit with appropriate code
    if [ $FAILED_CHECKS -eq 0 ]; then
        log_success "KNIRV Network deployment validation completed successfully!"
        exit 0
    else
        log_error "KNIRV Network deployment validation failed!"
        exit 1
    fi
}

# Run main function with all arguments
main "$@"
