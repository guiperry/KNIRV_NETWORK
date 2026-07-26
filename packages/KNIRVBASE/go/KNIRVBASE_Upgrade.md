# KNIRVBASE Upgrade: `.nrv` Format — Phase 2 (ASIC-Native Spec)

**Status:** Phase 2 — Building on fully implemented Phase 1 foundation.
**Module:** `knirvbase` (Go 1.24.6)
**Root:** `packages/KNIRVBASE/go/` (all paths below are relative to this root)

> **🔗 CROSS-REFERENCE — 12-Slot Bitmask Specification**: Phase 2 implements the **12-slot Bitmask Specification** from `packages/KNIRVHASHER/docs/DATA-MAPPER.md`. This specification defines how the 80-byte Bracket encodes semantic information for the ASIC's 21-pass temporal loop:
> - **Slots 0-3** (Identity Zone): LSH Projections (64 bytes) → Semantic Compass for Passes 1-7
> - **Slots 4-5** (Syntactic Registers): POS, Tense, Dependency → **REQUIRED for Syntactic Steering (Passes 8-14)**
> - **Slots 6-8** (Memory Zone): History XOR (maintained in FrameTicker memory) → Temporal loop recurrence
> - **Slot 9** (Intent): Question/Command/Code flags
> - **Slot 10** (Domain): Math/Code/Prose classification
> - **Slot 11** (Temporal Salt): `(PosIndex << 16) | TemporalSalt` — Contextual Anchor for warm uniqueness
>
> The 80-byte Bracket binary layout MUST preserve this specification. Without Syntactic Registers (Slots 4-5), the 21-pass loop collapses to random hashing.

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

> **🔗 CROSS-REFERENCE**: The Bracket binary format encodes the **12-slot Bitmask Specification** from `packages/KNIRVHASHER/docs/DATA-MAPPER.md`. This preserves the Semantic Coherence required for the ASIC's 21-pass temporal loop with Syntactic Steering (Passes 8-14).

**Bracket binary layout (80 bytes):**

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
| `0x3D` | 4B | LSH Salt | Slot 11 (Salt) | `(PosIndex << 16) | TemporalSalt` — entropy anchor for warm uniqueness |
| `0x41` | 15B | Reserved | (Future Expansion) | Padding to 80 bytes |

Total: 32 + 4 + 1 + 1 + 1 + 1 + 1 + 2 + 4 + 14 + 4 + 15 = **80 bytes**.

> **⚠️ CRITICAL**: 
> - **Slots 0-3** (Identity Zone): LSH Projections (32B) → Semantic Compass for Passes 1-7
> - **Slots 4-5** (Syntactic Registers): POSTag, Tense, Plurality, DepHead → **REQUIRED for Syntactic Steering (Passes 8-14)**
> - **Slots 6-8** (Memory Zone): 14B XOR recursive → Temporal loop recurrence
> - **Slot 9** (Intent): IntentFlags → Question/Command/Code detection
> - **Slot 10** (Domain): DomainSig → Math/Code/Prose classification
> - **Slot 11** (Temporal Salt): LSH Salt → `(PosIndex << 16) | TemporalSalt` — Contextual Anchor (warm uniqueness)

Without Syntactic Registers (Slots 4-5), the 21-pass loop collapses to random hashing.

