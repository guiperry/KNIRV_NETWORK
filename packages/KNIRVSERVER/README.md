# KNIRV-SERVER: Deterministic Validation & Active Memory Fabric

KNIRV-SERVER is the central orchestration and memory layer of the KNIRV Network. It provides a high-performance, secure, and semantic environment for AI agents to operate within a **Deterministic Validation Environment (DVE)**. By merging five specialized Go submodules, SERVER delivers a unified "Markdown Fabric" for reasoning persistence and verifiable solution execution.

## 🏗️ Technical Architecture

KNIRV-SERVER leverages a hybrid architecture combining high-throughput memory streaming with post-quantum secured persistence.

### 1. The Markdown Fabric (Active Memory Layer)
SERVER transforms all intelligence artifacts into human-readable, machine-executable **Markdown (.md)** files.
*   **Reasoning Traces (KNIRVGRAPH)**: Stored as `.md` context records, documenting the agent's thought process through Network Resolution Vectors (NRV).
*   **Solution Vault (KNIRVCHAIN)**: Stores ErrorNodes and SolutionNodes. Solutions contain executable code blocks (interpreted Go/Shell) secured by PQC signatures.
*   **Persistence (KNIRVBASE)**: All `.md` nodes are transparently encrypted at rest using **Kyber-768** and signed with **Dilithium-3**.

### 2. Living Memory Projection (Apache Arrow Flight)
While data is persisted as Markdown, it is projected in real-time into the **Memory Fabric** using **Apache Arrow**.
*   **Sub-millisecond Streaming**: Context records are saved into Arrow buffers for immediate delivery to agents.
*   **Binary Portability**: Cross-language memory access for agents written in Python, Rust, or Go.
*   **Tick Data Streaming**: Phase 6 implements high-performance financial tick data streaming with Arrow IPC format.

### 3. Runtime Security & Verification (eBPF Guardian)
SERVER implements a hardware-assisted security model to perform **Key Neural Intelligence Reasoning Validation**.
*   **Syscall Monitoring**: Uses eBPF to trace every system call made by an agent process.
*   **Intent Correlation**: Correlates the agent's stated "Intent" (from its reasoning trace) with its "Observed Action" (from eBPF).
*   **Virtual Containers**: Provides isolated namespaces and cgroup-based resource limits for agent execution.

### 4. Secure Transport (KNIRVROUTER)
Integrated **TURN Server** logic facilitates reliable P2P synchronization of the Markdown Fabric across restrictive NAT environments.

### 5. Unified Memory System (Consolidated Architecture)
The memory system has been consolidated from 5 separate implementations into a single `UnifiedMemorySystem`:

| Component | Description |
|-----------|-------------|
| `services/memory/service.go` | Main orchestrator merging ActiveMemory, KnowledgeBase, and Arrow Flight |
| `services/memory/ontology.go` | In-memory knowledge graph with entity/relation tracking |
| `services/memory/reasoning.go` | Reasoning trace generation and NRV processing |
| `services/memory/vault.go` | ErrorNode/SolutionNode management with PQC encryption |
| `services/memory/graphrag.go` | GraphRAG client with FFI bridge to Rust graphrag-rs |
| `services/memory/flight.go` | Arrow Flight server for real-time memory streaming |
| `services/memory/handlers.go` | HTTP handlers for unified `/api/v1/memory/` endpoints |
| `storage/memory_store.go` | Unified storage interface across Markdown, GraphRAG, and Ontology |

**Key Features:**
- **Single API prefix**: All memory operations now use `/api/v1/memory/` (legacy endpoints preserved with redirects)
- **Cross-backend queries**: Query GraphRAG, Ontology, or both simultaneously
- **Auto-sync**: Markdown saves automatically trigger GraphRAG reindexing
- **PQC encryption**: Consistent post-quantum encryption across all persistent storage
- **Feature flags**: Enable/disable backends (markdown, graphrag, ontology) via `MemoryConfig`

**Usage Example:**
```go
cfg := &memory.MemoryConfig{
    EnabledBackends: []string{"markdown", "graphrag", "ontology"},
    PQCEncryption:    true,
    ArrowStreaming:   true,
    SyncInterval:     5 * time.Minute,
    EnableAutoSync:   true,
}
system, _ := memory.NewUnifiedMemorySystem(cfg, logger)
system.StoreInteraction(ctx, "agent-1", "error desc", "solution code")
```

---

## 🚀 FinTech Validator: Financial AI Agent Validation

