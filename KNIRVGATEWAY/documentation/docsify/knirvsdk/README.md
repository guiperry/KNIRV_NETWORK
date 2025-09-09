# KNIRV SDK

Software Development Kits for the KNIRV Network ecosystem, including complete KNIRVGATEWAY integration.

## Available SDKs

### Gateway SDKs ⭐ NEW
- **[Go Gateway SDK](#/go/gateway/README.md)** - Complete KNIRVGATEWAY integration for Go
- **[TypeScript Gateway SDK](#/ts/gateway/README.md)** - Complete KNIRVGATEWAY integration for TypeScript/JavaScript
- **[Python Gateway SDK](#/py/gateway/README.md)** - ✅ **NEW!** Complete KNIRVGATEWAY integration for Python

### Core KNIRV SDKs
- [Go Transaction SDK](#/go/transaction/README.md) - KNIRVCHAIN transaction management
- [Go Transmission SDK](#/go/transmission/README.md) - Network transmission and communication
- [TypeScript Transmission SDK](#/ts/transmission/README.md) - Web transmission libraries
- [Python Transaction SDK](#/py/transaction/README.md) - KNIRVCHAIN transaction management for Python
- [Python Transmission SDK](#/py/transmission/README.md) - Network transmission for Python

### Unified SDKs 🚧 In Development
- **[TypeScript Unified SDK](#/ts/unified/README.md)** - 🚧 Combined access to all KNIRV services
- **Python Unified SDK** - 🚧 *Planned* - Combined access to all KNIRV services
- **Go Unified SDK** - 🚧 *Planned* - Combined access to all KNIRV services

> **Note**: See [UNIFIED_SDK_IMPLEMENTATION_PLAN.md](#/unified_sdk_implementation_plan.md) for detailed implementation roadmap.

## Gateway SDK Features ⭐ NEW

The new Gateway SDKs provide complete integration with KNIRVGATEWAY services:

### Economics Service
- **Skill Invocation**: Process skill invocations with economic transactions
- **LLM Registration**: Handle LLM registration and fees
- **Validation Rewards**: Process validation rewards for network participants
- **Fee Calculation**: Calculate network fees for transactions
- **Metrics & Analytics**: Economic metrics, burn tracking, and network statistics
- **Transaction Management**: Economic transaction processing and history

### API Gateway
- **Service Routing**: Route requests to appropriate KNIRV components
- **Health Monitoring**: Comprehensive service health checks
- **Status Management**: Service status and configuration management

### Integration Management
- **Component Connectivity**: Test and monitor KNIRV component connections
- **Cross-Service Communication**: Manage communication between services
- **Real-time Monitoring**: Live system health and performance monitoring

### PoAu-D Consensus Management ⭐ NEW
- **Consensus Control**: Enable/disable PoAu-D consensus mechanism
- **Network Authors**: Manage Network Author Peers (NAPs) for block proposal
- **Plugin Author Peers**: Handle PAP registration and delegation
- **Status Monitoring**: Real-time PoAu-D status and delegation statistics
- **Transaction Delegation**: Automatic routing of MCP transactions to capability owners
- **Hybrid Mining**: PoAu-D with PoW fallback for maximum reliability

## Core SDK Features

The existing SDKs provide fundamental KNIRV network functionality:

1. **Transaction Management**: Complete blockchain transaction handling
2. **Network Communication**: Transmission protocols and client-server interactions
3. **URI Resolution**: Parse and resolve `knirv://` URIs
4. **Peer Discovery**: Find peers on the KNIRVCHAIN DHT
5. **Resource Fetching**: Connect to peers and fetch resources
6. **Error Handling**: Handle errors and retries gracefully

## KNIRV URI Structure

```
knirv://<ID>.<ResourceType>/<OptionalSubPath>?param1=value1&param2=value2
```

- **Scheme:** `knirv`
- **Authority:** `<ID>.<ResourceType>` (e.g., `mychain.chain`, `content123.nrn`)
  - `<ID>`: Unique identifier for the chain, content, or node.
  - `<ResourceType>`: Specifies the type of resource (e.g., `chain`, `nrn`).
- **Path:** `/<OptionalSubPath>` (e.g., `/block`, `/content`, defaults to `/` if omitted).
- **Query:** `?param1=value1...` (Standard URL query parameters).

For a detailed specification, please refer to the [KNIRVCHAIN URI Scheme Design Document](#/../docs/URI_Generation_GO.md).

## Quick Start

### KNIRVGATEWAY Integration

#### Go Gateway SDK
```go
import "github.com/cloud-equities/KNIRVGATEWAY/sdk/go/gateway"

// Create client
client := gateway.NewClient()

// Process skill invocation
skillResp, err := client.Economics.Skills.Invoke(ctx, gateway.SkillInvocationRequest{
    UserID:  "user123",
    SkillID: "skill456",
    Amount:  "100000", // 0.1 NRN
})

// Get economic metrics
metrics, err := client.Economics.Metrics.Get(ctx)

// Check health
isHealthy, err := client.Health.Check(ctx)
```

#### TypeScript Gateway SDK
```typescript
import { KNIRVGatewayClient } from '@knirv/gateway-sdk';

// Create client
const client = new KNIRVGatewayClient();

// Process skill invocation
const skillResp = await client.economics.invokeSkill({
  user_id: 'user123',
  skill_id: 'skill456',
  amount: '100000', // 0.1 NRN
});

// Get economic metrics
const metrics = await client.economics.getMetrics();

// Check health
const health = await client.health.checkEconomicsHealth();
```

### PoAu-D Consensus Management

#### Go Example

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/cloud-equities/KNIRVGATEWAY/sdk/go/gateway"
)

func main() {
    // Create PoAu-D client
    client := gateway.NewPoAuDClient()
    ctx := context.Background()

    // Enable PoAu-D consensus
    resp, err := client.EnableConsensus(ctx)
    if err != nil {
        log.Fatalf("Failed to enable PoAu-D: %v", err)
    }
    fmt.Printf("PoAu-D enabled: %s\n", resp.Message)

    // Add Network Author
    addResp, err := client.AddNetworkAuthor(ctx, "knirv1abc123def456")
    if err != nil {
        log.Fatalf("Failed to add network author: %v", err)
    }
    fmt.Printf("Added Network Author: %s\n", addResp.Message)

    // Get status
    status, err := client.GetConsensusStatus(ctx)
    if err != nil {
        log.Fatalf("Failed to get status: %v", err)
    }
    fmt.Printf("PoAu-D Status: enabled=%t, authors=%d\n",
        status.Enabled, status.NetworkAuthorsCount)
}
```

#### TypeScript Example

```typescript
import { PoAuDClient } from '@knirv/gateway-sdk';

async function main() {
    // Create PoAu-D client
    const client = new PoAuDClient();

    // Enable PoAu-D consensus
    const resp = await client.enableConsensus();
    console.log(`PoAu-D enabled: ${resp.message}`);

    // Add Network Author
    const addResp = await client.addNetworkAuthor('knirv1abc123def456');
    console.log(`Added Network Author: ${addResp.message}`);

    // Get status
    const status = await client.getConsensusStatus();
    console.log(`PoAu-D Status: enabled=${status.enabled}, authors=${status.network_authors_count}`);
}

main().catch(console.error);
```

## Legacy Examples

### Go Transaction SDK

```go
package main

import (
    "fmt"
    "log"

    "github.com/cloud-equities/KNIRVCHAIN_GO_ROOT_MCP/sdk/go/client"
)

func main() {
    // Create a client with default configuration
    config := client.DefaultConfig()
    knirvClient, err := client.New(config)
    if err != nil {
        log.Fatalf("Failed to create KNIRV client: %v", err)
    }
    defer knirvClient.Close()
    
    // Bootstrap the client
    if err := knirvClient.Bootstrap(); err != nil {
        log.Fatalf("Failed to bootstrap client: %v", err)
    }
    
    // Fetch a resource
    resource, err := knirvClient.FetchResource("knirv://mychain-alpha.chain/block?number=123")
    if err != nil {
        log.Fatalf("Failed to fetch resource: %v", err)
    }
    
    // Use the resource data
    fmt.Println(resource.String())
}
```

### Python

```python
import asyncio
from knirv_uri_sdk import KnirvClient, KnirvClientError

async def main():
    # Create a client with bootstrap peers
    client = KnirvClient(
        bootstrap_peers=[
            "/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
            # Add your own bootstrap peers here
        ],
        log_enabled=True
    )
    
    try:
        # Bootstrap the client
        await client.bootstrap()
        
        # Fetch a resource
        resource = await client.fetch_resource("knirv://mychain-alpha.chain/block?number=123")
        
        # Use the resource data
        print(str(resource))
    except KnirvClientError as e:
        print(f"Error: {e}")
    finally:
        # Close the client
        await client.close()

# Run the async function
asyncio.run(main())
```

### JavaScript/TypeScript

```typescript
import { KnirvClient, KnirvClientError } from 'knirv-uri-sdk';

async function main() {
    // Create a client with bootstrap peers
    const client = new KnirvClient({
        bootstrapPeers: [
            '/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ',
            // Add your own bootstrap peers here
        ],
        logEnabled: true
    });
    
    try {
        // Bootstrap the client
        await client.bootstrap();
        
        // Fetch a resource
        const resource = await client.fetchResource('knirv://mychain-alpha.chain/block?number=123');
        
        // Use the resource data
        console.log(resource.toString());
    } catch (e) {
        if (e instanceof KnirvClientError) {
            console.error(`Client error: ${e.message}`);
        } else {
            console.error(`Unexpected error: ${e instanceof Error ? e.message : String(e)}`);
        }
    } finally {
        // Close the client
        await client.stop();
    }
}

main().catch(console.error);
```

## License

[MIT License](#/license)