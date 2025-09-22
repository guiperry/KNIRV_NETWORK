#!/bin/bash

# KNIRV Testnet IP Address Utility
# Gets the current public IP address of the testnet instance by Instance ID
# This handles the case where the IP changes when the instance is stopped/started

set -e

# Configuration
# Instance ID will be determined dynamically or passed as parameter
INSTANCE_ID=""
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
    local instance_id="$1"
    
    # If no instance ID provided, try to find a running KNIRV testnet instance
    if [ -z "$instance_id" ]; then
        print_step "No instance ID provided, searching for running KNIRV testnet instances..."
        instance_id=$(aws ec2 describe-instances \
            --filters "Name=tag:Project,Values=KNIRV" "Name=tag:Environment,Values=testnet" "Name=instance-state-name,Values=running" \
            --query 'Reservations[*].Instances[*].[InstanceId]' \
            --output text \
            --region "$REGION" | head -1)
        
        if [ -z "$instance_id" ]; then
            print_error "No running KNIRV testnet instances found"
            print_warning "Please provide an instance ID or ensure a KNIRV testnet instance is running"
            exit 1
        fi
        print_success "Found running instance: $instance_id"
    fi
    
    print_step "Getting current IP address for instance $instance_id..."
    
    # Check if instance exists and get its state
    INSTANCE_STATE=$(aws ec2 describe-instances \
        --instance-ids "$instance_id" \
        --query 'Reservations[0].Instances[0].State.Name' \
        --output text \
        --region "$REGION" 2>/dev/null || echo "not-found")
    
    if [ "$INSTANCE_STATE" = "not-found" ]; then
        print_error "Instance $instance_id not found"
        exit 1
    fi
    
    if [ "$INSTANCE_STATE" != "running" ]; then
        print_error "Instance $instance_id is in state: $INSTANCE_STATE"
        print_warning "Please start the instance first: aws ec2 start-instances --instance-ids $instance_id"
        exit 1
    fi
    
    # Get the current public IP address of the instance
    TESTNET_IP=$(aws ec2 describe-instances \
        --instance-ids "$instance_id" \
        --query 'Reservations[0].Instances[0].PublicIpAddress' \
        --output text \
        --region "$REGION")
    
    if [ "$TESTNET_IP" = "None" ] || [ -z "$TESTNET_IP" ]; then
        print_error "Instance $instance_id does not have a public IP address"
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
    local instance_id="$1"
    
    # If no instance ID provided, try to find a KNIRV testnet instance
    if [ -z "$instance_id" ]; then
        print_step "No instance ID provided, searching for KNIRV testnet instances..."
        instance_id=$(aws ec2 describe-instances \
            --filters "Name=tag:Project,Values=KNIRV" "Name=tag:Environment,Values=testnet" \
            --query 'Reservations[*].Instances[*].[InstanceId]' \
            --output text \
            --region "$REGION" | head -1)
        
        if [ -z "$instance_id" ]; then
            print_error "No KNIRV testnet instances found"
            exit 1
        fi
        print_success "Found instance: $instance_id"
    fi
    
    print_step "Starting instance $instance_id..."
    
    aws ec2 start-instances --instance-ids "$instance_id" --region "$REGION" > /dev/null
    
    print_step "Waiting for instance to be running..."
    aws ec2 wait instance-running --instance-ids "$instance_id" --region "$REGION"
    
    print_success "Instance is now running"
    
    # Get the new IP address
    get_instance_ip "$instance_id"
}

stop_instance() {
    local instance_id="$1"
    
    # If no instance ID provided, try to find a running KNIRV testnet instance
    if [ -z "$instance_id" ]; then
        print_step "No instance ID provided, searching for running KNIRV testnet instances..."
        instance_id=$(aws ec2 describe-instances \
            --filters "Name=tag:Project,Values=KNIRV" "Name=tag:Environment,Values=testnet" "Name=instance-state-name,Values=running" \
            --query 'Reservations[*].Instances[*].[InstanceId]' \
            --output text \
            --region "$REGION" | head -1)
        
        if [ -z "$instance_id" ]; then
            print_error "No running KNIRV testnet instances found"
            exit 1
        fi
        print_success "Found running instance: $instance_id"
    fi
    
    print_step "Stopping instance $instance_id..."
    
    aws ec2 stop-instances --instance-ids "$instance_id" --region "$REGION" > /dev/null
    
    print_step "Waiting for instance to be stopped..."
    aws ec2 wait instance-stopped --instance-ids "$instance_id" --region "$REGION"
    
    print_success "Instance is now stopped"
}

show_status() {
    local instance_id="$1"
    
    # If no instance ID provided, show all KNIRV testnet instances
    if [ -z "$instance_id" ]; then
        print_step "Getting status for all KNIRV testnet instances..."
        
        INSTANCE_INFO=$(aws ec2 describe-instances \
            --filters "Name=tag:Project,Values=KNIRV" "Name=tag:Environment,Values=testnet" \
            --query 'Reservations[*].Instances[*].{InstanceId:InstanceId,State:State.Name,PublicIP:PublicIpAddress,PrivateIP:PrivateIpAddress,InstanceType:InstanceType,LaunchTime:LaunchTime}' \
            --output table \
            --region "$REGION")
        
        if [ -z "$INSTANCE_INFO" ] || [ "$INSTANCE_INFO" = "None" ]; then
            print_warning "No KNIRV testnet instances found"
            return 0
        fi
    else
        print_step "Getting instance status for $instance_id..."
        
        INSTANCE_INFO=$(aws ec2 describe-instances \
            --instance-ids "$instance_id" \
            --query 'Reservations[0].Instances[0].{State:State.Name,PublicIP:PublicIpAddress,PrivateIP:PrivateIpAddress,InstanceType:InstanceType,LaunchTime:LaunchTime}' \
            --output table \
            --region "$REGION")
    fi
    
    echo "$INSTANCE_INFO"
}

# Main execution
case "${1:-get-ip}" in
    get-ip|ip)
        get_instance_ip "$2"
        ;;
    start)
        start_instance "$2"
        ;;
    stop)
        stop_instance "$2"
        ;;
    status)
        show_status "$2"
        ;;
    --help|-h)
        echo "Usage: $0 [command] [instance-id]"
        echo ""
        echo "Commands:"
        echo "  get-ip, ip    Get current IP address (default)"
        echo "  start         Start the testnet instance"
        echo "  stop          Stop the testnet instance"
        echo "  status        Show instance status"
        echo "  --help, -h    Show this help message"
        echo ""
        echo "Parameters:"
        echo "  instance-id   Optional instance ID. If not provided, script will search for KNIRV testnet instances"
        echo ""
        echo "Examples:"
        echo "  $0 get-ip                    # Get IP of first running KNIRV testnet instance"
        echo "  $0 get-ip i-1234567890abcdef # Get IP of specific instance"
        echo "  $0 start                     # Start first KNIRV testnet instance"
        echo "  $0 status                    # Show status of all KNIRV testnet instances"
        echo ""
        echo "Region: $REGION"
        exit 0
        ;;
    *)
        print_error "Unknown command: $1"
        echo "Use --help for usage information"
        exit 1
        ;;
esac
