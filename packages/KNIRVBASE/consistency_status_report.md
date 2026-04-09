# KNIRVBASE Go → TypeScript Consistency Status Report

**Date:** 2026-04-09  
**Scope:** Go SDK (`go/`) → TypeScript SDK (`ts/`) translation gaps  
**Source of truth:** Go implementation

---

## Summary

The TypeScript SDK covers the core database operations, networking, CRDT, auth, crypto, embedding, query, and storage layers well. However, several significant gaps exist — ranging from entirely missing modules to schema mismatches between equivalent types — that block full parity with the Go SDK.

---

## 1. Missing: `NewNRV` Factory + `NRVDataset` API

**Severity: Critical**

### What Go has

`pkg/knirvbase/knirvbase.go`:

```go
// Factory that creates a DB backed by NRVStorage with PQC key pair
func NewNRV(ctx context.Context, opts Options, keyPair *pqc.PQCKeyPair) (*DB, error)

// Method on DB — only works when created via NewNRV
func (d *DB) Dataset(name string) *NRVDataset

// NRVDataset — high-level NRV operations
func (ds *NRVDataset) AppendBracket(ctx context.Context, b *nrv.Bracket, thermo nrv.ThermoAtmosphere) error
func (ds *NRVDataset) StreamBrackets(ctx context.Context, goldOnly bool) (<-chan *nrv.Bracket, error)
func (ds *NRVDataset) GetFrame(ctx context.Context, frameID string) (*nrv.FrameEntry, []*nrv.Bracket, error)
func (ds *NRVDataset) SetLinguistic(token, unit string) error
```

### What TS has

`lib/index.ts`: Only `New()` and `NewDistributedDatabase()`. No `NewNRV()`, no `dataset()` method on `DB`, no `NRVDataset` class.

### Work needed

- Add `NewNRV(ctx, opts, keyPair: PQCKeyPair): Promise<DB>` factory to `lib/index.ts`
- Add `dataset(name: string): NRVDataset` method to `DB` class (guard: must be NRVStorage-backed)
- Implement and export `NRVDataset` class with `appendBracket`, `streamBrackets`, `getFrame`, `setLinguistic`

---

## 2. Missing: Distributed Tracing Module

**Severity: High**

### What Go has

`internal/tracing/tracing.go`:

```go
func InitTracer(serviceName, jaegerEndpoint string) (*sdktrace.TracerProvider, error)
func StartSpan(ctx context.Context, operationName string, attrs ...attribute.KeyValue) (context.Context, trace.Span)
```

OpenTelemetry SDK + Jaeger exporter.

### What TS has

Nothing. No `components/tracing/` directory exists.

### Work needed

- Create `ts/src/components/tracing/index.ts`
- Implement `initTracer(serviceName: string, jaegerEndpoint: string): TracerProvider` using `@opentelemetry/sdk-trace-node` + `@opentelemetry/exporter-jaeger`
- Implement `startSpan(operationName: string, attrs?: Record<string, any>): Span`
- Export from `lib/index.ts`

---

## 3. Missing: Apache Arrow Flight Server

**Severity: High**

### What Go has

`internal/network/flight_server.go`:

```go
type FlightServer struct { storage *NRVStorage; schema *arrow.Schema }
func NewFlightServer(storage *NRVStorage) *FlightServer
func (s *FlightServer) StreamBrackets(ticket string, server BracketStreamServer) error

type FlightClient struct { conn io.Reader }
func NewFlightClient(conn io.Reader) *FlightClient
func (c *FlightClient) StreamBrackets(ctx context.Context, ticket string) ([]*nrv.Bracket, error)

// Utility functions (standalone, no server needed)
func BracketsToFlightData(brackets []*nrv.Bracket) ([]byte, error)
func FlightDataToBrackets(data []byte) ([]*nrv.Bracket, error)
```

