# HEART × Gorgonite: Implementation Plan

> Step-by-step implementation guide derived from `transformer_opportunities.md`.
> Covers the full integration of the migrated Gorgonite transformer code in
> `packages/KNIRVHASHER/pkg/hashing/transformer/` into the live KNIRVHASHER
> application, and the buildout of all three WASM execution paths.
>
> **Current state**: The transformer files (`main.go`, `heart_service.go`,
> `dynamic_graph.go`, `cerebras_bridge.go`) were migrated from KNIRVHEART but remain
> declared as `package main`. They cannot be imported by the KNIRVHASHER application.
> All phases below resolve this, then layer in the unified HEART system.

---

## Dependency Map

```
Phase 0 — Package Refactor (prerequisite for all phases)
    │
    ├── Phase 1 — Tokeniser + Deterministic Embedder
    │       │
    │       ├── Phase 2 — Gorgonite GPT Integration into HEARTService
    │       │       │
    │       │       ├── Phase 3 — HEARTService Endpoint & Inquiry Refactor
    │       │       │       │
    │       │       │       ├── Phase 4 — Multi-Source Data Ingestion (DATA_MINER)
    │       │       │       │
    │       │       │       ├── Phase 5 — 4-Stage Hierarchical Decision Pipeline
    │       │       │       │       │
    │       │       │       │       ├── Phase 6 — WASM Compilation Pipeline
    │       │       │       │       │       (tinygo + wazero)
    │       │       │       │       │
    │       │       │       │       └── Phase 7 — Bidirectional Verification
    │       │       │       │
    │       │       │       └── Phase 8 — HashNetwork Optional Fast-Path
    │       │       │
    │       │       └── Phase 9 — Softmax + Positional Encoding Fixes
    │       │
    │       └── Phase 10 — Variance-Guided Head Pruning + DynamicGraph NAS
    │
    ├── Phase 11 — ES Updates to EvolutionaryHarness (HashNetwork, optional)
    │
    ├── Phase 12 — Curriculum Training on Combined Corpus
    │
    └── Phase 13 — Entropy-Spike Detection + Self-Improving Loop
```

---

## Phase 0 — Transformer Package Refactor

**Problem**: All files in `packages/KNIRVHASHER/pkg/hashing/transformer/` are declared
`package main`. They cannot be imported. `main.go` also uses `gorgonia.org/gorgonia` and
`gorgonia.org/tensor` which are not in the KNIRVHASHER `go.mod`.

**Files affected**:
- `pkg/hashing/transformer/main.go` → declares `package main`, `func main()`
- `pkg/hashing/transformer/heart_service.go` → declares `package main`
- `pkg/hashing/transformer/dynamic_graph.go` → declares `package main`
- `pkg/hashing/transformer/cerebras_bridge.go` → declares `package main`
- `pkg/hashing/transformer/transformer.go` → correctly declares `package transformer` — leave as-is

### Step 0.1 — Add Gorgonia dependencies to go.mod

```bash
cd packages/KNIRVHASHER
go get gorgonia.org/gorgonia@latest
go get gorgonia.org/tensor@latest
go mod tidy
```

### Step 0.2 — Rename package declarations

In each file, change `package main` → `package transformer`.

| File | Change |
|------|--------|
| `pkg/hashing/transformer/main.go` | `package main` → `package transformer`; remove `func main()` (move to `cmd/heart/main.go`, Phase 2) |
| `pkg/hashing/transformer/heart_service.go` | `package main` → `package transformer` |
| `pkg/hashing/transformer/dynamic_graph.go` | `package main` → `package transformer` |
| `pkg/hashing/transformer/cerebras_bridge.go` | `package main` → `package transformer` |

### Step 0.3 — Resolve import conflicts

`main.go` defines its own `TransformerConfig`. `transformer.go` also defines `TransformerConfig`
(the hash-based version). Rename to resolve:

- **`main.go`** `TransformerConfig` → `GorgoniteConfig` (Gorgonia-based, uses `float64` weights)
- **`transformer.go`** `TransformerConfig` → keep as `HasherTransformerConfig` (seed-based, uses `[32]byte`)

Update all references within each file accordingly. The two structs co-exist in the same package
for two distinct model implementations.

### Step 0.4 — Fix internal references

`heart_service.go` (originally `package main`) references `CerebrasBridge` from
`cerebras_bridge.go`. After the package rename these are in the same package — no import needed.
Verify the following cross-file references compile:

- `heart_service.go` → `CerebrasBridge`, `NetworkMetricsProcessor` ✓ (same package)
- `cerebras_bridge.go` → `GPT` (the Gorgonia model type from `main.go`) ✓ (same package)
- `dynamic_graph.go` → `DynamicNode`, `Operation` ✓ (self-contained)

### Step 0.5 — Create cmd entrypoint

Create `packages/KNIRVHASHER/cmd/heart/main.go`:

```go
package main

import (
    "flag"
    "log"
    "knirvhasher/pkg/hashing/transformer"
    "knirvhasher/pkg/hashing/factory"
)

func main() {
    addr := flag.String("addr", ":8090", "HEARTService listen address")
    useHashNetwork := flag.Bool("hashnetwork", false, "Enable optional HashNetwork fast path")
    useCerebras := flag.Bool("cerebras", false, "Enable optional Cerebras WSE2 path")
    flag.Parse()

    cfg := transformer.DefaultHEARTConfig(*useHashNetwork, *useCerebras)
    svc, err := transformer.NewHEARTService(cfg, factory.DefaultHashMethodConfig())
    if err != nil {
        log.Fatalf("heart init: %v", err)
    }
    log.Printf("HEARTService listening on %s", *addr)
    log.Fatal(svc.ListenAndServe(*addr))
}
```

**Verification**: `cd packages/KNIRVHASHER && go build ./pkg/hashing/transformer/... && go build ./cmd/heart/`

---

## Phase 1 — Tokeniser + Deterministic Embedder Wire-up

**Addresses**: Option 5 (cl100k tokeniser), Custom Deterministic Embedder (System Fingerprint)

**Problem**: `main.go` contains a stub `BPETokenizer` whose `loadVocab()` returns an empty map —
every token encodes as `<UNK>`. The real cl100k tokeniser already exists at
`pipeline/2_DATA_ENCODER/pkg/tokenizer/tokenizer.go`. The deterministic embedder exists at
`pipeline/2_DATA_ENCODER/pkg/embeddings/deterministic.go` (`github.com/guiperry/text-embedder`).

### Step 1.1 — Extract tokeniser to shared module

Move (or symlink) `pipeline/2_DATA_ENCODER/pkg/tokenizer/` into a shared internal location:

```
packages/KNIRVHASHER/
  internal/
    shared/
      tokenizer/   ← copy or re-export from pipeline/2_DATA_ENCODER/pkg/tokenizer
      embeddings/  ← copy or re-export from pipeline/2_DATA_ENCODER/pkg/embeddings
```

