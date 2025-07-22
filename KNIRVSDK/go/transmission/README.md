# KNIRV Client Transmission SDK for Go

The KNIRV Client Transmission SDK for Go provides a high-level interface for interacting with the KNIRVCHAIN network. It simplifies the process of resolving `knirv://` URIs, discovering peers on the private DHT, connecting to them, and fetching the underlying resources.

## Installation

```bash
go get github.com/cloud-equities/KNIRVCHAIN_CONNECTION_SDK/go
```

## Features

- Parse `knirv://` URIs into their components (ID, ResourceType, Path, Query)
- Discover peers on the KNIRVCHAIN DHT that provide specific resources
- Connect to peers and fetch resources using libp2p streams
- Handle errors and retries gracefully

## Usage

### Parsing URIs

```go
import "github.com/cloud-equities/KNIRVCHAIN/sdk/go/transmission/parser"

func main() {
    uriString := "knirv://mychain-alpha.chain/block?number=123&validator=node1"
    
    parsedURI, err := parser.Parse(uriString)
    if err != nil {
        log.Fatalf("Failed to parse URI: %v", err)
    }
    
    fmt.Printf("ID: %s\n", parsedURI.ID)                 // "mychain-alpha"
    fmt.Printf("ResourceType: %s\n", parsedURI.ResourceType) // "chain"
    fmt.Printf("Path: %s\n", parsedURI.Path)             // "/block"
    fmt.Printf("Query parameter 'number': %s\n", parsedURI.GetQueryParam("number")) // "123"
}
```

### Fetching Resources

```go
import (
    "github.com/cloud-equities/KNIRVCHAIN/sdk/go/transmission/client"
    "log"
)

func main() {
    // Create a client with default configuration
    config := client.DefaultConfig()
    knirvClient, err := client.New(config)
    if err != nil {
        log.Fatalf("Failed to create KNIRV client: %v", err)
    }
    defer knirvClient.Close()
    
    // Bootstrap the client (connect to DHT network)
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
    
    // Or get the raw bytes
    data := resource.Bytes()
    // Process data...
}
```

### Custom Configuration

```go
config := client.ClientConfig{
    BootstrapPeers: []string{
        "/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
        // Add your own bootstrap peers here
    },
    P2PPort:    6000,  // Specify a port for the libp2p host
    LogEnabled: true,  // Enable logging
}

knirvClient, err := client.New(config)
```

## Error Handling

The SDK provides specific error types to help with error handling:

- `parser.ErrInvalidURI`: Returned when a URI is invalid
- `client.ErrClientClosed`: Returned when operations are attempted on a closed client
- `client.ErrResourceNotFound`: Returned when a resource cannot be found
- `client.ErrConnectionFailed`: Returned when connection to a peer fails
- `client.ErrFetchFailed`: Returned when fetching a resource fails

Example:

```go
resource, err := knirvClient.FetchResource(uriString)
if err != nil {
    switch {
    case errors.Is(err, client.ErrResourceNotFound):
        log.Printf("Resource not found: %s", uriString)
    case errors.Is(err, client.ErrConnectionFailed):
        log.Printf("Failed to connect to peer: %v", err)
    default:
        log.Printf("Unexpected error: %v", err)
    }
    return
}
```

## Example

See the `example` directory for a complete example of using the SDK.

## License

[MIT License](LICENSE)