# Memory System Consolidation Plan

## Current State (5 Separate Implementations)

| Component | Implementation | Purpose | Storage | Key Dependencies |
|-----------|----------------|---------|---------|-------------------|
| **Active Memory** | `backend/internal/services/active_memory/service.go` | Encrypted .md vault for agent interactions | `MarkdownStorageDriver` + PQC encryption | VaultService, ReasoningEngine, Arrow Flight streaming |
| **Knowledge Base** | `backend/internal/services/knowledge_base/service.go` | GraphRAG-powered knowledge base | FFI → Rust graphrag-rs | Arrow, FFI bridge, embedding models |
| **KNIRVGRAPH** | `pkg/knirvgraph/manager.go` | Graph database + sync | External binary (socket) | SyncManager, error nodes, KNIRVGRAPH binary |
| **Ontology** | `backend/internal/services/cognitiveengine/ontology.go` | In-memory knowledge graph | `map[string]*OntologyEntity` | IC-M-E TemporalHypergraph, entity/relation tracking |
| **Nexus Server** | `backend/internal/server/server.go` | Arrow Flight streaming + eBPF events | Arrow Flight + eBPF Guardian | Intent tracking, event broadcasting |

---

## Core Problem

All five components implement the **same concept**: a knowledge representation and retrieval system. They are fragmented across:

- **3 storage backends**: Markdown files on disk, GraphRAG index in memory, in-memory Go maps
- **5+ API prefixes**: `/api/memory/`, `/api/v1/knowledge-base/`, `/api/knirvgraph/`, `/api/nexus/`, `/api/ontology/`
- **2 encryption schemes**: PQC for Markdown, none for others
- **3 graph representations**: Ontology entities/relations, GraphRAG nodes/edges, TemporalHypergraph
- **2 event systems**: eBPF Guardian events, IC-M-E signal routing

---

## Proposed Unified Architecture

```
UnifiedMemorySystem
├── Storage Layer (unified)
│   ├── MarkdownStorageDriver (encryption/PQC) → persistent .md files
│   ├── GraphRAGIndex (FFI) → queryable vector/graph index
│   └── OntologyStore (in-memory) → reasoning graph
├── Streaming Layer
│   └── ArrowFlightServer (Arrow Flight + eBPF event capture)
├── Graph Layer
│   ├── KNIRVGRAPH (external sync/error nodes)
│   └── TemporalHypergraph (IC-M-E integration)
└── Services
    ├── VaultService (ErrorNode/SolutionNode management)
    └── ReasoningEngine (trace generation + GraphRAG queries)
```

---

## Dependency Map

```
MarkdownStorageDriver (pqc.EncryptionManager)
    ↑
VaultService → ActiveMemoryService
    ↑
ReasoningEngine → KNIRVGRAPHEngine
    ↑
TemporalHypergraph ← DVEOntologyManager
    ↑
IC-M-E Service (Intentional Context Memory Engine)
    ↑
ArrowFlightServer (Arrow Flight + eBPF events)
```

---

## Consolidation Steps

### Step 1: Create `UnifiedMemorySystem` Interface

**New module**: `backend/internal/services/memory/`

Merge the following into a single cohesive interface:
- `ActiveMemoryService` - encrypted markdown vault
- `KnowledgeBase` lifecycle - GraphRAG operations
- `ArrowFlightServer` - event streaming

Create a single config struct with feature flags:
```go
type MemoryConfig struct {
    EnabledBackends    []string  // "markdown", "graphrag", "ontology"
    PQCEencryption    bool
    ArrowStreaming    bool
    GraphRAGModel     string
    SyncInterval      time.Duration
}
```

### Step 2: Unify Storage Backends into `MemoryStore`

Create `backend/internal/storage/memory_store.go`:

- **Keep** `MarkdownStorageDriver` for encrypted persistence (already well-designed)
- **Keep** `GraphRAGIndex` for vector/graph queries (FFI bridge to Rust)
- **Add auto-sync**: `.md` save → GraphRAG reindex → Ontology update

