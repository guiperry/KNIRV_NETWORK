#!/bin/bash

# KNIRV D-TEN Production Deployment and Real-Network Testing Integration Script
# This script integrates the new production deployment with existing testing infrastructure

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
DEPLOYMENT_DIR="$PROJECT_ROOT/deployment"
INTEGRATION_TESTS_DIR="$PROJECT_ROOT/integration-tests"

# Deployment modes
DEPLOYMENT_MODE="local"  # local, kubernetes, docker-compose
ENVIRONMENT="development"  # development, staging, production
ENABLE_MONITORING=true
RUN_TESTS=true
CLEANUP_ON_FAILURE=true
WAIT_FOR_SERVICES=true
TEST_TIMEOUT="30m"

# Service endpoints
GATEWAY_URL="http://localhost:8000"
MONITORING_URL="http://localhost:3000"

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

log_header() {
    echo -e "${PURPLE}╔══════════════════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${PURPLE}║${NC} $1 ${PURPLE}║${NC}"
    echo -e "${PURPLE}╚══════════════════════════════════════════════════════════════════════════════╝${NC}"
}

# Function to show usage
show_usage() {
    echo "KNIRV D-TEN Production Deployment and Testing Integration"
    echo ""
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Deployment Options:"
    echo "  --mode MODE              Deployment mode: local, kubernetes, docker-compose (default: local)"
    echo "  --env ENVIRONMENT        Environment: development, staging, production (default: development)"
    echo "  --no-monitoring          Disable monitoring stack deployment"
    echo "  --no-tests               Skip running tests after deployment"
    echo "  --no-cleanup             Don't cleanup on failure"
    echo "  --no-wait                Don't wait for services to be ready"
    echo ""
    echo "Testing Options:"
    echo "  --test-timeout DURATION  Test timeout (default: 30m)"
    echo "  --integration-only       Run integration tests only"
    echo "  --production-tests       Run production test suite only"
    echo "  --comprehensive          Run both integration and production tests"
    echo ""
    echo "Service Options:"
    echo "  --gateway-url URL        Gateway URL (default: http://localhost:8000)"
    echo "  --monitoring-url URL     Monitoring URL (default: http://localhost:3000)"
    echo ""
    echo "Control Options:"
    echo "  --deploy-only            Deploy services without testing"
    echo "  --test-only              Run tests against existing deployment"
    echo "  --teardown               Teardown existing deployment"
    echo "  --status                 Check deployment status"
    echo ""
    echo "Examples:"
    echo "  $0                                    # Deploy locally and run all tests"
    echo "  $0 --mode kubernetes --env staging   # Deploy to Kubernetes staging"
    echo "  $0 --test-only --comprehensive       # Run all tests against existing deployment"
    echo "  $0 --deploy-only --no-monitoring     # Deploy without monitoring or tests"
    echo "  $0 --teardown                        # Teardown existing deployment"
}

# Function to check prerequisites
check_prerequisites() {
    log_step "Checking prerequisites for deployment mode: $DEPLOYMENT_MODE"

    case $DEPLOYMENT_MODE in
        "kubernetes")
            if ! command -v kubectl &> /dev/null; then
                log_error "kubectl is required for Kubernetes deployment"
                exit 1
            fi
            if ! kubectl cluster-info &> /dev/null; then
                log_error "Cannot connect to Kubernetes cluster"
                exit 1
            fi
            ;;
        "docker-compose")
            if ! command -v docker-compose &> /dev/null && ! command -v docker &> /dev/null; then
                log_error "Docker and docker-compose are required"
                exit 1
            fi
            ;;
        "local")
            if ! command -v go &> /dev/null; then
                log_error "Go is required for local deployment"
                exit 1
            fi
            ;;
    esac

    if [ "$RUN_TESTS" = true ]; then
        if [ ! -d "$INTEGRATION_TESTS_DIR" ]; then
            log_error "Integration tests directory not found: $INTEGRATION_TESTS_DIR"
            exit 1
        fi
    fi

    log_info "Prerequisites check passed"
}

