#!/bin/bash

# KNIRV Testnet IP Address Utility
# Gets the current public IP address of the testnet instance by Instance ID
# This handles the case where the IP changes when the instance is stopped/started

set -e

# Configuration
INSTANCE_ID="i-06813be8a8a23ea5b"
REGION="us-east-2"
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TESTNET_IP_FILE="$PROJECT_ROOT/deployment/ansible/testnet_ip.txt"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

get_instance_ip() {
    print_step "Getting current IP address for instance $INSTANCE_ID..."
    
    # Check if instance exists and get its state
    INSTANCE_STATE=$(aws ec2 describe-instances \
        --instance-ids "$INSTANCE_ID" \
        --query 'Reservations[0].Instances[0].State.Name' \
        --output text \
        --region "$REGION" 2>/dev/null || echo "not-found")
    
    if [ "$INSTANCE_STATE" = "not-found" ]; then
        print_error "Instance $INSTANCE_ID not found"
        exit 1
    fi
    
    if [ "$INSTANCE_STATE" != "running" ]; then
        print_error "Instance $INSTANCE_ID is in state: $INSTANCE_STATE"
        print_warning "Please start the instance first: aws ec2 start-instances --instance-ids $INSTANCE_ID"
        exit 1
    fi
    
    # Get the current public IP address of the instance
    TESTNET_IP=$(aws ec2 describe-instances \
        --instance-ids "$INSTANCE_ID" \
        --query 'Reservations[0].Instances[0].PublicIpAddress' \
        --output text \
        --region "$REGION")
    
    if [ "$TESTNET_IP" = "None" ] || [ -z "$TESTNET_IP" ]; then
        print_error "Instance $INSTANCE_ID does not have a public IP address"
        print_warning "Please ensure the instance has a public IP assigned"
        exit 1
    fi
    
    print_success "Current IP address: $TESTNET_IP"
    
    # Update the IP file for other scripts
    mkdir -p "$(dirname "$TESTNET_IP_FILE")"
    echo "$TESTNET_IP" > "$TESTNET_IP_FILE"
    
    # Update SSH config if it exists
    update_ssh_config
    
    echo "$TESTNET_IP"
}

update_ssh_config() {
    print_step "Updating SSH configuration..."
    
    # Remove existing knirv-testnet configuration
    if [ -f ~/.ssh/config ]; then
        # Create a backup
        cp ~/.ssh/config ~/.ssh/config.backup
        
        # Remove existing knirv-testnet section
        sed -i '/^# KNIRVTESTNET Configuration$/,/^$/d' ~/.ssh/config
    fi
    
    # Add new configuration
    cat >> ~/.ssh/config << EOF

# KNIRVTESTNET Configuration
Host knirv-testnet
    HostName $TESTNET_IP
    User ubuntu
    IdentityFile ~/.ssh/AEGONG.pem
    StrictHostKeyChecking no
    UserKnownHostsFile /dev/null
EOF
    
    print_success "SSH configuration updated"
    print_success "You can now connect with: ssh knirv-testnet"
}

start_instance() {
    print_step "Starting instance $INSTANCE_ID..."
    
    aws ec2 start-instances --instance-ids "$INSTANCE_ID" --region "$REGION" > /dev/null
    
    print_step "Waiting for instance to be running..."
    aws ec2 wait instance-running --instance-ids "$INSTANCE_ID" --region "$REGION"
    
    print_success "Instance is now running"
    
    # Get the new IP address
    get_instance_ip
}

stop_instance() {
    print_step "Stopping instance $INSTANCE_ID..."
    
    aws ec2 stop-instances --instance-ids "$INSTANCE_ID" --region "$REGION" > /dev/null
    
    print_step "Waiting for instance to be stopped..."
    aws ec2 wait instance-stopped --instance-ids "$INSTANCE_ID" --region "$REGION"
    
    print_success "Instance is now stopped"
}

show_status() {
    print_step "Getting instance status..."
    
    INSTANCE_INFO=$(aws ec2 describe-instances \
        --instance-ids "$INSTANCE_ID" \
        --query 'Reservations[0].Instances[0].{State:State.Name,PublicIP:PublicIpAddress,PrivateIP:PrivateIpAddress,InstanceType:InstanceType}' \
        --output table \
        --region "$REGION")
    
    echo "$INSTANCE_INFO"
}

# Main execution
case "${1:-get-ip}" in
    get-ip|ip)
        get_instance_ip
        ;;
    start)
        start_instance
        ;;
    stop)
        stop_instance
        ;;
    status)
        show_status
        ;;
    --help|-h)
        echo "Usage: $0 [command]"
        echo ""
        echo "Commands:"
        echo "  get-ip, ip    Get current IP address (default)"
        echo "  start         Start the testnet instance"
        echo "  stop          Stop the testnet instance"
        echo "  status        Show instance status"
        echo "  --help, -h    Show this help message"
        echo ""
        echo "Instance ID: $INSTANCE_ID"
        echo "Region: $REGION"
        exit 0
        ;;
    *)
        print_error "Unknown command: $1"
        echo "Use --help for usage information"
        exit 1
        ;;
esac
