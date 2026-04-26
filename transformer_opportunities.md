# HEART × Gorgonite: Unified Ontological Intelligence System

> Holistic architectural strategy converging `packages/KNIRVHEART/HEART/go_transformer`
> (Gorgonite GPT / HEARTService) and `packages/KNIRVHASHER/pipeline` into a unified
> intelligence fabric with two co-equal operational paths:
>
> **Primary — Guardrail Enforcement**: Policy Badges crafted in the Badge Lab carry
> `values` + `ontology` signals that initialise the 4-Stage Hierarchical Decision Pipeline.
> HEART produces a compiled `rule.wasm` applied to the I/O interface of the DVE that
> inspired the badge.
>
> **Secondary — Error Resolution + System Patching**: Errors discovered in an active DVE
> trigger HEART to produce `resolution.wasm` (applied back to the originating DVE's
> execution context). No resolution path activates without a live DVE as its origin.
> Internal system errors produce `patch.wasm` (system-wide). 
>
> HashNetwork is an optional inference accelerator — enabled per-client via
> `HEARTConfig.UseHashNetwork bool`, following the same "if available, use it" pattern
> already established by `CerebrasBridge`. CerebrasBridge itself remains optional for
> bulk high-throughput inference.
>
> All embedding operations use the custom fully deterministic text embedder
> (`github.com/guiperry/text-embedder/pkg/embed` → `embed.Embed(text) []float32`).
> No stochastic or model-served embeddings anywhere in the pipeline.
>
> Informed by: codebase analysis + `packages/KNIRVHASHER/ES_Relativity_Report.md`
> (Evolution Strategies at Scale, arXiv 2509.24372v2) + `packages/KNIRVHASHER/Cognition_Implementation.md`
> (Deep Insight Theorem Proving + Agent RL with Bidirectional Verifier, Apr 17 2026,
> City University of Hong Kong / Tsinghua / Fudan / ByteDance).

---

## System Fingerprint

### go_transformer — Gorgonite / HEARTService

| Component | Description |
|-----------|-------------|
| **GPT model** | Multi-layer transformer in pure Go via Gorgonia (static `ExprGraph` + `TapeMachine`) |
| **DynamicGraph** | Re-materialises a fresh static graph each forward pass — zero-cost depth changes |
| **CerebrasBridge** | *Optional*: exports Gorgonia `float32` weights to NPZ, shells out to `cs_python` for Cerebras WSE2 bulk inference |
| **HEARTService** | HTTP server (`/heart/advise`, `/heart/resolve`, `/heart/patch`, `/heart/health`, `/heart/stats`); receives `PolicyBadgeInquiry`, `DVEErrorInquiry`, or `SystemPatchInquiry`; returns the appropriate `WASMDecision` |
| **NetworkMetricsProcessor** | Adaptive Z-score normalisation of incoming context vectors → `HEARTInput` structs |

**Known gaps in current code:**
- Softmax commented out in `SelfAttention.Forward`
- Positional encoding disabled (line 415, `NewGPT`)
- Multi-head concat replaced with head sum (lossy)
- `BPETokenizer` is a complete stub — `loadVocab()` returns empty map, every token encodes as `<UNK>`
- No real dataset consumer — `CreateDummyDataset` is the only data source

---

### Custom Deterministic Text Embedder

**Package**: `github.com/guiperry/text-embedder/pkg/embed`
**API**: `embed.Embed(text string) []float32`
**Interface**: `DeterministicService` in `pipeline/2_DATA_ENCODER/pkg/embeddings/deterministic.go`

The deterministic embedder is the sole embedding provider across the entire system. It is
fully deterministic: identical input text always produces identical `[]float32` output with
no model serving, no network calls, no randomness. This replaces Ollama BGE-768 everywhere.

| Usage Site | Role |
|-----------|------|
| Stage 2 `DATA_ENCODER` | Primary text-to-vector conversion for all ingested documents |
| `HEARTService` | Policy Badge signal embedding for Stage 2 policy principle lookup |
| HashNetwork training (optional) | Concept embeddings for `(concept, guardrailClass)` pairs |
| Gorgonite fine-tuning | Training example embeddings for curriculum stages |
| `VarianceAnalyzer` | Input vectors for top-24 signal dimension selection |
| `findSimilarErrors()` | Embedding-based similarity search over past DVE errors |

Because `embed.Embed()` is deterministic, the `VarianceAnalyzer`'s `bge_signal_indices.json`
output is stable across restarts. Embeddings can be pre-computed, cached, and shared across
nodes without consistency risk.

---

### HashNetwork (Optional) — Fast-Path Guardrail Classifier

When enabled for a client, provides sub-millisecond first-pass verdicts. When disabled, all
inquiries route directly to Gorgonite.

| Component | Description |
|-----------|-------------|
| **HashNetwork** | 3-layer network; parameters are 32-byte seed arrays; forward pass: `SHA-256(input ∥ seed) → float [0, 1]` |
| **RecursiveEngine** | 21 temporal passes + majority vote → `ConsensusResult{ConfidenceScore, VoteCount, ClassDistribution}` |
| **EvolutionaryHarness** | Population-based seed training with z-score reward normalisation (`CalculateBitMatchAdvantage`); upgradeable to ES weighted update (Options 13–17) |
| **EvoGRPO** | Placeholder for ES+GRPO fusion; shares training corpus with Gorgonite when enabled |

**Activation**: `HEARTConfig.UseHashNetwork bool` — per-client. High-confidence verdicts
(≥ 0.85) short-circuit Gorgonite; lower-confidence verdicts pass `ClassDistribution` as a
soft conditioning hint.

---

### KNIRVHASHER Data Pipeline Stages

| Stage | Name | Multiple Source Roles |
|-------|------|-----------------------|
| 0 | `DATA_CONNECTOR` | Connects to all active data sources: client storage (S3, GDrive, SharePoint), ArXiv API (for ML/systems research), DVE telemetry streams (active error logs from live DVE sessions) |
| 1 | `DATA_MINER` | Per-source document parser: policy chunking for client data; XML/Atom parsing for ArXiv; structured error log extraction for DVE telemetry. All sources emit `chan SourceRecord` — a unified record type with `SourceType: ClientData|ArXiv|DVETelemetry` |
| 2 | `DATA_ENCODER` | tiktoken cl100k tokenisation; deterministic embeddings (`embed.Embed()`); `VarianceAnalyzer` (top-24 signal dimensions); `Mapper` (→ 12 `uint32` slots); `SlidingWindowGenerator` |
| 3 | `DATA_TRAINER` | Gorgonite fine-tuning on all source records; optional HashNetwork seed training on `(concept, WASMClass)` pairs derived from the combined corpus |

ArXiv ingestion specifically covers categories relevant to error resolution
(`cs.AI`, `cs.LG`, `cs.SE`, `cs.CR`) and is used as pre-training signal for the
`resolution.wasm` path. Client business data is the primary corpus for the `rule.wasm`
guardrail path.

---

## Operating Objective

Every client deployment provisions a dedicated HEART instance. The unified system handles
three execution paths. Guardrail enforcement is the primary operational focus.

