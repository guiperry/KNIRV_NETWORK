

---

**Source**: KNIRVGRAPH/docs/NRV_SYSTEM.md

# KNIRV Network Resolution Vector (NRV) System

## Overview

The Network Resolution Vector (NRV) system is a core component of KNIRVGRAPH that implements Month 5 of the KNIRV_D-TEN Comprehensive Implementation Plan. It provides a distributed vector-based approach to network resource resolution, error tracking, and skill-based problem resolution.

## Architecture

### Core Components

1. **NetworkResolutionVector**: Represents a vector in the NRV system with coordinates, confidence, and metadata
2. **ErrorNode**: Represents errors in the system that need resolution
3. **SkillNode**: Represents skills that can resolve specific types of errors
4. **NRVSystem**: The main system that manages vectors, errors, and skills

### Key Features

- **Vector Management**: Create, store, and resolve network resolution vectors
- **Error Tracking**: Track and categorize system errors
- **Skill Matching**: Match skills to error types for automated resolution
- **Confidence Decay**: Automatic confidence degradation over time
- **Distributed Storage**: Support for DHT-based distributed storage (future enhancement)
- **REST API**: Complete HTTP API for all NRV operations

## API Endpoints

### Vector Operations

#### Get All Vectors
```
GET /nrv/vectors
```
Returns all vectors in the system.

#### Create Vector
```
POST /nrv/vectors
Content-Type: application/json

{
  "target_hash": "string",
  "coordinates": [float64, ...],
  "metadata": {
    "key": "value"
  }
}
```

#### Resolve Target
```
GET /nrv/vectors/resolve/{targetHash}
```
Returns all vectors that resolve to the specified target hash.

### Error Operations

#### Get All Errors
```
GET /nrv/errors
```
Returns all error nodes in the system.

#### Create Error
```
POST /nrv/errors
Content-Type: application/json

{
  "error_type": "string",
  "description": "string",
  "context": {
    "key": "value"
  },
  "severity": int
}
```

### Skill Operations

#### Get All Skills
```
GET /nrv/skills
```
Returns all skill nodes in the system.

#### Create Skill
```
POST /nrv/skills
Content-Type: application/json

{
  "skill_type": "string",
  "capabilities": ["string", ...],
  "requirements": {
    "key": "value"
  }
}
```

#### Get Skills for Error Type
```
GET /nrv/skills/for-error/{errorType}
```
Returns all skills that can handle the specified error type.

## Usage Examples

### Creating a Network Resolution Vector

```bash
curl -X POST http://localhost:8080/nrv/vectors \
  -H "Content-Type: application/json" \
  -d '{
    "target_hash": "blockchain-node-1",
    "coordinates": [1.0, 2.0, 3.0],
    "metadata": {
      "type": "blockchain_node",
      "location": "us-east-1",
      "capacity": 1000
    }
  }'
```

### Creating an Error Node

```bash
curl -X POST http://localhost:8080/nrv/errors \
  -H "Content-Type: application/json" \
  -d '{
    "error_type": "network_connectivity",
    "description": "Unable to connect to blockchain node",
    "context": {
      "target_node": "blockchain-node-1",
      "error_code": "CONN_TIMEOUT"
    },
    "severity": 3
  }'
```

### Creating a Skill Node

```bash
curl -X POST http://localhost:8080/nrv/skills \
  -H "Content-Type: application/json" \
  -d '{
    "skill_type": "network_diagnostics",
    "capabilities": ["network_connectivity", "dns_resolution"],
    "requirements": {
      "min_confidence": 0.8,
      "timeout": 30
    }
  }'
```

## Configuration

The NRV system can be configured using the `NRVConfig` struct:

```go
type NRVConfig struct {
    MaxVectors        int           // Maximum number of vectors to store
    VectorTTL         time.Duration // Time-to-live for vectors
    ConfidenceDecay   float64       // Confidence decay factor (0-1)
    ValidationTimeout time.Duration // Timeout for validation operations
    DHTBootstrapPeers []string      // DHT bootstrap peers (future)
}
```

Default configuration:
- MaxVectors: 10,000
- VectorTTL: 24 hours
- ConfidenceDecay: 0.95
- ValidationTimeout: 30 seconds

## Integration

The NRV system is integrated into KNIRVGRAPH through:

1. **App Integration**: The `App` struct includes an `nrvSystem` field
2. **RPC Integration**: All NRV endpoints are exposed through the RPC server
3. **Storage Integration**: Vectors, errors, and skills are stored in memory with future persistence support

## Testing

Run the NRV system tests:

```bash
go test ./internal/nrv/
```

Run the demo application:

```bash
go run examples/nrv_demo.go
```

## Future Enhancements

1. **DHT Integration**: Full libp2p DHT support for distributed storage
2. **Persistence**: Database storage for vectors, errors, and skills
3. **Advanced Algorithms**: Machine learning for vector optimization
4. **Monitoring**: Metrics and monitoring for system performance
5. **Security**: Cryptographic signatures and validation
6. **Clustering**: Support for multi-node NRV clusters

## Implementation Status

✅ **Completed (Month 5)**:
- Core NRV system implementation
- Vector management and resolution
- Error node creation and tracking
- Skill node creation and matching
- REST API endpoints
- Integration with KNIRVGRAPH
- Comprehensive testing
- Documentation and examples

🔄 **Future Phases**:
- DHT integration (Month 6)
- Persistence layer (Month 7)
- Advanced algorithms (Month 8)
- Production deployment (Month 9)

## Performance Characteristics

- **Vector Creation**: O(1) time complexity
- **Target Resolution**: O(n) where n is the number of vectors
- **Skill Matching**: O(m*k) where m is skills and k is capabilities
- **Memory Usage**: Approximately 1KB per vector/error/skill node
- **Concurrent Operations**: Thread-safe with read-write mutexes

## Error Handling

The NRV system includes comprehensive error handling:

- Invalid input validation
- Resource not found errors
- System unavailable responses
- Graceful degradation under load
- Automatic cleanup of expired resources

This completes the Month 5 implementation of the KNIRV_D-TEN Comprehensive Implementation Plan for the NRV system in KNIRVGRAPH.


---

<div class="footer-links">


© 2025 KNIRV Network
</div>
