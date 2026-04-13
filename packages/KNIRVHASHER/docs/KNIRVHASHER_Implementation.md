# KNIRVHASHER × KNIRV Network Implementation Plan

## Goal

> **Experimental Stealth Mode.** KNIRVHASHER is a standalone dataset collection engine.
> Its full feature set is not a runtime dependency of any other KNIRV component. The
> only portion that surfaces in other services is **pipeline phases 1–3**, which are
> called non-interactively as a background process. The primary mission of KNIRVHASHER
> in its current form is to gather as many training datasets as possible so that all
> KNIRV servers can be empowered via a future global model update.

Ingest user ontology data and server activity, train user-centric logic gate hash
networks, and produce `.nrv` datasets for future distribution across the KNIRV
network. Security rule enforcement is **not** a KNIRVHASHER responsibility.

---

## Architecture Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Deployment Model | **Standalone / Experimental** | Full feature in stealth mode; not a runtime dependency of any KNIRV service |
| Data Transfer | **gRPC** (encrypted) | Direct to local hasher instance; PQC-encrypted payloads |
| gRPC Transport | **Unix socket** (`/var/run/hasher.sock`) | Better for containerized environments |
| Training Triggers | **On-demand + Scheduled + Event-driven** | Dataset collection across all operational scenarios |
| Seed Storage | **KNIRVBASE (.nrv files, local instance)** | Each KNIRV component owns its own KNIRVBASE; hasher writes to its own |
| Data Format | **.nrv** | Binary container with Arrow modalities; zero-copy streaming via Flight |
| Connector Language | **Go** | Native to hasher, no TS overhead |
| Connector Location | **0_DATA_CONNECTOR** | Pre-pipeline data preparation |
| KNIRVSERVER Usage | **Pipeline phases 1–3 only** | Non-interactive background calls during onboarding; full hasher stays isolated |
| Guardrail Enforcement | **Not KNIRVHASHER's concern** | Enforcement is a KNIRVSERVER responsibility using its own KNIRVBASE |
| Fallback | **Graceful degradation** | Pipeline is advisory; data collection continues independently |

---

## System Architecture

There are two distinct contexts in which the pipeline runs:

1. **KNIRVHASHER (standalone)** — the full feature, running independently, collecting
   datasets from any KNIRV node. In experimental stealth mode; not a runtime dependency
   of any service.
2. **KNIRVSERVER background pipeline** — phases 1–3 only, called non-interactively
   during user onboarding to build `.nrv` entries in KNIRVSERVER's own KNIRVBASE.

```
╔═════════════════════════════════════════════════════════════════════╗
║          KNIRVHASHER  [EXPERIMENTAL — STANDALONE]                   ║
║                                                                     ║
║  ┌──────────────┐  gRPC (Unix sock)  ┌──────────────────────────┐  ║
║  │ KNIRVSERVER  │ ─────────────────▶ │  0_DATA_CONNECTOR (Go)   │  ║
║  │  DVE data    │                    │  normalizer / .nrv enc.  │  ║
║  └──────────────┘                    └────────────┬─────────────┘  ║
║                                                   │                ║
║                                                   ▼                ║
║  ┌────────────────────────────────────────────────────────────┐   ║
║  │                   HASHER PIPELINE                          │   ║
║  │                                                            │   ║
║  │  [Phase 1]         [Phase 2]         [Phase 3]            │   ║
║  │  1_DATA_MINER  ──▶ 2_DATA_ENCODER ──▶ 3_DATA_TRAINER     │   ║
║  │  SpaCy NLP        BGE Embed            Evo-GRPO seeds     │   ║
║  │                                              │             │   ║
║  │                                              ▼             │   ║
║  │  ┌─────────────────────────────────────────────────────┐  │   ║
║  │  │           USER LOGIC GATE TRAINING                  │  │   ║
║  │  │  • Security constraints → Slot 4/10 tokens         │  │   ║
║  │  │  • Violations → Negative reinforcement             │  │   ║
║  │  │  • Trained seeds → UserSecurityGates               │  │   ║
║  │  └─────────────────────────────────────────────────────┘  │   ║
║  └───────────────────────────────────────────────┬────────────┘   ║
║                                                  │                 ║
║                                                  ▼                 ║
║  ┌────────────────────────────────────────────────────────────┐   ║
║  │           KNIRVHASHER KNIRVBASE (local instance)           │   ║
║  │  • Collection: hasher_seeds  (payload = .nrv path)        │   ║
║  │  • PQC-signed .nrv datasets; Apache Flight streaming      │   ║
║  │  • Dataset accumulation for future global model update    │   ║
║  └────────────────────────────────────────────────────────────┘   ║
╚═════════════════════════════════════════════════════════════════════╝

         ┆  phases 1–3 extracted (non-interactive, no gRPC dep)
         ┆
╔═════════════════════════════════════════════════════════════════════╗
║                   KNIRVSERVER                                       ║
║                                                                     ║
║  ┌──────────────────────────────────────────────────────────────┐  ║
║  │  Cognitive Engine — Background Onboarding Loop               │  ║
║  │                                                              │  ║
║  │  On user onboarding data:                                    │  ║
║  │    Phase 1: DATA_MINER  (SpaCy NLP)                         │  ║
║  │    Phase 2: DATA_ENCODER (BGE embeddings → .nrv brackets)   │  ║
║  │    Phase 3: DATA_TRAINER (Evo-GRPO → trained seeds)         │  ║
║  │                              │                               │  ║
║  │                              ▼                               │  ║
║  │  ┌──────────────────────────────────────────────────────┐   │  ║
║  │  │     KNIRVSERVER KNIRVBASE (local instance)           │   │  ║
║  │  │  • One .nrv per policy / value / rule                │   │  ║
║  │  │  • Data ingestion, mapping, encoding, streaming      │   │  ║
║  │  │  • Apache Flight (internal consumers only)           │   │  ║
║  │  └──────────────────────────────────────────────────────┘   │  ║
║  └──────────────────────────────────────────────────────────┘   ║
║                                                                     ║
║  DVE / GuardrailEngine ◀── Monitor (separate, not KNIRVHASHER)     ║
║  (enforcement via KNIRVSERVER's own .nrv data — see §VI)           ║
╚═════════════════════════════════════════════════════════════════════╝
```

---

## Phase 1: KNIRVSERVER Hasher gRPC Service

### 1.1 Proto Definition

**File:** `backend/internal/proto/hasher.proto`

