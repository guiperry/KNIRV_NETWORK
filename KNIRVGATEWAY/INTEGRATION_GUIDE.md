# KNIRV Network Integration Guide

This document provides comprehensive guidance for integrating with the KNIRV Network API Gateway and component communication layer implemented in Month 10.

## Overview

The KNIRV Network now features a unified API Gateway that provides:

- **Centralized Access Point**: Single entry point for all KNIRV components
- **Service Discovery**: Automatic registration and health monitoring
- **Authentication & Authorization**: Token-based security across all services
- **Rate Limiting**: Protection against abuse and overload
- **Real-time Communication**: WebSocket support for live updates
- **Monitoring & Metrics**: Comprehensive observability

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Client Apps   │    │   Web Frontend  │    │  External APIs  │
└─────────┬───────┘    └─────────┬───────┘    └─────────┬───────┘
          │                      │                      │
          └──────────────────────┼──────────────────────┘
                                 │
                    ┌─────────────▼─────────────┐
                    │     API Gateway          │
                    │     (Port 8000)          │
                    │                          │
                    │ • Authentication         │
                    │ • Rate Limiting          │
                    │ • Service Discovery      │
                    │ • Health Monitoring      │
                    │ • WebSocket Support      │
                    └─────────────┬─────────────┘
                                  │
          ┌───────────────────────┼───────────────────────┐
          │                       │                       │
    ┌─────▼─────┐    ┌─────▼─────┐    ┌─────▼─────┐    ┌─────▼─────┐    ┌─────▼─────┐
    │KNIRVCHAIN │    │KNIRVGRAPH │    │KNIRVNEXUS │    │ KNIRVROOT │    │KNIRVROUTER│
    │Port 8080  │    │Port 8081  │    │Port 8082  │    │Port 5000  │    │Port 3478  │
    └───────────┘    └───────────┘    └───────────┘    └───────────┘    └───────────┘
```

## Quick Start

### 1. Start the API Gateway

```bash
cd shared-integration
./api-gateway/start-gateway.sh start
```

### 2. Verify Gateway Status

```bash
curl http://localhost:8000/gateway/health
```

### 3. List Available Services

```bash
curl http://localhost:8000/gateway/services
```

## Service Integration

### KNIRVCHAIN Integration

**Base URL**: `http://localhost:8000/knirvchain`

**Available Endpoints**:
- `GET /knirvchain/wallets/*` - Wallet operations (no auth)
- `GET|POST /knirvchain/nrn/*` - NRN token operations (auth required)
- `GET|POST /knirvchain/skill/*` - Skill management (auth required)
- `GET|POST /knirvchain/llm/*` - LLM operations (auth required)
- `GET /knirvchain/blocks` - Block information (no auth)

**Example**:
```bash
# Get blocks (no authentication)
curl http://localhost:8000/knirvchain/blocks

# Get NRN balance (authentication required)
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8000/knirvchain/nrn/balance/address123
```

### KNIRVGRAPH Integration

**Base URL**: `http://localhost:8000/knirvgraph`

**Available Endpoints**:
- `GET /knirvgraph/height` - Graph height (no auth)
- `GET|POST /knirvgraph/node/*` - Node operations (no auth)
- `GET|POST /knirvgraph/edge/*` - Edge operations (no auth)
- `GET|POST /knirvgraph/graph/*` - Graph operations (no auth)
- `GET /knirvgraph/account/*` - Account queries (no auth)
- `POST /knirvgraph/transaction` - Submit transaction (auth required)
- `GET|POST /knirvgraph/nrv/*` - NRV operations (auth required)

**Example**:
```bash
# Get graph height
curl http://localhost:8000/knirvgraph/height

# Submit transaction (authentication required)
curl -X POST -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"from":"addr1","to":"addr2","amount":100}' \
  http://localhost:8000/knirvgraph/transaction
```

### KNIRVNEXUS Integration

**Base URL**: `http://localhost:8000/knirvnexus`

**Available Endpoints**:
- `GET|POST|PUT|DELETE /knirvnexus/api/v1/agents/*` - Agent management (auth required)
- `GET|POST /knirvnexus/api/v1/workflows/*` - Workflow operations (auth required)
- `GET|POST /knirvnexus/api/v1/mcp/*` - MCP operations (auth required)
- `GET|POST /knirvnexus/api/v1/inference/*` - AI inference (auth required)
- `GET /knirvnexus/desktop/*` - Desktop interface (no auth)

**Example**:
```bash
# List agents (authentication required)
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8000/knirvnexus/api/v1/agents

# Create workflow (authentication required)
curl -X POST -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"test-workflow","steps":[]}' \
  http://localhost:8000/knirvnexus/api/v1/workflows
```

### KNIRVROOT Integration

**Base URL**: `http://localhost:8000/knirvroot`

**Available Endpoints**:
- `GET /knirvroot/chain` - Chain information (no auth)
- `POST /knirvroot/block` - Submit block (auth required)
- `POST /knirvroot/transaction` - Submit transaction (auth required)
- `GET|POST /knirvroot/mcp/*` - MCP operations (auth required)
- `GET|POST /knirvroot/payment/*` - Payment processing (auth required)
- `GET|POST /knirvroot/bridge/*` - Bridge operations (auth required)
- `POST /knirvroot/test/faucet` - Test faucet (no auth)
- `GET /knirvroot/ping` - Health check (no auth)

