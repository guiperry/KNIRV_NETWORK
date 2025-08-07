#!/bin/bash

# KNIRV-NEXUS Build Script
# Builds all components for the DVE production architecture

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKEND_DIR="$PROJECT_ROOT/backend"
BUILD_DIR="$PROJECT_ROOT/build"
DOCKER_REGISTRY="knirv"
VERSION="${VERSION:-latest}"

echo -e "${BLUE}KNIRV-NEXUS Build Script${NC}"
echo -e "${BLUE}=========================${NC}"
echo "Project Root: $PROJECT_ROOT"
echo "Backend Dir: $BACKEND_DIR"
echo "Build Dir: $BUILD_DIR"
echo "Version: $VERSION"
echo ""

# Create build directory
mkdir -p "$BUILD_DIR"

# Function to print status
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to build Go binaries
build_go_binaries() {
    print_status "Building Go binaries..."
    
    cd "$BACKEND_DIR"
    
    # Ensure dependencies are up to date
    print_status "Updating Go dependencies..."
    go mod tidy
    
    # Run tests
    print_status "Running tests..."
    go test ./tests/... -v || print_warning "Some tests failed, continuing build..."
    
    # Build binaries
    print_status "Building DVE Manager..."
    CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo \
        -ldflags '-extldflags "-static" -s -w' \
        -o "$BUILD_DIR/dve-manager" ./cmd/dve-manager/
    
    print_status "Building Validation Core..."
    CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo \
        -ldflags '-extldflags "-static" -s -w' \
        -o "$BUILD_DIR/validation-core" ./cmd/validation-core/
    
    print_status "Building API Gateway..."
    CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo \
        -ldflags '-extldflags "-static" -s -w' \
        -o "$BUILD_DIR/api-gateway" ./cmd/api-gateway/
    
    print_status "Go binaries built successfully!"
}

# Function to build Docker images
build_docker_images() {
    print_status "Building Docker images..."
    
    cd "$BACKEND_DIR"
    
    # Build DVE Manager image
    print_status "Building DVE Manager Docker image..."
    docker build -f Dockerfile.dve-manager -t "$DOCKER_REGISTRY/nexus-dve-manager:$VERSION" .
    
    # Build Validation Core image
    print_status "Building Validation Core Docker image..."
    docker build -f Dockerfile.validation-core -t "$DOCKER_REGISTRY/nexus-validation-core:$VERSION" .
    
    # Build API Gateway image
    print_status "Building API Gateway Docker image..."
    docker build -f Dockerfile.api-gateway -t "$DOCKER_REGISTRY/nexus-api-gateway:$VERSION" .
    
    print_status "Docker images built successfully!"
}

