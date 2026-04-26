# Transformer × Pipeline Merger Opportunities

> Holistic architectural strategy for converging `packages/KNIRVHEART/HEART/go_transformer`
> (Gorgonite GPT / HEARTService) and `packages/KNIRVHASHER/pipeline` (4-stage ASIC
> pipeline + HashNetwork inference system) into a unified, co-evolving compute fabric.
> Each option is independently implementable; taken together they form a coherent whole
> described in the final integration diagram.
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
| **CerebrasBridge** | Exports Gorgonia `float32` weights to NPZ, shells out to `cs_python` for Cerebras WSE2 inference |
| **HEARTService** | HTTP server (`/heart/analyze`, `/heart/health`, `/heart/stats`); receives `HEARTErrorInquiry` from KNIRVCORTEX; synthesises 16-float measurement vectors; routes through `CerebrasBridge` |
| **NetworkMetricsProcessor** | Adaptive Z-score normalisation of KNIRV node telemetry → `HEARTInput` structs |

**Known gaps in current code:**
- Softmax commented out in `SelfAttention.Forward`
- Positional encoding disabled (line 415, `NewGPT`)
- Multi-head concat replaced with head sum (lossy)
- `BPETokenizer` is a complete stub — `loadVocab()` returns empty map, every token encodes as `<UNK>`
- No real dataset consumer — `CreateDummyDataset` is the only data source

---

### KNIRVHASHER — Full Architecture

**Pipeline stages:**

| Stage | Name | Description |
|-------|------|-------------|
| 0 | `DATA_CONNECTOR` | USB→MIPS device driver for Antminer S3, cross-compiled `GOOS=linux GOARCH=mips GOMIPS=softfloat` |
| 1 | `DATA_MINER` | ArXiv API client (XML/Atom), orchestrator with graceful shutdown, checkpoint persistence, configurable GPU/CPU worker pool |
| 2 | `DATA_ENCODER` | tiktoken cl100k_base; deterministic text embedding client (primary); Ollama BGE-768 embedding client (fallback); `VarianceAnalyzer` (top-24 high-variance dims from 768-dim space); `Mapper` (768→24 float32 random projection, sigmoid→int16, bit-packed 2×int16→uint32 → 12 hardware slots); `SlidingWindowGenerator` |
| 3 | `DATA_TRAINER` | Docker/OpenWRT packaged Go trainer; ships to `/dev/bitmain-asic` |

**HashNetwork inference engine (`pkg/hashing/neural/`):**
- A 3-layer neural network whose **parameters are 32-byte seed arrays**, not float tensors
- Forward pass: `output = SHA-256(input ∥ seed) → normalised float [0, 1]`
- No matrix multiplication — computation is entirely cryptographic hashing
- Parameters: `Seeds1`, `Seeds2`, `SeedsOut`

**RecursiveEngine (`pkg/hashing/inference/recursive.go`):**
- 21 temporal forward passes over the same input
- Each pass: input jitter → HashNetwork → prediction + confidence
- Aggregates via **majority vote consensus** — produces `ConsensusResult` with confidence, vote count, class distribution
- Optional per-pass seed rotation (XOR-based, deterministic)

**JitterEngine (`pkg/hashing/jitter/jitter_engine.go`):**
- 21-pass temporal loop for dynamic associative hashing
- Each pass: Double-SHA256 → extract lookup key → flash-search jitter vector → XOR jitter into Bitcoin header
- `HuntGoldenNonce`: evaluates nonce candidates tracking alignment to `targetTokenID`
- `ComputePassReward`: bit-matching reward per pass (leading zeros in XOR output)
- Alignment ≥ 0.95 = "found" — maps hash output to token space
- **Note**: inference-time only; ES/gradient methods do not apply here

**EvolutionaryHarness (`pipeline/3_DATA_TRAINER/pkg/training/evolutionary.go`):**
- Population of seeds (`SeedPopulation`, `GroupSize=128`)
- `EvaluatePopulationBatch`: builds Bitcoin headers from `FeatureVector + nonce`, runs `Execute21PassLoopBatch`, extracts golden nonces
- Multi-component reward: alignment + stability + format + exact-match bonus
- `CalculateBitMatchAdvantage`: **z-score normalisation of Hamming bit-match scores** (identical to ES Algorithm 1 normalisation step)
- `SelectAndMutate`: GA-style hard elitism (top 25%) + `BitcoinAwareMutate` (bit-flip)
- Dynamic Difficulty Scaling (DDS): progressive target mask 8→32 bits across epochs
- `StaticMidstate`: binary flag that freezes jitter in early generations

**EvoGRPO (`pipeline/3_DATA_TRAINER/internal/evo_grpo/evo_grpo.go`):**
- Placeholder for Evolution Strategies + GRPO fusion
- Population → fitness eval → selection → crossover/mutation loop skeleton
- Currently uses stub fitness values; mathematical foundation not yet implemented

**Hardware:**
- Bitmain Antminer S3 — MIPS AR9330 CPU @ 400 MHz, 61 MB RAM, 32× BM1382 ASIC chips (~500 GH/s SHA-256)
- **BM1382 constraint:** chips are hard-wired for the Bitcoin mining loop (`SHA256(SHA256(header + nonce)) < target`); they do not support arbitrary deterministic hashing. All uses of the ASIC must pack inputs into the 80-byte block header format via the `0x52 (TXTASK)` protocol and interpret returned nonces, not hash digests.
- CUDA path: `pkg/hashing/methods/cuda/`

**Training data:**
- AlpacaDataCleaned — 52K instruction/response pairs + Dolly-15k, GPTeacher, GSM-8k extensions

**ES alignment status (from ES Relativity Report):**
- ✅ Population-based, gradient-free reward optimisation — already implemented
- ✅ Inference-only compute — no backpropagation anywhere in `3_DATA_TRAINER`
- ✅ Z-score reward normalisation (`CalculateBitMatchAdvantage`) — already correct
- ⚠️ Update rule — **GA hard elitism** where ES requires **reward-weighted perturbation sum Σ Rₙεₙ** — highest-value gap to close
- ⚠️ Noise model — bit-flip mutation instead of Gaussian/temperature-scaled byte perturbation
- ⚠️ Population size — 128, reducible to 30–64 after ES update is in place
- ⚠️ `StaticMidstate` — binary flag, should become continuous σ schedule unified with DDS

---

## Architectural Merger Options

Options 1–12 address structural integration of the two systems. Options 13–17 address
applying the ES paper's findings to `EvolutionaryHarness` and `EvoGRPO`, then
propagating those improvements back into the merged architecture. Options 18–25 apply the
generator-verifier duality, hierarchical reasoning decomposition, curriculum learning, and
bidirectional tool-augmented verification techniques from the Cognition Implementation paper.

---

### Option 1 — Live ArXiv Error-Knowledge Feed into HEART

**What**: Wire Stage 1 `DATA_MINER` directly into the HEART training loop. Search ArXiv for categories relevant to AI failure modes (`cs.AI`, `cs.LG`, `cs.SE`, `cs.CR`) and continuously inject paper abstracts as synthetic error context into `HEARTErrorInquiry` structs, using them as unsupervised pre-training signal for the Gorgonite model.

**How**:
1. Extend `paper_manager.go` to emit a `chan PaperRecord` instead of writing to disk.
2. Add a `HEARTTrainingAdapter` that converts `ArxivEntry.Summary + Categories` into `HEARTErrorInquiry` with `ErrorType` mapped from arXiv category and `ErrorMessage` from the abstract.
3. Feed the channel into `TrainModel()` replacing `CreateDummyDataset`.

**Value Added**:
- HEART continuously learns from the world's ML/systems research without manual curation.
- Closes the loop between the network's live errors (CORTEX→HEART) and their theoretical underpinnings (ArXiv→HEART).
- Transforms Stage 1 from a batch ETL tool into a streaming knowledge ingestion engine.
- Enables HEART to surface citations ("similar error found in arXiv:2301.07041") in `SimilarErrors` responses.
- The same ArXiv feed can also supply `(abstract, category)` pairs as training records for `EvolutionaryHarness.EvaluatePopulationBatch` — one pipeline feeding two fundamentally different model types.

---

### Option 2 — Variance-Guided Attention Head Pruning

**What**: Run Stage 2's `VarianceAnalyzer` over the HEART transformer's internal attention weight distributions during a warm-up pass, then prune heads whose output variance falls below a threshold. Dynamically re-wire the `MultiHeadAttention` block to use only the surviving heads.

**How**:
1. After `TrainModel()` epoch 0, iterate over `block.attention.heads` and record the variance of each head's output matrix.
2. Feed those per-head variance vectors into `VarianceAnalyzer.Sample()` and call `Calculate()`.
3. Drop heads not in `GetSignalIndices()` — set their `wQuery/wKey/wValue` to zero tensors and skip in `Forward()`.
4. Reinitialise surviving heads with Glorot, resume training.

