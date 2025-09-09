#!/bin/bash

# KNIRV Network Factuality Slice Capability Node Initialization Script
# This script initializes the factuality slice capability node and performs
# comprehensive network health validation after first deployment.

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
LOG_FILE="$PROJECT_ROOT/logs/factuality-slice-init.log"
CAPABILITY_CONFIG_FILE="$PROJECT_ROOT/config/factuality-slice-config.json"

# Network endpoints
KNIRV_CONTROLLER_URL="${KNIRV_CONTROLLER_URL:-http://localhost:3000}"
KNIRV_ROUTER_URL="${KNIRV_ROUTER_URL:-http://localhost:8085}"
KNIRV_GRAPH_URL="${KNIRV_GRAPH_URL:-http://localhost:8081}"
KNIRV_CHAIN_URL="${KNIRV_CHAIN_URL:-http://localhost:8080}"
KNIRV_ORACLE_URL="${KNIRV_ORACLE_URL:-http://localhost:8086}"
KNIRV_NEXUS_URL="${KNIRV_NEXUS_URL:-http://localhost:8090}"

# Factuality slice configuration
FACTUALITY_SLICE_ID="factuality-slice-001"
CAPABILITY_TYPE="factuality-verification"
SLICE_VERSION="1.0.0"
INITIALIZATION_TIMEOUT=300

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1" | tee -a "$LOG_FILE"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1" | tee -a "$LOG_FILE"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1" | tee -a "$LOG_FILE"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" | tee -a "$LOG_FILE"
}

log_step() {
    echo -e "${PURPLE}[STEP]${NC} $1" | tee -a "$LOG_FILE"
}

# Create necessary directories
create_directories() {
    log_step "Creating necessary directories"
    
    mkdir -p "$(dirname "$LOG_FILE")"
    mkdir -p "$(dirname "$CAPABILITY_CONFIG_FILE")"
    mkdir -p "$PROJECT_ROOT/data/factuality-slice"
    
    log_success "Directories created successfully"
}

# Check prerequisites
check_prerequisites() {
    log_step "Checking prerequisites"
    
    # Check required tools
    local required_tools=("curl" "jq" "nc")
    for tool in "${required_tools[@]}"; do
        if ! command -v "$tool" &> /dev/null; then
            log_error "Required tool '$tool' is not installed"
            exit 1
        fi
    done
    
    log_success "All prerequisites satisfied"
}

# Wait for service to be ready
wait_for_service() {
    local service_name="$1"
    local service_url="$2"
    local timeout="${3:-60}"
    
    log_info "Waiting for $service_name to be ready..."
    
    local attempt=1
    local max_attempts=$((timeout / 5))
    
    while [[ $attempt -le $max_attempts ]]; do
        if curl -s -f "$service_url/health" > /dev/null 2>&1; then
            log_success "$service_name is ready"
            return 0
        fi
        
        log_info "Attempt $attempt/$max_attempts: $service_name not ready, waiting..."
        sleep 5
        ((attempt++))
    done
    
    log_error "$service_name failed to become ready within $timeout seconds"
    return 1
}

