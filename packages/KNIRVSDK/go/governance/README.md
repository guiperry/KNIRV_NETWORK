# KNIRV Governance SDK

Embedded governance client for Go-based agent runtimes. Talks to the
KNIRVSERVER `GovernanceAgent` over a Unix domain socket so agent frameworks can
perform zero-trust checks, policy evaluation, and MCP tool-call auditing without
REST round trips.

## Installation

```sh
go get github.com/KNIRV/KNIRV_NETWORK/KNIRVSDK/go/governance
```

## Wire Protocol

Length-prefixed JSON frames over a Unix socket:

1. 4-byte big-endian frame length
2. JSON body

Request: `{"method": "<method>", "args": <method-specific JSON>}`

Response: `{"ok": true, "result": <method-specific JSON>}` or
`{"ok": false, "error": "..."}`

The server socket path defaults to `/var/run/knirvserver/governance.sock`.

## Methods

| Method               | Args                                                     | Result              |
|----------------------|----------------------------------------------------------|---------------------|
| `check_identity`     | `{"node_id","agent_id"}`                                 | `TrustEnvelope`     |
| `evaluate_policy`    | `PolicyInput`                                            | `PolicyDecision`    |
| `record_tool_call`   | `{"agent_id","node_id","tool_name","args"}`              | `{"status"}`        |

## Usage

```go
client, err := governance.NewClient("") // "" -> DefaultSocketPath
if err != nil { /* server socket not up */ }
defer client.Close()

env, err := client.CheckIdentity("node-1", "agent-1")

decision, err := client.EvaluatePolicy(governance.PolicyInput{
    NodeID:     "node-1",
    AgentID:    "agent-1",
    Action:     "invoke_tool",
    ActionType: "tool_call",
})

status, err := client.RecordToolCall("agent-1", "node-1", "search_web", map[string]interface{}{
    "query": "knirv",
})
```

The HTTP `GovernanceClient` in `client.go` remains available for REST-based
integrations; the socket `Client` in `socket.go` is the low-latency
enforcement path.
