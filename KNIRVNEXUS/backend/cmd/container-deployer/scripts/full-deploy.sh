#!/bin/bash

# KNIRV-NEXUS Full Deployment Script
# This script performs a complete deployment including infrastructure, application, monitoring, and security

set -euo pipefail

# Script configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
DEPLOYMENT_DIR="${SCRIPT_DIR}/.."
ANSIBLE_DIR="${DEPLOYMENT_DIR}/ansible"

# Default values
ENVIRONMENT="${ENVIRONMENT:-production}"
AWS_REGION="${AWS_REGION:-us-east-1}"
INSTANCE_TYPE="${INSTANCE_TYPE:-t3.medium}"
WORKER_COUNT="${WORKER_COUNT:-2}"
ENABLE_SSL="${ENABLE_SSL:-true}"
ENABLE_MONITORING="${ENABLE_MONITORING:-true}"
ENABLE_SECURITY="${ENABLE_SECURITY:-true}"
DRY_RUN="${DRY_RUN:-false}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
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

log_step() {
    echo -e "${PURPLE}[STEP]${NC} $1"
}

# Help function
show_help() {
    cat << EOF
KNIRV-NEXUS Full Deployment Script

This script performs a complete deployment of KNIRV-NEXUS including:
- AWS Infrastructure provisioning
- Application deployment
- Monitoring setup (Prometheus, Grafana, Alertmanager)
- Security hardening
- SSL/TLS configuration
- DNS management

Usage: $0 [OPTIONS]

Options:
    -e, --environment ENV       Deployment environment (default: production)
    -r, --region REGION         AWS region (default: us-east-1)
    -t, --instance-type TYPE    EC2 instance type (default: t3.medium)
    -w, --workers COUNT         Number of worker nodes (default: 2)
    --no-ssl                    Disable SSL/TLS setup
    --no-monitoring             Disable monitoring setup
    --no-security               Disable security hardening
    --dry-run                   Show what would be done without executing
    -h, --help                  Show this help message

Required Environment Variables:
    AWS_PROFILE                 AWS profile to use
    CLOUDFLARE_API_TOKEN        CloudFlare API token
    CLOUDFLARE_ZONE             CloudFlare zone name (e.g., knirv.com)
    DOMAIN_NAME                 Domain name for the application
    SSL_EMAIL                   Email for SSL certificate registration
    ALERT_EMAIL                 Email for alerts and notifications

Optional Environment Variables:
    SSH_KEY_NAME                SSH key name for EC2 instances
    SLACK_WEBHOOK_URL           Slack webhook for notifications
    GIT_REPO                    Git repository URL
    GIT_BRANCH                  Git branch to deploy

Examples:
    # Full production deployment
    $0 --environment production

    # Staging deployment without SSL
    $0 --environment staging --no-ssl

    # Development deployment with minimal resources
    $0 --environment development --instance-type t3.micro --workers 1

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
            --no-ssl)
                ENABLE_SSL="false"
                shift
                ;;
            --no-monitoring)
                ENABLE_MONITORING="false"
                shift
                ;;
            --no-security)
                ENABLE_SECURITY="false"
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
    log_step "Validating prerequisites..."
    
    # Check required tools
    local required_tools=("ansible" "aws" "git" "jq" "curl")
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
    
    # Check required environment variables
    local required_vars=("CLOUDFLARE_API_TOKEN" "CLOUDFLARE_ZONE" "DOMAIN_NAME")
    for var in "${required_vars[@]}"; do
        if [[ -z "${!var:-}" ]]; then
            log_error "Environment variable $var is required"
            exit 1
        fi
    done
    
    if [[ "$ENABLE_SSL" == "true" && -z "${SSL_EMAIL:-}" ]]; then
        log_error "SSL_EMAIL environment variable is required when SSL is enabled"
        exit 1
    fi
    
    # Install Ansible collections
    if ! ansible-galaxy collection list | grep -q "amazon.aws"; then
        log_info "Installing Ansible collections..."
        if [[ "$DRY_RUN" != "true" ]]; then
            ansible-galaxy collection install -r "${ANSIBLE_DIR}/requirements.yml"
        fi
    fi
    
    log_success "Prerequisites validated"
}

