# API Refactor: Unified Proxy Architecture

**Goal:** Route all submodule API endpoints through KNIRVGATEWAY via Unix sockets (except KNIRVORACLE which stays TCP), and ensure all WebGUI links resolve to working backend endpoints.

---

## 1. Current Architecture

### 1.1 Request Flow

```
Public Internet
      │
      ▼
KNIRVGATEWAY (port 8080 / 8888)
  │
  ├─ /api/v1/*  (except /api/v1/info) ──► backend.sock ──► KNIRVSERVER backend
  │
  ├─ /chain/*                    ──► chain.sock ──► KNIRVCHAIN
  ├─ /graph/*                    ──► graph.sock ──► KNIRVGRAPH
  ├─ /wallet/info                ──► oracle (TCP/socket)
  ├─ /transaction                ──► oracle (TCP/socket)
  │
  ├─ /api/objects                  (mock data, no backend)
  ├─ /api/transactions             (mock data, no backend)
  ├─ /api/blocks                   (mock data, no backend)
  ├─ /api/assets                   (mock data, no backend)
  ├─ /api/view/{id}                (mock data, no backend)
  ├─ /api/network-monitor/*        (mock data, no backend)
  └─ /api                          (catch-all mock, no backend)
```

### 1.2 What's Missing / Broken

| Issue | Detail |
|-------|--------|
| **Mock data handlers** | `/api/objects`, `/api/transactions`, `/api/blocks`, `/api/assets`, `/api/view/{id}` on the gateway return fake data instead of proxying to the real submodule |
| **KNIRVAGENT not proxied** | KNIRVAGENT is spawned on-demand per DVE on TCP port 8081; no gateway proxy route |
| **KNIRVHASHER not proxied** | gRPC-only, no HTTP surface at all |
| **KNIRVSHELL not socked** | KNIRVSHELLService runs in-process in KNIRVSERVER backend, not as a separate socket service |
| **WebGUI hardcoded mock routes** | The WebGUI frontend calls `/api/objects`, `/api/transactions` etc. which hit mock handlers, not real submodule data |
| **Oracle proxy inconsistency** | Oracle is proxied via TCP `localhost:1317` by default, with optional socket override |
| **Gateway /api/v1/info special-case** | Handled locally by the gateway instead of proxied to backend |
| **Submodule boundary confusion** | `/transaction` is routed to oracle while KNIRVCHAIN also has a `/transaction` route — no clear boundary between what oracle owns vs what KNIRVCHAIN owns |

---

## 2. Target Architecture

### 2.1 Submodule API Responsibility Boundary

This is the core architectural decision governing all routing:

| Submodule | Owns | Examples |
|-----------|------|----------|
| **KNIRVORACLE** | Cosmos blockchain queries — transactions, blocks, wallet balances, token info, governance, staking, IBC, cross-chain | `GET /api/oracle/v3/token/balance/{address}`, `POST /api/oracle/v3/token/transfer`, `GET /api/oracle/v3/economics/metrics`, `POST /api/oracle/v3/crosschain/transfer` |
| **KNIRVCHAIN** | Agent ecosystem — credentials, task context, capabilities/skills, MCP contexts, badge/NFT minting, agent facts, resource capabilities, PoAu-D | `GET /api/chain/mcp/capability/list`, `POST /api/chain/agent/capability/invoke`, `POST /api/chain/nft/upload`, `GET /api/chain/agent/agent-facts/{id}`, `POST /api/chain/poaud/enable` |
| **KNIRVGRAPH** | Knowledge graph — NRV vectors, errors, skills, graph traversal, economics metrics | `GET /api/graph/node/{nodeID}`, `POST /api/graph/graph/traverse`, `POST /api/graph/nrv/errors` |
| **KNIRVSERVER backend** | Application logic — DVE management, plugins, auth, payments, shell, onboarding, cognitive engine, knowledge base, health | `GET /api/v1/dve/nodes`, `POST /api/v1/auth/login`, `GET /api/v1/shell/sessions` |
| **KNIRVAGENT** | DVE supervisor agent — per-DVE subprocess for agent interaction | `POST /api/agent/{dveId}/session`, `POST /api/agent/{dveId}/input` |
| **KNIRVSHELL** | Standalone shell binary — command execution, wallet operations | `POST /api/shell/execute`, `GET /api/shell/sessions` |
| **KNIRVHASHER** | Data hashing pipeline — grpc-only, bridge via backend | No direct HTTP; `POST /api/v1/hasher/ping` through backend |

