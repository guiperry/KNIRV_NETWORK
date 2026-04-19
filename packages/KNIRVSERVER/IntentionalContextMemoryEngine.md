# Intentional Context Memory Engine (ICME)
## Integration Guide for KNIRV-SERVER

**Synthesized from:** `ContextMemoryEngine.md` (Pulse HQ GPU Pipeline) + `IntentEngineering.md` (Organizational Intent Framework)
**Target System:** KNIRV-SERVER Deterministic Validation Environment
**Language:** Go 1.24+ (primary, including ML capabilities via CGO)

---

## Table of Contents

1. [Conceptual Overview](#1-conceptual-overview)
2. [Architecture](#2-architecture)
3. [Prerequisites & Dependencies](#3-prerequisites--dependencies)
4. [Layer 1 — Intentional Signal Ingestion](#4-layer-1--intentional-signal-ingestion)
5. [Layer 2 — Intent Engineering Framework](#5-layer-2--intent-engineering-framework)
6. [Layer 3 — KNIRVGRAPH Enhancement (Temporal Hypergraph)](#6-layer-3--knirvgraph-enhancement-temporal-hypergraph)
7. [Layer 4 — Embedding Generation & Semantic Indexing](#7-layer-4--embedding-generation--semantic-indexing)
8. [Layer 5 — Hybrid Retrieval API](#8-layer-5--hybrid-retrieval-api)
9. [Layer 6 — Feedback & Alignment Loops](#9-layer-6--feedback--alignment-loops)
10. [Configuration Changes](#10-configuration-changes)
11. [API Reference](#11-api-reference)
12. [Frontend Integration](#12-frontend-integration)
13. [Testing Strategy](#13-testing-strategy)
14. [Deployment & Scaling](#14-deployment--scaling)

---

## 1. Conceptual Overview

### What Is the ICME?

The **Intentional Context Memory Engine (ICME)** fuses two independently powerful paradigms into a single coherent system layered onto KNIRV-SERVER:

| Source Concept | Contribution to ICME |
|---|---|
| **ContextMemoryEngine** (Pulse HQ) | Cloudflare/Ollama embeddings, spaCy NER via Go CGO, FAISS semantic indexing, real-time streaming |
| **IntentEngineering** | Machine-readable organizational goals (OKRs), delegation hierarchies, hard/soft decision boundaries, alignment feedback loops |

Without intent, a context memory engine is a passive recorder — it knows *what happened* but not *what should happen*. Without rich context memory, intent engineering becomes shallow rule-matching with no temporal awareness. Together they form a system where:

- Every agent action is validated not just against a syscall trace (existing eBPF layer) but against **structured organizational intent parameters**.
- Every memory retrieval is **semantically ranked by relevance to declared intent**, not just recency.
- The temporal hypergraph captures **why** decisions were made alongside **what** was done, enabling deterministic replay with full causal context.
- Feedback from KNIRV-SERVER's Factuality Slice Validator (evidence-grounded confidence scores) closes the loop, **continuously refining intent parameters**.

### Relationship to Existing KNIRV-SERVER Systems

```
EXISTING SYSTEMS                    ICME ENHANCEMENTS
───────────────────                ───────────────────
NexusMemoryServer (Arrow Flight) ←→ Semantic Embedding Index (FAISS)
ActiveMemoryService              ←→ Intentional Signal Router
ReasoningGraph (KNIRVGRAPH)      ←→ Temporal Hypergraph (ENHANCED)
VaultService (ErrorNode/Solution)←→ Intent-Annotated Vault Nodes
CognitiveEngine                 ←→ Intent Engineering Framework
Factuality Slice Validator     ←→ Alignment Feedback Loop
eBPF Intent-Action Correlation   ←→ Intent Constraint Enforcement
```

The ICME **extends existing KNIRVGRAPH** rather than creating a new hypergraph. It integrates as an additional service layer (`icme`) that wires into existing service interfaces.

### Intent Scope: Global vs Per-DVE

| Scope | Description | Storage Key Prefix |
|-------|-------------|-------------------|
| **Global** | Organization-wide OKRs and objectives | `icme:global:objectives:{name}` |
| **Per-DVE** | Workspace-specific objectives bound to DVE ID | `icme:dve:{dve_id}:objectives:{name}` |

---

## 2. Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    KNIRV-SERVER ICME STACK                      │
├─────────────────────────────────────────────────────────────────┤
│  FRONTEND (Next.js 15)                                          │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────┐   │
│  │  CDE Panel       │  │  Onboarding      │  │ KNIRVGRAPH   │   │
│  │  - Intent        │  │  OKR Form        │  │ Modal        │   │
│  │  - Context       │  │  (Global Intent) │  │ (Enhanced)   │   │
│  │  - Alignment     │  │                  │  │              │   │
│  └──────────┬───────┘  └────────┬─────────┘  └──────┬───────┘   │
├─────────────┼────────────────────┼───────────────────┼──────────┤
│  GO BACKEND (port 8082)          │                   │          │
│  ┌──────────▼────────────────────▼───────────────────▼───────┐  │
│  │              ICME Service (new)                           │  │
│  │  ┌─────────────────┐  ┌──────────────────────────────┐    │  │
│  │  │ Intent Registry │  │    Hybrid Retrieval Engine   │    │  │
│  │  │ (KNIRVBASE)     │  │                              │    │  │
│  │  └────────┬────────┘  └──────────────┬───────────────┘    │  │
│  │  ┌────────▼────────┐  ┌──────────────▼───────────────┐    │  │
│  │  │ KNIRVGRAPH      │  │  Embedding Engine            │    │  │
│  │  │ (ENHANCED)      │  │  - Cloudflare → Ollama       │    │  │
│  │  └────────┬────────┘  │  - spaCy NER (Go CGO)        │    │  │
│  │           │           └──────────────┬───────────────┘    │  │
│  │  ┌────────▼─────────────────────────▼─────────────────┐   │  │
│  │  │  FAISS Index Manager                               │   │  │
│  │  └────────────────────────────────────────────────────┘   │  │
│  └───────────────────────────────────────────────────────────┘  │
│              │                          │                       │
│  ┌───────────▼──────────────────────────▼────────────────────┐  │
│  │           EXISTING SERVICES (unchanged interfaces)        │  │
│  │  ActiveMemoryService | VaultService | CognitiveEngine     │  │
│  │  NexusMemoryServer   | MarkdownFabric | KNIRVBASE         │  │
│  │  eBPF Manager       | FintechValidator                    │  │
│  └───────────────────────────────────────────────────────────┘  │
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
      │  - Extract entities via spaCy NER (Go CGO)
      │
      ▼
[2] Intent Constraint Check (eBPF + IntentRegistry)
      │  - Does this action fall within authorized_actions?
      │  - Apply trade-off weights
      │  - Enforce hard_boundaries
      │
      ▼
[3] KNIRVGRAPH Enhancement (Temporal Hypergraph Update)
      │  - Add entity nodes and relationship edges
      │  - Stamp edges with timestamp + intent_objective
      │  - Prune edges outside time window
      │
      ▼
[4] Embedding + FAISS Index Update
      │  - Generate vector via Cloudflare → Ollama fallback (Go)
      │  - Add to FAISS index with metadata
      │  - Store mapping: vector_id → graph_node_id
      │
      ▼
[5] Active Memory + Vault Persistence
      │  - Write ErrorNode / SolutionNode (existing vault)
      │  - Write ContextRecord with intent annotation (enhanced KNIRVGRAPH)
      │  - Encrypt and persist to Markdown Fabric
      │
      ▼
[6] Arrow Flight Streaming
        - Broadcast updated context + intent to subscribed agents
        - Schema includes intent_objective, alignment_score fields
```

---

## 3. Prerequisites & Dependencies

### Environment Configuration (.env)

Create or update `.env` in the backend directory:

```bash
# Cloudflare Embeddings (primary)
CLOUDFLARE_EMBEDDINGS_URL="https://embeddings.knirv.com"

# Ollama Fallback (auto-installed if Cloudflare fails)
OLLAMA_BASE_URL="http://"
OLLAMA_EMlocalhost:11434BEDDING_MODEL="nomic-embed-text"

# spaCy Model (auto-downloaded)
SPACY_MODEL="en_core_web_sm"
```

### Go Dependencies (add to `go.mod`)

```
require (
    // Existing deps retained...

    // ICME additions
    github.com/DataIntelligenceCrew/go-faiss v0.3.0
    github.com/am-sokolov/go-spacy v1.0.0  // For spaCy NER via CGO
)
```

> **Note:** `go-faiss` requires `libfaiss` installed on the host. On Ubuntu/Debian:
> ```bash
> apt-get install -y libfaiss-dev
> ```
> spaCy Go bindings require Python 3.9+ and spaCy installed.

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
	SourceFactuality SignalSource = "factuality"
	SourceEBPF       SignalSource = "ebpf"
)

// IntentScope defines whether an objective is global or per-DVE
type IntentScope string

const (
	ScopeGlobal IntentScope = "global"
	ScopeDVE    IntentScope = "dve"
)

// IntentionalSignal is the enriched form of any event entering the ICME.
type IntentionalSignal struct {
	ID              string            `json:"id"`
	AgentID         string            `json:"agent_id"`
	DVEID           string            `json:"dve_id"`           // Optional: for per-DVE objectives
	Source          SignalSource      `json:"source"`
	Content         string            `json:"content"`
	Timestamp       time.Time         `json:"timestamp"`
	Scope           IntentScope       `json:"scope"`             // global or dve
	ObjectiveName   string            `json:"objective_name"`
	AuthorizedActs  []string          `json:"authorized_acts"`
	TradeOffWeights map[string]float64`json:"trade_off_weights"`
	HardBoundaries  []string          `json:"hard_boundaries"`
	AlignmentScore  float64           `json:"alignment_score"`
	Entities        []ExtractedEntity `json:"entities"`
	Relations       []ExtractedRelation `json:"relations"`
	EmbeddingID     int64             `json:"embedding_id"`
}

// ExtractedEntity represents an entity from NER.
type ExtractedEntity struct {
	ID    string  `json:"id"`
	Text  string  `json:"text"`
	Label string  `json:"label"` // PERSON, ORG, CONFIG, EVENT, ERROR, SOLUTION
	Score float32 `json:"score"`
	Start int     `json:"start"`
	End   int     `json:"end"`
}

// ExtractedRelation represents a relationship between entities.
type ExtractedRelation struct {
	FromEntityID string  `json:"from_entity_id"`
	ToEntityID   string  `json:"to_node_id"`
	RelationType string  `json:"relation_type"` // CAUSED_BY, RESOLVED_BY, DEPENDS_ON, TRIGGERS
	Confidence   float32 `json:"confidence"`
}

// IntentObjective is the machine-readable encoding of an organizational goal.
type IntentObjective struct {
	Name              string             `json:"name"`
	Scope             IntentScope        `json:"scope"`  // global or dve
	DVEID             string             `json:"dve_id"` // Required if scope=dve
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
	DVEID           string    `json:"dve_id"`
	ObjectiveName   string    `json:"objective_name"`
	SignalID        string    `json:"signal_id"`
	Decision        string    `json:"decision"`
	Outcome         string    `json:"outcome"`
	AlignmentScore  float64   `json:"alignment_score"`
	FidelityScore   float64   `json:"fidelity_score"`
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

// SignalRouter accepts raw events from existing KNIRV-SERVER services,
// annotates them with intent context, extracts entities, and enqueues
// them for graph and index updates.
type SignalRouter struct {
	intentRegistry  *IntentRegistry
	nerClient       *spacy.Client  // Go spaCy CGO bindings
	embedProvider   *EmbeddingProvider
	graphEngine     *KNIRVGRAPHEngine
	indexManager    *FAISSIndexManager
	activeMemory    ActiveMemoryInterface
	logger          *zap.Logger
	signalCh        chan *IntentionalSignal
}

func NewSignalRouter(
	intentRegistry *IntentRegistry,
	nerClient *spacy.Client,
	embedProvider *EmbeddingProvider,
	graphEngine *KNIRVGRAPHEngine,
	indexManager *FAISSIndexManager,
	activeMemory ActiveMemoryInterface,
	logger *zap.Logger,
) *SignalRouter {
	r := &SignalRouter{
		intentRegistry: intentRegistry,
		nerClient:      nerClient,
		embedProvider:  embedProvider,
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
	for i := 0; i < 4; i++ {
		go r.processLoop(ctx)
	}
}

// Ingest is called by existing KNIRV-SERVER services to push a raw signal.
func (r *SignalRouter) Ingest(agentID, dveID string, source SignalSource, content string) {
	sig := &IntentionalSignal{
		ID:        uuid.NewString(),
		AgentID:   agentID,
		DVEID:     dveID,
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
	// Step 1: Determine scope and get intent objective
	sig.Scope = ScopeGlobal
	if sig.DVEID != "" {
		obj := r.intentRegistry.GetObjectiveForDVE(sig.AgentID, sig.DVEID)
		if obj != nil {
			sig.Scope = ScopeDVE
			sig.ObjectiveName = obj.Name
			sig.AuthorizedActs = obj.AuthorizedActions
			sig.TradeOffWeights = obj.TradeOffs
			sig.HardBoundaries = obj.HardBoundaries
		}
	}
	// Fall back to global objective
	if sig.ObjectiveName == "" {
		obj := r.intentRegistry.GetGlobalObjectiveForAgent(sig.AgentID)
		if obj != nil {
			sig.ObjectiveName = obj.Name
			sig.AuthorizedActs = obj.AuthorizedActions
			sig.TradeOffWeights = obj.TradeOffs
			sig.HardBoundaries = obj.HardBoundaries
		}
	}

	// Step 2: Extract entities and relations via spaCy (Go CGO)
	ents, rels, err := r.nerClient.ExtractEntitiesAndRelations(sig.Content)
	if err != nil {
		r.logger.Debug("icme ner skipped", zap.Error(err))
	} else {
		sig.Entities = ents
		sig.Relations = rels
	}

	// Step 3: Generate embedding via Cloudflare → Ollama fallback (Go)
	embedding, err := r.embedProvider.GetEmbedding(sig.Content)
	if err != nil {
		return fmt.Errorf("embedding generation: %w", err)
	}

	// Step 4: Add to FAISS index
	vecID, err := r.indexManager.Add(sig.ID, embedding, VectorMeta{
		SignalID:  sig.ID,
		AgentID:   sig.AgentID,
		Summary:   sig.Content[:min(200, len(sig.Content))],
		Objective: sig.ObjectiveName,
		DVEID:     sig.DVEID,
	})
	if err != nil {
		return fmt.Errorf("faiss index add: %w", err)
	}
	sig.EmbeddingID = vecID

	// Step 5: Update KNIRVGRAPH temporal hypergraph
	r.graphEngine.InsertSignal(sig)

	// Step 6: Persist to active memory
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

### 5.1 Intent Registry (Using KNIRVBASE)

Create `backend/internal/services/icme/intent_registry.go`:

```go
package icme

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"backend_server/internal/data_engine"
)

const (
	// KNIRVBASE key prefixes
	kbKeyGlobalObj      = "icme:global:objectives:"      // global objectives
	kbKeyDVEObj         = "icme:dve:%s:objectives:"     // per-DVE objectives: format with DVE ID
	kbKeyGlobalBind     = "icme:global:agent_bindings:"  // global agent → objective
	kbKeyDVEBind        = "icme:dve:%s:agent_bindings:"  // per-DVE agent → objective: format with DVE ID
	kbKeyAlignment     = "icme:alignment:%s:%s:"         // alignment: agentID:recordID
)

// IntentRegistry stores and serves IntentObjectives using KNIRVBASE (via BuntDBManager).
type IntentRegistry struct {
	db     *data_engine.BuntDBManager
	mu     sync.RWMutex
	cache  map[string]*IntentObjective       // global objectives: name → objective
	dveCache map[string]map[string]*IntentObjective  // per-DVE: dveID → (name → objective)
	globalBinds  map[string]string            // agentID → global objectiveName
	dveBinds     map[string]map[string]string // dveID → (agentID → objectiveName)
	alignmentCache []*AlignmentRecord        // recent alignment records for drift detection
	logger      *zap.Logger
}

func NewIntentRegistry(db *data_engine.BuntDBManager, logger *zap.Logger) (*IntentRegistry, error) {
	r := &IntentRegistry{
		db:        db,
		cache:     make(map[string]*IntentObjective),
		dveCache:  make(map[string]map[string]*IntentObjective),
		globalBinds: make(map[string]string),
		dveBinds:   make(map[string]map[string]string),
		alignmentCache: make([]*AlignmentRecord, 0, 100),
		logger:     logger,
	}
	if err := r.loadFromDB(); err != nil {
		return nil, fmt.Errorf("intent registry load: %w", err)
	}
	return r, nil
}

// RegisterObjective persists a new or updated IntentObjective to KNIRVBASE.
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

	// Determine key based on scope
	var key string
	if obj.Scope == ScopeGlobal {
		key = kbKeyGlobalObj + obj.Name
	} else {
		key = fmt.Sprintf(kbKeyDVEObj+obj.Name, obj.DVEID)
	}

	data, err := json.Marshal(obj)
	if err != nil {
		return fmt.Errorf("marshal objective: %w", err)
	}

	if err := r.db.StoreJSON(key, data); err != nil {
		return fmt.Errorf("knirvbase store objective: %w", err)
	}

	// Update in-memory cache
	r.mu.Lock()
	if obj.Scope == ScopeGlobal {
		r.cache[obj.Name] = obj
	} else {
		if r.dveCache[obj.DVEID] == nil {
			r.dveCache[obj.DVEID] = make(map[string]*IntentObjective)
		}
		r.dveCache[obj.DVEID][obj.Name] = obj
	}
	r.mu.Unlock()

	r.logger.Info("icme intent objective registered",
		zap.String("name", obj.Name),
		zap.String("scope", string(obj.Scope)),
		zap.Int("version", obj.Version),
	)
	return nil
}

// BindAgentToObjective maps an agent to an objective (global or per-DVE).
func (r *IntentRegistry) BindAgentToObjective(agentID, objectiveName string, dveID string) error {
	r.mu.RLock()
	var exists bool
	if dveID == "" {
		_, exists = r.cache[objectiveName]
	} else {
		if dveCache, ok := r.dveCache[dveID]; ok {
			_, exists = dveCache[objectiveName]
		}
	}
	r.mu.RUnlock()
	if !exists {
		return fmt.Errorf("objective %q not found", objectiveName)
	}

	// Determine bind key
	var bindKey string
	if dveID == "" {
		bindKey = kbKeyGlobalBind + agentID
	} else {
		bindKey = fmt.Sprintf(kbKeyDVEBind+agentID, dveID)
	}

	if err := r.db.StoreJSON(bindKey, objectiveName); err != nil {
		return fmt.Errorf("knirvbase bind agent: %w", err)
	}

	r.mu.Lock()
	if dveID == "" {
		r.globalBinds[agentID] = objectiveName
	} else {
		if r.dveBindings[dveID] == nil {
			r.dveBindings[dveID] = make(map[string]string)
		}
		r.dveBindings[dveID][agentID] = objectiveName
	}
	r.mu.Unlock()
	return nil
}

// GetObjectiveForAgent returns the IntentObjective bound to an agent (checks DVE first, then global).
func (r *IntentRegistry) GetObjectiveForAgent(agentID, dveID string) *IntentObjective {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Check DVE-specific binding first
	if dveID != "" {
		if binds, ok := r.dveBindings[dveID]; ok {
			if objName, ok := binds[agentID]; ok {
				if objs, ok := r.dveCache[dveID]; ok {
					return objs[objName]
				}
			}
		}
	}

	// Fall back to global binding
	if objName, ok := r.globalBinds[agentID]; ok {
		return r.cache[objName]
	}
	return nil
}

// GetGlobalObjectiveForAgent returns only global objective for an agent.
func (r *IntentRegistry) GetGlobalObjectiveForAgent(agentID string) *IntentObjective {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if objName, ok := r.globalBinds[agentID]; ok {
		return r.cache[objName]
	}
	return nil
}

// GetObjectiveForDVE returns per-DVE objective for an agent.
func (r *IntentRegistry) GetObjectiveForDVE(agentID, dveID string) *IntentObjective {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if binds, ok := r.dveBindings[dveID]; ok {
		if objName, ok := binds[agentID]; ok {
			if objs, ok := r.dveCache[dveID]; ok {
				return objs[objName]
			}
		}
	}
	return nil
}

// EvaluateTradeOffs returns the weighted priority score for a decision context.
func (r *IntentRegistry) EvaluateTradeOffs(agentID, dveID string, context map[string]float64) float64 {
	obj := r.GetObjectiveForAgent(agentID, dveID)
	if obj == nil {
		return 0.5
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

// IsActionAuthorized checks whether a proposed action is within authorized_actions.
func (r *IntentRegistry) IsActionAuthorized(agentID, dveID, action string) bool {
	obj := r.GetObjectiveForAgent(agentID, dveID)
	if obj == nil {
		return true
	}
	for _, a := range obj.AuthorizedActions {
		if a == action {
			return true
		}
	}
	return false
}

// ViolatesHardBoundary checks if a proposed action violates any hard limit.
func (r *IntentRegistry) ViolatesHardBoundary(agentID, dveID, action string) bool {
	obj := r.GetObjectiveForAgent(agentID, dveID)
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

// RecordAlignment stores an alignment measurement to KNIRVBASE.
func (r *IntentRegistry) RecordAlignment(rec *AlignmentRecord) error {
	if rec.ID == "" {
		rec.ID = uuid.NewString()
	}
	data, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	key := fmt.Sprintf(kbKeyAlignment, rec.AgentID, rec.ID)
	if err := r.db.StoreJSON(key, data); err != nil {
		return err
	}

	r.mu.Lock()
	r.alignmentCache = append(r.alignmentCache, rec)
	if len(r.alignmentCache) > 100 {
		r.alignmentCache = r.alignmentCache[len(r.alignmentCache)-100:]
	}
	r.mu.Unlock()

	return nil
}

// ListAlignmentRecords returns all alignment records for an agent.
func (r *IntentRegistry) ListAlignmentRecords(agentID string) ([]*AlignmentRecord, error) {
	// Implementation would iterate over matching keys in KNIRVBASE
	// Returns all alignment records for the given agentID
	return nil, nil
}

// GetRecentAlignmentRecords returns the most recent alignment records for an objective.
func (r *IntentRegistry) GetRecentAlignmentRecords(objectiveName string, limit int) []*AlignmentRecord {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var records []*AlignmentRecord
	for _, rec := range r.alignmentCache {
		if rec.ObjectiveName == objectiveName {
			records = append(records, rec)
		}
	}

	if len(records) > limit {
		return records[len(records)-limit:]
	}
	return records
}

// ListObjectives returns all objectives (global and for a specific DVE).
func (r *IntentRegistry) ListObjectives(dveID string) []*IntentObjective {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*IntentObjective
	for _, obj := range r.cache {
		result = append(result, obj)
	}
	if dveID != "" {
		if dveObjs, ok := r.dveCache[dveID]; ok {
			for _, obj := range dveObjs {
				result = append(result, obj)
			}
		}
	}
	return result
}

// loadFromDB hydrates the in-memory cache from KNIRVBASE on startup.
func (r *IntentRegistry) loadFromDB() error {
	// Implementation would scan KNIRVBASE keys matching icme:* prefixes
	// and hydrate the cache maps
	return nil
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
	DVEID        string
	Action       string
	CustomerTier string
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
type DelegationFramework struct {
	registry *IntentRegistry
	logger   *zap.Logger
}

func NewDelegationFramework(registry *IntentRegistry, logger *zap.Logger) *DelegationFramework {
	return &DelegationFramework{registry: registry, logger: logger}
}

// Resolve determines the appropriate response to a proposed action.
func (d *DelegationFramework) Resolve(ctx DecisionContext) DecisionResult {
	// Hard boundary check first
	if d.registry.ViolatesHardBoundary(ctx.AgentID, ctx.DVEID, ctx.Action) {
		d.logger.Warn("icme hard boundary violation",
			zap.String("agent_id", ctx.AgentID),
			zap.String("dve_id", ctx.DVEID),
			zap.String("action", ctx.Action),
		)
		return DecisionResult{
			Approved: false,
			Action:   "deny",
			Reason:   "hard boundary violated: " + ctx.Action,
		}
	}

	// Authorization check
	if !d.registry.IsActionAuthorized(ctx.AgentID, ctx.DVEID, ctx.Action) {
		return DecisionResult{
			Approved: false,
			Action:   "escalate_to_manager",
			Reason:   "action not in authorized list: " + ctx.Action,
		}
	}

	// Tier-based priority override
	if ctx.CustomerTier == "VIP" {
		return DecisionResult{
			Approved: true,
			Action:   "escalate_to_specialist",
			Reason:   "VIP tier prioritizes satisfaction",
		}
	}

	// Amount-based threshold
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

## 6. Layer 3 — KNIRVGRAPH Enhancement (Temporal Hypergraph)

### 6.1 Extend Existing KNIRVGRAPH Engine

Modify `backend/internal/reasoning/graph/engine.go` to add temporal hypergraph capabilities:

```go
package graph

import (
	"sync"
	"time"

	"backend_server/internal/services/icme"
	"backend_server/internal/storage/mdstorage"
)

// HyperNode represents an entity extracted from a signal.
type HyperNode struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"` // PERSON, ORG, ERROR, SOLUTION, CONFIG, EVENT
	Text       string                 `json:"text"`
	Attributes map[string]interface{} `json:"attributes"`
	FirstSeen  time.Time              `json:"first_seen"`
	LastSeen   time.Time              `json:"last_seen"`
	SignalIDs  []string               `json:"signal_ids"`
}

// HyperEdge represents a relationship between nodes with temporal metadata.
type HyperEdge struct {
	ID              string                 `json:"id"`
	FromNodeID      string                 `json:"from_node_id"`
	ToNodeID        string                 `json:"to_node_id"`
	RelationType    string                 `json:"relation_type"`
	Timestamp       time.Time              `json:"timestamp"`
	SignalID        string                 `json:"signal_id"`
	AgentID         string                 `json:"agent_id"`
	DVEID           string                 `json:"dve_id"`
	ObjectiveName   string                 `json:"objective_name"`
	AlignmentScore  float64                `json:"alignment_score"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// KNIRVGRAPHEngine extends the existing ReasoningEngine with temporal hypergraph.
type KNIRVGRAPHEngine struct {
	storage       *mdstorage.MarkdownStorageDriver
	hypergraph    *TemporalHypergraph
	windowSize    time.Duration
	maxNodes      int
	mu            sync.RWMutex
	logger        *zap.Logger
}

// TemporalHypergraph maintains the live temporal graph of entities.
type TemporalHypergraph struct {
	mu         sync.RWMutex
	nodes      map[string]*HyperNode
	edges      map[string][]*HyperEdge
	textIndex  map[string]string
	windowSize time.Duration
	maxNodes   int
	logger     *zap.Logger
}

func NewTemporalHypergraph(windowSize time.Duration, maxNodes int, logger *zap.Logger) *TemporalHypergraph {
	return &TemporalHypergraph{
		nodes:      make(map[string]*HyperNode),
		edges:     make(map[string][]*HyperEdge),
		textIndex: make(map[string]string),
		windowSize: windowSize,
		maxNodes:   maxNodes,
		logger:     logger,
	}
}

// InsertSignal adds a signal's entities and relations to the hypergraph.
func (hg *TemporalHypergraph) InsertSignal(sig *icme.IntentionalSignal) {
	hg.mu.Lock()
	defer hg.mu.Unlock()

	nodeIDs := make(map[string]string)
	for _, ent := range sig.Entities {
		nodeID := hg.upsertNode(ent, sig)
		nodeIDs[ent.ID] = nodeID
	}

	for _, rel := range sig.Relations {
		fromNodeID, ok1 := nodeIDs[rel.FromEntityID]
		toNodeID, ok2 := nodeIDs[rel.ToEntityID]
		if !ok1 || !ok2 {
			continue
		}
		edge := &HyperEdge{
			ID:             sig.ID + ":" + rel.FromEntityID + ":" + rel.ToEntityID,
			FromNodeID:     fromNodeID,
			ToNodeID:       toNodeID,
			RelationType:   rel.RelationType,
			Timestamp:      sig.Timestamp,
			SignalID:       sig.ID,
			AgentID:        sig.AgentID,
			DVEID:          sig.DVEID,
			ObjectiveName:  sig.ObjectiveName,
			AlignmentScore: sig.AlignmentScore,
			Metadata: map[string]interface{}{
				"confidence": rel.Confidence,
				"source":     string(sig.Source),
			},
		}
		hg.edges[fromNodeID] = append(hg.edges[fromNodeID], edge)
	}

	hg.pruneOldEdges()
}

// Neighbors returns all nodes reachable from nodeID within maxHops.
func (hg *TemporalHypergraph) Neighbors(nodeID string, maxHops int, relType string) []*HyperNode {
	hg.mu.RLock()
	defer hg.mu.RUnlock()

	visited := make(map[string]bool)
	var result []*HyperNode
	hg.dfs(nodeID, maxHops, relType, visited, &result)
	return result
}

// Snapshot returns a point-in-time view of the graph.
func (hg *TemporalHypergraph) Snapshot() (nodes []*HyperNode, edges []*HyperEdge) {
	hg.mu.RLock()
	defer hg.mu.RUnlock()
	for _, n := range hg.nodes {
		nodes = append(nodes, n)
	}
	for _, edgeList := range hg.edges {
		edges = append(edges, edgeList...)
	}
	return
}

// Additional methods: upsertNode, dfs, pruneOldEdges (similar to original spec)
```

---

## 7. Layer 4 — Embedding Generation & Semantic Indexing

### 7.1 Cloudflare → Ollama Fallback Embedding Provider

Create `backend/internal/services/icme/embedding_provider.go`:

```go
package icme

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type EmbeddingProvider struct {
	CloudflareURL string
	OllamaURL     string
	Model         string
	HTTPClient    *http.Client
	UseCloudflare bool
	mu            sync.RWMutex
}

type CloudflareRequest struct {
	Text string `json:"text"`
}

type CloudflareResponse struct {
	Success   bool      `json:"success"`
	Embedding []float32 `json:"embedding"`
}

type OllamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type OllamaResponse struct {
	Embedding []float32 `json:"embedding"`
}

func NewEmbeddingProvider() (*EmbeddingProvider, error) {
	cloudflareURL := os.Getenv("CLOUDFLARE_EMBEDDINGS_URL")
	if cloudflareURL == "" {
		cloudflareURL = "https://embeddings.knirv.com"
	}

	ollamaURL := os.Getenv("OLLAMA_BASE_URL")
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}

	model := os.Getenv("OLLAMA_EMBEDDING_MODEL")
	if model == "" {
		model = "nomic-embed-text"
	}

	provider := &EmbeddingProvider{
		CloudflareURL: cloudflareURL,
		OllamaURL:     ollamaURL,
		Model:         model,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
		UseCloudflare: true,
	}

	// Check if Ollama is installed, install if not
	if err := provider.ensureOllamaInstalled(); err != nil {
		provider.logger.Warn("ollama not available", zap.Error(err))
	}

	return provider, nil
}

// ensureOllamaInstalled checks if Ollama is installed, installs if not
func (p *EmbeddingProvider) ensureOllamaInstalled() error {
	// Check if Ollama is available
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", p.OllamaURL+"/api/tags", nil)
	resp, err := p.HTTPClient.Do(req)
	if err == nil && resp.StatusCode == http.StatusOK {
		resp.Body.Close()
		return nil // Already installed
	}

	// Try to install Ollama
	fmt.Println("Installing Ollama...")
	cmd := exec.Command("bash", "-c", "curl -fsSL https://ollama.ai/install.sh | sh")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install ollama: %w", err)
	}

	// Start Ollama server in background
	go func() {
		cmd := exec.Command("ollama", "serve")
		cmd.Start()
	}()

	time.Sleep(2 * time.Second)
	return nil
}

// GetEmbedding tries Cloudflare first, falls back to Ollama
func (p *EmbeddingProvider) GetEmbedding(text string) ([]float32, error) {
	cleanText := strings.TrimSpace(text)
	if len(cleanText) == 0 {
		return nil, fmt.Errorf("empty text provided")
	}

	// Try Cloudflare first
	p.mu.RLock()
	useCF := p.UseCloudflare
	p.mu.RUnlock()

	if useCF {
		embedding, err := p.getCloudflareEmbedding(cleanText)
		if err == nil {
			return embedding, nil
		}
		fmt.Printf("Cloudflare failed: %v, falling back to Ollama\n", err)
	}

	// Fall back to Ollama
	return p.getOllamaEmbedding(cleanText)
}

func (p *EmbeddingProvider) getCloudflareEmbedding(text string) ([]float32, error) {
	req := CloudflareRequest{Text: text}
	jsonData, _ := json.Marshal(req)

	httpReq, _ := http.NewRequest("POST", p.CloudflareURL, bytes.NewBuffer(jsonData))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("cloudflare request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("cloudflare returned %d: %s", resp.StatusCode, string(body))
	}

	var cfResp CloudflareResponse
	if err := json.NewDecoder(resp.Body).Decode(&cfResp); err != nil {
		return nil, fmt.Errorf("decode failed: %w", err)
	}

	if !cfResp.Success || len(cfResp.Embedding) == 0 {
		return nil, fmt.Errorf("empty embedding")
	}

	return cfResp.Embedding, nil
}

func (p *EmbeddingProvider) getOllamaEmbedding(text string) ([]float32, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	req := OllamaRequest{Model: p.Model, Prompt: text}
	jsonData, _ := json.Marshal(req)

	httpReq, _ := http.NewRequestWithContext(ctx, "POST", p.OllamaURL+"/api/embeddings", bytes.NewBuffer(jsonData))
	httpReq.Header.Set("Content-Type", "application/json")

	ollamaClient := &http.Client{Timeout: 90 * time.Second}
	resp, err := ollamaClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama returned %d: %s", resp.StatusCode, string(body))
	}

	var ollamaResp OllamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("decode failed: %w", err)
	}

	if len(ollamaResp.Embedding) == 0 {
		return nil, fmt.Errorf("empty embedding")
	}

	return ollamaResp.Embedding, nil
}
```

### 7.2 spaCy NER Provider (Go CGO)

Create `backend/internal/services/icme/ner_provider.go`:

```go
package icme

import (
	"fmt"

	"github.com/am-sokolov/go-spacy"
)

type NERProvider struct {
	nlp *spacy.NLP
}

func NewNERProvider() (*NERProvider, error) {
	// Try to load spaCy model, download if not available
	nlp, err := spacy.NewNLP("en_core_web_sm")
	if err != nil {
		// Model not found, download it
		fmt.Println("Downloading spaCy model en_core_web_sm...")
		if err := downloadSpacyModel(); err != nil {
			return nil, fmt.Errorf("spaCy model download failed: %w", err)
		}
		nlp, err = spacy.NewNLP("en_core_web_sm")
		if err != nil {
			return nil, fmt.Errorf("spaCy initialization failed: %w", err)
		}
	}

	return &NERProvider{nlp: nlp}, nil
}

func downloadSpacyModel() error {
	// Run: python -m spacy download en_core_web_sm
	return nil // Implementation would run the download command
}

// ExtractEntitiesAndRelations extracts named entities and infers relations.
func (n *NERProvider) ExtractEntitiesAndRelations(text string) ([]ExtractedEntity, []ExtractedRelation, error) {
	doc := n.nlp.ReadDoc(text)

	entities := make([]ExtractedEntity, 0)
	ents := doc.Ents()
	for _, ent := range ents {
		entities = append(entities, ExtractedEntity{
			ID:    fmt.Sprintf("ent_%d", ent.Start),
			Text:  ent.Text,
			Label: ent.Label,
			Score: 0.9, // spaCy doesn't provide confidence by default
			Start: ent.Start,
			End:   ent.End,
		})
	}

	// Simple co-occurrence relation extraction
	relations := make([]ExtractedRelation, 0)
	entList := entities
	for i, e1 := range entList {
		for _, e2 := range entList[i+1:] {
			rel := inferRelation(e1.Label, e2.Label)
			if rel != "" {
				relations = append(relations, ExtractedRelation{
					FromEntityID: e1.ID,
					ToEntityID:   e2.ID,
					RelationType: rel,
					Confidence:   0.75,
				})
			}
		}
	}

	return entities, relations, nil
}

func inferRelation(label1, label2 string) string {
	// Domain-specific relation inference
	if label1 == "ERROR" && label2 == "SOLUTION" {
		return "RESOLVED_BY"
	}
	if label1 == "CONFIG" && label2 == "EVENT" {
		return "TRIGGERS"
	}
	return ""
}
```

### 7.3 FAISS Index Manager

Create `backend/internal/services/icme/faiss_manager.go` (similar to original spec, using KNIRVBASE for metadata):

```go
package icme

import (
	"encoding/json"
	"fmt"
	"sync"

	faiss "github.com/DataIntelligenceCrew/go-faiss"
	"go.uber.org/zap"

	"backend_server/internal/data_engine"
)

const EmbedDim = 768 // Standard dimension for nomic-embed-text

type VectorMeta struct {
	VectorID  int64  `json:"vector_id"`
	SignalID  string `json:"signal_id"`
	AgentID   string `json:"agent_id"`
	DVEID     string `json:"dve_id"`
	Summary   string `json:"summary"`
	Objective string `json:"objective"`
}

type FAISSIndexManager struct {
	mu       sync.Mutex
	index    *faiss.IndexImpl
	nextID   int64
	db       *data_engine.BuntDBManager
	logger   *zap.Logger
}

func NewFAISSIndexManager(db *data_engine.BuntDBManager, logger *zap.Logger) (*FAISSIndexManager, error) {
	idx, err := faiss.NewIndexFlatL2(EmbedDim)
	if err != nil {
		return nil, fmt.Errorf("faiss index create: %w", err)
	}
	return &FAISSIndexManager{
		index:  idx,
		db:     db,
		logger: logger,
	}, nil
}

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

	// Persist metadata to KNIRVBASE
	data, _ := json.Marshal(meta)
	key := fmt.Sprintf("icme:vectors:%d", id)
	if err := m.db.StoreJSON(key, data); err != nil {
		m.logger.Warn("icme faiss meta persist failed", zap.Error(err))
	}

	return id, nil
}

func (m *FAISSIndexManager) Search(query []float32, k int) ([]VectorMeta, []float32, error) {
	// Implementation similar to original spec
	return nil, nil, nil
}
```

---

## 8. Layer 5 — Hybrid Retrieval API

Create `backend/internal/services/icme/hybrid_search.go`:

```go
package icme

import (
	"context"
	"sort"

	"go.uber.org/zap"
)

type HybridResult struct {
	SignalID       string        `json:"signal_id"`
	AgentID        string        `json:"agent_id"`
	DVEID          string        `json:"dve_id"`
	Content        string        `json:"content"`
	ObjectiveName  string        `json:"objective_name"`
	VectorScore    float32       `json:"vector_score"`
	GraphHops     int           `json:"graph_hops"`
	CombinedScore float64       `json:"combined_score"`
	Nodes          []*HyperNode  `json:"related_nodes"`
	AlignmentScore float64      `json:"alignment_score"`
}

type HybridSearchEngine struct {
	faissManager *FAISSIndexManager
	graph        *TemporalHypergraph
	embedProvider *EmbeddingProvider
	intentReg    *IntentRegistry
	logger       *zap.Logger
}

func NewHybridSearchEngine(
	faissManager *FAISSIndexManager,
	graph *TemporalHypergraph,
	embedProvider *EmbeddingProvider,
	intentReg *IntentRegistry,
	logger *zap.Logger,
) *HybridSearchEngine {
	return &HybridSearchEngine{
		faissManager: faissManager,
		graph:        graph,
		embedProvider: embedProvider,
		intentReg:    intentReg,
		logger:       logger,
	}
}

func (s *HybridSearchEngine) Search(ctx context.Context, query, agentID, dveID string, topK int) ([]HybridResult, error) {
	// Step 1: Embed query
	embedding, err := s.embedProvider.GetEmbedding(query)
	if err != nil {
		return nil, err
	}

	// Step 2: FAISS search
	metas, scores, err := s.faissManager.Search(embedding, topK*3)
	if err != nil {
		return nil, err
	}

	// Step 3: Graph traversal + score fusion
	obj := s.intentReg.GetObjectiveForAgent(agentID, dveID)
	results := make([]HybridResult, 0, len(metas))

	for i, meta := range metas {
		nodes := s.graph.Neighbors(meta.SignalID, 2, "")

		alignBoost := 0.0
		if obj != nil && meta.Objective == obj.Name {
			alignBoost = 0.15
		}

		vecSim := 1.0 / (1.0 + float64(scores[i]))
		combined := vecSim + float64(len(nodes))*0.01 + alignBoost

		results = append(results, HybridResult{
			SignalID:       meta.SignalID,
			AgentID:        meta.AgentID,
			DVEID:          meta.DVEID,
			Content:        meta.Summary,
			ObjectiveName:   meta.Objective,
			VectorScore:    scores[i],
			GraphHops:      len(nodes),
			CombinedScore:  combined,
			Nodes:          nodes,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].CombinedScore > results[j].CombinedScore
	})

	if len(results) > topK {
		return results[:topK], nil
	}
	return results, nil
}
```

---

## 9. Layer 6 — Feedback & Alignment Loops

### 9.1 Factuality Slice Adapter

The ICME integrates with the existing `FactualityValidator` in `backend/internal/services/validation/llm_validators.go` (line 101). The adapter bridges intent objectives, user ontology, and onboarding preferences to compute domain-specific factuality scores.

Create `backend/internal/services/icme/factuality_adapter.go`:

```go
package icme

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

type FactualityAdapter struct {
	validationEndpoint string
	httpClient        *http.Client
	intentRegistry    *IntentRegistry
	ontologyStore     *OntologyStore
	userPreferences  *PreferenceStore
	logger            *zap.Logger
}

type OntologyStore struct {
	db *BuntDBManager
}

type PreferenceStore struct {
	db *BuntDBManager
}

type FactualityRequest struct {
	Prompt            string   `json:"prompt"`
	Response          string   `json:"response"`
	AgentID           string   `json:"agent_id"`
	DVEID             string   `json:"dve_id"`
	OntologyDomains   []string `json:"ontology_domains"`
	ObjectiveName     string   `json:"objective_name"`
	PreferenceWeights map[string]float64 `json:"preference_weights"`
}

type FactualityResponse struct {
	IsAccurate   bool                      `json:"is_accurate"`
	Confidence   float64                   `json:"confidence"`
	Citations    []int                     `json:"citations"`
	Refused      bool                      `json:"refused"`
	Explanation  string                    `json:"explanation"`
	DomainScores map[string]float64        `json:"domain_scores"`
}

func NewFactualityAdapter(
	validationEndpoint string,
	intentRegistry *IntentRegistry,
	db *BuntDBManager,
	logger *zap.Logger,
) *FactualityAdapter {
	return &FactualityAdapter{
		validationEndpoint: validationEndpoint,
		httpClient:         &http.Client{Timeout: 30 * time.Second},
		intentRegistry:    intentRegistry,
		ontologyStore:     &OntologyStore{db: db},
		userPreferences:   &PreferenceStore{db: db},
		logger:            logger,
	}
}

func (f *FactualityAdapter) ValidateFactuality(ctx context.Context, signal *IntentionalSignal) (*FactualityResponse, error) {
	obj := f.intentRegistry.GetObjectiveForAgent(signal.AgentID, signal.DVEID)
	
	ontologyDomains := f.getOntologyDomains(obj)
	preferenceWeights := f.getPreferenceWeights(signal.AgentID, signal.DVEID)

	req := FactualityRequest{
		Prompt:            signal.Content,
		Response:          signal.Content,
		AgentID:           signal.AgentID,
		DVEID:             signal.DVEID,
		OntologyDomains:   ontologyDomains,
		ObjectiveName:     signal.ObjectiveName,
		PreferenceWeights: preferenceWeights,
	}

	jsonData, _ := json.Marshal(req)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", f.validationEndpoint+"/api/validation/factuality", bytes.NewBuffer(jsonData))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := f.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("factuality validation request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("factuality validation returned %d", resp.StatusCode)
	}

	var factResp FactualityResponse
	if err := json.NewDecoder(resp.Body).Decode(&factResp); err != nil {
		return nil, fmt.Errorf("decode factuality response: %w", err)
	}

	f.logger.Debug("factuality validated",
		zap.String("agent_id", signal.AgentID),
		zap.Float64("confidence", factResp.Confidence),
		zap.Bool("is_accurate", factResp.IsAccurate),
	)

	return &factResp, nil
}

func (f *FactualityAdapter) getOntologyDomains(obj *IntentObjective) []string {
	if obj == nil {
		return []string{"general"}
	}

	domains := make([]string, 0, len(obj.DataSources))
	for _, ds := range obj.DataSources {
		domain := extractDomain(ds)
		if domain != "" {
			domains = append(domains, domain)
		}
	}

	if len(domains) == 0 {
		domains = []string{"general"}
	}
	return domains
}

func (f *FactualityAdapter) getPreferenceWeights(agentID, dveID string) map[string]float64 {
	weights := make(map[string]float64)

	prefKey := fmt.Sprintf("icme:preferences:%s:%s", dveID, agentID)
	var prefData []byte
	if err := f.userPreferences.db.GetJSON(prefKey, &prefData); err == nil {
		json.Unmarshal(prefData, &weights)
	}

	if len(weights) == 0 {
		weights = map[string]float64{
			"accuracy":    0.4,
			"relevance":   0.3,
			"coherence":   0.2,
			"safety":      0.1,
		}
	}
	return weights
}

func extractDomain(dataSource string) string {
	domainKeywords := map[string][]string{
		"technical":   {"api", "code", "sdk", "implementation"},
		"scientific":  {"research", "study", "experiment", "data"},
		"historical":  {"history", "event", "timeline", "past"},
		"financial":   {"finance", "money", "market", "investment"},
		"medical":     {"health", "medical", "clinical", "patient"},
	}

	lower := strings.ToLower(dataSource)
	for domain, keywords := range domainKeywords {
		for _, kw := range keywords {
			if strings.Contains(lower, kw) {
				return domain
			}
		}
	}
	return "general"
}

func (f *FactualityAdapter) ComputeAlignmentScore(ctx context.Context, signal *IntentionalSignal) (float64, error) {
	factResp, err := f.ValidateFactuality(ctx, signal)
	if err != nil {
		return 0.5, err
	}

	obj := f.intentRegistry.GetObjectiveForAgent(signal.AgentID, signal.DVEID)
	if obj == nil {
		return factResp.Confidence, nil
	}

	baseScore := factResp.Confidence

	domainPenalty := 0.0
	for domain, score := range factResp.DomainScores {
		expected, ok := obj.TradeOffs[domain]
		if ok {
			domainPenalty += (expected - score) * 0.1
		}
	}

	preferenceWeights := f.getPreferenceWeights(signal.AgentID, signal.DVEID)
	preferenceBoost := 0.0
	for pref, weight := range preferenceWeights {
		if strings.Contains(factResp.Explanation, pref) {
			preferenceBoost += weight * 0.05
		}
	}

	alignmentScore := baseScore + preferenceBoost + domainPenalty

	if alignmentScore < 0 {
		alignmentScore = 0
	}
	if alignmentScore > 1 {
		alignmentScore = 1
	}

	return alignmentScore, nil
}
```

### 9.2 Alignment Loop

Create `backend/internal/services/icme/alignment_loop.go`:

```go
package icme

import (
	"context"
	"time"

	"go.uber.org/zap"
)

type AlignmentLoop struct {
	registry        *IntentRegistry
	factualityAdapter *FactualityAdapter
	logger          *zap.Logger
	evalInterval    time.Duration
	driftThreshold float64
}

func NewAlignmentLoop(
	registry *IntentRegistry,
	factualityAdapter *FactualityAdapter,
	evalInterval time.Duration,
	driftThreshold float64,
	logger *zap.Logger,
) *AlignmentLoop {
	return &AlignmentLoop{
		registry:          registry,
		factualityAdapter: factualityAdapter,
		evalInterval:      evalInterval,
		driftThreshold:    driftThreshold,
		logger:            logger,
	}
}

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

func (l *AlignmentLoop) runEvaluation(ctx context.Context) {
	objectives := l.registry.ListObjectives("")
	for _, obj := range objectives {
		records := l.registry.GetRecentAlignmentRecords(obj.Name, 10)
		if len(records) < 2 {
			continue
		}

		var drift float64
		for i := 1; i < len(records); i++ {
			drift += records[i].AlignmentScore - records[i-1].AlignmentScore
		}
		drift /= float64(len(records) - 1)

		if drift < -l.driftThreshold {
			l.logger.Warn("alignment drift detected",
				zap.String("objective", obj.Name),
				zap.Float64("drift", drift),
			)
		}
	}
}

func (l *AlignmentLoop) Evaluate(ctx context.Context, signal *IntentionalSignal) (*AlignmentRecord, error) {
	alignmentScore, err := l.factualityAdapter.ComputeAlignmentScore(ctx, signal)
	if err != nil {
		return nil, err
	}

	obj := l.registry.GetObjectiveForAgent(signal.AgentID, signal.DVEID)
	decision := "approved"
	if obj != nil {
		authorized := false
		for _, a := range obj.AuthorizedActions {
			if a == signal.Content {
				authorized = true
				break
			}
		}
		if !authorized {
			decision = "escalated"
			alignmentScore *= 0.8
		}
	}

	rec := &AlignmentRecord{
		ID:             "",
		AgentID:        signal.AgentID,
		DVEID:          signal.DVEID,
		ObjectiveName:  signal.ObjectiveName,
		SignalID:       signal.ID,
		Decision:       decision,
		Outcome:        "completed",
		AlignmentScore: alignmentScore,
		FidelityScore:  alignmentScore,
		Timestamp:      time.Now(),
	}

	if err := l.registry.RecordAlignment(rec); err != nil {
		l.logger.Error("failed to record alignment", zap.Error(err))
	}

	return rec, nil
}
```

---

## 10. Configuration Changes

### 10.1 Config Struct

Add to `backend/internal/config/config.go`:

```go
type ICMEConfig struct {
	Enabled                bool          `mapstructure:"enabled"`
	CloudflareEmbeddingsURL string      `mapstructure:"cloudflare_embeddings_url"`
	OllamaBaseURL         string        `mapstructure:"ollama_base_url"`
	OllamaEmbeddingModel  string        `mapstructure:"ollama_embedding_model"`
	SpacyModel            string        `mapstructure:"spacy_model"`
	GraphWindowSize       time.Duration `mapstructure:"graph_window_size"`
	GraphMaxNodes         int           `mapstructure:"graph_max_nodes"`
	EmbedDimension        int           `mapstructure:"embed_dimension"`
	DefaultTopK           int           `mapstructure:"default_top_k"`
	AlignmentEvalInterval time.Duration `mapstructure:"alignment_eval_interval"`
	DriftThreshold        float64       `mapstructure:"drift_threshold"`
	SignalQueueSize       int           `mapstructure:"signal_queue_size"`
	SignalWorkers         int           `mapstructure:"signal_workers"`
}
```

### 10.2 YAML Config

```yaml
icme:
  enabled: true
  cloudflare_embeddings_url: "https://embeddings.knirv.com"
  ollama_base_url: "http://localhost:11434"
  ollama_embedding_model: "nomic-embed-text"
  spacy_model: "en_core_web_sm"
  graph_window_size: "10m"
  graph_max_nodes: 10000
  embed_dimension: 768
  default_top_k: 10
  alignment_eval_interval: "5m"
  drift_threshold: 0.20
  signal_queue_size: 512
  signal_workers: 4
```

---

## 11. API Reference

| Method | Path | Description |
|--------|------|-------------|
| `GET`  | `/api/icme/objectives` | List all objectives (global and per-DVE) |
| `POST` | `/api/icme/objectives` | Create a new objective |
| `GET`  | `/api/icme/objectives/{name}` | Get objective by name |
| `PUT`  | `/api/icme/objectives/{name}` | Update objective |
| `POST` | `/api/icme/agents/{agentID}/bind` | Bind agent to objective (global) |
| `POST` | `/api/icme/dve/{dveID}/agents/{agentID}/bind` | Bind agent to objective (per-DVE) |
| `GET`  | `/api/icme/agents/{agentID}/objective` | Get agent's current objective |
| `GET`  | `/api/icme/search?q=&agent_id=&dve_id=&top_k=` | Hybrid semantic + graph search |
| `GET`  | `/api/icme/alignment/{agentID}` | Get alignment history |
| `POST` | `/api/icme/alignment/evaluate` | Evaluate and record alignment |
| `GET`  | `/api/icme/graph/snapshot` | Full hypergraph snapshot |
| `GET`  | `/api/icme/graph/neighbors/{nodeID}` | Graph neighbors |
| `POST` | `/api/icme/delegate` | Resolve action via delegation |

---

## 12. Frontend Integration

### 12.1 CDE Panel Modals

Add to `frontend/src/components/dashboard/dve-workspace-panel.tsx`:

```tsx
import IntentConsole from './icme/IntentConsole';
import ContextExplorer from './icme/ContextExplorer';
import AlignmentMonitor from './icme/AlignmentMonitor';

export default function CDEPanel({ node, ...props }) {
  const [showIntent, setShowIntent] = useState(false);
  const [showContext, setShowContext] = useState(false);
  const [showAlignment, setShowAlignment] = useState(false);

  // Add toolbar buttons
  // <button onClick={() => setShowIntent(true)} title="Intent Console">
  //   <TargetIcon className="w-4 h-4" />
  // </button>

  return (
    <>
      {showIntent && <IntentConsole isOpen={showIntent} onClose={() => setShowIntent(false)} nodeId={node?.id} />}
      {showContext && <ContextExplorer isOpen={showContext} onClose={() => setShowContext(false)} nodeId={node?.id} />}
      {showAlignment && <AlignmentMonitor isOpen={showAlignment} onClose={() => setShowAlignment(false)} nodeId={node?.id} />}
    </>
  );
}
```

### 12.2 Onboarding OKR Form

Add to `frontend/src/components/onboarding/onboarding-guide.tsx`:

```tsx
interface OKRData {
  objectives: Array<{
    name: string;
    description: string;
    keyResults: string[];
  }>;
  tradeOffs: Record<string, number>;
  globalIntent: string;
}

function OKRIntentForm({ onSubmit }: { onSubmit: (data: OKRData) => void }) {
  const [objectives, setObjectives] = useState<OKRData['objectives']>([]);
  const [globalIntent, setGlobalIntent] = useState('');

  const handleSubmit = () => {
    onSubmit({ objectives, tradeOffs: {}, globalIntent });
  };

  return (
    <div className="space-y-6">
      <h3>Define Your Organization's Intent</h3>
      <p>Set global OKRs and objectives that will guide all DVE agents.</p>
      
      {/* Objective inputs */}
      <div>
        <Label>Global Intent Statement</Label>
        <Textarea 
          value={globalIntent}
          onChange={(e) => setGlobalIntent(e.target.value)}
          placeholder="e.g., Maximize validation accuracy while minimizing false positives"
        />
      </div>
      
      {/* Key Results Builder */}
      {/* Trade-off Sliders */}
      
      <Button onClick={handleSubmit}>Save Intent Configuration</Button>
    </div>
  );
}
```

---

## 13. Testing Strategy

Create `backend/internal/services/icme/icme_test.go`:

```go
package icme_test

import (
	"testing"

	"backend_server/internal/services/icme"
)

func TestIntentRegistry_GlobalAndDVEScope(t *testing.T) {
	// Test global objective registration
	// Test per-DVE objective registration
	// Test binding precedence (DVE overrides global)
}

func TestDelegationFramework_HardBoundary(t *testing.T) {
	// Test hard boundary enforcement
}

func TestTemporalHypergraph_InsertAndNeighbors(t *testing.T) {
	// Test node/edge insertion and traversal
}
```

---

## 14. Deployment & Scaling

### 14.1 Main Server Integration

Add to `backend/cmd/backend_server/main.go`:

```go
if cfg.ICME.Enabled {
	// Initialize embedding provider (Cloudflare → Ollama)
	embedProvider, err := icme.NewEmbeddingProvider()
	if err != nil {
		logger.Fatal("icme embedding provider", zap.Error(err))
	}

	// Initialize spaCy NER
	nerClient, err := icme.NewNERProvider()
	if err != nil {
		logger.Fatal("icme ner provider", zap.Error(err))
	}

	// Initialize KNIRVGRAPH with hypergraph
	graphEngine := graph.NewKNIRVGRAPHEngine(mdStorage, cfg.ICME.GraphWindowSize, cfg.ICME.GraphMaxNodes, logger)

	// Initialize FAISS
	faissIdx, err := icme.NewFAISSIndexManager(dbManager, logger)
	if err != nil {
		logger.Fatal("icme faiss index", zap.Error(err))
	}

	// Initialize Intent Registry
	intentReg, err := icme.NewIntentRegistry(dbManager, logger)
	if err != nil {
		logger.Fatal("icme intent registry", zap.Error(err))
	}

	// Initialize services
	delegation := icme.NewDelegationFramework(intentReg, logger)
	searchEngine := icme.NewHybridSearchEngine(faissIdx, graphEngine.Hypergraph(), embedProvider, intentReg, logger)
	
	// Initialize Factuality Slice Adapter (integrates with existing validation service)
	factualityAdapter := icme.NewFactualityAdapter(
		"http://localhost:"+cfg.API.Port,
		intentReg,
		dbManager,
		logger,
	)
	alignmentLoop := icme.NewAlignmentLoop(
		intentReg,
		factualityAdapter,
		cfg.ICME.AlignmentEvalInterval,
		cfg.ICME.DriftThreshold,
		logger,
	)

	// Initialize Signal Router
	signalRouter := icme.NewSignalRouter(intentReg, nerClient, embedProvider, graphEngine, faissIdx, activeMemoryService, logger)

	// Create ICME Service
	icmeSvc := icme.NewICMEService(intentReg, graphEngine, faissIdx, searchEngine, delegation, alignmentLoop, signalRouter, logger)

	// Register routes
	icmeSvc.RegisterRoutes(s.router, s.authMiddleware)

	// Start background goroutines
	signalRouter.Start(ctx)
	go alignmentLoop.Start(ctx)

	logger.Info("icme initialized",
		zap.String("cloudflare", cfg.ICME.CloudflareEmbeddingsURL),
		zap.String("ollama", cfg.ICME.OllamaBaseURL),
	)
}
```

---

## Summary

The **Intentional Context Memory Engine** integrates:

| Capability | Implementation |
|-----------|---------------|
| **Database** | KNIRVBASE via existing BuntDBManager |
| **Embedding** | Cloudflare → Ollama fallback (Go HTTP) |
| **NER** | spaCy via Go CGO bindings |
| **Hypergraph** | Enhanced KNIRVGRAPH with temporal edges |
| **Intent Scope** | Both Global and Per-DVE workspace |
| **Frontend** | CDE Panel modals + Onboarding OKR form |

The feedback loop between the alignment scorer and Factuality Slice confidence scores closes the intent engineering cycle, ensuring the system continuously self-corrects rather than drifting from declared organizational goals over time.
