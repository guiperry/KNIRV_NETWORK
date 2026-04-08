# KNIRVBASE Upgrade: `.nrv` Format — Phase 2 (ASIC-Native Spec)

**Status:** Phase 2 — Building on fully implemented Phase 1 foundation.
**Module:** `github.com/knirvcorp/knirvbase/go` (Go 1.24.6)
**Root:** `packages/KNIRVBASE/go/` (all paths below are relative to this root)

---

## 1. Purpose & Scope

This document describes Phase 2 of the `.nrv` upgrade. **Phase 1 is fully implemented** and operational. Phase 2 restructures the data model around the ASIC validation pipeline, redefines the Frame/Bracket hierarchy, adds a 1-second Ticker-driven flush cycle, introduces Arrow Flight streaming, and extends KNIRVQL with hardware-aware filters.

### 1.1 Terminology Change (Critical)

> **Phase 1 used the term "Frame" to mean a single data record.** That term has been reassigned. All Phase 1 "Frames" are now called **Brackets**.

| Term | Phase 1 Meaning | Phase 2 Meaning |
|---|---|---|
| **Frame** | Single multimodal data record | **1-second temporal snapshot** — the linguistic container holding N brackets and environmental metadata |
| **Bracket** | *(did not exist)* | **80-byte binary record** representing one completed ASIC task (LSH projections + Golden Seed) |

Every Go type, method, and KNIRVQL keyword that previously used "Frame" in the single-record sense must be migrated to "Bracket". The word "Frame" is reserved exclusively for the 1-second temporal unit going forward.

---

## 2. Phase 1 Baseline (Already Implemented — Do Not Rewrite)

The following files are complete and form the foundation for Phase 2. Phase 2 extends them; it does not replace them.

| File | Package | Status | Role |
|---|---|---|---|
| `pkg/nrv/spec.go` | `nrv` | ✅ Done | Magic bytes, alignment constants, modality type enum |
| `pkg/nrv/frame.go` | `nrv` | ✅ Done — **will be extended** | `FrameEntry`, `GlobalMetrics`, `Registry`, `Frame`, `ThermoData` |
| `pkg/nrv/codec.go` | `nrv` | ✅ Done — **will be extended** | Header encode/decode, `EncodeFrame` → becomes `EncodeBracket` |
| `internal/storage/nrv_writer.go` | `storage` | ✅ Done — **will be extended** | `NRVWriter`, `AppendFrame` → `AppendBracket`, flock, WAL |
| `internal/storage/nrv_reader.go` | `storage` | ✅ Done — **will be extended** | `NRVReader`, mmap, `GetFrame`, `StreamFrames` → `StreamBrackets` |
| `internal/storage/nrv_compactor.go` | `storage` | ✅ Done — **will be updated** | 20% tombstone threshold Rewrite-and-Swap |
| `internal/storage/wal.go` | `storage` | ✅ Done | WAL for crash recovery |
| `internal/storage/nrv_storage.go` | `storage` | ✅ Done — **will be updated** | `NRVStorage` implementing `Storage` interface |
| `internal/query/knirvql.go` | `query` | ✅ Done — **will be extended** | `GET MEMORY.MODALITY(type)` syntax |
| `pkg/knirvbase/knirvbase.go` | `knirvbase` | ✅ Done — **will be updated** | `NewNRV`, `NRVDataset`, `AppendFrame`, `StreamFrames`, `GetModality` |
| `internal/storage/storage.go` | `storage` | ✅ Done — unchanged | `FileStorage`, `Storage` interface |
| `internal/network/network_manager.go` | `network` | ✅ Done — **will be extended** | P2P `NetworkManager` |

### 2.1 New Files Required (Phase 2)

| File | Package | Role |
|---|---|---|
| `internal/storage/nrv_ticker.go` | `storage` | `FrameTicker` — 1-second buffer goroutine, flushes complete frames |
| `internal/network/flight_server.go` | `network` | Apache Arrow Flight producer, Gold/Research dual-stream |
| `pkg/nrv/bracket.go` | `nrv` | `Bracket`, `BracketMeta`, `DeltaType` — the 80-byte binary unit |

---

## 3. Updated `.nrv` Binary Format Specification

The file layout is unchanged at the structural level. What changes is the **semantic interpretation** of both chunks.

### 3.1 File Layout

```
[Header 12B] [Chunk 0: Resolution Registry — JSON + 5MB padding] [Chunk 1: Multi-Modal Buffer — 80-byte Brackets, 8B-aligned]
```

### 3.2 Global Header (12 bytes — unchanged)

| Offset | Size | Field | Value |
|---|---|---|---|
| `0x00` | 4 B | Magic | `0x4E525621` = ASCII `NRV!` |
| `0x04` | 4 B | Version | `0x00000001` (little-endian uint32) |
| `0x08` | 4 B | TotalLength | uint32, full file size in bytes (little-endian) |

Header is always written last. Readers validate magic before proceeding.

### 3.3 Chunk 0: Resolution Registry (JSON)

The registry now indexes **Frames** (1-second windows), not individual records. Each frame entry maps to a contiguous slice of 80-byte brackets in Chunk 1.

**Registry JSON schema:**

```json
{
  "version": 1,
  "dataset_id": "<uuid>",
  "dataset_version": "<vector-clock-json>",
  "chunk0_length": 5242880,
  "frame_count": 60,
  "tombstone_count": 3,
  "global_metrics": { ... },
  "frames": [ ... ],
  "pqc_manifest": { ... }
}
```

**Updated `global_metrics` object:**

