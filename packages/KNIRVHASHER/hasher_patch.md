# HasherTransformer + HashNetwork Unification Patch

**Goal:** Merge `HasherTransformer` and `HashNetwork` into a single seed-based neural system with hardware-accelerated inference and safe software fallbacks.

**Status:** Planning

---

## 1. Current State

| Component | File | Uses `hashMethod`? | Hardware Path |
|-----------|------|-------------------|---------------|
| `HasherTransformer` | `pkg/hashing/transformer/gpt.go:982` | Field exists, setter exists, **never read in Forward()** | None — all projections use `seedToFloat` |
| `HashNetwork` | `pkg/hashing/neural/network.go:13` | No field at all | `RecursiveEngine.runHardwareInference()` calls `hashMethod.ComputeBatch` |
| `RecursiveEngine` | `pkg/hashing/inference/recursive.go:20` | Yes, via `e.hashMethod` | Full 21-pass temporal loop with jitter + consensus |
| `HasherWrapper` | `pipeline/3_DATA_TRAINER/pkg/simulator/hasher_wrapper.go:19` | Yes, wraps `core.HashMethod` | Software/CUDA/ASIC/eBPF/uBPF auto-detect |
| `EvolutionaryHarness` | `pipeline/3_DATA_TRAINER/pkg/training/evolutionary.go` | Via `HasherWrapper` | `Execute21PassLoopBatch` for batch seed evaluation |

**Key Gap:** `HasherTransformer.Forward()` computes entirely in software despite having a `hashMethod` field. `HashNetwork` has no such field but its inference is routed through `RecursiveEngine` which does use hardware. These two systems share the same conceptual model (seed-based weights, hash-derived activations) but have no shared code path.

---

## 2. Unified Architecture

```
                    ┌──────────────────────────┐
                    │   UnifiedHasherEngine    │
                    │  (replaces both systems) │
                    └──────────┬───────────────┘
                               │
               ┌───────────────┼───────────────┐
               │               │               │
      ┌────────▼──────┐ ┌─────▼──────┐ ┌──────▼────────────┐
      │ Transformer   │ │ Recursive  │ │  FeedForward      │
      │ Mode          │ │ Mode       │ │  Mode             │
      │ (MHA + FFN)   │ │ (21-pass)  │ │  (simple 3-layer) │
      │ with HW accel │ │ temporal   │ │  with HW accel    │
      └───────┬──────┘ └─────┬──────┘ └──────┬────────────┘
              │               │               │
              └───────────────┼───────────────┘
                              │
                    ┌─────────▼─────────┐
                    │  HardwareRouter  │
                    │  ┌─────────────┐ │
                    │  │ ComputeBatch│ │ ← hashMethod.ComputeBatch
                    │  │ (ASIC/CUDA) │ │
                    │  └─────────────┘ │
                    │  ┌─────────────┐ │
                    │  │ seedToFloat │ │ ← Software fallback
                    │  │ (CPU)       │ │
                    │  └─────────────┘ │
                    └──────────────────┘
```

### 2.1 Design Principles

1. **One seed representation**: All weights stored as `[32]byte` seeds. No `float32` parameter tensors.
2. **Hardware-first, software-fallback**: Every projection attempts `hashMethod.ComputeBatch` first; falls back to `seedToFloat` if hardware is unavailable or errors.
3. **Mode polymorphism**: The same seed network can run in Transformer mode (multi-head attention + FFN layers) or Recursive mode (21-pass temporal ensemble with jitter + consensus). Switching modes does not require retraining.
4. **Train-once, run-anywhere**: Seeds trained by the evolutionary harness are valid for all modes. No mode-specific fine-tuning.

---

## 3. Proposed API

### 3.1 Core Struct

```go
// UnifiedHasherEngine replaces HasherTransformer, HashNetwork, and RecursiveEngine
// for seed-based inference with optional hardware acceleration.
type UnifiedHasherEngine struct {
    config   *UnifiedConfig
    seeds    *SeedStore            // All seed parameters in one place
    hashMethod core.HashMethod     // nil = software fallback
    mode     InferenceMode         // "transformer" | "recursive" | "feedforward"
    stats    *EngineStats
    mu       sync.RWMutex
}

type UnifiedConfig struct {
    VocabSize    int
    EmbedDim     int
    NumHeads     int
    NumLayers    int
    ContextLen   int
    Hidden1      int
    Hidden2      int
    OutputSize   int
    FFNHiddenDim int
    Activation   string // "hash" | "tanh" | "sigmoid"
    Passes       int    // for recursive mode (default 21)
    Jitter       float64
    SeedRotation bool
}

type InferenceMode string
const (
    ModeTransformer  InferenceMode = "transformer"
    ModeRecursive    InferenceMode = "recursive"
    ModeFeedforward  InferenceMode = "feedforward"
)
```