```go
type MemoryStore struct {
    markdown  *mdstorage.MarkdownStorageDriver
    graphrag  *knowledge_base.GraphRAGClient
    ontology  *cognitiveengine.DVEOntologyManager
    autoSync  bool
}
```

### Step 3: Merge APIs (5+ endpoints → 1 prefix)

**New unified API prefix**: `/api/v1/memory/`

| Old Endpoint | New Endpoint | Description |
|--------------|---------------|-------------|
| `/api/memory/mcp/store` | `/api/v1/memory/store` | Store interaction |
| `/api/memory/mcp/execute/{id}` | `/api/v1/memory/execute/{id}` | Execute solution |
| `/api/v1/knowledge-base/objects` | `/api/v1/memory/knowledge-bases` | List knowledge bases |
| `/api/v1/knowledge-base/objects/{id}` | `/api/v1/memory/knowledge-bases/{id}` | Get knowledge base |
| `/api/v1/knowledge-base/objects/{id}/query` | `/api/v1/memory/knowledge-bases/{id}/query` | Query knowledge base |
| `/api/knirvgraph/error-node` | `/api/v1/memory/error-nodes` | Create error node |
| `/api/knirvgraph/error-nodes` | `/api/v1/memory/error-nodes` | List error nodes |
| `/api/nexus/events` | `/api/v1/memory/events` | Get events (Arrow Flight) |
| `/api/ontology/entities` | `/api/v1/memory/ontology/entities` | List entities |
| `/api/ontology/relations` | `/api/v1/memory/ontology/relations` | List relations |

**Deprecate** (keep for 6 months with redirects):
- `/api/memory/` (legacy Active Memory)
- `/api/v1/knowledge-base/` (legacy Knowledge Base)
- `/api/knirvgraph/` (legacy KNIRVGRAPH)
- `/api/nexus/` (legacy Nexus)
- `/api/ontology/` (legacy Ontology)

### Step 4: Consolidate KNIRVGRAPH Integration

**Move** `SyncManager` into `UnifiedMemorySystem`:
- Keep external binary wrapper (`pkg/knirvgraph/manager.go`) for process management
- Route error node operations through unified API
- Merge `KNIRVGRAPHEngine` + `ReasoningEngine` into single query interface

```go
type QueryRequest struct {
    Query    string
    Mode     string  // "graphrag", "ontology", "hybrid"
    Limit    int
    Sources  []string  // which backends to query
}
```

### Step 5: Update Agent Integration

Update `AGENTS.md` to reflect single module:
```markdown
## Memory System

UnifiedMemorySystem — All knowledge representation in one module.

| Package | Tech | Module file |
|---------|------|-------------|
| `backend/internal/services/memory/` | PQC + GraphRAG + Ontology | `services/memory/service.go` |
| `backend/internal/storage/memory_store.go` | Markdown + Arrow | `storage/memory_store.go` |
| `pkg/knirvgraph/` | External graph binary | `pkg/knirvgraph/manager.go` |
```

---

## Files to Merge/Refactor

| Action | Current Files | New Location |
|--------|--------------|---------------|
| **Merge** | `active_memory/service.go` + `knowledge_base/service.go` | `services/memory/service.go` |
| **Consolidate** | `server/server.go` (Nexus) | `services/memory/flight.go` |
| **Unify** | `cognitiveengine/ontology.go` | `services/memory/ontology.go` |
| **Keep** | `knirvgraph/manager.go` | `pkg/knirvgraph/` (external binary wrapper) |
| **Keep** | `knowledge_base/graphrag_ffi.go` | `services/memory/graphrag.go` (FFI bridge) |
| **Keep** | `storage/mdstorage/driver.go` | `storage/memory_store.go` (encryption layer) |
| **Keep** | `reasoning/graph/engine.go` | `services/memory/reasoning.go` (trace generation) |
| **Keep** | `vault/service.go` | `services/memory/vault.go` (ErrorNode/SolutionNode) |
| **Update** | `cmd/backend_server/main.go` | Wire `UnifiedMemorySystem` |
| **Update** | `web/api_router.go` | Add `/api/v1/memory/` routes |
| **Create** | New handlers | `web/memory_handlers.go` |

