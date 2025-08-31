#!/bin/bash
set -e

# KNIRV Network Monitor - Setup Script
# This script sets up the complete monitoring infrastructure

echo "🚀 KNIRV NETWORK MONITOR - SETUP"
echo "================================="

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
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

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
CONFIG_DIR="$PROJECT_ROOT/config"

print_status "Project root: $PROJECT_ROOT"
print_status "Config directory: $CONFIG_DIR"

# Check prerequisites
check_prerequisites() {
    print_status "Checking prerequisites..."
    
    # Check Docker
    if ! command -v docker &> /dev/null; then
        print_error "Docker is not installed. Please install Docker first."
        exit 1
    fi
    
    # Check Docker Compose
    if ! command -v docker-compose &> /dev/null; then
        print_error "Docker Compose is not installed. Please install Docker Compose first."
        exit 1
    fi
    
    # Check Go (for building the monitor application)
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed. Please install Go 1.21+ first."
        exit 1
    fi
    
    print_success "All prerequisites are available"
}

# Create directory structure
create_directories() {
    print_status "Creating directory structure..."
    
    mkdir -p "$CONFIG_DIR"/{prometheus,grafana/{provisioning/{datasources,dashboards},dashboards},alertmanager,logstash/{pipeline},kibana,nginx/{ssl}}
    mkdir -p "$PROJECT_ROOT"/{logs,data,bin}
    
    print_success "Directory structure created"
}