```
┌──────────────────────────────────────────────────────────────────────────┐
│                         WASM Type Decision                               │
│                                                                          │
│  PolicyBadgeInquiry  ──────────────────────────► rule.wasm               │
│  (from Badge Lab, ontology signals attached)     applied to DVE I/O      │
│                                                  interface               │
│                                                                          │
│  DVEErrorInquiry  ─────────────────────────────► resolution.wasm         │
│  (error from active DVE only; no DVE = no path)  applied to originating  │
│                                                  DVE execution context   │
│                                                                          │
│  SystemPatchInquiry ───────────────────────────► patch.wasm              │
│  (internal system error, no DVE origin)           applied system-wide    │
└──────────────────────────────────────────────────────────────────────────┘
```

**DVE Scoping Rule**: A DVE (Deterministic Validation Environment) is an active containerized terminal session in
which the HERO agent operates. Resolution and guardrail outputs are scoped to the DVE that
originated them:
- `rule.wasm` is applied exclusively to the I/O interface of the DVE linked to the Policy
  Badge that created it. It cannot be applied cross-DVE without explicit badge re-issuance.
- `resolution.wasm` is applied to the same DVE's execution context that surfaced the error.
  If the DVE has terminated, the resolution is queued for the next session of the same type.
- `patch.wasm` has no DVE scope — it targets internal system components directly.

**WASM Interface Contracts (TinyGo `//export` directives)**:
```go
// rule.wasm
//export GuardrailClass  // returns 0=Permit | 1=Block | 2=Escalate
//export Resolve         // func Resolve(ctx GuardrailContext) GuardrailDecision

// resolution.wasm
//export ErrorClass      // returns 0=Construction | 1=TheoremCall | 2=Transformation
//export Resolve         // func Resolve(ctx ResolutionContext) ResolutionResult

// patch.wasm
//export PatchScope      // returns 0=Hot | 1=Restart | 2=Migration
//export Resolve         // func Resolve(ctx PatchContext) PatchResult
```

All three WASM types share the `Resolve` export name. HERO identifies the WASM type by
calling the class export first via `wazero` — `GuardrailClass()`, `ErrorClass()`, or
`PatchScope()` — before dispatching `Resolve()` with the appropriate context type.

---

## Policy Badge → Action Inquiry Pipeline

The Badge Lab (`packages/KNIRVSERVER/frontend/src/components/dashboard/badge-lab-panel.tsx`)
is the primary entry point for the `rule.wasm` guardrail path. Users craft Policy Badges
with two semantic signal dimensions:

**Values signals** (organisational/behavioural):
`Guidelines | Customs | Etiquette | Mission Statement | Stated Values | Goals & Objectives | Insights`

**Ontology signals** (data/knowledge domain):
`Trade Secrets | Business Logic | User Data | Rules | Regulations | Procedures | Policies | FAQs | Customer Service Bullets`

When a badge is submitted, the frontend POSTs to `/api/knirvshell/chain/badge/create` with
`metadata: {values: string[], ontology: string[]}`. This payload, along with the badge `name`
(free-text prompt) and `badge_type`, becomes the `PolicyBadgeInquiry` that initialises Stage 1
of the 4-Stage Hierarchical Decision Pipeline in HEARTService.

```
Badge Lab UI
  ├── prompt (free-text badge purpose)
  ├── selectedValues []string  ← Values signal dimension
  └── selectedOntology []string ← Ontology signal dimension
              │
              ▼ POST /api/knirvshell/chain/badge/create
  PolicyBadgeInquiry {
    Name:            badge.name,
    BadgeType:       "capability",
    ValuesSignals:   metadata.values,
    OntologySignals: metadata.ontology,
    DVEContext:      active_dve_id,   // scopes the resulting rule.wasm
  }
              │
              ▼ HEARTService /heart/advise
  4-Stage Hierarchical Decision Pipeline → rule.wasm
```

Stage 2's `IdentifyRelevantPolicies()` uses `OntologySignals` as the primary semantic query
against the client's ingested business data corpus. `ValuesSignals` condition Stage 3's
decision sketch — a badge tagged `[Rules, Regulations, Policies]` produces harder Block/Escalate
rules than one tagged `[Guidelines, Etiquette]`.

---

## Architectural Options

Options 1–12 address structural integration. Options 13–17 address the optional HashNetwork
ES training path. Options 18–25 apply generator-verifier duality, hierarchical reasoning,
curriculum learning, and bidirectional verification from the Cognition Implementation paper.

---

### Option 1 — Multi-Source Data Feed into HEART Training

**What**: Wire Stage 1's multi-source document parser directly into HEART's training loop as
a unified `chan SourceRecord` stream. Three concurrent sources feed the same training adapter:
client business data (for the `rule.wasm` path), ArXiv papers (for the `resolution.wasm`
path), and DVE telemetry (for both).

**How**:
1. Extend `paper_manager.go` → `source_manager.go`: orchestrates three concurrent goroutines —
   `ClientDataWorker`, `ArXivWorker`, `DVETelemetryWorker` — each emitting `SourceRecord{SourceType, ChunkText, PolicyClass, EffectiveDate, DVEContextID}` onto a shared buffered channel.
2. Add a `UnifiedTrainingAdapter` that routes `SourceRecord` by `SourceType`:
   - `ClientData` → `GuardrailInquiry` training pair with `ExpectedWASMClass: rule`
   - `ArXiv` → `DVEErrorInquiry` training pair with `ExpectedWASMClass: resolution`
   - `DVETelemetry` → routed by error origin: DVE-sourced → resolution, system-internal → patch
3. Feed the unified channel into `TrainModel()` replacing `CreateDummyDataset`.
4. ArXiv categories: `cs.AI`, `cs.LG`, `cs.SE`, `cs.CR` — maps to `ErrorType` in the resolution training records.

**Value Added**:
- Single training pipeline handles all three WASM generation paths — no separate training jobs.
- ArXiv pre-training is isolated to the `resolution.wasm` path and does not contaminate client-specific guardrail training.
- DVE telemetry provides live, production-quality error examples that generalise better than ArXiv-only data.
- `EffectiveDate` in `PolicyRecord` enforces temporal validity for client policies.

---

### Option 2 — Variance-Guided Attention Head Pruning for Domain Specificity

**What**: Run Stage 2's `VarianceAnalyzer` over HEART's attention weight distributions during
the initial client onboarding pass. Prune heads whose output variance falls below threshold.
Surviving heads specialise on the semantic dimensions carrying the most signal in this client's
domain.

**How**:
1. After the first fine-tuning epoch on client data, iterate over `block.attention.heads` and record output variance per head using deterministic embedding-based input probes.
2. Feed per-head variance vectors into `VarianceAnalyzer.Sample()` and call `Calculate()`.
3. Drop heads below threshold — set `wQuery/wKey/wValue` to zero tensors, skip in `Forward()`.
4. Reinitialise surviving heads with Glorot; resume fine-tuning.

