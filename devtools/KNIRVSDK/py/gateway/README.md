# KNIRV Gateway SDK for Python

The official Python library for the KNIRV Gateway API, providing access to Economics, Gateway, Health, Integration, and PoAuD services.

## Installation

```bash
pip install knirv-gateway-sdk
```

## Quick Start

```python
import asyncio
from knirv_gateway_sdk import KNIRVGateway

async def main():
    # Initialize the client
    client = KNIRVGateway(
        base_url="https://gateway.knirv.com",
        api_key="your-api-key"
    )
    
    # Use the Economics service
    skills = await client.economics.skills.list()
    print(f"Found {len(skills)} skills")
    
    # Use the Health service
    health = await client.health.check()
    print(f"Service status: {health.status}")
    
    # Close the client
    await client.close()

# Run the example
asyncio.run(main())
```

## Services

### Economics Service

The Economics service provides access to skills, LLM, validation, fees, metrics, transactions, burn, and rules management.

```python
# List skills
skills = await client.economics.skills.list(category="ai")

# Invoke a skill
invocation = await client.economics.skills.invoke(
    skill_id="skill-123",
    user_id="user-456",
    amount="100"
)

# Generate text with LLM
from knirv_gateway_sdk.types import LLMRequest

llm_request = LLMRequest(
    prompt="Hello, world!",
    model="gpt-3.5-turbo",
    max_tokens=100
)
response = await client.economics.llm.generate(llm_request)
print(response.response)

# Validate data
from knirv_gateway_sdk.types import ValidationRequest

validation_request = ValidationRequest(
    data={"field": "value"},
    rules=["required", "string"]
)
result = await client.economics.validation.validate(validation_request)
print(f"Valid: {result.valid}")
```

### Gateway Service

The Gateway service provides route management and status monitoring.

```python
# List routes
routes = await client.gateway.routes.list()

# Create a new route
route = await client.gateway.routes.create(
    path="/api/v1/test",
    method="GET",
    handler="test_handler",
    enabled=True
)

# Check gateway health
health = await client.gateway.health.check()
print(f"Gateway status: {health.status}")
```

### Health Service

The Health service provides comprehensive health monitoring.

```python
# Check overall health
health = await client.health.check()

# Check specific service health
economics_health = await client.health.check("economics")

# Check all services
all_health = await client.health.check_all()

# Simple ping
ping_result = await client.health.ping()
```

### Integration Service

The Integration service manages external service integrations.

```python
# List integrations
integrations = await client.integration.list()

# Create a new integration
integration = await client.integration.create(
    name="External API",
    integration_type="webhook",
    config={
        "url": "https://api.example.com/webhook",
        "secret": "webhook-secret"
    }
)

# Test an integration
test_result = await client.integration.test(integration.id)
```

### PoAuD Service

The PoAuD (Proof of Audit) service provides audit trail and verification capabilities.

```python
# Create an audit record
audit = await client.poaud.create_audit(
    entity_id="entity-123",
    entity_type="transaction",
    action="create",
    details={"amount": "100", "recipient": "user-456"},
    auditor="system"
)

# Create proof of audit
proof = await client.poaud.create_proof(
    audit_id=audit.id,
    validator="validator-node-1"
)

# Verify proof
verification = await client.poaud.verify_proof(
    audit_id=audit.id,
    proof_hash=proof.proof_hash
)
```

## Configuration

### Environment Variables

You can configure the client using environment variables:

```bash
export KNIRV_GATEWAY_BASE_URL="https://gateway.knirv.com"
export KNIRV_GATEWAY_API_KEY="your-api-key"
```

### Client Options

```python
client = KNIRVGateway(
    base_url="https://gateway.knirv.com",
    api_key="your-api-key",
    timeout=60.0,
    max_retries=3,
    debug=True
)
```

### Service-Specific Clients

You can create clients optimized for specific services:

```python
# Economics service client
economics_client = KNIRVGateway.create_economics_client(
    base_url="https://economics.knirv.com",
    api_key="your-api-key"
)

# Gateway service client
gateway_client = KNIRVGateway.create_gateway_client(
    base_url="https://gateway.knirv.com",
    api_key="your-api-key"
)
```

## Error Handling

The SDK provides comprehensive error handling:

```python
from knirv_gateway_sdk import (
    KNIRVGatewayError,
    APIError,
    AuthenticationError,
    NotFoundError,
    RateLimitError
)

try:
    skills = await client.economics.skills.list()
except AuthenticationError:
    print("Invalid API key")
except NotFoundError:
    print("Resource not found")
except RateLimitError:
    print("Rate limit exceeded")
except APIError as e:
    print(f"API error: {e}")
except KNIRVGatewayError as e:
    print(f"SDK error: {e}")
```

## Development

### Requirements

- Python 3.8+
- httpx
- pydantic
- typing-extensions

### Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

This project is licensed under the Apache License 2.0 - see the LICENSE file for details.

## Support

For support, please contact support@knirv.com or visit our documentation at https://docs.knirv.com.
