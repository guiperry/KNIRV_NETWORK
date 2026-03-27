#!/bin/bash

# KNIRV-SERVER Deployment Script
# This script automates the deployment of KNIRV-SERVER to AWS EC2

set -euo pipefail

# Script configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
DEPLOYMENT_DIR="${SCRIPT_DIR}/.."
ANSIBLE_DIR="${DEPLOYMENT_DIR}/ansible"

# Default values
ENVIRONMENT="${ENVIRONMENT:-testnet}"
AWS_REGION="${AWS_REGION:-us-east-1}"
INSTANCE_TYPE="${INSTANCE_TYPE:-t3.medium}"
WORKER_COUNT="${WORKER_COUNT:-2}"
SKIP_INFRASTRUCTURE="${SKIP_INFRASTRUCTURE:-false}"
SKIP_DNS="${SKIP_DNS:-false}"
SKIP_APPLICATION="${SKIP_APPLICATION:-false}"
DRY_RUN="${DRY_RUN:-false}"

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

# Help function
show_help() {
    cat << EOF
KNIRV-SERVER Deployment Script

Usage: $0 [OPTIONS]

Options:
    -e, --environment ENV       Deployment environment (default: testnet)
    -r, --region REGION         AWS region (default: us-east-1)
    -t, --instance-type TYPE    EC2 instance type (default: t3.medium)
    -w, --workers COUNT         Number of worker nodes (default: 2)
    --skip-infrastructure       Skip infrastructure deployment
    --skip-dns                  Skip DNS configuration
    --skip-application          Skip application deployment
    --dry-run                   Show what would be done without executing
    -h, --help                  Show this help message

Environment Variables:
    AWS_PROFILE                 AWS profile to use
    AWS_REGION                  AWS region
    CLOUDFLARE_API_TOKEN        CloudFlare API token
    CLOUDFLARE_ZONE             CloudFlare zone name
    SSH_KEY_NAME                SSH key name for EC2 instances

Examples:
    # Deploy to testnet
    $0 --environment testnet

    # Deploy to staging with custom instance type
    $0 --environment staging --instance-type t3.small

    # Deploy only application (skip infrastructure)
    $0 --skip-infrastructure --skip-dns

    # Dry run to see what would be deployed
    $0 --dry-run

EOF
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -e|--environment)
                ENVIRONMENT="$2"
                shift 2
                ;;
            -r|--region)
                AWS_REGION="$2"
                shift 2
                ;;
            -t|--instance-type)
                INSTANCE_TYPE="$2"
                shift 2
                ;;
            -w|--workers)
                WORKER_COUNT="$2"
                shift 2
                ;;
            --skip-infrastructure)
                SKIP_INFRASTRUCTURE="true"
                shift
                ;;
            --skip-dns)
                SKIP_DNS="true"
                shift
                ;;
            --skip-application)
                SKIP_APPLICATION="true"
                shift
                ;;
            --dry-run)
                DRY_RUN="true"
                shift
                ;;
            -h|--help)
                show_help
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done
}

# Validate prerequisites
validate_prerequisites() {
    log_info "Validating prerequisites..."
    
    # Check required tools
    local required_tools=("ansible" "aws" "git" "jq")
    for tool in "${required_tools[@]}"; do
        if ! command -v "$tool" &> /dev/null; then
            log_error "$tool is required but not installed"
            exit 1
        fi
    done
    
    # Check AWS credentials
    if ! aws sts get-caller-identity &> /dev/null; then
        log_error "AWS credentials not configured or invalid"
        exit 1
    fi
    
    # Check Ansible collections
    if ! ansible-galaxy collection list | grep -q "amazon.aws"; then
        log_warning "Amazon AWS collection not found, installing..."
        ansible-galaxy collection install -r "${ANSIBLE_DIR}/requirements.yml"
    fi
    
    # Validate environment-specific requirements
    if [[ "$SKIP_DNS" != "true" ]]; then
        if [[ -z "${CLOUDFLARE_API_TOKEN:-}" ]]; then
            log_error "CLOUDFLARE_API_TOKEN environment variable is required for DNS management"
            exit 1
        fi
        
        if [[ -z "${CLOUDFLARE_ZONE:-}" ]]; then
            log_error "CLOUDFLARE_ZONE environment variable is required for DNS management"
            exit 1
        fi
    fi
    
    log_success "Prerequisites validated"
}

