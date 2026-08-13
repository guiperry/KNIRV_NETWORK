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
| gRPC Transport | **Unix socket assigned by KNIRVSERVER; dynamic port fallback** | Primary: socket path pushed by KNIRVSERVER. Fallback: dynamic allocation from configured range |
| Pipeline Data Formats | **`.md` → `.arrow` → `.nrv`** | Raw ingest as Markdown; normalised records as Arrow IPC; encoded output as NRV brackets |
| Pipeline Concurrency | **Sequential jobs; parallel phases within a job** | One training job at a time per node; phases may overlap via goroutines at KNIRVBASE collection boundaries |
| Training Triggers | **On-demand + Scheduled + Event-driven** | Dataset collection across all operational scenarios |
| Seed Storage | **KNIRVBASE (.nrv files, embedded per node)** | Each node owns its own KNIRVBASE; sovereign architectural philosophy |
| Seed Versioning | **Full historical retention per user** | Never overwrite; all datasets retained for rollback, drift analysis, and future global model aggregation |
| KNIRVBASE Topology | **Embedded per hasher node** | Sovereign architecture — no shared cluster; data exchanged via network layer (Flight, IBC) |
| Connector Language | **Go** | Native to hasher, no TS overhead |
| Connector Location | **0_DATA_CONNECTOR** | Opens shared KNIRVBASE instance; all pipeline stages share the same db handle |
| KNIRVSERVER Usage | **Pipeline phases 1–3 only** | Non-interactive background calls during onboarding; full hasher stays isolated |
| Guardrail Enforcement | **Not KNIRVHASHER's concern** | Enforcement is a KNIRVSERVER responsibility using its own KNIRVBASE |
| Enforcement Handoff | **KNIRVSERVER polls `hasher_seeds` (pull)** | Cognitive Engine polls on configurable interval; hasher never pushes — full decoupling |
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
║  │  DVE data    │                    │  KNIRVBASE (imported)    │  ║
║  └──────────────┘                    │  writes raw → .md files  │  ║
║                                      └────────────┬─────────────┘  ║
║                                                   │ KNIRVBASE       ║
║                                                   │ Flight (.md)    ║
║                                                   ▼                 ║
║                                      ┌────────────────────────┐    ║
║                                      │    1_DATA_MINER        │    ║
║                                      │  reads .md → clean +   │    ║
║                                      │  normalize → .arrow    │    ║
║                                      └────────────┬───────────┘    ║
║                                                   │ KNIRVBASE       ║
║                                                   │ Flight (.arrow) ║
║                                                   ▼                 ║
║                                      ┌────────────────────────┐    ║
║                                      │    2_DATA_ENCODER      │    ║
║                                      │  reads .arrow → encode │    ║
║                                      │  + pack → .nrv         │    ║
║                                      └────────────┬───────────┘    ║
║                  (all stages share the KNIRVBASE   │                ║
║                   instance opened in connector)    ▼                ║
║  ┌────────────────────────────────────────────────────────────┐   ║
║  │                   HASHER PIPELINE                          │   ║
║  │                                                            │   ║
║  │  [Phase 1]         [Phase 2]         [Phase 3]            │   ║
║  │  1_DATA_MINER  ──▶ 2_DATA_ENCODER ──▶ 3_DATA_SEEDER     │   ║
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
│   └── writer/
│       └── writer.go           # MDWriter — writes raw decrypted chunks as .md files
├── config/
│   └── connector.yaml
├── go.mod                      # imports github.com/knirvcorp/knirvbase
└── Makefile
```

`0_DATA_CONNECTOR` is responsible for receiving the gRPC stream, decrypting each chunk,
and persisting it as a raw **`.md` file** in the `connector_raw` KNIRVBASE collection.
No normalisation or encoding happens here. The single KNIRVBASE instance opened in this
stage is passed by reference to `1_DATA_MINER` and `2_DATA_ENCODER` so all three stages
read and write to the same database.

The `normalizer` and `cleaner` packages live in `1_DATA_MINER`.
The `nrv_encoder` lives in `2_DATA_ENCODER`.

### 2.2 Main Entry Point

**File:** `cmd/connector/main.go`

`0_DATA_CONNECTOR` imports KNIRVBASE from `github.com/knirvcorp/knirvbase` and opens the
single shared database instance used by all pipeline stages. Its only job is to receive
the gRPC stream, decrypt each chunk, and write the raw content as a **`.md` file** into
the `connector_raw` KNIRVBASE collection. The `db` handle is then passed to the miner
and encoder so they share the same database without re-opening it.

```go
import (
    "github.com/knirvcorp/knirvbase"
)

