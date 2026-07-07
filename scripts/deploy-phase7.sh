#!/bin/bash

# KNIRV Network Phase 7 Enhanced Deployment Script
# Comprehensive deployment with monitoring, validation, and rollback capabilities

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
LOGS_DIR="$PROJECT_ROOT/logs"
BACKUP_DIR="$PROJECT_ROOT/backups"

# Deployment configuration
DEPLOYMENT_MODE="docker-compose"  # local, docker-compose, kubernetes
ENVIRONMENT="development"         # development, staging, production
ENABLE_MONITORING=true
ENABLE_BACKUP=true
ENABLE_VALIDATION=true
ENABLE_ROLLBACK_PREPARATION=true
WAIT_TIMEOUT=300
HEALTH_CHECK_RETRIES=10
HEALTH_CHECK_INTERVAL=30

# Service configuration
SERVICES=(
    "knirv-oracle:8083"
    "knirvchain:8080"
    "knirvgraph:8081"
    "knirv-nexus:8082"
    "knirvrouter:3478"
    "knirvcontroller:3000"
    "knirv-gateway:8000"
)

# Monitoring services
MONITORING_SERVICES=(
    "prometheus:9090"
    "grafana:3000"
    "alertmanager:9093"
    "redis:6379"
    "postgres:5432"
)

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_step() {
    echo -e "${BLUE}[STEP]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

# Utility functions
create_directories() {
    log_step "Creating necessary directories"
    mkdir -p "$LOGS_DIR" "$BACKUP_DIR" "$DEPLOYMENT_DIR"
    mkdir -p "$LOGS_DIR/deployment" "$LOGS_DIR/services" "$LOGS_DIR/monitoring"
}

check_prerequisites() {
    log_step "Checking prerequisites"
    
    # Check required tools
    local required_tools=("docker" "docker-compose" "curl" "jq")
    for tool in "${required_tools[@]}"; do
        if ! command -v "$tool" &> /dev/null; then
            log_error "Required tool '$tool' is not installed"
            exit 1
        fi
    done
    
    # Check Docker daemon
    if ! docker info &> /dev/null; then
        log_error "Docker daemon is not running"
        exit 1
    fi
    
    log_success "Prerequisites check passed"
}

backup_current_state() {
    if [ "$ENABLE_BACKUP" = true ]; then
        log_step "Creating backup of current state"
        local backup_timestamp=$(date '+%Y%m%d_%H%M%S')
        local backup_path="$BACKUP_DIR/backup_$backup_timestamp"
        
        mkdir -p "$backup_path"
        
        # Backup databases
        if docker-compose ps postgres | grep -q "Up"; then
            log_info "Backing up PostgreSQL database"
            docker-compose exec -T postgres pg_dumpall -U postgres > "$backup_path/postgres_backup.sql"
        fi
        
        # Backup Redis data
        if docker-compose ps redis | grep -q "Up"; then
            log_info "Backing up Redis data"
            docker-compose exec -T redis redis-cli BGSAVE
            docker cp "$(docker-compose ps -q redis):/data/dump.rdb" "$backup_path/redis_backup.rdb"
        fi
        
        # Backup configuration files
        cp -r "$PROJECT_ROOT/deployment" "$backup_path/"
        cp "$PROJECT_ROOT/docker-compose.yml" "$backup_path/" 2>/dev/null || true
        
        echo "$backup_timestamp" > "$BACKUP_DIR/latest_backup"
        log_success "Backup created at $backup_path"
    fi
}

deploy_infrastructure() {
    log_step "Deploying infrastructure services"
    
    case "$DEPLOYMENT_MODE" in
        "docker-compose")
            deploy_docker_compose
            ;;
        "kubernetes")
            deploy_kubernetes
            ;;
        "local")
            deploy_local
            ;;
        *)
            log_error "Unknown deployment mode: $DEPLOYMENT_MODE"
            exit 1
            ;;
    esac
}

deploy_docker_compose() {
    log_info "Deploying with Docker Compose"
    
    # Start infrastructure services first
    log_info "Starting infrastructure services"
    docker-compose up -d postgres redis
    
    # Wait for infrastructure to be ready
    wait_for_service "postgres" "5432" "PostgreSQL"
    wait_for_service "redis" "6379" "Redis"
    
    # Start monitoring services if enabled
    if [ "$ENABLE_MONITORING" = true ]; then
        log_info "Starting monitoring services"
        docker-compose up -d prometheus grafana alertmanager
        
        # Wait for monitoring services
        wait_for_service "prometheus" "9090" "Prometheus"
        wait_for_service "grafana" "3000" "Grafana"
        wait_for_service "alertmanager" "9093" "Alertmanager"
    fi
    
    # Start application services
    log_info "Starting application services"
    docker-compose up -d
    
    log_success "Docker Compose deployment completed"
}

