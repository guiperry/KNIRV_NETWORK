# KNIRVHASHER Pipeline Upgrade & Refactor: Local Ontology / HuggingFace / arXiv → `.nrv` → KNIRVBASE

**Status:** Planning — merges the original Phase 2 KNIRVBASE data-flow upgrade with the pipeline-structure refactor (Stage 0 entry point, `data-mapper` rename, spaCy fix, single embedded KNIRVBASE binary). These two efforts are interleaved on purpose: the structural refactor is largely a **prerequisite** for the data-flow upgrade to land cleanly, not a separate track.
**Source specs:** `packages/KNIRVBASE/go/KNIRVBASE_Upgrade.md` (Phase 2, ASIC-Native `.nrv`/Frame/Bracket binary format — unchanged by this document), `docs/DATA-MAPPER.md` (12-slot Bitmask Specification — unchanged by this document)
**Pipeline root:** `packages/KNIRVHASHER/pipeline/`

> **🚨 ARCHITECTURAL WARNING (unchanged from the original spec)**: This upgrade introduces LSH projections as a **parameterization** of the Identity Zone (Slots 0-3), NOT a replacement for the Semantic Coherence Mapper. The 12-slot Bitmask Specification from `DATA-MAPPER.md` is MANDATORY and must be preserved. Key requirements:
> - **Slots 0-3 (Identity Zone)**: LSH projections via SemanticMapper (replaces "Naive Variance")
> - **Slots 4-5 (Syntactic Registers)**: POS, Tense, Dependency — **REQUIRED for Syntactic Steering**
> - **Slots 6-8 (Memory Zone)**: History XOR from Flash Search — **REQUIRED for 21-pass recurrence**
> - **Slot 9 (Intent Flags)**: Question/Command/Code detection
> - **Slot 10 (Domain Signature)**: Math/Code/Prose classification — this document additionally makes Slot 10 the **KNIRVBASE collection key** (see §7.4)
> - **Slot 11 (Temporal Lock)**: Position + Salt for uniqueness
>
> The 21-pass temporal loop requires Syntactic Steering (Passes 8-14 using Slots 4-5). If these are missing, the system collapses to random hashing. **This is also why the spaCy fix in §5.6 matters**: without a working NLP Bridge, Slots 4-5 go permanently zero-filled.

This document does **not** touch the 12-slot bitmask spec, the ASIC hashing engine, or the Frame/Bracket `.nrv` binary format. It changes: what triggers the pipeline, where source-selection and NLP live, how failures are surfaced, and how the pipeline talks to KNIRVBASE.

---

## 1. Investigation Evidence (current state, confirmed by direct code inspection)

Before drafting the target architecture below, the current codebase was audited to confirm each problem actually exists as described and to find root causes rather than guessing. This table is the evidence base for every section that follows — cite it back if a section's rationale is unclear.

| # | Observation | Confirmed by |
|---|---|---|
| 1 | Pipeline starts at stage 1, not stage 0 | `internal/cli/controller/controller.go:850-897` — `buildPipelineStages()`'s default case (`"goat"`, which is what `production` mode uses per `pipeline/1_DATA_MINER/internal/app/config.go:212`) returns a stage list that **omits `dataConnectorStage` entirely**. Only `"arxiv"` and `"demo"` pipeline types include it. Comment at controller.go:857-859 confirms this is intentional today: *"Goat/MAPPER mode fetches from Hugging Face API directly via data-miner and must NOT run the data-connector first."* |
| 2 | "data miner" should be "data mapper" | `docs/DATA-MAPPER.md` already documents this stage's NLP+embedding+tensor work under the "HASHER-MAPPER" name, and controller.go's own comments already call it "MAPPER mode" (controller.go:884, :892) despite the binary/directory still being named `data-miner`/`1_DATA_MINER`. The rename formalizes naming that's already informally in use. |
| 3 | Source-download logic belongs in stage 0 | `pipeline/1_DATA_MINER/internal/app/processor.go:1511-1750` (`RunGoatMiningPhase`) fetches directly from `datasets-server.huggingface.co` from inside the miner. `pipeline/1_DATA_MINER/internal/arxiv/*` fetches directly from `export.arxiv.org` from inside the miner. Meanwhile `pipeline/0_DATA_CONNECTOR/cmd/data-connector/main.go` exists but does something unrelated today (polls KNIRVSERVER's `ExportSecurityData` gRPC for security-event data, not training corpus data) — **the "Before" state below corrects an inaccurate assumption in the original version of this document**, which described Stage 0's current output as "CSV / local files." |
| 4 | spaCy silently fails, shouldn't | Two independent, concrete root causes found — see §5.6. |
| 5 | Stage 0 should pick local ontology → GOAT/HF → arxiv.org | No such priority chain exists anywhere. Source selection today is a **static CLI/config flag** (`GoatMode` / `EnableArxivMining` in `config.go`), not runtime availability detection. "Local ontology" maps to KNIRVSERVER's real `/api/ontology/{stats,entities,relations,search}` REST endpoints, backed by `OntologyManager` in `KNIRV_CORP/packages/server/backend_server/internal/services/memory/ontology.go` — this is not currently consumed by KNIRVHASHER at all. |
| 6 | Processed records should land in KNIRVBASE as a domain-specific `.nrv` object | The original version of §6/§7 below already specced `AppendBracketDirect`/Arrow Flight submission, but keyed collections by raw `DatasetID` rather than domain — refined in §7.4. Still unimplemented — Stage 3 (`3_DATA_SEEDER`) currently writes BestSeed CSV / BPF weights, not KNIRVBASE. |
| 7 | KNIRVBASE should broadcast `.nrv` frames via Arrow Flight | `packages/KNIRVBASE/go/internal/network/flight_server.go` **already implements** a `FlightServer` with `StreamBrackets`/ticket parsing over Arrow IPC. It is never instantiated by a running server — `cmd/node/main.go` only opens the DB as a library and never starts the Flight server. |
| 8a | KNIRVBASE started inside the data connector | `pipeline/0_DATA_CONNECTOR/cmd/data-connector/main.go:31-36` calls `knirvbase.New(ctx, ...)` directly (Go module import: `github.com/knirvcorp/knirvbase v1.0.7` in `0_DATA_CONNECTOR/go.mod` and `1_DATA_MINER/go.mod`). This violates the repo's own "no cross-package Go imports" convention (`KNIRV_NETWORK/CLAUDE.md`) and pins KNIRVBASE to whatever version was last vendored. |
| 8b | Dead KNIRVBASE instance inside KNIRVSERVER | `KNIRV_CORP/packages/server/backend_server/internal/database/knirvbase.go` (`KNIRVBASEManager`) — a **separate, parallel reimplementation** (markdown+PQC file storage, not the real `knirvcorp/knirvbase` module) gated by `cfg.Database.UseKNIRVBASE`, defaulted `false` ("Disabled by default during Phase 1" — `internal/database/buntdb.go:117`). Wired into `cmd/backend_server/main.go:1074` and `buntdb.go:274-430` but never enabled. **This file lives in the separate `KNIRV_CORP` git repository, not `KNIRV_NETWORK`** — `packages/KNIRVSERVER` in this repo only vendors the compiled `backend_server` binary, so this removal is a follow-up PR in `KNIRV_CORP`, tracked here but executed there. |
| 9 | `pkg/knirvbase`'s internal storage/writer types aren't actually importable from outside the module | Original §6.5 (Frame Assembly via FrameTicker) example code calls `storage.NewNRVWriter(...)` from `github.com/knirvcorp/knirvbase/internal/storage` — Go's `internal/` visibility rule means **no external module can import this**, including `2_DATA_ENCODER`. This was an unnoticed bug in the original spec, corrected in §6.5.5. |

---

## 2. Overview

The current KNIRVHASHER pipeline mines PDF/arXiv documents, encodes embeddings into 12-slot ASIC Parquet frames, and runs evolutionary training to find optimal SHA-256 seeds. The output stays local to the pipeline, and — per the evidence table above — the pipeline doesn't even reliably *start* at its own Stage 0.

This upgrade transforms the pipeline into a **KNIRVBASE data ingestion system**: Stage 0 pulls records from a priority chain of sources (local ontology → HuggingFace → arXiv), Stage 1 (renamed `data-mapper`) embeds and NLP-tags them, Stage 2 encodes them into the Phase 2 `.nrv` binary format (80-byte Brackets inside 1-second Frames), and Stage 3 submits them into a single standalone KNIRVBASE binary — organized by **domain** — via its Go SDK client and Arrow Flight.

### 2.1 Before and After

| Stage | Current Behavior | New Behavior |
|---|---|---|
| 0_DATA_CONNECTOR | Polls KNIRVSERVER's `ExportSecurityData` gRPC for security telemetry; starts an in-process KNIRVBASE instance via Go import; **not part of the default pipeline run** (skipped by `buildPipelineStages`'s default case) | Runs a three-tier source-priority chain (local ontology → HuggingFace GOAT → arXiv), keeps the existing security-telemetry connector as one connector among several, talks to KNIRVBASE over the network instead of importing it, and is **unconditionally the first stage of every pipeline run** |
| 1_DATA_MINER | Fetches HuggingFace GOAT dataset and arXiv papers directly from inside the miner; NLP Bridge (spaCy) fails silently and near-invisibly in practice | Renamed **`1_DATA_MAPPER`**; no longer talks to any external data source — reads `RawRecord` shards staged by Stage 0 only; spaCy enabled by default with loud, non-fatal failure diagnostics |
| 2_DATA_ENCODER | `training_frames.parquet` (12 ASIC slots) | `.nrv` files (80-byte Brackets, Phase 2 format), written with a locally-owned encoder (not an invalid cross-module `internal/` import) |
| 3_DATA_SEEDER | BestSeed CSV / BPF weights | KNIRVBASE ingest via the standalone `knirvbase` binary's client (`AppendBracketDirect`-equivalent RPC or Arrow Flight), collections keyed by **domain** (Math/Code/Prose/Academic), not raw dataset ID |
| KNIRVBASE | Three parallel, divergent implementations: library-imported into `0_DATA_CONNECTOR`, a dead custom reimplementation inside KNIRVSERVER, and a Flight server that's implemented but never run | **One** binary, embedded into and extracted from the KNIRVHASHER bundle exactly like the other pipeline binaries, running the real Arrow Flight service continuously |

### 2.2 Key Terminology (from KNIRVBASE Phase 2 Spec)

> **🔗 CROSS-REFERENCE**: The Frame/Bracket hierarchy is defined in `packages/KNIRVBASE/go/KNIRVBASE_Upgrade.md` Section 3. This section summarizes the key concepts for the pipeline. Unchanged by this refactor.