KNIRV-SERVER includes a comprehensive **FinTech Validator** for deterministic validation of financial AI agents. This system transforms AI failures into collective knowledge through interconnected sovereign layers.

### Phase 1: Evidence Packs & Financial Ontologies
- **Evidence Packs**: Immutable audit trails capturing validation results, compliance checks, and execution traces
- **Financial Ontologies**: Regulatory frameworks (AML, KYC, SEC, Basel III) as machine-readable rules
- **PQC Signing**: All evidence packs signed with post-quantum cryptography (ML-DSA-65)

### Phase 2: Scenario Injection & Compliance Engine
- **Regulatory Scenarios**: Pre-built test scenarios for regulatory compliance testing
- **Compliance Engine**: Automated validation against financial regulations
- **Certificate of Correctness**: PQC-signed certificates attesting to agent validation results

### Phase 3: Deterministic Replay (eBPF Trajectory Capture)
- **eBPF Trajectory Capture**: Syscall-level execution tracing of financial AI agents
- **Execution Replay**: Deterministic replay of captured trajectories for verification
- **Trajectory Comparison**: Compare execution runs to detect non-deterministic behavior

### Phase 4: NRV Financial Semantic Engine & Fidelity Scoring
- **NRV (Network Resolution Vector) Traces**: Semantic reasoning traces capturing agent decision-making
- **Fidelity Scoring**: Quantify how closely agent reasoning matches expected financial logic
- **Semantic Distance**: Calculate regulatory alignment distance
- **Risk Detection**: KYC bypass detection, position limit violation monitoring

### Phase 5: Financial Compliance Dashboard (Frontend)
- **React/TypeScript Dashboard**: Real-time visualization of validation results
- **Fidelity Score Display**: Risk assessment with confidence indicators
- **Compliance Status**: Multi-regulation compliance tracking (AML, KYC, SEC, Basel)
- **Trajectory Visualization**: Execution path visualization
- **Evidence Pack Management**: View, export, and verify evidence bundles

### Phase 6: Arrow Flight Tick Data Streaming
- **Apache Arrow IPC**: High-performance binary data format for financial tick data
- **Tick Data Streams**: Real-time streaming of trade/quote data with buffering
- **Bar Data Aggregation**: OHLCV bar generation from tick streams
- **Query & Filter**: Time-range, price-range, and volume-based filtering
- **Export**: Export tick data as Arrow IPC files

---

## 🤖 Agentic Runtime: oh-my-pi Integration

KNIRV-SERVER transforms each DVE into an autonomous, intelligent workspace via the **oh-my-pi** agentic runtime. This provides every DVE with a "batteries-included" capability for coding, research, and validation.

### Architecture

- **`ObjectTypeAgent`** — New runtime object type routed via the Unified Container Manager
- **Container Image**: `knirv-agent-oh-my-pi:latest` — includes oh-my-pi Rust engine, Python/IPython kernels, LSP servers for 40+ languages, and a headless browser
- **Viewport Proxy** — Real HTTP/WebSocket reverse proxy tunneling the agent terminal into the SERVER dashboard
- **eBPF Security Profile** — 70-syscall allowlist specific to agent tooling (git, python, curl, browser) enforcing DVE isolation
- **Active Memory Integration** — Agent task results written as Markdown to the DVE's persistent Markdown Fabric via `ActiveMemoryService.RecordInteraction`
- **Cognitive Engine Integration** — Real-time resource usage reporting for adaptive allocation

### P2P Capability Advertising

The agentic runtime supports P2P discovery through capability advertising:

- **`CapabilityAgenticRuntime`** — Advertises agentic runtime support with version info, supported tools, and configuration
- **`CapabilityDVERouting`** — Advertises DVE routing capability
- **`CapabilityTEEAttestation`** — Advertises TEE attestation support

Nodes can discover agentic runtime nodes using `FindAgenticRuntimeNodes()`.

### Security Implementation

The `oh-my-pi` agent runtime includes specialized eBPF security profiles:

| Component | Description |
|----------|-------------|
| `OhMyPiAgentSyscalls` | 70+ syscalls including clone, execve, socket, connect for tool execution |
| `OhMyPiAgentPaths` | Allowed filesystem paths: /workspace, /tmp, /usr, /lib, /bin |
| `OhMyPiAgentNetworks` | Allowed networks: 127.0.0.0/8, 10.0.0.0/8, 172.16.0.0/12 |
| `NewOhMyPiAgentPolicyConfig()` | Creates eBPF `AgentPolicyConfig` with 8GB memory, 80% CPU limits |