Arrow IPC wire format with schema: `frame_id`, `lsh_salt`, `subsecond_us`, `asic_loops`, `golden_seed`, `drift_score`, `bracket_type`, `projections` (64-byte fixed binary), `frame_timestamp`.

Ticket format: `"<gold|all>.<collection>"` — e.g., `"gold.training"`.

### What TS has

Nothing. No Arrow IPC, no Flight protocol, no batch streaming.

### Work needed

- Add `apache-arrow` (or `@apache-arrow/ts`) dependency to `ts/package.json`
- Create `ts/src/components/network/flight_server.ts` implementing:
  - `FlightServer` class with the same Arrow schema
  - `FlightClient` class
  - `bracketsToFlightData(brackets: Bracket[]): Uint8Array`
  - `flightDataToBrackets(data: Uint8Array): Bracket[]`
- Export from `lib/index.ts`

---

## 4. Missing: Standalone Indexing Module (`internal/indexing/`)

**Severity: High**

### What Go has

`internal/indexing/indexing.go` + `hnsw.go` + `debug_hnsw.go`:

```go
type IndexType string // "semantic" | "temporal" | "category" | "fulltext" | "tag"

// Generic block interface (uuid-based, used across all index types)
type Block interface {
    GetBlockID() uuid.UUID
    GetTimestamp() int64
    GetCategory() types.MemoryCategory
    GetSemanticVector() []float32
}

type Index interface {
    Add(ctx context.Context, block Block) error
    Search(ctx context.Context, query interface{}) ([]uuid.UUID, error)
    Remove(ctx context.Context, blockID uuid.UUID) error
    Rebuild(ctx context.Context) error
}

type MultiIndexManager struct{}
func NewMultiIndexManager() *MultiIndexManager
func (mim *MultiIndexManager) RegisterIndex(indexType IndexType, index Index)
func (mim *MultiIndexManager) GetIndex(indexType IndexType) Index
func (mim *MultiIndexManager) AddBlock(ctx context.Context, block Block) error
// ... Search, RemoveBlock, RebuildAll
```

Dedicated HNSW implementation with debug utilities in `debug_hnsw.go`.

### What TS has

HNSW exists embedded in `storage/storage.ts` as `HNSWIndex` (coupled to storage layer). No standalone `components/indexing/` module, no `Block` interface, no `MultiIndexManager`.

### Work needed

- Create `ts/src/components/indexing/index.ts` with:
  - `Block` interface (using string IDs, compatible with TS ecosystem)
  - `Index` interface with `add`, `search`, `remove`, `rebuild`
  - `IndexType` enum: `Semantic`, `Temporal`, `Category`, `FullText`, `Tag`
  - `MultiIndexManager` class
- Move/refactor `HNSWIndex` out of `storage.ts` into `components/indexing/hnsw.ts`
- Export from `lib/index.ts`

---

## 5. Schema Mismatch: `FrameEntry`

**Severity: High**

### Go `FrameEntry` (`pkg/nrv/frame.go`)

```go
type FrameEntry struct {
    ID            string
    TimestampUnix int64
    Tombstone     *int64
    Linguistic    LinguisticMapping
    Thermo        ThermoAtmosphere
    Z3            Z3Result
    Brackets      BracketBinaryMap
    BracketIndex  []BracketMeta
}
```

### TS `FrameEntry` (`components/storage/nrv/codec.ts`)

```ts
interface FrameEntry {
  id: string;
  offset: number;       // file byte offset — not in Go
  length: number;       // payload length — not in Go
  tombstone?: number;   // Go: *int64; TS: number
  verified: boolean;    // not in Go FrameEntry
  ergoRank: number;     // not in Go FrameEntry
  modalities: ModalityMap; // not in Go FrameEntry
}
```

These are structurally divergent types serving different roles. The Go `FrameEntry` is the **registry-level semantic entry** (linguistic, thermo, Z3, bracket index). The TS `FrameEntry` is a **file-level index entry** (byte offset, length, modalities). Both are valid representations but they are not interchangeable and the TS type is missing the semantic fields.

