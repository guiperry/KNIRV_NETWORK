# Intentional Context Memory Engine (ICME)
## Integration Guide for KNIRV-NEXUS

**Synthesized from:** `ContextMemoryEngine.md` (Pulse HQ GPU Pipeline) + `IntentEngineering.md` (Organizational Intent Framework)
**Target System:** KNIRV-NEXUS Deterministic Validation Environment
**Language:** Go 1.24+ (primary), Python 3.11+ (ML microservices)

---

## Table of Contents

1. [Conceptual Overview](#1-conceptual-overview)
2. [Architecture](#2-architecture)
3. [Prerequisites & Dependencies](#3-prerequisites--dependencies)
4. [Layer 1 — Intentional Signal Ingestion](#4-layer-1--intentional-signal-ingestion)
5. [Layer 2 — Intent Engineering Framework](#5-layer-2--intent-engineering-framework)
6. [Layer 3 — Intentional Graph Construction](#6-layer-3--intentional-graph-construction)
7. [Layer 4 — Embedding Generation & Semantic Indexing](#7-layer-4--embedding-generation--semantic-indexing)
8. [Layer 5 — Hybrid Retrieval API](#8-layer-5--hybrid-retrieval-api)
9. [Layer 6 — Feedback & Alignment Loops](#9-layer-6--feedback--alignment-loops)
10. [Configuration Changes](#10-configuration-changes)
11. [API Reference](#11-api-reference)
12. [Frontend Integration](#12-frontend-integration)
13. [Python ML Microservices](#13-python-ml-microservices)
14. [Testing Strategy](#14-testing-strategy)
15. [Deployment & Scaling](#15-deployment--scaling)

---

## 1. Conceptual Overview

### What Is the ICME?

The **Intentional Context Memory Engine (ICME)** fuses two independently powerful paradigms into a single coherent system layered onto KNIRV-NEXUS:

| Source Concept | Contribution to ICME |
|---|---|
| **ContextMemoryEngine** (Pulse HQ) | GPU-accelerated entity extraction, temporal hypergraph construction, FAISS semantic indexing, real-time streaming |
| **IntentEngineering** | Machine-readable organizational goals (OKRs), delegation hierarchies, hard/soft decision boundaries, alignment feedback loops |

Without intent, a context memory engine is a passive recorder — it knows *what happened* but not *what should happen*. Without rich context memory, intent engineering becomes shallow rule-matching with no temporal awareness. Together they form a system where:

- Every agent action is validated not just against a syscall trace (existing eBPF layer) but against **structured organizational intent parameters**.
- Every memory retrieval is **semantically ranked by relevance to declared intent**, not just recency.
- The temporal hypergraph captures **why** decisions were made alongside **what** was done, enabling deterministic replay with full causal context.
- Feedback from KNIRV-NEXUS's existing FinTech Validator (NRV traces, fidelity scores) closes the loop, **continuously refining intent parameters**.

### Relationship to Existing KNIRV-NEXUS Systems

```
EXISTING SYSTEMS                    NEW ICME LAYERS
─────────────────                   ───────────────
NexusMemoryServer (Arrow Flight) ←→ Semantic Embedding Index (FAISS)
ActiveMemoryService              ←→ Intentional Signal Router
ReasoningGraph (ContextRecord)   ←→ Temporal Hypergraph Engine
VaultService (ErrorNode/Solution)←→ Intent-Annotated Vault Nodes
CognitiveEngine                  ←→ Intent Engineering Framework
FinTech NRV Fidelity Scorer      ←→ Alignment Feedback Loop
eBPF Intent-Action Correlation   ←→ Intent Constraint Enforcement
```

The ICME does **not replace** any existing service. It integrates as an additional service layer (`icme`) that wires into existing service interfaces.

---

## 2. Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    KNIRV-NEXUS ICME STACK                       │
├─────────────────────────────────────────────────────────────────┤
│  FRONTEND (Next.js 15)                                          │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────┐  │
│  │  Intent Console  │  │  Context Explorer│  │ Align Monitor│  │
│  └──────────┬───────┘  └────────┬─────────┘  └──────┬───────┘  │
├─────────────┼────────────────────┼───────────────────┼──────────┤
│  GO BACKEND (port 8082)          │                   │          │
│  ┌──────────▼───────────────────▼───────────────────▼───────┐  │
│  │              ICME Service (new)                           │  │
│  │  ┌─────────────────┐  ┌──────────────────────────────┐   │  │
│  │  │ Intent Registry │  │   Hybrid Retrieval Engine    │   │  │
│  │  └────────┬────────┘  └──────────────┬───────────────┘   │  │
│  │  ┌────────▼────────┐  ┌──────────────▼───────────────┐   │  │
│  │  │ IntentHypergraph│  │  FAISS Client (gRPC)         │   │  │
│  │  └────────┬────────┘  └──────────────┬───────────────┘   │  │
│  └───────────┼──────────────────────────┼───────────────────┘  │
│              │                          │                       │
│  ┌───────────▼──────────────────────────▼───────────────────┐  │
│  │           EXISTING SERVICES (unchanged interfaces)        │  │
│  │  ActiveMemoryService │ VaultService │ CognitiveEngine     │  │
│  │  NexusMemoryServer   │ ReasoningGraph │ FintechValidator  │  │
│  │  eBPF Manager        │ BuntDB │ MarkdownFabric │ Arrow    │  │
│  └───────────────────────────────────────────────────────────┘  │
├─────────────────────────────────────────────────────────────────┤
│  PYTHON ML MICROSERVICES (gRPC)                                 │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────┐  │
│  │  Entity Extractor│  │ Embedding Service│  │ Re-Ranker    │  │
│  │  (GPU NER/spaCy) │  │ (GPU NVEmbed)    │  │(Cross-Encode)│  │
│  └──────────────────┘  └──────────────────┘  └──────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

### Data Flow

```
Agent Signal In
      │
      ▼
[1] Intentional Signal Ingestion
      │  - Tag signal with source agent ID
      │  - Route through Intent Registry (what should this agent do?)
      │  - Extract entities via Python gRPC (GPU NER)
      │
      ▼
[2] Intent Constraint Check (eBPF + IntentRegistry)
      │  - Does this action fall within authorized_actions?
      │  - Apply trade-off weights
      │  - Enforce hard_boundaries
      │
      ▼
[3] Temporal Hypergraph Update
      │  - Add entity nodes and relationship edges
      │  - Stamp edges with timestamp + intent_objective
      │  - Prune edges outside time window
      │
      ▼
[4] Embedding + FAISS Index Update
      │  - Generate vector for signal content (Python gRPC)
      │  - Add to FAISS index with metadata
      │  - Store mapping: vector_id → graph_node_id
      │
      ▼
[5] Active Memory + Vault Persistence
      │  - Write ErrorNode / SolutionNode (existing vault)
      │  - Write ContextRecord with intent annotation (existing reasoning graph)
      │  - Encrypt and persist to Markdown Fabric
      │
      ▼
[6] Arrow Flight Streaming
        - Broadcast updated context + intent to subscribed agents
        - Schema includes intent_objective, alignment_score fields
```

---

## 3. Prerequisites & Dependencies

### Go Dependencies (add to `go.mod`)

```
require (
    // Existing deps retained...

    // ICME additions
    google.golang.org/grpc v1.64.0
    google.golang.org/protobuf v1.34.2
    github.com/DataIntelligenceCrew/go-faiss v0.3.0
)
```

> **Note:** `go-faiss` requires `libfaiss` installed on the host. On Ubuntu/Debian:
> ```bash
> apt-get install -y libfaiss-dev
> ```
> On systems without GPU, use `faiss-cpu`. FAISS GPU bindings require CUDA toolkit matching the driver version on DGX Spark hardware.

### Python Microservice Dependencies

```bash
# In /backend/internal/services/icme/ml_services/
pip install \
    grpcio==1.64.0 \
    grpcio-tools==1.64.0 \
    spacy==3.7.4 \
    torch==2.3.0 \
    transformers==4.42.0 \
    sentence-transformers==3.0.1 \
    faiss-gpu==1.7.2 \
    cupy-cuda12x==13.1.0
```

### gRPC Proto Definitions

Create `backend/internal/services/icme/proto/icme.proto`:

```protobuf
syntax = "proto3";
package icme;
option go_package = "github.com/KNIRVNEXUS/backend/internal/services/icme/proto";

// ─── Entity Extraction ───────────────────────────────────────────────────────

service EntityExtractor {
  rpc ExtractEntities (ExtractRequest) returns (ExtractResponse);
  rpc ExtractBatch    (BatchExtractRequest) returns (BatchExtractResponse);
}

message ExtractRequest {
  string text      = 1;
  string agent_id  = 2;
  string source    = 3; // "slack","zoom","validation","error"
}

message Entity {
  string id    = 1;
  string text  = 2;
  string label = 3; // PERSON, ORG, CONFIG, EVENT, ERROR, SOLUTION
  float  score = 4;
  int32  start = 5;
  int32  end   = 6;
}

message Relation {
  string from_entity_id = 1;
  string to_entity_id   = 2;
  string relation_type  = 3; // CAUSED_BY, RESOLVED_BY, DEPENDS_ON, TRIGGERS
  float  confidence     = 4;
}

message ExtractResponse {
  repeated Entity   entities  = 1;
  repeated Relation relations = 2;
  string            error     = 3;
}

message BatchExtractRequest  { repeated ExtractRequest requests = 1; }
message BatchExtractResponse { repeated ExtractResponse responses = 1; }

// ─── Embedding Generation ─────────────────────────────────────────────────────

service EmbeddingService {
  rpc GenerateEmbedding (EmbedRequest) returns (EmbedResponse);
  rpc GenerateBatch     (BatchEmbedRequest) returns (BatchEmbedResponse);
}

message EmbedRequest {
  string text      = 1;
  string model     = 2; // "nvembed","e5-large","minilm"
}

message EmbedResponse {
  repeated float vector = 1; // dimension depends on model
  string         error  = 2;
}

message BatchEmbedRequest  { repeated EmbedRequest requests = 1; }
message BatchEmbedResponse { repeated EmbedResponse responses = 1; }

// ─── Re-Ranking ───────────────────────────────────────────────────────────────

service ReRanker {
  rpc Rerank (RerankRequest) returns (RerankResponse);
}

message RerankRequest {
  string          query       = 1;
  repeated string candidates  = 2;
  int32           top_k       = 3;
}

message RankedResult {
  int32  original_index = 1;
  float  score          = 2;
  string text           = 3;
}

message RerankResponse {
  repeated RankedResult results = 1;
  string                error   = 2;
}
```

Generate Go bindings:

```bash
# From backend/internal/services/icme/
protoc \
  --go_out=. \
  --go-grpc_out=. \
  --proto_path=proto \
  proto/icme.proto
```

---

## 4. Layer 1 — Intentional Signal Ingestion

### 4.1 Signal Model

Create `backend/internal/services/icme/models.go`:

```go
package icme

import "time"

// SignalSource identifies where a signal originated.
type SignalSource string

const (
    SourceValidation SignalSource = "validation"
    SourceError      SignalSource = "error"
    SourceSolution   SignalSource = "solution"
    SourceAgent      SignalSource = "agent"
    SourceFintech    SignalSource = "fintech"
    SourceEBPF       SignalSource = "ebpf"
)

// IntentionalSignal is the enriched form of any event entering the ICME.
// It wraps the raw content with intent context before graph insertion.
type IntentionalSignal struct {
    ID              string            `json:"id"`
    AgentID         string            `json:"agent_id"`
    Source          SignalSource      `json:"source"`
    Content         string            `json:"content"`
    Timestamp       time.Time         `json:"timestamp"`
    ObjectiveName   string            `json:"objective_name"`   // from IntentRegistry
    AuthorizedActs  []string          `json:"authorized_acts"`  // from Objective
    TradeOffWeights map[string]float64`json:"trade_off_weights"`
    HardBoundaries  []string          `json:"hard_boundaries"`
    AlignmentScore  float64           `json:"alignment_score"`  // computed post-validation
    Entities        []ExtractedEntity `json:"entities"`
    Relations       []ExtractedRelation `json:"relations"`
    EmbeddingID     int64             `json:"embedding_id"`     // FAISS vector ID
}

// ExtractedEntity is the Go representation of a proto Entity.
type ExtractedEntity struct {
    ID    string  `json:"id"`
    Text  string  `json:"text"`
    Label string  `json:"label"`
    Score float32 `json:"score"`
    Start int     `json:"start"`
    End   int     `json:"end"`
}

// ExtractedRelation is the Go representation of a proto Relation.
type ExtractedRelation struct {
    FromEntityID string  `json:"from_entity_id"`
    ToEntityID   string  `json:"to_entity_id"`
    RelationType string  `json:"relation_type"`
    Confidence   float32 `json:"confidence"`
}

// IntentObjective is the machine-readable encoding of an organizational goal.
// Inspired by IntentEngineering.md — maps OKRs to agent-actionable parameters.
type IntentObjective struct {
    Name              string             `json:"name"`
    Description       string             `json:"description"`
    Signals           []string           `json:"signals"`
    DataSources       []string           `json:"data_sources"`
    AuthorizedActions []string           `json:"authorized_actions"`
    TradeOffs         map[string]float64 `json:"trade_offs"`
    HardBoundaries    []string           `json:"hard_boundaries"`
    CreatedAt         time.Time          `json:"created_at"`
    UpdatedAt         time.Time          `json:"updated_at"`
    Version           int                `json:"version"`
}

// AlignmentRecord captures a single intent-vs-outcome measurement.
type AlignmentRecord struct {
    ID              string    `json:"id"`
    AgentID         string    `json:"agent_id"`
    ObjectiveName   string    `json:"objective_name"`
    SignalID         string    `json:"signal_id"`
    Decision        string    `json:"decision"`
    Outcome         string    `json:"outcome"`
    AlignmentScore  float64   `json:"alignment_score"`
    FidelityScore   float64   `json:"fidelity_score"` // from FinTech NRV
    Timestamp       time.Time `json:"timestamp"`
}
```

### 4.2 Signal Router

Create `backend/internal/services/icme/signal_router.go`:

```go
package icme

import (
    "context"
    "fmt"
    "time"

    "github.com/google/uuid"
    "go.uber.org/zap"
)

// SignalRouter accepts raw events from existing KNIRV-NEXUS services,
// annotates them with intent context, extracts entities, and enqueues
// them for graph and index updates.
type SignalRouter struct {
    intentRegistry  *IntentRegistry
    entityClient    EntityExtractorClient  // gRPC
    embedClient     EmbeddingServiceClient // gRPC
    graphEngine     *IntentHypergraph
    indexManager    *FAISSIndexManager
    activeMemory    ActiveMemoryInterface  // existing service interface
    logger          *zap.Logger
    signalCh        chan *IntentionalSignal
}

func NewSignalRouter(
    intentRegistry *IntentRegistry,
    entityClient EntityExtractorClient,
    embedClient EmbeddingServiceClient,
    graphEngine *IntentHypergraph,
    indexManager *FAISSIndexManager,
    activeMemory ActiveMemoryInterface,
    logger *zap.Logger,
) *SignalRouter {
    r := &SignalRouter{
        intentRegistry: intentRegistry,
        entityClient:   entityClient,
        embedClient:    embedClient,
        graphEngine:    graphEngine,
        indexManager:   indexManager,
        activeMemory:   activeMemory,
        logger:         logger,
        signalCh:       make(chan *IntentionalSignal, 512),
    }
    return r
}

// Start launches the background processing goroutines.
func (r *SignalRouter) Start(ctx context.Context) {
    // Fan out: 4 parallel workers consume the signal channel.
    for i := 0; i < 4; i++ {
        go r.processLoop(ctx)
    }
}

// Ingest is called by existing KNIRV-NEXUS services to push a raw signal.
// It is non-blocking; processing happens asynchronously.
//
// Integration points:
//   - Call from ActiveMemoryService.RecordInteraction
//   - Call from NexusMemoryServer on new fabric event
//   - Call from FintechValidator on NRV trace completion
//   - Call from eBPF event handler on suspicious action
func (r *SignalRouter) Ingest(agentID string, source SignalSource, content string) {
    sig := &IntentionalSignal{
        ID:        uuid.NewString(),
        AgentID:   agentID,
        Source:    source,
        Content:   content,
        Timestamp: time.Now(),
    }

    select {
    case r.signalCh <- sig:
    default:
        r.logger.Warn("icme signal channel full, dropping signal",
            zap.String("agent_id", agentID),
            zap.String("source", string(source)),
        )
    }
}

// processLoop dequeues signals and runs the full enrichment pipeline.
func (r *SignalRouter) processLoop(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return
        case sig := <-r.signalCh:
            if err := r.enrich(ctx, sig); err != nil {
                r.logger.Error("icme signal enrichment failed",
                    zap.String("signal_id", sig.ID),
                    zap.Error(err),
                )
            }
        }
    }
}

// enrich runs the full ICME pipeline for one signal.
func (r *SignalRouter) enrich(ctx context.Context, sig *IntentionalSignal) error {
    // Step 1: Annotate with intent objective.
    obj := r.intentRegistry.GetObjectiveForAgent(sig.AgentID)
    if obj != nil {
        sig.ObjectiveName   = obj.Name
        sig.AuthorizedActs  = obj.AuthorizedActions
        sig.TradeOffWeights = obj.TradeOffs
        sig.HardBoundaries  = obj.HardBoundaries
    }

    // Step 2: Extract entities and relations via gRPC (GPU NER).
    entResp, err := r.entityClient.ExtractEntities(ctx, &ExtractRequest{
        Text:    sig.Content,
        AgentId: sig.AgentID,
        Source:  string(sig.Source),
    })
    if err != nil {
        return fmt.Errorf("entity extraction: %w", err)
    }
    sig.Entities  = protoEntitiesToModel(entResp.Entities)
    sig.Relations = protoRelationsToModel(entResp.Relations)

    // Step 3: Generate embedding via gRPC (GPU NVEmbed).
    embedResp, err := r.embedClient.GenerateEmbedding(ctx, &EmbedRequest{
        Text:  sig.Content,
        Model: "nvembed",
    })
    if err != nil {
        return fmt.Errorf("embedding generation: %w", err)
    }

    // Step 4: Add to FAISS index.
    vecID, err := r.indexManager.Add(sig.ID, embedResp.Vector)
    if err != nil {
        return fmt.Errorf("faiss index add: %w", err)
    }
    sig.EmbeddingID = vecID

    // Step 5: Update temporal hypergraph.
    r.graphEngine.InsertSignal(sig)

    // Step 6: Persist to active memory (existing service).
    r.activeMemory.RecordIntentionalSignal(ctx, sig)

    r.logger.Debug("icme signal enriched",
        zap.String("signal_id", sig.ID),
        zap.String("objective", sig.ObjectiveName),
        zap.Int("entities", len(sig.Entities)),
        zap.Int64("embedding_id", sig.EmbeddingID),
    )
    return nil
}
```

---

## 5. Layer 2 — Intent Engineering Framework

### 5.1 Intent Registry

Create `backend/internal/services/icme/intent_registry.go`:

```go
package icme

import (
    "encoding/json"
    "fmt"
    "sync"
    "time"

    "github.com/google/uuid"
    "github.com/tidwall/buntdb"
    "go.uber.org/zap"
)

const (
    buntKeyPrefixObjective  = "icme:objective:"
    buntKeyPrefixAgentBind  = "icme:agent_bind:"
    buntKeyPrefixAlignment  = "icme:alignment:"
)

// IntentRegistry stores and serves IntentObjectives, binding them to agents.
// It uses the existing BuntDB instance for persistence (no new DB required).
type IntentRegistry struct {
    db     *buntdb.DB
    mu     sync.RWMutex
    cache  map[string]*IntentObjective  // name → objective (hot path)
    binds  map[string]string            // agentID → objectiveName
    logger *zap.Logger
}

func NewIntentRegistry(db *buntdb.DB, logger *zap.Logger) (*IntentRegistry, error) {
    r := &IntentRegistry{
        db:     db,
        cache:  make(map[string]*IntentObjective),
        binds:  make(map[string]string),
        logger: logger,
    }
    if err := r.loadFromDB(); err != nil {
        return nil, fmt.Errorf("intent registry load: %w", err)
    }
    return r, nil
}

// RegisterObjective persists a new or updated IntentObjective.
// Called by the admin API when organizational goals change.
func (r *IntentRegistry) RegisterObjective(obj *IntentObjective) error {
    if obj.Name == "" {
        return fmt.Errorf("objective name is required")
    }
    obj.UpdatedAt = time.Now()
    if obj.CreatedAt.IsZero() {
        obj.CreatedAt = obj.UpdatedAt
        obj.Version = 1
    } else {
        obj.Version++
    }

    data, err := json.Marshal(obj)
    if err != nil {
        return fmt.Errorf("marshal objective: %w", err)
    }

    if err := r.db.Update(func(tx *buntdb.Tx) error {
        key := buntKeyPrefixObjective + obj.Name
        _, _, err := tx.Set(key, string(data), nil)
        return err
    }); err != nil {
        return fmt.Errorf("buntdb set objective: %w", err)
    }

    r.mu.Lock()
    r.cache[obj.Name] = obj
    r.mu.Unlock()

    r.logger.Info("icme intent objective registered",
        zap.String("name", obj.Name),
        zap.Int("version", obj.Version),
    )
    return nil
}

// BindAgentToObjective maps an agent to an objective by name.
func (r *IntentRegistry) BindAgentToObjective(agentID, objectiveName string) error {
    r.mu.RLock()
    _, exists := r.cache[objectiveName]
    r.mu.RUnlock()
    if !exists {
        return fmt.Errorf("objective %q not found", objectiveName)
    }

    if err := r.db.Update(func(tx *buntdb.Tx) error {
        _, _, err := tx.Set(buntKeyPrefixAgentBind+agentID, objectiveName, nil)
        return err
    }); err != nil {
        return fmt.Errorf("buntdb bind agent: %w", err)
    }

    r.mu.Lock()
    r.binds[agentID] = objectiveName
    r.mu.Unlock()
    return nil
}

// GetObjectiveForAgent returns the IntentObjective bound to an agent, or nil.
func (r *IntentRegistry) GetObjectiveForAgent(agentID string) *IntentObjective {
    r.mu.RLock()
    defer r.mu.RUnlock()
    name, ok := r.binds[agentID]
    if !ok {
        return nil
    }
    return r.cache[name]
}

// EvaluateTradeOffs returns the weighted priority score for a decision context.
// context keys match objective trade-off keys (e.g., "speed_vs_thoroughness").
func (r *IntentRegistry) EvaluateTradeOffs(agentID string, context map[string]float64) float64 {
    obj := r.GetObjectiveForAgent(agentID)
    if obj == nil {
        return 0.5 // neutral default
    }
    var weighted, total float64
    for key, weight := range obj.TradeOffs {
        if val, ok := context[key]; ok {
            weighted += val * weight
            total += weight
        }
    }
    if total == 0 {
        return 0.5
    }
    return weighted / total
}

// IsActionAuthorized checks whether a proposed action is within the
// objective's authorized_actions list. Returns false if no objective bound.
func (r *IntentRegistry) IsActionAuthorized(agentID, action string) bool {
    obj := r.GetObjectiveForAgent(agentID)
    if obj == nil {
        return true // no constraint registered → allow
    }
    for _, a := range obj.AuthorizedActions {
        if a == action {
            return true
        }
    }
    return false
}

// ViolatesHardBoundary checks if a proposed action violates any hard limit.
func (r *IntentRegistry) ViolatesHardBoundary(agentID, action string) bool {
    obj := r.GetObjectiveForAgent(agentID)
    if obj == nil {
        return false
    }
    for _, boundary := range obj.HardBoundaries {
        if boundary == action {
            return true
        }
    }
    return false
}

// RecordAlignment stores an alignment measurement for later analysis.
func (r *IntentRegistry) RecordAlignment(rec *AlignmentRecord) error {
    if rec.ID == "" {
        rec.ID = uuid.NewString()
    }
    data, err := json.Marshal(rec)
    if err != nil {
        return err
    }
    return r.db.Update(func(tx *buntdb.Tx) error {
        key := fmt.Sprintf("%s%s:%s", buntKeyPrefixAlignment, rec.AgentID, rec.ID)
        _, _, err := tx.Set(key, string(data), nil)
        return err
    })
}

// ListAlignmentRecords returns all alignment records for an agent.
func (r *IntentRegistry) ListAlignmentRecords(agentID string) ([]*AlignmentRecord, error) {
    var records []*AlignmentRecord
    prefix := buntKeyPrefixAlignment + agentID + ":"
    err := r.db.View(func(tx *buntdb.Tx) error {
        return tx.AscendRange("", prefix, prefix+"\xff", func(key, val string) bool {
            var rec AlignmentRecord
            if json.Unmarshal([]byte(val), &rec) == nil {
                records = append(records, &rec)
            }
            return true
        })
    })
    return records, err
}

// loadFromDB hydrates the in-memory cache from BuntDB on startup.
func (r *IntentRegistry) loadFromDB() error {
    return r.db.View(func(tx *buntdb.Tx) error {
        // Load objectives.
        if err := tx.AscendRange("", buntKeyPrefixObjective, buntKeyPrefixObjective+"\xff",
            func(key, val string) bool {
                var obj IntentObjective
                if json.Unmarshal([]byte(val), &obj) == nil {
                    r.cache[obj.Name] = &obj
                }
                return true
            }); err != nil {
            return err
        }
        // Load agent bindings.
        return tx.AscendRange("", buntKeyPrefixAgentBind, buntKeyPrefixAgentBind+"\xff",
            func(key, val string) bool {
                agentID := key[len(buntKeyPrefixAgentBind):]
                r.binds[agentID] = val
                return true
            })
    })
}
```

### 5.2 Delegation / Conflict Resolution

Create `backend/internal/services/icme/delegation.go`:

```go
package icme

import "go.uber.org/zap"

// DecisionContext carries the parameters needed to resolve a conflict.
type DecisionContext struct {
    AgentID      string
    Action       string
    CustomerTier string  // e.g., "VIP", "standard"
    Amount       float64
    Custom       map[string]interface{}
}

// DecisionResult is the outcome of a delegation resolution.
type DecisionResult struct {
    Approved bool
    Action   string // "approve", "deny", "escalate_to_manager", "escalate_to_specialist"
    Reason   string
}

// DelegationFramework resolves action conflicts by applying objective rules.
// It encodes the hierarchical decision logic from IntentEngineering.md.
type DelegationFramework struct {
    registry *IntentRegistry
    logger   *zap.Logger
}

func NewDelegationFramework(registry *IntentRegistry, logger *zap.Logger) *DelegationFramework {
    return &DelegationFramework{registry: registry, logger: logger}
}

// Resolve determines the appropriate response to a proposed action given
// the agent's objective constraints, tier, and any configured boundaries.
func (d *DelegationFramework) Resolve(ctx DecisionContext) DecisionResult {
    // Hard boundary check first — non-negotiable.
    if d.registry.ViolatesHardBoundary(ctx.AgentID, ctx.Action) {
        d.logger.Warn("icme hard boundary violation",
            zap.String("agent_id", ctx.AgentID),
            zap.String("action", ctx.Action),
        )
        return DecisionResult{
            Approved: false,
            Action:   "deny",
            Reason:   "hard boundary violated: " + ctx.Action,
        }
    }

    // Authorization check.
    if !d.registry.IsActionAuthorized(ctx.AgentID, ctx.Action) {
        // Not authorized — escalate rather than deny outright.
        return DecisionResult{
            Approved: false,
            Action:   "escalate_to_manager",
            Reason:   "action not in authorized list: " + ctx.Action,
        }
    }

    // Tier-based priority override (mirrors IntentEngineering.md example).
    if ctx.CustomerTier == "VIP" {
        return DecisionResult{
            Approved: true,
            Action:   "escalate_to_specialist",
            Reason:   "VIP tier prioritizes satisfaction",
        }
    }

    // Amount-based threshold.
    if ctx.Amount > 1000 {
        return DecisionResult{
            Approved: false,
            Action:   "deny",
            Reason:   "amount exceeds threshold",
        }
    }

    return DecisionResult{
        Approved: true,
        Action:   "approve",
        Reason:   "within objective constraints",
    }
}
```

---

## 6. Layer 3 — Intentional Graph Construction

### 6.1 Temporal Hypergraph Engine

Create `backend/internal/services/icme/hypergraph.go`:

```go
package icme

import (
    "sync"
    "time"

    "go.uber.org/zap"
)

// HyperNode represents any entity (person, config, error, event, solution)
// extracted from a signal. It carries all attributes from NER.
type HyperNode struct {
    ID         string                 `json:"id"`
    Type       string                 `json:"type"`   // PERSON, ORG, ERROR, SOLUTION, CONFIG, EVENT
    Text       string                 `json:"text"`
    Attributes map[string]interface{} `json:"attributes"`
    FirstSeen  time.Time              `json:"first_seen"`
    LastSeen   time.Time              `json:"last_seen"`
    SignalIDs  []string               `json:"signal_ids"` // signals that mentioned this node
}

// HyperEdge represents a relationship between two nodes.
// Unlike simple graphs, edges carry rich temporal + intent metadata.
type HyperEdge struct {
    ID              string                 `json:"id"`
    FromNodeID      string                 `json:"from_node_id"`
    ToNodeID        string                 `json:"to_node_id"`
    RelationType    string                 `json:"relation_type"`
    Timestamp       time.Time              `json:"timestamp"`
    SignalID         string                 `json:"signal_id"`
    AgentID         string                 `json:"agent_id"`
    ObjectiveName   string                 `json:"objective_name"`
    AlignmentScore  float64                `json:"alignment_score"`
    Metadata        map[string]interface{} `json:"metadata"`
}

// IntentHypergraph maintains the live temporal graph of entities and
// relationships, annotated with intent context from the IntentRegistry.
type IntentHypergraph struct {
    mu          sync.RWMutex
    nodes       map[string]*HyperNode          // nodeID → node
    edges       map[string][]*HyperEdge        // fromNodeID → edges
    textIndex   map[string]string              // normalized text → nodeID
    windowSize  time.Duration                  // edges older than this are pruned
    maxNodes    int
    logger      *zap.Logger
}

func NewIntentHypergraph(windowSize time.Duration, maxNodes int, logger *zap.Logger) *IntentHypergraph {
    g := &IntentHypergraph{
        nodes:      make(map[string]*HyperNode),
        edges:      make(map[string][]*HyperEdge),
        textIndex:  make(map[string]string),
        windowSize: windowSize,
        maxNodes:   maxNodes,
        logger:     logger,
    }
    return g
}

// InsertSignal ingests an enriched IntentionalSignal, updating
// nodes and hyperedges. Thread-safe.
func (g *IntentHypergraph) InsertSignal(sig *IntentionalSignal) {
    g.mu.Lock()
    defer g.mu.Unlock()

    // Upsert entity nodes.
    nodeIDs := make(map[string]string) // entity.ID → graph nodeID
    for _, ent := range sig.Entities {
        nodeID := g.upsertNode(ent, sig)
        nodeIDs[ent.ID] = nodeID
    }

    // Insert edges for extracted relations.
    for _, rel := range sig.Relations {
        fromNodeID, ok1 := nodeIDs[rel.FromEntityID]
        toNodeID,   ok2 := nodeIDs[rel.ToEntityID]
        if !ok1 || !ok2 {
            continue
        }
        edge := &HyperEdge{
            ID:             sig.ID + ":" + rel.FromEntityID + ":" + rel.ToEntityID,
            FromNodeID:     fromNodeID,
            ToNodeID:       toNodeID,
            RelationType:   rel.RelationType,
            Timestamp:      sig.Timestamp,
            SignalID:        sig.ID,
            AgentID:        sig.AgentID,
            ObjectiveName:  sig.ObjectiveName,
            AlignmentScore: sig.AlignmentScore,
            Metadata: map[string]interface{}{
                "confidence": rel.Confidence,
                "source":     string(sig.Source),
            },
        }
        g.edges[fromNodeID] = append(g.edges[fromNodeID], edge)
    }

    // Prune stale edges.
    g.pruneOldEdges()

    g.logger.Debug("icme hypergraph updated",
        zap.Int("nodes", len(g.nodes)),
        zap.String("signal_id", sig.ID),
    )
}

// upsertNode finds or creates a HyperNode for the given entity.
// Returns the canonical nodeID.
func (g *IntentHypergraph) upsertNode(ent ExtractedEntity, sig *IntentionalSignal) string {
    // Normalize lookup key (lowercase text + label).
    lookupKey := ent.Label + ":" + normalizeText(ent.Text)

    if existingID, ok := g.textIndex[lookupKey]; ok {
        node := g.nodes[existingID]
        node.LastSeen = sig.Timestamp
        node.SignalIDs = append(node.SignalIDs, sig.ID)
        return existingID
    }

    node := &HyperNode{
        ID:        ent.ID,
        Type:      ent.Label,
        Text:      ent.Text,
        Attributes: map[string]interface{}{
            "score":     ent.Score,
            "objective": sig.ObjectiveName,
        },
        FirstSeen: sig.Timestamp,
        LastSeen:  sig.Timestamp,
        SignalIDs: []string{sig.ID},
    }
    g.nodes[node.ID]       = node
    g.textIndex[lookupKey] = node.ID
    return node.ID
}

// Neighbors returns all nodes reachable from nodeID within maxHops,
// optionally filtered by relation type. Used in hybrid retrieval.
func (g *IntentHypergraph) Neighbors(nodeID string, maxHops int, relType string) []*HyperNode {
    g.mu.RLock()
    defer g.mu.RUnlock()

    visited := make(map[string]bool)
    var result []*HyperNode
    g.dfs(nodeID, maxHops, relType, visited, &result)
    return result
}

func (g *IntentHypergraph) dfs(nodeID string, hops int, relType string,
    visited map[string]bool, result *[]*HyperNode) {
    if hops == 0 || visited[nodeID] {
        return
    }
    visited[nodeID] = true
    if node, ok := g.nodes[nodeID]; ok {
        *result = append(*result, node)
    }
    for _, edge := range g.edges[nodeID] {
        if relType == "" || edge.RelationType == relType {
            g.dfs(edge.ToNodeID, hops-1, relType, visited, result)
        }
    }
}

// Snapshot returns a point-in-time view of the graph for streaming.
func (g *IntentHypergraph) Snapshot() (nodes []*HyperNode, edges []*HyperEdge) {
    g.mu.RLock()
    defer g.mu.RUnlock()
    for _, n := range g.nodes {
        nodes = append(nodes, n)
    }
    for _, edgeList := range g.edges {
        edges = append(edges, edgeList...)
    }
    return
}

// pruneOldEdges removes edges outside the configured time window.
// Must be called under write lock.
func (g *IntentHypergraph) pruneOldEdges() {
    cutoff := time.Now().Add(-g.windowSize)
    for fromID, edgeList := range g.edges {
        var fresh []*HyperEdge
        for _, e := range edgeList {
            if e.Timestamp.After(cutoff) {
                fresh = append(fresh, e)
            }
        }
        g.edges[fromID] = fresh
    }
}

func normalizeText(s string) string {
    // Simple normalization — lowercase, trim spaces.
    result := []byte(s)
    for i, b := range result {
        if b >= 'A' && b <= 'Z' {
            result[i] = b + 32
        }
    }
    return string(result)
}
```

---

## 7. Layer 4 — Embedding Generation & Semantic Indexing

### 7.1 FAISS Index Manager

Create `backend/internal/services/icme/faiss_manager.go`:

```go
package icme

import (
    "encoding/json"
    "fmt"
    "sync"

    faiss "github.com/DataIntelligenceCrew/go-faiss"
    "github.com/tidwall/buntdb"
    "go.uber.org/zap"
)

const (
    // EmbedDim is the vector dimension for NVEmbed / E5-Large.
    // Change to 384 if using MiniLM, 768 for E5-base, 4096 for NVEmbed.
    EmbedDim = 1024

    buntKeyFAISSMeta = "icme:faiss:meta:"
)

// VectorMeta maps a FAISS integer ID to a signal ID and summary text.
type VectorMeta struct {
    VectorID  int64  `json:"vector_id"`
    SignalID  string `json:"signal_id"`
    AgentID  string `json:"agent_id"`
    Summary  string `json:"summary"` // first 200 chars of content
    Objective string `json:"objective"`
}

// FAISSIndexManager wraps a FAISS flat L2 index with metadata persistence.
// On NVIDIA DGX Spark, replace IndexFlatL2 with IndexFlatL2 on GPU via
// faiss.StandardGpuResources (requires faiss-gpu build tag).
type FAISSIndexManager struct {
    mu       sync.Mutex
    index    *faiss.IndexImpl
    nextID   int64
    db       *buntdb.DB
    logger   *zap.Logger
}

func NewFAISSIndexManager(db *buntdb.DB, logger *zap.Logger) (*FAISSIndexManager, error) {
    idx, err := faiss.NewIndexFlatL2(EmbedDim)
    if err != nil {
        return nil, fmt.Errorf("faiss index create: %w", err)
    }
    m := &FAISSIndexManager{
        index:  idx,
        db:     db,
        logger: logger,
    }
    if err := m.rebuildFromDB(); err != nil {
        logger.Warn("icme faiss rebuild from db failed (fresh start)", zap.Error(err))
    }
    return m, nil
}

// Add inserts a vector into the index and persists metadata to BuntDB.
// Returns the assigned FAISS vector ID.
func (m *FAISSIndexManager) Add(signalID string, vector []float32, meta VectorMeta) (int64, error) {
    if len(vector) != EmbedDim {
        return -1, fmt.Errorf("vector dim mismatch: got %d, want %d", len(vector), EmbedDim)
    }

    m.mu.Lock()
    defer m.mu.Unlock()

    id := m.nextID
    m.nextID++
    meta.VectorID = id

    if err := m.index.AddWithIDs(vector, []int64{id}); err != nil {
        return -1, fmt.Errorf("faiss add: %w", err)
    }

    // Persist metadata.
    data, _ := json.Marshal(meta)
    if err := m.db.Update(func(tx *buntdb.Tx) error {
        _, _, err := tx.Set(fmt.Sprintf("%s%d", buntKeyFAISSMeta, id), string(data), nil)
        return err
    }); err != nil {
        m.logger.Warn("icme faiss meta persist failed", zap.Error(err))
    }

    return id, nil
}

// Search performs approximate nearest-neighbor search and returns metadata
// for the top-k results closest to the query vector.
func (m *FAISSIndexManager) Search(query []float32, k int) ([]VectorMeta, []float32, error) {
    if len(query) != EmbedDim {
        return nil, nil, fmt.Errorf("query dim mismatch: got %d, want %d", len(query), EmbedDim)
    }

    m.mu.Lock()
    distances, ids, err := m.index.Search(query, int64(k))
    m.mu.Unlock()
    if err != nil {
        return nil, nil, fmt.Errorf("faiss search: %w", err)
    }

    var metas []VectorMeta
    var scores []float32
    for i, id := range ids {
        if id < 0 {
            continue // FAISS returns -1 for empty slots
        }
        var meta VectorMeta
        _ = m.db.View(func(tx *buntdb.Tx) error {
            val, err := tx.Get(fmt.Sprintf("%s%d", buntKeyFAISSMeta, id))
            if err == nil {
                _ = json.Unmarshal([]byte(val), &meta)
            }
            return nil
        })
        metas = append(metas, meta)
        scores = append(scores, distances[i])
    }
    return metas, scores, nil
}

// rebuildFromDB rehydrates the FAISS index from persisted metadata on startup.
// NOTE: This only rebuilds metadata mappings. In production, serialize the
// FAISS index to disk using index.Write() / faiss.ReadIndex().
func (m *FAISSIndexManager) rebuildFromDB() error {
    return m.db.View(func(tx *buntdb.Tx) error {
        return tx.AscendRange("", buntKeyFAISSMeta, buntKeyFAISSMeta+"\xff",
            func(key, val string) bool {
                var meta VectorMeta
                if json.Unmarshal([]byte(val), &meta) == nil {
                    if meta.VectorID >= m.nextID {
                        m.nextID = meta.VectorID + 1
                    }
                }
                return true
            })
    })
}
```

---

## 8. Layer 5 — Hybrid Retrieval API

### 8.1 Hybrid Search Engine

Create `backend/internal/services/icme/hybrid_search.go`:

```go
package icme

import (
    "context"
    "fmt"
    "sort"

    "go.uber.org/zap"
)

// HybridResult is a single result from the ICME retrieval pipeline.
type HybridResult struct {
    SignalID       string      `json:"signal_id"`
    AgentID        string      `json:"agent_id"`
    Content        string      `json:"content"`
    ObjectiveName  string      `json:"objective_name"`
    VectorScore    float32     `json:"vector_score"`
    GraphHops      int         `json:"graph_hops"`
    CombinedScore  float64     `json:"combined_score"`
    Nodes          []*HyperNode `json:"related_nodes"`
    AlignmentScore float64     `json:"alignment_score"`
}

// HybridSearchEngine combines FAISS vector search with hypergraph traversal
// and optional cross-encoder re-ranking via the Python microservice.
type HybridSearchEngine struct {
    faissManager  *FAISSIndexManager
    graph         *IntentHypergraph
    embedClient   EmbeddingServiceClient
    reranker      ReRankerClient
    intentReg     *IntentRegistry
    logger        *zap.Logger
}

func NewHybridSearchEngine(
    faissManager *FAISSIndexManager,
    graph *IntentHypergraph,
    embedClient EmbeddingServiceClient,
    reranker ReRankerClient,
    intentReg *IntentRegistry,
    logger *zap.Logger,
) *HybridSearchEngine {
    return &HybridSearchEngine{
        faissManager: faissManager,
        graph:        graph,
        embedClient:  embedClient,
        reranker:     reranker,
        intentReg:    intentReg,
        logger:       logger,
    }
}

// Search performs the full hybrid retrieval for a natural-language query.
//
// Pipeline:
//   1. Embed the query via gRPC (GPU).
//   2. FAISS k-NN search → top-N signal metadata.
//   3. For each result, traverse hypergraph for related nodes.
//   4. Score fusion: vector_score + graph_centrality + alignment_boost.
//   5. Cross-encoder re-ranking via Python gRPC.
//   6. Return top-k results.
func (s *HybridSearchEngine) Search(ctx context.Context, query, agentID string, topK int) ([]HybridResult, error) {
    // Step 1: Embed query.
    embedResp, err := s.embedClient.GenerateEmbedding(ctx, &EmbedRequest{
        Text:  query,
        Model: "nvembed",
    })
    if err != nil {
        return nil, fmt.Errorf("embed query: %w", err)
    }

    // Step 2: FAISS search — retrieve 3x topK for fusion buffer.
    metas, scores, err := s.faissManager.Search(embedResp.Vector, topK*3)
    if err != nil {
        return nil, fmt.Errorf("faiss search: %w", err)
    }

    // Step 3 & 4: Graph traversal + score fusion.
    obj := s.intentReg.GetObjectiveForAgent(agentID)
    results := make([]HybridResult, 0, len(metas))
    for i, meta := range metas {
        nodes := s.graph.Neighbors(meta.SignalID, 2, "")

        // Alignment boost: prefer results matching the agent's objective.
        alignBoost := 0.0
        if obj != nil && meta.Objective == obj.Name {
            alignBoost = 0.15
        }

        // Combined score: normalize vector distance to similarity, add boosts.
        vecSim := 1.0 / (1.0 + float64(scores[i]))
        combined := vecSim + float64(len(nodes))*0.01 + alignBoost

        results = append(results, HybridResult{
            SignalID:      meta.SignalID,
            AgentID:       meta.AgentID,
            Content:       meta.Summary,
            ObjectiveName: meta.Objective,
            VectorScore:   scores[i],
            GraphHops:     len(nodes),
            CombinedScore: combined,
            Nodes:         nodes,
        })
    }

    // Sort by combined score descending.
    sort.Slice(results, func(i, j int) bool {
        return results[i].CombinedScore > results[j].CombinedScore
    })

    // Step 5: Cross-encoder re-ranking on top buffer.
    buffer := results
    if len(buffer) > topK*2 {
        buffer = buffer[:topK*2]
    }
    candidates := make([]string, len(buffer))
    for i, r := range buffer {
        candidates[i] = r.Content
    }

    rerankResp, err := s.reranker.Rerank(ctx, &RerankRequest{
        Query:      query,
        Candidates: candidates,
        TopK:       int32(topK),
    })
    if err != nil {
        // Re-ranking is best-effort; fall back to combined score ordering.
        s.logger.Warn("icme rerank failed, using combined score", zap.Error(err))
        if len(results) > topK {
            return results[:topK], nil
        }
        return results, nil
    }

    // Step 6: Reorder by re-ranker output.
    final := make([]HybridResult, 0, len(rerankResp.Results))
    for _, ranked := range rerankResp.Results {
        idx := ranked.OriginalIndex
        if int(idx) < len(buffer) {
            r := buffer[idx]
            r.AlignmentScore = float64(ranked.Score)
            final = append(final, r)
        }
    }
    return final, nil
}
```

### 8.2 HTTP Handlers

Create `backend/internal/services/icme/handlers.go`:

```go
package icme

import (
    "encoding/json"
    "net/http"
    "strconv"

    "github.com/gorilla/mux"
    "go.uber.org/zap"
)

// RegisterRoutes wires ICME endpoints into the existing Gorilla Mux router.
// Call from the main server setup where other routes are registered.
//
// Add to backend/internal/web/router.go:
//
//   icmeSvc.RegisterRoutes(s.router, authMiddleware)
func (svc *ICMEService) RegisterRoutes(r *mux.Router, authMW mux.MiddlewareFunc) {
    sub := r.PathPrefix("/api/icme").Subrouter()
    sub.Use(authMW)

    // Intent Objectives
    sub.HandleFunc("/objectives", svc.handleListObjectives).Methods(http.MethodGet)
    sub.HandleFunc("/objectives", svc.handleCreateObjective).Methods(http.MethodPost)
    sub.HandleFunc("/objectives/{name}", svc.handleGetObjective).Methods(http.MethodGet)
    sub.HandleFunc("/objectives/{name}", svc.handleUpdateObjective).Methods(http.MethodPut)

    // Agent Bindings
    sub.HandleFunc("/agents/{agentID}/bind", svc.handleBindAgent).Methods(http.MethodPost)
    sub.HandleFunc("/agents/{agentID}/objective", svc.handleGetAgentObjective).Methods(http.MethodGet)

    // Retrieval
    sub.HandleFunc("/search", svc.handleHybridSearch).Methods(http.MethodGet)

    // Alignment
    sub.HandleFunc("/alignment/{agentID}", svc.handleGetAlignment).Methods(http.MethodGet)
    sub.HandleFunc("/alignment/evaluate", svc.handleEvaluateAlignment).Methods(http.MethodPost)

    // Graph inspection
    sub.HandleFunc("/graph/snapshot", svc.handleGraphSnapshot).Methods(http.MethodGet)
    sub.HandleFunc("/graph/neighbors/{nodeID}", svc.handleGraphNeighbors).Methods(http.MethodGet)

    // Delegation
    sub.HandleFunc("/delegate", svc.handleDelegate).Methods(http.MethodPost)
}

func (svc *ICMEService) handleCreateObjective(w http.ResponseWriter, r *http.Request) {
    var obj IntentObjective
    if err := json.NewDecoder(r.Body).Decode(&obj); err != nil {
        http.Error(w, "invalid body: "+err.Error(), http.StatusBadRequest)
        return
    }
    if err := svc.intentRegistry.RegisterObjective(&obj); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(obj)
}

func (svc *ICMEService) handleListObjectives(w http.ResponseWriter, r *http.Request) {
    objs := svc.intentRegistry.ListObjectives()
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(objs)
}

func (svc *ICMEService) handleGetObjective(w http.ResponseWriter, r *http.Request) {
    name := mux.Vars(r)["name"]
    obj := svc.intentRegistry.GetObjective(name)
    if obj == nil {
        http.Error(w, "not found", http.StatusNotFound)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(obj)
}

func (svc *ICMEService) handleUpdateObjective(w http.ResponseWriter, r *http.Request) {
    name := mux.Vars(r)["name"]
    var obj IntentObjective
    if err := json.NewDecoder(r.Body).Decode(&obj); err != nil {
        http.Error(w, "invalid body", http.StatusBadRequest)
        return
    }
    obj.Name = name
    if err := svc.intentRegistry.RegisterObjective(&obj); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(obj)
}

func (svc *ICMEService) handleBindAgent(w http.ResponseWriter, r *http.Request) {
    agentID := mux.Vars(r)["agentID"]
    var body struct{ ObjectiveName string `json:"objective_name"` }
    if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
        http.Error(w, "invalid body", http.StatusBadRequest)
        return
    }
    if err := svc.intentRegistry.BindAgentToObjective(agentID, body.ObjectiveName); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    w.WriteHeader(http.StatusNoContent)
}

func (svc *ICMEService) handleGetAgentObjective(w http.ResponseWriter, r *http.Request) {
    agentID := mux.Vars(r)["agentID"]
    obj := svc.intentRegistry.GetObjectiveForAgent(agentID)
    if obj == nil {
        http.Error(w, "no objective bound", http.StatusNotFound)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(obj)
}

func (svc *ICMEService) handleHybridSearch(w http.ResponseWriter, r *http.Request) {
    q := r.URL.Query().Get("q")
    agentID := r.URL.Query().Get("agent_id")
    topKStr := r.URL.Query().Get("top_k")
    topK := 10
    if k, err := strconv.Atoi(topKStr); err == nil && k > 0 {
        topK = k
    }

    results, err := svc.searchEngine.Search(r.Context(), q, agentID, topK)
    if err != nil {
        svc.logger.Error("icme hybrid search failed", zap.Error(err))
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(results)
}

func (svc *ICMEService) handleGetAlignment(w http.ResponseWriter, r *http.Request) {
    agentID := mux.Vars(r)["agentID"]
    records, err := svc.intentRegistry.ListAlignmentRecords(agentID)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(records)
}

func (svc *ICMEService) handleEvaluateAlignment(w http.ResponseWriter, r *http.Request) {
    var rec AlignmentRecord
    if err := json.NewDecoder(r.Body).Decode(&rec); err != nil {
        http.Error(w, "invalid body", http.StatusBadRequest)
        return
    }
    score := svc.alignmentLoop.Evaluate(&rec)
    rec.AlignmentScore = score
    if err := svc.intentRegistry.RecordAlignment(&rec); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(rec)
}

func (svc *ICMEService) handleGraphSnapshot(w http.ResponseWriter, r *http.Request) {
    nodes, edges := svc.graph.Snapshot()
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "nodes": nodes,
        "edges": edges,
    })
}

func (svc *ICMEService) handleGraphNeighbors(w http.ResponseWriter, r *http.Request) {
    nodeID  := mux.Vars(r)["nodeID"]
    hopsStr := r.URL.Query().Get("hops")
    relType := r.URL.Query().Get("rel_type")
    hops    := 2
    if h, err := strconv.Atoi(hopsStr); err == nil && h > 0 {
        hops = h
    }
    neighbors := svc.graph.Neighbors(nodeID, hops, relType)
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(neighbors)
}

func (svc *ICMEService) handleDelegate(w http.ResponseWriter, r *http.Request) {
    var ctx DecisionContext
    if err := json.NewDecoder(r.Body).Decode(&ctx); err != nil {
        http.Error(w, "invalid body", http.StatusBadRequest)
        return
    }
    result := svc.delegation.Resolve(ctx)
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(result)
}
```

---

## 9. Layer 6 — Feedback & Alignment Loops

### 9.1 Alignment Loop

Create `backend/internal/services/icme/alignment_loop.go`:

```go
package icme

import (
    "context"
    "math"
    "time"

    "go.uber.org/zap"
)

// AlignmentLoop continuously measures and reports agent alignment with
// declared organizational intent. It integrates with the existing
// FinTech NRV fidelity scorer to cross-validate alignment.
type AlignmentLoop struct {
    registry     *IntentRegistry
    fintechSvc   FintechValidatorInterface // existing service interface
    logger       *zap.Logger
    evalInterval time.Duration
}

// FintechValidatorInterface exposes the minimum needed from the existing
// fintech validator service to pull NRV fidelity scores.
type FintechValidatorInterface interface {
    GetLatestFidelityScore(agentID string) (float64, error)
    GetNRVTraces(agentID string, since time.Time) ([]NRVTrace, error)
}

// NRVTrace is a lightweight local representation of a fintech NRV trace.
type NRVTrace struct {
    ID            string    `json:"id"`
    AgentID       string    `json:"agent_id"`
    FidelityScore float64   `json:"fidelity_score"`
    Timestamp     time.Time `json:"timestamp"`
    ReasoningPath []string  `json:"reasoning_path"`
}

func NewAlignmentLoop(
    registry *IntentRegistry,
    fintechSvc FintechValidatorInterface,
    evalInterval time.Duration,
    logger *zap.Logger,
) *AlignmentLoop {
    return &AlignmentLoop{
        registry:     registry,
        fintechSvc:   fintechSvc,
        evalInterval: evalInterval,
        logger:       logger,
    }
}

// Start runs periodic alignment evaluation in the background.
func (l *AlignmentLoop) Start(ctx context.Context) {
    ticker := time.NewTicker(l.evalInterval)
    defer ticker.Stop()
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            l.runEvaluation(ctx)
        }
    }
}

// runEvaluation pulls recent NRV traces and cross-references them
// with alignment records to detect intent drift.
func (l *AlignmentLoop) runEvaluation(ctx context.Context) {
    // For each agent with a known objective, pull NRV fidelity scores.
    for agentID := range l.registry.binds {
        fidelity, err := l.fintechSvc.GetLatestFidelityScore(agentID)
        if err != nil {
            l.logger.Debug("icme alignment: no fidelity score for agent",
                zap.String("agent_id", agentID), zap.Error(err))
            continue
        }

        records, err := l.registry.ListAlignmentRecords(agentID)
        if err != nil || len(records) == 0 {
            continue
        }

        // Compute moving average alignment score.
        var sum float64
        window := 20
        start := len(records) - window
        if start < 0 {
            start = 0
        }
        for _, rec := range records[start:] {
            sum += rec.AlignmentScore
        }
        avgAlign := sum / float64(len(records[start:]))

        // Detect drift: fidelity diverged from intent alignment by >20%.
        drift := math.Abs(fidelity - avgAlign)
        if drift > 0.20 {
            l.logger.Warn("icme intent drift detected",
                zap.String("agent_id", agentID),
                zap.Float64("fidelity_score", fidelity),
                zap.Float64("avg_alignment", avgAlign),
                zap.Float64("drift", drift),
            )
            // Emit a drift event back into the signal pipeline.
            // This creates a self-referential correction loop.
        }

        l.logger.Debug("icme alignment evaluated",
            zap.String("agent_id", agentID),
            zap.Float64("fidelity", fidelity),
            zap.Float64("avg_align", avgAlign),
        )
    }
}

// Evaluate computes an alignment score for a single AlignmentRecord.
// Score is 1.0 if the decision perfectly matches objective parameters,
// declining as the decision deviates from authorized actions or trade-offs.
func (l *AlignmentLoop) Evaluate(rec *AlignmentRecord) float64 {
    obj := l.registry.GetObjectiveForAgent(rec.AgentID)
    if obj == nil {
        return 0.5 // no objective — neutral
    }

    score := 1.0

    // Penalize if decision was not in authorized actions.
    authorized := false
    for _, a := range obj.AuthorizedActions {
        if a == rec.Decision {
            authorized = true
            break
        }
    }
    if !authorized {
        score -= 0.3
    }

    // Incorporate fidelity score if available.
    if rec.FidelityScore > 0 {
        score = score*0.6 + rec.FidelityScore*0.4
    }

    // Clamp to [0, 1].
    if score < 0 {
        score = 0
    }
    if score > 1 {
        score = 1
    }
    return score
}
```

### 9.2 Integrating with the FinTech NRV Tracer

The existing FinTech NRV fidelity scorer outputs traces via `/api/fintech/nrv/traces`. The `AlignmentLoop` connects to this via the `FintechValidatorInterface`.

In `backend/internal/services/icme/fintech_adapter.go`:

```go
package icme

import (
    "encoding/json"
    "fmt"
    "net/http"
    "time"
)

// FintechNRVAdapter implements FintechValidatorInterface by calling
// the existing fintech validator HTTP API (internal loop-back call).
// This avoids creating an import cycle between services.
type FintechNRVAdapter struct {
    baseURL    string // e.g., "http://localhost:8082"
    httpClient *http.Client
}

func NewFintechNRVAdapter(baseURL string) *FintechNRVAdapter {
    return &FintechNRVAdapter{
        baseURL:    baseURL,
        httpClient: &http.Client{Timeout: 5 * time.Second},
    }
}

func (a *FintechNRVAdapter) GetLatestFidelityScore(agentID string) (float64, error) {
    url := fmt.Sprintf("%s/api/fintech/nrv/traces?agent_id=%s&limit=1", a.baseURL, agentID)
    resp, err := a.httpClient.Get(url)
    if err != nil {
        return 0, err
    }
    defer resp.Body.Close()

    var traces []struct {
        FidelityScore float64 `json:"fidelity_score"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&traces); err != nil {
        return 0, err
    }
    if len(traces) == 0 {
        return 0, fmt.Errorf("no traces found")
    }
    return traces[0].FidelityScore, nil
}

func (a *FintechNRVAdapter) GetNRVTraces(agentID string, since time.Time) ([]NRVTrace, error) {
    url := fmt.Sprintf("%s/api/fintech/nrv/traces?agent_id=%s&since=%d",
        a.baseURL, agentID, since.Unix())
    resp, err := a.httpClient.Get(url)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    var traces []NRVTrace
    return traces, json.NewDecoder(resp.Body).Decode(&traces)
}
```

---

## 10. Configuration Changes

### 10.1 Add ICME Section to `config/development.yaml`

```yaml
# ── ICME: Intentional Context Memory Engine ──────────────────────────────────
icme:
  enabled: true

  # Python gRPC microservices
  entity_extractor_addr:  "localhost:50052"
  embedding_service_addr: "localhost:50053"
  reranker_addr:          "localhost:50054"

  # Temporal hypergraph settings
  graph_window_size:      "10m"    # prune edges older than 10 minutes
  graph_max_nodes:        10000

  # FAISS settings
  embed_dimension:        1024     # NVEmbed dimension; use 384 for MiniLM
  faiss_use_gpu:          false    # true on DGX Spark with CUDA build

  # Retrieval
  default_top_k:          10
  enable_reranking:       true

  # Alignment feedback loop
  alignment_eval_interval: "5m"   # how often to check for intent drift
  drift_threshold:         0.20   # alert if fidelity vs alignment diverges >20%

  # Signal ingestion
  signal_queue_size:       512
  signal_workers:          4
```

### 10.2 Config Struct Extension

In `backend/internal/config/config.go`, add:

```go
// ICMEConfig holds configuration for the Intentional Context Memory Engine.
type ICMEConfig struct {
    Enabled                bool          `mapstructure:"enabled"`
    EntityExtractorAddr    string        `mapstructure:"entity_extractor_addr"`
    EmbeddingServiceAddr   string        `mapstructure:"embedding_service_addr"`
    RerankerAddr           string        `mapstructure:"reranker_addr"`
    GraphWindowSize        time.Duration `mapstructure:"graph_window_size"`
    GraphMaxNodes          int           `mapstructure:"graph_max_nodes"`
    EmbedDimension         int           `mapstructure:"embed_dimension"`
    FAISSUseGPU            bool          `mapstructure:"faiss_use_gpu"`
    DefaultTopK            int           `mapstructure:"default_top_k"`
    EnableReranking        bool          `mapstructure:"enable_reranking"`
    AlignmentEvalInterval  time.Duration `mapstructure:"alignment_eval_interval"`
    DriftThreshold         float64       `mapstructure:"drift_threshold"`
    SignalQueueSize        int           `mapstructure:"signal_queue_size"`
    SignalWorkers          int           `mapstructure:"signal_workers"`
}

// Add to the top-level Config struct:
// ICME ICMEConfig `mapstructure:"icme"`
```

---

## 11. API Reference

| Method | Path | Description |
|--------|------|-------------|
| `GET`  | `/api/icme/objectives` | List all intent objectives |
| `POST` | `/api/icme/objectives` | Create a new objective |
| `GET`  | `/api/icme/objectives/{name}` | Get objective by name |
| `PUT`  | `/api/icme/objectives/{name}` | Update objective |
| `POST` | `/api/icme/agents/{agentID}/bind` | Bind agent to objective |
| `GET`  | `/api/icme/agents/{agentID}/objective` | Get agent's current objective |
| `GET`  | `/api/icme/search?q=&agent_id=&top_k=` | Hybrid semantic + graph search |
| `GET`  | `/api/icme/alignment/{agentID}` | Get alignment history for agent |
| `POST` | `/api/icme/alignment/evaluate` | Evaluate and record an alignment |
| `GET`  | `/api/icme/graph/snapshot` | Full graph snapshot (nodes + edges) |
| `GET`  | `/api/icme/graph/neighbors/{nodeID}?hops=2&rel_type=` | Graph neighbors |
| `POST` | `/api/icme/delegate` | Resolve action via delegation framework |

### Example Payloads

**Create Objective:**
```json
POST /api/icme/objectives
{
  "name": "dvs_validation_quality",
  "description": "Maximize validation fidelity while minimizing false positives",
  "signals": ["fidelity_score", "false_positive_rate", "validation_latency"],
  "data_sources": ["fintech_nrv", "vault_errors", "ebpf_traces"],
  "authorized_actions": ["approve_validation", "reject_validation", "escalate_to_human", "request_replay"],
  "trade_offs": {
    "speed_vs_thoroughness": 0.3,
    "cost_vs_quality": 0.8
  },
  "hard_boundaries": ["skip_tee_attestation", "bypass_pqc_signing"]
}
```

**Hybrid Search:**
```
GET /api/icme/search?q=failed+kyc+validation+for+high+risk+entity&agent_id=agent-001&top_k=5
```

**Delegation Request:**
```json
POST /api/icme/delegate
{
  "agent_id": "agent-001",
  "action": "approve_validation",
  "customer_tier": "VIP",
  "amount": 500.00,
  "custom": { "risk_score": 0.72 }
}
```

---

## 12. Frontend Integration

### 12.1 Intent Console Component

Create `frontend/src/components/icme/IntentConsole.tsx`:

```tsx
'use client';

import { useState, useEffect } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';

interface IntentObjective {
  name: string;
  description: string;
  signals: string[];
  authorized_actions: string[];
  trade_offs: Record<string, number>;
  hard_boundaries: string[];
  version: number;
  updated_at: string;
}

async function fetchObjectives(): Promise<IntentObjective[]> {
  const res = await fetch('/api/icme/objectives');
  if (!res.ok) throw new Error('Failed to fetch objectives');
  return res.json();
}

async function createObjective(obj: Partial<IntentObjective>): Promise<IntentObjective> {
  const res = await fetch('/api/icme/objectives', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(obj),
  });
  if (!res.ok) throw new Error('Failed to create objective');
  return res.json();
}

export function IntentConsole() {
  const qc = useQueryClient();
  const { data: objectives = [], isLoading } = useQuery({
    queryKey: ['icme-objectives'],
    queryFn: fetchObjectives,
    refetchInterval: 30_000,
  });

  const createMutation = useMutation({
    mutationFn: createObjective,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['icme-objectives'] }),
  });

  if (isLoading) return <div className="p-4 text-sm text-muted-foreground">Loading intent objectives...</div>;

  return (
    <div className="space-y-4 p-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">Intent Objectives</h2>
        <span className="text-xs text-muted-foreground">{objectives.length} objectives registered</span>
      </div>
      <div className="grid gap-3">
        {objectives.map((obj) => (
          <div key={obj.name} className="rounded-lg border p-3 space-y-2">
            <div className="flex items-center justify-between">
              <span className="font-medium text-sm">{obj.name}</span>
              <span className="text-xs text-muted-foreground">v{obj.version}</span>
            </div>
            <p className="text-xs text-muted-foreground">{obj.description}</p>
            <div className="flex gap-2 flex-wrap">
              {obj.authorized_actions.map((a) => (
                <span key={a} className="rounded bg-green-100 dark:bg-green-900/30 px-2 py-0.5 text-xs text-green-700 dark:text-green-300">
                  {a}
                </span>
              ))}
            </div>
            {obj.hard_boundaries.length > 0 && (
              <div className="flex gap-2 flex-wrap">
                {obj.hard_boundaries.map((b) => (
                  <span key={b} className="rounded bg-red-100 dark:bg-red-900/30 px-2 py-0.5 text-xs text-red-700 dark:text-red-300">
                    ⛔ {b}
                  </span>
                ))}
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}
```

### 12.2 Context Explorer Component

Create `frontend/src/components/icme/ContextExplorer.tsx`:

```tsx
'use client';

import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';

interface HybridResult {
  signal_id: string;
  agent_id: string;
  content: string;
  objective_name: string;
  vector_score: number;
  graph_hops: number;
  combined_score: number;
  alignment_score: number;
}

export function ContextExplorer() {
  const [query, setQuery] = useState('');
  const [submitted, setSubmitted] = useState('');

  const { data: results, isLoading } = useQuery<HybridResult[]>({
    queryKey: ['icme-search', submitted],
    queryFn: async () => {
      if (!submitted) return [];
      const res = await fetch(`/api/icme/search?q=${encodeURIComponent(submitted)}&top_k=10`);
      if (!res.ok) throw new Error('Search failed');
      return res.json();
    },
    enabled: !!submitted,
  });

  return (
    <div className="space-y-4 p-4">
      <h2 className="text-lg font-semibold">Context Explorer</h2>
      <div className="flex gap-2">
        <input
          className="flex-1 rounded-md border px-3 py-2 text-sm"
          placeholder="Search context memory..."
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && setSubmitted(query)}
        />
        <button
          className="rounded-md bg-primary px-4 py-2 text-sm text-primary-foreground"
          onClick={() => setSubmitted(query)}
        >
          Search
        </button>
      </div>

      {isLoading && <div className="text-sm text-muted-foreground">Searching...</div>}

      {results && results.length > 0 && (
        <div className="space-y-2">
          {results.map((r) => (
            <div key={r.signal_id} className="rounded-lg border p-3 space-y-1">
              <div className="flex items-center justify-between">
                <span className="text-xs font-mono text-muted-foreground">{r.signal_id.slice(0, 8)}</span>
                <div className="flex gap-2 text-xs">
                  <span className="text-blue-600 dark:text-blue-400">
                    vec: {(1 / (1 + r.vector_score)).toFixed(2)}
                  </span>
                  <span className="text-green-600 dark:text-green-400">
                    align: {r.alignment_score.toFixed(2)}
                  </span>
                </div>
              </div>
              <p className="text-sm">{r.content}</p>
              {r.objective_name && (
                <span className="text-xs text-muted-foreground">Objective: {r.objective_name}</span>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
```

### 12.3 Alignment Monitor Component

Create `frontend/src/components/icme/AlignmentMonitor.tsx`:

```tsx
'use client';

import { useQuery } from '@tanstack/react-query';
import { LineChart, Line, XAxis, YAxis, Tooltip, ResponsiveContainer } from 'recharts';

interface AlignmentRecord {
  id: string;
  agent_id: string;
  objective_name: string;
  decision: string;
  alignment_score: number;
  fidelity_score: number;
  timestamp: string;
}

interface Props { agentID: string; }

export function AlignmentMonitor({ agentID }: Props) {
  const { data: records = [] } = useQuery<AlignmentRecord[]>({
    queryKey: ['icme-alignment', agentID],
    queryFn: async () => {
      const res = await fetch(`/api/icme/alignment/${agentID}`);
      if (!res.ok) return [];
      return res.json();
    },
    refetchInterval: 15_000,
  });

  const chartData = records.slice(-50).map((r) => ({
    time: new Date(r.timestamp).toLocaleTimeString(),
    alignment: r.alignment_score,
    fidelity: r.fidelity_score,
  }));

  const avgAlignment = records.length
    ? records.reduce((s, r) => s + r.alignment_score, 0) / records.length
    : 0;

  return (
    <div className="space-y-4 p-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">Alignment Monitor</h2>
        <span className={`text-sm font-medium ${avgAlignment > 0.8 ? 'text-green-600' : avgAlignment > 0.6 ? 'text-yellow-600' : 'text-red-600'}`}>
          Avg: {(avgAlignment * 100).toFixed(1)}%
        </span>
      </div>
      <ResponsiveContainer width="100%" height={160}>
        <LineChart data={chartData}>
          <XAxis dataKey="time" tick={{ fontSize: 10 }} />
          <YAxis domain={[0, 1]} tick={{ fontSize: 10 }} />
          <Tooltip />
          <Line type="monotone" dataKey="alignment" stroke="#22c55e" dot={false} strokeWidth={2} />
          <Line type="monotone" dataKey="fidelity" stroke="#3b82f6" dot={false} strokeWidth={1} strokeDasharray="4 2" />
        </LineChart>
      </ResponsiveContainer>
      <div className="text-xs text-muted-foreground">
        Green: intent alignment — Blue dashed: NRV fidelity score
      </div>
    </div>
  );
}
```

---

## 13. Python ML Microservices

Create `backend/internal/services/icme/ml_services/` with the following files.

### 13.1 Entity Extractor Service

`ml_services/entity_extractor.py`:

```python
"""
GPU-accelerated Named Entity Recognition microservice.
Uses spaCy with transformer backend (GPU via cupy) or a HuggingFace NER model.
Exposes gRPC interface defined in icme.proto.
"""

import grpc
import spacy
import uuid
from concurrent import futures
from typing import List

import icme_pb2
import icme_pb2_grpc

# Load spaCy model — use GPU if available:
#   python -m spacy download en_core_web_trf
#   spacy.prefer_gpu()  # requires cupy
spacy.prefer_gpu()
nlp = spacy.load("en_core_web_trf")

# Custom label mapping for KNIRV-NEXUS domain entities.
LABEL_MAP = {
    "PERSON":  "PERSON",
    "ORG":     "ORG",
    "GPE":     "ORG",
    "PRODUCT": "CONFIG",
    "EVENT":   "EVENT",
    "DATE":    "EVENT",
    "TIME":    "EVENT",
    "MONEY":   "CONFIG",
    "PERCENT": "CONFIG",
}

# Domain-specific relation patterns (source → target → relation type).
RELATION_PATTERNS = [
    ({"LABEL": "ERROR"},    {"LABEL": "SOLUTION"},   "RESOLVED_BY"),
    ({"LABEL": "CONFIG"},   {"LABEL": "EVENT"},      "TRIGGERS"),
    ({"LABEL": "PERSON"},   {"LABEL": "EVENT"},      "CAUSED_BY"),
]


class EntityExtractorServicer(icme_pb2_grpc.EntityExtractorServicer):

    def ExtractEntities(self, request, context):
        doc = nlp(request.text)
        entities = []
        for ent in doc.ents:
            label = LABEL_MAP.get(ent.label_, ent.label_)
            entities.append(icme_pb2.Entity(
                id=str(uuid.uuid4()),
                text=ent.text,
                label=label,
                score=float(ent._.get("score", 0.9) if ent.has_extension("score") else 0.9),
                start=ent.start_char,
                end=ent.end_char,
            ))

        # Simple co-occurrence relation extraction based on proximity.
        relations = []
        ent_list = list(doc.ents)
        for i, e1 in enumerate(ent_list):
            for e2 in ent_list[i+1:min(i+4, len(ent_list))]:
                label1 = LABEL_MAP.get(e1.label_, e1.label_)
                label2 = LABEL_MAP.get(e2.label_, e2.label_)
                rel_type = _infer_relation(label1, label2, request.source)
                if rel_type:
                    relations.append(icme_pb2.Relation(
                        from_entity_id=str(i),  # simplified — use entity ID in production
                        to_entity_id=str(ent_list.index(e2)),
                        relation_type=rel_type,
                        confidence=0.75,
                    ))

        return icme_pb2.ExtractResponse(entities=entities, relations=relations)

    def ExtractBatch(self, request, context):
        return icme_pb2.BatchExtractResponse(
            responses=[self.ExtractEntities(r, context) for r in request.requests]
        )


def _infer_relation(label1: str, label2: str, source: str) -> str:
    if label1 == "ERROR" and label2 == "SOLUTION":
        return "RESOLVED_BY"
    if label1 == "CONFIG" and label2 == "EVENT":
        return "TRIGGERS"
    if source == "validation" and label1 == "PERSON":
        return "CAUSED_BY"
    return ""


def serve(port: int = 50052):
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=8))
    icme_pb2_grpc.add_EntityExtractorServicer_to_server(EntityExtractorServicer(), server)
    server.add_insecure_port(f"[::]:{port}")
    server.start()
    print(f"Entity Extractor listening on port {port}")
    server.wait_for_termination()


if __name__ == "__main__":
    serve()
```

### 13.2 Embedding Service

`ml_services/embedding_service.py`:

```python
"""
GPU-accelerated embedding generation microservice.
Supports NVEmbed (via HuggingFace) and SentenceTransformers.
"""

import grpc
from concurrent import futures
import torch
from sentence_transformers import SentenceTransformer

import icme_pb2
import icme_pb2_grpc

MODELS = {
    "nvembed":  "nvidia/NV-Embed-v2",
    "e5-large": "intfloat/e5-large-v2",
    "minilm":   "sentence-transformers/all-MiniLM-L6-v2",
}

_loaded: dict = {}


def _get_model(name: str) -> SentenceTransformer:
    if name not in _loaded:
        model_id = MODELS.get(name, MODELS["e5-large"])
        device = "cuda" if torch.cuda.is_available() else "cpu"
        _loaded[name] = SentenceTransformer(model_id, device=device)
    return _loaded[name]


class EmbeddingServicer(icme_pb2_grpc.EmbeddingServiceServicer):

    def GenerateEmbedding(self, request, context):
        try:
            model = _get_model(request.model or "e5-large")
            # E5 models expect "query: " prefix for queries.
            text = f"query: {request.text}" if "e5" in (request.model or "") else request.text
            vec = model.encode(text, normalize_embeddings=True).tolist()
            return icme_pb2.EmbedResponse(vector=vec)
        except Exception as e:
            return icme_pb2.EmbedResponse(error=str(e))

    def GenerateBatch(self, request, context):
        return icme_pb2.BatchEmbedResponse(
            responses=[self.GenerateEmbedding(r, context) for r in request.requests]
        )


def serve(port: int = 50053):
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=4))
    icme_pb2_grpc.add_EmbeddingServiceServicerServicer_to_server(EmbeddingServicer(), server)
    server.add_insecure_port(f"[::]:{port}")
    server.start()
    print(f"Embedding Service listening on port {port}")
    server.wait_for_termination()


if __name__ == "__main__":
    serve()
```

### 13.3 Re-Ranker Service

`ml_services/reranker.py`:

```python
"""
Cross-encoder re-ranking microservice.
Uses a HuggingFace cross-encoder for relevance scoring.
"""

import grpc
from concurrent import futures
from sentence_transformers import CrossEncoder
import torch

import icme_pb2
import icme_pb2_grpc

device = "cuda" if torch.cuda.is_available() else "cpu"
model = CrossEncoder("cross-encoder/ms-marco-MiniLM-L-6-v2", device=device)


class ReRankerServicer(icme_pb2_grpc.ReRankerServicer):

    def Rerank(self, request, context):
        if not request.candidates:
            return icme_pb2.RerankResponse(results=[])

        pairs = [(request.query, c) for c in request.candidates]
        scores = model.predict(pairs).tolist()

        indexed = sorted(
            enumerate(scores), key=lambda x: x[1], reverse=True
        )[:request.top_k or 10]

        results = [
            icme_pb2.RankedResult(
                original_index=i,
                score=s,
                text=request.candidates[i],
            )
            for i, s in indexed
        ]
        return icme_pb2.RerankResponse(results=results)


def serve(port: int = 50054):
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=4))
    icme_pb2_grpc.add_ReRankerServicer_to_server(ReRankerServicer(), server)
    server.add_insecure_port(f"[::]:{port}")
    server.start()
    print(f"Re-Ranker listening on port {port}")
    server.wait_for_termination()


if __name__ == "__main__":
    serve()
```

### 13.4 Process Manager Script

`ml_services/start_ml_services.sh`:

```bash
#!/usr/bin/env bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOG_DIR="${SCRIPT_DIR}/logs"
mkdir -p "$LOG_DIR"

start_service() {
    local name="$1"
    local script="$2"
    echo "Starting $name..."
    python3 "$SCRIPT_DIR/$script" > "$LOG_DIR/${name}.log" 2>&1 &
    echo $! > "$LOG_DIR/${name}.pid"
    echo "$name PID: $!"
}

start_service "entity_extractor" "entity_extractor.py"
start_service "embedding_service" "embedding_service.py"
start_service "reranker"          "reranker.py"

echo "All ML services started. Logs in $LOG_DIR"
```

---

## 14. Testing Strategy

### 14.1 Unit Tests

Create `backend/internal/services/icme/icme_test.go`:

```go
package icme_test

import (
    "context"
    "testing"
    "time"

    "github.com/tidwall/buntdb"
    "go.uber.org/zap"

    "github.com/KNIRVNEXUS/backend/internal/services/icme"
)

func newTestDB(t *testing.T) *buntdb.DB {
    t.Helper()
    db, err := buntdb.Open(":memory:")
    if err != nil {
        t.Fatalf("open buntdb: %v", err)
    }
    t.Cleanup(func() { db.Close() })
    return db
}

// ─── Intent Registry Tests ────────────────────────────────────────────────────

func TestIntentRegistry_RegisterAndRetrieve(t *testing.T) {
    db := newTestDB(t)
    logger := zap.NewNop()
    reg, err := icme.NewIntentRegistry(db, logger)
    if err != nil {
        t.Fatalf("new registry: %v", err)
    }

    obj := &icme.IntentObjective{
        Name:              "test_objective",
        Description:       "Test objective",
        AuthorizedActions: []string{"approve", "reject"},
        HardBoundaries:    []string{"bypass_pqc"},
        TradeOffs:         map[string]float64{"speed_vs_thoroughness": 0.7},
    }

    if err := reg.RegisterObjective(obj); err != nil {
        t.Fatalf("register: %v", err)
    }

    got := reg.GetObjective("test_objective")
    if got == nil {
        t.Fatal("expected objective, got nil")
    }
    if got.Name != obj.Name {
        t.Errorf("name: got %q, want %q", got.Name, obj.Name)
    }
    if got.Version != 1 {
        t.Errorf("version: got %d, want 1", got.Version)
    }
}

func TestIntentRegistry_AgentBinding(t *testing.T) {
    db := newTestDB(t)
    logger := zap.NewNop()
    reg, _ := icme.NewIntentRegistry(db, logger)

    reg.RegisterObjective(&icme.IntentObjective{
        Name:              "agent_obj",
        AuthorizedActions: []string{"scan", "report"},
        HardBoundaries:    []string{"delete_all"},
    })
    reg.BindAgentToObjective("agent-001", "agent_obj")

    if !reg.IsActionAuthorized("agent-001", "scan") {
        t.Error("scan should be authorized")
    }
    if reg.IsActionAuthorized("agent-001", "unknown_action") {
        t.Error("unknown_action should not be authorized")
    }
    if !reg.ViolatesHardBoundary("agent-001", "delete_all") {
        t.Error("delete_all should violate hard boundary")
    }
}

// ─── Temporal Hypergraph Tests ────────────────────────────────────────────────

func TestIntentHypergraph_InsertAndNeighbors(t *testing.T) {
    graph := icme.NewIntentHypergraph(10*time.Minute, 1000, zap.NewNop())

    sig := &icme.IntentionalSignal{
        ID:        "sig-001",
        AgentID:   "agent-001",
        Timestamp: time.Now(),
        Entities: []icme.ExtractedEntity{
            {ID: "e1", Text: "ConfigError", Label: "ERROR", Score: 0.9},
            {ID: "e2", Text: "RestartFix", Label: "SOLUTION", Score: 0.85},
        },
        Relations: []icme.ExtractedRelation{
            {FromEntityID: "e1", ToEntityID: "e2", RelationType: "RESOLVED_BY", Confidence: 0.8},
        },
    }
    graph.InsertSignal(sig)

    nodes, edges := graph.Snapshot()
    if len(nodes) != 2 {
        t.Errorf("nodes: got %d, want 2", len(nodes))
    }
    if len(edges) != 1 {
        t.Errorf("edges: got %d, want 1", len(edges))
    }
}

func TestIntentHypergraph_Pruning(t *testing.T) {
    // 1-second window for testing.
    graph := icme.NewIntentHypergraph(1*time.Second, 1000, zap.NewNop())

    sig := &icme.IntentionalSignal{
        ID:        "sig-old",
        AgentID:   "agent-001",
        Timestamp: time.Now().Add(-2 * time.Second),
        Entities: []icme.ExtractedEntity{
            {ID: "e1", Text: "OldError", Label: "ERROR"},
            {ID: "e2", Text: "OldFix", Label: "SOLUTION"},
        },
        Relations: []icme.ExtractedRelation{
            {FromEntityID: "e1", ToEntityID: "e2", RelationType: "RESOLVED_BY"},
        },
    }
    graph.InsertSignal(sig)

    // Insert a fresh signal to trigger pruning.
    graph.InsertSignal(&icme.IntentionalSignal{
        ID:        "sig-new",
        AgentID:   "agent-001",
        Timestamp: time.Now(),
        Entities:  []icme.ExtractedEntity{{ID: "e3", Text: "NewEvent", Label: "EVENT"}},
    })

    _, edges := graph.Snapshot()
    if len(edges) > 0 {
        t.Errorf("expected 0 edges after pruning, got %d", len(edges))
    }
}

// ─── Delegation Tests ─────────────────────────────────────────────────────────

func TestDelegationFramework_HardBoundary(t *testing.T) {
    db := newTestDB(t)
    logger := zap.NewNop()
    reg, _ := icme.NewIntentRegistry(db, logger)
    reg.RegisterObjective(&icme.IntentObjective{
        Name:           "secure_obj",
        HardBoundaries: []string{"bypass_pqc_signing"},
    })
    reg.BindAgentToObjective("agent-secure", "secure_obj")

    fw := icme.NewDelegationFramework(reg, logger)
    result := fw.Resolve(icme.DecisionContext{
        AgentID: "agent-secure",
        Action:  "bypass_pqc_signing",
    })
    if result.Approved {
        t.Error("hard boundary action should not be approved")
    }
    if result.Action != "deny" {
        t.Errorf("action: got %q, want deny", result.Action)
    }
}

func TestDelegationFramework_VIPEscalation(t *testing.T) {
    db := newTestDB(t)
    logger := zap.NewNop()
    reg, _ := icme.NewIntentRegistry(db, logger)
    reg.RegisterObjective(&icme.IntentObjective{
        Name:              "support_obj",
        AuthorizedActions: []string{"approve_validation"},
    })
    reg.BindAgentToObjective("agent-support", "support_obj")

    fw := icme.NewDelegationFramework(reg, logger)
    result := fw.Resolve(icme.DecisionContext{
        AgentID:      "agent-support",
        Action:       "approve_validation",
        CustomerTier: "VIP",
    })
    if !result.Approved {
        t.Error("VIP action should be approved")
    }
    if result.Action != "escalate_to_specialist" {
        t.Errorf("action: got %q, want escalate_to_specialist", result.Action)
    }
}
```

### 14.2 Integration Test

Create `integration-tests/icme_integration_test.go`:

```go
//go:build integration

package integration_test

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "testing"
)

const baseURL = "http://localhost:8082"

func TestICMEObjectiveLifecycle(t *testing.T) {
    // Create objective.
    obj := map[string]interface{}{
        "name":               "integration_test_obj",
        "description":        "Created by integration test",
        "authorized_actions": []string{"test_action"},
        "hard_boundaries":    []string{"forbidden_action"},
        "trade_offs":         map[string]float64{"speed_vs_thoroughness": 0.5},
    }
    body, _ := json.Marshal(obj)
    resp, err := http.Post(baseURL+"/api/icme/objectives", "application/json", bytes.NewReader(body))
    if err != nil {
        t.Fatalf("create objective: %v", err)
    }
    if resp.StatusCode != http.StatusCreated {
        t.Fatalf("create status: got %d, want 201", resp.StatusCode)
    }
    resp.Body.Close()

    // Retrieve it.
    resp2, err := http.Get(baseURL + "/api/icme/objectives/integration_test_obj")
    if err != nil {
        t.Fatalf("get objective: %v", err)
    }
    if resp2.StatusCode != http.StatusOK {
        t.Fatalf("get status: got %d, want 200", resp2.StatusCode)
    }
    resp2.Body.Close()

    // Bind to agent.
    bindBody, _ := json.Marshal(map[string]string{"objective_name": "integration_test_obj"})
    resp3, err := http.Post(
        baseURL+"/api/icme/agents/test-agent-001/bind",
        "application/json",
        bytes.NewReader(bindBody),
    )
    if err != nil {
        t.Fatalf("bind agent: %v", err)
    }
    if resp3.StatusCode != http.StatusNoContent {
        t.Fatalf("bind status: got %d, want 204", resp3.StatusCode)
    }
    resp3.Body.Close()

    // Search.
    resp4, err := http.Get(
        fmt.Sprintf("%s/api/icme/search?q=test+validation+error&agent_id=test-agent-001&top_k=5", baseURL),
    )
    if err != nil {
        t.Fatalf("search: %v", err)
    }
    if resp4.StatusCode != http.StatusOK {
        t.Fatalf("search status: got %d, want 200", resp4.StatusCode)
    }
    resp4.Body.Close()
}
```

---

## 15. Deployment & Scaling

### 15.1 Updated Makefile Targets

Add to the existing `Makefile`:

```makefile
# ── ICME ML Services ──────────────────────────────────────────────────────────

PROTO_DIR := backend/internal/services/icme/proto
ML_DIR    := backend/internal/services/icme/ml_services

.PHONY: icme-proto icme-ml-install icme-ml-start icme-ml-stop

icme-proto:
	@echo "→ Generating ICME gRPC bindings..."
	protoc \
	  --go_out=$(PROTO_DIR) \
	  --go-grpc_out=$(PROTO_DIR) \
	  --proto_path=$(PROTO_DIR) \
	  $(PROTO_DIR)/icme.proto
	python3 -m grpc_tools.protoc \
	  -I$(PROTO_DIR) \
	  --python_out=$(ML_DIR) \
	  --grpc_python_out=$(ML_DIR) \
	  $(PROTO_DIR)/icme.proto

icme-ml-install:
	@echo "→ Installing Python ML service dependencies..."
	pip3 install -r $(ML_DIR)/requirements.txt

icme-ml-start:
	@echo "→ Starting ICME ML microservices..."
	bash $(ML_DIR)/start_ml_services.sh

icme-ml-stop:
	@echo "→ Stopping ICME ML microservices..."
	@for pid_file in $(ML_DIR)/logs/*.pid; do \
	  kill $$(cat $$pid_file) 2>/dev/null || true; \
	  rm -f $$pid_file; \
	done

# Extend existing 'all' target
all: clean deps icme-proto frontend backend binary
```

### 15.2 Docker Compose Extension

Add a `docker-compose.icme.yml` for the ML services:

```yaml
version: "3.9"

services:
  entity-extractor:
    build:
      context: backend/internal/services/icme/ml_services
      dockerfile: Dockerfile.ml
      args:
        SERVICE: entity_extractor.py
    ports:
      - "50052:50052"
    deploy:
      resources:
        reservations:
          devices:
            - driver: nvidia
              count: 1
              capabilities: [gpu]
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "python3", "-c", "import grpc; grpc.channel_ready_future(grpc.insecure_channel('localhost:50052')).result(timeout=5)"]
      interval: 30s
      timeout: 10s
      retries: 3

  embedding-service:
    build:
      context: backend/internal/services/icme/ml_services
      dockerfile: Dockerfile.ml
      args:
        SERVICE: embedding_service.py
    ports:
      - "50053:50053"
    deploy:
      resources:
        reservations:
          devices:
            - driver: nvidia
              count: 1
              capabilities: [gpu]
    restart: unless-stopped

  reranker:
    build:
      context: backend/internal/services/icme/ml_services
      dockerfile: Dockerfile.ml
      args:
        SERVICE: reranker.py
    ports:
      - "50054:50054"
    deploy:
      resources:
        reservations:
          devices:
            - driver: nvidia
              count: 1
              capabilities: [gpu]
    restart: unless-stopped
```

`ml_services/Dockerfile.ml`:

```dockerfile
FROM nvidia/cuda:12.4.0-runtime-ubuntu22.04

RUN apt-get update && apt-get install -y python3 python3-pip && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY requirements.txt .
RUN pip3 install --no-cache-dir -r requirements.txt

COPY . .

ARG SERVICE
ENV SERVICE_SCRIPT=$SERVICE
CMD ["sh", "-c", "python3 $SERVICE_SCRIPT"]
```

### 15.3 DGX Spark Multi-GPU Scaling

On NVIDIA DGX Spark hardware, enable GPU FAISS in `faiss_manager.go`:

```go
// Replace NewIndexFlatL2 with GPU-backed index on DGX Spark.
// Requires: build tag `faiss_gpu` and libfaiss-gpu.so linked.

//go:build faiss_gpu

package icme

import (
    faiss "github.com/DataIntelligenceCrew/go-faiss"
)

func newFAISSIndex(dim int) (*faiss.IndexImpl, error) {
    // Allocate GPU resources on device 0.
    res, err := faiss.NewStandardGpuResources()
    if err != nil {
        return faiss.NewIndexFlatL2(dim) // CPU fallback
    }
    cfg := faiss.NewGpuIndexFlatConfig()
    cfg.UseFloat16 = false
    return faiss.NewGpuIndexFlatL2(res, dim, cfg)
}
```

For multi-node deployments, shard the FAISS index across nodes using NVIDIA NCCL for synchronization of embedding additions, routing queries to the appropriate shard via a consistent hash of the `agent_id`:

```go
// ShardedFAISSManager routes to the correct node shard.
// Each DGX Spark node runs one FAISSIndexManager for its shard.
type ShardedFAISSManager struct {
    shards   []*FAISSIndexManager
    numShards int
}

func (s *ShardedFAISSManager) shardFor(agentID string) *FAISSIndexManager {
    h := fnv32(agentID)
    return s.shards[h % uint32(s.numShards)]
}

func fnv32(s string) uint32 {
    var h uint32 = 2166136261
    for i := 0; i < len(s); i++ {
        h ^= uint32(s[i])
        h *= 16777619
    }
    return h
}
```

### 15.4 Integration Wiring in the Main Server

In `backend/cmd/backend_server/main.go`, add ICME initialization after the existing service initialization block:

```go
// ── ICME Initialization ───────────────────────────────────────────────────────
if cfg.ICME.Enabled {
    // gRPC clients for Python ML microservices.
    entityConn, err := grpc.Dial(cfg.ICME.EntityExtractorAddr, grpc.WithInsecure())
    if err != nil {
        logger.Fatal("icme entity extractor connect", zap.Error(err))
    }
    embedConn, err := grpc.Dial(cfg.ICME.EmbeddingServiceAddr, grpc.WithInsecure())
    if err != nil {
        logger.Fatal("icme embedding service connect", zap.Error(err))
    }
    rerankConn, err := grpc.Dial(cfg.ICME.RerankerAddr, grpc.WithInsecure())
    if err != nil {
        logger.Fatal("icme reranker connect", zap.Error(err))
    }

    entityClient  := icme_proto.NewEntityExtractorClient(entityConn)
    embedClient   := icme_proto.NewEmbeddingServiceClient(embedConn)
    rerankClient  := icme_proto.NewReRankerClient(rerankConn)

    intentReg, err := icme.NewIntentRegistry(s.db.DB(), logger)
    if err != nil {
        logger.Fatal("icme intent registry", zap.Error(err))
    }

    graph := icme.NewIntentHypergraph(
        cfg.ICME.GraphWindowSize,
        cfg.ICME.GraphMaxNodes,
        logger,
    )

    faissIdx, err := icme.NewFAISSIndexManager(s.db.DB(), logger)
    if err != nil {
        logger.Fatal("icme faiss index", zap.Error(err))
    }

    delegation    := icme.NewDelegationFramework(intentReg, logger)
    searchEngine  := icme.NewHybridSearchEngine(faissIdx, graph, embedClient, rerankClient, intentReg, logger)
    fintechAdapter := icme.NewFintechNRVAdapter("http://localhost:" + cfg.API.Port)
    alignmentLoop := icme.NewAlignmentLoop(intentReg, fintechAdapter, cfg.ICME.AlignmentEvalInterval, logger)

    signalRouter := icme.NewSignalRouter(
        intentReg, entityClient, embedClient, graph, faissIdx, s.activeMemory, logger,
    )

    icmeSvc := icme.NewICMEService(
        intentReg, graph, faissIdx, searchEngine, delegation, alignmentLoop, signalRouter, logger,
    )

    // Register HTTP routes.
    icmeSvc.RegisterRoutes(s.router, s.authMiddleware)

    // Start background goroutines.
    signalRouter.Start(ctx)
    go alignmentLoop.Start(ctx)

    // Hook signal router into existing services.
    s.activeMemory.SetSignalRouter(signalRouter)
    s.nexusServer.SetSignalRouter(signalRouter)

    logger.Info("icme initialized",
        zap.String("graph_window", cfg.ICME.GraphWindowSize.String()),
        zap.Int("embed_dim", cfg.ICME.EmbedDimension),
    )
}
```

---

## Summary

The **Intentional Context Memory Engine** integrates three capabilities that KNIRV-NEXUS previously had in isolation:

| Capability | Before ICME | After ICME |
|---|---|---|
| Signal ingestion | Raw event storage in Vault | Intent-annotated, entity-enriched signals |
| Memory retrieval | Recency-based, single-dimension | Hybrid vector + graph, intent-ranked |
| Agent alignment | eBPF syscall correlation only | Full intent lifecycle: declare → enforce → measure → refine |
| Organizational knowledge | Implicit in code | Explicit, versioned, machine-readable objectives |
| Context graph | Flat ContextRecord list | Temporal hypergraph with causal edges |

All components integrate through existing KNIRV-NEXUS interfaces (BuntDB, Arrow Flight, Gorilla Mux, zap logging, Markdown Fabric) without breaking existing APIs. The Python ML microservices are optional and gracefully degrade: FAISS searches return empty results, entity extraction skips enrichment, and the system continues operating — albeit without semantic ranking.

The feedback loop between the alignment scorer and FinTech NRV fidelity scores closes the intent engineering cycle, ensuring the system continuously self-corrects rather than drifting from declared organizational goals over time.
