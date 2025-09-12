# DHT Control API

The KNIRVGATEWAY now supports on-demand DHT (Distributed Hash Table) initialization through REST API endpoints. This allows the gateway to start successfully without DHT and enables DHT functionality when needed.

## Overview

- **Default Behavior**: DHT is disabled by default to ensure reliable deployment
- **On-Demand Start**: Use API endpoints to start DHT when ready
- **Full Control**: Start, stop, restart, and monitor DHT status

## API Endpoints

### Health Check
```
GET /health
```
Returns gateway status including DHT initialization state.

**Response:**
```json
{
  "status": "healthy",
  "mode": "persistent",
  "timestamp": 1694123456789,
  "chainId": "testnet",
  "dhtInitialized": false,
  "dhtStartupInProgress": false,
  "dht": {
    "status": "not_initialized"
  }
}
```

### Start DHT
```
POST /dht/start
```
Initializes and starts the DHT network.

**Response (Success):**
```json
{
  "message": "DHT started successfully",
  "status": {
    "status": "running",
    "connectedPeers": 3,
    "discoveredServices": 1
  }
}
```

### Stop DHT
```
POST /dht/stop
```
Stops the DHT network and cleans up resources.

**Response:**
```json
{
  "message": "DHT stopped successfully"
}
```

### Restart DHT
```
GET /dht/restart
```
Stops and restarts the DHT network.

**Response:**
```json
{
  "message": "DHT restarted successfully",
  "status": {
    "status": "running",
    "connectedPeers": 2,
    "discoveredServices": 0
  }
}
```

### DHT Status
```
GET /dht/status
```
Returns detailed DHT network status.

**Response:**
```json
{
  "status": "running",
  "connectedPeers": 3,
  "discoveredServices": 1,
  "networkId": "testnet",
  "peerId": "12D3KooW..."
}
```

## Usage Examples

### Using curl

```bash
# Check gateway health
curl https://your-app.onrender.com/health

# Start DHT
curl -X POST https://your-app.onrender.com/dht/start

# Check DHT status
curl https://your-app.onrender.com/dht/status

# Stop DHT
curl -X POST https://your-app.onrender.com/dht/stop

# Restart DHT
curl https://your-app.onrender.com/dht/restart
```

### Using the test script

```bash
# Check status
npm run dht:control:status

# Start DHT
npm run dht:control:start

# Stop DHT
npm run dht:control:stop

# Restart DHT
npm run dht:control:restart

# Test against remote server
node scripts/test-dht-control.js start https://your-app.onrender.com
```

## Deployment Workflow

1. **Deploy**: Gateway starts successfully with DHT disabled
2. **Verify**: Check `/health` endpoint to confirm deployment
3. **Initialize**: Send `POST /dht/start` when ready to enable P2P features
4. **Monitor**: Use `/dht/status` to monitor network connectivity

## Environment Variables

The following environment variables affect DHT behavior:

- `DISABLE_DHT=true` - Explicitly disable DHT (overrides API control)
- `KNIRV_BOOTSTRAP_PEERS` - Comma-separated list of bootstrap peers
- `DHT_PORT` - Port for DHT networking (default: auto-assign)
- `INTERNAL_API_KEY` - API key for internal communication

## Error Handling

- DHT start failures are gracefully handled and logged
- Gateway continues operating even if DHT fails
- API endpoints return appropriate error codes and messages
- Provision endpoint returns empty array when DHT is unavailable

## Benefits

1. **Reliable Deployment**: Gateway starts without network dependencies
2. **Debugging**: Easy to isolate DHT issues from basic gateway functionality
3. **Flexibility**: Start DHT only when needed or when network conditions are optimal
4. **Monitoring**: Real-time visibility into DHT status and performance