### Work needed

- Rename TS `FrameEntry` to `FrameIndexEntry` (or `FrameFileEntry`) to reflect its role
- Add a new `FrameEntry` interface in TS that mirrors Go's semantic fields:
  ```ts
  interface FrameEntry {
    id: string;
    timestampUnix: number;
    tombstone?: number;
    linguistic: LinguisticMapping;
    thermo: ThermoAtmosphere;
    z3: Z3Result;
    brackets: BracketBinaryMap;
    bracketIndex: BracketMeta[];
  }
  ```
- Update all references in `nrv_storage.ts`, `reader.ts`, `writer.ts`, `nrv_storage.ts`

---

## 6. Schema Mismatch: `GlobalMetrics`

**Severity: Medium**

### Go `GlobalMetrics` (`pkg/nrv/frame.go`)

```go
type GlobalMetrics struct {
    AvgTempCMean      float32
    AvgTempCMax       float32
    PeakVoltVMean     float32
    ClockMHzMean      float32
    TotalBracketCount int
    ValidFrameCount   int
    InvalidFrameCount int
    CompactedAt       *string
}
```

### TS `GlobalMetrics` (`components/storage/nrv/codec.ts`)

```ts
interface GlobalMetrics {
  featureMin: Float32Array;        // not in Go
  featureMax: Float32Array;        // not in Go
  featureMean: Float32Array;       // not in Go
  featureStd: Float32Array;        // not in Go
  thermoCorrelationCoefficient: number; // not in Go
  ergoRankSum: number;             // not in Go
  verifiedFrameCount: number;
  compactedAt?: string;
}
```

The TS `GlobalMetrics` tracks ML feature statistics rather than thermal/bracket counts. These do not match and a binary NRV file written by Go cannot be deserialized correctly in TS and vice versa.

### Work needed

- Replace TS `GlobalMetrics` fields to match Go:
  ```ts
  interface GlobalMetrics {
    avgTempCMean: number;
    avgTempCMax: number;
    peakVoltVMean: number;
    clockMHzMean: number;
    totalBracketCount: number;
    validFrameCount: number;
    invalidFrameCount: number;
    compactedAt?: string;
  }
  ```
- Update `writer.ts`, `compactor.ts`, and any code computing these metrics

---

## 7. Missing: NRV Bracket Types

**Severity: High**

### What Go has (`pkg/nrv/bracket.go`)

```go
type DeltaType string
const (DeltaTypeI DeltaType = "I"; DeltaTypeP DeltaType = "P")
const BracketSize = 80

type Bracket struct {
    ID          string
    LSHSalt     uint32
    Projections [64]byte
    SubSecondUS uint32
    ASICLoops   uint32
    GoldenSeed  uint32
    Meta        *BracketMeta
}

type BracketMeta struct {
    ID         string
    Type       DeltaType
    AnchorID   *string
    Offset     int
    DriftScore float64
}

type LinguisticMapping struct { Token string; Unit string }
type ThermoAtmosphere  struct { AvgTempC float32; PeakVoltV float32; ClockMHz float32 }
type Z3Result          struct { Status string; Relevance float64 }
type BracketBinaryMap  struct { Count int; Offset int64; Length int }
```

### What TS has

None of these types are defined or exported anywhere in the TS SDK. `nrv_storage.ts` uses `any` for bracket parameters.

### Work needed

- Create `ts/src/components/storage/nrv/bracket.ts` with:
  ```ts
  export type DeltaType = 'I' | 'P';
  export const BracketSize = 80;
  export interface Bracket { id: string; lshSalt: number; projections: Uint8Array; subSecondUS: number; asicLoops: number; goldenSeed: number; meta?: BracketMeta; }
  export interface BracketMeta { id: string; type: DeltaType; anchorId?: string; offset: number; driftScore: number; }
  export interface LinguisticMapping { token: string; unit: string; }
  export interface ThermoAtmosphere { avgTempC: number; peakVoltV: number; clockMHz: number; }
  export interface Z3Result { status: string; relevance: number; }
  export interface BracketBinaryMap { count: number; offset: number; length: number; }
  ```
