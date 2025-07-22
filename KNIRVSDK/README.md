# KNIRVCHAIN Client SDKs

This directory contains client SDKs for interacting with the KNIRVCHAIN network in various programming languages. These SDKs provide a high-level interface for resolving `knirv://` URIs, discovering peers on the private DHT, connecting to them, and fetching the underlying resources.

## Available SDKs

- [Go SDK](go/README.md) - For Go applications
- [Python SDK](python/README.md) - For Python applications
- [JavaScript/TypeScript SDK](js/README.md) - For Node.js and browser applications

## Common Features

All SDKs provide the following core functionality:

1. **URI Parsing**: Parse `knirv://` URIs into their components (ID, ResourceType, Path, Query)
2. **Peer Discovery**: Find peers on the KNIRVCHAIN DHT that provide specific resources
3. **Resource Fetching**: Connect to peers and fetch resources using libp2p streams
4. **Error Handling**: Handle errors and retries gracefully

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

For a detailed specification, please refer to the [KNIRVCHAIN URI Scheme Design Document](../docs/URI_Generation_GO.md).

## Usage Examples

### Go

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

[MIT License](LICENSE)