# KNIRVCHAIN SDK for Python

[![Python Versions](https://img.shields.io/badge/python-3.8%2B-blue.svg)](https://www.python.org/downloads/)

The KNIRVCHAIN SDK for Python provides a comprehensive set of tools and libraries for interacting with the KNIRVCHAIN network. This SDK enables developers to build applications that can seamlessly integrate with the KNIRVCHAIN ecosystem, leveraging its distributed and secure infrastructure.

## Overview

KNIRVCHAIN is a distributed ledger technology platform that provides a secure, scalable, and efficient infrastructure for decentralized applications. The Python SDK offers a complete toolkit for developers to interact with the KNIRVCHAIN network, including:

- **Gateway SDK**: For accessing KNIRV Gateway services including Economics, Health, Integration, and PoAuD
- **Transaction SDK**: For creating, signing, and submitting transactions to the KNIRVCHAIN network
- **Transmission SDK**: For resolving `knirv://` URIs, discovering peers, and fetching resources from the network

## Components

### Gateway SDK

The Gateway SDK provides comprehensive access to KNIRV Gateway services. It allows developers to:

- **Economics Service**: Manage skills, LLM interactions, validation, fees, metrics, transactions, burn operations, and rules
- **Gateway Service**: Handle route management, health checks, and status monitoring
- **Health Service**: Perform comprehensive health monitoring across all services
- **Integration Service**: Manage external service integrations and webhooks
- **PoAuD Service**: Create and verify Proof of Audit records with cryptographic verification

The Gateway SDK features async/await support, comprehensive error handling, type safety with Pydantic models, and environment-based configuration.

### Transaction SDK

The Transaction SDK provides a high-level interface for interacting with the KNIRVCHAIN transaction API. It allows developers to:

- Create and manage transactions on the KNIRVCHAIN network
- Interact with chain data and capabilities
- Handle transaction signing and verification
- Manage API authentication and error handling

The SDK includes both synchronous and asynchronous clients, comprehensive error handling, and full type definitions for all requests and responses.

### Transmission SDK

The Transmission SDK simplifies the process of interacting with the KNIRVCHAIN network's peer-to-peer infrastructure. Key features include:

- Parsing `knirv://` URIs into their components (ID, ResourceType, Path, Query)
- Discovering peers on the KNIRVCHAIN DHT that provide specific resources
- Connecting to peers and fetching resources using libp2p streams
- Handling errors and retries gracefully
- Async/await support for efficient network operations

## Installation

### Transaction SDK

```bash
pip install knirvchain_transaction_sdk
```

### Transmission SDK

```bash
pip install knirv-transmission-sdk
```

## Quick Start

### Using the Transaction SDK

```python
import os
from knirvchain_transaction_sdk import KnirvchainTransactionSDK

# Initialize the client
client = KnirvchainTransactionSDK(
    api_key=os.environ.get("KNIRVCHAIN_TRANSACTION_SDK_API_KEY"),
)

# Retrieve chain information
chain = client.chain.retrieve()
print(chain.chain_id)

# For async usage
from knirvchain_transaction_sdk import AsyncKnirvchainTransactionSDK
import asyncio

async_client = AsyncKnirvchainTransactionSDK(
    api_key=os.environ.get("KNIRVCHAIN_TRANSACTION_SDK_API_KEY"),
)

async def main():
    chain = await async_client.chain.retrieve()
    print(chain.chain_id)

asyncio.run(main())
```

### Using the Transmission SDK

```python
import asyncio
from knirv_uri_sdk import KnirvClient, parse_knirv_uri

# Parse a KNIRV URI
uri_string = "knirv://mychain-alpha.chain/block?number=123"
parsed_uri = parse_knirv_uri(uri_string)
print(f"ID: {parsed_uri.id}")
print(f"ResourceType: {parsed_uri.resource_type}")
print(f"Path: {parsed_uri.path}")

# Fetch a resource from the network
async def fetch_resource():
    client = KnirvClient(
        bootstrap_peers=[
            "/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
            # Add your own bootstrap peers here
        ],
        log_enabled=True
    )
    
    try:
        await client.bootstrap()
        resource = await client.fetch_resource(uri_string)
        print(str(resource))
        return resource
    finally:
        await client.close()

asyncio.run(fetch_resource())
```

## Documentation

For detailed documentation on each component of the SDK:

- **Transaction SDK**: See the [Transaction SDK README](transaction/README.md) and [API documentation](transaction/api.md)
- **Transmission SDK**: See the [Transmission SDK README](transmission/README.md) and examples in the `examples` directory

## Error Handling

Both SDKs provide comprehensive error handling with specific error types to help with debugging and graceful error recovery. See the component-specific documentation for details on error types and handling strategies.

## Requirements

- Python 3.8 or higher
- Internet connection for network operations
- API key for Transaction SDK (obtain from KNIRVCHAIN)

## Contributing

See the [contributing documentation](./CONTRIBUTING.md) for guidelines on how to contribute to the KNIRVCHAIN SDK.

## License

[MIT License](LICENSE)