# Deploy infrastructure
deploy_infrastructure() {
    log_step "Deploying AWS infrastructure..."
    
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
    else
        if ! eval "$ansible_cmd"; then
            log_error "Infrastructure deployment failed"
            exit 1
        fi
    fi
    
    log_success "Infrastructure deployed successfully"
}

# Configure DNS
configure_dns() {
    log_step "Configuring CloudFlare DNS..."
    
    # Get infrastructure information
    local infra_file="${ANSIBLE_DIR}/infrastructure-${ENVIRONMENT}.env"
    if [[ ! -f "$infra_file" ]]; then
        log_error "Infrastructure file not found: $infra_file"
        exit 1
    fi
    
    source "$infra_file"
    
    cd "$ANSIBLE_DIR"
    
    local ansible_cmd="ansible-playbook playbooks/cloudflare-dns.yml"
    ansible_cmd+=" -e env=$ENVIRONMENT"
    ansible_cmd+=" -e cloudflare_api_token=$CLOUDFLARE_API_TOKEN"
    ansible_cmd+=" -e cloudflare_zone=$CLOUDFLARE_ZONE"
    ansible_cmd+=" -e main_server_ip=$MAIN_INSTANCE_PUBLIC_IP"
    ansible_cmd+=" -e load_balancer_dns=$LOAD_BALANCER_DNS"
    ansible_cmd+=" -e domain_name=$DOMAIN_NAME"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        ansible_cmd+=" --check"
        log_info "[DRY RUN] Would run: $ansible_cmd"
    else
        if ! eval "$ansible_cmd"; then
            log_error "DNS configuration failed"
            exit 1
        fi
    fi
    
    log_success "DNS configured successfully"
}

# Deploy application
deploy_application() {
    log_step "Deploying KNIRV-NEXUS application..."
    
    cd "$ANSIBLE_DIR"
    
    local ansible_cmd="ansible-playbook playbooks/application.yml"
    ansible_cmd+=" -e env=$ENVIRONMENT"
    ansible_cmd+=" -e enable_ssl=$ENABLE_SSL"
    ansible_cmd+=" -e domain_name=$DOMAIN_NAME"
    ansible_cmd+=" -i inventory/aws_ec2.yml"
    
    if [[ -n "${SSL_EMAIL:-}" ]]; then
        ansible_cmd+=" -e ssl_email=$SSL_EMAIL"
    fi
    
    if [[ -n "${GIT_REPO:-}" ]]; then
        ansible_cmd+=" -e git_repo=$GIT_REPO"
    fi
    
    if [[ -n "${GIT_BRANCH:-}" ]]; then
        ansible_cmd+=" -e git_branch=$GIT_BRANCH"
    fi
    
    if [[ "$DRY_RUN" == "true" ]]; then
        ansible_cmd+=" --check"
        log_info "[DRY RUN] Would run: $ansible_cmd"
    else
        if ! eval "$ansible_cmd"; then
            log_error "Application deployment failed"
            exit 1
        fi
    fi
    
    log_success "Application deployed successfully"
}

# Setup monitoring
setup_monitoring() {
    if [[ "$ENABLE_MONITORING" != "true" ]]; then
        log_info "Skipping monitoring setup"
        return 0
    fi
    
    log_step "Setting up monitoring (Prometheus, Grafana, Alertmanager)..."
    
    cd "$ANSIBLE_DIR"
    
    local ansible_cmd="ansible-playbook playbooks/monitoring.yml"
    ansible_cmd+=" -e env=$ENVIRONMENT"
    ansible_cmd+=" -i inventory/aws_ec2.yml"
    
    if [[ -n "${ALERT_EMAIL:-}" ]]; then
        ansible_cmd+=" -e alert_email=$ALERT_EMAIL"
    fi
    
    if [[ -n "${SLACK_WEBHOOK_URL:-}" ]]; then
        ansible_cmd+=" -e slack_webhook_url=$SLACK_WEBHOOK_URL"
    fi
    
    if [[ "$DRY_RUN" == "true" ]]; then
        ansible_cmd+=" --check"
        log_info "[DRY RUN] Would run: $ansible_cmd"
    else
        if ! eval "$ansible_cmd"; then
            log_error "Monitoring setup failed"
            exit 1
        fi
    fi
    
    log_success "Monitoring setup completed"
}