### Agent Command Center (Frontend)

Accessible from the DVE dashboard, the Agent Command Center provides:
- Agent launch/stop controls with live status badge
- Viewport iframe for direct terminal interaction
- Task submission form (type: research / coding / validation / analysis; priority: low → critical)
- Real-time task list with expandable detail panels
- Task statistics (total, pending, completed, failed)

### Agent API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/dve/{id}/agent/status` | GET | Agent running status + task counts |
| `/api/dve/{id}/agent/launch` | POST | Provision oh-my-pi container for DVE |
| `/api/dve/{id}/agent` | DELETE | Stop and destroy agent container |
| `/api/dve/{id}/agent/tasks` | GET | List all tasks for DVE |
| `/api/dve/{id}/agent/tasks` | POST | Submit a new agent task |
| `/api/dve/{id}/agent/tasks/{taskID}` | GET | Get single task with Markdown output |

---

## 🧠 Cognitive Engine — Enhanced Architecture

The `backend/internal/services/cognitiveengine` package has been substantially upgraded with five new production subsystems. All features are wired automatically during server initialization in `main.go`.

### I. Configurable Background Operation (`config.go`)

All loop intervals are now controlled via `EngineConfig` (previously hardcoded):

| Parameter | Default | Description |
|-----------|---------|-------------|
| `LearningInterval` | 30 s | How often the main learning cycle runs |
| `MetricsInterval` | 60 s | How often node metrics are collected |
| `PatternAnalysisInterval` | 5 m | How often failure patterns are analysed |
| `WorkerPoolSize` | 4 | Number of concurrent validation-result workers |
| `TaskQueueCapacity` | 256 | Buffered work queue depth |
| `GuardrailCheckInterval` | 10 s | How often DVE policies are evaluated |
| `EBPFTelemetryInterval` | 15 s | eBPF process_telemetry polling interval |
| `OntologyUpdateInterval` | 2 m | How often the KNIRVGRAPH hypergraph is synced |

The engine also reacts to *events* immediately — a burst of validation failures or a resource-pressure alert triggers an out-of-band learning cycle without waiting for the next ticker tick.

### II. Worker Pool for Concurrent Ingestion (`task_worker.go`)

Validation results are now processed by a **goroutine pool** rather than inline. This decouples result ingestion from learning-state mutation:

```
ValidationResult → ProcessValidationResult() → EventBus (EventValidationResult)
                                              → TaskWorkerPool.Submit()
                                                   └─ worker.processWorkItem() → updateTaskMetrics / updateNodeMetrics / ...
```

Back-pressure: when the queue is full the engine falls back to synchronous processing so no result is ever dropped.

### III. Real eBPF Resource Telemetry (`resource_telemetry.go`, `ebpf_bridge.go`)

`ResourceTelemetryCollector` reads the kernel-level `process_telemetry` eBPF map (CPU time, memory, net I/O, context switches, page faults) via `ebpf.Manager.GetProcessMetrics()`, merging it with Go runtime stats. This data replaces the previous task-rate heuristic in `calculateResourceUtilization`.

`EBPFBridge` wraps the collector with:
- **Telemetry polling loop** — fires `EventResourcePressure` when CPU > 85 % or memory > 85 %
- **Security feedback ingestion** — `InjectEBPFSecurityFeedback()` accepts pre-parsed LSM/XDP events and reduces a node's success rate accordingly, triggering immediate re-learning
- **Panic isolation switch** — `TriggerPanicIsolation(nodeID)` records kernel-level isolation and stops task routing to the compromised node

```go
// Wire from DVEManager's LSM audit parser:
cognitiveEngine.InjectEBPFSecurityFeedback(cognitiveengine.SecurityEventFeedback{
    NodeID: nodeID, EventType: "lsm_block", Severity: "critical", ...
})
```

### IV. Per-DVE Guardrail Policy Enforcement (`guardrail_engine.go`)

`GuardrailEngine` evaluates configurable `PolicyRule` objects against every node's live metrics every 10 seconds. Built-in rules:

| Rule ID | Metric | Condition | Severity | Remediation |
|---------|--------|-----------|----------|-------------|
| `dveguard_low_success` | `success_rate` | < 0.4 | critical | quarantine node |
| `dveguard_slow_response` | `avg_processing_time` | > 300 s | warning | redistribute tasks |
| `dveguard_high_resource` | `resource_utilization` | > 0.95 | critical | scale resources |
| `dveguard_panic_trigger` | `violation_count` | > 5 | panic | kernel isolation |