```json
{
  "avg_temp_c_mean": 71.4,
  "avg_temp_c_max": 89.2,
  "peak_volt_v_mean": 1.35,
  "clock_mhz_mean": 550.0,
  "total_bracket_count": 18432,
  "valid_frame_count": 57,
  "invalid_frame_count": 3,
  "compacted_at": "2026-04-08T14:22:00Z"
}
```

**Each entry in `frames` array (one per 1-second window):**

```json
{
  "id": "<uuid>",
  "timestamp_unix": 1712580120,
  "tombstone": null,
  "linguistic": {
    "token": "resolution",
    "unit": "word"
  },
  "thermo": {
    "avg_temp_c": 71.4,
    "peak_volt_v": 1.37,
    "clock_mhz": 553.0
  },
  "z3": {
    "status": "VALID",
    "relevance": 0.92
  },
  "brackets": {
    "count": 307,
    "offset": 5242892,
    "length": 24560
  },
  "bracket_index": [
    { "id": "<uuid>", "type": "I", "anchor_id": null, "offset": 0,  "drift_score": 0.0 },
    { "id": "<uuid>", "type": "P", "anchor_id": "<I-bracket-uuid>", "offset": 80, "drift_score": 0.014 }
  ]
}
```

Field notes:
- `brackets.offset` — absolute byte offset in the file where this frame's bracket array starts.
- `brackets.length` — `count × 80` (before delta compression) or actual compressed byte length.
- `bracket_index` — per-bracket delta metadata: `type` is `I` (intra/anchor) or `P` (predicted/delta), `anchor_id` points to the I-bracket this P-bracket diffs against, `drift_score` is the Euclidean distance used to trigger new I-brackets.
- `tombstone` — `null` when live; Unix nanosecond int64 when deleted.
- `z3.status` — `VALID` or `INVALID`; controls Gold-stream eligibility.

**`pqc_manifest` — per-frame Dilithium-3 signatures (unchanged schema):**

```json
{
  "key_id": "<PQCKeyPair.ID>",
  "algorithm": "Dilithium-3",
  "file_signature": "<base64>",
  "frame_signatures": {
    "<frame-id>": "<base64 Dilithium-3 signature of this frame's bracket binary>"
  }
}
```

### 3.4 Chunk 1: Multi-Modal Buffer (80-byte Brackets)

Each bracket is exactly **80 bytes**, aligned to 8-byte boundaries. All brackets for a given frame are stored contiguously.

**Bracket binary layout:**

| Section | Offset | Size | Description |
|---|---|---|---|
| LSH Salt | `0x00` | 4 B | Version field used as the LSH forest seed (uint32 LE) |
| Projections A–D | `0x04` | 32 B | First half of 128-bit LSH projections (8 × float32 LE) |
| Projections E–H | `0x24` | 32 B | Second half of 128-bit LSH projections (8 × float32 LE) |
| Metadata | `0x44` | 8 B | Sub-second timestamp (uint32 LE, microseconds) + ASIC Loop Count (uint32 LE, range 1–21) |
| Golden Seed | `0x4C` | 4 B | Solved nonce (uint32 LE) — the result of the ASIC pass |

Total: 4 + 32 + 32 + 8 + 4 = **80 bytes**.

**Delta encoding (P-Brackets):** For P-brackets, bytes `0x04–0x43` (Projections A–H, 64 bytes) store the **XOR-diff** against the anchor I-bracket's projection bytes. LSH Salt, Metadata, and Golden Seed are always stored as absolute values regardless of bracket type.

---

## 4. Phase 2 Implementation

### 4.1 New Public Types (`pkg/nrv/bracket.go`)

Create `pkg/nrv/bracket.go`:

```go
package nrv

// DeltaType identifies whether a bracket is a full snapshot or a delta.
type DeltaType string

const (
    DeltaTypeI DeltaType = "I" // Intra — absolute values
    DeltaTypeP DeltaType = "P" // Predicted — XOR-delta against anchor
)

// BracketSize is the fixed size of every bracket in the binary buffer.
const BracketSize = 80

// Bracket is the in-memory representation of a single 80-byte ASIC record.
type Bracket struct {
    ID          string    // registry-only; not stored in binary
    LSHSalt     uint32    // bytes 0x00–0x03
    Projections [64]byte  // bytes 0x04–0x43 (absolute or XOR-diff for P-brackets)
    SubSecondUS uint32    // bytes 0x44–0x47: sub-second timestamp in microseconds
    ASICLoops   uint32    // bytes 0x48–0x4B: loop count (1–21)
    GoldenSeed  uint32    // bytes 0x4C–0x4F: solved nonce
}

// BracketMeta is the registry entry for one bracket within a frame.
type BracketMeta struct {
    ID         string    `json:"id"`
    Type       DeltaType `json:"type"`
    AnchorID   *string   `json:"anchor_id"` // nil for I-brackets
    Offset     int       `json:"offset"`    // byte offset relative to frame start in Chunk 1
    DriftScore float64   `json:"drift_score"`
}

// LinguisticMapping describes the linguistic unit this 1-second frame represents.
type LinguisticMapping struct {
    Token string `json:"token"`
    Unit  string `json:"unit"` // "syllable" or "word"
}

// ThermoAtmosphere is the hardware state snapshot for a 1-second frame.
type ThermoAtmosphere struct {
    AvgTempC  float32 `json:"avg_temp_c"`
    PeakVoltV float32 `json:"peak_volt_v"`
    ClockMHz  float32 `json:"clock_mhz"`
}

// Z3Result holds the formal verification outcome for a frame.
type Z3Result struct {
    Status    string  `json:"status"`    // "VALID" or "INVALID"
    Relevance float64 `json:"relevance"` // correlation score to adjacent frames
}

// BracketBinaryMap describes where this frame's bracket array lives in Chunk 1.
type BracketBinaryMap struct {
    Count  int   `json:"count"`
    Offset int64 `json:"offset"` // absolute file offset
    Length int   `json:"length"` // count × 80 (or actual compressed size)
}
```