**Delta encoding (P-Brackets):** For P-brackets, bytes `0x04–0x43` (Projections A–H, 32 bytes) store the **XOR-diff** against the anchor I-bracket's projection bytes. LSH Salt, Metadata, and Golden Seed are always stored as absolute values regardless of bracket type.

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
// The 80-byte binary encodes the 12-slot Bitmask Specification from DATA-MAPPER.md
// to preserve Semantic Coherence for the 21-pass temporal loop.
type Bracket struct {
	ID          string    // registry-only; not stored in binary
	
	// bytes 0x00–0x1F: Slots 0-3 (Identity Zone) - LSH Projections (32B)
	Projections [32]byte  // 16-dim LSH → Semantic Compass for Passes 1-7
	
	// bytes 0x20–0x23: Temporal Ticker
	SubSecondUS uint32    // sub-second timestamp in microseconds
	
	// bytes 0x24–0x27: Slots 4-5 (Syntactic Registers)
	// CRITICAL: Required for Syntactic Steering in Passes 8-14
	POSTag      uint8     // Slot 4, bits 0-7: POS Tag (0x01=Noun, 0x02=Verb, etc.)
	Tense       uint8     // Slot 4, bits 8-11: Tense (0x1=Past, 0x2=Present, etc.)
	Plurality   uint8     // Slot 4, bits 12-15: Plurality (0x1=Singular, 0x2=Plural)
	DepHead     int8      // Slot 5: Dependency Head index (1 byte, -128 to 127)
	
	// bytes 0x28–0x29: Slot 9 (Intent)
	IntentFlags uint8     // bits 0-3: IS_QUESTION(0x1), IS_COMMAND(0x2), IS_CODE(0x4)
	
	// bytes 0x29–0x2A: Slot 10 (Domain) - note: overlaps with DomainSig below
	DomainSig   uint16    // bits 8-15: Domain Signature (0x1000=Prose, 0x2000=Math, 0x3000=Code)
	
	// bytes 0x2B–0x2E: GoldenSeed (Nonce Target)
	GoldenSeed  uint32    // solved nonce — the result of the ASIC pass
	
	// bytes 0x2F–0x3C: Slots 6-8 (Memory Zone) - XOR recursive for temporal loop
	Memory      [14]byte // Recursive Context Bridge - History XOR for 21-pass recurrence
	
	// bytes 0x3D–0x40: Slot 11 (Temporal Salt - Entropy Anchor)
	LSHSalt     uint32    // Slot 11: (PosIndex << 16) | TemporalSalt — Contextual Anchor for warm uniqueness
	
	// bytes 0x41–0x4F: Reserved for future expansion
	Reserved    [15]byte // padding to 80 bytes
}

// SyntacticProfile holds the linguistic metadata for Syntactic Steering
// Corresponds to bytes 0x24-0x27 (4 bytes) in the 80-byte bracket
type SyntacticProfile struct {
    POSTag     uint8 // Slot 4, bits 0-7: POS Tag ID
    Tense      uint8 // Slot 4, bits 8-11: Tense ID
    Plurality  uint8 // Slot 4, bits 12-15: Plurality
    DepHead    int8  // Slot 5: Dependency Head index (1 byte, -128 to 127)
}