Custom rules and remediators can be registered at runtime:

```go
ce.guardrailEngine.AddPolicy(&cognitiveengine.PolicyRule{...})
ce.guardrailEngine.RegisterRemediator("my_action", func(ctx, v) error { ... })
```

**Feedback loop**: after each evaluation cycle `RefinePolicy()` auto-adjusts thresholds when trigger rate exceeds 50%, preventing alert fatigue while tightening governance over time.

### V. DVE Ontology & KNIRVGRAPH Integration (`ontology.go`)

`DVEOntologyManager` converts the engine's `LearningState` into typed knowledge-graph entities (`dve_node`, `validation_task`, `adaptation_event`, `failure_pattern`, …) and pushes them into the KNIRVGRAPH `TemporalHypergraph` as `IntentionalSignal` objects every 2 minutes.

This enables graph-based reasoning queries such as:
- Which nodes are consistently associated with failure patterns of type `timeout`?
- Which adaptation events correlate with improved success rates on `inference` tasks?

The hypergraph is wired automatically when ICME is enabled:

```go
// Happens automatically in main.go when ICME is enabled:
cognitiveEngine.SetKNIRVGRAPHEngine(graphEngine, logger)
```

### ICME Embedder Configuration (Text Embedder Thresholds)

The ICME (Intentional Context Memory Engine) uses `github.com/guiperry/text-embedder` for deterministic, local embedding generation. All similarity thresholds are configurable via config file or environment variables.

#### Config File (`config.yaml`)

```yaml
icme:
  enabled: true
  duplicate_threshold: 0.97           # FAISS - reject vectors above this similarity
  node_similarity_threshold: 0.90       # Hypergraph - merge nodes above this
  search_cache_threshold: 0.97           # Hybrid search cache hit threshold
  search_cache_ttl_seconds: 30           # Hybrid search cache TTL
  prompt_cache_threshold: 0.97             # Inference prompt cache threshold

validation:
  semantic_keyword_threshold: 0.65       # SemanticKeywordValidator threshold

fintech:
  intent_threshold: 0.65                 # Intent distance similarity threshold
```

#### Environment Variables (Precedence Over Config File)

| Parameter | Env Variable | Default |
|-----------|------------|---------|
| FAISS duplicate threshold | `KNIRV_FAISS_DUPLICATE_THRESHOLD` | 0.97 |
| Node similarity threshold | `KNIRV_HYPERGRAPH_NODE_THRESHOLD` | 0.90 |
| Search cache threshold | `KNIRV_SEARCH_CACHE_THRESHOLD` | 0.97 |
| Search cache TTL | `KNIRV_SEARCH_CACHE_TTL_SECONDS` | 30 |
| Prompt cache threshold | `KNIRV_PROMPT_CACHE_THRESHOLD` | 0.97 |
| Semantic keyword | `KNIRV_SEMANTIC_KEYWORD_THRESHOLD` | 0.65 |
| Intent similarity | `KNIRV_INTENT_SIMILARITY_THRESHOLD` | 0.65 |

#### Threshold Guidelines

| Use Case | Recommended | Rationale |
|---------|------------|---------|
| Duplicate vector rejection | 0.97 | Only reject near-identical content |
| Semantic node merge | 0.90 | Allow surface variation, catch true duplicates |
| Prompt/search cache | 0.97 | Must not conflate distinct queries |
| Semantic keyword validation | 0.65 | Concept presence, not exact identity |

### Cognitive Engine API additions

| Method | Description |
|--------|-------------|
| `GetGuardrailViolations(limit)` | Recent policy violations with remediation status |
| `GetOntologyStats()` | Count of entities and relations currently indexed |
| `GetLatestTelemetry()` | Most recent eBPF/runtime resource snapshot |
| `InjectEBPFSecurityFeedback(event)` | Feed kernel security event into learning loop |
| `SetEBPFManager(mgr)` | Wire eBPF manager (called from `main.go`) |
| `SetKNIRVGRAPHEngine(hg, logger)` | Wire KNIRVGRAPH hypergraph (called from `main.go`) |

---

## 🧬 Badge-to-WASM Mapping (WASM Integration)

KNIRV-SERVER implements the Badge-to-WASM Mapping system described in `WASM_Integration_Investigation.md`. This system links DVE Badge ontology tags to pre-compiled `.wasm` modules generated by the **KNIRVHASHER** transformer pipeline (TinyGo → WASM), enabling deterministic guardrail enforcement and automated error resolution.

