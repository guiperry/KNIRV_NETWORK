#!/bin/bash

# KNIRVTESTNET Infrastructure Deployment Script
# Deploys dedicated testnet infrastructure to AWS with Cloudflare DNS integration

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Configuration
ENVIRONMENT="testnet"
ANSIBLE_DIR="$SCRIPT_DIR"
INVENTORY_FILE="$ANSIBLE_DIR/inventory/testnet.ini"
PLAYBOOK="$ANSIBLE_DIR/testnet-infrastructure-playbook.yml"
ENV_FILE="$ANSIBLE_DIR/.env"

# Functions
print_header() {
    echo -e "${BLUE}=================================${NC}"
    echo -e "${BLUE}🧪 KNIRVTESTNET Infrastructure${NC}"
    echo -e "${BLUE}=================================${NC}"
    echo ""
}

print_step() {
    echo -e "${PURPLE}[STEP]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_prerequisites() {
    print_step "Checking prerequisites..."
    
    # Check if ansible is installed
    if ! command -v ansible-playbook &> /dev/null; then
        print_error "ansible-playbook not found. Please install Ansible."
        exit 1
    fi
    
    # Check if .env file exists
    if [ ! -f "$ENV_FILE" ]; then
        print_error ".env file not found at $ENV_FILE"
        print_warning "Please copy .env.example to .env and configure your settings"
        exit 1
    fi
    
    # Source environment variables
    source "$ENV_FILE"
    
    # Check required environment variables
    if [ -z "$CLOUDFLARE_API_TOKEN" ]; then
        print_error "CLOUDFLARE_API_TOKEN not set in .env file"
        exit 1
    fi
    
    if [ -z "$CLOUDFLARE_ZONE_ID" ]; then
        print_error "CLOUDFLARE_ZONE_ID not set in .env file"
        exit 1
    fi
    
    if [ -z "$AWS_ACCESS_KEY_ID" ]; then
        print_error "AWS_ACCESS_KEY_ID not set in .env file"
        exit 1
    fi
    
    if [ -z "$AWS_SECRET_ACCESS_KEY" ]; then
        print_error "AWS_SECRET_ACCESS_KEY not set in .env file"
        exit 1
    fi
    
    print_success "Prerequisites check passed"
}

install_ansible_collections() {
    print_step "Installing Ansible collections..."
    
    cd "$ANSIBLE_DIR"
    ansible-galaxy collection install -r requirements.yml --force
    
    print_success "Ansible collections installed"
}

create_inventory() {
    print_step "Creating testnet inventory..."
    
    mkdir -p "$(dirname "$INVENTORY_FILE")"
    
    cat > "$INVENTORY_FILE" << EOF
[testnet_nodes]
# This will be populated after EC2 instance creation

[testnet_nodes:vars]
ansible_user=ubuntu
ansible_ssh_private_key_file=~/.ssh/knirv-testnet-key.pem
ansible_ssh_common_args="-o StrictHostKeyChecking=no"

[all:vars]
environment=testnet
cloud_provider=aws
region=us-east-1
domain_name=knirv.com
EOF
    
    print_success "Testnet inventory created"
}

deploy_infrastructure() {
    print_step "Deploying KNIRVTESTNET infrastructure..."
    
    cd "$ANSIBLE_DIR"
    
    # Run the testnet infrastructure playbook
    ansible-playbook \
        -i "$INVENTORY_FILE" \
        "$PLAYBOOK" \
        -e "environment=$ENVIRONMENT" \
        -e "cloudflare_api_token=$CLOUDFLARE_API_TOKEN" \
        -e "cloudflare_zone_id=$CLOUDFLARE_ZONE_ID" \
        -e "domain_name=${DOMAIN_NAME:-knirv.com}" \
        -e "key_pair_name=${KEY_PAIR_NAME:-knirv-testnet-key}" \
        -v
    
    print_success "Infrastructure deployment completed"
}

update_dns_records() {
    print_step "Updating Cloudflare DNS records..."
    
    # Get the public IP from the deployment
    if [ -f "$ANSIBLE_DIR/testnet_ip.txt" ]; then
        TESTNET_IP=$(cat "$ANSIBLE_DIR/testnet_ip.txt")
        print_success "Testnet IP: $TESTNET_IP"
        
        # Update DNS records via Cloudflare API
        curl -X PUT "https://api.cloudflare.com/client/v4/zones/$CLOUDFLARE_ZONE_ID/dns_records" \
             -H "Authorization: Bearer $CLOUDFLARE_API_TOKEN" \
             -H "Content-Type: application/json" \
             --data "{\"type\":\"A\",\"name\":\"testnet\",\"content\":\"$TESTNET_IP\",\"ttl\":300}"
        
        print_success "DNS records updated"
    else
        print_warning "Could not find testnet IP file"
    fi
}

