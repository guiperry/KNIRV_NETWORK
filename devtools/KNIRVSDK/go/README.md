# KNIRVCHAIN SDK for Go

The KNIRVCHAIN SDK for Go provides a comprehensive set of tools for interacting with the KNIRVCHAIN network. This SDK consists of two main components: the Transaction SDK and the Transmission SDK, each designed to handle different aspects of KNIRVCHAIN functionality.

## Overview

KNIRVCHAIN is a decentralized network designed for secure, efficient, and reliable blockchain operations. This SDK enables developers to:

- Create, sign, and submit transactions to the KNIRVCHAIN network
- Resolve `knirv://` URIs and discover resources on the network
- Connect to peers on the private DHT (Distributed Hash Table)
- Fetch blockchain resources like blocks, transactions, and chain information
- Interact with the KNIRVCHAIN REST API

## Components

### Transaction SDK

The Transaction SDK provides convenient access to the KNIRVCHAIN Transaction REST API from applications written in Go. It allows you to:

- Query chain information
- Manage blocks and transactions
- Interact with transaction pools
- Generate URIs for KNIRVCHAIN resources
- Monitor network health

[Learn more about the Transaction SDK](./transaction/README.md)

### Transmission SDK

The Transmission SDK offers a high-level interface for interacting with the KNIRVCHAIN network's peer-to-peer layer. It simplifies:

- Parsing `knirv://` URIs into their components
- Discovering peers on the KNIRVCHAIN DHT
- Connecting to peers and fetching resources using libp2p streams
- Handling errors and retries gracefully

[Learn more about the Transmission SDK](./transmission/README.md)

## Installation

### Transaction SDK

```go
import (
    "github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction" // imported as transaction
)
```

Or to pin the version:

```sh
go get -u 'github.com/cloud-equities/KNIRVCHAIN/sdk/go@v0.0.1-alpha.0'
```

### Transmission SDK

```bash
go get github.com/cloud-equities/KNIRVCHAIN/sdk/go/transmission
```

## Quick Start

### Using the Transaction SDK

```go
package main

import (
    "context"
    "fmt"

    "github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction"
    "github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction/option"
)

func main() {
    client := knirvchaintransactionsdk.NewClient(
        option.WithAPIKey("My API Key"), // defaults to os.LookupEnv("KNIRVCHAIN_TRANSACTION_SDK_API_KEY")
    )
    chain, err := client.Chain.Get(context.TODO())
    if err != nil {
        panic(err.Error())
    }
    fmt.Printf("%+v\n", chain.ChainID)
}
```

### Using the Transmission SDK

```go
package main

import (
    "fmt"
    "log"

    "github.com/cloud-equities/KNIRVCHAIN/sdk/go/transmission/client"
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
}
```

## Integration Example

The following example demonstrates how to use both SDKs together to fetch transaction information and then resolve related resources:

```go
package main

import (
    "context"
    "fmt"
    "log"

    txsdk "github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction"
    "github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction/option"
    
    "github.com/cloud-equities/KNIRVCHAIN/sdk/go/transmission/client"
)

func main() {
    // Initialize Transaction SDK client
    txClient := txsdk.NewClient(
        option.WithAPIKey("My API Key"),
    )
    
    // Get transaction information
    txInfo, err := txClient.Transaction.Get(context.TODO(), "tx_123456")
    if err != nil {
        log.Fatalf("Failed to get transaction: %v", err)
    }
    
    // Generate URI for related block
    blockURI := fmt.Sprintf("knirv://%s.chain/block?number=%d", txInfo.ChainID, txInfo.BlockNumber)
    
    // Initialize Transmission SDK client
    transmissionConfig := client.DefaultConfig()
    transmissionClient, err := client.New(transmissionConfig)
    if err != nil {
        log.Fatalf("Failed to create transmission client: %v", err)
    }
    defer transmissionClient.Close()
    
    // Bootstrap the client
    if err := transmissionClient.Bootstrap(); err != nil {
        log.Fatalf("Failed to bootstrap client: %v", err)
    }
    
    // Fetch the block resource using the URI
    blockResource, err := transmissionClient.FetchResource(blockURI)
    if err != nil {
        log.Fatalf("Failed to fetch block resource: %v", err)
    }
    
    fmt.Printf("Transaction: %+v\n", txInfo)
    fmt.Printf("Related Block: %s\n", blockResource.String())
}
```

## Requirements

- Go 1.18 or later
- For the Transaction SDK: Internet access to the KNIRVCHAIN API endpoints
- For the Transmission SDK: Network access to connect to the KNIRVCHAIN DHT

## Documentation

- [Transaction SDK API Documentation](./transaction/api.md)
- [Transmission SDK Examples](./transmission/example)

## Error Handling

Both SDKs provide specific error types to help with error handling. See the individual README files for details on error handling for each component.

## Contributing

See the contributing documentation for each component:
- [Transaction SDK Contributing Guide](./transaction/CONTRIBUTING.md)
- For Transmission SDK contributions, please open an issue or pull request in the repository

## License

The KNIRVCHAIN SDK for Go is available under the MIT License. See the LICENSE files in each component directory for details.