- Replace all `any` usage in `nrv_storage.ts` with these types
- Export from `lib/index.ts`

---

## 8. Missing: NRV Codec Bracket-Level Functions

**Severity: Medium**

### What Go has (`pkg/nrv/codec.go`)

```go
func EncodeBracket(b *Bracket) [80]byte
func DecodeBracket(buf [80]byte) Bracket
func XORProjections(current, anchor [64]byte) [64]byte     // compute projection delta
func ApplyProjectionDelta(delta, anchor [64]byte) [64]byte  // apply projection delta
```

### What TS has

`codec.ts` only has `encodeFrame`/`decodeFrame`. No bracket-level binary codec, no projection delta functions.

### Work needed

- Add to `ts/src/components/storage/nrv/codec.ts`:
  ```ts
  export function encodeBracket(b: Bracket): Uint8Array  // 80 bytes
  export function decodeBracket(buf: Uint8Array): Bracket
  export function xorProjections(current: Uint8Array, anchor: Uint8Array): Uint8Array  // 64 bytes
  export function applyProjectionDelta(delta: Uint8Array, anchor: Uint8Array): Uint8Array
  ```

---

## 9. Signature Mismatch: `NRVStorage.appendBracket` Missing Thermo Parameter

**Severity: Medium**

### Go signature

```go
func (s *NRVStorage) AppendBracketDirect(collection string, b *nrv.Bracket, thermo nrv.ThermoAtmosphere) error
```

### TS signature (`nrv_storage.ts`)

```ts
async appendBracket(frameId: string, bracket: any): Promise<void>
```

The `thermo ThermoAtmosphere` parameter is missing. The Go ticker (`FrameTicker`) accumulates thermo samples per frame and passes the aggregate atmosphere at flush time. Without this parameter the TS SDK cannot compute `GlobalMetrics` thermal fields correctly.

### Work needed

- Update TS signature to:
  ```ts
  async appendBracket(collection: string, bracket: Bracket, thermo: ThermoAtmosphere): Promise<void>
  ```
- Propagate thermo data into the frame ticker and metrics computation

---

## 10. Missing: `QueryPlan.SortOrder`

**Severity: Low**

### Go `QueryPlan` (`internal/query/optimizer.go`)

```go
type QueryPlan struct {
    // ...
    SortOrder SortOrder  // { Field string; Ascending bool }
    // ...
}
```

### TS `QueryPlan` (`components/query/optimizer.ts`)

No `sortOrder` field.

### Work needed

- Add to `optimizer.ts`:
  ```ts
  export interface SortOrder { field: string; ascending: boolean; }
  // Add to QueryPlan:
  sortOrder?: SortOrder;
  ```
- Wire into query execution logic

---

## 11. Missing: `generateHybridToken`

**Severity: Low**

### What Go has (`internal/auth/auth.go`)

```go
func (tm *TokenManager) GenerateHybridToken(userID, walletAddr, sessionToken string, permissions []Permission) (string, error)
```

Embeds a session token (from an external auth provider) alongside the JWT claims — enables hybrid auth where a blockchain wallet session is validated alongside a standard JWT.

### What TS has

`refreshToken(oldToken: string): string` — rotates an existing JWT, different semantic.

### Work needed

- Add to `TokenManager` in `ts/src/components/auth/token_manager.ts`:
  ```ts
  generateHybridToken(userId: string, walletAddr: string, sessionToken: string, permissions: Permission[]): string
  ```

---

## Parity Summary Table

