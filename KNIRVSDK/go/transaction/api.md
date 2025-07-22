# Chain

Response Types:

- <a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction">knirvchaintransactionsdk</a>.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#ChainGetResponse">ChainGetResponse</a>

Methods:

- <code title="get /chain">client.Chain.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#ChainService.Get">Get</a>(ctx <a href="https://pkg.go.dev/context">context</a>.<a href="https://pkg.go.dev/context#Context">Context</a>) (<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction">knirvchaintransactionsdk</a>.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#ChainGetResponse">ChainGetResponse</a>, <a href="https://pkg.go.dev/builtin#error">error</a>)</code>

# Block

Params Types:

- <a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction">knirvchaintransactionsdk</a>.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#BlockParam">BlockParam</a>

Response Types:

- <a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction">knirvchaintransactionsdk</a>.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#Block">Block</a>
- <a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction">knirvchaintransactionsdk</a>.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#BlockSubmitResponse">BlockSubmitResponse</a>

Methods:

- <code title="post /block">client.Block.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#BlockService.Submit">Submit</a>(ctx <a href="https://pkg.go.dev/context">context</a>.<a href="https://pkg.go.dev/context#Context">Context</a>, body <a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction">knirvchaintransactionsdk</a>.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#BlockSubmitParams">BlockSubmitParams</a>) (<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction">knirvchaintransactionsdk</a>.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#BlockSubmitResponse">BlockSubmitResponse</a>, <a href="https://pkg.go.dev/builtin#error">error</a>)</code>

# Transaction

Params Types:

- <a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction">knirvchaintransactionsdk</a>.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#TransactionParam">TransactionParam</a>

Response Types:

- <a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction">knirvchaintransactionsdk</a>.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#Transaction">Transaction</a>
- <a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction">knirvchaintransactionsdk</a>.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#TransactionSubmitResponse">TransactionSubmitResponse</a>

Methods:

- <code title="post /transaction">client.Transaction.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#TransactionService.Submit">Submit</a>(ctx <a href="https://pkg.go.dev/context">context</a>.<a href="https://pkg.go.dev/context#Context">Context</a>, body <a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction">knirvchaintransactionsdk</a>.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#TransactionSubmitParams">TransactionSubmitParams</a>) (<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction">knirvchaintransactionsdk</a>.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#TransactionSubmitResponse">TransactionSubmitResponse</a>, <a href="https://pkg.go.dev/builtin#error">error</a>)</code>

# TxnPool

Methods:

- <code title="get /txn_pool">client.TxnPool.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#TxnPoolService.Get">Get</a>(ctx <a href="https://pkg.go.dev/context">context</a>.<a href="https://pkg.go.dev/context#Context">Context</a>) ([]<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction">knirvchaintransactionsdk</a>.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#Transaction">Transaction</a>, <a href="https://pkg.go.dev/builtin#error">error</a>)</code>

# Info

Response Types:

- <a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction">knirvchaintransactionsdk</a>.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#InfoGetResponse">InfoGetResponse</a>

Methods:

- <code title="get /info">client.Info.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#InfoService.Get">Get</a>(ctx <a href="https://pkg.go.dev/context">context</a>.<a href="https://pkg.go.dev/context#Context">Context</a>) (<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction">knirvchaintransactionsdk</a>.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#InfoGetResponse">InfoGetResponse</a>, <a href="https://pkg.go.dev/builtin#error">error</a>)</code>

# Health

Response Types:

- <a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction">knirvchaintransactionsdk</a>.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#HealthCheckResponse">HealthCheckResponse</a>

Methods:

- <code title="get /health">client.Health.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#HealthService.Check">Check</a>(ctx <a href="https://pkg.go.dev/context">context</a>.<a href="https://pkg.go.dev/context#Context">Context</a>) (<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction">knirvchaintransactionsdk</a>.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#HealthCheckResponse">HealthCheckResponse</a>, <a href="https://pkg.go.dev/builtin#error">error</a>)</code>

# Ping

Methods:

- <code title="get /ping">client.Ping.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#PingService.Check">Check</a>(ctx <a href="https://pkg.go.dev/context">context</a>.<a href="https://pkg.go.dev/context#Context">Context</a>) (<a href="https://pkg.go.dev/builtin#string">string</a>, <a href="https://pkg.go.dev/builtin#error">error</a>)</code>

# Peers

Response Types:

- <a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction">knirvchaintransactionsdk</a>.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#PeerListResponse">PeerListResponse</a>

Methods:

- <code title="get /peers">client.Peers.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#PeerService.List">List</a>(ctx <a href="https://pkg.go.dev/context">context</a>.<a href="https://pkg.go.dev/context#Context">Context</a>) ([]<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction">knirvchaintransactionsdk</a>.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#PeerListResponse">PeerListResponse</a>, <a href="https://pkg.go.dev/builtin#error">error</a>)</code>

# UriGenerator

Response Types:

- <a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction">knirvchaintransactionsdk</a>.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#UriGeneratorNewResponse">UriGeneratorNewResponse</a>

Methods:

- <code title="post /uriGenerator">client.UriGenerator.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#UriGeneratorService.New">New</a>(ctx <a href="https://pkg.go.dev/context">context</a>.<a href="https://pkg.go.dev/context#Context">Context</a>, body <a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction">knirvchaintransactionsdk</a>.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#UriGeneratorNewParams">UriGeneratorNewParams</a>) (<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction">knirvchaintransactionsdk</a>.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#UriGeneratorNewResponse">UriGeneratorNewResponse</a>, <a href="https://pkg.go.dev/builtin#error">error</a>)</code>

# Mcp

Response Types:

- <a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction">knirvchaintransactionsdk</a>.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#ContextRecord">ContextRecord</a>