### 3.2 SeedStore

```go
type SeedStore struct {
    Embeddings [][][32]byte      // [vocabSize][embedDim][32]byte
    Positional [][][32]byte      // [contextLen][embedDim][32]byte
    Layers     []TransformerLayerSeeds
    OutputSeed [32]byte
    // Feedforward/Recursive compatibility
    Seeds1     [][32]byte
    Seeds2     [][32]byte
    SeedsOut   [][32]byte
}

type TransformerLayerSeeds struct {
    QuerySeeds  [][][32]byte
    KeySeeds    [][][32]byte
    ValueSeeds  [][][32]byte
    OutputSeeds [][][32]byte
    FFNSeeds    [][][32]byte
}
```

### 3.3 HardwareRouter

```go
type HardwareRouter struct {
    hashMethod core.HashMethod
    fallback   FallbackStrategy
    cache      *ProjectionCache
}

type FallbackStrategy int
const (
    FallbackSoftware FallbackStrategy = iota // seedToFloat on CPU
    FallbackError                             // return error if hardware fails
    FallbackMixed                              // use hardware where possible, software for edge cases
)

func (r *HardwareRouter) Project(input []float32, seeds [][32]byte) ([]float32, error)
func (r *HardwareRouter) ProjectBatch(inputs [][]float32, seeds [][][32]byte) ([][]float32, error)
func (r *HardwareRouter) HashToVocab(hidden []float32, outputSeed [32]byte, vocabSize int) []float32
```

---

## 4. Migration Steps

### Step 1: Extract Shared Seed Utilities

**New file:** `pkg/hashing/transformer/seed_utils.go`

Extract common seed-to-float and projection logic from both `gpt.go` and `neural/network.go` into shared functions:

```go
func SeedToFloat(seed [32]byte) float32
func ProjectSeeds(input []float32, seeds [][32]byte, activation string) []float32
func ProjectSeeds2D(input []float32, seeds [][][32]byte, activation string) []float32
func ProjectBack(input []float32, targetDim int, activation string) []float32
func HashToVocab(hidden []float32, outputSeed [32]byte, vocabSize int) []float32
func LayerNorm(x float32, min, max float32) float32
```

**Risk:** Low — pure extraction, no behavior change.

### Step 2: Implement HardwareRouter

**New file:** `pkg/hashing/transformer/hardware_router.go`

```go
type HardwareRouter struct {
    hashMethod core.HashMethod
    fallback   FallbackStrategy
}

func NewHardwareRouter(method core.HashMethod, strategy FallbackStrategy) *HardwareRouter

func (r *HardwareRouter) Project(input []float32, seeds [][32]byte) ([]float32, error) {
    if r.hashMethod != nil && r.hashMethod.IsAvailable() {
        // Prepare inputs: for each seed, concatenate input || seed
        inputs := make([][]byte, len(seeds))
        inputBytes := floatSliceToBytes(input)
        for i, seed := range seeds {
            combined := make([]byte, len(inputBytes)+32)
            copy(combined, inputBytes)
            copy(combined[len(inputBytes):], seed[:])
            inputs[i] = combined
        }
        hashes, err := r.hashMethod.ComputeBatch(inputs)
        if err == nil {
            return hashesToFloats(hashes), nil
        }
        if r.fallback == FallbackError {
            return nil, fmt.Errorf("hardware projection failed: %w", err)
        }
    }
    // Software fallback
    return ProjectSeeds(input, seeds, "hash"), nil
}
```

**Risk:** Medium — need to validate `ComputeBatch` output matches `seedToFloat` distribution within acceptable bounds.

### Step 3: Rewrite HasherTransformer.Forward() to Use HardwareRouter

**Modify:** `pkg/hashing/transformer/gpt.go`

Replace all `ht.seedToFloat(seed)` calls inside projection functions with `router.Project()`:

```go
func (ht *HasherTransformer) Forward(tokenIDs []int) []float32 {
    router := NewHardwareRouter(ht.hashMethod, FallbackMixed)
    hidden := make([][]float32, len(tokenIDs))
    for i, id := range tokenIDs {
        hidden[i] = ht.embedToken(id, i) // embedToken also uses router
    }
    for l := 0; l < ht.Config.NumLayers; l++ {
        hidden = ht.forwardHasherLayer(hidden, l, router)
    }
    return ht.averagePool(hidden)
}

func (ht *HasherTransformer) forwardHasherLayer(hidden [][]float32, layerIdx int, router *HardwareRouter) [][]float32 {
    // Attention via router
    attn := ht.hasherMultiHeadAttention(hidden, layerIdx, router)
    // ... residual + layernorm ...
    ffn := ht.hasherFFN(hidden, layerIdx, router)
    // ... residual + layernorm ...
    return hidden
}
```

**Risk:** High — changes core inference path. Must maintain backward compatibility with existing `seedToFloat` behavior when `hashMethod == nil`.

### Step 4: Add Transformer Mode to RecursiveEngine

**Modify:** `pkg/hashing/inference/recursive.go`

Add `ModeTransformer` as a valid inference mode in `RecursiveEngine`:

```go
func (e *RecursiveEngine) Infer(input []byte) (*RecursiveResult, error) {
    switch e.Mode {
    case ModeTransformer:
        return e.inferTransformer(input)
    case ModeRecursive:
        return e.inferRecursive(input)
    case ModeFeedforward:
        return e.inferFeedforward(input)
    }
}

func (e *RecursiveEngine) inferTransformer(input []byte) (*RecursiveResult, error) {
    // Convert input to token IDs
    tokenIDs := e.tokenizer.Encode(string(input))
    // Run UnifiedHasherEngine in transformer mode
    engine := NewUnifiedHasherEngine(e.seeds, e.hashMethod, ModeTransformer)
    hidden := engine.Forward(tokenIDs)
    logits := engine.HashToVocab(hidden)
    // Return as single-pass consensus
    return &RecursiveResult{...}, nil
}
```

**Risk:** Medium — bridges two different input domains (raw bytes vs token IDs).

### Step 5: Unify SeedStore Between Pipeline and Inference

**Modify:** `pipeline/3_DATA_TRAINER/pkg/training/evolutionary.go`

Currently, training produces `SeedPopulation` with `map[uint32][]byte`. After unification:

1. `SeedPopulation.Seeds` maps to `UnifiedHasherEngine.Seeds`
2. Training harness writes `SeedStore` directly (not just CSV)
3. `FlashManager.DeployWeights()` writes `SeedStore` to BPF/device

**New interface:**

```go
type SeedStoreWriter interface {
    WriteSeedStore(store *SeedStore) error
    ReadSeedStore() (*SeedStore, error)
}
```

Implementations:
- `CSVSeedStoreWriter` (existing, backward compat)
- `BPFSeedStoreWriter` (new, for device deployment)
- `NRVSeedStoreWriter` (new, for .nrv bracket embedding)

**Risk:** Medium — changes training output format. Need migration path for existing CSV weights.

### Step 6: Wire UnifiedHasherEngine into HEART Service

**Modify:** `pkg/hashing/transformer/heart_service.go`

Replace dual `gpt` + `generatorSwitcher` with single `unifiedEngine`:

```go
type HEARTService struct {
    unifiedEngine *UnifiedHasherEngine
    // ... other fields unchanged
}

func NewHEARTServiceWithConfig(cfg *HEARTConfig) (*HEARTService, error) {
    engine, err := NewUnifiedHasherEngineFromConfig(cfg)
    if err != nil {
        return nil, err
    }
    return &HEARTService{
        unifiedEngine: engine,
        // ...
    }, nil
}
```

`runGorgoniteInference()` becomes `runUnifiedInference()` which delegates to `unifiedEngine.Forward()` in transformer mode or `unifiedEngine.Infer()` in recursive mode depending on WASM type.

**Risk:** High — changes HEART service entry point. Must preserve existing HTTP handlers (`/heart/analyze`, `/heart/resolve`, `/heart/patch`).

### Step 7: Deprecation Plan

| Deprecated | Replacement | Removal Target |
|------------|-------------|----------------|
| `HasherTransformer.Forward()` | `UnifiedHasherEngine.Forward()` with `ModeTransformer` | v2.0 |
| `HasherTransformer.GenerateToken()` | `UnifiedHasherEngine.GenerateToken()` | v2.0 |
| `HashNetwork.Forward()` | `UnifiedHasherEngine.Forward()` with `ModeFeedforward` | v2.0 |
| `HashNetwork.Predict()` | `UnifiedHasherEngine.Predict()` | v2.0 |
| `RecursiveEngine` | `UnifiedHasherEngine.Infer()` with `ModeRecursive` | v2.0 |
| `HasherTrainer` | Pipeline `3_DATA_TRAINER` (already supersedes) | v2.0 |
| `seedToFloat()` (package-level) | `HardwareRouter.Project()` | v2.0 |