# Build application
build_application() {
    log_info "Building KNIRV-SERVER application..."
    
    cd "$PROJECT_ROOT"
    
    # Build frontend
    log_info "Building Next.js frontend..."
    if [[ "$DRY_RUN" != "true" ]]; then
        npm install
        npm run build
    else
        log_info "[DRY RUN] Would run: npm install && npm run build"
    fi
    
    # Build backend
    log_info "Building Go backend..."
    cd "${PROJECT_ROOT}/backend"
    if [[ "$DRY_RUN" != "true" ]]; then
        go mod tidy
        go build -o bin/nexus-server ./cmd/nexus-server
    else
        log_info "[DRY RUN] Would run: go mod tidy && go build"
    fi
    
    log_success "Application built successfully"
}

# Deploy infrastructure
deploy_infrastructure() {
    if [[ "$SKIP_INFRASTRUCTURE" == "true" ]]; then
        log_info "Skipping infrastructure deployment"
        return 0
    fi
    
    log_info "Deploying infrastructure to AWS..."
    
    cd "$ANSIBLE_DIR"
    
    local ansible_cmd="ansible-playbook playbooks/infrastructure.yml"
    ansible_cmd+=" -e env=$ENVIRONMENT"
    ansible_cmd+=" -e aws_region=$AWS_REGION"
    ansible_cmd+=" -e instance_type=$INSTANCE_TYPE"
    ansible_cmd+=" -e worker_count=$WORKER_COUNT"
    
    if [[ -n "${SSH_KEY_NAME:-}" ]]; then
        ansible_cmd+=" -e key_name=$SSH_KEY_NAME"
    fi
    
    if [[ "$DRY_RUN" == "true" ]]; then
        ansible_cmd+=" --check"
        log_info "[DRY RUN] Would run: $ansible_cmd"
    fi
    
    if ! eval "$ansible_cmd"; then
        log_error "Infrastructure deployment failed"
        exit 1
    fi
    
    log_success "Infrastructure deployed successfully"
}

# Configure DNS
configure_dns() {
    if [[ "$SKIP_DNS" == "true" ]]; then
        log_info "Skipping DNS configuration"
        return 0
    fi
    
    log_info "Configuring CloudFlare DNS..."
    
    # Get infrastructure information
    local infra_file="${ANSIBLE_DIR}/infrastructure-${ENVIRONMENT}.env"
    if [[ ! -f "$infra_file" ]]; then
        log_error "Infrastructure file not found: $infra_file"
        log_error "Please run infrastructure deployment first"
        exit 1
    fi
    
    # Source infrastructure variables
    source "$infra_file"
    
    cd "$ANSIBLE_DIR"
    
    local ansible_cmd="ansible-playbook playbooks/cloudflare-dns.yml"
    ansible_cmd+=" -e env=$ENVIRONMENT"
    ansible_cmd+=" -e cloudflare_api_token=$CLOUDFLARE_API_TOKEN"
    ansible_cmd+=" -e cloudflare_zone=$CLOUDFLARE_ZONE"
    ansible_cmd+=" -e main_server_ip=$MAIN_INSTANCE_PUBLIC_IP"
    ansible_cmd+=" -e load_balancer_dns=$LOAD_BALANCER_DNS"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        ansible_cmd+=" --check"
        log_info "[DRY RUN] Would run: $ansible_cmd"
    fi
    
    if ! eval "$ansible_cmd"; then
        log_error "DNS configuration failed"
        exit 1
    fi
    
    log_success "DNS configured successfully"
}

# Deploy application
deploy_application() {
    if [[ "$SKIP_APPLICATION" == "true" ]]; then
        log_info "Skipping application deployment"
        return 0
    fi
    
    log_info "Deploying KNIRV-SERVER application..."
    
    cd "$ANSIBLE_DIR"
    
    local ansible_cmd="ansible-playbook playbooks/application.yml"
    ansible_cmd+=" -e env=$ENVIRONMENT"
    ansible_cmd+=" -i inventory/aws_ec2.yml"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        ansible_cmd+=" --check"
        log_info "[DRY RUN] Would run: $ansible_cmd"
    fi
    
    if ! eval "$ansible_cmd"; then
        log_error "Application deployment failed"
        exit 1
    fi
    
    log_success "Application deployed successfully"
}