Alternatively, since both are in the same Go module (`knirvhasher`), import directly:

```go
// In pkg/hashing/transformer/main.go (now package transformer)
import (
    knirvtokenizer "knirvhasher/pipeline/2_DATA_ENCODER/pkg/tokenizer"
    "knirvhasher/pipeline/2_DATA_ENCODER/pkg/embeddings"
)
```

### Step 1.2 — Replace BPETokenizer in `main.go`

Locate the stub tokeniser struct in `main.go` (the `BPETokenizer` type with `loadVocab()`
returning empty map). Replace with:

```go
// Remove: type BPETokenizer struct { ... }
// Remove: func NewBPETokenizer(...) ...
// Remove: func (b *BPETokenizer) loadVocab() ...
// Remove: func (b *BPETokenizer) Encode(...) ...

// Add to GorgoniteConfig or HEARTService init:
tokenizer, err := knirvtokenizer.New()
if err != nil {
    return nil, fmt.Errorf("tokenizer init: %w", err)
}
```

Update `GorgoniteConfig.VocabSize` default from `50257` → `100277` (cl100k actual vocab size).

### Step 1.3 — Replace Ollama BGE-768 with deterministic embedder

In any location calling Ollama or BGE embeddings, replace with:

```go
import "knirvhasher/pipeline/2_DATA_ENCODER/pkg/embeddings"

embedder := embeddings.NewDeterministicService()
vec, err := embedder.GetEmbedding(text)
// vec is []float32, fully deterministic, no network call
```

The `DeterministicService` implements the `EmbeddingProvider` interface — it is a drop-in
replacement for the Ollama client in `pipeline/2_DATA_ENCODER/pkg/ollama/ollama.go`.

Wire the deterministic embedder into:
- `HEARTService.processInquiry()` — for embedding the inquiry text before Stage 2 lookup
- `findSimilarErrors()` in `heart_service.go` — currently returns hardcoded scores; replace with `embed.Embed()` cosine similarity over a cached embedding store
- Stage 2 `DATA_ENCODER` pipeline (already uses `embeddings.DeterministicService` but with a fallback to Ollama — remove Ollama path)

### Step 1.4 — Update VarianceAnalyzer to use deterministic embedder

In `pipeline/2_DATA_ENCODER/pkg/analyzer/variance.go`, wherever embeddings are produced for
`Sample()`, confirm `DeterministicService.GetBatchEmbeddings()` is the call path. The
`bge_signal_indices.json` output is now guaranteed stable across restarts.

**Verification**: `go test ./pipeline/2_DATA_ENCODER/pkg/tokenizer/... ./pipeline/2_DATA_ENCODER/pkg/embeddings/...`

---

## Phase 2 — Gorgonite GPT Integration into HEARTService

**Addresses**: Core transformer integration gap

**Problem**: `HEARTService` in `heart_service.go` holds a `*CerebrasBridge` field but there
is no `*GPT` (Gorgonia model) field. The service currently has no model to run inference with —
it uses hardcoded heuristics in `generateRecommendedActions()`.

### Step 2.1 — Add GPT field to HEARTService

In `heart_service.go`, update `HEARTService` struct:

```go
type HEARTService struct {
    gpt       *GPT                   // Gorgonite transformer (always present)
    bridge    *CerebrasBridge        // Optional: Cerebras WSE2 acceleration
    processor *NetworkMetricsProcessor
    tokenizer *knirvtokenizer.Tokenizer
    embedder  *embeddings.DeterministicService
    hashNet   *RecursiveEngineWrapper // Optional: HashNetwork fast path (Phase 8)
    config    *HEARTConfig
    stats     *HEARTServiceStats
    mu        sync.RWMutex
}
```

### Step 2.2 — Add HEARTConfig

Create `pkg/hashing/transformer/config.go`:

```go
package transformer

import "knirvhasher/pkg/hashing/factory"

type HEARTConfig struct {
    // Model config
    Gorgonite GorgoniteConfig

    // Optional paths
    UseHashNetwork bool   // Enable HashNetwork fast-path (Option 3)
    UseCerebras    bool   // Enable CerebrasBridge bulk path
    CerebrasProgramDir  string
    CerebrasWeightsPath string

    // WASM compilation
    TinyGoPath   string // path to tinygo binary
    WASMOutDir   string // directory for compiled .wasm files
    AuditLogDir  string // per-client audit log root

    // Thresholds
    HashNetworkConfidenceThreshold float32 // default 0.85
    EntropySpikethreshold          float64 // default 3.0 nats
    MaxTurns                       int     // multi-turn loop limit, default 3
}

func DefaultHEARTConfig(useHashNetwork, useCerebras bool) *HEARTConfig {
    return &HEARTConfig{
        Gorgonite:                      DefaultGorgoniteConfig(),
        UseHashNetwork:                 useHashNetwork,
        UseCerebras:                    useCerebras,
        TinyGoPath:                     "tinygo",
        WASMOutDir:                     "/var/heart/wasm",
        AuditLogDir:                    "/var/heart/audits",
        HashNetworkConfidenceThreshold: 0.85,
        EntropySpikethreshold:          3.0,
        MaxTurns:                       3,
    }
}
```

### Step 2.3 — NewHEARTService constructor

Replace the current ad-hoc construction in `heart_service.go` with:

```go
func NewHEARTService(cfg *HEARTConfig, hashCfg *factory.HashMethodConfig) (*HEARTService, error) {
    tok, err := knirvtokenizer.New()
    if err != nil {
        return nil, fmt.Errorf("tokenizer: %w", err)
    }

    g := gorgonia.NewGraph()
    gpt := NewGPT(g, &cfg.Gorgonite)

    svc := &HEARTService{
        gpt:       gpt,
        tokenizer: tok,
        embedder:  embeddings.NewDeterministicService(),
        config:    cfg,
        stats:     &HEARTServiceStats{ErrorTypeCounts: make(map[string]uint64), HeuristicUsage: make(map[string]uint64)},
    }

    if cfg.UseCerebras {
        svc.bridge = NewCerebrasBridge(cfg.CerebrasProgramDir, cfg.CerebrasWeightsPath, false)
    }

    if cfg.UseHashNetwork {
        // Phase 8: wire RecursiveEngine
        svc.hashNet = newRecursiveEngineWrapper(hashCfg)
    }

    return svc, nil
}
```

### Step 2.4 — Wire GPT into processInquiry

Replace the hardcoded heuristics in `generateRecommendedActions()` with real Gorgonite
inference. The stub `switch errorType { ... default: return genericRecommendations }` is
replaced by:

```go
func (s *HEARTService) runGorgoniteInference(ctx context.Context, prompt string) (string, error) {
    tokens, err := s.tokenizer.Encode(prompt)
    if err != nil {
        return "", err
    }
    // Truncate to ContextLen
    if len(tokens) > s.config.Gorgonite.ContextLen {
        tokens = tokens[len(tokens)-s.config.Gorgonite.ContextLen:]
    }

    // If Cerebras is available, use it; otherwise use Gorgonia TapeMachine
    if s.bridge != nil {
        // delegate to CerebrasBridge.RunInference (existing method)
        return s.runCerebrasInference(ctx, tokens)
    }
    return s.runGorgoriaInference(tokens)
}
```

**Verification**: `go build ./cmd/heart/` — HEARTService starts, responds to `/heart/health`

---

## Phase 3 — HEARTService Endpoint & Inquiry Refactor

**Addresses**: Option 19 (Stage 1 Action Inquiry variants), Operating Objective (WASM type
decision), Policy Badge → Action Inquiry Pipeline

### Step 3.1 — Define new inquiry and response types

Create `pkg/hashing/transformer/types.go`:

```go
package transformer

// WASMType identifies which binary class to produce
type WASMType string
const (
    WASMTypeRule       WASMType = "rule"
    WASMTypeResolution WASMType = "resolution"
    WASMTypePatch      WASMType = "patch"
)

// PolicyBadgeInquiry is emitted by the Badge Lab (badge-lab-panel.tsx)
// POST /api/knirvshell/chain/badge/create → HEARTService /heart/advise
type PolicyBadgeInquiry struct {
    Name            string   `json:"name"`
    BadgeType       string   `json:"badge_type"`
    ValuesSignals   []string `json:"values_signals"`   // Guidelines|Customs|Etiquette|...
    OntologySignals []string `json:"ontology_signals"` // Rules|Policies|User Data|...
    DVEContext      string   `json:"dve_context"`      // active DVE session ID
}

// DVEErrorInquiry is emitted when a live DVE surfaces an error
type DVEErrorInquiry struct {
    DVESessionID string            `json:"dve_session_id"` // required; no DVE = rejected
    ErrorType    string            `json:"error_type"`
    ErrorMessage string            `json:"error_message"`
    ErrorContext string            `json:"error_context"`
    StackTrace   string            `json:"stack_trace,omitempty"`
    Metadata     map[string]string `json:"metadata,omitempty"`
}

// SystemPatchInquiry is for internal errors with no DVE origin
type SystemPatchInquiry struct {
    ComponentID string `json:"component_id"`
    ErrorCode   string `json:"error_code"`
    SystemState string `json:"system_state"`
}

// WASMDecision is the unified response for all three paths
type WASMDecision struct {
    WASMType            WASMType `json:"wasm_type"`
    WASMPath            string   `json:"wasm_path"`    // path to compiled binary
    WASMHash            string   `json:"wasm_hash"`    // SHA-256 hex of binary
    Rationale           string   `json:"rationale"`    // Stage 3 sketch (human-readable)
    BidirectionalVerified bool   `json:"bidirectional_verified"`
    WazeroExecPassed    bool     `json:"wazero_exec_passed"`
    HashNetworkVerified bool     `json:"hash_network_verified,omitempty"`
    VerifierConfidence  float32  `json:"verifier_confidence,omitempty"`
    TurnCount           int      `json:"turn_count"`
    ForwardVerifierMsg  string   `json:"forward_verifier_msg,omitempty"`
    BackwardVerifierMsg string   `json:"backward_verifier_msg,omitempty"`
}
```

### Step 3.2 — Add new HTTP endpoints

In `heart_service.go`, register:

```go
func (s *HEARTService) ListenAndServe(addr string) error {
    mux := http.NewServeMux()
    mux.HandleFunc("/heart/advise",  s.handleAdvise)   // rule.wasm   (primary)
    mux.HandleFunc("/heart/resolve", s.handleResolve)  // resolution.wasm
    mux.HandleFunc("/heart/patch",   s.handlePatch)    // patch.wasm
    mux.HandleFunc("/heart/health",  s.handleHealth)
    mux.HandleFunc("/heart/stats",   s.handleStats)
    return http.ListenAndServe(addr, mux)
}
```

### Step 3.3 — Implement classifyInquiry

```go
// classifyInquiry determines WASMType from request path + body
func (s *HEARTService) classifyInquiry(r *http.Request) WASMType {
    switch r.URL.Path {
    case "/heart/advise":
        return WASMTypeRule
    case "/heart/resolve":
        return WASMTypeResolution
    case "/heart/patch":
        return WASMTypePatch
    }
    return WASMTypeRule // default
}
```

For `/heart/resolve`, validate that `DVESessionID` is present and non-empty before processing:

```go
func (s *HEARTService) handleResolve(w http.ResponseWriter, r *http.Request) {
    var inq DVEErrorInquiry
    if err := json.NewDecoder(r.Body).Decode(&inq); err != nil {
        http.Error(w, "bad request", http.StatusBadRequest)
        return
    }
    if inq.DVESessionID == "" {
        http.Error(w, "resolution requires active DVE session", http.StatusUnprocessableEntity)
        return
    }
    decision, err := s.process(r.Context(), WASMTypeResolution, inq)
    // ...
}
```

---

## Phase 4 — Multi-Source Data Ingestion

**Addresses**: Option 1 (Multi-Source Data Feed), KNIRVHASHER Pipeline Stages table

**Files to modify**:
- `pipeline/1_DATA_MINER/internal/app/paper_manager.go` → extend to `source_manager.go`
- `pipeline/1_DATA_MINER/internal/app/orchestrator.go` → add worker coordination

### Step 4.1 — Define unified SourceRecord

Create `pipeline/1_DATA_MINER/internal/app/source_record.go`:

```go
package app

type SourceType string
const (
    SourceClientData  SourceType = "client_data"
    SourceArXiv       SourceType = "arxiv"
    SourceDVETelemetry SourceType = "dve_telemetry"
)

type SourceRecord struct {
    SourceType    SourceType
    DocumentID    string
    ChunkText     string
    PolicyClass   string    // for client data: Permit|Block|Escalate label if known
    EffectiveDate string    // RFC3339; client data temporal validity
    DVEContextID  string    // for DVE telemetry records
    ArXivCategory string    // for ArXiv records: cs.AI, cs.LG, cs.SE, cs.CR
}
```

### Step 4.2 — Implement source_manager.go

Create `pipeline/1_DATA_MINER/internal/app/source_manager.go` alongside `paper_manager.go`:

```go
package app

// SourceManager orchestrates all three data source workers
type SourceManager struct {
    out         chan SourceRecord
    clientCfg   ClientDataConfig   // S3/GDrive/SharePoint connection
    arxivCfg    ArXivConfig        // existing ArXiv config
    dveCfg      DVETelemetryConfig // DVE stream endpoint
}

func (sm *SourceManager) Run(ctx context.Context) {
    var wg sync.WaitGroup
    wg.Add(3)
    go func() { defer wg.Done(); sm.runClientDataWorker(ctx) }()
    go func() { defer wg.Done(); sm.runArXivWorker(ctx) }()  // reuse existing ArxivWorker
    go func() { defer wg.Done(); sm.runDVETelemetryWorker(ctx) }()
    wg.Wait()
    close(sm.out)
}
```