| Term | Meaning | 12-Slot Mapping |
|---|---|---|
| **Bracket** | 80-byte binary record: LSH Projections (32B) + SubSecondUS (4B) + Syntactic (4B) + Intent+Domain (3B) + GoldenSeed (4B) + Memory (14B) + LSH Salt (4B) + Reserved (15B) | Encodes all 12 slots |
| **Frame** | 1-second temporal window holding N Brackets + ThermoAtmosphere + LinguisticMapping + Z3Result | Container for bracket batch |
| **I-Bracket** | Absolute (intra) bracket — stores full LSH projections (Slots 0-3) | Full semantic snapshot |
| **P-Bracket** | Delta bracket — stores XOR-diff of projections against anchor I-Bracket | Compressed delta |
| **FrameTicker** | 1-second goroutine that buffers brackets, computes I/P type, XOR-diffs, and flushes to NRVWriter | Orchestrates frame assembly |
| **MemoryZone** | In-memory state (Slots 6-8) maintained by FrameTicker for temporal loop recurrence | History XOR computation |

### 2.3 Frame/Bracket Hierarchy

```
.nrv File
│
├─ Chunk 0: Resolution Registry (JSON)
│   └─ frames: [FrameEntry, FrameEntry, ...]
│       ├─ id: "<uuid>"
│       ├─ timestamp_unix: 1712580120
│       ├─ linguistic: {token, unit}
│       ├─ thermo: {avg_temp_c, peak_volt_v, clock_mhz}
│       ├─ z3: {status, relevance}
│       ├─ brackets: {count: 307, offset: 5242892, length: 24560}
│       └─ bracket_index: [BracketMeta, ...]
│           │        └─ type: "I" | "P"
│           │        └─ anchor_id: "<I-bracket-uuid>" (for P-brackets)
│           │        └─ drift_score: 0.014
│           │
│           ▼
│
└─ Chunk 1: Multi-Modal Buffer (80-byte Brackets)
    │
    ├─ Frame 0 (1-second window)
    │   ├─ Bracket[0] (I): 80 bytes
    │   ├─ Bracket[1] (P): 80 bytes ← XOR diff
    │   ├─ Bracket[2] (P): 80 bytes ← XOR diff
    │   └─ ... (N brackets)
    │
    ├─ Frame 1 (1-second window)
    │   └─ ...
    │
    └─ ...
```

**Frame Entry Registry JSON:**

```json
{
  "id": "frame-uuid",
  "timestamp_unix": 1712580120,
  "linguistic": {"token": "resolution", "unit": "word"},
  "thermo": {"avg_temp_c": 71.4, "peak_volt_v": 1.37, "clock_mhz": 553.0},
  "z3": {"status": "VALID", "relevance": 0.92},
  "brackets": {"count": 307, "offset": 5242892, "length": 24560},
  "bracket_index": [
    {"id": "bkt-0", "type": "I", "anchor_id": null, "offset": 0, "drift_score": 0.0},
    {"id": "bkt-1", "type": "P", "anchor_id": "bkt-0", "offset": 80, "drift_score": 0.014}
  ]
}
```

**Bracket Binary Layout (80 bytes):**

| Offset | Size | Field | Slot Mapping | Pass Utility |
|---|---|---|---|---|
| `0x00` | 32B | LSH Projections | Slots 0-3 (Compass) | Passes 1-7: Topic Anchoring |
| `0x20` | 4B | SubSecondUS | (Temporal Ticker) | Frame synchronization |
| `0x24` | 1B | POSTag | Slot 4 (Syntactic) | Passes 8-14: Syntactic Steering |
| `0x25` | 1B | Tense | Slot 4 (Syntactic) | Passes 8-14: Syntactic Steering |
| `0x26` | 1B | Plurality | Slot 4 (Syntactic) | Passes 8-14: Syntactic Steering |
| `0x27` | 1B | DepHead | Slot 5 (Dependency) | Passes 8-14: Structural Logic |
| `0x28` | 1B | IntentFlags | Slot 9 (Intent) | Identity Stabilization |
| `0x29` | 2B | DomainSig | Slot 10 (Domain) | Mode Enforcement (e.g., Math) — **also drives KNIRVBASE collection naming, §7.4** |
| `0x2B` | 4B | GoldenSeed | (Nonce Target) | The solved "Weight" |
| `0x2F` | 14B | Memory (XOR recursive) | Slots 6-8 (Temporal) | Recursive Context Bridge |
| `0x3D` | 4B | LSH Salt | Slot 11 (Lock) | Prevents Collision loops |
| `0x41` | 15B | Reserved | (Future Expansion) | Padding to 80 bytes |

---

## 3. Stage 0: DATA_CONNECTOR — Source-Priority Ingestor

**Package:** `pipeline/0_DATA_CONNECTOR`
**Goal:** Be the pipeline's real entry point (§4 below fixes the controller so this is enforced), decide *which* of three source tiers to pull from at runtime, normalize whatever it pulls to a standard internal schema, and hand off to Stage 1 (`1_DATA_MAPPER`). Also stops instantiating KNIRVBASE in-process (§7).

### 3.1 Three-tier source priority (local ontology → HuggingFace GOAT → arXiv)

This replaces the current static `GoatMode`/`EnableArxivMining` config flags (`pipeline/1_DATA_MINER/internal/app/config.go`) with runtime availability detection, run inside `0_DATA_CONNECTOR`:

1. **Local ontology (first choice).** Query KNIRVSERVER's `/api/ontology/stats` (backed by `OntologyManager` in `KNIRV_CORP/.../internal/services/memory/ontology.go`, routes registered in `internal/web/ontology_handlers.go`). If `entityCount > 0` (threshold TBD — see Open Questions §14.2), pull records via `/api/ontology/entities` and `/api/ontology/relations`, normalize to `RawRecord` (§3.2 below), and use this source for the batch.
2. **GOAT / HuggingFace (second choice).** If the local ontology is empty/unavailable (connection refused, 0 entities), fall back to the `HuggingFaceConnector` (§3.4).
3. **arXiv.org (third choice, "exhausted" fallback).** When HuggingFace pagination is exhausted for the configured dataset(s) — i.e. `datasets-server.huggingface.co/rows` returns an empty `rows` array or a 4xx indicating the offset is past the end of the dataset — fall back to the arXiv connector migrated in from Stage 1 (§3.5).

This is a state machine per connector **run**, not per-record: check ontology → if empty, run HF until exhausted → if exhausted, run arXiv. Persist "which tier we're on" in the connector's existing bbolt checkpoint store (already used for HF dedup, see §3.4) so restarts resume from the correct tier instead of re-probing the ontology every batch. Provide an explicit override flag to force one tier for testing/debugging, but the unconditional default must be this three-tier chain — a single `pipelineType` no longer means "which binary/flags do we run" (see §4), it now only selects which config profile the connector loads for this chain.

### 3.2 New: Local Ontology Connector (`connector/ontology.go`)

**Responsibilities:**
- Call `GET {knirvserver_addr}/api/ontology/stats` to check `entityCount`/`relationCount` before committing to this tier.
- If non-empty, page through `GET /api/ontology/entities` and `GET /api/ontology/relations`, mapping each `OntologyEntity`/`OntologyRelation` to a `RawRecord` (entity description + related relations joined into `Text`, entity `Type` and involved relation types as `Tags`).
- No pagination-exhaustion fallback logic is needed within this tier — exhausting the ontology (all entities/relations consumed) simply means "this batch is done," not "fall through to HF," since the ontology snapshot doesn't refresh mid-run. Re-check `/api/ontology/stats` at the start of the *next* connector run in case KNIRVSERVER's ontology has grown.

### 3.3 `RawRecord` (internal canonical schema, shared by all three connectors)

```go
type RawRecord struct {
    DatasetID string   // e.g. "tatsu-lab/alpaca", "knirvserver-ontology", "arxiv:cs.LG"
    Split     string   // "train", "test", "ontology", etc.
    Index     int64    // row index within the shard
    Text      string   // normalized plain text (instruction + response joined)
    Tags      []string // optional labels / categories from dataset/entity metadata
}
```

### 3.4 HuggingFace Connector (`connector/huggingface.go`)

**Migrated in from `pipeline/1_DATA_MINER/internal/app/processor.go:1511-1750` (`RunGoatMiningPhase`)** — same HTTP behavior, relocated so Stage 1 (renamed `1_DATA_MAPPER`) no longer talks to the network at all. See §5.2 for the corresponding removal from the miner/mapper side.

**Responsibilities:**
- Authenticate to HuggingFace Hub (token via `HF_TOKEN` env var)
- Enumerate dataset splits (train / validation / test)
- Stream dataset shards as Parquet from the HuggingFace CDN — no full download required
- Support datasets with `text`, `input`/`output`, `instruction`/`response`, or `messages` (chat) columns
- Emit records conforming to the shared `RawRecord` schema (§3.3)
- Detect pagination exhaustion (empty `rows` / offset-past-end 4xx) and signal Tier 3 fallback per §3.1

**Column normalization rules (apply in order, first match wins):**

| Dataset column(s) present | Text assembly |
|---|---|
| `text` | use directly |
| `instruction` + `output` | `"### Instruction:\n" + instruction + "\n\n### Response:\n" + output` |
| `input` + `output` | `input + "\n" + output` |
| `messages` (chat list) | join all `content` fields with `"\n"` |
| fallback | concatenate all string columns |

**Config struct:**

```go
type HuggingFaceConfig struct {
    DatasetIDs   []string // e.g. ["tatsu-lab/alpaca", "Open-Orca/OpenOrca"]
    Splits       []string // defaults to ["train"]
    MaxRowsPerDS int      // 0 = unlimited
    ShardWorkers int      // parallel shard download workers (default 4)
    CacheDir     string   // local cache path (default ~/.cache/knirvhasher/hf)
    Token        string   // HF_TOKEN override (reads env if empty)
}
```

**HuggingFace API endpoints to use:**

- Dataset info: `https://huggingface.co/api/datasets/{dataset_id}`
- Parquet shard listing: `https://huggingface.co/api/datasets/{dataset_id}/parquet/{split}`
- Shard download: URLs returned from the listing endpoint (Bearer `HF_TOKEN`)

Rate-limit all requests to ≤ 1 request/second per dataset. Retry with exponential backoff (max 5 retries).

### 3.5 arXiv Connector (`connector/arxiv.go`)

**Migrated in from `pipeline/1_DATA_MINER/internal/arxiv/{arxiv_client,arxiv_miner,arxiv_worker}.go`.** Same client behavior against `export.arxiv.org` (`BaseURL: "http://export.arxiv.org/api/query"`), relocated to Stage 0 as the Tier 3 fallback per §3.1. `ArxivCategories`, `ArxivMaxPapers`, `ArxivDownloadDelay` config fields move here from Stage 1's `Config`/`types.go`.

### 3.6 Existing: KNIRVSERVER security-export connector

`0_DATA_CONNECTOR/cmd/data-connector/main.go`'s current behavior (poll `HasherTrainingServiceClient.ExportSecurityData`, decrypt chunks, write raw `.md` files) is a legitimate, distinct source (security telemetry) — **keep it**, but restructure `main.go` from a single-purpose poller into a small connector registry that runs this poller *alongside* the ontology/HF/arXiv connectors above, all emitting the shared `RawRecord` schema from §3.3. It is not part of the three-tier priority chain (§3.1) — it runs independently on its own poll interval, same as today.