**Value Added**:
- Client-specific models carry only the attention capacity relevant to their domain.
- `bge_signal_indices.json` persists the pruning decision per client — architecture is auditable.
- Smaller model → lower latency guardrail decisions for high-throughput HERO deployments.
- Pruning is re-run after each major client corpus update, keeping specialisation current.

---

### Option 3 — HashNetwork as Optional Fast-Path Guardrail Classifier

**What**: When HashNetwork is enabled, `RecursiveEngine`'s 21-pass consensus provides a
sub-millisecond first-pass verdict before Gorgonite's full inference. High-confidence verdicts
(≥ 0.85) short-circuit Gorgonite. Low-confidence verdicts pass `ClassDistribution` as a
soft hint. Disabled clients: complete no-op, identical control flow to `CerebrasBridge`.

**How**:
1. On any `HEARTInquiry`, tokenise the proposed action via cl100k (Option 5), map through `mapper.MapToSlots()` → 12-slot `[12]uint32` input.
2. Feed into `RecursiveEngine.Execute21PassLoop()`. `ConsensusResult.ClassDistribution[3]` maps to `{Permit|Construction|Hot, Block|TheoremCall|Restart, Escalate|Transformation|Migration}` — the class index meaning depends on the inquiry type (the same 3-class structure spans all three WASM types).
3. `ConfidenceScore ≥ 0.85` → return `WASMDecision` from HashNetwork directly; log `HashNetworkShortCircuit: true`.
4. `ConfidenceScore < 0.85` → pass `ClassDistribution` as attention conditioning prefix to Gorgonite.
5. `HEARTConfig.UseHashNetwork = false` → skip entirely.

**Value Added**:
- <1 ms classification for routine high-confidence guardrail checks.
- Clients with simple, rule-bound ontologies get effective rule-engine speed; nuanced clients rely on Gorgonite.
- The same 3-class structure applies to all three WASM output types — one HashNetwork serves all execution paths.

---

### Option 4 — Sliding Window Temporal Context for Sequential Analysis

**What**: Apply `SlidingWindowGenerator` to two temporal sequences:
(a) HERO's recent actions (for the guardrail path — detecting sequential policy violations);
(b) DVE error event history (for the resolution path — detecting cascading failure patterns).

**How**:
1. Maintain per-session buffers: `ActionHistoryBuffer[(clientID, sessionID)]` and `DVEErrorBuffer[(dveID, sessionID)]`.
2. On each inquiry, generate `SlidingWindow` structs over the appropriate flattened sequence. Each "token" is a quantised embedding index from `embed.Embed()`.
3. Embed via `EmbeddingLayer` and pass through transformer blocks with the existing `createCausalMask()`.

**Value Added**:
- Detects sequential policy violations invisible at the single-action level (e.g., individually permissible data accesses that collectively constitute a prohibited aggregation pattern).
- Detects cascading failure patterns in DVE error sequences before they compound — `resolution.wasm` generated early targets the root failure, not a symptom.
- `createCausalMask()` already in Gorgonite — direct API-level connection, no new infrastructure.

---

### Option 5 — cl100k Tokeniser Replacing the BPETokenizer Stub

**What**: Swap the non-functional `BPETokenizer` stub (`loadVocab()` returns empty map) for the
real tiktoken cl100k_base service already built in `2_DATA_ENCODER/pkg/tokenizer/tokenizer.go`.
Prerequisite for all three execution paths.

**How**:
1. Extract `2_DATA_ENCODER/pkg/tokenizer` into a shared module under `packages/KNIRVBASE/go/`.
2. Replace `NewBPETokenizer(vocabPath)` in `main.go` with `tokenizer.New()`.
3. Update `TransformerConfig.VocabSize` from 50257 to 100277 (cl100k actual vocab size).
4. Wire `PrepareDataset()` to call `tokenizer.Encode()`.

**Value Added**:
- Unblocks all three WASM generation paths simultaneously.
- cl100k is GPT-4/o-series compatible — deterministic embeddings and Gorgonite tokens share vocabulary, enabling hybrid retrieval-augmented reasoning.
- If HashNetwork is enabled, the same cl100k token IDs define the target classification space for seed training — one tokenisation scheme across all model paths.

---

### Option 6 — Cryptographic Audit Trail per WASM Output

**What**: At each WASM generation event, produce a cryptographic commitment: hash the inquiry +
the compiled WASM binary + the ontology version (for rules) or DVE session ID (for resolutions)
that produced it. Store in content-addressed audit storage.

**How**:
1. After `HEARTService` produces any `WASMDecision`, serialise `{inquiryHash, wasmSHA256, wasmType, sourceContextID, modelCheckpointHash, timestamp}`.
2. Compute SHA-256 over the serialisation via Go's `crypto/sha256`. Optional: hardware ASIC attestation if HashNetwork hardware path is enabled.
3. Append to per-client append-only audit log: `audits/<clientID>/<sha256hex>.json`.
4. `sourceContextID` is the DVE session ID for resolution/patch, or the Policy Badge chain ID for rule — enabling full traceability.

**Value Added**:
- Every HERO action traceable to the specific WASM binary, ontology version, DVE session, and model checkpoint that governed it.
- Rule rollback: if a Policy Badge produces a misbehaving `rule.wasm`, restore the previous badge version by its chain ID.
- Block decisions are the highest-priority audit entries — compliance teams can query all blocked HERO actions grouped by DVE, badge, and WASM type.

---

### Option 7 — Quantised Gorgonite for Edge DVE Deployment

**What**: Apply the mapper's `2×int16→uint32` bit-packing to Gorgonite model weights for
deployment in edge environments — embedded DVE nodes, air-gapped enterprise endpoints, or
low-latency IoT-adjacent HERO agents where float32 inference is not viable.

**How**:
1. Add `QuantiseWeights(model *GPT) map[string][12]uint32` calling `mapper.MapToSlots()` on each weight row.
2. Implement `DequantiseForward()` with fixed-point dot product.
3. Cross-compile inference-only path per existing `Makefile` pattern.

**Value Added**:
- Guardrail enforcement and error resolution at the edge — no cloud round-trip for locally-deployed HERO agents.
- Quantisation seed doubles as weight obfuscation — client model weights are not recoverable without the seed.
- Three deployment tiers: INT16 edge, float32 server, Cerebras WSE2 accelerated bulk.

---

### Option 8 — Dynamic Graph NAS for Per-Client Architecture Sizing

**What**: Use `DynamicGraph`'s per-pass rebuild capability to auto-size HEART's architecture
during client onboarding. Clients with compact, rule-structured ontologies (insurance, banking,
regulatory) get shallow models; clients with nuanced, contextual domains (legal, media, creative)
get deeper models.

**How**:
1. During the first fine-tuning epoch, record activation variance across transformer blocks via `DynamicGraph.Operations` log.
2. Feed into `VarianceAnalyzer.Sample()` — high-frequency/low-variance ops are pruning candidates; low-frequency/high-variance ops are duplication candidates.
3. Modify `dg.Params` to add/remove `TransformerBlock` groups before the next epoch.

