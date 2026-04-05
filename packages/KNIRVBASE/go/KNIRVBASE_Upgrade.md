# KNIRVBASE Upgrade: `.nrv` (Noted Resolution Vector) Format

**Status:** Implemented — Phases 1-6 complete. Core storage engine, public API, KNIRVQL extensions, and tombstone system are functional.
**Module:** `github.com/knirvcorp/knirvbase/go` (Go 1.24.6)
**Root:** `packages/KNIRVBASE/go/` (all paths below are relative to this root)

---

## 1. Purpose & Scope

This plan transitions KNIRVBASE's storage layer from per-document `.json` files to a unified binary dataset container called **`.nrv` (Noted Resolution Vector)**. The `.nrv` format is designed for:

- **Zero-copy streaming** of KNIRVHASHER training frames via `mmap`
- **Append-only integrity** with Dilithium-3 PQC signatures per frame
- **CRDT-compatible versioning** via vector clock deltas for incremental P2P sync
- **Hardware telemetry correlation**: pairing SHA-256 ASIC thermodynamics (from Antminer S3) with formal Z3-verified seeds

This upgrade does **not** remove existing KNIRVBASE infrastructure. It adds a parallel storage path (`NRVStorage`) alongside `FileStorage`. The existing `Storage` interface, `DistributedDatabase`, KNIRVQL parser, CRDT resolver, PQC crypto, and network layers are **reused, not replaced**.

---

## 2. Current State Inventory

### 2.1 What Exists (Do Not Rewrite)

| File | Package | Role |
|---|---|---|
| `internal/storage/storage.go` | `storage` | `FileStorage` — per-doc `.json` + blob refs, Kyber/Dilithium signing |
| `internal/storage/index.go` | `storage` | `IndexManager` — B-Tree, HNSW, Tag indexes |
| `internal/types/types.go` | `types` | `DistributedDocument`, `CRDTOperation`, `MemoryCategory`, `NetworkConfig` |
| `internal/crypto/pqc/keys.go` | `pqc` | `PQCKeyPair` wrapping Kyber-768 + Dilithium-3 |
| `internal/crypto/pqc/dilithium.go` | `pqc` | `DilithiumSign`, `DilithiumVerify` via `circl/sign/dilithium/mode3` |
| `internal/crypto/pqc/kyber.go` | `pqc` | `KyberEncrypt`, `KyberDecrypt` via `circl/kem/kyber/kyber768` |
| `internal/crypto/pqc/encryption.go` | `pqc` | `EncryptionManager` with field-level Kyber encryption |
| `internal/clock/vector_clock.go` | `clock` | `VectorClock`, `Compare`, `Merge`, `Increment` |
| `internal/resolver/crdt_resolver.go` | `resolver` | LWW + vector-clock CRDT conflict resolution (`ResolveConflict`) |
| `internal/collection/distributed_collection.go` | `collection` | `DistributedCollection`, `LocalCollection` |
| `internal/database/distributed_database.go` | `distributed` | `DistributedDatabase`, `NewDistributedDatabase` |
| `internal/query/knirvql.go` | `query` | KNIRVQL parser: GET/SET/DELETE/CREATE INDEX/DROP |
| `internal/query/optimizer.go` | `query` | `QueryOptimizer`, `QueryPlan` |
| `internal/embedding/` | `embedding` | TF-IDF, LSA, `Embedder` |
| `internal/indexing/hnsw.go` | `indexing` | HNSW vector index |
| `internal/network/network_manager.go` | `network` | P2P `NetworkManager` |
| `internal/auth/auth.go` | `auth` | JWT-based auth |
| `internal/monitoring/monitoring.go` | `monitoring` | Prometheus metrics |
| `internal/tracing/tracing.go` | `tracing` | OpenTelemetry/Jaeger |
| `pkg/knirvbase/knirvbase.go` | `knirvbase` | Public API: `DB`, `Collection`, `New` |

### 2.2 What Must Be Created (This Upgrade)

| File | Package | Role |
|---|---|---|
| `pkg/nrv/spec.go` | `nrv` | Magic bytes, alignment constants, modality type enum |
| `pkg/nrv/frame.go` | `nrv` | `FrameEntry`, `GlobalMetrics`, `Registry` structs (JSON-serializable) |
| `pkg/nrv/codec.go` | `nrv` | `EncodeHeader`, `DecodeHeader` — binary header encode/decode |
| `internal/storage/nrv_writer.go` | `storage` | `NRVWriter` — flock, atomic append, WAL |
| `internal/storage/nrv_reader.go` | `storage` | `NRVReader` — `mmap` random access, modality filtering |
| `internal/storage/nrv_compactor.go` | `storage` | `Compactor` — background rewrite-and-swap goroutine |
| `internal/storage/wal.go` | `storage` | `WAL` — write-ahead log for crash recovery |
| `internal/storage/nrv_storage.go` | `storage` | `NRVStorage` implementing the `Storage` interface |

### 2.3 What Must Be Modified (This Upgrade)

