# Tunnel Registry Protocol Fix

## Problem
The tunnel registry services were receiving HTTP requests on their protocol ports, causing JSON parsing errors:

```
ERROR: [ControlListener] Error processing data from ::1:60478-5wx6v: Unexpected token 'H', "HEAD / HTT"... is not valid JSON
ERROR: [PublicRelayListener] Invalid initial message from external client: HEAD / HTTP/1.1. Error: Unexpected token 'H', "HEAD / HTTP/1.1" is not valid JSON
```

## Root Cause
- **Control Listener** (port 4001) and **Public Relay Listener** (port 4000) expect JSON protocol messages
- External clients (web browsers, health checks, load balancers) were making HTTP requests to these ports
- The services were trying to parse HTTP requests as JSON, which failed

## Solution Implemented

### 1. Protocol Detection
Both listeners now detect HTTP requests by checking for HTTP methods at the beginning of incoming data:
- GET, POST, HEAD, PUT, DELETE, OPTIONS

### 2. Graceful HTTP Response
When an HTTP request is detected, the services now respond with:
- HTTP 200 OK status
- Informative message about the service purpose
- Instructions to use the correct HTTP API port (3004)

### 3. Files Modified

#### `tunneling/controlListener.js`
- Added HTTP request detection and proper response
- Fixed `connectionRegistry` reference to use global variable
- Maintains backward compatibility with JSON protocol

#### `tunneling/publicRelayListener.js`  
- Added HTTP request detection and proper response
- Provides clear instructions for tunnel protocol usage

## Expected Behavior After Fix

### For HTTP Requests (Web Browsers, Health Checks)
```
Client sends: HEAD / HTTP/1.1
Service responds: HTTP/1.1 200 OK with service information
```

### For Protocol Requests (Internal Nodes, Tunnel Clients)
```
Client sends: {"action": "IDENTIFY", "devId": "Qm..."}
Service responds: JSON protocol response
```

## Testing

### Test HTTP Requests
```bash
# Test Control Listener (port 4001)
curl -X HEAD http://localhost:4001/

# Test Public Relay Listener (port 4000)  
curl -X HEAD http://localhost:4000/
```

### Test Protocol Requests
```bash
# Test Control Listener with JSON
echo '{"action": "PING"}' | nc localhost 4001

# Test Public Relay Listener with PeerID
echo 'QmExamplePeerId' | nc localhost 4000
```

## Benefits

1. **No More Error Spam**: HTTP requests no longer cause JSON parsing errors
2. **Better User Experience**: Clear error messages for misconfigured clients
3. **Service Discovery**: HTTP responses help users understand service purposes
4. **Backward Compatibility**: JSON protocol continues to work unchanged

## Deployment
The fix is automatically applied when the tunnel registry service restarts. No configuration changes required.