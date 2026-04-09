# KNIRVHASHER Pipeline Upgrade: HuggingFace → `.nrv` → KNIRVBASE

**Status:** Planning — Phase 2 of KNIRVBASE Upgrade is the target spec.
**Source spec:** `packages/KNIRVBASE/go/KNIRVBASE_Upgrade.md` (Phase 2, ASIC-Native)
**Pipeline root:** `packages/KNIRVHASHER/pipeline/`

---

## 1. Overview

The current KNIRVHASHER pipeline mines PDF/arXiv documents, encodes embeddings into 12-slot ASIC Parquet frames, and runs evolutionary training to find optimal SHA-256 seeds. The output stays local to the pipeline.

This upgrade transforms the pipeline into a **KNIRVBASE data ingestion system**: it pulls datasets from HuggingFace, encodes them into the Phase 2 `.nrv` binary format (80-byte Brackets inside 1-second Frames), and writes them directly into KNIRVBASE via the Go SDK.

### 1.1 Before and After

| Stage | Current Output | New Output |
|---|---|---|
| 0_DATA_CONNECTOR | CSV / local files | HuggingFace datasets (Parquet streaming) |
| 1_DATA_MINER | `ai_knowledge_base.parquet` (768-dim embeddings) | `mined_records.parquet` (text + LSH-ready embeddings) |
| 2_DATA_ENCODER | `training_frames.parquet` (12 ASIC slots) | `.nrv` files (80-byte Brackets, Phase 2 format) |
| 3_DATA_TRAINER | BestSeed CSV / BPF weights | KNIRVBASE ingest via `AppendBracketDirect` + Arrow Flight |

### 1.2 Key Terminology (from KNIRVBASE Phase 2 Spec)

| Term | Meaning |
|---|---|
| **Bracket** | 80-byte binary record: LSH Salt (4B) + Projections A–H (64B) + Metadata (8B) + Golden Seed (4B) |
| **Frame** | 1-second temporal window holding N Brackets + thermo + linguistic metadata |
| **I-Bracket** | Absolute (intra) bracket — stores full LSH projections |
| **P-Bracket** | Delta bracket — stores XOR-diff of projections against anchor I-Bracket |
| **FrameTicker** | 1-second goroutine that flushes buffered Brackets into a Frame via `NRVWriter.AppendFrame` |

---

## 2. Stage 0: DATA_CONNECTOR — HuggingFace Ingestor

**Package:** `pipeline/0_DATA_CONNECTOR`
**Goal:** Replace or extend the existing connector to pull datasets from HuggingFace Hub, normalize them to a standard internal schema, and hand off to Stage 1.

### 2.1 New: HuggingFace Connector (`connector/huggingface.go`)

Add a `HuggingFaceConnector` alongside the existing connectors.

**Responsibilities:**
- Authenticate to HuggingFace Hub (token via `HF_TOKEN` env var)
- Enumerate dataset splits (train / validation / test)
- Stream dataset shards as Parquet from the HuggingFace CDN — no full download required
- Support datasets with `text`, `input`/`output`, `instruction`/`response`, or `messages` (chat) columns
- Emit records conforming to the shared `RawRecord` schema below

**`RawRecord` (internal canonical schema):**

```go
type RawRecord struct {
    DatasetID string   // e.g. "tatsu-lab/alpaca"
    Split     string   // "train", "test", etc.
    Index     int64    // row index within the shard
    Text      string   // normalized plain text (instruction + response joined)
    Tags      []string // optional labels / categories from dataset metadata
}
```

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

### 2.2 Update: Cleaner (`connector/cleaner/`)

Extend the existing cleaner to handle:
- Unicode normalization (NFC)
- Remove HTML tags / markdown artifacts
- Deduplicate records by hashed `Text` content (bbolt checkpoint, same as current miner)
- Min/max length filter: skip records with `len(Text) < 32` or `len(Text) > 16384`

### 2.3 Config File Update (`config/`)

Add `huggingface` section to the connector config YAML:

```yaml
huggingface:
  datasets:
    - tatsu-lab/alpaca
    - Open-Orca/OpenOrca
    - HuggingFaceH4/ultrachat_200k
  splits: [train]
  max_rows_per_dataset: 50000
  shard_workers: 4
  cache_dir: ~/.cache/knirvhasher/hf
```

### 2.4 Writer Update (`connector/writer/`)

The writer must output `RawRecord` as Parquet shards to `~/.local/share/knirvhasher/connector/` for Stage 1 to consume. Keep the existing bbolt checkpoint to avoid reprocessing rows.

---

## 3. Stage 1: DATA_MINER — Embedding Generator

**Package:** `pipeline/1_DATA_MINER`
**Goal:** Transform `RawRecord` Parquet shards into embedding records with 768-dim vectors, output as `mined_records.parquet`. The embedding step is unchanged; what changes is the input source (now includes HuggingFace via Stage 0 output) and the output path convention.

