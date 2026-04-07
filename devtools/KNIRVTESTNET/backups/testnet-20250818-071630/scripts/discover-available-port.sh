#!/bin/bash

# KNIRV Testnet - Dynamic Port Discovery Script
# Finds available ports for services when gateway coordinates startup

set -e

# Function to check if a port is available
is_port_available() {
    local port=$1
    ! lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1
}

# Function to find next available port starting from a base port
find_available_port() {
    local base_port=$1
    local max_attempts=${2:-50}
    local current_port=$base_port
    local attempt=1
    
    while [[ $attempt -le $max_attempts ]]; do
        if is_port_available $current_port; then
            echo $current_port
            return 0
        fi
        
        ((current_port++))
        ((attempt++))
    done
    
    echo "ERROR: No available port found starting from $base_port" >&2
    return 1
}

# Function to register port with gateway (if gateway is running)
register_port_with_gateway() {
    local service_name=$1
    local port=$2
    local gateway_url=${3:-"http://localhost:8888"}
    
    # Try to register with gateway service discovery
    if curl -s "$gateway_url/gateway/health" >/dev/null 2>&1; then
        # Gateway is running, try to register
        curl -s -X POST "$gateway_url/gateway/register" \
            -H "Content-Type: application/json" \
            -d "{\"service\":\"$service_name\",\"port\":$port,\"host\":\"localhost\"}" \
            >/dev/null 2>&1 || true
        
        echo "Registered $service_name on port $port with gateway" >&2
    else
        echo "Gateway not available for service registration" >&2
    fi
}

# Main function
main() {
    local service_name=""
    local base_port=""
    local register_with_gateway=false
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --service)
                service_name="$2"
                shift 2
                ;;
            --base-port)
                base_port="$2"
                shift 2
                ;;
            --register)
                register_with_gateway=true
                shift
                ;;
            --help)
                echo "Usage: $0 --service SERVICE_NAME --base-port BASE_PORT [--register]"
                echo ""
                echo "Options:"
                echo "  --service SERVICE_NAME    Name of the service"
                echo "  --base-port BASE_PORT     Starting port to search from"
                echo "  --register                Register with gateway service discovery"
                echo "  --help                    Show this help message"
                exit 0
                ;;
            *)
                echo "Unknown option: $1" >&2
                exit 1
                ;;
        esac
    done
    
    # Validate required arguments
    if [ -z "$service_name" ] || [ -z "$base_port" ]; then
        echo "Error: --service and --base-port are required" >&2
        echo "Use --help for usage information" >&2
        exit 1
    fi
    
    # Validate base_port is a number
    if ! [[ "$base_port" =~ ^[0-9]+$ ]]; then
        echo "Error: base-port must be a number" >&2
        exit 1
    fi
    
    # Find available port
    local available_port
    available_port=$(find_available_port "$base_port")
    
    if [ $? -eq 0 ]; then
        echo "$available_port"
        
        # Register with gateway if requested
        if [ "$register_with_gateway" = true ]; then
            register_port_with_gateway "$service_name" "$available_port"
        fi
        
        exit 0
    else
        echo "Error: Could not find available port starting from $base_port" >&2
        exit 1
    fi
}

# Run main function if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
