# HEART × Gorgonite: Enterprise Ontological Guardrail System

> Architectural strategy for deploying `HEART` (HEARTService + Gorgonite GPT) as a
> real-time ontological guardrail advisor for the HERO agent, operating on each enterprise
> client's onboarded business data. HEART enforces the client's business ontology by
> advising HERO on whether proposed actions comply, violate, or require human escalation
> before execution.
>
> HashNetwork is an optional inference accelerator — enabled per-client for high-volume or
> latency-sensitive deployments. All options function without it; the same "if available,
> use it; otherwise proceed" pattern already established by `CerebrasBridge` applies.
> CerebrasBridge itself remains optional for bulk high-throughput inference.
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
| **DynamicGraph** | Wrapper that re-materialises a fresh static graph each forward pass — pseudo-dynamic dispatch, zero architectural cost for depth changes |
| **CerebrasBridge** | *Optional*: exports Gorgonia `float32` weights to NPZ, shells out to `cs_python` for Cerebras WSE2 bulk inference |
| **HEARTService** | HTTP server (`/heart/advise`, `/heart/health`, `/heart/stats`); receives `GuardrailInquiry` from HERO; evaluates proposed action against client ontology; returns `GuardrailDecision` |
| **NetworkMetricsProcessor** | Adaptive Z-score normalisation of incoming context vectors → `HEARTInput` structs |

**Known gaps in current code:**
- Softmax commented out in `SelfAttention.Forward`
- Positional encoding disabled (line 415, `NewGPT`)
- Multi-head concat replaced with head sum (lossy)
- `BPETokenizer` is a complete stub — `loadVocab()` returns empty map, every token encodes as `<UNK>`
- No real dataset consumer — `CreateDummyDataset` is the only data source

---

### HashNetwork (Optional) — Fast-Path Guardrail Classifier

When enabled for a client, HashNetwork provides a sub-millisecond first-pass guardrail verdict
before Gorgonite's full inference. When disabled, all inquiries route directly to Gorgonite.

| Component | Description |
|-----------|-------------|
| **HashNetwork** | 3-layer neural network whose parameters are 32-byte seed arrays; forward pass: `SHA-256(input ∥ seed) → normalised float [0, 1]`; no matrix multiplication |
| **RecursiveEngine** | 21 temporal passes + majority vote consensus → `ConsensusResult{ConfidenceScore, VoteCount, ClassDistribution}` |
| **EvolutionaryHarness** | Population-based seed training: z-score reward normalisation (`CalculateBitMatchAdvantage`), GA elitism (upgradeable to ES weighted update via Options 13–17) |
| **EvoGRPO** | Placeholder for ES+GRPO fusion; intended to share training corpus with Gorgonite when enabled |

**Activation**: `HEARTConfig.UseHashNetwork bool` — per-client flag. When `true`, HashNetwork high-confidence verdicts (≥ 0.85) short-circuit Gorgonite; lower-confidence verdicts pass `ClassDistribution` as a soft conditioning hint to Gorgonite rather than wasting it.

---

## Operating Objective

Every enterprise client is provisioned with a dedicated HEART instance. The client's onboarded
business data — policies, SOPs, contracts, ontologies, compliance documents, product catalogs,
org charts, approval workflows — forms HEART's knowledge base. HEART's sole function is to
advise HERO in real-time:

> *"Is this action the HERO agent is about to take consistent with this client's business ontology?"*

The HEART → HERO interaction is a guardrail loop:

```
HERO proposes action
        │
        ▼ GuardrailInquiry
  HEARTService /heart/advise
        │
        ├── [Optional] HashNetwork fast path (confidence ≥ 0.85 → short-circuit)
        │
        └── Gorgonite full inference (always available)
              │
              ├── Bidirectional verification (forward + backward agents)
              │
              └── GuardrailDecision {Permit | Block | Escalate}
                        │
              HERO acts / halts / queues for human review
```

**Three guardrail outcomes:**
- **Permit** — action is ontologically consistent with the client's declared policies. HERO proceeds.
- **Block** — action violates client policy. HERO halts; the `rule.wasm` `Enforce()` return carries the specific violation and the policy section breached.
- **Escalate** — action is in a grey zone not resolved by existing rules. HERO queues for human approval; the `rule.wasm` carries the relevant policy sections for the reviewer.

---

## Data Ingestion

Client business data is the training and retrieval corpus for each HEART instance. The pipeline
stages from the KNIRVHASHER codebase are repurposed:

| Stage | Original Role | Enterprise Guardrail Role |
|-------|--------------|--------------------------|
| 0 | `DATA_CONNECTOR` | Client data ingestor — connects to client storage (S3, GDrive, SharePoint, database exports, REST APIs) and loads business documents |
| 1 | `DATA_MINER` | Document parser — PDF/DOCX/JSON/HTML extraction; policy chunking; change-detection for updated documents; outputs `chan PolicyRecord` |
| 2 | `DATA_ENCODER` | tiktoken cl100k tokenisation; BGE-768 embeddings; `VarianceAnalyzer` for domain signal dimension selection; `Mapper` for slot encoding; `SlidingWindowGenerator` for sequential policy context |
| 3 | `DATA_TRAINER` | Gorgonite fine-tuning on client corpus; optional HashNetwork seed training on ontology-derived `(concept, guardrailClass)` pairs |

No external data source (ArXiv, Alpaca, etc.) is required or used in the enterprise guardrail
deployment. The client's uploaded or connected business data is the entire knowledge base.

---

## Architectural Options

Options 1–12 address structural integration for the enterprise guardrail use case. Options 13–17
address the optional HashNetwork training path using the Evolution Strategies paper. Options 18–25
apply the generator-verifier duality, hierarchical reasoning decomposition, curriculum learning,
and bidirectional verification from the Cognition Implementation paper.

---

### Option 1 — Live Client Data Feed into HEART Training

**What**: Wire Stage 1's document parser directly into HEART's training loop. As the client
onboards new policies, updates SOPs, or modifies their ontology, changes are immediately ingested
as fine-tuning records — HEART adapts to the client's evolving business context without manual
retraining cycles.

**How**:
1. Extend `paper_manager.go` → `document_manager.go`: emit `chan PolicyRecord` instead of writing to disk. `PolicyRecord` carries `{DocumentID, ChunkText, PolicyClass, EffectiveDate, ChangeType}`.
2. Add a `GuardrailTrainingAdapter` that converts `PolicyRecord.ChunkText + PolicyClass` into `GuardrailInquiry` training pairs: `{ActionType: policy_section_heading, ActionContext: chunk_text, ExpectedDecision: Permit|Block|Escalate}`.
3. Feed the channel into `TrainModel()` replacing `CreateDummyDataset`.
4. Emit `OntologyDrift` events (Option 9) when new ingestion changes the top-24 signal dimensions — triggers a prompt model checkpoint and notifies the client that their guardrail knowledge base has been updated.