The `runArXivWorker` can directly reuse `internal/arxiv/arxiv_worker.go` — wrap it to emit
`SourceRecord` instead of the existing `PaperRecord`.

### Step 4.3 — UnifiedTrainingAdapter

Create `pipeline/1_DATA_MINER/internal/app/training_adapter.go`:

```go
func (a *UnifiedTrainingAdapter) Route(rec SourceRecord) TrainingPair {
    switch rec.SourceType {
    case SourceClientData:
        return TrainingPair{InquiryType: "badge", ExpectedWASMClass: "rule", ...}
    case SourceArXiv:
        return TrainingPair{InquiryType: "dve_error", ExpectedWASMClass: "resolution", ...}
    case SourceDVETelemetry:
        if rec.DVEContextID != "" {
            return TrainingPair{InquiryType: "dve_error", ExpectedWASMClass: "resolution", ...}
        }
        return TrainingPair{InquiryType: "patch", ExpectedWASMClass: "patch", ...}
    }
    return TrainingPair{}
}
```

---

## Phase 5 — 4-Stage Hierarchical Decision Pipeline

**Addresses**: Option 19 (unified 4-stage pipeline for all WASM types)

### Step 5.1 — Create pipeline.go

Create `pkg/hashing/transformer/pipeline.go`:

```go
package transformer

// Stage1Result is the classified, validated inquiry
type Stage1Result struct {
    WASMType WASMType
    Raw      interface{} // PolicyBadgeInquiry | DVEErrorInquiry | SystemPatchInquiry
}

// Stage2Result contains core principles/techniques identified before generation
type Stage2Result struct {
    // rule.wasm path
    PolicyPrinciples []PolicyPrinciple
    // resolution.wasm path
    ErrorClass       ErrorClass
    CoreTechniques   []CoreTechnique
    // patch.wasm path
    PatchScope       PatchScope
    AffectedComponents []string
}

// Stage3Result is the human-readable decision sketch
type Stage3Result struct {
    Sketch    string
    Rationale string
    // rule.wasm: Permit|Block|Escalate pre-decision
    // resolution.wasm: pseudocode plan
    // patch.wasm: component impact map
}

// Stage4Result is the generated TinyGo source
type Stage4Result struct {
    Source   string // TinyGo source code
    WASMType WASMType
}

// runPipeline executes all four stages for the given inquiry
func (s *HEARTService) runPipeline(ctx context.Context, wasmType WASMType, inquiry interface{}, priorFailures []string) (*Stage4Result, *Stage3Result, error) {
    s1 := s.stage1(wasmType, inquiry)
    s2, err := s.stage2(ctx, s1, priorFailures)
    if err != nil {
        return nil, nil, err
    }
    s3, err := s.stage3(ctx, s1, s2)
    if err != nil {
        return nil, nil, err
    }
    s4, err := s.stage4(ctx, s1, s2, s3)
    if err != nil {
        return nil, s3, err
    }
    return s4, s3, nil
}
```

### Step 5.2 — Stage 2: IdentifyRelevantPolicies / IdentifyCoreErrorTechniques

Stage 2 performs an embedding-similarity lookup against the ingested corpus using
`DeterministicService.GetEmbedding()`:

```go
func (s *HEARTService) stage2(ctx context.Context, s1 Stage1Result, priorFailures []string) (*Stage2Result, error) {
    switch s1.WASMType {
    case WASMTypeRule:
        inq := s1.Raw.(PolicyBadgeInquiry)
        // Embed OntologySignals, query policy corpus
        principles, err := s.lookupPolicyPrinciples(inq.OntologySignals, inq.ValuesSignals)
        return &Stage2Result{PolicyPrinciples: principles}, err

    case WASMTypeResolution:
        inq := s1.Raw.(DVEErrorInquiry)
        // Classify ErrorClass, identify core techniques
        errorClass, techniques := s.classifyError(ctx, inq)
        return &Stage2Result{ErrorClass: errorClass, CoreTechniques: techniques}, nil

    case WASMTypePatch:
        inq := s1.Raw.(SystemPatchInquiry)
        scope, components := s.identifyPatchScope(inq)
        return &Stage2Result{PatchScope: scope, AffectedComponents: components}, nil
    }
    return nil, fmt.Errorf("unknown wasm type")
}
```

`lookupPolicyPrinciples` uses `s.embedder.GetBatchEmbeddings(ontologySignals)` and cosine
similarity against the pre-indexed policy corpus embeddings stored in memory at startup.

---

## Phase 6 — WASM Compilation Pipeline

**Addresses**: Option 12 (Multi-Path WASM Compilation), WASM Interface Contracts

### Step 6.1 — Create compiler.go

Create `pkg/hashing/transformer/compiler.go`:

```go
package transformer

import (
    "context"
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
)

// WASMCompiler compiles TinyGo source to a typed WASM binary
type WASMCompiler struct {
    tinyGoPath string
    outDir     string
}

func NewWASMCompiler(tinyGoPath, outDir string) *WASMCompiler {
    return &WASMCompiler{tinyGoPath: tinyGoPath, outDir: outDir}
}

type CompileResult struct {
    WASMPath string
    WASMHash string // SHA-256 hex
}

func (c *WASMCompiler) Compile(ctx context.Context, source string, wasmType WASMType) (*CompileResult, error) {
    // Write source to temp file
    srcFile, err := os.CreateTemp("", fmt.Sprintf("heart_%s_*.go", wasmType))
    if err != nil {
        return nil, err
    }
    defer os.Remove(srcFile.Name())
    if _, err := srcFile.WriteString(source); err != nil {
        return nil, err
    }
    srcFile.Close()

    // Output path
    outPath := filepath.Join(c.outDir, fmt.Sprintf("%s_%d.wasm", wasmType, timeNow()))

    // exec.Command pattern from CerebrasBridge.RunInference()
    cmd := exec.CommandContext(ctx, c.tinyGoPath,
        "build",
        "-o", outPath,
        "-target", "wasm",
        srcFile.Name(),
    )
    out, err := cmd.CombinedOutput()
    if err != nil {
        return nil, fmt.Errorf("tinygo compile: %w\nstderr: %s", err, string(out))
    }

    // Content-address
    data, _ := os.ReadFile(outPath)
    hash := sha256.Sum256(data)

    // Rename to content-addressed path
    hashHex := hex.EncodeToString(hash[:])
    finalPath := filepath.Join(c.outDir, hashHex+".wasm")
    os.Rename(outPath, finalPath)

    return &CompileResult{WASMPath: finalPath, WASMHash: hashHex}, nil
}
```

