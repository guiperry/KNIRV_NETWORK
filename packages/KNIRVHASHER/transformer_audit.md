# KNIRVHASHER × transformer_implementation.md Audit Report

> **Audited against**: `transformer_implementation.md`
> **Audit date**: 2026-04-27
> **Package**: `packages/KNIRVHASHER/pkg/hashing/transformer/`
> **Note**: The default hash path is `neural.HashNetwork` + `inference.RecursiveEngine`, wired in `cmd/driver/hasher-host/main.go`. `hashnet_wrapper.go` is a legacy stub — not the active path.

---

## Executive Summary

The transformer package has **Phase 0, 4, 5, 9 fully complete**. Phase 2-3 are mostly complete. Phases 8, 11, 12, 13 have significant gaps. The `HasherTransformer` (hash-seed, hardware-accelerated) lives in `gpt.go:935+` and is the default path via `hasher-host`, not the `Gorgonite` GPT.

**Build Status**: ✅ Compiles cleanly

---

## Phase Status Matrix

| Phase | Description | Status | Gap Level |
|-------|-------------|--------|-----------|
| **0** | Package refactor | ✅ COMPLETE | None |
| **1** | Tokeniser + Deterministic Embedder | ⚠️ PARTIAL | Medium |
| **2** | GPT Integration into HEARTService | ✅ COMPLETE* | Low |
| **3** | Endpoints + Inquiry Refactor | ✅ COMPLETE* | Low |
| **4** | Multi-Source DATA_MINER | ✅ COMPLETE | None |
| **5** | 4-Stage Pipeline | ✅ COMPLETE | None |
| **6** | WASM Compilation + wazero | ✅ COMPLETE | None |
| **7** | Bidirectional Verification | ⚠️ PARTIAL | Medium |
| **8** | HashNetwork Fast-Path | ⚠️ PARTIAL** | Medium |
| **9** | Softmax + Positional Encoding | ✅ COMPLETE | None |
| **10** | Head Pruning + DynamicGraph NAS | ⚠️ PARTIAL | Low |
| **11** | ES Weighted Update + TRPO | ✅ COMPLETE | None |
| **12** | Curriculum Training | ✅ COMPLETE | None |
| **13** | Entropy-Spike + Ontology Drift | ⚠️ PARTIAL | Medium |
| **Audit** | Audit Trail | ✅ COMPLETE | None |

\* `HEARTService` uses `HasherTransformer` path via `hasher-host`, not `Gorgonite` GPT directly.
\** Active hash path is `inference.RecursiveEngine` + `neural.HashNetwork` in `cmd/driver/hasher-host/main.go:510-514`. `hashnet_wrapper.go` is a stub.

---

## Detailed Findings

### Phase 0 — Package Refactor ✅

**Status: COMPLETE**

- All files in `pkg/hashing/transformer/` declare `package transformer`
- `gorgonia.org/gorgonia` and `gorgonia.org/tensor` present in `go.mod`
- `cmd/heart/main.go` exists with correct entrypoint
- `transformer.go` mentioned in spec does not exist — its contents (`HasherTransformer`) are in `gpt.go:935+`

---

### Phase 1 — Tokeniser + Deterministic Embedder ⚠️

**Status: PARTIAL**

| Item | Status | Notes |
|------|--------|-------|
| `GorgoniteConfig.VocabSize` | ✅ | Set to `100277` in `gpt.go:34` |
| cl100k tokeniser | ⚠️ | Uses `github.com/pkoukk/tiktoken-go` (`tiktoken`), not `pipeline/2_DATA_ENCODER/pkg/tokenizer` |
| Deterministic embedder | ⚠️ | Should import from `github.com/guiperry/text-embedder` (Go). Missing from `go.mod` — needs `go get github.com/guiperry/text-embedder` |
| `embedder` field on HEARTService | ✅ | Present at `heart_service.go:26` (`*embeddings.DeterministicService`) |
| `BPETokenizer` stub removal | ✅ | No stub tokeniser found in `gpt.go` |

**Required Actions:**
1. Add `embedder` field to `HEARTService` struct (`*embeddings.DeterministicService`)
2. Wire `embeddings.NewDeterministicService()` in `NewHEARTServiceWithConfig()`
3. Replace hardcoded `findSimilarErrors()` with embedder-based cosine similarity
4. Import from `knirvhasher/pipeline/2_DATA_ENCODER/pkg/embeddings` (Go package, not external lib)

