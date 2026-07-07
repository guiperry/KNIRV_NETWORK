#!/bin/bash
# KNIRVORACLE Deployment Script
# Integrates KNIRVORACLE deployment with KNIRV Network infrastructure

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
KNIRVORACLE_DIR="$PROJECT_ROOT/packages/KNIRVORACLE"
if [[ ! -d "$KNIRVORACLE_DIR" ]]; then
    KNIRVORACLE_DIR="$PROJECT_ROOT/KNIRVORACLE"
fi
ANSIBLE_DIR="$SCRIPT_DIR"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Usage function
usage() {
    cat << EOF
Usage: $0 [OPTIONS] COMMAND

KNIRVORACLE Deployment Script for KNIRV Network

COMMANDS:
    build           Build KNIRVORACLE binary
    deploy          Deploy KNIRVORACLE to infrastructure
    infrastructure  Deploy infrastructure + KNIRVORACLE
    dns-only        Update DNS records only
    verify          Verify KNIRVORACLE deployment
    logs            Show KNIRVORACLE logs
    status          Show KNIRVORACLE status
    restart         Restart KNIRVORACLE service
    help            Show this help message

OPTIONS:
    -e, --env ENV           Environment (production, staging, development, testnet)
    -i, --inventory FILE    Ansible inventory file
    -v, --verbose           Verbose output
    -h, --help              Show this help message

EXAMPLES:
    $0 build
    $0 deploy --env production
    $0 infrastructure --env staging
    $0 verify --env production --verbose
    $0 dns-only --env testnet

ENVIRONMENT VARIABLES:
    CLOUDFLARE_API_TOKEN    CloudFlare API token for DNS updates
    CLOUDFLARE_ZONE_ID      CloudFlare zone ID
    ANSIBLE_VAULT_PASSWORD  Ansible vault password
    KNIRVORACLE_API_KEY     KNIRVORACLE API key

EOF
}

# Parse command line arguments
ENVIRONMENT="production"
INVENTORY=""
VERBOSE=""
COMMAND=""

while [[ $# -gt 0 ]]; do
    case $1 in
        -e|--env)
            ENVIRONMENT="$2"
            shift 2
            ;;
        -i|--inventory)
            INVENTORY="$2"
            shift 2
            ;;
        -v|--verbose)
            VERBOSE="-vvv"
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        build|deploy|infrastructure|dns-only|verify|logs|status|restart|help)
            COMMAND="$1"
            shift
            ;;
        *)
            log_error "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Validate command
if [[ -z "$COMMAND" ]]; then
    log_error "No command specified"
    usage
    exit 1
fi

# Set inventory file if not specified
if [[ -z "$INVENTORY" ]]; then
    INVENTORY="$ANSIBLE_DIR/inventory/hosts.ini"
fi

# Validate environment
case "$ENVIRONMENT" in
    production|staging|development|testnet)
        ;;
    *)
        log_error "Invalid environment: $ENVIRONMENT"
        log_error "Valid environments: production, staging, development, testnet"
        exit 1
        ;;
esac

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check if ansible is installed
    if ! command -v ansible-playbook &> /dev/null; then
        log_error "ansible-playbook is not installed"
        exit 1
    fi
    
    # Check if KNIRVORACLE directory exists
    if [[ ! -d "$KNIRVORACLE_DIR" ]]; then
        log_error "KNIRVORACLE directory not found: $KNIRVORACLE_DIR"
        exit 1
    fi
    
    # Check if inventory file exists
    if [[ ! -f "$INVENTORY" ]]; then
        log_error "Inventory file not found: $INVENTORY"
        exit 1
    fi
    
    # Check environment file
    local env_file="$ANSIBLE_DIR/environments/${ENVIRONMENT}.yml"
    if [[ ! -f "$env_file" ]]; then
        log_error "Environment file not found: $env_file"
        exit 1
    fi
    
    log_success "Prerequisites check passed"
}

# Build KNIRVORACLE binary
build_knirvoracle() {
    log_info "Building KNIRVORACLE binary..."
    
    cd "$KNIRVORACLE_DIR"
    
    # Create bin directory if it doesn't exist
    mkdir -p bin
    
    # Build the binary
    if go build -o bin/knirvoracle .; then
        log_success "KNIRVORACLE binary built successfully"
        ls -la bin/knirvoracle
    else
        log_error "Failed to build KNIRVORACLE binary"
        exit 1
    fi
    
    cd "$SCRIPT_DIR"
}

