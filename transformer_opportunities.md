# Transformer × Pipeline Merger Opportunities

> Holistic architectural strategy for converging `packages/KNIRVHEART/HEART/go_transformer`
> (Gorgonite GPT / HEARTService) and `packages/KNIRVHASHER/pipeline` (4-stage ASIC
> pipeline + HashNetwork inference system) into a unified, co-evolving compute fabric.
> Each option is independently implementable; taken together they form a coherent whole
> described in the final integration diagram.
>
> Informed by: codebase analysis + `packages/KNIRVHASHER/ES_Relativity_Report.md`
> (Evolution Strategies at Scale, arXiv 2509.24372v2).

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
propagating those improvements back into the merged architecture.

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

### Option 12 — AlpacaDataCleaned as HEART Fine-Tuning Corpus for `skill.md` Generation

**What**: Use the 52K Alpaca instruction/response pairs as a supervised fine-tuning dataset for HEART, teaching it to produce structured `skill.md`-format output from error inquiries. This directly supports the KNIRVCHAIN flow: `ErrorNode → SkillNode` via HERO Model reading `skill.md` files.

**How**:
1. Map Alpaca's `{instruction, input, output}` schema to `HEARTErrorInquiry{ErrorType: instruction, ErrorContext: input}` → `HEARTHeuristicResponse{AnalysisSummary: output}`.
2. Add a `skill.md` rendering step to `HEARTService.processInquiry()`.
3. Fine-tune Gorgonite on this corpus after ArXiv pre-training (Option 1), using LR 3e-5 vs 3e-4.
4. Stage 2 `VarianceAnalyzer` identifies which Alpaca instruction dimensions carry the most signal, enabling corpus pruning to highest-value examples.

**ES connection**: The same Alpaca `{instruction, output}` pairs used to fine-tune Gorgonite can become training records for `EvolutionaryHarness.EvaluatePopulationBatch` — tokenised instruction as `FeatureVector`, correct output token ID as `targetTokenID`. Both model types train on the same corpus in parallel, producing complementary capabilities: Gorgonite generates rich natural-language skill summaries; HashNetwork produces fast binary classification of error type. The two outputs can be fused in HEART's response.

**Value Added**:
- Bridges HEART's heuristic response generation (hardcoded switch statements) to learned skill synthesis.
- AlpacaDataCleaned is already present and licensed — free training signal unused today.
- `skill.md` files produced by HEART become on-chain knowledge assets on KNIRVCHAIN.
- Demonstrates the full pipeline: Antminer hardware trains both models → on-chain skills minted → HERO Model resolves future errors.

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

## Implementation Priority Matrix

| Option | Effort | Impact | Novel Factor | Recommended Order |
|--------|--------|--------|--------------|-------------------|
| 5 — cl100k Tokeniser | Low | High | Low | **1st** — unblocks all text training |
| 13 — ES Weighted Update | Low | Very High | High | **2nd** — highest-value ES change |
| 14 — Mirrored Sampling | Low | High | Medium | **3rd** — pair with Option 13 |
| 4 — Sliding Window Metrics | Low | High | Medium | **4th** — enables temporal HEART |
| 15 — σ Annealing / DDS Unification | Low | Medium | Medium | **5th** — smooth training dynamics |
| 1 — ArXiv Live Feed | Medium | High | Medium | **6th** — continuous learning |
| 9 — Embedding Drift Detector | Low | Medium | High | **7th** — zero new math |
| 16 — Complete EvoGRPO | Medium | High | High | **8th** — fulfils design intent |
| 2 — Variance-Guided Pruning | Medium | Medium | Medium | **9th** — model efficiency |
| 12 — Alpaca Fine-Tuning | Medium | High | Low | **10th** — skill.md output |
| 11 — NRV Positional Encoding | Low | Medium | High | **11th** — fixes disabled code |
| 3 — ASIC LSH Attention | High | Very High | Very High | **12th** — hardware-native attention |
| 7 — INT16 MIPS Quantisation | High | High | High | **13th** — edge deployment |
| 6 — Proof-of-Training | Medium | High | Very High | **14th** — on-chain trust |
| 8 — Dynamic NAS | High | Medium | Very High | **15th** — self-organising architecture |
| 10 — Unified Checkpoint | Medium | Medium | High | **16th** — operational integrity |
| 17 — ES on HashNetwork Seeds | High | Very High | Very High | **17th** — full model unification |

---