**Example**:
```bash
# Get chain info
curl http://localhost:8000/knirvroot/chain

# Use test faucet
curl -X POST -H "Content-Type: application/json" \
  -d '{"address":"test_address"}' \
  http://localhost:8000/knirvroot/test/faucet
```

### KNIRVROUTER Integration

**Base URL**: `http://localhost:8000/knirvrouter`

**Available Endpoints**:
- `GET|POST /knirvrouter/api/connectivity/*` - Connectivity operations (no auth)
- `GET|POST /knirvrouter/api/proof/*` - Proof operations (no auth)
- `POST /knirvrouter/api/mint/*` - Minting operations (auth required)
- `GET /knirvrouter/api/stats/*` - Statistics (no auth)
- `GET|POST /knirvrouter/turn/*` - TURN server operations (no auth)
- `GET /knirvrouter/ws` - WebSocket endpoint (no auth)

**Example**:
```bash
# Get connectivity stats
curl http://localhost:8000/knirvrouter/api/stats/connectivity

# Generate proof
curl -X POST -H "Content-Type: application/json" \
  -d '{"data":"proof_data"}' \
  http://localhost:8000/knirvrouter/api/proof/generate
```

## Authentication

### Getting a Token

```bash
curl -X POST http://localhost:8000/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "password"}'
```

Response:
```json
{
  "token": "token_admin_1234567890",
  "user": "admin"
}
```

### Using Tokens

Include the token in requests using the Authorization header:

```bash
curl -H "Authorization: Bearer token_admin_1234567890" \
  http://localhost:8000/knirvchain/nrn/balance/address123
```

Or as a query parameter:
```bash
curl "http://localhost:8000/knirvchain/nrn/balance/address123?token=token_admin_1234567890"
```

### Token Management

```bash
# Validate token
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8000/auth/validate

# Logout (revoke token)
curl -X POST -H "Authorization: Bearer $TOKEN" \
  http://localhost:8000/auth/logout
```

## WebSocket Communication

Connect to `ws://localhost:8000/gateway/ws` for real-time updates:

```javascript
const ws = new WebSocket('ws://localhost:8000/gateway/ws');

// Send ping
ws.send(JSON.stringify({type: 'ping'}));

// Subscribe to service events
ws.send(JSON.stringify({
  type: 'subscribe',
  service: 'knirvchain'
}));

// Get real-time metrics
ws.send(JSON.stringify({type: 'get_metrics'}));

// Handle responses
ws.onmessage = function(event) {
  const data = JSON.parse(event.data);
  console.log('Received:', data);
};
```

## Monitoring & Observability

### Health Checks

```bash
# Gateway health
curl http://localhost:8000/gateway/health

# Service health (via gateway)
curl http://localhost:8000/gateway/services
```

### Metrics

```bash
# Gateway metrics
curl http://localhost:8000/gateway/metrics
```

Response includes:
- Total requests
- Success/failure rates
- Per-service metrics
- Average response times

### Service Registration

Register new services dynamically:

```bash
curl -X POST http://localhost:8000/gateway/services \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-service",
    "url": "http://localhost:9000",
    "health_path": "/health",
    "timeout": "30s",
    "routes": [
      {
        "path": "/api/*",
        "methods": ["GET", "POST"],
        "auth_required": true
      }
    ]
  }'
```

## Error Handling

The gateway returns standard HTTP status codes:

- `200` - Success
- `401` - Authentication required
- `403` - Forbidden (insufficient permissions)
- `404` - Service or endpoint not found
- `429` - Rate limit exceeded
- `503` - Service unavailable (unhealthy)
- `500` - Internal server error

## Rate Limiting

Default rate limit: 100 requests per minute per client IP.

When rate limited, you'll receive:
```json
{
  "error": "Rate limit exceeded"
}
```

## Best Practices

1. **Always use HTTPS in production**
2. **Store tokens securely** (not in localStorage for web apps)
3. **Handle rate limiting gracefully** with exponential backoff
4. **Monitor service health** before making requests
5. **Use WebSocket for real-time features** instead of polling
6. **Implement proper error handling** for all status codes
7. **Cache authentication tokens** until expiration

## Migration from Direct Service Access

If you're currently accessing KNIRV services directly:

**Before**:
```bash
curl http://localhost:8080/blocks  # Direct KNIRVCHAIN access
```

**After**:
```bash
curl http://localhost:8000/knirvchain/blocks  # Via gateway
```

Benefits of using the gateway:
- Centralized authentication
- Rate limiting protection
- Health monitoring
- Metrics collection
- Future-proof API versioning

## Troubleshooting

### Common Issues

1. **503 Service Unavailable**: Target service is not running or unhealthy
2. **401 Unauthorized**: Missing or invalid authentication token
3. **429 Rate Limited**: Too many requests, implement backoff
4. **404 Not Found**: Check service name and path in URL

### Debug Commands

```bash
# Check gateway status
./api-gateway/start-gateway.sh status

# View gateway logs
./api-gateway/start-gateway.sh logs

# Test gateway functionality
./api-gateway/test-gateway.sh

# Check service registration
curl http://localhost:8000/gateway/services | jq .
```

## Support

For issues with the API Gateway:
1. Check the logs: `./api-gateway/start-gateway.sh logs`
2. Verify service health: `curl http://localhost:8000/gateway/health`
3. Run the test suite: `./api-gateway/test-gateway.sh`
4. Check individual service status in the services list