---

### Phase 2 — GPT Integration into HEARTService ✅

**Status: COMPLETE** (with caveat that `HasherTransformer` is the active model, not `Gorgonite` GPT)

| Item | Status | Notes |
|------|--------|-------|
| `HEARTService.gpt` field | ✅ | Present at `heart_service.go:20` |
| `HEARTConfig` | ✅ | `config.go` matches spec exactly |
| `NewHEARTServiceWithConfig` | ✅ | Wires `gpt`, `bridge`, `tokenizer`, `compiler`, `verifier`, `auditor`, `hashNet` |
| `runGorgoniteInference` | ✅ | Present at `heart_service.go:203`, uses `hs.gpt.Forward(false)` |
| `factory` import in config | ✅ | Not needed — `HashNetworkWrapper` is legacy stub |

**Note**: The active inference path is `HasherTransformer` (hash-seed, `gpt.go:952`) used by `hasher-host` via `inference.RecursiveEngine`, not the Gorgonia GPT.

---

### Phase 3 — Endpoints + Inquiry Refactor ✅

**Status: COMPLETE**

| Item | Status | Notes |
|------|--------|-------|
| `WASMType` constants | ✅ | `types.go:5-9` |
| `PolicyBadgeInquiry` | ✅ | Present (note: field is `BadgeType` not `BadgeType`) |
| `DVEErrorInquiry` | ✅ | Present (note: field is `DVESessionID` not `DVESessionID`) |
| `SystemPatchInquiry` | ✅ | Present in `types.go` |
| `WASMDecision` | ✅ | Present (note: typo `BidirectionalVerified` at line 39, should be `BidirectionalVerified`) |
| `classifyInquiry()` | ⚠️ | Package-level function, not `HEARTService` method |
| `/heart/advise|resolve|patch` | ✅ | Registered in `ListenAndServe()` at `heart_service.go:175-184` |
| `/heart/health`, `/heart/stats` | ✅ | Present |
| DVE session validation | ✅ | `handleResolve` checks `DVESessionID == ""` |
| Multi-turn loop | ✅ | `process()` at `heart_service.go:276` implements turn loop |

---

### Phase 4 — Multi-Source DATA_MINER ✅

**Status: COMPLETE**

Files exist in `pipeline/1_DATA_MINER/internal/app/`:
- `source_record.go` ✅
- `source_manager.go` ✅
- `training_adapter.go` ✅

---

### Phase 5 — 4-Stage Pipeline ✅

**Status: COMPLETE**

`pipeline.go` implements all four stages:
- `Stage1Result`, `Stage2Result`, `Stage3Result`, `Stage4Result` ✅
- `runPipeline()` ✅
- GPT-augmented `stage2GPT()`, `stage3GPT()`, `stage4GPT()` ✅
- Fallback heuristic paths for when GPT/tokenizer unavailable ✅

---

### Phase 6 — WASM Compilation ✅

**Status: COMPLETE**

| Item | Status | Notes |
|------|--------|-------|
| `WASMCompiler` | ✅ | `compiler.go` matches spec |
| TinyGo templates | ✅ | `templates.go` has `RuleTemplate`, `ResolutionTemplate`, `PatchTemplate` |
| `CompileResult` | ✅ | Returns `WASMPath` + `WASMHash` (SHA-256) |
| wazero integration | ✅ | `WazeroGate` in `wazero_gate.go`; `WazeroPool` present |
| Content-addressed output | ✅ | `compiler.go:52-54` renames to hash-named file |

---

### Phase 7 — Bidirectional Verification ⚠️

**Status: PARTIAL**

| Item | Status | Notes |
|------|--------|-------|
| `BidirectionalVerifier` | ✅ | Present in `verifier.go:9-34` — **Go implementation** (no Python needed) |
| `Verify()` method | ✅ | Calls forward + backward pass via wazero + hashnet |
| Forward/Backward separate methods | ❌ | Uses single `Verify()`, not `ForwardVerify()`/`BackwardVerify()` |
| Python subprocess | ✅ | Not needed — Go `BidirectionalVerifier` is the implementation |
| `ForwardVerifier`/`BackwardVerifier` | ✅ | Separate types exist in `verifier.go:104-163` but unused |
| Confidence calculation | ✅ | `calculateConfidence()` returns float32 |