**Value Added**:
- Client policy updates propagate to HEART guardrails within the same pipeline cycle — no deployment lag when policies change.
- `EffectiveDate` allows HEART to enforce temporal validity: a policy that became effective yesterday does not trigger retroactive violations.
- The same document chunks feed Gorgonite fine-tuning and (optionally) HashNetwork seed training — one ingestion pipeline, two model types.
- Change-detection in `document_manager.go` produces a diff-based training set: only modified policy sections require new fine-tuning, not a full corpus rerun.

---

### Option 2 — Variance-Guided Attention Head Pruning for Domain Specificity

**What**: Run Stage 2's `VarianceAnalyzer` over HEART's attention weight distributions during the
initial client onboarding pass. Prune heads whose output variance falls below threshold. Surviving
heads specialise on the semantic dimensions that carry the most signal in this client's specific
domain vocabulary.

**How**:
1. After the first fine-tuning epoch on client data, iterate over `block.attention.heads` and record output variance per head.
2. Feed per-head variance vectors into `VarianceAnalyzer.Sample()` and call `Calculate()`.
3. Drop heads below threshold — set `wQuery/wKey/wValue` to zero tensors, skip in `Forward()`.
4. Reinitialise surviving heads with Glorot; resume fine-tuning.

**Value Added**:
- Pruned model carries only the attention capacity relevant to this client's domain — a law firm's HEART does not waste capacity on engineering ontologies.
- `bge_signal_indices.json` persists the pruning decision — architecture is auditable and reproducible per client.
- Smaller model → lower latency guardrail decisions for high-throughput HERO deployments.
- Extends the dimensionality reduction philosophy (768→24) into HEART's internal structure.

---

### Option 3 — HashNetwork as Optional Fast-Path Guardrail Classifier

**What**: When HashNetwork is enabled (`HEARTConfig.UseHashNetwork = true`), use `RecursiveEngine`'s
21-pass majority-vote consensus as a sub-millisecond first-pass guardrail verdict. High-confidence
verdicts (≥ 0.85) short-circuit full Gorgonite inference. For clients where HashNetwork is
disabled, this option is a complete no-op — same "if accelerator available, use it" pattern as
`CerebrasBridge`.

**How**:
1. On `GuardrailInquiry`, tokenise the proposed action via cl100k (Option 5), map through `mapper.MapToSlots()` to produce a 12-slot `[12]uint32` input vector.
2. Feed into `RecursiveEngine.Execute21PassLoop()`. `ConsensusResult` carries `{ConfidenceScore, VoteCount, ClassDistribution[3]}` where indices map to `{0: Permit, 1: Block, 2: Escalate}`.
3. If `ConfidenceScore ≥ 0.85`, return `GuardrailDecision` from HashNetwork directly — skip Gorgonite. Log `HashNetworkShortCircuit: true` in response metadata.
4. If `ConfidenceScore < 0.85`, pass to Gorgonite full inference with `HashNetworkHint: ClassDistribution` as an attention conditioning prefix — the fast model informs the slow model rather than discarding its work.
5. **Disabled path**: when `HEARTConfig.UseHashNetwork = false`, steps 2–4 are skipped entirely.

**Value Added**:
- <1 ms classification for routine, high-confidence guardrail cases — critical for HERO agents operating at high action frequency (e.g., automated data pipeline orchestrators).
- Clients with simple, rule-bound ontologies (where HashNetwork achieves high confidence routinely) get an effective rule-engine speed; clients with nuanced ontologies rely primarily on Gorgonite.
- Identical to the `CerebrasBridge` availability pattern — the codebase already has the "if accelerator available, use it" control flow.

---

### Option 4 — Sliding Window Action History for Sequential Guardrail Context

**What**: Apply `SlidingWindowGenerator` to the sequence of HERO's recent actions rather than
processing each `GuardrailInquiry` in isolation. HEART attends over a causal history of what
HERO has already done before deciding whether the next proposed action is consistent.

**How**:
1. Maintain a per-session `ActionHistoryBuffer` keyed by `(clientID, sessionID)`.
2. On each `GuardrailInquiry`, generate `SlidingWindow` structs over the flattened action token sequence (each "token" is a quantised action embedding index).
3. Embed each window's context tokens via `EmbeddingLayer` and pass through transformer blocks.
4. The causal attention mask (`createCausalMask()` already in Gorgonite) ensures HEART cannot attend to future actions — only past ones.

**Value Added**:
- HEART detects sequential policy violations invisible at the single-action level — e.g., a series of individually permissible data accesses that collectively constitute a prohibited data aggregation pattern.
- Enables stateful guardrails: "HERO has already sent 3 external emails this session" as context for evaluating a 4th.
- `createCausalMask()` already constructed in Gorgonite — direct API-level connection, no new infrastructure.
- Window context length maps to `TransformerConfig.ContextLen`, naturally bounding compute per session.

---

### Option 5 — cl100k Tokeniser Replacing the BPETokenizer Stub

**What**: Swap the non-functional `BPETokenizer` in `main.go` (stubbed `loadVocab()` returning
empty map) for the real tiktoken cl100k_base service already built in Stage 2's `tokenizer.go`.
This is a prerequisite for all text-based guardrail reasoning.

**How**:
1. Extract `2_DATA_ENCODER/pkg/tokenizer` into a shared module under `packages/KNIRVBASE/go/` or import directly.
2. Replace `NewBPETokenizer(vocabPath)` in `main.go` with `tokenizer.New()`.
3. Update `TransformerConfig.VocabSize` from 50257 to 100277 (cl100k actual vocab size).
4. Wire `PrepareDataset()` to call `tokenizer.Encode()`.

**Value Added**:
- Unblocks all text-based guardrail reasoning immediately — the single highest-leverage fix in the codebase.
- cl100k is GPT-4/o-series compatible — BGE-768 embeddings and Gorgonite tokens share vocabulary, enabling hybrid retrieval-augmented guardrail checks (embed the proposed action, retrieve similar past policy decisions, pass as context).
- If HashNetwork is enabled, the same cl100k token IDs define the target classification space for seed training — one tokenisation scheme across all model paths.

---

### Option 6 — Proof-of-Guardrail Audit Trail

**What**: At each guardrail decision, produce a cryptographic commitment: hash the `GuardrailInquiry`
+ `GuardrailDecision` + the client ontology version hash that produced the decision. Store this
audit record in content-addressed storage. For clients requiring compliance attestation, the hash
chain proves that every HERO action was reviewed against the declared ontology version at decision
time.