### Step 6.2 — TinyGo source template per WASM type

Create `pkg/hashing/transformer/templates.go` with per-type TinyGo source scaffolds:

```go
package transformer

const ruleWASMTemplate = `
package main

import "unsafe"

//export GuardrailClass
func GuardrailClass() uint32 { return {{.Class}} } // 0=Permit|1=Block|2=Escalate

//export Resolve
func Resolve(ctxPtr uintptr, ctxLen uint32) uint64 {
    // {{.GeneratedLogic}}
    return 0
}

func main() {}
`

const resolutionWASMTemplate = `
package main

//export ErrorClass
func ErrorClass() uint32 { return {{.Class}} } // 0=Construction|1=TheoremCall|2=Transformation

//export Resolve
func Resolve(ctxPtr uintptr, ctxLen uint32) uint64 {
    // {{.GeneratedLogic}}
    return 0
}

func main() {}
`

const patchWASMTemplate = `
package main

//export PatchScope
func PatchScope() uint32 { return {{.Class}} } // 0=Hot|1=Restart|2=Migration

//export Resolve
func Resolve(ctxPtr uintptr, ctxLen uint32) uint64 {
    // {{.GeneratedLogic}}
    return 0
}

func main() {}
`
```

Gorgonite's Stage 4 output fills `{{.GeneratedLogic}}` and `{{.Class}}` — the template
structure ensures every compiled binary exports both the class function and `Resolve`.

### Step 6.3 — wazero runtime execution gate

Add dependency: `go get github.com/tetratelabs/wazero@latest`

Create `pkg/hashing/transformer/wazero_gate.go`:

```go
package transformer

import (
    "context"
    "fmt"
    "os"
    "github.com/tetratelabs/wazero"
)

type WazeroGate struct{}

type WazeroResult struct {
    ClassValue uint32
    Passed     bool
    ErrorMsg   string
}

func (g *WazeroGate) Execute(ctx context.Context, wasmPath string, wasmType WASMType) (*WazeroResult, error) {
    data, err := os.ReadFile(wasmPath)
    if err != nil {
        return &WazeroResult{Passed: false, ErrorMsg: err.Error()}, nil
    }

    rt := wazero.NewRuntime(ctx)
    defer rt.Close(ctx)

    mod, err := rt.Instantiate(ctx, data)
    if err != nil {
        return &WazeroResult{Passed: false, ErrorMsg: fmt.Sprintf("wazero compile: %v", err)}, nil
    }

    // Call class export
    classExport := classExportName(wasmType) // "GuardrailClass"|"ErrorClass"|"PatchScope"
    classFn := mod.ExportedFunction(classExport)
    if classFn == nil {
        return &WazeroResult{Passed: false, ErrorMsg: classExport + " not exported"}, nil
    }
    res, err := classFn.Call(ctx)
    if err != nil {
        return &WazeroResult{Passed: false, ErrorMsg: fmt.Sprintf("%s call: %v", classExport, err)}, nil
    }

    // Call Resolve with empty context (smoke test)
    resolveFn := mod.ExportedFunction("Resolve")
    if resolveFn == nil {
        return &WazeroResult{Passed: false, ErrorMsg: "Resolve not exported"}, nil
    }
    if _, err = resolveFn.Call(ctx, 0, 0); err != nil {
        return &WazeroResult{Passed: false, ErrorMsg: fmt.Sprintf("Resolve smoke test: %v", err)}, nil
    }

    return &WazeroResult{ClassValue: uint32(res[0]), Passed: true}, nil
}

func classExportName(t WASMType) string {
    switch t {
    case WASMTypeRule:       return "GuardrailClass"
    case WASMTypeResolution: return "ErrorClass"
    case WASMTypePatch:      return "PatchScope"
    }
    return "GuardrailClass"
}
```

---

## Phase 7 — Bidirectional Verification

**Addresses**: Option 21 (Bidirectional WASM Verification)

### Step 7.1 — Create verifier.go

Create `pkg/hashing/transformer/verifier.go`:

```go
package transformer

import (
    "context"
    "encoding/json"
    "fmt"
    "os/exec"
)

type VerifierAgent struct {
    pythonPath   string
    verifierScript string // path to verifier.py
}

type VerifyResult struct {
    Passed bool
    Msg    string
}

// ForwardVerify: premises → conclusion (sufficiency)
func (v *VerifierAgent) ForwardVerify(ctx context.Context, inquiry interface{}, wasmPath string) VerifyResult {
    return v.runPython(ctx, "forward", inquiry, wasmPath)
}

// BackwardVerify: conclusion → premises (necessity)
func (v *VerifierAgent) BackwardVerify(ctx context.Context, inquiry interface{}, wasmPath string) VerifyResult {
    return v.runPython(ctx, "backward", inquiry, wasmPath)
}

// runPython follows the exec.Command pattern of CerebrasBridge.RunInference()
func (v *VerifierAgent) runPython(ctx context.Context, direction string, inquiry interface{}, wasmPath string) VerifyResult {
    payload, _ := json.Marshal(map[string]interface{}{
        "direction": direction,
        "inquiry":   inquiry,
        "wasm_path": wasmPath,
    })

    cmd := exec.CommandContext(ctx, v.pythonPath, v.verifierScript)
    cmd.Stdin = bytes.NewReader(payload)
    out, err := cmd.Output()
    if err != nil {
        return VerifyResult{Passed: false, Msg: fmt.Sprintf("verifier process: %v", err)}
    }

    var result struct {
        Passed bool   `json:"passed"`
        Msg    string `json:"msg"`
    }
    if err := json.Unmarshal(out, &result); err != nil {
        return VerifyResult{Passed: false, Msg: "verifier response parse failed"}
    }
    return VerifyResult{Passed: result.Passed, Msg: result.Msg}
}
```

Create stub `pkg/hashing/transformer/verifier.py` (to be fleshed out per policy domain):

```python
#!/usr/bin/env python3
import json, sys

payload = json.load(sys.stdin)
direction = payload["direction"]
wasm_path = payload["wasm_path"]

# TODO: implement domain-specific forward/backward checks
# For now: pass if wasm_path exists
import os
passed = os.path.exists(wasm_path)
print(json.dumps({"passed": passed, "msg": "stub verifier"}))
```

### Step 7.2 — Multi-turn cognitive loop

In `pkg/hashing/transformer/pipeline.go`, add the loop wrapper:

```go
func (s *HEARTService) process(ctx context.Context, wasmType WASMType, inquiry interface{}) (*WASMDecision, error) {
    var priorFailures []string
    var lastS3 *Stage3Result

    for turn := 0; turn < s.config.MaxTurns; turn++ {
        s4, s3, err := s.runPipeline(ctx, wasmType, inquiry, priorFailures)
        lastS3 = s3
        if err != nil {
            priorFailures = append(priorFailures, fmt.Sprintf("pipeline error: %v", err))
            continue
        }

        // Compile
        compiled, err := s.compiler.Compile(ctx, s4.Source, wasmType)
        if err != nil {
            priorFailures = append(priorFailures, fmt.Sprintf("compile error: %v", err))
            continue // retry Stage 4
        }

        // wazero gate
        wazeroRes, _ := s.wazeroGate.Execute(ctx, compiled.WASMPath, wasmType)
        if !wazeroRes.Passed {
            priorFailures = append(priorFailures, fmt.Sprintf("wazero: %s", wazeroRes.ErrorMsg))
            continue
        }

        // Bidirectional verification
        fwd := s.verifier.ForwardVerify(ctx, inquiry, compiled.WASMPath)
        bwd := s.verifier.BackwardVerify(ctx, inquiry, compiled.WASMPath)

        if fwd.Passed && bwd.Passed {
            s.writeAuditLog(inquiry, compiled, wasmType, turn+1)
            return &WASMDecision{
                WASMType: wasmType, WASMPath: compiled.WASMPath, WASMHash: compiled.WASMHash,
                Rationale: lastS3.Rationale, BidirectionalVerified: true,
                WazeroExecPassed: true, TurnCount: turn + 1,
            }, nil
        }

        priorFailures = append(priorFailures, fwd.Msg, bwd.Msg)
    }

    // MaxTurns exhausted → escalate
    s.routeToGapQueue(inquiry, wasmType, priorFailures)
    return &WASMDecision{
        WASMType: wasmType, Rationale: lastS3.Rationale,
        BidirectionalVerified: false, TurnCount: s.config.MaxTurns,
    }, nil
}
```

---

## Phase 8 — HashNetwork Optional Fast-Path

**Addresses**: Option 3 (HashNetwork as Optional Fast-Path), Option 18 (Generator-Verifier
Duality)

**Prerequisite**: `HEARTConfig.UseHashNetwork = true`

### Step 8.1 — Create RecursiveEngineWrapper

Create `pkg/hashing/transformer/hashnet_wrapper.go`:

```go
package transformer

import (
    "knirvhasher/pkg/hashing/factory"
    "knirvhasher/pkg/hashing/inference"
    "knirvhasher/pkg/hashing/neural"
)

// RecursiveEngineWrapper wraps the existing RecursiveEngine for use inside HEARTService
type RecursiveEngineWrapper struct {
    engine *inference.RecursiveEngine
}

func newRecursiveEngineWrapper(cfg *factory.HashMethodConfig) *RecursiveEngineWrapper {
    f := factory.NewHashMethodFactory(cfg)
    hashMethod := f.GetBestMethod()
    network := neural.NewHashNetwork(3, 32) // 3 layers, 32-byte seeds
    engine, _ := inference.NewRecursiveEngineWithHashMethod(network, hashMethod, 21, 0.01, true)
    return &RecursiveEngineWrapper{engine: engine}
}

type HashNetResult struct {
    ClassDistribution [3]float32 // [Permit|Construction|Hot, Block|TheoremCall|Restart, Escalate|Transform|Migration]
    ConfidenceScore   float32
    ShortCircuit      bool // true if confidence >= threshold
}

func (w *RecursiveEngineWrapper) Classify(inputBytes []byte, threshold float32) *HashNetResult {
    result := w.engine.Execute(inputBytes)
    dist := [3]float32{}
    for i, v := range result.ClassDistribution {
        if i < 3 {
            dist[i] = float32(v)
        }
    }
    return &HashNetResult{
        ClassDistribution: dist,
        ConfidenceScore:   float32(result.ConfidenceScore),
        ShortCircuit:      float32(result.ConfidenceScore) >= threshold,
    }
}
```

### Step 8.2 — Wire into process() fast-path

In `pkg/hashing/transformer/pipeline.go`, prepend to `process()`:

```go
// HashNetwork fast path (optional)
if s.hashNet != nil {
    tokens, _ := s.tokenizer.Encode(inquiryText(inquiry))
    inputBytes := tokensToBytes(tokens)
    hnRes := s.hashNet.Classify(inputBytes, s.config.HashNetworkConfidenceThreshold)
    if hnRes.ShortCircuit {
        // Return decision directly without running Gorgonite
        decision := shortCircuitDecision(wasmType, hnRes)
        decision.HashNetworkVerified = true
        decision.VerifierConfidence = hnRes.ConfidenceScore
        s.writeAuditLog(inquiry, nil, wasmType, 0)
        return decision, nil
    }
    // Low confidence: pass ClassDistribution as context hint to Stage 2
    // (store in context.Context for stage2() to consume)
    ctx = contextWithHashNetHint(ctx, hnRes.ClassDistribution)
}
```

---

## Phase 9 — Softmax + Positional Encoding Fixes

**Addresses**: Options 9, 11 code gaps identified in System Fingerprint

### Step 9.1 — Re-enable softmax in SelfAttention.Forward

In `main.go` (now `package transformer`), locate `SelfAttention.Forward`. The softmax call
is commented out — uncomment it and verify the shape:

```go
// Before:
// scores, _ = gorgonia.SoftMax(scores) // COMMENTED OUT

// After:
scores, err = gorgonia.SoftMax(scores)
if err != nil {
    return nil, fmt.Errorf("softmax: %w", err)
}
```

This unblocks: attention mask application (Option 3), entropy calculation (Option 23), and
positional encoding (Step 9.2).

### Step 9.2 — Re-enable positional encoding

In `main.go`, locate `NewGPT()` around line 415 where `posEnc` addition is disabled:

```go
// Before:
// embedded, err = gorgonia.Add(embedded, posEncOutput) // DISABLED

// After:
embedded, err = gorgonia.Add(embedded, posEncOutput)
if err != nil {
    return nil, fmt.Errorf("positional encoding add: %w", err)
}
```

### Step 9.3 — Fix multi-head concat (replace sum with concat)

In `MultiHeadAttention.Forward`, the head outputs are summed:

```go
// Before: combine by summing (lossy)
combined = headOutputs[0]
for i := 1; i < numHeads; i++ {
    combined, _ = gorgonia.Add(combined, headOutputs[i])
}

// After: concatenate along last axis (correct)
combined, err = gorgonia.Concat(1, headOutputs...)
if err != nil {
    return nil, fmt.Errorf("head concat: %w", err)
}
```

---

## Phase 10 — Variance-Guided Head Pruning + DynamicGraph NAS

**Addresses**: Options 2 and 8

### Step 10.1 — Head variance probe

Create `pkg/hashing/transformer/pruner.go`:

