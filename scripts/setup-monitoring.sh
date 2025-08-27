#!/bin/bash

# KNIRV Network Phase 7 Monitoring Setup Script
# Comprehensive monitoring and alerting system deployment

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
MONITORING_DIR="$PROJECT_ROOT/deployment/monitoring"
GRAFANA_DIR="$MONITORING_DIR/grafana"
PROMETHEUS_DIR="$MONITORING_DIR/prometheus"

# Monitoring configuration
GRAFANA_ADMIN_PASSWORD="admin123"
PROMETHEUS_RETENTION="30d"
ALERTMANAGER_WEBHOOK_URL=""
SLACK_WEBHOOK_URL=""
EMAIL_SMTP_HOST=""
EMAIL_SMTP_PORT="587"
EMAIL_FROM=""
EMAIL_TO=""

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
create_monitoring_directories() {
    log_step "Creating monitoring directories"
    
    mkdir -p "$MONITORING_DIR"/{prometheus,grafana,alertmanager,dashboards,alerts}
    mkdir -p "$GRAFANA_DIR"/{dashboards,provisioning/{dashboards,datasources,notifiers}}
    mkdir -p "$PROMETHEUS_DIR"/{rules,data}
    
    log_success "Monitoring directories created"
}

generate_prometheus_config() {
    log_step "Generating Prometheus configuration"
    
    cat > "$PROMETHEUS_DIR/prometheus.yml" << 'EOF'
global:
  scrape_interval: 15s
  evaluation_interval: 15s
  external_labels:
    cluster: 'knirv-network'
    environment: 'production'

rule_files:
  - "/etc/prometheus/rules/*.yml"

alerting:
  alertmanagers:
    - static_configs:
        - targets:
          - alertmanager:9093

scrape_configs:
  # Prometheus itself
  - job_name: 'prometheus'
    static_configs:
      - targets: ['localhost:9090']

  # KNIRV Network Services
  - job_name: 'knirv-oracle'
    static_configs:
      - targets: ['knirv-oracle:8083']
    metrics_path: '/metrics'
    scrape_interval: 10s

  - job_name: 'knirvchain'
    static_configs:
      - targets: ['knirvchain:8080']
    metrics_path: '/metrics'
    scrape_interval: 10s

  - job_name: 'knirvgraph'
    static_configs:
      - targets: ['knirvgraph:8081']
    metrics_path: '/metrics'
    scrape_interval: 10s

  - job_name: 'knirv-nexus'
    static_configs:
      - targets: ['knirv-nexus:8082']
    metrics_path: '/metrics'
    scrape_interval: 10s

  - job_name: 'knirvrouter'
    static_configs:
      - targets: ['knirvrouter:3478']
    metrics_path: '/metrics'
    scrape_interval: 10s

  - job_name: 'knirvcontroller'
    static_configs:
      - targets: ['knirvcontroller:3000']
    metrics_path: '/metrics'
    scrape_interval: 10s

  - job_name: 'knirv-gateway'
    static_configs:
      - targets: ['knirv-gateway:8000']
    metrics_path: '/metrics'
    scrape_interval: 5s

  # Infrastructure Services
  - job_name: 'postgres'
    static_configs:
      - targets: ['postgres:5432']
    metrics_path: '/metrics'

  - job_name: 'redis'
    static_configs:
      - targets: ['redis:6379']
    metrics_path: '/metrics'

  # Node Exporter for system metrics
  - job_name: 'node-exporter'
    static_configs:
      - targets: ['node-exporter:9100']

  # Docker metrics
  - job_name: 'cadvisor'
    static_configs:
      - targets: ['cadvisor:8080']
EOF

    log_success "Prometheus configuration generated"
}