```protobuf
syntax = "proto3";

package hasher;

service HasherService {
    rpc ExportSecurityData(ExportRequest) returns (stream EncryptedChunk);
    rpc TriggerTraining(TrainingRequest) returns (TrainingResponse);
    rpc GetTrainingStatus(TrainingStatusRequest) returns (TrainingStatusResponse);
    rpc GetUserRules(RulesRequest) returns (RulesResponse);
    rpc ValidateAction(ActionRequest) returns (ActionResponse);
}

message ExportRequest {
    string org_id = 1;
    string user_id = 2;
    DataType data_type = 3;
    bool encrypted = 4;
}

enum DataType {
    ALL = 0;
    ONTOLOGY = 1;
    GUARDRAILS = 2;
    ACTIVITY = 3;
    MARKDOWN = 4;
}

message EncryptedChunk {
    bytes data = 1;
    string chunk_id = 2;
    bool is_last = 3;
}

message TrainingRequest {
    string org_id = 1;
    string user_id = 2;
    TrainingTrigger trigger = 3;
    map<string, string> options = 4;
}

enum TrainingTrigger {
    ON_DEMAND = 0;
    SCHEDULED = 1;
    GUARDRAIL_VIOLATION = 2;
}

message TrainingResponse {
    string training_id = 1;
    string status = 2;
}

message ActionRequest {
    string user_id = 1;
    string action = 2;
    map<string, string> context = 3;
}

message ActionResponse {
    bool allowed = 1;
    float confidence = 2;
    repeated string violations = 3;
    repeated string applied_rules = 4;
}
```

### 1.2 Hasher Integration Hook

**File:** `backend/internal/services/dve/hasher_integration.go`

```go
type HasherIntegration struct {
    grpcClient  hasherpb.HasherServiceClient
    dveManager  *DVEManager
    guardrailMgr *guardrails.DynamicGuardrailManager
    ontologyMgr  *DVEOntologyManager
    kvbase      knirvbase.Collection
}

func (hi *HasherIntegration) OnGuardrailViolation(violation *GuardrailViolation) error {
    return hi.TriggerTraining(violation.NodeID, GUARDRAIL_VIOLATION)
}

func (hi *HasherIntegration) OnValidationComplete(result *TaskResult) error {
    // Analyze patterns, trigger training if needed
}

func (hi *HasherIntegration) ExportUserData(orgID, userID string) (<-chan *EncryptedChunk, error)
func (hi *HasherIntegration) TriggerTraining(orgID, userID string, trigger TrainingTrigger) error
func (hi *HasherIntegration) ValidateAction(userID, action string, ctx map[string]string) (*ActionResponse, error)
```

### 1.3 Data Export Hook (KNIRVSERVER → KNIRVHASHER)

**File:** `backend/internal/services/dve/hasher_export.go`

When the full KNIRVHASHER feature is active, KNIRVSERVER can optionally stream DVE
data to the standalone KNIRVHASHER over gRPC for dataset collection. This is an
**opt-in, fire-and-forget** export — KNIRVSERVER does not block on or depend on
a response.

```go
// HasherExporter sends DVE snapshots to a running KNIRVHASHER instance for
// dataset collection. Safe to disable; KNIRVSERVER operation is unaffected.
type HasherExporter struct {
    grpcClient  hasherpb.HasherServiceClient
    enabled     bool  // false until KNIRVHASHER is out of stealth
}

func (he *HasherExporter) ExportOnboardingData(orgID, userID string) {
    if !he.enabled {
        return // stealth mode — no-op
    }
    go func() {
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()
        _, _ = he.grpcClient.ExportSecurityData(ctx, &ExportRequest{
            OrgId:     orgID,
            UserId:    userID,
            DataType:  DataType_ALL,
            Encrypted: true,
        })
    }()
}
```

---

## Phase 2: 0_DATA_CONNECTOR (Go)

### 2.1 Directory Structure

```
pipeline/0_DATA_CONNECTOR/
├── cmd/
│   └── connector/
│       └── main.go
├── internal/
│   ├── grpc/
│   │   └── client.go
│   ├── normalizer/
│   │   ├── normalizer.go
│   │   └── security_mapper.go
│   ├── cleaner/
│   │   └── cleaner.go
│   ├── encoder/
│   │   └── arrow_encoder.go
│   └── writer/
│       └── writer.go
├── config/
│   └── connector.yaml
├── go.mod
└── Makefile
```

### 2.2 Main Entry Point

**File:** `cmd/connector/main.go`

```go
func main() {
    config := LoadConfig()
    
    client := grpc.NewClient(config.HasherAddr)
    defer client.Close()
    
    stream, err := client.ExportSecurityData(&ExportRequest{
        OrgId: config.OrgID,
        UserId: config.UserID,
        DataType: DataType_ALL,
        Encrypted: true,
    })
    
    normalizer := normalizer.NewSecurityNormalizer()
    encoder := encoder.NewArrowEncoder()
    writer := writer.NewFileWriter(config.OutputDir)
    
    for chunk := range stream {
        records := normalizer.Process(chunk.Data)
        frames := encoder.Encode(records)
        writer.Write(frames)
    }
}
```

### 2.3 Security Normalizer

**File:** `internal/normalizer/security_mapper.go`

```go
type SecurityNormalizer struct {
    schema *SecuritySchema
}

type SecurityRecord struct {
    FileName   string    `arrow:"file_name"`
    ChunkID    int32     `arrow:"chunk_id"`
    Content    string    `arrow:"content"`
    Embedding  []float32 `arrow:"embedding"`
    Tokens     []string  `arrow:"tokens"`
    POSTags    []int     `arrow:"pos_tags"`
    DepHashes  []uint32  `arrow:"dep_hashes"`
    SecurityTags []string `arrow:"security_tags"`
}

var SECURITY_TAG_MAPPINGS = map[string]SecurityMapping{
    "guardrail_block": {
        Slot10: 0x2400, // Logic/Set domain
        Slot4:  0x07,  // PREP (constraint marker)
        Weight: -1.0,   // Negative reinforcement
    },
    "guardrail_warn": {
        Slot10: 0x2401, // Guardrail subdomain
        Slot4:  0x04,  // ADV
        Weight: -0.5,
    },
    "security_constraint": {
        Slot10: 0x2400,
        Slot4:  0x02,  // VERB (action)
        Weight: 1.0,
    },
    "violation": {
        Slot10: 0x2402, // Violation subdomain
        Slot4:  0x01,  // NOUN (subject)
        Weight: -2.0,   // Strong negative
    },
}
```

### 2.4 Arrow Encoder

**File:** `internal/encoder/arrow_encoder.go`

The embedding field is **fixed-size 16-dimensional** (`FixedSizeList<Float32>(16)`), matching
the 16 variance-selected BGE dimensions from `vector_mapper.go` and the 16-dim LSH
projection space used by `projectionsToFloat32` in KNIRVBASE's `flight_server.go`. The
Arrow Knowledge Base (the parquet vocabulary backing Flash Search) is re-indexed to this
same 16-dim schema — do not use variable-length lists.

