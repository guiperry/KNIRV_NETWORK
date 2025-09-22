#!/bin/bash

# KNIRV Testnet Health Check Script
# Dynamically discovers and checks services on AWS EC2 testnet instance
# Uses remote health check script for accurate service discovery

# Configuration
INSTANCE_ID="i-06813be8a8a23ea5b"
REGION="us-east-2"
TESTNET_IP=""  # Will be discovered dynamically
SSH_KEY="~/.ssh/AEGONG.pem"
REMOTE_SCRIPT="/tmp/remote-health-check.sh"
LOCAL_SCRIPT="scripts/remote-health-check.sh"

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to discover testnet IP address
discover_testnet_ip() {
    echo -e "${YELLOW}🔍 Discovering testnet IP address...${NC}"

    # Check if instance exists and get its state
    local instance_state=$(aws ec2 describe-instances \
        --instance-ids "$INSTANCE_ID" \
        --query 'Reservations[0].Instances[0].State.Name' \
        --output text \
        --region "$REGION" 2>/dev/null || echo "not-found")

    if [ "$instance_state" = "not-found" ]; then
        echo -e "${RED}❌ Instance $INSTANCE_ID not found${NC}"
        exit 1
    fi

    if [ "$instance_state" != "running" ]; then
        echo -e "${RED}❌ Instance $INSTANCE_ID is in state: $instance_state${NC}"
        echo -e "${YELLOW}💡 You can start it with: aws ec2 start-instances --instance-ids $INSTANCE_ID${NC}"
        exit 1
    fi

    # Get the current public IP address
    TESTNET_IP=$(aws ec2 describe-instances \
        --instance-ids "$INSTANCE_ID" \
        --query 'Reservations[0].Instances[0].PublicIpAddress' \
        --output text \
        --region "$REGION")

    if [ "$TESTNET_IP" = "None" ] || [ -z "$TESTNET_IP" ]; then
        echo -e "${RED}❌ Instance $INSTANCE_ID does not have a public IP address${NC}"
        exit 1
    fi

    echo -e "${GREEN}✅ Discovered testnet IP: $TESTNET_IP${NC}"
}

echo -e "${BLUE}🔍 KNIRV Testnet Health Check${NC}"
echo -e "${BLUE}================================${NC}"

# Function to deploy remote health check script
deploy_remote_script() {
    echo -e "${YELLOW}📤 Deploying remote health check script...${NC}"

    if [ ! -f "$LOCAL_SCRIPT" ]; then
        echo -e "${RED}❌ Local script not found: $LOCAL_SCRIPT${NC}"
        exit 1
    fi

    # Upload the script to the remote server
    if scp -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no "$LOCAL_SCRIPT" ubuntu@"$TESTNET_IP":"$REMOTE_SCRIPT" >/dev/null 2>&1; then
        # Make it executable
        ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$TESTNET_IP" "chmod +x $REMOTE_SCRIPT" >/dev/null 2>&1
        echo -e "${GREEN}✅ Remote script deployed successfully${NC}"
        return 0
    else
        echo -e "${RED}❌ Failed to deploy remote script${NC}"
        return 1
    fi
}

# Function to run remote health check
run_remote_health_check() {
    local format=${1:-"table"}

    echo -e "${YELLOW}🔍 Running dynamic service discovery on remote server...${NC}"

    # Run the remote health check script
    ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$TESTNET_IP" "$REMOTE_SCRIPT --$format" 2>/dev/null

    if [ $? -eq 0 ]; then
        echo ""
        echo -e "${GREEN}✅ Remote health check completed successfully${NC}"
        return 0
    else
        echo -e "${RED}❌ Remote health check failed${NC}"
        return 1
    fi
}