| File | Change |
|---|---|
| `internal/query/knirvql.go` | Add `GET MEMORY.MODALITY(type)` syntax, `QueryModality` type, `ModalityFilter` |
| `internal/collection/distributed_collection.go` | Add `StreamFrames(ctx, modalityType) (<-chan nrv.FrameEntry, error)` method |
| `pkg/knirvbase/knirvbase.go` | Add `Dataset(name string) *NRVDataset` public API |

---

## 3. `.nrv` Binary Format Specification

### 3.1 File Layout

```
[Header 12B] [Registry Chunk — JSON + padding] [Binary Buffer — 8B-aligned frames]
```

### 3.2 Header (12 bytes, fixed offset 0x00)

| Offset | Size | Field | Value |
|---|---|---|---|
| `0x00` | 4 B | Magic | `0x4E525621` = ASCII `NRV!` |
| `0x04` | 4 B | Version | `0x00000001` (little-endian uint32) |
| `0x08` | 4 B | TotalLength | uint32, full file size in bytes (little-endian) |

The header is **always the last field written** in any append or compaction. Readers validate the magic bytes before proceeding.

### 3.3 Registry Chunk (Chunk 0 — JSON, variable length)

Immediately follows the header at offset `0x0C`. Pre-allocated with **5 MB of whitespace padding** on first write to support in-place registry updates without shifting binary data.

The registry is a JSON object serialized as UTF-8:

```json
{
  "version": 1,
  "dataset_id": "<uuid>",
  "dataset_version": "<vector-clock-json>",
  "chunk0_length": 5242880,
  "frame_count": 1024,
  "tombstone_count": 12,
  "global_metrics": { ... },
  "frames": [ ... ],
  "pqc_manifest": { ... }
}
```

**`global_metrics` object:**

```json
{
  "feature_min": [float32 × 12],
  "feature_max": [float32 × 12],
  "feature_mean": [float32 × 12],
  "feature_std": [float32 × 12],
  "thermo_correlation_coefficient": float64,
  "ergo_rank_sum": float64,
  "verified_frame_count": int,
  "compacted_at": "RFC3339 timestamp or null"
}
```

**Each entry in `frames` array:**

```json
{
  "id": "<uuid>",
  "offset": 5242892,
  "length": 256,
  "tombstone": null,
  "verified": true,
  "ergo_rank": 0.87,
  "modalities": {
    "vector": { "offset": 0, "length": 48 },
    "seed":   { "offset": 48, "length": 32 },
    "thermo": { "offset": 80, "length": 16 },
    "proof":  { "offset": 96, "length": 160 }
  }
}
```

- `offset` is the **absolute byte offset** from the start of the file.
- `tombstone` is `null` when the frame is live; set to a Unix nanosecond timestamp (int64) upon deletion.
- `modalities` maps modality names to relative offsets within that frame's binary segment.

**`pqc_manifest` object:**

```json
{
  "key_id": "<PQCKeyPair.ID>",
  "algorithm": "Dilithium-3",
  "file_signature": "<base64-encoded Dilithium-3 signature of entire file excluding this field>",
  "frame_signatures": {
    "<frame-id>": "<base64-encoded Dilithium-3 signature of that frame's binary segment>"
  }
}
```

### 3.4 Binary Buffer (Chunk 1 — 8-byte-aligned frames)

Starts immediately after the 5 MB registry region. Each frame's binary data consists of its modalities laid out contiguously, each **8-byte aligned** (pad with zero bytes as needed).

**Modality layouts:**

| Modality | Size | Contents |
|---|---|---|
| `vector` | 48 B | 12 × float32 (little-endian) = feature vector from KNIRVHASHER |
| `seed` | 32 B | Raw 32-byte candidate SHA-256 seed from ASIC |
| `thermo` | 16 B | 4 × float32: CPU temp (°C), voltage (V), frequency (MHz), fan RPM |
| `proof` | variable (8B-aligned) | UTF-8 SMT-LIB2 trace from Z3; length stored in registry |

Total frame size = `align8(48 + 32 + 16 + len(proof))`. Minimum frame with empty proof = 96 bytes.

---

## 4. Implementation: Phase 1 — Public Spec (`pkg/nrv`)

**Create directory:** `pkg/nrv/`

### 4.1 `pkg/nrv/spec.go`

```go
package nrv

// Magic bytes and version
const (
    Magic   uint32 = 0x4E525621 // "NRV!"
    Version uint32 = 1
    Alignment       = 8                // all binary segments must be 8-byte aligned
    RegistryPadding = 5 * 1024 * 1024 // 5 MB pre-allocated for JSON registry
    HeaderSize      = 12
)

// ModalityType identifies a data modality within a frame
type ModalityType string

const (
    ModalityVector ModalityType = "vector"
    ModalitySeed   ModalityType = "seed"
    ModalityThermo ModalityType = "thermo"
    ModalityProof  ModalityType = "proof"
)

// Align8 returns n rounded up to the nearest 8-byte boundary
func Align8(n int) int {
    return (n + 7) &^ 7
}
```

### 4.2 `pkg/nrv/frame.go`