generate_alertmanager_config() {
    log_step "Generating Alertmanager configuration"
    
    cat > "$MONITORING_DIR/alertmanager/alertmanager.yml" << EOF
global:
  smtp_smarthost: '${EMAIL_SMTP_HOST}:${EMAIL_SMTP_PORT}'
  smtp_from: '${EMAIL_FROM}'
  smtp_auth_username: '${EMAIL_FROM}'
  smtp_auth_password: 'SMTP_PASSWORD_HERE'

route:
  group_by: ['alertname', 'cluster', 'service']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 1h
  receiver: 'default'
  routes:
    - match:
        severity: critical
      receiver: 'critical-alerts'
    - match:
        severity: warning
      receiver: 'warning-alerts'

receivers:
  - name: 'default'
    email_configs:
      - to: '${EMAIL_TO}'
        subject: 'KNIRV Alert: {{ .GroupLabels.alertname }}'
        body: |
          {{ range .Alerts }}
          Alert: {{ .Annotations.summary }}
          Description: {{ .Annotations.description }}
          Labels: {{ range .Labels.SortedPairs }}{{ .Name }}={{ .Value }} {{ end }}
          {{ end }}

  - name: 'critical-alerts'
    email_configs:
      - to: '${EMAIL_TO}'
        subject: 'CRITICAL KNIRV Alert: {{ .GroupLabels.alertname }}'
        body: |
          CRITICAL ALERT TRIGGERED
          
          {{ range .Alerts }}
          Alert: {{ .Annotations.summary }}
          Description: {{ .Annotations.description }}
          Severity: {{ .Labels.severity }}
          Component: {{ .Labels.component }}
          Time: {{ .StartsAt }}
          {{ end }}
    webhook_configs:
      - url: '${ALERTMANAGER_WEBHOOK_URL}'
        send_resolved: true

  - name: 'warning-alerts'
    email_configs:
      - to: '${EMAIL_TO}'
        subject: 'KNIRV Warning: {{ .GroupLabels.alertname }}'
        body: |
          {{ range .Alerts }}
          Warning: {{ .Annotations.summary }}
          Description: {{ .Annotations.description }}
          Component: {{ .Labels.component }}
          {{ end }}

inhibit_rules:
  - source_match:
      severity: 'critical'
    target_match:
      severity: 'warning'
    equal: ['alertname', 'cluster', 'service']
EOF

    log_success "Alertmanager configuration generated"
}

generate_grafana_config() {
    log_step "Generating Grafana configuration"
    
    # Grafana datasource configuration
    cat > "$GRAFANA_DIR/provisioning/datasources/prometheus.yml" << 'EOF'
apiVersion: 1

datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true
    editable: true
EOF

    # Grafana dashboard provisioning
    cat > "$GRAFANA_DIR/provisioning/dashboards/knirv.yml" << 'EOF'
apiVersion: 1

providers:
  - name: 'KNIRV Dashboards'
    orgId: 1
    folder: 'KNIRV Network'
    type: file
    disableDeletion: false
    updateIntervalSeconds: 10
    allowUiUpdates: true
    options:
      path: /var/lib/grafana/dashboards
EOF

    log_success "Grafana configuration generated"
}