## Cross-Cutting Integration Architecture

The complete merged system — all 17 options in production — produces the following data flow:

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
Mapper (768→12 uint32)                        │          AlpacaDataCleaned ────────►│
    │ bit-packed slots                        │                                     │
    ├──────────────────────────────────────────────────────────────────────────────►│
    │                                                                               │
    ├──► BM1382 ASIC (Opt 3) ──────────────── LSH attention mask                    │
    │       │                                      │                                │
    │       │ SHA-256 (Opt 6) ────────────────► KNIRVCHAIN SkillNode                │
    │       │                                                                       │
    │       │    EvolutionaryHarness (Opt 13 ES update, 14 mirrored, 15 σ anneal)   │
    │       └──► SeedPopulation ──────────────────────────────────────────────────► │
    │                   │                                                           │
    │                   ▼ (Opt 16)                                                  │
    │              EvoGRPO (ES weighted update + GRPO advantage)                    │
    │                   │                                                           ▼
    │                   ▼ (Opt 17)              ┌──────────────────────────────────┐
    │              HashNetwork ─── fast path ──►│  HEART Inference Router          │
    │                                           │  (confidence-gated)              │
    ▼ (Opt 4) temporal windows                  │                                  │
SlidingWindowGenerator                          │                                  │
    │                                           │                                  │
    ▼ (Opt 8) DynamicGraph NAS                  │                                  │
Gorgonite GPT (go_transformer)                  │                                  │
    │ float32 weights (Opt 2 pruned)            │                                  │
    ├──► INT16 quantise (Opt 7) ──► MIPS binary ──► Antminer CPU ─── slow path ──► │
    ├──► NPZ export ──────────────► CerebrasBridge ──► Cerebras WSE2 ─── bulk ───► │
    └──► UnifiedCheckpoint (Opt 10, ASIC-signed, seeds + weights + miner state)    │
              │                                 └──────────────────────────────────┘
              ▼ content-addressed store                         │
         KNIRVCHAIN                                             │
                                                                ▼
                                                     HEARTService /heart/analyze
                                                                │
                                                                ▼
                                                      skill.md rendering (Opt 12)
                                                                │
                                                                ▼
                                                     KNIRVCHAIN ErrorNode → SkillNode
                                                                │
                                                                ▼
                                                        HERO Model reads skill.md
```

**Three inference tiers on one piece of hardware:**

| Tier | Model | Hardware | Latency | Use Case |
|------|-------|----------|---------|----------|
| Fast | HashNetwork (SHA-256 seeds) | BM1382 ASIC / MIPS CPU | <1 ms | High-confidence token classification |
| Medium | Gorgonite INT16 | MIPS CPU (quantised) | ~100 ms | Contextual error analysis, edge nodes |
| Full | Gorgonite float32 | Cerebras WSE2 | ~10 ms at scale | Deep inference, HEART production |

**Two training regimes, one optimisation algorithm (post Options 13–16):**

| Model | Parameter Type | ES Update | Reward Signal |
|-------|---------------|-----------|---------------|
| HashNetwork | 32-byte seed arrays | `seed_t ← seed_{t-1} + α·Σ Rₙεₙ` | Hamming bit-match alignment |
| Gorgonite | float32 weight tensors | `θ_t ← θ_{t-1} + α·Σ Rₙεₙ` | Cross-entropy on cl100k tokens |

The Antminer S3's 32 BM1382 chips serve four distinct roles in this architecture, all implemented via the Bitcoin mining loop (`SHA256(SHA256(header + nonce)) < target`) with inputs packed into the 80-byte `0x52 (TXTASK)` header format:
1. **Training** — nonce-search evaluation for `EvolutionaryHarness.EvaluatePopulationBatch`; golden nonces are the reward signal
2. **Attention routing** — deterministic bucket-ID generation for sparse transformer attention (Option 3); Difficulty-1 target, fixed nonce range, golden nonce = bucket key in memory-mapped B-tree; Temporal Recursive Algorithm handles collisions within the 61 MB RAM limit
3. **Attestation** — Merkle root over model states for KNIRVCHAIN proof-of-training (Option 6); state blobs streamed as mining headers, per-block nonces accumulate into the Merkle structure
4. **Verification** — real-time checkpoint signature validation on unified checkpoint load (Option 10); same header packing, deterministic nonce check

None of these roles require hardware modification. All four exploit the single operation the BM1382 natively executes — finding nonces in the Bitcoin mining loop.