| Feature | Go | TypeScript | Status |
|---|---|---|---|
| `NewNRV` factory | `NewNRV(ctx, opts, keyPair)` | Missing | ❌ Not implemented |
| `DB.dataset()` + `NRVDataset` | Full API | Missing | ❌ Not implemented |
| Distributed tracing | `InitTracer`, `StartSpan` | Missing | ❌ Not implemented |
| Arrow Flight server/client | `FlightServer`, `FlightClient` | Missing | ❌ Not implemented |
| `BracketsToFlightData` / `FlightDataToBrackets` | Present | Missing | ❌ Not implemented |
| Standalone indexing module | `MultiIndexManager`, `Block`, `Index` interfaces | Missing | ❌ Not implemented |
| `Bracket` / `BracketMeta` types | `pkg/nrv/bracket.go` | Missing | ❌ Not implemented |
| `ThermoAtmosphere` type | Present | Missing | ❌ Not implemented |
| `LinguisticMapping` type | Present | Missing | ❌ Not implemented |
| `Z3Result` type | Present | Missing | ❌ Not implemented |
| `BracketBinaryMap` type | Present | Missing | ❌ Not implemented |
| `DeltaType` enum | `"I"` / `"P"` | Missing | ❌ Not implemented |
| `EncodeBracket` / `DecodeBracket` | Present | Missing | ❌ Not implemented |
| `XORProjections` / `ApplyProjectionDelta` | Present | Missing | ❌ Not implemented |
| `FrameEntry` semantic fields | Linguistic, Thermo, Z3, BracketIndex | File-offset schema only | ⚠️ Schema mismatch |
| `GlobalMetrics` fields | Thermal + bracket counts | ML feature stats | ⚠️ Schema mismatch |
| `appendBracket` thermo param | 3rd param: `ThermoAtmosphere` | 2 params, no thermo | ⚠️ Signature mismatch |
| `QueryPlan.SortOrder` | Present | Missing | ⚠️ Partial |
| `generateHybridToken` | Present | Missing (`refreshToken` ≠ equivalent) | ⚠️ Partial |
| Core CRUD + Collection | Full | Full | ✅ Parity |
| Vector clock | Full | Full | ✅ Parity |
| CRDT resolver | Full | Full | ✅ Parity |
| Network manager | Full | Full | ✅ Parity |
| PQC crypto (ML-KEM-768 + ML-DSA-65) | Full | Full | ✅ Parity |
| JWT auth | Full | Full | ✅ Parity |
| TF-IDF + LSA embeddings | Full | Full | ✅ Parity |
| KNIRVQL parser | Full | Full | ✅ Parity |
| Query optimizer (core) | Full | Full | ✅ Parity |
| FileStorage + NRV storage | Full | Full | ✅ Parity |
| WAL | Full | Full | ✅ Parity |
| Security (AES-GCM, PBKDF2) | Full | Full | ✅ Parity |
| Monitoring (Prometheus metrics) | Full | Full | ✅ Parity |
| Logging | Full | Full | ✅ Parity |
| DHT | Full | Full | ✅ Parity |
| Index types (BTree, GIN, HNSW, Tag) | Full | Full | ✅ Parity |

---

## Recommended Implementation Order

1. **NRV Bracket types** (`bracket.ts`) — unblocks everything else NRV-related
2. **`FrameEntry` + `GlobalMetrics` schema fix** — required for binary compatibility with Go
3. **`appendBracket` thermo signature fix** — required for correct metrics
4. **`EncodeBracket`/`DecodeBracket` + projection delta functions** — codec parity
5. **`NewNRV` + `DB.dataset()` + `NRVDataset`** — exposes NRV API at top-level
6. **Arrow Flight server** — high-throughput bracket streaming
7. **Standalone indexing module** — semantic/temporal/category/fulltext/tag
8. **Distributed tracing** — observability
9. **`QueryPlan.SortOrder`** — optimizer completeness
10. **`generateHybridToken`** — auth completeness