### 3.1 Input Change

Add a new input source alongside the existing arXiv/PDF paths:

```
Primary:   ~/.local/share/knirvhasher/connector/*.parquet  (from Stage 0)
Fallback:  ~/.local/share/data-miner/ai_knowledge_base.parquet (legacy)
```

The miner reads `RawRecord` from Stage 0 Parquet files. The `Text` field replaces PDF chunk text.

### 3.2 Embedding Output Schema

Output schema is unchanged (`DocumentRecord` with `file_name`, `chunk_id`, `content`, `embedding [768]float32`) but `file_name` is set to `"{DatasetID}/{Split}/{Index}"` for HuggingFace records.

### 3.3 No Chunking for Short Records

If `len(content) ≤ 512` characters, skip sliding-window chunking — emit a single chunk. Chunking remains enabled for longer text. This avoids fragmenting instruction-tuning records unnecessarily.

### 3.4 Output Path

Output to: `~/.local/share/knirvhasher/miner/mined_records.parquet`

---

## 4. Stage 2: DATA_ENCODER — `.nrv` Bracket Encoder

**Package:** `pipeline/2_DATA_ENCODER`
**Goal:** Replace the 12-slot ASIC Parquet output with Phase 2 `.nrv` files. Each mined embedding record becomes one or more 80-byte Brackets, grouped into 1-second Frames via `FrameTicker`, and written to `.nrv` via `NRVWriter`.

This is the most significant stage rewrite.

### 4.1 Dependency: KNIRVBASE Go SDK

Add to `go.mod`:

```
require github.com/knirvcorp/knirvbase/go v1.0.1  // replace with local path during dev
```

Replace directive (during development):

```
replace github.com/knirvcorp/knirvbase/go => ../../../KNIRVBASE/go
```

### 4.2 LSH Projection Mapper (replaces 12-slot ASIC mapper)

**New file:** `pkg/lshmap/lshmap.go`

Maps a 768-dim `[]float32` embedding to the 64-byte `Projections` field of a `Bracket` (16 × float32 = 64 bytes = the LSH projection array).

**Algorithm:**

1. **Project**: Multiply the 768-dim embedding by a fixed random projection matrix `R ∈ ℝ^{16×768}` (seeded with `LSHSeed` config value, generated once at startup using `math/rand` with the seed).
2. **Normalize**: L2-normalize the 16-dim result vector.
3. **Encode**: Cast each `float32` to its 4-byte little-endian representation, pack into `[64]byte`.

```go
package lshmap

import (
    "encoding/binary"
    "math"
    "math/rand"
)

const ProjectionDim = 16 // LSH output dimensions
const EmbeddingDim  = 768

type LSHMapper struct {
    matrix [ProjectionDim][EmbeddingDim]float32 // random projection matrix
}

func NewLSHMapper(seed int64) *LSHMapper {
    rng := rand.New(rand.NewSource(seed))
    m := &LSHMapper{}
    for i := range m.matrix {
        for j := range m.matrix[i] {
            m.matrix[i][j] = float32(rng.NormFloat64())
        }
    }
    return m
}

// Map projects a 768-dim embedding into a [64]byte Bracket projection field.
func (m *LSHMapper) Map(embedding []float32) [64]byte {
    var proj [ProjectionDim]float32
    for i := 0; i < ProjectionDim; i++ {
        var dot float32
        for j, v := range embedding {
            dot += m.matrix[i][j] * v
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
    // Encode to [64]byte
    var out [64]byte
    for i, v := range proj {
        binary.LittleEndian.PutUint32(out[i*4:], math.Float32bits(v))
    }
    return out
}
```

### 4.3 LSH Salt Derivation

The `LSHSalt` (uint32) is the lower 32 bits of a FNV-1a hash of `"{DatasetID}/{ChunkID}"`. This gives a deterministic, record-specific salt:

```go
import "hash/fnv"

func deriveSalt(datasetID string, chunkID int32) uint32 {
    h := fnv.New32a()
    h.Write([]byte(fmt.Sprintf("%s/%d", datasetID, chunkID)))
    return h.Sum32()
}
```

### 4.4 Bracket Construction

For each `DocumentRecord` from Stage 1:

```go
bracket := &nrv.Bracket{
    ID:          uuid.New().String(),
    LSHSalt:     deriveSalt(record.FileName, record.ChunkID),
    Projections: lshMapper.Map(record.Embedding),
    SubSecondUS: uint32(time.Now().UnixMicro() % 1_000_000),
    ASICLoops:   1,      // Stage 2 sets 1; Stage 3 (trainer) updates to actual loop count
    GoldenSeed:  0,      // Stage 2 sets 0; Stage 3 fills in the solved nonce
}
```