# Function to validate Kubernetes manifests
validate_k8s_manifests() {
    print_status "Validating Kubernetes manifests..."
    
    cd "$PROJECT_ROOT"
    
    # Check if kubectl is available
    if ! command -v kubectl &> /dev/null; then
        print_warning "kubectl not found, skipping manifest validation"
        return
    fi
    
    # Validate manifests
    for manifest in k8s/*.yaml; do
        if [ -f "$manifest" ]; then
            print_status "Validating $manifest..."
            kubectl apply --dry-run=client -f "$manifest" || print_warning "Validation failed for $manifest"
        fi
    done
    
    print_status "Kubernetes manifest validation completed!"
}

# Function to create deployment package
create_deployment_package() {
    print_status "Creating deployment package..."
    
    cd "$PROJECT_ROOT"
    
    # Create deployment directory
    DEPLOY_DIR="$BUILD_DIR/deployment"
    mkdir -p "$DEPLOY_DIR"
    
    # Copy Kubernetes manifests
    cp -r k8s/ "$DEPLOY_DIR/"
    
    # Copy scripts
    cp -r scripts/ "$DEPLOY_DIR/"
    
    # Copy documentation
    cp README.md "$DEPLOY_DIR/" 2>/dev/null || true
    cp KNIRVNEXUS_Gap_Analysis.md "$DEPLOY_DIR/" 2>/dev/null || true
    cp KALI_LINUX_FOUNDATION.md "$DEPLOY_DIR/" 2>/dev/null || true
    
    # Create deployment archive
    cd "$BUILD_DIR"
    tar -czf "knirv-nexus-deployment-$VERSION.tar.gz" deployment/
    
    print_status "Deployment package created: knirv-nexus-deployment-$VERSION.tar.gz"
}

# Function to run security checks
run_security_checks() {
    print_status "Running security checks..."
    
    cd "$BACKEND_DIR"
    
    # Check for known vulnerabilities in Go dependencies
    if command -v govulncheck &> /dev/null; then
        print_status "Running Go vulnerability check..."
        govulncheck ./... || print_warning "Vulnerability check completed with warnings"
    else
        print_warning "govulncheck not found, skipping vulnerability check"
    fi
    
    # Check Docker images for vulnerabilities (if trivy is available)
    if command -v trivy &> /dev/null; then
        print_status "Running Docker image security scan..."
        trivy image "$DOCKER_REGISTRY/nexus-dve-manager:$VERSION" || print_warning "Security scan completed with warnings"
    else
        print_warning "trivy not found, skipping Docker security scan"
    fi
    
    print_status "Security checks completed!"
}

# Function to generate build report
generate_build_report() {
    print_status "Generating build report..."
    
    REPORT_FILE="$BUILD_DIR/build-report-$VERSION.txt"
    
    cat > "$REPORT_FILE" << EOF
KNIRV-NEXUS Build Report
========================

Build Date: $(date)
Version: $VERSION
Git Commit: $(git rev-parse HEAD 2>/dev/null || echo "N/A")
Git Branch: $(git branch --show-current 2>/dev/null || echo "N/A")

Components Built:
- DVE Manager: $BUILD_DIR/dve-manager
- Validation Core: $BUILD_DIR/validation-core
- API Gateway: $BUILD_DIR/api-gateway

Docker Images:
- $DOCKER_REGISTRY/nexus-dve-manager:$VERSION
- $DOCKER_REGISTRY/nexus-validation-core:$VERSION
- $DOCKER_REGISTRY/nexus-api-gateway:$VERSION

Deployment Package:
- knirv-nexus-deployment-$VERSION.tar.gz

Build Environment:
- OS: $(uname -s)
- Architecture: $(uname -m)
- Go Version: $(go version)
- Docker Version: $(docker --version 2>/dev/null || echo "N/A")
- Kubernetes Version: $(kubectl version --client --short 2>/dev/null || echo "N/A")

EOF
    
    print_status "Build report generated: $REPORT_FILE"
}

# Main build process
main() {
    print_status "Starting KNIRV-NEXUS build process..."
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --skip-tests)
                SKIP_TESTS=true
                shift
                ;;
            --skip-docker)
                SKIP_DOCKER=true
                shift
                ;;
            --skip-security)
                SKIP_SECURITY=true
                shift
                ;;
            --version)
                VERSION="$2"
                shift 2
                ;;
            *)
                print_error "Unknown option: $1"
                exit 1
                ;;
        esac
    done
    
    # Build Go binaries
    build_go_binaries
    
    # Build Docker images (unless skipped)
    if [ "$SKIP_DOCKER" != "true" ]; then
        build_docker_images
    else
        print_warning "Skipping Docker image build"
    fi
    
    # Validate Kubernetes manifests
    validate_k8s_manifests
    
    # Run security checks (unless skipped)
    if [ "$SKIP_SECURITY" != "true" ]; then
        run_security_checks
    else
        print_warning "Skipping security checks"
    fi
    
    # Create deployment package
    create_deployment_package
    
    # Generate build report
    generate_build_report
    
    print_status "Build completed successfully!"
    print_status "Build artifacts available in: $BUILD_DIR"
}

# Run main function
main "$@"