**Required Actions:**
1. Go `BidirectionalVerifier` is already the implementation — no Python needed
2. Add `ForwardVerify()`/`BackwardVerify()` methods to `BidirectionalVerifier` if separate directions desired

---

### Phase 8 — HashNetwork Fast-Path ⚠️

**Status: PARTIAL** (Active path is different from spec)

**Important**: The spec describes wrapping `inference.RecursiveEngine`, but the **active implementation** is:

```go
// cmd/driver/hasher-host/main.go:488-514
network, _ := neural.NewHashNetwork(*inputSize, *hidden1, *hidden2, *outputSize)
engine, _ := inference.NewRecursiveEngineWithHashMethod(network, hashMethod, *passes, *jitter, *seedRotation)
```

The `HashNetworkWrapper` in `hashnet_wrapper.go` is a **legacy stub** (just SHA-256, not wired). The real hash network is:
- `pkg/hashing/neural/hash_network.go` → `HashNetwork` struct
- `pkg/hashing/inference/recursive_engine.go` → `RecursiveEngine`
- `pkg/hashing/methods/asic/asic.go` → `ASICMethod` (hardware path)

| Item | Status | Notes |
|------|--------|-------|
| `RecursiveEngineWrapper` per spec | ❌ | Doesn't exist; `HashNetworkWrapper` is different |
| `HashNetworkWrapper` | ⚠️ | Stub — just `ComputeHash()` (SHA-256) |
| Fast-path in `process()` | ⚠️ | Present but uses `ComputeHash()`, not real classification |
| `Classify()` method | ❌ | Missing from wrapper |
| `HashNetResult` struct | ❌ | Not defined |

**Required Actions:**
1. Note: active path is `hasher-host` → `RecursiveEngine` → `HashNetwork`, not `HEARTService` path
2. Either align `HEARTService` to use same `inference.RecursiveEngine`, or update spec

---

### Phase 9 — Softmax + Positional Encoding ✅

**Status: COMPLETE**

| Item | Status | Notes |
|------|--------|-------|
| Softmax in `SelfAttention.Forward` | ✅ | `gpt.go:193`: `gorgonia.SoftMax(scores)` active |
| Positional encoding add | ✅ | `gpt.go:424`: `gorgonia.Add(x, posEnc)` active |
| Multi-head concat | ✅ | `gpt.go:256`: `gorgonia.Concat(1, headOutputs...)` (correct) |
| `HasherTransformer` positional encoding | ✅ | `gpt.go:1046+`: hash-based position embedding |

---

### Phase 10 — Head Pruning + DynamicGraph NAS ⚠️

**Status: PARTIAL**

| Item | Status | Notes |
|------|--------|-------|
| `ModelPruner` / `TransformerPruner` | ✅ | Present in `pruner.go` |
| Variance-guided pruning | ❌ | Uses magnitude/threshold, not `analyzer.VarianceAnalyzer` |
| `HeadPruner` per spec | ❌ | `pruner.go` doesn't use `VarianceAnalyzer` |
| DynamicGraph NAS hook | ❌ | Not implemented |
| `DynamicGraph` struct | ✅ | Exists in `dynamic_graph.go` |

**Required Actions:**
1. Rewrite pruning to use `pipeline/2_DATA_ENCODER/pkg/analyzer.VarianceAnalyzer`
2. Implement `PruneAfterEpoch()` with variance feedback
3. Add DynamicGraph NAS hook in training loop

---

### Phase 11 — ES Weighted Update + TRPO ✅

**Status: COMPLETE**

| Item | Status | Notes |
|------|--------|-------|
| `ESWeightedUpdate()` | ✅ | `evolutionary.go:992-1082` — full implementation |
| Mirrored sampling | ✅ | `mirroredSample()` at `evolutionary.go:971-986` |
| σ annealing | ✅ | `sigma()` cosine annealing at `evolutionary.go:958-967` |
| TRPO trust region | ✅ | L2-norm clamp at `evolutionary.go:1042-1053` |
| EvoGRPO | ✅ | `evo_grpo.go` exists; ES path wired through `ESWeightedUpdate` |

---

### Phase 12 — Curriculum Training ✅

**Status: COMPLETE**