# Generate configuration files
generate_configs() {
    print_status "Generating configuration files..."
    
    # Prometheus configuration
    cat > "$CONFIG_DIR/prometheus.yml" << 'EOF'
global:
  scrape_interval: 15s
  evaluation_interval: 15s

rule_files:
  - "alert_rules.yml"

alerting:
  alertmanagers:
    - static_configs:
        - targets:
          - alertmanager:9093

scrape_configs:
  - job_name: 'prometheus'
    static_configs:
      - targets: ['localhost:9090']

  - job_name: 'node-exporter'
    static_configs:
      - targets: ['node-exporter:9100']

  - job_name: 'cadvisor'
    static_configs:
      - targets: ['cadvisor:8080']

  - job_name: 'knirv-monitor-api'
    static_configs:
      - targets: ['knirv-monitor-api:8090']
    metrics_path: '/metrics'

  # KNIRV Production Services
  - job_name: 'knirv-production-ipfs'
    static_configs:
      - targets: ['host.docker.internal:5001']
    metrics_path: '/debug/metrics/prometheus'
    scrape_interval: 30s

  - job_name: 'knirv-production-oracle'
    static_configs:
      - targets: ['host.docker.internal:8083']
    metrics_path: '/metrics'
    scrape_interval: 30s

  - job_name: 'knirv-production-chain'
    static_configs:
      - targets: ['host.docker.internal:8080']
    metrics_path: '/metrics'
    scrape_interval: 30s

  - job_name: 'knirv-production-graph'
    static_configs:
      - targets: ['host.docker.internal:8081']
    metrics_path: '/metrics'
    scrape_interval: 30s

  - job_name: 'knirv-production-nexus'
    static_configs:
      - targets: ['host.docker.internal:8082']
    metrics_path: '/metrics'
    scrape_interval: 30s

  - job_name: 'knirv-production-router'
    static_configs:
      - targets: ['host.docker.internal:5001']
    metrics_path: '/metrics'
    scrape_interval: 30s
EOF

    # Alert rules
    cat > "$CONFIG_DIR/alert_rules.yml" << 'EOF'
groups:
  - name: knirv_alerts
    rules:
      - alert: ServiceDown
        expr: up == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Service {{ $labels.instance }} is down"
          description: "{{ $labels.job }} on {{ $labels.instance }} has been down for more than 1 minute."

      - alert: HighCPUUsage
        expr: 100 - (avg by(instance) (irate(node_cpu_seconds_total{mode="idle"}[5m])) * 100) > 80
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High CPU usage on {{ $labels.instance }}"
          description: "CPU usage is above 80% for more than 5 minutes."

      - alert: HighMemoryUsage
        expr: (node_memory_MemTotal_bytes - node_memory_MemAvailable_bytes) / node_memory_MemTotal_bytes * 100 > 90
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High memory usage on {{ $labels.instance }}"
          description: "Memory usage is above 90% for more than 5 minutes."

      - alert: DiskSpaceLow
        expr: (node_filesystem_avail_bytes / node_filesystem_size_bytes) * 100 < 10
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Low disk space on {{ $labels.instance }}"
          description: "Disk space is below 10% on {{ $labels.mountpoint }}."

      - alert: IPFSStorageHigh
        expr: ipfs_repo_size_bytes / ipfs_repo_size_max_bytes * 100 > 90
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "IPFS storage usage high"
          description: "IPFS repository is using more than 90% of available storage."
EOF

    # AlertManager configuration
    cat > "$CONFIG_DIR/alertmanager.yml" << 'EOF'
global:
  smtp_smarthost: 'localhost:587'
  smtp_from: 'alerts@knirv.com'

route:
  group_by: ['alertname']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 1h
  receiver: 'web.hook'

receivers:
  - name: 'web.hook'
    webhook_configs:
      - url: 'http://knirv-monitor-api:8090/webhook/alerts'
        send_resolved: true

  - name: 'slack'
    slack_configs:
      - api_url: 'YOUR_SLACK_WEBHOOK_URL'
        channel: '#knirv-alerts'
        title: 'KNIRV Alert'
        text: '{{ range .Alerts }}{{ .Annotations.summary }}{{ end }}'

inhibit_rules:
  - source_match:
      severity: 'critical'
    target_match:
      severity: 'warning'
    equal: ['alertname', 'dev', 'instance']
EOF

    # Grafana datasource provisioning
    mkdir -p "$CONFIG_DIR/grafana/provisioning/datasources"
    cat > "$CONFIG_DIR/grafana/provisioning/datasources/prometheus.yml" << 'EOF'
apiVersion: 1

datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true
    editable: true

  - name: Elasticsearch
    type: elasticsearch
    access: proxy
    url: http://elasticsearch:9200
    database: "knirv-logs-*"
    interval: Daily
    timeField: "@timestamp"
    editable: true
EOF

    # Grafana dashboard provisioning
    mkdir -p "$CONFIG_DIR/grafana/provisioning/dashboards"
    cat > "$CONFIG_DIR/grafana/provisioning/dashboards/knirv.yml" << 'EOF'
apiVersion: 1

providers:
  - name: 'KNIRV Dashboards'
    orgId: 1
    folder: 'KNIRV'
    type: file
    disableDeletion: false
    updateIntervalSeconds: 10
    allowUiUpdates: true
    options:
      path: /var/lib/grafana/dashboards
EOF

    # Logstash pipeline
    cat > "$CONFIG_DIR/logstash/pipeline/knirv.conf" << 'EOF'
input {
  beats {
    port => 5044
  }
  
  tcp {
    port => 5000
    codec => json
  }
  
  udp {
    port => 5000
    codec => json
  }
}

filter {
  if [fields][service] {
    mutate {
      add_field => { "service_name" => "%{[fields][service]}" }
    }
  }
  
  if [message] =~ /ERROR/ {
    mutate {
      add_field => { "log_level" => "error" }
    }
  } else if [message] =~ /WARN/ {
    mutate {
      add_field => { "log_level" => "warning" }
    }
  } else if [message] =~ /INFO/ {
    mutate {
      add_field => { "log_level" => "info" }
    }
  } else {
    mutate {
      add_field => { "log_level" => "debug" }
    }
  }
  
  date {
    match => [ "timestamp", "ISO8601" ]
  }
}

output {
  elasticsearch {
    hosts => ["elasticsearch:9200"]
    index => "knirv-logs-%{+YYYY.MM.dd}"
  }
  
  stdout {
    codec => rubydebug
  }
}
EOF

    # Logstash configuration
    cat > "$CONFIG_DIR/logstash/logstash.yml" << 'EOF'
http.host: "0.0.0.0"
xpack.monitoring.elasticsearch.hosts: [ "http://elasticsearch:9200" ]
EOF

    # Kibana configuration
    cat > "$CONFIG_DIR/kibana/kibana.yml" << 'EOF'
server.name: knirv-kibana
server.host: "0.0.0.0"
elasticsearch.hosts: [ "http://elasticsearch:9200" ]
monitoring.ui.container.elasticsearch.enabled: true
EOF

    # Nginx configuration
    cat > "$CONFIG_DIR/nginx/nginx.conf" << 'EOF'
events {
    worker_connections 1024;
}

http {
    upstream grafana {
        server grafana:3000;
    }
    
    upstream kibana {
        server kibana:5601;
    }
    
    upstream prometheus {
        server prometheus:9090;
    }
    
    upstream knirv-api {
        server knirv-monitor-api:8090;
    }
    
    server {
        listen 80;
        server_name localhost;
        
        location /grafana/ {
            proxy_pass http://grafana/;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
        }
        
        location /kibana/ {
            proxy_pass http://kibana/;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
        }
        
        location /prometheus/ {
            proxy_pass http://prometheus/;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
        }
        
        location /api/ {
            proxy_pass http://knirv-api/;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
        }
        
        location / {
            return 301 /grafana/;
        }
    }
}
EOF

    print_success "Configuration files generated"
}

