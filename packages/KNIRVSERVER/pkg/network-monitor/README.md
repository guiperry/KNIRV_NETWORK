# KNIRV Network Monitor

A unified monitoring platform for KNIRV production and testnet networks with custom Go/Fyne GUI dashboard, centralized logging, metrics collection, and intelligent alerting.

## 🎯 Overview

The KNIRV Network Monitor provides comprehensive observability for:
- **Production Network**: Full KNIRV D-TEN stack monitoring
- **Testnet Network**: Development and testing environment monitoring
- **IPFS Integration**: Distributed storage monitoring
- **Real-time Dashboards**: Custom Go/Fyne GUI application
- **Centralized Logging**: ELK Stack integration
- **Metrics Collection**: Prometheus & Grafana
- **Intelligent Alerting**: Slack, PagerDuty, and custom notifications

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    KNIRV Network Monitor                    │
├─────────────────────────────────────────────────────────────┤
│  Custom Go/Fyne Dashboard                                  │
│  ├── Real-time Metrics                                     │
│  ├── Service Health Status                                 │
│  ├── Log Aggregation View                                  │
│  ├── Alert Management                                      │
│  └── Network Topology                                      │
├─────────────────────────────────────────────────────────────┤
│  Monitoring Stack                                          │
│  ├── Prometheus (Metrics Collection)                       │
│  ├── Grafana (Web Dashboards)                             │
│  ├── Elasticsearch (Log Storage)                          │
│  ├── Logstash (Log Processing)                            │
│  ├── Kibana (Log Analysis)                                │
│  └── AlertManager (Alert Routing)                         │
├─────────────────────────────────────────────────────────────┤
│  Data Sources                                              │
│  ├── KNIRV Services (Production & Testnet)                │
│  ├── IPFS Nodes                                           │
│  ├── System Metrics                                       │
│  ├── Application Logs                                     │
│  └── Custom Metrics                                       │
└─────────────────────────────────────────────────────────────┘
```

## 🚀 Quick Start

### 1. Setup Monitoring Stack
```bash
# Start monitoring infrastructure
cd network-monitor
./scripts/setup-monitoring.sh

# Deploy monitoring stack
docker-compose -f docker-compose.monitoring.yml up -d

# Start KNIRV Network Monitor GUI
./bin/knirv-network-monitor
```

### 2. Monitor Production Network
```bash
# Start production network with monitoring
./scripts/start-production-monitoring.sh

# View real-time dashboard
./bin/knirv-network-monitor --network production
```

### 3. Monitor Testnet
```bash
# Start testnet with monitoring
./scripts/start-testnet-monitoring.sh

# View testnet dashboard
./bin/knirv-network-monitor --network testnet
```

## 📊 Features

### Custom Go/Fyne Dashboard
- **Real-time Service Status**: Live health checks for all KNIRV services
- **Performance Metrics**: CPU, memory, network, and custom application metrics
- **Log Viewer**: Integrated log streaming and search
- **Alert Center**: Real-time alert notifications and management
- **Network Topology**: Visual representation of service dependencies
- **Historical Data**: Time-series charts and trend analysis

### Monitoring Capabilities
- **Service Health**: Automated health checks for all KNIRV components
- **Performance Metrics**: API latency, throughput, error rates
- **Resource Monitoring**: CPU, memory, disk, network utilization
- **IPFS Metrics**: Storage usage, peer connections, content operations
- **Custom Metrics**: KNIRV-specific business metrics

### Logging & Analysis
- **Centralized Logging**: All service logs in one place
- **Log Correlation**: Cross-service request tracing
- **Error Tracking**: Automated error detection and categorization
- **Performance Analysis**: Slow query and bottleneck identification
- **Security Monitoring**: Access logs and security event tracking

### Alerting & Notifications
- **Smart Alerts**: ML-based anomaly detection
- **Multi-channel Notifications**: Slack, PagerDuty, email, SMS
- **Alert Escalation**: Automatic escalation based on severity
- **Alert Correlation**: Reduce noise by grouping related alerts
- **Custom Rules**: Flexible alerting rules for specific scenarios

## 🔧 Configuration

### Network Profiles
```yaml
# config/networks.yaml
production:
  name: "KNIRV Production Network"
  services:
    - name: "ipfs"
      url: "http://localhost:5001"
      health_endpoint: "/api/v0/version"
    - name: "knirvoracle"
      url: "http://localhost:8083"
      health_endpoint: "/health"
    # ... other services