```go
package nrv

import "github.com/knirvcorp/knirvbase/go/internal/clock"

// ModalityIndex maps a modality type to its relative offset and length within a frame's binary segment
type ModalityIndex struct {
    Offset int `json:"offset"`
    Length int `json:"length"`
}

// FrameEntry is one entry in the Chunk 0 Registry
type FrameEntry struct {
    ID         string                     `json:"id"`
    Offset     int64                      `json:"offset"`  // absolute file offset
    Length     int                        `json:"length"`  // total binary length of this frame
    Tombstone  *int64                     `json:"tombstone"` // nil = live; Unix nanoseconds = deleted
    Verified   bool                       `json:"verified"`
    ERGORank   float64                    `json:"ergo_rank"`
    Modalities map[ModalityType]ModalityIndex `json:"modalities"`
}

// GlobalMetrics holds aggregate statistics for the entire dataset
type GlobalMetrics struct {
    FeatureMin                   [12]float32 `json:"feature_min"`
    FeatureMax                   [12]float32 `json:"feature_max"`
    FeatureMean                  [12]float32 `json:"feature_mean"`
    FeatureStd                   [12]float32 `json:"feature_std"`
    ThermoCorrelationCoefficient float64     `json:"thermo_correlation_coefficient"`
    ERGORankSum                  float64     `json:"ergo_rank_sum"`
    VerifiedFrameCount           int         `json:"verified_frame_count"`
    CompactedAt                  *string     `json:"compacted_at"` // RFC3339 or nil
}

// PQCManifest holds the dataset-level Dilithium-3 signature and per-frame signatures
type PQCManifest struct {
    KeyID           string            `json:"key_id"`
    Algorithm       string            `json:"algorithm"`
    FileSignature   string            `json:"file_signature"`   // base64
    FrameSignatures map[string]string `json:"frame_signatures"` // frameID -> base64
}

// Registry is the full Chunk 0 JSON object
type Registry struct {
    Version        int                    `json:"version"`
    DatasetID      string                 `json:"dataset_id"`
    DatasetVersion clock.VectorClock      `json:"dataset_version"`
    Chunk0Length   int                    `json:"chunk0_length"`
    FrameCount     int                    `json:"frame_count"`
    TombstoneCount int                    `json:"tombstone_count"`
    GlobalMetrics  GlobalMetrics          `json:"global_metrics"`
    Frames         []FrameEntry           `json:"frames"`
    PQCManifest    PQCManifest            `json:"pqc_manifest"`
}

// Frame holds the decoded binary data for a single frame (in-memory representation)
type Frame struct {
    ID      string
    Vector  [12]float32    // 48 bytes
    Seed    [32]byte       // 32 bytes
    Thermo  ThermoData     // 16 bytes
    Proof   []byte         // variable length UTF-8 SMT-LIB2 trace
}

// ThermoData contains hardware telemetry captured during ASIC hashing
type ThermoData struct {
    TempCelsius float32 // CPU temperature
    VoltageV    float32 // supply voltage
    FreqMHz     float32 // clock frequency
    FanRPM      float32 // fan speed
}
```

### 4.3 `pkg/nrv/codec.go`

```go
package nrv

import (
    "encoding/binary"
    "fmt"
    "io"
)

// Header is the decoded 12-byte file header
type Header struct {
    Magic       uint32
    Version     uint32
    TotalLength uint32
}

// EncodeHeader writes a 12-byte header to w (little-endian)
func EncodeHeader(w io.Writer, h Header) error {
    buf := make([]byte, HeaderSize)
    binary.LittleEndian.PutUint32(buf[0:4], h.Magic)
    binary.LittleEndian.PutUint32(buf[4:8], h.Version)
    binary.LittleEndian.PutUint32(buf[8:12], h.TotalLength)
    _, err := w.Write(buf)
    return err
}

// DecodeHeader reads a 12-byte header from r and validates the magic bytes
func DecodeHeader(r io.Reader) (Header, error) {
    buf := make([]byte, HeaderSize)
    if _, err := io.ReadFull(r, buf); err != nil {
        return Header{}, fmt.Errorf("nrv: read header: %w", err)
    }
    h := Header{
        Magic:       binary.LittleEndian.Uint32(buf[0:4]),
        Version:     binary.LittleEndian.Uint32(buf[4:8]),
        TotalLength: binary.LittleEndian.Uint32(buf[8:12]),
    }
    if h.Magic != Magic {
        return Header{}, fmt.Errorf("nrv: invalid magic bytes: got 0x%X, want 0x%X", h.Magic, Magic)
    }
    return h, nil
}

// EncodeFrame serializes a Frame to a byte slice with 8-byte alignment enforced
func EncodeFrame(f *Frame) ([]byte, ModalityMap) {
    proofAligned := Align8(len(f.Proof))
    total := 48 + 32 + 16 + proofAligned
    buf := make([]byte, total)

    // vector: 12 × float32 little-endian
    for i, v := range f.Vector {
        binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(v))
    }
    // seed: 32 raw bytes
    copy(buf[48:], f.Seed[:])
    // thermo: 4 × float32 little-endian
    binary.LittleEndian.PutUint32(buf[80:], math.Float32bits(f.Thermo.TempCelsius))
    binary.LittleEndian.PutUint32(buf[84:], math.Float32bits(f.Thermo.VoltageV))
    binary.LittleEndian.PutUint32(buf[88:], math.Float32bits(f.Thermo.FreqMHz))
    binary.LittleEndian.PutUint32(buf[92:], math.Float32bits(f.Thermo.FanRPM))
    // proof: variable UTF-8 (remaining bytes zero-padded by make)
    copy(buf[96:], f.Proof)

    modalities := ModalityMap{
        ModalityVector: ModalityIndex{Offset: 0, Length: 48},
        ModalitySeed:   ModalityIndex{Offset: 48, Length: 32},
        ModalityThermo: ModalityIndex{Offset: 80, Length: 16},
        ModalityProof:  ModalityIndex{Offset: 96, Length: len(f.Proof)},
    }
    return buf, modalities
}

// ModalityMap is a shorthand type used during encoding
type ModalityMap = map[ModalityType]ModalityIndex
```