**Value Added**:
- Architecture auto-sizes to the complexity of the client's domain without manual hyperparameter tuning.
- Runs during onboarding, not as a separate training job.
- The `DynamicGraph`'s per-pass rebuild limitation becomes its key NAS feature rather than a liability.

---

### Option 9 — Ontology Drift Detection as HEART Self-Monitoring Signal

**What**: Run `VarianceAnalyzer` in streaming mode over deterministic embeddings of incoming
`SourceRecord` batches. When top-24 signal indices shift significantly between batches, emit an
`OntologyDrift` event — the pipeline detects when the semantic landscape has materially changed.

**How**:
1. Extend `VarianceAnalyzer` with `DeltaSignalIndices(prev []int) float32` measuring Jaccard distance between two runs' top-24 index sets.
2. When Jaccard delta > 0.4: minor → recompute embeddings and append new records; major → full corpus retrain; critical → Block all HERO decisions and notify client administrator.
3. DVE telemetry drift signals a changing error landscape — triggers `resolution.wasm` retraining for the affected DVE type.

**Value Added**:
- Prevents HERO operating under stale guardrails when client policies or DVE error patterns have changed.
- Drift events logged to the audit trail (Option 6) — compliance teams see exactly when and why the knowledge base was updated.
- Zero new math — `VarianceAnalyzer` already does 95% of the computation.

---

### Option 10 — Unified Checkpoint with DVE Context Versioning

**What**: Merge Gorgonite model weights, optional HashNetwork seed population, client ontology
manifest, and active DVE session registry into a single checkpoint envelope.

**How**:
1. Define `UnifiedCheckpoint{ModelWeights, SeedPopulation (optional), OntologyManifest []PolicyRecord, DVESessionRegistry map[string]DVEContext, OntologyVersionHash [32]byte, CheckpointSignature [32]byte}`.
2. Serialise all sub-states, compute SHA-256, store at `checkpoints/<clientID>/<sha256hex>/checkpoint.bin`.
3. `DVESessionRegistry` maps active DVE IDs to their error history snapshots — ensures `resolution.wasm` generated post-checkpoint can reference the correct DVE state.

**Value Added**:
- Full reproducibility: given a checkpoint, any node can reproduce every HEART decision for both the guardrail and resolution paths.
- `OntologyVersionHash` enables rollback to a prior badge policy version.
- `DVESessionRegistry` enables post-mortem analysis of DVE error sequences against the model that handled them.

---

### Option 11 — NRV Positional Encoding for Policy Document and Error Log Structure

**What**: Apply the mapper's random projection to transformer sequence positions, encoding
position `i` as `mapper.MapToSlots(positionalSinusoids[i])` — 12 uint32 values. Re-enables
the currently disabled sinusoidal `PositionalEncoding`.

**How**:
1. Generate sinusoidal position vectors of length 768 for positions 0..ContextLen.
2. Pass each through `mapper.MapToSlots()` → 12 uint32 values.
3. Unpack to float32 (reverse int16 bit-packing) and add to embedding.
4. Re-enable the `posEnc` addition in `NewGPT()` at line 415.

**Value Added**:
- Re-enables positional encoding with zero new math — fixes a known regression.
- Mapper's sigmoid+quantisation bounds the positional signal, preventing embedding scale mismatch with deterministic `embed.Embed()` output.
- Deterministic projection seed means positional encoding is identical across all HEART instances — consistent behaviour in multi-node DVE deployments.

---

### Option 12 — Multi-Path WASM Compilation: `rule.wasm` / `resolution.wasm` / `patch.wasm`

**What**: HEART fine-tunes on the client's combined corpus (client data + ArXiv + DVE telemetry)
and learns to generate TinyGo source for whichever of the three WASM types is appropriate for
the current inquiry. HERO executes the compiled binary directly — no LLM interpretation at
enforcement or resolution time.

**How**:
1. **WASM type selection logic**: `HEARTService.classifyInquiry()` determines WASM type from the inquiry source before Stage 1:
   - `PolicyBadgeInquiry` → `rule.wasm` (guardrail; primary path)
   - `DVEErrorInquiry` (active DVE confirmed) → `resolution.wasm` (error resolution; secondary path)
   - `SystemPatchInquiry` (no DVE origin) → `patch.wasm` (internal patch; tertiary path)
2. **Compilation pipeline** (shared across all three types):
   - Gorgonite emits TinyGo source via `WASMResponse.Source string` after Stage 4.
   - `exec.Command("tinygo", "build", "-o", "<type>.wasm", "-target", "wasm", srcPath)` — same `exec.Command` pattern as `CerebrasBridge.RunInference()`.
   - `wazero` executes compiled binary against bidirectional verifier test cases (Option 21) before deployment.
   - Compilation failure → hard rejection → multi-turn loop (Option 24) retries Stage 4 with compiler stderr as revision prompt.
3. **Deployment scope** (enforced post-compilation):
   - `rule.wasm`: registered to the DVE I/O interface identified in `PolicyBadgeInquiry.DVEContext`.
   - `resolution.wasm`: dispatched to the originating `DVEErrorInquiry.DVESessionID` execution context.
   - `patch.wasm`: applied system-wide; requires elevated `PatchScope` flag.
4. Content-address each compiled binary via SHA-256. Identical binaries across nodes produce identical identifiers — automatic deduplication.
5. Stage 2 `VarianceAnalyzer` identifies which corpus dimensions carry the most signal per WASM type, enabling source-specific corpus pruning.

**Value Added**:
- HERO executes a typed contract (`Resolve(ctx T) R`) with zero LLM ambiguity at runtime — WASM sandboxing prevents a malformed binary from escaping the HERO runtime.
- Compilation failure as a training signal: TinyGo compiler stderr teaches Gorgonite syntactic correctness without manual labelling.
- All three WASM types share the same compilation toolchain, runtime (`wazero`), and audit trail (Option 6) — one operational pipeline for three execution contexts.

---

## ES Integration Options (Options 13–17)

*Apply to the optional HashNetwork training path only. No-ops when `HEARTConfig.UseHashNetwork = false`. The reward signal is classification accuracy on `(concept, WASMClass)` pairs from the combined corpus — not Bitcoin mining alignment.*

---

### Option 13 — Replace GA Elitism with ES Weighted Update (HashNetwork Only)

**What**: Replace `SelectAndMutate` hard-elitism (top 25%) with the ES weighted perturbation
sum: `seed_t ← seed_{t-1} + α · (1/N) · Σ Rₙεₙ`. All N members contribute to every update.

**How**:
1. After `CalculateBitMatchAdvantage` normalises rewards (z-score already correct), compute weighted sum: `εₙ = seeds[n] - baseSeed`; accumulate `Rₙ · εₙ`.
2. Update `baseSeed ← baseSeed + α · (1/N) · Σ Rₙεₙ`, clipped to `[0, 255]`.
3. Generate next population by sampling Gaussian noise around updated `baseSeed`.
4. Reward: classification accuracy on `(concept, WASMClass)` where `WASMClass ∈ {rule, resolution, patch}`.