// IntentDomain holds the intent and domain classification
// Corresponds to bytes 0x28-0x2A (3 bytes) in the 80-byte bracket
type IntentDomain struct {
    IntentFlags uint8  // Slot 9 (byte 0x28): Question/Command/Code flags
    DomainSig  uint16 // Slot 10 (bytes 0x29-0x2A): Domain Signature (Math/Code/Prose)
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

> **🔗 CROSS-REFERENCE**: The slot mapping follows `packages/KNIRVHASHER/docs/DATA-MAPPER.md`:
> - **Slots 0-3** (Identity Zone): LSH Projections — the "Semantic Compass" for Passes 1-7
> - **Slots 4-5** (Syntactic Registers): POS, Tense, Dependency — **REQUIRED for Passes 8-14** (Syntactic Steering)
> - **Slots 6-8** (Memory Zone): History XOR — stored in memory by FrameTicker, not in bracket binary
> - **Slot 9** (Intent): Question/Command/Code detection
> - **Slot 10** (Domain): Math/Code/Prose classification
> - **Slot 11** (Temporal Salt): `(PosIndex << 16) | TemporalSalt` — Contextual Anchor for warm uniqueness

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
// The encoding follows the 12-slot Bitmask Specification from DATA-MAPPER.md:
//   - bytes 0x00-0x1F: Projections [32]byte (Slots 0-3: Identity Zone - LSH)
//   - bytes 0x20-0x23: SubSecondUS (Temporal Ticker)
//   - bytes 0x24-0x27: Syntactic Registers (Slots 4-5: POSTag, Tense, Plurality, DepHead)
//   - bytes 0x28-0x2A: Intent + Domain (Slot 9: IntentFlags, Slot 10: DomainSig)
//   - bytes 0x2B-0x2E: GoldenSeed (Nonce Target)
//   - bytes 0x2F-0x3C: Memory [14]byte (Slots 6-8: XOR recursive for temporal loop)
//   - bytes 0x3D-0x40: LSHSalt (Slot 11: Temporal Salt - Contextual Anchor for warm uniqueness)
//   - bytes 0x41-0x4F: Reserved [15]byte (padding to 80 bytes)
func EncodeBracket(b *Bracket) [BracketSize]byte {
    var buf [BracketSize]byte
    // Projections (Slots 0-3: Identity Zone - 16-dim LSH)
    copy(buf[0:32], b.Projections[:])
    // SubSecondUS
    binary.LittleEndian.PutUint32(buf[32:36], b.SubSecondUS)
    // Syntactic Registers (Slots 4-5: POS, Tense, Plurality, DepHead)
    buf[36] = b.POSTag
    buf[37] = b.Tense
    buf[38] = b.Plurality
    buf[39] = byte(b.DepHead)
    // Intent + Domain (Slot 9 + Slot 10)
    buf[40] = b.IntentFlags
    binary.LittleEndian.PutUint16(buf[41:43], b.DomainSig)
    // GoldenSeed
    binary.LittleEndian.PutUint32(buf[43:47], b.GoldenSeed)
    // Memory (Slots 6-8: XOR recursive for temporal loop)
    copy(buf[47:61], b.Memory[:])
    // LSHSalt (Slot 11: Temporal Salt - Contextual Anchor)
    binary.LittleEndian.PutUint32(buf[61:65], b.LSHSalt)
    // Reserved (padding)
    return buf
}

// DecodeBracket parses an 80-byte buffer into a Bracket.
func DecodeBracket(buf [BracketSize]byte) Bracket {
    var b Bracket
    // Projections (Slots 0-3)
    copy(b.Projections[:], buf[0:32])
    // SubSecondUS
    b.SubSecondUS = binary.LittleEndian.Uint32(buf[32:36])
    // Syntactic Registers (Slots 4-5)
    b.POSTag = buf[36]
    b.Tense = buf[37]
    b.Plurality = buf[38]
    b.DepHead = int8(buf[39])
    // Intent + Domain (Slot 9 + Slot 10)
    b.IntentFlags = buf[40]
    b.DomainSig = binary.LittleEndian.Uint16(buf[41:43])
    // GoldenSeed
    b.GoldenSeed = binary.LittleEndian.Uint32(buf[43:47])
    // Memory (Slots 6-8)
    copy(b.Memory[:], buf[47:61])
    // LSHSalt (Slot 11)
    b.LSHSalt = binary.LittleEndian.Uint32(buf[61:65])
    return b
}

// EncodeSyntactic encodes SyntacticProfile into a uint32 for slot packing
func EncodeSyntactic(sp SyntacticProfile) uint32 {
    var val uint32
    val |= uint32(sp.POSTag) // bits 0-7
    val |= uint32(sp.Tense) << 8 // bits 8-11
    val |= uint32(sp.Plurality) << 12 // bits 12-15
    val |= uint32(uint16(sp.DepHead)) << 16 // bits 16-31
    return val
}

// DecodeSyntactic decodes a uint32 into SyntacticProfile
func DecodeSyntactic(val uint32) SyntacticProfile {
    return SyntacticProfile{
        POSTag:    uint8(val & 0xFF),
        Tense:     uint8((val >> 8) & 0xF),
        Plurality: uint8((val >> 12) & 0xF),
        DepHead:   int16(val >> 16),
    }
}

// EncodeIntentDomain encodes IntentDomain into a uint32 for slot packing
func EncodeIntentDomain(id IntentDomain) uint32 {
    var val uint32
    val |= uint32(id.IntentFlags) & 0xF // bits 0-3
    val |= uint32(id.DomainSig) << 8 // bits 8-15
    return val
}

// DecodeIntentDomain decodes a uint32 into IntentDomain
func DecodeIntentDomain(val uint32) IntentDomain {
    return IntentDomain{
        IntentFlags: uint8(val & 0xF),
        DomainSig:   uint16((val >> 8) & 0xFF),
    }
}

// XORProjections returns the XOR-diff of `current` projections against `anchor` projections.
// Used to produce P-bracket payloads (32 bytes instead of 64).
func XORProjections(current, anchor [32]byte) [32]byte {
    var diff [32]byte
    for i := range diff {
        diff[i] = current[i] ^ anchor[i]
    }
    return diff
}

// ApplyProjectionDelta reconstructs absolute projections from a P-bracket XOR-diff and its anchor.
func ApplyProjectionDelta(delta, anchor [32]byte) [32]byte {
    return XORProjections(delta, anchor) // XOR is its own inverse
}
```

**Deprecate** `EncodeFrame` (the old multimodal frame codec). It can remain for backward compatibility with existing data but should not be used for new writes.

### 4.4 New File: `internal/storage/nrv_ticker.go`

The `FrameTicker` is the central coordination component for Phase 2. It owns a 1-second window, buffers incoming brackets, and flushes a complete `FrameEntry` to the `NRVWriter` at each tick boundary.

> **🔗 CROSS-REFERENCE**: The FrameTicker preserves the **12-slot Bitmask Specification**:
> - **Slots 0-3** (Identity Zone): Encoded in bracket.Projections
> - **Slots 4-5** (Syntactic Registers): Encoded in bracket.POSTag, Tense, Plurality, DepHead
> - **Slots 6-8** (Memory Zone): Maintained in `historyXOR` — the XOR of previous bracket hashes for the 21-pass temporal loop recurrence
> - **Slot 9** (Intent): Encoded in bracket.IntentFlags
> - **Slot 10** (Domain): Encoded in bracket.DomainSig
> - **Slot 11** (Temporal Salt): `(PosIndex << 16) | TemporalSalt` — Contextual Anchor for warm uniqueness

```go
package storage

import (
    "context"
    "math"
    "sync"
    "time"

    "github.com/google/uuid"
    "knirvbase/pkg/nrv"
)

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

// PendingBracket holds a bracket and its computed delta metadata before flush.
type PendingBracket struct {
    Bracket    *nrv.Bracket
    DeltaType  nrv.DeltaType
    AnchorID   *string
    DriftScore float64
    
    // Memory Zone (Slots 6-8): Stored per-bracket for temporal loop recurrence
    // These are NOT in the 80-byte bracket binary - maintained in memory
    HistoryXOR [3]uint32
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
    
    // Memory Zone (Slots 6-8): Maintained in memory for 21-pass temporal loop
    memoryZone MemoryZone

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
// Also updates the Memory Zone (Slots 6-8) for the 21-pass temporal loop recurrence.
func (ft *FrameTicker) AppendBracket(b *nrv.Bracket, thermo nrv.ThermoAtmosphere) {
    ft.mu.Lock()
    defer ft.mu.Unlock()

    var deltaType nrv.DeltaType
    var anchorID *string
    var driftScore float64

    // Compute Memory Zone (Slots 6-8): Rolling XOR for temporal loop recurrence
    // This drives the 21-pass loop's history-aware hashing
    historyXOR := ft.computeMemoryXOR(b)

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
        HistoryXOR: historyXOR, // Slot 6-8 for temporal loop
    })
    ft.thermoSamples = append(ft.thermoSamples, thermo)
    ft.bktCount++
}

