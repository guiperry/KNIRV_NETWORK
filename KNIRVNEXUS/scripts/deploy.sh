#!/bin/bash

# KNIRV-NEXUS Deployment Script
# Deploys the DVE production architecture to Kubernetes

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
K8S_DIR="$PROJECT_ROOT/k8s"
NAMESPACE="knirv-nexus"
VERSION="${VERSION:-latest}"

echo -e "${BLUE}KNIRV-NEXUS Deployment Script${NC}"
echo -e "${BLUE}==============================${NC}"
echo "Project Root: $PROJECT_ROOT"
echo "Kubernetes Dir: $K8S_DIR"
echo "Namespace: $NAMESPACE"
echo "Version: $VERSION"
echo ""

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

# Function to check prerequisites
check_prerequisites() {
    print_status "Checking prerequisites..."
    
    # Check if kubectl is available
    if ! command -v kubectl &> /dev/null; then
        print_error "kubectl is required but not installed"
        exit 1
    fi
    
    # Check if we can connect to Kubernetes cluster
    if ! kubectl cluster-info &> /dev/null; then
        print_error "Cannot connect to Kubernetes cluster"
        exit 1
    fi
    
    # Check if Podman is available (for rootless containers)
    if ! command -v podman &> /dev/null; then
        print_warning "Podman not found - using Docker instead"
    fi
    
    print_status "Prerequisites check completed!"
}

# Function to create namespace and basic resources
setup_namespace() {
    print_status "Setting up namespace and basic resources..."
    
    cd "$K8S_DIR"
    
    # Apply namespace configuration
    kubectl apply -f namespace.yaml
    
    # Wait for namespace to be ready
    kubectl wait --for=condition=Active namespace/$NAMESPACE --timeout=60s
    
    print_status "Namespace setup completed!"
}

# Function to create secrets
setup_secrets() {
    print_status "Setting up secrets..."
    
    cd "$K8S_DIR"
    
    # Check if secrets already exist
    if kubectl get secret knirv-nexus-secrets -n $NAMESPACE &> /dev/null; then
        print_warning "Secrets already exist, skipping creation"
        return
    fi
    
    # Apply secrets
    kubectl apply -f secrets.yaml
    
    print_status "Secrets setup completed!"
}

# Function to create config maps
setup_configmaps() {
    print_status "Setting up config maps..."
    
    cd "$K8S_DIR"
    
    # Apply config maps
    kubectl apply -f configmap.yaml
    
    print_status "Config maps setup completed!"
}

# Function to deploy DVE Manager
deploy_dve_manager() {
    print_status "Deploying DVE Manager..."
    
    cd "$K8S_DIR"
    
    # Apply DVE Manager deployment
    kubectl apply -f dve-manager-deployment.yaml
    
    # Wait for deployment to be ready
    kubectl wait --for=condition=available deployment/dve-manager -n $NAMESPACE --timeout=300s
    
    print_status "DVE Manager deployed successfully!"
}

# Function to deploy Validation Core
deploy_validation_core() {
    print_status "Deploying Validation Core..."
    
    cd "$K8S_DIR"
    
    # Apply Validation Core deployment
    kubectl apply -f validation-core-deployment.yaml
    
    # Wait for deployment to be ready
    kubectl wait --for=condition=available deployment/validation-core -n $NAMESPACE --timeout=300s
    
    print_status "Validation Core deployed successfully!"
}

# Function to deploy API Gateway
deploy_api_gateway() {
    print_status "Deploying API Gateway..."
    
    cd "$K8S_DIR"
    
    # Apply API Gateway deployment
    kubectl apply -f api-gateway-deployment.yaml
    
    # Wait for deployment to be ready
    kubectl wait --for=condition=available deployment/api-gateway -n $NAMESPACE --timeout=300s
    
    print_status "API Gateway deployed successfully!"
}

# Function to verify deployment
verify_deployment() {
    print_status "Verifying deployment..."
    
    # Check pod status
    print_status "Pod status:"
    kubectl get pods -n $NAMESPACE
    
    # Check service status
    print_status "Service status:"
    kubectl get services -n $NAMESPACE
    
    # Check ingress status
    print_status "Ingress status:"
    kubectl get ingress -n $NAMESPACE 2>/dev/null || print_warning "No ingress resources found"
    
    # Check HPA status
    print_status "HPA status:"
    kubectl get hpa -n $NAMESPACE 2>/dev/null || print_warning "No HPA resources found"
    
    # Run health checks
    print_status "Running health checks..."
    
    # Check DVE Manager health
    if kubectl get pods -n $NAMESPACE -l app=dve-manager -o jsonpath='{.items[0].status.phase}' | grep -q "Running"; then
        print_status "DVE Manager is running"
    else
        print_warning "DVE Manager may not be healthy"
    fi
    
    # Check Validation Core health
    if kubectl get pods -n $NAMESPACE -l app=validation-core -o jsonpath='{.items[0].status.phase}' | grep -q "Running"; then
        print_status "Validation Core is running"
    else
        print_warning "Validation Core may not be healthy"
    fi
    
    # Check API Gateway health
    if kubectl get pods -n $NAMESPACE -l app=api-gateway -o jsonpath='{.items[0].status.phase}' | grep -q "Running"; then
        print_status "API Gateway is running"
    else
        print_warning "API Gateway may not be healthy"
    fi
    
    print_status "Deployment verification completed!"
}