**How**:
1. After `HEARTService` produces a `GuardrailDecision`, serialise `{inquiry, decision, ontologyVersionHash, modelCheckpointHash, timestamp}`.
2. Compute SHA-256 over the serialisation (via Go's `crypto/sha256` — no special hardware required; if HashNetwork hardware is enabled, optionally route through it for hardware-attested hashing).
3. Append the digest to a per-client append-only audit log at `audits/<clientID>/<sha256hex>.json`.
4. The audit log is content-addressed — identical decisions on identical inputs produce identical hashes, enabling deduplication across multi-agent deployments.

**Value Added**:
- Compliance-ready audit trail: every HERO action is traceable to the specific ontology version and model checkpoint that approved or blocked it.
- `ontologyVersionHash` links each decision to the exact policy document revision — essential for retroactive compliance audits when policies change.
- Block decisions are the highest-priority audit entries — regulators can extract all blocked HERO actions for a time period in O(1) via the audit log index.
- Hardware attestation (optional) provides stronger provenance for regulated industries (finance, healthcare, defence) without requiring hardware at all clients.

---

### Option 7 — Quantised Gorgonite for Edge Guardrail Deployment

**What**: Apply the mapper's `2×int16→uint32` bit-packing scheme to Gorgonite model weights for
deployment in resource-constrained environments — embedded industrial controllers, edge IoT
orchestrators, air-gapped enterprise endpoints — where full float32 inference is not viable.

**How**:
1. Add `QuantiseWeights(model *GPT) map[string][12]uint32` calling `mapper.MapToSlots()` on each weight row.
2. Implement `DequantiseForward()` with fixed-point dot product for INT16 pairs.
3. Cross-compile the inference-only path for the target architecture per the existing `Makefile` pattern.

**Value Added**:
- HEART guardrail enforcement available at the edge — no cloud round-trip for locally-deployed HERO agents.
- Quantisation seed doubles as a weight obfuscation parameter — client model weights are not recoverable without the seed.
- Three deployment tiers: INT16 edge (fast, constrained), float32 server (full), Cerebras WSE2 (high-throughput bulk) — the same three-tier pattern as the original architecture, reapplied to the guardrail domain.

---

### Option 8 — Dynamic Graph NAS for Per-Client Architecture Sizing

**What**: Use `DynamicGraph`'s per-pass graph reconstruction capability to run a lightweight
neural architecture search during client onboarding. Clients with compact, rule-structured
ontologies (insurance, banking, regulatory compliance) get shallow models; clients with nuanced,
contextual ontologies (media, legal, creative) get deeper models. Architecture auto-sizes to the
complexity of the client's domain without manual hyperparameter tuning.

**How**:
1. During the first fine-tuning epoch, record activation variance across transformer blocks via the `DynamicGraph.Operations` log.
2. Feed frequency/variance vectors into `VarianceAnalyzer.Sample()` — high-frequency/low-variance ops are pruning candidates; low-frequency/high-variance ops are duplication candidates.
3. Before the next epoch, modify `dg.Params` to add or remove `TransformerBlock` parameter groups.
4. Depth changes are zero-cost — `DynamicGraph.Forward()` already rebuilds from scratch each pass.

**Value Added**:
- A simple-ontology client gets a HEART model deployable on a low-cost VM; a complex-ontology client automatically receives the depth it requires.
- Fully automated — NAS runs during onboarding, not as a separate training job requiring ML expertise.
- The DynamicGraph's per-pass rebuild limitation becomes its key NAS feature rather than a liability.

---

### Option 9 — Ontology Drift Detection as HEART Self-Monitoring Signal

**What**: Run `VarianceAnalyzer` in streaming mode over BGE embeddings of incoming policy
documents. When the top-24 signal indices shift significantly between ingestion batches, emit an
internal `OntologyDrift` event — the pipeline detects when the client's business context has
materially changed and alerts before HEART operates on stale knowledge.

**How**:
1. Extend `VarianceAnalyzer` with `DeltaSignalIndices(prev []int) float32` measuring Jaccard distance between two ingestion runs' top-24 index sets.
2. When Jaccard delta exceeds threshold (suggest 0.4), emit `GuardrailInquiry{ActionType: "OntologyDrift", ActionContext: json(prevIndices, newIndices)}` internally.
3. HEART classifies the drift severity: minor → append new documents and recompute embeddings; major → trigger full retraining from updated corpus; critical → Block all HERO decisions until retraining completes, notify client administrator.

**Value Added**:
- Prevents HERO from operating under stale guardrails when a client's policies have materially changed.
- `OntologyDrift` events are logged to the audit trail (Option 6) — compliance teams see exactly when and why the guardrail knowledge base was updated.
- Zero new math — `VarianceAnalyzer` already does 95% of the computation; one new method and a threshold comparison.

---

### Option 10 — Unified Checkpoint with Client Ontology Versioning

**What**: Merge Gorgonite model weights, optional HashNetwork seed population, and the client
ontology document manifest (list of ingested documents + their content hashes) into a single
checkpoint envelope. One checkpoint fully describes the guardrail state for a client at a point
in time — enabling rollback, reproduction, and migration.

**How**:
1. Define `ClientCheckpoint{ModelWeights map[string][]float32, SeedPopulation [][]byte, OntologyManifest []PolicyRecord, OntologyVersionHash [32]byte, CheckpointSignature [32]byte}`. `SeedPopulation` is nil when HashNetwork is disabled.
2. On save, serialise all sub-states, compute SHA-256 over the concatenation.
3. Store at `checkpoints/<clientID>/<sha256hex>/checkpoint.bin`.
4. On load, verify `CheckpointSignature` before deserialising any sub-state.

**Value Added**:
- Full guardrail reproducibility: given a checkpoint, any node can exactly reproduce every HEART decision made at that point in time.
- `OntologyVersionHash` enables rollback: if a policy update degrades guardrail quality, restore to the previous checkpoint version with a single operation.
- Content-addressed deduplication: clients with identical policy sets produce identical checkpoint hashes — infrastructure savings for multi-tenant deployments with shared policy frameworks.

---

### Option 11 — NRV Positional Encoding for Policy Document Structure

**What**: Apply the mapper's random projection to sequence positions in the transformer, encoding
position `i` as `mapper.MapToSlots(positionalSinusoids[i])` — 12 uint32 values. Re-enables the
currently disabled sinusoidal `PositionalEncoding` with a hardware-grounded alternative that is
deterministic and reproducible across all HEART instances sharing the same projection seed.

**How**:
1. Generate sinusoidal position vectors of length 768 for positions 0..ContextLen.
2. Pass each through `mapper.MapToSlots()` → 12 uint32 values.
3. Unpack back to float32 (reverse int16 bit-packing) and add to embedding.
4. Re-enable the `posEnc` addition in `NewGPT()` at line 415.

**Value Added**:
- Re-enables positional encoding with zero new math — fixes a known regression in the current code.
- Mapper's sigmoid+quantisation bounds the positional signal, preventing embedding scale mismatch with client document embeddings.
- Deterministic projection seed means positional encoding is identical across all HEART instances for the same client — consistent guardrail behaviour in horizontally-scaled deployments.

---

### Option 12 — Client Ontology Fine-Tuning for Compiled `rule.wasm` Guardrail Rules

**What**: Use the client's onboarded business data as the supervised fine-tuning corpus for
HEART, teaching it to generate TinyGo source code compiled to a `rule.wasm` binary — a compiled,
executable guardrail rule implementing `Resolve(ctx GuardrailContext) GuardrailDecision`. HERO
executes the WASM module directly for deterministic guardrail enforcement — no LLM interpretation
at enforcement time, no ambiguity post-compilation.

**How**:
1. Map client policy sections to `{PolicySection, ActionType, ExpectedDecision}` training pairs. The `PolicySection` text becomes a TinyGo function body implementing `Resolve(ctx GuardrailContext) GuardrailDecision`, encoding the policy logic as executable code. Each compiled rule is exact, testable, and independently auditable.
2. Add a **rule compilation pipeline** to `HEARTService` as a post-generation step after Stage 4 (Option 19):
   - Gorgonite emits TinyGo source via `GuardrailResponse.RuleSource string`.
   - `exec.Command("tinygo", "build", "-o", "rule.wasm", "-target", "wasm", srcPath)` compiles — the same `exec.Command` pattern already established by `CerebrasBridge.RunInference()`.
   - `wazero` (pure-Go WASM runtime, zero CGo) executes the compiled binary against bidirectional verifier test cases (Option 21) to confirm runtime correctness before deployment.
   - Compilation failure → hard rejection → multi-turn loop (Option 24) retries Stage 4 with `"compilation error: <tinygo stderr>"` as a revision prompt.
3. Content-address the compiled `rule.wasm` via SHA-256 (Go `crypto/sha256`; optional hardware attestation if client has HashNetwork enabled): the digest becomes the rule's deployment identifier. Identical rules from different training runs produce identical identifiers — automatic deduplication.
4. Fine-tune Gorgonite on the client corpus after initial pre-training, LR 3e-5. Training objective is compilable, executable TinyGo code implementing the client's policies, not prose.
5. Stage 2 `VarianceAnalyzer` identifies which policy dimensions carry the most signal, enabling corpus pruning to the highest-value training examples.

**Value Added**:
- HERO executes `Resolve(ctx GuardrailContext) GuardrailDecision` with a typed contract — guardrail enforcement is machine-verifiable. Ambiguity is eliminated once the rule binary is deployed.
- WASM sandboxing means a malformed or adversarially-crafted `rule.wasm` cannot escape the HERO runtime — safety is a property of the format, not the generator.
- Compilation failure as a training signal teaches Gorgonite to generate syntactically correct policy-logic code without any manual labelling — the TinyGo compiler is the annotator.
- Rules are version-controlled by their content hash — policy history is auditable at the binary level.

---

## ES Integration Options (Options 13–17)

*These options apply to the optional HashNetwork training path only. All are no-ops when
`HEARTConfig.UseHashNetwork = false`. Listed in recommended implementation order if HashNetwork
is enabled for a client.*

---

### Option 13 — Replace GA Elitism with ES Weighted Update (HashNetwork Only)

**What**: Replace the `SelectAndMutate` hard-elitism step in `EvolutionaryHarness` with the ES
weighted perturbation sum: `seed_t ← seed_{t-1} + α · (1/N) · Σ Rₙεₙ`. The reward signal is
classification accuracy on the client's `(concept, guardrailClass)` ontology pairs — not Bitcoin
mining alignment. All N population members contribute to every update (soft), not just the top
25% (hard).

**How**:
1. In `SelectAndMutate`, after `CalculateBitMatchAdvantage` normalises rewards (z-score already implemented correctly), compute the weighted sum: for each perturbation `εₙ = seeds[n] - baseSeed`, accumulate `Rₙ · εₙ`.
2. Update `baseSeed ← baseSeed + α · (1/N) · Σ Rₙεₙ`, clipped to `[0, 255]`.
3. Generate next population by sampling Gaussian noise around the updated `baseSeed` (temperature-scaled byte perturbations) rather than randomly mutating elites.
4. Replace the mining-loop reward with classification accuracy on client ontology `(concept, guardrailClass)` pairs.

**Value Added**:
- ES paper demonstrates 15.5× lower std-dev across runs vs. GA elitism; equivalent improvement for HashNetwork seed convergence on the client's concept taxonomy.
- `CalculateBitMatchAdvantage` already implements the ES reward normalisation step — only the weighted sum update is the missing piece.
- Gives `EvoGRPO` a proper mathematical foundation instead of placeholder logic.

---

### Option 14 — Mirrored Sampling for Population Evaluation (HashNetwork Only)

**What**: For each noise vector ε sampled around `baseSeed`, also evaluate the antithetic pair
`(seed + ε, seed - ε)`. Halves variance at zero additional compute cost.

**How**:
1. In `EvaluatePopulationBatch`, generate N/2 positive perturbations and their mirrors simultaneously.
2. Reward estimate: `(1/N) Σ [R(seed+ε) - R(seed-ε)] · ε` (antithetic estimator).

**Value Added**:
- 50% variance reduction in the gradient estimate for free — same throughput, better signal.
- Recommended as a simultaneous implementation with Option 13.

---

### Option 15 — σ Annealing Schedule Tied to Client Onboarding Phase (HashNetwork Only)

**What**: Replace the binary `StaticMidstate` flag with a continuous σ annealing schedule tied
to the client's onboarding progress. Early phases (bulk ingestion of new client data): large σ
(broad exploration of the concept space). Later phases (incremental policy updates): small σ
(precision refinement around established classifications).

**How**:
1. Compute `σ(epoch) = σ_max · exp(-epoch / τ)` where `τ` is a decay constant.
2. Use `σ(epoch)` as both seed perturbation noise magnitude and the difficulty scaling factor for the training target mask.
3. Replace `StaticMidstate` boolean with `σ_threshold` below which full difficulty engages.
4. Expose `σ_max`, `τ` as `TrainConfig` parameters.

**Value Added**:
- Eliminates the arbitrary binary transition in training dynamics — convergence curves become smooth.
- σ annealing is the mechanism behind ES's 15.5× std-dev advantage; applying it imports that consistency.
- The same σ schedule can govern Gorgonite's learning rate warmup during client onboarding, unifying training tempo across both models.

---

### Option 16 — Complete EvoGRPO with Client Ontology as Reward (HashNetwork Only)

**What**: Implement the `EvoGRPO` struct using Options 13–15 as its mathematical base, with
guardrail classification accuracy on the client's ontology as the fitness function — replacing
stub fitness values with real evaluation against the client's `(concept, expectedGuardrailClass)`
pairs.

**How**:
1. Replace stub fitness in `EvoGRPO` with a call to `EvolutionaryHarness.EvaluatePopulationBatch` using client ontology pairs as the evaluation dataset.
2. Implement crossover as antithetic seed interpolation: `child = α·elite + (1-α)·mirror_elite`.
3. Wire GRPO advantage weighting as the reward normalisation layer, with `CalculateBitMatchAdvantage` as its backing implementation.
4. Expose the ES update rule as the default optimiser; GA elitism as a fallback flag for comparison.

**Value Added**:
- Fulfils `EvoGRPO`'s design intent — it was never meant to hold placeholder logic.
- HashNetwork and Gorgonite share the same client ontology training corpus — one dataset, two architecturally independent model types, cross-validating each other.

---

### Option 17 — HashNetwork Seeds Trained on Client Concept Space (HashNetwork Only)

**What**: Treat all of HashNetwork's seeds (`Seeds1`, `Seeds2`, `SeedsOut`) as a full parameter
vector θ and apply the ES update from Option 13 directly — optimising against the client's
`(business_concept_embedding, guardrailClass)` pairs. This is the deepest HashNetwork
integration: the network learns this specific client's concept taxonomy from scratch.

**How**:
1. Flatten `Seeds1 + Seeds2 + SeedsOut` into a single byte vector θ.
2. Sample N Gaussian perturbations, evaluate each against the client dataset, compute rewards via `CalculateBitMatchAdvantage`.
3. Apply ES weighted update: `θ_t ← θ_{t-1} + α · (1/N) · Σ Rₙεₙ`.
4. Run Gorgonite and HashNetwork training concurrently on the same corpus — `VarianceAnalyzer` compares which concept categories each model classifies most confidently. Route inference to the more confident model per concept class.

**Value Added**:
- One client dataset trains two architecturally independent models simultaneously — cross-validation for free.
- Cases where both models disagree are automatically routed to Escalate — model disagreement is itself the strongest guardrail signal.
- HashNetwork handles high-frequency, routine guardrail checks; Gorgonite handles nuanced edge cases requiring contextual reasoning.

---

## Cognition Options (Options 18–25)

*These options apply the generator-verifier duality, hierarchical reasoning decomposition,
curriculum learning, and bidirectional verification from the Cognition Implementation paper.
Each is independently implementable and does not require HashNetwork.*

---

### Option 18 — Generator-Verifier Duality: HashNetwork (Optional) as Deterministic Critic

**What**: When HashNetwork is enabled, use `RecursiveEngine`'s 21-pass consensus as a fast
verifier for every Gorgonite-generated guardrail decision. Gorgonite generates → HashNetwork
scores → if confidence < 0.85, route to multi-turn loop (Option 24). When HashNetwork is
disabled, verification relies solely on the bidirectional agents (Option 21).

**How**:
1. After `HEARTService` produces a `GuardrailDecision`, tokenise the decision summary and feed into `RecursiveEngine`'s 21-pass loop.
2. `ConsensusResult.ClassDistribution[3]` maps to `{Permit, Block, Escalate}`.
3. Add `VerifierConfidence float32` and `HashNetworkVerified bool` to `GuardrailDecision`.
4. When HashNetwork is disabled: `HashNetworkVerified` is omitted; `BidirectionalVerified` (Option 21) is the sole trust signal.

**Cognition paper connection**: The paper identifies that static verifiers outputting uninterpretable scalar scores fail because they rely on superficial pattern matching. HashNetwork is the structural opposite — its `ConsensusResult` is a majority vote across 21 temporally diverse SHA-256 seed evaluations, architecturally independent of Gorgonite's float32 attention. A hallucinated Gorgonite guardrail decision that sounds plausible cannot fool HashNetwork's hash-alignment test on the client's concept space.

**Value Added**:
- Two architecturally independent models cross-validating each guardrail decision — <1 ms when HashNetwork is available.
- `ModelDisagreement: true` is the strongest escalation signal for human review — surfaced automatically when the two models diverge.
- Model disagreement cases feed the training queue directly (Option 23) — the joint uncertainty frontier of both models drives the next retraining cycle.

---

### Option 19 — Hierarchical Guardrail Decomposition: 4-Stage HEART Decision Pipeline

**What**: Replace `HEARTService.processInquiry()`'s monolithic single-shot generation with a
4-stage conditional pipeline. Monolithic generation fails when the proposed action touches
multiple policy domains simultaneously — HEART's `default` fallback in
`generateRecommendedActions()` is the direct symptom of this failure mode.

**How**:
1. **Stage 1 — Action Inquiry**: the raw `GuardrailInquiry` (existing, no change required).
2. **Stage 2 — Core Policy Principles**: `IdentifyRelevantPolicies(inquiry)` — extract which policy sections and guardrail class (Option 22) the proposed action touches, before attempting a decision. Output: `[]PolicyPrinciple{PolicyID, Section, GuardrailClass, Rationale}`.
3. **Stage 3 — Decision Sketch**: generate a structured decision outline conditioned on `PolicyPrinciple` output — a high-level `Permit|Block|Escalate` with rationale, no executable rule source yet.
4. **Stage 4 — Executable Rule Source**: generate TinyGo source implementing `Resolve(ctx GuardrailContext) GuardrailDecision`, conditioned on both the sketch and the core principles. This source is passed to the rule compilation pipeline (Option 12) to produce `rule.wasm`. If TinyGo compilation fails, compiler stderr is fed back as a Stage 4 revision prompt (Option 24 multi-turn loop).

**Cognition paper connection**: The paper's four stages (question → core techniques → proof sketch → full proof) map exactly onto these four stages. The paper's key insight — that naming core techniques explicitly before generating the proof prevents "blind wandering into dead ends" — applies directly: naming the relevant policy principles before generating the decision prevents generating guardrail logic that addresses the wrong policy constraint.

**Value Added**:
- Prevents the exponential probability decay on multi-domain policy conflicts — the primary failure mode of the current single-pass system.
- Stage 3 sketch is surfaced in `GuardrailDecision.Rationale` — human reviewers in Escalate cases see HEART's reasoning structure, not just its verdict.
- Stage 2's policy identification is the natural input to the bidirectional verifier (Option 21) — the verifier checks that the compiled rule is consistent with the named principles, not just the raw inquiry.

---

### Option 20 — Curriculum Training: Apprentice→Journeyman→Expert on Client Data

**What**: Structure the client ontology fine-tuning (Option 12) into three progressive stages
rather than flat supervised fine-tuning. Curriculum-trained small models match models several
times larger trained without it — critical for edge guardrail deployment (Option 7).

**How**:
1. **Apprentice stage**: fine-tune on raw `(ActionType, PolicySection, Decision)` pairs. Objective: learn the vocabulary and syntax of the client's domain. LR: 3e-4.
2. **Journeyman stage**: fine-tune on `(ActionType, PolicySection, DecisionSketch, Decision)` triples — model must generate the decision sketch before the final verdict. Objective: internalise that policy principles drive decisions, not the reverse. LR: 1e-4.
3. **Expert stage**: fine-tune on the full 4-stage hierarchy from Option 19 — action inquiry → core principles → sketch → full `rule.wasm` TinyGo source. Objective: teach HEART to identify pivotal policy intersections and govern the entire decision chain. LR: 3e-5.

**Cognition paper connection**: The paper's three stages — apprentice (syntax), journeyman (structure), expert (insight extraction) — directly produce the three decoupled learning objectives that prevent memorisation. The expert stage is the analogue of teaching an LLM to identify core proof techniques, and the paper shows this generalises to out-of-distribution problems far better than flat fine-tuning.

**Value Added**:
- Three curriculum checkpoints at increasing capability: apprentice model for edge deployment, expert model for server deployment with full Cerebras acceleration.
- Prevents memorisation of client policy examples — the journeyman stage forces structural understanding before the expert stage allows full rule generation.
- If HashNetwork is enabled, σ annealing schedule (Option 15) maps directly: apprentice = high σ (broad exploration), expert = low σ (precision refinement) — unified training tempo across both models.

---

### Option 21 — Bidirectional `rule.wasm` Verification via Policy Checker + Python Subprocess

**What**: Add forward and backward verification agents to `HEARTService`. A compiled `rule.wasm`
must pass both directions before being deployed as an active guardrail — preventing HERO from
being governed by rules that contradict the policies they purport to implement.

**How**:
1. **Forward agent** — `ForwardVerify(inquiry GuardrailInquiry, rule string) (bool, string, error)`: given the action proposal (premises) and compiled `rule.wasm` (conclusion), verify the rule logic correctly enforces the stated policy constraint. Python subprocess call (`exec.Command("python3", "verifier.py", ...)`) — JSON in, JSON out, same pattern as `CerebrasBridge.RunInference()`.
2. **Backward agent** — `BackwardVerify(inquiry GuardrailInquiry, rule string) (bool, string, error)`: given the `rule.wasm` conclusion, reverse-trace to verify it is consistent with the client's declared ontology — does the rule's enforcement logic match the policy section it claims to implement? Structured constraint check, not an LLM call.
3. **wazero runtime execution** — before either agent runs, execute `rule.wasm` against a set of sample `GuardrailContext` inputs via `wazero` (pure-Go WASM runtime, zero CGo). `wazero.Compile()` + `module.ExportedFunction("Resolve").Call()` must succeed — runtime correctness is a prerequisite for logical verification. Catches panics, out-of-memory, and interface violations that static analysis cannot.
4. Add `BidirectionalVerified bool`, `ForwardVerifierMsg string`, `BackwardVerifierMsg string`, `WazeroExecPassed bool` to `GuardrailDecision`. All three gates (wazero, forward, backward) must pass before a rule is deployed to production.

**Cognition paper connection**: The paper's forward agent traces premises→conclusion (sufficiency check); the backward agent traces conclusion→premises (necessity check). Together they perform a bidirectional logical consistency check that eliminates the confirmation bias of a single-direction verifier. The paper uses Python/Lean 4 as deterministic external tools — the Python constraint validator and wazero runtime are structurally identical equivalents for the guardrail domain.

**Value Added**:
- Deployed guardrail rules are formally verified — HERO cannot be blocked or permitted by a rule that contradicts the policy it purports to implement.
- Verification failures surface as `ForwardVerifierMsg`/`BackwardVerifierMsg` — the failure reason is a training signal for the next curriculum stage (Option 20).
- wazero sandboxing ensures verification itself cannot be compromised by a malformed binary.

---

### Option 22 — Three Guardrail Classes as HEART Decision Taxonomy

**What**: Replace hardcoded decision logic with a three-class guardrail taxonomy derived from
the client's ontology structure. Each `rule.wasm` exports its class as a first-class WASM
function — machine-readable without parsing any text.

**How**:
1. Define `GuardrailClass` as a typed enum: `Permit | Block | Escalate`.
   - **Permit** — action is explicitly authorised by the client's ontology. HERO proceeds without human review.
   - **Block** — action is explicitly prohibited. HERO halts; the `rule.wasm` `Enforce()` return carries the specific policy violation and the authoritative policy section.
   - **Escalate** — action is in a grey zone not resolved by existing rules. HERO queues for human approval; the `rule.wasm` carries the relevant conflicting policy sections for the reviewer.
2. `GuardrailClass` is exported as a first-class WASM function: `func GuardrailClass() uint32` returning `0=Permit | 1=Block | 2=Escalate`. HERO calls this export via `wazero` before executing `Enforce()`, enabling class-based routing without parsing any text. TinyGo emits this as `//export GuardrailClass` alongside `//export Enforce`.
3. During expert-stage curriculum training (Option 20), `GuardrailClass` annotation is the first token of Stage 2 output — Gorgonite learns to classify the action's guardrail class before generating the decision rationale.
4. Rule repository indexes by `GuardrailClass` (read via `wazero` at deploy time) — HERO can filter applicable rules by class before executing `Enforce()`.

**Cognition paper connection**: The paper's three insight classes (Construction, Theorem Call, Mathematical Transformation) are annotated into 100K theorem-proof pairs, creating the structured training signal that teaches the LLM to reason about reasoning type before generating the proof. KNIRV's equivalent is annotating client policy decisions with `GuardrailClass` — a structurally identical data engineering step that teaches HEART to classify the decision type before generating the enforcement logic.

**Value Added**:
- Human reviewers handling Escalate cases see exactly which policy sections are in tension — the class is machine-readable, the rationale is human-readable.
- Block rules are highest-priority for audit (Option 6) — compliance teams can extract all blocked HERO actions grouped by GuardrailClass with a single index query.
- Class-indexed rule repository enables fast rule lookup without full `Enforce()` execution for high-frequency routine checks.

---

### Option 23 — Entropy-Spike Detection as Ontology Gap Signal

**What**: Monitor Gorgonite's per-token prediction entropy during guardrail decision generation.
When entropy spikes above threshold — the model is maximally uncertain — this signals an
ontology gap: the client's onboarded data does not clearly cover the proposed action type. Route
that inquiry to HashNetwork (if enabled) and flag it for client review as a documentation gap.

**How**:
1. Re-enable softmax in `SelfAttention.Forward` (prerequisite). Per-token entropy: `H = -Σ p·log(p)` over the vocabulary.
2. During `processInquiry()`, instrument each generation step. When `H > threshold` (suggest 3.0 nats):
   a. **Route to HashNetwork (optional)**: run `RecursiveEngine`'s 21-pass loop; use `ConsensusResult` to select the next decision token if confidence ≥ 0.85. Skip if HashNetwork is disabled.
   b. **Flag as ontology gap**: add `(inquiry, spike_position, entropy_value)` to a `GapQueue chan OntologyGap`. A background goroutine batches these for client notification: "HEART encountered a policy area not clearly covered by your current onboarded documents. Consider adding documentation on: [action_type]."
3. High-entropy cases are sorted by spike magnitude — highest-uncertainty action types are surfaced to the client first.

**Cognition paper connection**: The paper identifies entropy spikes as the signal that a model is facing a "core technique" — the pivotal conceptual leap where generation either succeeds or collapses. In the guardrail domain, entropy spikes are the signal that HEART is facing a policy gap rather than a technical difficulty — the same uncertainty, a different root cause. The response is the same: flag it, route it, and feed it back into training.

**Value Added**:
- Entropy spikes are actionable business intelligence: they tell the client exactly which action types their current policies do not clearly address, enabling targeted documentation efforts.
- Self-supervising training signal: high-entropy cases that are subsequently resolved via human escalation are fed back into the fine-tuning corpus — HEART learns from its own uncertainty.
- Restoring softmax (prerequisite) also unblocks Options 3 and 11.

---

### Option 24 — Multi-Turn Cognitive Loop for Deliberative Guardrail Enforcement

**What**: Convert `processInquiry()` from a single-pass function to an iterative refinement
loop: generate → verify → revise → re-verify, with early exit when verification passes or
`MaxTurns` is exhausted. Grey-zone Escalate cases receive the most deliberation; clear
Permit/Block cases exit early.

**How**:
1. Wrap the 4-stage pipeline (Option 19) in a loop with configurable `maxTurns` (default 3):
   ```
   for turn := 0; turn < maxTurns; turn++ {
       decision = generateHierarchical(inquiry, priorFailures)
       fwd, fwdMsg = ForwardVerify(inquiry, decision)
       bwd, bwdMsg = BackwardVerify(inquiry, decision)
       if fwd && bwd { return decision with BidirectionalVerified=true }
       priorFailures = append(priorFailures, {fwdMsg, bwdMsg})
   }
   return decision with BidirectionalVerified=false, GuardrailClass=Escalate
   ```
2. On verification failure, regenerate only the failing stage — Stage 3 sketch if backward verification fails, Stage 4 rule body if forward fails. Not the full hierarchy.
3. Add `TurnCount int` and `AverageTurns float64` to `HEARTServiceStats` — rising average turns indicate ontology gaps requiring new client data ingestion.
4. `MaxTurns`-exhausted cases are automatically added to the `GapQueue` (Option 23) and routed to Escalate — the hardest decisions always get human review.

**Cognition paper connection**: The paper describes "multi-turn cognitive looping" as essential for trustworthy verification in critical domains — the verifier is not a single forward pass but a deliberative multi-turn agent that challenges, receives revisions, and re-challenges. The per-stage revision design limits cost: only the failing stage is regenerated, not the full hierarchy.

**Value Added**:
- Turns HEART from a single-pass policy lookup into a deliberative reasoning system with no new model weights.
- `AverageTurns` is a real-time ontology health signal — rising average turns indicate the client needs to add documentation.
- `MaxTurns=1` is a drop-in replacement for the current single-pass behaviour — zero regression risk during rollout.

---

### Option 25 — TRPO Trust Region for EvoGRPO Stability (HashNetwork Only)

**What**: Add a trust-region constraint to the ES weighted update in `EvoGRPO` (Option 13),
preventing any single update from moving the base seed distribution beyond a Hamming distance
budget. Prevents catastrophic seed population collapse during early training when the client
corpus is small. Applies only when HashNetwork is enabled.

**How**:
1. After computing unconstrained ES update `Δseed = α · (1/N) · Σ Rₙεₙ`, compute `d = HammingDistance(baseSeed, baseSeed + Δseed)`.
2. If `d > budget`, scale `α' = α · budget / d` and apply the scaled update. O(n) on byte arrays — computationally negligible.
3. Budget governed by σ annealing (Option 15): `budget(epoch) = budget_max · exp(-epoch / τ)`. Early epochs allow large seed moves (broad exploration); late epochs enforce small moves (precision refinement).
4. Log `TrustRegionViolations uint64` in training stats — a high violation count indicates `σ_max` or `α` are too large.

**Cognition paper connection**: The paper uses TRPO to train the verifier agent, preventing catastrophic policy collapse during RL training. The Hamming distance budget provides the same stability guarantee for the discrete seed space: no single epoch can shift the population's learned classification distribution beyond a recoverable distance.

**Value Added**:
- Prevents training instability that produces high-variance guardrail classification — especially important during initial client onboarding when the training corpus is small.
- Hamming distance constraint is orders of magnitude cheaper than KL-divergence computation — no probability distribution estimation required.
- Completes the full ES training stack: principled update (13) + variance reduction (14) + noise schedule (15) + stability constraint (25).

---

## Implementation Priority Matrix

| Option | Effort | Impact | HashNetwork Dep | Recommended Order |
|--------|--------|--------|-----------------|-------------------|
| 5 — cl100k Tokeniser | Low | High | No | **1st** — unblocks all text reasoning |
| 1 — Client Data Feed | Low | Very High | No | **2nd** — core ingestion pipeline |
| 9 — Ontology Drift Detection | Low | High | No | **3rd** — self-monitoring, near-zero cost |
| 4 — Sliding Window Action History | Low | High | No | **4th** — stateful guardrails |
| 22 — Three Guardrail Classes | Low | Medium | No | **5th** — annotates training data |
| 19 — 4-Stage Hierarchical Decision | Medium | Very High | No | **6th** — core deliberation pipeline |
| 12 — Client Ontology Fine-Tuning | Medium | Very High | No | **7th** — rule.wasm compilation path |
| 21 — Bidirectional Verification | Medium | Very High | No | **8th** — requires Options 12, 19 |
| 24 — Multi-Turn Cognitive Loop | Medium | Very High | No | **9th** — requires Options 19, 21 |
| 20 — Curriculum Training | Medium | High | No | **10th** — requires Options 1, 12, 19 |
| 6 — Proof-of-Guardrail Audit Trail | Low | High | Optional | **11th** — compliance attestation |
| 23 — Entropy-Spike / Ontology Gap | Medium | High | Optional | **12th** — requires softmax re-enabled |
| 18 — Generator-Verifier Duality | Low | Very High | Optional | **13th** — adds fast verify path |
| 2 — Variance-Guided Head Pruning | Medium | Medium | No | **14th** — domain-specialised model |
| 11 — NRV Positional Encoding | Low | Medium | No | **15th** — fixes disabled code |
| 8 — Dynamic NAS per Client | High | Medium | No | **16th** — auto architecture sizing |
| 10 — Unified Checkpoint | Medium | Medium | Optional | **17th** — operational integrity |
| 7 — INT16 Edge Quantisation | High | High | No | **18th** — edge deployment |
| 3 — HashNetwork Fast-Path | Low | High | Required | **19th** — enable after HashNetwork trained |
| 13 — ES Weighted Update | Low | Very High | Required | **20th** — if HashNetwork enabled |
| 14 — Mirrored Sampling | Low | High | Required | **21st** — pair with Option 13 |
| 15 — σ Annealing | Low | Medium | Required | **22nd** — smooth training dynamics |
| 16 — Complete EvoGRPO | Medium | High | Required | **23rd** — fulfils design intent |
| 25 — TRPO Trust Region | Low | High | Required | **24th** — completes ES stack |
| 17 — HashNetwork on Client Concept Space | High | Very High | Required | **25th** — full model unification |

---

## Cross-Cutting Integration Architecture

The complete merged system — all 25 options in production with HashNetwork enabled — produces
the following data flow:

```
Enterprise Client Data                    HERO Agent
(policies, SOPs, contracts,                   │
 ontologies, procedures)                  proposes action
         │                                    │
         ▼ (Opt 1)                            │
Stage 1: Document Parser                      │
(PDF/DOCX/JSON/HTML extraction,               │
 change-detection, PolicyRecord chan)         │
         │                                    │
         ▼ (Opt 5) cl100k                     │
Stage 2: DATA_ENCODER                         │
(tiktoken + BGE-768)                          │
         │                                    │
         ├── SlidingWindows (Opt 4) ──────────► ActionHistoryBuffer
         │                                    │
         ▼ (Opt 9)                            │
VarianceAnalyzer ─ Jaccard drift              │
         │ ─ OntologyDrift event              │
         │                                    │
         │ top-24 signal indices              │
         ├──────────────► (Opt 2) Attention Head Pruning
         │                                    │
         ▼ (Opt 11)                           │
Mapper (768→12 uint32)                        │
         │                                    │
         ├── [Optional] HashNetwork Training (Opts 13–17)
         │   EvolutionaryHarness ← client (concept, guardrailClass) pairs
         │   ES weighted update + mirrored sampling + σ anneal + TRPO
         │   EvoGRPO (ES + GRPO advantage)
         │                                    │
         ▼ (Opt 8) DynamicGraph NAS           │
Gorgonite GPT ◄─────────────────────── GuardrailInquiry
(Opt 20 curriculum-trained:                   │
 apprentice→journeyman→expert)        ┌───────▼───────────────────────────────────┐
float32 weights (Opt 2 pruned)        │              HEARTService                 │
         │                            │                                           │
         │                            │  [Optional] HashNetwork Fast Path (Opt 3) │
         │                            │  confidence ≥ 0.85 → short-circuit        │
         │                            │  confidence < 0.85 → hint to Gorgonite    │
         │                            │                                           │
         │                            │  Opt 19: 4-Stage Hierarchical Decision    │
         │                            │  Stage 1: Action Inquiry                  │
         │                            │  Stage 2: Core Policy Principles (Opt 22) │
         │                            │  Stage 3: Decision Sketch                 │
         │                            │  Stage 4: TinyGo rule source              │
         │                            │                                           │
         │                            │  Opt 24: Multi-Turn Cognitive Loop        │
         │                            │  generate → verify → revise → re-verify   │
         │                            │  MaxTurns-exhausted → Escalate + GapQueue │
         │                            │                                           │
         │                            │  Opt 21: Bidirectional Verification       │
         │                            │  wazero: GuardrailClass() + Enforce()     │
         │                            │  Forward agent (Python subprocess)        │
         │                            │  Backward agent (constraint check)        │
         │                            │                                           │
         │                            │  TinyGo source → tinygo build → rule.wasm │
         │                            │  BidirectionalVerified + WazeroExecPassed │
         │                            └───────┬───────────────────────────────────┘
         │                                    │
         │                            GuardrailDecision
         │                            {Permit | Block | Escalate}
         │                            + rationale + rule.wasm ref
         │                                    │
         ├── INT16 quantise (Opt 7) ──► edge deployment
         ├── NPZ export ──────────────► CerebrasBridge ──► Cerebras WSE2 (optional)
         └── UnifiedCheckpoint (Opt 10, ontology version + model state)
                   │
                   ▼ content-addressed audit store (Opt 6)
         audits/<clientID>/<sha256hex>.json
         (every HERO decision traceable to
          ontology version + model checkpoint)
                                             │
                                    HERO acts / halts / escalates to human
                                    (Opt 23: entropy gaps → GapQueue
                                     → client notified of documentation gaps)
```

**Three inference tiers:**

| Tier | Model | Condition | Latency | Use Case |
|------|-------|-----------|---------|----------|
| Fast | HashNetwork (optional) | confidence ≥ 0.85, client enabled | <1 ms | Routine, high-frequency guardrail checks |
| Standard | Gorgonite float32 | always available | ~200 ms | Multi-policy conflicts, nuanced context |
| Accelerated | Gorgonite via Cerebras WSE2 | optional hardware | ~10 ms at scale | High-throughput bulk guardrail evaluation |

**Two training regimes (when HashNetwork is enabled):**

| Model | Parameter Type | Reward Signal | Trust Region |
|-------|---------------|---------------|--------------|
| HashNetwork | 32-byte seed arrays | Guardrail classification accuracy on client ontology | Hamming distance budget (Opt 25) |
| Gorgonite | float32 weight tensors | Cross-entropy on client policy corpus | σ annealing step size (Opt 15) |

**Three reasoning layers in HEARTService (post Options 18–24):**

| Layer | Mechanism | Latency | Role |
|-------|-----------|---------|------|
| Generation | Gorgonite 4-stage hierarchical (Opt 19) | ~200ms | Produce guardrail decision + rule.wasm draft |
| Verification | wazero + bidirectional forward+backward (Opt 21) | ~50ms | Validate before deployment |
| Deliberation | Multi-turn cognitive loop (Opt 24) | ×turns | Refine on failure; escalate on exhaustion |

**Self-improving ontology coverage loop (post Options 20, 23, 24):**

```
Gorgonite uncertainty (Opt 23 entropy spike on action type X)
    ↓
OntologyGap flagged → client notified: "add documentation on X"
    ↓
Client adds policy documentation → Stage 1 ingestion pipeline
    ↓
Fine-tuning on new examples (Opt 20 curriculum stage)
    ↓
Lower entropy on action type X at next occurrence
    ↓
Coverage gap closed; GapQueue shifts to next uncovered frontier
```