---

## Migration Path

### Phase 1: Preparation (Week 1-2)
1. Create `services/memory/` module directory
2. Define `UnifiedMemorySystem` interface
3. Implement adapter pattern for existing services (no breaking changes)
4. Write comprehensive tests for new interface

### Phase 2: Core Migration (Week 3-4)
1. Move `ActiveMemoryService` → `memory/service.go`
2. Move `KnowledgeBase` → `memory/knowledge_base.go`
3. Move `ArrowFlightServer` → `memory/flight.go`
4. Update `main.go` to optionally wire `UnifiedMemorySystem`

### Phase 3: API Unification (Week 5-6)
1. Implement new `/api/v1/memory/` handlers
2. Add deprecation notices to old endpoints (HTTP 299 with `Warning` header)
3. Create redirect middleware for old → new paths
4. Update CLI tool to use new endpoints

### Phase 4: KNIRVGRAPH & Ontology (Week 7-8)
1. Move `SyncManager` into `UnifiedMemorySystem`
2. Consolidate `KNIRVGRAPHEngine` + `ReasoningEngine`
3. Merge `DVEOntologyManager` into unified ontology store
4. Update IC-M-E integration to use unified interface

### Phase 5: Cleanup (Week 9-10)
1. Remove legacy modules (or move to `legacy/` directory)
2. Update all imports across codebase
3. Update documentation (`AGENTS.md`, `server-api.yaml`)
4. Full test suite run
5. Deploy behind feature flag

---

## Result

**Before**: 5 fragmented systems, 5+ API prefixes, 3 storage backends, 2 encryption schemes

**After**: 1 `UnifiedMemorySystem` with:
- 1 API prefix: `/api/v1/memory/`
- 1 storage interface: `MemoryStore`
- 1 encryption scheme: PQC for all persistent data
- 1 graph interface: Query any backend (GraphRAG, Ontology, Hybrid)
- 1 event stream: Arrow Flight with eBPF capture
- Feature flags for gradual rollout

---

## Benefits

1. **Reduced cognitive load**: Developers work with 1 system, not 5
2. **Consistent API**: Single prefix, consistent patterns
3. **Unified encryption**: PQC across all persistent storage
4. **Cross-backend queries**: Query GraphRAG and Ontology simultaneously
5. **Simplified maintenance**: 1 module to update, not 5
6. **Clear dependency chain**: `MarkdownStorage` → `VaultService` → `ReasoningEngine` → `IC-M-E`
7. **Event unification**: eBPF events + ontology signals in 1 Arrow Flight stream

---

## Risks & Mitigations

| Risk | Mitigation |
|------|-------------|
| Breaking changes for CLI tools | Keep old endpoints with redirects for 6 months |
| Data migration complexity | Auto-migration: .md files → GraphRAG index on first run |
| Performance regression | Feature flags; can toggle backends independently |
| External KNIRVGRAPH binary compatibility | Keep `pkg/knirvgraph/` wrapper unchanged initially |
| IC-M-E integration breakage | Adapter pattern; keep `TemporalHypergraph` interface stable |

---

## Success Metrics

- [ ] 5 modules → 1 `services/memory/` module
- [ ] 5+ API prefixes → 1 `/api/v1/memory/` prefix
- [ ] 3 storage backends → 1 `MemoryStore` interface
- [ ] 0 breaking changes (old endpoints redirect for 6 months)
- [ ] 100% test coverage on new `UnifiedMemorySystem`
- [ ] PQC encryption on all persistent storage
- [ ] Arrow Flight streaming for all event types
- [ ] Cross-backend query support (GraphRAG + Ontology)