```go
// VectorDims is the fixed dimensionality of all BGE embedding vectors in the pipeline.
// Must match vector_mapper.go variance selection, Arrow KB index, and KNIRVBASE
// Projections layout (32 bytes → 16 × uint16 dimensions).
const VectorDims = 16

type ArrowEncoder struct {
    schema *arrow.Schema
    pool   memory.Allocator
}

func NewArrowEncoder() *ArrowEncoder {
    // embedding: FixedSizeList<Float32>(16) — locked to VectorDims.
    // Variable-length lists are NOT permitted; the Flash Search helper and
    // KNIRVBASE CalcBracketDriftScore both assume a fixed 16-dim stride.
    embeddingType := arrow.FixedSizeListOf(VectorDims, arrow.PrimitiveTypes.Float32)
    schema := arrow.NewSchema([]arrow.Field{
        {Name: "file_name",     Type: arrow.BinaryTypes.String},
        {Name: "chunk_id",      Type: arrow.PrimitiveTypes.Int32},
        {Name: "content",       Type: arrow.BinaryTypes.String},
        {Name: "embedding",     Type: embeddingType},              // 16-dim fixed
        {Name: "tokens",        Type: arrow.ListOf(arrow.BinaryTypes.String)},
        {Name: "pos_tags",      Type: arrow.ListOf(arrow.PrimitiveTypes.Int32)},
        {Name: "dep_hashes",    Type: arrow.ListOf(arrow.PrimitiveTypes.Uint32)},
        {Name: "security_tags", Type: arrow.ListOf(arrow.BinaryTypes.String)},
        {Name: "domain_sig",    Type: arrow.PrimitiveTypes.Uint16}, // Slot 10 (0x2000=Math)
        {Name: "slot4_raw",     Type: arrow.PrimitiveTypes.Uint32}, // packed slot4 register
    }, nil)
    return &ArrowEncoder{schema: schema, pool: memory.NewGoAllocator()}
}

func (e *ArrowEncoder) Encode(records []*SecurityRecord) (*arrow.Buffer, error) {
    // Build Arrow record batch; each embedding row must have exactly VectorDims values.
    // Return IPC-encoded buffer (IPC stream format for socket streaming,
    // IPC file format for Arrow KB parquet re-index).
}
```

### 2.5 Config

**File:** `config/connector.yaml`

```yaml
hasher:
  socket: "/var/run/hasher.sock"  # Unix socket for container compatibility
  timeout: 300

source:
  org_id: "org_default"
  user_id: "all"

output:
  arrow_dir: "/tmp/hasher/frames/arrow"
  json_dir: "/tmp/hasher/frames/json"
  batch_size: 100

processing:
  max_concurrent: 4
  pii_scrub: true
  deduplicate: true
```

---

## Phase 3: Pipeline Updates

### 3.1 Data Miner Updates

**File:** `pipeline/1_DATA_MINER/internal/app/knirv.go` (new)

```go
func (c *Config) LoadKnirvInput() ([]*app.DocumentRecord, error) {
    arrowFiles, err := filepath.Glob(c.InputDir + "/*.arrow")
    jsonFiles, err := filepath.Glob(c.InputDir + "/*.json")
    
    var records []*app.DocumentRecord
    for _, f := range append(arrowFiles, jsonFiles...) {
        recs, err := loadRecords(f)
        records = append(records, recs...)
    }
    return records, nil
}

func loadRecords(path string) ([]*app.DocumentRecord, error) {
    if strings.HasSuffix(path, ".arrow") {
        return loadArrowRecords(path)
    }
    return loadJSONRecords(path)
}
```

### 3.2 Data Encoder: Tensor Packer (Re-tooled)

**File:** `pipeline/2_DATA_ENCODER/internal/tensor_packer.go` (modify)

#### Slot 4 Register Layout

The Slot 4 register is a `uint32` in the in-memory tensor. Its 32 bits are split into
two zones whose semantics depend on the domain:

```
uint32 Slot4Register:
  bits  0–7   (mask 0x000000FF): Operator zone — POS/Grammar constraint or Math operator
  bits  8–31  (mask 0xFFFFFF00): Jitter zone   — 24-bit Flash Search jitter payload
```

In **Math Mode** (`DomainSig == 0x2000`) the operator zone encodes the Math operator
token (add, mul, eq, etc.) rather than a natural-language POS tag. The jitter zone is
always populated from the Flash Search helper regardless of domain.

```go
// DomainMath is the Slot 10 domain signature for Math Mode.
const DomainMath uint16 = 0x2000

// MathSymbol enumerates the three Symbolic Categories emitted by the NLP
// Refraction Layer (nlp_bridge.go §3.3) for Math Mode frames. These replace
// the previous fine-grained operator enum. Specific arithmetic operations
// (add, mul, etc.) are captured by the 16-dim LSH projection in Slots 0-3;
// Slot 4 carries only the structural role of the token.
type MathSymbol uint8

const (
    // MathSymbolOperand: a numeric literal or measured quantity.
    // SpaCy: dep=nummod, pos=NUM
    MathSymbolOperand  MathSymbol = 0x01

    // MathSymbolVariable: an algebraic placeholder or named quantity.
    // SpaCy: dep=nsubj (in math context), pos=SYM
    MathSymbolVariable MathSymbol = 0x02

    // MathSymbolOperator: an arithmetic or relational operator.
    // SpaCy: dep=ROOT (where token value is an operator), pos=VERB
    // The specific operator (add/mul/eq/lt/…) is encoded in Slots 0-3.
    MathSymbolOperator MathSymbol = 0x04
)

// PackSlot4 builds the 32-bit Slot 4 register.
//
//   operatorByte — POS tag (non-math) or MathOperator (math mode); fills 0x000000FF.
//   jitterPayload — 24-bit value from FlashSearch; fills 0xFFFFFF00.
//
// The jitter payload is always injected regardless of domain. In Math Mode the
// operator byte must be a MathOperator constant; any value > 0xFF is truncated.
func PackSlot4(operatorByte uint8, jitterPayload uint32) uint32 {
    return uint32(operatorByte) | ((jitterPayload & 0x00FFFFFF) << 8)
}

// UnpackSlot4 extracts the operator byte and jitter payload from a packed register.
func UnpackSlot4(reg uint32) (operatorByte uint8, jitterPayload uint32) {
    operatorByte  = uint8(reg & 0x000000FF)
    jitterPayload = (reg >> 8) & 0x00FFFFFF
    return
}

// TensorPacker orchestrates all 12 slots into the NeuralFrame binary.
type TensorPacker struct {
    flashSearch *FlashSearchHelper
    arrowKB     *ArrowKnowledgeBase // 16-dim re-indexed KB
}

// Orchestrate builds the full 12-slot uint32 array for one bracket.
// It injects the Temporal Salt (Slot 11) and Flash Search jitter (Slot 4 upper bits).
func (tp *TensorPacker) Orchestrate(base *SlotVector, pos uint16, domainSig uint16) []uint32 {
    slots := base.Copy()

    // Slot 11: Temporal Salt — (PosIndex << 16) | TemporalSalt
    salt := uint16(time.Now().UnixNano() & 0xFFFF)
    slots[11] = uint32(pos) | (uint32(salt) << 16)

    // Flash Search: use first 4 bytes of current hash state as lookup key.
    jitter := tp.flashSearch.Lookup(slots[:4], tp.arrowKB)

    // Slot 4: pack operator (0xFF zone) + jitter payload (0xFFFFFF zone).
    var operatorByte uint8
    if domainSig == DomainMath {
        operatorByte = uint8(slots[4] & 0xFF) // preserve Math operator set by encoder
    } else {
        operatorByte = uint8(slots[4] & 0xFF) // POS/Grammar from nlp_bridge
    }
    slots[4] = PackSlot4(operatorByte, jitter)

    return slots
}

// SaveTrainingFrames writes frames in .nrv bracket format.
func (tp *TensorPacker) SaveTrainingFrames(frames []*NeuralFrame, outputPath string) error {
    return tp.saveNRV(frames, outputPath)
}

func (tp *TensorPacker) saveNRV(frames []*NeuralFrame, path string) error {
    schema := arrow.NewSchema([]arrow.Field{
        {Name: "frame_id",    Type: arrow.PrimitiveTypes.Int32},
        {Name: "slots",       Type: arrow.FixedSizeListOf(12, arrow.PrimitiveTypes.Uint32)},
        {Name: "embedding",   Type: arrow.FixedSizeListOf(VectorDims, arrow.PrimitiveTypes.Float32)},
        {Name: "target_token",Type: arrow.PrimitiveTypes.Int32},
        {Name: "domain_sig",  Type: arrow.PrimitiveTypes.Uint16},
    }, nil)

    writer, err := ipc.NewFileWriter(os.Create(path), ipc.WithSchema(schema))
    if err != nil {
        return err
    }
    defer writer.Close()
    // populate record builder from frames and write batches
    return nil
}
```

