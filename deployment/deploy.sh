#!/bin/bash

set -e

# KNIRV D-TEN Production Deployment Script
# Months 14-18 Implementation
# Includes KNIRVORACLE deployment integration

echo "🚀 Starting KNIRV D-TEN Production Deployment..."

# Configuration
NAMESPACE="knirv-production"
DOCKER_REGISTRY="knirv"
VERSION="v1.0.0"
MONITORING_ENABLED=true
BACKUP_ENABLED=true

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

# Check prerequisites
check_prerequisites() {
    log_step "Checking prerequisites..."

    # Check if kubectl is installed
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is not installed. Please install kubectl first."
        exit 1
    fi

    # Check if docker is installed
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed. Please install Docker first."
        exit 1
    fi

    # Check if helm is installed (optional)
    if ! command -v helm &> /dev/null; then
        log_warn "Helm is not installed. Some features may not be available."
    fi

    # Check kubectl connection
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot connect to Kubernetes cluster. Please check your kubeconfig."
        exit 1
    fi

    log_info "Prerequisites check completed successfully."
}

# Create namespace
create_namespace() {
    log_step "Creating Kubernetes namespace..."

    if kubectl get namespace "$NAMESPACE" &> /dev/null; then
        log_info "Namespace $NAMESPACE already exists."
    else
        kubectl create namespace "$NAMESPACE"
        log_info "Created namespace $NAMESPACE."
    fi

    # Label namespace for monitoring
    kubectl label namespace "$NAMESPACE" monitoring=enabled --overwrite
}

# Deploy secrets
deploy_secrets() {
    log_step "Deploying secrets..."

    # Create blockchain secrets
    kubectl create secret generic blockchain-secrets \
        --namespace="$NAMESPACE" \
        --from-literal=xion-rpc-url="https://rpc.xion.burnt.com:443" \
        --dry-run=client -o yaml | kubectl apply -f -

    # Create database secrets
    kubectl create secret generic database-secrets \
        --namespace="$NAMESPACE" \
        --from-literal=connection-string="postgresql://knirv:knirv123@postgres:5432/knirv" \
        --dry-run=client -o yaml | kubectl apply -f -

    log_info "Secrets deployed successfully."
}

# Build and push Docker images
build_and_push_images() {
    log_step "Building and pushing Docker images..."

    # List of services to build
    services=("knirvchain" "knirvgraph" "knirvnexus" "knirvoracle" "knirvrouter")

    for service in "${services[@]}"; do
        log_info "Building $service..."

        # Build Docker image
        docker build -t "$DOCKER_REGISTRY/$service:$VERSION" "./$service/"

        # Push to registry (if registry is configured)
        if [ "$DOCKER_REGISTRY" != "knirv" ]; then
            docker push "$DOCKER_REGISTRY/$service:$VERSION"
            log_info "Pushed $service:$VERSION to registry."
        else
            log_info "Built $service:$VERSION locally."
        fi
    done

    log_info "All images built successfully."
}

# Deploy KNIRV stack
deploy_knirv_stack() {
    log_step "Deploying KNIRV production stack..."

    # Apply optimization configuration
    kubectl apply -f deployment/production-config/optimization.yaml

    log_info "KNIRV stack deployed successfully."
}

# Deploy monitoring stack
deploy_monitoring() {
    if [ "$MONITORING_ENABLED" = true ]; then
        log_step "Deploying monitoring stack..."

        # Create monitoring namespace
        kubectl create namespace knirv-monitoring --dry-run=client -o yaml | kubectl apply -f -

        # Deploy Prometheus
        kubectl create configmap prometheus-config \
            --namespace=knirv-monitoring \
            --from-file=deployment/monitoring/prometheus.yml \
            --dry-run=client -o yaml | kubectl apply -f -

        kubectl create configmap prometheus-rules \
            --namespace=knirv-monitoring \
            --from-file=deployment/monitoring/alert_rules.yml \
            --dry-run=client -o yaml | kubectl apply -f -

        # Deploy Alertmanager
        kubectl create configmap alertmanager-config \
            --namespace=knirv-monitoring \
            --from-file=deployment/monitoring/alertmanager.yml \
            --dry-run=client -o yaml | kubectl apply -f -

        # Start monitoring stack with Docker Compose
        docker-compose -f deployment/docker-compose.monitoring.yml up -d

        log_info "Monitoring stack deployed successfully."
        log_info "Grafana available at: http://localhost:3000 (admin/admin123)"
        log_info "Prometheus available at: http://localhost:9090"
        log_info "Alertmanager available at: http://localhost:9093"
    else
        log_info "Monitoring deployment skipped."
    fi
}

# Verify deployment
verify_deployment() {
    log_step "Verifying deployment..."

    # Wait for pods to be ready
    log_info "Waiting for pods to be ready..."
    kubectl wait --for=condition=ready pod -l app=knirv-stack --namespace="$NAMESPACE" --timeout=300s

    # Check service status
    log_info "Checking service status..."
    kubectl get pods,services,ingress --namespace="$NAMESPACE"

    # Run health checks
    log_info "Running health checks..."
    sleep 30  # Wait for services to fully start

    # Check if services are responding
    if kubectl get service knirv-service --namespace="$NAMESPACE" &> /dev/null; then
        log_info "✓ KNIRV services are running"
    else
        log_error "✗ KNIRV services are not running properly"
        return 1
    fi

    log_info "Deployment verification completed successfully."
}