Old APIs delegate to new implementation during transition period:

```go
// gpt.go — backward compat wrapper
func (ht *HasherTransformer) Forward(tokenIDs []int) []float32 {
    engine, _ := NewUnifiedHasherEngineFromHasherTransformer(ht)
    return engine.Forward(tokenIDs)
}
```

---

## 5. Fallback Strategy

### 5.1 Hardware Availability Tiers

```
Tier 1: ASIC available
  → hashMethod.IsAvailable() == true, method is ASICMethod
  → All projections use ComputeBatch
  → 21-pass loop runs on ASIC
  → Expected: 10^9+ hashes/sec

Tier 2: CUDA available
  → hashMethod.IsAvailable() == true, method is CudaMethod
  → All projections use ComputeBatch
  → 21-pass loop runs on GPU
  → Expected: 10^6-10^7 hashes/sec

Tier 3: eBPF/uBPF available
  → hashMethod.IsAvailable() == true, method is EbpfMethod or UbpfMethod
  → Batch hashing via kernel/userspace eBPF VM
  → 21-pass loop runs on eBPF
  → Expected: 10^5-10^6 hashes/sec

Tier 4: Software fallback (always available)
  → hashMethod == nil or IsAvailable() == false
  → All projections use seedToFloat (CPU)
  → 21-pass loop runs in Go
  → Expected: 10^3-10^4 hashes/sec
```

### 5.2 Graceful Degradation Rules

1. **Per-projection fallback**: If `ComputeBatch` fails mid-forward-pass, fall back to `seedToFloat` for that projection only. Do not abort the entire inference.
2. **Per-pass fallback**: In recursive mode, if a single pass's hardware call fails, retry once with software. If software also fails, skip the pass and continue with remaining passes. Consensus aggregates from whatever passes succeeded (minimum 1 pass required).
3. **Startup validation**: At engine initialization, probe hardware availability and log the active tier. Never surprise-fallback silently during runtime.
4. **Cache successful projections**: `ProjectionCache` stores recent `(inputHash, seedHash) → float32[]` mappings. In software fallback mode, this compensates for lower throughput.

### 5.3 Fallback Configuration

```go
type FallbackConfig struct {
    Strategy         FallbackStrategy  // default: FallbackMixed
    MinPasses        int               // minimum valid passes in recursive mode (default: 1)
    MaxRetries       int               // retries per projection before software fallback (default: 1)
    EnableCache      bool              // projection result caching (default: true)
    CacheSize        int               // max cache entries (default: 10000)
    LogFallbacks     bool              // log every fallback event (default: true)
}
```

---

## 6. Seed Compatibility

### 6.1 Existing Seeds Are Valid

Current seeds from the evolutionary trainer are `[32]byte` arrays. The unified engine accepts the same format:

```go
// Existing training output
type SeedPopulation struct {
    Seeds map[uint32][]byte  // 32-byte seeds
}

// New unified format
type SeedStore struct {
    Seeds1 [][32]byte  // direct mapping from SeedPopulation.Seeds
    Seeds2 [][32]byte
    SeedsOut [][32]byte
}
```

No retraining required. Migration is a format conversion, not a retraining.

### 6.2 Cross-Mode Validation

After unification, validate that a seed trained on `HashNetwork` produces consistent predictions in `UnifiedHasherEngine` transformer mode:

```go
func ValidateSeedCrossMode(seed [32]byte, input []byte) error {
    // Run in feedforward mode (old HashNetwork path)
    legacyNet := NewHashNetwork(inputSize, hidden1, hidden2, outputSize)
    legacyPred, _ := legacyNet.Predict(input)

    // Run in unified engine, transformer mode
    engine := NewUnifiedHasherEngine(seeds, nil, ModeTransformer)
    unifiedPred, _ := engine.Predict(input)

    if legacyPred != unifiedPred {
        return fmt.Errorf("cross-mode mismatch: legacy=%d unified=%d", legacyPred, unifiedPred)
    }
    return nil
}
```

Tolerance: predictions must match exactly in software-fallback mode (deterministic). In hardware mode, allow ±1 class deviation due to floating-point differences between ASIC/CUDA and CPU hash outputs.

---

## 7. Testing Requirements

### 7.1 Unit Tests

