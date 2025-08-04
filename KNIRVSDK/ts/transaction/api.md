# Chain

Types:

- <code><a href="./src/resources/chain.ts">ChainRetrieveResponse</a></code>

Methods:

- <code title="get /chain">client.chain.<a href="./src/resources/chain.ts">retrieve</a>() -> ChainRetrieveResponse</code>

# Block

Types:

- <code><a href="./src/resources/block.ts">Block</a></code>
- <code><a href="./src/resources/block.ts">BlockSubmitResponse</a></code>

Methods:

- <code title="post /block">client.block.<a href="./src/resources/block.ts">submit</a>({ ...params }) -> BlockSubmitResponse</code>

# Transaction

Types:

- <code><a href="./src/resources/transaction.ts">Transaction</a></code>
- <code><a href="./src/resources/transaction.ts">TransactionSubmitResponse</a></code>

Methods:

- <code title="post /transaction">client.transaction.<a href="./src/resources/transaction.ts">submit</a>({ ...params }) -> TransactionSubmitResponse</code>

# TxnPool

Types:

- <code><a href="./src/resources/txn-pool.ts">TxnPoolRetrieveResponse</a></code>

Methods:

- <code title="get /txn_pool">client.txnPool.<a href="./src/resources/txn-pool.ts">retrieve</a>() -> TxnPoolRetrieveResponse</code>

# Info

Types:

- <code><a href="./src/resources/info.ts">InfoRetrieveResponse</a></code>

Methods:

- <code title="get /info">client.info.<a href="./src/resources/info.ts">retrieve</a>() -> InfoRetrieveResponse</code>

# Health

Types:

- <code><a href="./src/resources/health.ts">HealthCheckResponse</a></code>

Methods:

- <code title="get /health">client.health.<a href="./src/resources/health.ts">check</a>() -> HealthCheckResponse</code>

# Ping

Types:

- <code><a href="./src/resources/ping.ts">PingCheckResponse</a></code>

Methods:

- <code title="get /ping">client.ping.<a href="./src/resources/ping.ts">check</a>() -> string</code>

# Peers

Types:

- <code><a href="./src/resources/peers.ts">PeerListResponse</a></code>

Methods:

- <code title="get /peers">client.peers.<a href="./src/resources/peers.ts">list</a>() -> PeerListResponse</code>

# UriGenerator

Types:

- <code><a href="./src/resources/uri-generator.ts">UriGeneratorCreateResponse</a></code>

Methods:

- <code title="post /uriGenerator">client.uriGenerator.<a href="./src/resources/uri-generator.ts">create</a>({ ...params }) -> UriGeneratorCreateResponse</code>

# Mcp

Types:

- <code><a href="./src/resources/mcp/mcp.ts">ContextRecord</a></code>
- <code><a href="./src/resources/mcp/mcp.ts">McpRetrieveCapabilitiesResponse</a></code>
- <code><a href="./src/resources/mcp/mcp.ts">McpRetrieveContextsResponse</a></code>

Methods:

- <code title="get /mcp/context/{context_id}">client.mcp.<a href="./src/resources/mcp/mcp.ts">retrieve</a>(contextID) -> ContextRecord</code>
- <code title="get /mcp/capabilities">client.mcp.<a href="./src/resources/mcp/mcp.ts">retrieveCapabilities</a>({ ...params }) -> McpRetrieveCapabilitiesResponse</code>
- <code title="get /mcp/contexts">client.mcp.<a href="./src/resources/mcp/mcp.ts">retrieveContexts</a>({ ...params }) -> McpRetrieveContextsResponse</code>

## Capability

Types:

- <code><a href="./src/resources/mcp/capability.ts">BaseDescriptor</a></code>
- <code><a href="./src/resources/mcp/capability.ts">CapabilityDescriptor</a></code>
- <code><a href="./src/resources/mcp/capability.ts">CapabilityUpdateResponse</a></code>
- <code><a href="./src/resources/mcp/capability.ts">CapabilityInvokeResponse</a></code>
- <code><a href="./src/resources/mcp/capability.ts">CapabilityPrepareRegistrationResponse</a></code>
- <code><a href="./src/resources/mcp/capability.ts">CapabilityRetrieveInvocationsResponse</a></code>

Methods:

- <code title="get /mcp/capability/{capability_id}">client.mcp.capability.<a href="./src/resources/mcp/capability.ts">retrieve</a>(capabilityID) -> CapabilityDescriptor</code>
- <code title="post /mcp/capability/update">client.mcp.capability.<a href="./src/resources/mcp/capability.ts">update</a>({ ...params }) -> CapabilityUpdateResponse</code>
- <code title="post /mcp/capability/invoke">client.mcp.capability.<a href="./src/resources/mcp/capability.ts">invoke</a>({ ...params }) -> CapabilityInvokeResponse</code>
- <code title="post /mcp/capability/prepare_registration">client.mcp.capability.<a href="./src/resources/mcp/capability.ts">prepareRegistration</a>({ ...params }) -> CapabilityPrepareRegistrationResponse</code>
- <code title="get /mcp/capability/{capability_id}/invocations">client.mcp.capability.<a href="./src/resources/mcp/capability.ts">retrieveInvocations</a>(capabilityID) -> CapabilityRetrieveInvocationsResponse</code>
