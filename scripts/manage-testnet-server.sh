#!/bin/bash

# KNIRV Testnet Server Management Script
# Provides utilities for managing the AWS EC2 testnet instance

# Configuration
INSTANCE_ID="i-06813be8a8a23ea5b"
REGION="us-east-2"
SSH_KEY="~/.ssh/AEGONG.pem"

# Color codes for output
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

print_header() {
    echo -e "${BLUE}🖥️  KNIRV Testnet Server Management${NC}"
    echo -e "${BLUE}===================================${NC}"
    echo ""
}

# Function to get current server status
get_server_status() {
    print_step "Getting server status..."
    
    local instance_state=$(aws ec2 describe-instances \
        --instance-ids "$INSTANCE_ID" \
        --query 'Reservations[0].Instances[0].State.Name' \
        --output text \
        --region "$REGION" 2>/dev/null || echo "not-found")
    
    if [ "$instance_state" = "not-found" ]; then
        print_error "Instance $INSTANCE_ID not found"
        return 1
    fi
    
    local public_ip=$(aws ec2 describe-instances \
        --instance-ids "$INSTANCE_ID" \
        --query 'Reservations[0].Instances[0].PublicIpAddress' \
        --output text \
        --region "$REGION" 2>/dev/null || echo "None")
    
    echo "Instance ID: $INSTANCE_ID"
    echo "State: $instance_state"
    echo "Public IP: $public_ip"
    echo "Region: $REGION"
    
    return 0
}

# Function to start the server
start_server() {
    print_step "Starting EC2 instance..."
    
    aws ec2 start-instances --instance-ids "$INSTANCE_ID" --region "$REGION"
    
    print_step "Waiting for instance to start..."
    aws ec2 wait instance-running --instance-ids "$INSTANCE_ID" --region "$REGION"
    
    print_success "Instance started successfully"
    get_server_status
}

# Function to stop the server
stop_server() {
    print_step "Stopping EC2 instance..."
    
    aws ec2 stop-instances --instance-ids "$INSTANCE_ID" --region "$REGION"
    
    print_step "Waiting for instance to stop..."
    aws ec2 wait instance-stopped --instance-ids "$INSTANCE_ID" --region "$REGION"
    
    print_success "Instance stopped successfully"
}

# Function to restart the server
restart_server() {
    print_step "Restarting EC2 instance..."
    
    aws ec2 reboot-instances --instance-ids "$INSTANCE_ID" --region "$REGION"
    
    print_step "Waiting for instance to restart (60 seconds)..."
    sleep 60
    
    print_success "Instance restarted successfully"
    get_server_status
}

# Function to clean the server for fresh deployment
clean_server() {
    print_step "Cleaning server for fresh deployment..."
    
    local public_ip=$(aws ec2 describe-instances \
        --instance-ids "$INSTANCE_ID" \
        --query 'Reservations[0].Instances[0].PublicIpAddress' \
        --output text \
        --region "$REGION")
    
    if [ "$public_ip" = "None" ] || [ -z "$public_ip" ]; then
        print_error "Cannot get server IP address"
        return 1
    fi
    
    print_warning "This will stop all services and remove all deployment files!"
    read -p "Are you sure you want to clean the server? (y/N): " -n 1 -r
    echo
    
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        print_warning "Clean operation cancelled"
        return 0
    fi
    
    print_step "Stopping all Docker containers..."
    ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$public_ip" \
        "docker stop \$(docker ps -q) 2>/dev/null || true; docker rm \$(docker ps -aq) 2>/dev/null || true" 2>/dev/null || true
    
    print_step "Stopping all Podman containers..."
    ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$public_ip" \
        "podman stop \$(podman ps -q) 2>/dev/null || true; podman rm \$(podman ps -aq) 2>/dev/null || true" 2>/dev/null || true
    
    print_step "Killing Node.js processes..."
    ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$public_ip" \
        "pkill -f node 2>/dev/null || true; pkill -f npm 2>/dev/null || true" 2>/dev/null || true
    
    print_step "Removing deployment directory..."
    ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$public_ip" \
        "sudo rm -rf /opt/knirv-testnet" 2>/dev/null || true
    
    print_step "Creating fresh deployment directory..."
    ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$public_ip" \
        "sudo mkdir -p /opt/knirv-testnet && sudo chown ubuntu:ubuntu /opt/knirv-testnet" 2>/dev/null || true
    
    print_success "Server cleaned successfully - ready for fresh deployment"
}

# Function to show server logs
show_logs() {
    print_step "Fetching server logs..."
    
    local public_ip=$(aws ec2 describe-instances \
        --instance-ids "$INSTANCE_ID" \
        --query 'Reservations[0].Instances[0].PublicIpAddress' \
        --output text \
        --region "$REGION")
    
    if [ "$public_ip" = "None" ] || [ -z "$public_ip" ]; then
        print_error "Cannot get server IP address"
        return 1
    fi
    
    echo -e "${YELLOW}Docker containers:${NC}"
    ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$public_ip" "docker ps" 2>/dev/null || echo "No Docker containers or Docker not available"
    
    echo ""
    echo -e "${YELLOW}Podman containers:${NC}"
    ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$public_ip" "podman ps" 2>/dev/null || echo "No Podman containers or Podman not available"
    
    echo ""
    echo -e "${YELLOW}Node.js processes:${NC}"
    ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$public_ip" "ps aux | grep -E '(node|npm)' | grep -v grep" 2>/dev/null || echo "No Node.js processes running"
    
    echo ""
    echo -e "${YELLOW}Listening ports:${NC}"
    ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$public_ip" "netstat -tulpn | grep LISTEN" 2>/dev/null || echo "Cannot get port information"
}

# Function to connect to server
connect_server() {
    local public_ip=$(aws ec2 describe-instances \
        --instance-ids "$INSTANCE_ID" \
        --query 'Reservations[0].Instances[0].PublicIpAddress' \
        --output text \
        --region "$REGION")
    
    if [ "$public_ip" = "None" ] || [ -z "$public_ip" ]; then
        print_error "Cannot get server IP address"
        return 1
    fi
    
    print_step "Connecting to server at $public_ip..."
    ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$public_ip"
}

# Main execution
main() {
    print_header
    
    case "${1:-}" in
        status)
            get_server_status
            ;;
        start)
            start_server
            ;;
        stop)
            stop_server
            ;;
        restart)
            restart_server
            ;;
        clean)
            clean_server
            ;;
        logs)
            show_logs
            ;;
        connect|ssh)
            connect_server
            ;;
        --help|-h|help)
            echo "Usage: $0 <command>"
            echo ""
            echo "Commands:"
            echo "  status    Show current server status"
            echo "  start     Start the EC2 instance"
            echo "  stop      Stop the EC2 instance"
            echo "  restart   Restart the EC2 instance"
            echo "  clean     Clean server for fresh deployment (removes all files)"
            echo "  logs      Show server logs and running processes"
            echo "  connect   SSH into the server"
            echo "  help      Show this help message"
            echo ""
            echo "Examples:"
            echo "  $0 status                    # Check server status"
            echo "  $0 restart                  # Restart server"
            echo "  $0 clean                    # Clean for fresh deployment"
            echo "  $0 logs                     # View server status"
            ;;
        *)
            echo "Unknown command: ${1:-}"
            echo "Use '$0 help' for usage information"
            exit 1
            ;;
    esac
}

# Run main function
main "$@"