| Test | Validates |
|------|-----------|
| `TestHardwareRouter_Project_SoftwareFallback` | seedToFloat path when hashMethod == nil |
| `TestHardwareRouter_Project_HardwareSuccess` | ComputeBatch path when hardware available |
| `TestHardwareRouter_Project_HardwareFailureFallsBack` | graceful degradation mid-forward-pass |
| `TestUnifiedEngine_TransformerMode_OutputMatchesHasherTransformer` | bit-exact match with old Forward() in software mode |
| `TestUnifiedEngine_RecursiveMode_OutputMatchesRecursiveEngine` | bit-exact match with old Infer() in software mode |
| `TestUnifiedEngine_FeedforwardMode_OutputMatchesHashNetwork` | bit-exact match with old Predict() in software mode |
| `TestUnifiedEngine_ModeSwitch_NoRetrainRequired` | same seeds, different modes, all produce valid outputs |
| `TestSeedStore_RoundTrip` | serialize → deserialize preserves all seeds |
| `TestProjectionCache_HitRate` | cache reduces redundant computations by >80% |

### 7.2 Integration Tests

| Test | Validates |
|------|-----------|
| `TestHardwareTierDetection` | correct tier selected at startup based on available hardware |
| `TestEndToEnd_TransformerToRecursive` | train seeds → run in transformer mode → run in recursive mode → consensus valid |
| `TestPipeline3_SeedsFeedUnifiedEngine` | CSV seeds from evolutionary trainer load into UnifiedHasherEngine without modification |
| `TestHEARTService_UnifiedEngine` | all HTTP endpoints return identical responses before/after unification |
| `Test21PassLoop_HardwareDegradation` | simulate hardware failure mid-loop → remaining passes produce valid consensus |

### 7.3 Benchmark Tests

| Benchmark | Target |
|-----------|--------|
| `BenchmarkUnifiedEngine_TransformerMode_Hardware` | >10x throughput vs software |
| `BenchmarkUnifiedEngine_TransformerMode_Software` | within 20% of current HasherTransformer.Forward() |
| `BenchmarkUnifiedEngine_RecursiveMode_Hardware` | >100x throughput vs current RecursiveEngine software path |
| `BenchmarkUnifiedEngine_RecursiveMode_Software` | within 20% of current RecursiveEngine.Infer() |
| `BenchmarkProjectionCache` | <1µs per cached projection lookup |

---

## 8. Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| `ComputeBatch` output distribution differs from `seedToFloat` | Model predictions change | Run cross-mode validation on 1000+ seeds before merging. If divergence >5%, normalize hardware outputs to match software distribution. |
| Performance regression in software fallback | Slower inference when hardware unavailable | Profile `HardwareRouter.Project()` overhead. Target <5% overhead vs direct `seedToFloat`. Use function inlining. |
| Breaking changes to HEART service | Production downtime | Keep old `HasherTransformer` as fallback behind feature flag. Switch via config: `"inference_mode": "unified" \| "legacy"`. |
| Seed serialization format change | Training pipeline breaks | Maintain CSV reader in parallel with new SeedStore. Migrate lazily on first load. |
| Thread safety on `hashMethod` | Race conditions in concurrent inference | Document that `hashMethod` must be set before any inference calls. Add `sync.Once` guard. |

---

## 9. Implementation Order

| Step | Description | Effort | Dependencies |
|------|-------------|--------|--------------|
| 1 | Extract `seed_utils.go` shared functions | Small | None |
| 2 | Implement `HardwareRouter` with fallback logic | Medium | Step 1 |
| 3 | Rewrite `HasherTransformer.Forward()` to use `HardwareRouter` | Medium | Step 2 |
| 4 | Add transformer mode to `RecursiveEngine` | Medium | Step 2 |
| 5 | Unify `SeedStore` between pipeline and inference | Medium | Step 1 |
| 6 | Wire `UnifiedHasherEngine` into HEART service | Large | Steps 3-5 |
| 7 | Deprecation wrappers + feature flag | Small | Steps 3-6 |
| 8 | Full test suite + benchmarks | Large | Steps 1-7 |

**Estimated total effort:** 6-8 engineering days.

---

## 10. Success Criteria

1. `HasherTransformer.Forward()` produces identical output to current implementation when `hashMethod == nil` (software-only mode).
2. `UnifiedHasherEngine` in `ModeRecursive` produces identical output to current `RecursiveEngine.Infer()` when `hashMethod == nil`.
3. All existing pipeline tests pass without modification.
4. HEART service HTTP endpoints return byte-identical JSON responses before/after unification.
5. Hardware tier detection at startup completes in <100ms.
6. Software fallback incurs <5% overhead vs direct `seedToFloat`.
7. No breaking changes to public API — old types delegate to new types via wrappers.

