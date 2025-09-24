# KNIRV D-TEN Deployment Documentation

[TOC]

## Overview

This document provides comprehensive instructions for deploying KNIRV D-TEN, including production and testnet deployments, KNIRVORACLE integration, and detailed troubleshooting information.  This encompasses Months 14-18 implementations.

## Production Deployment

### KNIRV D-TEN Production Deployment

This section details the production deployment of KNIRV D-TEN (Months 14-18 implementation), including KNIRVORACLE integration.

#### Overview

The deployment includes:

- **KNIRVORACLE deployment** with embedded services
- **Production-optimized Kubernetes manifests** (`production-config/optimization.yaml`)
- **Comprehensive monitoring stack** (Prometheus, Grafana, Alertmanager, Node Exporter, cAdvisor) (`monitoring/`)
- **Infrastructure provisioning** with Ansible (`ansible/`)
- **CloudFlare DNS management**
- **Final test suite** (`testing/final-test-suite.sh`) for validation
- **Automated deployment scripts** (`deploy.sh`, `ansible/deploy-knirvoracle.sh`)
- **Security hardening configurations**

#### Directory Structure

```
deployment/
├── ansible/                       # Ansible deployment automation
│   ├── knirvoracle-deployment.yml # KNIRVORACLE deployment playbook
│   ├── infrastructure-playbook.yml# Infrastructure provisioning
│   ├── deploy-knirvoracle.sh      # KNIRVORACLE deployment script
│   ├── templates/                 # Configuration templates
│   ├── environments/              # Environment-specific configs
│   └── KNIRVORACLE-DEPLOYMENT.md  # KNIRVORACLE deployment docs
├── production-config/
│   └── optimization.yaml          # Kubernetes deployment with optimization settings
├── monitoring/
│   ├��─ prometheus.yml             # Prometheus configuration
│   ├── alert_rules.yml            # Alert rules for monitoring
│   ├── alertmanager.yml           # Alertmanager configuration
│   └── grafana-dashboard.json     # Grafana dashboard
├── testing/
│   └── final-test-suite.sh        # Comprehensive test suite
├── docker-compose.monitoring.yml  # Monitoring stack Docker Compose
├── deploy.sh                      # Main deployment script
└── README.md                      # This file
```

#### Prerequisites

1. **Kubernetes cluster** (v1.20+)
2. **kubectl** configured and connected
3. **Docker** installed and running
4. **Sufficient cluster resources**: 8+ CPU cores, 16+ GB RAM, 100+ GB storage

#### Quick Start

##### Full Production Deployment

```bash
# Deploy everything (KNIRV stack + monitoring)
./deploy.sh deploy

# Deploy infrastructure + KNIRVORACLE
./deploy.sh infrastructure

# Deploy KNIRVORACLE only
./deploy.sh knirvoracle
```

##### Deploy Only Monitoring

```bash
# Deploy monitoring stack only
./deploy.sh monitoring
```

##### Run Tests Only

```bash
# Run the final test suite
./deploy.sh test
```

#### Deployment Components

##### KNIRV Stack

- **API Gateway** (Port 8000)
- **KNIRVCHAIN** (Port 8080) - Blockchain layer
- **KNIRVGRAPH** (Port 8081) - NRV graph processing
- **KNIRVNEXUS** (Port 8082) - LLM inference engine
- **KNIRVORACLE** (Port 8083) - Core orchestration with XION bridge
- **KNIRVROUTER** (Ports 3478/5349/9090) - Connectivity with proof engine

##### Monitoring Stack

- **Prometheus** (Port 9090) - Metrics collection
- **Grafana** (Port 3000) - Visualization dashboard
- **Alertmanager** (Port 9093) - Alert management
- **Node Exporter** (Port 9100) - System metrics
- **cAdvisor** (Port 8080) - Container metrics

#### Key Features

##### Performance Optimization

- Connection pooling and caching
- Resource limits and requests
- Horizontal pod autoscaling
- Load balancing configuration

##### Security Hardening

- TLS 1.3 enforcement
- JWT authentication with rotation
- Rate limiting and CORS protection
- Network policies and security contexts

##### Monitoring & Alerting

