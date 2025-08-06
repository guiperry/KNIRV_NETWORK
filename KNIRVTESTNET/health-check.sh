#!/bin/bash

# KNIRV Testnet Health Check Script
# Monitors all testnet services and reports their status

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
WATCH_MODE=false
WATCH_INTERVAL=10
DETAILED=false

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -w|--watch)
            WATCH_MODE=true
            shift
            ;;
        -i|--interval)
            WATCH_INTERVAL="$2"
            shift 2
            ;;
        -d|--detailed)
            DETAILED=true
            shift
            ;;
        -t|--timeout)
            TIMEOUT="$2"
            shift 2
            ;;
        -h|--help)
            echo "KNIRV Testnet Health Check"
            echo ""
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  -w, --watch           Watch mode - continuously monitor services"
            echo "  -i, --interval SEC    Watch interval in seconds (default: 10)"
            echo "  -d, --detailed        Show detailed service information"
            echo "  -t, --timeout SEC     HTTP timeout in seconds (default: 5)"
            echo "  -h, --help           Show this help message"
            echo ""
            echo "Examples:"
            echo "  $0                    Single health check"
            echo "  $0 --watch           Continuous monitoring"
            echo "  $0 -w -i 5           Watch with 5-second intervals"
            echo "  $0 --detailed        Detailed service information"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[✓]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[⚠]${NC} $1"
}

print_error() {
    echo -e "${RED}[✗]${NC} $1"
}

print_header() {
    echo -e "${PURPLE}$1${NC}"
}

# Function to check HTTP endpoint
check_http_endpoint() {
    local url=$1
    local timeout=${2:-$TIMEOUT}
    
    if curl -s -f --max-time "$timeout" "$url" >/dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

# Function to check if process is running
check_process() {
    local pid_file=$1
    
    if [ -f "$pid_file" ]; then
        local pid=$(cat "$pid_file" 2>/dev/null)
        if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
            echo "$pid"
            return 0
        fi
    fi
    return 1
}

# Function to get service info
get_service_info() {
    local name=$1
    local url=$2
    local pid_file=$3
    local port=$4
    
    local status="UNKNOWN"
    local pid=""
    local response_time=""
    local details=""
    
    # Check if process is running
    if pid=$(check_process "$pid_file"); then
        status="RUNNING"
        
        # Check HTTP endpoint
        if [ -n "$url" ]; then
            local start_time=$(date +%s%N)
            if check_http_endpoint "$url"; then
                local end_time=$(date +%s%N)
                response_time=$(( (end_time - start_time) / 1000000 ))
                status="HEALTHY"
            else
                status="UNHEALTHY"
            fi
        fi
    else
        status="STOPPED"
    fi
    
    # Get additional details if requested
    if [ "$DETAILED" = true ] && [ "$status" = "HEALTHY" ]; then
        case $name in
            "KNIRV-ROOT")
                details=$(curl -s --max-time 2 "$url" 2>/dev/null | jq -r '.mode // "unknown"' 2>/dev/null || echo "")
                ;;
            "KNIRVCHAIN")
                details=$(curl -s --max-time 2 "$url" 2>/dev/null | jq -r '.mode // "unknown"' 2>/dev/null || echo "")
                ;;
            "KNIRVGRAPH")
                details=$(curl -s --max-time 2 "$url" 2>/dev/null | jq -r '.mode // "unknown"' 2>/dev/null || echo "")
                ;;
            "KNIRV-NEXUS")
                details=$(curl -s --max-time 2 "$url" 2>/dev/null | jq -r '.mode // "unknown"' 2>/dev/null || echo "")
                ;;
            "KNIRV-ROUTER")
                details=$(curl -s --max-time 2 "http://localhost:$port/health" 2>/dev/null | jq -r '.status // "unknown"' 2>/dev/null || echo "")
                ;;
            "KNIRV-GATEWAY")
                details=$(curl -s --max-time 2 "$url" 2>/dev/null | jq -r '.mode // "unknown"' 2>/dev/null || echo "")
                ;;
        esac
    fi
    
    echo "$status|$pid|$response_time|$details"
}