```go
package transformer

import "knirvhasher/pipeline/2_DATA_ENCODER/pkg/analyzer"

// HeadPruner uses VarianceAnalyzer to identify and prune low-variance attention heads
type HeadPruner struct {
    analyzer  *analyzer.VarianceAnalyzer
    threshold float64
}

func (hp *HeadPruner) PruneAfterEpoch(model *GPT, probeTexts []string, embedder EmbeddingProvider) {
    // Run probes through each head, record output variance
    // Feed into analyzer.Sample() / Calculate()
    // Zero out heads below threshold
}
```

### Step 10.2 — DynamicGraph NAS hook

After each training epoch, `DynamicGraph.Forward()` log is available. Add variance feedback:

```go
// In training loop (Phase 12):
if epoch > 0 && epoch % nasInterval == 0 {
    variance := measureBlockVariance(model, probeInputs)
    adjustDepth(model.dynamicGraph, variance) // add or remove TransformerBlock
}
```

---

## Phase 11 — ES Updates to EvolutionaryHarness (HashNetwork Only)

**Addresses**: Options 13–17 (skipped when `UseHashNetwork = false`)

**Files to modify**:
- `pipeline/3_DATA_TRAINER/pkg/training/evolutionary.go`
- `pipeline/3_DATA_TRAINER/internal/evo_grpo/evo_grpo.go`

### Step 11.1 — Replace GA elitism with ES weighted update

In `evolutionary.go`, `SelectAndMutate()`:

```go
// Remove: hard sort + keep top 25% + randomly mutate

// Add: ES weighted perturbation sum
func (eh *EvolutionaryHarness) ESWeightedUpdate(seeds [][]byte, rewards []float32, baseSeed []byte, alpha float32) []byte {
    normalized := eh.CalculateBitMatchAdvantage(rewards) // z-score already implemented
    updated := make([]byte, len(baseSeed))
    copy(updated, baseSeed)

    for n, seed := range seeds {
        epsilon := seedDelta(seed, baseSeed)       // εₙ = seeds[n] - baseSeed
        weight := normalized[n]                    // Rₙ (z-scored)
        for i := range updated {
            delta := int(float32(epsilon[i]) * weight * alpha / float32(len(seeds)))
            updated[i] = clampByte(int(updated[i]) + delta)
        }
    }
    return updated
}
```

Change the reward signal source: replace Bitcoin mining nonce alignment with WASMClass
classification accuracy on `(concept_embedding, expectedWASMClass)` pairs from the combined
training corpus (passed in via `EvaluatePopulationBatch`).

### Step 11.2 — Mirrored sampling (Option 14)

In `EvaluatePopulationBatch`, change from N random perturbations to N/2 + N/2 mirrors:

```go
// Generate N/2 positive perturbations
for i := 0; i < N/2; i++ {
    eps[i] = gaussianBytePerturbation(baseSeed, sigma)
    eps[N/2+i] = mirrorPerturbation(baseSeed, eps[i]) // seed - (eps[i] - baseSeed)
}
```

### Step 11.3 — σ annealing (Option 15)

Replace `StaticMidstate bool` with a computed σ schedule:

```go
func (eh *EvolutionaryHarness) sigma() float64 {
    return eh.SigmaMax * math.Exp(-float64(eh.Epoch)/eh.Tau)
}
```

### Step 11.4 — TRPO trust region (Option 25)

After computing the ES update `Δseed`, add:

```go
budget := eh.BudgetMax * math.Exp(-float64(eh.Epoch)/eh.Tau)
d := hammingDistance(baseSeed, updated)
if float64(d) > budget {
    alpha *= float32(budget) / float32(d)
    updated = eh.ESWeightedUpdate(seeds, rewards, baseSeed, alpha)
}
```

### Step 11.5 — Complete EvoGRPO (Option 16)

In `evo_grpo.go`, replace stub fitness values with a call to `EvolutionaryHarness.EvaluatePopulationBatch`
using the `(concept, WASMClass)` pairs from the combined corpus as the evaluation dataset.

---

## Phase 12 — Curriculum Training on Combined Corpus

**Addresses**: Option 20

### Step 12.1 — Define curriculum stages in DATA_TRAINER config

In `pipeline/3_DATA_TRAINER/internal/config/types.go`, add:

```go
type CurriculumStage string
const (
    CurriculumApprentice  CurriculumStage = "apprentice"  // LR 3e-4
    CurriculumJourneyman  CurriculumStage = "journeyman"  // LR 1e-4
    CurriculumExpert      CurriculumStage = "expert"       // LR 3e-5
)

type CurriculumConfig struct {
    Stage           CurriculumStage
    LearningRate    float64
    AdvanceTrigger  string // "validation_loss_plateau"
}
```

### Step 12.2 — Stage-gated training data selection

In `trainer.go`:

- **Apprentice**: load raw `(InquiryType, SourceChunk, WASMClass)` pairs from the `SourceManager` channel
- **Journeyman**: load `(InquiryType, SourceChunk, Stage3Sketch, WASMClass)` — include Stage 3 sketch in target; model must generate sketch before class verdict
- **Expert**: load full 4-stage hierarchy from existing WASM decision audit logs (Option 6) — use prior verified decisions as training examples; target is Stage 4 TinyGo source

### Step 12.3 — Tie curriculum stage to σ annealing

The σ schedule from Phase 11 maps directly to curriculum stage:
- Apprentice epoch range → high σ (broad HashNetwork seed exploration)
- Expert epoch range → low σ (precision refinement)

Pass the curriculum epoch as the `Epoch` field to `EvolutionaryHarness.sigma()`.

---

## Phase 13 — Entropy-Spike Detection + Self-Improving Loop

**Addresses**: Options 23 (Entropy-Spike), 9 (Ontology Drift)

**Prerequisite**: Phase 9 (softmax re-enabled)

### Step 13.1 — Instrument per-token entropy

In `runGorgoriaInference()`, after each generation step:

```go
// After softmax output is computed:
probs := softmaxOutput // []float32 over vocabulary
entropy := 0.0
for _, p := range probs {
    if p > 0 {
        entropy -= float64(p) * math.Log(float64(p))
    }
}

if entropy > s.config.EntropySpikethreshold {
    s.handleEntropySpike(ctx, wasmType, currentContext, entropy)
}
```

### Step 13.2 — handleEntropySpike routing

```go
func (s *HEARTService) handleEntropySpike(ctx context.Context, wasmType WASMType, context []int, entropy float64) {
    // Optional: route to HashNetwork for next-token suggestion
    if s.hashNet != nil {
        inputBytes := tokensToBytes(context)
        hnRes := s.hashNet.Classify(inputBytes, s.config.HashNetworkConfidenceThreshold)
        if hnRes.ShortCircuit {
            // Use HashNetwork's class suggestion as next-token hint
            // (inject into generation context)
        }
    }

    // Route to appropriate queue by WASM type
    switch wasmType {
    case WASMTypeRule:
        s.gapQueue <- OntologyGap{Context: context, Entropy: entropy}
    case WASMTypeResolution:
        s.novelErrorQueue <- NovelError{Context: context, Entropy: entropy}
    case WASMTypePatch:
        s.alertQueue <- SystemAlert{Context: context, Entropy: entropy}
    }
}
```

