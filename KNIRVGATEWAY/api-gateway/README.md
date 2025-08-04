# KNIRV Network API Gateway

The KNIRV Network API Gateway is a unified entry point for all KNIRV components, providing service discovery, load balancing, authentication, rate limiting, and real-time communication capabilities.

## Features

- **Service Registration & Discovery**: Automatic registration and health monitoring of KNIRV components
- **Request Routing**: Intelligent routing based on service names and paths
- **Authentication**: Token-based authentication with configurable scopes
- **Rate Limiting**: Per-client rate limiting to prevent abuse
- **Health Monitoring**: Continuous health checks of registered services
- **WebSocket Support**: Real-time communication for live updates
- **Metrics Collection**: Request metrics and performance monitoring
- **CORS Support**: Cross-origin resource sharing for web applications

## Architecture

The API Gateway acts as a reverse proxy that sits between clients and KNIRV services:

```
Client → API Gateway → KNIRV Services
                    ├── KNIRVCHAIN (port 8080)
                    ├── KNIRVGRAPH (port 8081)
                    ├── KNIRVNEXUS (port 8082)
                    ├── KNIRVROOT (port 5000)
                    └── KNIRVROUTER (port 3478)
```

## Quick Start

### 1. Build the Gateway

```bash
cd shared-integration
go build -o api-gateway/gateway api-gateway/gateway.go
```

### 2. Start the Gateway

```bash
# Using the startup script (recommended)
./api-gateway/start-gateway.sh start

# Or run directly
./api-gateway/gateway
```

### 3. Verify Operation

```bash
# Check gateway health
curl http://localhost:8000/gateway/health

# View registered services
curl http://localhost:8000/gateway/services

# Check metrics
curl http://localhost:8000/gateway/metrics
```

## API Endpoints

### Gateway Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/gateway/health` | Gateway health status |
| GET | `/gateway/metrics` | Performance metrics |
| GET | `/gateway/services` | List registered services |
| POST | `/gateway/services` | Register a new service |
| PUT | `/gateway/services/{service}` | Update service configuration |
| DELETE | `/gateway/services/{service}` | Unregister a service |

