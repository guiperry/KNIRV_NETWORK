#!/bin/bash

# KNIRVTESTNET Services Deployment Script
# Deploys KNIRVTESTNET services to AWS EC2 instance via Docker, Podman, or Native
# Enhanced with container runtime detection, interactive selection, and native npm deployment
# Includes XION bridge and KNIRVANA components for complete testnet deployment

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Configuration
TESTNET_DIR="$PROJECT_ROOT/KNIRVTESTNET"
ANSIBLE_DIR="$PROJECT_ROOT/deployment/ansible"
TESTNET_IP_FILE="$ANSIBLE_DIR/testnet_ip.txt"
SSH_KEY="~/.ssh/AEGONG.pem"
INSTANCE_ID="i-06813be8a8a23ea5b"

# Container runtime selection (will be set by detect_container_runtime)
CONTAINER_RUNTIME=""
COMPOSE_COMMAND=""

# Functions
print_header() {
    echo -e "${BLUE}=================================${NC}"
    echo -e "${BLUE}🧪 KNIRVTESTNET Services Deploy${NC}"
    echo -e "${BLUE}=================================${NC}"
    echo ""
}

print_step() {
    echo -e "${PURPLE}[STEP]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

# New function to detect and select container runtime
detect_container_runtime() {
    print_step "Detecting available container runtimes..."

    local docker_available=false
    local podman_available=false
    local docker_running_services=false
    local podman_running_services=false

    # Check if Docker is available and running
    if command -v docker &> /dev/null; then
        if docker info &> /dev/null; then
            docker_available=true
            print_success "Docker is available and running"

            # Check for running KNIRV services in Docker
            local docker_knirv_containers=$(docker ps --filter "name=knirv" --format "{{.Names}}" 2>/dev/null || true)
            if [ -n "$docker_knirv_containers" ]; then
                docker_running_services=true
                print_warning "Found running KNIRV services in Docker:"
                echo "$docker_knirv_containers" | sed 's/^/  - /'
            fi
        else
            print_warning "Docker is installed but not running"
        fi
    else
        print_warning "Docker is not available"
    fi

    # Check if Podman is available
    if command -v podman &> /dev/null; then
        podman_available=true
        print_success "Podman is available"

        # Check for running KNIRV services in Podman
        local podman_knirv_containers=$(podman ps --filter "name=knirv" --format "{{.Names}}" 2>/dev/null || true)
        if [ -n "$podman_knirv_containers" ]; then
            podman_running_services=true
            print_warning "Found running KNIRV services in Podman:"
            echo "$podman_knirv_containers" | sed 's/^/  - /'
        fi
    else
        print_warning "Podman is not available"
    fi

    # Handle cases where services are already running
    if [ "$docker_running_services" = true ] && [ "$podman_running_services" = true ]; then
        print_error "KNIRV services are running in both Docker and Podman!"
        print_error "Please stop services in one runtime before proceeding:"
        echo "  Stop Docker services: docker stop \$(docker ps --filter \"name=knirv\" -q)"
        echo "  Stop Podman services: podman stop \$(podman ps --filter \"name=knirv\" -q)"
        exit 1
    elif [ "$docker_running_services" = true ]; then
        print_warning "KNIRV services are already running in Docker"
        read -p "Do you want to stop existing Docker services and redeploy? (y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            print_status "Stopping existing Docker services..."
            docker stop $(docker ps --filter "name=knirv" -q) 2>/dev/null || true
            docker rm $(docker ps -a --filter "name=knirv" -q) 2>/dev/null || true
        else
            print_error "Deployment cancelled"
            exit 0
        fi
    elif [ "$podman_running_services" = true ]; then
        print_warning "KNIRV services are already running in Podman"
        read -p "Do you want to stop existing Podman services and redeploy? (y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            print_status "Stopping existing Podman services..."
            podman stop $(podman ps --filter "name=knirv" -q) 2>/dev/null || true
            podman rm $(podman ps -a --filter "name=knirv" -q) 2>/dev/null || true
        else
            print_error "Deployment cancelled"
            exit 0
        fi
    fi

    # Interactive selection - always show all options including Native
    echo ""
    print_step "Multiple deployment options available. Please choose:"
    echo "  1) Docker (traditional container runtime)"
    echo "  2) Podman (rootless, daemonless container runtime)"
    echo "  3) Native (upload KNIRVTESTNET directory and run npm scripts)"
    echo ""
    while true; do
        read -p "Enter your choice (1, 2, or 3): " choice
        case $choice in
            1)
                if [ "$docker_available" = false ]; then
                    print_error "Docker is not available. Please install Docker or choose another option."
                    continue
                fi
                CONTAINER_RUNTIME="docker"
                print_success "Selected Docker as container runtime"
                break
                ;;
            2)
                if [ "$podman_available" = false ]; then
                    print_error "Podman is not available. Please install Podman or choose another option."
                    continue
                fi
                CONTAINER_RUNTIME="podman"
                COMPOSE_COMMAND="podman-compose"
                print_success "Selected Podman as container runtime"
                break
                ;;
            3)
                CONTAINER_RUNTIME="native"
                print_success "Selected Native deployment"
                break
                ;;
            *)
                print_error "Invalid choice. Please enter 1, 2, or 3."
                ;;
        esac
    done

    # Set up Docker Compose command (handle both v1 and v2)
    if [ "$CONTAINER_RUNTIME" = "docker" ]; then
        if command -v docker-compose &> /dev/null; then
            COMPOSE_COMMAND="docker-compose"
            print_success "Using Docker Compose v1 (docker-compose)"
        elif docker compose version &> /dev/null; then
            COMPOSE_COMMAND="docker compose"
            print_success "Using Docker Compose v2 (docker compose)"
        else
            print_error "Docker Compose not found. Please install Docker Compose."
            exit 1
        fi
    fi

    # Verify Podman compose command is available
    if [ "$CONTAINER_RUNTIME" = "podman" ] && ! command -v "$COMPOSE_COMMAND" &> /dev/null; then
        print_warning "podman-compose not found. Installing via pip..."
        pip3 install podman-compose || {
            print_error "Failed to install podman-compose. Please install manually:"
            echo "  pip3 install podman-compose"
            exit 1
        }
    fi

    print_success "Container runtime configured: $CONTAINER_RUNTIME with $COMPOSE_COMMAND"
}

get_instance_ip() {
    # Use the utility script to get current IP (only the IP, not the verbose output)
    TESTNET_IP=$("$PROJECT_ROOT/scripts/get-testnet-ip.sh" get-ip 2>/dev/null | tail -n1)
    if [ $? -ne 0 ] || [ -z "$TESTNET_IP" ]; then
        print_error "Failed to get testnet IP address"
        # Try the verbose version to see what's wrong
        "$PROJECT_ROOT/scripts/get-testnet-ip.sh" get-ip
        exit 1
    fi
    print_success "Testnet IP: $TESTNET_IP"
}