**Note:** `EncodeFrame` requires `"math"` import for `math.Float32bits`. Add this import.

---

## 5. Implementation: Phase 2 — Storage Engine (`internal/storage`)

### 5.1 `internal/storage/wal.go` — Write-Ahead Log

The WAL is a sidecar file named `<dataset>.nrv.wal`. It records in-flight append operations so that crash recovery can truncate the `.nrv` file to the last signed length.

```go
package storage

import (
    "encoding/json"
    "os"
    "sync"
)

// WALEntry records a single in-flight append
type WALEntry struct {
    FrameID       string `json:"frame_id"`
    LastGoodLength int64  `json:"last_good_length"` // file length before this append
    Committed     bool   `json:"committed"`
}

// WAL manages the write-ahead log file
type WAL struct {
    path string
    mu   sync.Mutex
}

func NewWAL(path string) *WAL { return &WAL{path: path} }

// Begin writes a WALEntry for a frame about to be appended.
// Call Commit after the append and registry update are complete.
func (w *WAL) Begin(entry WALEntry) error { /* marshal + append to WAL file */ }

// Commit marks the entry as committed
func (w *WAL) Commit(frameID string) error { /* update committed=true */ }

// Recover reads the WAL and returns the last known-good file length
// (the minimum LastGoodLength among uncommitted entries). Returns -1 if WAL is clean.
func (w *WAL) Recover() (int64, error) { /* scan entries, find min uncommitted */ }

// Truncate deletes all committed entries from the WAL
func (w *WAL) Truncate() error { os.Remove(w.path); return nil }
```

### 5.2 `internal/storage/nrv_writer.go` — Append Writer

```go
package storage

import (
    "encoding/json"
    "os"
    "sync"
    "syscall"

    "github.com/knirvcorp/knirvbase/go/internal/crypto/pqc"
    "github.com/knirvcorp/knirvbase/go/pkg/nrv"
)

// NRVWriter handles atomic appends to a single .nrv dataset file
type NRVWriter struct {
    path     string
    keyPair  *pqc.PQCKeyPair // for Dilithium-3 frame signing
    wal      *WAL
    mu       sync.Mutex
    registry *nrv.Registry  // in-memory copy of Chunk 0
    file     *os.File
}

// NewNRVWriter opens or creates an .nrv file at path, recovering from WAL if needed
func NewNRVWriter(path string, keyPair *pqc.PQCKeyPair) (*NRVWriter, error)

// AppendFrame adds a frame to the dataset:
//  1. Acquire flock(LOCK_EX) on the file
//  2. WAL.Begin(lastGoodLength)
//  3. Encode frame binary via nrv.EncodeFrame
//  4. Dilithium-3 sign the frame bytes
//  5. Write binary to EOF
//  6. Update in-memory registry: add FrameEntry, update GlobalMetrics
//  7. Marshal registry JSON, write to Chunk 0 region (within 5MB padding)
//  8. Update 12-byte header TotalLength (seek to 0x00, overwrite)
//  9. WAL.Commit(frameID)
// 10. Release flock
func (w *NRVWriter) AppendFrame(frame *nrv.Frame, verified bool, ergoRank float64) error

// flockAcquire wraps syscall.Flock for exclusive lock
func flockAcquire(f *os.File) error {
    return syscall.Flock(int(f.Fd()), syscall.LOCK_EX)
}

// flockRelease wraps syscall.Flock for unlock
func flockRelease(f *os.File) error {
    return syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
}

// initFile creates a new .nrv file with header and empty registry in padded Chunk 0
func (w *NRVWriter) initFile() error

// Close flushes and closes the underlying file
func (w *NRVWriter) Close() error
```

**Crash recovery in `NewNRVWriter`:** After loading the file, call `wal.Recover()`. If it returns a length > -1, truncate the file to that length using `os.Truncate(path, length)`, then reload the registry.

### 5.3 `internal/storage/nrv_reader.go` — mmap Reader