| Item | Status | Notes |
|------|--------|-------|
| `CurriculumStage` types | ✅ | `internal/config/types.go:79-85` — Apprentice/Journeyman/Expert |
| `CurriculumConfig` | ✅ | `internal/config/types.go:88-94` |
| Stage-gated data selection | ✅ | `trainer.go:134-153` `defaultDataSelector()` filters by difficulty |
| `Trainer.TrainEpoch()` | ✅ | `trainer.go:57-75` — runs epoch, returns loss + advance flag |
| `Trainer.AdvanceStage()` | ✅ | `trainer.go:98-113` — promotes through Apprentice→Journeyman→Expert |
| Tie to σ annealing | ⚠️ | Not yet wired — curriculum stage not passed to `ESWeightedUpdate` step |

---

### Phase 13 — Entropy-Spike + Ontology Drift ⚠️

**Status: PARTIAL**

| Item | Status | Notes |
|------|--------|-------|
| `computeEntropy()` | ✅ | `heart_service.go:248-272` |
| Per-token entropy in `runGorgoniteInference` | ✅ | `heart_service.go:224-231` |
| `handleEntropySpike()` | ✅ | Present at `heart_service.go:238` |
| Gap queue routing | ❌ | Doesn't route to `OntologyGap`/`NovelError`/`SystemAlert` queues |
| HashNetwork next-token suggestion | ❌ | Not implemented in spike handler |
| `DeltaSignalIndices()` | ❌ | Not in `analyzer/variance.go` |
| OntologyDrift detection | ❌ | Not implemented |

**Required Actions:**
1. Implement gap queue routing by WASM type in `handleEntropySpike()`
2. Add `DeltaSignalIndices()` to `pipeline/2_DATA_ENCODER/pkg/analyzer/variance.go`
3. Emit `OntologyDrift` event when delta > 0.4

---

## Audit Trail ✅

**Status: COMPLETE**

| Item | Status | Notes |
|------|--------|-------|
| `AuditRecord` struct | ✅ | `audit.go:12-19` matches spec |
| `Auditor.WriteAuditLog()` | ✅ | Writes JSON-lines to daily files |
| `writeAuditLog` in `HEARTService` | ✅ | Called from `buildDecision()` at `heart_service.go:409-417` |
| `InquiryHash` calculation | ✅ | SHA-256 of inquiry JSON |
| Daily log rotation | ✅ | `heart_audit_YYYY-MM-DD.jsonl` |

---

## Files Referenced in Code (Active Paths)

```
packages/KNIRVHASHER/
  cmd/
    heart/
      main.go                          ← Phase 0: HEARTService entrypoint
    driver/
      hasher-host/
        main.go                      ← ACTIVE PATH: wires neural.HashNetwork + inference.RecursiveEngine

  pkg/hashing/
    transformer/
      gpt.go                        ← Phase 0: GorgoniteConfig + HasherTransformerConfig + GPT + HasherTransformer
      config.go                      ← Phase 2: HEARTConfig
      types.go                       ← Phase 3: WASMType, inquiry types, WASMDecision
      heart_service.go               ← Phase 2-3: HEARTService, endpoints, process() loop
      pipeline.go                    ← Phase 5: 4-stage pipeline
      compiler.go                    ← Phase 6: WASMCompiler (tinygo)
      templates.go                   ← Phase 6: TinyGo source templates
      wazero_gate.go                 ← Phase 6: WazeroGate + WazeroPool
      verifier.go                    ← Phase 7: BidirectionalVerifier + Forward/Backward verifiers
      hashnet_wrapper.go             ← ⚠️ LEGACY STUB (not active path)
      pruner.go                      ← Phase 10: ModelPruner + TransformerPruner
      audit.go                       ← Audit: AuditRecord + Auditor
      cerebras_bridge.go             ← Phase 0: CerebrasBridge
      dynamic_graph.go               ← Phase 0: DynamicGraph wrapper

    neural/
      hash_network.go               ← ACTIVE: neural.HashNetwork (used by hasher-host)
    inference/
      recursive_engine.go           ← ACTIVE: inference.RecursiveEngine
    methods/asic/
      asic.go                      ← ACTIVE: ASICMethod (hardware SHA-256)

  pipeline/
    1_DATA_MINER/
      internal/app/
        source_record.go             ← Phase 4: SourceRecord
        source_manager.go           ← Phase 4: SourceManager
        training_adapter.go         ← Phase 4: UnifiedTrainingAdapter
    3_DATA_TRAINER/
      pkg/training/
        evolutionary.go             ← Phase 11: NEEDS ES update
      internal/
        evo_grpo/
          evo_grpo.go              ← Phase 11: NEEDS EvoGRPO completion
```