### 4.2 Updates to `pkg/nrv/frame.go`

The existing `FrameEntry` struct must be replaced with a schema that models the 1-second frame. The old `Frame` struct (single-record in-memory type) becomes `Bracket`.

**Remove** the existing `Frame` struct (single-record type — now represented by `Bracket` in `bracket.go`).

**Replace** `FrameEntry` with:

```go
// FrameEntry is one entry in the Chunk 0 Registry — represents a 1-second window.
type FrameEntry struct {
    ID            string            `json:"id"`
    TimestampUnix int64             `json:"timestamp_unix"`
    Tombstone     *int64            `json:"tombstone"`
    Linguistic    LinguisticMapping `json:"linguistic"`
    Thermo        ThermoAtmosphere  `json:"thermo"`
    Z3            Z3Result          `json:"z3"`
    Brackets      BracketBinaryMap  `json:"brackets"`
    BracketIndex  []BracketMeta     `json:"bracket_index"`
}
```

**Replace** `GlobalMetrics` with:

```go
type GlobalMetrics struct {
    AvgTempCMean       float32  `json:"avg_temp_c_mean"`
    AvgTempCMax        float32  `json:"avg_temp_c_max"`
    PeakVoltVMean      float32  `json:"peak_volt_v_mean"`
    ClockMHzMean       float32  `json:"clock_mhz_mean"`
    TotalBracketCount  int      `json:"total_bracket_count"`
    ValidFrameCount    int      `json:"valid_frame_count"`
    InvalidFrameCount  int      `json:"invalid_frame_count"`
    CompactedAt        *string  `json:"compacted_at"`
}
```

**Keep** `PQCManifest`, `Registry`, and `ThermoData` (used elsewhere) — `ThermoData` can remain for legacy compatibility but is superseded by `ThermoAtmosphere` in frame registry entries.

### 4.3 Updates to `pkg/nrv/codec.go`

**Add** `EncodeBracket` and `DecodeBracket` alongside the existing `EncodeFrame`/`DecodeHeader`:

```go
// EncodeBracket serializes a Bracket to its 80-byte wire format.
func EncodeBracket(b *Bracket) [BracketSize]byte {
    var buf [BracketSize]byte
    binary.LittleEndian.PutUint32(buf[0:4], b.LSHSalt)
    copy(buf[4:68], b.Projections[:])
    binary.LittleEndian.PutUint32(buf[68:72], b.SubSecondUS)
    binary.LittleEndian.PutUint32(buf[72:76], b.ASICLoops)
    binary.LittleEndian.PutUint32(buf[76:80], b.GoldenSeed)
    return buf
}

// DecodeBracket parses an 80-byte buffer into a Bracket.
func DecodeBracket(buf [BracketSize]byte) Bracket {
    var b Bracket
    b.LSHSalt = binary.LittleEndian.Uint32(buf[0:4])
    copy(b.Projections[:], buf[4:68])
    b.SubSecondUS = binary.LittleEndian.Uint32(buf[68:72])
    b.ASICLoops = binary.LittleEndian.Uint32(buf[72:76])
    b.GoldenSeed = binary.LittleEndian.Uint32(buf[76:80])
    return b
}

// XORProjections returns the XOR-diff of `current` projections against `anchor` projections.
// Used to produce P-bracket payloads.
func XORProjections(current, anchor [64]byte) [64]byte {
    var diff [64]byte
    for i := range diff {
        diff[i] = current[i] ^ anchor[i]
    }
    return diff
}

// ApplyProjectionDelta reconstructs absolute projections from a P-bracket XOR-diff and its anchor.
func ApplyProjectionDelta(delta, anchor [64]byte) [64]byte {
    return XORProjections(delta, anchor) // XOR is its own inverse
}
```

**Deprecate** `EncodeFrame` (the old multimodal frame codec). It can remain for backward compatibility with existing data but should not be used for new writes.

### 4.4 New File: `internal/storage/nrv_ticker.go`

The `FrameTicker` is the central coordination component for Phase 2. It owns a 1-second window, buffers incoming brackets, and flushes a complete `FrameEntry` to the `NRVWriter` at each tick boundary.

