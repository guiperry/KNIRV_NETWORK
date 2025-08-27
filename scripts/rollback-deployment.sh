#!/bin/bash

# KNIRV Network Phase 7 Rollback Mechanism
# Safe rollback procedures for deployment failures or issues

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
BACKUP_DIR="$PROJECT_ROOT/backups"
LOGS_DIR="$PROJECT_ROOT/logs"

# Rollback configuration
ROLLBACK_MODE="auto"  # auto, manual, emergency
BACKUP_ID=""
FORCE_ROLLBACK=false
PRESERVE_DATA=true
VALIDATE_ROLLBACK=true
ROLLBACK_TIMEOUT=600

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
check_prerequisites() {
    log_step "Checking rollback prerequisites"
    
    # Check if backup directory exists
    if [ ! -d "$BACKUP_DIR" ]; then
        log_error "Backup directory not found: $BACKUP_DIR"
        exit 1
    fi
    
    # Check for available backups
    if [ -z "$(ls -A "$BACKUP_DIR" 2>/dev/null)" ]; then
        log_error "No backups found in $BACKUP_DIR"
        exit 1
    fi
    
    # Check required tools
    local required_tools=("docker" "docker-compose" "curl" "jq")
    for tool in "${required_tools[@]}"; do
        if ! command -v "$tool" &> /dev/null; then
            log_error "Required tool '$tool' is not installed"
            exit 1
        fi
    done
    
    log_success "Prerequisites check passed"
}

list_available_backups() {
    log_step "Available backups:"
    
    local backup_count=0
    for backup_dir in "$BACKUP_DIR"/backup_*; do
        if [ -d "$backup_dir" ]; then
            local backup_name=$(basename "$backup_dir")
            local backup_date=$(echo "$backup_name" | sed 's/backup_//' | sed 's/_/ /')
            local backup_size=$(du -sh "$backup_dir" | cut -f1)
            
            echo -e "  ${CYAN}$backup_name${NC} - $backup_date ($backup_size)"
            backup_count=$((backup_count + 1))
        fi
    done
    
    if [ $backup_count -eq 0 ]; then
        log_error "No backups found"
        exit 1
    fi
    
    # Show latest backup
    if [ -f "$BACKUP_DIR/latest_backup" ]; then
        local latest=$(cat "$BACKUP_DIR/latest_backup")
        echo -e "\n  ${GREEN}Latest backup: backup_$latest${NC}"
    fi
}

select_backup() {
    if [ -n "$BACKUP_ID" ]; then
        log_info "Using specified backup: $BACKUP_ID"
        return
    fi
    
    case "$ROLLBACK_MODE" in
        "auto")
            # Use latest backup automatically
            if [ -f "$BACKUP_DIR/latest_backup" ]; then
                BACKUP_ID="backup_$(cat "$BACKUP_DIR/latest_backup")"
                log_info "Auto-selected latest backup: $BACKUP_ID"
            else
                log_error "No latest backup marker found"
                exit 1
            fi
            ;;
        "manual")
            # Interactive backup selection
            list_available_backups
            echo ""
            read -p "Enter backup ID (e.g., backup_20250827_120000): " BACKUP_ID
            ;;
        "emergency")
            # Use latest backup without confirmation
            if [ -f "$BACKUP_DIR/latest_backup" ]; then
                BACKUP_ID="backup_$(cat "$BACKUP_DIR/latest_backup")"
                log_warn "Emergency rollback using latest backup: $BACKUP_ID"
            else
                log_error "No latest backup available for emergency rollback"
                exit 1
            fi
            ;;
    esac
    
    # Validate backup exists
    if [ ! -d "$BACKUP_DIR/$BACKUP_ID" ]; then
        log_error "Backup not found: $BACKUP_DIR/$BACKUP_ID"
        exit 1
    fi
}

create_pre_rollback_backup() {
    if [ "$PRESERVE_DATA" = true ]; then
        log_step "Creating pre-rollback backup"
        
        local pre_rollback_timestamp=$(date '+%Y%m%d_%H%M%S')
        local pre_rollback_path="$BACKUP_DIR/pre_rollback_$pre_rollback_timestamp"
        
        mkdir -p "$pre_rollback_path"
        
        # Backup current databases
        if docker-compose ps postgres | grep -q "Up"; then
            log_info "Backing up current PostgreSQL state"
            docker-compose exec -T postgres pg_dumpall -U postgres > "$pre_rollback_path/postgres_current.sql" 2>/dev/null || true
        fi
        
        if docker-compose ps redis | grep -q "Up"; then
            log_info "Backing up current Redis state"
            docker-compose exec -T redis redis-cli BGSAVE 2>/dev/null || true
            docker cp "$(docker-compose ps -q redis):/data/dump.rdb" "$pre_rollback_path/redis_current.rdb" 2>/dev/null || true
        fi
        
        # Backup current configuration
        cp -r "$PROJECT_ROOT/deployment" "$pre_rollback_path/" 2>/dev/null || true
        cp "$PROJECT_ROOT/docker-compose.yml" "$pre_rollback_path/" 2>/dev/null || true
        
        log_success "Pre-rollback backup created at $pre_rollback_path"
    fi
}