**Key delineation — Oracle vs KNIRVCHAIN:**

- **KNIRVORACLE** is the gateway to the underlying Cosmos blockchain. It answers queries about the canonical blockchain state: transactions that have been committed, block headers, account balances, token supply, staking data, governance proposals, IBC connections. It can also submit transactions to the Cosmos chain (mint, transfer, burn, vote, cross-chain transfer).

- **KNIRVCHAIN** is the application layer built *on top of* the Cosmos chain. It manages the KNIRV ecosystem's agent-related state: agent credentials (which agents exist and their identities), agent task context (what tasks an agent is working on), agent capabilities and skills (MCP capability registry), badge/NFT minting and ownership, resource capability linking, and Proof-of-Authority Delegation (PoAu-D) state.

- **Wallet and balance queries** (e.g., "how many tokens does this address have?") go to **oracle**.
- **Badge ownership checks** (e.g., "which badges does this agent hold?") go to **KNIRVCHAIN**.
- **Transaction submission** (e.g., "send tokens from A to B") goes to **oracle**.
- **Transaction history** (e.g., "show me the last 10 blocks") goes to **oracle**.
- **Capability registration** (e.g., "register a new MCP capability") goes to **KNIRVCHAIN**.

### 2.2 Request Flow After Refactor

```
Public Internet
      │
      ▼
KNIRVGATEWAY (port 8080 / 8888)
  │
  ├─ /api/v1/*                      ──► backend.sock ──► KNIRVSERVER backend
  │   (includes /api/v1/info)
  │
  ├─ /api/chain/*                   ──► chain.sock ──► KNIRVCHAIN
  │   (agent credentials, capabilities, badges, MCP tasks)
  │
  ├─ /api/graph/*                   ──► graph.sock ──► KNIRVGRAPH
  │
  ├─ /api/oracle/*                  ──► oracle (TCP localhost:1317)
  │   (Cosmos txns, blocks, balances, staking, governance)
  │
  ├─ /api/shell/*                   ──► shell.sock ──► KNIRVSHELL daemon
  ├─ /api/agent/*                   ──► agent.sock ──► KNIRVAGENT (on-demand)
  ├─ /api/hasher/*                  ──► backend.sock (bridged from backend's hasher handlers)
  │
  ├─ /api/objects                  ──► chain.sock ──► KNIRVCHAIN agent objects
  ├─ /api/transactions             ──► oracle (TCP) ──➤ Cosmos tx history
  ├─ /api/blocks                   ──► oracle (TCP) ──➤ Cosmos block headers
  ├─ /api/assets                   ──► chain.sock ──► KNIRVCHAIN badges/NFTs
  ├─ /api/view/{id}                ──► [TBD per WebGUI audit in Phase 8]
  │
  ├─ /chain/*                      ──► 301 redirect ──► /api/chain/*
  ├─ /wallet/info                  ──► 301 redirect ──► /api/oracle/wallet/info
  ├─ /transaction                  ──► 301 redirect ──► /api/oracle/transaction
  └─ /devs                         ──► 301 redirect ──► /api/chain/devs
```

### 2.3 Socket Layout

```
/var/lib/knirvserver/sockets/
  ├── backend.sock       (KNIRVSERVER backend — already exists)
  ├── chain.sock         (KNIRVCHAIN — already exists)
  ├── graph.sock         (KNIRVGRAPH — already exists)
  ├── shell.sock         (KNIRVSHELL — NEW, extract KNIRVSHELL into a socket daemon)
  ├── hasher.sock        (KNIRVHASHER — already exists, gRPC only)
  ├── agent.sock         (KNIRVAGENT — NEW, change from TCP to socket)
  └── oracle TCP         (localhost:1317 — unchanged, explicitly TCP)
```

