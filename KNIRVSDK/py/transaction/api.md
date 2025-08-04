# Chain

Types:

```python
from knirvchain_transaction_sdk.types import ChainRetrieveResponse
```

Methods:

- <code title="get /chain">client.chain.<a href="./src/knirvchain_transaction_sdk/resources/chain.py">retrieve</a>() -> <a href="./src/knirvchain_transaction_sdk/types/chain_retrieve_response.py">ChainRetrieveResponse</a></code>

# Block

Types:

```python
from knirvchain_transaction_sdk.types import Block, BlockSubmitResponse
```

Methods:

- <code title="post /block">client.block.<a href="./src/knirvchain_transaction_sdk/resources/block.py">submit</a>(\*\*<a href="src/knirvchain_transaction_sdk/types/block_submit_params.py">params</a>) -> <a href="./src/knirvchain_transaction_sdk/types/block_submit_response.py">BlockSubmitResponse</a></code>

# Transaction

Types:

```python
from knirvchain_transaction_sdk.types import Transaction, TransactionSubmitResponse
```

Methods:

- <code title="post /transaction">client.transaction.<a href="./src/knirvchain_transaction_sdk/resources/transaction.py">submit</a>(\*\*<a href="src/knirvchain_transaction_sdk/types/transaction_submit_params.py">params</a>) -> <a href="./src/knirvchain_transaction_sdk/types/transaction_submit_response.py">TransactionSubmitResponse</a></code>

# TxnPool

Types:

```python
from knirvchain_transaction_sdk.types import TxnPoolRetrieveResponse
```

Methods:

- <code title="get /txn_pool">client.txn_pool.<a href="./src/knirvchain_transaction_sdk/resources/txn_pool.py">retrieve</a>() -> <a href="./src/knirvchain_transaction_sdk/types/txn_pool_retrieve_response.py">TxnPoolRetrieveResponse</a></code>

# Info

Types:

```python
from knirvchain_transaction_sdk.types import InfoRetrieveResponse
```

Methods:

- <code title="get /info">client.info.<a href="./src/knirvchain_transaction_sdk/resources/info.py">retrieve</a>() -> <a href="./src/knirvchain_transaction_sdk/types/info_retrieve_response.py">InfoRetrieveResponse</a></code>

# Health

Types:

```python
from knirvchain_transaction_sdk.types import HealthCheckResponse
```

Methods:

- <code title="get /health">client.health.<a href="./src/knirvchain_transaction_sdk/resources/health.py">check</a>() -> <a href="./src/knirvchain_transaction_sdk/types/health_check_response.py">HealthCheckResponse</a></code>

# Ping

Types:

```python
from knirvchain_transaction_sdk.types import PingCheckResponse
```

Methods:

- <code title="get /ping">client.ping.<a href="./src/knirvchain_transaction_sdk/resources/ping.py">check</a>() -> str</code>

# Peers

Types:

```python
from knirvchain_transaction_sdk.types import PeerListResponse
```

Methods:

- <code title="get /peers">client.peers.<a href="./src/knirvchain_transaction_sdk/resources/peers.py">list</a>() -> <a href="./src/knirvchain_transaction_sdk/types/peer_list_response.py">PeerListResponse</a></code>

# UriGenerator

Types:

```python
from knirvchain_transaction_sdk.types import UriGeneratorCreateResponse
```

Methods:

- <code title="post /uriGenerator">client.uri_generator.<a href="./src/knirvchain_transaction_sdk/resources/uri_generator.py">create</a>(\*\*<a href="src/knirvchain_transaction_sdk/types/uri_generator_create_params.py">params</a>) -> <a href="./src/knirvchain_transaction_sdk/types/uri_generator_create_response.py">UriGeneratorCreateResponse</a></code>

# Mcp

Types:

```python
from knirvchain_transaction_sdk.types import (
    ContextRecord,
    McpRetrieveCapabilitiesResponse,
    McpRetrieveContextsResponse,
)
```

Methods:

- <code title="get /mcp/context/{context_id}">client.mcp.<a href="./src/knirvchain_transaction_sdk/resources/mcp/mcp.py">retrieve</a>(context_id) -> <a href="./src/knirvchain_transaction_sdk/types/context_record.py">ContextRecord</a></code>
- <code title="get /mcp/capabilities">client.mcp.<a href="./src/knirvchain_transaction_sdk/resources/mcp/mcp.py">retrieve_capabilities</a>(\*\*<a href="src/knirvchain_transaction_sdk/types/mcp_retrieve_capabilities_params.py">params</a>) -> <a href="./src/knirvchain_transaction_sdk/types/mcp_retrieve_capabilities_response.py">McpRetrieveCapabilitiesResponse</a></code>
- <code title="get /mcp/contexts">client.mcp.<a href="./src/knirvchain_transaction_sdk/resources/mcp/mcp.py">retrieve_contexts</a>(\*\*<a href="src/knirvchain_transaction_sdk/types/mcp_retrieve_contexts_params.py">params</a>) -> <a href="./src/knirvchain_transaction_sdk/types/mcp_retrieve_contexts_response.py">McpRetrieveContextsResponse</a></code>

## Capability

Types:

```python
from knirvchain_transaction_sdk.types.mcp import (
    BaseDescriptor,
    CapabilityDescriptor,
    CapabilityUpdateResponse,
    CapabilityInvokeResponse,
    CapabilityPrepareRegistrationResponse,
    CapabilityRetrieveInvocationsResponse,
)
```

Methods:

- <code title="get /mcp/capability/{capability_id}">client.mcp.capability.<a href="./src/knirvchain_transaction_sdk/resources/mcp/capability.py">retrieve</a>(capability_id) -> <a href="./src/knirvchain_transaction_sdk/types/mcp/capability_descriptor.py">CapabilityDescriptor</a></code>
- <code title="post /mcp/capability/update">client.mcp.capability.<a href="./src/knirvchain_transaction_sdk/resources/mcp/capability.py">update</a>(\*\*<a href="src/knirvchain_transaction_sdk/types/mcp/capability_update_params.py">params</a>) -> <a href="./src/knirvchain_transaction_sdk/types/mcp/capability_update_response.py">CapabilityUpdateResponse</a></code>
- <code title="post /mcp/capability/invoke">client.mcp.capability.<a href="./src/knirvchain_transaction_sdk/resources/mcp/capability.py">invoke</a>(\*\*<a href="src/knirvchain_transaction_sdk/types/mcp/capability_invoke_params.py">params</a>) -> <a href="./src/knirvchain_transaction_sdk/types/mcp/capability_invoke_response.py">CapabilityInvokeResponse</a></code>
- <code title="post /mcp/capability/prepare_registration">client.mcp.capability.<a href="./src/knirvchain_transaction_sdk/resources/mcp/capability.py">prepare_registration</a>(\*\*<a href="src/knirvchain_transaction_sdk/types/mcp/capability_prepare_registration_params.py">params</a>) -> <a href="./src/knirvchain_transaction_sdk/types/mcp/capability_prepare_registration_response.py">CapabilityPrepareRegistrationResponse</a></code>
- <code title="get /mcp/capability/{capability_id}/invocations">client.mcp.capability.<a href="./src/knirvchain_transaction_sdk/resources/mcp/capability.py">retrieve_invocations</a>(capability_id) -> <a href="./src/knirvchain_transaction_sdk/types/mcp/capability_retrieve_invocations_response.py">CapabilityRetrieveInvocationsResponse</a></code>