---

## 11. Open Questions

1. **Should `HasherTransformer.GorgoniteConfig` and `UnifiedConfig` be merged or kept separate?** Answer: merge. The parameters overlap enough that maintaining two configs is technical debt.

2. **Should the evolutionary trainer be updated to output `SeedStore` directly, or keep CSV and convert on load?** Answer: keep CSV output for backward compat, add optional `SeedStore` binary output for production use.

3. **Should `projectToVocab` use `hashMethod.ComputeBatch` when hardware is available?** Answer: Yes — this is the highest-impact single change. The vocab projection is the most expensive step in transformer inference, and it currently runs SHA-256 in software for every token.

4. **Should the 21-pass loop be configurable per-request, or fixed at engine creation?** Answer: fixed at engine creation. Changing pass count mid-inference breaks consensus semantics.

---

## 12. Pipeline Implementation Gaps

The unification patch addresses inference-layer fragmentation, but the pipeline itself has four structural gaps that block production data flow. These must be resolved in parallel with or immediately after the unification work.

### 12.1 Gap: Stage 0 DATA_CONNECTOR — HuggingFace Ingestor Missing

**Location:** `pipeline/0_DATA_CONNECTOR/`  
**Spec:** `pipeline/KNIRVHASHER_pipeline_upgrade.md` §2  
**Impact:** Cannot ingest external datasets. Pipeline is limited to local/arXiv sources only.

**What exists:** Directory structure and a generic cleaner (`internal/cleaner/cleaner.go`), but no HuggingFace connector.

**What is missing:**

| Component | Description |
|-----------|-------------|
| `connector/huggingface.go` | `HuggingFaceConnector` that authenticates via `HF_TOKEN`, streams Parquet shards from the HuggingFace CDN, and emits `RawRecord` structs |
| Column normalization | Rule-based assembly of `text`, `instruction+output`, `input+output`, and `messages` columns into canonical `RawRecord.Text` |
| Writer update | Output `RawRecord` as Parquet shards to `~/.local/share/knirvhasher/connector/` for Stage 1 consumption |
| bbolt checkpoint | Deduplication guard to avoid reprocessing rows across restarts |

**Blocking dependency:** Stage 1 miner expects Parquet input from `~/.local/share/knirvhasher/connector/*.parquet`. Without Stage 0, only the legacy `ai_knowledge_base.parquet` path works.

**Effort:** 2–3 days. Can be developed independently of the unification patch.

### 12.2 Gap: Stage 2 DATA_ENCODER — SemanticMapper Not Implemented

**Location:** `pipeline/2_DATA_ENCODER/`  
**Spec:** `pipeline/KNIRVHASHER_pipeline_upgrade.md` §4  
**Impact:** 12-slot bracket encoding uses "naive variance" dimension selection instead of principled LSH projections. This breaks compliance with the 21-pass temporal loop's Syntactic Steering requirement (Passes 8–14).

**What exists:** `pkg/mapper/mapper.go` implements a variance-based 12-slot mapper. `pkg/mapper/variance_mapper.go` provides fallback variance selection. The 80-byte bracket binary format and NRV writer are functional.

**What is missing:**

| Component | Description |
|-----------|-------------|
| `pkg/semantic/semantic_mapper.go` | `SemanticMapper` that projects 768-dim embeddings to 16-dim via seeded LSH matrix, then encodes to 32-byte `Projections` field (Slots 0–3) |
| `config/v2_schema.yaml` | Schema-driven bracket configuration declaring LSH matrix seed, normalization, and slot-to-bracket field mapping |
| NLP bridge integration | Slot 4–5 (POS, Tense, Dependency) must be populated from spaCy output in Stage 1. Currently `POSTag`, `Tense`, `DepHead` fields exist in `DocumentRecord` but are not guaranteed populated. |
| FrameTicker | 1-second goroutine that buffers brackets, computes I/P bracket type (full vs XOR-diff), maintains MemoryZone (Slots 6–8), and flushes to `NRVWriter` |

**Critical constraint:** The existing `pkg/mapper/mapper.go` must be preserved, not replaced. The LSH mapper is a parameterization of Slots 0–3 only. Removing the variance mapper destroys the 12-slot Bitmask Specification.

**Effort:** 3–4 days. Depends on Stage 1 NLP bridge output being reliable.

### 12.3 Gap: End-to-End Pipeline Wiring Fragmented