#### Flash Search Helper

**File:** `pipeline/2_DATA_ENCODER/internal/flash_search.go`

```go
// FlashSearchHelper performs the Lookup Key → Jitter Vector retrieval described
// in LSH_Salt.md. The first 4 bytes of the current hash state are used as a key
// into the 16-dim Arrow Knowledge Base. The returned 24-bit jitter value is
// injected into Slot 4 bits 8–31 by TensorPacker.Orchestrate.
type FlashSearchHelper struct{}

// Lookup queries the Arrow KB for the nearest 16-dim entry matching hashPrefix
// (first 4 bytes of the ASIC hash state) and extracts a 24-bit jitter payload.
func (f *FlashSearchHelper) Lookup(hashPrefix []uint32, kb *ArrowKnowledgeBase) uint32 {
    // 1. Derive lookup key from hash prefix
    key := hashPrefix[0] // 32-bit key from first hash word

    // 2. Nearest-neighbour search in 16-dim embedding space
    entry := kb.NearestByHashKey(key)
    if entry == nil {
        return 0
    }

    // 3. Fold 16 float32 dims into 24-bit jitter via XOR cascade
    var jitter uint32
    for i, v := range entry.Embedding {
        bits := math.Float32bits(v)
        jitter ^= bits >> (i % 8)
    }
    return jitter & 0x00FFFFFF
}
```

### 3.3 Vector Mapper: 16-Dimension Selection

**File:** `pipeline/2_DATA_ENCODER/internal/vector_mapper.go` (modify)

`vector_mapper.go` is updated from 4-dimension to **16-dimension** variance selection.
The BGE-Base model outputs 768-dim embeddings. The mapper runs a one-time offline
variance analysis across the local dataset and keeps the 16 dimensions with the
highest statistical variance — the 16 dimensions that carry the most discriminative
information for the current corpus.

The selected 16 indices are persisted to `config/variance_indices.json` so the same
dimensions are used consistently across training, inference, and Arrow KB indexing.

```go
// VectorMapper selects VectorDims (16) variance-maximising dimensions from a
// 768-dim BGE-Base embedding and quantises them for Slots 0–3 of the bracket.
type VectorMapper struct {
    selectedIdx [VectorDims]int // 16 dimension indices, sorted by variance descending
}

// NewVectorMapper loads pre-computed variance indices from config.
// Run ComputeVarianceIndices once offline to generate the config file.
func NewVectorMapper(indicesPath string) (*VectorMapper, error) {
    data, err := os.ReadFile(indicesPath)
    if err != nil {
        return nil, err
    }
    var vm VectorMapper
    return &vm, json.Unmarshal(data, &vm.selectedIdx)
}

// ComputeVarianceIndices analyses a corpus of BGE-Base embeddings, ranks all 768
// dimensions by variance, and writes the top VectorDims indices to indicesPath.
func ComputeVarianceIndices(embeddings [][768]float32, indicesPath string) error {
    variances := make([]float64, 768)
    n := float64(len(embeddings))
    for dim := 0; dim < 768; dim++ {
        var sum, sumSq float64
        for _, e := range embeddings {
            v := float64(e[dim])
            sum += v
            sumSq += v * v
        }
        mean := sum / n
        variances[dim] = sumSq/n - mean*mean
    }

    // Rank by variance descending, keep top VectorDims
    type ranked struct{ idx int; v float64 }
    ranks := make([]ranked, 768)
    for i, v := range variances {
        ranks[i] = ranked{i, v}
    }
    sort.Slice(ranks, func(i, j int) bool { return ranks[i].v > ranks[j].v })

    var selected [VectorDims]int
    for i := 0; i < VectorDims; i++ {
        selected[i] = ranks[i].idx
    }
    data, _ := json.Marshal(selected)
    return os.WriteFile(indicesPath, data, 0644)
}

// Map extracts and quantises the 16 selected dimensions from a full BGE embedding.
// Output: [16]float32 ready for projectionsToFloat32 / Bracket.Projections packing.
func (vm *VectorMapper) Map(full [768]float32) [VectorDims]float32 {
    var out [VectorDims]float32
    for i, idx := range vm.selectedIdx {
        out[i] = full[idx]
    }
    return out
}

// MapToProjections converts the 16 selected dimensions into the 32-byte
// Bracket.Projections field (16 × uint16, little-endian, normalised to [0, 65535]).
func (vm *VectorMapper) MapToProjections(full [768]float32) [32]byte {
    dims := vm.Map(full)
    var p [32]byte
    for i, f := range dims {
        if f < 0 { f = 0 }
        if f > 1 { f = 1 }
        q := uint16(f * 65535.0)
        p[i*2]   = byte(q)
        p[i*2+1] = byte(q >> 8)
    }
    return p
}
```

### 3.4 NLP Refraction Layer

**File:** `pipeline/1_DATA_MINER/internal/nlp_bridge.go` (modify)

The Refraction Layer sits between SpaCy's POS tagger and the Tensor Packer. It
translates raw SpaCy dependency/POS output into the three `MathSymbol` Symbolic
Categories when `DomainSig == DomainMath`. Without this layer, the Inference Watchdog
drops ~90% of mathematical frames as Syntactic Noise because SpaCy tags produce POS
values (e.g. `0x06` for `VERB`) that fall outside the `{0x01, 0x02, 0x04}` gate.

#### Symbolic Mapping Table

| SpaCy Dependency | SpaCy POS | Context Condition | KNIRV Slot 4 Symbol |
|:---|:---|:---|:---|
| `nummod` | `NUM` | any | `0x01` — Operand |
| `nsubj` | `SYM` | Math context | `0x02` — Variable |
| `ROOT` | `VERB` | Token is operator | `0x04` — Operator |
| `ROOT` | `NUM` | Token is value | `0x01` — Operand |
| `attr`, `dobj` | `SYM` | Math context | `0x02` — Variable |

A token is in "Math context" when at least one ancestor in the dependency tree
has `DomainSig == DomainMath` OR when the sentence contains an operator ROOT token.