**Value Added**:
- ES paper: 15.5× lower std-dev vs. GA elitism — equivalent improvement for HashNetwork convergence on the 3-class WASM taxonomy.
- `CalculateBitMatchAdvantage` already implements ES reward normalisation — only the weighted sum update is missing.
- Gives `EvoGRPO` a proper mathematical foundation.

---

### Option 14 — Mirrored Sampling for Population Evaluation (HashNetwork Only)

**What**: For each noise vector ε, also evaluate `(seed + ε, seed - ε)`. Halves variance at
zero additional compute cost.

**How**: In `EvaluatePopulationBatch`, generate N/2 positive perturbations and mirrors.
Reward estimate: `(1/N) Σ [R(seed+ε) - R(seed-ε)] · ε`.

**Value Added**: 50% variance reduction for free. Recommended as simultaneous with Option 13.

---

### Option 15 — σ Annealing Tied to Ingestion Phases (HashNetwork Only)

**What**: Replace binary `StaticMidstate` with continuous σ annealing tied to corpus ingestion
phase. Bulk ingestion (new client): large σ (broad exploration). Incremental updates: small σ
(precision refinement).

**How**: `σ(epoch) = σ_max · exp(-epoch / τ)`. Governs both seed perturbation magnitude and
difficulty scaling. `StaticMidstate` → `σ_threshold`. Same schedule governs Gorgonite LR warmup.

**Value Added**: Smooth convergence curves; σ annealing is the mechanism behind ES's 15.5× std-dev advantage.

---

### Option 16 — Complete EvoGRPO with Multi-Source Corpus as Reward (HashNetwork Only)

**What**: Implement `EvoGRPO` using Options 13–15 as its mathematical base, with 3-class WASM
classification accuracy on the combined corpus as the fitness function.

**How**:
1. Replace stub fitness with `EvolutionaryHarness.EvaluatePopulationBatch` on `(concept, WASMClass)` pairs from the combined multi-source corpus.
2. Crossover: `child = α·elite + (1-α)·mirror_elite`.
3. Wire GRPO advantage weighting; `CalculateBitMatchAdvantage` as backing implementation.

**Value Added**: Fulfils `EvoGRPO`'s design intent. HashNetwork and Gorgonite share one training corpus; their disagreements are the strongest escalation signal.

---

### Option 17 — HashNetwork Seeds Trained on 3-Class WASM Concept Space (HashNetwork Only)

**What**: Treat all of HashNetwork's seeds as parameter vector θ; apply ES update from Option 13
directly against `(concept_embedding, WASMClass)` pairs from the combined corpus.

**How**:
1. Flatten `Seeds1 + Seeds2 + SeedsOut` → byte vector θ.
2. Sample N Gaussian perturbations; evaluate via `CalculateBitMatchAdvantage`; apply ES update.
3. Gorgonite and HashNetwork train concurrently. `VarianceAnalyzer` compares confidence per concept class. Route inference to the more confident model per WASM type.

**Value Added**: HashNetwork handles high-confidence routine guardrail checks (<1 ms); Gorgonite handles nuanced multi-policy or multi-DVE-error cases requiring contextual reasoning.

---

## Cognition Options (Options 18–25)

*Apply the generator-verifier duality, hierarchical reasoning, curriculum learning, and
bidirectional verification from the Cognition Implementation paper to all three WASM paths.
Each is independently implementable.*

---

### Option 18 — Generator-Verifier Duality: HashNetwork (Optional) as Deterministic Critic

**What**: When HashNetwork is enabled, use `RecursiveEngine`'s 21-pass consensus as a fast
verifier for every Gorgonite-generated `WASMDecision`. When disabled, bidirectional agents
(Option 21) are the sole trust gate.

**How**:
1. After `HEARTService` produces any `WASMDecision`, tokenise the decision summary and feed into `RecursiveEngine`'s 21-pass loop.
2. `ConsensusResult.ClassDistribution[3]` maps to the appropriate 3-class space for the WASM type.
3. Add `VerifierConfidence float32` and `HashNetworkVerified bool` to `WASMDecision`.
4. When disabled: `HashNetworkVerified` omitted; `BidirectionalVerified` (Option 21) is sole trust signal.

**Cognition paper connection**: HashNetwork is architecturally independent of Gorgonite's float32 attention — a hallucinated guardrail decision that sounds plausible cannot fool HashNetwork's hash-alignment test on the client's concept space. The two models' computational independence is the correctness guarantee.

**Value Added**:
- `ModelDisagreement: true` is the strongest escalation signal for any of the three WASM paths.
- Disagreement cases feed the training queue (Option 23) — the joint uncertainty frontier of both models drives the next retraining cycle.

---

### Option 19 — Unified 4-Stage Hierarchical Decision Pipeline

**What**: Replace `processInquiry()`'s monolithic single-shot generation with a 4-stage
conditional pipeline applicable to all three WASM types. The stage content adapts to the inquiry
type; the structure is invariant.

**How**:

**Stage 1 — Action Inquiry** *(varies by WASM type)*:
- **`rule.wasm` path**: `PolicyBadgeInquiry` from Badge Lab — carries `ValuesSignals[]`, `OntologySignals[]`, `DVEContext`, and the badge `Name`. This is the primary entry point.
- **`resolution.wasm` path**: `DVEErrorInquiry` — carries `DVESessionID`, `ErrorType`, `ErrorMessage`, `ErrorContext`. Requires a live DVE session ID; rejected without one.
- **`patch.wasm` path**: `SystemPatchInquiry` — carries `ComponentID`, `ErrorCode`, `SystemState`. No DVE association.

**Stage 2 — Core Principles / Techniques**:
- **`rule.wasm`**: `IdentifyRelevantPolicies(inquiry)` — uses `OntologySignals` to query the client corpus via deterministic embeddings; outputs `[]PolicyPrinciple{PolicyID, Section, OntologyClass, ValuesAlignment, Rationale}`. `ValuesSignals` condition the `Block`/`Escalate` threshold.
- **`resolution.wasm`**: `IdentifyCoreErrorTechniques(inquiry)` — extracts the `ErrorClass` (Construction / TheoremCall / MathematicalTransformation) and names the pivotal conceptual tools required before any resolution attempt.
- **`patch.wasm`**: `IdentifyPatchScope(inquiry)` — determines if the patch is a hot fix, restart, or migration; identifies affected components.

**Stage 3 — Decision / Resolution Sketch**:
A high-level structured outline conditioned on Stage 2 output — `Permit|Block|Escalate` with rationale for `rule.wasm`; pseudocode resolution plan for `resolution.wasm`; component impact map for `patch.wasm`. No executable source yet. Surfaced in `WASMDecision.Rationale` for human review of Escalate/complex cases.

**Stage 4 — Executable Source**:
Generate TinyGo source implementing the appropriate `Resolve(ctx T) R` interface, conditioned on both the Stage 2 principles and the Stage 3 sketch. Passed to the compilation pipeline (Option 12) to produce the typed WASM binary. If TinyGo compilation fails, compiler stderr is fed back as a Stage 4 revision prompt (Option 24 multi-turn loop).