```go
package storage

import (
    "syscall"

    "github.com/knirvcorp/knirvbase/go/pkg/nrv"
)

// NRVReader provides zero-copy mmap-based access to .nrv binary frames
type NRVReader struct {
    path     string
    data     []byte         // mmap region
    registry *nrv.Registry  // parsed from Chunk 0
}

// NewNRVReader maps the file into memory and parses the registry
func NewNRVReader(path string) (*NRVReader, error)

// GetFrame retrieves and decodes a single frame by ID.
// Returns nil, nil if the frame is tombstoned.
func (r *NRVReader) GetFrame(id string) (*nrv.Frame, error)

// GetModality returns raw bytes for a single modality from a frame.
// No allocation — returns a slice of the mmap region.
func (r *NRVReader) GetModality(frameID string, modality nrv.ModalityType) ([]byte, error)

// StreamFrames returns a channel that yields live (non-tombstoned) frames.
// Filters to modalityFilter if non-empty.
func (r *NRVReader) StreamFrames(modalityFilter nrv.ModalityType) <-chan *nrv.Frame

// VerifyFrame verifies the Dilithium-3 signature for a frame's binary data.
// Uses the public key from the PQCManifest's key_id (caller must supply key store).
func (r *NRVReader) VerifyFrame(id string, keyPair *pqc.PQCKeyPair) (bool, error)

// Close unmaps the memory region
func (r *NRVReader) Close() error {
    return syscall.Munmap(r.data)
}
```

**mmap implementation:** Use `syscall.Mmap(int(f.Fd()), 0, fileLen, syscall.PROT_READ, syscall.MAP_SHARED)`. The returned slice is the entire file. Access frame binary data at `data[frame.Offset : frame.Offset+int64(frame.Length)]`.

### 5.4 `internal/storage/nrv_compactor.go` — Background Compaction

```go
package storage

import (
    "os"
    "sync"
    "time"

    "github.com/knirvcorp/knirvbase/go/internal/crypto/pqc"
    "github.com/knirvcorp/knirvbase/go/pkg/nrv"
)

const compactionThreshold = 0.20 // 20% tombstone ratio triggers compaction

// Compactor monitors tombstone ratio and runs rewrite-and-swap as needed.
// Only one compaction per dataset can be active at a time.
type Compactor struct {
    datasetPath string
    keyPair     *pqc.PQCKeyPair
    once        sync.Once
    running     bool
    mu          sync.Mutex
    stopCh      chan struct{}
}

// NewCompactor creates a compactor for the given .nrv file
func NewCompactor(path string, keyPair *pqc.PQCKeyPair) *Compactor

// MaybeCompact checks the tombstone ratio and starts a background goroutine if the
// threshold is exceeded. Safe to call after every Delete operation.
func (c *Compactor) MaybeCompact(registry *nrv.Registry)

// Start begins the compaction loop, polling every 30 seconds
func (c *Compactor) Start()

// Stop shuts down the compaction loop
func (c *Compactor) Stop()

// compact performs the rewrite-and-swap:
//  1. Open original .nrv via NRVReader
//  2. Create temp file at path + ".tmp"
//  3. Initialize new NRVWriter on temp file
//  4. For each live (non-tombstoned) FrameEntry in registry.Frames:
//     a. Read binary via NRVReader.GetFrame
//     b. Append to new NRVWriter (preserving ergo_rank, verified status)
//  5. Rebuild GlobalMetrics from scratch on new registry
//  6. Sign the new file with Dilithium-3 (update PQCManifest.FileSignature)
//  7. Flush and close new NRVWriter
//  8. os.Rename(tmpPath, originalPath) — atomic swap
//  9. Delete WAL sidecar for original file
func (c *Compactor) compact() error
```

**Concurrency guarantee:** Compaction uses `sync.Once` scoped to one cycle. Reads from the original file (via mmap) continue uninterrupted during compaction. Any writes arriving during compaction are serialized by the `NRVWriter`'s `flock`.

### 5.5 `internal/storage/nrv_storage.go` — `NRVStorage` implementing `Storage`

`NRVStorage` implements the existing `Storage` interface from `storage.go`. Collections map 1:1 to `.nrv` files on disk. The collection name becomes the file name: `<baseDir>/<collection>.nrv`.