# Apply security hardening
apply_security() {
    if [[ "$ENABLE_SECURITY" != "true" ]]; then
        log_info "Skipping security hardening"
        return 0
    fi
    
    log_step "Applying security hardening..."
    
    cd "$ANSIBLE_DIR"
    
    local ansible_cmd="ansible-playbook playbooks/security.yml"
    ansible_cmd+=" -e env=$ENVIRONMENT"
    ansible_cmd+=" -e enable_ssl=$ENABLE_SSL"
    ansible_cmd+=" -e domain_name=$DOMAIN_NAME"
    ansible_cmd+=" -i inventory/aws_ec2.yml"
    
    if [[ -n "${SSL_EMAIL:-}" ]]; then
        ansible_cmd+=" -e ssl_email=$SSL_EMAIL"
    fi
    
    if [[ -n "${ALERT_EMAIL:-}" ]]; then
        ansible_cmd+=" -e alert_email=$ALERT_EMAIL"
    fi
    
    if [[ "$DRY_RUN" == "true" ]]; then
        ansible_cmd+=" --check"
        log_info "[DRY RUN] Would run: $ansible_cmd"
    else
        if ! eval "$ansible_cmd"; then
            log_error "Security hardening failed"
            exit 1
        fi
    fi
    
    log_success "Security hardening completed"
}

# Run comprehensive health checks
run_health_checks() {
    log_step "Running comprehensive health checks..."
    
    # Get infrastructure information
    local infra_file="${ANSIBLE_DIR}/infrastructure-${ENVIRONMENT}.env"
    if [[ -f "$infra_file" ]]; then
        source "$infra_file"
        
        local base_url="https://${DOMAIN_NAME}"
        if [[ "$ENABLE_SSL" != "true" ]]; then
            base_url="http://${MAIN_INSTANCE_PUBLIC_IP}:8082"
        fi
        
        # Health check endpoints
        local endpoints=(
            "${base_url}/api/v1/health"
            "${base_url}/api/v1/status"
        )
        
        if [[ "$ENABLE_MONITORING" == "true" ]]; then
            endpoints+=(
                "http://${MAIN_INSTANCE_PUBLIC_IP}:9090/-/healthy"  # Prometheus
                "http://${MAIN_INSTANCE_PUBLIC_IP}:3000/api/health" # Grafana
            )
        fi
        
        if [[ "$DRY_RUN" != "true" ]]; then
            for endpoint in "${endpoints[@]}"; do
                log_info "Checking: $endpoint"
                if curl -s -f "$endpoint" > /dev/null; then
                    log_success "✓ $endpoint"
                else
                    log_warning "✗ $endpoint"
                fi
            done
        else
            log_info "[DRY RUN] Would check health endpoints"
        fi
    fi
    
    log_success "Health checks completed"
}