```go
// MathContextDetector decides whether a token sits inside a mathematical
// expression, enabling the Refraction Layer to apply Symbolic Mapping.
type MathContextDetector struct {
    operatorPOS  map[string]bool // POS tags SpaCy assigns to operator-role tokens
    operatorDeps map[string]bool // dependency labels for operator-role tokens
}

func NewMathContextDetector() *MathContextDetector {
    return &MathContextDetector{
        operatorPOS:  map[string]bool{"VERB": true, "SYM": true},
        operatorDeps: map[string]bool{"ROOT": true},
    }
}

// Refract converts a SpaCy LinguisticProfile to a Slot 4 MathSymbol.
// Returns (symbol, true) when the token is math-context; (0, false) otherwise.
func (d *MathContextDetector) Refract(profile LinguisticProfile, inMathCtx bool) (MathSymbol, bool) {
    if !inMathCtx {
        return 0, false
    }
    switch {
    case profile.POSTag == POSNum:
        return MathSymbolOperand, true
    case profile.POSTag == POSSYM && profile.DepLabel == "nsubj":
        return MathSymbolVariable, true
    case profile.DepLabel == "ROOT" && d.operatorPOS[profile.POSTagStr]:
        return MathSymbolOperator, true
    case profile.DepLabel == "attr" || profile.DepLabel == "dobj":
        return MathSymbolVariable, true
    default:
        return MathSymbolOperand, true // safe default: treat unknown math tokens as operands
    }
}
```

**Integration in pipeline:** `nlp_bridge.go` calls `Refract` for every token when
the frame's `DomainSig` is `DomainMath`. The returned `MathSymbol` is passed as the
`operatorByte` argument to `PackSlot4` in `TensorPacker.Orchestrate`, replacing the
raw SpaCy POS tag value.

### 3.5 Arrow Knowledge Base Re-Indexing

**File:** `pipeline/2_DATA_ENCODER/internal/arrow_kb.go` (new)

The Arrow Knowledge Base backing the Flash Search helper must be re-indexed from
4-dim to **16-dim** vectors. Any existing parquet or Arrow IPC files built with the
old 4-dim schema are incompatible and must be regenerated.

```go
// ArrowKBEntry is one row in the 16-dim Arrow Knowledge Base.
type ArrowKBEntry struct {
    TokenID   uint32              // vocabulary token identifier
    HashKey   uint32              // first 4 bytes of the associated ASIC hash (lookup key)
    Embedding [VectorDims]float32 // 16 variance-selected BGE dimensions
    DomainSig uint16              // Slot 10 domain at time of indexing
    Slot4Raw  uint32              // packed Slot 4 register at time of indexing
}

// ArrowKnowledgeBase provides nearest-neighbour lookup over the 16-dim index.
type ArrowKnowledgeBase struct {
    entries []ArrowKBEntry
    // Future: replace linear scan with ANN index (e.g. HNSW) once KB > 1M entries
}

// HammingThreshold is the maximum acceptable bit-distance between the ASIC hash
// prefix and the nearest KB entry. Matches with distance > HammingThreshold are
// semantically unreliable and trigger a Low-Coherence Fault.
// Tunable via config; default 8 bits (25% of a 32-bit key).
const HammingThreshold = 8

// LowCoherenceFault is the sentinel token ID emitted by NearestByHashKey when
// the best Hamming match exceeds HammingThreshold. Code 0xDEAD forces the
// caller (Detokenizer / Evo-GRPO) to restart the 21-pass loop or reconsider
// the current Jitter Vector rather than resolve to a hallucinated token.
const LowCoherenceFault uint32 = 0xDEAD

// NearestByHashKey returns the KB entry whose HashKey is closest to key using
// bitwise Hamming distance, plus the distance itself.
//
// If the KB is empty or the closest entry exceeds HammingThreshold, the
// method returns (nil, LowCoherenceFault). Callers must check the second
// return value before using the entry — a LowCoherenceFault must never be
// silently coerced into a token resolution.
func (kb *ArrowKnowledgeBase) NearestByHashKey(key uint32) (*ArrowKBEntry, uint32) {
    if len(kb.entries) == 0 {
        return nil, LowCoherenceFault
    }
    best := &kb.entries[0]
    bestDist := bits.OnesCount32(best.HashKey ^ key)
    for i := 1; i < len(kb.entries); i++ {
        d := bits.OnesCount32(kb.entries[i].HashKey ^ key)
        if d < bestDist {
            bestDist = d
            best = &kb.entries[i]
        }
    }
    if bestDist > HammingThreshold {
        return nil, LowCoherenceFault
    }
    return best, uint32(bestDist)
}

// ArrowKBSchema is the canonical Arrow schema for the 16-dim Knowledge Base.
// All KB files (parquet re-index and IPC stream) must use this schema.
var ArrowKBSchema = arrow.NewSchema([]arrow.Field{
    {Name: "token_id",  Type: arrow.PrimitiveTypes.Uint32},
    {Name: "hash_key",  Type: arrow.PrimitiveTypes.Uint32},
    {Name: "embedding", Type: arrow.FixedSizeListOf(VectorDims, arrow.PrimitiveTypes.Float32)},
    {Name: "domain_sig",Type: arrow.PrimitiveTypes.Uint16},
    {Name: "slot4_raw", Type: arrow.PrimitiveTypes.Uint32},
}, nil)
```

> **Migration:** Run `pipeline/2_DATA_ENCODER/cmd/reindex/main.go` to rebuild the
> Arrow KB from raw BGE embeddings using the new `variance_indices.json`. This is a
> one-time offline step. The output replaces `ai_knowledgebase.parquet`.

### 3.5 Math Mode Enforcement

**File:** `pipeline/2_DATA_ENCODER/internal/math_mode.go` (new)

When `DomainSig == 0x2000`, the 16-dim LSH drift between two brackets **must be
masked to Slot 4 constraints** before the drift score is computed. Unmasked drift in
Math Mode allows the ASIC to wander into non-mathematical semantic regions, producing
operator hallucination — resolved tokens that are semantically plausible but
arithmetically invalid (e.g., returning a verb where a numeral is required).

The mask zeros out all drift dimensions whose variance contribution is not
attributable to the Syntactic Register (Slot 4 / Slot 5 influence zone). Only
dimensions that correlate with POS/Tense/Dependency metadata are retained.