**Cognition paper connection**: The paper's key finding — that naming core techniques explicitly before generating the proof prevents "blind wandering into dead ends" — applies to all three WASM paths. Naming the relevant policy principles (for rules), the error technique class (for resolutions), or the patch scope (for patches) before generating the executable source prevents generating logic that addresses the wrong constraint.

**Value Added**:
- Stage 2 output is the natural input to the bidirectional verifier (Option 21) — the verifier checks that the compiled WASM is consistent with the named principles, not just the raw inquiry.
- Stage 3 sketch surfaces in `WASMDecision.Rationale` — human reviewers in Escalate cases see reasoning structure, not just a verdict.
- The same 4-stage structure handles all three WASM types — one pipeline, three execution contexts.

---

### Option 20 — Curriculum Training: Apprentice→Journeyman→Expert on Combined Corpus

**What**: Structure fine-tuning into three progressive stages across the combined multi-source
corpus. Curriculum-trained small models match models several times larger trained without it —
critical for edge DVE deployment (Option 7).

**How**:
1. **Apprentice stage**: raw `(InquiryType, SourceChunk, WASMClass)` pairs. Objective: vocabulary and syntax of all three domains. LR: 3e-4.
2. **Journeyman stage**: `(InquiryType, SourceChunk, DecisionSketch, WASMClass)` triples — model generates the sketch before the class verdict. LR: 1e-4.
3. **Expert stage**: full 4-stage hierarchy from Option 19 — inquiry → core principles → sketch → full TinyGo source for the appropriate WASM type. LR: 3e-5.

**Value Added**:
- Three checkpoints: apprentice for edge DVE deployment, expert for server deployment with Cerebras acceleration.
- The journeyman stage prevents memorisation of specific policy or error examples by forcing structural understanding first.
- If HashNetwork is enabled, σ annealing (Option 15) maps directly: apprentice = high σ, expert = low σ — unified training tempo across both models.

---

### Option 21 — Bidirectional WASM Verification via Policy Checker + Python Subprocess

**What**: Add forward and backward verification agents to `HEARTService`. Any compiled WASM
binary must pass both directions before deployment — preventing HERO from operating under a
`rule.wasm` that contradicts the Policy Badge that created it, or a `resolution.wasm` that
would worsen the DVE error it is meant to fix.

**How**:
1. **Forward agent** — `ForwardVerify(inquiry HEARTInquiry, wasmPath string) (bool, string, error)`: given the inquiry (premises) and compiled WASM (conclusion), verify the execution logic correctly addresses the stated constraint. Python subprocess call (`exec.Command("python3", "verifier.py", ...)`) — JSON in, JSON out, same pattern as `CerebrasBridge.RunInference()`.
2. **Backward agent** — `BackwardVerify(inquiry HEARTInquiry, wasmPath string) (bool, string, error)`: given the WASM conclusion, reverse-trace to verify it is consistent with the declared source — for `rule.wasm`, does the enforcement logic match the Policy Badge's ontology signals? For `resolution.wasm`, does the resolution not worsen the DVE's declared error state?
3. **wazero runtime execution** — before either agent: execute the compiled binary against sample context inputs via `wazero` (pure-Go WASM runtime, zero CGo). `wazero.Compile()` + `module.ExportedFunction("Resolve").Call()` + class export function must all succeed. Catches panics, OOM, and interface violations.
4. Add `BidirectionalVerified bool`, `ForwardVerifierMsg string`, `BackwardVerifierMsg string`, `WazeroExecPassed bool` to `WASMDecision`. All three gates must pass before deployment.

**Value Added**:
- Formally verified WASM binaries — HERO cannot be blocked or resolved by a binary that contradicts its declared source (badge or DVE error).
- Verification failures surface as training signals for the next curriculum stage (Option 20).
- wazero sandboxing ensures verification cannot be compromised by a malformed binary.

---

### Option 22 — Three-Class Output Taxonomy as Exported WASM Functions

**What**: Each WASM binary exports its class as a first-class WASM function — machine-readable
without text parsing. HERO calls the class export via `wazero` before dispatching `Resolve()`.

**How**:
1. **`rule.wasm`** exports `GuardrailClass() uint32`: `0=Permit | 1=Block | 2=Escalate`.
   - **Permit**: action is ontologically consistent with the Policy Badge's declared ontology signals. HERO proceeds.
   - **Block**: action violates a matched policy principle. HERO halts; `Resolve()` return carries the specific violation and policy section.
   - **Escalate**: action is in a grey zone. HERO queues for human approval; `Resolve()` carries the relevant conflicting policy sections.
2. **`resolution.wasm`** exports `ErrorClass() uint32`: `0=Construction | 1=TheoremCall | 2=MathematicalTransformation`.
   - **Construction**: introduce new capability (retry wrapper, circuit breaker, fallback). Root cause: missing component.
   - **TheoremCall**: invoke an existing `SkillNode` from the KNIRVCHAIN registry that has already resolved a similar DVE error.
   - **MathematicalTransformation**: recast the error into a different domain where a known solution exists — the most novel class.
3. **`patch.wasm`** exports `PatchScope() uint32`: `0=Hot | 1=Restart | 2=Migration`.
4. TinyGo emits these as `//export GuardrailClass` / `//export ErrorClass` / `//export PatchScope` alongside `//export Resolve`.
5. KNIRVCHAIN indexes WASM binaries by their exported class values (read via `wazero` at commit time) — enables class-filtered lookups without executing `Resolve()`.

**Cognition paper connection**: The paper's three insight classes (Construction, Theorem Call, Mathematical Transformation) annotated into 100K theorem-proof pairs create the structured training signal that teaches the LLM to reason about reasoning type before generating the proof. The `ErrorClass` taxonomy for `resolution.wasm` is a direct adoption. `GuardrailClass` and `PatchScope` extend this to the other two WASM paths.

**Value Added**:
- All three WASM types are class-queryable at <1 ms via `wazero` — HERO can route without parsing any text.
- `ErrorClass=TheoremCall` resolution candidates trigger a KNIRVCHAIN lookup for existing SkillNodes before generating new source — deduplication at the resolution level.
- Class-indexed rule/resolution/patch repositories enable fast pre-filtering before full `Resolve()` execution.

---

### Option 23 — Entropy-Spike Detection as Ontology Gap / Novel Error Signal

**What**: Monitor Gorgonite's per-token prediction entropy during any WASM generation. Entropy
spikes signal: for the `rule.wasm` path, an ontology gap in the client's onboarded data; for
the `resolution.wasm` path, a novel DVE error type not well-covered by training; for `patch.wasm`,
an unexpected system state.

**How**:
1. Re-enable softmax in `SelfAttention.Forward` (prerequisite). Per-token entropy: `H = -Σ p·log(p)`.
2. During `processInquiry()`, instrument each generation step. When `H > threshold` (suggest 3.0 nats):
   a. **Route to HashNetwork (optional)**: run `RecursiveEngine`'s 21-pass loop; use `ConsensusResult` to select the next token if confidence ≥ 0.85.
   b. **Flag by WASM path**: `rule.wasm` spike → `GapQueue` (client notified of undocumented policy area); `resolution.wasm` spike → `NovelErrorQueue` (novel DVE error flagged for ArXiv scan); `patch.wasm` spike → alert on unexpected system state.