# Generate deployment report
generate_report() {
    log_step "Generating deployment report..."
    
    local report_file="${DEPLOYMENT_DIR}/deployment-report-${ENVIRONMENT}-$(date +%Y%m%d-%H%M%S).md"
    
    cat > "$report_file" << EOF
# KNIRV-NEXUS Deployment Report

**Deployment Date:** $(date)
**Environment:** $ENVIRONMENT
**AWS Region:** $AWS_REGION
**Instance Type:** $INSTANCE_TYPE
**Worker Count:** $WORKER_COUNT

## Configuration
- SSL/TLS: $([ "$ENABLE_SSL" == "true" ] && echo "Enabled" || echo "Disabled")
- Monitoring: $([ "$ENABLE_MONITORING" == "true" ] && echo "Enabled" || echo "Disabled")
- Security Hardening: $([ "$ENABLE_SECURITY" == "true" ] && echo "Enabled" || echo "Disabled")
- Domain: $DOMAIN_NAME

## Infrastructure
EOF

    # Add infrastructure details if available
    local infra_file="${ANSIBLE_DIR}/infrastructure-${ENVIRONMENT}.env"
    if [[ -f "$infra_file" ]]; then
        echo "" >> "$report_file"
        echo "### AWS Resources" >> "$report_file"
        echo '```' >> "$report_file"
        cat "$infra_file" >> "$report_file"
        echo '```' >> "$report_file"
    fi
    
    # Add DNS details if available
    local dns_file="${ANSIBLE_DIR}/dns-${ENVIRONMENT}.env"
    if [[ -f "$dns_file" ]]; then
        echo "" >> "$report_file"
        echo "### DNS Configuration" >> "$report_file"
        echo '```' >> "$report_file"
        cat "$dns_file" >> "$report_file"
        echo '```' >> "$report_file"
    fi
    
    cat >> "$report_file" << EOF

## Access URLs
- Application: https://$DOMAIN_NAME
- API: https://$DOMAIN_NAME/api/v1
- Health Check: https://$DOMAIN_NAME/api/v1/health

EOF

    if [[ "$ENABLE_MONITORING" == "true" ]]; then
        cat >> "$report_file" << EOF
## Monitoring URLs
- Prometheus: http://$(source "$infra_file" 2>/dev/null && echo "$MAIN_INSTANCE_PUBLIC_IP" || echo "IP"):9090
- Grafana: http://$(source "$infra_file" 2>/dev/null && echo "$MAIN_INSTANCE_PUBLIC_IP" || echo "IP"):3000
- Alertmanager: http://$(source "$infra_file" 2>/dev/null && echo "$MAIN_INSTANCE_PUBLIC_IP" || echo "IP"):9093

EOF
    fi
    
    cat >> "$report_file" << EOF
## Next Steps
1. Verify all services are running correctly
2. Configure Grafana dashboards and alerts
3. Set up backup procedures
4. Review security audit results
5. Configure monitoring alerts

## Support
For issues or questions, contact the KNIRV-NEXUS team.
EOF
    
    log_success "Deployment report saved to: $report_file"
}

# Main deployment function
main() {
    echo "=========================================="
    echo "    KNIRV-NEXUS Full Deployment"
    echo "=========================================="
    echo ""
    
    log_info "Starting KNIRV-NEXUS full deployment..."
    log_info "Environment: $ENVIRONMENT"
    log_info "AWS Region: $AWS_REGION"
    log_info "Domain: $DOMAIN_NAME"
    log_info "SSL/TLS: $([ "$ENABLE_SSL" == "true" ] && echo "Enabled" || echo "Disabled")"
    log_info "Monitoring: $([ "$ENABLE_MONITORING" == "true" ] && echo "Enabled" || echo "Disabled")"
    log_info "Security: $([ "$ENABLE_SECURITY" == "true" ] && echo "Enabled" || echo "Disabled")"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log_warning "DRY RUN MODE - No actual changes will be made"
    fi
    
    echo ""
    
    # Execute deployment steps
    validate_prerequisites
    deploy_infrastructure
    configure_dns
    deploy_application
    setup_monitoring
    apply_security
    run_health_checks
    generate_report
    
    echo ""
    echo "=========================================="
    log_success "KNIRV-NEXUS deployment completed successfully!"
    echo "=========================================="
    
    if [[ "$DRY_RUN" != "true" ]]; then
        log_info "Application URL: https://$DOMAIN_NAME"
        log_info "API URL: https://$DOMAIN_NAME/api/v1"
        
        if [[ "$ENABLE_MONITORING" == "true" ]]; then
            source "${ANSIBLE_DIR}/infrastructure-${ENVIRONMENT}.env" 2>/dev/null || true
            log_info "Monitoring URLs:"
            log_info "  - Prometheus: http://${MAIN_INSTANCE_PUBLIC_IP:-IP}:9090"
            log_info "  - Grafana: http://${MAIN_INSTANCE_PUBLIC_IP:-IP}:3000"
        fi
    fi
}

# Parse arguments and run main function
parse_args "$@"
main