- Real-time metrics collection
- Custom KNIRV-specific alerts
- Health checks and connectivity monitoring
- Performance and error rate tracking

#### KNIRVORACLE Deployment

KNIRVORACLE is the core oracle service with embedded agent services. See `ansible/KNIRVORACLE-DEPLOYMENT.md` for detailed documentation.

##### Quick KNIRVORACLE Deployment

```bash
# Deploy infrastructure + KNIRVORACLE
cd deployment/ansible
./deploy-knirvoracle.sh infrastructure --env production

# Deploy KNIRVORACLE only
./deploy-knirvoracle.sh deploy --env production

# Update DNS records only
./deploy-knirvoracle.sh dns-only --env production
```

##### KNIRVORACLE Services

- **Oracle API** (Port 1317) - `oracle.knirv.com`
- **Bootnode Registry** (Port 3006) - `bootnode-registry.knirv.com`
- **Tunnel Registry** (Port 3003) - `tunnel-registry.knirv.com`
- **Notary System** (Port 3007) - `notary-system.knirv.com`
- **Network Monitor** (Port 3008) - `network-monitor.knirv.com`
- **NANDA-ANS** (Port 3009) - `nanda-ans.knirv.com`

#### Configuration

##### Environment Variables

```bash
# CloudFlare DNS (required for KNIRVORACLE)
CLOUDFLARE_API_TOKEN=your-cloudflare-api-token
CLOUDFLARE_ZONE_ID=your-zone-id

# XION Integration
XION_RPC=https://rpc.xion.burnt.com:443
NRN_CONTRACT_ADDR=xion1nrncontractaddress

# Database
DATABASE_URL=postgresql://user:pass@host:5432/knirv

# Monitoring
PROOF_INTERVAL=5m
MINTING_ENABLED=true

# KNIRVORACLE
KNIRVORACLE_API_KEY=your-api-key
```

##### Resource Requirements

**Minimum per service:** CPU: 500m-1000m, Memory: 512Mi-2Gi

**Recommended for production:** CPU: 1000m-3000m, Memory: 1Gi-4Gi

#### Testing

The final test suite validates:

1. **Service Health**
2. **Authentication**
3. **LLM Registration**
4. **NRV System**
5. **Token Economics**
6. **Cross-Chain Bridge**
7. **Load Performance**
8. **Security**
9. **WebSocket**
10. **KNIRV-ROUTER**
11. **Data Consistency**

##### Running Tests

```bash
# Run all tests
./deploy.sh test

# Run specific test categories
cd testing/
./final-test-suite.sh
```

#### Monitoring

##### Accessing Dashboards

- **Grafana**: http://localhost:3000 (admin/admin123)
- **Prometheus**: http://localhost:9090
- **Alertmanager**: http://localhost:9093

##### Key Metrics

- `knirv_router_connectivity_score`
- `knirv_bridge_pending_transactions`
- `http_request_duration_seconds`
- `up`

##### Alerts

- Service downtime
- High CPU/memory usage
- API response time degradation
- Bridge transaction failures
- Connectivity proof failures

#### Troubleshooting

##### Common Issues

1. **Pods not starting**: `kubectl describe pod -n knirv-production`, `kubectl logs -n knirv-production <pod-name>`
2. **Service connectivity issues**: `kubectl get svc -n knirv-production`, `kubectl port-forward -n knirv-production svc/knirv-service 8080:8080`
3. **Monitoring not working**: `docker-compose -f docker-compose.monitoring.yml logs`

##### Rollback

```bash
./deploy.sh rollback
```

#### Security Considerations

##### Production Checklist

- [ ] Update default passwords
- [ ] Configure TLS certificates
- [ ] Set up network policies
- [ ] Enable audit logging
- [ ] Configure backup procedures
- [ ] Review RBAC permissions
- [ ] Set up secret rotation
- [ ] Configure firewall rules

##### Secrets Management

```bash
kubectl create secret generic blockchain-secrets \
  --from-literal=xion-rpc-url="YOUR_XION_RPC" \
  --namespace=knirv-production

kubectl create secret generic database-secrets \
  --from-literal=connection-string="YOUR_DB_CONNECTION" \
  --namespace=knirv-production
```