func main() {
    config := LoadConfig()

    // Open the single shared KNIRVBASE instance for the whole pipeline.
    db, err := knirvbase.Open(config.KNIRVBASEDir)
    if err != nil {
        log.Fatalf("open knirvbase: %v", err)
    }
    defer db.Close()

    client := grpc.NewClient(config.HasherAddr)
    defer client.Close()

    stream, err := client.ExportSecurityData(&ExportRequest{
        OrgId:     config.OrgID,
        UserId:    config.UserID,
        DataType:  DataType_ALL,
        Encrypted: true,
    })
    if err != nil {
        log.Fatalf("export stream: %v", err)
    }

    // MDWriter saves each decrypted chunk as a raw .md file in connector_raw.
    // No normalisation or encoding happens here.
    w := writer.NewMDWriter(db.Collection("connector_raw"))

    for chunk := range stream {
        if err := w.WriteChunk(chunk); err != nil {
            log.Printf("write chunk %s: %v", chunk.ChunkId, err)
        }
    }

    // Hand the open db to downstream stages (invoked sequentially or as
    // goroutines sharing the same db handle).
    miner.Run(ctx, db)
    encoder.Run(ctx, db)
}
```

**`go.mod` (excerpt)**

```
module github.com/knirvcorp/knirvhasher/pipeline/0_DATA_CONNECTOR

require (
    github.com/knirvcorp/knirvbase v0.1.0
)
```

### 2.3 Security Normalizer

> **Location change:** This normalizer now lives in `1_DATA_MINER`. See §3.1 for the
> updated placement. The schema definition below is retained here for reference.

**File:** `pipeline/1_DATA_MINER/internal/normalizer/security_mapper.go`

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

### 2.4 NRV Encoder

> **Location:** `2_DATA_ENCODER`. See §3.2 for the directory layout and the
> `EncoderApp.Run` wiring that feeds this encoder from the KNIRVBASE Flight stream.

**File:** `pipeline/2_DATA_ENCODER/internal/encoder/nrv_encoder.go`

`NRVEncoder` reads **`.arrow` IPC batches** produced by `1_DATA_MINER` from the
`miner_processed` KNIRVBASE collection, runs the BGE embedding through TensorPacker,
and writes each packed record as an **80-byte `.nrv` Tier-3 Bracket** into the
`encoder_output` KNIRVBASE collection via `NRVWriter`.

The embedding field is **fixed-size 16-dimensional**, matching the 16 variance-selected
BGE dimensions from `vector_mapper.go` and the 16-dim LSH projection space used by
`projectionsToFloat32` in KNIRVBASE's `flight_server.go`. Variable-length embedding
lists are not permitted; KNIRVBASE `CalcBracketDriftScore` assumes a fixed 16-dim stride.

```go
// VectorDims is the fixed dimensionality of all BGE embedding vectors in the pipeline.
// Must match vector_mapper.go variance selection, NRV KB index, and KNIRVBASE
// Projections layout (32 bytes → 16 × uint16 dimensions).
const VectorDims = 16

// NRVEncoder reads .arrow IPC batches from the miner_processed KNIRVBASE collection
// and encodes each SecurityRecord into an 80-byte .nrv Tier-3 Bracket.
type NRVEncoder struct {
    db     knirvbase.DB
    writer *writer.NRVWriter
    packer *TensorPacker
}

func NewNRVEncoder(db knirvbase.DB) *NRVEncoder {
    return &NRVEncoder{
        db:     db,
        writer: writer.NewNRVWriter(db.Collection("encoder_output")),
        packer: NewTensorPacker(),
    }
}