### Architecture

```
Badge NFT (ontology tags)
    │
    ▼
BadgeWASMMapper ────► rules.wasm ────► GuardrailMiddleware (ViewportProxy)
    │                    │                    │
    │                    │                    └─ GuardrailClass() uint32
    │                    │
    │                    └── resolution.wasm ──► ResolutionService
    │                                             │
    │                                             ├─ resolveError() bool
    │                                             └─ ErrorClass() uint32
    │                                                    │
    │                                                    ▼
    │                                         GuardrailEngine remediation
    │
    └── eBPF EventCollector ───► ResolutionSignal ───► ResolutionService.Submit()
```

### Components

| Package | File | Description |
|---------|------|-------------|
| `services/wasm/badge_wasm_mapper.go` | Core mapper | Badge ontology tag → WASM type → .wasm path mapping with persistence |
| `services/wasm/wazero_gate.go` | WASM runtime | Wazero-based runtime for loading and executing .wasm binaries; includes WazeroPool |
| `services/wasm/resolution_service.go` | Resolution worker | eBPF-triggered error resolution via resolution.wasm with async signal processing |
| `services/wasm/guardrail_middleware.go` | Guardrail middleware | HTTP middleware that validates requests via rules.wasm GuardrailClass() |

### WASM Types & Expected Exports

| WASM Type | Tag Classification | Export | Returns | Use Case |
|-----------|-------------------|--------|---------|----------|
| `rule` | `guardrail`, `policy`, `compliance`, `lsm`, `seccomp` | `GuardrailClass()` | `uint32` | Policy validation at the viewport proxy |
| `resolution` | `resolution`, `recovery`, `error`, `heal`, `restart` | `resolveError()` | `bool` | Automated error recovery from eBPF signals |
| `resolution` | (same) | `ErrorClass()` | `uint32` | Error classification for observability |
| `patch` | `patch`, `update`, `hotfix`, `migrate` | `applyPatch()` | `bool` | Future patch application |

### Integration Points

1. **GuardrailEngine**: `InjectBadgeRules()` now also calls `BadgeWASMMapper.RegisterBadge()` when a WASM mapper is configured via `SetWASMMapper()`.

2. **ViewportProxy**: `SetGuardrailValidator()` + `SetGuardrailContext()` enable WASM guardrail checks before proxying. `HandleHTTP()` invokes `GuardrailClass()` and returns 403 if the class is non-zero.

3. **eBPF EventCollector**: `SubscribeResolution()` registers a `ResolutionSignalHandler`. Events matching resolution badge mappings are forwarded to `ResolutionService.Submit()`.

4. **ResolutionService**: Background workers consume `ResolutionSignal` entries, invoke `resolveError()` and `ErrorClass()` via WazeroPool, and dispatch results to the `RemediationSink`.

### Usage

```go
// Create the mapper, pointing at KNIRVHASHER's WASM output directory.
wasmMapper := wasm.NewBadgeWASMMapper(
    wasm.DefaultWASMDir(),            // /var/heart/wasm or $KNIRV_WASM_OUT_DIR
    "/var/lib/knirvserver/wasm-mappings.json",
)

// Wire into the GuardrailEngine so InjectBadgeRules also registers WASM.
guardrailEngine.SetWASMMapper(wasmMapper)

// Create guardrail middleware for the viewport proxy.
guardrailMW := wasm.NewGuardrailMiddleware(wasmMapper, 8)
viewportProxy.SetGuardrailValidator(guardrailMW)
viewportProxy.SetGuardrailContext(dveID, badgeID)

// Create the resolution service and wire into eBPF.
resSvc := wasm.NewResolutionService(wasmMapper, remediationSink, 4, 256)
resSvc.Start(ctx, 2)
eventCollector.SubscribeResolution(func(event *ebpf.SyscallEvent, badgeID, tag string) {
    resSvc.Submit(&wasm.ResolutionSignal{
        DVEID: dveID, NodeID: nodeID, BadgeID: badgeID,
        Tag: tag, SyscallID: event.SyscallID, PID: event.PID,
    })
})
```

### Tag Classification Patterns

| Pattern | WASM Classification |
|---------|-------------------|
| `guardrail:*`, `policy:*`, `compliance:*`, `lsm:*`, `seccomp:*`, `ontology:guard:*` | `rule` |
| `resolution:*`, `resolve:*`, `recovery:*`, `error:*`, `remediation:*`, `heal:*`, `restart:*`, `rollback:*`, `ontology:resolve:*` | `resolution` |
| `patch:*`, `update:*`, `hotfix:*`, `migrate:*` | `patch` |
| Anything else | `rule` (fallback) |