---

## 3. Phase-by-Phase Plan

### Phase 1: Oracle → Standardized `/api/oracle/*` Proxy

**Problem:** Oracle proxy is inconsistent — uses TCP by default, optionally Unix socket. Some `/wallet/info` and `/transaction` routes are special-cased to oracle. No `/api/oracle/*` prefix exists yet.

**Solution:** Create a dedicated `/api/oracle/*` proxy route. The oracle handles all Cosmos blockchain queries:
- Wallet/balance queries (`/wallet/info` → `/api/oracle/wallet/info`)
- Transaction queries and submission (`/transaction` → `/api/oracle/transaction`)
- Cosmos blockchain state (blocks, staking, governance, IBC)
- Token operations (mint, transfer, burn)

**Steps:**

1. **In `packages/KNIRVGATEWAY/internal/server/server.go`:**
   - Register `r.PathPrefix("/api/oracle/")` → oracle TCP proxy (to `localhost:1317` or configured `OracleSocketPath`)
   - Add redirects:
     - `/wallet/info` → 301 → `/api/oracle/wallet/info`
     - `/transaction` → 301 → `/api/oracle/transaction`
   - Remove the old `handleOracleGet` and `handleOraclePost` handlers
   - Remove the old `/wallet/info` and `/transaction` special-cased route registrations
   - Keep `OracleSocketPath` logic but under the new `/api/oracle/` prefix

**Files changed:**
- `packages/KNIRVGATEWAY/internal/server/server.go`

---

### Phase 2: KNIRVCHAIN → Standardized `/api/chain/*` Proxy

**Problem:** `/chain/*` is already proxied to `chain.sock` (good), but should be under `/api/chain/*` for consistency. Some routes like `/devs` and `/txn_pool` have special-cased mock handlers instead of going to chain.

**Solution:** Create `/api/chain/*` prefix that proxies to `chain.sock`. KNIRVCHAIN owns:
- Agent credentials and identity
- Agent task context and history
- Agent capabilities and skills (MCP capability registry, resource capabilities)
- Badge and NFT minting/ownership/attachments
- PoAu-D state (network authors, status)
- Internal DHT routing and chain sync

**Steps:**

1. **In `packages/KNIRVGATEWAY/internal/server/server.go`:**
   - Register `r.PathPrefix("/api/chain/").Handler(chainProxy)`
   - Keep existing `/chain/*` routes as redirect to `/api/chain/*`
   - Remove the special-cased `/chain` (GET) handler — let it fall through to the chain proxy
   - Remove the `/txn_pool` mock handler — proxy to `/api/chain/txn_pool` instead
   - Remove the `/devs` mock handler — proxy to `/api/chain/devs` instead
   - Update the existing `/chain/*` passthrough to use the same proxy

**Files changed:**
- `packages/KNIRVGATEWAY/internal/server/server.go`

---

### Phase 3: KNIRVGRAPH → Standardized `/api/graph/*` Proxy

**Problem:** `/graph/*` is already proxied to `graph.sock` (good), but should be under `/api/graph/*` for consistency.

**Solution:** Add `/api/graph/*` prefix alongside existing flat path. Keep `/graph/*` as redirect.

**Steps:**

1. **In `packages/KNIRVGATEWAY/internal/server/server.go`:**
   - Register `r.PathPrefix("/api/graph/").Handler(graphProxy)`
   - Keep existing `/graph/*` as redirect to `/api/graph/*`

**Files changed:**
- `packages/KNIRVGATEWAY/internal/server/server.go`

---

### Phase 4: Gateway WebGUI Mock Endpoints → Real Submodule Proxies

**Problem:** The WebGUI frontend calls `/api/objects`, `/api/transactions`, `/api/blocks`, `/api/assets`, `/api/view/{id}` which hit mock data handlers on the gateway.

**Solution:** Replace mock handlers with targeted proxies to the correct submodule based on the responsibility boundary:

| Gateway Route | Data Type | Target Submodule | Target Path |
|---------------|-----------|-----------------|-------------|
| `GET /api/transactions` | Cosmos tx history | **KNIRVORACLE** | `/api/oracle/v3/economics/metrics` (nearest endpoint) |
| `GET /api/blocks` | Cosmos block headers | **KNIRVORACLE** | `/api/oracle/v3/consensus/status` (nearest endpoint) |
| `GET /api/objects` | Agent objects/capabilities | **KNIRVCHAIN** | `/api/chain/mcp/capability/list` |
| `GET /api/assets` | Badges/NFTs | **KNIRVCHAIN** | `/api/chain/nft/list` |
| `GET /api/view/{id}` | Unknown — TBD | **TBD after WebGUI audit** | — |

**Note:** Some WebGUI frontend routes may expect data shapes that don't exactly match the submodule's existing endpoints. In those cases, add a thin handler on the KNIRVSERVER backend (accessible via `backend.sock`) that transforms the data. For example, the backend can query both the oracle (for balances) and the chain (for badges) and merge results.

**Steps:**

1. **In `packages/KNIRVGATEWAY/internal/server/server.go`:**
   - Remove the 5 mock handler functions (`handleObjects`, `handleTransactions`, `handleBlocks`, `handleAssets`, `handleView`)
   - Replace with proxy registrations:
     - `GET /api/objects` → `chainProxy` (chain.sock)
     - `GET /api/transactions` → `oracleProxy` (oracle TCP)
     - `GET /api/blocks` → `oracleProxy` (oracle TCP)
     - `GET /api/assets` → `chainProxy` (chain.sock)
     - `GET /api/view/{id}` → TBD (deferred to Phase 8)
   - Remove the catch-all mock handler at `r.PathPrefix("/api").HandlerFunc(s.handleMockAPI)` — this currently catches unmatched `/api/*` requests and returns mock data, which masks missing routes

**Files changed:**
- `packages/KNIRVGATEWAY/internal/server/server.go`

---

### Phase 5: KNIRVSHELL → Unix Socket Daemon

**Problem:** KNIRVSHELL runs in-process in the backend server. It needs to be extracted into a standalone socket daemon so the gateway can proxy to it.

**Solution:** Create a `shell.sock` Unix socket listener for KNIRVSHELL, managed by a `knirvshell.Manager` (following the pattern from `knirvagent.Manager` and `knirvgraph.Manager`). The gateway then proxies `/api/shell/*` → `shell.sock`.

**Steps:**

1. **In `packages/KNIRVSERVER/pkg/knirvshell/`:**
   - Add `manager.go` with a `Manager` struct (Start/Stop/HealthCheck lifecycle) following the pattern from `pkg/knirvagent/manager.go`
   - Manager spawns `bin/knirvshell` as subprocess listening on `shell.sock`
   - Add `--socket-path` flag support to the KNIRVSHELL binary
   - This may require changes to the KNIRVSHELL package itself (separate repo at `github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL`)

2. **In `packages/KNIRVSERVER/backend/cmd/backend_server/main.go`:**
   - Initialize `knirvshell.Manager` in `NewServer()`
   - Configure its socket path from `cfg.SocketDir/shell.sock`
   - Remove the in-process `knirvshellService` instantiation
   - Pass `knirvshellManager` to the API router (or let the gateway handle shell routes directly)

3. **In `packages/KNIRVGATEWAY/internal/server/server.go`:**
   - Add a new proxy `shellProxy` pointing to `shell.sock`
   - Register `r.PathPrefix("/api/shell/").Handler(shellProxy)`
   - Add redirect: `/api/v1/shell/*` → 307 → `/api/shell/*`
   - Add redirect: `/api/knirvshell/*` → 307 → `/api/shell/*`

4. **In `packages/KNIRVSERVER/backend/internal/config/config.go`:**
   - Add `ShellSocketPath string \`mapstructure:"shell_socket"\`` to `GatewayConfig`
   - Add YAML field `shell_socket` to config files