#### Maintenance

##### Regular Tasks

1. **Monitor resource usage**
2. **Update dependencies**
3. **Backup data**
4. **Review logs**
5. **Test disaster recovery**

##### Scaling

```bash
kubectl scale deployment knirv-production-stack --replicas=5 -n knirv-production
```

#### Support

1. Check the monitoring dashboards
2. Review application logs
3. Run the test suite for validation
4. Check Kubernetes events and pod status

#### Next Steps

1. Configure DNS and SSL certificates
2. Set up external monitoring integrations
3. Implement backup and disaster recovery
4. Configure CI/CD pipelines
5. Set up log aggregation
6. Plan capacity scaling strategies

#### Production Deployment Checklist

- [ ] All services containerized and orchestrated with Kubernetes
- [ ] SSL/TLS certificates configured with automatic renewal
- [ ] Database backups automated with point-in-time recovery
- [ ] Monitoring and alerting configured with Prometheus/Grafana
- [ ] Log aggregation with ELK stack
- [ ] CDN configured for static assets
- [ ] Auto-scaling policies configured
- [ ] Disaster recovery procedures documented
- [ ] Security hardening completed
- [ ] Performance optimization validated
- [ ] Automated deployment pipeline set up
- [ ] Documentation updated with production setup instructions
- [ ] User acceptance testing performed

#### Success Metrics

- 99.9% uptime SLA
- <500ms average API response time
- <5% error rate under normal load
- Support for 10,000+ concurrent users
- 24/7 monitoring and alerting

#### Future Sprints

- [ ] Update Kubernetes deployments
- [ ] Performance optimization
- [ ] Advanced monitoring integration
- [ ] Security hardening
- [ ] Production deployment


## Testnet Deployment Fixes

### KNIRV Testnet Deployment Fixes Summary

#### Overview

This section summarizes fixes implemented to resolve deployment errors in the KNIRV testnet deployment system.

#### Issues Resolved

##### 1. Native Deployment File Upload Issue

**Problem**: File upload reaching server capacity due to uploading the entire KNIRVTESTNET directory including `node_modules`.

**Solutions Implemented**:

- **Local Build Process**: Modified `prepare_native_testnet_files()` to build testnet-gateway locally before upload.
- **Exclude node_modules**: Added rsync-based file copying that excludes `node_modules`, `dist`, `build`, and other large development artifacts.
- **Server Cleanup**: Added `clean_server_deployment()` function to clean old files while preserving data.
- **Server-side npm install**: Updated `deploy_native_testnet()` to install npm dependencies on the server after upload.
- **Fallback Support**: Added fallback to `cp` with find exclusions if rsync is not available.

**Key Changes**:

```bash
# Before: Uploaded entire directory including node_modules
cp -r "$TESTNET_DIR" "$TEMP_DIR/knirvtestnet"

# After: Exclude large development files
rsync -av --exclude='node_modules' --exclude='dist' --exclude='build' \
      --exclude='.next' --exclude='.netlify' --exclude='*.log' \
      "$TESTNET_DIR/" "$TEMP_DIR/knirvtestnet/"
```

##### 2. Docker Deployment Network Issue

**Problem**: NGINX network manager over-complicating the system and services not reachable through the network.

**Solutions Implemented**:

- **Subnet Detection**: Added `detect_subnet_environment()` to check if the server is behind NAT/subnet.
- **Conditional NGINX**: Created `generate_docker_compose_file()` that conditionally includes NGINX based on subnet detection.
- **UFW Configuration**: Added `configure_ufw_firewall()` to match AWS EC2 security group ports.
- **Port Verification**: Enhanced CloudFlare DNS integration with proper port verification before updates.

**Key Features**:

- NGINX network manager only activates when the server is detected behind a subnet.
- Direct port exposure when the server has a public IP.
- UFW firewall automatically configured to match AWS EC2 ports.
- Comprehensive port verification before DNS updates.

##### 3. Separate EC2 Instance Management

**Problem**: All deployment types using the same EC2 instance, causing overwrites.

**Solutions Implemented**:

- **Instance Separation**: Added separate instance IDs for Native, Docker, and Podman deployments.
- **Automatic Instance Creation**: Added `create_deployment_instance()` for missing instances.
- **Instance Tagging**: Implemented a proper tagging system for deployment tracking.
- **Dynamic Selection**: Added `select_deployment_instance()` based on deployment type.

**Instance Configuration**:

```bash
NATIVE_INSTANCE_ID="i-06813be8a8a23ea5b"      # Native deployment
DOCKER_INSTANCE_ID="i-0a1b2c3d4e5f6g7h8"     # Docker deployment  
PODMAN_INSTANCE_ID="i-0x1y2z3a4b5c6d7e8f"     # Podman deployment
```

##### 4. Incremental Deployment Option

**Problem**: No mechanism for uploading only changed files vs. full deployment.

**Solutions Implemented**:

- **Change Detection**: Added `check_for_changes()` using MD5 checksums.
- **Incremental Upload**: Implemented `incremental_upload_native()` and `incremental_upload_container()`.
- **Command Line Options**: Added `--incremental`, `--full`, and `--force` flags.
- **Fallback Mechanism**: Automatic fallback to full deployment if incremental fails.

**Usage Examples**:

```bash
# Incremental deployment (only upload changed files)
./deploy-testnet-services.sh --incremental

# Force full deployment
./deploy-testnet-services.sh --force

# Full deployment (default)
./deploy-testnet-services.sh --full
```

##### 5. Enhanced CloudFlare DNS Integration

**Problem**: DNS updates happening without proper port verification.

**Solutions Implemented**:

- **Port Verification**: Added `verify_service_ports()` to test port accessibility.
- **UFW Status Check**: Enhanced verification to check UFW and netstat output.
- **Error Handling**: Comprehensive error handling with detailed logging.
- **Selective Updates**: Only update DNS for verified healthy services.

**Verification Process**:

1. Wait 45 seconds for services to initialize.
2. Check UFW status and listening ports.
3. Test port connectivity for each service.
4. Only update DNS for accessible services.
5. Provide detailed error reporting.

#### New Command Line Options

```bash
Usage: ./deploy-testnet-services.sh [options]

Options:
  --help, -h         Show help message
  --force            Skip confirmations and force full deployment
  --incremental      Enable incremental deployment (only upload changed files)
  --full             Force full deployment (default)

Deployment Modes:
  --incremental      Only upload files that have changed since last deployment
  --full             Upload all files (default, slower but more reliable)
  --force            Force full deployment even if no changes detected
```

#### Architecture Improvements

##### Network Management

- **Conditional NGINX**: Only active when behind subnet/NAT.
- **Direct Port Exposure**: When the server has a public IP.
- **UFW Integration**: Automatic firewall configuration.

##### Instance Management

- **Deployment Isolation**: Separate instances prevent overwrites.
- **Automatic Provisioning**: Missing instances created automatically.
- **Proper Tagging**: Deployment tracking and management.

##### File Management

- **Smart Exclusions**: Exclude development artifacts.
- **Incremental Sync**: Only upload changed files.
- **Local Building**: Build before upload to reduce server load.

#### Testing and Verification

The deployment system now includes comprehensive verification:

1. **Prerequisites Check**: SSH, disk space, instance availability.
2. **Subnet Detection**: Automatic network environment detection.
3. **UFW Configuration**: Firewall setup matching AWS security groups.
4. **Port Verification**: Test all service ports before DNS updates.
5. **Health Monitoring**: Comprehensive service health checks.

#### Benefits

1. **Reduced Upload Time**: Incremental deployments significantly faster.
2. **Server Capacity**: No more server capacity issues from large uploads.
3. **Network Reliability**: Conditional NGINX prevents over-complication.
4. **Deployment Isolation**: Separate instances prevent conflicts.
5. **Better Monitoring**: Enhanced verification and error reporting.
6. **Idempotent Operations**: Safe to run multiple times.

#### Files Modified

- `scripts/deploy-testnet-services.sh` - Main deployment script with all enhancements.
- Enhanced functions for subnet detection, UFW configuration, incremental deployment.
- Improved CloudFlare DNS integration with port verification.
- Added separate EC2 instance management.

All changes maintain backward compatibility while adding new functionality.