### WASM Binary Discovery

The mapper discovers `.wasm` files in two phases:
1. **Type-prefixed names**: `rule_*.wasm`, `resolution_*.wasm` (intermediate names from KNIRVHASHER).
2. **Hash-named files**: `<sha256hex>.wasm` (final names after verification). The most recently modified candidate is used.

Call `RefreshWASMDir()` after KNIRVHASHER generates new WASM to update all mappings.

---

## 🏗️ Codebase Unification

The runtime and node-management layers have been consolidated for a stable foundation:

- **`internal/runtime/provisioning.go`** — `PortAllocator` and `SSHProvisioner` now live in the `runtime` package; `UnifiedContainerManagerWithProvisioning` wraps them into an atomic `ProvisionAndCreate` call
- **`internal/services/dvemanager/node_lifecycle.go`** — `DVEManager` gains `RegisterDVENode`, `Heartbeat`, and `GetNodeSession` methods, centralising node lifecycle without a dependency on the `dvecreation` package
- **`internal/services/p2p/capability_advertising.go`** — P2P capability advertising for agentic runtime support
- **`internal/runtime/agent_security.go`** — eBPF security profiles specific to oh-my-pi agent tooling
- All HTTP handlers are registered through `internal/web` following the single-package handler pattern

### Unified Container Manager Extensions

The `UnifiedContainerManager` now includes:

| Method | Description |
|--------|-------------|
| `SetCognitiveEngine(engine)` | Wire Cognitive Engine for agent resource reporting |
| `CreateAgentRuntimeCapability(nodeID)` | Create P2P capability advertisement |
| `GetAgentPolicy(containerID)` | Get security policy for agent container |
| `UpdateAgentResourceUsage(dveID, cpu, memory)` | Report resource usage to Cognitive Engine |
| `initializeOhMyPiAgentPolicy(container)` | Initialize oh-my-pi specific security profile |

---

## 🛠️ Technical Stack

-   **Backend**: Go 1.24+ (Strict concurrency, ported PQC primitives).
-   **Frontend**: Next.js 15 (App Router, Tailwind CSS 4, shadcn/ui).
-   **Security**: eBPF (CO-RE), Linux Security Modules (LSM), TEE Support (SGX/TDX).
-   **Protocols**: Model Context Protocol (MCP), Arrow Flight, TURN (STUN).
-   **Storage**: BuntDB (Metadata) + Markdown Fabric (Encrypted Content).
-   **Data Format**: Apache Arrow IPC for high-performance streaming.

---

## 📡 API Endpoints

### FinTech Validator API

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/fintech/status` | GET | Service status |
| `/api/fintech/validate` | POST | Run comprehensive validation |
| `/api/fintech/evidence` | GET | List evidence packs |
| `/api/fintech/evidence/{id}` | GET | Get evidence pack |
| `/api/fintech/evidence/{id}/export` | GET | Export evidence bundle |
| `/api/fintech/ontologies` | GET | List financial ontologies |
| `/api/fintech/scenarios` | GET | List regulatory scenarios |
| `/api/fintech/scenarios/validate` | POST | Run scenario validation |
| `/api/fintech/certificates/issue` | POST | Issue certificate of correctness |
| `/api/fintech/trajectories` | GET | List execution trajectories |
| `/api/fintech/trajectories/{id}/replay` | POST | Replay trajectory |
| `/api/fintech/nrv/traces` | GET | List NRV reasoning traces |
| `/api/fintech/nrv/traces/{id}/score` | POST | Calculate fidelity score |
| `/api/fintech/nrv/traces/{id}/distance` | POST | Calculate semantic distance |
| `/api/fintech/nrv/traces/{id}/detect/kyc-bypass` | GET | Detect KYC bypass |
| `/api/fintech/ticks/streams` | POST | Create tick data stream |
| `/api/fintech/ticks/{symbol}` | GET | Get ticks for symbol |
| `/api/fintech/ticks/{symbol}/add` | POST | Add tick to stream |
| `/api/fintech/ticks/query` | POST | Query ticks with filters |
| `/api/fintech/ticks/export` | GET | Export ticks as Arrow IPC |

### Model Context Protocol (MCP)
- `POST /api/memory/mcp/store`: Encrypts and registers an interaction trace.
- `POST /api/memory/mcp/execute/{id}`: Loads and runs a solution node from the PQC Vault.

### Arrow Flight Memory Stream
- **Endpoint**: `:50051`
- **Schema**: `timestamp (int64)`, `agent_id (string)`, `intent (string)`, `observed_action (string)`, `verified (bool)`.

---

## 🧪 Build & Deployment

### Environment Requirements
- **OS**: Linux (Kernel 5.10+ required for eBPF features). Kali Linux or Ubuntu 22.04+ recommended.
- **Tools**: Go 1.24, Node.js 18+, `build-essential`, `libseccomp-dev`.

### Unified Build Process
```bash
# 1. Install frontend dependencies
npm install

