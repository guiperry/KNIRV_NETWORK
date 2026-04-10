# KNIRVBASE Upgrade Assessment: Phase 2 Gaps & Inconsistencies

**Assessed against:** `packages/KNIRVBASE/go/KNIRVBASE_Upgrade.md`  
**Date:** 2026-04-09  
**Status:** Multiple critical gaps requiring implementation

---

## 1. `pkg/nrv/bracket.go` — Missing 12-Slot Fields

### GAP: Incomplete Bracket Structure

| Spec Field | Offset | Status |
|---|---|---|
| `LSHSalt` (Slot 11) | 0x00 | ✅ Present |
| `Projections` (Slots 0-3) | 0x04 | ✅ Present |
| `SubSecondUS` | 0x44 | ✅ Present |
| **Semantic Metadata (Slots 4-5)** | **0x48** | ❌ **MISSING** |
| `ASICLoops` | 0x4C | ✅ Present |
| **Intent + Domain (Slots 9-10)** | **0x50** | ❌ **MISSING** |
| Reserved (Slots 6-8) | 0x54 | ❌ **MISSING** |
| `GoldenSeed` | 0x58 | ✅ Present |

**Missing Types:**
- `SyntacticProfile` struct (POS, Tense, Plurality, DepHead)
- `IntentDomain` struct (IntentFlags, DomainSig)

**Impact:** Without Syntactic Registers (Slots 4-5), the 21-pass temporal loop collapses to random hashing. The ASIC pipeline requires POS/Tense/Dependency for Syntactic Steering in Passes 8-14.

---

## 2. `pkg/nrv/codec.go` — Incomplete Bracket Encoding

### GAP: EncodeBracket/DecodeBracket Do Not Match Spec

The current `EncodeBracket` writes only 76 bytes, not 80:

```go
// Current (INCORRECT):
binary.LittleEndian.PutUint32(buf[0:4], b.LSHSalt)        // 4B
copy(buf[4:68], b.Projections[:])                         // 64B
binary.LittleEndian.PutUint32(buf[68:72], b.SubSecondUS)  // 4B
binary.LittleEndian.PutUint32(buf[72:76], b.ASICLoops)    // 4B
binary.LittleEndian.PutUint32(buf[76:80], b.GoldenSeed)  // 4B
// Total: 76 bytes — WRONG
```

**Required (per spec at section 4.3):**
- Bytes 0x48-0x4B: Semantic Metadata (POSTag, Tense, Plurality, DepHead)
- Bytes 0x50-0x53: IntentFlags (1B) + DomainSig (2B) + 1B reserved
- Bytes 0x54-0x57: Reserved for future Memory Zone

**Missing Helper Functions:**
- `EncodeSyntactic(sp SyntacticProfile) uint32`
- `DecodeSyntactic(val uint32) SyntacticProfile`
- `EncodeIntentDomain(id IntentDomain) uint32`
- `DecodeIntentDomain(val uint32) IntentDomain`

---

## 3. `internal/storage/nrv_ticker.go` — Missing Memory Zone

### GAP: No Memory Zone for Slots 6-8

The spec (section 4.4) requires the FrameTicker to maintain:

```go
// Required per spec:
type MemoryZone struct {
    HistoryXOR [3]uint32  // Slots 6, 7, 8: Rolling XOR of previous bracket hashes
    SeedXOR    uint32     // Initial seed for recursive hashing
}

type PendingBracket struct {
    // ... existing fields ...
    HistoryXOR [3]uint32  // Slot 6-8 for temporal loop
}
```

**Missing Methods:**
- `computeMemoryXOR(b *nrv.Bracket) [3]uint32` — computes rolling XOR
- `hashBracket(b *nrv.Bracket) uint32` — hashes bracket for XOR accumulation

**Impact:** The 21-pass temporal loop recurrence (Slots 6-8) is not implemented. This is critical for the ASIC's temporal hashing.

---

## 4. `internal/query/knirvql.go` — Missing Semantic Filters

### GAP: No 12-Slot-Aware Filtering

The spec (section 4.10) requires these filter keys:

| Filter Key | Spec Section | Status |
|---|---|---|
| `z3_status` | Frame metadata | ✅ Implemented |
| `avg_temp_c` | ThermoAtmosphere | ✅ Implemented |
| `drift_score` | BracketMeta.DriftScore | ✅ Implemented |
| `bracket_type` | BracketMeta.Type | ✅ Implemented |
| **`pos_tag`** | **Slot 4 (bits 0-7)** | ❌ **MISSING** |
| **`tense`** | **Slot 4 (bits 8-11)** | ❌ **MISSING** |
| **`domain`** | **Slot 10** | ❌ **MISSING** |
| **`intent_flags`** | **Slot 9** | ❌ **MISSING** |
| **`lsh_salt`** | **Slot 11** | ❌ **MISSING** |

**Impact:** ASIC-aware semantic filtering is unavailable. Users cannot filter by grammatical constraints (POS, tense) or domain (Math/Code/Prose).

---

## 5. Summary of Priority Issues

| Priority | Issue | Files Affected | 12-Slot Impact |
|---|---|---|---|
| **CRITICAL** | Missing Syntactic Registers (Slots 4-5) in Bracket | `pkg/nrv/bracket.go`, `pkg/nrv/codec.go` | Passes 8-14 fail |
| **CRITICAL** | EncodeBracket wrong size (76 vs 80 bytes) | `pkg/nrv/codec.go` | Binary format corrupted |
| **HIGH** | Missing Memory Zone (Slots 6-8) | `internal/storage/nrv_ticker.go` | Temporal loop fails |
| **MEDIUM** | Missing semantic filters | `internal/query/knirvql.go` | No ASIC-aware queries |
| **LOW** | Missing Intent/Domain fields (Slots 9-10) | `pkg/nrv/bracket.go`, `pkg/nrv/codec.go` | Intent classification unavailable |

---

## 6. Already Implemented (Verified)

| Component | Status | Notes |
|---|---|---|
| `NRVStorage.tickers` map | ✅ Done | FrameTicker instances managed |
| `FrameTicker.run()` / `flush()` | ✅ Done | 1-second tick cycle works |
| `NRVWriter.AppendFrame` new signature | ✅ Done | bracketBuf, bracketIndex, thermo, ling |
| `NRVReader.StreamBrackets` | ✅ Done | goldOnly filtering works |
| `NRVReader.decodeBrackets` P-bracket XOR | ✅ Done | Delta reconstruction works |
| `Compactor.MaybeCompact` INVALID ratio | ✅ Done | Includes invalid frames in threshold |
| FlightServer dual-stream | ✅ Done | Gold/Research ticket parsing |
| `NRVDataset` renamed methods | ✅ Done | AppendBracket, StreamBrackets, GetFrame |
| KNIRVQL bracket field parsing | ✅ Done | MEMORY.BRACKET(golden_seed) syntax |

---

## 7. Recommendations

1. **Immediate:** Add Semantic Metadata fields to `Bracket` struct and fix `EncodeBracket` to 80 bytes
2. **Immediate:** Add MemoryZone tracking to `FrameTicker`
3. **Soon:** Add semantic filter cases to `knirvql.go`
4. **Testing:** Add unit tests for the new encoding paths before integration testing