```go
package storage

import (
    "context"
    "sync"
    "time"

    "github.com/google/uuid"
    "github.com/knirvcorp/knirvbase/go/pkg/nrv"
)

const (
    iFrameInterval = 50    // write an I-bracket every N brackets
    driftThreshold = 0.25  // Euclidean drift that forces a new I-bracket
)

// PendingBracket holds a bracket and its computed delta metadata before flush.
type PendingBracket struct {
    Bracket    *nrv.Bracket
    DeltaType  nrv.DeltaType
    AnchorID   *string
    DriftScore float64
}

// FrameTicker buffers incoming brackets and flushes 1-second frames to an NRVWriter.
type FrameTicker struct {
    writer    *NRVWriter
    interval  time.Duration
    mu        sync.Mutex
    pending   []PendingBracket
    lastIBkt  *nrv.Bracket // the most recent I-bracket, for XOR-delta calculation
    lastIID   string
    bktCount  int          // total brackets since last I-bracket

    // Frame-level metadata accumulated during the window
    thermoSamples []nrv.ThermoAtmosphere
    linguistic    nrv.LinguisticMapping

    ticker *time.Ticker
    stopCh chan struct{}
    wg     sync.WaitGroup
}

// NewFrameTicker constructs and starts a FrameTicker.
func NewFrameTicker(w *NRVWriter, interval time.Duration) *FrameTicker {
    ft := &FrameTicker{
        writer:   w,
        interval: interval,
        stopCh:   make(chan struct{}),
    }
    ft.ticker = time.NewTicker(interval)
    ft.wg.Add(1)
    go ft.run()
    return ft
}

// AppendBracket adds a bracket to the current 1-second window. Thread-safe.
func (ft *FrameTicker) AppendBracket(b *nrv.Bracket, thermo nrv.ThermoAtmosphere) {
    ft.mu.Lock()
    defer ft.mu.Unlock()

    var deltaType nrv.DeltaType
    var anchorID *string
    var driftScore float64

    // Determine I vs P bracket
    if ft.lastIBkt == nil || ft.bktCount%iFrameInterval == 0 {
        deltaType = nrv.DeltaTypeI
        ft.lastIBkt = b
        ft.lastIID = b.ID
    } else {
        driftScore = euclideanDrift(b.Projections, ft.lastIBkt.Projections)
        if driftScore > driftThreshold {
            // Drift spike: force a new I-bracket
            deltaType = nrv.DeltaTypeI
            ft.lastIBkt = b
            ft.lastIID = b.ID
        } else {
            deltaType = nrv.DeltaTypeP
            id := ft.lastIID
            anchorID = &id
            // XOR the projections for storage
            b.Projections = nrv.XORProjections(b.Projections, ft.lastIBkt.Projections)
        }
    }

    ft.pending = append(ft.pending, PendingBracket{
        Bracket:    b,
        DeltaType:  deltaType,
        AnchorID:   anchorID,
        DriftScore: driftScore,
    })
    ft.thermoSamples = append(ft.thermoSamples, thermo)
    ft.bktCount++
}

// SetLinguistic sets the linguistic context for the current frame window.
func (ft *FrameTicker) SetLinguistic(token, unit string) {
    ft.mu.Lock()
    defer ft.mu.Unlock()
    ft.linguistic = nrv.LinguisticMapping{Token: token, Unit: unit}
}

// Stop halts the ticker and flushes any remaining brackets as a final partial frame.
func (ft *FrameTicker) Stop() {
    close(ft.stopCh)
    ft.wg.Wait()
}

func (ft *FrameTicker) run() {
    defer ft.wg.Done()
    for {
        select {
        case <-ft.ticker.C:
            ft.flush()
        case <-ft.stopCh:
            ft.ticker.Stop()
            ft.flush()
            return
        }
    }
}

func (ft *FrameTicker) flush() {
    ft.mu.Lock()
    pending := ft.pending
    thermo := ft.thermoSamples
    ling := ft.linguistic
    ft.pending = nil
    ft.thermoSamples = nil
    ft.mu.Unlock()

    if len(pending) == 0 {
        return
    }

    frameID := uuid.New().String()
    atmosphere := aggregateThermo(thermo)
    bracketMetas := make([]nrv.BracketMeta, len(pending))
    for i, pb := range pending {
        bracketMetas[i] = nrv.BracketMeta{
            ID:         pb.Bracket.ID,
            Type:       pb.DeltaType,
            AnchorID:   pb.AnchorID,
            Offset:     i * nrv.BracketSize,
            DriftScore: pb.DriftScore,
        }
    }

    // Serialize bracket array
    buf := make([]byte, len(pending)*nrv.BracketSize)
    for i, pb := range pending {
        encoded := nrv.EncodeBracket(pb.Bracket)
        copy(buf[i*nrv.BracketSize:], encoded[:])
    }

    _ = ft.writer.AppendFrame(frameID, buf, bracketMetas, atmosphere, ling)
}

// euclideanDrift computes the Euclidean distance between two 64-byte projection arrays
// interpreted as 16 × float32 vectors.
func euclideanDrift(a, b [64]byte) float64 { /* ... */ return 0 }

// aggregateThermo reduces a slice of per-bracket samples to a per-frame summary.
func aggregateThermo(samples []nrv.ThermoAtmosphere) nrv.ThermoAtmosphere { /* ... */ return nrv.ThermoAtmosphere{} }
```

### 4.5 Updates to `internal/storage/nrv_writer.go`

**Rename `AppendFrame(frame *nrv.Frame, verified bool, ergoRank float64)`** → **`AppendFrame(frameID string, bracketBuf []byte, bracketIndex []nrv.BracketMeta, thermo nrv.ThermoAtmosphere, ling nrv.LinguisticMapping)`**.

The new signature reflects that the writer now receives a pre-serialized bracket buffer (produced by `FrameTicker.flush()`) alongside the frame-level metadata. Z3 validation runs inside the writer after the data is committed:

```go
func (w *NRVWriter) AppendFrame(
    frameID string,
    bracketBuf []byte,
    bracketIndex []nrv.BracketMeta,
    thermo nrv.ThermoAtmosphere,
    ling nrv.LinguisticMapping,
) error {
    w.mu.Lock()
    defer w.mu.Unlock()

    if err := flockAcquire(w.file); err != nil {
        return err
    }
    defer flockRelease(w.file)

    // WAL: record pre-write file length for crash recovery
    info, err := w.file.Stat()
    if err != nil {
        return err
    }
    if err := w.wal.Begin(WALEntry{FrameID: frameID, LastGoodLength: info.Size()}); err != nil {
        return err
    }

    // Sign the bracket buffer with Dilithium-3
    var sig []byte
    if w.keyPair != nil {
        sig, err = w.keyPair.Sign(bracketBuf)
        if err != nil {
            return fmt.Errorf("nrv: sign brackets: %w", err)
        }
    }

    // Append bracket buffer to EOF (Chunk 1)
    offset := info.Size()
    if _, err := w.file.Seek(0, 2); err != nil {
        return err
    }
    if _, err := w.file.Write(bracketBuf); err != nil {
        return err
    }

    // Run Z3 validation (placeholder — integrate actual Z3 gate here)
    z3 := nrv.Z3Result{Status: "VALID", Relevance: 1.0}

    // Build FrameEntry for the registry
    entry := nrv.FrameEntry{
        ID:            frameID,
        TimestampUnix: time.Now().Unix(),
        Tombstone:     nil,
        Linguistic:    ling,
        Thermo:        thermo,
        Z3:            z3,
        Brackets: nrv.BracketBinaryMap{
            Count:  len(bracketIndex),
            Offset: offset,
            Length: len(bracketBuf),
        },
        BracketIndex: bracketIndex,
    }

    w.registry.Frames = append(w.registry.Frames, entry)
    w.registry.FrameCount++
    if z3.Status == "VALID" {
        w.registry.GlobalMetrics.ValidFrameCount++
    } else {
        w.registry.GlobalMetrics.InvalidFrameCount++
    }
    w.registry.GlobalMetrics.TotalBracketCount += len(bracketIndex)

    if sig != nil {
        w.registry.PQCManifest.FrameSignatures[frameID] = base64.StdEncoding.EncodeToString(sig)
    }

    if err := w.saveRegistry(); err != nil {
        return err
    }

    return w.wal.Commit(frameID)
}
```

**Also add `AppendBracketDirect`** for the NRVStorage.Insert path used by legacy callers that submit one bracket at a time (routes through the ticker buffer instead of writing directly):

```go
// AppendBracketDirect enqueues a bracket to the FrameTicker for deferred flush.
// This is the preferred path for ASIC pipeline integration.
func (s *NRVStorage) AppendBracketDirect(collection string, b *nrv.Bracket, thermo nrv.ThermoAtmosphere) error {
    ticker, err := s.getOrCreateTicker(collection)
    if err != nil {
        return err
    }
    ticker.AppendBracket(b, thermo)
    return nil
}
```

### 4.6 Updates to `internal/storage/nrv_reader.go`

**Rename `StreamFrames` → `StreamBrackets`**. The new method decodes the bracket buffer for each live, valid frame and emits individual `*nrv.Bracket` values:

```go
// StreamBrackets emits decoded brackets from all live frames. Tombstoned and
// INVALID frames are skipped unless goldOnly is false.
func (r *NRVReader) StreamBrackets(goldOnly bool) <-chan *nrv.Bracket {
    ch := make(chan *nrv.Bracket, 256)
    go func() {
        defer close(ch)
        for _, entry := range r.registry.Frames {
            if entry.Tombstone != nil {
                continue
            }
            if goldOnly && entry.Z3.Status != "VALID" {
                continue
            }
            brackets := r.decodeBrackets(entry)
            for _, b := range brackets {
                ch <- b
            }
        }
    }()
    return ch
}

// GetFrame returns the FrameEntry and its decoded brackets by frame ID.
func (r *NRVReader) GetFrame(id string) (*nrv.FrameEntry, []*nrv.Bracket, error) {
    for _, entry := range r.registry.Frames {
        if entry.ID == id {
            if entry.Tombstone != nil {
                return nil, nil, nil
            }
            brackets := r.decodeBrackets(entry)
            return &entry, brackets, nil
        }
    }
    return nil, nil, fmt.Errorf("nrv: frame not found: %s", id)
}

// decodeBrackets reads the bracket buffer for a frame and reconstructs absolute values
// for P-brackets using their anchor I-bracket.
func (r *NRVReader) decodeBrackets(entry nrv.FrameEntry) []*nrv.Bracket {
    buf := r.data[entry.Brackets.Offset : entry.Brackets.Offset+int64(entry.Brackets.Length)]
    anchors := make(map[string]*nrv.Bracket)
    brackets := make([]*nrv.Bracket, len(entry.BracketIndex))

    for i, meta := range entry.BracketIndex {
        var raw [nrv.BracketSize]byte
        copy(raw[:], buf[meta.Offset:meta.Offset+nrv.BracketSize])
        b := nrv.DecodeBracket(raw)
        b.ID = meta.ID

        if meta.Type == nrv.DeltaTypeP && meta.AnchorID != nil {
            if anchor, ok := anchors[*meta.AnchorID]; ok {
                b.Projections = nrv.ApplyProjectionDelta(b.Projections, anchor.Projections)
            }
        }

        anchors[meta.ID] = &b
        brackets[i] = &b
    }
    return brackets
}
```

**Update `VerifyFrame`** to sign/verify the bracket buffer (not the old multimodal blob).

### 4.7 Updates to `internal/storage/nrv_compactor.go`

The compaction trigger must now include `INVALID` frames in the ratio, not just tombstoned entries:

```go
func (c *Compactor) MaybeCompact(registry *nrv.Registry) {
    c.mu.Lock()
    defer c.mu.Unlock()

    if c.running || registry.FrameCount == 0 {
        return
    }

    deadFrames := registry.TombstoneCount + registry.GlobalMetrics.InvalidFrameCount
    ratio := float64(deadFrames) / float64(registry.FrameCount)
    if ratio >= compactionThreshold {
        c.running = true
        go func() {
            defer func() { c.mu.Lock(); c.running = false; c.mu.Unlock() }()
            _ = c.compact()
        }()
    }
}
```

