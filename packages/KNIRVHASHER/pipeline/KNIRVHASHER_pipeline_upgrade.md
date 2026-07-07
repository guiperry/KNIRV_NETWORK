# KNIRVHASHER Pipeline Upgrade: HuggingFace → `.nrv` → KNIRVBASE

**Status:** Planning — Phase 2 of KNIRVBASE Upgrade is the target spec.
**Source spec:** `packages/KNIRVBASE/go/KNIRVBASE_Upgrade.md` (Phase 2, ASIC-Native)
**Pipeline root:** `packages/KNIRVHASHER/pipeline/`

> **🚨 ARCHITECTURAL WARNING**: This upgrade introduces LSH projections as a **parameterization** of the Identity Zone (Slots 0-3), NOT a replacement for the Semantic Coherence Mapper. The 12-slot Bitmask Specification from `DATA-MAPPER.md` is MANDATORY and must be preserved. Key requirements:
> - **Slots 0-3 (Identity Zone)**: LSH projections via SemanticMapper (replaces "Naive Variance")
> - **Slots 4-5 (Syntactic Registers)**: POS, Tense, Dependency — **REQUIRED for Syntactic Steering**
> - **Slots 6-8 (Memory Zone)**: History XOR from Flash Search — **REQUIRED for 21-pass recurrence**
> - **Slot 9 (Intent Flags)**: Question/Command/Code detection
> - **Slot 10 (Domain Signature)**: Math/Code/Prose classification
> - **Slot 11 (Temporal Lock)**: Position + Salt for uniqueness
>
> The 21-pass temporal loop requires Syntactic Steering (Passes 8-14 using Slots 4-5). If these are missing, the system collapses to random hashing.

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

> **🔗 CROSS-REFERENCE**: The Frame/Bracket hierarchy is defined in `packages/KNIRVBASE/go/KNIRVBASE_Upgrade.md` Section 3. This section summarizes the key concepts for the pipeline.

| Term | Meaning | 12-Slot Mapping |
|---|---|---|
| **Bracket** | 80-byte binary record: LSH Projections (32B) + SubSecondUS (4B) + Syntactic (4B) + Intent+Domain (3B) + GoldenSeed (4B) + Memory (14B) + LSH Salt (4B) + Reserved (15B) | Encodes all 12 slots |
| **Frame** | 1-second temporal window holding N Brackets + ThermoAtmosphere + LinguisticMapping + Z3Result | Container for bracket batch |
| **I-Bracket** | Absolute (intra) bracket — stores full LSH projections (Slots 0-3) | Full semantic snapshot |
| **P-Bracket** | Delta bracket — stores XOR-diff of projections against anchor I-Bracket | Compressed delta |
| **FrameTicker** | 1-second goroutine that buffers brackets, computes I/P type, XOR-diffs, and flushes to NRVWriter | Orchestrates frame assembly |
| **MemoryZone** | In-memory state (Slots 6-8) maintained by FrameTicker for temporal loop recurrence | History XOR computation |

### 1.3 Frame/Bracket Hierarchy

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
| `0x29` | 2B | DomainSig | Slot 10 (Domain) | Mode Enforcement (e.g., Math) |
| `0x2B` | 4B | GoldenSeed | (Nonce Target) | The solved "Weight" |
| `0x2F` | 14B | Memory (XOR recursive) | Slots 6-8 (Temporal) | Recursive Context Bridge |
| `0x3D` | 4B | LSH Salt | Slot 11 (Lock) | Prevents Collision loops |
| `0x41` | 15B | Reserved | (Future Expansion) | Padding to 80 bytes |

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
**Goal:** Preserve the 12-slot ASIC format while adapting to Phase 2 `.nrv` output. The LSH Mapper is integrated as a parameterizable component within the existing Semantic Coherence framework, NOT as a replacement.

> **⚠️ CRITICAL: This stage MUST preserve the 12-slot Bitmask Specification from `DATA-MAPPER.md`. The LSH Mapper is a parameterization of the Identity Zone (Slots 0-3), NOT a replacement for the entire semantic architecture.**

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

### 4.2 Semantic Coherence Mapper with LSH Parameterization

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
│  │ Domain Classify │───► Slot 10 (Domain Signature)
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
2. **Syntactic Registers (Slots 4-5)**: Extract POS, Tense, Dependency from NLP Bridge. These MUST be preserved for the 21-pass loop's Syntactic Steering (Passes 8-14).
3. **Memory Zone (Slots 6-8)**: XOR with previous bracket hash for recurrence.
4. **Intent Flags (Slot 9)**: Detect Question/Command/Code markers.
5. **Domain Signature (Slot 10)**: Classify Math/Code/Prose environment.
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