```go
// DomainMath and MathOperator constants are defined in tensor_packer.go (§3.2).

// MathModeDriftMask is the component-wise multiplier applied to a 16-dim drift
// vector when DomainSig == DomainMath. Each element is 1.0 if the dimension is
// syntactically loaded (influences Slot 4/5), 0.0 otherwise.
// Populated offline from the variance analysis.
var MathModeDriftMask [VectorDims]float32

// LoadMathModeMask reads the pre-computed mask from config.
func LoadMathModeMask(path string) error {
    data, err := os.ReadFile(path)
    if err != nil {
        return err
    }
    return json.Unmarshal(data, &MathModeDriftMask)
}

// MaskedDrift computes the Euclidean drift between two 16-dim projection vectors,
// applying MathModeDriftMask when domainSig is DomainMath.
// This is the enforcement gate that prevents operator hallucination.
func MaskedDrift(a, b [VectorDims]float32, domainSig uint16) float64 {
    var sum float64
    for i := 0; i < VectorDims; i++ {
        diff := float64(a[i] - b[i])
        if domainSig == DomainMath {
            diff *= float64(MathModeDriftMask[i]) // zero out non-syntactic dims
        }
        sum += diff * diff
    }
    return sum
}

// EnforceMathSlot4 validates that the operator byte in a packed Slot 4 register
// is one of the three Symbolic Categories emitted by the NLP Refraction Layer
// (§3.4) when DomainSig == DomainMath.
//
// Any value outside {0x01, 0x02, 0x04} means a raw SpaCy POS tag leaked through
// without passing through nlp_bridge.go Refraction — the bracket is rejected to
// prevent Syntactic Noise from polluting the training corpus.
func EnforceMathSlot4(slot4 uint32, domainSig uint16) error {
    if domainSig != DomainMath {
        return nil
    }
    sym := MathSymbol(slot4 & 0xFF)
    switch sym {
    case MathSymbolOperand, MathSymbolVariable, MathSymbolOperator:
        return nil
    default:
        return fmt.Errorf("math mode: invalid symbol 0x%02X in Slot 4 — "+
            "raw SpaCy POS tag bypassed Refraction Layer (expected 0x01/0x02/0x04)", sym)
    }
}
```

**Integration in `TensorPacker.Orchestrate`** — the enforcement gate runs after
`PackSlot4` and before the frame is finalised:

```go
if err := EnforceMathSlot4(slots[4], domainSig); err != nil {
    return nil, err // reject bracket — prevents operator hallucination upstream
}
```

### 3.6 Security Schema

**File:** `pipeline/2_DATA_ENCODER/config/security_schema.yaml`

```yaml
domain:
  name: "SECURITY"
  slot10_base: 0x2400

subdomains:
  - id: 0x00
    name: "Constraint"
    slot10: 0x2400
  - id: 0x01
    name: "Guardrail"
    slot10: 0x2401
  - id: 0x02
    name: "Violation"
    slot10: 0x2402

pos_mappings:
  - id: 0x01
    name: "SUBJECT"
    spaCy_tags: ["NN", "NNS", "NNP", "NNPS"]
  - id: 0x02
    name: "ACTION"
    spaCy_tags: ["VB", "VBD", "VBG", "VBN", "VBP", "VBZ"]
  - id: 0x07
    name: "PREP"
    spaCy_tags: ["IN"]
```

---

## Phase 4: User Logic Gate Training

### 4.1 Training Module

**File:** `pipeline/3_DATA_TRAINER/pkg/training/user_security_gates.go`

```go
type UserSecurityGates struct {
    OrgID       string
    UserID      string
    Constraints []SecurityConstraint
    Patterns    []BehaviorPattern
    Seeds       []Seed
    Rules       []LogicalRule
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

type SecurityConstraint struct {
    RuleID   string
    Text     string
    Type     string  // "allow", "deny", "flag"
    Severity string  // "low", "medium", "high", "critical"
    Tags     []string
}

type BehaviorPattern struct {
    TaskType string
    Success  bool
    Metrics  map[string]float64
    ActionHash uint64
}

type LogicalRule struct {
    RuleType   string   // "subsumption", "disjoint", "constraint"
    Premises   []string
    Conclusion string
    Source     string   // "guardrail", "ontology", "user"
}

// SecurityFitness aggregates the four fitness axes used by Evo-GRPO during
// UserSecurityGate training. All fields are in [0.0, 1.0] unless noted.
type SecurityFitness struct {
    // Alignment (f.A): cosine similarity between the 16-dim LSH projection of the
    // generated frame and the ground-truth "Golden Frame" stored in the Arrow KB.
    // Computed as: dot(generated, golden) / (||generated|| * ||golden||)
    // A score < 0.7 causes the candidate to be discarded before selection.
    Alignment float64

    // Stability (f.S): 1 − Var(DriftScore[15..21]) across the final 7 passes of
    // Stave 3 (passes 15-21). A high-variance trajectory indicates the seed is
    // oscillating rather than converging; a stable seed yields f.S ≈ 1.0.
    // Computed as: 1.0 - clamp(Var(driftScores[15:22]), 0, 1)
    Stability float64

    Format           float64
    ConstraintScore  float64  // Does seed match constraints?
    ViolationPenalty float64  // Does seed avoid violations?
}

func (f *SecurityFitness) Total() float64 {
    return f.Alignment * 0.25 +
           f.Stability * 0.20 +
           f.Format * 0.15 +
           f.ConstraintScore * 0.25 +
           f.ViolationPenalty * 0.15
}

func (tg *UserTrainer) TrainUserGates(ctx context.Context, gates *UserSecurityGates) (*TrainedGates, error) {
    // 1. Extract constraint tokens from SecurityConstraints
    constraintTokens := tg.extractConstraintTokens(gates.Constraints)
    
    // 2. Extract violation patterns from BehaviorPatterns
    violationPatterns := tg.extractViolationPatterns(gates.Patterns)
    
    // 3. Run GRPO with security-aware fitness and difficulty modulation
    population := tg.initPopulation(len(constraintTokens))
    
    for gen := 0; gen < gates.MaxGenerations; gen++ {
        results := tg.evaluateWithSecurity(population, constraintTokens, violationPatterns)

        // Modulate GoldenSeed target difficulty per constraint type before selection.
        // Approved-context candidates receive a low-difficulty nonce target (2^16);
        // denied/sensitive candidates receive a high-difficulty nonce target (2^32).
        for _, r := range results {
            r.TargetDifficulty = tg.difficulty.TargetFor(r.ConstraintType)
        }

        population = tg.selectAndMutate(results)
        
        if gates.CheckConvergence(results) {
            break
        }
    }
    
    // 4. Build logical rules from constraints
    rules := tg.buildLogicalRules(gates.Constraints, gates.Patterns)
    
    return &TrainedGates{
        Seeds:      population.BestSeeds(),
        Rules:      rules,
        Fitness:    population.BestFitness(),
        UserID:     gates.UserID,
        TrainedAt:  time.Now(),
    }, nil
}
```

### 4.2 Training Triggers

```go
type TrainingScheduler struct {
    onDemand   chan *TrainingRequest
    scheduled  *cron.Cron
    violation  chan *GuardrailViolation
    client     *grpc.Client
}

func (ts *TrainingScheduler) Start() {
    // On-demand handler
    go ts.handleOnDemand()
    
    // Scheduled (e.g., every 24 hours)
    ts.scheduled.AddFunc("0 0 * * *", ts.runScheduledTraining)
    
    // Event-driven from guardrails
    go ts.handleViolations()
}
```

### 4.3 Difficulty Modulator

The `DifficultyModulator` is the policy bridge between Evo-GRPO and the
`GoldenSeed` nonce field. Rather than training toward an arbitrary nonce
target, Evo-GRPO modulates the **hash difficulty** based on the policy
classification of each candidate, making nonce magnitude a
hardware-validated proof of policy context (see Directive 1).

**File:** `pipeline/4_SEED_TRAINER/difficulty_modulator.go`