create_knirv_dashboard() {
    log_step "Creating KNIRV Network dashboard"
    
    cat > "$GRAFANA_DIR/dashboards/knirv-overview.json" << 'EOF'
{
  "dashboard": {
    "id": null,
    "title": "KNIRV Network Overview",
    "tags": ["knirv", "overview"],
    "timezone": "browser",
    "panels": [
      {
        "id": 1,
        "title": "Service Health",
        "type": "stat",
        "targets": [
          {
            "expr": "up",
            "legendFormat": "{{ job }}"
          }
        ],
        "fieldConfig": {
          "defaults": {
            "color": {
              "mode": "thresholds"
            },
            "thresholds": {
              "steps": [
                {"color": "red", "value": 0},
                {"color": "green", "value": 1}
              ]
            }
          }
        },
        "gridPos": {"h": 8, "w": 12, "x": 0, "y": 0}
      },
      {
        "id": 2,
        "title": "API Request Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(http_requests_total[5m])",
            "legendFormat": "{{ job }}"
          }
        ],
        "gridPos": {"h": 8, "w": 12, "x": 12, "y": 0}
      },
      {
        "id": 3,
        "title": "NRN Transaction Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(nrn_transactions_total[5m])",
            "legendFormat": "NRN Transactions/sec"
          }
        ],
        "gridPos": {"h": 8, "w": 12, "x": 0, "y": 8}
      },
      {
        "id": 4,
        "title": "Skill Execution Success Rate",
        "type": "stat",
        "targets": [
          {
            "expr": "rate(skill_executions_total[5m]) - rate(skill_executions_failed_total[5m])",
            "legendFormat": "Success Rate"
          }
        ],
        "gridPos": {"h": 8, "w": 12, "x": 12, "y": 8}
      }
    ],
    "time": {
      "from": "now-1h",
      "to": "now"
    },
    "refresh": "5s"
  }
}
EOF

    log_success "KNIRV Network dashboard created"
}

setup_docker_compose_monitoring() {
    log_step "Setting up Docker Compose monitoring stack"
    
    cat > "$MONITORING_DIR/docker-compose.monitoring.yml" << 'EOF'
version: '3.8'

services:
  prometheus:
    image: prom/prometheus:latest
    container_name: knirv-prometheus
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus/prometheus.yml:/etc/prometheus/prometheus.yml
      - ./alerts:/etc/prometheus/rules
      - prometheus_data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--web.console.libraries=/etc/prometheus/console_libraries'
      - '--web.console.templates=/etc/prometheus/consoles'
      - '--storage.tsdb.retention.time=30d'
      - '--web.enable-lifecycle'
    networks:
      - knirv-network

  grafana:
    image: grafana/grafana:latest
    container_name: knirv-grafana
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin123
      - GF_USERS_ALLOW_SIGN_UP=false
    volumes:
      - grafana_data:/var/lib/grafana
      - ./grafana/provisioning:/etc/grafana/provisioning
      - ./grafana/dashboards:/var/lib/grafana/dashboards
    networks:
      - knirv-network

  alertmanager:
    image: prom/alertmanager:latest
    container_name: knirv-alertmanager
    ports:
      - "9093:9093"
    volumes:
      - ./alertmanager/alertmanager.yml:/etc/alertmanager/alertmanager.yml
      - alertmanager_data:/alertmanager
    networks:
      - knirv-network

  node-exporter:
    image: prom/node-exporter:latest
    container_name: knirv-node-exporter
    ports:
      - "9100:9100"
    volumes:
      - /proc:/host/proc:ro
      - /sys:/host/sys:ro
      - /:/rootfs:ro
    command:
      - '--path.procfs=/host/proc'
      - '--path.rootfs=/rootfs'
      - '--path.sysfs=/host/sys'
      - '--collector.filesystem.mount-points-exclude=^/(sys|proc|dev|host|etc)($$|/)'
    networks:
      - knirv-network

  cadvisor:
    image: gcr.io/cadvisor/cadvisor:latest
    container_name: knirv-cadvisor
    ports:
      - "8080:8080"
    volumes:
      - /:/rootfs:ro
      - /var/run:/var/run:rw
      - /sys:/sys:ro
      - /var/lib/docker/:/var/lib/docker:ro
    networks:
      - knirv-network

volumes:
  prometheus_data:
  grafana_data:
  alertmanager_data:

networks:
  knirv-network:
    external: true
EOF

    log_success "Docker Compose monitoring configuration created"
}