3. High-entropy cases feed the training queue by magnitude — highest-uncertainty cases retrain first.

**Value Added**:
- `rule.wasm` spikes are actionable business intelligence: tell the client which action types their current Policy Badges do not clearly address.
- `resolution.wasm` spikes trigger an ArXiv scan (Stage 1 `ArXivWorker`) for the novel error type — system self-bootstraps research for errors it cannot resolve.
- Restoring softmax (prerequisite) also unblocks Options 3 and 11.

---

### Option 24 — Multi-Turn Cognitive Loop for Deliberative WASM Generation

**What**: Convert `processInquiry()` to an iterative refinement loop: generate → verify → revise
→ re-verify, with early exit on success or escalation on exhaustion. Applies to all three WASM
types; `MaxTurns` is configurable per type.

**How**:
1. Wrap the 4-stage pipeline (Option 19) in a loop:
   ```
   for turn := 0; turn < maxTurns; turn++ {
       decision = generateHierarchical(inquiry, priorFailures)
       fwd, fwdMsg = ForwardVerify(inquiry, decision)
       bwd, bwdMsg = BackwardVerify(inquiry, decision)
       if fwd && bwd { return decision with BidirectionalVerified=true }
       priorFailures = append(priorFailures, {fwdMsg, bwdMsg})
   }
   return decision with BidirectionalVerified=false, class=Escalate|NovelError|AlertState
   ```
2. On failure, regenerate only the failing stage (Stage 3 sketch if backward fails, Stage 4 source if forward fails).
3. Add `TurnCount int`, `AverageTurns float64` to `HEARTServiceStats`. Rising average turns per WASM type indicates data quality issues in that path's corpus.
4. `MaxTurns`-exhausted `resolution.wasm` cases are added to `NovelErrorQueue` (Option 23) — the hardest errors always get human review and ArXiv scan.

**Value Added**:
- `MaxTurns=1` is a drop-in replacement for current single-pass behaviour — zero regression risk.
- `AverageTurns` per WASM type is a real-time health signal: rising `rule.wasm` turns → client needs more Policy Badges; rising `resolution.wasm` turns → DVE error landscape has shifted.
- Max-turn failures automatically feed the appropriate training queue — hardest cases become next training batch.

---

### Option 25 — TRPO Trust Region for EvoGRPO Stability (HashNetwork Only)

**What**: Add a trust-region constraint to the ES weighted update in `EvoGRPO` (Option 13),
preventing any single update from shifting the base seed distribution beyond a Hamming distance
budget. Prevents catastrophic collapse during early training on small client corpora.

**How**:
1. After computing unconstrained update `Δseed = α · (1/N) · Σ Rₙεₙ`, compute `d = HammingDistance(baseSeed, baseSeed + Δseed)`.
2. If `d > budget`: `α' = α · budget / d`; apply scaled update.
3. `budget(epoch) = budget_max · exp(-epoch / τ)` — unified with σ annealing (Option 15).
4. Log `TrustRegionViolations uint64`.

**Value Added**:
- Prevents training instability when the combined corpus (client + ArXiv + DVE) is small or imbalanced early in onboarding.
- Completes the full ES training stack: principled update (13) + variance reduction (14) + noise schedule (15) + stability (25).

---

## Implementation Priority Matrix

| Option | Effort | Impact | HashNetwork Dep | Recommended Order |
|--------|--------|--------|-----------------|-------------------|
| 5 — cl100k Tokeniser | Low | High | No | **1st** — unblocks all three WASM paths |
| 1 — Multi-Source Data Feed | Medium | Very High | No | **2nd** — core ingestion for all paths |
| 9 — Ontology / Error Drift Detection | Low | High | No | **3rd** — self-monitoring, near-zero cost |
| 4 — Sliding Window Temporal Context | Low | High | No | **4th** — stateful guardrails + cascading error detection |
| 22 — Three-Class WASM Taxonomy | Low | Medium | No | **5th** — annotates training data, all paths |
| 19 — 4-Stage Hierarchical Pipeline | Medium | Very High | No | **6th** — core decision structure, all paths |
| 12 — Multi-Path WASM Compilation | Medium | Very High | No | **7th** — rule/resolution/patch compilation |
| 21 — Bidirectional Verification | Medium | Very High | No | **8th** — requires Options 12, 19 |
| 24 — Multi-Turn Cognitive Loop | Medium | Very High | No | **9th** — requires Options 19, 21 |
| 20 — Curriculum Training | Medium | High | No | **10th** — requires Options 1, 12, 19 |
| 6 — Cryptographic Audit Trail | Low | High | Optional | **11th** — compliance + DVE traceability |
| 23 — Entropy-Spike Detection | Medium | High | Optional | **12th** — requires softmax re-enabled |
| 18 — Generator-Verifier Duality | Low | Very High | Optional | **13th** — fast verify path |
| 2 — Variance-Guided Head Pruning | Medium | Medium | No | **14th** — domain-specialised model |
| 11 — NRV Positional Encoding | Low | Medium | No | **15th** — fixes disabled code |
| 8 — Dynamic NAS per Client | High | Medium | No | **16th** — auto architecture sizing |
| 10 — Unified Checkpoint | Medium | Medium | Optional | **17th** — operational integrity + DVE context |
| 7 — INT16 Edge Quantisation | High | High | No | **18th** — edge DVE deployment |
| 3 — HashNetwork Fast-Path | Low | High | Required | **19th** — enable after HashNetwork trained |
| 13 — ES Weighted Update | Low | Very High | Required | **20th** — if HashNetwork enabled |
| 14 — Mirrored Sampling | Low | High | Required | **21st** — pair with Option 13 |
| 15 — σ Annealing | Low | Medium | Required | **22nd** — smooth training dynamics |
| 16 — Complete EvoGRPO | Medium | High | Required | **23rd** — fulfils design intent |
| 25 — TRPO Trust Region | Low | High | Required | **24th** — completes ES stack |
| 17 — HashNetwork on Combined Concept Space | High | Very High | Required | **25th** — full model unification |

---

## Cross-Cutting Integration Architecture