# Function to deploy services based on mode
deploy_services() {
    log_step "Deploying KNIRV services in $DEPLOYMENT_MODE mode"

    case $DEPLOYMENT_MODE in
        "kubernetes")
            deploy_kubernetes
            ;;
        "docker-compose")
            deploy_docker_compose
            ;;
        "local")
            deploy_local
            ;;
        *)
            log_error "Unknown deployment mode: $DEPLOYMENT_MODE"
            exit 1
            ;;
    esac

    log_info "Service deployment completed"
}

# Function to deploy to Kubernetes
deploy_kubernetes() {
    log_info "Deploying to Kubernetes cluster..."

    if [ ! -f "$DEPLOYMENT_DIR/deploy.sh" ]; then
        log_error "Kubernetes deployment script not found"
        exit 1
    fi

    cd "$DEPLOYMENT_DIR"
    
    local deploy_args=""
    if [ "$ENABLE_MONITORING" = false ]; then
        deploy_args="$deploy_args --no-monitoring"
    fi

    ./deploy.sh deploy $deploy_args

    if [ "$WAIT_FOR_SERVICES" = true ]; then
        wait_for_kubernetes_services
    fi
}

# Function to deploy with Docker Compose
deploy_docker_compose() {
    log_info "Deploying with Docker Compose..."

    if [ ! -f "$DEPLOYMENT_DIR/docker-compose.knirv-production.yml" ]; then
        log_error "KNIRV production Docker Compose file not found"
        exit 1
    fi

    cd "$DEPLOYMENT_DIR"

    # Start KNIRV production stack (includes IPFS)
    log_info "Starting KNIRV production services with IPFS..."
    docker-compose -f docker-compose.knirv-production.yml up -d

    # Start monitoring stack
    if [ "$ENABLE_MONITORING" = true ]; then
        if [ -f "docker-compose.monitoring.yml" ]; then
            docker-compose -f docker-compose.monitoring.yml up -d
        else
            log_warn "Monitoring compose file not found, skipping monitoring"
        fi
    fi

    if [ "$WAIT_FOR_SERVICES" = true ]; then
        wait_for_docker_services
    fi
}

# Function to deploy locally
deploy_local() {
    log_info "Deploying services locally..."

    # Setup IPFS for production if not already configured
    if [ ! -f "./scripts/setup-ipfs-production.sh" ]; then
        log_error "IPFS setup script not found"
        exit 1
    fi

    log_info "Setting up IPFS for KNIRV production network..."
    ./scripts/setup-ipfs-production.sh

    # Use existing manage-knirv.sh script (now includes IPFS)
    "$SCRIPT_DIR/manage-knirv.sh" start all --background

    if [ "$ENABLE_MONITORING" = true ]; then
        log_info "Starting monitoring stack..."
        cd "$DEPLOYMENT_DIR"
        docker-compose -f docker-compose.monitoring.yml up -d
    fi

    if [ "$WAIT_FOR_SERVICES" = true ]; then
        wait_for_local_services
    fi
}

# Function to wait for Kubernetes services
wait_for_kubernetes_services() {
    log_step "Waiting for Kubernetes services to be ready..."

    local namespace="knirv-production"
    local timeout=300

    kubectl wait --for=condition=ready pod -l app=knirv-stack --namespace="$namespace" --timeout="${timeout}s"

    log_info "Kubernetes services are ready"
}

# Function to wait for Docker services
wait_for_docker_services() {
    log_step "Waiting for Docker services to be ready..."

    local timeout=300
    local start_time=$(date +%s)

    while true; do
        local current_time=$(date +%s)
        local elapsed=$((current_time - start_time))

        if [ $elapsed -gt $timeout ]; then
            log_error "Timeout waiting for Docker services"
            exit 1
        fi

        if check_service_health "$GATEWAY_URL/health" "Gateway"; then
            break
        fi

        sleep 10
    done

    log_info "Docker services are ready"
}