### 4.3 v2_schema.yaml — Schema-Driven Bracket Configuration

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
    description: "Math/Code/Prose environment classification"
  
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

### 4.4 Bracket Construction (Updated)

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

> **⚠️ ATTENTION**: The `meta.POSTag`, `meta.Tense`, and `meta.DepHead` fields MUST be populated from the NLP Bridge (spaCy). If these are missing, the 21-pass loop will lose Syntactic Steering (Passes 8-14) and collapse to random hashing.

### 4.5 Associative Jitter Vector (Flash Search Integration)

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

`GoldenSeed` and `ASICLoops` are left at zero/1 in Stage 2. Stage 3 updates these after ASIC training passes. The `.nrv` file is written first with placeholder values; Stage 3 patches them via `NRVWriter` before submitting to KNIRVBASE.

### 4.5 Frame Assembly via FrameTicker

> **🔗 CROSS-REFERENCE**: Full FrameTicker implementation is in `packages/KNIRVBASE/go/KNIRVBASE_Upgrade.md` Section 4.4. This section summarizes for the pipeline.

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

### 4.7 Preserve (NOT Remove)

> **⚠️ CRITICAL CORRECTION**: The following components MUST be preserved. The LSH Mapper does NOT replace them:

- `pkg/mapper/mapper.go` — **KEEP**: The 12-slot Semantic Coherence Mapper is the core of the 21-pass temporal loop
- `pkg/mapper/variance_mapper.go` — **KEEP**: Provides variance-selected dimensions as fallback
- `pkg/schema/output.go` — **KEEP**: TrainingFrame schema for backward compatibility
- `pkg/sliding/` — **KEEP**: Sliding window chunking for long documents
- `pkg/tokenizer/` — **KEEP**: Token counting for ASICLoops estimation
- `pkg/schema/input.go` — **KEEP**: Input schema unchanged

New additions:
- `pkg/semantic/semantic_mapper.go` — Integrates LSH into 12-slot framework
- `config/v2_schema.yaml` — Schema-driven bracket configuration

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
  RawRecord → Embed (Ollama/Cloudflare) → NLP Bridge (spaCy)
  Output: mined_records.parquet (text + embedding + POS/Tense/DepHead)
     │
     ▼
[2_DATA_ENCODER]
  DocumentRecord → SemanticMapper (12-slot Spec → LSH Projections)
  ├─ Slots 0-3: LSH Projections (64B) ← Identity Zone
  ├─ Slots 4-5: POS + Tense + Dependency ← Syntactic Registers (REQUIRED)
  ├─ Slots 6-8: History XOR ← Memory Zone (Flash Search Jitter)
  ├─ Slot 9: Intent Flags ← Question/Command/Code
  ├─ Slot 10: Domain Signature ← Math/Code/Prose
  └─ Slot 11: Temporal Lock ← Position + Salt
  FrameTicker (1s windows) → NRVWriter
  Output: ~/.local/share/knirvhasher/encoder/{dataset}_{shard}.nrv
     │
     ▼
[3_DATA_TRAINER]
  vHasher GRPO (21-pass loop with Syntactic Steering)
  ├─ Passes 1-7: Anchor check (Slot 0)
  ├─ Passes 8-14: Syntactic Steering (Slots 4-5) ← CRITICAL
  └─ Passes 15-21: Entropy resolution (Slot 3)
  Patch GoldenSeed + ASICLoops per Bracket
  AppendBracketDirect / Arrow Flight → KNIRVBASE NRVStorage
     │
     ▼
KNIRVBASE (packages/KNIRVBASE/go)
  NRVStorage → FrameTicker → NRVWriter → .nrv on disk
  Arrow Flight server → streaming reads for consumers
