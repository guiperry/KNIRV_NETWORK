# KNIRV Network — Shared ProtoBuf Definitions

Single source of truth for all inter-package gRPC and serialization contracts.
All packages are submoduled by **KNIRVSERVER** — each package compiles its own
generated code from these definitions, not from a shared generated output.

## Directory Structure

```
shared-proto/
├── cortex/v1/cortex.proto          # Core inference, skill, capability, property messages
├── lora/v1/lora.proto              # LoRA adapter engine (skill weights, training, compilation)
├── memory/v1/memory.proto          # Memory protocol (episodic, semantic, procedural, working)
├── agent/v1/agent.proto            # Agent compilation, execution, deployment, validation
├── graph/v1/graph.proto            # KNIRVGRAPH error context, clusters, node submission
├── chain/v1/chain.proto            # KNIRVCHAIN on-chain skill invocation and compilation
├── blockchain/v1/blockchain.proto  # KNIRVSERVER ↔ KNIRVCHAIN gRPC service interface
├── hasher/v1/hasher.proto          # KNIRVHASHER ASIC service + training pipeline service
├── knirvchain/v1/
│   ├── bootnode_key.proto          # Encrypted bootnode key schema (KNIRVCHAIN)
│   ├── hashing.proto               # Block, transaction, LLM rooting protos
│   ├── mcp_context.proto           # MCP context record and capability invocation tx data
│   ├── mcp_descriptors.proto       # MCP capability descriptor types (resource, tool, prompt)
│   └── root_key.proto              # Encrypted root key schema (KNIRVCHAIN)
└── knirvserver/v1/
    ├── bootnode_key.proto          # Encrypted bootnode key schema (KNIRVSERVER)
    └── root_key.proto              # Encrypted root key schema (KNIRVSERVER, superset of chain)
```

## Package Map

| Proto namespace | Go package | Used by |
|-----------------|-----------|---------|
| `knirv.cortex.v1` | `pkg/gen/knirv/cortex/v1` | KNIRVARENA (TS), KNIRVSERVER |
| `knirv.lora.v1` | `pkg/gen/knirv/lora/v1` | KNIRVARENA (TS), agent/v1 |
| `knirv.memory.v1` | `pkg/gen/knirv/memory/v1` | KNIRVARENA (TS), KNIRVSERVER |
| `knirv.agent.v1` | `pkg/gen/knirv/agent/v1` | KNIRVARENA (TS) |
| `knirv.graph.v1` | `pkg/gen/knirv/graph/v1` | KNIRVARENA (TS), KNIRVSERVER, KNIRVGRAPH, KNIRVCHAIN — owns the canonical `ErrorNode` |
| `knirv.chain.v1` | `pkg/gen/knirv/chain/v1` | KNIRVARENA (TS), KNIRVCHAIN |
| `knirv.blockchain.v1` | `backend_server/internal/proto/blockchain` | KNIRVSERVER → KNIRVCHAIN |
| `hasher.v1` | `knirvhasher/proto/hasher/v1` | KNIRVHASHER, KNIRVSERVER |
| `KNIRVCHAIN` | `KNIRVCHAIN/internal/protocol/proto` | KNIRVCHAIN internal |
| `KNIRVSERVER` | `./;proto` | KNIRVSERVER internal |

## Proto Descriptions

### `cortex/v1/cortex.proto`
Core inference protocol used by all KNIRVARENA TypeScript agent cores and KNIRVSERVER:
- **InferenceInput/Output** — main inference interface
- **Config** — runtime LLM configuration (tokens, temperature, model_id)
- **Context / MemoryPolicy** — memory retrieval and tool context
- **Tool / ToolSchema / ParameterSpec** — MCP tool system
- **ForgeManifest / ModelDimensions / TokenizerInfo** — model metadata
- **RuntimeConfig / ValidationResult** — WASM runtime binding
- **SkillInvocationRequest/Response / SkillChain** — skill execution
- **CapabilityInvocationRequest/Response** — MCP capability invocation
- **PropertyInvocationRequest/Response** — on-chain property access
- **AgentCompilationRequest/Result** — agent build (backward compat; prefer `agent/v1`)