**ES connection**: `CalculateBitMatchAdvantage` in `EvolutionaryHarness` and `VarianceAnalyzer.Calculate()` both perform z-score normalisation over a population. The same normalisation primitive governs which seeds survive in `EvolutionaryHarness` and which attention heads survive in Gorgonite. A shared `SignalFilter` package would unify both.

**Value Added**:
- Reduces Gorgonite graph size proportional to heads pruned (potentially 50%+ on untrained models).
- Adapts head count to the actual information content of the error distribution HEART is observing.
- `bge_signal_indices.json` persists the pruning decision across restarts — architecture is auditable.
- Extends the KNIRVHASHER dimensionality reduction philosophy (768→24) into HEART's internal structure.

---

### Option 3 — BM1382 ASIC as Deterministic Bucket Generator for Attention LSH

**What**: Replace the learned Q·Kᵀ attention scoring with a hardware LSH step executed on the BM1382 chips. Testing has confirmed the ASIC is **hard-wired for the Bitcoin mining loop** — it finds nonces where `SHA256(SHA256(header + nonce)) < target`; it does not provide arbitrary deterministic hashing (`SHA256(input ∥ seed) → fixed output`). The revised design repurposes the mining hardware's natural operation: by setting a **Difficulty-1 target**, the first valid **nonce** discovered becomes the LSH bucket signature. This preserves the 500 GH/s throughput advantage while working within the ASIC's actual capabilities.

**How**:
1. At inference time, bit-pack each Q and K vector using the existing `mapper.MapToSlots()` (768→12 uint32), then map the resulting 128-bit LSH projection into the 80-byte Bitcoin block header structure required by the `0x52 (TXTASK)` protocol. The dispatcher constructs binary headers that look like valid mining work.
2. Specify a **Nonce Range** (e.g., 0–1,000,000). Within that range the ASIC is deterministic: the same projection data always produces the same **Golden Nonce**, making bucket assignment reproducible across nodes.
3. The discovered nonce (32 bits) is the bucket ID. Queries and keys whose mining tasks yield the same nonce fall in the same attention bucket.
4. **RAM constraint (61 MB):** implement a **Temporal Recursive Algorithm** to handle bucket collisions. When a collision is detected, "mine deeper" by using the previously found nonce as a seed for the next mining task, collecting a sequence of nonces rather than one 128-bit signature. The `LSHIndex` stores 32-bit nonces as keys in a memory-mapped B-tree, keeping the index footprint within the 61 MB limit.
5. Pass the hardware-generated sparse mask into `SelfAttention.Forward(x, mask, training)` — the mask parameter is wired but currently unused because softmax is commented out; this option is the natural motivation to re-enable softmax.

**ES connection**: The BM1382 chips are already the compute substrate for `EvolutionaryHarness.EvaluatePopulationBatch` (which similarly exploits the mining loop to find golden nonces against a difficulty target). The same physical cores serve two roles in a merged system: seed evaluation during training and bucket-ID generation during HEART inference. Both workloads use the mining loop natively; a scheduler can time-share them with no hardware changes.

**Value Added**:
- ~500 GH/s nonce-search throughput enables millions of LSH bucket assignments per second — far beyond MIPS CPU capability.
- Converts sunk hardware cost into a genuine inference accelerator by repurposing the mining hardware's natural state rather than fighting its architecture.
- Attention is deterministically auditable for any given input and nonce range — value for KNIRVCHAIN verification.
- Golden-nonce determinism within a fixed nonce range means bucket assignments are reproducible across all KNIRV nodes running the same projection data.
- Hardware-native sparse attention aligns with Longformer/BigBird/Flash Attention research direction.

---

### Option 4 — Sliding Window Temporal Context for Network Metrics

**What**: Apply the `SlidingWindowGenerator` from Stage 2 to the sequence of `NetworkMetrics` snapshots that `NetworkMetricsProcessor` accumulates in `metricsBuffer`. Each window becomes one transformer input, and HEART attends over a causal history of node health rather than processing a single snapshot in isolation.

**How**:
1. Change `metricsBuffer` from a flat `[][]float32` to a ring buffer keyed by `(nodeID, timeSlice)`.
2. On each call to `ProcessRawMetrics`, generate `SlidingWindow` structs over the flattened token sequence (each "token" is a quantised metric vector index).
3. Embed each window's context tokens via a lightweight linear layer (or reuse the existing `EmbeddingLayer` with metric-index vocabulary), then pass through the transformer blocks.
4. The target is the next metric vector — HEART does next-state prediction, not just pattern classification.

**ES connection**: The JitterEngine's 21-pass loop achieves a related effect via temporal averaging of hash passes. Both mechanisms address the same problem — making discontinuous reward signals tractable — but at different layers. Combining sliding window context (training) with 21-pass consensus (inference) gives HEART both historical trajectory and point-in-time confidence simultaneously.

**Value Added**:
- HEART predicts imminent failures rather than only classifying current ones — reactive → predictive.
- `createCausalMask()` already constructed in Gorgonite aligns perfectly with temporal window ordering.
- Stage 2 sliding window logic reused with zero new abstraction — direct API-level connection.
- Window context length maps to HEART's `ContextLen`, naturally bounding compute.

---

### Option 5 — cl100k Tokeniser Replacing the BPETokenizer Stub

**What**: Swap the non-functional `BPETokenizer` in `main.go` (which has a stubbed `loadVocab()` returning an empty map) for the real tiktoken cl100k_base service already built and tested in Stage 2's `tokenizer.go`.

**How**:
1. Extract `2_DATA_ENCODER/pkg/tokenizer` into a shared module under `packages/KNIRVBASE/go/` or import it directly.
2. Replace `NewBPETokenizer(vocabPath)` in the transformer's `main.go` with `tokenizer.New()`.
3. Update `TransformerConfig.VocabSize` from 50257 to 100277 (cl100k actual vocab size).
4. Wire `PrepareDataset()` to call `tokenizer.Encode()` — one-line change.

**ES connection**: Both systems are currently blocked from using real text data. `EvolutionaryHarness` needs tokenised training records; Gorgonite needs a working tokeniser. Fixing this unblocks both simultaneously. The same cl100k token IDs that feed Gorgonite's embedding layer also define the `targetTokenID` space that `HuntGoldenNonce` hunts for in the JitterEngine — a shared semantic token space bridging both model types.

**Value Added**:
- Unblocks actual text pre-training in Gorgonite immediately.
- Both systems share one tokenisation scheme — ArXiv text mined in Stage 1 consumed by both without re-encoding.
- cl100k is GPT-4/o-series compatible — Ollama BGE embeddings and Gorgonite tokens share vocabulary, enabling hybrid retrieval-augmented inference.

---

### Option 6 — ASIC-Anchored Proof-of-Training for KNIRVCHAIN

**What**: At the end of each training epoch — for both Gorgonite weight updates and `EvolutionaryHarness` seed population convergence — use the BM1382 chips to compute a SHA-256 Merkle root over the full parameter state. Commit this hash to KNIRVCHAIN as a `SkillNode` property.

**How**:
1. After `SaveModel()` serialises Gorgonite weights, and after `SelectAndMutate()` finalises the seed population, stream both state blobs in 64-byte blocks to the ASIC via the Stage 0 connector.
2. ASIC returns SHA-256 per block; a software Merkle tree produces a root hash.
3. Broadcast `{root_hash, epoch, model_config_hash, seed_population_hash}` to KNIRVCHAIN on the `ErrorNode → SkillNode` mining path.
4. Any node can re-run training on the same data and challenge the commitment via `integration-tests/modp_formal_verification_test.go`.