```go
// DifficultyApproved sets a 2^16 target: ASIC resolves quickly, producing
// a statistically low nonce — proof that no adversarial compute was required.
const DifficultyApproved uint32 = 1 << 16

// DifficultyFlagged is the intermediate tier for ambiguous or audit-required
// actions. Difficulty is set to 2^24 — meaningful compute, not a block.
const DifficultyFlagged uint32 = 1 << 24

// DifficultyDenied sets a 2^32 target: ASIC must burn the maximum nonce
// space, producing a statistically high nonce — Hardware-Validated Proof
// of Intent (HVPI). A high nonce is unforgeable without ASIC-class hardware.
const DifficultyDenied uint32 = 1<<32 - 1

type ConstraintType uint8

const (
    ConstraintApproved  ConstraintType = 0x01
    ConstraintFlagged   ConstraintType = 0x02
    ConstraintDenied    ConstraintType = 0x03
)

// DifficultyModulator maps policy constraint types to GoldenSeed nonce
// difficulty targets consumed by the Evo-GRPO trainer during TrainUserGates.
type DifficultyModulator struct{}

// TargetFor returns the uint32 nonce difficulty target for the given
// constraint classification. Evo-GRPO passes this value as TargetDifficulty
// into the seed evaluation step, biasing nonce evolution toward the tier.
func (d *DifficultyModulator) TargetFor(ct ConstraintType) uint32 {
    switch ct {
    case ConstraintApproved:
        return DifficultyApproved
    case ConstraintFlagged:
        return DifficultyFlagged
    case ConstraintDenied:
        return DifficultyDenied
    default:
        return DifficultyFlagged // conservative default
    }
}
```

**Nonce Interpretation:** At evaluation time, a `GoldenSeed` nonce near zero
indicates an approved-context bracket (resolved with 2^16 work). A nonce near
`0xFFFFFFFF` indicates a denied/sensitive bracket (resolved with 2^32 work).
This asymmetry is detectable without re-running the hash, making policy
context readable directly from the bracket's binary layout at
`GoldenSeed` bytes `0x29-0x2C`.

---

## Phase 5: KNIRVBASE Storage

KNIRVBASE is the exclusive storage and streaming layer for trained seed data. Seeds
are serialized into `.nrv` datasets — the tiered binary container from
`packages/KNIRVBASE/go/NRV_Master_Specification.md` — rather than raw JSON payloads.
The `hasher_rules` collection is **not used by KNIRVHASHER**; guardrail rule storage
and enforcement are the responsibility of KNIRVSERVER (see
`packages/KNIRVSERVER/KNIRV_Cognitive_Engine_Opportunities_Report.md` Section VI).

### 5.1 Collection Schema

**Collection:** `hasher_seeds`

Each entry's payload is a reference to a `.nrv` file on the KNIRVBASE server. The
`.nrv` file encodes one complete training run as a sequence of 80-byte Brackets:
- **Tier 1 Global Vault**: dataset identity, PQC manifest (Dilithium-3 + Kyber-768),
  total bracket count.
- **Tier 2 Temporal Frames**: one frame per training epoch; includes thermodynamic
  metadata and the `drift_score` between consecutive seed vectors.
- **Tier 3 Data Brackets**: each Bracket's `GoldenSeed` (Slot 11 nonce) is the
  trained seed value; `Projections` (Slots 0-3) encode the semantic context the seed
  was trained on; `DomainSig` (Slot 10) encodes the security domain tag.

```json
{
  "id": "user_{userID}_seeds_{trainingID}",
  "entryType": "MEMORY",
  "payload": {
    "org_id": "org_xxx",
    "user_id": "user_xxx",
    "training_id": "train_xxx",
    "trigger": "on_demand",
    "nrv_path": "/var/knirvbase/hasher_seeds/user_xxx/train_xxx.nrv",
    "nrv_dataset_id": "knirv_seeds_train_xxx",
    "bracket_count": 307,
    "fitness_mean": 0.95,
    "pqc_key_id": "dilithium3_key_001",
    "created_at": "2026-04-01T00:00:00Z"
  }
}
```

The `.nrv` file at `nrv_path` is self-describing: KNIRVSERVER retrieves brackets via
Apache Flight (`gold.hasher_seeds_{userID}`) to read only Z3-verified, Dilithium-3
signed frames without loading the full file.

### 5.2 Storage Client

**File:** `pkg/storage/knirvbase_client.go`

```go
type KNIRVBASEClient struct {
    db          knirvbase.DB
    nrvDatasets *knirvbase.NRVDataset
}

// SaveTrainedSeeds writes trained seed data as a .nrv dataset and registers
// the dataset path in the hasher_seeds collection.
func (c *KNIRVBASEClient) SaveTrainedSeeds(seeds *TrainedGates) error {
    // 1. Build .nrv dataset from trained brackets
    ds, err := c.nrvDatasets.NewNRV(fmt.Sprintf("knirv_seeds_%s", seeds.TrainingID))
    if err != nil {
        return err
    }

    for _, bracket := range seeds.ToBrackets() {
        if err := ds.AppendBracket(bracket); err != nil {
            return err
        }
    }

    nrvPath, err := ds.Flush() // writes, PQC-signs, returns absolute path
    if err != nil {
        return err
    }

    // 2. Register path in hasher_seeds collection
    collection := c.db.Collection("hasher_seeds")
    _, err = collection.Insert(map[string]interface{}{
        "id":           fmt.Sprintf("user_%s_seeds_%s", seeds.UserID, seeds.TrainingID),
        "entryType":    "MEMORY",
        "payload": map[string]interface{}{
            "org_id":         seeds.OrgID,
            "user_id":        seeds.UserID,
            "training_id":    seeds.TrainingID,
            "trigger":        seeds.Trigger,
            "nrv_path":       nrvPath,
            "nrv_dataset_id": ds.DatasetID(),
            "bracket_count":  len(seeds.ToBrackets()),
            "fitness_mean":   seeds.MeanFitness(),
            "pqc_key_id":     ds.PQCKeyID(),
            "created_at":     time.Now(),
        },
    })
    return err
}

// GetUserSeedPaths returns the .nrv paths for a user's trained seed datasets,
// newest first. KNIRVSERVER uses these paths to open Apache Flight streams.
func (c *KNIRVBASEClient) GetUserSeedPaths(userID string) ([]string, error) {
    collection := c.db.Collection("hasher_seeds")
    docs, err := collection.FindAll()
    if err != nil {
        return nil, err
    }
    var paths []string
    for _, doc := range docs {
        if doc["user_id"] == userID {
            if p, ok := doc["nrv_path"].(string); ok {
                paths = append(paths, p)
            }
        }
    }
    return paths, nil
}
```

---

## Phase 6: KNIRVBASE Server Integration

KNIRVHASHER owns its own KNIRVBASE instance. No other KNIRV component reads directly
from KNIRVHASHER's KNIRVBASE at runtime — the datasets it accumulates are for future
distribution via a planned global model update. Phase 6 covers the KNIRVBASE wiring
local to KNIRVHASHER for storage and Flight-based streaming of its training outputs.

### 6.1 KNIRVBASE Server in the Pipeline

KNIRVHASHER embeds a lightweight KNIRVBASE server instance (or connects to a shared
one) to handle all `.nrv` I/O. The pipeline writes to KNIRVBASE; KNIRVSERVER reads
from it via Apache Flight.

