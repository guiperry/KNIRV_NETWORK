#!/bin/bash

# KNIRV Testnet Status Checker
# Checks the status of all testnet services

echo "🔍 KNIRV Testnet Status Check"
echo "============================="

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to discover service port from PID file
discover_service_port() {
    local service_name="$1"
    local pid_file="$2"
    local default_port="$3"

    if [ -f "$pid_file" ]; then
        local pid=$(cat "$pid_file" 2>/dev/null)
        if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
            # Get all listening ports for this process
            local ports=$(lsof -Pan -p "$pid" -i 2>/dev/null | grep LISTEN | sed 's/.*:\([0-9]*\).*/\1/' | sort -n)

            # For specific services, prefer certain ports
            case "$service_name" in
                "KNIRV-ORACLE")
                    if echo "$ports" | grep -q "^1317$"; then
                        echo "1317"
                        return 0
                    fi
                    ;;
                "KNIRV-GATEWAY")
                    if echo "$ports" | grep -q "^8888$"; then
                        echo "8888"
                        return 0
                    fi
                    ;;
                "KNIRV-ROUTER")
                    if echo "$ports" | grep -q "^8086$"; then
                        echo "8086"
                        return 0
                    fi
                    ;;
                "NANDA-ANS")
                    if echo "$ports" | grep -q "^9002$"; then
                        echo "9002"
                        return 0
                    fi
                    ;;
            esac

            # Return first available port if preferred not found
            if [ -n "$ports" ]; then
                echo "$ports" | head -1
                return 0
            fi
        fi
    fi

    # Return default port if discovery fails
    echo "$default_port"
}

# Function to check if a service is running with dynamic port discovery
check_service() {
    local service_name="$1"
    local default_port="$2"
    local health_endpoint="$3"
    local pid_file="$4"

    # Discover actual port
    local port=$(discover_service_port "$service_name" "$pid_file" "$default_port")
    local url="http://localhost:${port}${health_endpoint}"

    if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
        if curl -s --max-time 5 "$url" >/dev/null 2>&1; then
            echo -e "✅ ${GREEN}$service_name${NC} - Running on port $port (health: OK)"
        else
            echo -e "⚠️  ${YELLOW}$service_name${NC} - Port $port open but health check failed"
            echo -e "   📍 Checked: $url"
        fi
    else
        echo -e "❌ ${RED}$service_name${NC} - Not running on port $port"
    fi
}

# Function to check PID files
check_pid_file() {
    local service_name="$1"
    local pid_file="$2"
    
    if [ -f "$pid_file" ]; then
        local pid=$(cat "$pid_file")
        if ps -p "$pid" > /dev/null 2>&1; then
            echo -e "📋 ${GREEN}$service_name${NC} - PID $pid (running)"
        else
            echo -e "📋 ${YELLOW}$service_name${NC} - PID file exists but process not running"
        fi
    else
        echo -e "📋 ${RED}$service_name${NC} - No PID file found"
    fi
}

echo -e "\n${BLUE}Service Status (Dynamic Port Discovery):${NC}"
check_service "KNIRV-ORACLE" "1317" "/health" "data/knirvoracle.pid"
check_service "KNIRVCHAIN" "8090" "/health" "data/knirvchain.pid"
check_service "KNIRVGRAPH" "8082" "/height" "data/knirvgraph.pid"
check_service "KNIRV-NEXUS" "8084" "/health" "data/knirvnexus.pid"
check_service "KNIRV-ROUTER" "8086" "/status" "data/knirvrouter.pid"
check_service "KNIRV-GATEWAY" "8888" "/gateway/health" "data/knirvgateway.pid"
check_service "NANDA-ANS" "9002" "/" "data/nanda-ans.pid"
check_service "HEALTH-MONITOR" "10001" "/health-monitor/status" "data/health-monitor.pid"
check_service "KNIRVTESTNET-SERVER" "10000" "/health" "data/knirvtestnet-server.pid"

echo -e "\n${BLUE}PID Files Status:${NC}"
check_pid_file "KNIRV-ORACLE" "data/knirvoracle.pid"
check_pid_file "KNIRVCHAIN" "data/knirvchain.pid"
check_pid_file "KNIRVGRAPH" "data/knirvgraph.pid"
check_pid_file "KNIRV-NEXUS" "data/knirvnexus.pid"
check_pid_file "KNIRV-ROUTER" "data/knirvrouter.pid"
check_pid_file "KNIRV-GATEWAY" "data/knirvgateway.pid"
check_pid_file "NANDA ANS" "data/nanda-ans.pid"
check_pid_file "Health Monitor" "data/health-monitor.pid"

echo -e "\n${BLUE}Log Files:${NC}"
for log_file in logs/*.log; do
    if [ -f "$log_file" ]; then
        size=$(du -h "$log_file" | cut -f1)
        modified=$(stat -c %y "$log_file" | cut -d' ' -f1,2 | cut -d'.' -f1)
        echo -e "📄 ${GREEN}$(basename "$log_file")${NC} - Size: $size, Modified: $modified"
    fi
done

echo -e "\n${BLUE}Quick Access URLs:${NC}"
echo "🌐 KNIRVTESTNET Portal: http://localhost:10000"
echo "🏥 Health Monitor: http://localhost:10001/health-monitor"
echo "🔗 KNIRV-ORACLE: http://localhost:1317"
echo "⛓️  KNIRVCHAIN: http://localhost:8090"
echo "📊 KNIRVGRAPH: http://localhost:8082"
echo "🔒 KNIRV-NEXUS API: http://localhost:8084"
echo "🔒 KNIRV-NEXUS GUI: http://localhost:8083"
echo "🌐 KNIRV-ROUTER: http://localhost:8086"
echo "🚪 KNIRV-GATEWAY: http://localhost:8888"
echo "🤖 NANDA ANS: http://localhost:9002"

echo -e "\n${BLUE}Useful Commands:${NC}"
echo "📊 View all logs: npm run logs"
echo "🛑 Stop testnet: npm run testnet:stop"
echo "🔄 Restart testnet: npm run testnet:restart"
echo "🔍 List processes: npm run services:list"