```go
package storage

import (
    "context"
    "encoding/json"
    "fmt"
    "sync"

    "github.com/knirvcorp/knirvbase/go/internal/crypto/pqc"
    "github.com/knirvcorp/knirvbase/go/pkg/nrv"
)

// NRVStorage implements Storage using .nrv dataset files
type NRVStorage struct {
    baseDir  string
    keyPair  *pqc.PQCKeyPair
    writers  map[string]*NRVWriter      // collection -> writer
    readers  map[string]*NRVReader      // collection -> reader
    compact  map[string]*Compactor
    mu       sync.RWMutex
}

func NewNRVStorage(baseDir string, keyPair *pqc.PQCKeyPair) *NRVStorage

// Insert maps the document's "payload" fields to nrv.Frame modalities.
// For KNIRVHASHER frames: payload must contain "vector" ([]float32 len 12),
// "seed" ([]byte len 32), "thermo" (ThermoData), "proof" (string).
// "verified" (bool) and "ergo_rank" (float64) are extracted from top-level doc fields.
func (s *NRVStorage) Insert(ctx context.Context, collection string, doc map[string]interface{}) error

// Find retrieves a frame by ID, returns it as map[string]interface{} with "payload" key
func (s *NRVStorage) Find(ctx context.Context, collection, id string) (map[string]interface{}, error)

// FindAll returns all live frames as documents (tombstoned frames are excluded)
func (s *NRVStorage) FindAll(ctx context.Context, collection string) ([]map[string]interface{}, error)

// Delete tombstones a frame by writing tombstone timestamp to registry (no binary removal)
func (s *NRVStorage) Delete(ctx context.Context, collection, id string) error

// Update tombstones the old frame and inserts a new one with the merged fields
func (s *NRVStorage) Update(ctx context.Context, collection, id string, update map[string]interface{}) error

// GetModality returns raw bytes for a specific modality from a specific frame.
// This is the primary access method for the KNIRVHASHER training loader.
func (s *NRVStorage) GetModality(ctx context.Context, collection, frameID string, mod nrv.ModalityType) ([]byte, error)

// StreamFrames returns a channel of live frames for the named collection.
// If modalityFilter is non-empty, only that modality's data is populated in the frame struct.
func (s *NRVStorage) StreamFrames(ctx context.Context, collection string, modalityFilter nrv.ModalityType) (<-chan *nrv.Frame, error)

// --- Storage interface passthrough (KV, index, markdown) ---
// Put/Get/DeleteKey/StoreObject/GetObject/ProjectToMarkdown delegate to an embedded
// FileStorage instance for KV operations (NRV format is collection-only).
// CreateIndex/DropIndex/GetIndex/GetIndexesForCollection/QueryIndex also delegate to FileStorage.

func (s *NRVStorage) Close() error // closes all writers, readers, stops compactors
```

**Document ↔ Frame mapping:** When `Insert` is called with a generic document (not a KNIRVHASHER frame), store the document JSON as the `proof` modality (UTF-8 bytes). Set `vector`, `seed`, and `thermo` to zero values. This maintains backward compatibility with non-ML collections using the `Storage` interface.

---

## 6. Implementation: Phase 3 — Deletion (Tombstone System)

Deletion is **logical only**. Physical removal happens only during compaction.

### 6.1 Delete Operation in `NRVStorage.Delete`

1. Acquire write lock on the collection's `NRVWriter`
2. Find `FrameEntry` in `registry.Frames` by ID
3. Set `entry.Tombstone = &nowNano` (Unix nanoseconds)
4. Increment `registry.TombstoneCount`
5. Marshal updated registry, write to Chunk 0 padding region
6. Release lock
7. Call `compactor.MaybeCompact(registry)` — triggers background compaction if ratio ≥ 20%

### 6.2 Reader Filtering

`NRVReader.StreamFrames` and `NRVStorage.FindAll` skip any `FrameEntry` where `Tombstone != nil`. This is enforced at the reader layer, not the caller.

### 6.3 Compaction Trigger

After any `Delete`, evaluate:

```go
ratio := float64(registry.TombstoneCount) / float64(registry.FrameCount)
if ratio >= compactionThreshold {
    go compactor.compact()
}
```

Use `sync.Once` reset after each compaction cycle completes to prevent concurrent compactions.

---

## 7. Implementation: Phase 4 — KNIRVQL Extensions

### 7.1 Modify `internal/query/knirvql.go`

Add a new query type and syntax:

```
GET MEMORY.MODALITY(vector) WHERE dataset = "KNIRVHASHER_alpha" AND verified = true
GET MEMORY.MODALITY(seed) WHERE ergo_rank > 0.8 LIMIT 100
```

**New `QueryType` constant:**

```go
QueryGetModality QueryType = iota // add after QueryDropCollection
```

**New `ModalityFilter` on `Query` struct:**

```go
type Query struct {
    // ... existing fields ...
    ModalityType nrv.ModalityType // set when Type == QueryGetModality
}
```

**Parser change in `parseGet`:** Detect `MEMORY.MODALITY(type)` pattern as the first token after `GET`. Extract the modality name from within parentheses. Set `q.Type = QueryGetModality` and `q.ModalityType = nrv.ModalityType(extractedName)`.

**Execution in `Query.Execute`:** When `Type == QueryGetModality`, call `nrvStorage.StreamFrames(ctx, collectionName, q.ModalityType)` and return the channel. Callers are responsible for consuming it.

### 7.2 Modify `internal/collection/distributed_collection.go`

Add method to `DistributedCollection`:

```go
// StreamFrames returns a channel of live frames from the underlying NRVStorage.
// Returns an error if the storage backend does not implement NRV streaming.
func (c *DistributedCollection) StreamFrames(ctx context.Context, modalityFilter nrv.ModalityType) (<-chan *nrv.Frame, error) {
    nrvStore, ok := c.store.(*NRVStorage) // type assertion
    if !ok {
        return nil, fmt.Errorf("collection %q: storage backend does not support NRV streaming", c.name)
    }
    return nrvStore.StreamFrames(ctx, c.name, modalityFilter)
}
```

---

## 8. Implementation: Phase 5 — Public API (`pkg/knirvbase`)

### 8.1 Modify `pkg/knirvbase/knirvbase.go`

Add `NRVDataset` type and `DB.Dataset` method:

```go
// NRVDataset wraps a collection backed by NRVStorage, providing KNIRVHASHER-specific APIs
type NRVDataset struct {
    name    string
    storage *storage.NRVStorage
    inner   *coll.DistributedCollection
}

// Dataset returns an NRVDataset for KNIRVHASHER training data access.
// Panics if the DB was not created with an NRVStorage backend.
func (d *DB) Dataset(name string) *NRVDataset {
    nrvStore, ok := d.store.(*storage.NRVStorage)
    if !ok {
        panic("DB was not created with NRVStorage — use NewNRV() constructor")
    }
    return &NRVDataset{
        name:    name,
        storage: nrvStore,
        inner:   d.db.Collection(name, d.store),
    }
}

// AppendFrame adds a KNIRVHASHER frame to the dataset
func (ds *NRVDataset) AppendFrame(ctx context.Context, frame *nrv.Frame, verified bool, ergoRank float64) error

// StreamFrames returns a channel of live frames for ML training consumption
func (ds *NRVDataset) StreamFrames(ctx context.Context, modalityFilter nrv.ModalityType) (<-chan *nrv.Frame, error)

// GetModality returns raw bytes for one modality from one frame (zero-copy from mmap)
func (ds *NRVDataset) GetModality(ctx context.Context, frameID string, mod nrv.ModalityType) ([]byte, error)
```

Add a `NewNRV` constructor:

```go
// NewNRV creates a DB backed by NRVStorage for dataset workloads
func NewNRV(ctx context.Context, opts Options, keyPair *pqc.PQCKeyPair) (*DB, error) {
    store := storage.NewNRVStorage(opts.DataDir, keyPair)
    dopts := db.DistributedDbOptions{}
    dopts.Distributed.Enabled = opts.DistributedEnabled
    dopts.Distributed.NetworkID = opts.DistributedNetworkID
    dopts.Distributed.BootstrapPeers = opts.DistributedBootstrapPeers
    inner, err := db.NewDistributedDatabase(ctx, dopts, store)
    if err != nil {
        return nil, err
    }
    return &DB{db: inner, store: store}, nil
}
```

---

## 9. Implementation: Phase 6 — Security & P2P Sync

### 9.1 Per-Frame Dilithium-3 Signing

In `NRVWriter.AppendFrame`, after encoding the frame binary:

```go
sig, err := keyPair.Sign(frameBinary)
if err != nil {
    return fmt.Errorf("nrv: sign frame: %w", err)
}
registry.PQCManifest.FrameSignatures[frame.ID] = base64.StdEncoding.EncodeToString(sig)
```

After each compaction, re-sign the entire file by hashing all binary bytes (excluding `PQCManifest.FileSignature`) and storing the result in `PQCManifest.FileSignature`.

### 9.2 Kyber-768 Encrypted Modalities (Optional)

For sensitive collections, the `proof` modality (which may contain raw Z3 traces or agent context) can be Kyber-encrypted before writing to binary:

```go
if collection in sensitiveCollections {
    proofBytes, _ = keyPair.Encrypt(frame.Proof)
    frame.Proof = proofBytes
}
```

Mark encrypted proofs in the registry: add `"proof_encrypted": true` to the `FrameEntry.Modalities` map entry (extend `ModalityIndex` with an `Encrypted bool` field).

### 9.3 Incremental P2P Sync via Vector Clock

The existing `internal/resolver/crdt_resolver.go` and `internal/clock/vector_clock.go` are reused without modification.

**Sync protocol for `.nrv` datasets:**

1. Peer A sends `registry.DatasetVersion` (a `clock.VectorClock`) in the `MsgSyncRequest` message
2. Peer B compares via `clock.Compare(local, remote)`
3. If Peer B's version is `clock.After`, it sends only the **binary tail** starting from the offset of the first frame Peer A doesn't have
4. Incremental transfer = binary bytes from `firstNewFrame.Offset` to `header.TotalLength`
5. Peer A appends received bytes, then merges the received registry entries (no duplicates by frame ID)
6. After merge, increment `registry.DatasetVersion` for Peer A's node ID via `clock.Increment`

This allows P2P sync to transfer only new frames without retransmitting the full file.

---

## 10. KNIRVHASHER Integration

### 10.1 Data Source

KNIRVHASHER (`packages/KNIRVHASHER/`) produces frames from Antminer S3 ASIC hashing operations:

- **Feature vector** (`vector` modality): 12 float32 values derived from SHA-256 intermediate state
- **Candidate seed** (`seed` modality): 32-byte raw seed submitted to ASIC
- **Thermodynamic telemetry** (`thermo` modality): CPU temp, voltage, frequency, fan RPM from `/sys/` or ASIC firmware
- **Z3 proof trace** (`proof` modality): SMT-LIB2 output from the formal verification gate

### 10.2 Ingestion Pipeline

KNIRVHASHER writes frames to KNIRVBASE via `NRVDataset.AppendFrame`. The pipeline for each frame is:

1. **Capture:** ASIC returns hash result → extract 12-dim feature vector + 32-byte seed
2. **Thermo-tag:** Read hardware telemetry from `/sys/class/hwmon/` or USB device stats
3. **Z3 gate (CPU-bound):** Run Z3 SMT solver on seed to verify formal properties; capture SMT-LIB2 trace
4. **Append:** Call `NRVDataset.AppendFrame(ctx, frame, verified=<z3result>, ergoRank=<score>)`