---

## Duplicate / Confusing Elements

| File | Issue | Resolution |
|------|-------|------------|
| `gpt.go:935+` | `HasherTransformer` is separate from `GPT` (`gpt.go:364+`) — two transformer implementations in one file | By design: `HasherTransformer` = hash-seed (hardware), `GPT` = Gorgonia (software) |
| `hashnet_wrapper.go` | Stub `HashNetworkWrapper` conflicts with active `neural.HashNetwork` | Document as legacy; active path is via `hasher-host` |
| `WASMDecision.BidirectionalVerified` | Typo: should be `BidirectionalVerified` (`types.go:39`) | Fix field name |
| `DVESessionID` vs `DVESessionID` | Inconsistent casing in `types.go:20` | Standardize to `DVESessionID` |
| `BadgeType` vs `BadgeType` | Inconsistent casing in `types.go:13` | Standardize to `BadgeType` |
| `verifier.py` | Exists in `transformer/` dir but isn't wired to `BidirectionalVerifier` | Implement subprocess pattern per Phase 7.1 |

---

## Missing Files (per spec)

```
packages/KNIRVHASHER/
  pkg/hashing/transformer/
    verifier.py                      ← Phase 7: stub exists but needs wiring

  pipeline/3_DATA_TRAINER/
    internal/config/
      types.go                      ← Phase 12: NEEDS CurriculumStage + CurriculumConfig
```

---

## Recommended Priority Order

| Priority | Phase | Reason |
|----------|-------|--------|
| 1 | **Phase 13** | Entropy-spike detection for self-improvement — gap queue routing still missing |
| 2 | **Phase 12 (σ tie)** | Wire curriculum stage into `ESWeightedUpdate` step parameter |
| 3 | **Phase 7** | Wire `verifier.py` subprocess for real bidirectional verification |
| 4 | **Phase 1** | Wire embedder into `findSimilarErrors()` cosine similarity |
| 5 | **Phase 10** | Variance-guided pruning for model efficiency |

---

## HasherService vs HasherTrainingService Distinction

These are two separate gRPC services with zero operational overlap.

| Aspect | HasherService | HasherTrainingService |
|--------|---------------|----------------------|
| Proto package | `hasher.v1` | `hasher.v1` (KNIRVHASHER) / `hasher` (KNIRVSERVER) |
| gRPC path | `/hasher.v1.HasherService/...` | `/hasher.v1.HasherTrainingService/...` |
| Concern | ASIC computation — hash, batch, mine, verify | Dataset collection, training orchestration, rule management |
| Hosted in | `KNIRVHASHER/cmd/driver/hasher-server` | `KNIRVSERVER` (server-side impl); KNIRVHASHER acts as client |
| Go server impl | `HasherServer` in `internal/driver/device/server.go` | `UnimplementedHasherTrainingServiceServer` (stub only in KNIRVHASHER) |
| Registered at | `hasher-server/main.go:277` | `KNIRVSERVER/backend/internal/services/dvemanager/hasher_grpc_server.go` |

**Naming fix (this audit):** The proto previously declared `service hasherService` (lowercase), making the gRPC path `/hasher.v1.hasherService/...` inconsistent with `HasherTrainingService`. Fixed to `service HasherService` so both services use PascalCase and their paths are consistent.

---

## Key Architectural Notes

1. **Two transformer implementations**: `GPT` (Gorgonia, float32) and `HasherTransformer` (hash-seed, [32]byte) coexist in `gpt.go`. The latter is the default for `hasher-host`.

2. **Active hash path**: `cmd/driver/hasher-host/main.go` → `neural.HashNetwork` → `inference.RecursiveEngine` → (optional) `asic.ASICMethod`. The `HEARTService` in `heart_service.go` is a separate HTTP service path.

3. **`HasherTransformerConfig`** (hash-based) is in `gpt.go:939`, **`GorgoniteConfig`** (Gorgonia) is in `gpt.go:22`. These are correctly separated per Phase 0.3.

4. **`transformer.go`** mentioned in the spec does not exist — its presumed contents are in `gpt.go:935+` (`HasherTransformer`).