```

---

## 8. Frame/Bracket Architecture Deep Dive

> **🔗 CROSS-REFERENCE**: Full specification in `packages/KNIRVBASE/go/KNIRVBASE_Upgrade.md` Section 3 (Binary Format) and Section 4.4 (FrameTicker).

This section clarifies how the Frame/Bracket hierarchy works specifically for the KNIRVHASHER pipeline.

### 8.1 Why Frames?

The **1-second Frame** serves as the temporal container for the ASIC pipeline:
- **Batching**: Groups multiple bracket submissions into a single atomic write
- **Delta Compression**: Enables I/P bracket distinction (full vs XOR-diff)
- **Temporal Context**: Each Frame carries `timestamp_unix` and `linguistic` metadata
- **Hardware Metrics**: Each Frame records `ThermoAtmosphere` (temp/voltage/clock)
- **Z3 Validation**: Each Frame gets a `Z3Result` determining Gold/Research stream eligibility

### 8.2 FrameTicker Behavior

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

### 8.3 I-Bracket vs P-Bracket Decision

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

### 8.4 MemoryZone (Slots 6-8)

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

### 8.5 Registry Structure

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

### 8.6 Stage 2 → Stage 3 Handoff

**Stage 2 Output:**
- `.nrv` files with Brackets in Chunk 1
- Registry in Chunk 0 with FrameEntries
- `GoldenSeed = 0`, `ASICLoops = 1` (placeholders)

**Stage 3 Input:**
- Reads `.nrv` files via `NRVReader`
- Runs vHasher GRPO to find optimal `GoldenSeed` per bracket
- Updates `ASICLoops` based on actual pass count used
- Re-encodes brackets with solved values
- Submits to KNIRVBASE via `AppendBracketDirect` or Arrow Flight

> **⚠️ NOTE**: Stage 3 does NOT re-run FrameTicker. The Frame structure is preserved from Stage 2. Only the `GoldenSeed` and `ASICLoops` fields are patched in-place.

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
> **⚠️ MANDATORY**: These tests verify the 12-slot Bitmask Specification is preserved.

- Unit test: `TestLSHMapperReproducibility` — same seed + same embedding → identical `[64]byte`
- Unit test: `TestSemanticMapperSlotPreservation` — verify all 12 slots are populated per the bitmask spec
- Unit test: `TestSyntacticProfileExtraction` — verify POS, Tense, Dependency are extracted for Slots 4-5
- Unit test: `TestBracketEncodeDecodeRoundTrip` — encode → decode → field equality
- Unit test: `TestFrameTickerFlush` — insert 350 brackets over 2.5 seconds → verify 2 full frames + 1 partial flushed on Stop
- Integration test: `Test21PassLoopIntegrity` — verify the SemanticMapper output can drive the 21-pass loop with proper Syntactic Steering

### 9.3 Stage 3
- Unit test: `TestGoldenSeedPatch` — verify bracket `GoldenSeed` is non-zero after one GRPO pass
- Unit test: `TestSyntacticSteeringInTraining` — verify Passes 8-14 use Slot 4-5 for grammatical validation
- Integration test: submit 1000 brackets via `AppendBracketDirect` → verify KNIRVBASE `NRVReader.StreamBrackets` returns them all

### 9.4 End-to-End
- Bootstrap test using `AlpacaDataCleaned-main` → run all 4 stages → assert `.nrv` file is valid (magic bytes, non-zero frame count, all brackets 80-byte aligned)
- **CRITICAL**: Verify the 21-pass loop produces deterministic consensus when given semantically coherent brackets

---

## 10. Implementation Order

| Order | Stage | Key Work |
|---|---|---|
| 1 | Stage 2 | SemanticMapper (preserving 12-slot spec), v2_schema.yaml, Bracket construction |
| 2 | Bootstrap | AlpacaDataCleaned → RawRecord → Stage 1 → Stage 2 (smoke test end-to-end) |
| 3 | Stage 1 Update | Ensure NLP Bridge (spaCy) outputs POS/Tense/DepHead for Slot 4-5 |
| 4 | Stage 3 | GoldenSeed patch loop + Syntactic Steering in 21-pass loop |
| 5 | Stage 0 | HuggingFaceConnector, Cleaner updates, config YAML |
| 6 | Testing | Unit + integration tests per §9 (including 21-pass loop integrity) |
| 7 | Flight | Arrow Flight bulk streaming path (after single-bracket path is stable) |

> **⚠️ CRITICAL**: Start with Stage 2 because it has the clearest dependency on the KNIRVBASE Phase 2 spec and can be developed/tested against the existing Stage 1 `mined_records` output without needing Stage 0.

**PRIORITY**: Before any coding, verify that Stage 1's NLP Bridge (spaCy) outputs `POSTag`, `Tense`, and `DepHead` — these are REQUIRED for the 21-pass loop's Syntactic Steering (Passes 8-14).