### Authentication

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/auth/login` | Authenticate and get token |
| POST | `/auth/logout` | Revoke authentication token |
| GET | `/auth/validate` | Validate authentication token |

### Service Proxying

All requests to `/{service}/*` are proxied to the corresponding KNIRV component:

- `/knirvchain/*` → KNIRVCHAIN service
- `/knirvgraph/*` → KNIRVGRAPH service
- `/knirvnexus/*` → KNIRVNEXUS service
- `/knirvroot/*` → KNIRVROOT service
- `/knirvrouter/*` → KNIRVROUTER service

### PoAu-D Consensus Management

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| POST | `/poaud/enable` | Enable PoAu-D consensus mechanism | Yes |
| POST | `/poaud/disable` | Disable PoAu-D (fallback to PoW) | Yes |
| GET | `/poaud/status` | Get PoAu-D status and statistics | No |
| POST | `/poaud/network-authors/add` | Add Network Author Peer | Yes |
| POST | `/poaud/network-authors/remove` | Remove Network Author Peer | Yes |
| GET | `/poaud/network-authors` | List all Network Author Peers | No |

### WebSocket

- `ws://localhost:8000/gateway/ws` - Real-time gateway communication

## Configuration

The gateway can be configured via `config.yaml`:

```yaml
gateway:
  port: 8000
  rate_limit:
    requests_per_minute: 100
  auth:
    token_duration: "24h"

services:
  knirvchain:
    url: "http://localhost:8080"
    health_path: "/health"
    timeout: "30s"
```

## Authentication

### Login

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

Include the token in requests:

```bash
# Header method (recommended)
curl -H "Authorization: Bearer token_admin_1234567890" \
  http://localhost:8000/knirvchain/nrn/balance/address123

# Query parameter method
curl "http://localhost:8000/knirvchain/nrn/balance/address123?token=token_admin_1234567890"
```

## PoAu-D Consensus Management

### Enable PoAu-D Consensus

```bash
curl -X POST http://localhost:8000/poaud/enable \
  -H "Authorization: Bearer token_admin_1234567890"
```

### Check PoAu-D Status

```bash
curl http://localhost:8000/poaud/status
```

### Add Network Author

```bash
curl -X POST http://localhost:8000/poaud/network-authors/add \
  -H "Authorization: Bearer token_admin_1234567890" \
  -H "Content-Type: application/json" \
  -d '{"address": "knirv1abc123def456ghi789"}'
```

### List Network Authors

```bash
curl http://localhost:8000/poaud/network-authors
```

### Remove Network Author

```bash
curl -X POST http://localhost:8000/poaud/network-authors/remove \
  -H "Authorization: Bearer token_admin_1234567890" \
  -H "Content-Type: application/json" \
  -d '{"address": "knirv1abc123def456ghi789"}'
```

### Disable PoAu-D Consensus

```bash
curl -X POST http://localhost:8000/poaud/disable \
  -H "Authorization: Bearer token_admin_1234567890"
```

## WebSocket Communication

Connect to the WebSocket endpoint for real-time updates:

```javascript
const ws = new WebSocket('ws://localhost:8000/gateway/ws');

// Send ping
ws.send(JSON.stringify({type: 'ping'}));

// Subscribe to service events
ws.send(JSON.stringify({
  type: 'subscribe',
  service: 'knirvchain'
}));

// Get metrics
ws.send(JSON.stringify({type: 'get_metrics'}));
```

## Management Script

Use the provided script for easy management:

```bash
# Start the gateway
./api-gateway/start-gateway.sh start

# Check status
./api-gateway/start-gateway.sh status

# View logs
./api-gateway/start-gateway.sh logs

# Stop the gateway
./api-gateway/start-gateway.sh stop

# Restart the gateway
./api-gateway/start-gateway.sh restart
```

## Service Registration

Services are automatically registered on startup, but you can also register services dynamically:

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

## Monitoring

### Health Checks

The gateway continuously monitors service health:

- Health checks run every 30 seconds
- Services are marked unhealthy if health endpoint fails
- Unhealthy services receive no traffic
- Health status changes are broadcast via WebSocket

### Metrics

Available metrics include:

- Total requests
- Successful/failed requests
- Per-service metrics (requests, errors, latency)
- Average response times

## Security

- **Authentication**: Token-based with configurable expiration
- **Rate Limiting**: Per-client request limiting
- **CORS**: Configurable cross-origin policies
- **Headers**: Custom headers added to proxied requests

## Troubleshooting

### Common Issues

1. **Service not responding**: Check if the target service is running
2. **Authentication failures**: Verify token validity and scopes
3. **Rate limiting**: Check if client is exceeding request limits
4. **CORS errors**: Verify CORS configuration for web clients

### Logs

Check the gateway logs for detailed information:

```bash
./api-gateway/start-gateway.sh logs
```

### Health Status

Monitor service health:

```bash
curl http://localhost:8000/gateway/health
```

## Development

### Building

```bash
go build -o api-gateway/gateway api-gateway/gateway.go
```

### Testing

```bash
# Test gateway health
curl http://localhost:8000/gateway/health

# Test service proxying
curl http://localhost:8000/knirvchain/blocks

# Test authentication
curl -X POST http://localhost:8000/auth/login \
  -d '{"username": "admin", "password": "password"}'
```

## Integration with KNIRV Components

The gateway is designed to work seamlessly with all KNIRV components:

- **KNIRVCHAIN**: Blockchain operations and NRN token management
- **KNIRVGRAPH**: Graph database queries and NRV operations  
- **KNIRVNEXUS**: Agent management and workflow execution
- **KNIRVROOT**: MCP operations and payment processing
- **KNIRVROUTER**: Connectivity proofs and network routing

Each component maintains its existing API while gaining the benefits of centralized authentication, monitoring, and management through the gateway.