```
 DATA SOURCES                              BADGE LAB              ACTIVE DVE
 ┌─────────────┐                               │                      │
 │ Client Data │ ─────────────────────────────┐│                      │
 │ (policies,  │                              ││  PolicyBadgeInquiry  │  DVEErrorInquiry
 │  SOPs, etc) │                              ││  {ValuesSignals,     │  {DVESessionID,
 │             │                              ││   OntologySignals,   │   ErrorType,
 │ ArXiv API   │ ─────────────────────────────┤│   DVEContext}        │   ErrorMessage}
 │ (cs.AI/LG/  │                              ││         │            │        │
 │  SE/CR)     │                              ││         │            │        │
 │             │                              ││         │            │        │
 │ DVE         │ ─────────────────────────────┘│         │            │        │
 │ Telemetry   │                               │         │            │        │
 └─────────────┘                               │         │            │        │
         │                                     │         │            │        │
         ▼ (Opt 1) chan SourceRecord           │         ▼            ▼        │
Stage 1: source_manager.go                     │    HEARTService /heart/advise │
  ├── ClientDataWorker                         │         │            │        │
  ├── ArXivWorker                              │    classifyInquiry() │        │
  └── DVETelemetryWorker                       │    ┌────▼────────────▼──────┐ │
         │                                     │    │  WASM Type Decision    │ │
         ▼ (Opt 5) cl100k                      │    │  Badge  → rule.wasm    │ │
Stage 2: DATA_ENCODER                          │    │  DVEErr → resolution   │ │
  ├── tiktoken cl100k                          │    │  SysPatch → patch.wasm │ │
  ├── embed.Embed() [deterministic]            │    └────────────┬───────────┘ │
  ├── VarianceAnalyzer (top-24 dims)           │                 │             │
  ├── Mapper (768→12 uint32)                   │    [Optional] HashNetwork     │
  └── SlidingWindowGenerator                   │    Fast Path (Opt 3)          │
         │                                     │    confidence ≥ 0.85          │
         ▼ (Opt 9)                             │    → short-circuit            │
VarianceAnalyzer Jaccard drift                 │    < 0.85 → hint to Gorgonite │
  ├── OntologyDrift event                      │                 │             │
  └── top-24 signal indices                    │    Opt 19: 4-Stage Pipeline   │
         │                                     │    ┌────────────▼───────────┐ │
         ├──────────► (Opt 2) Head Pruning     │    │ Stage 1: Action Inquiry│ │
         │                                     │    │ (PolicyBadge|DVEError  │ │
         ▼ (Opt 11)                            │    │  |SystemPatch)         │ │
Mapper (768→12 uint32)                         │    │                        │ │
         │                                     │    │ Stage 2: Core          │ │
         ├── [Optional] HashNetwork            │    │ Principles/Techniques  │ │
         │   Training (Opts 13–17)             │    │ (OntologySignals →     │ │
         │   EvolutionaryHarness               │    │  PolicyPrinciple[] OR  │ │
         │   ← (concept, WASMClass) pairs      │    │  ErrorClass OR         │ │
         │   ES update, mirrored, σ, TRPO      │    │  PatchScope)           │ │
         │                                     │    │                        │ │
         ▼ (Opt 8) DynamicGraph NAS            │    │ Stage 3: Sketch        │ │
Gorgonite GPT                                  │    │ (Permit|Block|Escalate │ │
(Opt 20: apprentice→journeyman→expert)         │    │  OR resolution plan    │ │
float32 weights (Opt 2 pruned)                 │    │  OR patch impact map)  │ │
         │                                     │    │                        │ │
         └── Fine-tuned on combined corpus ───►│    │ Stage 4: TinyGo Source │ │
             (client + ArXiv + DVE)            │    │ → tinygo build         │ │
                                               │    │ → rule|resolution|     │ │
                                               │    │   patch .wasm          │ │
                                               │    └────────────┬───────────┘ │
                                               │                 │             │
                                               │    Opt 24: Multi-Turn Loop    │
                                               │    generate→verify→revise     │
                                               │    MaxTurns-exhausted→Escalate│
                                               │                 │             │
                                               │    Opt 21: Bidirectional      │
                                               │    wazero: class() + Resolve()│
                                               │    Forward agent (Python)     │
                                               │    Backward agent (constraint)│
                                               │    BidirectionalVerified flag │
                                               │                 │             │
                                               └─────────────────┼─────────────┘
                                                                 │
                                                    WASMDecision + rule.wasm
                                                    (or resolution/patch .wasm)
                                                                 │
                                               ┌─────────────────▼──────────────────┐
                                               │        Deployment Dispatch         │
                                               │                                    │
                                               │  rule.wasm  ──► DVE I/O interface  │
                                               │               (scoped to badge DVE)│
                                               │                                    │
                                               │  resolution.wasm ► DVE execution   │
                                               │               context (same DVE)   │
                                               │                                    │
                                               │  patch.wasm ──► system-wide        │
                                               └─────────────────┬──────────────────┘
                                                                 │
                                               Opt 6: Audit log
                                               audits/<clientID>/<sha256hex>.json
                                               {inquiryHash, wasmSHA256, wasmType,
                                                sourceContextID, timestamp}
```

**Three inference tiers:**

| Tier | Model | Condition | Latency | Use Case |
|------|-------|-----------|---------|----------|
| Fast | HashNetwork (optional) | confidence ≥ 0.85, client enabled | <1 ms | Routine high-frequency guardrail checks |
| Standard | Gorgonite float32 | always available | ~200 ms | Multi-policy conflicts, novel DVE errors |
| Accelerated | Gorgonite via Cerebras WSE2 | optional hardware | ~10 ms at scale | High-throughput bulk WASM generation |

**Three WASM execution contexts:**

| WASM Type | Trigger | Deployment Target | DVE Scope |
|-----------|---------|-------------------|-----------|
| `rule.wasm` | Policy Badge from Badge Lab | DVE I/O interface (enforces at boundary) | Scoped to badge's `DVEContext` |
| `resolution.wasm` | Active DVE error discovery | DVE execution context (applied in-session) | Scoped to `DVESessionID` |
| `patch.wasm` | Internal system error | System-wide (no DVE association) | None |

**Two training regimes (when HashNetwork is enabled):**

| Model | Parameter Type | Reward Signal | Trust Region |
|-------|---------------|---------------|--------------|
| HashNetwork | 32-byte seed arrays | 3-class WASM classification accuracy on combined corpus | Hamming distance budget (Opt 25) |
| Gorgonite | float32 weight tensors | Cross-entropy on combined corpus (client + ArXiv + DVE) | σ annealing step size (Opt 15) |

**Three reasoning layers in HEARTService (post Options 18–24):**

| Layer | Mechanism | Latency | Role |
|-------|-----------|---------|------|
| Generation | Gorgonite 4-stage hierarchical (Opt 19) | ~200ms | Produce typed WASM draft for rule/resolution/patch |
| Verification | wazero + bidirectional forward+backward (Opt 21) | ~50ms | Validate before deployment |
| Deliberation | Multi-turn cognitive loop (Opt 24) | ×turns | Refine on failure; escalate on exhaustion |

**Self-improving coverage loop (post Options 20, 23, 24):**

```
Gorgonite entropy spike on inquiry type X
    ↓
rule.wasm path → GapQueue → client notified: "add Policy Badge or documentation for X"
resolution.wasm path → NovelErrorQueue → ArXiv scan triggered for error type X
patch.wasm path → AlertState → system diagnostic scan for component X
    ↓
New training data ingested (Option 1 multi-source pipeline)
    ↓
Fine-tuning on new examples (Option 20 curriculum stage)
    ↓
Lower entropy on X at next occurrence
    ↓
Coverage gap closed; queues shift to next frontier
```