# Deploy infrastructure
deploy_infrastructure() {
    log_info "Deploying KNIRV Network infrastructure..."
    
    ansible-playbook \
        -i "$INVENTORY" \
        -e "environment=$ENVIRONMENT" \
        -e "@environments/${ENVIRONMENT}.yml" \
        $VERBOSE \
        infrastructure-playbook.yml
    
    log_success "Infrastructure deployment completed"
}

# Deploy KNIRVORACLE
deploy_knirvoracle() {
    log_info "Deploying KNIRVORACLE to $ENVIRONMENT environment..."
    
    # Ensure binary is built
    if [[ ! -f "$KNIRVORACLE_DIR/bin/knirvoracle" ]]; then
        log_warning "KNIRVORACLE binary not found, building..."
        build_knirvoracle
    fi
    
    ansible-playbook \
        -i "$INVENTORY" \
        -e "environment=$ENVIRONMENT" \
        -e "@environments/${ENVIRONMENT}.yml" \
        $VERBOSE \
        knirvoracle-deployment.yml
    
    log_success "KNIRVORACLE deployment completed"
}

# Update DNS records only
update_dns() {
    log_info "Updating CloudFlare DNS records for KNIRVORACLE..."
    
    # Extract just the DNS section from the KNIRVORACLE deployment playbook
    ansible-playbook \
        -i "$INVENTORY" \
        -e "environment=$ENVIRONMENT" \
        -e "@environments/${ENVIRONMENT}.yml" \
        $VERBOSE \
        --tags "dns" \
        knirvoracle-deployment.yml
    
    log_success "DNS records updated"
}

# Verify deployment
verify_deployment() {
    log_info "Verifying KNIRVORACLE deployment..."
    
    ansible \
        -i "$INVENTORY" \
        knirv_nodes \
        -m shell \
        -a "systemctl status knirvoracle" \
        $VERBOSE
    
    ansible \
        -i "$INVENTORY" \
        knirv_nodes \
        -m shell \
        -a "/opt/knirv/knirvoracle/scripts/monitor.sh" \
        $VERBOSE
    
    log_success "Deployment verification completed"
}

# Show logs
show_logs() {
    log_info "Showing KNIRVORACLE logs..."
    
    ansible \
        -i "$INVENTORY" \
        knirv_nodes \
        -m shell \
        -a "journalctl -u knirvoracle -n 50 --no-pager" \
        $VERBOSE
}

# Show status
show_status() {
    log_info "Showing KNIRVORACLE status..."
    
    ansible \
        -i "$INVENTORY" \
        knirv_nodes \
        -m shell \
        -a "systemctl status knirvoracle --no-pager" \
        $VERBOSE
}

# Restart service
restart_service() {
    log_info "Restarting KNIRVORACLE service..."
    
    ansible \
        -i "$INVENTORY" \
        knirv_nodes \
        -m systemd \
        -a "name=knirvoracle state=restarted" \
        --become \
        $VERBOSE
    
    log_success "KNIRVORACLE service restarted"
}

# Main execution
main() {
    log_info "KNIRVORACLE Deployment Script"
    log_info "Environment: $ENVIRONMENT"
    log_info "Inventory: $INVENTORY"
    log_info "Command: $COMMAND"
    echo ""
    
    check_prerequisites
    
    case "$COMMAND" in
        build)
            build_knirvoracle
            ;;
        deploy)
            deploy_knirvoracle
            ;;
        infrastructure)
            deploy_infrastructure
            deploy_knirvoracle
            ;;
        dns-only)
            update_dns
            ;;
        verify)
            verify_deployment
            ;;
        logs)
            show_logs
            ;;
        status)
            show_status
            ;;
        restart)
            restart_service
            ;;
        help)
            usage
            ;;
        *)
            log_error "Unknown command: $COMMAND"
            usage
            exit 1
            ;;
    esac
    
    log_success "Operation completed successfully!"
}

# Run main function
main "$@"