Methods:

- <code title="get /mcp/context/{context_id}">client.Mcp.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#McpService.Get">Get</a>(ctx <a href="https://pkg.go.dev/context">context</a>.<a href="https://pkg.go.dev/context#Context">Context</a>, contextID <a href="https://pkg.go.dev/builtin#string">string</a>) (<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction">knirvchaintransactionsdk</a>.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#ContextRecord">ContextRecord</a>, <a href="https://pkg.go.dev/builtin#error">error</a>)</code>
- <code title="get /mcp/capabilities">client.Mcp.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#McpService.GetCapabilities">GetCapabilities</a>(ctx <a href="https://pkg.go.dev/context">context</a>.<a href="https://pkg.go.dev/context#Context">Context</a>, query <a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction">knirvchaintransactionsdk</a>.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#McpGetCapabilitiesParams">McpGetCapabilitiesParams</a>) ([]<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction">knirvchaintransactionsdk</a>.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#CapabilityDescriptorUnion">CapabilityDescriptorUnion</a>, <a href="https://pkg.go.dev/builtin#error">error</a>)</code>
- <code title="get /mcp/contexts">client.Mcp.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#McpService.GetContexts">GetContexts</a>(ctx <a href="https://pkg.go.dev/context">context</a>.<a href="https://pkg.go.dev/context#Context">Context</a>, query <a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction">knirvchaintransactionsdk</a>.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#McpGetContextsParams">McpGetContextsParams</a>) ([]<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction">knirvchaintransactionsdk</a>.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#ContextRecord">ContextRecord</a>, <a href="https://pkg.go.dev/builtin#error">error</a>)</code>

## Capability

Params Types:

- <a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction">knirvchaintransactionsdk</a>.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#BaseDescriptorParam">BaseDescriptorParam</a>
- <a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction">knirvchaintransactionsdk</a>.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#CapabilityDescriptorUnionParam">CapabilityDescriptorUnionParam</a>

Response Types:

- <a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction">knirvchaintransactionsdk</a>.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#BaseDescriptor">BaseDescriptor</a>
- <a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction">knirvchaintransactionsdk</a>.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#CapabilityDescriptorUnion">CapabilityDescriptorUnion</a>
- <a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction">knirvchaintransactionsdk</a>.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#McpCapabilityUpdateResponse">McpCapabilityUpdateResponse</a>
- <a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction">knirvchaintransactionsdk</a>.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#McpCapabilityInvokeResponse">McpCapabilityInvokeResponse</a>
- <a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction">knirvchaintransactionsdk</a>.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#McpCapabilityPrepareRegistrationResponse">McpCapabilityPrepareRegistrationResponse</a>

Methods:

- <code title="get /mcp/capability/{capability_id}">client.Mcp.Capability.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#McpCapabilityService.Get">Get</a>(ctx <a href="https://pkg.go.dev/context">context</a>.<a href="https://pkg.go.dev/context#Context">Context</a>, capabilityID <a href="https://pkg.go.dev/builtin#string">string</a>) (<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction">knirvchaintransactionsdk</a>.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#CapabilityDescriptorUnion">CapabilityDescriptorUnion</a>, <a href="https://pkg.go.dev/builtin#error">error</a>)</code>
- <code title="post /mcp/capability/update">client.Mcp.Capability.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#McpCapabilityService.Update">Update</a>(ctx <a href="https://pkg.go.dev/context">context</a>.<a href="https://pkg.go.dev/context#Context">Context</a>, body <a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction">knirvchaintransactionsdk</a>.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#McpCapabilityUpdateParams">McpCapabilityUpdateParams</a>) (<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction">knirvchaintransactionsdk</a>.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#McpCapabilityUpdateResponse">McpCapabilityUpdateResponse</a>, <a href="https://pkg.go.dev/builtin#error">error</a>)</code>
- <code title="post /mcp/capability/invoke">client.Mcp.Capability.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#McpCapabilityService.Invoke">Invoke</a>(ctx <a href="https://pkg.go.dev/context">context</a>.<a href="https://pkg.go.dev/context#Context">Context</a>, body <a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction">knirvchaintransactionsdk</a>.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#McpCapabilityInvokeParams">McpCapabilityInvokeParams</a>) (<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction">knirvchaintransactionsdk</a>.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#McpCapabilityInvokeResponse">McpCapabilityInvokeResponse</a>, <a href="https://pkg.go.dev/builtin#error">error</a>)</code>
- <code title="post /mcp/capability/prepare_registration">client.Mcp.Capability.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#McpCapabilityService.PrepareRegistration">PrepareRegistration</a>(ctx <a href="https://pkg.go.dev/context">context</a>.<a href="https://pkg.go.dev/context#Context">Context</a>, body <a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction">knirvchaintransactionsdk</a>.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#McpCapabilityPrepareRegistrationParams">McpCapabilityPrepareRegistrationParams</a>) (<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction">knirvchaintransactionsdk</a>.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#McpCapabilityPrepareRegistrationResponse">McpCapabilityPrepareRegistrationResponse</a>, <a href="https://pkg.go.dev/builtin#error">error</a>)</code>
- <code title="get /mcp/capability/{capability_id}/invocations">client.Mcp.Capability.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#McpCapabilityService.GetInvocations">GetInvocations</a>(ctx <a href="https://pkg.go.dev/context">context</a>.<a href="https://pkg.go.dev/context#Context">Context</a>, capabilityID <a href="https://pkg.go.dev/builtin#string">string</a>) ([]<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction">knirvchaintransactionsdk</a>.<a href="https://pkg.go.dev/github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction#ContextRecord">ContextRecord</a>, <a href="https://pkg.go.dev/builtin#error">error</a>)</code>