check_prerequisites() {
    print_step "Checking prerequisites..."

    # Detect and select container runtime first
    detect_container_runtime

    # Get current IP address dynamically from AWS
    get_instance_ip

    # Check if SSH key exists
    if [ ! -f "${SSH_KEY/#\~/$HOME}" ]; then
        print_error "SSH key not found at $SSH_KEY"
        print_warning "Please ensure your AWS key pair is available"
        exit 1
    fi

    # Check if KNIRVTESTNET directory exists
    if [ ! -d "$TESTNET_DIR" ]; then
        print_error "KNIRVTESTNET directory not found at $TESTNET_DIR"
        exit 1
    fi

    # Check if docker-compose.yml exists
    if [ ! -f "$TESTNET_DIR/docker-compose.yml" ]; then
        print_error "docker-compose.yml not found in KNIRVTESTNET directory"
        exit 1
    fi

    print_success "Prerequisites check passed"
    print_success "Testnet IP: $TESTNET_IP"
    print_success "Container Runtime: $CONTAINER_RUNTIME"
}

test_ssh_connection() {
    print_step "Testing SSH connection to testnet..."

    if ssh -i "${SSH_KEY/#\~/$HOME}" -o ConnectTimeout=10 -o StrictHostKeyChecking=no ubuntu@"$TESTNET_IP" "echo 'SSH connection successful'" > /dev/null 2>&1; then
        print_success "SSH connection established"
    else
        print_error "Cannot connect to testnet via SSH"
        print_warning "Please check your SSH key and security group settings"
        exit 1
    fi
}

