# KNIRV Client Transmission SDK for Python

The KNIRV Client Transmission SDK for Python provides a high-level interface for interacting with the KNIRVCHAIN network. It simplifies the process of resolving `knirv://` URIs, discovering peers on the private DHT, connecting to them, and fetching the underlying resources.

## Installation

```bash
pip install knirv-transmission-sdk
```

## Features

- Parse `knirv://` URIs into their components (ID, ResourceType, Path, Query)
- Discover peers on the KNIRVCHAIN DHT that provide specific resources
- Connect to peers and fetch resources using libp2p streams
- Handle errors and retries gracefully
- Async/await support for efficient network operations

## Usage

### Parsing URIs

```python
from knirv_uri_sdk import parse_knirv_uri, KnirvURIError

try:
    uri_string = "knirv://mychain-alpha.chain/block?number=123&validator=node1"
    parsed_uri = parse_knirv_uri(uri_string)
    
    print(f"ID: {parsed_uri.id}")                 # "mychain-alpha"
    print(f"ResourceType: {parsed_uri.resource_type}") # "chain"
    print(f"Path: {parsed_uri.path}")             # "/block"
    print(f"Query parameter 'number': {parsed_uri.get_query_param('number')}") # "123"
except KnirvURIError as e:
    print(f"Error parsing URI: {e}")
```

### Fetching Resources

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
        # Bootstrap the client (connect to DHT network)
        await client.bootstrap()
        
        # Fetch a resource
        resource = await client.fetch_resource("knirv://mychain-alpha.chain/block?number=123")
        
        # Use the resource data
        print(str(resource))
        
        # Or get the raw bytes
        data = resource.bytes()
        # Process data...
    except KnirvClientError as e:
        print(f"Error: {e}")
    finally:
        # Close the client
        await client.close()

# Run the async function
asyncio.run(main())
```

### Custom Configuration

```python
client = KnirvClient(
    bootstrap_peers=[
        "/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
        # Add your own bootstrap peers here
    ],
    p2p_port=6000,  # Specify a port for the libp2p host
    log_enabled=True  # Enable logging
)
```

## Error Handling

The SDK provides specific error types to help with error handling:

- `KnirvURIError`: Base exception for URI parsing errors
- `KnirvClientError`: Base exception for client errors
- `ResourceNotFoundError`: Raised when a resource cannot be found
- `ConnectionFailedError`: Raised when connection to a peer fails
- `FetchFailedError`: Raised when fetching a resource fails
- `ClientClosedError`: Raised when operations are attempted on a closed client

Example:

```python
from knirv_uri_sdk import (
    KnirvClient, KnirvClientError, ResourceNotFoundError,
    ConnectionFailedError, FetchFailedError
)

async def fetch_with_error_handling(uri_string):
    client = KnirvClient(bootstrap_peers=[...])
    try:
        await client.bootstrap()
        return await client.fetch_resource(uri_string)
    except ResourceNotFoundError:
        print(f"Resource not found: {uri_string}")
    except ConnectionFailedError as e:
        print(f"Failed to connect to peer: {e}")
    except FetchFailedError as e:
        print(f"Failed to fetch resource: {e}")
    except KnirvClientError as e:
        print(f"Unexpected client error: {e}")
    finally:
        await client.close()
```

## Example

See the `examples` directory for complete examples of using the SDK.

## License

[MIT License](LICENSE)