# Validate network connectivity
validate_network() {
    log_step "Validating network connectivity"
    
    local services=(
        "KNIRVCONTROLLER:$KNIRV_CONTROLLER_URL"
        "KNIRVROUTER:$KNIRV_ROUTER_URL"
        "KNIRVGRAPH:$KNIRV_GRAPH_URL"
        "KNIRVCHAIN:$KNIRV_CHAIN_URL"
        "KNIRVORACLE:$KNIRV_ORACLE_URL"
        "KNIRVNEXUS:$KNIRV_NEXUS_URL"
    )
    
    local failed_services=()
    
    for service_info in "${services[@]}"; do
        local service_name=$(echo "$service_info" | cut -d':' -f1)
        local service_url=$(echo "$service_info" | cut -d':' -f2-)
        
        if ! wait_for_service "$service_name" "$service_url" 30; then
            failed_services+=("$service_name")
        fi
    done
    
    if [[ ${#failed_services[@]} -gt 0 ]]; then
        log_warning "Some services are not available: ${failed_services[*]}"
        log_warning "Continuing with factuality slice initialization..."
    else
        log_success "All network services are healthy"
    fi
}

# Generate factuality slice configuration
generate_capability_config() {
    log_step "Generating factuality slice capability configuration"
    
    cat > "$CAPABILITY_CONFIG_FILE" << EOF
{
  "capability": {
    "id": "$FACTUALITY_SLICE_ID",
    "type": "$CAPABILITY_TYPE",
    "version": "$SLICE_VERSION",
    "name": "Factuality Verification Slice",
    "description": "Capability node for verifying factual accuracy of agent responses",
    "created_at": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
    "status": "initializing"
  },
  "configuration": {
    "verification_threshold": 0.85,
    "confidence_levels": ["low", "medium", "high", "verified"],
    "supported_domains": ["general", "technical", "scientific", "historical"],
    "max_concurrent_verifications": 10,
    "cache_duration_hours": 24
  },
  "network_integration": {
    "controller_endpoint": "$KNIRV_CONTROLLER_URL",
    "router_endpoint": "$KNIRV_ROUTER_URL",
    "graph_endpoint": "$KNIRV_GRAPH_URL",
    "chain_endpoint": "$KNIRV_CHAIN_URL",
    "oracle_endpoint": "$KNIRV_ORACLE_URL",
    "nexus_endpoint": "$KNIRV_NEXUS_URL"
  },
  "initialization": {
    "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
    "script_version": "1.0.0",
    "deployment_environment": "${DEPLOYMENT_ENV:-production}"
  }
}
EOF
    
    log_success "Factuality slice configuration generated: $CAPABILITY_CONFIG_FILE"
}

# Register capability with KNIRV network
register_capability() {
    log_step "Registering factuality slice capability with KNIRV network"
    
    # Register with KNIRVCONTROLLER
    if curl -s -f "$KNIRV_CONTROLLER_URL/health" > /dev/null 2>&1; then
        local registration_payload=$(jq -c . "$CAPABILITY_CONFIG_FILE")
        
        local response=$(curl -s -X POST \
            -H "Content-Type: application/json" \
            -d "$registration_payload" \
            "$KNIRV_CONTROLLER_URL/api/capabilities/register" 2>/dev/null || echo '{"error": "registration_failed"}')
        
        if echo "$response" | jq -e '.success' > /dev/null 2>&1; then
            log_success "Capability registered with KNIRVCONTROLLER"
        else
            log_warning "Failed to register with KNIRVCONTROLLER: $(echo "$response" | jq -r '.error // "unknown_error"')"
        fi
    else
        log_warning "KNIRVCONTROLLER not available for capability registration"
    fi
    
    # Register with KNIRVORACLE
    if curl -s -f "$KNIRV_ORACLE_URL/health" > /dev/null 2>&1; then
        local oracle_payload=$(jq -c '{
            "capability_id": .capability.id,
            "capability_type": .capability.type,
            "version": .capability.version,
            "configuration": .configuration
        }' "$CAPABILITY_CONFIG_FILE")
        
        local response=$(curl -s -X POST \
            -H "Content-Type: application/json" \
            -d "$oracle_payload" \
            "$KNIRV_ORACLE_URL/api/capabilities" 2>/dev/null || echo '{"error": "registration_failed"}')
        
        if echo "$response" | jq -e '.success' > /dev/null 2>&1; then
            log_success "Capability registered with KNIRVORACLE"
        else
            log_warning "Failed to register with KNIRVORACLE: $(echo "$response" | jq -r '.error // "unknown_error"')"
        fi
    else
        log_warning "KNIRVORACLE not available for capability registration"
    fi
}

# Perform end-to-end network health check
perform_health_check() {
    log_step "Performing comprehensive network health check"
    
    local health_report="$PROJECT_ROOT/logs/network-health-$(date +%Y%m%d-%H%M%S).json"
    local overall_status="healthy"
    
    # Initialize health report
    echo '{"timestamp": "'$(date -u +"%Y-%m-%dT%H:%M:%SZ")'", "services": {}, "summary": {"total": 0, "healthy": 0, "unhealthy": 0}}' > "$health_report"
    
    local services=(
        "KNIRVCONTROLLER:$KNIRV_CONTROLLER_URL"
        "KNIRVROUTER:$KNIRV_ROUTER_URL"
        "KNIRVGRAPH:$KNIRV_GRAPH_URL"
        "KNIRVCHAIN:$KNIRV_CHAIN_URL"
        "KNIRVORACLE:$KNIRV_ORACLE_URL"
        "KNIRVNEXUS:$KNIRV_NEXUS_URL"
    )
    
    for service_info in "${services[@]}"; do
        local service_name=$(echo "$service_info" | cut -d':' -f1)
        local service_url=$(echo "$service_info" | cut -d':' -f2-)
        
        local start_time=$(date +%s%3N)
        local service_status="unhealthy"
        local error_message=""
        
        if curl -s -f "$service_url/health" > /dev/null 2>&1; then
            service_status="healthy"
            log_success "$service_name health check passed"
        else
            error_message="Health endpoint not responding"
            overall_status="degraded"
            log_warning "$service_name health check failed"
        fi
        
        local end_time=$(date +%s%3N)
        local response_time=$((end_time - start_time))
        
        # Update health report
        local service_report=$(jq --arg name "$service_name" \
                                 --arg status "$service_status" \
                                 --arg url "$service_url" \
                                 --arg response_time "${response_time}ms" \
                                 --arg error "$error_message" \
                                 --arg timestamp "$(date -u +"%Y-%m-%dT%H:%M:%SZ")" \
                                 '.services[$name] = {
                                     "status": $status,
                                     "url": $url,
                                     "response_time": $response_time,
                                     "error": $error,
                                     "last_check": $timestamp
                                 }' "$health_report")
        echo "$service_report" > "$health_report"
        
        # Update summary
        local summary_update=$(jq '.summary.total += 1 | 
                                  if .services.'$service_name'.status == "healthy" 
                                  then .summary.healthy += 1 
                                  else .summary.unhealthy += 1 
                                  end' "$health_report")
        echo "$summary_update" > "$health_report"
    done
    
    # Add overall status to report
    local final_report=$(jq --arg status "$overall_status" '.overall_status = $status' "$health_report")
    echo "$final_report" > "$health_report"
    
    log_success "Network health check completed. Report saved to: $health_report"
    
    # Display summary
    local healthy_count=$(jq -r '.summary.healthy' "$health_report")
    local total_count=$(jq -r '.summary.total' "$health_report")
    
    log_info "Health Check Summary: $healthy_count/$total_count services healthy"
    
    if [[ "$overall_status" == "healthy" ]]; then
        log_success "Network is fully operational"
        return 0
    else
        log_warning "Network is operational but some services are degraded"
        return 1
    fi
}

# Update capability status
update_capability_status() {
    local status="$1"
    
    log_step "Updating factuality slice capability status to: $status"
    
    local updated_config=$(jq --arg status "$status" \
                             --arg timestamp "$(date -u +"%Y-%m-%dT%H:%M:%SZ")" \
                             '.capability.status = $status | 
                              .capability.last_updated = $timestamp' \
                             "$CAPABILITY_CONFIG_FILE")
    
    echo "$updated_config" > "$CAPABILITY_CONFIG_FILE"
    
    log_success "Capability status updated to: $status"
}

# Main execution function
main() {
    log_info "Starting KNIRV Factuality Slice Capability Node Initialization"
    log_info "================================================================"
    
    # Create log entry
    echo "$(date -u +"%Y-%m-%dT%H:%M:%SZ") - Factuality Slice Initialization Started" >> "$LOG_FILE"
    
    # Execute initialization steps
    create_directories
    check_prerequisites
    validate_network
    generate_capability_config
    register_capability
    
    # Perform comprehensive health check
    if perform_health_check; then
        update_capability_status "active"
        log_success "Factuality slice capability node initialization completed successfully"
        log_success "Network validation passed - all systems operational"
        exit 0
    else
        update_capability_status "degraded"
        log_warning "Factuality slice capability node initialized with warnings"
        log_warning "Some network services are not fully operational"
        exit 1
    fi
}

# Execute main function
main "$@"