# Function to show access information
show_access_info() {
    print_status "Access Information:"
    
    # Get API Gateway service info
    API_GATEWAY_IP=$(kubectl get service api-gateway-service -n $NAMESPACE -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "Pending")
    API_GATEWAY_PORT=$(kubectl get service api-gateway-service -n $NAMESPACE -o jsonpath='{.spec.ports[0].port}')
    
    echo ""
    echo -e "${BLUE}API Gateway:${NC}"
    echo "  External IP: $API_GATEWAY_IP"
    echo "  Port: $API_GATEWAY_PORT"
    echo "  URL: http://$API_GATEWAY_IP:$API_GATEWAY_PORT"
    
    # Get ingress info if available
    INGRESS_HOST=$(kubectl get ingress api-gateway-ingress -n $NAMESPACE -o jsonpath='{.spec.rules[0].host}' 2>/dev/null || echo "Not configured")
    if [ "$INGRESS_HOST" != "Not configured" ]; then
        echo "  Ingress URL: https://$INGRESS_HOST"
    fi
    
    echo ""
    echo -e "${BLUE}Useful Commands:${NC}"
    echo "  View logs: kubectl logs -f deployment/api-gateway -n $NAMESPACE"
    echo "  Port forward: kubectl port-forward service/api-gateway-service 8080:80 -n $NAMESPACE"
    echo "  Scale deployment: kubectl scale deployment api-gateway --replicas=3 -n $NAMESPACE"
    echo "  Delete deployment: kubectl delete namespace $NAMESPACE"
    echo ""
}

# Function to cleanup deployment
cleanup_deployment() {
    print_status "Cleaning up deployment..."
    
    # Delete namespace (this will delete all resources in the namespace)
    kubectl delete namespace $NAMESPACE --ignore-not-found=true
    
    # Wait for namespace to be deleted
    kubectl wait --for=delete namespace/$NAMESPACE --timeout=120s 2>/dev/null || true
    
    print_status "Cleanup completed!"
}

# Function to update deployment
update_deployment() {
    print_status "Updating deployment..."
    
    # Update image tags to new version
    kubectl set image deployment/dve-manager dve-manager=knirv/nexus-dve-manager:$VERSION -n $NAMESPACE
    kubectl set image deployment/validation-core validation-core=knirv/nexus-validation-core:$VERSION -n $NAMESPACE
    kubectl set image deployment/api-gateway api-gateway=knirv/nexus-api-gateway:$VERSION -n $NAMESPACE
    
    # Wait for rollout to complete
    kubectl rollout status deployment/dve-manager -n $NAMESPACE --timeout=300s
    kubectl rollout status deployment/validation-core -n $NAMESPACE --timeout=300s
    kubectl rollout status deployment/api-gateway -n $NAMESPACE --timeout=300s
    
    print_status "Update completed!"
}

# Main deployment process
main() {
    print_status "Starting KNIRV-NEXUS deployment process..."
    
    # Parse command line arguments
    OPERATION="deploy"
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --cleanup)
                OPERATION="cleanup"
                shift
                ;;
            --update)
                OPERATION="update"
                shift
                ;;
            --verify-only)
                OPERATION="verify"
                shift
                ;;
            --version)
                VERSION="$2"
                shift 2
                ;;
            --namespace)
                NAMESPACE="$2"
                shift 2
                ;;
            *)
                print_error "Unknown option: $1"
                echo "Usage: $0 [--cleanup|--update|--verify-only] [--version VERSION] [--namespace NAMESPACE]"
                exit 1
                ;;
        esac
    done
    
    # Check prerequisites
    check_prerequisites
    
    case $OPERATION in
        "cleanup")
            cleanup_deployment
            ;;
        "update")
            update_deployment
            verify_deployment
            show_access_info
            ;;
        "verify")
            verify_deployment
            show_access_info
            ;;
        "deploy")
            setup_namespace
            setup_secrets
            setup_configmaps
            deploy_dve_manager
            deploy_validation_core
            deploy_api_gateway
            verify_deployment
            show_access_info
            ;;
    esac
    
    print_status "Deployment process completed successfully!"
}

# Run main function
main "$@"
