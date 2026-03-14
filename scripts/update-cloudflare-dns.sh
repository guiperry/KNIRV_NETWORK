#!/bin/bash

# KNIRV CloudFlare DNS Update Script
# Updates CloudFlare DNS records only for healthy services

# Configuration
ZONE_NAME="knirv.network"
TESTNET_SUBDOMAIN_PREFIX="testnet"

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
    echo -e "${BLUE}🌐 KNIRV CloudFlare DNS Update${NC}"
    echo -e "${BLUE}==============================${NC}"
    echo ""
}

# Function to check if CloudFlare credentials are available
check_cloudflare_credentials() {
    if [ -z "$CLOUDFLARE_API_TOKEN" ] && [ -z "$CLOUDFLARE_EMAIL" ]; then
        print_error "CloudFlare credentials not found"
        print_warning "Please set either:"
        print_warning "  CLOUDFLARE_API_TOKEN (recommended)"
        print_warning "  or CLOUDFLARE_EMAIL + CLOUDFLARE_API_KEY"
        return 1
    fi
    
    if [ -n "$CLOUDFLARE_API_TOKEN" ]; then
        print_success "Using CloudFlare API Token authentication"
        return 0
    elif [ -n "$CLOUDFLARE_EMAIL" ] && [ -n "$CLOUDFLARE_API_KEY" ]; then
        print_success "Using CloudFlare Email + API Key authentication"
        return 0
    else
        print_error "Incomplete CloudFlare credentials"
        return 1
    fi
}

# Function to get zone ID
get_zone_id() {
    local zone_name=$1
    
    if [ -n "$CLOUDFLARE_API_TOKEN" ]; then
        curl -s -X GET "https://api.cloudflare.com/client/v4/zones?name=$zone_name" \
            -H "Authorization: Bearer $CLOUDFLARE_API_TOKEN" \
            -H "Content-Type: application/json" | \
            jq -r '.result[0].id // empty'
    else
        curl -s -X GET "https://api.cloudflare.com/client/v4/zones?name=$zone_name" \
            -H "X-Auth-Email: $CLOUDFLARE_EMAIL" \
            -H "X-Auth-Key: $CLOUDFLARE_API_KEY" \
            -H "Content-Type: application/json" | \
            jq -r '.result[0].id // empty'
    fi
}

# Function to get existing DNS record
get_dns_record() {
    local zone_id=$1
    local record_name=$2
    
    if [ -n "$CLOUDFLARE_API_TOKEN" ]; then
        curl -s -X GET "https://api.cloudflare.com/client/v4/zones/$zone_id/dns_records?name=$record_name" \
            -H "Authorization: Bearer $CLOUDFLARE_API_TOKEN" \
            -H "Content-Type: application/json" | \
            jq -r '.result[0].id // empty'
    else
        curl -s -X GET "https://api.cloudflare.com/client/v4/zones/$zone_id/dns_records?name=$record_name" \
            -H "X-Auth-Email: $CLOUDFLARE_EMAIL" \
            -H "X-Auth-Key: $CLOUDFLARE_API_KEY" \
            -H "Content-Type: application/json" | \
            jq -r '.result[0].id // empty'
    fi
}

