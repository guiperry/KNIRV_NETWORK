#!/bin/bash

# Remote KNIRV Testnet Health Check Script
# Dynamically discovers and checks services running on AWS EC2 testnet instance
# This script runs ON the remote server to check local services

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
TIMEOUT=5
OUTPUT_FORMAT="json"  # json, table, or simple
HEALTH_CHECK_FILE="/tmp/knirv-health-check.json"

print_header() {
    echo -e "${BLUE}🔍 KNIRV Testnet Dynamic Health Check${NC}"
    echo -e "${BLUE}=====================================${NC}"
    echo "Timestamp: $(date)"
    echo "Hostname: $(hostname)"
    echo "IP Address: $(hostname -I | awk '{print $1}')"
    echo ""
}

# Function to discover running services on specific ports
discover_services() {
    # Get all listening ports (redirect status messages to stderr)
    echo "🔍 Discovering running services..." >&2

    local listening_ports=$(netstat -tulpn 2>/dev/null | grep LISTEN | awk '{print $4}' | sed 's/.*://' | sort -n | uniq)
    
    # Known KNIRV service ports and their expected services
    declare -A known_services=(
        ["22"]="SSH"
        ["80"]="HTTP"
        ["443"]="HTTPS"
        ["1317"]="KNIRVORACLE"
        ["5001"]="IPFS-API"
        ["8080"]="IPFS-Gateway"
        ["8081"]="IPFS-Gateway-Alt"
        ["8082"]="KNIRVGRAPH"
        ["8084"]="KNIRVNEXUS-1"
        ["8085"]="KNIRVNEXUS-2"
        ["8086"]="KNIRVROUTER"
        ["8090"]="KNIRVCHAIN"
        ["10000"]="TESTNET-GATEWAY"
        ["3000"]="Node-App"
        ["3001"]="Node-App-Alt"
        ["4001"]="IPFS-Swarm"
        ["26656"]="Tendermint-P2P"
        ["26657"]="Tendermint-RPC"
    )
    
    # Discover services
    local discovered_services=()
    
    for port in $listening_ports; do
        local service_name="${known_services[$port]:-Unknown-$port}"
        local process_info=$(netstat -tulpn 2>/dev/null | grep ":$port " | head -1)
        local pid=$(echo "$process_info" | awk '{print $7}' | cut -d'/' -f1)
        local process_name=$(echo "$process_info" | awk '{print $7}' | cut -d'/' -f2)
        
        # Try to get more info about the process (clean up command line)
        if [ -n "$pid" ] && [ "$pid" != "-" ]; then
            local cmd_line=$(ps -p "$pid" -o comm --no-headers 2>/dev/null | tr -d '\n' | head -c 20)
            # Use | as delimiter to avoid issues with spaces and colons
            discovered_services+=("$port|$service_name|$process_name|$pid|$cmd_line")
        else
            discovered_services+=("$port|$service_name|$process_name|-|unknown")
        fi
    done
    
    echo "${discovered_services[@]}"
}

# Function to check if a service is healthy
check_service_health() {
    local port=$1
    local service_name=$2
    local process_name=$3
    local pid=$4
    
    local status="UNKNOWN"
    local response_time=""
    local details=""
    local health_endpoint=""
    
    # Determine health endpoint based on service
    case $service_name in
        "KNIRVORACLE")
            health_endpoint="/health"
            ;;
        "KNIRVCHAIN")
            health_endpoint="/health"
            ;;
        "KNIRVGRAPH")
            health_endpoint="/health"
            ;;
        "KNIRVNEXUS"*)
            health_endpoint="/health"
            ;;
        "KNIRVROUTER")
            health_endpoint="/status"
            ;;
        "IPFS-API")
            health_endpoint="/api/v0/version"
            ;;
        "IPFS-Gateway"*)
            health_endpoint="/"
            ;;
        "TESTNET-GATEWAY")
            health_endpoint="/health"
            ;;
        *)
            health_endpoint="/health"
            ;;
    esac
    
    # Check if process is running
    if [ "$pid" != "-" ] && [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
        status="RUNNING"
        
        # Check HTTP endpoint
        local start_time=$(date +%s%N)
        if curl -s --connect-timeout $TIMEOUT --max-time $TIMEOUT "http://localhost:$port$health_endpoint" >/dev/null 2>&1; then
            local end_time=$(date +%s%N)
            response_time=$(( (end_time - start_time) / 1000000 ))
            status="HEALTHY"
            
            # Get additional details for known services
            case $service_name in
                "IPFS-API")
                    details=$(curl -s --max-time 2 "http://localhost:$port/api/v0/version" 2>/dev/null | jq -r '.Version // "unknown"' 2>/dev/null || echo "ipfs")
                    ;;
                "KNIRVORACLE"|"KNIRVCHAIN"|"KNIRVGRAPH"|"KNIRVNEXUS"*|"KNIRVROUTER")
                    details=$(curl -s --max-time 2 "http://localhost:$port$health_endpoint" 2>/dev/null | jq -r '.status // .message // "healthy"' 2>/dev/null || echo "service")
                    ;;
            esac
        else
            status="UNHEALTHY"
        fi
    else
        status="STOPPED"
    fi
    
    echo "$status|$response_time|$details"
}

