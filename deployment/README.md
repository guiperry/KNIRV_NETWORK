# KNIRV D-TEN Production Deployment

This directory contains the production deployment configuration for KNIRV D-TEN (Months 14-18 implementation), including KNIRVORACLE integration.

## Overview

The deployment includes:
- **KNIRVORACLE deployment** with embedded services
- **Production-optimized Kubernetes manifests**
- **Comprehensive monitoring stack** (Prometheus, Grafana, Alertmanager)
- **Infrastructure provisioning** with Ansible
- **CloudFlare DNS management**
- **Final test suite** for validation
- **Automated deployment scripts**
- **Security hardening configurations**

## Directory Structure

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

# Deploy infrastructure + KNIRVORACLE
./deploy.sh infrastructure

# Deploy KNIRVORACLE only
./deploy.sh knirvoracle
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
- **KNIRVORACLE** (Port 8083) - Core orchestration with XION bridge
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

## KNIRVORACLE Deployment

KNIRVORACLE is the core oracle service with embedded agent services. See [ansible/KNIRVORACLE-DEPLOYMENT.md](ansible/KNIRVORACLE-DEPLOYMENT.md) for detailed documentation.

### Quick KNIRVORACLE Deployment

```bash
# Deploy infrastructure + KNIRVORACLE
cd deployment/ansible
./deploy-knirvoracle.sh infrastructure --env production

# Deploy KNIRVORACLE only
./deploy-knirvoracle.sh deploy --env production

# Update DNS records only
./deploy-knirvoracle.sh dns-only --env production
```

### KNIRVORACLE Services

- **Oracle API** (Port 1317) - `oracle.knirv.com`
- **Bootnode Registry** (Port 3006) - `bootnode-registry.knirv.com`
- **Tunnel Registry** (Port 3003) - `tunnel-registry.knirv.com`
- **Notary System** (Port 3007) - `notary-system.knirv.com`
- **Network Monitor** (Port 3008) - `network-monitor.knirv.com`
- **NANDA-ANS** (Port 3009) - `nanda-ans.knirv.com`

## Configuration

### Environment Variables

Key environment variables for production:

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



**Production Deployment Checklist:**

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

**Success Metrics:**
- 99.9% uptime SLA
- <500ms average API response time
- <5% error rate under normal load
- Support for 10,000+ concurrent users
- 24/7 monitoring and alerting

---
### Future Sprints
- [ ] Update Kubernetes deployments
- [ ] Performance optimization
- [ ] Advanced monitoring integration
- [ ] Security hardening
- [ ] Production deployment