# Function to create or update DNS record
update_dns_record() {
    local zone_id=$1
    local record_name=$2
    local record_type=$3
    local record_content=$4
    local record_ttl=${5:-300}
    
    local record_id=$(get_dns_record "$zone_id" "$record_name")
    
    local json_data=$(jq -n \
        --arg type "$record_type" \
        --arg name "$record_name" \
        --arg content "$record_content" \
        --argjson ttl $record_ttl \
        '{type: $type, name: $name, content: $content, ttl: $ttl}')
    
    if [ -n "$record_id" ] && [ "$record_id" != "null" ]; then
        # Update existing record
        if [ -n "$CLOUDFLARE_API_TOKEN" ]; then
            local response=$(curl -s -X PUT "https://api.cloudflare.com/client/v4/zones/$zone_id/dns_records/$record_id" \
                -H "Authorization: Bearer $CLOUDFLARE_API_TOKEN" \
                -H "Content-Type: application/json" \
                -d "$json_data")
        else
            local response=$(curl -s -X PUT "https://api.cloudflare.com/client/v4/zones/$zone_id/dns_records/$record_id" \
                -H "X-Auth-Email: $CLOUDFLARE_EMAIL" \
                -H "X-Auth-Key: $CLOUDFLARE_API_KEY" \
                -H "Content-Type: application/json" \
                -d "$json_data")
        fi
        
        local success=$(echo "$response" | jq -r '.success')
        if [ "$success" = "true" ]; then
            print_success "Updated DNS record: $record_name -> $record_content"
        else
            print_error "Failed to update DNS record: $record_name"
            echo "$response" | jq -r '.errors[]?.message // "Unknown error"'
        fi
    else
        # Create new record
        if [ -n "$CLOUDFLARE_API_TOKEN" ]; then
            local response=$(curl -s -X POST "https://api.cloudflare.com/client/v4/zones/$zone_id/dns_records" \
                -H "Authorization: Bearer $CLOUDFLARE_API_TOKEN" \
                -H "Content-Type: application/json" \
                -d "$json_data")
        else
            local response=$(curl -s -X POST "https://api.cloudflare.com/client/v4/zones/$zone_id/dns_records" \
                -H "X-Auth-Email: $CLOUDFLARE_EMAIL" \
                -H "X-Auth-Key: $CLOUDFLARE_API_KEY" \
                -H "Content-Type: application/json" \
                -d "$json_data")
        fi
        
        local success=$(echo "$response" | jq -r '.success')
        if [ "$success" = "true" ]; then
            print_success "Created DNS record: $record_name -> $record_content"
        else
            print_error "Failed to create DNS record: $record_name"
            echo "$response" | jq -r '.errors[]?.message // "Unknown error"'
        fi
    fi
}

# Function to test service health
test_service_health() {
    local ip=$1
    local port=$2
    local endpoint=${3:-"/health"}
    
    curl -s --connect-timeout 5 --max-time 10 "http://$ip:$port$endpoint" >/dev/null 2>&1
    return $?
}