`GoldenSeed` and `ASICLoops` are left at zero/1 in Stage 2. Stage 3 updates these after ASIC training passes. The `.nrv` file is written first with placeholder values; Stage 3 patches them via `NRVWriter` before submitting to KNIRVBASE.

### 4.5 Frame Assembly via FrameTicker

Instantiate one `FrameTicker` per output `.nrv` file (one file per dataset shard):

```go
writer, _ := storage.NewNRVWriter(outputPath, keyPair)
ticker := storage.NewFrameTicker(writer, 1*time.Second)

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

### 4.6 Output

`.nrv` files written to: `~/.local/share/knirvhasher/encoder/{dataset_id_slug}_{shard}.nrv`

One file per dataset shard (≤ 10,000 records per file to keep file sizes manageable).

### 4.7 Remove

- `pkg/schema/output.go` (TrainingFrame Parquet schema) — no longer needed
- `pkg/sliding/` — sliding window is handled in Stage 1 now
- The 12-slot ASIC mapper in `pkg/mapper/mapper.go` — replaced by `pkg/lshmap/lshmap.go`

Keep and re-use:
- `pkg/tokenizer/` — still used to derive `ASICLoops` placeholder from token count
- `pkg/schema/input.go` — input schema unchanged

### 4.8 CLI Flags

```
-input  string   Stage 1 output Parquet (default ~/.local/share/knirvhasher/miner/mined_records.parquet)
-output string   Output directory for .nrv files (default ~/.local/share/knirvhasher/encoder/)
-shard  int      Max records per .nrv file (default 10000)
-seed   int64    LSH projection matrix seed (default 1337)
-key    string   Path to Dilithium-3 key file for PQC signing (optional)
```

---

## 5. Stage 3: DATA_TRAINER → KNIRVBASE Ingestor

**Package:** `pipeline/3_DATA_TRAINER`
**Goal:** After the evolutionary training pass fills in `GoldenSeed` and `ASICLoops` per-bracket, submit the completed `.nrv` files to KNIRVBASE via `AppendBracketDirect` (single-bracket path) or Arrow Flight bulk streaming.

### 5.1 Two Submission Paths

#### Path A: `AppendBracketDirect` (default)

For datasets under ~100k brackets total. Reads each `.nrv` file, iterates brackets, submits one-by-one through the KNIRVBASE `NRVStorage.AppendBracketDirect` method:

```go
storage, _ := nrvstorage.Open(knirvbaseDataDir, keyPair)

reader, _ := nrvreader.Open(nrvFilePath)
for bracket := range reader.StreamBrackets(ctx) {
    thermo := thermoFromBracket(bracket) // reconstruct from bracket metadata
    _ = storage.AppendBracketDirect(collectionName, bracket, thermo)
}
```

`collectionName` = `"{DatasetID}"` (e.g. `"tatsu-lab/alpaca"`).

#### Path B: Arrow Flight Streaming (bulk, large datasets)

For datasets over 100k brackets. Connect to the KNIRVBASE Arrow Flight server (`internal/network/flight_server.go`) and stream bracket batches:

```go
client, _ := flight.NewClientWithMiddleware(knirvbaseFlightAddr, nil, nil)
defer client.Close()