# Run health checks
run_health_checks() {
    log_info "Running health checks..."
    
    # Get infrastructure information
    local infra_file="${ANSIBLE_DIR}/infrastructure-${ENVIRONMENT}.env"
    if [[ -f "$infra_file" ]]; then
        source "$infra_file"
        
        # Check main server health
        local health_url="http://${MAIN_INSTANCE_PUBLIC_IP}:8082/api/v1/health"
        log_info "Checking health endpoint: $health_url"
        
        if [[ "$DRY_RUN" != "true" ]]; then
            local max_attempts=30
            local attempt=1
            
            while [[ $attempt -le $max_attempts ]]; do
                if curl -s -f "$health_url" > /dev/null; then
                    log_success "Health check passed"
                    return 0
                fi
                
                log_info "Health check attempt $attempt/$max_attempts failed, retrying in 10 seconds..."
                sleep 10
                ((attempt++))
            done
            
            log_error "Health check failed after $max_attempts attempts"
            return 1
        else
            log_info "[DRY RUN] Would check: $health_url"
        fi
    else
        log_warning "Infrastructure file not found, skipping health checks"
    fi
}

# Generate deployment summary
generate_summary() {
    log_info "Generating deployment summary..."
    
    local summary_file="${DEPLOYMENT_DIR}/deployment-summary-${ENVIRONMENT}-$(date +%Y%m%d-%H%M%S).txt"
    
    cat > "$summary_file" << EOF
KNIRV-SERVER Deployment Summary
==============================

Deployment Date: $(date)
Environment: $ENVIRONMENT
AWS Region: $AWS_REGION
Instance Type: $INSTANCE_TYPE
Worker Count: $WORKER_COUNT

Deployment Steps:
- Infrastructure: $([ "$SKIP_INFRASTRUCTURE" == "true" ] && echo "Skipped" || echo "Deployed")
- DNS Configuration: $([ "$SKIP_DNS" == "true" ] && echo "Skipped" || echo "Configured")
- Application: $([ "$SKIP_APPLICATION" == "true" ] && echo "Skipped" || echo "Deployed")

EOF

    # Add infrastructure details if available
    local infra_file="${ANSIBLE_DIR}/infrastructure-${ENVIRONMENT}.env"
    if [[ -f "$infra_file" ]]; then
        echo "Infrastructure Details:" >> "$summary_file"
        cat "$infra_file" >> "$summary_file"
        echo "" >> "$summary_file"
    fi
    
    # Add DNS details if available
    local dns_file="${ANSIBLE_DIR}/dns-${ENVIRONMENT}.env"
    if [[ -f "$dns_file" ]]; then
        echo "DNS Configuration:" >> "$summary_file"
        cat "$dns_file" >> "$summary_file"
        echo "" >> "$summary_file"
    fi
    
    log_success "Deployment summary saved to: $summary_file"
}

# Main deployment function
main() {
    log_info "Starting KNIRV-SERVER deployment..."
    log_info "Environment: $ENVIRONMENT"
    log_info "AWS Region: $AWS_REGION"
    log_info "Instance Type: $INSTANCE_TYPE"
    log_info "Worker Count: $WORKER_COUNT"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log_warning "DRY RUN MODE - No actual changes will be made"
    fi
    
    # Validate prerequisites
    validate_prerequisites
    
    # Build application
    build_application
    
    # Deploy infrastructure
    deploy_infrastructure
    
    # Configure DNS
    configure_dns
    
    # Deploy application
    deploy_application
    
    # Run health checks
    run_health_checks
    
    # Generate summary
    generate_summary
    
    log_success "KNIRV-SERVER deployment completed successfully!"
    
    if [[ "$SKIP_DNS" != "true" && -n "${CLOUDFLARE_ZONE:-}" ]]; then
        log_info "Application should be available at: https://nexus.${CLOUDFLARE_ZONE}"
    fi
}

# Parse arguments and run main function
parse_args "$@"
main