The `compact()` method must also skip `INVALID` frames (in addition to tombstoned ones) when building the output file.

### 4.8 Updates to `internal/storage/nrv_storage.go`

Add ticker management to `NRVStorage`:

```go
type NRVStorage struct {
    baseDir   string
    keyPair   *pqc.PQCKeyPair
    writers   map[string]*NRVWriter
    readers   map[string]*NRVReader
    tickers   map[string]*FrameTicker   // NEW
    compactor map[string]*Compactor
    mu        sync.RWMutex
    fileStore *FileStorage
}

func (s *NRVStorage) getOrCreateTicker(collection string) (*FrameTicker, error) {
    s.mu.Lock()
    defer s.mu.Unlock()

    if t, ok := s.tickers[collection]; ok {
        return t, nil
    }

    w, err := s.getOrCreateWriterLocked(collection)
    if err != nil {
        return nil, err
    }

    t := NewFrameTicker(w, time.Second)
    s.tickers[collection] = t
    return t, nil
}
```

Update `Close()` to stop all tickers before closing writers:

```go
func (s *NRVStorage) Close() error {
    s.mu.Lock()
    defer s.mu.Unlock()

    for _, t := range s.tickers {
        t.Stop() // flushes pending brackets
    }
    for _, w := range s.writers {
        w.Close()
    }
    for _, r := range s.readers {
        r.Close()
    }
    for _, c := range s.compactor {
        c.Stop()
    }
    return nil
}
```

### 4.9 New File: `internal/network/flight_server.go`

The Arrow Flight server maps the `.nrv` Chunk 1 buffer for zero-copy streaming. It implements the dual-stream policy:

- **Gold stream** (`ticket = "gold.<collection>"`) — emits only brackets from frames where `z3.status == "VALID"`.
- **Research stream** (`ticket = "research.<collection>"`) — emits all brackets regardless of Z3 status.

```go
package network

import (
    "context"
    "fmt"
    "strings"

    "github.com/apache/arrow/go/v15/arrow"
    "github.com/apache/arrow-go/v15/arrow/flight"
    "github.com/apache/arrow-go/v15/arrow/memory"
    stor "github.com/knirvcorp/knirvbase/go/internal/storage"
    "github.com/knirvcorp/knirvbase/go/pkg/nrv"
)

// FlightServer implements arrow/flight.FlightServer for `.nrv` bracket streaming.
type FlightServer struct {
    flight.BaseFlightServer
    storage *stor.NRVStorage
    alloc   memory.Allocator
}

func NewFlightServer(storage *stor.NRVStorage) *FlightServer {
    return &FlightServer{
        storage: storage,
        alloc:   memory.NewGoAllocator(),
    }
}

// DoGet streams brackets for a collection. Ticket format: "<stream>.<collection>"
// where stream is "gold" or "research".
func (s *FlightServer) DoGet(ticket *flight.Ticket, stream flight.FlightService_DoGetServer) error {
    parts := strings.SplitN(string(ticket.Ticket), ".", 2)
    if len(parts) != 2 {
        return fmt.Errorf("flight: invalid ticket format, expected <stream>.<collection>")
    }
    streamType, collection := parts[0], parts[1]
    goldOnly := streamType == "gold"

    reader, err := s.storage.GetReader(collection)
    if err != nil {
        return fmt.Errorf("flight: open reader: %w", err)
    }

    schema := bracketArrowSchema()
    writer := flight.NewRecordWriter(stream, ipc.WithSchema(schema))
    defer writer.Close()

    bracketCh := reader.StreamBrackets(goldOnly)
    batch := make([]*nrv.Bracket, 0, 1024)

    for b := range bracketCh {
        batch = append(batch, b)
        if len(batch) >= 1024 {
            if err := s.flushBatch(writer, batch, schema); err != nil {
                return err
            }
            batch = batch[:0]
        }
    }
    if len(batch) > 0 {
        return s.flushBatch(writer, batch, schema)
    }
    return nil
}

// bracketArrowSchema returns the Arrow schema for an 80-byte bracket.
func bracketArrowSchema() *arrow.Schema {
    return arrow.NewSchema([]arrow.Field{
        {Name: "id",           Type: arrow.BinaryTypes.String},
        {Name: "lsh_salt",     Type: arrow.PrimitiveTypes.Uint32},
        {Name: "projections",  Type: arrow.ListOf(arrow.PrimitiveTypes.Float32)},
        {Name: "subsecond_us", Type: arrow.PrimitiveTypes.Uint32},
        {Name: "asic_loops",   Type: arrow.PrimitiveTypes.Uint32},
        {Name: "golden_seed",  Type: arrow.PrimitiveTypes.Uint32},
    }, nil)
}

func (s *FlightServer) flushBatch(writer *ipc.Writer, batch []*nrv.Bracket, schema *arrow.Schema) error {
    // Build Arrow RecordBatch from bracket slice and write to stream
    // ... (standard Arrow builder pattern)
    return nil
}
```

Add `GetReader` accessor to `NRVStorage` (package-internal method exposed for Flight):

```go
func (s *NRVStorage) GetReader(collection string) (*NRVReader, error) {
    return s.getOrCreateReader(collection)
}
```

**Arrow dependency:** Add to `go.mod`:
```
require github.com/apache/arrow/go/v15 v15.x.x
```

### 4.10 Updates to `internal/query/knirvql.go`