stop_current_services() {
    log_step "Stopping current services"
    
    # Graceful shutdown with timeout
    timeout $ROLLBACK_TIMEOUT docker-compose down --timeout 30 || {
        log_warn "Graceful shutdown timed out, forcing stop"
        docker-compose kill
        docker-compose down --volumes
    }
    
    # Clean up orphaned containers
    docker container prune -f 2>/dev/null || true
    
    log_success "Current services stopped"
}

restore_databases() {
    local backup_path="$BACKUP_DIR/$BACKUP_ID"
    
    log_step "Restoring databases from backup"
    
    # Start infrastructure services
    docker-compose up -d postgres redis
    
    # Wait for services to be ready
    log_info "Waiting for database services to start"
    sleep 30
    
    # Restore PostgreSQL
    if [ -f "$backup_path/postgres_backup.sql" ]; then
        log_info "Restoring PostgreSQL database"
        
        # Drop existing connections and recreate database
        docker-compose exec -T postgres psql -U postgres -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = 'knirv';" 2>/dev/null || true
        docker-compose exec -T postgres dropdb -U postgres knirv 2>/dev/null || true
        docker-compose exec -T postgres createdb -U postgres knirv 2>/dev/null || true
        
        # Restore from backup
        docker-compose exec -T postgres psql -U postgres < "$backup_path/postgres_backup.sql"
        
        log_success "PostgreSQL database restored"
    else
        log_warn "No PostgreSQL backup found in $backup_path"
    fi
    
    # Restore Redis
    if [ -f "$backup_path/redis_backup.rdb" ]; then
        log_info "Restoring Redis data"
        
        # Stop Redis, replace data file, restart
        docker-compose stop redis
        docker cp "$backup_path/redis_backup.rdb" "$(docker-compose ps -q redis):/data/dump.rdb"
        docker-compose start redis
        
        log_success "Redis data restored"
    else
        log_warn "No Redis backup found in $backup_path"
    fi
}

restore_configuration() {
    local backup_path="$BACKUP_DIR/$BACKUP_ID"
    
    log_step "Restoring configuration files"
    
    # Restore deployment configuration
    if [ -d "$backup_path/deployment" ]; then
        log_info "Restoring deployment configuration"
        cp -r "$backup_path/deployment" "$PROJECT_ROOT/"
        log_success "Deployment configuration restored"
    fi
    
    # Restore docker-compose.yml
    if [ -f "$backup_path/docker-compose.yml" ]; then
        log_info "Restoring docker-compose.yml"
        cp "$backup_path/docker-compose.yml" "$PROJECT_ROOT/"
        log_success "Docker Compose configuration restored"
    fi
    
    # Restore environment files
    if [ -f "$backup_path/.env" ]; then
        log_info "Restoring environment configuration"
        cp "$backup_path/.env" "$PROJECT_ROOT/"
        log_success "Environment configuration restored"
    fi
}

restart_services() {
    log_step "Restarting services with restored configuration"
    
    # Start services in order
    log_info "Starting infrastructure services"
    docker-compose up -d postgres redis
    
    # Wait for infrastructure
    sleep 30
    
    log_info "Starting application services"
    docker-compose up -d
    
    # Wait for services to stabilize
    log_info "Waiting for services to stabilize"
    sleep 60
    
    log_success "Services restarted"
}

validate_rollback() {
    if [ "$VALIDATE_ROLLBACK" = true ]; then
        log_step "Validating rollback"
        
        local validation_failed=false
        
        # Check service health
        local services=("knirv-oracle:8083" "knirvchain:8080" "knirvgraph:8081" "knirv-nexus:8082")
        
        for service_info in "${services[@]}"; do
            local service_name=$(echo "$service_info" | cut -d':' -f1)
            local service_port=$(echo "$service_info" | cut -d':' -f2)
            
            log_info "Checking $service_name health"
            
            if ! nc -z localhost "$service_port" 2>/dev/null; then
                log_error "$service_name is not responding on port $service_port"
                validation_failed=true
                continue
            fi
            
            # Check health endpoint
            local health_url="http://localhost:$service_port/health"
            if curl -s -f "$health_url" &>/dev/null; then
                local health_status=$(curl -s "$health_url" | jq -r '.status' 2>/dev/null || echo "unknown")
                if [ "$health_status" = "healthy" ] || [ "$health_status" = "ok" ]; then
                    log_success "$service_name health check passed"
                else
                    log_error "$service_name health check failed: $health_status"
                    validation_failed=true
                fi
            else
                log_warn "$service_name health endpoint not available"
            fi
        done
        
        # Check database connectivity
        if docker-compose exec -T postgres psql -U postgres -c "SELECT 1;" &>/dev/null; then
            log_success "PostgreSQL connectivity verified"
        else
            log_error "PostgreSQL connectivity failed"
            validation_failed=true
        fi
        
        if docker-compose exec -T redis redis-cli ping | grep -q "PONG"; then
            log_success "Redis connectivity verified"
        else
            log_error "Redis connectivity failed"
            validation_failed=true
        fi
        
        if [ "$validation_failed" = true ]; then
            log_error "Rollback validation failed"
            return 1
        else
            log_success "Rollback validation passed"
            return 0
        fi
    fi
}