### 3.7 Cleaner (`connector/cleaner/`)

Already partially present at `0_DATA_CONNECTOR/internal/cleaner/cleaner.go` — extend to handle:
- Unicode normalization (NFC)
- Remove HTML tags / markdown artifacts
- Deduplicate records by hashed `Text` content (bbolt checkpoint, same store used for the Tier 1/2/3 state machine in §3.1)
- Min/max length filter: skip records with `len(Text) < 32` or `len(Text) > 16384`

### 3.8 Config File Update (`config/`)

```yaml
knirvserver_addr: "localhost:8084"   # HTTP API for ontology tier + gRPC for security-export connector

ontology:
  min_entity_count: 1   # threshold before preferring this tier — see Open Questions §14.2

huggingface:
  datasets:
    - tatsu-lab/alpaca
    - Open-Orca/OpenOrca
    - HuggingFaceH4/ultrachat_200k
  splits: [train]
  max_rows_per_dataset: 50000
  shard_workers: 4
  cache_dir: ~/.cache/knirvhasher/hf

arxiv:
  categories: ["cs.LG", "cs.CL"]
  max_papers: 50
  download_delay_seconds: 2

knirvbase:
  addr: "localhost:50052"   # standalone knirvbase binary's client-facing address — see §7
```

### 3.9 Writer Update (`connector/writer/`)

The writer outputs `RawRecord` as Parquet shards to `~/.local/share/knirvhasher/connector/` for Stage 1 to consume. Keep the existing bbolt checkpoint to avoid reprocessing rows.

### 3.10 KNIRVBASE dependency removal

`cmd/data-connector/main.go:31-36` currently calls `knirvbase.New(ctx, knirvbase.Options{DataDir: ...})` directly and imports `github.com/knirvcorp/knirvbase v1.0.7` in `go.mod`. Once §7's standalone `knirvbase` binary exists, remove this import entirely; the `writer.NewMDWriter(collection)` call (currently backed by a locally-opened `db.Collection("connector_raw")`) becomes a network call — a small gRPC client against the running `knirvbase` binary's insert API, using the `knirvbase.addr` config value from §3.8.

---

## 4. Make the pipeline start at Stage 0

**File:** `internal/cli/controller/controller.go`

- `buildPipelineStages()`'s default (`"goat"`) case must prepend `dataConnectorStage` like the `"arxiv"` and `"demo"` cases already do.
- Delete the "must NOT run the data-connector first" comment and the special-case reasoning behind it — it becomes false once §3 moves HF/ontology/arXiv fetching into `0_DATA_CONNECTOR`.
- `data-mapper` (renamed from `data-miner`, §5.1) stops taking `-goat`/`-arxiv-enable` source-selection flags — it becomes a pure consumer of whatever `0_DATA_CONNECTOR` staged. Its CLI surface shrinks to input/output paths, worker count, chunking, and NLP toggles.
- `RunPipeline`'s per-batch loop (controller.go:544-655) already restarts the full stage list every batch, so once `dataConnectorStage` is unconditionally first, every pipeline type runs 0→1→2→3 uniformly. The `pipelineType` parameter's job shrinks from "which binary/flags do we run" to "which config profile does the connector load" (§3.1/§3.8).

> **⚠️ SEQUENCING NOTE**: This change must land **after** §3 (Stage 0 doing the right thing) and **after** §5 (the rename), not before — see §12 Implementation Order. Flipping the controller first would just make the current, unrelated security-export connector the pipeline's stage 0, with GOAT/arXiv fetching still live inside the miner. That's a pointless intermediate state.

---

## 5. Stage 1: DATA_MAPPER (renamed from DATA_MINER) — Embedding Generator

**Package:** `pipeline/1_DATA_MAPPER` (renamed from `pipeline/1_DATA_MINER`)
**Goal:** Transform `RawRecord` Parquet shards (now exclusively produced by Stage 0) into embedding records with 768-dim vectors and NLP metadata, output as `mined_records.parquet`.

### 5.1 Rename mechanics (`data-miner` → `data-mapper`)

Mechanical rename, no behavior change by itself. ~30 files reference `data-miner`/`1_DATA_MINER` (full list gathered during investigation):

- **Directory:** `pipeline/1_DATA_MINER/` → `pipeline/1_DATA_MAPPER/`
- **Go module:** `module data-miner` → `module data-mapper` (`go.mod`), `cmd/data-miner/` → `cmd/data-mapper/`
- **Binary name:** `Makefile:6 BINARY_NAME := data-miner` → `data-mapper`; scripts (`run.sh`, `run_hybrid.sh`, `run_optimized.sh`, `run_workflow.sh`, `test_workflow.sh`, `kill_dataminer.sh` → `kill_datamapper.sh`)
- **Embedded binary set:** `internal/cli/embedded/binaries.go` — `AvailableBinaries` entry `Name: "data-miner"` → `"data-mapper"`; `internal/cli/embedded/bin/data-miner` → `internal/cli/embedded/bin/data-mapper`
- **Controller:** `internal/cli/controller/controller.go` — every `BinName: "data-miner"` / `stage.BinName == "data-miner"` reference, plus `cmd/cli/main.go` and `internal/cli/ui/ui.go` references
- **Docs:** `pipeline/1_DATA_MAPPER/README.md`, `docs/ARXIV_IMPLEMENTATION_SUMMARY.md`, `docs/DATA_MANAGEMENT_FIXES.md`, `docs/HYBRID_EMBEDDINGS_COMPLETE.md`, `docs/IMPLEMENTATION_SUMMARY.md`, `pipeline/2_DATA_ENCODER/docs/Data_Encoder_Spec.md`, `pipeline/2_DATA_ENCODER/README.md`, this document, top-level `Makefile`, `scripts/kill_hasher.sh`, `pipeline/scripts/bootstrap_alpaca.sh`
- Internal package code itself (`internal/app/*.go`) keeps its `package app` name; only the module path, binary name, and external references change. `data_dirs_test.go` / `tests/*.go` string literals referencing `data-miner` paths need updating too.

Do this rename **before** the source-fetch-migration work in §5.2 lands, so that migration happens on the already-renamed tree and there's a single clean diff to review for the rename itself (see §12 Implementation Order).

### 5.2 Input change: source-fetch logic removed, `RawRecord` consumption added

**Removed from this package** (moved to `0_DATA_CONNECTOR`, §3.4/§3.5):
- `internal/app/processor.go:1511-1750` (`RunGoatMiningPhase` + the `hfURL`/checkpoint/GOAT-Alpaca-JSON logic)
- `internal/arxiv/{arxiv_client,arxiv_miner,arxiv_worker}.go`
- Config fields `GoatMode`, `EnableArxivMining`, `ArxivCategories`, `ArxivMaxPapers`, `ArxivDownloadDelay`, and the source-selection portion of `CloudflareLimit`

**Stays in this package** — PDF text extraction, chunking, Ollama/Cloudflare embedding calls, NLP Bridge (§5.6), `DocumentRecord` construction, and `internal/app/knirv.go`'s `.knirv`/Arrow local-file loading (a generic input-format reader, now reused specifically for reading Stage 0's `RawRecord` output).

**New input paths:**

```
Primary:   ~/.local/share/knirvhasher/connector/*.parquet  (from Stage 0, all three tiers land here uniformly)
Fallback:  ~/.local/share/data-mapper/ai_knowledge_base.parquet (legacy, pre-refactor local runs)
```

The mapper reads `RawRecord` from Stage 0 Parquet files. The `Text` field replaces PDF chunk text.

### 5.3 Embedding Output Schema

Output schema is unchanged (`DocumentRecord` with `file_name`, `chunk_id`, `content`, `embedding [768]float32`) but `file_name` is set to `"{DatasetID}/{Split}/{Index}"`, where `DatasetID` may now be a HuggingFace dataset name, `"knirvserver-ontology"`, or `"arxiv:{category}"` depending on which Stage 0 tier produced the record.

### 5.4 No Chunking for Short Records

If `len(content) ≤ 512` characters, skip sliding-window chunking — emit a single chunk. Chunking remains enabled for longer text. This avoids fragmenting instruction-tuning records and short ontology-entity descriptions unnecessarily.

### 5.5 Output Path

Output to: `~/.local/share/knirvhasher/mapper/mined_records.parquet`

### 5.6 spaCy NLP Bridge: enable by default, fail loud (not silently), stay non-fatal

Two concrete, independently-confirmed root causes — fix both:

**Root cause A — `ensureSpacyInstalled()` never checks for the model, only the package.**
`internal/app/nlp_bridge.go:19-35`:
```go
func ensureSpacyInstalled() {
    if err := exec.Command("python3", "-c", "import spacy").Run(); err == nil {
        return   // <-- BUG: returns here even if en_core_web_sm was never downloaded
    }
    ...
    exec.Command("python3", "-m", "spacy", "download", "--quiet", "en_core_web_sm").Run()
}
```
If `spacy` the Python package is present (e.g. pulled in as a transitive dependency elsewhere) but the `en_core_web_sm` model was never downloaded, this function returns immediately without ever attempting the model download. `NewNLPBridge()`'s retry-once loop then calls `spacy.NewNLP("en_core_web_sm")` a second time, hits the identical `OSError: Can't find model 'en_core_web_sm'` from the Python side, and gives up — surfaced only as a single `log.Printf("Worker %d failed to initialize NLP Bridge: %v", id, err)` in `processor.go:174-179` (and two other call sites). This is exactly the "silent fallback" reported.

**Fix:** change the check to `python3 -c "import spacy; spacy.load('en_core_web_sm')"` (verified locally — `import spacy` alone succeeds even when the model is missing; adding `spacy.load(...)` correctly fails). On failure of *that* check, always run both `pip install spacy` and `spacy download en_core_web_sm` (don't gate the download behind the pip-install branch).

**Root cause B — `libspacy_wrapper.so` is not in the embedded-binary bundle.**
`internal/cli/embedded/binaries.go` embeds `data-connector`, `data-mapper` (post-rename), `data-encoder`, `data-seeder`, `hasher-host`, `libcuda_hash.so` via `go:embed all:bin/*`, but **not** `libspacy_wrapper.so`. `controller.go:578-597` sets `LD_LIBRARY_PATH=<extracted-bin-dir>` for the data-mapper/data-seeder subprocess launch specifically so the CGO-linked go-spacy binding can find its `.so` — but since the `.so` itself is never embedded/extracted into that directory, the dynamic linker can't resolve it there. In the dev tree it works only because `pipeline/1_DATA_MAPPER/spacy/lib/libspacy_wrapper.so` happens to already exist on disk at a path the Makefile's `run*` targets point `LD_LIBRARY_PATH` at directly — that path isn't part of the embedded/extracted production bundle at all. In the real embedded-binary deployment path this is likely a harder failure (the OS can't load the binary's shared-library dependency at `exec`), not something `NewNLPBridge()`'s error handling can even catch gracefully — validate directly (§11, T2) rather than assuming.

**Fix:** add `libspacy_wrapper.so` to `internal/cli/embedded/bin/`, add a corresponding entry to `internal/cli/embedded/binaries.go` (either in `AvailableBinaries` or a lighter-weight "asset" embed list if it shouldn't be listed as a launchable binary), and extract it into `binDir` alongside the pipeline binaries before the data-mapper subprocess launches. The Python-side `spacy` package + `en_core_web_sm` model still need to be available in the target deployment environment — root cause A's fix self-heals this via pip at first run, given outbound network (see Open Questions §14.1 for offline deployment).

**Debug logging (explicit ask):** add structured debug logging around `NewNLPBridge()`/`ensureSpacyInstalled()`:
- Log the exact Python check command, its combined stdout+stderr, and exit code on failure (not just `err`)
- Log `LD_LIBRARY_PATH` / `PYTHONPATH` as seen by the Go process at call time
- Log which of the two attempts (pre-install check vs. post-install retry) failed and why
- Log a single unmistakable `WARN`-level line when the bridge ultimately falls back to nil — e.g. `"⚠️  spaCy NLP unavailable — proceeding WITHOUT POS/Tense/Dependency metadata (Slots 4-5 will be zero-filled); see debug log above for cause"` — so it's impossible to miss in `controller.go`'s `streamPipelineOutput` log stream.
- Keep behavior non-fatal exactly as today (`nlpBridge == nil` continues with empty tokens) — the ask is visibility and default-on behavior, not making it a hard pipeline dependency. Per the architectural warning at the top of this document, missing Slots 4-5 degrades the 21-pass loop's Syntactic Steering rather than crashing it, so non-fatal-but-loud is the correct severity.

---

## 6. Stage 2: DATA_ENCODER — `.nrv` Bracket Encoder

**Package:** `pipeline/2_DATA_ENCODER`
**Goal:** Preserve the 12-slot ASIC format while adapting to Phase 2 `.nrv` output. The LSH Mapper is integrated as a parameterizable component within the existing Semantic Coherence framework, NOT as a replacement. This stage's design is **unaffected** by the Stage 0/1 rework above — it operates purely on Stage 1's stable `DocumentRecord` output schema (§5.3), so it can be implemented independently and in parallel with §3–§5 (see §12).

> **⚠️ CRITICAL: This stage MUST preserve the 12-slot Bitmask Specification from `DATA-MAPPER.md`. The LSH Mapper is a parameterization of the Identity Zone (Slots 0-3), NOT a replacement for the entire semantic architecture.**

### 6.1 Semantic Coherence Mapper with LSH Parameterization

**File:** `pkg/semantic/semantic_mapper.go`

> **⚠️ DO NOT DELETE THE EXISTING MAPPER**: The 12-slot ASIC mapper in `pkg/mapper/mapper.go` is the **Semantic Coherence Mapper**—the core of the 21-pass temporal loop. The LSH Mapper is a **parameterizable projection** that populates only Slots 0-3 (Identity Zone), NOT a replacement for the entire semantic architecture.

The upgrade introduces a new `SemanticMapper` that integrates the LSH projection as a configurable option within the existing 12-slot framework:

**Architecture:**

```
768-dim Embedding
       │
       ▼
┌─────────────────────────┐
│   SemanticMapper       │
│  ┌─────────────────┐   │
│  │ LSH Projection  │───► Slots 0-3 (Identity Zone)
│  │ (16-dim → 64B)  │   │
│  └─────────────────┘   │
│  ┌─────────────────┐   │
│  │ NLP Bridge      │───► Slots 4-5 (Syntactic Registers)
│  │ (POS/Tense)     │   │
│  └─────────────────┘   │
│  ┌─────────────────┐   │
│  │ History XOR     │───► Slots 6-8 (Memory Zone)  
│  └─────────────────┘   │
│  ┌─────────────────┐   │
│  │ Intent Detect   │───► Slot 9 (Intent Flags)
│  └─────────────────┘   │
│  ┌─────────────────┐   │
│  │ Domain Classify │───► Slot 10 (Domain Signature) — feeds §7.4's KNIRVBASE collection key
│  └─────────────────┘   │
│  ┌─────────────────┐   │
│  │ Temporal Lock   │───► Slot 11 (Position + Salt)
│  └─────────────────┘   │
└─────────────────────────┘
       │
       ▼
  12 Slot ASIC Frame (48 bytes → expanded to 80-byte Bracket)
```

**Slot Mapping to Bracket Fields:**

| Bracket Field | Source | Slot(s) | Notes |
|---|---|---|---|
| `LSHSalt` (4B) | Temporal Lock | Slot 11 (partial) | FNV-1a of DatasetID/ChunkID |
| `Projections` (32B) | Identity Zone | Slots 0-3 | LSH projection of 768→16 dims |
| `SubSecondUS` (4B) | Timestamp | N/A | Microsecond within second |
| `ASICLoops` (4B) | Training | N/A | Set to 1 in Stage 2 |
| `GoldenSeed` (4B) | Training | N/A | Set to 0 in Stage 2 |

**Algorithm:**

1. **Semantic Compass (Slots 0-3)**: Project 768-dim embedding to 16-dim via seeded LSH matrix, expand to 64 bytes for Bracket.Projections. This replaces the "Naive Variance" model with principled LSH.
2. **Syntactic Registers (Slots 4-5)**: Extract POS, Tense, Dependency from NLP Bridge (now enabled-by-default per §5.6). These MUST be preserved for the 21-pass loop's Syntactic Steering (Passes 8-14).
3. **Memory Zone (Slots 6-8)**: XOR with previous bracket hash for recurrence.
4. **Intent Flags (Slot 9)**: Detect Question/Command/Code markers.
5. **Domain Signature (Slot 10)**: Classify Math/Code/Prose environment — this classification also becomes the KNIRVBASE collection name in Stage 3 (§7.4).
6. **Temporal Lock (Slot 11)**: Position index + salt for uniqueness.

```go
package semantic

import (
    "encoding/binary"
    "hash/fnv"
    "math"
    "math/rand"
)

type SemanticMapper struct {
    lshMatrix    [16][768]float32
    seed         int64
    useSemantic  bool  // Toggle for semantic coherence mode
    varianceFile string // Path to variance indices (if useSemantic=true)
}

func NewSemanticMapper(seed int64, useSemantic bool) *SemanticMapper {
    m := &SemanticMapper{
        seed:        seed,
        useSemantic: useSemantic,
    }
    // Initialize LSH projection matrix (used for Slots 0-3)
    rng := rand.New(rand.NewSource(seed))
    for i := range m.lshMatrix {
        for j := range m.lshMatrix[i] {
            m.lshMatrix[i][j] = float32(rng.NormFloat64())
        }
    }
    return m
}

// MapToBracket converts a 768-dim embedding to a Bracket via the 12-slot specification
// This preserves the Semantic Coherence required for the 21-pass temporal loop
func (m *SemanticMapper) MapToBracket(embedding []float32, meta *RecordMetadata) *nrv.Bracket {
    // === SLOTS 0-3: IDENTITY ZONE (LSH Projections) ===
    // Maps to Bracket.Projections (32 bytes)
    // CRITICAL: These must be LSH projections, NOT raw variance-selected dims
    projections := m.projectToLSH(embedding)
    
    // === SLOTS 4-5: SYNTACTIC REGISTERS (Linguistic Profile) ===
    // CRITICAL: Must preserve POS, Tense, Dependency for Syntactic Steering
    // These are embedded in the projections for backward-compat, 
    // but Stage 3 trainer reads NLP metadata separately
    syntacticProfile := m.extractSyntacticProfile(meta)
    
    // === SLOTS 6-8: MEMORY ZONE (History XOR) ===
    // Retrieved from Flash Search Associative Jitter Vector
    
    // === SLOTS 9: INTENT FLAGS ===
    intentFlags := m.detectIntentFlags(meta.Text)
    
    // === SLOT 10: DOMAIN SIGNATURE ===
    domainSig := m.classifyDomain(meta.Text)
    
    // === SLOT 11: TEMPORAL LOCK ===
    salt := deriveSalt(meta.DatasetID, meta.ChunkID)
    
    return &nrv.Bracket{
        LSHSalt:     salt,
        Projections: projections,
        SubSecondUS: meta.SubSecondUS,
        GoldenSeed:  0,
    }
}

func (m *SemanticMapper) projectToLSH(embedding []float32) [32]byte {
    var proj [16]float32
    for i := 0; i < 16; i++ {
        var dot float32
        for j := 0; j < 768; j++ {
            dot += embedding[j] * m.lshMatrix[i][j]
        }
        proj[i] = dot
    }
    // L2 normalize
    var norm float32
    for _, v := range proj {
        norm += v * v
    }
    norm = float32(math.Sqrt(float64(norm)))
    if norm > 0 {
        for i := range proj {
            proj[i] /= norm
        }
    }
    // Encode to [32]byte using float16 (half-precision): 16 floats × 2 bytes = 32 bytes
    // Using float16 for LSH projections to fit 16-dim vectors in 32B
    var out [32]byte
    for i, v := range proj {
        // Convert float32 to float16 and pack into 2 bytes
        h := float32ToFloat16(v)
        out[i*2] = byte(h)
        out[i*2+1] = byte(h >> 8)
    }
    return out
}

// float32ToFloat16 converts a float32 to float16 (half-precision)
func float32ToFloat16(f float32) uint16 {
    // IEEE 754 half-precision: 1 sign bit, 5 exponent bits, 10 mantissa bits
    bits := math.Float32bits(f)
    sign := bits >> 31
    exp := int((bits >> 23) & 0xFF)
    mant := bits & 0x7FFFFF
    
    if exp == 0 {
        // Zero or denormal
        return uint16(sign << 15)
    }
    if exp == 255 {
        // Inf or NaN
        return uint16(sign<<15 | 0x7C00)
    }
    
    // Normal number: adjust exponent for half-precision
    newExp := exp - 127 + 15
    if newExp <= 0 {
        // Underflow: denormal or zero
        return uint16(sign << 15)
    }
    if newExp >= 31 {
        // Overflow: infinity
        return uint16(sign<<15 | 0x7C00)
    }
    
    // Round mantissa to 10 bits
    mant >>= 13
    return uint16(sign<<15 | uint32(newExp)<<10 | mant)

// extractSyntacticProfile extracts POS, Tense, Dependency for Slots 4-5
// CRITICAL: This MUST be preserved for the 21-pass loop's Syntactic Steering
func (m *SemanticMapper) extractSyntacticProfile(meta *RecordMetadata) SyntacticProfile {
    return SyntacticProfile{
        POSTag:    meta.POSTag,   // Slot 4, bits 0-7
        Tense:     meta.Tense,     // Slot 4, bits 8-11
        Dependency: meta.DepHead,   // Slot 5
    }
}

// detectIntentFlags determines Slot 9 (Intent Flags)
func (m *SemanticMapper) detectIntentFlags(text string) uint32 {
    var flags uint32
    lower := strings.ToLower(text)
    if strings.Contains(lower, "?") || strings.HasPrefix(lower, "what") {
        flags |= 0x1 // IS_QUESTION
    }
    if strings.HasPrefix(lower, "write") || strings.HasPrefix(lower, "create") {
        flags |= 0x2 // IS_COMMAND
    }
    if strings.Contains(lower, "func ") || strings.Contains(lower, "def ") {
        flags |= 0x4 // IS_CODE
    }
    return flags
}

// classifyDomain determines Slot 10 (Domain Signature)
func (m *SemanticMapper) classifyDomain(text string) uint32 {
    lower := strings.ToLower(text)
    if strings.Contains(lower, "solve") || strings.Contains(lower, "equation") ||
       strings.Contains(lower, "calculate") || strings.ContainsAny(lower, "+-*/=") {
        return 0x2000 // DOMAIN_MATH
    }
    if strings.Contains(lower, "func ") || strings.Contains(lower, "def ") ||
       strings.Contains(lower, "class ") || strings.Contains(lower, "import ") {
        return 0x3000 // DOMAIN_CODE
    }
    return 0x1000 // DOMAIN_PROSE (default)
}