# Function to wait for local services
wait_for_local_services() {
    log_step "Waiting for local services to be ready..."

    # Use existing health check from manage-knirv.sh
    local max_attempts=30
    local attempt=0

    while [ $attempt -lt $max_attempts ]; do
        if "$SCRIPT_DIR/manage-knirv.sh" health &> /dev/null; then
            log_info "Local services are ready"
            return 0
        fi

        attempt=$((attempt + 1))
        log_info "Waiting for services... (attempt $attempt/$max_attempts)"
        sleep 10
    done

    log_error "Timeout waiting for local services"
    exit 1
}

# Function to check service health
check_service_health() {
    local url=$1
    local service_name=$2

    if curl -s -f "$url" > /dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

# Function to run integration tests
run_integration_tests() {
    log_step "Running integration tests..."

    cd "$INTEGRATION_TESTS_DIR"

    # Use existing integration test runner
    local test_args="--timeout $TEST_TIMEOUT"
    if [ "$DEPLOYMENT_MODE" != "local" ]; then
        test_args="$test_args --skip-setup --no-teardown"
    fi

    if "$SCRIPT_DIR/run-integration-tests.sh" $test_args; then
        log_info "Integration tests passed"
        return 0
    else
        log_error "Integration tests failed"
        return 1
    fi
}

# Function to run production tests
run_production_tests() {
    log_step "Running production test suite..."

    if [ ! -f "$DEPLOYMENT_DIR/testing/final-test-suite.sh" ]; then
        log_error "Production test suite not found"
        return 1
    fi

    cd "$DEPLOYMENT_DIR/testing"

    # Set gateway URL for production tests
    export GATEWAY_URL="$GATEWAY_URL"

    if ./final-test-suite.sh; then
        log_info "Production tests passed"
        return 0
    else
        log_error "Production tests failed"
        return 1
    fi
}

# Function to run comprehensive tests
run_comprehensive_tests() {
    log_step "Running comprehensive test suite..."

    local integration_result=0
    local production_result=0

    # Run integration tests
    run_integration_tests || integration_result=$?

    # Run production tests
    run_production_tests || production_result=$?

    if [ $integration_result -eq 0 ] && [ $production_result -eq 0 ]; then
        log_info "All comprehensive tests passed"
        return 0
    else
        log_error "Some comprehensive tests failed (Integration: $integration_result, Production: $production_result)"
        return 1
    fi
}

# Function to teardown deployment
teardown_deployment() {
    log_step "Tearing down deployment..."

    case $DEPLOYMENT_MODE in
        "kubernetes")
            cd "$DEPLOYMENT_DIR"
            ./deploy.sh rollback
            ;;
        "docker-compose")
            cd "$DEPLOYMENT_DIR"
            docker-compose -f docker-compose.monitoring.yml down
            ;;
        "local")
            "$SCRIPT_DIR/manage-knirv.sh" stop all
            if [ "$ENABLE_MONITORING" = true ]; then
                cd "$DEPLOYMENT_DIR"
                docker-compose -f docker-compose.monitoring.yml down
            fi
            ;;
    esac

    log_info "Teardown completed"
}

# Function to check deployment status
check_deployment_status() {
    log_step "Checking deployment status..."

    case $DEPLOYMENT_MODE in
        "kubernetes")
            kubectl get pods,services,ingress --namespace=knirv-production
            ;;
        "docker-compose")
            cd "$DEPLOYMENT_DIR"
            docker-compose -f docker-compose.monitoring.yml ps
            ;;
        "local")
            "$SCRIPT_DIR/manage-knirv.sh" status all
            ;;
    esac
}