**Add new filter keys** understood by `parseGet`:

| Filter Key | Operator | Example |
|---|---|---|
| `z3_status` | `=` | `WHERE z3_status = VALID` |
| `avg_temp_c` | `<`, `>`, `<=`, `>=` | `WHERE avg_temp_c < 85` |
| `drift_score` | `<`, `>` | `WHERE drift_score > 0.1` |
| `bracket_type` | `=` | `WHERE bracket_type = I` |

**Add new query syntax** to `parseGet`:

```
GET MEMORY.BRACKET(golden_seed) WHERE ...
GET MEMORY WHERE z3_status = VALID
GET MEMORY WHERE avg_temp_c < 85 AND z3_status = VALID
```

Add `QueryGetBracketField` to the `QueryType` enum:

```go
const (
    QueryGet QueryType = iota
    QuerySet
    QueryDelete
    QueryCreateIndex
    QueryCreateCollection
    QueryDropIndex
    QueryDropCollection
    QueryGetModality      // existing: GET MEMORY.MODALITY(type)
    QueryGetBracketField  // new:      GET MEMORY.BRACKET(field)
)
```

Update `parseGet` to detect `MEMORY.BRACKET(` prefix identically to how `MEMORY.MODALITY(` is handled, setting `q.Type = QueryGetBracketField` and storing the field name in `q.ModalityType` (reuse the field, rename it to `FieldName` in a follow-up cleanup).

Update `matchesFilter` to resolve frame-level fields from the top-level document (not just `payload`):

```go
case "z3_status":
    if z3, ok := doc["z3"].(map[string]interface{}); ok {
        return fmt.Sprintf("%v", z3["status"]) == fmt.Sprintf("%v", filter.Value)
    }
    return false
case "avg_temp_c":
    if thermo, ok := doc["thermo"].(map[string]interface{}); ok {
        return compareValues(thermo["avg_temp_c"], filter.Value) matches operator
    }
    return false
```

### 4.11 Updates to `pkg/knirvbase/knirvbase.go`

**Rename `NRVDataset` methods** to use the new terminology:

| Old Method | New Method | Notes |
|---|---|---|
| `AppendFrame(ctx, frame, verified, ergoRank)` | `AppendBracket(ctx, bracket, thermo)` | Routes to `FrameTicker.AppendBracket` |
| `StreamFrames(ctx, modalityFilter)` | `StreamBrackets(ctx, goldOnly bool)` | Routes to `NRVReader.StreamBrackets` |
| `GetModality(ctx, frameID, mod)` | `GetFrame(ctx, frameID)` | Returns `*nrv.FrameEntry` + brackets |

Updated `NRVDataset`:

```go
// AppendBracket enqueues a bracket into the current 1-second frame window via the FrameTicker.
func (ds *NRVDataset) AppendBracket(ctx context.Context, b *nrv.Bracket, thermo nrv.ThermoAtmosphere) error {
    return ds.storage.AppendBracketDirect(ds.name, b, thermo)
}

// StreamBrackets returns a channel of decoded brackets. If goldOnly is true, only
// brackets from Z3-VALID frames are emitted (Gold stream). Otherwise all live brackets
// are emitted (Research stream).
func (ds *NRVDataset) StreamBrackets(ctx context.Context, goldOnly bool) (<-chan *nrv.Bracket, error) {
    return ds.storage.StreamBrackets(ctx, ds.name, goldOnly)
}

// GetFrame returns the FrameEntry registry metadata and decoded brackets for one 1-second frame.
func (ds *NRVDataset) GetFrame(ctx context.Context, frameID string) (*nrv.FrameEntry, []*nrv.Bracket, error) {
    return ds.storage.GetFrame(ctx, ds.name, frameID)
}

// SetLinguistic sets the current linguistic context on the FrameTicker for the named collection.
func (ds *NRVDataset) SetLinguistic(token, unit string) error {
    return ds.storage.SetLinguistic(ds.name, token, unit)
}
```

**Remove `Raw()` and `RawCollection()`** from `DB`. These exposed internal types and are no longer needed now that the dataset API covers all access patterns.

---

## 5. Validation Flow Integration

The full ASIC pipeline integration through KNIRVBASE:

```
ASIC (0x52 protocol)
    ↓  80-byte task result
NRVDataset.AppendBracket(ctx, bracket, thermo)
    ↓
NRVStorage.AppendBracketDirect(collection, bracket, thermo)
    ↓
FrameTicker.AppendBracket(bracket, thermo)   ← buffers, computes I/P type, XOR-diffs
    │
    │ every 1 second
    ↓
FrameTicker.flush()
    ↓
NRVWriter.AppendFrame(frameID, bracketBuf, bracketIndex, thermo, linguistic)
    ↓  flock + WAL begin
    ├─ Write bracketBuf → EOF (Chunk 1)
    ├─ Sign bracketBuf → Dilithium-3 → PQCManifest
    ├─ Run Z3 gate → FrameEntry.Z3.Status = "VALID" | "INVALID"
    ├─ Update Registry.Frames → saveRegistry() → WriteAt(Chunk 0)
    ├─ Update Header.TotalLength → EncodeHeader()
    └─ WAL commit
    ↓
MaybeCompact() ← triggered if (tombstoned + invalid) / total ≥ 20%
```

---

## 6. Testing Requirements

### 6.1 Unit Tests