stream, _ := client.DoPut(ctx)
for _, nrvFile := range nrvFiles {
    reader, _ := nrvreader.Open(nrvFile)
    schema := bracketArrowSchema()
    writer := flight.NewRecordWriter(stream, ipc.WithSchema(schema))
    for bracket := range reader.StreamBrackets(ctx) {
        writer.Write(bracketToRecord(bracket))
    }
    writer.Close()
}
```

The Flight server handles I/P bracket distinction, `FrameTicker` assembly, and registry updates internally.

### 5.2 GoldenSeed + ASICLoops Patch

Before submission, Stage 3 runs the existing vHasher evolutionary training loop against each `.nrv` file's brackets to solve for optimal seeds. The patch writes the solved values back into the Bracket structs before `AppendBracketDirect`:

```
bracket.GoldenSeed = solvedNonce
bracket.ASICLoops  = uint32(actualLoopCount)
```

This preserves the existing evolutionary GRPO logic. The only change is that output goes to KNIRVBASE instead of a CSV/BPF map.

### 5.3 KNIRVBASE Connection Config

Add to the trainer's config JSON:

```json
{
  "knirvbase": {
    "data_dir": "~/.local/share/knirvbase",
    "flight_addr": "localhost:8815",
    "submission_mode": "direct",
    "collection_prefix": "hasher",
    "key_file": "~/.config/knirvhasher/dilithium.key"
  }
}
```

`submission_mode`: `"direct"` (AppendBracketDirect) or `"flight"` (Arrow Flight). Auto-select `"flight"` when bracket count > 100,000.

### 5.4 Remove / Deprecate

- BPF map deployment (`deployment.bpf_map_path`) — no longer the sink
- Flash deployment logic and rollback (no longer needed for KNIRVBASE path)
- CSV weight storage (`storage/base_path`) — replaced by `.nrv` files

Keep:
- vHasher simulator and evolutionary GRPO harness (GoldenSeed search is unchanged)
- Checkpoint manager (checkpoints which `.nrv` files have been submitted)
- Cross-hardware validator (still validates bracket binary layout before submission)

---

## 6. New: AlpacaDataCleaned Bootstrap

**Directory:** `pipeline/AlpacaDataCleaned-main` (already present)

This directory contains the Alpaca dataset in a pre-cleaned form. Rather than running it through Stage 0, add a **bootstrap script** `scripts/bootstrap_alpaca.sh` that:

1. Converts `AlpacaDataCleaned-main/*.json` → `RawRecord` Parquet using the cleaner logic
2. Writes output to `~/.local/share/knirvhasher/connector/alpaca_bootstrap.parquet`
3. Stages 1–3 then process it identically to HuggingFace data

This lets the pipeline be tested end-to-end with local data before wiring up the HuggingFace connector.

---

## 7. End-to-End Data Flow

```
HuggingFace Hub
     │  (Parquet streaming, Bearer HF_TOKEN)
     ▼
[0_DATA_CONNECTOR]
  HuggingFaceConnector → Cleaner → Writer
  Output: ~/.local/share/knirvhasher/connector/*.parquet
     │
     ▼
[1_DATA_MINER]
  RawRecord → Embed (Ollama/Cloudflare) → DocumentRecord
  Output: ~/.local/share/knirvhasher/miner/mined_records.parquet
     │
     ▼
[2_DATA_ENCODER]
  DocumentRecord → LSH Projections → Bracket
  FrameTicker (1s windows) → NRVWriter
  Output: ~/.local/share/knirvhasher/encoder/{dataset}_{shard}.nrv
     │
     ▼
[3_DATA_TRAINER]
  vHasher GRPO → patch GoldenSeed + ASICLoops per Bracket
  AppendBracketDirect / Arrow Flight → KNIRVBASE NRVStorage
     │
     ▼
KNIRVBASE (packages/KNIRVBASE/go)
  NRVStorage → FrameTicker → NRVWriter → .nrv on disk
  Arrow Flight server → streaming reads for consumers
```

---

## 8. Shared Internal Schema Changes

### 8.1 Common `types` Package (new)

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

### 8.2 Parquet Column Naming

All inter-stage Parquet files use snake_case column names matching the struct tags in `types.go`. No change to existing `DocumentRecord` column names — Stage 1 output is backward-compatible.

---

## 9. Testing Requirements

### 9.1 Stage 0
- Unit test: `TestHuggingFaceNormalization` — verify each column-normalization rule against a mock dataset schema
- Integration test: download the first 100 rows of `tatsu-lab/alpaca` and verify `RawRecord.Text` is non-empty

### 9.2 Stage 2
- Unit test: `TestLSHMapperReproducibility` — same seed + same embedding → identical `[64]byte`
- Unit test: `TestBracketEncodeDecodeRoundTrip` — encode → decode → field equality
- Unit test: `TestFrameTickerFlush` — insert 350 brackets over 2.5 seconds → verify 2 full frames + 1 partial flushed on Stop

### 9.3 Stage 3
- Unit test: `TestGoldenSeedPatch` — verify bracket `GoldenSeed` is non-zero after one GRPO pass
- Integration test: submit 1000 brackets via `AppendBracketDirect` → verify KNIRVBASE `NRVReader.StreamBrackets` returns them all

### 9.4 End-to-End
- Bootstrap test using `AlpacaDataCleaned-main` → run all 4 stages → assert `.nrv` file is valid (magic bytes, non-zero frame count, all brackets 80-byte aligned)

---

## 10. Implementation Order

| Order | Stage | Key Work |
|---|---|---|
| 1 | Stage 2 | LSHMapper, Bracket construction, FrameTicker wiring, NRVWriter output |
| 2 | Bootstrap | AlpacaDataCleaned → RawRecord → Stage 1 → Stage 2 (smoke test end-to-end) |
| 3 | Stage 3 | GoldenSeed patch loop + AppendBracketDirect submission to KNIRVBASE |
| 4 | Stage 0 | HuggingFaceConnector, Cleaner updates, config YAML |
| 5 | Stage 1 | Stage 0 Parquet input, per-record chunking logic |
| 6 | Testing | Unit + integration tests per §9 |
| 7 | Flight | Arrow Flight bulk streaming path (after single-bracket path is stable) |

Start with Stage 2 because it has the clearest dependency on the KNIRVBASE Phase 2 spec and can be developed/tested against the existing Stage 1 `mined_records` output without needing Stage 0.