// computeMemoryXOR computes the rolling XOR for Slots 6-8 (Memory Zone)
// This provides the "history" that drives the 21-pass temporal loop recurrence
func (ft *FrameTicker) computeMemoryXOR(b *nrv.Bracket) [3]uint32 {
    // Slot 6: XOR of current bracket hash with previous Slot 6
    slot6 := ft.memoryZone.HistoryXOR[0] ^ hashBracket(b)
    // Slot 7: XOR of Slot 6 with previous Slot 7 (deepening recurrence)
    slot7 := ft.memoryZone.HistoryXOR[1] ^ slot6
    // Slot 8: XOR of Slot 7 with previous Slot 8 (deepest recurrence)
    slot8 := ft.memoryZone.HistoryXOR[2] ^ slot7
    
    // Update memory zone for next iteration
    ft.memoryZone.HistoryXOR[0] = slot6
    ft.memoryZone.HistoryXOR[1] = slot7
    ft.memoryZone.HistoryXOR[2] = slot8
    
    return [3]uint32{slot6, slot7, slot8}
}

// hashBracket computes a simple hash of the bracket for memory zone XOR
// Uses Projections (32B, Slots 0-3), GoldenSeed, and LSHSalt for hash
func hashBracket(b *nrv.Bracket) uint32 {
    h := uint32(b.LSHSalt)
    for i := 0; i < 32; i += 4 {
        h ^= binary.LittleEndian.Uint32(b.Projections[i:i+4])
    }
    h ^= b.SubSecondUS
    h ^= b.GoldenSeed
    return h
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
    stor "knirvbase/internal/storage"
    "knirvbase/pkg/nrv"
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

| Filter Key | Operator | Example | 12-Slot Mapping |
|---|---|---|---|
| `z3_status` | `=` | `WHERE z3_status = VALID` | Frame metadata |
| `avg_temp_c` | `<`, `>`, `<=`, `>=` | `WHERE avg_temp_c < 85` | ThermoAtmosphere |
| `drift_score` | `<`, `>` | `WHERE drift_score > 0.1` | BracketMeta.DriftScore |
| `bracket_type` | `=` | `WHERE bracket_type = I` | BracketMeta.Type |
| `pos_tag` | `=` | `WHERE pos_tag = NOUN` | **Slot 4 (bits 0-7)** — Syntactic Register |
| `tense` | `=` | `WHERE tense = PAST` | **Slot 4 (bits 8-11)** — Syntactic Register |
| `domain` | `=` | `WHERE domain = MATH` | **Slot 10** — Domain Signature |
| `intent_flags` | `&` | `WHERE intent_flags & 0x1` | **Slot 9** — Intent Flags |
| `lsh_salt` | `=` | `WHERE lsh_salt = 0x12345678` | **Slot 11** — Temporal Salt (Contextual Anchor) |

> **⚠️ IMPORTANT**: The semantic filters (`pos_tag`, `tense`, `domain`, `intent_flags`) correspond to the **12-slot Bitmask Specification** from DATA-MAPPER.md. These enable ASIC-aware filtering where:
> - `pos_tag` and `tense` enable filtering by grammatical constraints (Slot 4-5)
> - `domain` enables filtering by Math/Code/Prose environment (Slot 10)
> - `intent_flags` enables filtering by Question/Command/Code markers (Slot 9)

**Add new query syntax** to `parseGet`:

```
GET MEMORY.BRACKET(golden_seed) WHERE ...
GET MEMORY WHERE z3_status = VALID
GET MEMORY WHERE avg_temp_c < 85 AND z3_status = VALID
GET MEMORY WHERE domain = MATH AND pos_tag = VERB  # ASIC semantic filtering
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
case "pos_tag":
    // Slot 4, bits 0-7: POS Tag from Syntactic Register
    if bracket, ok := doc["bracket"].(map[string]interface{}); ok {
        if proj, ok := bracket["projections"].(map[string]interface{}); ok {
            return fmt.Sprintf("%v", proj["pos_tag"]) == fmt.Sprintf("%v", filter.Value)
        }
    }
    return false
case "domain":
    // Slot 10: Domain Signature (0x1000=Prose, 0x2000=Math, 0x3000=Code)
    if bracket, ok := doc["bracket"].(map[string]interface{}); ok {
        if meta, ok := bracket["meta"].(map[string]interface{}); ok {
            return fmt.Sprintf("%v", meta["domain"]) == fmt.Sprintf("%v", filter.Value)
        }
    }
    return false
case "intent_flags":
    // Slot 9: Intent Flags (IS_QUESTION=0x1, IS_COMMAND=0x2, IS_CODE=0x4)
    if bracket, ok := doc["bracket"].(map[string]interface{}); ok {
        if meta, ok := bracket["meta"].(map[string]interface{}); ok {
            flags, _ := meta["intent_flags"].(uint8)
            val, _ := filter.Value.(uint8)
            return (flags & val) == val
        }
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

| Test | File | Assertions | 12-Slot Coverage |
|---|---|---|---|
| `TestEncodeBracket_RoundTrip` | `pkg/nrv/codec_test.go` | `EncodeBracket` → `DecodeBracket` round-trips all fields exactly | All 12 slots |
| `TestXORProjections_Inverse` | `pkg/nrv/codec_test.go` | `ApplyProjectionDelta(XORProjections(a,b), b) == a` for all inputs | Slots 0-3 |
| `TestEncodeSyntactic_PackUnpack` | `pkg/nrv/codec_test.go` | `EncodeSyntactic` → `DecodeSyntactic` preserves all fields | Slots 4-5 |
| `TestEncodeIntentDomain_PackUnpack` | `pkg/nrv/codec_test.go` | `EncodeIntentDomain` → `DecodeIntentDomain` preserves all fields | Slots 9-10 |
| `TestFrameTicker_Flush` | `internal/storage/nrv_ticker_test.go` | After 1s tick, NRVWriter has 1 FrameEntry with correct bracket count | All 12 slots |
| `TestFrameTicker_IBracketFrequency` | `internal/storage/nrv_ticker_test.go` | Every 50th bracket is type `I`; P-brackets have non-nil `anchor_id` | Slots 0-3 |
| `TestFrameTicker_DriftSpike` | `internal/storage/nrv_ticker_test.go` | Drift > 0.25 forces an I-bracket outside of the fixed interval | Slots 0-3 |
| `TestFrameTicker_MemoryZone` | `internal/storage/nrv_ticker_test.go` | HistoryXOR computed correctly for Slots 6-8 temporal recurrence | Slots 6-8 |
| `TestNRVWriter_AppendFrame_NewSignature` | `internal/storage/nrv_writer_test.go` | Per-frame Dilithium-3 signature present in PQCManifest | All slots |
| `TestNRVReader_StreamBrackets_GoldOnly` | `internal/storage/nrv_reader_test.go` | Gold stream skips INVALID frames; Research stream includes them | All slots |
| `TestNRVReader_DecodePBracket` | `internal/storage/nrv_reader_test.go` | P-bracket projections reconstructed correctly from anchor | Slots 0-3 |
| `TestCompactor_InvalidFrameRatio` | `internal/storage/nrv_compactor_test.go` | Compaction fires when INVALID frames push ratio ≥ 20% | All slots |
| `TestKNIRVQL_Z3StatusFilter` | `internal/query/knirvql_test.go` | `WHERE z3_status = VALID` returns only valid-frame brackets | Frame metadata |
| `TestKNIRVQL_ThermoFilter` | `internal/query/knirvql_test.go` | `WHERE avg_temp_c < 85` filters frames correctly | Frame metadata |
| `TestKNIRVQL_BracketFieldQuery` | `internal/query/knirvql_test.go` | `GET MEMORY.BRACKET(golden_seed)` parses to `QueryGetBracketField` | All slots |
| `TestKNIRVQL_SemanticFilters` | `internal/query/knirvql_test.go` | `WHERE pos_tag = VERB AND domain = MATH` filters correctly | Slots 4-5, 10 |

### 6.2 Integration Tests

| Test | Scenario | 12-Slot Validation |
|---|---|---|
| `TestEndToEnd_ASICPipeline` | Append 1000 brackets over 3 seconds → verify 3 FrameEntries in registry with correct counts, Z3 status, and Dilithium-3 signatures | All 12 slots encoded and retrievable |
| `TestEndToEnd_SemanticCoherence` | Append brackets with known POS/Tense/Domain → query via semantic filters → verify Syntactic Steering works | Slots 4-5, 9-10 |
| `TestEndToEnd_21PassTemporalLoop` | Feed brackets through vHasher 21-pass loop → verify deterministic consensus with semantic coherence | All 12 slots |
| `TestEndToEnd_FlightGoldStream` | Flight client with `gold.<collection>` ticket → only brackets from VALID frames received | All slots |
| `TestEndToEnd_FlightResearchStream` | Flight client with `research.<collection>` ticket → all brackets including INVALID frames received | All slots |
| `TestEndToEnd_CompactionPreservesGold` | Mark frames INVALID, trigger compaction → output file contains only VALID/live frames | All slots |
| `TestEndToEnd_CrashRecovery` | Kill writer mid-flush → WAL recovery on re-open → no partial frames in registry | All slots |

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

# ASIC semantic filtering (requires 12-slot spec)
GET MEMORY WHERE pos_tag = VERB AND domain = MATH
GET MEMORY WHERE intent_flags & 0x1 AND tense = PAST
```

> **🔗 NOTE**: The semantic filters (`pos_tag`, `tense`, `domain`, `intent_flags`) only work on Phase 2 files with the 12-slot Bracket format. Phase 1 files will return empty results for these filters.

---

## 8. 21-Pass Temporal Loop Integration

The Phase 2 Bracket format is specifically designed to drive the **21-pass temporal loop** in the KNIRVHASHER ASIC pipeline. This section documents how the 12-slot specification maps to the loop passes.

### 8.1 Pass Structure

| Pass Range | Slot Zone | Purpose | Bracket Field |
|---|---|---|---|
| **Passes 1-7** | Identity Zone (Slots 0-3) | Semantic Compass | `Projections` (32B LSH) |
| **Passes 8-14** | Syntactic Registers (Slots 4-5) | **Syntactic Steering** | `POSTag`, `Tense`, `Plurality`, `DepHead` |
| **Passes 15-18** | Memory Zone (Slots 6-8) | Temporal Recurrence | `Memory` (14B XOR recursive) |
| **Passes 19-20** | Intent + Domain (Slots 9-10) | Logical Filtering | `IntentFlags`, `DomainSig` |
| **Pass 21** | Temporal Salt (Slot 11) | Entropy Anchor (warm uniqueness) | `(PosIndex << 16) | TemporalSalt` + `GoldenSeed` |

### 8.2 Syntactic Steering (Passes 8-14)

> **⚠️ CRITICAL**: Passes 8-14 perform Syntactic Steering using Slots 4-5 (POS, Tense, Dependency). If these fields are zero/empty, the loop cannot validate grammatical coherence and falls back to random hashing.

The FrameTicker's `HistoryXOR` computation (Slots 6-8) drives the temporal recurrence:
- Each bracket's hash is XORed with the previous bracket's hash
- This creates a "rolling memory" that the ASIC uses to maintain context across passes
- The XOR is deep: Slot 6 → Slot 7 → Slot 8 (each builds on the previous)

### 8.3 FrameTicker Memory Zone

The FrameTicker maintains `MemoryZone` in memory (NOT in the 80-byte bracket):
```go
type MemoryZone struct {
    HistoryXOR [3]uint32  // Slots 6, 7, 8: Rolling XOR of previous bracket hashes
    SeedXOR    uint32     // Initial seed for recursive hashing
}
```

This enables:
1. **Temporal Coherence**: Each pass builds on the previous pass's hash
2. **Delta Encoding**: P-brackets store XOR-diffs, reducing storage while preserving temporal context
3. **Syntactic Validation**: Passes 8-14 use Slot 4-5 to reject grammatically invalid token resolutions

---

## 9. Backlog (V3)

- **Reasoning Ledger**: `agent_context` field on `FrameEntry` linking to immutable agent-access log.
- **Cross-dataset joins**: multi-file KNIRVQL spanning multiple `.nrv` datasets.
- **GPU-native loaders**: Rust Vulkan/CUDA kernel reading Chunk 1 directly to VRAM.
- **Advanced compression**: Zstd for Golden Seed sequences and Z3 relevance score arrays within the registry.
- **KNIRVQL analytics**: aggregate queries on thermodynamic metrics across frames (e.g., `AVG(avg_temp_c) GROUP BY z3_status`).