### Step 13.3 — OntologyDrift detection (Option 9)

Extend `pipeline/2_DATA_ENCODER/pkg/analyzer/variance.go`:

```go
// DeltaSignalIndices computes Jaccard distance between two top-24 index sets
func DeltaSignalIndices(prev, curr []int) float32 {
    prevSet := make(map[int]bool)
    for _, i := range prev { prevSet[i] = true }
    intersect, union := 0, len(prevSet)
    for _, i := range curr {
        if prevSet[i] { intersect++ } else { union++ }
    }
    if union == 0 { return 0 }
    return 1.0 - float32(intersect)/float32(union) // Jaccard distance
}
```

Invoke after each `SourceManager` batch completes. Emit internal `OntologyDrift` event
to HEARTService when delta > 0.4.

---

## Audit Trail (Option 6)

Create `pkg/hashing/transformer/audit.go`:

```go
package transformer

import (
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "time"
)

type AuditRecord struct {
    InquiryHash         string   `json:"inquiry_hash"`
    WASMSha256          string   `json:"wasm_sha256"`
    WASMType            WASMType `json:"wasm_type"`
    SourceContextID     string   `json:"source_context_id"` // DVE session ID or badge chain ID
    ModelCheckpointHash string   `json:"model_checkpoint_hash"`
    Timestamp           string   `json:"timestamp"`
}

func (s *HEARTService) writeAuditLog(inquiry interface{}, compiled *CompileResult, wasmType WASMType, turns int) {
    inquiryJSON, _ := json.Marshal(inquiry)
    inquiryHash := sha256.Sum256(inquiryJSON)

    record := AuditRecord{
        InquiryHash:     hex.EncodeToString(inquiryHash[:]),
        WASMType:        wasmType,
        SourceContextID: sourceContextID(inquiry),
        Timestamp:       time.Now().UTC().Format(time.RFC3339),
    }
    if compiled != nil {
        record.WASMSha256 = compiled.WASMHash
    }

    data, _ := json.Marshal(record)
    hashHex := hex.EncodeToString(sha256.Sum256(data)[:])
    path := filepath.Join(s.config.AuditLogDir, hashHex+".json")
    os.MkdirAll(s.config.AuditLogDir, 0755)
    os.WriteFile(path, data, 0644)
}
```

---

## Implementation Sequence Summary

| Phase | Description | Key Files Created / Modified | Blocking Deps |
|-------|-------------|------------------------------|---------------|
| **0** | Package refactor: `package main` → `package transformer` | `transformer/{main,heart_service,dynamic_graph,cerebras_bridge}.go`; `cmd/heart/main.go` | None |
| **1** | Tokeniser + deterministic embedder | `go.mod`; imports in `transformer/` | Phase 0 |
| **2** | GPT wired into HEARTService | `transformer/config.go`, `HEARTService.gpt` field, `NewHEARTService()` | Phase 0, 1 |
| **3** | New endpoints + inquiry types | `transformer/types.go`, `/heart/advise|resolve|patch` | Phase 2 |
| **4** | Multi-source DATA_MINER | `source_manager.go`, `source_record.go`, `training_adapter.go` | Phase 0 |
| **5** | 4-Stage pipeline | `transformer/pipeline.go`, stage functions | Phase 2, 3 |
| **6** | WASM compilation + wazero | `transformer/compiler.go`, `wazero_gate.go`, `templates.go` | Phase 5 |
| **7** | Bidirectional verification + multi-turn loop | `transformer/verifier.go`, `verifier.py`, loop in `process()` | Phase 6 |
| **8** | HashNetwork fast-path (optional) | `transformer/hashnet_wrapper.go`, fast-path in `process()` | Phase 2, 3 |
| **9** | Softmax + positional encoding fixes | `transformer/main.go` line edits | Phase 0 |
| **10** | Head pruning + DynamicGraph NAS | `transformer/pruner.go` | Phase 1, 9 |
| **11** | ES weighted update + mirrored sampling + σ annealing + TRPO | `pipeline/3_DATA_TRAINER/pkg/training/evolutionary.go`, `evo_grpo.go` | Phase 4 |
| **12** | Curriculum training | `DATA_TRAINER` config + stage-gated loader | Phase 4, 11 |
| **13** | Entropy-spike + ontology drift | `transformer/main.go` generation loop; `analyzer/variance.go` | Phase 9 |
| **Audit** | Audit trail | `transformer/audit.go` | Phase 6 |

---

## Testing Checkpoints

After each phase, run:

```bash
# Phase 0
cd packages/KNIRVHASHER
go build ./pkg/hashing/transformer/...
go build ./cmd/heart/

# Phase 1
go test ./pipeline/2_DATA_ENCODER/pkg/tokenizer/...
go test ./pipeline/2_DATA_ENCODER/pkg/embeddings/...

# Phase 2-3
./cmd/heart/heart --addr :8090 &
curl -s http://localhost:8090/heart/health | jq .

# Phase 6
echo '{"name":"test badge","values_signals":["Rules"],"ontology_signals":["Policies"],"dve_context":"dve-001"}' \
  | curl -s -X POST http://localhost:8090/heart/advise -d @- | jq .wasm_type

# Phase 7
# Same request: check BidirectionalVerified=true in response JSON

# Phase 11
go test ./pipeline/3_DATA_TRAINER/pkg/training/...

# Full integration
make tests
```

---

## File Tree: New Files Created

```
packages/KNIRVHASHER/
  cmd/
    heart/
      main.go                          ← Phase 0: HEARTService entrypoint
  pkg/hashing/transformer/
    config.go                          ← Phase 2: HEARTConfig
    types.go                           ← Phase 3: inquiry + decision types
    pipeline.go                        ← Phase 5: 4-stage pipeline
    compiler.go                        ← Phase 6: WASMCompiler (tinygo)
    templates.go                       ← Phase 6: TinyGo source templates
    wazero_gate.go                     ← Phase 6: wazero runtime gate
    verifier.go                        ← Phase 7: bidirectional verifier (Go)
    verifier.py                        ← Phase 7: Python verifier subprocess
    hashnet_wrapper.go                 ← Phase 8: RecursiveEngineWrapper
    pruner.go                          ← Phase 10: HeadPruner
    audit.go                           ← Audit: AuditRecord writer
  pipeline/1_DATA_MINER/
    internal/app/
      source_record.go                 ← Phase 4: unified SourceRecord
      source_manager.go               ← Phase 4: multi-source orchestration
      training_adapter.go             ← Phase 4: UnifiedTrainingAdapter
```
