# KNIRV Client Transmission SDK

## Table of Contents

- [Overview](#overview)
- [Installation](#installation)
- [Features](#features)
- [Usage](#usage)
    - [Parsing URIs](#parsing-uris)
    - [Fetching Resources](#fetching-resources)
    - [Custom Configuration](#custom-configuration)
- [Error Handling](#error-handling)
- [Testing](#testing)
    - [Testing the Go SDK](#testing-the-go-sdk)
    - [Testing the Python SDK](#testing-the-python-sdk)
    - [Testing the JavaScript/TypeScript SDK](#testing-the-javascripttypescript-sdk)
    - [Testing with Real KNIRVCHAIN Network](#testing-with-real-knirvchain-network)
    - [Example Configuration](#example-configuration)
    - [Troubleshooting](#troubleshooting)
    - [Debugging](#debugging)
- [Example](#example)
- [License](#license)


## Overview

The KNIRV Client Transmission SDK provides a high-level interface for interacting with the KNIRVCHAIN network.  It simplifies resolving `knirv://` URIs, discovering peers on the private DHT, connecting to them, and fetching underlying resources.  This SDK is available in Go, Python, and JavaScript/TypeScript.


## Installation

**Go:**

```bash
go get github.com/cloud-equities/KNIRVCHAIN_CONNECTION_SDK/go
```

**Python:**

1. Navigate to the Python SDK directory: `cd /home/gperry/Documents/GitHub/cloud-equities/KNIRVCHAIN_GO_ROOT_MCP/sdk/python`
2. Create a virtual environment: `python -m venv venv`
3. Activate the virtual environment: `source venv/bin/activate` (On Windows: `venv\Scripts\activate`)
4. Install the SDK: `pip install -e .`

**JavaScript/TypeScript:**

1. Navigate to the JavaScript SDK directory: `cd /home/gperry/Documents/GitHub/cloud-equities/KNIRVCHAIN_GO_ROOT_MCP/sdk/js`
2. Install dependencies: `npm install` or `yarn install`
3. Build the SDK: `npm run build` or `yarn build`


## Features

- Parse `knirv://` URIs into components (ID, ResourceType, Path, Query)
- Discover peers on the KNIRVCHAIN DHT providing specific resources
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

The SDK provides specific error types:

- `parser.ErrInvalidURI`: Invalid URI.
- `client.ErrClientClosed`: Operations on a closed client.
- `client.ErrResourceNotFound`: Resource not found.
- `client.ErrConnectionFailed`: Connection to a peer failed.
- `client.ErrFetchFailed`: Fetching a resource failed.

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

## Testing

### Testing the Go SDK

**Setup:**

1. `cd /home/gperry/Documents/GitHub/cloud-equities/KNIRVCHAIN_GO_ROOT_MCP/sdk/go`
2. `go mod tidy`

**Running Tests:**

1. `cd parser`
2. `go test -v`
3. `cd ../client`
4. `go test -v`

**Running the Example:**

1. `cd ../example`
2. `go build -o knirv-example`
3. `./knirv-example knirv://mychain-alpha.chain/block?number=123`

### Testing the Python SDK

**Setup:**

1. `cd /home/gperry/Documents/GitHub/cloud-equities/KNIRVCHAIN_GO_ROOT_MCP/sdk/python`
2. `python -m venv venv`
3. `source venv/bin/activate`  (On Windows: `venv\Scripts\activate`)
4. `pip install -e .`
5. `pip install pytest`

**Running Tests:**

`pytest`

**Running the Example:**

`python examples/fetch_resource.py knirv://mychain-alpha.chain/block?number=123`

### Testing the JavaScript/TypeScript SDK

**Setup:**

1. `cd /home/gperry/Documents/GitHub/cloud-equities/KNIRVCHAIN_GO_ROOT_MCP/sdk/js`
2. `npm install` or `yarn install`
3. `npm run build` or `yarn build`

**Running Tests:**

`npm test` or `yarn test`

**Running the Example:**

`npx ts-node examples/fetch-resource.ts knirv://mychain-alpha.chain/block?number=123`

### Testing with Real KNIRVCHAIN Network

Replace default bootstrap peers with actual KNIRVCHAIN network peers and use valid `knirv://` URIs.

### Example Configuration

**Go:**

```go
config := client.ClientConfig{
    BootstrapPeers: []string{
        "/ip4/your-node-ip/tcp/port/p2p/peer-id",
        // Add more peers here
    },
    LogEnabled: true,
}
```

**Python:**

```python
client = KnirvClient(
    bootstrap_peers=[
        "/ip4/your-node-ip/tcp/port/p2p/peer-id",
        # Add more peers here
    ],
    log_enabled=True
)
```

**JavaScript/TypeScript:**

```javascript
const client = new KnirvClient({
    bootstrapPeers: [
        '/ip4/your-node-ip/tcp/port/p2p/peer-id',
        // Add more peers here
    ],
    logEnabled: true
});
```

### Troubleshooting

**Common Issues:**

- **Connection Failures:** Check network connectivity, firewall settings, and peer multiaddresses.
- **Resource Not Found:** Verify URI format, resource existence, and provider status.
- **Build Errors:**  (Go: `go mod tidy`; Python: check virtual environment; JavaScript: verify npm/yarn dependencies).

### Debugging

Enable `logEnabled` for detailed logging (see Example Configuration above).


## Example

See the `example` directory for a complete example (Go SDK).


## License

[MIT License](LICENSE)