type SyntacticProfile struct {
    POSTag     uint8
    Tense      uint8
    Dependency int16
}

type RecordMetadata struct {
    ID        string
    DatasetID string
    ChunkID   int32
    Text      string
    POSTag    uint8
    Tense     uint8
    DepHead   int16
    SubSecondUS uint32
}
```

### 6.2 v2_schema.yaml — Schema-Driven Bracket Configuration

> **⚠️ MANDATORY**: To prevent architectural drift, all Phase 2 bracket configurations MUST be declared in `v2_schema.yaml`. This ensures the LSH projection can be reconfigured without breaking the 12-slot specification.

**File:** `config/v2_schema.yaml`

```yaml
version: "2.0"
name: "KNIRVHASHER Phase 2 Bracket Schema"

# Bracket field mapping
bracket_fields:
  lsh_salt:
    source: "slot_11"
    encoding: "fnv1a_32"
    format: "{DatasetID}/{ChunkID}"
  
  projections:
    source: "slots_0_3"
    encoding: "lsh_16dim"
    matrix_seed: 1337
    normalize: true
  
  sub_second_us:
    source: "timestamp"
    encoding: "uint32_le"
  
  asic_loops:
    source: "training"
    default: 1
  
  golden_seed:
    source: "training"
    default: 0

# 12-slot to Bracket mapping (REQUIRED for 21-pass loop compatibility)
slot_mapping:
  identity_zone:
    slots: [0, 1, 2, 3]
    bracket_field: "projections"
    description: "LSH projections for semantic compass"
  
  syntactic_registers:
    slots: [4, 5]
    bracket_field: "meta.syntactic"
    description: "POS, Tense, Dependency (REQUIRED for Syntactic Steering)"
    required: true
  
  memory_zone:
    slots: [6, 7, 8]
    bracket_field: "meta.history_xor"
    description: "Recurrent history for temporal loop"
    required: true
  
  intent_flags:
    slot: 9
    bracket_field: "meta.intent"
    description: "Question/Command/Code detection"
  
  domain_signature:
    slot: 10
    bracket_field: "meta.domain"
    description: "Math/Code/Prose environment classification — also the KNIRVBASE collection key (§7.4)"
  
  temporal_lock:
    slot: 11
    bracket_field: "lsh_salt"
    description: "Position index + salt for uniqueness"

# LSH projection configuration
lsh_config:
  input_dim: 768
  output_dim: 16
  normalize: true
  seed: 1337

# Linguistic bridge (REQUIRED for 21-pass loop)
linguistic_bridge:
  enabled: true
  pos_tag_bits: 8
  tense_bits: 4
  dependency_bits: 16
  description: "Preserves syntactic steering in passes 8-14"
```

### 6.3 Bracket Construction (Updated)

For each `DocumentRecord` from Stage 1:

```go
// Load v2 schema for configuration
schema, _ := config.LoadV2Schema("config/v2_schema.yaml")

// Initialize SemanticMapper with schema config
mapper := semantic.NewSemanticMapper(
    schema.LSHConfig.Seed,
    schema.LinguisticBridge.Enabled,
)

// Build metadata from Stage 1 record (includes NLP from spaCy)
meta := &semantic.RecordMetadata{
    ID:        uuid.New().String(),
    DatasetID: record.DatasetID,
    ChunkID:   record.ChunkID,
    Text:      record.Content,
    POSTag:    record.POSTag,   // From spaCy NLP Bridge
    Tense:     record.Tense,
    DepHead:   record.DepHead,
    SubSecondUS: uint32(time.Now().UnixMicro() % 1_000_000),
}

// Map to Bracket via 12-slot specification
bracket := mapper.MapToBracket(record.Embedding, meta)

// CRITICAL: Ensure GoldenSeed and ASICLoops are set correctly
// Stage 3 will update these after training
```

> **⚠️ ATTENTION**: The `meta.POSTag`, `meta.Tense`, and `meta.DepHead` fields MUST be populated from the NLP Bridge (spaCy). If these are missing (i.e. §5.6's fix hasn't landed, or spaCy is still failing), the 21-pass loop will lose Syntactic Steering (Passes 8-14) and collapse to random hashing.

### 6.4 Associative Jitter Vector (Flash Search Integration)

> **⚠️ PRESERVATION MANDATORY**: The `Flash Search` must retrieve the **Associative Jitter Vector** for XOR operations in the temporal loop. This is how the 21-pass recurrence maintains semantic coherence across iterations.

```go
// Flash Search retrieves the Associative Jitter Vector for memory zone (Slots 6-8)
type JitterVector struct {
    HistoryXOR [3]uint32  // Slots 6-8
    Entropy    float32   // Drift score for delta encoding
}

func (m *SemanticMapper) GetJitterVector(hash uint32) JitterVector {
    // Query the Flash Search for neighbor hashes
    // Returns the XOR of previous bracket hashes
    // This drives the 21-pass temporal loop recurrence
    neighbor := flashSearch.FindNearest(hash)
    return JitterVector{
        HistoryXOR: neighbor.HistoryXOR,
        Entropy:    neighbor.DriftScore,
    }
}
```

`GoldenSeed` and `ASICLoops` are left at zero/1 in Stage 2. Stage 3 updates these after ASIC training passes. The `.nrv` file is written first with placeholder values; Stage 3 patches them before submitting to KNIRVBASE.

### 6.5 Frame Assembly via FrameTicker

> **🔗 CROSS-REFERENCE**: Full FrameTicker implementation is in `packages/KNIRVBASE/go/KNIRVBASE_Upgrade.md` Section 4.4. This section summarizes for the pipeline, **with one correction — see §6.5.5**.

The `FrameTicker` orchestrates the Frame/Bracket hierarchy:
1. **Buffers brackets** for 1-second windows
2. **Computes I/P type**: Every 50th bracket is an I-bracket (full projection); others are P-brackets (XOR diff)
3. **Computes drift**: If Euclidean distance between projections > 0.25, forces a new I-bracket
4. **Maintains MemoryZone**: Rolling XOR for Slots 6-8 (temporal loop recurrence)
5. **Flushes to NRVWriter**: Produces FrameEntry in registry + bracket binary in Chunk 1

**FrameTicker Configuration:**

```go
const (
    iFrameInterval = 50    // write an I-bracket every N brackets
    driftThreshold = 0.25  // Euclidean drift that forces a new I-bracket
)

// MemoryZone holds the History XOR state for Slots 6-8 (Memory Zone)
// This is maintained in memory by FrameTicker for the 21-pass temporal loop
type MemoryZone struct {
    HistoryXOR [3]uint32  // Slots 6, 7, 8: Rolling XOR of previous bracket hashes
    SeedXOR    uint32     // Initial seed for recursive hashing
}
```

**Frame Assembly Flow:**

```
Incoming Bracket
       │
       ▼
┌──────────────────┐
│ FrameTicker      │
│ ┌────────────────┴──────────────┐
│ │ Compute MemoryZone (Slots 6-8)│
│ │ historyXOR = rolling XOR     │
│ └────────────────────────────────┘
       │
       ▼
┌──────────────────┐
│ I/P Determination│
│ - If first or Nth→ I-bracket   │
│ - If drift >0.25 → I-bracket   │
│ - Else → P-bracket (XOR diff)  │
└──────────────────┘
       │
       ▼
┌──────────────────┐
│ Append to Buffer │ ──► pending[] + thermoSamples
└──────────────────┘
       │
       │ (every 1 second)
       ▼
┌──────────────────┐
│ Flush            │
│ - Serialize brackets         │
│ - Build FrameEntry           │
│ - Write to NRVWriter         │
│ - Update Registry            │
└──────────────────┘
```

**Instantiate FrameTicker:**

```go
writer, _ := nrvio.NewWriter(outputPath, keyPair)   // see §6.5.5 — local package, not knirvbase/internal
ticker := nrvio.NewFrameTicker(writer, 1*time.Second)