# Function to update DNS for healthy services
update_healthy_services() {
    local testnet_ip=$1
    
    if [ -z "$testnet_ip" ]; then
        print_error "Testnet IP address not provided"
        return 1
    fi
    
    print_step "Getting CloudFlare zone ID for $ZONE_NAME..."
    local zone_id=$(get_zone_id "$ZONE_NAME")
    
    if [ -z "$zone_id" ] || [ "$zone_id" = "null" ]; then
        print_error "Could not get zone ID for $ZONE_NAME"
        return 1
    fi
    
    print_success "Zone ID: $zone_id"
    
    # Define services to check and update
    declare -A services=(
        ["oracle-$TESTNET_SUBDOMAIN_PREFIX"]="1317:/health"
        ["chain-$TESTNET_SUBDOMAIN_PREFIX"]="8090:/health"
        ["graph-$TESTNET_SUBDOMAIN_PREFIX"]="8082:/health"
        ["nexus-$TESTNET_SUBDOMAIN_PREFIX"]="8084:/health"
        ["nexus-gui-$TESTNET_SUBDOMAIN_PREFIX"]="8085:/health"
        ["router-$TESTNET_SUBDOMAIN_PREFIX"]="8086:/status"
        ["gateway-$TESTNET_SUBDOMAIN_PREFIX"]="10000:/health"
        ["knirvana-$TESTNET_SUBDOMAIN_PREFIX"]="3000:/health"
        ["xion-bridge-$TESTNET_SUBDOMAIN_PREFIX"]="8088:/health"
        ["ipfs-api-$TESTNET_SUBDOMAIN_PREFIX"]="5001:/api/v0/version"
        ["ipfs-gateway-$TESTNET_SUBDOMAIN_PREFIX"]="8081:/"
    )
    
    print_step "Testing service health and updating DNS records..."
    
    local healthy_count=0
    local total_count=${#services[@]}
    
    for service_name in "${!services[@]}"; do
        local port_endpoint="${services[$service_name]}"
        local port=$(echo "$port_endpoint" | cut -d: -f1)
        local endpoint=$(echo "$port_endpoint" | cut -d: -f2)
        
        echo -n "  Testing $service_name (port $port): "
        
        if test_service_health "$testnet_ip" "$port" "$endpoint"; then
            echo -e "${GREEN}HEALTHY${NC}"
            update_dns_record "$zone_id" "$service_name.$ZONE_NAME" "A" "$testnet_ip" 300
            ((healthy_count++))
        else
            echo -e "${RED}UNHEALTHY${NC}"
            print_warning "Skipping DNS update for $service_name (service not responding)"
        fi
    done
    
    echo ""
    print_success "DNS update completed: $healthy_count/$total_count services healthy"
    
    if [ $healthy_count -gt 0 ]; then
        print_step "Updated DNS records:"
        for service_name in "${!services[@]}"; do
            local port_endpoint="${services[$service_name]}"
            local port=$(echo "$port_endpoint" | cut -d: -f1)
            local endpoint=$(echo "$port_endpoint" | cut -d: -f2)
            
            if test_service_health "$testnet_ip" "$port" "$endpoint"; then
                echo "  ✅ $service_name.$ZONE_NAME -> $testnet_ip"
            fi
        done
    fi
}

# Main execution
main() {
    print_header
    
    case "${1:-}" in
        update)
            local testnet_ip="$2"
            if [ -z "$testnet_ip" ]; then
                print_error "Usage: $0 update <testnet_ip>"
                exit 1
            fi
            
            if ! check_cloudflare_credentials; then
                exit 1
            fi
            
            update_healthy_services "$testnet_ip"
            ;;
        test)
            local testnet_ip="$2"
            if [ -z "$testnet_ip" ]; then
                print_error "Usage: $0 test <testnet_ip>"
                exit 1
            fi
            
            print_step "Testing service health on $testnet_ip..."
            
            declare -A test_services=(
                ["KNIRVORACLE"]="1317:/health"
                ["KNIRVCHAIN"]="8090:/health"
                ["KNIRVGRAPH"]="8082:/health"
                ["KNIRVSERVER"]="8084:/health"
                ["KNIRVROUTER"]="8086:/status"
                ["GATEWAY"]="10000:/health"
                ["KNIRVANA"]="3000:/health"
                ["XION-BRIDGE"]="8088:/health"
                ["IPFS-API"]="5001:/api/v0/version"
                ["IPFS-GATEWAY"]="8081:/"
            )
            
            for service_name in "${!test_services[@]}"; do
                local port_endpoint="${test_services[$service_name]}"
                local port=$(echo "$port_endpoint" | cut -d: -f1)
                local endpoint=$(echo "$port_endpoint" | cut -d: -f2)
                
                echo -n "  $service_name (port $port): "
                
                if test_service_health "$testnet_ip" "$port" "$endpoint"; then
                    echo -e "${GREEN}HEALTHY${NC}"
                else
                    echo -e "${RED}UNHEALTHY${NC}"
                fi
            done
            ;;
        --help|-h|help)
            echo "Usage: $0 <command> [options]"
            echo ""
            echo "Commands:"
            echo "  update <ip>    Update CloudFlare DNS records for healthy services"
            echo "  test <ip>      Test service health without updating DNS"
            echo "  help           Show this help message"
            echo ""
            echo "Environment Variables:"
            echo "  CLOUDFLARE_API_TOKEN    CloudFlare API token (recommended)"
            echo "  CLOUDFLARE_EMAIL        CloudFlare account email"
            echo "  CLOUDFLARE_API_KEY      CloudFlare API key"
            echo ""
            echo "Examples:"
            echo "  $0 test 18.189.74.120"
            echo "  $0 update 18.189.74.120"
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