// Run opens a Flight stream on the miner_processed collection, reads each
// .arrow file path from the entry payload, memory-maps the IPC file, and
// encodes every SecurityRecord row into a .nrv Bracket via NRVWriter.
func (e *NRVEncoder) Run(ctx context.Context) error {
    stream, err := e.db.Collection("miner_processed").FlightStream(ctx)
    if err != nil {
        return fmt.Errorf("open miner_processed flight stream: %w", err)
    }

    for entry := range stream {
        arrowPath, _ := entry.Payload["arrow_path"].(string)
        records, err := loadArrowBatch(arrowPath) // mmap + IPC decode
        if err != nil {
            log.Printf("nrv_encoder: load %s: %v", arrowPath, err)
            continue
        }

        for i, rec := range records {
            slots := e.packer.Orchestrate(rec.ToSlotVector(), uint16(i), rec.DomainSig)
            bracket := &knirvbase.Bracket{
                // Slots 0-3: 16-dim LSH projections packed as 16 × uint16 (32 bytes)
                Projections: slotsToProjections(slots),
                // Slot 4: packed syntax byte (POSTag | Tense | Plurality)
                SyntacticByte: uint8(slots[4] & 0xFF),
                // Slot 5: dependency head
                DepHead: uint8(slots[5]),
                // Slot 10: domain signature (e.g. 0x2400 = Security domain)
                DomainSig: uint16(slots[10]),
                // Slots 6-8: recursive context memory (18 bytes)
                ContextMemory: slots6to8(slots[6:9]),
                // Slot 11: GoldenSeed / LSH Salt
                GoldenSeed: slots[11],
            }
            if err := e.writer.WriteBracket(bracket); err != nil {
                log.Printf("nrv_encoder: write bracket %d: %v", i, err)
            }
        }
    }
    return nil
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

knirvbase:
  data_dir: "/var/knirvbase/hasher"      # shared database for all pipeline stages
  collections:
    connector_raw:   "connector_raw"     # .md files — raw decrypted gRPC chunks
    miner_processed: "miner_processed"   # .arrow files — cleaned + normalised records
    encoder_output:  "encoder_output"    # .nrv brackets — fully encoded output
  batch_size: 100

processing:
  max_concurrent: 4
  pii_scrub: true
  deduplicate: true
```

---

## Phase 3: Pipeline Updates

### 3.1 Data Miner Updates

`1_DATA_MINER` receives the shared KNIRVBASE handle opened by `0_DATA_CONNECTOR`. It
reads raw **`.md` files** from the `connector_raw` collection via Apache Flight, runs
each document through the SpaCy NLP pipeline (clean → normalize), and writes the output
as **`.arrow` files** (Apache Arrow IPC) into the `miner_processed` KNIRVBASE collection
for `2_DATA_ENCODER` to consume.

#### Directory additions

```
pipeline/1_DATA_MINER/
├── internal/
│   ├── app/
│   │   └── knirv.go             # KNIRVBASE Flight consumer; orchestrates stages
│   ├── normalizer/
│   │   ├── normalizer.go
│   │   └── security_mapper.go   # SecurityRecord schema + tag mappings
│   ├── cleaner/
│   │   └── cleaner.go           # PII scrub, dedup, token sanitisation
│   └── writer/
│       └── writer.go            # ArrowWriter — writes .arrow IPC files to KNIRVBASE
└── internal/nlp_bridge.go       # MathContextDetector (§3.4)
```

**File:** `pipeline/1_DATA_MINER/internal/app/knirv.go`

```go
import (
    "github.com/apache/arrow/go/v14/arrow"
    "github.com/apache/arrow/go/v14/arrow/ipc"
    "github.com/apache/arrow/go/v14/arrow/memory"
    "github.com/knirvcorp/knirvbase"
    "github.com/knirvcorp/knirvhasher/pipeline/1_DATA_MINER/internal/cleaner"
    "github.com/knirvcorp/knirvhasher/pipeline/1_DATA_MINER/internal/normalizer"
    "github.com/knirvcorp/knirvhasher/pipeline/1_DATA_MINER/internal/writer"
)

// MinerApp reads raw .md files from the KNIRVBASE connector_raw collection,
// cleans and normalises each document via SpaCy NLP, then writes the output
// as .arrow IPC files into miner_processed for 2_DATA_ENCODER.
type MinerApp struct {
    db         knirvbase.DB
    normalizer *normalizer.SecurityNormalizer
    cleaner    *cleaner.Cleaner
    writer     *writer.ArrowWriter
}

func NewMinerApp(db knirvbase.DB) *MinerApp {
    return &MinerApp{
        db:         db,
        normalizer: normalizer.NewSecurityNormalizer(),
        cleaner:    cleaner.New(),
        writer:     writer.NewArrowWriter(db.Collection("miner_processed")),
    }
}

// Run opens a Flight stream on the connector_raw collection (raw .md files),
// processes each document through the cleaner → normalizer chain, and writes
// the resulting SecurityRecords as a .arrow IPC batch to miner_processed.
func (m *MinerApp) Run(ctx context.Context) error {
    stream, err := m.db.Collection("connector_raw").FlightStream(ctx)
    if err != nil {
        return fmt.Errorf("open connector_raw flight stream: %w", err)
    }

    for mdDoc := range stream {
        // mdDoc.RawData is the decrypted Markdown text written by 0_DATA_CONNECTOR.
        cleaned, err := m.cleaner.CleanMarkdown(mdDoc.RawData)
        if err != nil {
            log.Printf("cleaner: skip %s: %v", mdDoc.ID, err)
            continue
        }

        records, err := m.normalizer.Process(cleaned)
        if err != nil {
            log.Printf("normalizer: skip %s: %v", mdDoc.ID, err)
            continue
        }

        // Write the batch of SecurityRecords as a single .arrow IPC file.
        if err := m.writer.WriteBatch(mdDoc.ID, records); err != nil {
            log.Printf("arrow writer: %v", err)
        }
    }
    return nil
}
```

**File:** `pipeline/1_DATA_MINER/internal/writer/writer.go`

```go
// ArrowWriter serialises batches of SecurityRecords into Apache Arrow IPC files
// and registers each file in the miner_processed KNIRVBASE collection.
// The .arrow file path is stored as the collection entry's payload so that
// 2_DATA_ENCODER can locate and memory-map it via Apache Flight.
type ArrowWriter struct {
    collection knirvbase.Collection
    pool       memory.Allocator
    schema     *arrow.Schema
}

func NewArrowWriter(collection knirvbase.Collection) *ArrowWriter {
    embeddingType := arrow.FixedSizeListOf(VectorDims, arrow.PrimitiveTypes.Float32)
    schema := arrow.NewSchema([]arrow.Field{
        {Name: "file_name",     Type: arrow.BinaryTypes.String},
        {Name: "chunk_id",      Type: arrow.PrimitiveTypes.Int32},
        {Name: "content",       Type: arrow.BinaryTypes.String},
        {Name: "embedding",     Type: embeddingType},
        {Name: "tokens",        Type: arrow.ListOf(arrow.BinaryTypes.String)},
        {Name: "pos_tags",      Type: arrow.ListOf(arrow.PrimitiveTypes.Int32)},
        {Name: "dep_hashes",    Type: arrow.ListOf(arrow.PrimitiveTypes.Uint32)},
        {Name: "security_tags", Type: arrow.ListOf(arrow.BinaryTypes.String)},
        {Name: "domain_sig",    Type: arrow.PrimitiveTypes.Uint16},
        {Name: "slot4_raw",     Type: arrow.PrimitiveTypes.Uint32},
    }, nil)
    return &ArrowWriter{collection: collection, pool: memory.NewGoAllocator(), schema: schema}
}

// WriteBatch encodes records as an Arrow IPC stream file and registers the
// resulting .arrow path in the miner_processed collection.
func (w *ArrowWriter) WriteBatch(docID string, records []*SecurityRecord) error {
    path := filepath.Join(w.collection.DataDir(), docID+".arrow")
    f, err := os.Create(path)
    if err != nil {
        return fmt.Errorf("arrow writer: create %s: %w", path, err)
    }
    defer f.Close()

    fw, err := ipc.NewFileWriter(f, ipc.WithSchema(w.schema), ipc.WithAllocator(w.pool))
    if err != nil {
        return err
    }
    if err := fw.Write(buildRecordBatch(w.schema, w.pool, records)); err != nil {
        fw.Close()
        return err
    }
    fw.Close()

    // Register the .arrow path in miner_processed so 2_DATA_ENCODER can stream it.
    return w.collection.Insert(map[string]interface{}{
        "id":      docID,
        "payload": map[string]interface{}{"arrow_path": path},
    })
}
```

### 3.2 Data Encoder: NRV Encoder + Tensor Packer (Re-tooled)

`2_DATA_ENCODER` receives the shared KNIRVBASE handle. It reads the **`.arrow` IPC
files** registered in the `miner_processed` collection by `1_DATA_MINER`, runs the
BGE embedding + TensorPacker stages, and writes the result as **`.nrv` Tier-3
Brackets** into the `encoder_output` KNIRVBASE collection for `3_DATA_SEEDER`.

#### Directory additions

```
pipeline/2_DATA_ENCODER/
├── internal/
│   ├── encoder/
│   │   └── nrv_encoder.go   # NRVEncoder — reads .arrow, writes .nrv brackets
│   └── writer/
│       └── writer.go        # NRVWriter — writes .nrv brackets to KNIRVBASE
```

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
    nrvKB       *NRVKnowledgeBase // 16-dim re-indexed NRV KB
}

// Orchestrate builds the full 12-slot uint32 array for one bracket.
// It injects the Temporal Salt (Slot 11) and Flash Search jitter (Slot 4 upper bits).
func (tp *TensorPacker) Orchestrate(base *SlotVector, pos uint16, domainSig uint16) []uint32 {
    slots := base.Copy()

    // Slot 11: Temporal Salt — (PosIndex << 16) | TemporalSalt
    salt := uint16(time.Now().UnixNano() & 0xFFFF)
    slots[11] = uint32(pos) | (uint32(salt) << 16)

    // Flash Search: use first 4 bytes of current hash state as lookup key.
    jitter := tp.flashSearch.Lookup(slots[:4], tp.nrvKB)

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

// SaveTrainingFrames writes frames as .nrv Tier-3 Brackets into the
// encoder_output KNIRVBASE collection via the embedded NRVWriter.
func (tp *TensorPacker) SaveTrainingFrames(frames []*NeuralFrame, w *writer.NRVWriter) error {
    for _, frame := range frames {
        bracket := &knirvbase.Bracket{
            Projections:   slotsToProjections(frame.Slots),   // Slots 0-3 → 32 bytes
            SyntacticByte: uint8(frame.Slots[4] & 0xFF),      // Slot 4 operator byte
            DepHead:       uint8(frame.Slots[5]),              // Slot 5
            IntentFlags:   uint8(frame.Slots[9]),              // Slot 9
            DomainSig:     uint16(frame.Slots[10]),            // Slot 10
            ContextMemory: slots6to8(frame.Slots[6:9]),        // Slots 6-8 → 18 bytes
            GoldenSeed:    frame.Slots[11],                    // Slot 11 nonce
            DomainSig16:   uint16(frame.DomainSig),
        }
        if err := w.WriteBracket(bracket); err != nil {
            return fmt.Errorf("saveTrainingFrames: frame %d: %w", frame.FrameID, err)
        }
    }
    return nil
}
```

#### Flash Search Helper

**File:** `pipeline/2_DATA_ENCODER/internal/flash_search.go`

```go
// FlashSearchHelper performs the Lookup Key → Jitter Vector retrieval described
// in LSH_Salt.md. The first 4 bytes of the current hash state are used as a key
// into the 16-dim NRV Knowledge Base. The returned 24-bit jitter value is
// injected into Slot 4 bits 8–31 by TensorPacker.Orchestrate.
type FlashSearchHelper struct{}

// Lookup queries the NRV KB for the nearest 16-dim entry matching hashPrefix
// (first 4 bytes of the ASIC hash state) and extracts a 24-bit jitter payload.
func (f *FlashSearchHelper) Lookup(hashPrefix []uint32, kb *NRVKnowledgeBase) uint32 {
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
dimensions are used consistently across training, inference, and NRV KB indexing.

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

### 3.5 NRV Knowledge Base Re-Indexing

**File:** `pipeline/2_DATA_ENCODER/internal/nrv_kb.go` (new)

The NRV Knowledge Base backing the Flash Search helper must be re-indexed from
4-dim to **16-dim** vectors and stored as `.nrv` brackets in KNIRVBASE rather than
parquet or Arrow IPC files. Any existing `.parquet` or `.arrow` KB files built with the
old 4-dim schema are incompatible and must be regenerated.

```go
// NRVKBEntry is one row in the 16-dim NRV Knowledge Base.
type NRVKBEntry struct {
    TokenID   uint32              // vocabulary token identifier
    HashKey   uint32              // first 4 bytes of the associated ASIC hash (lookup key)
    Embedding [VectorDims]float32 // 16 variance-selected BGE dimensions
    DomainSig uint16              // Slot 10 domain at time of indexing
    Slot4Raw  uint32              // packed Slot 4 register at time of indexing
}

// NRVKnowledgeBase provides nearest-neighbour lookup over the 16-dim index.
// Entries are loaded from the `kb_vocab` KNIRVBASE collection at startup.
type NRVKnowledgeBase struct {
    entries []NRVKBEntry
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
func (kb *NRVKnowledgeBase) NearestByHashKey(key uint32) (*NRVKBEntry, uint32) {
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

// LoadFromKNIRVBASE populates the NRVKnowledgeBase from the `kb_vocab` collection
// via Apache Flight. Call this once at startup before any Lookup calls.
func (kb *NRVKnowledgeBase) LoadFromKNIRVBASE(ctx context.Context, db knirvbase.DB) error {
    stream, err := db.Collection("kb_vocab").FlightStream(ctx)
    if err != nil {
        return fmt.Errorf("nrv_kb: open flight stream: %w", err)
    }
    for bracket := range stream {
        entry := NRVKBEntry{
            TokenID:   bracket.GoldenSeed,
            HashKey:   binary.LittleEndian.Uint32(bracket.Projections[:4]),
            DomainSig: bracket.DomainSig,
            Slot4Raw:  uint32(bracket.SyntacticByte),
        }
        copy(entry.Embedding[:], bracket.EmbeddingFloat32())
        kb.entries = append(kb.entries, entry)
    }
    return nil
}
```

> **Migration:** Run `pipeline/2_DATA_ENCODER/cmd/reindex/main.go` to rebuild the
> NRV KB from raw BGE embeddings using the new `variance_indices.json` and write the
> output into the `kb_vocab` KNIRVBASE collection as `.nrv` brackets. This is a
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

**File:** `pipeline/3_DATA_SEEDER/pkg/training/user_security_gates.go`

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
│   │
│   │  ┌─ KNIRVBASE (shared, opened once in 0_DATA_CONNECTOR) ──────────────┐
│   │  │  connector_raw     → .md files   (raw decrypted Markdown)          │
│   │  │  miner_processed   → .arrow files (cleaned + normalised records)   │
│   │  │  encoder_output    → .nrv brackets (fully encoded, 80-byte ASIC)   │
│   │  └────────────────────────────────────────────────────────────────────┘
│   │
│   ├── 0_DATA_CONNECTOR/                    # gRPC ingest → .md → connector_raw
│   │   ├── cmd/connector/main.go            # opens KNIRVBASE; hands db to miner+encoder
│   │   ├── internal/
│   │   │   ├── grpc/client.go
│   │   │   └── writer/writer.go             # MDWriter → connector_raw (.md files)
│   │   ├── config/connector.yaml            # knirvbase.data_dir + collection names
│   │   └── go.mod                           # requires github.com/knirvcorp/knirvbase
│   │
│   ├── 1_DATA_MINER/                        # .md → SpaCy NLP → .arrow → miner_processed
│   │   ├── internal/
│   │   │   ├── app/knirv.go                 # Flight stream on connector_raw; ArrowWriter out
│   │   │   ├── normalizer/
│   │   │   │   ├── normalizer.go
│   │   │   │   └── security_mapper.go       # SecurityRecord schema + tag mappings
│   │   │   ├── cleaner/
│   │   │   │   └── cleaner.go               # PII scrub, dedup, token sanitisation
│   │   │   └── writer/writer.go             # ArrowWriter → miner_processed (.arrow files)
│   │   └── internal/nlp_bridge.go           # MathContextDetector (§3.4)
│   │
│   ├── 2_DATA_ENCODER/                      # .arrow → BGE embed + TensorPack → .nrv
│   │   ├── internal/
│   │   │   ├── encoder/nrv_encoder.go       # reads .arrow; writes .nrv brackets (§2.4)
│   │   │   ├── writer/writer.go             # NRVWriter → encoder_output (.nrv brackets)
│   │   │   ├── tensor_packer.go             # 12-slot orchestration; NRVKnowledgeBase
│   │   │   ├── nrv_kb.go                    # NRVKnowledgeBase (Flash Search)
│   │   │   ├── flash_search.go
│   │   │   ├── vector_mapper.go
│   │   │   └── math_mode.go
│   │   └── config/
│   │       ├── security_schema.yaml
│   │       └── variance_indices.json
│   │
│   └── 3_DATA_SEEDER/
│       └── pkg/training/user_security_gates.go
├── pkg/
│   └── storage/
│       ├── knirvbase_client.go              # .nrv path registry (hasher_seeds)
│       └── knirvbase_server.go              # embedded KNIRVBASE server + Flight
└── internal/proto/hasher.proto

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
| 2 | Build `0_DATA_CONNECTOR` — import `github.com/knirvcorp/knirvbase`; open shared DB; `MDWriter` saves raw gRPC chunks as `.md` into `connector_raw` | High | P0 |
| 3 | Build `1_DATA_MINER` — `ArrowWriter` reads `.md` from `connector_raw` via Flight; clean + normalise; write `.arrow` IPC to `miner_processed` | Medium | P1 |
| 3 | Build `2_DATA_ENCODER` — `NRVEncoder` reads `.arrow` from `miner_processed` via Flight; TensorPacker → 80-byte `.nrv` brackets; `NRVWriter` to `encoder_output` | Medium | P1 |
| 3 | Extract phases 1–3 for non-interactive use in KNIRVSERVER background | Medium | P1 |
| 4 | Implement `user_security_gates.go` | High | P1 |
| 5 | Create `knirvbase_client.go` + `knirvbase_server.go` (KNIRVHASHER local instance) | Medium | P1 |
| 6 | Wire embedded KNIRVBASE server + FrameTicker flush | Medium | P1 |
| 7 | Add training scheduler (cron) | Low | P2 |
| 8 | End-to-end testing (KNIRVHASHER standalone) | High | P2 |

---

## Testing Strategy

### Unit Tests
- `0_DATA_CONNECTOR`: gRPC client decrypt, MDWriter → `connector_raw` (`.md` files stored correctly)
- `1_DATA_MINER`: Cleaner (PII scrub, dedup), SecurityNormalizer, ArrowWriter — confirm `.arrow` IPC schema matches §3.1; Flight stream consumer reads `.md` entries
- `2_DATA_ENCODER`: NRVEncoder reads `.arrow` batch → TensorPacker → NRVWriter; NRVKnowledgeBase Hamming lookup; NRVWriter produces valid 80-byte brackets in `encoder_output`
- `user_security_gates.go`: Fitness calculation, seed extraction
- `knirvbase_server.go`: AppendTrainingBracket, FlushFrame, PQC sign
- `knirvbase_client.go`: SaveTrainedSeeds writes correct .nrv path entry

### Integration Tests
```
KNIRVSERVER DVE → gRPC → 0_DATA_CONNECTOR
  → connector_raw (.md) via KNIRVBASE Flight
  → 1_DATA_MINER (clean + normalise)
  → miner_processed (.arrow) via KNIRVBASE Flight
  → 2_DATA_ENCODER (BGE embed + TensorPack)
  → encoder_output (.nrv brackets) via KNIRVBASE Flight
  → 3_DATA_SEEDER (Evo-GRPO seeds)
  → hasher_seeds collection (.nrv datasets)
KNIRVSERVER Flight client → KNIRVBASE gold stream → seed brackets → enforcement
```

### Format Validation
| Stage | Input | Output | Validation |
|-------|-------|--------|------------|
| `0_DATA_CONNECTOR` | gRPC `EncryptedChunk` | `.md` in `connector_raw` | File is valid UTF-8 Markdown |
| `1_DATA_MINER` | `.md` from `connector_raw` | `.arrow` in `miner_processed` | IPC schema matches §3.1 ArrowWriter schema; `embedding` is `FixedSizeList<Float32>(16)` |
| `2_DATA_ENCODER` | `.arrow` from `miner_processed` | `.nrv` in `encoder_output` | Bracket is 80 bytes; Slot 4 passes `EnforceMathSlot4`; GoldenSeed in correct nonce tier |

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

## Architecture Decisions (Resolved)

The following questions were resolved and are now canonical. Implementation sections
have been updated accordingly.

---

### 1. gRPC Transport — Unix Socket + Dynamic Port Fallback

**Decision:** Primary transport is a Unix socket path **assigned by KNIRVSERVER** at
startup. If no socket path is provided, the hasher falls back to dynamic port
allocation from a configured range.

**`config/connector.yaml`:**

```yaml
hasher:
  # KNIRVSERVER writes the assigned socket path here at startup.
  # Leave empty to trigger dynamic port fallback.
  socket: ""                         # e.g. "/var/run/knirvhasher_<nodeID>.sock"
  dynamic_port_range: "50100-50199"  # fallback range
  timeout: 300
```

**`internal/grpc/client.go`:**

```go
// DialHasher connects via the Unix socket assigned by KNIRVSERVER.
// Falls back to a dynamically allocated port if no socket path is set.
func DialHasher(cfg *Config) (*grpc.ClientConn, error) {
    if cfg.Socket != "" {
        return grpc.Dial("unix://"+cfg.Socket, grpc.WithTransportCredentials(tlsCreds))
    }
    port, err := allocateDynamicPort(cfg.DynamicPortRange)
    if err != nil {
        return nil, fmt.Errorf("grpc dial: dynamic port allocation failed: %w", err)
    }
    return grpc.Dial(fmt.Sprintf("localhost:%d", port), grpc.WithTransportCredentials(tlsCreds))
}
```

---

### 2. Training Concurrency — Sequential Jobs, Parallel Pipeline Phases

**Decision:** Training jobs are queued and executed **sequentially** — one job at a
time per hasher node. Within a single job the pipeline phases (`1_DATA_MINER`,
`2_DATA_ENCODER`, `3_DATA_SEEDER`) may execute **in parallel** as goroutines where
their KNIRVBASE collection boundaries allow it. The collections serve as the
back-pressure boundary between phases.

**`pipeline/3_DATA_SEEDER/pkg/training/scheduler.go`:**

```go
// TrainingQueue serialises training jobs. Phases within each job may overlap
// via goroutines; the next job does not start until the current one finishes.
type TrainingQueue struct {
    jobs chan *TrainingJob
}

func (q *TrainingQueue) Start(ctx context.Context, db knirvbase.DB) {
    for job := range q.jobs {
        // Miner must complete before encoder has records to read.
        if err := miner.Run(ctx, db); err != nil {
            log.Printf("miner: %v", err)
            continue
        }
        // Encoder and trainer can overlap once encoder has written its first batch.
        var wg sync.WaitGroup
        wg.Add(2)
        go func() { defer wg.Done(); encoder.Run(ctx, db) }()
        go func() { defer wg.Done(); RunTrainer(ctx, job, db) }()
        wg.Wait()
    }
}
```

---

### 3. Seed Versioning — Full Historical Retention Per User

**Decision:** All `.nrv` datasets are retained for every user indefinitely. No dataset
is overwritten or deleted. The KNIRVBASE compactor reclaims only tombstoned brackets
within a file; it never removes dataset files. Full history enables rollback, semantic
drift analysis over time, and future global model aggregation across nodes.

**Impact on `knirvbase_client.go`:** `SaveTrainedSeeds` always inserts a **new** entry
with a unique `training_id`. It never upserts. `GetUserSeedPaths` returns all paths
newest-first so KNIRVSERVER reads the most recent dataset by default.

---

### 4. KNIRVBASE Topology — Embedded Per Node (Sovereign Architecture)

**Decision:** Each hasher node runs its own **embedded KNIRVBASE instance**. There is
no shared cluster. This is a direct expression of KNIRV's **sovereign architectural
philosophy** — every node owns its data independently. Datasets are exchanged through
the network layer (Apache Flight, IBC), never through a shared database backend.

**Impact on `knirvbase_server.go`:** `HasherKNIRVBASE` is always constructed with a
node-local `dataDir`. No cluster config, service discovery, or remote storage is
required or supported at this layer.

---

### 5. Guardrail Enforcement Handoff — Periodic Pull by Cognitive Engine

**Decision:** KNIRVSERVER's Cognitive Engine **polls** the `hasher_seeds` KNIRVBASE
collection on a configurable interval. The hasher never pushes or notifies — it only
writes. This preserves full decoupling: the hasher can be offline, slow, or in stealth
mode without affecting KNIRVSERVER's availability.

**KNIRVSERVER `backend/internal/services/cognitiveengine/cognitive_engine.go`:**

```go
// SeedPoller periodically checks the hasher_seeds KNIRVBASE collection for
// new .nrv datasets and loads them into the active enforcement layer.
type SeedPoller struct {
    client   *KNIRVBASEClient
    interval time.Duration        // configurable; default 5 minutes
    lastSeen map[string]time.Time // userID → timestamp of last training_id loaded
}

func (p *SeedPoller) Start(ctx context.Context) {
    ticker := time.NewTicker(p.interval)
    defer ticker.Stop()
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            p.poll(ctx)
        }
    }
}

func (p *SeedPoller) poll(ctx context.Context) {
    entries, err := p.client.ListNewSeedEntries(p.lastSeen)
    if err != nil {
        log.Printf("seed poller: %v", err)
        return
    }
    for _, entry := range entries {
        if err := p.loadSeedDataset(ctx, entry.NRVPath); err != nil {
            log.Printf("seed poller: load %s: %v", entry.NRVPath, err)
            continue
        }
        p.lastSeen[entry.UserID] = entry.CreatedAt
    }
}
```