# Build the monitoring application
build_application() {
    print_status "Building KNIRV Network Monitor application..."
    
    cd "$PROJECT_ROOT"
    
    # Initialize Go module if not exists
    if [ ! -f "go.mod" ]; then
        go mod init github.com/knirv/network-monitor
    fi
    
    # Download dependencies
    go mod tidy
    
    # Build the application
    go build -o bin/knirv-network-monitor ./cmd/monitor
    
    print_success "Application built successfully"
}

# Create default configuration
create_default_config() {
    print_status "Creating default configuration..."
    
    cat > "$CONFIG_DIR/config.yaml" << 'EOF'
active_network: "production"

networks:
  production:
    name: "KNIRV Production Network"
    description: "Main KNIRV production network"
    services:
      - name: "ipfs"
        url: "http://localhost:5001"
        health_endpoint: "/api/v0/version"
        check_interval: "30s"
        timeout: "10s"
        critical: true
      - name: "knirvoracle"
        url: "http://localhost:8083"
        health_endpoint: "/health"
        check_interval: "30s"
        timeout: "10s"
        critical: true
      - name: "knirvchain"
        url: "http://localhost:8080"
        health_endpoint: "/health"
        check_interval: "30s"
        timeout: "10s"
        critical: true
      - name: "knirvgraph"
        url: "http://localhost:8081"
        health_endpoint: "/height"
        check_interval: "30s"
        timeout: "10s"
        critical: true
      - name: "knirvnexus"
        url: "http://localhost:8082"
        health_endpoint: "/health"
        check_interval: "30s"
        timeout: "10s"
        critical: true
      - name: "knirvrouter"
        url: "http://localhost:5001"
        health_endpoint: "/status"
        check_interval: "30s"
        timeout: "10s"
        critical: true
    endpoints:
      prometheus: "http://localhost:9090"
      grafana: "http://localhost:3000"
      kibana: "http://localhost:5601"
      alertmanager: "http://localhost:9093"

  testnet:
    name: "KNIRV Testnet"
    description: "KNIRV development and testing network"
    services:
      - name: "ipfs"
        url: "http://localhost:5001"
        health_endpoint: "/api/v0/version"
        check_interval: "30s"
        timeout: "10s"
        critical: true
      - name: "knirvoracle"
        url: "http://localhost:1317"
        health_endpoint: "/health"
        check_interval: "30s"
        timeout: "10s"
        critical: true
    endpoints:
      prometheus: "http://localhost:9091"
      grafana: "http://localhost:3001"
      kibana: "http://localhost:5602"
      alertmanager: "http://localhost:9094"

monitoring:
  enabled: true
  check_interval: "30s"
  metrics_retention: "720h"

logging:
  enabled: true
  level: "info"
  retention: "168h"

alerting:
  enabled: true

gui:
  theme: "dark"
  refresh_interval: "5s"
  window_size:
    width: 1200
    height: 800
EOF

    print_success "Default configuration created"
}

# Main setup function
main() {
    print_status "Starting KNIRV Network Monitor setup..."
    
    check_prerequisites
    create_directories
    generate_configs
    build_application
    create_default_config
    
    print_success "Setup completed successfully!"
    echo ""
    print_status "Next steps:"
    print_status "1. Start monitoring stack: docker-compose -f docker-compose.monitoring.yml up -d"
    print_status "2. Start KNIRV Network Monitor: ./bin/knirv-network-monitor"
    print_status "3. Access Grafana: http://localhost:3000 (admin/admin123)"
    print_status "4. Access Kibana: http://localhost:5601"
    print_status "5. Access Prometheus: http://localhost:9090"
    echo ""
    print_status "For more information, see README.md"
}

# Run main function
main "$@"