# 2. Build the Next.js frontend
npm run build

# 3. Build the Go backend
cd backend
go mod tidy
go build -o ../bin/backend_server ./main.go
cd ..

# 4. Compile the Unified Binary (Main Wrapper)
go build -o knirv-server main.go
```

### Configuration

Enable FinTech Validator phases in configuration:

```go
config := &fintech_validator.Config{
    // Phase 1: Evidence Packs & Ontologies
    EnableAMLChecks:       true,
    EnableKYCCheks:        true,
    EnableSECCheks:        true,
    EnableBaselCheks:      true,
    
    // Phase 2: Scenarios & Certification
    EnableScenarioTesting: true,
    EnableCertification:   true,
    
    // Phase 3: Trajectory Capture
    EnableTrajectoryCapture: true,
    EnableReplayEngine:      true,
    
    // Phase 4: NRV & Fidelity
    EnableNRVTracing:       true,
    EnableFidelityScoring:  true,
    
    // Phase 6: Tick Data Streaming
    EnableTickDataStreaming: true,
    TickDataServerPort:     "8819",
    MaxStreamBufferSize:    1000,
}
```

---

## 🌐 Progressive Web Application (PWA)

KNIRVSERVER includes a Progressive Web Application (PWA) that provides a native-like desktop experience directly through the browser.

### Features

- **Installable Client**: Users can install the application to their desktop for offline access
- **System Metrics HUD**: Real-time CPU, memory, and system information display
- **Offline Support**: Service worker caching ensures functionality without network
- **Cross-Platform**: Works on Windows, macOS, Linux, and mobile browsers

### Architecture

```
┌─────────────────────────────────────────┐
│         Browser / PWA Client            │
├─────────────────────────────────────────┤
│ - React Frontend (Next.js static export) │
│ - Service Worker (offline support)      │
│ - HUD React Component                  │
└──────────────────┬──────────────────────┘
                   │ HTTP/HTTPS
                   │ All requests use relative /api/* paths
                   ▼
┌─────────────────────────────────────────────┐
│      KNIRVSERVER Wrapper (Port 8090)       │
├─────────────────────────────────────────────┤
│ - Static frontend server                  │
│ - PWA manifest & service worker handler    │
│ - Reverse proxy middleware              │
└──────┬──────────────────────────┬───────────┘
       │ /api/v1/* routes          │ Static files
       │ (to Unix socket)          │
       ▼                          ▼
┌────────────────────────────────────────┐
│ Backend Services (Unix Socket)         │
│ /var/run/knirvserver-backend.sock      │
├────────────────────────────────────────┤
│ - GET /api/v1/system/info              │
│ - WS /api/v1/system/metrics/stream     │
└────────────────────────────────────────┘
```

### API Endpoints

| Endpoint | Description |
|----------|------------|
| `GET /api/v1/system/info` | Get current system metrics (CPU, memory, uptime) |
| `GET /api/v1/system/detail` | Get detailed system information |
| `WS /api/v1/system/metrics/stream` | WebSocket stream for real-time metrics |

### PWA Components

- **HudOverlay**: React component displaying system metrics (`frontend/src/components/hud/`)
- **usePWAInstall**: Hook for handling PWA installation prompts (`frontend/src/hooks/usePWAInstall.ts`)

## 🔄 Self-Updater (Production)

KNIRV-SERVER includes an auto-updater that polls GitHub Releases for new versions and performs atomic binary updates.

### Configuration

Add to `config/production.yaml`:

```yaml
updater:
  enabled: true
  poll_interval: 10m
  github_repo: "knirvcorp/KNIRV_NETWORK"
```

Set `DEFAULT_GITHUB_TOKEN` environment variable containing the GitHub token for private repository access.

### How It Works

1. Server polls GitHub Releases API every 10 minutes (configurable)
2. Compares release `tag_name` (commit SHA) to current `GitCommit`
3. Downloads new binary and SHA256 checksum
4. Verifies checksum before execution
5. Atomically swaps binary: current → `.prev`, new → current
6. Calls `syscall.Exec` to restart with new binary

### Rollback

```bash
# Manual rollback
mv /usr/local/bin/knirv-server /usr/local/bin/knirv-server.bad
mv /usr/local/bin/knirv-server.prev /usr/local/bin/knirv-server
systemctl restart knirv-server
```

---

## 🏗️ OverlayFS DVE Workspace

KNIRV-SERVER provides **OverlayFS + Linux User Namespace** isolation for DVE workspaces, replacing the previous eBPF-only virtual container approach. Each workspace gets its own writable CoW layer sharing a read-only BusyBox base.

### Architecture

```
/var/lib/knirvserver/
├── busybox-rootfs/          ← read-only lower layer (shared, ~5MB)
│   ├── bin/busybox          ← static binary with 400+ applets via symlink
│   ├── etc/passwd           ← minimal passwd for DVE user
│   └── .knirvdve-ready      ← bootstrap marker
└── workspaces/{dveID}/
    ├── upper/               ← per-DVE writable CoW layer
    ├── work/                ← OverlayFS internal scratch
    └── merged/              ← DVE sees this as its root /
```

### Isolation Model

| Layer | Technology | Privilege | Speed |
|-------|-----------|-----------|-------|
| Filesystem | OverlayFS (kernel) | None (Linux ≥3.18 userns) | ~8ms workspace creation |
| Namespace | `CLONE_NEWUSER \| CLONE_NEWNS` | No root required | Instant |
| Skill execution | Wazero WASM (pure Go) | No host syscalls | Compile+run |
| Fallback | fuse-overlayfs | No root | Slightly slower |

### Files

| File | Purpose |
|------|---------|
| `backend/internal/services/dve_workspace/dve_service.go` | DVE service with createVirtualDVEAsync |
| `backend/internal/services/dve_workspace/overlayfs.go` | OverlayWorkspace mount/unmount/destroy |
| `backend/internal/services/dve_workspace/namespace.go` | DVENamespace spawn/teardown + helper |
| `backend/internal/services/dve_workspace/busybox.go` | BusyBox rootfs bootstrap (embedded/package/download) |
| `backend/internal/services/dve_workspace/wazero_executor.go` | Wazero WASM skill executor |
| `frontend/src/components/dashboard/dve-workspace-settings.tsx` | Runtime config UI panel |

### API Endpoints

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/api/dve-workspace/config` | Get current runtime DVEConfig |
| `PUT` | `/api/dve-workspace/config` | Hot-update config (affects new workspaces) |
| `GET` | `/api/dve-workspace/rootfs-status` | Check BusyBox rootfs bootstrap state |
| `POST` | `/api/dve-workspace/rootfs-bootstrap` | Trigger one-time rootfs setup |
| `GET` | `/api/dve-workspace/stats` | Active workspaces, CoW disk usage, WASM queue depth |

### Configuration

```yaml
dve_workspace:
  enable_overlayfs: true
  busybox_rootfs_path: "/var/lib/knirvserver/busybox-rootfs"
  busybox_source: "embedded"       # embedded | package | download
  fuse_overlayfs_bin: "fuse-overlayfs"
  skill_exec_timeout: "120s"
  skill_max_memory_mb: 512
  max_concurrent_wasm: 10
  workspace_retention_hours: 48
```

### BuntDB Migration

Existing BuntDB data keyed under legacy `cde_*` prefixes is automatically migrated to `dve_*` prefixes at startup via `BuntDBManager.MigrateCDEKeysToDVE()`. Safe to call multiple times — already-migrated records are skipped.

---

## 🔒 Security Constraints

- **Hardware Enclaves**: TEE features (SGX/TDX) require specific hardware support and BIOS configuration.
- **eBPF Capabilities**: Requires `CAP_SYS_ADMIN` or `CAP_BPF` privileges for program attachment.
- **PQC Keys**: Master key rotation requires manual re-encryption of the Markdown Fabric.
- **Agent Isolation**: oh-my-pi agents run with 70+ syscall allowlist and restricted filesystem/network access.

---

## 📄 License
Copyright 2026 KNIRV-SERVER. Distributed under the **GPL-3.0-or-later** license.

---

**KNIRV-SERVER**: *Where semantic reasoning meets verifiable execution.* 🚀
