#!/bin/bash

# KNIRV Network Infrastructure Deployment Script
# Purpose: Deploy infrastructure and update DNS using Ansible
# Usage: ./deploy-infrastructure.sh [environment] [options]

set -e

# =============================================================================
# CONFIGURATION
# =============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
ANSIBLE_DIR="$SCRIPT_DIR"

# Default values
ENVIRONMENT="production"
CLOUD_PROVIDER="aws"
DRY_RUN=false
VERBOSE=false
SKIP_DNS=false
FORCE=false

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# =============================================================================
# HELPER FUNCTIONS
# =============================================================================

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

show_usage() {
    cat << EOF
KNIRV Network Infrastructure Deployment

Usage: $0 [environment] [options]

Environments:
  production    Deploy to production environment (default)
  development   Deploy to development environment
  staging       Deploy to staging environment

Options:
  --cloud-provider PROVIDER  Cloud provider (aws, gcp, azure, digitalocean)
  --dry-run                  Show what would be done without executing
  --verbose                  Enable verbose output
  --skip-dns                 Skip DNS updates
  --force                    Force deployment without confirmation
  --help                     Show this help message

Examples:
  $0 production                           # Deploy production infrastructure
  $0 development --cloud-provider aws     # Deploy dev on AWS
  $0 production --dry-run                 # Preview production deployment
  $0 staging --skip-dns --verbose         # Deploy staging without DNS updates

Required Environment Variables:
  CLOUDFLARE_API_TOKEN       Cloudflare API token for DNS management
  CLOUDFLARE_ZONE_ID         Cloudflare zone ID for the domain
  AWS_ACCESS_KEY_ID          AWS access key (if using AWS)
  AWS_SECRET_ACCESS_KEY      AWS secret key (if using AWS)

EOF
}

check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check if Ansible is installed
    if ! command -v ansible-playbook &> /dev/null; then
        log_error "Ansible is not installed. Please install Ansible first."
        log_info "Install with: pip install ansible"
        exit 1
    fi
    
    # Check if required Ansible collections are installed
    log_info "Checking Ansible collections..."
    ansible-galaxy collection list | grep -q "amazon.aws" || {
        log_warning "Installing amazon.aws collection..."
        ansible-galaxy collection install amazon.aws
    }
    
    ansible-galaxy collection list | grep -q "community.general" || {
        log_warning "Installing community.general collection..."
        ansible-galaxy collection install community.general
    }
    
    # Check cloud provider credentials
    case $CLOUD_PROVIDER in
        aws)
            if [[ -z "$AWS_ACCESS_KEY_ID" || -z "$AWS_SECRET_ACCESS_KEY" ]]; then
                log_error "AWS credentials not found. Please set AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY"
                exit 1
            fi
            ;;
        gcp)
            if [[ -z "$GOOGLE_APPLICATION_CREDENTIALS" ]]; then
                log_error "GCP credentials not found. Please set GOOGLE_APPLICATION_CREDENTIALS"
                exit 1
            fi
            ;;
    esac
    
    # Check Cloudflare credentials (optional for DNS updates)
    if [[ "$SKIP_DNS" == "false" ]]; then
        if [[ -z "$CLOUDFLARE_API_TOKEN" || -z "$CLOUDFLARE_ZONE_ID" ]]; then
            log_warning "Cloudflare credentials not found. DNS updates will be skipped."
            log_warning "Set CLOUDFLARE_API_TOKEN and CLOUDFLARE_ZONE_ID to enable DNS updates."
            SKIP_DNS=true
        fi
    fi
    
    log_success "Prerequisites check completed"
}

validate_environment() {
    local env_file="$ANSIBLE_DIR/environments/$ENVIRONMENT.yml"
    
    if [[ ! -f "$env_file" ]]; then
        log_error "Environment file not found: $env_file"
        log_info "Available environments: production, development, staging"
        exit 1
    fi
    
    log_info "Using environment: $ENVIRONMENT"
    log_info "Environment file: $env_file"
}

show_deployment_summary() {
    log_info "Deployment Summary:"
    echo "===================="
    echo "Environment: $ENVIRONMENT"
    echo "Cloud Provider: $CLOUD_PROVIDER"
    echo "Dry Run: $DRY_RUN"
    echo "Skip DNS: $SKIP_DNS"
    echo "Verbose: $VERBOSE"
    echo "Working Directory: $ANSIBLE_DIR"
    echo ""
    
    if [[ "$FORCE" == "false" && "$DRY_RUN" == "false" ]]; then
        read -p "Continue with deployment? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_info "Deployment cancelled by user"
            exit 0
        fi
    fi
}

run_ansible_playbook() {
    log_info "Starting infrastructure deployment..."
    
    cd "$ANSIBLE_DIR"
    
    # Build ansible-playbook command
    local cmd="ansible-playbook infrastructure-playbook.yml"
    cmd="$cmd -e environment=$ENVIRONMENT"
    cmd="$cmd -e cloud_provider=$CLOUD_PROVIDER"
    cmd="$cmd --extra-vars @environments/$ENVIRONMENT.yml"
    
    # Add Cloudflare variables if available
    if [[ "$SKIP_DNS" == "false" && -n "$CLOUDFLARE_API_TOKEN" ]]; then
        cmd="$cmd -e cloudflare_api_token=$CLOUDFLARE_API_TOKEN"
        cmd="$cmd -e cloudflare_zone_id=$CLOUDFLARE_ZONE_ID"
    fi
    
    # Add optional flags
    if [[ "$DRY_RUN" == "true" ]]; then
        cmd="$cmd --check --diff"
    fi
    
    if [[ "$VERBOSE" == "true" ]]; then
        cmd="$cmd -vvv"
    fi
    
    log_info "Executing: $cmd"
    
    # Run the playbook
    if eval "$cmd"; then
        log_success "Infrastructure deployment completed successfully!"
        
        if [[ "$DRY_RUN" == "false" ]]; then
            log_info "Next steps:"
            echo "1. Deploy KNIRV applications: cd $PROJECT_ROOT && ./deployment/deploy.sh deploy"
            echo "2. Monitor services: /opt/knirv/scripts/health-check.sh"
            echo "3. Access Grafana: http://YOUR_IP:3000"
            echo "4. Check logs: tail -f $ANSIBLE_DIR/ansible.log"
        fi
    else
        log_error "Infrastructure deployment failed!"
        log_info "Check the logs for details: $ANSIBLE_DIR/ansible.log"
        exit 1
    fi
}

cleanup() {
    log_info "Cleaning up temporary files..."
    # Add any cleanup tasks here
}

# =============================================================================
# MAIN SCRIPT
# =============================================================================

main() {
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            production|development|staging)
                ENVIRONMENT="$1"
                shift
                ;;
            --cloud-provider)
                CLOUD_PROVIDER="$2"
                shift 2
                ;;
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            --verbose)
                VERBOSE=true
                shift
                ;;
            --skip-dns)
                SKIP_DNS=true
                shift
                ;;
            --force)
                FORCE=true
                shift
                ;;
            --help)
                show_usage
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                show_usage
                exit 1
                ;;
        esac
    done
    
    # Set trap for cleanup
    trap cleanup EXIT
    
    # Run deployment steps
    log_info "KNIRV Network Infrastructure Deployment"
    log_info "========================================"
    
    check_prerequisites
    validate_environment
    show_deployment_summary
    run_ansible_playbook
    
    log_success "Deployment process completed!"
}

# Run main function with all arguments
main "$@"
