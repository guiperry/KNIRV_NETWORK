# HASHER Assertion-Layer & LM Split — Implementation Plan

**Status:** Phases 0–6 are implemented in the current working tree. Phase 2 carries versioned assertion spans from the encoder through the seeder and commits mining targets to `(context, span)`; Phase 3 makes the byte-hash attestation contract explicit, limits software hashing to development/equivalence use, and scopes seed projections to the LM; Phase 4 replaces the non-functional Gorgonia scaffold (`gpt.go`) with a real, tested, backprop-trained transformer; Phase 5 bridges LM candidates to the versioned attestation ledger; and Phase 6 has traced and recorded the historical infrastructure boundaries that Phase 5 must not assume. This supersedes the earlier draft of this document — every open question there now has an owner decision, recorded in Section 1. What follows is sequenced, file-level implementation work, not a menu of options.

### Phase 4 completion note

The scaffold this replaced looked real (genuine Gorgonia matmuls, softmax, `gorgonia.Grad` calls) but wasn't: `EmbeddingLayer.Forward` ignored its token-ID argument and returned a fixed matrix for any input; `GPT.loss` was `mean(logits)` with no dependency on any target token; `TrainModel`'s loop never read `dataset.inputs`/`targets` into the forward pass; `runGorgoniteInference`'s legacy branch called `hs.gpt.Forward(false)` ignoring the `tokens` parameter entirely. Same "real ops, disconnected from real data" pattern as everything else found this session.

Fixed, in `pkg/hashing/transformer/gpt.go`:
- Real embedding lookup: `one-hot(tokenIDs) x embeddings` matmul, so output genuinely depends on input tokens (`TestGPTForward_DependsOnInput`, `TestGPTForward_VariesByPosition`).
- Real cross-entropy loss against real targets, via `gorgonia.LogSoftMax` — not plain `SoftMax` + `Log`, which produces `-Inf` at underflowed probabilities and `0 * -Inf = NaN` when Hadamard-multiplied against the one-hot target vector.
- FoX-style attention ported into differentiable form: causal-masked softmax multiplied by a fixed temporal-decay matrix (`foxDecayMatrix`), rather than the original hash-based version's unnormalized `exp(scores)` — softmax's built-in numerical stability was judged safer than hand-rolling clamped-exp in a differentiable graph; documented as an intentional adaptation, not full formula fidelity.
- `gorgonia.Grad` ordering bug: it performs symbolic differentiation and must run *before* the `TapeMachine` is constructed (it inserts nodes into the graph); the original scaffold — and this rewrite's first draft — called it after, which left added gradient nodes outside the machine's compiled tape and surfaced as a nil-value panic in the solver step once an actual test exercised it.
- Gradient clipping (`gorgonia.WithClip(1.0)`), without which an early unclipped step reliably drove the loss to `NaN` within ~4 steps on the test model.
- `GPT` no longer holds permanent gorgonia nodes tied to one graph; weights persist as plain `*tensor.Dense` (`namedParams()`), rebuilt into a fresh graph each `Forward`/`TrainStep` call — `SaveModel`/`LoadModel`/`cerebras_bridge.go`'s weight export updated to match.

Verified real, not just non-erroring: `TestGPTTrainStep_LossDecreases` trains 41 steps on a fixed (context, target) pair and asserts the loss actually drops (27.96 → 0.016) *and* is never `NaN`/`Inf` at any step — a plain `>=` comparison against a diverged loss would have passed silently, since IEEE 754 makes every comparison against `NaN` false; the test checks explicitly.

**Update:** `GPT` is now the default inference path in `runGorgoniteInference` — the `InferenceMode == "legacy"` gate (whose polarity was backwards from its name: the hash-seed `UnifiedHasherEngine` ran by default, GPT was what `"legacy"` mode opted into) is removed. GPT runs unconditionally when initialized; `unifiedEngine` is now a plain fallback used only if GPT is nil or its forward pass errors, not a config-selectable mode.