### `lora/v1/lora.proto`
LoRA adapter engine for the skill system:
- **LoRAAdapter** — skill weights and metadata
- **SkillCompilationRequest** — skill creation from solutions/errors
- **SkillInvocationRequest/Response** — skill execution
- **SkillChain** — skill composition and merging
- **LoRATrainingConfig / LoRAWeights / LoRAModelConfig** — training parameters
- **LoRACompilationResult** — compilation output with performance score

### `memory/v1/memory.proto`
Comprehensive memory management:
- **MemoryRequest/Response** — generic CRUD + search operations
- **EpisodicMemory / EpisodicMemoryQuery** — event-based memory
- **SemanticTriple / SemanticMemoryQuery** — knowledge graph storage
- **ProceduralMemory / ProcedureStep** — skill and procedure storage
- **WorkingMemory / WorkingMemoryItem** — short-term active memory
- **MemoryConsolidationRequest/Result** — memory optimization passes
- **MemoryStatistics** — usage and performance metrics

### `agent/v1/agent.proto`
Agent lifecycle management (imports `lora/v1`):
- **AgentCompilationRequest/Result / AgentManifest** — agent building with embedded skills
- **AgentConfig** — platform-specific configuration (typescript, golang, rust)
- **AgentExecutionRequest/Result** — runtime execution interface
- **AgentDeploymentRequest/Result / DeploymentTarget** — multi-platform deployment
- **AgentValidationRequest/Result / ValidationTest** — testing and validation

### `graph/v1/graph.proto`
KNIRVGRAPH error context, cluster management, and the canonical node types:
- **ErrorContext** — rich error payload (agent info, environment, stack trace, task context)
- **ErrorClusterQueryRequest/Response / SkillNodeResult** — find skills for an error type
- **ErrorCluster** — grouped error set with bounty amount
- **ErrorNodeSubmissionRequest/Response** — submit new error nodes to the graph
- **ErrorNode** — **the** canonical error node, shared by KNIRVGRAPH (drq/nrv), KNIRVCHAIN (types)
  and KNIRVARENA. Carries identity, classification, provenance (`model_origin`), clustering id,
  payload (`failure_context` + structured `context`), lifecycle `status`, resolution path, bounty
  and timestamps. Deliberately excludes embedding vectors, queue position and scheduler priority —
  those are local DRQ clustering/scheduling state, not cross-service contract.
- **ContextNode** — a node attached to an `ErrorNode` carrying execution context; this context is
  the input half of what a uLoRA compile transmutes into adapter weights
- **ResolutionPath / ResolutionStep** — how an error was resolved
- **NodeStatus** — OPEN / IN_PROGRESS / RESOLVED / VALIDATED / REJECTED

#### One definition per concept

`ErrorNode` previously existed in four divergent places: this file's `ErrorContext`,
`chain/v1.ErrorNode` (5 fields), `lora/v1.ErrorContext` (3 fields) and `cortex/v1.ErrorContext`
(3 fields). The three duplicates were **removed**, not deprecated, and the `SkillData`/
`SkillTrainingData` messages that embedded them now carry `repeated string error_ids`. Nothing
consumed them: no Go code was ever generated for `knirv.chain.v1`, `knirv.lora.v1` or
`knirv.cortex.v1`, and KNIRVARENA's live code uses its own hand-written interfaces
(`core/protobuf/ErrorContextHandler.ts`), not these messages. `ErrorContext` in *this* file is
retained — it is the inbound submission payload, a different concept from the persisted node.

### `chain/v1/chain.proto`
On-chain skill operations via KNIRVCHAIN:
- **LoRaAdapterSkill** — on-chain skill with packed byte weights
- **SkillInvocationRequest/Response** — invoke a skill by ID
- **SkillCompilationRequest / SkillMetadata / SkillTrainingData** — compile skill from errors+solutions
  (`SkillTrainingData` carries `repeated string error_ids`; the nodes themselves are
  `knirv.graph.v1.ErrorNode`, not a chain-local duplicate)
