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
| **KNIRVCHAIN** | Agent ecosystem — credentials, task context, capabilities/skills, MCP contexts, badge/NFT minting, agent facts, resource capabilities, PoAu-D, bootstrap peer registry | `GET /api/chain/mcp/capability/list`, `POST /api/chain/agent/capability/invoke`, `POST /api/chain/nft/upload`, `GET /api/chain/agent/agent-facts/{id}`, `POST /api/chain/poaud/enable`, `GET /api/chain/bootnodes` |
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
  ├─ /api/agent/{dveId}/*          ──► agent-{dveId}.sock ──► KNIRVAGENT (per-DVE, on-demand)
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
  └─ /bootnodes                    ──► 301 redirect ──► /api/chain/bootnodes
```

### 2.3 Socket Layout

```
/var/lib/knirvserver/sockets/
  ├── backend.sock       (KNIRVSERVER backend — already exists)
  ├── chain.sock         (KNIRVCHAIN — already exists)
  ├── graph.sock         (KNIRVGRAPH — already exists)
  ├── shell.sock         (KNIRVSHELL — NEW, extract KNIRVSHELL into a socket daemon)
  ├── hasher.sock        (KNIRVHASHER — already exists, gRPC only)
  ├── agent-{dveId}.sock  (KNIRVAGENT — NEW, per-DVE sockets, one per active DVE)
  └── oracle TCP          (localhost:1317 — runtime-discovered port via port file or env var)
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
   - Register `r.PathPrefix("/api/oracle/")` → oracle TCP proxy. The oracle TCP address is resolved at runtime (not a static config field) — the oracle binary binds to an arbitrary available port and advertises it via a well-known discovery mechanism (e.g., a port file in the socket directory, or an environment variable propagated from oracle startup). The gateway reads the live port per-request. Default fallback: `localhost:1317`.
   - Add redirects:
     - `/wallet/info` → 301 → `/api/oracle/wallet/info`
     - `/transaction` → 301 → `/api/oracle/transaction`
   - Remove the old `handleOracleGet` and `handleOraclePost` handlers
   - Remove the old `/wallet/info` and `/transaction` special-cased route registrations
   - Keep the runtime port discovery logic but under the new `/api/oracle/` prefix

**Files changed:**
- `packages/KNIRVGATEWAY/internal/server/server.go`

---

### Phase 2: KNIRVCHAIN → Standardized `/api/chain/*` Proxy

**Problem:** `/chain/*` is already proxied to `chain.sock` (good), but should be under `/api/chain/*` for consistency. Some routes like `/bootnodes` and `/txn_pool` have special-cased mock handlers instead of going to chain.

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
   - Remove the `/bootnodes` mock handler — proxy to `/api/chain/bootnodes` instead
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
| `GET /api/objects` | Agent objects/capabilities | **KNIRVCHAIN** | `/api/chain/mcp/capability/list` — edit WebGUI to consume chain-native shape |
| `GET /api/transactions` | Cosmos tx history | **KNIRVORACLE** | `/api/oracle/v3/economics/metrics` — edit WebGUI or add backend wrapper |
| `GET /api/blocks` | Cosmos block headers | **KNIRVORACLE** | `/api/oracle/v3/consensus/status` — edit WebGUI or add backend wrapper |
| `GET /api/assets` | Badges/NFTs | **KNIRVCHAIN** | `/api/chain/nft/list` |
| `GET /api/view/{id}` | 3D model viewer HTML | **KNIRVCHAIN** | `/api/chain/nft/{id}` — if shape doesn't match, prefer editing the WebGUI component that renders the viewer

**Note:** Most mock endpoints return data in shapes the frontend already expects. For endpoints where the submodule data shape doesn't match, prefer editing the WebGUI frontend (or the backend response) to align shapes — there is no need for a separate transform layer. See Q3 in §9 for exact expected shapes.

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
   - Remove the `/devs` mock handler (replaced by `/bootnodes` — see Phase 2)

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

### Phase 6: KNIRVAGENT → Multi-DVE Concurrent Socket Sessions

**Problem:** KNIRVAGENT listens on TCP port 8081 — one port, one process, one agent. A KNIRV Network deployment may have multiple DVE containers running concurrently, and each DVE needs its own KNIRVAGENT supervisor subprocess. With the current single-agent model, agents block on each other, sessions collide, and a DVE cannot be managed independently.

**Solution:** Adopt a **per-DVE agent subprocess** model. Each DVE gets its own KNIRVAGENT binary instance, listening on a dedicated Unix socket named `agent-{dveId}.sock`. The manager orchestrates N concurrent agent processes, and the gateway routes `/api/agent/{dveId}/*` to the correct socket.

#### Design

```
DVE Container A  ──►  knirvagent (pid 1001)  ──►  agent-dve-a.sock
DVE Container B  ──►  knirvagent (pid 1002)  ──►  agent-dve-b.sock
DVE Container C  ──►  knirvagent (pid 1003)  ──►  agent-dve-c.sock
                              │
                              ▼
                    AgentManager (thread-safe map[DVEID]*AgentProcess)
                              │
                              ▼
                    Gateway: /api/agent/{dveId}/session
                             /api/agent/{dveId}/input
```

#### Socket Layout

```
/var/lib/knirvserver/sockets/
  ├── agent-dve-a.sock    (KNIRVAGENT for DVE "a")
  ├── agent-dve-b.sock    (KNIRVAGENT for DVE "b")
  ├── agent-dve-c.sock    (KNIRVAGENT for DVE "c")
  └── ...                 (one per active DVE)
```

#### Gateway Route Pattern

```
/api/agent/{dveId}/session  ──► agent-{dveId}.sock  ──► POST /session
/api/agent/{dveId}/input    ──► agent-{dveId}.sock  ──► POST /input
/api/agent/{dveId}/output   ──► agent-{dveId}.sock  ──► GET  /output
/api/agent/{dveId}/health   ──► agent-{dveId}.sock  ──► GET  /health
/api/agent/{dveId}/events   ──► agent-{dveId}.sock  ──► WS   /events
/api/agent/{dveId}/*        ──► agent-{dveId}.sock  ──► passthrough
```

**Steps:**

1. **Refactor `packages/KNIRVSERVER/pkg/knirvagent/manager.go` — multi-process orchestration:**

   ```go
   type AgentProcess struct {
       DVEID      string
       Cmd        *exec.Cmd
       SocketPath string
       PID        int
       StartedAt  time.Time
       Healthy    bool
       mu         sync.RWMutex
   }

   type AgentManager struct {
       mu         sync.RWMutex
       agents     map[string]*AgentProcess  // keyed by DVEID
       socketDir  string
       binaryPath string
   }
   ```

   - `StartAgent(dveID string) (*AgentProcess, error)` — spawns a new `bin/knirvagent` subprocess listening on `{socketDir}/agent-{dveID}.sock`. Passes `--dve-id` and `--socket-path` flags to the binary. Blocks until the socket file appears (poll with 100ms interval, 5s timeout) or returns error.
   - `StopAgent(dveID string) error` — signals SIGTERM to the subprocess, waits up to 10s, then SIGKILL. Removes the socket file on cleanup.
   - `GetAgent(dveID string) (*AgentProcess, error)` — returns the agent for a DVE, or an error if not running.
   - `ListAgents() []*AgentProcess` — returns all running agents.
   - `StopAll() error` — stops every agent process in parallel with a 15s global timeout (used during server shutdown).
   - `HealthCheck(dveID string) bool` — pings the agent's socket. Updates `AgentProcess.Healthy`.
   - Add a periodic health-check goroutine (30s interval, configurable) that reaps agents whose socket is gone or process has exited.
   - Maintain a **max concurrent agents** limit (default 32, configurable via `AgentMaxConcurrent`). `StartAgent` returns an error when the limit is reached.

2. **Update `packages/KNIRVSERVER/backend/cmd/backend_server/main.go`:**
   - Initialize `knirvagent.AgentManager` in `NewServer()` with `socketDir` and `binaryPath` from config
   - Wire the agent lifecycle into the DVE management flow:
     - On DVE creation/init → call `agentManager.StartAgent(dveID)`
     - On DVE teardown/stop → call `agentManager.StopAgent(dveID)`
     - Pass the agent manager reference to the API router for health/status endpoints
   - Register `POST /api/v1/admin/dve/{dveId}/agent/start` and `POST /api/v1/admin/dve/{dveId}/agent/stop` as admin-only lifecycle hooks (so administrators can restart a misbehaving agent without restarting the DVE)

3. **Update `packages/KNIRVGATEWAY/internal/server/server.go` — dynamic per-DVE proxying:**

   The single static `agentProxy` won't work because the socket path is dynamic (`agent-{dveId}.sock`). Use a **dynamic reverse proxy** that extracts `{dveId}` from the URL and creates/strips the socket path at runtime:

   ```go
   // DynamicAgentProxy creates an httputil.ReverseProxy that
   // extracts {dveId} from /api/agent/{dveId}/... and proxies
   // to agent-{dveId}.sock, stripping the /api/agent/{dveId} prefix.
   func (s *Server) dynamicAgentProxy() http.Handler {
       return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
           // Extract /api/agent/{dveId}/... — e.g. /api/agent/dve-a/session
           parts := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/api/agent/"), "/", 2)
           if len(parts) < 1 || parts[0] == "" {
               http.Error(w, `{"error":"missing dveId"}`, http.StatusBadRequest)
               return
           }
           dveID := parts[0]

           socketPath := filepath.Join(s.config.SocketDir, fmt.Sprintf("agent-%s.sock", dveID))

           // Check if socket exists
           if _, err := os.Stat(socketPath); os.IsNotExist(err) {
               http.Error(w, fmt.Sprintf(`{"error":"no agent running for DVE %s"}`, dveID),
                   http.StatusServiceUnavailable)
               return
           }

           // Rewrite path: strip /api/agent/{dveId} prefix
           r.URL.Path = "/"
           if len(parts) == 2 {
               r.URL.Path = "/" + parts[1]
           }

           proxy := httputil.NewSingleHostReverseProxy(&url.URL{Scheme: "http", Host: "unix"})
           proxy.Transport = &http.Transport{
               DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
                   return net.DialTimeout("unix", socketPath, 2*time.Second)
               },
           }

           proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
               http.Error(w, fmt.Sprintf(`{"error":"agent %s unavailable: %s"}`, dveID, err),
                   http.StatusBadGateway)
           }

           proxy.ServeHTTP(w, r)
       })
   }
   ```

   Route registrations:

   ```go
   // In setupRoutes():
   r.PathPrefix("/api/agent/").Handler(s.dynamicAgentProxy())
   ```

   The proxy:
   - Returns **503** when the socket doesn't exist (agent not started for that DVE)
   - Returns **502** when the socket exists but the agent doesn't respond (process crashed, socket stale)
   - Returns **400** when `dveId` is missing from the path

4. **In `packages/KNIRVSERVER/backend/internal/config/config.go`:**
   - Add to `GatewayConfig`:
     ```go
     AgentSocketDir     string `mapstructure:"agent_socket_dir"`      // e.g., /var/lib/knirvserver/sockets/
     AgentMaxConcurrent int    `mapstructure:"agent_max_concurrent"`  // default 32
     ```

5. **In the KNIRVAGENT binary (`bin/knirvagent`):**
   - Add `--dve-id` flag (string, e.g. "dve-a") for self-identification
   - Add `--socket-path` flag (string, e.g. "/var/lib/knirvserver/sockets/agent-dve-a.sock") replacing the old `:8081` default
   - Remove the TCP listener default — the binary must listen on the Unix socket exclusively
   - The binary's HTTP handler already supports `/session`, `/input`, `/output` — no handler changes needed, just listener type

6. **Concurrency and lifecycle considerations:**
   - **Thread safety:** All `AgentManager` methods use `sync.RWMutex`. The `agents` map is protected at all times. `GetAgent`/`StopAgent` use `RLock` for reads, `Lock` for mutations.
   - **Startup race:** `StartAgent` must block until the socket is actually listening (poll with 100ms interval, 5s timeout). Without this, the gateway might proxy a request before the agent is ready.
   - **Cleanup on DVE destroy:** `StopAgent` must be idempotent — if the process already exited, just clean up the socket file and remove the map entry.
   - **Socket file leak guard:** Register a deferred cleanup via `process.Process.Kill` + `os.Remove(socketPath)` for edge cases where the process exits unexpectedly. The periodic health-check goroutine also collects orphaned socket files.
   - **Graceful shutdown:** `StopAll()` iterates the agents map, sends SIGTERM to each, waits 10s, then SIGKILL remaining. If `StopAll` takes more than 15s, it returns an error listing the PIDs that didn't respond. The server shutdown sequence calls `StopAll` before closing the main listener.
   - **Resource limits:** If `AgentMaxConcurrent` is reached, `StartAgent` returns a clear error message. Expose current count via `POST /api/v1/admin/agent/stats`.
   - **Logging:** Every agent lifecycle event (start, stop, health failure, reap) emits a structured log line with `dveID`, `pid`, `socketPath`, and duration.

**API Contract (per-agent Unix socket):**

| Method | Path | Description |
|--------|------|-------------|
| POST | `/session` | Create or resume an agent session |
| POST | `/input` | Send user input to the agent |
| GET | `/output` | Get the latest agent output/response |
| GET | `/health` | Agent health check (always responds 200 when running) |
| WS | `/events` | WebSocket stream of agent events (thinking, output, errors) |

The gateway strips the `/api/agent/{dveId}` prefix so the agent binary sees clean paths. For example:
- `POST /api/agent/dve-a/session` → `socket:agent-dve-a.sock` → `POST /session`
- `GET /api/agent/dve-b/health` → `socket:agent-dve-b.sock` → `GET /health`
- `WS /api/agent/dve-c/events` → `socket:agent-dve-c.sock` → `WS /events`

**Files changed:**
- `packages/KNIRVSERVER/pkg/knirvagent/manager.go` (rewrite for multi-process)
- `packages/KNIRVGATEWAY/internal/server/server.go` (dynamic per-DVE proxy)
- `packages/KNIRVSERVER/backend/internal/config/config.go` (new config fields)
- `packages/KNIRVSERVER/backend/cmd/backend_server/main.go` (wire lifecycle into DVE flow)
- `bin/knirvagent` (add `--dve-id` and `--socket-path` flags, remove TCP default)
- `packages/KNIRVSERVER/config/production.yaml` (add agent config section)

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

5. **Fix any data shape mismatches** by editing the WebGUI components to consume the submodule's native format, or adding a thin backend wrapper where the mismatch is large

**Files changed:**
- Multiple WebGUI frontend files

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
    ShellSocketPath string `mapstructure:"shell_socket"`               // NEW
    AgentSocketDir  string `mapstructure:"agent_socket_dir"`              // NEW — directory for per-DVE agent sockets
    AgentMaxConcurrent int `mapstructure:"agent_max_concurrent"`          // NEW — max concurrent agent processes (default 32)
    // Oracle TCP address — NOT a static config field; resolved at runtime.
    // The oracle binary binds to an arbitrary available port and advertises it
    // via a well-known discovery mechanism (e.g., socket file with the port
    // number, a port file in the socket directory, or a caddy-style
    // environment variable propagated from the oracle's startup). The gateway
    // reads the live port at request time. Default fallback: localhost:1317.
}
```

### New fields in gateway's internal config:

```go
type Config struct {
    // ... existing fields ...
    ShellSocketPath string  // NEW
    AgentSocketDir  string  // NEW — directory for per-DVE agent sockets
    AgentMaxConcurrent int  // NEW — max concurrent agent processes (default 32)
}
```

### YAML additions (`config/production.yaml`):

```yaml
gateway:
  backend_socket: /var/lib/knirvserver/sockets/backend.sock
  chain_socket: /var/lib/knirvserver/sockets/chain.sock
  graph_socket: /var/lib/knirvserver/sockets/graph.sock
  shell_socket: /var/lib/knirvserver/sockets/shell.sock       # NEW
  agent_socket_dir: /var/lib/knirvserver/sockets/             # NEW — directory for per-DVE agent sockets
  agent_max_concurrent: 32                                     # NEW — max concurrent agent processes
  # oracle_tcp_address — resolved at runtime, not a static config field.
  # The oracle port is discovered dynamically (see GatewayConfig comment above).
  # Hardcoded fallback: localhost:1317
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
| 6 | `curl http://localhost:8888/api/agent/missing-dve/session` returns 503 (no agent for this DVE) |
| 6 | `curl http://localhost:8888/api/agent/dve-a/health` returns 200 (DVE "dve-a" agent running) |
| 6 | `curl http://localhost:8888/api/agent/dve-b/health` returns 200 (DVE "dve-b" agent running, different socket) |
| 6 | `curl --unix-socket /var/.../agent-dve-a.sock http://localhost/session` returns session OK directly |
| 6 | `curl --unix-socket /var/.../agent-dve-b.sock http://localhost/session` returns session OK (independent session) |
| 6 | Start 33 agents → `POST /api/v1/admin/agent/stats` returns `{"running":32,"max":32}` with error on 33rd |
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
| **KNIRVAGENT on-demand lifecycle** | Per-DVE sockets mean dynamic socket paths — the gateway must resolve `agent-{dveId}.sock` at request time. Returns 503 when no agent running for that DVE. Periodic health-check goroutine reaps orphaned sockets. Max concurrent agents limit prevents resource exhaustion |
| **gRPC→HTTP bridge complexity** | Keep KNIRVHASHER gRPC-only; expose HTTP endpoints in the backend server that internally call gRPC (simpler than adding an HTTP server to KNIRVHASHER) |
| **WebGUI hardcoded URL paths** | Some frontend paths may be baked into the SPA bundle; may need a build step or runtime config override via `NEXT_PUBLIC_BACKEND_URL` — or just edit the WebGUI to call the correct paths |
| **Backward compatibility** | Old flat paths (`/chain`, `/wallet/info`, `/transaction`) need permanent 301 redirects for any external consumers — `/devs` is removed entirely, no redirect needed |
| **Gateway startup ordering** | Gateway must handle missing sockets gracefully (proxy returns 502/503) — sockets may appear after gateway starts |

---

## 8. Apache Camel Integration Patterns

Apache Camel provides a set of enterprise integration patterns (EIPs) for routing, transforming, and mediating messages between endpoints. Below is a plan to implement the most impactful Camel patterns as a lightweight Go library (`pkg/camel`) within the KNIRVGATEWAY, then wire them into the gateway's route definitions.

### 8.1 Overall Approach

The Camel integration lives in a new package `pkg/camel/` inside the KNIRVGATEWAY module. Each pattern is a standalone Go type that implements a common `Processor` interface:

```go
// Processor is the core Camel abstraction — something that processes a message.
type Processor interface {
    Process(ctx context.Context, msg *Message) error
}

// Message wraps the HTTP request/response for routing through the pipeline.
type Message struct {
    Request  *http.Request
    Response http.ResponseWriter
    Headers  http.Header
    Body     []byte
    // Route-scoped exchange properties (not sent to endpoints)
    Properties map[string]interface{}
}

// Endpoint represents a destination the router can send to.
type Endpoint interface {
    URI() string
    CreateProducer() (Producer, error)
}

// Producer sends a message to an endpoint.
type Producer interface {
    Process(ctx context.Context, msg *Message) error
}
```

Routes are built by chaining processors in a pipeline:

```go
route := camel.NewRoute("wallet-info")
    .From("/wallet/info")
    .Filter(header("Authorization", isNotEmpty))
    .Throttle(100)
    .Transform(jsonPath("$.address"))
    .To("socket:///var/lib/knirvserver/sockets/oracle.sock?path=/cosmos/auth/v1beta1/accounts/")
    .CircuitBreaker(5, 30*time.Second)
    .WireTap("/api/v1/audit-log")
```

### 8.2 Pattern Implementations

#### 8.2.1 Content-Based Router

Routes messages to different endpoints based on message content (headers, body, query params).

```go
// ContentBasedRouter evaluates predicates against the message and routes
// to the first matching endpoint.
type ContentBasedRouter struct {
    choices []Choice
    otherwise Endpoint
}

type Choice struct {
    Predicate func(*Message) bool
    Endpoint  Endpoint
}

// DSL usage:
router := camel.NewContentBasedRouter().
    When(hasHeader("X-Role", "Root"), chainEndpoint).
    When(bodyContains("type\":\"nft"), oracleEndpoint).
    Otherwise(backendEndpoint)
```

**Integration into gateway routes:**

```go
r.Handle("/api/proxy", router.Build())
```

#### 8.2.2 Message Transformer

Transforms request/response payloads between formats.

```go
type Transformer func(*Message) error

var camelToSnake Transformer = func(msg *Message) error {
    // Convert JSON camelCase keys to snake_case
    return transformJSON(msg, camelToSnakeCase)
}

// DSL usage:
route := camel.NewRoute("transform-test").
    Transform(camelToSnake).
    To(backendEndpoint)
```

**Built-in transformers to implement:**

| Transformer | Description |
|---|---|
| `jsonPath(path)` | Extract/replace JSON fields using JSONPath expressions |
| `xmlToJson` | Convert XML body to JSON |
| `jsonToXml` | Convert JSON body to XML |
| `setHeader(name, value)` | Add or overwrite an HTTP header |
| `removeHeader(name)` | Strip an HTTP header |
| `bodyTransform(fn)` | Arbitrary body transformation via callback |
| `marshal(toFormat)` | Marshal body to JSON, XML, YAML, or Protobuf |
| `unmarshal(fromFormat)` | Unmarshal body from a format |

**Config file approach (YAML):**

```yaml
transforms:
  - type: jsonPath
    expression: "$.data.balances[0].amount"
    target: "$.balance"
  - type: setHeader
    name: X-Submodule
    value: oracle
  - type: removeHeader
    name: X-Internal-Token
```

#### 8.2.3 Protocol Bridge

Already partially implemented via `newSocketProxy` and `newHTTPProxy`. Generalize into a Camel endpoint:

```go
// Endpoint URI schemes:
//   socket:///path/to/sock?path=/api/endpoint
//   http://host:port/path
//   https://host:port/path
//   direct://route-name  (in-process — call another route synchronously)
//   seda://route-name    (in-process — call another route asynchronously via channel)
//   log://category       (log message and continue)

socketEP, _ := camel.ParseEndpoint("socket:///var/lib/knirvserver/sockets/chain.sock?path=/api/v1/chain/latest")
httpEP, _ := camel.ParseEndpoint("http://localhost:1317/cosmos/auth/v1beta1/accounts/")
```

The `direct` and `seda` endpoints enable route composition — one route can call another route internally:

```go
// Route A: receives WebGUI requests and delegates
camel.NewRoute("webgui-chain").
    From("/chain").
    To("direct://chain-latest")

// Route B: the actual chain call (reusable, called by other routes too)
camel.NewRoute("chain-latest").
    From("direct://chain-latest").
    Transform(addCacheHeaders).
    To("socket:///var/.../chain.sock?path=/api/v1/chain/latest")
```

#### 8.2.4 Circuit Breaker

Prevents cascading failures by stopping traffic to a failing backend after N consecutive failures.

```go
// CircuitBreaker wraps an endpoint. After `threshold` consecutive failures
// within `window`, it opens the circuit and returns a cached error response
// for `cooldown` duration before trying again (half-open).
type CircuitBreaker struct {
    threshold int           // failures before opening (default 5)
    window    time.Duration // sliding window (default 30s)
    cooldown  time.Duration // time before half-open retry (default 10s)
}

// State machine: CLOSED → OPEN → HALF-OPEN → CLOSED
```

**Config:**

```yaml
circuitBreaker:
  threshold: 5
  window: 30s
  cooldown: 10s
  fallbackResponse:
    statusCode: 503
    body: '{"error":"service temporarily unavailable"}'
```

**Integration into gateway:**

```go
socketEP := camel.NewSocketEndpoint(socketPath, upstreamPath)
cb := camel.NewCircuitBreaker(socketEP, camel.CircuitBreakerConfig{
    Threshold: 5,
    Window:    30 * time.Second,
    Cooldown:  10 * time.Second,
})

r.Handle("/api/oracle/wallet/info", cb.Build())
```

#### 8.2.5 Throttler

Rate-limits requests to a backend endpoint.

```go
type Throttler struct {
    maxRequests int           // max requests per time window
    window      time.Duration // time window (default 1s)
    burst       int           // burst size (default = maxRequests)
}
```

**Config:**

```yaml
throttler:
  maxRequests: 100
  window: 1s
  burst: 20
  response:
    statusCode: 429
    body: '{"error":"rate limit exceeded","retryAfter":1}'
```

**Implementation detail:** Use a token bucket algorithm (`golang.org/x/time/rate.Limiter`) per route, keyed by client IP or `X-API-Key` header.

```go
throttler := camel.NewThrottler(100, time.Second)
r.Handle("/api/v1/dve/nodes", throttler.Wrap(backendHandler))
```

#### 8.2.6 Wire Tap

Mirrors a copy of every message to a secondary endpoint (for logging, auditing, metrics).

```go
type WireTap struct {
    target    Endpoint  // where to send the copy
    predicate func(*Message) bool // optional — only tap matching messages
}

// DSL usage:
route := camel.NewRoute("wallet-info").
    WireTap("log://audit.wallet").
    WireTap("seda://metrics-collector").
    To(oracleEndpoint)
```

**Implementation:** The wire tap sends an asynchronous copy to the target. It does NOT block the main flow — the tap runs in a goroutine with a buffered channel (buffer = 1000). If the channel is full, the tap is skipped (non-blocking).

**Default tap targets:**

| URI | Purpose |
|---|---|
| `log://audit.{route}` | Log request/response metadata to the structured logger |
| `seda://metrics` | Route to the metrics collector (request count, latency histogram) |
| `http://localhost:9090/api/v1/audit-log` | Forward to external audit service (when configured) |

#### 8.2.7 Recipient List

Fan-out a single request to multiple backends and optionally aggregate the responses.

```go
type RecipientList struct {
    endpoints  []Endpoint
    strategy   AggregationStrategy // how to combine responses
    parallel   bool                // fan-out in parallel (default: sequential)
}

type AggregationStrategy interface {
    // Aggregate combines multiple responses into one.
    // Called once per recipient, receives accumulated result and new response.
    Aggregate(accumulated *Message, next *Message) (*Message, error)
}
```

**Built-in aggregation strategies:**

| Strategy | Behavior |
|---|---|
| `FirstElement` | Return only the first successful response |
| `LastElement` | Return only the last successful response |
| `CombineJSON` | Merge all JSON responses into a single JSON object |
| `CombineJSONArray` | Concatenate all JSON array responses |
| `Custom(fn)` | User-provided aggregation function |

**Use case — health dashboard:**

```go
route := camel.NewRoute("health-all").
    RecipientList([]Endpoint{
        socketEP("backend.sock", "/health"),
        socketEP("chain.sock", "/health"),
        socketEP("graph.sock", "/health"),
    }, camel.CombineJSON{}).
    To("direct://response")
```

#### 8.2.8 Splitter

Splits a single message into multiple sub-messages, processing each independently.

```go
type Splitter struct {
    expression Expression // how to split (JSONPath, regex, etc.)
}

// Example: Split a JSON array of transaction IDs and process each one
route := camel.NewRoute("batch-tx").
    Split(jsonPath("$.transactions[*]")).
    To(oracleEndpoint)
```

**Implementation:** The splitter iterates over the extracted elements, creates a new `Message` per element, and passes each through the remaining pipeline. Results are collected and combined (or discarded if the caller only needs side effects).

#### 8.2.9 Aggregator

Collects related messages and emits a combined result once a completion condition is met.

```go
type Aggregator struct {
    correlationExpr Expression   // how to group messages (e.g., jsonPath("$.orderId"))
    completionSize  int           // emit after N messages collected
    completionTimeout time.Duration // emit after timeout even if size not reached
    strategy        AggregationStrategy
}
```

**Use case — batch oracle queries:**

```go
// Collect up to 10 wallet balance requests within 5s, then batch-query the oracle
route := camel.NewRoute("batch-balance").
    Aggregator(jsonPath("$.address"), 10, 5*time.Second, combineJSON).
    To(oracleEndpoint)
```

#### 8.2.10 Dead Letter Channel

Captures messages that failed processing and routes them to a dead letter endpoint for later analysis.

```go
type DeadLetterChannel struct {
    deadLetterEndpoint Endpoint
    maxRedeliveries    int
    redeliveryDelay    time.Duration
}
```

**Config:**

```yaml
deadLetterChannel:
  enabled: true
  maxRedeliveries: 3
  redeliveryDelay: 1s
  endpoint: "seda://dead-letter-queue"
```

**Integration:** The dead letter channel wraps an endpoint with retry logic. After `maxRedeliveries` failures, the message is sent to `deadLetterEndpoint` instead of returning an error to the client.

#### 8.2.11 Idempotent Consumer

Deduplicates messages to ensure exactly-once processing.

```go
type IdempotentConsumer struct {
    idExpression Expression   // how to extract the unique message ID
    repository   IdRepository // where to store seen IDs (memory, Redis, etc.)
}

type IdRepository interface {
    Contains(key string) (bool, error)
    Add(key string) error
    Remove(key string) error
}
```

**Use case — transaction submission:**

```go
route := camel.NewRoute("submit-tx").
    IdempotentConsumer(jsonPath("$.tx_hash"), redisRepo).
    To(oracleEndpoint)
```

If the same transaction hash arrives twice, the second request is silently dropped (returns the cached response from the first invocation).

### 8.3 Integration into Gateway Route Setup

The `setupRoutes()` function in `server.go` will wire Camel routes alongside existing gorilla/mux routes. Camel routes are compiled into `http.Handler` adapters:

```go
func (s *Server) setupRoutes() error {
    r := mux.NewRouter()

    // === Camel-managed routes ===

    // Wallet info: throttled, circuit-brokered oracle proxy
    walletRoute := camel.NewRoute("wallet-info").
        Throttle(100, time.Second).
        CircuitBreaker(5, 10*time.Second, 30*time.Second).
        To(socketEndpoint(s.config.OracleSocketPath, "/cosmos/auth/v1beta1/accounts/"))
    r.Handle("/wallet/info", camel.ToHTTPHandler(walletRoute))

    // Chain: content-based routing to different chain endpoints
    chainRouter := camel.NewContentBasedRouter().
        When(hasQueryParam("type", "capability"), socketEndpoint(s.config.ChainSocketPath, "/api/v1/chain/mcp/capability/list")).
        When(hasQueryParam("type", "nft"), socketEndpoint(s.config.ChainSocketPath, "/api/v1/chain/nft/list")).
        Otherwise(socketEndpoint(s.config.ChainSocketPath, "/api/v1/chain/latest"))
    r.Handle("/api/chain", camel.ToHTTPHandler(chainRouter))

    // === Existing gorilla/mux routes remain untouched ===
    r.HandleFunc("/health", s.handleHealth).Methods("GET")
    // ... all existing routes ...
}
```

### 8.4 Configuration File (YAML)

Camel routes can be defined externally in a YAML config file, loaded at gateway startup:

```yaml
# routes.yaml
routes:
  - id: wallet-info
    from: /wallet/info
    steps:
      - throttle:
          maxRequests: 100
          window: 1s
      - transform:
          type: setHeader
          name: X-Target
          value: oracle
      - circuitBreaker:
          threshold: 5
          window: 30s
          cooldown: 10s
      - to:
          uri: socket:///var/lib/knirvserver/sockets/oracle.sock
          path: /cosmos/auth/v1beta1/accounts/

  - id: chain-latest
    from: /api/chain
    steps:
      - to:
          uri: socket:///var/lib/knirvserver/sockets/chain.sock
          path: /api/v1/chain/latest

  - id: health-dashboard
    from: /api/health/all
    steps:
      - recipientList:
          parallel: true
          endpoints:
            - socket:///var/lib/knirvserver/sockets/backend.sock?path=/health
            - socket:///var/lib/knirvserver/sockets/chain.sock?path=/health
            - socket:///var/lib/knirvserver/sockets/graph.sock?path=/health
      - transform:
          type: jsonPath
          expression: "$"
          target: "$.services"
      - to:
          uri: direct://response
```

### 8.5 Package Structure

```
packages/KNIRVGATEWAY/
  pkg/camel/
    route.go              — Route definition, builder DSL
    message.go            — Message type
    processor.go          — Processor interface
    endpoint.go           — Endpoint, Producer interfaces
    content_router.go     — ContentBasedRouter
    transformer.go        — MessageTransformer
    circuit_breaker.go    — CircuitBreaker
    throttler.go          — Throttler
    wire_tap.go           — WireTap
    recipient_list.go     — RecipientList
    splitter.go           — Splitter
    aggregator.go         — Aggregator
    dead_letter.go        — DeadLetterChannel
    idempotent.go         — IdempotentConsumer
    expression.go         — JSONPath, header, body expression evaluators
    http_adapter.go       — ToHTTPHandler adapter (Camel → net/http)
    yaml_loader.go        — Load route definitions from YAML
```

### 8.6 Implementation Priority

| Pattern | Priority | Effort | Reason |
|---------|----------|--------|--------|
| Content-Based Router | P0 | 1 day | Enables intelligent routing based on headers/body — most impactful for the gateway |
| Protocol Bridge (unify socket + HTTP) | P0 | 1 day | Generalize existing `newSocketProxy`/`newHTTPProxy` into URI-based endpoints |
| Circuit Breaker | P0 | 1 day | Prevents cascading failures when a backend goes down |
| Throttler | P1 | 0.5 day | Rate limiting — needed for production deployment |
| Wire Tap | P1 | 0.5 day | Audit logging and metrics collection |
| Message Transformer | P1 | 1 day | JSONPath extraction, header manipulation — needed for WebGUI data shape mismatch |
| Dead Letter Channel | P2 | 0.5 day | Failed message handling for production reliability |
| Recipient List | P2 | 1 day | Fan-out queries — useful for health dashboards |
| Splitter | P3 | 0.5 day | Batch processing — lower immediate need |
| Aggregator | P3 | 1 day | Batch processing — useful for bulk oracle queries |
| Idempotent Consumer | P3 | 0.5 day | Exactly-once semantics — important for transaction submission |
| YAML config loader | P3 | 1 day | Externalized route config — nice-to-have for operations |

### 8.7 Migration Path

1. **Phase A** — Create `pkg/camel/` with core types (Processor, Message, Endpoint, Route)
2. **Phase B** — Implement P0 patterns (content router, protocol bridge, circuit breaker)
3. **Phase C** — Refactor existing `server.go` route setup to use Camel routes for chain/oracle/backend endpoints
4. **Phase D** — Implement P1 patterns (throttler, wire tap, transformer)
5. **Phase E** — Add YAML config loader for externalized route definitions
6. **Phase F** — Implement P2/P3 patterns as needed

---
## 9. Answered Questions

### Q1. KNIRVSHELL `--socket-path` flag support?

**Answer:** Not invasive — the `--socket-path` flag can be added to the KNIRVSHELL binary as needed. The existing binary already parses command-line arguments; adding a `--socket-path` flag to switch its listener from TCP to Unix socket requires minimal changes.

### Q2. WebGUI frontend path configuration — build-time vs runtime?

**Answer:** The WebGUI is rebuilt via `npm run build` during the full project rebuild, producing a Next.js static export. In addition, paths are runtime-configurable via the `NEXT_PUBLIC_BACKEND_URL` environment variable, which is used by all Next.js API route files (`pages/api/objects.ts`, `pages/api/blocks.ts`, `pages/api/assets.ts`, `pages/api/view/[id].ts`). The current default is `'/api/v1'` — pointing to the gateway's own proxy.

**Implication for Phase 9:** Paths can be changed at build time (static export) or at runtime (if the Next.js API routes are served from a Node server). The static export via `next export` does NOT serve the API routes — those only work in a Node runtime. In static mode (current setup), the Go gateway's `handler.go` routes handle the API calls instead, proving the value of the Go handlers.

### Q3. WebGUI data shapes for mock endpoints

**Answer:** Analysis of the WebGUI frontend (in `packages/KNIRVGATEWAY/internal/embedded/webgui/src/`) reveals the following expected data shapes. For endpoints where the submodule data shape doesn't match, prefer editing the WebGUI frontend (or the backend response) to align shapes — no separate transform layer is needed.

#### `/api/objects` (frontend: `nft-property-explorer.js` + `pages/api/objects.ts`)

Used by the NFT Property Explorer page. Expected shape:

```json
[
  {
    "id": "mock-1",
    "name": "Mock Cube",
    "src": "",
    "object_type": "glb"
  },
  {
    "id": "mock-2",
    "name": "Mock Sphere",
    "object_type": "glb"
  }
]
```

- Frontend accesses `objectsData.length` (must be an array)
- Displaying `obj.id`, `obj.name`, `obj.object_type` in the object list
- Selecting an object sets `selectedObject` and calls `Viewport` with `modelId={selectedObject.id}`
- **Target submodule:** KNIRVCHAIN (`/api/chain/mcp/capability/list`)
- **Shape alignment:** Edit `nft-property-explorer.js` to consume chain-native capability shape (map chain fields to expected display fields)

#### `/api/transactions` (frontend: `nft-property-explorer.js`)

Expected shape:

```json
[
  {
    "id": "tx-1",
    "operation": "upload",
    "asset": "0x1234567890abcdef...",
    "metadata": "",
    "timestamp": "2024-01-15T10:30:00Z"
  }
]
```

- Frontend accesses `tx.id.substring(0, 8)`, `tx.operation`, `tx.asset.substring(0, 6)`, `new Date(tx.timestamp)`
- **Target submodule:** KNIRVORACLE (`/api/oracle/v3/economics/metrics`)
- **Shape alignment:** Edit `nft-property-explorer.js` to consume Cosmos SDK tx format, or add a thin backend endpoint at `/api/v1/transactions` that maps Cosmos fields

#### `/api/blocks` (frontend: `nft-property-explorer.js`)

Expected shape:

```json
[
  {
    "index": 0,
    "id": "block-1234",
    "data": "Transaction data",
    "timestamp": "2024-01-15T10:30:00Z",
    "prev_hash": "0xabcd...",
    "hash": "0x1234..."
  }
]
```

- Frontend accesses `block.index`, `block.id.substring(0, 8)`, `block.hash.substring(0, 6)`, `new Date(block.timestamp)`
- **Target submodule:** KNIRVORACLE (`/api/oracle/v3/consensus/status`)
- **Shape alignment:** Edit `nft-property-explorer.js` to consume Cosmos block format directly, or add thin backend endpoint

#### `/api/assets` (frontend: `pages/api/assets.ts` + Go `handler.go`)

The Next.js API route at `pages/api/assets.ts` proxies to `BACKEND_URL + /api/assets`. The Go handler in `handler.go` also proxies to the backend. However, no frontend component directly renders assets data — the route exists as a passthrough. The data shape is unconstrained (whatever the backend returns). For the gateway proxy, route to KNIRVCHAIN badge/NFT data.

- **Target submodule:** KNIRVCHAIN (`/api/chain/nft/list`)
- **Shape alignment:** Minimal — no frontend UI directly consumes the shape, mainly needed for API consistency

#### `/api/view/{id}` (frontend: `Viewport.tsx` + `pages/api/view/[id].ts`)

Returns a complete HTML page with Three.js 3D viewer embedded in an iframe. The backend is queried for object details:

```json
{
  "id": "mock-1",
  "name": "Mock Cube",
  "file_path": "/assets/models/cube.gltf",
  "object_type": "glb",
  "asset_type": "glb",
  "data": {
    "author": "Unknown",
    "license": "Unknown"
  },
  "created_at": "2024-01-15T10:30:00Z",
  "transaction": "0x1234..."
}
```

- Frontend uses: `objectData.name`, `objectData.file_path` (model URL), `objectData.object_type || objectData.asset_type`, `objectData.data?.author`, `objectData.data?.license`, `objectData.created_at`, `objectData.transaction`
- Falls back to `/assets/models/cube.gltf` for the model if `file_path` is empty
- **Target submodule:** KNIRVCHAIN (single NFT object detail at `/api/chain/nft/{id}`)
- **Shape alignment:** Edit `Viewport.tsx` and the viewer HTML template in `handler.go` to consume chain-native NFT shape. Falls back to `/assets/models/cube.gltf` if no model URL

#### Summary of Shape Alignment

| Gateway Route | Submodule Source | Alignment Effort | Notes |
|---------------|----------------|-----------------|-------|
| `/api/objects` | KNIRVCHAIN `/api/chain/mcp/capability/list` | Medium | Edit WebGUI to consume chain-native capability shape |
| `/api/transactions` | KNIRVORACLE Cosmos SDK | High | Edit WebGUI or add thin backend wrapper |
| `/api/blocks` | KNIRVORACLE `/api/oracle/v3/consensus/status` | Medium | Edit WebGUI or add thin backend wrapper |
| `/api/assets` | KNIRVCHAIN `/api/chain/nft/list` | Low | Passthrough — no frontend UI directly consumes shape |
| `/api/view/{id}` | KNIRVCHAIN `/api/chain/nft/{id}` | Medium | Edit WebGUI viewer component to use chain-native NFT shape |

**Recommendation:** For each endpoint where the submodule data shape doesn't match what the WebGUI expects, prefer editing the WebGUI frontend to consume the submodule's native shape. If the mismatch is too large, edit the backend to return a friendlier shape. Avoid a separate transform layer — keep shape alignment in the component or handler that owns the data.

### Q4. Oracle TCP address configurable?

**Answer:** Yes, but not as a static config field. The oracle TCP address is resolved at runtime — the oracle binary binds to an arbitrary available port and advertises it via a well-known discovery mechanism. Two approaches:

1. **Port file:** The oracle writes its active port to a well-known file on startup (e.g., `/var/lib/knirvserver/sockets/oracle.port`). The gateway reads this file at request time (with in-memory caching + TTL).
2. **Environment variable:** The oracle process propagates its port via `ORACLE_PORT` (set at startup) and the gateway reads it from the environment.

The gateway falls back to `localhost:1317` when no port file or env var is present. The default value `localhost:1317` remains unchanged.

### Q5. External consumers depending on mock data?

**Answer:** No. There are no external consumers depending on the mock data served at `/api/objects`, `/api/transactions`, etc. These endpoints are consumed only by the WebGUI frontend, which will be updated to consume real data through gateway proxies.

### Q6. `/devs` endpoint — removed, replaced with `/bootnodes`

**Answer:** The `/devs` endpoint is no longer needed and there is no backward-compatibility requirement. Remove it entirely. Replace it with a `/bootnodes` endpoint that returns active bootstrap peers on the KNIRV Network.

The WebGUI Peers page (`peers.js`) already fetches `/mcp/dev/list` and expects:

```json
{
  "devs": [
    {
      "id": "peer-1",
      "name": "Bootstrap Node 1",
      "is_bootnode": true,
      "connected": true,
      "address": "/ip4/10.0.0.1/tcp/26656",
      "last_seen": "2024-01-15T10:30:00Z",
      "capabilities": ["mining", "validation"]
    }
  ]
}
```

**Action:**
- **Gateway:** Remove the `/devs` mock handler and route. Add `/bootnodes` → 301 → `/api/chain/bootnodes` (KNIRVCHAIN peer info).
- **KNIRVCHAIN:** Implement `/bootnodes` as a new endpoint returning the active bootstrap peer list in the shape above.
- **Frontend (`BlockchainContext.js`):** Change the old `api.get('/devs')` call (line 73) to `api.get('/bootnodes')` — or simply remove it since the Peers page (`peers.js`) already fetches from `/mcp/dev/list` directly.
- **No redirect needed:** `/devs` can safely 404 — no backward compatibility required.