**Files changed:**
- `packages/KNIRVSERVER/pkg/knirvshell/manager.go` (NEW)
- `packages/KNIRVSERVER/backend/cmd/backend_server/main.go`
- `packages/KNIRVGATEWAY/internal/server/server.go`
- `packages/KNIRVSERVER/backend/internal/config/config.go`
- `packages/KNIRVSERVER/config/production.yaml`
- `github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL` (external — socket flag + listener)

---

### Phase 6: KNIRVAGENT → Unix Socket

**Problem:** KNIRVAGENT listens on TCP port 8081 by default. It needs to switch to a Unix socket so the gateway can proxy to it.

**Solution:** Change the KNIRVAGENT manager to listen on `agent.sock` instead of TCP port 8081. Add gateway proxy routes.

**Steps:**

1. **In `packages/KNIRVSERVER/pkg/knirvagent/manager.go`:**
   - Change the default listener from `:8081` to `{socketDir}/agent.sock`
   - Update the `startProcess()` method to pass `--socket-path` flag to `bin/knirvagent`
   - Keep the health check working over the socket

2. **In `packages/KNIRVGATEWAY/internal/server/server.go`:**
   - Add a new proxy `agentProxy` pointing to `agent.sock`
   - Register `r.PathPrefix("/api/agent/").Handler(agentProxy)`
   - Note: KNIRVAGENT starts on-demand per DVE — the proxy should return 503 when the agent process is not running

3. **In `packages/KNIRVSERVER/backend/internal/config/config.go`:**
   - Add `AgentSocketPath string \`mapstructure:"agent_socket"\`` to `GatewayConfig`

**Files changed:**
- `packages/KNIRVSERVER/pkg/knirvagent/manager.go`
- `packages/KNIRVGATEWAY/internal/server/server.go`
- `packages/KNIRVSERVER/backend/internal/config/config.go`

---

### Phase 7: KNIRVHASHER gRPC → HTTP Bridge

**Problem:** KNIRVHASHER is gRPC-only with no HTTP surface. The gateway can't HTTP-proxy to a gRPC socket.

**Solution:** Expose HTTP endpoints in the backend server that internally delegate to KNIRVHASHER via gRPC. The gateway proxies `/api/hasher/*` → `backend.sock`, which the backend handles internally.

**Steps:**

1. **In `packages/KNIRVSERVER/backend/internal/web/api_router.go`:**
   - Add `/api/v1/hasher/*` routes that use the existing `ar.hasherIntegration` to translate HTTP → gRPC
   - Minimal endpoint surface: `/api/v1/hasher/status`, `/api/v1/hasher/ping`

2. **In `packages/KNIRVGATEWAY/internal/server/server.go`:**
   - Register `r.PathPrefix("/api/hasher/").Handler(backendProxy)` (falls through to backend.sock which handles it)

**Files changed:**
- `packages/KNIRVSERVER/backend/internal/web/api_router.go`
- `packages/KNIRVSERVER/backend/internal/web/hasher_handlers.go` (NEW)

---

### Phase 8: Gateway `/api/v1/info` Unification

**Problem:** `/api/v1/info` is handled locally by the gateway while everything else under `/api/v1/` is proxied to the backend. This creates inconsistency.

**Solution:** Move `/api/v1/info` to the backend and remove the gateway's local handler.

**Steps:**

1. **In `packages/KNIRVGATEWAY/internal/server/server.go`:**
   - Remove the `handleAPIInfo` handler
   - Remove the special-case route for `/api/v1/info`
   - Let `/api/v1/info` fall through to the existing `backendProxy` (which handles all other `/api/v1/*` routes)

2. **In `packages/KNIRVSERVER/backend/cmd/backend_server/main.go` (setupRoutes):**
   - Verify `/api/v1/info` is already handled by the backend (it likely is — check setupRoutes for an `/api/v1/info` handler)

**Files changed:**
- `packages/KNIRVGATEWAY/internal/server/server.go`

---

### Phase 9: WebGUI Frontend Link Resolution Audit

**Problem:** Some WebGUI frontend pages (35+ SPA pages) may reference API endpoints that don't exist or hit mock data.