**File:** `pkg/storage/knirvbase_server.go`

```go
// HasherKNIRVBASE wraps the KNIRVBASE storage stack configured for the hasher.
type HasherKNIRVBASE struct {
    storage    *storage.NRVStorage
    flight     *network.FlightServer
    compactor  *storage.NRVCompactor
    collection string // "hasher_seeds"
}

func NewHasherKNIRVBASE(dataDir string) (*HasherKNIRVBASE, error) {
    stor, err := storage.NewNRVStorage(dataDir)
    if err != nil {
        return nil, err
    }
    return &HasherKNIRVBASE{
        storage:    stor,
        flight:     network.NewFlightServer(stor),
        compactor:  storage.NewNRVCompactor(stor),
        collection: "hasher_seeds",
    }, nil
}

// AppendTrainingBracket writes a trained bracket to the active .nrv dataset.
func (h *HasherKNIRVBASE) AppendTrainingBracket(b *nrv.Bracket) error {
    return h.storage.AppendBracket(h.collection, b)
}

// FlushFrame closes the current temporal frame, PQC-signs it (Dilithium-3),
// and returns the absolute path of the written .nrv file.
func (h *HasherKNIRVBASE) FlushFrame() (string, error) {
    return h.storage.FlushFrame(h.collection)
}

// StreamGoldBrackets opens an Apache Flight stream for Z3-verified brackets only.
// KNIRVSERVER consumes this stream to read trained seeds without copying files.
func (h *HasherKNIRVBASE) StreamGoldBrackets(ctx context.Context, srv network.BracketStreamServer) error {
    return h.flight.StreamBrackets(fmt.Sprintf("gold.%s", h.collection), srv)
}
```

### 6.2 Ticker-Driven Frame Flush

The `FrameTicker` (from `KNIRVBASE_Upgrade.md` §4 `internal/storage/nrv_ticker.go`)
drives the 1-second flush cycle. After each flush, the hasher_seeds KNIRVBASE
collection entry is updated with the new `.nrv` path.

```go
func (h *HasherKNIRVBASE) StartTicker(ctx context.Context, client *KNIRVBASEClient, userID, orgID, trainingID string) {
    ticker := time.NewTicker(time.Second)
    defer ticker.Stop()
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            path, err := h.FlushFrame()
            if err != nil || path == "" {
                continue
            }
            _ = client.UpdateNRVPath(userID, trainingID, path)
        }
    }
}
```

---

## File Structure Summary

```
KNIRVHASHER/
├── pipeline/
│   ├── 0_DATA_CONNECTOR/          # NEW: Go connector
│   │   ├── cmd/connector/main.go
│   │   ├── internal/
│   │   │   ├── grpc/client.go
│   │   │   ├── normalizer/
│   │   │   ├── encoder/nrv_encoder.go  # writes .nrv brackets
│   │   │   └── writer/
│   │   ├── config/connector.yaml
│   │   └── go.mod
│   ├── 1_DATA_MINER/
│   │   └── internal/app/knirv.go   # NEW: Load .nrv via Flight
│   ├── 2_DATA_ENCODER/
│   │   └── internal/tensor_packer.go   # MODIFY: .nrv bracket output
│   └── 3_DATA_TRAINER/
│       └── pkg/training/user_security_gates.go  # NEW
├── pkg/
│   └── storage/
│       ├── knirvbase_client.go          # NEW: .nrv path registry
│       └── knirvbase_server.go          # NEW: embedded KNIRVBASE server
└── internal/proto/hasher.proto          # NEW

KNIRVSERVER/
├── backend/
│   └── internal/
│       ├── proto/hasher.proto               # NEW (optional — only when stealth lifted)
│       ├── services/
│       │   └── dve/
│       │       └── hasher_export.go         # NEW: fire-and-forget export to KNIRVHASHER
│       └── web/
│           └── routes.go                    # MODIFY (pipeline phase 1-3 trigger endpoint)
└── Makefile                                 # MODIFY
```

---

## Implementation Order

| Phase | Task | Effort | Priority |
|-------|------|--------|----------|
| 1 | Define `proto/hasher.proto` (KNIRVHASHER gRPC server) | Low | P0 |
| 1 | Implement KNIRVHASHER gRPC server | Medium | P0 |
| 1 | Create `hasher_export.go` in KNIRVSERVER DVE (fire-and-forget) | Low | P0 |
| 2 | Build `0_DATA_CONNECTOR` in Go (writes .nrv brackets) | High | P0 |
| 3 | Update pipeline for .nrv format (bracket output) | Medium | P1 |
| 3 | Extract phases 1–3 for non-interactive use in KNIRVSERVER background | Medium | P1 |
| 4 | Implement `user_security_gates.go` | High | P1 |
| 5 | Create `knirvbase_client.go` + `knirvbase_server.go` (KNIRVHASHER local instance) | Medium | P1 |
| 6 | Wire embedded KNIRVBASE server + FrameTicker flush | Medium | P1 |
| 7 | Add training scheduler (cron) | Low | P2 |
| 8 | End-to-end testing (KNIRVHASHER standalone) | High | P2 |

---

## Testing Strategy

### Unit Tests
- `0_DATA_CONNECTOR`: Normalizer, .nrv bracket encoder
- `user_security_gates.go`: Fitness calculation, seed extraction
- `knirvbase_server.go`: AppendTrainingBracket, FlushFrame, PQC sign
- `knirvbase_client.go`: SaveTrainedSeeds writes correct .nrv path entry

### Integration Tests
```
KNIRVSERVER DVE → gRPC → 0_DATA_CONNECTOR → Pipeline → KNIRVBASE Server (.nrv)
KNIRVSERVER Flight client → KNIRVBASE gold stream → seed brackets → enforcement
```

### Validation Tests
| Test | Expected |
|------|----------|
| Training run completes → .nrv file created | `nrv_path` in hasher_seeds entry |
| KNIRVBASE Flight stream opened | Gold brackets stream without error |
| .nrv bracket count matches trained seeds | `bracket_count` == `len(seeds)` |
| Compactor triggered at 20% tombstones | Rewrite-and-swap completes |
| KNIRVSERVER reads seed brackets via Flight | Brackets decoded, DriftScore computed |
| Guardrail enforcement using seed brackets | See Cognitive Engine Report §VI tests |

---

## Open Questions

1. **gRPC Port**: Should the hasher gRPC service run on a fixed port (e.g., `:50051`) or dynamic?

2. **Training Concurrency**: Should multiple training jobs run in parallel, or queue sequentially?

3. **Seed Versioning**: Keep all `.nrv` datasets per user (historical), or only the latest? The KNIRVBASE compactor handles tombstoning, so history is cheap to retain.

4. **KNIRVBASE Topology**: Embedded KNIRVBASE server per hasher node vs. shared KNIRVBASE cluster? Shared enables Flight streaming to multiple KNIRVSERVER consumers.

5. **Guardrail Enforcement Handoff**: How does KNIRVSERVER know when new seed brackets are available? Push (hasher notifies via gRPC) vs. pull (KNIRVSERVER polls `hasher_seeds` collection).
