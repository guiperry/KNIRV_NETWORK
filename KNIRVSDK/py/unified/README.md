# KNIRV SDK for Python

The unified Python SDK for KNIRV Network services, providing access to Gateway, Transaction, and Transmission services through a single, cohesive interface.

## Features

- **Unified Interface**: Single client for all KNIRV services
- **Type Safety**: Full type hints and runtime validation
- **Async/Await Support**: Built for modern async Python applications
- **Automatic Retries**: Configurable retry logic with exponential backoff
- **Environment Configuration**: Easy configuration via environment variables
- **Comprehensive Error Handling**: Detailed exception hierarchy

## Installation

```bash
pip install knirv-sdk
```

## Quick Start

```python
import asyncio
from knirv_sdk import KNIRVClient

async def main():
    # Create client with default configuration
    async with KNIRVClient() as client:
        # Use Gateway services
        skills = await client.gateway.economics.skills.list()
        print(f"Found {len(skills['data'])} skills")
        
        # Use Transaction services
        chain_info = await client.transaction.get_chain_info()
        print(f"Chain height: {chain_info['height']}")
        
        # Use Transmission services
        uri = client.transmission.parse_uri("knirv://skill123/execute")
        print(f"Parsed URI: {uri.id} -> {uri.resource_type}")

if __name__ == "__main__":
    asyncio.run(main())
```

## Configuration

### Environment Variables

```bash
# Global settings
export KNIRV_API_KEY="your-api-key"
export KNIRV_ENVIRONMENT="production"  # or "staging", "development"

# Service-specific URLs (optional)
export KNIRV_GATEWAY_URL="https://gateway.knirv.com"
export KNIRV_TRANSACTION_URL="https://transaction.knirv.com"
export KNIRV_TRANSMISSION_URL="https://transmission.knirv.com"
```

### Programmatic Configuration

```python
from knirv_sdk import KNIRVClient, ClientConfig

# Custom configuration
config = ClientConfig(
    api_key="your-api-key",
    environment="development",
    timeout=60,
    retries=5,
    debug=True
)

client = KNIRVClient(config)
```

## Services

### Gateway Service

Access to economics, health, integration, and PoAuD services:

```python
# Economics - Skills
skills = await client.gateway.economics.skills.list()
skill = await client.gateway.economics.skills.get("skill-id")
new_skill = await client.gateway.economics.skills.create({
    "name": "My Skill",
    "description": "A test skill",
    "cost": 100
})

# Economics - LLM
cost_estimate = await client.gateway.economics.llm.estimate_cost({
    "text": "Hello world",
    "model": "gpt-4"
})

# Health monitoring
health = await client.gateway.health.check()
```

### Transaction Service

Blockchain transaction management:

```python
# Create and submit transactions
tx = await client.transaction.create_transaction({
    "from": "address1",
    "to": "address2",
    "amount": 100
})

result = await client.transaction.submit_transaction(tx["id"])

# Get blockchain info
chain_info = await client.transaction.get_chain_info()
latest_block = await client.transaction.get_latest_block()
```

### Transmission Service

URI parsing and data transmission:

```python
# Parse KNIRV URIs
uri = client.transmission.parse_uri("knirv://skill123/execute?param=value")
print(f"ID: {uri.id}, Type: {uri.resource_type}")

# Generate new URIs
new_uri = await client.transmission.generate_uri({
    "resource_type": "skill",
    "id": "skill123",
    "action": "execute"
})

# Transmit data
result = await client.transmission.transmit_data(
    data={"message": "hello"},
    destination="knirv://target123"
)
```

## Error Handling

The SDK provides a comprehensive exception hierarchy:

```python
from knirv_sdk import (
    KNIRVSDKError,
    KNIRVAPIError,
    KNIRVValidationError,
    KNIRVConnectionError,
    KNIRVTimeoutError,
    KNIRVAuthenticationError,
    KNIRVNotFoundError,
)

try:
    skill = await client.gateway.economics.skills.get("invalid-id")
except KNIRVNotFoundError:
    print("Skill not found")
except KNIRVAuthenticationError:
    print("Authentication failed")
except KNIRVAPIError as e:
    print(f"API error: {e.message} (status: {e.status_code})")
```

## Development

### For Development Environment

```python
from knirv_sdk import create_development_client

# Automatically configured for development
client = create_development_client()
```

### For Testing

```python
from knirv_sdk import create_testing_client

# Configured with shorter timeouts and fewer retries
client = create_testing_client()
```

## License

MIT License - see LICENSE file for details.

## Support

- Documentation: https://docs.knirv.com/sdk/python
- Issues: https://github.com/knirv-network/knirv-sdk/issues
- Community: https://discord.gg/knirv