testnet:
  name: "KNIRV Testnet"
  services:
    - name: "ipfs"
      url: "http://localhost:5001"
      health_endpoint: "/api/v0/version"
    - name: "knirvoracle"
      url: "http://localhost:1317"
      health_endpoint: "/health"
    # ... other services
```

### Alert Rules
```yaml
# config/alerts.yaml
rules:
  - name: "Service Down"
    condition: "service_up == 0"
    severity: "critical"
    channels: ["slack", "pagerduty"]
    
  - name: "High API Latency"
    condition: "api_latency_p95 > 1000"
    severity: "warning"
    channels: ["slack"]
    
  - name: "IPFS Storage Full"
    condition: "ipfs_storage_usage > 90"
    severity: "warning"
    channels: ["slack", "email"]
```

## 📱 GUI Dashboard Features

### Main Dashboard
- **Network Overview**: High-level health status
- **Service Grid**: Individual service status cards
- **Real-time Metrics**: Live updating charts
- **Alert Summary**: Current alerts and their status

### Service Details
- **Individual Service View**: Detailed metrics for each service
- **Log Streaming**: Real-time log output
- **Performance Charts**: Historical performance data
- **Configuration View**: Current service configuration

### Log Viewer
- **Unified Log Stream**: All services in one view
- **Advanced Filtering**: Filter by service, level, time range
- **Search Functionality**: Full-text search across all logs
- **Export Capabilities**: Export logs for analysis

### Alert Management
- **Alert Dashboard**: Current and historical alerts
- **Alert Details**: Full context for each alert
- **Acknowledgment**: Mark alerts as acknowledged
- **Escalation Control**: Manual escalation controls

## 🛠️ Development

### Building the GUI Application
```bash
# Install dependencies
go mod tidy

# Build for current platform
go build -o bin/knirv-network-monitor ./cmd/monitor

# Build for multiple platforms
./scripts/build-all.sh
```

### Running in Development Mode
```bash
# Start with debug logging
./bin/knirv-network-monitor --debug --network testnet

# Start with specific config
./bin/knirv-network-monitor --config ./config/custom.yaml
```

## 📈 Metrics Collected

### System Metrics
- CPU usage per core and overall
- Memory usage (RAM, swap)
- Disk I/O and space utilization
- Network traffic and connections

### Application Metrics
- Request rate and latency
- Error rates by endpoint
- Database query performance
- Cache hit/miss rates

### KNIRV-Specific Metrics
- NRV token operations
- Skill execution metrics
- Graph node operations
- Consensus participation
- IPFS content operations

### IPFS Metrics
- Storage utilization
- Peer connections
- Content retrieval times
- Pin operations
- Gateway requests

## 🚨 Alert Types

### Critical Alerts
- Service completely down
- Database connection lost
- IPFS node unreachable
- Security breach detected

### Warning Alerts
- High resource utilization
- Elevated error rates
- Slow response times
- Storage approaching limits

### Info Alerts
- Service restarts
- Configuration changes
- Scheduled maintenance
- Performance improvements

## 🔐 Security

### Access Control
- Role-based access to monitoring data
- Secure API endpoints
- Encrypted data transmission
- Audit logging for all actions

### Data Protection
- Sensitive data masking in logs
- Secure storage of metrics
- Regular security updates
- Compliance with data protection regulations

## 📚 Documentation

- [Installation Guide](./docs/installation.md)
- [Configuration Reference](./docs/configuration.md)
- [API Documentation](./docs/api.md)
- [Troubleshooting Guide](./docs/troubleshooting.md)
- [Development Guide](./docs/development.md)

## 🤝 Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md) for guidelines on contributing to the KNIRV Network Monitor.

## 📄 License

This project is licensed under the MIT License - see [LICENSE](./LICENSE) file for details.