# Function to display service status
display_service_status() {
    local name=$1
    local url=$2
    local pid_file=$3
    local port=$4
    
    local info=$(get_service_info "$name" "$url" "$pid_file" "$port")
    local status=$(echo "$info" | cut -d'|' -f1)
    local pid=$(echo "$info" | cut -d'|' -f2)
    local response_time=$(echo "$info" | cut -d'|' -f3)
    local details=$(echo "$info" | cut -d'|' -f4)
    
    local status_icon=""
    local status_color=""
    
    case $status in
        "HEALTHY")
            status_icon="✓"
            status_color="${GREEN}"
            ;;
        "RUNNING")
            status_icon="⚠"
            status_color="${YELLOW}"
            ;;
        "UNHEALTHY")
            status_icon="✗"
            status_color="${RED}"
            ;;
        "STOPPED")
            status_icon="○"
            status_color="${RED}"
            ;;
        *)
            status_icon="?"
            status_color="${YELLOW}"
            ;;
    esac
    
    printf "  ${status_color}${status_icon}${NC} %-15s %-10s" "$name" "$status"
    
    if [ -n "$pid" ]; then
        printf " PID:%-6s" "$pid"
    else
        printf " %-10s" ""
    fi
    
    if [ -n "$response_time" ]; then
        printf " %4sms" "$response_time"
    else
        printf " %6s" ""
    fi
    
    if [ "$DETAILED" = true ] && [ -n "$details" ]; then
        printf " (%s)" "$details"
    fi
    
    printf " %s\n" "$url"
}

# Function to perform health check
perform_health_check() {
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    
    if [ "$WATCH_MODE" = true ]; then
        clear
    fi
    
    print_header "🏥 KNIRV TESTNET HEALTH CHECK"
    print_header "============================="
    echo "Timestamp: $timestamp"
    echo ""
    
    # Service definitions
    declare -a services=(
        "KNIRV-ROOT|http://localhost:1317/health|data/knirvroot.pid|1317"
        "KNIRVCHAIN|http://localhost:8080/health|data/knirvchain.pid|8080"
        "KNIRVGRAPH|http://localhost:8081/health|data/knirvgraph.pid|8081"
        "KNIRV-NEXUS|http://localhost:8082/health|data/knirvnexus.pid|8082"
        "KNIRV-ROUTER|http://localhost:5001/health|data/knirvrouter.pid|5001"
        "KNIRV-GATEWAY|http://localhost:8888/gateway/health|data/knirvgateway.pid|8888"
    )
    
    local healthy_count=0
    local total_count=${#services[@]}
    
    echo "Service Status:"
    for service_def in "${services[@]}"; do
        IFS='|' read -r name url pid_file port <<< "$service_def"
        display_service_status "$name" "$url" "$pid_file" "$port"
        
        local info=$(get_service_info "$name" "$url" "$pid_file" "$port")
        local status=$(echo "$info" | cut -d'|' -f1)
        if [ "$status" = "HEALTHY" ]; then
            healthy_count=$((healthy_count + 1))
        fi
    done
    
    echo ""
    echo "Overall Status:"
    if [ $healthy_count -eq $total_count ]; then
        print_success "All services are healthy ($healthy_count/$total_count)"
    elif [ $healthy_count -gt 0 ]; then
        print_warning "Some services are unhealthy ($healthy_count/$total_count healthy)"
    else
        print_error "All services are down (0/$total_count healthy)"
    fi
    
    # Show testnet endpoints if all services are healthy
    if [ $healthy_count -eq $total_count ]; then
        echo ""
        echo "🔗 Testnet Endpoints:"
        echo "  Gateway:        http://localhost:8888"
        echo "  Health:         http://localhost:8888/gateway/health"
        echo "  Services:       http://localhost:8888/gateway/services"
        echo "  Testnet Status: http://localhost:8888/gateway/testnet/status"
        echo "  Auth Tokens:    http://localhost:8888/auth/testnet-tokens"
    fi
    
    if [ "$WATCH_MODE" = true ]; then
        echo ""
        echo "Press Ctrl+C to stop monitoring..."
        echo "Next check in ${WATCH_INTERVAL} seconds..."
    fi
}

# Main execution
if [ "$WATCH_MODE" = true ]; then
    print_status "Starting continuous health monitoring (interval: ${WATCH_INTERVAL}s)"
    print_status "Press Ctrl+C to stop..."
    echo ""
    
    # Trap Ctrl+C to exit gracefully
    trap 'echo ""; print_status "Health monitoring stopped."; exit 0' INT
    
    while true; do
        perform_health_check
        sleep "$WATCH_INTERVAL"
    done
else
    perform_health_check
fi