| Test | File | Assertions |
|---|---|---|
| `TestEncodeBracket_RoundTrip` | `pkg/nrv/codec_test.go` | `EncodeBracket` → `DecodeBracket` round-trips all fields exactly |
| `TestXORProjections_Inverse` | `pkg/nrv/codec_test.go` | `ApplyProjectionDelta(XORProjections(a,b), b) == a` for all inputs |
| `TestFrameTicker_Flush` | `internal/storage/nrv_ticker_test.go` | After 1s tick, NRVWriter has 1 FrameEntry with correct bracket count |
| `TestFrameTicker_IBracketFrequency` | `internal/storage/nrv_ticker_test.go` | Every 50th bracket is type `I`; P-brackets have non-nil `anchor_id` |
| `TestFrameTicker_DriftSpike` | `internal/storage/nrv_ticker_test.go` | Drift > 0.25 forces an I-bracket outside of the fixed interval |
| `TestNRVWriter_AppendFrame_NewSignature` | `internal/storage/nrv_writer_test.go` | Per-frame Dilithium-3 signature present in PQCManifest |
| `TestNRVReader_StreamBrackets_GoldOnly` | `internal/storage/nrv_reader_test.go` | Gold stream skips INVALID frames; Research stream includes them |
| `TestNRVReader_DecodePBracket` | `internal/storage/nrv_reader_test.go` | P-bracket projections reconstructed correctly from anchor |
| `TestCompactor_InvalidFrameRatio` | `internal/storage/nrv_compactor_test.go` | Compaction fires when INVALID frames push ratio ≥ 20% |
| `TestKNIRVQL_Z3StatusFilter` | `internal/query/knirvql_test.go` | `WHERE z3_status = VALID` returns only valid-frame brackets |
| `TestKNIRVQL_ThermoFilter` | `internal/query/knirvql_test.go` | `WHERE avg_temp_c < 85` filters frames correctly |
| `TestKNIRVQL_BracketFieldQuery` | `internal/query/knirvql_test.go` | `GET MEMORY.BRACKET(golden_seed)` parses to `QueryGetBracketField` |

### 6.2 Integration Tests

| Test | Scenario |
|---|---|
| `TestEndToEnd_ASICPipeline` | Append 1000 brackets over 3 seconds → verify 3 FrameEntries in registry with correct counts, Z3 status, and Dilithium-3 signatures |
| `TestEndToEnd_FlightGoldStream` | Flight client with `gold.<collection>` ticket → only brackets from VALID frames received |
| `TestEndToEnd_FlightResearchStream` | Flight client with `research.<collection>` ticket → all brackets including INVALID frames received |
| `TestEndToEnd_CompactionPreservesGold` | Mark frames INVALID, trigger compaction → output file contains only VALID/live frames |
| `TestEndToEnd_CrashRecovery` | Kill writer mid-flush → WAL recovery on re-open → no partial frames in registry |

### 6.3 Benchmarks

| Benchmark | Target |
|---|---|
| `BenchmarkAppendBracket` | Throughput of `FrameTicker.AppendBracket` under concurrent callers |
| `BenchmarkFlush_1000Brackets` | Time for `FrameTicker.flush()` with 1000 pending brackets |
| `BenchmarkStreamBrackets_Gold` | Flight Gold stream throughput in MB/s for a 10k-bracket file |
| `BenchmarkDecodePBracket` | XOR reconstruction overhead vs direct I-bracket decode |

---

## 7. Migration Notes

### 7.1 Existing `.nrv` Files

Phase 1 files used the old per-record `FrameEntry` schema. They are not directly readable by the Phase 2 reader. A one-time migration tool should be provided:

- Read each Phase 1 `FrameEntry` as a legacy record.
- Treat each record as a single I-Bracket in a 1-second frame (one frame per record, `timestamp_unix = entry creation time`).
- Map legacy fields: `Vector` → `Projections` (pack as float32 bytes), `Seed` → `GoldenSeed` (first 4 bytes), `Thermo` → `ThermoAtmosphere`.
- Z3 status: `verified == true` → `VALID`, else → `INVALID`.
- Write to a new Phase 2 `.nrv` file.

### 7.2 Callers of `NRVDataset`

| Old Call | New Call |
|---|---|
| `ds.AppendFrame(ctx, frame, true, 0.9)` | `ds.AppendBracket(ctx, bracket, thermo)` |
| `ds.StreamFrames(ctx, nrv.ModalityVector)` | `ds.StreamBrackets(ctx, false)` for Research or `(ctx, true)` for Gold |
| `ds.GetModality(ctx, id, nrv.ModalitySeed)` | `_, brackets, _ := ds.GetFrame(ctx, frameID)` then read `brackets[i].GoldenSeed` |

### 7.3 KNIRVQL Callers

Existing queries using `GET MEMORY.MODALITY(...)` remain valid. New queries:

```
# Gold-only fetch
GET MEMORY WHERE z3_status = VALID

# Hardware-filtered fetch
GET MEMORY WHERE avg_temp_c < 85 AND z3_status = VALID

# Bracket field projection
GET MEMORY.BRACKET(golden_seed) WHERE z3_status = VALID
```

---

## 8. Backlog (V3)

- **Reasoning Ledger**: `agent_context` field on `FrameEntry` linking to immutable agent-access log.
- **Cross-dataset joins**: multi-file KNIRVQL spanning multiple `.nrv` datasets.
- **GPU-native loaders**: Rust Vulkan/CUDA kernel reading Chunk 1 directly to VRAM.
- **Advanced compression**: Zstd for Golden Seed sequences and Z3 relevance score arrays within the registry.
- **KNIRVQL analytics**: aggregate queries on thermodynamic metrics across frames (e.g., `AVG(avg_temp_c) GROUP BY z3_status`).