check_disk_space() {
    print_step "Checking available disk space on testnet server..."

    # Get disk usage information
    DISK_INFO=$(ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$TESTNET_IP" "df -h / | tail -n1")
    DISK_USAGE=$(echo "$DISK_INFO" | awk '{print $5}' | sed 's/%//')
    AVAILABLE_SPACE=$(echo "$DISK_INFO" | awk '{print $4}')

    print_status "Current disk usage: $DISK_USAGE% (Available: $AVAILABLE_SPACE)"

    # Check if disk usage is too high (>80%)
    if [ "$DISK_USAGE" -gt 80 ]; then
        print_error "Disk usage is too high ($DISK_USAGE%). Available space: $AVAILABLE_SPACE"
        print_warning "Please clean up the server or increase disk size before deployment"
        print_warning "You can clean up with: ssh knirv-testnet 'sudo rm -rf /opt/knirv-testnet'"
        exit 1
    fi

    print_success "Sufficient disk space available ($AVAILABLE_SPACE free)"
}

prepare_native_testnet_files() {
    print_step "Preparing KNIRVTESTNET directory for native deployment..."
    print_status "Including XION bridge and KNIRVANA components..."

    # Create temporary directory for native deployment
    TEMP_DIR=$(mktemp -d)

    # Copy the entire KNIRVTESTNET directory
    print_status "Copying KNIRVTESTNET directory..."
    cp -r "$TESTNET_DIR" "$TEMP_DIR/knirvtestnet"

    # Add XION bridge components from KNIRVORACLE
    if [ -d "KNIRVORACLE" ]; then
        print_status "Adding XION bridge components..."
        mkdir -p "$TEMP_DIR/knirvtestnet/xion-bridge"
        cp KNIRVORACLE/xion_bridge.go "$TEMP_DIR/knirvtestnet/xion-bridge/" 2>/dev/null || true
        cp -r KNIRVORACLE/xion-bridge/* "$TEMP_DIR/knirvtestnet/xion-bridge/" 2>/dev/null || true
    fi

    # Add KNIRVANA ts-client web game
    if [ -d "KNIRVANA/ts-client" ]; then
        print_status "Adding KNIRVANA ts-client web game..."
        mkdir -p "$TEMP_DIR/knirvtestnet/knirvana-game"
        cp -r KNIRVANA/ts-client/* "$TEMP_DIR/knirvtestnet/knirvana-game/" 2>/dev/null || true
    fi

    # Add KNIRVANA orb-menu if available
    if [ -d "KNIRVANA/orb-menu" ]; then
        print_status "Adding KNIRVANA orb-menu..."
        mkdir -p "$TEMP_DIR/knirvtestnet/knirvana-orb"
        cp -r KNIRVANA/orb-menu/* "$TEMP_DIR/knirvtestnet/knirvana-orb/" 2>/dev/null || true
    fi

    print_success "Native testnet files prepared in $TEMP_DIR"
}

upload_native_testnet_files() {
    print_step "Uploading KNIRVTESTNET directory to testnet server..."

    # Upload the entire prepared directory
    print_status "Uploading complete KNIRVTESTNET with XION bridge and KNIRVANA components..."
    scp -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no -r "$TEMP_DIR/knirvtestnet" ubuntu@"$TESTNET_IP":/opt/knirv-testnet

    print_success "Native testnet files uploaded to testnet server"
}

prepare_testnet_files() {
    print_step "Preparing testnet files for deployment..."

    # Create temporary directory for deployment files
    TEMP_DIR=$(mktemp -d)

    # Only copy essential files for production deployment (not the entire KNIRVTESTNET directory)
    print_status "Copying essential files only to avoid disk space issues..."

    # Create directory structure
    mkdir -p "$TEMP_DIR/bin"
    mkdir -p "$TEMP_DIR/config"
    mkdir -p "$TEMP_DIR/data"

    # Copy only the essential binaries (not duplicates or development tools)
    cp "$TESTNET_DIR/bin/knirvoracle" "$TEMP_DIR/bin/" 2>/dev/null || true
    cp "$TESTNET_DIR/bin/knirvchain" "$TEMP_DIR/bin/" 2>/dev/null || true
    cp "$TESTNET_DIR/bin/knirvgraph" "$TEMP_DIR/bin/" 2>/dev/null || true
    cp "$TESTNET_DIR/bin/knirvnexus" "$TEMP_DIR/bin/" 2>/dev/null || true
    cp "$TESTNET_DIR/bin/knirvrouter" "$TEMP_DIR/bin/" 2>/dev/null || true
    cp "$TESTNET_DIR/bin/knirv-server.wasm" "$TEMP_DIR/bin/" 2>/dev/null || true
    cp "$TESTNET_DIR/bin/wasmtime" "$TEMP_DIR/bin/" 2>/dev/null || true

    # Copy configuration files
    cp -r "$TESTNET_DIR/config"/* "$TEMP_DIR/config/" 2>/dev/null || true

    # Copy essential data directories (excluding large development files)
    print_status "Copying data directories (excluding node_modules, dist, build, etc.)..."

    # Copy knirvchain data (small config files only)
    if [ -d "$TESTNET_DIR/data/knirvchain" ]; then
        mkdir -p "$TEMP_DIR/data/knirvchain"
        cp "$TESTNET_DIR/data/knirvchain"/*.toml "$TEMP_DIR/data/knirvchain/" 2>/dev/null || true
        cp "$TESTNET_DIR/data/knirvchain"/*.db "$TEMP_DIR/data/knirvchain/" 2>/dev/null || true
    fi

    # Copy knirvgraph data (config only, not large data files)
    if [ -d "$TESTNET_DIR/data/knirvgraph" ]; then
        mkdir -p "$TEMP_DIR/data/knirvgraph"
        cp "$TESTNET_DIR/data/knirvgraph"/*.yaml "$TEMP_DIR/data/knirvgraph/" 2>/dev/null || true
        cp "$TESTNET_DIR/data/knirvgraph"/*.yml "$TEMP_DIR/data/knirvgraph/" 2>/dev/null || true
    fi

    # Copy knirvnexus data (config files only, NOT the portal directory)
    if [ -d "$TESTNET_DIR/data/knirvnexus" ]; then
        mkdir -p "$TEMP_DIR/data/knirvnexus"
        cp "$TESTNET_DIR/data/knirvnexus"/*.yaml "$TEMP_DIR/data/knirvnexus/" 2>/dev/null || true
        cp "$TESTNET_DIR/data/knirvnexus"/*.yml "$TEMP_DIR/data/knirvnexus/" 2>/dev/null || true
        cp "$TESTNET_DIR/data/knirvnexus"/*.db "$TEMP_DIR/data/knirvnexus/" 2>/dev/null || true
        cp "$TESTNET_DIR/data/knirvnexus"/*.config "$TEMP_DIR/data/knirvnexus/" 2>/dev/null || true
        # Copy small directories but exclude portal
        [ -d "$TESTNET_DIR/data/knirvnexus/reports" ] && cp -r "$TESTNET_DIR/data/knirvnexus/reports" "$TEMP_DIR/data/knirvnexus/" 2>/dev/null || true
    fi

    # Copy knirvoracle data (small config files only)
    if [ -d "$TESTNET_DIR/data/knirvoracle" ]; then
        mkdir -p "$TEMP_DIR/data/knirvoracle"
        cp -r "$TESTNET_DIR/data/knirvoracle/genesis" "$TEMP_DIR/data/knirvoracle/" 2>/dev/null || true
    fi

    # Copy knirvrouter data (small config files only)
    if [ -d "$TESTNET_DIR/data/knirvrouter" ]; then
        mkdir -p "$TEMP_DIR/data/knirvrouter"
        cp "$TESTNET_DIR/data/knirvrouter"/*.env "$TEMP_DIR/data/knirvrouter/" 2>/dev/null || true
        cp "$TESTNET_DIR/data/knirvrouter"/*.yaml "$TEMP_DIR/data/knirvrouter/" 2>/dev/null || true
        cp "$TESTNET_DIR/data/knirvrouter"/*.yml "$TEMP_DIR/data/knirvrouter/" 2>/dev/null || true
    fi

    # Copy testnet-gateway (source files only, NOT node_modules, .netlify, etc.)
    if [ -d "$TESTNET_DIR/data/testnet-gateway" ]; then
        mkdir -p "$TEMP_DIR/data/testnet-gateway"
        # Copy essential files but exclude large development directories
        rsync -av --exclude='node_modules' --exclude='dist' --exclude='build' --exclude='.next' \
              --exclude='.netlify' --exclude='*.log' --exclude='.git' --exclude='.cache' \
              "$TESTNET_DIR/data/testnet-gateway/" "$TEMP_DIR/data/testnet-gateway/" 2>/dev/null || {
            # Fallback if rsync is not available
            find "$TESTNET_DIR/data/testnet-gateway" -type f \
                ! -path "*/node_modules/*" ! -path "*/dist/*" ! -path "*/build/*" ! -path "*/.next/*" \
                ! -path "*/.netlify/*" ! -name "*.log" ! -path "*/.git/*" ! -path "*/.cache/*" \
                -exec cp --parents {} "$TEMP_DIR/data/" \; 2>/dev/null || true
        }
    fi

    # Copy docker-compose file
    cp "$TESTNET_DIR/docker-compose.yml" "$TEMP_DIR/" 2>/dev/null || true

    # Create production docker-compose file using pre-built binaries (fixed port conflicts)
    print_status "Creating production docker-compose file..."
    cat > "$TEMP_DIR/docker-compose-prod.yml" << 'EOF'
version: '3.8'

services:
  # NGINX Network Manager - Routes traffic to containers
  nginx:
    image: nginx:alpine
    container_name: knirv-testnet-nginx
    ports:
      - "80:80"       # HTTP
      - "443:443"     # HTTPS
      - "1317:1317"   # KNIRVORACLE
      - "8090:8090"   # KNIRVCHAIN
      - "8082:8082"   # KNIRVGRAPH
      - "8084:8084"   # KNIRVNEXUS-1
      - "8085:8085"   # KNIRVNEXUS-2
      - "8086:8086"   # KNIRVROUTER
      - "10000:10000" # TESTNET-GATEWAY
      - "3000:3000"   # KNIRVANA
      - "8088:8088"   # XION-BRIDGE
      - "5001:5001"   # IPFS-API
      - "8081:8081"   # IPFS-GATEWAY
    volumes:
      - ./config/nginx.conf:/etc/nginx/nginx.conf:ro
      - ./logs/nginx:/var/log/nginx
    depends_on:
      - ipfs
      - knirv-oracle
      - knirv-chain
      - knirv-graph
      - knirv-nexus
      - knirv-router
      - testnet-gateway
      - knirvana
      - xion-bridge
    networks:
      - knirv-testnet
    restart: unless-stopped

  # IPFS for distributed storage
  ipfs:
    image: ipfs/kubo:latest
    container_name: knirv-testnet-ipfs
    expose:
      - "5001"        # API port (internal only)
      - "8080"        # Gateway port (internal only)
    volumes:
      - ./data/ipfs:/data/ipfs
    environment:
      - IPFS_PROFILE=server
    networks:
      - knirv-testnet
    restart: unless-stopped

  # Testnet Gateway - Node.js application
  testnet-gateway:
    image: node:20-alpine
    container_name: knirv-testnet-gateway
    working_dir: /app
    expose:
      - "10000"       # Internal port only
    environment:
      - DEPLOYMENT_ENV=production
      - KNIRVORACLE_API=http://knirv-oracle:1317
      - KNIRVCHAIN_API=http://knirv-chain:8090
      - KNIRVGRAPH_API=http://knirv-graph:8082
      - KNIRVNEXUS_API=http://knirv-nexus:8084
      - KNIRVROUTER_API=http://knirv-router:8086
    volumes:
      - ./data/testnet-gateway:/app
    command: sh -c "npm install --production && npm start"
    depends_on:
      - knirv-oracle
      - knirv-chain
      - knirv-graph
      - knirv-nexus
      - knirv-router
    networks:
      - knirv-testnet
    restart: unless-stopped

  # KNIRV Oracle - Using pre-built binary
  knirv-oracle:
    image: ubuntu:22.04
    container_name: knirv-oracle
    expose:
      - "1317"        # Internal port only
      - "26657"       # Tendermint RPC
      - "26656"       # Tendermint P2P
    volumes:
      - ./bin/knirvoracle:/usr/local/bin/knirvoracle:ro
      - ./data/knirvoracle:/root/.knirvoracle
      - ./config:/root/config:ro
    command: /usr/local/bin/knirvoracle start
    depends_on:
      - ipfs
    networks:
      - knirv-testnet
    restart: unless-stopped

  # KNIRV Chain - Using pre-built binary
  knirv-chain:
    image: ubuntu:22.04
    container_name: knirv-chain
    expose:
      - "8090"        # Internal port only
    volumes:
      - ./bin/knirvchain:/usr/local/bin/knirvchain:ro
      - ./data/knirvchain:/app/data
      - ./config:/app/config:ro
    command: /usr/local/bin/knirvchain --port 8090
    depends_on:
      - knirv-oracle
      - ipfs
    networks:
      - knirv-testnet
    restart: unless-stopped

  # KNIRV Graph - Using pre-built binary
  knirv-graph:
    image: ubuntu:22.04
    container_name: knirv-graph
    expose:
      - "8082"        # Internal port only
    volumes:
      - ./bin/knirvgraph:/usr/local/bin/knirvgraph:ro
      - ./data/knirvgraph:/app/data
      - ./config:/app/config:ro
    command: /usr/local/bin/knirvgraph --port 8082
    depends_on:
      - knirv-oracle
      - ipfs
    networks:
      - knirv-testnet
    restart: unless-stopped

  # KNIRV Nexus - Using pre-built binary
  knirv-nexus:
    image: ubuntu:22.04
    container_name: knirv-nexus
    expose:
      - "8084"        # Internal port only
    volumes:
      - ./bin/knirvnexus:/usr/local/bin/knirvnexus:ro
      - ./data/knirvnexus:/app/data
      - ./config:/app/config:ro
    command: /usr/local/bin/knirvnexus --port 8084
    depends_on:
      - knirv-oracle
    networks:
      - knirv-testnet
    restart: unless-stopped

  # KNIRV Router - Using pre-built binary
  knirv-router:
    image: ubuntu:22.04
    container_name: knirv-router
    expose:
      - "8086"        # Internal port only
      - "3478"        # STUN/TURN port
    volumes:
      - ./bin/knirvrouter:/usr/local/bin/knirvrouter:ro
      - ./data/knirvrouter:/app/data
      - ./config:/app/config:ro
    command: /usr/local/bin/knirvrouter --port 8086
    depends_on:
      - knirv-oracle
    networks:
      - knirv-testnet
    restart: unless-stopped

  # KNIRVANA Game - TypeScript client
  knirvana:
    image: node:20-alpine
    container_name: knirv-testnet-knirvana
    working_dir: /app
    expose:
      - "3000"        # Internal port only
    volumes:
      - ./data/knirvana:/app
    environment:
      - NODE_ENV=production
      - KNIRVGRAPH_API=http://knirv-graph:8082
      - KNIRVCHAIN_API=http://knirv-chain:8090
      - KNIRVORACLE_API=http://knirv-oracle:1317
    command: sh -c "npm install && npm start"
    depends_on:
      - knirv-graph
      - knirv-chain
    networks:
      - knirv-testnet
    restart: unless-stopped

  # XION Bridge - Cross-chain bridge
  xion-bridge:
    image: node:20-alpine
    container_name: knirv-testnet-xion-bridge
    working_dir: /app
    expose:
      - "8088"        # Internal port only
    volumes:
      - ./data/xion-bridge:/app
    environment:
      - NODE_ENV=production
      - KNIRVORACLE_API=http://knirv-oracle:1317
      - KNIRVCHAIN_API=http://knirv-chain:8090
      - BRIDGE_PORT=8088
    command: sh -c "npm install && npm start"
    depends_on:
      - knirv-oracle
      - knirv-chain
    networks:
      - knirv-testnet
    restart: unless-stopped

networks:
  knirv-testnet:
    driver: bridge

volumes:
  ipfs_data:
EOF

    # Create appropriate compose file based on selected runtime
    if [ "$CONTAINER_RUNTIME" = "podman" ]; then
        print_status "Creating Podman-specific compose file..."
        # Update docker-compose.yml for production deployment
        sed -i 's|build:|#build:|g' "$TEMP_DIR/docker-compose.yml"
        sed -i 's|context: \.\./|#context: ../|g' "$TEMP_DIR/docker-compose.yml"
        sed -i 's|dockerfile: Dockerfile|#dockerfile: Dockerfile|g' "$TEMP_DIR/docker-compose.yml"

        # Create Podman-specific compose file
        cat > "$TEMP_DIR/podman-compose-prod.yml" << 'EOF'
version: '3.8'

services:
  ipfs:
    image: docker.io/ipfs/kubo:latest # Use explicit registry for Podman
    container_name: knirv-testnet-ipfs
    ports:
      - "5001:5001"
      - "8080:8080"
    volumes:
      - ./data/ipfs:/data/ipfs:Z # Add :Z for SELinux
    environment:
      - IPFS_PROFILE=server
    networks:
      - knirv-testnet
    restart: unless-stopped

  knirvoracle:
    image: knirvoracle:latest
    container_name: knirv-testnet-root
    ports:
      - "1317:1317"
      - "26657:26657"
      - "26656:26656"
    volumes:
      - ./data/knirvoracle:/root/.knirvoracle:Z
      - ./config/knirvoracle-config.yaml:/root/config.yaml:Z
    depends_on:
      - ipfs
    networks:
      - knirv-testnet
    environment:
      - KNIRV_NETWORK=testnet
      - KNIRV_CHAIN_ID=knirv-testnet-1
    restart: unless-stopped

  knirvchain:
    image: knirvchain:latest
    container_name: knirv-testnet-chain
    ports:
      - "8080:8080"
    volumes:
      - ./data/knirvchain:/app/data:Z
      - ./config/knirvchain-config.toml:/app/config.toml:Z
    depends_on:
      - knirvoracle
      - ipfs
    networks:
      - knirv-testnet
    environment:
      - RUST_LOG=info
    restart: unless-stopped

  knirvgraph:
    image: knirvgraph:latest
    container_name: knirv-testnet-graph
    ports:
      - "8081:8081"
      - "4001:4001"
    volumes:
      - ./data/knirvgraph:/app/data:Z
      - ./config/knirvgraph-config.yaml:/app/config.yaml:Z
    depends_on:
      - knirvoracle
      - ipfs
    networks:
      - knirv-testnet
    environment:
      - GO_ENV=testnet
    restart: unless-stopped

  knirvnexus-1:
    image: knirvnexus:latest
    container_name: knirv-testnet-nexus-1
    ports:
      - "8082:8082"
    volumes:
      - ./data/knirvnexus/node-1:/app/data:Z
      - ./config/knirvnexus-config.yaml:/app/config.yaml:Z
    environment:
      - NODE_ID=1
      - MOCK_TEE=true
    depends_on:
      - knirvoracle
    networks:
      - knirv-testnet
    restart: unless-stopped

  knirvnexus-2:
    image: knirvnexus:latest
    container_name: knirv-testnet-nexus-2
    ports:
      - "8083:8083"
    volumes:
      - ./data/knirvnexus/node-2:/app/data:Z
      - ./config/knirvnexus-config.yaml:/app/config.yaml:Z
    environment:
      - NODE_ID=2
      - MOCK_TEE=true
    depends_on:
      - knirvoracle
    networks:
      - knirv-testnet
    restart: unless-stopped

  knirvrouter:
    image: knirvrouter:latest
    container_name: knirv-testnet-router
    ports:
      - "8086:8086"
      - "3478:3478"
    volumes:
      - ./data/knirvrouter:/app/data:Z
      - ./config/knirvrouter-config.yaml:/app/config.yaml:Z
    depends_on:
      - knirvoracle
    networks:
      - knirv-testnet
    restart: unless-stopped

  knirvgateway:
    image: knirvgateway:latest
    container_name: knirv-testnet-gateway
    ports:
      - "8087:8087"
    volumes:
      - ./config/knirvgateway-config.json:/app/config.json:Z
    depends_on:
      - knirvoracle
      - knirvchain
      - knirvgraph
      - knirvnexus-1
      - knirvnexus-2
      - knirvrouter
    networks:
      - knirv-testnet
    environment:
      - NODE_ENV=testnet
    restart: unless-stopped

networks:
  knirv-testnet:
    name: knirv-testnet
    driver: bridge
EOF
    else
        print_status "Creating Docker-specific compose file..."
        # For Docker, create a production version that uses pre-built images
        cat > "$TEMP_DIR/docker-compose-prod.yml" << 'EOF'
version: '3.8'

services:
  ipfs:
    image: ipfs/kubo:latest
    container_name: knirv-testnet-ipfs
    ports:
      - "5001:5001"
      - "8080:8080"
    volumes:
      - ./data/ipfs:/data/ipfs
    environment:
      - IPFS_PROFILE=server
    networks:
      - knirv-testnet
    restart: unless-stopped

  knirvoracle:
    image: knirvoracle:latest
    container_name: knirv-testnet-root
    ports:
      - "1317:1317"
      - "26657:26657"
      - "26656:26656"
    volumes:
      - ./data/knirvoracle:/root/.knirvoracle
      - ./config/knirvoracle-config.yaml:/root/config.yaml
    depends_on:
      - ipfs
    networks:
      - knirv-testnet
    environment:
      - KNIRV_NETWORK=testnet
      - KNIRV_CHAIN_ID=knirv-testnet-1
    restart: unless-stopped

  knirvchain:
    image: knirvchain:latest
    container_name: knirv-testnet-chain
    ports:
      - "8080:8080"
    volumes:
      - ./data/knirvchain:/app/data
      - ./config/knirvchain-config.toml:/app/config.toml
    depends_on:
      - knirvoracle
      - ipfs
    networks:
      - knirv-testnet
    environment:
      - RUST_LOG=info
    restart: unless-stopped

  knirvgraph:
    image: knirvgraph:latest
    container_name: knirv-testnet-graph
    ports:
      - "8081:8081"
      - "4001:4001"
    volumes:
      - ./data/knirvgraph:/app/data
      - ./config/knirvgraph-config.yaml:/app/config.yaml
    depends_on:
      - knirvoracle
      - ipfs
    networks:
      - knirv-testnet
    environment:
      - GO_ENV=testnet
    restart: unless-stopped

  knirvnexus-1:
    image: knirvnexus:latest
    container_name: knirv-testnet-nexus-1
    ports:
      - "8082:8082"
    volumes:
      - ./data/knirvnexus/node-1:/app/data
      - ./config/knirvnexus-config.yaml:/app/config.yaml
    environment:
      - NODE_ID=1
      - MOCK_TEE=true
    depends_on:
      - knirvoracle
    networks:
      - knirv-testnet
    restart: unless-stopped

  knirvnexus-2:
    image: knirvnexus:latest
    container_name: knirv-testnet-nexus-2
    ports:
      - "8083:8083"
    volumes:
      - ./data/knirvnexus/node-2:/app/data
      - ./config/knirvnexus-config.yaml:/app/config.yaml
    environment:
      - NODE_ID=2
      - MOCK_TEE=true
    depends_on:
      - knirvoracle
    networks:
      - knirv-testnet
    restart: unless-stopped

  knirvrouter:
    image: knirvrouter:latest
    container_name: knirv-testnet-router
    ports:
      - "8086:8086"
      - "3478:3478"
    volumes:
      - ./data/knirvrouter:/app/data
      - ./config/knirvrouter-config.yaml:/app/config.yaml
    depends_on:
      - knirvoracle
    networks:
      - knirv-testnet
    restart: unless-stopped

  knirvgateway:
    image: knirvgateway:latest
    container_name: knirv-testnet-gateway
    ports:
      - "8087:8087"
    volumes:
      - ./config/knirvgateway-config.json:/app/config.json
    depends_on:
      - knirvoracle
      - knirvchain
      - knirvgraph
      - knirvnexus-1
      - knirvnexus-2
      - knirvrouter
    networks:
      - knirv-testnet
    environment:
      - NODE_ENV=testnet
    restart: unless-stopped

networks:
  knirv-testnet:
    name: knirv-testnet
    driver: bridge
EOF
    fi

    print_success "Testnet files prepared in $TEMP_DIR"
    echo "$TEMP_DIR" > /tmp/testnet_temp_dir
}

setup_container_permissions() {
    print_step "Setting up container runtime permissions on server..."

    if [ "$CONTAINER_RUNTIME" = "podman" ]; then
        print_status "Configuring Podman permissions..."
        ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$TESTNET_IP" << 'EOF'
# Enable lingering for the ubuntu user (allows rootless containers to persist)
sudo loginctl enable-linger ubuntu

# Ensure proper permissions for rootless Podman
mkdir -p ~/.config/containers
mkdir -p ~/.local/share/containers

# Set up proper SELinux contexts if available
if command -v setsebool &> /dev/null; then
    sudo setsebool -P container_manage_cgroup true
fi

# Ensure proper ownership of container directories
sudo chown -R ubuntu:ubuntu /opt/knirv-testnet
chmod -R 755 /opt/knirv-testnet/data
chmod -R 755 /opt/knirv-testnet/config

echo "Podman permissions configured"
EOF
    else
        print_status "Configuring Docker permissions..."
        ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$TESTNET_IP" << 'EOF'
# Add ubuntu user to docker group if not already added
if ! groups ubuntu | grep -q docker; then
    sudo usermod -aG docker ubuntu
    echo "Added ubuntu user to docker group"
fi

# Ensure Docker daemon is running
sudo systemctl enable docker
sudo systemctl start docker

# Ensure proper ownership of container directories
sudo chown -R ubuntu:ubuntu /opt/knirv-testnet
chmod -R 755 /opt/knirv-testnet/data
chmod -R 755 /opt/knirv-testnet/config

echo "Docker permissions configured"
EOF
    fi

    print_success "Container runtime permissions configured"
}

upload_testnet_files() {
    print_step "Uploading testnet files to server..."

    TEMP_DIR=$(cat /tmp/testnet_temp_dir)

    # Create testnet directory on server
    ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$TESTNET_IP" \
        "sudo mkdir -p /opt/knirv-testnet && sudo chown ubuntu:ubuntu /opt/knirv-testnet"

    # Upload files
    scp -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no -r "$TEMP_DIR"/* ubuntu@"$TESTNET_IP":/opt/knirv-testnet/

    # Set up container runtime permissions
    setup_container_permissions

    print_success "Files uploaded to testnet server"
}

install_node_dependencies() {
    print_step "Installing Node.js dependencies on testnet server..."

    # First, ensure Node.js is installed
    ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$TESTNET_IP" \
        "if ! command -v node >/dev/null 2>&1; then \
            echo 'Installing Node.js...' && \
            curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash - && \
            sudo apt-get install -y nodejs; \
        else \
            echo 'Node.js is already installed'; \
        fi"

    # Check if testnet-gateway exists and needs npm install
    ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$TESTNET_IP" \
        "cd /opt/knirv-testnet && if [ -d data/testnet-gateway ] && [ -f data/testnet-gateway/package.json ]; then \
            echo 'Installing testnet-gateway dependencies...' && \
            cd data/testnet-gateway && \
            npm install --production --no-optional --no-audit --no-fund; \
        else \
            echo 'No testnet-gateway package.json found, skipping npm install'; \
        fi"

    print_success "Node.js dependencies installed"
}

build_container_images() {
    print_step "Building container images on testnet server..."
    print_warning "Using real KNIRV binaries for production testnet deployment."

    # Build images on the server using the selected container runtime
    if [ "$CONTAINER_RUNTIME" = "podman" ]; then
        print_status "Building images with Podman..."
        ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$TESTNET_IP" << 'EOF'
cd /opt/knirv-testnet

# Build KNIRV services using real binaries
echo "Building KNIRVORACLE..."
podman build -t knirvoracle:latest -f - . << 'CONTAINERFILE'
FROM ubuntu:22.04
RUN apt-get update && apt-get install -y curl ca-certificates
COPY bin/knirvoracle /app/knirvoracle
WORKDIR /app
EXPOSE 1317 26656 26657
CMD ["./knirvoracle"]
CONTAINERFILE

echo "Building KNIRVCHAIN..."
podman build -t knirvchain:latest -f - . << 'CONTAINERFILE'
FROM ubuntu:22.04
RUN apt-get update && apt-get install -y curl ca-certificates
COPY bin/knirvchain /app/knirvchain
WORKDIR /app
EXPOSE 8080
CMD ["./knirvchain"]
CONTAINERFILE

echo "Building KNIRVGRAPH..."
podman build -t knirvgraph:latest -f - . << 'CONTAINERFILE'
FROM ubuntu:22.04
RUN apt-get update && apt-get install -y curl ca-certificates
COPY bin/knirvgraph /app/knirvgraph
WORKDIR /app
EXPOSE 8081
CMD ["./knirvgraph"]
CONTAINERFILE

echo "Building KNIRVNEXUS..."
podman build -t knirvnexus:latest -f - . << 'CONTAINERFILE'
FROM ubuntu:22.04
RUN apt-get update && apt-get install -y curl ca-certificates
COPY bin/knirvnexus /app/knirvnexus
WORKDIR /app
EXPOSE 8082 8083
CMD ["./knirvnexus"]
CONTAINERFILE

echo "Building KNIRVROUTER..."
podman build -t knirvrouter:latest -f - . << 'CONTAINERFILE'
FROM ubuntu:22.04
RUN apt-get update && apt-get install -y curl ca-certificates
COPY bin/knirvrouter /app/knirvrouter
WORKDIR /app
EXPOSE 8086 3478
CMD ["./knirvrouter"]
CONTAINERFILE

echo "Building KNIRVGATEWAY (using WASM runtime)..."
podman build -t knirvgateway:latest -f - . << 'CONTAINERFILE'
FROM ubuntu:22.04
RUN apt-get update && apt-get install -y curl ca-certificates
COPY bin/wasmtime /app/wasmtime
COPY bin/knirv-server.wasm /app/knirv-server.wasm
WORKDIR /app
EXPOSE 8087
CMD ["./wasmtime", "knirv-server.wasm"]
CONTAINERFILE

echo "Podman images built successfully"
EOF
    else
        print_status "Building images with Docker..."
        ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$TESTNET_IP" << 'EOF'
cd /opt/knirv-testnet

# Build KNIRV services using real binaries with Docker
echo "Building KNIRVORACLE..."
docker build -t knirvoracle:latest -f - . << 'DOCKERFILE'
FROM ubuntu:22.04
RUN apt-get update && apt-get install -y curl ca-certificates
COPY bin/knirvoracle /app/knirvoracle
WORKDIR /app
EXPOSE 1317 26656 26657
CMD ["./knirvoracle"]
DOCKERFILE

echo "Building KNIRVCHAIN..."
docker build -t knirvchain:latest -f - . << 'DOCKERFILE'
FROM ubuntu:22.04
RUN apt-get update && apt-get install -y curl ca-certificates
COPY bin/knirvchain /app/knirvchain
WORKDIR /app
EXPOSE 8080
CMD ["./knirvchain"]
DOCKERFILE

echo "Building KNIRVGRAPH..."
docker build -t knirvgraph:latest -f - . << 'DOCKERFILE'
FROM ubuntu:22.04
RUN apt-get update && apt-get install -y curl ca-certificates
COPY bin/knirvgraph /app/knirvgraph
WORKDIR /app
EXPOSE 8081
CMD ["./knirvgraph"]
DOCKERFILE

echo "Building KNIRVNEXUS..."
docker build -t knirvnexus:latest -f - . << 'DOCKERFILE'
FROM ubuntu:22.04
RUN apt-get update && apt-get install -y curl ca-certificates
COPY bin/knirvnexus /app/knirvnexus
WORKDIR /app
EXPOSE 8082 8083
CMD ["./knirvnexus"]
DOCKERFILE

echo "Building KNIRVROUTER..."
docker build -t knirvrouter:latest -f - . << 'DOCKERFILE'
FROM ubuntu:22.04
RUN apt-get update && apt-get install -y curl ca-certificates
COPY bin/knirvrouter /app/knirvrouter
WORKDIR /app
EXPOSE 8086 3478
CMD ["./knirvrouter"]
DOCKERFILE

echo "Building KNIRVGATEWAY (using WASM runtime)..."
docker build -t knirvgateway:latest -f - . << 'DOCKERFILE'
FROM ubuntu:22.04
RUN apt-get update && apt-get install -y curl ca-certificates
COPY bin/wasmtime /app/wasmtime
COPY bin/knirv-server.wasm /app/knirv-server.wasm
WORKDIR /app
EXPOSE 8087
CMD ["./wasmtime", "knirv-server.wasm"]
DOCKERFILE

echo "Docker images built successfully"
EOF
    fi

    print_success "Container images built on testnet server"
}

deploy_native_testnet() {
    print_step "Deploying KNIRVTESTNET in Native mode..."
    print_status "Installing Node.js and running npm scripts on server..."

    # Upload the entire KNIRVTESTNET directory and run npm scripts
    ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$TESTNET_IP" << 'EOF'
cd /opt/knirv-testnet

# Install Node.js if not already installed
if ! command -v node >/dev/null 2>&1; then
    echo "Installing Node.js..."
    curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
    sudo apt-get install -y nodejs
else
    echo "Node.js is already installed: $(node --version)"
fi

# Install npm dependencies
echo "Installing npm dependencies..."
npm install

# Load testnet endpoints
echo "Loading testnet endpoints..."
npm run load-endpoints:testnet

# Start the testnet services
echo "Starting KNIRVTESTNET services..."
npm start

# Check if services are running
echo "Checking service status..."
sleep 10
npm run testnet:status || echo "Status check completed"

echo "Native deployment completed!"
EOF

    print_success "Native KNIRVTESTNET deployment completed"
}

deploy_testnet_services() {
    print_step "Deploying KNIRVTESTNET services..."

    # Deploy services using the selected container runtime
    if [ "$CONTAINER_RUNTIME" = "native" ]; then
        deploy_native_testnet
    elif [ "$CONTAINER_RUNTIME" = "podman" ]; then
        print_status "Deploying with Podman..."
        ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$TESTNET_IP" << 'EOF'
cd /opt/knirv-testnet

# Stop any existing services
podman-compose -f podman-compose-prod.yml down || true

# Start services
podman-compose -f podman-compose-prod.yml up -d

# Wait for services to start
sleep 30

# Check service status
podman-compose -f podman-compose-prod.yml ps
EOF
    else
        print_status "Deploying with Docker..."
        # Pass the compose command to the remote server
        ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$TESTNET_IP" << EOF
cd /opt/knirv-testnet

# Determine Docker Compose command on remote server
if command -v docker-compose &> /dev/null; then
    COMPOSE_CMD="docker-compose"
elif docker compose version &> /dev/null; then
    COMPOSE_CMD="docker compose"
else
    echo "Error: Docker Compose not found on remote server"
    exit 1
fi

echo "Using Docker Compose command: \$COMPOSE_CMD"

# Stop any existing services
\$COMPOSE_CMD -f docker-compose-prod.yml down || true

# Start services
\$COMPOSE_CMD -f docker-compose-prod.yml up -d

# Wait for services to start
sleep 30

# Check service status
\$COMPOSE_CMD -f docker-compose-prod.yml ps
EOF
    fi

    print_success "KNIRVTESTNET services deployed"
}

verify_deployment() {
    print_step "Verifying testnet deployment with dynamic service discovery..."

    # Deploy the dynamic health check script
    print_status "Deploying dynamic health check script..."
    if scp -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no scripts/remote-health-check.sh ubuntu@"$TESTNET_IP":/tmp/remote-health-check.sh >/dev/null 2>&1; then
        ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$TESTNET_IP" "chmod +x /tmp/remote-health-check.sh" >/dev/null 2>&1
        print_success "Dynamic health check script deployed"
    else
        print_warning "Could not deploy dynamic health check script, using basic verification"

        # Fallback to basic health check
        echo ""
        echo -e "${YELLOW}Basic Service Health Check:${NC}"

        basic_services=(
            "ipfs:5001:/api/v0/version"
            "testnet-gateway:10000:/"
            "knirv-oracle:1317:/health"
        )

        for service in "${basic_services[@]}"; do
            name=$(echo "$service" | cut -d: -f1)
            port=$(echo "$service" | cut -d: -f2)
            endpoint=$(echo "$service" | cut -d: -f3)

            if curl -s -f "http://$TESTNET_IP:$port$endpoint" > /dev/null 2>&1; then
                echo -e "  ${GREEN}✅ $name ($port): HEALTHY${NC}"
            else
                echo -e "  ${RED}❌ $name ($port): UNHEALTHY${NC}"
            fi
        done

        print_success "Basic deployment verification completed"
        return
    fi

    # Run dynamic health check
    print_status "Running dynamic service discovery..."
    echo ""
    echo -e "${YELLOW}Dynamic Service Discovery Results:${NC}"

    if ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$TESTNET_IP" "/tmp/remote-health-check.sh --simple" 2>/dev/null; then
        print_success "Dynamic deployment verification completed"
    else
        print_warning "Dynamic health check failed, but deployment may still be successful"
        print_status "Services may still be starting up - check again in a few minutes"
    fi
}

update_cloudflare_dns() {
    print_step "Updating CloudFlare DNS records for healthy services..."

    # Check if CloudFlare credentials are available
    if [ -z "$CLOUDFLARE_API_TOKEN" ] && [ -z "$CLOUDFLARE_EMAIL" ]; then
        print_warning "CloudFlare credentials not found - skipping DNS updates"
        print_warning "To enable DNS updates, set either:"
        print_warning "  export CLOUDFLARE_API_TOKEN='your-token'"
        print_warning "  or export CLOUDFLARE_EMAIL='email' and CLOUDFLARE_API_KEY='key'"
        return 0
    fi

    # Wait for services to fully start before testing
    print_status "Waiting 30 seconds for services to fully initialize..."
    sleep 30

    # Run CloudFlare DNS update script
    if [ -f "./scripts/update-cloudflare-dns.sh" ]; then
        chmod +x "./scripts/update-cloudflare-dns.sh"

        print_status "Testing service health before DNS updates..."
        ./scripts/update-cloudflare-dns.sh test "$TESTNET_IP"

        echo ""
        print_status "Updating DNS records for healthy services..."
        ./scripts/update-cloudflare-dns.sh update "$TESTNET_IP"

        if [ $? -eq 0 ]; then
            print_success "CloudFlare DNS records updated successfully"
        else
            print_warning "Some DNS updates may have failed - check logs above"
        fi
    else
        print_warning "CloudFlare DNS update script not found - skipping DNS updates"
    fi
}

cleanup() {
    print_step "Cleaning up temporary files..."

    if [ -f /tmp/testnet_temp_dir ]; then
        TEMP_DIR=$(cat /tmp/testnet_temp_dir)
        rm -rf "$TEMP_DIR"
        rm -f /tmp/testnet_temp_dir
    fi

    print_success "Cleanup completed"
}

display_summary() {
    echo ""
    echo -e "${GREEN}🎉 KNIRVTESTNET Services Deployment Complete!${NC}"
    echo -e "${BLUE}=============================================${NC}"
    echo ""
    echo -e "${YELLOW}Testnet Information:${NC}"
    echo -e "  🌐 Public IP: $TESTNET_IP"
    echo -e "  🔗 Testnet URL: https://testnet.knirv.com"
    echo ""
    echo -e "${YELLOW}Service Endpoints (via NGINX Network Manager):${NC}"
    echo -e "  🌐 NGINX Manager: http://$TESTNET_IP:80"
    echo -e "  📊 IPFS API: http://$TESTNET_IP:5001"
    echo -e "  📊 IPFS Gateway: http://$TESTNET_IP:8081"
    echo -e "  🌐 Testnet Gateway: http://$TESTNET_IP:10000"
    echo -e "  🏛️  KNIRVORACLE: http://$TESTNET_IP:1317"
    echo -e "  ⛓️  KNIRVCHAIN: http://$TESTNET_IP:8090"
    echo -e "  📈 KNIRVGRAPH: http://$TESTNET_IP:8082"
    echo -e "  🤖 KNIRVNEXUS DVE: http://$TESTNET_IP:8084"
    echo -e "  🤖 KNIRVNEXUS VAL: http://$TESTNET_IP:8085"
    echo -e "  🌐 KNIRVROUTER: http://$TESTNET_IP:8086"
    echo -e "  🎮 KNIRVANA Game: http://$TESTNET_IP:3000"
    echo -e "  🌉 XION Bridge: http://$TESTNET_IP:8088"
    echo ""
    echo -e "${YELLOW}KNIRVCONTROLLER PWA:${NC}"
    echo -e "  📱 PWA Testnet: https://controller-testnet.knirv.network"
    echo -e "  📱 Android Download: https://controller-testnet.knirv.network/android"
    echo -e "  📱 iOS Download: https://controller-testnet.knirv.network/ios"
    echo ""
    echo -e "${YELLOW}Next Steps:${NC}"
    echo -e "  1. Check service health: ./check-testnet-health.sh"
    echo -e "  2. Deploy KNIRVCONTROLLER PWA: make deploy-controller-pwa ENVIRONMENT=testnet"
    echo -e "  3. Update frontend: make update-testnet-frontend"
    if [ "$CONTAINER_RUNTIME" = "native" ]; then
        echo -e "  3. Monitor services: ssh knirv-testnet 'ps aux | grep knirv'"
        echo -e "  4. View logs: ssh knirv-testnet 'tail -f /opt/knirv-testnet/logs/*.log'"
        echo -e "  5. Restart services: ssh knirv-testnet 'cd /opt/knirv-testnet && npm restart'"
    elif [ "$CONTAINER_RUNTIME" = "podman" ]; then
        echo -e "  3. Monitor services: ssh knirv-testnet 'podman ps'"
        echo -e "  4. View logs: ssh knirv-testnet 'podman-compose -f /opt/knirv-testnet/podman-compose-prod.yml logs'"
    else
        echo -e "  3. Monitor services: ssh knirv-testnet 'docker ps'"
        echo -e "  4. View logs: ssh knirv-testnet 'cd /opt/knirv-testnet && (\$COMPOSE_CMD -f docker-compose-prod.yml logs || docker-compose -f docker-compose-prod.yml logs || docker compose -f docker-compose-prod.yml logs)'"
        echo -e "     (Use whichever Docker Compose command is available on the server)"
    fi
    echo ""
    echo -e "${YELLOW}Health Check Options:${NC}"
    echo -e "  • Dynamic discovery: ./check-testnet-health.sh"
    echo -e "  • JSON output: ./check-testnet-health.sh --json"
    echo -e "  • Skip CloudFlare: ./check-testnet-health.sh --no-cloudflare"
    echo ""
}

# Main execution
main() {
    print_header

    check_prerequisites
    test_ssh_connection
    check_disk_space

    if [ "$CONTAINER_RUNTIME" = "native" ]; then
        # Native deployment: upload entire KNIRVTESTNET directory
        prepare_native_testnet_files
        upload_native_testnet_files
    else
        # Container deployment: prepare optimized files
        prepare_testnet_files
        upload_testnet_files
        install_node_dependencies
        build_container_images
    fi

    deploy_testnet_services
    verify_deployment
    update_cloudflare_dns
    cleanup
    display_summary
}

# Handle script arguments
case "${1:-}" in
    --help|-h)
        echo "Usage: $0 [options]"
        echo ""
        echo "Options:"
        echo "  --help, -h    Show this help message"
        echo "  --force       Skip confirmation prompts"
        echo ""
        echo "Features:"
        echo "  - Automatic detection of Docker and Podman"
        echo "  - Interactive selection between container runtimes and native deployment"
        echo "  - Native deployment option using npm scripts (like local testnet)"
        echo "  - Includes XION bridge and KNIRVANA components"
        echo "  - Checks for already running services"
        echo "  - Proper permission setup for both Docker and Podman"
        echo "  - Graceful handling of existing containers"
        echo ""
        echo "Deployment Options:"
        echo "  1. Docker - Traditional containerized deployment"
        echo "  2. Podman - Rootless containerized deployment"
        echo "  3. Native - Upload KNIRVTESTNET directory and run npm scripts"
        echo ""
        echo "Prerequisites:"
        echo "  - Testnet infrastructure must be deployed first"
        echo "  - SSH key must be available at ~/.ssh/AEGONG.pem"
        echo "  - KNIRVTESTNET directory must exist"
        echo "  - For container deployment: Docker or Podman must be installed"
        echo "  - For native deployment: Node.js will be installed automatically"
        exit 0
        ;;
    --force)
        print_warning "Force mode - skipping confirmations"
        ;;
    *)
        # Confirm deployment
        echo -e "${YELLOW}This will deploy KNIRVTESTNET services to AWS EC2.${NC}"
        echo -e "${YELLOW}The script will:${NC}"
        echo -e "${YELLOW}  - Detect available deployment options (Docker/Podman/Native)${NC}"
        echo -e "${YELLOW}  - Include XION bridge and KNIRVANA components${NC}"
        echo -e "${YELLOW}  - Check for existing services and handle them gracefully${NC}"
        echo -e "${YELLOW}  - Set up proper permissions for the selected deployment method${NC}"
        echo -e "${YELLOW}  - Deploy services using the chosen deployment method${NC}"
        echo ""
        read -p "Continue? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            echo "Deployment cancelled."
            exit 0
        fi
        ;;
esac

# Run main function
main