# Test CloudFlare endpoints (from KNIRVTESTNET/config/endpoints.yaml)
test_cloudflare_endpoints() {
    echo -e "${BLUE}📡 Testing CloudFlare Endpoints (Production URLs):${NC}"

    local cloudflare_endpoints=(
        "KNIRVCHAIN:https://chain-testnet.knirv.network:/health"
        "KNIRVGRAPH:https://graph-testnet.knirv.network:/health"
        "KNIRVNEXUS:https://nexus-testnet.knirv.network:/health"
        "KNIRVNEXUS_GUI:https://nexus-gui-testnet.knirv.network:/health"
        "KNIRVNEXUS_DVE:https://nexus-dve-testnet.knirv.network:/health"
        "KNIRVORACLE:https://oracle-testnet.knirv.network:/health"
        "KNIRVROUTER:https://router-testnet.knirv.network:/health"
        "KNIRVANA:https://knirvana-api-testnet.knirv.network:/health"
        "KNIRVANA_GRAPH:https://interactive-graph-testnet.knirv.network:/"
        "KNIRVANALYTICS:https://analytics-testnet.knirv.network:/health"
    )

    for service_info in "${cloudflare_endpoints[@]}"; do
        IFS=':' read -r service_name url endpoint <<< "$service_info"

        echo -n "  🔍 $service_name: "

        # Test the CloudFlare endpoint
        if curl -s --connect-timeout 5 --max-time 10 "$url$endpoint" >/dev/null 2>&1; then
            echo -e "${GREEN}✅ HEALTHY${NC}"
        else
            echo -e "${RED}❌ UNHEALTHY${NC}"
        fi
    done
}

# Main execution
main() {
    # Parse command line arguments
    local show_cloudflare=true
    local output_format="table"
    local restart_server=false

    while [[ $# -gt 0 ]]; do
        case $1 in
            --no-cloudflare)
                show_cloudflare=false
                shift
                ;;
            --json)
                output_format="json"
                shift
                ;;
            --simple)
                output_format="simple"
                shift
                ;;
            --restart)
                restart_server=true
                shift
                ;;
            --help|-h)
                echo "Usage: $0 [--no-cloudflare] [--json|--simple] [--restart]"
                echo ""
                echo "Options:"
                echo "  --no-cloudflare  Skip CloudFlare endpoint testing"
                echo "  --json          Output remote results in JSON format"
                echo "  --simple        Output remote results in simple format"
                echo "  --restart       Restart the EC2 instance before health check"
                echo "  --help, -h      Show this help message"
                echo ""
                echo "This script:"
                echo "  1. Discovers the current testnet IP address using AWS CLI"
                echo "  2. Optionally restarts the EC2 instance"
                echo "  3. Deploys a dynamic health check script to the remote server"
                echo "  4. Discovers what services are actually running"
                echo "  5. Tests each discovered service's health endpoint"
                echo "  6. Optionally tests CloudFlare production endpoints"
                exit 0
                ;;
            *)
                echo "Unknown option: $1"
                echo "Use --help for usage information"
                exit 1
                ;;
        esac
    done

    # Discover testnet IP address
    discover_testnet_ip

    echo "Testing services on: $TESTNET_IP"
    echo "Using dynamic service discovery..."
    echo ""

    # Restart server if requested
    if [ "$restart_server" = true ]; then
        echo -e "${YELLOW}🔄 Restarting EC2 instance...${NC}"
        aws ec2 reboot-instances --instance-ids "$INSTANCE_ID" --region "$REGION"
        echo -e "${YELLOW}⏳ Waiting for instance to restart (60 seconds)...${NC}"
        sleep 60

        # Re-discover IP after restart (might change)
        discover_testnet_ip
        echo ""
    fi

    # Deploy and run remote health check
    if deploy_remote_script; then
        echo ""
        echo -e "${BLUE}🖥️  Dynamic Service Discovery Results:${NC}"
        run_remote_health_check "$output_format"
    else
        echo -e "${RED}❌ Could not deploy remote health check script${NC}"
        exit 1
    fi

    # Test CloudFlare endpoints if requested
    if [ "$show_cloudflare" = true ]; then
        echo ""
        test_cloudflare_endpoints
    fi

    echo ""
    echo -e "${BLUE}📊 Additional Information:${NC}"
    echo -e "  🌐 Network Monitor: http://$TESTNET_IP:1317/monitor (if KNIRVORACLE is running)"
    echo -e "  📋 Container Status: ssh ubuntu@$TESTNET_IP 'docker ps'"
    echo -e "  📝 Service Logs: ssh ubuntu@$TESTNET_IP 'docker logs <container_name>'"
    echo -e "  🔄 Re-run Health Check: $0"
}

# Run main function with all arguments
main "$@"