**Solution:** Audit all API calls in the WebGUI frontend and ensure they resolve to the new consolidated paths.

**Steps:**

1. **Search all WebGUI frontend JS/TS files** in `packages/KNIRVGATEWAY/internal/embedded/webgui/src/` for hardcoded API URLs

2. **Categorize each URL by submodule responsibility:**
   - Calls querying Cosmos blockchain state (blocks, txns, balances) → route to `/api/oracle/*`
   - Calls querying agent ecosystem (capabilities, badges, credentials) → route to `/api/chain/*`
   - Calls for knowledge graph data → route to `/api/graph/*`
   - Calls for shell/terminal → route to `/api/shell/*`
   - Calls for application logic (DVE, auth, plugins) → route to `/api/v1/*` (backend.sock)
   - Calls for supervisor agent → route to `/api/agent/*`

3. **Update any URLs** that reference old paths

4. **Verify each endpoint** returns real data (not mock) after phases 1-8

5. **Add any missing transform handlers** on the backend where the WebGUI expects a different data shape than what the submodule provides

**Files changed:**
- Multiple WebGUI frontend files
- Potentially new backend handler files for data transformation

---

## 4. Dependency Order

```
Phase 1 (oracle /api/oracle/*)   — no dependencies
    │
    ▼
Phase 2 (chain /api/chain/*)     — no dependencies
    │
    ▼
Phase 3 (graph /api/graph/*)     — no dependencies
    │
    ▼
Phase 4 (WebGUI mocks → real)    — depends on: Phases 1-2 (need oracle + chain proxy)
    │
    ▼
Phase 8 (/api/v1/info unify)     — no dependencies
    │
    ▼
Phase 5 (KNIRVSHELL socket)      — depends on: new Manager binary support
    │
    ▼
Phase 6 (KNIRVAGENT socket)      — depends on: agent binary socket flag
    │
    ▼
Phase 7 (KNIRVHASHER HTTP bridge) — depends on: none (uses existing HasherIntegration)
    │
    ▼
Phase 9 (WebGUI audit)           — depends on: all phases 1-8
```

Phases 1, 2, 3, and 8 can be done in parallel. Phase 4 depends on 1-2. Phases 5, 6, 7 each have unique dependencies. Phase 9 must be last.

---

## 5. Gateway Config Changes (all phases)

### New fields in `GatewayConfig` (`packages/KNIRVSERVER/backend/internal/config/config.go`):

```go
type GatewayConfig struct {
    // ... existing fields ...
    ShellSocketPath string `mapstructure:"shell_socket"`  // NEW
    AgentSocketPath string `mapstructure:"agent_socket"`  // NEW
    OracleSocketPath string `mapstructure:"oracle_socket"` // EXISTING but standardized
}
```

### New fields in gateway's internal config:

```go
type Config struct {
    // ... existing fields ...
    ShellSocketPath string  // NEW
    AgentSocketPath string  // NEW
}
```

### YAML additions (`config/production.yaml`):

```yaml
gateway:
  backend_socket: /var/lib/knirvserver/sockets/backend.sock
  chain_socket: /var/lib/knirvserver/sockets/chain.sock
  graph_socket: /var/lib/knirvserver/sockets/graph.sock
  shell_socket: /var/lib/knirvserver/sockets/shell.sock       # NEW
  agent_socket: /var/lib/knirvserver/sockets/agent.sock        # NEW
  oracle_socket: ""                                              # stays TCP
```

---

## 6. Verification Plan

### Per-phase testing:

| Phase | Test |
|-------|------|
| 1 | `curl http://localhost:8888/api/oracle/v3/health` returns oracle health |
| 1 | `curl http://localhost:8888/wallet/info` → 301 → `/api/oracle/wallet/info` → oracle data |
| 1 | `curl http://localhost:8888/transaction` → 301 → `/api/oracle/transaction` → oracle data |
| 2 | `curl http://localhost:8888/api/chain/health` returns KNIRVCHAIN health |
| 2 | `curl http://localhost:8888/api/chain/mcp/capability/list` returns capabilities |
| 2 | `curl http://localhost:8888/chain/health` → 301 → `/api/chain/health` → chain data |
| 3 | `curl http://localhost:8888/api/graph/health` returns graph health |
| 4 | `curl http://localhost:8888/api/objects` returns real chain data (not mock) |
| 4 | `curl http://localhost:8888/api/transactions` returns oracle data (not mock) |
| 4 | `curl http://localhost:8888/api/blocks` returns oracle data (not mock) |
| 4 | `curl http://localhost:8888/api/assets` returns chain badge/NFT data (not mock) |
| 5 | `curl --unix-socket /var/.../shell.sock http://localhost/sessions` returns shell sessions |
| 5 | `curl http://localhost:8888/api/shell/sessions` returns same data through gateway |
| 6 | `curl http://localhost:8888/api/agent/health` returns 503 when no DVE running |
| 7 | `curl http://localhost:8888/api/hasher/status` returns hasher status |
| 8 | `curl http://localhost:8888/api/v1/info` returns backend info (not gateway info) |
| 9 | WebGUI loads without any 404 or mock data in browser console |

### Integration test:

```bash
# Start backend
cd packages/KNIRVSERVER && go run main.go

# In another terminal, test all proxy routes
for route in \
  /api/v1/health \
  /api/oracle/v3/health \
  /api/chain/health \
  /api/graph/health \
  /api/shell/sessions \
  /api/objects \
  /api/transactions \
  /api/blocks \
  /api/assets; do
  echo "=== $route ==="
  curl -s http://localhost:8888$route | head -5
  echo
done
```

---

## 7. Risks and Tradeoffs

| Risk | Mitigation |
|------|-----------|
| **KNIRVSHELL binary doesn't support socket mode** | May require upstream changes to the KNIRVSHELL repo; fall back to keeping it in-process if changes are too invasive |
| **KNIRVAGENT on-demand lifecycle** | Socket doesn't exist when no DVE is active; proxy must return 503 gracefully. Add a health-check that skips missing sockets |
| **gRPC→HTTP bridge complexity** | Keep KNIRVHASHER gRPC-only; expose HTTP endpoints in the backend server that internally call gRPC (simpler than adding an HTTP server to KNIRVHASHER) |
| **WebGUI hardcoded URL paths** | Some frontend paths may be baked into the SPA bundle; may need a build step or runtime config override |
| **Backward compatibility** | Old flat paths (`/chain`, `/wallet/info`, `/transaction`) need permanent 301 redirects for any external consumers |
| **Gateway startup ordering** | Gateway must handle missing sockets gracefully (proxy returns 502/503) — sockets may appear after gateway starts |
| **WebGUI expects merged data** | If a WebGUI page needs data from both oracle (balance) and chain (badges), a backend aggregation handler may be needed rather than a simple proxy |
| **Oracle vs Chain endpoint naming** | WebGUI calls `/api/objects`, `/api/transactions` — these are frontend-specific shorthands. After audit in Phase 9, they may need backend transform handlers that reshape submodule response data into what the WebGUI expects |

---

## 8. Open Questions

1. Does the KNIRVSHELL binary at `github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL` support a `--socket-path` flag? If not, how invasive is adding it?

2. Does the WebGUI frontend bundle get rebuilt with path changes, or are paths runtime-configurable (e.g., via API_BASE_URL)?

3. What exact data shape does the WebGUI expect from `/api/objects`, `/api/transactions`, `/api/blocks`, `/api/assets`, `/api/view/{id}`? The answer determines whether a simple proxy to chain/oracle suffices or a backend transform handler is needed.

4. Should the oracle's TCP address be configurable via gateway config (currently hardcoded `localhost:1317`)?

5. Are there any external consumers that depend on the mock data currently served at `/api/objects`, `/api/transactions`, etc.?

6. For the `/devs` endpoint — is this Cosmos validator data (oracle) or KNIRVCHAIN peer data (chain)? The current redirect in the diagram sends it to chain — confirm during implementation.
