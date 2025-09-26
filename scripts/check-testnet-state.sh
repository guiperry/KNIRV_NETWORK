#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
ANSIBLE_DIR="$PROJECT_ROOT/deployment/ansible"
DEPLOYMENT_LOG="$ANSIBLE_DIR/deployment_session.log"

print_header() {
    echo -e "${BLUE}=================================${NC}"
    echo -e "${BLUE}🧪 KNIRV Testnet Status Checker${NC}"
    echo -e "${BLUE}=================================${NC}"
}

get_checkpoint() {
    if [[ -f "$DEPLOYMENT_LOG" ]]; then
        local last_entry=$(tail -n 1 "$DEPLOYMENT_LOG" 2>/dev/null)
        echo "$last_entry" | cut -d'-' -f2- | xargs
    else
        echo "No deployment log found"
    fi
}

check_services() {
    local ip=$(cat "$ANSIBLE_DIR/testnet_ip.txt" 2>/dev/null)
    if [ -z "$ip" ]; then
        echo -e "${RED}❌ No testnet IP found${NC}"
        return 1
    fi
    
    echo -e "${YELLOW}Checking service statuses...${NC}"
    ssh -i ~/.ssh/AEGONG.pem -o StrictHostKeyChecking=no ubuntu@"$ip" \
        "sudo netstat -tulpn | grep -E ':(1317|8090|8082|8084|8085|8086|10000|3000|8088|5001|8081)'"
}

main() {
    print_header
    
    local checkpoint=$(get_checkpoint)
    echo -e "${YELLOW}Last Deployment Checkpoint:${NC}"
    echo -e "  ${GREEN}$checkpoint${NC}"
    
    echo -e "\n${YELLOW}Service Port Status:${NC}"
    check_services
    
    echo -e "\n${YELLOW}Resume Options:${NC}"
    echo "1. To resume deployment:"
    echo "   make deploy-testnet-services RESUME=true"
    echo "2. To start fresh deployment:"
    echo "   make deploy-testnet-services CLEAN=true"
}

main