The Z3 gate is synchronous and CPU-bound. Run it in a dedicated goroutine pool (e.g., `runtime.NumCPU()` workers) to avoid blocking the ASIC polling loop. The ASIC USB interface targets ~10 Hz throughput (USB latency limit).

### 10.3 Training Loader

The HERO Model reads `.nrv` files via `NRVDataset.StreamFrames`:

```go
ds := db.Dataset("KNIRVHASHER_alpha")
frames, _ := ds.StreamFrames(ctx, nrv.ModalityVector)
for frame := range frames {
    if frame.Verified {
        // feed frame.Vector to HERO Model training loop
    }
}
```

The `ERGO` ranking system filters frames by `ergoRank` before streaming. High-rank frames (human-validated or Z3-proven) are prioritized. KNIRVQL query:

```knirvql
GET MEMORY.MODALITY(vector) WHERE dataset = "KNIRVHASHER_alpha" AND verified = true AND ergo_rank > 0.8 LIMIT 1000
```

### 10.4 Existing Training Data Migration

Current untracked files in `packages/KNIRVHASHER/`:
- `training_frames_with_seeds.arrow` — Apache Arrow format
- `training_frames_with_seeds.json` — JSON dump of same data

Write a one-time migration script at `packages/KNIRVHASHER/scripts/migrate_to_nrv.go` that:
1. Reads the `.json` file
2. Constructs `nrv.Frame` objects (set `Thermo` and `Proof` to zero/empty — telemetry was not captured)
3. Appends each frame via `NRVDataset.AppendFrame` with `verified=false, ergoRank=0.0`
4. Deletes the `.json` and `.arrow` files after successful migration

---

## 11. Project File Map (Final State)

```
pkg/nrv/
├── spec.go          # Magic, Version, Alignment, ModalityType constants
├── frame.go         # FrameEntry, Registry, GlobalMetrics, PQCManifest, Frame, ThermoData
└── codec.go         # EncodeHeader, DecodeHeader, EncodeFrame

internal/storage/
├── storage.go       # FileStorage (unchanged)
├── index.go         # IndexManager (unchanged)
├── wal.go           # WAL — write-ahead log
├── nrv_writer.go    # NRVWriter — flock + atomic append
├── nrv_reader.go    # NRVReader — mmap access
├── nrv_compactor.go # Compactor — background rewrite-and-swap
└── nrv_storage.go   # NRVStorage implementing Storage interface

internal/query/
└── knirvql.go       # Add QueryGetModality, ModalityType field, parser update

internal/collection/
└── distributed_collection.go  # Add StreamFrames() method

pkg/knirvbase/
└── knirvbase.go     # Add NRVDataset, NewNRV constructor, DB.Dataset method
```

---

## 12. Test Requirements

Each new file must have a corresponding `_test.go`. Critical test cases:

| Test | File | Assertion |
|---|---|---|
| Header encode/decode round-trip | `pkg/nrv/codec_test.go` | `DecodeHeader(EncodeHeader(h)) == h`; invalid magic returns error |
| Frame encode produces 8-byte-aligned output | `pkg/nrv/codec_test.go` | `len(EncodeFrame(f)) % 8 == 0` |
| WAL recover truncates file on crash | `internal/storage/wal_test.go` | Write partial frame, simulate crash, WAL.Recover returns correct truncation point |
| Append + read round-trip | `internal/storage/nrv_writer_test.go` | Append 3 frames, read each via NRVReader, values match |
| Tombstone excludes from FindAll | `internal/storage/nrv_storage_test.go` | Insert 5, delete 2, FindAll returns 3 |
| Compaction at 20% threshold | `internal/storage/nrv_compactor_test.go` | Insert 100, delete 25 → compaction triggered; file contains 75 live frames |
| StreamFrames modality filter | `internal/storage/nrv_reader_test.go` | Filter by `nrv.ModalityVector` returns only vector bytes per frame |
| Dilithium-3 frame signature verifies | `internal/storage/nrv_writer_test.go` | Signed frame verifies with same key; tampered bytes fail verification |
| KNIRVQL modality query parses | `internal/query/knirvql_test.go` | `GET MEMORY.MODALITY(vector)` → `QueryGetModality`, `ModalityType = "vector"` |
| P2P incremental sync | `internal/collection/distributed_collection_test.go` | Two peers, first appends 10 frames, second syncs and receives only frames it's missing |

Run all tests:

```bash
cd packages/KNIRVBASE/go && go test -v ./...
```

---

## 13. Deferred (V2 Backlog)

Do not implement in V1:

- **Reasoning Ledger:** Immutable per-frame agent access log stored in a `ledger` modality
- **Cross-dataset KNIRVQL joins:** Multi-file query execution (`JOIN MEMORY WHERE ...`)
- **GPU-native loader:** Rust/Vulkan kernel for direct `.nrv`-to-VRAM DMA transfer
- **Partial sync:** Selective frame sync based on modality type (currently syncs full frame binary tail)
- **Compression:** LZ4/Zstd compression for `proof` modality before write