show_rollback_summary() {
    log_step "Rollback Summary"
    
    echo -e "${CYAN}================================${NC}"
    echo -e "${CYAN}  KNIRV Network Rollback Complete${NC}"
    echo -e "${CYAN}================================${NC}"
    echo ""
    echo -e "${GREEN}Rollback Details:${NC}"
    echo -e "  • Backup Used: $BACKUP_ID"
    echo -e "  • Rollback Mode: $ROLLBACK_MODE"
    echo -e "  • Data Preserved: $PRESERVE_DATA"
    echo ""
    echo -e "${GREEN}Services Status:${NC}"
    
    # Show service status
    docker-compose ps
    
    echo ""
    echo -e "${GREEN}Next Steps:${NC}"
    echo -e "  1. Verify application functionality"
    echo -e "  2. Check logs for any issues: docker-compose logs"
    echo -e "  3. Run health checks: make health-check"
    echo -e "  4. Consider investigating the original issue"
    echo ""
}

# Emergency rollback function
emergency_rollback() {
    log_error "EMERGENCY ROLLBACK INITIATED"
    
    ROLLBACK_MODE="emergency"
    FORCE_ROLLBACK=true
    PRESERVE_DATA=false
    VALIDATE_ROLLBACK=false
    
    # Kill all services immediately
    docker-compose kill 2>/dev/null || true
    docker-compose down --volumes 2>/dev/null || true
    
    # Use latest backup
    select_backup
    restore_configuration
    restart_services
    
    log_warn "Emergency rollback completed - manual validation required"
}

# Main rollback function
main() {
    log_info "Starting KNIRV Network rollback procedure"
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --backup-id)
                BACKUP_ID="$2"
                shift 2
                ;;
            --mode)
                ROLLBACK_MODE="$2"
                shift 2
                ;;
            --force)
                FORCE_ROLLBACK=true
                shift
                ;;
            --no-preserve)
                PRESERVE_DATA=false
                shift
                ;;
            --no-validate)
                VALIDATE_ROLLBACK=false
                shift
                ;;
            --emergency)
                emergency_rollback
                exit 0
                ;;
            --list-backups)
                list_available_backups
                exit 0
                ;;
            --help)
                echo "Usage: $0 [OPTIONS]"
                echo "Options:"
                echo "  --backup-id ID      Specific backup to restore"
                echo "  --mode MODE         Rollback mode (auto, manual, emergency)"
                echo "  --force             Force rollback without confirmation"
                echo "  --no-preserve       Don't create pre-rollback backup"
                echo "  --no-validate       Skip rollback validation"
                echo "  --emergency         Emergency rollback (immediate)"
                echo "  --list-backups      List available backups"
                echo "  --help              Show this help message"
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                exit 1
                ;;
        esac
    done
    
    # Confirmation for non-emergency rollbacks
    if [ "$FORCE_ROLLBACK" != true ] && [ "$ROLLBACK_MODE" != "emergency" ]; then
        echo -e "${YELLOW}WARNING: This will rollback the KNIRV Network to a previous state.${NC}"
        echo -e "${YELLOW}Current data may be lost if not backed up.${NC}"
        echo ""
        read -p "Are you sure you want to proceed? (yes/no): " confirm
        
        if [ "$confirm" != "yes" ]; then
            log_info "Rollback cancelled by user"
            exit 0
        fi
    fi
    
    # Execute rollback steps
    check_prerequisites
    select_backup
    create_pre_rollback_backup
    stop_current_services
    restore_databases
    restore_configuration
    restart_services
    
    if validate_rollback; then
        show_rollback_summary
        log_success "KNIRV Network rollback completed successfully!"
    else
        log_error "Rollback validation failed - manual intervention required"
        exit 1
    fi
}

# Run main function with all arguments
main "$@"