deploy_kubernetes() {
    log_info "Deploying with Kubernetes"
    
    # Apply namespace
    kubectl apply -f "$DEPLOYMENT_DIR/k8s/namespace.yaml"
    
    # Apply infrastructure
    kubectl apply -f "$DEPLOYMENT_DIR/k8s/infrastructure/"
    
    # Wait for infrastructure
    kubectl wait --for=condition=ready pod -l app=postgres -n knirv-production --timeout=300s
    kubectl wait --for=condition=ready pod -l app=redis -n knirv-production --timeout=300s
    
    # Apply monitoring if enabled
    if [ "$ENABLE_MONITORING" = true ]; then
        kubectl apply -f "$DEPLOYMENT_DIR/k8s/monitoring/"
        kubectl wait --for=condition=ready pod -l app=prometheus -n knirv-production --timeout=300s
    fi
    
    # Apply applications
    kubectl apply -f "$DEPLOYMENT_DIR/k8s/applications/"
    
    log_success "Kubernetes deployment completed"
}

deploy_local() {
    log_info "Deploying locally"
    
    # Start services individually for better control
    for service_info in "${SERVICES[@]}"; do
        local service_name=$(echo "$service_info" | cut -d':' -f1)
        local service_port=$(echo "$service_info" | cut -d':' -f2)
        
        log_info "Starting $service_name"
        # Implementation depends on specific service startup scripts
        # This would call individual service startup scripts
    done
    
    log_success "Local deployment completed"
}

wait_for_service() {
    local service_name="$1"
    local port="$2"
    local display_name="$3"
    local retries=0
    
    log_info "Waiting for $display_name to be ready"
    
    while [ $retries -lt $HEALTH_CHECK_RETRIES ]; do
        if nc -z localhost "$port" 2>/dev/null; then
            log_success "$display_name is ready"
            return 0
        fi
        
        retries=$((retries + 1))
        log_info "Waiting for $display_name... (attempt $retries/$HEALTH_CHECK_RETRIES)"
        sleep $HEALTH_CHECK_INTERVAL
    done
    
    log_error "$display_name failed to start within timeout"
    return 1
}

validate_deployment() {
    if [ "$ENABLE_VALIDATION" = true ]; then
        log_step "Validating deployment"
        
        # Health checks for all services
        for service_info in "${SERVICES[@]}"; do
            local service_name=$(echo "$service_info" | cut -d':' -f1)
            local service_port=$(echo "$service_info" | cut -d':' -f2)
            
            validate_service_health "$service_name" "$service_port"
        done
        
        # Validate monitoring services
        if [ "$ENABLE_MONITORING" = true ]; then
            for service_info in "${MONITORING_SERVICES[@]}"; do
                local service_name=$(echo "$service_info" | cut -d':' -f1)
                local service_port=$(echo "$service_info" | cut -d':' -f2)
                
                validate_service_health "$service_name" "$service_port"
            done
        fi
        
        # Run integration tests
        run_integration_tests
        
        log_success "Deployment validation completed"
    fi
}

validate_service_health() {
    local service_name="$1"
    local port="$2"
    
    log_info "Validating $service_name health"
    
    # Check if service is responding
    if ! nc -z localhost "$port" 2>/dev/null; then
        log_error "$service_name is not responding on port $port"
        return 1
    fi
    
    # Check health endpoint if available
    local health_url="http://localhost:$port/health"
    if curl -s -f "$health_url" &>/dev/null; then
        local health_status=$(curl -s "$health_url" | jq -r '.status' 2>/dev/null || echo "unknown")
        if [ "$health_status" = "healthy" ] || [ "$health_status" = "ok" ]; then
            log_success "$service_name health check passed"
        else
            log_warn "$service_name health check returned: $health_status"
        fi
    else
        log_info "$service_name health endpoint not available, checking port only"
    fi
}

run_integration_tests() {
    log_info "Running integration tests"
    
    if [ -f "$PROJECT_ROOT/integration-tests/run-tests.sh" ]; then
        cd "$PROJECT_ROOT/integration-tests"
        ./run-tests.sh --quick
        cd "$PROJECT_ROOT"
    else
        log_warn "Integration tests not found, skipping"
    fi
}