# Function to generate JSON output
generate_json_output() {
    local services=("$@")
    
    echo "{"
    echo "  \"timestamp\": \"$(date -Iseconds)\","
    echo "  \"hostname\": \"$(hostname)\","
    echo "  \"ip_address\": \"$(hostname -I | awk '{print $1}')\","
    echo "  \"services\": ["
    
    local first=true
    for service_info in "${services[@]}"; do
        IFS='|' read -r port service_name process_name pid cmd_line <<< "$service_info"
        
        local health_result=$(check_service_health "$port" "$service_name" "$process_name" "$pid")
        IFS='|' read -r status response_time details <<< "$health_result"
        
        if [ "$first" = true ]; then
            first=false
        else
            echo ","
        fi
        
        echo "    {"
        echo "      \"port\": $port,"
        echo "      \"service_name\": \"$service_name\","
        echo "      \"process_name\": \"$process_name\","
        echo "      \"pid\": \"$pid\","
        echo "      \"status\": \"$status\","
        echo "      \"response_time_ms\": \"$response_time\","
        echo "      \"details\": \"$details\","
        echo "      \"command_line\": \"$cmd_line\""
        echo -n "    }"
    done
    
    echo ""
    echo "  ]"
    echo "}"
}

# Function to generate table output
generate_table_output() {
    local services=("$@")
    
    printf "%-6s %-20s %-15s %-8s %-10s %-8s %-20s\n" "PORT" "SERVICE" "PROCESS" "PID" "STATUS" "TIME(ms)" "DETAILS"
    printf "%-6s %-20s %-15s %-8s %-10s %-8s %-20s\n" "----" "-------" "-------" "---" "------" "--------" "-------"
    
    for service_info in "${services[@]}"; do
        IFS='|' read -r port service_name process_name pid cmd_line <<< "$service_info"
        
        local health_result=$(check_service_health "$port" "$service_name" "$process_name" "$pid")
        IFS='|' read -r status response_time details <<< "$health_result"
        
        # Color code status
        local status_colored=""
        case $status in
            "HEALTHY")
                status_colored="${GREEN}$status${NC}"
                ;;
            "RUNNING")
                status_colored="${YELLOW}$status${NC}"
                ;;
            "UNHEALTHY")
                status_colored="${RED}$status${NC}"
                ;;
            "STOPPED")
                status_colored="${RED}$status${NC}"
                ;;
            *)
                status_colored="$status"
                ;;
        esac
        
        printf "%-6s %-20s %-15s %-8s %-10b %-8s %-20s\n" \
            "$port" "$service_name" "$process_name" "$pid" "$status_colored" "$response_time" "$details"
    done
}

# Main execution
main() {
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --json)
                OUTPUT_FORMAT="json"
                shift
                ;;
            --table)
                OUTPUT_FORMAT="table"
                shift
                ;;
            --simple)
                OUTPUT_FORMAT="simple"
                shift
                ;;
            --timeout)
                TIMEOUT="$2"
                shift 2
                ;;
            --help|-h)
                echo "Usage: $0 [--json|--table|--simple] [--timeout SECONDS]"
                echo ""
                echo "Options:"
                echo "  --json     Output in JSON format (default)"
                echo "  --table    Output in table format"
                echo "  --simple   Output in simple format"
                echo "  --timeout  HTTP timeout in seconds (default: 5)"
                exit 0
                ;;
            *)
                echo "Unknown option: $1"
                exit 1
                ;;
        esac
    done
    
    # Discover services
    local discovered_services=($(discover_services))
    
    # Generate output based on format
    case $OUTPUT_FORMAT in
        "json")
            generate_json_output "${discovered_services[@]}" | tee "$HEALTH_CHECK_FILE"
            ;;
        "table")
            print_header
            generate_table_output "${discovered_services[@]}"
            ;;
        "simple")
            print_header
            echo -e "${YELLOW}Service Status:${NC}"
            for service_info in "${discovered_services[@]}"; do
                IFS='|' read -r port service_name process_name pid cmd_line <<< "$service_info"
                local health_result=$(check_service_health "$port" "$service_name" "$process_name" "$pid")
                IFS='|' read -r status response_time details <<< "$health_result"
                
                case $status in
                    "HEALTHY")
                        echo -e "  ${GREEN}✅ $service_name ($port): $status${NC}"
                        ;;
                    "RUNNING")
                        echo -e "  ${YELLOW}⚠️  $service_name ($port): $status${NC}"
                        ;;
                    *)
                        echo -e "  ${RED}❌ $service_name ($port): $status${NC}"
                        ;;
                esac
            done
            ;;
    esac
}

# Run main function
main "$@"