**Not yet done, out of this phase's scope:** training on the real corpus end-to-end (Phase 4 wires the trainer to accept real token sequences via `PrepareDataset`/`TrainModel`; an actual training run against `connector/records.jsonl`'s successor is an operational step, not a code gap); true mini-batched gradient averaging (`TrainModel` takes one SGD step per example, not per batch); bias terms (none of the attention/FFN weights have them, matching the prior scaffold — a reasonable increment, not a correctness gap).

### Runtime validation note

`libcuda_hash.so` is built with the seeder bundle and embedded by
`internal/cli/embedded/binaries.go`. KNIRVSERVER extracts it during
initialization to `/var/lib/knirvserver/bin/libcuda_hash.so`, and pipeline
subprocesses are launched with that directory in `LD_LIBRARY_PATH`. A direct
developer-side `go test` does not inherit the server's subprocess environment;
run CUDA-linked seeder tests as follows:

```bash
LD_LIBRARY_PATH=/var/lib/knirvserver/bin go test ./pkg/training ./pkg/storage
```

An unqualified loader error from direct `go test` means the extracted runtime
directory was not searched; it does not establish that the library is missing.

## 0. What's changing, in one paragraph

`pkg/hashing/transformer` stops trying to make SHA-256 seeds behave like trainable weights — that fights the avalanche effect and can't be optimized with anything but blind search. Instead, the system splits into two independent components: a **Language Model (LM)** with conventional float32 weights trained by ordinary backprop, and an **Attestation Layer** that keeps the ASIC/hash infrastructure but repoints it at proof-of-work-witnessed **atomic assertions** — span-level `(context → fact)` claims retrievable by LSH to ground the LM's output, not to generate it. `pipeline/3_DATA_TRAINER` is being renamed `3_DATA_SEEDER` to match: it seeds assertions, it doesn't train weights anymore.

---

## 1. Decision log

Every decision below was made against the specific finding cited; consult the finding's original section number (carried over from the prior draft) for full context if a decision needs revisiting.