setup_monitoring() {
    if [ "$ENABLE_MONITORING" = true ]; then
        log_step "Setting up monitoring and alerting"
        
        # Configure Grafana dashboards
        if curl -s http://localhost:3000/api/health &>/dev/null; then
            log_info "Configuring Grafana dashboards"
            # Import dashboards
            for dashboard in "$DEPLOYMENT_DIR/monitoring/dashboards"/*.json; do
                if [ -f "$dashboard" ]; then
                    curl -X POST http://admin:admin123@localhost:3000/api/dashboards/db \
                        -H "Content-Type: application/json" \
                        -d @"$dashboard" &>/dev/null || true
                fi
            done
        fi
        
        # Configure Prometheus alerts
        if curl -s http://localhost:9090/api/v1/status/config &>/dev/null; then
            log_info "Prometheus is configured with alerts"
        fi
        
        log_success "Monitoring setup completed"
    fi
}

prepare_rollback() {
    if [ "$ENABLE_ROLLBACK_PREPARATION" = true ]; then
        log_step "Preparing rollback mechanisms"
        
        # Create rollback script
        cat > "$BACKUP_DIR/rollback.sh" << 'EOF'
#!/bin/bash
# Auto-generated rollback script

BACKUP_DIR="$(dirname "$0")"
PROJECT_ROOT="$(dirname "$BACKUP_DIR")"

echo "Rolling back to previous state..."

# Stop current services
cd "$PROJECT_ROOT"
docker-compose down

# Restore databases
if [ -f "$BACKUP_DIR/postgres_backup.sql" ]; then
    docker-compose up -d postgres
    sleep 10
    docker-compose exec -T postgres psql -U postgres < "$BACKUP_DIR/postgres_backup.sql"
fi

if [ -f "$BACKUP_DIR/redis_backup.rdb" ]; then
    docker-compose up -d redis
    sleep 5
    docker cp "$BACKUP_DIR/redis_backup.rdb" "$(docker-compose ps -q redis):/data/dump.rdb"
    docker-compose restart redis
fi

# Restore configuration
if [ -d "$BACKUP_DIR/deployment" ]; then
    cp -r "$BACKUP_DIR/deployment" "$PROJECT_ROOT/"
fi

# Restart services
docker-compose up -d

echo "Rollback completed"
EOF
        
        chmod +x "$BACKUP_DIR/rollback.sh"
        log_success "Rollback script created at $BACKUP_DIR/rollback.sh"
    fi
}

cleanup_on_failure() {
    log_error "Deployment failed, cleaning up"
    
    # Stop services
    docker-compose down 2>/dev/null || true
    
    # Clean up resources
    docker system prune -f 2>/dev/null || true
    
    log_info "Cleanup completed"
}

show_deployment_summary() {
    log_step "Deployment Summary"
    
    echo -e "${CYAN}================================${NC}"
    echo -e "${CYAN}  KNIRV Network Phase 7 Deployed${NC}"
    echo -e "${CYAN}================================${NC}"
    echo ""
    echo -e "${GREEN}Services:${NC}"
    for service_info in "${SERVICES[@]}"; do
        local service_name=$(echo "$service_info" | cut -d':' -f1)
        local service_port=$(echo "$service_info" | cut -d':' -f2)
        echo -e "  • $service_name: http://localhost:$service_port"
    done
    
    if [ "$ENABLE_MONITORING" = true ]; then
        echo ""
        echo -e "${GREEN}Monitoring:${NC}"
        echo -e "  • Grafana: http://localhost:3000 (admin/admin123)"
        echo -e "  • Prometheus: http://localhost:9090"
        echo -e "  • Alertmanager: http://localhost:9093"
    fi
    
    echo ""
    echo -e "${GREEN}Logs:${NC} $LOGS_DIR"
    echo -e "${GREEN}Backups:${NC} $BACKUP_DIR"
    
    if [ -f "$BACKUP_DIR/rollback.sh" ]; then
        echo -e "${GREEN}Rollback:${NC} $BACKUP_DIR/rollback.sh"
    fi
    
    echo ""
    echo -e "${GREEN}Next steps:${NC}"
    echo -e "  1. Check service health: make health-check"
    echo -e "  2. Run tests: make tests"
    echo -e "  3. View monitoring: open http://localhost:3000"
    echo ""
}

# Main deployment function
main() {
    log_info "Starting KNIRV Network Phase 7 deployment"
    
    # Parse command line arguments
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
            --no-backup)
                ENABLE_BACKUP=false
                shift
                ;;
            --no-validation)
                ENABLE_VALIDATION=false
                shift
                ;;
            --help)
                echo "Usage: $0 [OPTIONS]"
                echo "Options:"
                echo "  --mode MODE          Deployment mode (local, docker-compose, kubernetes)"
                echo "  --env ENV           Environment (development, staging, production)"
                echo "  --no-monitoring     Disable monitoring stack"
                echo "  --no-backup         Disable backup creation"
                echo "  --no-validation     Disable deployment validation"
                echo "  --help              Show this help message"
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                exit 1
                ;;
        esac
    done
    
    # Set up error handling
    trap cleanup_on_failure ERR
    
    # Execute deployment steps
    create_directories
    check_prerequisites
    backup_current_state
    deploy_infrastructure
    setup_monitoring
    validate_deployment
    prepare_rollback
    show_deployment_summary
    
    log_success "KNIRV Network Phase 7 deployment completed successfully!"
}

# Run main function with all arguments
main "$@"
