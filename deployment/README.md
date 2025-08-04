# KNIRV D-TEN Production Deployment

This directory contains the production deployment configuration for KNIRV D-TEN (Months 14-18 implementation).

## Overview

The deployment includes:
- **Production-optimized Kubernetes manifests**
- **Comprehensive monitoring stack** (Prometheus, Grafana, Alertmanager)
- **Final test suite** for validation
- **Automated deployment scripts**
- **Security hardening configurations**

## Directory Structure

```
deployment/
├── production-config/
│   └── optimization.yaml          # Kubernetes deployment with optimization settings
├── monitoring/
│   ├── prometheus.yml             # Prometheus configuration
│   ├── alert_rules.yml            # Alert rules for monitoring
│   ├── alertmanager.yml           # Alertmanager configuration
│   └── grafana-dashboard.json     # Grafana dashboard
├── testing/
│   └── final-test-suite.sh        # Comprehensive test suite
├── docker-compose.monitoring.yml  # Monitoring stack Docker Compose
├── deploy.sh                      # Main deployment script
└── README.md                      # This file
```

## Prerequisites

Before deploying, ensure you have:

1. **Kubernetes cluster** (v1.20+)
2. **kubectl** configured and connected
3. **Docker** installed and running
4. **Sufficient cluster resources**:
   - 8+ CPU cores
   - 16+ GB RAM
   - 100+ GB storage

## Quick Start

### 1. Full Production Deployment

```bash
# Deploy everything (KNIRV stack + monitoring)
./deploy.sh deploy
```

### 2. Deploy Only Monitoring

```bash
# Deploy monitoring stack only
./deploy.sh monitoring
```

### 3. Run Tests Only

```bash
# Run the final test suite
./deploy.sh test
```

## Deployment Components

### KNIRV Stack

The main KNIRV D-TEN stack includes:

- **API Gateway** (Port 8000)
- **KNIRVCHAIN** (Port 8080) - Blockchain layer
- **KNIRVGRAPH** (Port 8081) - NRV graph processing
- **KNIRVNEXUS** (Port 8082) - LLM inference engine
- **KNIRVROOT** (Port 8083) - Core orchestration with XION bridge
- **KNIRVROUTER** (Ports 3478/5349/9090) - Connectivity with proof engine

### Monitoring Stack

- **Prometheus** (Port 9090) - Metrics collection
- **Grafana** (Port 3000) - Visualization dashboard
- **Alertmanager** (Port 9093) - Alert management
- **Node Exporter** (Port 9100) - System metrics
- **cAdvisor** (Port 8080) - Container metrics

### Key Features

#### Performance Optimization
- Connection pooling and caching
- Resource limits and requests
- Horizontal pod autoscaling
- Load balancing configuration

#### Security Hardening
- TLS 1.3 enforcement
- JWT authentication with rotation
- Rate limiting and CORS protection
- Network policies and security contexts

#### Monitoring & Alerting
- Real-time metrics collection
- Custom KNIRV-specific alerts
- Health checks and connectivity monitoring
- Performance and error rate tracking

## Configuration

### Environment Variables

Key environment variables for production:

```bash
# XION Integration
XION_RPC=https://rpc.xion.burnt.com:443
NRN_CONTRACT_ADDR=xion1nrncontractaddress

# Database
DATABASE_URL=postgresql://user:pass@host:5432/knirv

# Monitoring
PROOF_INTERVAL=5m
MINTING_ENABLED=true
```

### Resource Requirements

**Minimum per service:**
- CPU: 500m-1000m
- Memory: 512Mi-2Gi

**Recommended for production:**
- CPU: 1000m-3000m
- Memory: 1Gi-4Gi

## Testing

The final test suite validates:

1. **Service Health** - All services responding
2. **Authentication** - JWT token flow
3. **LLM Registration** - Model registration and retrieval
4. **NRV System** - Error/skill node creation and resolution
5. **Token Economics** - Skill invocation and token burning
6. **Cross-Chain Bridge** - XION bridge functionality
7. **Load Performance** - Concurrent user handling
8. **Security** - Rate limiting and authentication
9. **WebSocket** - Real-time connectivity
10. **KNIRV-ROUTER** - Connectivity proof engine
11. **Data Consistency** - Cross-service data integrity

### Running Tests

```bash
# Run all tests
./deploy.sh test

# Run specific test categories
cd testing/
./final-test-suite.sh
```

## Monitoring

### Accessing Dashboards

After deployment:

- **Grafana**: http://localhost:3000 (admin/admin123)
- **Prometheus**: http://localhost:9090
- **Alertmanager**: http://localhost:9093

### Key Metrics

Monitor these critical metrics:

- `knirv_router_connectivity_score` - Connectivity health
- `knirv_bridge_pending_transactions` - Bridge transaction queue
- `http_request_duration_seconds` - API response times
- `up` - Service availability

### Alerts

Configured alerts include:

- Service downtime
- High CPU/memory usage
- API response time degradation
- Bridge transaction failures
- Connectivity proof failures

## Troubleshooting

### Common Issues

1. **Pods not starting**
   ```bash
   kubectl describe pod -n knirv-production
   kubectl logs -n knirv-production <pod-name>
   ```

2. **Service connectivity issues**
   ```bash
   kubectl get svc -n knirv-production
   kubectl port-forward -n knirv-production svc/knirv-service 8080:8080
   ```

3. **Monitoring not working**
   ```bash
   docker-compose -f docker-compose.monitoring.yml logs
   ```

### Rollback

If deployment fails:

```bash
./deploy.sh rollback
```

## Security Considerations

### Production Checklist

- [ ] Update default passwords
- [ ] Configure TLS certificates
- [ ] Set up network policies
- [ ] Enable audit logging
- [ ] Configure backup procedures
- [ ] Review RBAC permissions
- [ ] Set up secret rotation
- [ ] Configure firewall rules

### Secrets Management

Update these secrets before production:

```bash
kubectl create secret generic blockchain-secrets \
  --from-literal=xion-rpc-url="YOUR_XION_RPC" \
  --namespace=knirv-production

kubectl create secret generic database-secrets \
  --from-literal=connection-string="YOUR_DB_CONNECTION" \
  --namespace=knirv-production
```

## Maintenance

### Regular Tasks

1. **Monitor resource usage** - Scale as needed
2. **Update dependencies** - Security patches
3. **Backup data** - Database and configurations
4. **Review logs** - Error patterns and performance
5. **Test disaster recovery** - Backup restoration

### Scaling

To scale services:

```bash
kubectl scale deployment knirv-production-stack --replicas=5 -n knirv-production
```

## Support

For issues and questions:

1. Check the monitoring dashboards
2. Review application logs
3. Run the test suite for validation
4. Check Kubernetes events and pod status

## Next Steps

After successful deployment:

1. Configure DNS and SSL certificates
2. Set up external monitoring integrations
3. Implement backup and disaster recovery
4. Configure CI/CD pipelines
5. Set up log aggregation
6. Plan capacity scaling strategies
