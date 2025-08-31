#!/bin/bash

# KNIRVTESTNET Services Deployment Script
# Deploys KNIRVTESTNET services to AWS EC2 instance via Docker Compose

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
SSH_KEY="~/.ssh/knirv-testnet-key.pem"

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

check_prerequisites() {
    print_step "Checking prerequisites..."
    
    # Check if testnet IP file exists
    if [ ! -f "$TESTNET_IP_FILE" ]; then
        print_error "Testnet IP file not found at $TESTNET_IP_FILE"
        print_warning "Please run 'make deploy-testnet-infrastructure' first"
        exit 1
    fi
    
    # Get testnet IP
    TESTNET_IP=$(cat "$TESTNET_IP_FILE")
    if [ -z "$TESTNET_IP" ]; then
        print_error "Testnet IP is empty"
        exit 1
    fi
    
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

prepare_testnet_files() {
    print_step "Preparing testnet files for deployment..."
    
    # Create temporary directory for deployment files
    TEMP_DIR=$(mktemp -d)
    
    # Copy KNIRVTESTNET files to temp directory
    cp -r "$TESTNET_DIR"/* "$TEMP_DIR/"
    
    # Update docker-compose.yml for production deployment
    sed -i 's|build:|#build:|g' "$TEMP_DIR/docker-compose.yml"
    sed -i 's|context: \.\./|#context: ../|g' "$TEMP_DIR/docker-compose.yml"
    sed -i 's|dockerfile: Dockerfile|#dockerfile: Dockerfile|g' "$TEMP_DIR/docker-compose.yml"
    
    # Add image references for production
    cat >> "$TEMP_DIR/podman-compose-prod.yml" << 'EOF'
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
    
    print_success "Testnet files prepared in $TEMP_DIR"
    echo "$TEMP_DIR" > /tmp/testnet_temp_dir
}

upload_testnet_files() {
    print_step "Uploading testnet files to server..."
    
    TEMP_DIR=$(cat /tmp/testnet_temp_dir)
    
    # Create testnet directory on server
    ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$TESTNET_IP" \
        "sudo mkdir -p /opt/knirv-testnet && sudo chown ubuntu:ubuntu /opt/knirv-testnet"
    
    # Upload files
    scp -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no -r "$TEMP_DIR"/* ubuntu@"$TESTNET_IP":/opt/knirv-testnet/
    
    print_success "Files uploaded to testnet server"
}

build_podman_images() {
    print_step "Building Podman images on testnet server..."
    print_warning "Best practice is to build images in a CI/CD pipeline and pull from a registry."
    
    # Build images on the server using Podman
    ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$TESTNET_IP" << 'EOF'
cd /opt/knirv-testnet

# Build KNIRV services (using mock builds for now)
echo "Building KNIRVORACLE..."
podman build -t knirvoracle:latest -f - . << 'CONTAINERFILE'
FROM ubuntu:22.04
RUN apt-get update && apt-get install -y curl
COPY scripts/mock-knirvoracle.py /app/mock-knirvoracle.py
WORKDIR /app
EXPOSE 1317 26656 26657
CMD ["python3", "mock-knirvoracle.py"]
CONTAINERFILE

echo "Building KNIRVCHAIN..."
podman build -t knirvchain:latest -f - . << 'CONTAINERFILE'
FROM ubuntu:22.04
RUN apt-get update && apt-get install -y curl python3
COPY scripts/mock-knirvchain.py /app/mock-knirvchain.py
WORKDIR /app
EXPOSE 8080
CMD ["python3", "mock-knirvchain.py"]
CONTAINERFILE

echo "Building KNIRVGRAPH..."
podman build -t knirvgraph:latest -f - . << 'CONTAINERFILE'
FROM ubuntu:22.04
RUN apt-get update && apt-get install -y curl python3
COPY scripts/mock-knirvgraph.py /app/mock-knirvgraph.py
WORKDIR /app
EXPOSE 8081
CMD ["python3", "mock-knirvgraph.py"]
CONTAINERFILE

echo "Building KNIRVNEXUS..."
podman build -t knirvnexus:latest -f - . << 'CONTAINERFILE'
FROM ubuntu:22.04
RUN apt-get update && apt-get install -y curl python3
COPY scripts/mock-knirvnexus.py /app/mock-knirvnexus.py
WORKDIR /app
EXPOSE 8082 8083
CMD ["python3", "mock-knirvnexus.py"]
CONTAINERFILE

echo "Building KNIRVROUTER..."
podman build -t knirvrouter:latest -f - . << 'CONTAINERFILE'
FROM ubuntu:22.04
RUN apt-get update && apt-get install -y curl python3
COPY scripts/mock-knirvrouter.py /app/mock-knirvrouter.py
WORKDIR /app
EXPOSE 8086
CMD ["python3", "mock-knirvrouter.py"]
CONTAINERFILE

echo "Building KNIRVGATEWAY..."
podman build -t knirvgateway:latest -f - . << 'CONTAINERFILE'
FROM ubuntu:22.04
RUN apt-get update && apt-get install -y curl python3
COPY scripts/mock-knirvgateway.py /app/mock-knirvgateway.py
WORKDIR /app
EXPOSE 8087
CMD ["python3", "mock-knirvgateway.py"]
CONTAINERFILE

echo "Podman images built successfully"
EOF
    
    print_success "Podman images built on testnet server"
}

deploy_testnet_services() {
    print_step "Deploying KNIRVTESTNET services..."
    
    # Deploy services using podman-compose
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
    
    print_success "KNIRVTESTNET services deployed"
}

verify_deployment() {
    print_step "Verifying testnet deployment..."
    
    # Test service endpoints
    services=(
        "ipfs:5001:/api/v0/version"
        "knirvchain:8080:/health"
        "knirvgraph:8081:/health"
        "knirvnexus-1:8082:/health"
        "knirvnexus-2:8083:/health"
        "knirvrouter:8086:/health"
        "knirvgateway:8087:/health"
        "knirvoracle:1317:/health"
    )
    
    echo ""
    echo -e "${YELLOW}Service Health Check:${NC}"
    
    for service in "${services[@]}"; do
        name=$(echo "$service" | cut -d: -f1)
        port=$(echo "$service" | cut -d: -f2)
        endpoint=$(echo "$service" | cut -d: -f3)
        
        if curl -s -f "http://$TESTNET_IP:$port$endpoint" > /dev/null 2>&1; then
            echo -e "  ${GREEN}✅ $name ($port): HEALTHY${NC}"
        else
            echo -e "  ${RED}❌ $name ($port): UNHEALTHY${NC}"
        fi
    done
    
    print_success "Deployment verification completed"
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
    echo -e "${YELLOW}Service Endpoints:${NC}"
    echo -e "  📊 IPFS API: http://$TESTNET_IP:5001"
    echo -e "  📊 IPFS Gateway: http://$TESTNET_IP:8080"
    echo -e "  ⛓️  KNIRVCHAIN: http://$TESTNET_IP:8080"
    echo -e "  📈 KNIRVGRAPH: http://$TESTNET_IP:8081"
    echo -e "  🤖 KNIRVNEXUS-1: http://$TESTNET_IP:8082"
    echo -e "  🤖 KNIRVNEXUS-2: http://$TESTNET_IP:8083"
    echo -e "  🌐 KNIRVROUTER: http://$TESTNET_IP:8086"
    echo -e "  🚪 KNIRVGATEWAY: http://$TESTNET_IP:8087"
    echo -e "  🏛️  KNIRVORACLE: http://$TESTNET_IP:1317"
    echo ""
    echo -e "${YELLOW}Next Steps:${NC}"
    echo -e "  1. Update frontend: make update-testnet-frontend"
    echo -e "  2. Monitor services: ssh knirv-testnet 'podman ps'"
    echo -e "  3. View logs: ssh knirv-testnet 'podman-compose -f /opt/knirv-testnet/podman-compose-prod.yml logs'"
    echo ""
}

# Main execution
main() {
    print_header
    
    check_prerequisites
    test_ssh_connection
    prepare_testnet_files
    upload_testnet_files
    build_podman_images
    deploy_testnet_services
    verify_deployment
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
        echo "Prerequisites:"
        echo "  - Testnet infrastructure must be deployed first"
        echo "  - SSH key must be available at ~/.ssh/knirv-testnet-key.pem"
        echo "  - KNIRVTESTNET directory must exist"
        exit 0
        ;;
    --force)
        print_warning "Force mode - skipping confirmations"
        ;;
    *)
        # Confirm deployment
        echo -e "${YELLOW}This will deploy KNIRVTESTNET services to AWS EC2.${NC}"
        echo -e "${YELLOW}This will stop any existing services and redeploy.${NC}"
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