- **Solution** — training data primitive

### `blockchain/v1/blockchain.proto`
gRPC service contract between KNIRVSERVER and KNIRVCHAIN (`BlockchainService`):
- **VerifyPayment** — confirm on-chain payment by tx hash
- **SubmitTransaction** — submit signed (+ PQC) transaction
- **GetBalance / GetBlockHeight** — chain state queries
- **RegisterDVENode** — register a DVE node with stake
- **CreateChainSession / ValidateSession** — PQC-signed session management
- **GetSecret** — retrieve session-scoped secrets

### `hasher/v1/hasher.proto`
Dual-service interface for the KNIRVHASHER ASIC device:

**HasherService** (hardware compute):
- **ComputeHash / ComputeBatch / StreamCompute** — SHA-256 via ASIC
- **GetMetrics / GetDeviceInfo** — eBPF performance telemetry
- **MineWork** — Bitcoin-style nonce search
- **VerifyMath** — mathematical derivation verification

**HasherTrainingService** (data pipeline, used by KNIRVSERVER):
- **ExportSecurityData** — stream DVE data (ontology, guardrails, activity) to hasher
- **TriggerTraining / GetTrainingStatus** — training job management
- **GetUserRules / ValidateAction** — query and apply trained security rules
- **StreamActivity** — real-time activity event stream

### `knirvchain/v1/`
Internal KNIRVCHAIN serialization schemas (package `KNIRVCHAIN`):
- **bootnode_key.proto** — `BootnodeKeyFileContentProto` / `EncryptedBootnodeKeyFile`
- **hashing.proto** — `TransactionProto`, `BlockProto`, `LLMRootingDataProto`
- **mcp_context.proto** — `ContextRecordProto`, `MCPInvokeCapabilityDataProto`
- **mcp_descriptors.proto** — all MCP capability descriptor types, register/update tx data
- **root_key.proto** — `RootKeyFileContentProto` (21 fields), `EncryptedRootKeyFile`

### `knirvserver/v1/`
Internal KNIRVSERVER serialization schemas (package `KNIRVSERVER`):
- **bootnode_key.proto** — identical structure to KNIRVCHAIN variant
- **root_key.proto** — superset of KNIRVCHAIN variant; adds fields 22-26 for KNIRVHASHER device
  credentials (`device_ip`, `device_password`, `device_username`) and Cloudflare tunnel
  extras (`cloudflare_embeddings_url`, `cloudflare_account_id`)

## Synchronization Targets

| Package | Proto consumer path | Language |
|---------|--------------------|----|
| KNIRVARENA | `packages/KNIRVARENA/src/core/protobuf/schemas/` | TypeScript |
| KNIRVSERVER | `packages/KNIRVSERVER/backend/internal/proto/` | Go |
| KNIRVCHAIN | `packages/KNIRVCHAIN/internal/protocol/proto/` | Go |
| KNIRVHASHER | `packages/KNIRVHASHER/internal/proto/hasher/v1/` | Go |

## Development Workflow

1. Edit the relevant `.proto` file in `shared-proto/`
2. Validate: `protoc --proto_path=shared-proto --descriptor_set_out=/dev/null <file>`
3. Copy to the consuming package's proto directory
4. Run `protoc` (or `buf generate`) within that package to regenerate code
5. Compile and test the affected package

## Key Invariants

- **KNIRVGATEWAY does not manage oracle** — oracle routes are KNIRVSERVER root-node only
- **No cross-package Go imports** — inter-service communication is HTTP/gRPC only
- **root.key presence enables oracle** — missing key = normal KNIRVSERVER operation
- **knirvserver root_key is a strict superset** of knirvchain root_key (fields 1-21 identical, 22-26 added)
- **hasher.v1 is the canonical hasher contract** — KNIRVSERVER's internal copy is derived from it