# Function to cleanup on failure
cleanup_on_failure() {
    if [ "$CLEANUP_ON_FAILURE" = true ]; then
        log_warn "Deployment failed, cleaning up..."
        teardown_deployment
    fi
}

# Parse command line arguments
DEPLOY_ONLY=false
TEST_ONLY=false
TEARDOWN_ONLY=false
STATUS_ONLY=false
INTEGRATION_ONLY=false
PRODUCTION_TESTS_ONLY=false
COMPREHENSIVE_TESTS=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --mode)
            DEPLOYMENT_MODE="$2"
            shift 2
            ;;
        --env)
            ENVIRONMENT="$2"
            shift 2
            ;;
        --no-monitoring)
            ENABLE_MONITORING=false
            shift
            ;;
        --no-tests)
            RUN_TESTS=false
            shift
            ;;
        --no-cleanup)
            CLEANUP_ON_FAILURE=false
            shift
            ;;
        --no-wait)
            WAIT_FOR_SERVICES=false
            shift
            ;;
        --test-timeout)
            TEST_TIMEOUT="$2"
            shift 2
            ;;
        --gateway-url)
            GATEWAY_URL="$2"
            shift 2
            ;;
        --monitoring-url)
            MONITORING_URL="$2"
            shift 2
            ;;
        --deploy-only)
            DEPLOY_ONLY=true
            RUN_TESTS=false
            shift
            ;;
        --test-only)
            TEST_ONLY=true
            shift
            ;;
        --teardown)
            TEARDOWN_ONLY=true
            shift
            ;;
        --status)
            STATUS_ONLY=true
            shift
            ;;
        --integration-only)
            INTEGRATION_ONLY=true
            RUN_TESTS=true
            shift
            ;;
        --production-tests)
            PRODUCTION_TESTS_ONLY=true
            RUN_TESTS=true
            shift
            ;;
        --comprehensive)
            COMPREHENSIVE_TESTS=true
            RUN_TESTS=true
            shift
            ;;
        --help)
            show_usage
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Set trap for cleanup on failure
trap cleanup_on_failure ERR

# Main execution
main() {
    log_header "KNIRV D-TEN Production Deployment and Testing Integration"

    log_info "Configuration:"
    log_info "  Deployment Mode: $DEPLOYMENT_MODE"
    log_info "  Environment: $ENVIRONMENT"
    log_info "  Enable Monitoring: $ENABLE_MONITORING"
    log_info "  Run Tests: $RUN_TESTS"
    log_info "  Gateway URL: $GATEWAY_URL"

    # Handle special modes
    if [ "$TEARDOWN_ONLY" = true ]; then
        teardown_deployment
        exit 0
    fi

    if [ "$STATUS_ONLY" = true ]; then
        check_deployment_status
        exit 0
    fi

    # Check prerequisites
    check_prerequisites

    # Deploy services unless test-only mode
    if [ "$TEST_ONLY" = false ]; then
        deploy_services
    fi

    # Run tests if enabled
    if [ "$RUN_TESTS" = true ]; then
        local test_result=0

        if [ "$INTEGRATION_ONLY" = true ]; then
            run_integration_tests || test_result=$?
        elif [ "$PRODUCTION_TESTS_ONLY" = true ]; then
            run_production_tests || test_result=$?
        elif [ "$COMPREHENSIVE_TESTS" = true ]; then
            run_comprehensive_tests || test_result=$?
        else
            # Default: run both integration and production tests
            run_comprehensive_tests || test_result=$?
        fi

        if [ $test_result -ne 0 ]; then
            log_error "Tests failed"
            exit $test_result
        fi
    fi

    log_header "Deployment and Testing Completed Successfully"
    log_info "Services are running and tests have passed"
    log_info "Gateway: $GATEWAY_URL"
    if [ "$ENABLE_MONITORING" = true ]; then
        log_info "Monitoring: $MONITORING_URL"
    fi
}

# Run main function
main "$@"