create_ssh_config() {
    print_step "Creating SSH configuration..."
    
    if [ -f "$ANSIBLE_DIR/testnet_ip.txt" ]; then
        TESTNET_IP=$(cat "$ANSIBLE_DIR/testnet_ip.txt")
        
        cat >> ~/.ssh/config << EOF

# KNIRVTESTNET Configuration
Host knirv-testnet
    HostName $TESTNET_IP
    User ubuntu
    IdentityFile ~/.ssh/knirv-testnet-key.pem
    StrictHostKeyChecking no
    UserKnownHostsFile /dev/null
EOF
        
        print_success "SSH configuration added"
        print_success "You can now connect with: ssh knirv-testnet"
    fi
}

display_summary() {
    echo ""
    echo -e "${GREEN}🎉 KNIRVTESTNET Infrastructure Deployment Complete!${NC}"
    echo -e "${BLUE}=================================================${NC}"
    echo ""
    
    if [ -f "$ANSIBLE_DIR/testnet_ip.txt" ]; then
        TESTNET_IP=$(cat "$ANSIBLE_DIR/testnet_ip.txt")
        echo -e "${YELLOW}Testnet Information:${NC}"
        echo -e "  🌐 Public IP: $TESTNET_IP"
        echo -e "  🔗 Testnet URL: https://testnet.knirv.com"
        echo -e "  🔑 SSH Access: ssh knirv-testnet"
        echo ""
        echo -e "${YELLOW}Service Endpoints:${NC}"
        echo -e "  📊 IPFS Gateway: http://$TESTNET_IP:8080"
        echo -e "  ⛓️  KNIRVCHAIN: http://$TESTNET_IP:8080"
        echo -e "  📈 KNIRVGRAPH: http://$TESTNET_IP:8081"
        echo -e "  🤖 KNIRVNEXUS: http://$TESTNET_IP:8082"
        echo -e "  🏛️  KNIRVORACLE: http://$TESTNET_IP:1317"
        echo -e "  🌐 KNIRVGATEWAY: http://$TESTNET_IP:8087"
        echo ""
        echo -e "${YELLOW}Next Steps:${NC}"
        echo -e "  1. Deploy testnet services: make deploy-testnet-services"
        echo -e "  2. Update frontend: make update-testnet-frontend"
        echo -e "  3. Monitor deployment: ssh knirv-testnet 'docker ps'"
        echo ""
    fi
}

# Main execution
main() {
    print_header
    
    check_prerequisites
    install_ansible_collections
    create_inventory
    deploy_infrastructure
    update_dns_records
    create_ssh_config
    display_summary
}

# Handle script arguments
case "${1:-}" in
    --help|-h)
        echo "Usage: $0 [options]"
        echo ""
        echo "Options:"
        echo "  --help, -h    Show this help message"
        echo "  --dry-run     Preview deployment without making changes"
        echo "  --force       Skip confirmation prompts"
        echo ""
        echo "Environment Variables (set in .env file):"
        echo "  CLOUDFLARE_API_TOKEN    Cloudflare API token"
        echo "  CLOUDFLARE_ZONE_ID      Cloudflare zone ID"
        echo "  AWS_ACCESS_KEY_ID       AWS access key"
        echo "  AWS_SECRET_ACCESS_KEY   AWS secret key"
        echo "  DOMAIN_NAME             Domain name (default: knirv.com)"
        echo "  KEY_PAIR_NAME           AWS key pair name (default: knirv-testnet-key)"
        exit 0
        ;;
    --dry-run)
        print_warning "Dry run mode - no changes will be made"
        # Add dry-run logic here
        exit 0
        ;;
    --force)
        print_warning "Force mode - skipping confirmations"
        ;;
    *)
        # Confirm deployment
        echo -e "${YELLOW}This will deploy KNIRVTESTNET infrastructure to AWS.${NC}"
        echo -e "${YELLOW}This may incur AWS charges.${NC}"
        echo ""
        read -p "Continue? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            echo "Deployment cancelled."
            exit 0
        fi
        ;;
esac

# Run main function
main
