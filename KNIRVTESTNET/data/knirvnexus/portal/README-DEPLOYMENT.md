# KNIRV-NEXUS Deployment Guide

## Overview

KNIRV-NEXUS has been refactored to support two operational modes:
- **Headless Mode**: API-only service for production deployments
- **GUI Mode**: Includes web interface for development and testing

## Architecture Changes

### Removed Components
- ❌ Standalone API Gateway service
- ❌ Separate frontend deployment

### New Components
- ✅ Integrated operational modes (headless/GUI)
- ✅ Viper-based configuration management
- ✅ Role-based authentication (Admin, Validator, Observer)
- ✅ KNIRVGATEWAY integration with SSE support
- ✅ NEXUS Portal in KNIRVGATEWAY

## Quick Start

### 1. Development Mode (GUI)

```bash
# Clone and setup
git clone <repository>
cd KNIRVNEXUS

# Install dependencies
npm install

# Run in GUI mode (includes frontend)
go run cmd/main.go --mode=gui --config=testnet

# Access the interface
open http://localhost:9080
```

### 2. Production Mode (Headless)

```bash
# Build for production
go build -o knirv-nexus cmd/main.go

# Run in headless mode (API only)
./knirv-nexus --mode=headless --config=production

# API available at http://localhost:8080
```

## Configuration

### Environment Variables

**Testnet (.env.testnet):**
```bash
KNIRV_MODE=gui
KNIRV_GUI_ENABLED=true
KNIRV_AUTH_REQUIRED=false
KNIRV_CHAIN_ID=knirv-nexus-testnet
```

**Production (.env.production):**
```bash
KNIRV_MODE=headless
KNIRV_GUI_ENABLED=false
KNIRV_AUTH_REQUIRED=true
KNIRV_CHAIN_ID=knirv-nexus-mainnet
```

### CLI Flags

```bash
# Operational modes
--mode=headless|gui          # Set operational mode
--config=testnet|production  # Load environment config

# Configuration management
--validate-config            # Validate configuration
--config-test               # Test configuration loading

# Service options
--api-port=8080             # Override API port
--gui-port=9080             # Override GUI port (GUI mode only)
```

## Role-Based Authentication

### User Roles

1. **Admin** - Full system access
   - Permissions: `*:*`
   - NEXUS Access: `dve:*`, `validation:*`, `system:*`

2. **Validator** - Scoped validation access
   - Permissions: `nexus:read`, `nexus:validate`, `nexus:update_assigned`
   - NEXUS Access: `dve:read`, `validation:read`, `validation:execute`, `system:read`

3. **Observer** - Read-only access
   - Permissions: `*:read`
   - NEXUS Access: `dve:read`, `validation:read`, `system:read`

### Testnet Tokens

```bash
# Admin access
curl -H "Authorization: Bearer testnet-admin-123" \
  http://localhost:8080/api/system/status

# Validator access
curl -H "Authorization: Bearer testnet-validator-456" \
  http://localhost:8080/api/validation/tasks

# Observer access
curl -H "Authorization: Bearer testnet-observer-789" \
  http://localhost:8080/api/dve/nodes
```

## KNIRVGATEWAY Integration

### NEXUS Portal

The NEXUS Portal is now integrated into KNIRVGATEWAY:

```bash
# Access via KNIRVGATEWAY
https://gateway.knirv.com/nexus/

# Local development
http://localhost:8888/nexus/
```

### API Routes

All NEXUS APIs are accessible through KNIRVGATEWAY:

```bash
# DVE Nodes
GET /gateway/nexus/dve-nodes

# Validation Tasks
GET /gateway/nexus/validation-tasks
POST /gateway/nexus/validation-tasks

# System Status
GET /gateway/nexus/system/status
GET /gateway/nexus/system/metrics
```

### Real-time Updates

SSE channels for real-time updates:

```bash
# System updates
GET /gateway/events/nexus-system

# DVE updates
GET /gateway/events/nexus-dve

# Validation updates
GET /gateway/events/nexus-validation
```

## Deployment Options

### 1. Kubernetes (Production)

```bash
# Deploy to Kubernetes
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/secrets.yaml
kubectl apply -f k8s/dve-manager-deployment.yaml
kubectl apply -f k8s/validation-core-deployment.yaml

# Check status
kubectl get pods -n knirv-nexus
```

### 2. Docker (Development)

```bash
# Build image
docker build -t knirv/nexus:latest .

# Run headless mode
docker run -p 8080:8080 \
  -e KNIRV_MODE=headless \
  knirv/nexus:latest

# Run GUI mode
docker run -p 8080:8080 -p 9080:9080 \
  -e KNIRV_MODE=gui \
  knirv/nexus:latest
```

### 3. Netlify (KNIRVGATEWAY)

```bash
# Deploy KNIRVGATEWAY with NEXUS Portal
cd KNIRVGATEWAY
netlify deploy --prod

# NEXUS Portal will be available at:
# https://your-site.netlify.app/nexus/
```

## Testing

### Run All Tests

```bash
# Complete test suite
./scripts/run-all-tests.sh

# Individual test suites
./KNIRVNEXUS/scripts/test-operational-modes.sh
./KNIRVGATEWAY/scripts/test-gateway-integration.sh
./KNIRVNEXUS/scripts/test-frontend.sh
```

### Test Results

Test results are saved to `test-results/` directory:
- `complete-test-suite.log` - Full test log
- `test-report.md` - Summary report
- Individual test logs

## Monitoring

### Health Checks

```bash
# Service health
curl http://localhost:8080/health

# Gateway health
curl http://localhost:8888/.netlify/functions/gateway-sse/gateway/health

# System status
curl http://localhost:8080/api/system/status
```

### Metrics

```bash
# Prometheus metrics
curl http://localhost:8080/metrics

# System metrics via gateway
curl http://localhost:8888/.netlify/functions/gateway-sse/nexus/system/metrics
```

## Troubleshooting

### Common Issues

1. **GUI not accessible in headless mode**
   - Expected behavior - GUI is disabled in headless mode
   - Use `--mode=gui` to enable GUI

2. **Authentication failures**
   - Check token format: `Bearer <token>`
   - Verify role permissions for endpoint
   - Use testnet tokens for development

3. **Configuration errors**
   - Run `--validate-config` to check configuration
   - Verify environment variables are set
   - Check file permissions for config files

### Logs

```bash
# Service logs
tail -f /app/logs/nexus.log

# Docker logs
docker logs <container-id>

# Kubernetes logs
kubectl logs -f deployment/dve-manager -n knirv-nexus
```

## Migration from Previous Version

### Breaking Changes

1. **API Gateway Removed**
   - All APIs now served directly by services
   - Use KNIRVGATEWAY for unified access

2. **Configuration Changes**
   - New Viper-based configuration
   - Environment-specific config files
   - CLI flags for operational modes

3. **Authentication Updates**
   - Role-based access control
   - New token format and permissions
   - Scoped access for validators

### Migration Steps

1. Update configuration files
2. Update deployment scripts
3. Test with new authentication tokens
4. Update client applications to use new API endpoints
5. Deploy and verify functionality

## Support

For issues and questions:
- Check logs in `test-results/` directory
- Review configuration with `--validate-config`
- Run test suite to identify issues
- Consult troubleshooting section above