**Location:** Cross-pipeline  
**Impact:** Trained seeds from Stage 3 do not feed back into either inference system (`HasherTransformer` or `RecursiveEngine`). The evolutionary trainer writes CSV/BPF maps, but the inference services load seeds independently or not at all.

**What exists:**

| Stage | Current Output | Current Consumer |
|-------|---------------|------------------|
| 0_DATA_CONNECTOR | None (not implemented) | — |
| 1_DATA_MINER | `mined_records.parquet` | Stage 2 (manually invoked) |
| 2_DATA_ENCODER | `training_frames.parquet` | Stage 3 (manually invoked) |
| 3_DATA_TRAINER | CSV seeds + BPF map dummy | No automated handoff to inference |

**What is missing:**

1. **SeedStore handoff**: Stage 3 must write `SeedStore` (binary format) in addition to CSV. The `UnifiedHasherEngine` must be able to load `SeedStore` at startup without CSV parsing.

2. **Hot-reload or restart-load**: After Stage 3 completes, the inference service must pick up new seeds. Current design requires manual restart. Need either:
   - File-watch reload (`fsnotify` on `SeedStore` path), or
   - Explicit reload endpoint (`POST /heart/reload-seeds`)

3. **Pipeline orchestrator**: A single entry point that runs Stage 0 → 1 → 2 → 3 sequentially, validates output at each stage, and triggers inference reload. Currently each stage is run independently via CLI.

**Proposed wiring:**

```
Stage 0 (HuggingFace)
    │  Parquet shards
    ▼
Stage 1 (Embed + NLP)
    │  mined_records.parquet
    ▼
Stage 2 (SemanticMapper + FrameTicker)
    │  .nrv files with brackets
    ▼
Stage 3 (Evolutionary GRPO)
    │  .nrv files with GoldenSeed + ASICLoops patched
    │  SeedStore binary
    ▼
UnifiedHasherEngine
    │  loads SeedStore at startup or via reload endpoint
    ▼
HEART Service / RecursiveEngine / HasherTransformer
    │  all three modes use same SeedStore
    ▼
Inference output
```

**Effort:** 2–3 days. Primarily wiring and a small orchestrator binary.

### 12.4 Gap: Flash Deployment Is Dummy

**Location:** `pipeline/3_DATA_TRAINER/pkg/deployment/flash.go`  
**Impact:** Trained weights cannot be deployed to actual BPF maps or eBPF devices. The `FlashManager` operates entirely in memory.

**What exists:**

| Component | Current State |
|-----------|--------------|
| `FlashManager` | Struct and API fully implemented |
| `BPFDummyInterface` | In-memory `map[string][]byte` — no real BPF interaction |
| `DeployWeights()` | Calls `bpf.updateElement()` which writes to dummy map |
| `validateDeployment()` | Reads back from dummy map — always succeeds |
| `rollbackWeights()` | Restores from JSON backup into dummy map |

**What is missing:**

| Component | Description |
|-----------|-------------|
| Real BPF map writer | Replace `BPFDummyInterface` with actual `bpf` syscall or `libbpf` Go bindings to write to `/sys/fs/bpf/hasher_weights` |
| Device validation | Read back from real BPF map and compare byte-for-byte against expected seeds. Current validation is a no-op. |
| Rollback to previous BPF state | Before flashing, snapshot current BPF map contents. On failure, restore snapshot. Currently only restores from JSON backup, not from device. |
| eBPF program loader | `BPFDummyInterface` has no eBPF program attached. The 21-pass temporal loop logic that should run in the kernel is not present. |

**Risk assessment:** This is the lowest-priority gap. The evolutionary trainer and inference engine work correctly in software/CUDA without BPF deployment. BPF deployment is an optimization for edge-device inference latency, not a correctness requirement.

**Effort:** 3–5 days. Requires kernel/eBPF toolchain and test hardware. High risk of platform-specific failures.

**Recommended approach:** Keep `BPFDummyInterface` as the default for development and CI. Implement `BPFRealInterface` behind a feature flag (`DeviceType: "bpf_real"`). Validate parity between dummy and real paths before enabling in production.

---

## 13. Combined Implementation Roadmap

### Phase A: Unification (Weeks 1–2)