**ES connection**: Because `EvolutionaryHarness` is already inference-only (the ES paper's key property), its full state at any epoch is completely determined by the seed population array. A SHA-256 Merkle root over that array is a compact, hardware-generated fingerprint of the entire training history — exactly the primitive needed for decentralised model provenance.

**Value Added**:
- Transforms KNIRVHASHER from a standalone training rig into an on-chain training oracle.
- KNIRVCHAIN miners can weight trust in HEART inference outputs by recency of their proof-of-training commitment.
- Opens decentralised model updates: multiple nodes train and commit proofs; chain resolves conflicts by comparing Merkle roots.
- SHA-256 is the entire KNIRVHASHER computational primitive — the attestation and the model are the same mathematical operation.

---

### Option 7 — Bit-Packed INT16 Weight Quantisation for MIPS Deployment

**What**: Apply the mapper's `2×int16→uint32` bit-packing scheme to Gorgonite model weights for edge deployment on the Antminer's MIPS CPU. The AR9330 at 400 MHz with 61 MB RAM cannot sustain float32 matrix multiply at inference speed, but packed INT16 arithmetic is ~4× cheaper and fits.

**How**:
1. Add `QuantiseWeights(model *GPT) map[string][12]uint32` that calls `mapper.MapToSlots()` on each weight row.
2. Implement a matching `DequantiseForward()` that unpacks int16 pairs and runs fixed-point dot product.
3. Cross-compile the inference-only path (`GOOS=linux GOARCH=mips GOMIPS=softfloat`) per the existing `Makefile` pattern.

**ES connection**: The HashNetwork already runs on the MIPS CPU via the Stage 0 connector. A quantised Gorgonite binary adds a second, complementary inference path on the same device: SHA-256 seed inference (HashNetwork) for fast classification and INT16 transformer inference (Gorgonite) for richer contextual analysis. A routing layer on the MIPS CPU selects the path based on query complexity, creating a two-tier inference system on a single piece of repurposed hardware.

**Value Added**:
- Antminer serves as training accelerator (ASIC), seed-inference host (MIPS+HashNetwork), and transformer-inference host (MIPS+Gorgonite) simultaneously.
- Mapper projection seed doubles as a weight obfuscation parameter — same weights look different under different seeds.
- No cloud dependency at inference time — aligns with D-TEN sovereignty.

---

### Option 8 — Dynamic Graph Architecture Search via Variance Feedback

**What**: Use the `DynamicGraph` wrapper's per-pass graph reconstruction capability to run a lightweight neural architecture search loop: after each training epoch, `VarianceAnalyzer` measures activation variance across transformer blocks, and `DynamicGraph` reconstructs the next epoch's graph with adjusted layer depth and head count.

**How**:
1. After each epoch, hook into `DynamicGraph.Operations` log to record op-type frequency and output variance.
2. Feed frequency vectors into `VarianceAnalyzer.Sample()` — high-frequency/low-variance ops are pruning candidates; low-frequency/high-variance ops are duplication candidates.
3. Before the next epoch's `Forward()`, modify `dg.Params` to add or remove `TransformerBlock` parameter groups.
4. Depth changes are zero-cost — `DynamicGraph.Forward()` already rebuilds from scratch each pass.

**ES connection**: `EvolutionaryHarness` varies seed count via population mutation; this option varies layer count via variance-guided NAS. Both are zeroth-order architecture searches — ES in seed space, NAS in depth/width space. A unified `ArchitectureSignal` struct could drive both: the same fitness landscape that narrows the seed population also narrows the transformer depth, producing co-adapting model pairs.

**Value Added**:
- The DynamicGraph's per-pass rebuild limitation becomes its key NAS feature.
- Model self-organises toward complexity matching the observed error distribution — simple distributions produce shallow, MIPS-deployable models.
- Fully automated model sizing without hyperparameter tuning.

---

### Option 9 — Continuous Embedding Drift Detector as HEART Anomaly Signal

**What**: Run `VarianceAnalyzer` in streaming mode over BGE embeddings produced by Stage 2. When the top-24 signal indices shift significantly between windows, emit a `HEARTErrorInquiry` of type `"DistributionShift"` — the pipeline becomes the anomaly detector for its own training distribution.

**How**:
1. Extend `VarianceAnalyzer` with `DeltaSignalIndices(prev []int) float32` measuring Jaccard distance between two runs' top-24 index sets.
2. Run the analyzer on each `SlidingWindow` batch from Stage 2.
3. When Jaccard delta exceeds threshold (e.g. 0.4), POST `HEARTErrorInquiry{ErrorType: "DistributionShift", ErrorContext: json(prevIndices, newIndices)}` to `/heart/analyze`.
4. HEART classifies the shift and recommends: re-run variance analysis, flush embedding cache, or trigger a checkpoint.

**ES connection**: Distribution shift in embedding space means the `FeatureVector` input to `EvaluatePopulationBatch` has drifted — seeds trained on old distributions should be deprioritised. A `DistributionShift` event can trigger a controlled seed population reset in `EvolutionaryHarness`, resetting `GroupSize` while preserving the top elite seeds, structurally equivalent to an ES σ expansion to re-explore the new landscape.

**Value Added**:
- Closes the observability gap between the data pipeline and both model types.
- `DistributionShift` events committed to KNIRVCHAIN create a timeline of knowledge landscape changes.
- Extremely low implementation cost — `VarianceAnalyzer` already does 95% of the math.

---

### Option 10 — Unified Checkpoint Protocol with Hardware Attestation

**What**: Merge three checkpoint systems — `1_DATA_MINER/internal/checkpoint/checkpoint.go` (paper download state), `SaveModel()`/`LoadModel()` in `main.go` (Gorgonite weights), and `EvolutionaryHarness` seed population state — into a single checkpoint envelope, ASIC-signed.

**How**:
1. Define `UnifiedCheckpoint{ModelWeights map[string][]float32, SeedPopulation [][]byte, MinerState MinerCheckpoint, PipelineEpoch int, ESUpdateRule string, ASICSignature [32]byte}`.
2. On save, serialise all three sub-states, ship concatenation to ASIC for SHA-256, append digest as `ASICSignature`.
3. On load, verify digest before deserialising any sub-state.
4. Store in content-addressed path `checkpoints/<sha256hex>/checkpoint.bin`.

**ES connection**: Adding `SeedPopulation [][]byte` to the checkpoint envelope captures the full `EvolutionaryHarness` state. Under the proposed ES weighted update (Option 13), the base seed vector and current σ must also be checkpointed — the unified format naturally accommodates them as additional fields without breaking the existing Gorgonite weight format.

**Value Added**:
- Eliminates mismatch risk between miner state, Gorgonite weights, and seed populations across restarts.
- Hardware attestation enables checkpoint sharing between KNIRV nodes without trust.
- Content-addressed deduplication across nodes — natural checkpoint diffing.
- ASIC expands from SHA-256 engine to hardware root of trust for the entire training stack.

---

### Option 11 — NRV Encoder as Positional Signal for Attention

**What**: Apply the mapper's random projection matrix to sequence positions in the transformer, encoding position `i` as `mapper.MapToSlots(positionalSinusoids[i])` — 12 uint32 values. Replaces the currently disabled sinusoidal `PositionalEncoding` with a hardware-grounded alternative and re-enables it.

**How**:
1. Generate sinusoidal position vectors of length 768 for positions 0..ContextLen (same formula as `NewPositionalEncoding`).
2. Pass each through `mapper.MapToSlots()` → 12 uint32 values.
3. Unpack back to float32 (reverse int16 bit-packing) and add to embedding.
4. Re-enable the `posEnc` addition in `NewGPT()` at line 415.

**ES connection**: The mapper projection seed is now load-bearing for both positional encoding (Gorgonite) and feature-to-slot mapping (HashNetwork). A seed rotation event — analogous to per-pass seed rotation in `RecursiveEngine` — could cycle positional encodings across HEART inference passes, giving the transformer the same multi-pass temporal diversity the JitterEngine achieves via XOR rotation. Two fundamentally different model types sharing the same seed-rotation primitive.

**Value Added**:
- Re-enables positional encoding with zero new math.
- Mapper's sigmoid+quantisation bounds the positional signal, preventing embedding scale mismatch.
- Deterministic projection seed makes positional encoding reproducible across all nodes.
- Position tokens in 12-slot format can be ASIC-verified in parallel with attention hashing (Option 3).

---

### Option 12 — AlpacaDataCleaned as HEART Fine-Tuning Corpus for `skill.wasm` Compilation

**What**: Use the 52K Alpaca instruction/response pairs as a supervised fine-tuning dataset for HEART, teaching it to generate TinyGo source code that is compiled to a `skill.wasm` binary, content-addressed by the ASIC, and committed to KNIRVCHAIN as a SkillNode. The HERO Model executes the WASM module directly for deterministic capability resolution — no LLM interpretation at execution time, no hallucination risk post-commitment.

**How**:
1. Map Alpaca's `{instruction, input, output}` schema to `HEARTErrorInquiry{ErrorType: instruction, ErrorContext: input}`. The `output` field becomes a TinyGo function body implementing `Resolve(ctx SkillContext) SkillResult`, not prose text. Alpaca's existing instruction-following format maps cleanly: instruction = error type, input = error context, output = resolution logic.
2. Add a **skill compilation pipeline** to `HEARTService.processInquiry()` as a post-generation step after Stage 4 (Option 19):
   - Gorgonite emits TinyGo source via `HEARTHeuristicResponse.SkillSource string`.
   - `exec.Command("tinygo", "build", "-o", "skill.wasm", "-target", "wasm", srcPath)` compiles to WASM — the same `exec.Command` pattern already established in `CerebrasBridge.RunInference()`.
   - `wazero` (pure-Go WASM runtime, zero CGo) executes the compiled binary against the bidirectional verifier's test cases (Option 21) to confirm runtime correctness before any on-chain commitment.
   - Compilation failure is a hard rejection — the multi-turn loop (Option 24) retries Stage 4 with `"compilation error: <tinygo stderr>"` appended as a challenge.
3. Content-address the compiled `skill.wasm` via ASIC SHA-256 (Option 6 mechanism): the digest becomes the SkillNode's on-chain identifier. Identical resolutions on different nodes produce identical binaries and identical identifiers — automatic deduplication.
4. Fine-tune Gorgonite on this corpus after ArXiv pre-training (Option 1), using LR 3e-5 vs 3e-4. The training objective is compilable, executable TinyGo code, not prose.
5. Stage 2 `VarianceAnalyzer` identifies which Alpaca instruction dimensions carry the most signal, enabling corpus pruning to highest-value examples.

**ES connection**: The same Alpaca `{instruction, output}` pairs feed `EvolutionaryHarness.EvaluatePopulationBatch` — tokenised instruction as `FeatureVector`, target output token ID as `targetTokenID`. WASM compilation success/failure becomes an additional binary reward component: `compile_success_bonus` for compilable outputs, penalty for compiler errors. This closes a direct feedback loop between the ES training signal and executable correctness — seeds that consistently produce compilable code are intrinsically preferred.

**Value Added**:
- HERO executes `Resolve(ctx SkillContext) SkillResult` with a typed contract — capability interfaces are machine-verifiable, not human-readable. Execution uncertainty is zero once the binary is committed.
- WASM's sandboxed execution model means a malformed or malicious `skill.wasm` cannot escape the HERO runtime — safety is a property of the format, not the generator.
- Content-addressing by ASIC digest enables SkillNode deduplication across the entire network — two independent nodes discovering the same resolution produce the same on-chain entry.
- Compilation failure as a training signal (via multi-turn rejection) teaches Gorgonite syntactic correctness without any manual labelling — the TinyGo compiler is the annotator.
- AlpacaDataCleaned is already present and licensed — free training signal unused today.

---

## ES Integration Options (Options 13–17)

*These options apply the Evolution Strategies paper (ES Relativity Report) directly to
`EvolutionaryHarness` and `EvoGRPO`, then propagate the improvements through the merged
architecture. They are listed in recommended implementation order.*

---

### Option 13 — Replace GA Elitism with ES Weighted Update

**What**: Replace the `SelectAndMutate` hard-elitism step (keep top 25%, discard rest) in `EvolutionaryHarness` with the ES weighted perturbation sum: `seed_t ← seed_{t-1} + α · (1/N) · Σ Rₙεₙ`. This is the single highest-value change identified in the ES Relativity Report.

**How**:
1. In `SelectAndMutate`, after `CalculateBitMatchAdvantage` normalises rewards (z-score already correct), compute the weighted sum: for each seed perturbation `εₙ = seeds[n] - baseSeed`, accumulate `Rₙ · εₙ`.
2. Update `baseSeed ← baseSeed + α · (1/N) · Σ Rₙεₙ`, clipped to `[0, 255]`.
3. Generate the next population by sampling Gaussian noise around the updated `baseSeed` (temperature-scaled byte perturbations) rather than randomly mutating elites.
4. Fill `EvoGRPO` with this as its mathematical foundation — the struct exists precisely for this purpose.

**Value Added**:
- All N population members contribute to every update (soft), not just the top 25% (hard) — principled gradient estimation with lower variance.
- ES paper demonstrates 15.5× lower std-dev across runs vs. GRPO; equivalent improvement applies here for seed convergence stability.
- `CalculateBitMatchAdvantage` (z-score normalisation) already implements the ES reward normalisation step exactly — the missing piece is only the weighted sum update.
- Gives `EvoGRPO` a proper mathematical foundation instead of placeholder logic.
- **Gorgonite connection**: Once `EvoGRPO` uses the ES update, the Gorgonite training loop can optionally adopt ES as an alternative to the vanilla solver — both models training under the same optimisation algorithm, diverging only in parameter representation (float32 vs 32-byte seed).

---

### Option 14 — Mirrored Sampling for Population Evaluation

**What**: For each noise vector ε sampled around `baseSeed`, also evaluate the antithetic pair: `(seed + ε, seed - ε)`. This halves variance at zero additional ASIC cost and is directly described in the ES paper.

**How**:
1. In `EvaluatePopulationBatch`, generate `N/2` positive perturbations and their mirrors simultaneously.
2. Reward estimate: `(1/N) Σ [R(seed+ε) - R(seed-ε)] · ε` (antithetic estimator).
3. The existing CUDA batch path in `pkg/hashing/methods/cuda/` already evaluates seeds in parallel — mirrored pairs just need to be generated before submission, not after.

**Value Added**:
- 50% variance reduction in the gradient estimate for free — same ASIC throughput, better signal.
- Pairs naturally with the CUDA batch path's parallel header generation.
- Makes the ES update in Option 13 substantially more stable — recommended as a simultaneous implementation.
- **Gorgonite connection**: If Gorgonite adopts ES training (see Option 13 comment), mirrored sampling of weight perturbations around each layer's current values provides the same variance reduction for float32 parameter updates.

---

### Option 15 — Unify DDS with σ Annealing Schedule

**What**: Replace the binary `StaticMidstate` flag with a continuous σ annealing schedule tied to `eh.Epoch`, unified with Dynamic Difficulty Scaling (DDS). Low epochs: large σ (broad Gaussian exploration, easy difficulty). High epochs: small σ (fine local refinement, hard difficulty).

**How**:
1. Compute `σ(epoch) = σ_max · exp(-epoch / τ)` where `τ` is a decay constant.
2. Use `σ(epoch)` as both the noise magnitude for seed perturbations and the difficulty scaling factor for the target mask (currently managed separately by DDS).
3. Remove `StaticMidstate` boolean; replace with `σ_threshold` below which jitter is fully enabled.
4. Expose `σ_max`, `τ` as `TrainConfig` parameters.

**Value Added**:
- Eliminates the arbitrary binary transition in training dynamics — convergence curves become smooth.
- DDS and noise schedule are now one mechanism, not two — fewer knobs, clearer behaviour.
- σ annealing is the mechanism by which ES achieves consistent results across runs (the 15.5× std-dev advantage); applying it here directly imports that consistency.
- **Gorgonite connection**: The same σ schedule can govern Gorgonite's learning rate warmup and decay (`WarmupSteps`, `MaxLR`, `MinLR` in `TrainConfig`), creating a unified training tempo for both models even though their optimisers differ fundamentally.

---

### Option 16 — Complete EvoGRPO with ES Foundation

**What**: Implement the `EvoGRPO` struct (`pipeline/3_DATA_TRAINER/internal/evo_grpo/evo_grpo.go`) properly using Options 13–15 as its mathematical base, replacing placeholder fitness values with the full `CalculateBitMatchAdvantage` → ES weighted update pipeline.

**How**:
1. Replace the stub fitness function in `EvoGRPO` with a call to `EvolutionaryHarness.EvaluatePopulationBatch` — the struct was designed to wrap the harness, not replace it.
2. Implement the crossover step as antithetic seed interpolation: `child = α·elite + (1-α)·mirror_elite` where α is drawn from a Beta distribution.
3. Wire the GRPO advantage weighting (group relative policy optimisation) as the reward normalisation layer, with `CalculateBitMatchAdvantage` as its backing implementation.
4. Expose the ES update rule as the default optimiser, GA elitism as a fallback flag for comparison.

**Value Added**:
- Fulfils the design intent of `EvoGRPO` — it was never meant to hold placeholder logic.
- GRPO advantage weighting + ES weighted update is a strictly stronger combination than either alone: GRPO provides group-relative credit assignment, ES provides principled perturbation-weighted update.
- Once functional, `EvoGRPO` becomes the shared training engine for both the HashNetwork (seed optimisation) and, optionally, Gorgonite (float32 weight perturbation) — one training algorithm for two model types.
- The ES paper notes N=30 suffices for billion-parameter models; reducing `GroupSize` from 128 to 30–64 after this implementation yields equivalent convergence at ~4× lower ASIC batch cost.

---

### Option 17 — ES Applied to HashNetwork Seeds as a Unified Model Optimisation Target

**What**: Treat all of HashNetwork's seeds (`Seeds1`, `Seeds2`, `SeedsOut`) as the full parameter vector θ and apply the ES update from Option 13 directly — optimising the network against a shared dataset of `(input_embedding, target_token_id)` pairs drawn from the same ArXiv/Alpaca corpus as Gorgonite. This is the most ambitious option and the one that most deeply merges both systems.

**How**:
1. Flatten `Seeds1 + Seeds2 + SeedsOut` into a single concatenated byte vector θ.
2. Sample N Gaussian perturbations around θ, evaluate each against the dataset via `Execute21PassLoopBatch`, compute rewards using `CalculateBitMatchAdvantage`.
3. Apply ES weighted update: `θ_t ← θ_{t-1} + α · (1/N) · Σ Rₙεₙ`.
4. The HashNetwork is far smaller than any LLM (hundreds of 32-byte seeds vs. billions of float32 parameters); N=30 converges in this tiny seed space trivially.
5. Run Gorgonite and HashNetwork training concurrently on the same corpus, with the `VarianceAnalyzer` comparing which token predictions each model is most confident on — route inference to whichever model is more confident per token class.

**Value Added**:
- A single dataset powers two fundamentally different neural architectures simultaneously — SHA-256-native computation (HashNetwork) and float32 matrix algebra (Gorgonite).
- HashNetwork fast-path handles high-confidence, low-complexity token classification (near O(1) per token via hash lookup); Gorgonite slow-path handles ambiguous, context-dependent cases.
- The two models cross-validate each other: cases where HashNetwork and Gorgonite disagree are routed to HEART's `/heart/analyze` as `"ModelDisagreement"` error inquiries — the models' disagreement itself becomes a training signal.
- Demonstrates the KNIRVHASHER thesis in its strongest form: SHA-256 ASIC hardware and transformer gradient computation are not competing paradigms but complementary inference regimes on the same knowledge domain.

---

## Cognition Options (Options 18–25)

*These options apply the generator-verifier duality, hierarchical reasoning decomposition,
curriculum learning, and bidirectional tool-augmented verification from the Cognition
Implementation paper (Apr 17 2026). They build on the ES options (13–17) and the
structural integration options (1–12), but each is independently implementable.*

---

### Option 18 — Generator-Verifier Duality: HashNetwork as Deterministic Critic for Gorgonite

**What**: Implement the paper's actor/critic duality by making HashNetwork's 21-pass `RecursiveEngine` the verifier for every Gorgonite-generated `skill.wasm` draft. Currently `processInquiry()` generates a response and returns it with no challenge step. The proposed loop: Gorgonite generates → HashNetwork scores via `ConsensusResult.ConfidenceScore` → if confidence < 0.7, the response is flagged `Unverified` and withheld from KNIRVCHAIN commitment.

**How**:
1. After `HEARTService.processInquiry()` produces a `HEARTHeuristicResponse`, extract the `AnalysisSummary` and tokenise it via cl100k (Option 5).
2. Feed the token sequence into `RecursiveEngine`'s 21-pass loop — the same inference path the HashNetwork already exposes. The resulting `ConsensusResult.ConfidenceScore` is the verifier's verdict.
3. Add `VerifierConfidence float32` and `BidirectionalVerified bool` fields to `HEARTHeuristicResponse`.
4. Only set `BidirectionalVerified = true` when HashNetwork confidence ≥ 0.7 AND the modP checker (Option 21) passes. KNIRVCHAIN SkillNode commits filter on this flag.

**Cognition paper connection**: The paper identifies that static verifiers outputting uninterpretable scalar scores fail because they rely on superficial pattern matching. HashNetwork is the opposite — its `ConsensusResult` is a majority vote across 21 temporally diverse SHA-256 seed evaluations, architecturally independent of Gorgonite's float32 attention. A hallucinated Gorgonite output that sounds plausible cannot fool HashNetwork's hash-alignment test. The two models' computational independence is the correctness guarantee.

**ES connection**: `ConsensusResult.ConfidenceScore` is a population-aggregated signal (21 passes = 21 population members, majority vote = hard elitism). After Option 13 replaces hard elitism with ES weighted update in `EvolutionaryHarness`, the same soft weighting can be applied here — `VerifierConfidence` becomes a weighted average of pass confidences rather than a majority vote, inheriting ES's variance reduction.

**Value Added**:
- Eliminates unverified `skill.wasm` files from KNIRVCHAIN — the chain carries only attested knowledge.
- HashNetwork verification adds <1ms latency (ASIC-speed SHA-256) against Gorgonite's generation time.
- KNIRVCORTEX can filter skill lookups by `BidirectionalVerified` status, creating a two-tier trust model for the knowledge graph.
- The two models cross-validate each other — their disagreement is itself a signal (see Option 23).

---

### Option 19 — Hierarchical Error Decomposition: 4-Stage HEART Resolution Pipeline

**What**: Replace `HEARTService.processInquiry()`'s monolithic single-shot generation with a 4-stage conditional pipeline mirroring the paper's hierarchical proof structure. The paper shows monolithic generation decays exponentially in probability with each "core technique" required — HEART's hardcoded `default` fallback in `generateRecommendedActions()` is the direct symptom of this failure in the current code.

**How**:
1. **Stage 1 — Error Inquiry**: the raw `HEARTErrorInquiry` (already exists, no change).
2. **Stage 2 — Core Error Techniques**: a new `IdentifyCorePatterns(inquiry)` call that extracts which of the three insight classes (Option 22) the error represents and names the pivotal conceptual tools required to resolve it, before attempting any resolution. Output: `[]CoreTechnique{Name, InsightClass, Description}`.
3. **Stage 3 — Resolution Sketch**: generate a structured `RecommendedActions` list conditioned on the `CoreTechnique` output — a high-level pseudocode outline only, no executable source yet.
4. **Stage 4 — Executable Source**: generate TinyGo source implementing `Resolve(ctx SkillContext) SkillResult`, conditioned on both the sketch and the core techniques. This source is passed to the skill compilation pipeline (Option 12) to produce `skill.wasm`. Because each stage is conditioned on all prior stages, internal contradiction is structurally prevented — the compiled binary cannot implement logic that contradicts the sketch it was given. If TinyGo compilation fails, the compiler stderr is fed back as a Stage 4 revision prompt (Option 24 multi-turn loop).

**Cognition paper connection**: The paper's four stages (question → core techniques → proof sketch → full proof) map exactly onto HEART's four stages. The paper's key insight — that naming core techniques explicitly before generating the proof prevents "blind wandering into dead ends" — applies directly to HEART: naming the error's core technique before generating the resolution prevents generating actions that address the wrong root cause.

**ES connection**: The 4-stage pipeline produces 4 checkpoints per inquiry. Stage 2's `CoreTechnique` output is a discrete classification that can be added to the `EvoGRPO` reward signal — seeds that produce outputs matching the correct core technique class receive an additional `exact-match bonus` in `EvolutionaryHarness`'s multi-component reward.

**Value Added**:
- Prevents the exponential probability decay on novel/complex multi-step errors — the primary failure mode of the current single-pass system.
- The sketch (Stage 3) is surfaced in `HEARTHeuristicResponse.DebugInsights`, giving KNIRVCORTEX operators visibility into HEART's reasoning structure, not just its conclusions.
- Stage 2's core technique identification is the natural input to the bidirectional verifier (Option 21) — the verifier checks that the full `skill.wasm` is logically consistent with the named techniques, not the raw inquiry.

---

### Option 20 — Curriculum Training: Apprentice→Journeyman→Expert for Gorgonite

**What**: Structure the AlpacaDataCleaned fine-tuning (Option 12) and ArXiv pre-training (Option 1) into three progressive curriculum stages rather than flat supervised fine-tuning on mixed data. The paper demonstrates curriculum-trained small models match much larger models trained without it — structurally relevant for Gorgonite's MIPS RAM constraint.

**How**:
1. **Apprentice stage**: Fine-tune Gorgonite on raw `(HEARTErrorInquiry, AnalysisSummary)` pairs from AlpacaDataCleaned mapped to error/resolution format. Objective: learn the vocabulary and syntax of error types and resolution language. LR: 3e-4, same as pre-training.
2. **Journeyman stage**: Fine-tune on `(HEARTErrorInquiry, RecommendedActions, AnalysisSummary)` triples — model must generate the action sketch before the summary. Objective: internalise that actions drive analysis, not vice versa. LR: 1e-4.
3. **Expert stage**: Fine-tune on the full 4-stage hierarchy from Option 19 — raw inquiry → core techniques → sketch → full `skill.wasm`. Objective: teach the model to identify pivotal conceptual steps and govern the entire generation chain. LR: 3e-5.

Each stage uses a lower learning rate than the previous, treating the prior stage's weights as a warm start.

**Cognition paper connection**: The paper's three stages — apprentice (syntax), journeyman (structure), expert (insight) — directly produce the three decoupled learning objectives that prevent memorisation. Teaching Gorgonite to identify core error techniques (expert stage) is the analogue of teaching an LLM to identify core proof techniques, and the paper shows this generalises to out-of-distribution problems far better than flat fine-tuning.

**ES connection**: Curriculum stages map to the σ annealing schedule in Option 15. Apprentice = high σ (broad exploration), journeyman = medium σ, expert = low σ (precision refinement). The curriculum advance trigger (move to next stage when validation loss plateaus) can be unified with the σ decay trigger — one schedule governs both training tempo and noise magnitude across both Gorgonite and `EvolutionaryHarness`.

**Value Added**:
- Curriculum-trained small models achieve reasoning quality of models several times larger — critical for MIPS edge deployment (Option 7).
- The three stages naturally produce three model checkpoints at increasing capability levels, enabling selective deployment: apprentice model on MIPS, expert model on Cerebras WSE2.
- Prevents memorisation of AlpacaDataCleaned examples — the journeyman stage forces structural understanding before the expert stage allows full-hierarchy generation.

---

### Option 21 — Bidirectional skill.wasm Verification via modP + Python Subprocess

**What**: Add forward and backward verification agents to `HEARTService`, using the existing modP P language checker as KNIRV's Lean 4 equivalent (deterministic formal tool) and Python subprocess verification in the pattern already established by `CerebrasBridge`. A `skill.wasm` must pass both directions before being committed to KNIRVCHAIN.

**How**:
1. **Forward agent** — `ForwardVerify(inquiry HEARTErrorInquiry, skill string) (bool, string, error)`: given the inquiry (premises) and generated `skill.wasm` (conclusion), verify the resolution steps logically address the identified root cause. Implemented as a Python subprocess call (`exec.Command("python3", "verifier.py", ...)`) in the exact pattern of `CerebrasBridge.RunInference()` — JSON in, JSON out.
2. **Backward agent** — `BackwardVerify(inquiry HEARTErrorInquiry, skill string) (bool, string, error)`: given the `skill.wasm` conclusion, reverse-trace to verify it satisfies the original inquiry's constraints — does the resolution apply to the error type, node ID, and confidence score range specified? This is a structured constraint check, not an LLM call.
3. **wazero runtime execution** — before either verifier agent runs, the compiled `skill.wasm` binary is executed against a set of sample `SkillContext` inputs using `wazero` (pure-Go WASM runtime, zero CGo). If `wazero.Compile()` or `module.ExportedFunction("Resolve").Call()` fails, the binary is rejected immediately — runtime correctness is a prerequisite for logical verification. The `wazero` step catches panics, out-of-memory, and interface violations that static analysis cannot.
4. **modP integration**: Any `skill.wasm` proposing a network topology change, new IBC message type, or node behaviour modification is translated to a P event and validated against `modp/monitors/network_invariants.p` via `bash modp/scripts/run-tests.sh` before the backward agent runs. This is already the purpose of `integration-tests/modp_formal_verification_test.go` — it needs to be wired into `HEARTService` as a pre-commitment gate rather than a post-hoc test.
5. Add `BidirectionalVerified bool`, `ForwardVerifierMsg string`, `BackwardVerifierMsg string`, `WazeroExecPassed bool` to `HEARTHeuristicResponse`. All three gates (wazero, forward, backward) must pass for commitment.

**Cognition paper connection**: The paper's forward agent traces premises→conclusion (sufficiency check); the backward agent traces conclusion→premises (necessity check). Together they perform a bidirectional logical consistency check that eliminates the confirmation bias of a single-direction verifier. The paper uses Python/Lean 4 as the deterministic external tools — KNIRV's modP P checker and a Python constraint validator are the exact structural equivalents.

**ES connection**: The verifier's forward and backward pass structure mirrors ES's mirrored sampling (Option 14): `(seed+ε, seed-ε)` evaluation pairs provide the same variance reduction that `(forward, backward)` verification provides for logical consistency. Both techniques use paired evaluation to reduce uncertainty about a candidate's quality at minimal additional cost.

**Value Added**:
- SkillNodes on KNIRVCHAIN carry a `BidirectionalVerified` flag — formally incorrect network interventions are rejected before propagating to other nodes.
- The modP integration means HEART's outputs are validated against the same formal models that govern the P language correctness proofs in `modp/` — HEART's knowledge and the network's formal specification stay in sync.
- Verification failures are surfaced as `ForwardVerifierMsg`/`BackwardVerifierMsg` in the response — the failure reason is itself a training signal for the next curriculum stage (Option 20).

---

### Option 22 — Three Insight Classes as HEART Error Taxonomy

**What**: Replace `identifyErrorPatterns()`'s hardcoded substring matching with a three-class insight taxonomy (Construction / Theorem Call / Mathematical Transformation) learned during curriculum training (Option 20), making each `skill.wasm`'s reasoning type machine-readable and indexable on KNIRVCHAIN.

**How**:
1. Define `InsightClass` as a typed enum: `Construction | TheoremCall | MathematicalTransformation`.
   - **Construction** — introduce new infrastructure (retry wrapper, circuit breaker, fallback endpoint). Root cause: missing capability.
   - **TheoremCall** — invoke an existing `SkillNode` from the KNIRVCHAIN registry that has already solved a similar error. Maps to `findSimilarErrors()` — instead of a hardcoded similarity score, it performs an actual chain lookup.
   - **MathematicalTransformation** — recast the error into a different problem domain where a known solution exists (e.g., transform a consensus timeout into a backpressure/throughput problem, applying network flow analysis rather than retry logic). The hardest class — reserved for novel error types.
2. Add `InsightClass InsightClass` to `ErrorPattern`. Because `skill.wasm` is now a compiled binary (not markdown), `InsightClass` is not stored as front-matter — it is instead **exported as a first-class WASM function**: `func InsightClass() uint32` returning `0=Construction | 1=TheoremCall | 2=MathematicalTransformation`. The HERO Model calls this export via `wazero` before executing `Resolve()`, allowing the chain to query the reasoning type without parsing any text. TinyGo emits this as a `//export InsightClass` directive alongside `//export Resolve`.
3. During expert-stage curriculum training (Option 20), include `InsightClass` annotation in the training target — Gorgonite learns to emit the correct `InsightClass()` return value as the first token of Stage 2 output (the numeric enum value, not a string).
4. KNIRVCHAIN indexes SkillNodes by `InsightClass` (read via `wazero` at commit time), enabling HERO Model to prefer `TheoremCall` skills (proven patterns) over `Construction` or `MathematicalTransformation` (novel approaches) when certainty is required.

**Cognition paper connection**: The paper's three classes (Construction, Theorem Call, Mathematical Transformation) are annotated into 100K theorem-proof pairs, creating the structured training signal that teaches the LLM to reason about reasoning. KNIRV's equivalent is annotating 52K Alpaca error-resolution pairs (plus ArXiv abstracts) with `InsightClass` — a structurally identical data engineering step.

**ES connection**: `InsightClass` classification accuracy becomes an additional reward dimension in `EvolutionaryHarness`'s multi-component reward function (currently: alignment + stability + format + exact-match). Seeds that correctly predict the insight class of a training example receive an `insight_class_bonus`, directly connecting the paper's taxonomy to the ES training signal.

**Value Added**:
- KNIRVCHAIN SkillNode queries gain a new filter dimension — callers can request "only TheoremCall skills" for high-stakes interventions.
- The `MathematicalTransformation` class identifies HEART's most novel outputs — these are the highest-value SkillNodes for distribution to other nodes and the highest-priority cases for human review.
- `InsightClass` annotation makes the reasoning audit trail legible to KNIRVCORTEX operators without reading the full `skill.wasm` body.

---

### Option 23 — Entropy-Spike Detection as Self-Supervised Training Signal

**What**: Monitor Gorgonite's per-token prediction entropy during generation. When entropy spikes above threshold at a generation step — the paper's "core technique moment" where the model is maximally uncertain — route that token to HashNetwork for fast classification and add the entire `HEARTErrorInquiry` to the next `EvoGRPO` training batch as a priority example. HEART's own uncertainty at inference time continuously generates its next training curriculum.

**How**:
1. Re-enable softmax in `SelfAttention.Forward` (prerequisite — also required by Options 3 and 11). Softmax outputs are probability distributions; per-token entropy is `H = -Σ p·log(p)` over the vocabulary.
2. During `processInquiry()`, instrument each generation step to record token entropy. When `H > threshold` (empirically tuned, suggest starting at 3.0 nats):
   a. **Route to HashNetwork**: tokenise the current context, run `RecursiveEngine`'s 21-pass loop, use `ConsensusResult` to select the next token if its `ConfidenceScore > 0.7`. This bridges the entropy spike with HashNetwork's deterministic classification.
   b. **Flag for training**: add `(inquiry, current_context, spike_position)` to a `TrainingQueue chan EntropySpike`. A background goroutine drains this queue into `EvoGRPO`'s next batch.
3. Entropy-spike cases are sorted by spike magnitude before entering the training queue — the highest-uncertainty cases train first. This is the paper's insight class identification made fully automated.

**Cognition paper connection**: The paper identifies entropy spikes as the signal that an LLM is facing a "core technique" — the pivotal conceptual leap where proof generation either succeeds or collapses. The paper's solution is to train the model explicitly on these moments via hierarchical fine-tuning. Option 23 goes further: instead of human annotation identifying these moments offline, Gorgonite identifies them at runtime via its own entropy signal, closing the annotation loop entirely.

**ES connection**: The `TrainingQueue` draining into `EvoGRPO` is a direct connection between Gorgonite's inference uncertainty and the HashNetwork's training curriculum. High-entropy cases that HashNetwork handles successfully (confidence ≥ 0.7) become positive training examples for the seed that handled them. Cases where both models fail become the highest-priority next-epoch seeds. The training distribution perpetually chases the joint uncertainty frontier of both models simultaneously.

**Value Added**:
- The training distribution automatically tracks the model's competence frontier — as HEART learns to handle certain error types, their entropy drops, they stop entering the training queue, and training budget shifts to genuinely novel cases.
- Eliminates the need for human annotation of "hard examples" — the curriculum is self-generating.
- The entropy signal is observable (`HEARTServiceStats.EntropySpikeCounts map[string]uint64`) — operators can see which error types are triggering the most uncertainty and prioritise manual review of those categories.
- Restoring softmax (prerequisite) unblocks Options 3 and 11 simultaneously.

---

### Option 24 — Multi-Turn Cognitive Loop for HEARTService

**What**: Convert `processInquiry()` from a single-pass function to an iterative refinement loop: generate → bidirectional verify → targeted revision → re-verify, with early exit when verification passes or `MaxTurns` is exhausted. Per-stage revision (not full regeneration) keeps the cost proportional to the number of failed stages, not the total response size.

**How**:
1. Wrap the 4-stage pipeline from Option 19 in a loop with `maxTurns int` (default 3, configurable in `HEARTServiceStats`):
   ```
   for turn := 0; turn < maxTurns; turn++ {
       response = generateHierarchical(inquiry, priorFailures)
       fwd, fwdMsg = ForwardVerify(inquiry, response)
       bwd, bwdMsg = BackwardVerify(inquiry, response)
       if fwd && bwd { return response with BidirectionalVerified=true }
       priorFailures = append(priorFailures, {fwdMsg, bwdMsg})
   }
   return response with BidirectionalVerified=false
   ```
2. On verification failure, append `priorFailures` to the Gorgonite context as "challenges" — the model regenerates only the stage that failed (Stage 3 sketch if backward verification fails, Stage 4 body if forward fails), not the entire hierarchy.
3. Add `TurnCount int` and `AverageTurns float64` to `HEARTServiceStats` — the average turn count is a real-time health signal for HEART's knowledge currency.
4. Cases requiring `MaxTurns` before giving up are automatically added to the entropy-spike training queue (Option 23) — max-turn failures are the hardest cases and the highest-priority training examples.

**Cognition paper connection**: The paper describes "multi-turn cognitive looping" as essential for trustworthy verification in critical domains — the verifier is not a single forward pass but a deliberative multi-turn agent that challenges, receives revisions, and re-challenges. The paper notes this "consumes significant computational resources but is essential for trustworthiness." The per-stage revision design limits that cost: only the failing stage is regenerated, not the full hierarchy.

**ES connection**: `MaxTurns` maps directly to ES's population size N — both answer "how many evaluations before accepting the current best." After Options 13 and 15 are implemented, the same σ annealing schedule that governs seed noise can govern `MaxTurns`: early in training (high σ, many candidates evaluated), allow 3 turns; late in training (low σ, precision refinement), reduce to 1. Training tempo and inference deliberation depth become one unified annealing schedule.

**Value Added**:
- Turns HEART from a lookup table with heuristics into a genuine deliberative reasoning system with no new model weights.
- `AverageTurns` in `HEARTServiceStats` is a real-time curriculum health signal — rising average turns indicate the model is encountering out-of-distribution errors and retraining is needed.
- Max-turn failures automatically feed Option 23's training queue — the hardest inference cases become the next training batch, creating a closed self-improvement loop.
- `MaxTurns=1` is a drop-in replacement for the current single-pass behaviour — zero regression risk during rollout.

---

### Option 25 — TRPO Trust Region for EvoGRPO Stability

**What**: Add a trust-region constraint to the ES weighted update in `EvoGRPO` (Option 13), preventing any single update from moving the base seed population's centre of mass beyond a Hamming distance budget. This is the mechanically simple equivalent of TRPO's KL-divergence constraint, adapted for the discrete byte seed space.

**How**:
1. After computing the unconstrained ES update `Δseed = α · (1/N) · Σ Rₙεₙ`, compute the Hamming distance `d = HammingDistance(baseSeed, baseSeed + Δseed)`.
2. If `d > budget`, scale `α` down: `α' = α · budget / d`, then apply the scaled update. This is O(n) on byte arrays — computationally negligible.
3. The Hamming distance budget is governed by the σ annealing schedule from Option 15: `budget(epoch) = budget_max · exp(-epoch / τ)`. Early epochs allow large seed moves (broad exploration); late epochs enforce small moves (precision refinement). The trust region shrinks with σ.
4. Log `TrustRegionViolations uint64` in training stats — a high violation count indicates σ_max or α are too large and should be reduced.

**Cognition paper connection**: The paper uses TRPO to train the verifier agent — TRPO's trust region constraint prevents catastrophic policy collapse during RL training by bounding how far the policy can move in a single update. The Hamming distance budget provides the same guarantee for the discrete seed space: no single epoch can shift the population's learned distribution beyond a recoverable distance. This is particularly important in the early training epochs where the reward landscape is chaotic (low bit-match scores, noisy Hamming distances) — the same regime where TRPO's stability advantages are most pronounced.

**ES connection**: Option 25 completes the full ES training stack alongside Options 13 (weighted update), 14 (mirrored sampling), and 15 (σ annealing). Together they provide: principled gradient estimation (13) + variance reduction (14) + noise schedule (15) + stability constraint (25). This four-component stack is a complete, principled implementation of the ES algorithm adapted for discrete byte seed space, giving `EvoGRPO` a rigorous mathematical foundation rather than placeholder logic.

**Value Added**:
- Prevents the training instability that produces the high std-dev variance the ES paper identifies as GA elitism's primary weakness.
- The Hamming distance constraint is orders of magnitude cheaper than KL-divergence computation — no probability distribution estimation required, just byte XOR and popcount.
- `TrustRegionViolations` is an actionable training health metric — operators can identify when hyperparameters need tuning without inspecting loss curves.
- Combines with Options 13–15 as a simultaneous implementation: all four changes modify `SelectAndMutate` and `EvaluatePopulationBatch` in one coherent refactor rather than four separate PRs.

---

## Implementation Priority Matrix

| Option | Effort | Impact | Novel Factor | Recommended Order |
|--------|--------|--------|--------------|-------------------|
| 5 — cl100k Tokeniser | Low | High | Low | **1st** — unblocks all text training |
| 13 — ES Weighted Update | Low | Very High | High | **2nd** — highest-value ES change |
| 14 — Mirrored Sampling | Low | High | Medium | **3rd** — pair with Option 13 |
| 4 — Sliding Window Metrics | Low | High | Medium | **4th** — enables temporal HEART |
| 15 — σ Annealing / DDS Unification | Low | Medium | Medium | **5th** — smooth training dynamics |
| 22 — Three Insight Classes | Low | Medium | Medium | **6th** — annotates training data, no code changes |
| 25 — TRPO Trust Region | Low | High | Medium | **7th** — completes ES stack with Options 13–15 |
| 1 — ArXiv Live Feed | Medium | High | Medium | **8th** — continuous learning |
| 9 — Embedding Drift Detector | Low | Medium | High | **9th** — zero new math |
| 18 — Generator-Verifier Duality | Low | Very High | High | **10th** — requires Option 5; HashNetwork verifier |
| 19 — 4-Stage Hierarchical Resolution | Medium | Very High | High | **11th** — requires Option 18 |
| 16 — Complete EvoGRPO | Medium | High | High | **12th** — fulfils design intent |
| 20 — Curriculum Training | Medium | High | High | **13th** — requires Options 1, 12, 19 |
| 2 — Variance-Guided Pruning | Medium | Medium | Medium | **14th** — model efficiency |
| 12 — Alpaca Fine-Tuning | Medium | High | Low | **15th** — skill.wasm output; input to Option 20 |
| 23 — Entropy-Spike Training Signal | Medium | High | Very High | **16th** — requires softmax re-enabled |
| 24 — Multi-Turn Cognitive Loop | Medium | Very High | High | **17th** — requires Options 19, 21 |
| 11 — NRV Positional Encoding | Low | Medium | High | **18th** — fixes disabled code |
| 21 — Bidirectional Verification via modP | Medium | Very High | High | **19th** — requires Options 18, 19 |
| 3 — ASIC LSH Attention | High | Very High | Very High | **20th** — hardware-native attention |
| 7 — INT16 MIPS Quantisation | High | High | High | **21st** — edge deployment |
| 6 — Proof-of-Training | Medium | High | Very High | **22nd** — on-chain trust |
| 8 — Dynamic NAS | High | Medium | Very High | **23rd** — self-organising architecture |
| 10 — Unified Checkpoint | Medium | Medium | High | **24th** — operational integrity |
| 17 — ES on HashNetwork Seeds | High | Very High | Very High | **25th** — full model unification |

---

## Cross-Cutting Integration Architecture

The complete merged system — all 25 options in production — produces the following data flow:

```
ArXiv API ──────────────────────────────────────────────────────────────────────────┐
    │                                                                               │
    ▼ (Opt 1)                                                                       │
1_DATA_MINER ──── PaperRecord chan ────────────────────────────────────────────────►│
    │                                                                               │
    ▼ (Opt 5) cl100k tokens shared by both model types                              │
2_DATA_ENCODER (tiktoken cl100k + Ollama BGE-768)                                   │
    │                                                                               │
    ├──── SlidingWindow + embeddings ──────────────────────────────────────────────►│
    │                                                                               │
    ▼ (Opt 9)                                                                       │
VarianceAnalyzer ──── Jaccard drift ──► HEARTErrorInquiry[DistributionShift] ──────►│
    │                                         │                                     │
    │ top-24 signal indices                   │                                     │
    ├────────────────────────────► (Opt 2) Attention Head Pruning                   │
    │                                         │                                     │
    ▼ (Opt 11)                                │                                     │
Mapper (768→12 uint32)                        │      AlpacaDataCleaned (Opt 22      │
    │ bit-packed slots                        │      InsightClass annotations) ────►│
    ├──────────────────────────────────────────────────────────────────────────────►│
    │                                                                               │
    ├──► BM1382 ASIC (Opt 3) ──────────────── LSH attention mask                    │
    │       │                                      │                                │
    │       │ SHA-256 (Opt 6) ────────────────► KNIRVCHAIN SkillNode                │
    │       │                                                                       │
    │       │    EvolutionaryHarness                                                │
    │       │    (Opt 13 ES update, 14 mirrored,                                    │
    │       │     15 σ anneal, 25 TRPO trust region)                                │
    │       └──► SeedPopulation ───────────────────────────────────────────────────►│
    │                   │                                                           │
    │                   ▼ (Opt 16)                                                  │
    │              EvoGRPO (ES weighted update + GRPO advantage)                    │
    │                   │   ▲ priority training records                             │
    │                   │   │ (Opt 23 entropy-spike queue)                          │
    │                   ▼ (Opt 17)              ┌──────────────────────────────────┐
    │              HashNetwork ─── fast path ──►│  HEART Inference Router          │
    │              (Opt 18 verifier critic)     │  (confidence-gated)              │
    ▼ (Opt 4) temporal windows                  │                                  │
SlidingWindowGenerator                          │                                  │
    │                                           │                                  │
    ▼ (Opt 8) DynamicGraph NAS                  │                                  │
Gorgonite GPT                                   │                                  │
    │ (Opt 20 curriculum-trained:               │                                  │
    │  apprentice→journeyman→expert)            │                                  │
    │ float32 weights (Opt 2 pruned)            │                                  │
    │                                           │                                  │
    │  ┌── Opt 19: 4-Stage Hierarchical ────────┤                                  │
    │  │  Stage 1: Error Inquiry                │                                  │
    │  │  Stage 2: Core Techniques (Opt 22)     │                                  │
    │  │  Stage 3: Resolution Sketch            │                                  │
    │  │  Stage 4: Full skill.wasm              │                                  │
    │  └────────────────────────────────────────┤                                  │
    │                                           │                                  │
    │  ┌── Opt 24: Multi-Turn Cognitive Loop    │                                  │
    │  │  generate → verify → revise → re-verify│                                  │
    │  │  max-turn failures → Opt 23 queue      │                                  │
    │  └────────────────────────────────────────┤                                  │
    │                                           │                                  │
    │  ┌── Opt 21: Bidirectional Verification   │                                  │
    │  │  Forward agent (Python subprocess)     │                                  │
    │  │  Backward agent (constraint check)     │                                  │
    │  │  modP formal checker (Lean 4 equiv.)   │                                  │
    │  └────────────────────────────────────────┤                                  │
    │                                           │                                  │
    ├──► INT16 quantise (Opt 7) ──► MIPS binary ──► Antminer CPU ─── slow path ───►│
    ├──► NPZ export ──────────────► CerebrasBridge ──► Cerebras WSE2 ─── bulk ────►│
    └──► UnifiedCheckpoint (Opt 10, ASIC-signed, seeds + weights + miner state)    │
              │                                 └──────────────────────────────────┘
              ▼ content-addressed store                         │
         KNIRVCHAIN                                             │
                                                                ▼
                                                     HEARTService /heart/analyze
                                                     (Opt 23: entropy monitored,
                                                      spikes → EvoGRPO queue)
                                                                │
                                                                ▼
                                                  TinyGo source gen (Opt 12)
                                                  → tinygo build -target wasm
                                                  → skill.wasm binary
                                                  wazero exec: InsightClass() + Resolve() (Opt 22)
                                                  wazero runtime gate → forward/backward verifier (Opt 21)
                                                  BidirectionalVerified + WazeroExecPassed flags
                                                                │
                                                                ▼
                                                     KNIRVCHAIN ErrorNode → SkillNode
                                                     (filtered by InsightClass,
                                                      BidirectionalVerified, Tier)
                                                                │
                                                                ▼
                                                        HERO Model executes skill.wasm
```

**Three inference tiers on one piece of hardware:**

| Tier | Model | Hardware | Latency | Use Case |
|------|-------|----------|---------|----------|
| Fast | HashNetwork (SHA-256 seeds) | BM1382 ASIC / MIPS CPU | <1 ms | High-confidence token classification |
| Medium | Gorgonite INT16 | MIPS CPU (quantised) | ~100 ms | Contextual error analysis, edge nodes |
| Full | Gorgonite float32 | Cerebras WSE2 | ~10 ms at scale | Deep inference, HEART production |

**Two training regimes, one optimisation algorithm (post Options 13–16, 25):**

| Model | Parameter Type | ES Update | Reward Signal | Trust Region |
|-------|---------------|-----------|---------------|--------------|
| HashNetwork | 32-byte seed arrays | `seed_t ← seed_{t-1} + α·Σ Rₙεₙ` | Hamming bit-match alignment | Hamming distance budget (Opt 25) |
| Gorgonite | float32 weight tensors | `θ_t ← θ_{t-1} + α·Σ Rₙεₙ` | Cross-entropy on cl100k tokens | σ annealing step size (Opt 15) |

**Three reasoning layers in HEARTService (post Options 18–24):**

| Layer | Mechanism | Latency | Role |
|-------|-----------|---------|------|
| Generation | Gorgonite 4-stage hierarchical (Opt 19) | ~200ms | Produce `skill.wasm` draft |
| Verification | Bidirectional forward+backward + modP (Opt 21) | ~50ms | Validate before commitment |
| Deliberation | Multi-turn cognitive loop up to MaxTurns (Opt 24) | ×turns | Refine on failure |

**Self-improving curriculum loop (post Options 20, 23, 24):**

```
Inference uncertainty (Opt 23 entropy spike)
    ↓
Priority training record → EvoGRPO queue
    ↓
Next training batch (Opt 20 curriculum stage)
    ↓
Updated Gorgonite weights
    ↓
Lower entropy on previously spiking tokens
    ↓
Training queue shifts to next frontier
```

The Antminer S3's 32 BM1382 chips serve four distinct roles in this architecture, all implemented via the Bitcoin mining loop (`SHA256(SHA256(header + nonce)) < target`) with inputs packed into the 80-byte `0x52 (TXTASK)` header format:
1. **Training** — nonce-search evaluation for `EvolutionaryHarness.EvaluatePopulationBatch`; golden nonces are the reward signal
2. **Attention routing** — deterministic bucket-ID generation for sparse transformer attention (Option 3); Difficulty-1 target, fixed nonce range, golden nonce = bucket key in memory-mapped B-tree; Temporal Recursive Algorithm handles collisions within the 61 MB RAM limit
3. **Attestation** — Merkle root over model states for KNIRVCHAIN proof-of-training (Option 6); state blobs streamed as mining headers, per-block nonces accumulate into the Merkle structure
4. **Verification** — real-time checkpoint signature validation on unified checkpoint load (Option 10); same header packing, deterministic nonce check

None of these roles require hardware modification. All four exploit the single operation the BM1382 natively executes — finding nonces in the Bitcoin mining loop.