for _, record := range records {
    bracket := buildBracket(record)
    thermo := nrv.ThermoAtmosphere{
        AvgTempC:  0,    // no hardware in Stage 2; set to 0
        PeakVoltV: 0,
        ClockMHz:  0,
    }
    ticker.AppendBracket(bracket, thermo)
    ticker.SetLinguistic(extractToken(record.Content), "word")
}
ticker.Stop() // flushes final partial frame
writer.Close()
```

`extractToken` returns the first whitespace-tokenized word of `record.Content` as a simple linguistic label.

#### 6.5.5 Correction: `NRVWriter`/`FrameTicker` cannot actually be imported from `knirvbase/internal/storage`

The original version of this section's example code called `storage.NewNRVWriter(...)` / `storage.NewFrameTicker(...)` from `github.com/knirvcorp/knirvbase/internal/storage`. **This does not compile** — Go's `internal/` package visibility rule only allows imports from code rooted at the parent of `internal/` (i.e. within `github.com/knirvcorp/knirvbase` itself). `2_DATA_ENCODER` is a separate module and cannot reach it, independent of anything else in this refactor.

This actually resolves cleanly given §7's move to a standalone KNIRVBASE binary: Stage 2 was never meant to hold a live connection to KNIRVBASE anyway (that's Stage 3's job, §7.5) — it only needs to **produce a correctly-formatted `.nrv` file on disk**. Two options, pick one during implementation:

- **(a) Preferred:** give Stage 2 its own small, local `pkg/nrvio` package (new, lives in `2_DATA_ENCODER`) that implements the same on-disk format (Chunk 0 registry JSON + Chunk 1 binary brackets, §2.3) independently — no dependency on `knirvbase` at all, just the shared `.nrv` format spec. This keeps Stage 2 fully decoupled from KNIRVBASE's own module, matching the "one embedded binary, everything else talks to it over the network or not at all" principle in §7.
- **(b) Alternative:** promote the relevant types (`NRVWriter`, `FrameTicker`, `ThermoAtmosphere`) from `internal/storage` to a new public `pkg/nrvio` package inside `knirvcorp/knirvbase` itself, and depend on that public package instead of `internal/`. Slightly less duplication, but re-introduces a build-time Go module dependency from `2_DATA_ENCODER` onto `knirvbase`, which cuts against §7's isolation goal.

Recommend (a) unless the two packages are expected to diverge from the format spec independently over time, in which case the duplication risk of (a) becomes worse than the coupling risk of (b).

### 6.6 Output

`.nrv` files written to: `~/.local/share/knirvhasher/encoder/{dataset_id_slug}_{shard}.nrv`

One file per dataset shard (≤ 10,000 records per file to keep file sizes manageable).

### 6.7 Preserve (NOT Remove)

> **⚠️ CRITICAL CORRECTION**: The following components MUST be preserved. The LSH Mapper does NOT replace them:

- `pkg/mapper/mapper.go` — **KEEP**: The 12-slot Semantic Coherence Mapper is the core of the 21-pass temporal loop
- `pkg/mapper/variance_mapper.go` — **KEEP**: Provides variance-selected dimensions as fallback
- `pkg/schema/output.go` — **KEEP**: TrainingFrame schema for backward compatibility
- `pkg/sliding/` — **KEEP**: Sliding window chunking for long documents
- `pkg/tokenizer/` — **KEEP**: Token counting for ASICLoops estimation
- `pkg/schema/input.go` — **KEEP**: Input schema unchanged

New additions:
- `pkg/semantic/semantic_mapper.go` — Integrates LSH into 12-slot framework
- `pkg/nrvio/` — Local `.nrv` writer/FrameTicker implementation (§6.5.5)
- `config/v2_schema.yaml` — Schema-driven bracket configuration

### 6.8 CLI Flags

```
-input  string   Stage 1 output Parquet (default ~/.local/share/knirvhasher/mapper/mined_records.parquet)
-output string   Output directory for .nrv files (default ~/.local/share/knirvhasher/encoder/)
-shard  int      Max records per .nrv file (default 10000)
-seed   int64    LSH projection matrix seed (default 1337)
-key    string   Path to Dilithium-3 key file for PQC signing (optional)
```

---

## 7. KNIRVBASE: Single Embedded Binary + Arrow Flight Broadcasting

**Packages:** `packages/KNIRVBASE/go` (server binary source), `packages/KNIRVHASHER/internal/cli/embedded` (embed/extract mechanism)
**Goal:** Replace three divergent KNIRVBASE integrations (library-imported into the data connector, a dead custom reimplementation inside KNIRVSERVER, and an implemented-but-never-run Flight server) with **one** binary that's embedded into the KNIRVHASHER bundle and run as a long-lived background service, reachable by every pipeline stage that needs it over the network.

### 7.1 Current state

- `packages/KNIRVBASE/go/internal/network/flight_server.go` **already implements** `FlightServer.StreamBrackets` and Arrow schema/ticket parsing. It is never instantiated by a running server — `cmd/node/main.go` only opens the DB as a library, inserts a couple of demo documents, and blocks on `select{}`. It never starts the Flight server.
- `pipeline/0_DATA_CONNECTOR/cmd/data-connector/main.go:31-36` starts a full in-process KNIRVBASE instance via `knirvbase.New(ctx, ...)` (Go import, `github.com/knirvcorp/knirvbase v1.0.7`) — this is the "started within the data connector" integration.
- `KNIRV_CORP/packages/server/backend_server/internal/database/knirvbase.go` (`KNIRVBASEManager`) is a **completely separate, parallel reimplementation** — markdown+PQC file storage, not the real `knirvcorp/knirvbase` module at all — wired into `cmd/backend_server/main.go:1074` but gated `false` by default (`cfg.Database.UseKNIRVBASE`, `buntdb.go:117`). Dead code in practice.

### 7.2 Target architecture

- Build `packages/KNIRVBASE/go/cmd/node/main.go` (currently a demo/example) out into a real server binary — rename its purpose (keep or rename the `cmd/node` directory, binary output is named `knirvbase`):
  - Opens the DB (`knirvbase.New`/`NewNRV`) — this remains the **only** place in the entire system that calls this constructor.
  - Starts the Arrow Flight service: wrap the existing `FlightServer` type in a real `flight.FlightServiceServer` gRPC implementation (the current type only exposes `StreamBrackets(ticket string, server BracketStreamServer) error`, not the full Arrow Flight gRPC interface — needs a thin adapter implementing `DoGet`/`DoPut`/`GetFlightInfo` etc. that delegates to the existing logic). Listen on the address already assumed by §7.5/§3.8 (`localhost:8815` for Flight, per this document's original config; a separate lightweight gRPC/HTTP port for simple `Collection.Insert/Find/FindAll` calls per §3.10's connector use case).
  - Exposes a minimal client-facing API (`Insert`, `Find`, `FindAll`, `AppendBracket`) so Stage 0 (§3.10) and Stage 3 (§7.5) can reach it over the network instead of importing the package.
- Embed the compiled `knirvbase` binary into KNIRVHASHER using the **exact existing pattern** already proven for `hasher-host`/pipeline binaries in `internal/cli/embedded/binaries.go` (`go:embed all:bin/*`, `AvailableBinaries`, `ExtractBinary`/`GetBinaryPath`) — add:
  ```go
  {
      Name:        "knirvbase",
      Description: "KNIRVBASE — embedded domain-specific .nrv datastore + Arrow Flight broadcaster",
      TargetOS:    "linux",
      TargetArch:  "amd64",
  },
  ```
  to `AvailableBinaries`, and drop the built binary at `internal/cli/embedded/bin/knirvbase`.
- `controller.go` needs a **new, distinct launch path** for `knirvbase` — unlike the four pipeline stages (`data-connector` → `data-mapper` → `data-encoder` → `data-seeder`), which run once per batch and exit, KNIRVBASE is a **long-lived background service**: start it once — alongside `hasher-host`, which already has its own persistent-process handling in `controller.go` (`StartDriver`/`StopDriver` around line 180/757) — before the first pipeline batch runs, and keep it running for the lifetime of the CLI session. All pipeline stages that need it talk to the already-running subprocess over localhost gRPC + Flight.

### 7.3 Removal list

- `github.com/knirvcorp/knirvbase v1.0.7` Go module dependency from `pipeline/0_DATA_CONNECTOR/go.mod` and `pipeline/1_DATA_MAPPER/go.mod` (currently used in `0_DATA_CONNECTOR/cmd/data-connector/main.go:32` and `1_DATA_MAPPER/internal/writer/{writer,arrow_writer}.go`) — replaced by the network client from §7.2.
- `KNIRV_CORP/packages/server/backend_server/internal/database/knirvbase.go` (`KNIRVBASEManager`) entirely, plus its wiring: `cmd/backend_server/main.go:1074`, `internal/database/buntdb.go:50,117,274-430` (`knirvbase` field, `NewKNIRVBASEManager(false, ...)` default construction, `SetKNIRVBASE`, and the `StoreObject`/`GetObject`/`Query` call sites), and the `cfg.Database.UseKNIRVBASE` config flag in `internal/config/config.go:312`. **This is a change in the separate `KNIRV_CORP` git repository** — a follow-up PR there, not part of the `KNIRV_NETWORK` changes in this document. No ordering dependency on anything else here; can happen anytime once someone confirms nothing outside `backend_server` reads `cfg.Database.UseKNIRVBASE`.

### 7.4 Domain-specific `.nrv` collections (Stage 3 write target)

Records land in KNIRVBASE keyed by **domain**, not raw source dataset — using the Slot 10 Domain Signature classification already produced by Stage 2's `SemanticMapper.classifyDomain()` (§6.1/§6.3): collections named `"math"`, `"code"`, `"prose"`, `"academic"`, etc., rather than one collection per HuggingFace dataset name or per arXiv category. This keeps KNIRVBASE's `.nrv` objects organized by semantic content regardless of which of the three §3.1 source tiers (or the security-export connector, §3.6) produced them. See §7.5 for the mechanical submission call.

### 7.5 Stage 3 (`3_DATA_SEEDER`) submission, updated for the standalone binary

**Package:** `pipeline/3_DATA_SEEDER`
**Goal:** After the evolutionary training pass fills in `GoldenSeed` and `ASICLoops` per-bracket, submit the completed `.nrv` files into the running `knirvbase` binary (§7.2) — via its client API for small batches, or Arrow Flight for bulk — keyed by domain (§7.4).

#### Path A: direct-insert client call (default, datasets under ~100k brackets total)

Reads each `.nrv` file (using the same `pkg/nrvio` reader as Stage 2's writer, §6.5.5), iterates brackets, and submits one-by-one through the `knirvbase` binary's client API — **not** an in-process `nrvstorage.Open(...)` call, since that package lives under `knirvbase/internal/` and, per §6.5.5, isn't importable from outside the module anyway:

```go
client, _ := knirvbaseclient.Dial(cfg.KNIRVBase.Addr) // gRPC client against the running knirvbase binary
defer client.Close()

reader, _ := nrvio.Open(nrvFilePath)
for bracket := range reader.StreamBrackets(ctx) {
    thermo := thermoFromBracket(bracket) // reconstruct from bracket metadata
    domain := domainNameFromBracket(bracket) // Slot 10 → "math" / "code" / "prose" / "academic" — see §7.4
    _ = client.AppendBracket(ctx, domain, bracket, thermo)
}
```

#### Path B: Arrow Flight streaming (bulk, datasets over 100k brackets)

Connect to the KNIRVBASE Arrow Flight server (§7.2) and stream bracket batches:

```go
client, _ := flight.NewClientWithMiddleware(knirvbaseFlightAddr, nil, nil)
defer client.Close()

stream, _ := client.DoPut(ctx)
for _, nrvFile := range nrvFiles {
    reader, _ := nrvio.Open(nrvFile)
    schema := bracketArrowSchema()
    writer := flight.NewRecordWriter(stream, ipc.WithSchema(schema))
    for bracket := range reader.StreamBrackets(ctx) {
        writer.Write(bracketToRecord(bracket))
    }
    writer.Close()
}
```

The Flight server handles I/P bracket distinction, `FrameTicker` assembly, and registry updates internally, using the domain-keyed collection from the ticket (`<stream>.<domain>` per the existing `parseTicket` convention in `flight_server.go`).

### 7.6 GoldenSeed + ASICLoops Patch

Before submission, Stage 3 runs the existing vHasher evolutionary training loop against each `.nrv` file's brackets to solve for optimal seeds. The patch writes the solved values back into the Bracket structs before submission:

```
bracket.GoldenSeed = solvedNonce
bracket.ASICLoops  = uint32(actualLoopCount)
```

This preserves the existing evolutionary GRPO logic. The only change is that output goes to the standalone KNIRVBASE binary instead of a CSV/BPF map.

### 7.7 KNIRVBASE Connection Config

Add to the trainer's config JSON:

```json
{
  "knirvbase": {
    "addr": "localhost:50052",
    "flight_addr": "localhost:8815",
    "submission_mode": "direct",
    "key_file": "~/.config/knirvhasher/dilithium.key"
  }
}
```

`submission_mode`: `"direct"` (Path A) or `"flight"` (Path B). Auto-select `"flight"` when bracket count > 100,000. Note this drops the original spec's `data_dir` and `collection_prefix` fields — the data directory now belongs to the standalone `knirvbase` binary's own launch config (§7.2), not the trainer, and the collection name is derived per-bracket from domain (§7.4) rather than a static prefix.

### 7.8 Remove / Deprecate

- BPF map deployment (`deployment.bpf_map_path`) — no longer the sink
- Flash deployment logic and rollback (no longer needed for KNIRVBASE path)
- CSV weight storage (`storage/base_path`) — replaced by `.nrv` files

Keep:
- vHasher simulator and evolutionary GRPO harness (GoldenSeed search is unchanged)
- Checkpoint manager (checkpoints which `.nrv` files have been submitted)
- Cross-hardware validator (still validates bracket binary layout before submission)

---

## 8. New: AlpacaDataCleaned Bootstrap

**Directory:** `pipeline/AlpacaDataCleaned-main` (already present)

This directory contains the Alpaca dataset in a pre-cleaned form. Rather than running it through Stage 0's network connectors, add a **bootstrap script** `scripts/bootstrap_alpaca.sh` that:

1. Converts `AlpacaDataCleaned-main/*.json` → `RawRecord` Parquet using the cleaner logic (§3.7)
2. Writes output to `~/.local/share/knirvhasher/connector/alpaca_bootstrap.parquet`
3. Stages 1–3 then process it identically to ontology/HuggingFace/arXiv data

This lets the pipeline be tested end-to-end with local data before wiring up the network connectors, and is the basis for the end-to-end smoke test in §11.

---

## 9. End-to-End Data Flow

```
KNIRVSERVER /api/ontology/*         HuggingFace Hub                  export.arxiv.org
  (Tier 1: local ontology)      (Tier 2: GOAT/HF datasets)         (Tier 3: exhaustion fallback)
     │                                  │                                  │
     └──────────────┬───────────────────┴──────────────────┬───────────────┘
                    ▼                                      │
            [0_DATA_CONNECTOR]                    (+ existing KNIRVSERVER
     Ontology / HF / arXiv Connectors →             security-export connector,
     Cleaner → Writer                                independent of the 3-tier chain)
     Output: ~/.local/share/knirvhasher/connector/*.parquet (RawRecord)
     Talks to KNIRVBASE binary over network (no in-process import)
                     │
                     ▼
            [1_DATA_MAPPER]  (renamed from 1_DATA_MINER)
     RawRecord → Embed (Ollama/Cloudflare) → NLP Bridge (spaCy, enabled by default)
     Output: mined_records.parquet (text + embedding + POS/Tense/DepHead)
                     │
                     ▼
            [2_DATA_ENCODER]
     DocumentRecord → SemanticMapper (12-slot Spec → LSH Projections)
     ├─ Slots 0-3: LSH Projections (64B) ← Identity Zone
     ├─ Slots 4-5: POS + Tense + Dependency ← Syntactic Registers (REQUIRED)
     ├─ Slots 6-8: History XOR ← Memory Zone (Flash Search Jitter)
     ├─ Slot 9: Intent Flags ← Question/Command/Code
     └─ Slot 10: Domain Signature ← Math/Code/Prose (→ KNIRVBASE collection key)
     └─ Slot 11: Temporal Lock ← Position + Salt
     FrameTicker (1s windows) → local pkg/nrvio writer (§6.5.5)
     Output: ~/.local/share/knirvhasher/encoder/{dataset}_{shard}.nrv
                     │
                     ▼
            [3_DATA_SEEDER]
     vHasher GRPO (21-pass loop with Syntactic Steering)
     ├─ Passes 1-7: Anchor check (Slot 0)
     ├─ Passes 8-14: Syntactic Steering (Slots 4-5) ← CRITICAL
     └─ Passes 15-21: Entropy resolution (Slot 3)
     Patch GoldenSeed + ASICLoops per Bracket
     Client RPC (direct) / Arrow Flight (bulk) → domain-keyed collection
                     │
                     ▼
      standalone `knirvbase` binary (embedded in + extracted from KNIRVHASHER bundle)
   NRVStorage → FrameTicker → NRVWriter → .nrv on disk, organized by domain
   Arrow Flight server (§7.2) → streaming reads for consumers
```

---

## 10. Frame/Bracket Architecture Deep Dive

> **🔗 CROSS-REFERENCE**: Full specification in `packages/KNIRVBASE/go/KNIRVBASE_Upgrade.md` Section 3 (Binary Format) and Section 4.4 (FrameTicker). Unchanged by this document except for the domain-keying note in §7.4 and the local-writer correction in §6.5.5.

### 10.1 Why Frames?

The **1-second Frame** serves as the temporal container for the ASIC pipeline:
- **Batching**: Groups multiple bracket submissions into a single atomic write
- **Delta Compression**: Enables I/P bracket distinction (full vs XOR-diff)
- **Temporal Context**: Each Frame carries `timestamp_unix` and `linguistic` metadata
- **Hardware Metrics**: Each Frame records `ThermoAtmosphere` (temp/voltage/clock)
- **Z3 Validation**: Each Frame gets a `Z3Result` determining Gold/Research stream eligibility

### 10.2 FrameTicker Behavior

The `FrameTicker` runs as a 1-second goroutine:

| Tick | Action |
|---|---|
| T+0s | Start ticker, empty buffer |
| T+0.5s | Append 150 brackets (50 I, 100 P) |
| T+1s | **Flush**: Write FrameEntry to registry, brackets to Chunk 1 |
| T+1.5s | Append 200 brackets |
| T+2s | **Flush**: Write FrameEntry to registry, brackets to Chunk 1 |
| ... | ... |
| On Stop | Flush final partial frame (≤1 second of data) |

### 10.3 I-Bracket vs P-Bracket Decision

```
Every bracket enters FrameTicker.AppendBracket():

1. Is this the FIRST bracket in the frame?
   YES → I-bracket (full projection)
   NO  → Continue

2. Is bracketCount % 50 == 0? (iFrameInterval)
   YES → I-bracket (forced anchor)
   NO  → Continue

3. Compute Euclidean drift: distance(current.projections, lastI.projections)
   IF drift > 0.25 → I-bracket (drift spike detected)
   ELSE → P-bracket (XOR diff against lastI)

For P-brackets:
   - Store XOR(current.projections, lastI.projections) in binary
   - Store anchor_id = lastI.ID in bracket_index
```

### 10.4 MemoryZone (Slots 6-8)

The `MemoryZone` is maintained **in memory** (NOT in the 80-byte bracket):

```go
type MemoryZone struct {
    HistoryXOR [3]uint32  // Slots 6, 7, 8
    SeedXOR    uint32     // Initial seed
}

// Computed on each AppendBracket:
func computeMemoryXOR(b *Bracket) [3]uint32 {
    slot6 := previous.HistoryXOR[0] ^ hash(b)
    slot7 := previous.HistoryXOR[1] ^ slot6
    slot8 := previous.HistoryXOR[2] ^ slot7
    return [3]uint32{slot6, slot7, slot8}
}
```

This provides the **temporal recurrence** for the 21-pass loop:
- Slot 6: XOR of current hash with previous Slot 6
- Slot 7: XOR of Slot 6 with previous Slot 7 (deepens recurrence)
- Slot 8: XOR of Slot 7 with previous Slot 8 (deepest recurrence)

### 10.5 Registry Structure

Each `.nrv` file's Chunk 0 (Resolution Registry) contains:

```json
{
  "version": 1,
  "dataset_id": "tatsu-lab/alpaca",
  "frame_count": 60,
  "global_metrics": {
    "total_bracket_count": 18432,
    "valid_frame_count": 57,
    "invalid_frame_count": 3
  },
  "frames": [
    {
      "id": "frame-uuid-0",
      "timestamp_unix": 1712580120,
      "linguistic": {"token": "resolution", "unit": "word"},
      "thermo": {"avg_temp_c": 71.4, "peak_volt_v": 1.37, "clock_mhz": 553.0},
      "z3": {"status": "VALID", "relevance": 0.92},
      "brackets": {"count": 307, "offset": 5242892, "length": 24560},
      "bracket_index": [
        {"id": "b0", "type": "I", "anchor_id": null, "offset": 0, "drift_score": 0.0},
        {"id": "b1", "type": "P", "anchor_id": "b0", "offset": 80, "drift_score": 0.014}
      ]
    }
  ]
}
```

### 10.6 Stage 2 → Stage 3 Handoff

**Stage 2 Output:**
- `.nrv` files with Brackets in Chunk 1
- Registry in Chunk 0 with FrameEntries
- `GoldenSeed = 0`, `ASICLoops = 1` (placeholders)

**Stage 3 Input:**
- Reads `.nrv` files via the local `pkg/nrvio` reader (§6.5.5)
- Runs vHasher GRPO to find optimal `GoldenSeed` per bracket
- Updates `ASICLoops` based on actual pass count used
- Re-encodes brackets with solved values
- Submits to the standalone `knirvbase` binary via client RPC or Arrow Flight (§7.5), domain-keyed (§7.4)

> **⚠️ NOTE**: Stage 3 does NOT re-run FrameTicker. The Frame structure is preserved from Stage 2. Only the `GoldenSeed` and `ASICLoops` fields are patched in-place.

### 10.7 Common `types` Package

Create `pipeline/types/types.go` (imported by all 4 stages):

```go
package types

// RawRecord is the canonical record emitted by Stage 0 and consumed by Stage 1.
type RawRecord struct {
    DatasetID string
    Split     string
    Index     int64
    Text      string
    Tags      []string
}

// MinedRecord is the Stage 1 output: a text chunk with its embedding.
// Identical to the existing DocumentRecord — aliased here for cross-stage clarity.
type MinedRecord struct {
    FileName  string    // "{DatasetID}/{Split}/{Index}"
    ChunkID   int32
    Content   string
    Embedding []float32 // 768-dim
}
```

### 10.8 Parquet Column Naming

All inter-stage Parquet files use snake_case column names matching the struct tags in `types.go`. No change to existing `DocumentRecord` column names — Stage 1 output is backward-compatible.

---

## 11. Testing Requirements

### 11.1 Stage 0
- Unit test: `TestHuggingFaceNormalization` — verify each column-normalization rule against a mock dataset schema
- Unit test: `TestOntologyTierSelection` — mock KNIRVSERVER `/api/ontology/stats` returning `entityCount: 0` → verify connector falls through to HF; verify `entityCount > 0` → verify connector prefers ontology
- Unit test: `TestHFExhaustionFallback` — mock HF returning an empty `rows` page → verify connector falls through to arXiv
- Integration test: download the first 100 rows of `tatsu-lab/alpaca` and verify `RawRecord.Text` is non-empty
- Integration test (T2 below, spaCy-adjacent but validated here since it's the embedded-extraction path): run `EnsureExtracted()` against a clean app-data directory and confirm `data-connector`, `data-mapper`, `data-encoder`, `data-seeder`, `knirvbase`, and `libspacy_wrapper.so` are all present in the extracted `binDir`

### 11.2 Stage 1 (`data-mapper`)
- Unit test: corrected `ensureSpacyInstalled()` check (`import spacy; spacy.load('en_core_web_sm')`) against a mocked "spacy present, model absent" Python environment (simulate via `PYTHONPATH` pointing at a stub `spacy` module without `en_core_web_sm`, or by testing the command string construction directly)
- Integration test: run the *actual* embedded-extraction path end-to-end — launch `data-mapper` from the extracted `binDir` with only `LD_LIBRARY_PATH=binDir` set, no dev-tree paths — and confirm `libspacy_wrapper.so` resolves and NLP is active. This is the scenario that's untested today and most likely to be the actual production failure mode (§5.6, Root cause B).
- Regression test: confirm `1_DATA_MAPPER` builds and runs with zero references to `internal/arxiv/*` or `RunGoatMiningPhase`-equivalent code (i.e. the source-fetch removal in §5.2 is complete, not partially done)

### 11.3 Stage 2
> **⚠️ MANDATORY**: These tests verify the 12-slot Bitmask Specification is preserved.

- Unit test: `TestLSHMapperReproducibility` — same seed + same embedding → identical `[64]byte`
- Unit test: `TestSemanticMapperSlotPreservation` — verify all 12 slots are populated per the bitmask spec
- Unit test: `TestSyntacticProfileExtraction` — verify POS, Tense, Dependency are extracted for Slots 4-5
- Unit test: `TestBracketEncodeDecodeRoundTrip` — encode → decode → field equality, using the new local `pkg/nrvio` (§6.5.5), not an `internal/` import
- Unit test: `TestFrameTickerFlush` — insert 350 brackets over 2.5 seconds → verify 2 full frames + 1 partial flushed on Stop
- Integration test: `Test21PassLoopIntegrity` — verify the SemanticMapper output can drive the 21-pass loop with proper Syntactic Steering

### 11.4 Stage 3
- Unit test: `TestGoldenSeedPatch` — verify bracket `GoldenSeed` is non-zero after one GRPO pass
- Unit test: `TestSyntacticSteeringInTraining` — verify Passes 8-14 use Slot 4-5 for grammatical validation
- Unit test: `TestDomainCollectionKeying` — verify a bracket with Slot 10 = `DOMAIN_MATH` is submitted to the `"math"` collection, not a dataset-ID-keyed collection
- Integration test: submit 1000 brackets via the client RPC path → verify a Flight client reading the same domain collection returns them all (§7.5 Path A + §7.2's Flight service, same data visible both ways)

### 11.5 KNIRVBASE (new)
- Integration test: start the standalone `knirvbase` binary, connect an Arrow Flight client to `localhost:8815`, and verify `StreamBrackets` returns previously-submitted brackets for the expected domain collection (T5 from the refactor plan)
- Regression test: confirm `go.mod` in `0_DATA_CONNECTOR` and `1_DATA_MAPPER` no longer requires `github.com/knirvcorp/knirvbase`, and that both packages build/run without it present in the module cache (T6)

### 11.6 Controller / pipeline sequencing
- Integration test: run the default pipeline profile end-to-end and assert `data-connector` is the first stage to execute for every batch (T4 — validates §4's controller fix actually landed and stayed landed)

### 11.7 End-to-End
- Bootstrap test using `AlpacaDataCleaned-main` (§8) → run all 4 stages against the now-running `knirvbase` binary → assert `.nrv` file is valid (magic bytes, non-zero frame count, all brackets 80-byte aligned) and the domain-keyed collection in KNIRVBASE contains the expected record count
- **CRITICAL**: Verify the 21-pass loop produces deterministic consensus when given semantically coherent brackets

---

## 12. Implementation Order

Unlike the original single-track ordering, this merged plan has two kinds of dependency: **hard blockers** (must land in order) and **parallelizable tracks** (independent, can proceed simultaneously). The table below replaces both the original document's Stage-2-first ordering and the refactor plan's separate ordering — they're reconciled here into one sequence.

| Phase | Work | Depends on | Rationale |
|---|---|---|---|
| **0** | **spaCy fix** (§5.6: root cause A + B, debug logging) | Nothing | Self-contained, highest value/lowest risk, fixable independent of every other phase. Do first. |
| **0** | **KNIRV_CORP dead `KNIRVBASEManager` removal** (§7.3, second bullet) | Nothing | Different repo, zero coupling to anything in this document. Runs on its own track, anytime. |
| **1** | **Rename `data-miner` → `data-mapper`** (§5.1) | Nothing (but should land before Phase 3, see below) | Mechanical; unblocks clean diffs for every subsequent change that touches this package. |
| **1** | **Stage 2 SemanticMapper / LSH work** (§6.1-§6.8, including the §6.5.5 local-writer correction) | Nothing — operates only on Stage 1's stable output schema (§5.3), which the rename and source-fetch migration don't change | Matches the original document's own reasoning for starting here: can be built and tested against existing Stage 1 output without needing Stage 0 changes. Runs in parallel with Phase 1's rename and Phase 2 below. |
| **2** | **Single embedded KNIRVBASE binary** (§7.1-§7.2) | Nothing hard-blocking, but logically should exist before Phases 4 and 7 need to point at it | Foundational infrastructure — both Stage 0's removal of its in-process DB (§3.10) and Stage 3's new ingestion path (§7.5) need something running to connect to. |
| **3** | **Migrate HF + arXiv connectors into Stage 0** (§3.4, §3.5) | Phase 1 (rename) — do this on the already-renamed `1_DATA_MAPPER` tree | Moves `RunGoatMiningPhase` and `internal/arxiv/*` out of the (now-renamed) mapper package and into the connector. |
| **3** | **Stage 0 → KNIRVBASE client swap** (§3.10) | Phase 2 (binary must exist to connect to) | Removes `data-connector`'s in-process `knirvbase.New()` call. |
| **4** | **Local ontology connector** (§3.2) + **three-tier priority chain** (§3.1, §3.8 config) | Phase 3 (HF/arXiv connectors must already be relocated before they can be slotted into a priority chain alongside ontology) | Builds the actual tier-selection state machine on top of the now-relocated connectors. |
| **5** | **Flip `controller.go` default pipeline to start at Stage 0** (§4) | Phase 4 | Only makes sense once Stage 0 does the right thing end-to-end — flipping earlier just makes the old security-export-only connector the default "stage 0" while source logic is still mid-migration. |
| **6** | **Stage 3: KNIRVBASE ingestion via standalone binary client** (§7.5-§7.8) + **domain-specific collection naming** (§7.4) | Phase 2 (binary running) AND Phase 1's parallel Stage 2 track (needs `classifyDomain()` to exist to derive the collection key) | Both prerequisites converge here — this is the join point between the two parallel tracks above. |
| **7** | **AlpacaDataCleaned bootstrap wiring** (§8) | Phases 3-6 substantially complete (needs all 4 stages in their new shape to smoke-test against) | Used as the primary end-to-end validation harness before declaring the refactor done. |
| **8** | **Full test suite** (§11) | All prior phases for the tests that exercise them; individual test groups (11.1-11.6) can and should be written incrementally alongside their corresponding phase rather than saved entirely for the end | §11.7's end-to-end test is the final gate and depends on everything. |

**Two genuinely parallel tracks worth calling out explicitly:**
- **Track A (Stage 0/1 restructure):** Phase 1 (rename) → Phase 3 (connector migration) → Phase 4 (ontology + priority chain) → Phase 5 (controller flip)
- **Track B (Stage 2/3 + KNIRVBASE):** Phase 1 (SemanticMapper) + Phase 2 (KNIRVBASE binary) in parallel → Phase 6 (Stage 3 ingestion, joins both)

Track A and Track B don't block each other until Phase 6, where Track B's domain classifier (from its Phase 1 sub-track) and Phase 2's running binary both become inputs to Stage 3's new write path. A two-person/two-session split along these tracks is reasonable if parallelizing the implementation work itself.

---

## 13. Open Questions

1. **Offline/air-gapped deployment:** §5.6's self-healing `pip install spacy && spacy download en_core_web_sm` needs outbound network on first run. If KNIRVHASHER needs to run fully offline, the model needs to be vendored into the embedded bundle instead (adds real size — `en_core_web_sm` is ~12-15MB compressed). Confirm deployment constraints before deciding.

Answer: There is no offline deployment constraint. Download the module on first run as needed. Ensure the pipeline waits for it to complete before proceeding.

2. **Ontology "availability" threshold:** is any `entityCount > 0` enough to prefer local ontology over HF (§3.1/§3.8's `min_entity_count: 1` placeholder), or should there be a minimum record count / freshness check before treating it as a viable batch source? Needs a product decision, not just an engineering one.

Answer: Yes, anything greater than zero records will suffice, as long as the pipeline loops to the next source once the current source has been exhausted.

3. **KNIRVBASE consensus/network config for the standalone binary:** today `knirvbase.Options{DistributedEnabled, DistributedNetworkID, BootstrapPeers}` are set by whichever caller constructs the `DB` in-process. Once it's a standalone binary (§7.2), these become CLI flags/config for that binary — need to decide whether KNIRVHASHER's embedded KNIRVBASE joins the same distributed network as other KNIRVBASE deployments in the fleet or runs standalone per-hasher-node.

Answer: Yes, The KNIRVHASHER's embedded KNIRVBASE joins the same distributed network as other KNIRVBASE deployments in the fleet with the same DistributedNetworkID.

4. **KNIRV_CORP follow-up PR:** §7.3's second removal item is out of this repo's scope by definition (separate git repo) — needs its own review/PR in `KNIRV_CORP`, referenced from here but not executed as part of this plan.

Answer: Thanks, I'll create a separate PR for the KNIRVBASE cleanup in KNIRVSERVER.

5. **`pkg/nrvio` duplication vs. coupling (§6.5.5):** decide between a fully independent local `.nrv` writer in `2_DATA_ENCODER` (duplicated format logic, zero coupling) versus promoting the relevant `knirvbase/internal/storage` types to a public `pkg/nrvio` inside the `knirvbase` module (shared logic, re-introduces a Go module dependency from the encoder onto KNIRVBASE). Recommendation in §6.5.5 leans toward the independent option; confirm before implementing.

Answer: Lets go the independent route with key comments mentioning the sync requirement if changes are made in the future.