| Order | Step | Description |
|-------|------|-------------|
| A1 | Extract `seed_utils.go` | Shared seed-to-float and projection utilities |
| A2 | Implement `HardwareRouter` | Hardware-first, software-fallback projection layer |
| A3 | Rewrite `HasherTransformer.Forward()` | Use `HardwareRouter`, preserve software-only behavior when `hashMethod == nil` |
| A4 | Add transformer mode to `RecursiveEngine` | Bridge raw-bytes and token-ID domains |
| A5 | Wire `UnifiedHasherEngine` into HEART service | Replace dual `gpt` + `generatorSwitcher` with single engine |
| A6 | Backward-compat wrappers + feature flag | `"inference_mode": "unified" | "legacy"` |

**Gate:** All existing tests pass. HEART endpoints return byte-identical JSON.

### Phase B: Pipeline Completion (Weeks 3–4)

| Order | Step | Description |
|-------|------|-------------|
| B1 | Stage 0 HuggingFace connector | Parquet streaming, column normalization, bbolt checkpoint |
| B2 | Stage 2 SemanticMapper | LSH projection for Slots 0–3, preserve variance mapper for Slots 4–11 |
| B3 | Stage 2 FrameTicker | 1-second windows, I/P bracket distinction, MemoryZone XOR |
| B4 | NLP bridge validation | Ensure Stage 1 outputs `POSTag`, `Tense`, `DepHead` for Slot 4–5 |
| B5 | SeedStore handoff | Stage 3 writes binary `SeedStore`; inference service loads at startup |
| B6 | Pipeline orchestrator | Single binary running Stage 0→1→2→3 with validation gates |

**Gate:** End-to-end run on AlpacaDataCleaned produces valid `.nrv` file with non-zero frame count and 80-byte aligned brackets.

### Phase C: Hardware Deployment (Week 5+)

| Order | Step | Description |
|-------|------|-------------|
| C1 | BPFRealInterface | Replace dummy with real BPF map syscalls |
| C2 | eBPF program loader | Load 21-pass temporal loop as eBPF kernel program |
| C3 | Parity validation | `BPFRealInterface` produces identical bracket writes to `BPFDummyInterface` |
| C4 | Feature flag rollout | Enable `DeviceType: "bpf_real"` in staging, then production |

**Gate:** 1000-bracket parity test passes: real BPF path produces byte-identical output to dummy path.

---

## 14. Updated Risk Register

Combining unification risks (§8) and pipeline risks (§12):

| Risk | Impact | Mitigation | Phase |
|------|--------|------------|-------|
| `ComputeBatch` output diverges from `seedToFloat` | Predictions change | Cross-mode validation on 1000+ seeds; normalize if divergence >5% | A |
| Software fallback overhead >5% | Slow inference in degraded mode | Profile and inline `HardwareRouter`; cache projections | A |
| HEART service breakage during unification | Production downtime | Feature flag `"inference_mode"`; legacy wrappers always available | A |
| Seed serialization migration breaks trainer | Pipeline fails to load existing CSV | Lazy CSV→SeedStore conversion; keep both formats during transition | A |
| Stage 2 removes variance mapper accidentally | 21-pass loop loses Syntactic Steering | Code review gate: `mapper.go` must remain importable; add `TestSlotPreservation` | B |
| NLP bridge missing Slot 4–5 data | Passes 8–14 collapse to random hashing | Validation gate in Stage 2: reject brackets with zeroed Slots 4–5 | B |
| BPF real interface platform-specific | CI fails on non-BPF hosts | Keep dummy as default; real interface behind `DeviceType` flag only | C |
| End-to-end wiring has no orchestrator | Manual stage execution error-prone | Build `cmd/pipeline-orchestrator/main.go` with `--stage` flags and `--validate` gates | B |

---

## 15. Summary

The KNIRVHASHER system has three independent fragmentation problems:

1. **Inference fragmentation**: `HasherTransformer` and `HashNetwork`/`RecursiveEngine` share the same seed-based model but have no unified code path. The `HasherTransformer` has a `hashMethod` field that is never used. The `HashNetwork` has no such field but routes through `RecursiveEngine` which does use hardware. Unification via `UnifiedHasherEngine` + `HardwareRouter` resolves this with safe software fallback at every level.

2. **Pipeline fragmentation**: Stages 0 and 2 are not implemented per the upgrade spec. Stage 3 output does not feed back into inference. The pipeline cannot ingest HuggingFace datasets, cannot produce compliant `.nrv` brackets with LSH projections, and has no orchestrator for end-to-end execution.

3. **Deployment fragmentation**: Flash deployment writes to an in-memory dummy interface instead of real BPF maps. This is acceptable for development but blocks edge-device deployment.

These three gaps are addressed in three sequential phases: unification first, pipeline completion second, hardware deployment third. Each phase has explicit validation gates and rollback paths via feature flags and backward-compat wrappers.