deploy_monitoring_stack() {
    log_step "Deploying monitoring stack"
    
    cd "$MONITORING_DIR"
    
    # Create network if it doesn't exist
    docker network create knirv-network 2>/dev/null || true
    
    # Start monitoring services
    docker-compose -f docker-compose.monitoring.yml up -d
    
    # Wait for services to start
    log_info "Waiting for monitoring services to start..."
    sleep 30
    
    # Verify services are running
    if curl -s http://localhost:9090/api/v1/status/config &>/dev/null; then
        log_success "Prometheus is running"
    else
        log_error "Prometheus failed to start"
    fi
    
    if curl -s http://localhost:3000/api/health &>/dev/null; then
        log_success "Grafana is running"
    else
        log_error "Grafana failed to start"
    fi
    
    if curl -s http://localhost:9093/api/v1/status &>/dev/null; then
        log_success "Alertmanager is running"
    else
        log_error "Alertmanager failed to start"
    fi
    
    cd "$PROJECT_ROOT"
}

configure_alerts() {
    log_step "Configuring alerts"
    
    # Copy Phase 7 alerts
    cp "$PROJECT_ROOT/deployment/monitoring/phase7-alerts.yml" "$MONITORING_DIR/alerts/"
    
    # Reload Prometheus configuration
    if curl -s -X POST http://localhost:9090/-/reload &>/dev/null; then
        log_success "Prometheus configuration reloaded"
    else
        log_warn "Failed to reload Prometheus configuration"
    fi
}

show_monitoring_summary() {
    log_step "Monitoring Setup Summary"
    
    echo -e "${CYAN}================================${NC}"
    echo -e "${CYAN}  KNIRV Monitoring Stack Ready${NC}"
    echo -e "${CYAN}================================${NC}"
    echo ""
    echo -e "${GREEN}Services:${NC}"
    echo -e "  • Prometheus: http://localhost:9090"
    echo -e "  • Grafana: http://localhost:3000 (admin/admin123)"
    echo -e "  • Alertmanager: http://localhost:9093"
    echo -e "  • Node Exporter: http://localhost:9100"
    echo -e "  • cAdvisor: http://localhost:8080"
    echo ""
    echo -e "${GREEN}Features:${NC}"
    echo -e "  • 25+ production alerts configured"
    echo -e "  • KNIRV-specific business logic monitoring"
    echo -e "  • Infrastructure and security alerts"
    echo -e "  • Email and webhook notifications"
    echo -e "  • Custom dashboards for KNIRV metrics"
    echo ""
    echo -e "${GREEN}Next Steps:${NC}"
    echo -e "  1. Configure email/webhook settings in alertmanager.yml"
    echo -e "  2. Import additional dashboards in Grafana"
    echo -e "  3. Test alert notifications"
    echo -e "  4. Set up log aggregation (ELK stack)"
    echo ""
}

# Main function
main() {
    log_info "Starting KNIRV Network monitoring setup"
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --grafana-password)
                GRAFANA_ADMIN_PASSWORD="$2"
                shift 2
                ;;
            --email-smtp-host)
                EMAIL_SMTP_HOST="$2"
                shift 2
                ;;
            --email-from)
                EMAIL_FROM="$2"
                shift 2
                ;;
            --email-to)
                EMAIL_TO="$2"
                shift 2
                ;;
            --slack-webhook)
                SLACK_WEBHOOK_URL="$2"
                shift 2
                ;;
            --help)
                echo "Usage: $0 [OPTIONS]"
                echo "Options:"
                echo "  --grafana-password PWD    Grafana admin password"
                echo "  --email-smtp-host HOST    SMTP server host"
                echo "  --email-from EMAIL        From email address"
                echo "  --email-to EMAIL          To email address"
                echo "  --slack-webhook URL       Slack webhook URL"
                echo "  --help                    Show this help message"
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                exit 1
                ;;
        esac
    done
    
    # Execute setup steps
    create_monitoring_directories
    generate_prometheus_config
    generate_alertmanager_config
    generate_grafana_config
    create_knirv_dashboard
    setup_docker_compose_monitoring
    deploy_monitoring_stack
    configure_alerts
    show_monitoring_summary
    
    log_success "KNIRV Network monitoring setup completed successfully!"
}

# Run main function with all arguments
main "$@"