# Run final tests
run_final_tests() {
    log_step "Running final test suite..."

    if [ -f "deployment/testing/final-test-suite.sh" ]; then
        # Make sure test data directory exists
        mkdir -p deployment/testing/test-data

        # Run the test suite
        bash deployment/testing/final-test-suite.sh

        if [ $? -eq 0 ]; then
            log_info "✅ All tests passed! KNIRV D-TEN is ready for production."
        else
            log_error "❌ Some tests failed. Please review and fix issues before production use."
            return 1
        fi
    else
        log_warn "Final test suite not found. Skipping tests."
    fi
}

# Backup current deployment (if exists)
backup_deployment() {
    if [ "$BACKUP_ENABLED" = true ]; then
        log_step "Creating backup of current deployment..."

        BACKUP_DIR="backups/$(date +%Y%m%d_%H%M%S)"
        mkdir -p "$BACKUP_DIR"

        # Backup Kubernetes resources
        kubectl get all --namespace="$NAMESPACE" -o yaml > "$BACKUP_DIR/kubernetes-resources.yaml" 2>/dev/null || true

        # Backup configurations
        cp -r deployment/production-config "$BACKUP_DIR/" 2>/dev/null || true

        log_info "Backup created in $BACKUP_DIR"
    fi
}

# Rollback function
rollback_deployment() {
    log_error "Deployment failed. Rolling back..."

    # Delete the namespace to clean up
    kubectl delete namespace "$NAMESPACE" --ignore-not-found=true

    # Stop monitoring stack
    docker-compose -f deployment/docker-compose.monitoring.yml down 2>/dev/null || true

    log_info "Rollback completed."
}

# Main deployment function
main() {
    log_info "KNIRV D-TEN Production Deployment - Months 14-18"
    log_info "=================================================="

    # Set trap for cleanup on error
    trap rollback_deployment ERR

    # Run deployment steps
    check_prerequisites
    backup_deployment
    create_namespace
    deploy_secrets
    build_and_push_images
    deploy_knirv_stack
    deploy_monitoring
    verify_deployment
    run_final_tests

    log_info ""
    log_info "🎉 KNIRV D-TEN Production Deployment Completed Successfully!"
    log_info "=================================================="
    log_info "Services are now running in namespace: $NAMESPACE"
    log_info "Monitoring dashboard: http://localhost:3000"
    log_info "API Gateway: Check your ingress configuration"
    log_info ""
    log_info "Next steps:"
    log_info "1. Configure DNS for your domain"
    log_info "2. Set up SSL certificates"
    log_info "3. Configure monitoring alerts"
    log_info "4. Set up backup procedures"
    log_info "5. Review security settings"
}

# Deploy KNIRVORACLE
deploy_knirvoracle() {
    log_step "Deploying KNIRVORACLE..."

    # Check if KNIRVORACLE deployment script exists
    if [[ -f "ansible/deploy-knirvoracle.sh" ]]; then
        log_info "Running KNIRVORACLE deployment script..."
        cd ansible
        ./deploy-knirvoracle.sh deploy --env production
        cd ..
        log_info "✅ KNIRVORACLE deployment completed"
    else
        log_warn "KNIRVORACLE deployment script not found, skipping..."
    fi
}

# Deploy infrastructure
deploy_infrastructure() {
    log_step "Deploying KNIRV Network Infrastructure..."

    # Check if infrastructure deployment script exists
    if [[ -f "ansible/deploy-infrastructure.sh" ]]; then
        log_info "Running infrastructure deployment script..."
        cd ansible
        ./deploy-infrastructure.sh --env production
        cd ..
        log_info "✅ Infrastructure deployment completed"
    else
        log_warn "Infrastructure deployment script not found, using KNIRVORACLE script..."
        deploy_knirvoracle
    fi
}

# Handle command line arguments
case "${1:-deploy}" in
    "deploy")
        main
        ;;
    "test")
        run_final_tests
        ;;
    "monitoring")
        deploy_monitoring
        ;;
    "rollback")
        rollback_deployment
        ;;
    "verify")
        verify_deployment
        ;;
    "knirvoracle")
        deploy_knirvoracle
        ;;
    "infrastructure")
        deploy_infrastructure
        ;;
    *)
        echo "Usage: $0 [deploy|test|monitoring|rollback|verify|knirvoracle|infrastructure]"
        echo "  deploy         - Full deployment (default)"
        echo "  test           - Run final test suite only"
        echo "  monitoring     - Deploy monitoring stack only"
        echo "  rollback       - Rollback deployment"
        echo "  verify         - Verify current deployment"
        echo "  knirvoracle    - Deploy KNIRVORACLE only"
        echo "  infrastructure - Deploy infrastructure + KNIRVORACLE"
        exit 1
        ;;
esac