| # | Finding | Decision |
|---|---|---|
| D1 | Hardware and software hash paths compute different functions (weighted-sum vs. hash-of-bytes) | Software path is retired for weight computation entirely (LM doesn't use hash-derived weights). What remains of hashing (attestation) unifies software and hardware on hash-of-concatenated-bytes. Real ASIC hardware is required for attestation to carry production PoW guarantees; a software path may exist for dev/test equivalence only, not as a security-equivalent fallback. |
| D2 | `attnForward` never routes through `HardwareRouter` | Correct as-is, retroactively: FoX attention belongs to the LM only, and the LM is software-only (float math, no hashing, no ASIC). The unused `router` parameter should be removed, not wired up. |
| D3 | `cmd/trainer` / `EvoGRPO` / `UserSecurityGates` / `GateNetwork` are placeholder code presented as a real, running system | Delete `cmd/trainer` (and its otherwise-uncalled dependencies `internal/evo_grpo`, `internal/gates`). |
| D4 | `connector/records.jsonl` looked like "the" raw-text source, but `1_DATA_MAPPER`/`0_DATA_CONNECTOR` support multiple source types | `connector/records.jsonl` is an old, Alpaca-only implementation artifact. All ingested sources need to be normalized into a common Alpaca-shaped schema before mapping/encoding — this is a `0_DATA_CONNECTOR` design task, not just a plumbing fix. |
| D5 | `seed_writes.jsonl` and `weights/layer_N.csv` are two separate, unreconciled artifacts from the same training run | `weights/layer_N.csv` (`CSVStorage`) is a legacy implementation and is not authoritative. `seed_writes.jsonl` (`DualSeedWriter`) is the canonical ledger going forward. |
| D6 | `pkg/hashing/transformer`'s `EvolveSeeds`/`HasherTrainer` is disconnected from `pipeline/3_DATA_SEEDER` | `pipeline/3_DATA_TRAINER` → renamed `3_DATA_SEEDER`. `pkg/hashing/transformer` becomes home to the **authoritative LM trainer** — real backprop (D9), not evolution-strategy search (see D10 — that machinery belongs on the attestation side, using different, already-existing code). |
| D7 | On an assertion-layer miss (LM's candidate isn't in the witnessed-assertion index) | Flag low confidence. Do not block generation. |
| D8 | Assertion granularity — per-token vs. per-span | Re-scope to spans (phrase/clause level), not individual tokens. |
| D9 | Where conventional LM weights live / what trains them | The LM is a small dedicated transformer with an embedding table, trained via ordinary backprop. This is now the LM's *only* optimizer — no evolution-strategy component. |
| D10 | Where the evolution-strategy pattern (population → mutate → score → select) belongs | **Corrected:** attestations, not the LM. It maps cleanly onto PoW nonce mining — a fitness-gated search over a discrete, unlabeled candidate space is exactly what proof-of-work is, and exactly what a gradient can't help with (per the avalanche-effect problem in Section 0). Concretely, this is the **existing, already-implemented** `EvolutionaryHarness` in `3_DATA_SEEDER`'s `pkg/training/evolutionary.go` — not `pkg/hashing/transformer`'s `EvolveSeeds`, which is retired (see Phase 0). See Phase 2 for the mechanical mapping. |

---

## 2. Target architecture

| | Language Model | Attestation Layer |
|---|---|---|
| **Parameters** | Conventional float32 weights + embedding table | 32-byte seeds (unchanged storage shape) |
| **Optimizer** | Backprop only (gradient descent, via the existing `gorgonia`-based `DynamicGraph` — Phase 4) | `EvolutionaryHarness`'s existing (1+1)-style evolution strategy — population of candidate nonces, mutation, fitness, selection — retargeted per Phase 2's new PoW-commitment objective (D10) |
| **Compute path** | Software only, no ASIC/hash dependency (D2) | Hardware-required for production legitimacy; software path for dev/test only (D1) |
| **Job** | Propose candidate spans/tokens conditioned on real context | Verify a candidate against previously-witnessed, cryptographically-proven `(context, assertion)` pairs (D8: span-level) |
| **Lives in** | `pkg/hashing/transformer` (rebuilt) | `pipeline/3_DATA_SEEDER` (renamed) + `pkg/hashing/jitter`/`hardware`/`methods/*` (unchanged location) |
| **On a miss** | N/A | Flag low confidence, don't block (D7); queue for mining |

```
   real corpus (normalized, D4)
          │
          ▼
   1_DATA_MAPPER (tokenize, POS, real token sequence)
          │                                   │
          ▼                                   ▼
   2_DATA_ENCODER                    3_DATA_SEEDER (renamed)
   (context embeddings,              (mines span-level assertions,
    span segmentation)                PoW-witnessed, D8)
          │                                   │
          ▼                                   ▼
   LM (pkg/hashing/transformer,       Attestation ledger (seed_writes.jsonl,
   backprop only, D9)                 canonical per D5, mined by
                                       EvolutionaryHarness, D10)
          │                                   │
          └──────────► candidate (span, ctx) ─┤
                                               ▼
                                    LSH lookup (Phase 6)
                                     hit  → confidence + proof
                                     miss → low-confidence flag (D7),
                                            enqueue for mining
```

---

## 3. Phased implementation plan

### Phase 0 — Housekeeping (no design risk, do first)

1. **Renamed `pipeline/3_DATA_TRAINER` → `pipeline/3_DATA_SEEDER`** (D6). Concretely: directory move; `go.mod` module `github.com/lab/hasher/data-trainer` → `github.com/lab/hasher/data-seeder`; `replace knirvhasher => ../..` line carries over unchanged; `cmd/data-trainer` → `cmd/data-seeder`; `Makefile` target at the (soon-to-be-old) `pipeline/3_DATA_TRAINER` path; any hardcoded `"3_DATA_TRAINER"` path strings (e.g. `cmd/data-trainer/main.go`'s data-path resolution, already seen referencing `knirvserver/knirvhasher/data`); `CLAUDE.md`'s pipeline stage list; README mentions.
2. **Delete `cmd/trainer`, `internal/evo_grpo`, `internal/gates`** (D3). Confirmed via import grep that nothing outside `cmd/trainer/main.go` references `internal/evo_grpo` or `internal/gates` — safe to remove as a unit. Correct the README language that currently describes `UserSecurityGates`/`Evo-GRPO` as if it's a running training system.
3. **Retire `weights/layer_N.csv` writing** (D5). This is **not** just a documentation change: `cmd/data-seeder/main.go:798` (`to.storage.SaveWeights(...)`, via `storage.NewCSVStorage` at line 324) is live, currently-running code. Remove the `CSVStorage` write call from the training loop (or the whole `pkg/storage/csv.go` path if nothing else reads it — confirm via `LoadWeights`/`GetLayerMetadata` callers before deleting outright). Leave existing `weights/*.csv` files on disk as historical record unless the user wants them purged.
4. **Delete or quarantine `pkg/training/evolutionary.go`'s `CalculateReward`** (dead path, hardcoded placeholder header, not called by the live `EvaluatePopulation`/`EvaluatePopulationBatch` path) so it can't be wired in by accident later.
5. **Retire `pkg/hashing/transformer/evolve.go` (`EvolveSeeds`) and `HasherTrainer.Train()`** (D10). Unlike `cmd/trainer` (D3), this was real, working code — not a stub — but it solved a problem that no longer exists: evolving hash-derived `SeedStore` weights, which D9 replaces with real float32 backprop, and which the attestation side doesn't need either since `EvolutionaryHarness` already does that job natively. Delete it rather than leaving it as dead code with a plausible-looking API that invites reuse; if any part of it (e.g. the contrastive fitness idea) is worth keeping, port the *idea*, not the code, into Phase 2's `EvolutionaryHarness` rework.

### Phase 1 — Real, normalized, multi-source text → real token sequences (D4)

1. Audit every source type `0_DATA_CONNECTOR` currently supports (D4 implies more than one; `connector/records.jsonl`'s Alpaca-only shape is the old implementation, not the full picture).
2. Design a canonical Alpaca-shaped intermediate schema (`instruction` / `input` / `output` text) that every source normalizes into before `1_DATA_MAPPER`.
3. **Cross-reference `pipeline/KNIRVHASHER_pipeline_upgrade.md`** — this existing planning document already investigates Stage 0/data-connector structure and a real, confirmed spaCy failure (§5.6, causing Slots 4–5 to go "permanently zero-filled") in detail, with code-line-cited evidence. Reconcile with this plan rather than duplicating the investigation; that document's own warning is directly relevant here: *"If [Syntactic Steering] is missing, the system collapses to random hashing."* That's the same failure mode as the `token_sequence` gap below, from a different angle — both should be fixed together.
4. Thread the real token sequence (tokenized once, in `1_DATA_MAPPER`, from the now-normalized source text) through `2_DATA_ENCODER`'s output schema into `training_frames.json`, replacing `tokenSequenceOrTarget`'s single-token-echo fallback with real data.
5. Update `ComputeContextHash`'s callers to consume real multi-token windows instead of the `[]int32{targetTokenID}` fallback.

**Acceptance check:** sample `training_frames.json` post-fix and confirm `token_sequence` is present and multi-token for records with `context_length > 1`; confirm `ComputeContextHash` output actually varies across different real contexts sharing the same target token (it currently doesn't — that's the bug).

The installed `/var/lib/knirvserver/knirvhasher/data/frames/training_frames.json`
may be a pre-Phase-1 artifact and must not be treated as post-fix evidence; the
encoder must be rerun to regenerate it. The code-level acceptance gate now
covers this invariant with encoder serialization, seeder ingestion, span, and
same-target/different-context commitment/hash regression tests.

### Phase 2 — Redefine the mining objective around spans (D7, D8, D10)

1. **Span segmentation.** Decide the span boundary source — reusing `1_DATA_MAPPER`'s dependency-parse output (already extracted for Slot 5) is the natural candidate before building a separate segmenter.
2. **Extend/replace the ledger schema** (`TrainingRecord`, `seed_writes.jsonl`) to key on span rather than single `target_token_id`. This is a breaking schema change — version it (e.g. `schema_version` field) so historical single-token entries aren't silently misread as spans.
3. **Redefine `calculateAlignmentReward`** (`pipeline/3_DATA_SEEDER/pkg/training/evolutionary.go`) around a real PoW commitment over `(real context embedding, asserted span)`, replacing the current "nonce bits resemble `uint32(targetTokenID)`" objective, which has no dependency on real context and no reason to correlate with meaning. `calculateStabilityReward`/`calculateFormatReward` can likely stay as secondary terms; `calculateAlignmentReward` and `IsWinningSeed`'s difficulty check need rework around the new commitment target.
4. Confirms/depends on Phase 1 landing first — a span-level context key is only meaningful once `ComputeContextHash` sees real tokens.

**D10's evolution-strategy mapping, made concrete.** `EvolutionaryHarness` already implements every piece of a real (1+1)/(μ+λ)-style evolution strategy — this phase retargets what each piece scores against, not the mechanism itself:

| ES concept | Existing implementation | What changes in this phase |
|---|---|---|
| Population | `SeedPopulation` — candidate nonces for one `(context, target)` job | Population is now indexed by span-context, not single token (item 2) |
| Mutation | `BitwiseMutation` / `BitcoinAwareMutate` — perturb nonce bits by advantage-weighted amount | Unchanged mechanism; still mutating nonce bits |
| Fitness | `calculateAlignmentReward` + `calculateStabilityReward` + `calculateFormatReward` → `Reward` | `calculateAlignmentReward` redefined per item 3: PoW commitment over real context + span, not bit-similarity to a raw token integer |
| Selection | `CalculateBitMatchAdvantage` (GRPO-style advantage) → `GetEliteSeeds` | Unchanged mechanism, now selecting against the redefined fitness |
| Output | `best_seed` written to `seed_writes.jsonl` via `DualSeedWriter` | Becomes the PoW witness for a span-level assertion, per D5/D8 |

No new evolutionary machinery needs to be built — the existing loop in `cmd/data-seeder/main.go`'s `trainRecord` (`EvaluatePopulation` per generation, elite selection, mutate, repeat) already does this shape of search. This phase only changes *what `Reward` measures* and *what a population entry represents*.

### Phase 3 — Unify hash semantics, scope FoX to the LM (D1, D2)

1. In whatever hosts the attestation-side hashing after this refactor: make the software fallback compute hash-of-concatenated-bytes identically to the hardware path (`computeBatchHW`'s existing approach), retiring `ProjectSeeds`/`ProjectSeeds2D`/`HashToVocab`'s weighted-sum-via-`SeedToFloat` semantics for anything attestation-related — those functions' original job (standing in for trainable weights) no longer exists once the LM owns real weights.
2. Document explicitly (code comment + `docs/ARCHITECTURE.md` update) that a software-only run of the attestation layer is for development/testing equivalence, not a substitute for real ASIC/hardware PoW guarantees in production.
3. Remove `attnForward`'s unused `router *HardwareRouter` parameter (D2) — it's correct that it's unused now, so stop signaling otherwise. Confirm no caller depends on the parameter's presence before removing.

### Phase 4 — Build the LM (D6, D9)

1. **Investigate reusing `pkg/hashing/transformer/dynamic_graph.go`'s `DynamicGraph` before building a new trainer from scratch.** It already wraps real Gorgonia autodiff: `Backward()` calls `gorgonia.Grad(loss.Value, dg.getLearnableNodes()...)`, runs a `gorgonia.NewTapeMachine`, and stores real gradients into `param.Grad`. This is not a placeholder — it's an unused, working backprop engine sitting in the same package. Confirm its `Forward()`/graph-construction side is equally real (not yet verified in this pass) before committing to it, but this is very likely the fastest path to D9 rather than introducing a new autodiff dependency.
2. Replace `SeedStore`-based Q/K/V/Output/FFN/embedding storage with real `float32` parameter tensors for the LM's forward pass.
3. Build the embedding table and train it via backprop over Phase 1's real token sequences — this is the point where the corpus (`connector/records.jsonl`'s successor, post-D4) becomes actually load-bearing for language modeling, rather than sitting unused as it effectively has been.
4. FoX-style attention (decay chain, sharp softmax weighting) carries over as pure float computation per D2/Phase 3 — the *math* that was implemented for FoX doesn't need to be thrown out, only its weight *substrate*.

### Phase 5 — Retrieval / bridge path (D7)

1. LM emits a candidate `(span, context_embedding)`.
2. Quantize the context embedding into an LSH bucket, reusing `2_DATA_ENCODER`'s Slot 0–3 (BGE-variance) mapping — contingent on Phase 6 item 2 confirming that mapping is real, not stubbed.
3. Look up the bucket against the attestation ledger (`FlashSearcher`-style, contingent on Phase 6 item 1 confirming it's live).
4. Hit → attach a confidence/grounding signal (and the PoW proof, for cross-node verification). Miss → flag low confidence (D7) and enqueue the `(context, span)` pair for mining (feeds Phase 2's `EvolutionaryHarness`) rather than blocking generation.

**Implemented:** `pkg/hashing/transformer/attestation_bridge.go` loads only
schema-v2 `seed_writes.jsonl` entries, indexes them by the Slot 0–3 identity
bucket, and then requires an exact length-prefixed `(context, span)` identity
match inside that bucket before reporting a hit. It rejects malformed or
payload/key-mismatched ledger rows, returns the decoded PoW witness on a hit,
and performs a non-blocking, de-duplicated enqueue on a miss. The bridge's
bucket quantizer is byte-for-byte equivalent to `2_DATA_ENCODER`'s
`TensorPacker` Slot 0–3 `[-1,1] → uint32` quantization and accepts the same
configured signal dimensions. `HEARTService` calls it after the LM proposes
its top next-token span, exposes the grounding result in `WASMDecision`, and
annotates its rationale without changing or blocking the LM output. Reloading
seed data also reloads the attestation index. `attestation_bridge_test.go`
covers hit/proof propagation, low-confidence enqueue behavior, and rejection
of a forged assertion identity.

### Phase 6 — Verification gate before trusting Phase 5's design details

**Complete.** The following was resolved by tracing the live call paths in the
current tree. The decisions are deliberately specific about what is live and
what Phase 5 must not reuse.

1. **Jitter/flash path — live for seeder mining, but not Phase 5 retrieval.**
   `TrainingOrchestrator.initializeComponents` starts `jitter.NewServer` at
   `/tmp/jitter.sock` before `Run`. `trainRecord` calls
   `EvaluatePopulationBatch`, which calls the active `core.HashMethod`'s
   `Execute21PassLoopBatch`; software, CUDA, and ASIC methods delegate that to
   `JitterEngine.Execute21PassLoopBatch`. The engine's default configuration
   enables flash search and specifies that same socket. Each of the 21 passes
   therefore calls the RPC server's `FlashManager.GetAssociativeJitter` when
   the socket is available. If it is unavailable, the engine falls back to its
   local `FlashSearcher` (or a generated default jitter), so an unavailable
   server does not disconnect mining but does remove that knowledge-base
   lookup. This RPC lookup is a legacy jitter input, not an assertion-ledger
   retrieval API; Phase 5 correctly uses its own v2-ledger index and exact
   `(context, span)` identity check instead of reusing it.

2. **Encoder identity slots — real computation, with backend-dependent
   provenance.** `performVarianceAnalysis` samples supplied 768-d embeddings
   or obtains them through `EmbeddingService`, computes per-dimension variance,
   selects the top 24 dimensions, and passes them to `mapper.TensorPacker`.
   `PackFrame` quantizes the first four selected dimensions with the same
   `[-1,1] → uint32` mapping used by `AttestationBridge.Bucket`; the bridge is
   therefore compatible with real encoder slot values. The old label
   “BGE-variance” is conditional, however: BGE is used only for the configured
   Cloudflare backend. The default is the deterministic local
   `text-embedder`, and failed analysis can intentionally use cached/default
   indices. Consequently, a bridge candidate must use the encoder's same
   embedding backend and selected-dimension configuration. A mismatch is
   fail-safe (the exact assertion identity prevents a false hit) but reduces
   retrieval recall; it is not evidence that every deployment used BGE.

3. **`DynamicGraph.Forward()` — not a usable Phase 4 training engine.** It
   reconstructs Gorgonia nodes and operations, but only builds a tape machine;
   it does not execute one. Its `Backward` path then adds gradients after that
   machine has been constructed, the ordering bug that made the old scaffold
   unsafe once exercised. It has no callers outside `dynamic_graph.go`.
   Phase 4's `GPT` intentionally bypasses it, rebuilding and executing its
   complete Gorgonia graph per `Forward`/`TrainStep` and persisting tensor
   values. Do not reuse `DynamicGraph` without a separate repair and tests.

4. **Population fitness — no stale incumbent comparison.** Each generation in
   `trainRecord` invokes `EvaluatePopulation` for the current population.
   `EvaluatePopulationBatch` rebuilds headers, obtains a fresh result for every
   candidate nonce, and recomputes reward and advantage across that generation
   only. `GetEliteSeeds` sorts only that freshly returned slice;
   `SelectAndMutate` carries seed bytes forward but no historical score or
   advantage. The Phase 2 reward redesign can therefore use this selection
   loop without inheriting the retired `EvolveSeeds` stale-baseline defect.

---

## 4. Sequencing

Phase 0 has no dependencies — do it first, it's pure cleanup, and now also covers retiring `EvolveSeeds` (D10). Phase 1 blocks Phase 2 (span-level context is meaningless without real tokens) and blocks Phase 4 (the LM's embedding table needs real token sequences to train against) — it's the single highest-leverage fix in this plan. Phase 3 is independent of everything else and can run in parallel throughout. Phase 5 depends on Phase 2 (ledger schema), Phase 4 (the LM has to exist to emit candidates), and Phase 6's items 1–2 (confirms what retrieval infrastructure is actually live). Phase 6 should be run **before** finalizing Phase 5's design, not after — it's a verification gate, not cleanup, and can start as soon as Phase 4 item 1 needs its `DynamicGraph` question answered, or as soon as Phase 2 item 3 needs its item 4 (fitness-noise inheritance) answered.

```
Phase 0 ──► Phase 1 ──┬──► Phase 2 ─────────────────┐
                       │                             │
                       └──► Phase 4 ─────────────────┤
                               │                     │
       Phase 3 (parallel) ─────┤                     │
                               ▼                     ▼
                           Phase 6 ─────────────► Phase 5
```

---

## 5. Risks

| Risk | Severity | Mitigation |
|------|----------|------------|
| `EvolutionaryHarness`'s selection loop turns out to share `EvolveSeeds`'s stale-incumbent fitness-noise problem | Medium | Phase 6 item 4 verifies this before Phase 2 item 3's reward redefinition ships |
| `DynamicGraph`'s `Forward()` turns out to be as unfinished as `EvoGRPO` was | Medium | Phase 6 item 3 verifies this before Phase 4 commits to reusing it; fallback is a fresh (small-scope) autodiff implementation |
| Phase 1's multi-source normalization (D4) is a bigger effort than it looks — `0_DATA_CONNECTOR` may have per-source logic that doesn't cleanly collapse into one Alpaca shape | Medium | Scope the audit (Phase 1, item 1) before committing to a schema design; reconcile with `KNIRVHASHER_pipeline_upgrade.md` which may have already scoped this |
| Span segmentation (D8) has no existing implementation to lean on if `1_DATA_MAPPER`'s dependency parse isn't suitable | Medium | Confirm parse quality/coverage early in Phase 2, before the ledger schema (item 2) is finalized around it |
| Deleting `weights/layer_N.csv` writes (Phase 0, item 3) removes a working, real, currently-improving fitness signal (`generation`/`fitness_score` tracked per token) before confirming `seed_writes.jsonl` alone is sufficient replacement | Low–Medium | Keep historical files; don't delete `pkg/storage/csv.go` itself until Phase 2's redefined ledger schema is confirmed to cover what it tracked |
| Removing `cmd/trainer`/`evo_grpo`/`gates`/`evolve.go` (Phase 0, items 2 and 5) turns out to have a caller this pass didn't find | Low | Re-run the import grep immediately before deletion, not just once during planning |
