# KNIRVBASE Phase 2 Upgrade — Audit Report (Rev 2)

**Auditor:** Claude Sonnet 4.6  
**Date:** 2026-04-08 (re-audit after worker agent pass)  
**Spec:** `go/KNIRVBASE_Upgrade.md`  
**Verdict:** ⚠️ **Improved but incomplete — several gaps closed, new quality issues introduced**

---

## Delta from Previous Audit

| Gap # | Previous Status | Current Status |
|-------|----------------|----------------|
| GAP 1 — Arrow Flight | ❌ Stub only | ✅ Arrow IPC implemented (with defects — see below) |
| GAP 2 — `Raw()`/`RawCollection()` | ❌ On `DB` directly | ⚠️ Moved to `accessor` struct, not removed |
| GAP 3 — `drift_score` filter | ❌ Wrong field (Z3 relevance) | ⚠️ Better (`avg_drift`), but semantically approximate |
| GAP 4 — `bracket_type` filter | ❌ Always false | ✅ Fixed — `Find()` now populates `brackets.types` |
| GAP 5 — Missing unit tests | ❌ 4 missing | ⚠️ 3 of 4 added; `DecodePBracket` added but weak |
| GAP 6 — `DriftSpike` skipped | ❌ `t.Skip()` | ⚠️ Body written, but float encoding is wrong |
| GAP 7 — Integration tests | ❌ All 5 missing | ⚠️ 5 added, but 3 are stubs or wrong assertions |
| GAP 7 — Benchmarks | ❌ All 4 missing | ❌ Still missing |

---

## Confirmed Resolved

### GAP 4 — `bracket_type` Filter ✅
`nrv_storage.go:Find()` now populates `brackets.types` as `[]string` of DeltaType values, and `brackets.avg_drift` as the mean drift across all brackets in the frame. The `matchesFilter` case for `bracket_type` correctly iterates the `types` slice.

### GAP 5 (partial) — Three KNIRVQL Tests ✅
`TestKNIRVQL_Z3StatusFilter`, `TestKNIRVQL_ThermoFilter`, and `TestKNIRVQL_BracketFieldQuery` are now present and test the correct parse-level behavior.

### `Bracket.Meta` field ✅
`Bracket` struct now has `Meta *BracketMeta`. The reader's `decodeBrackets` correctly populates it from the frame's `BracketIndex`, including `DriftScore`. This is an additive improvement.

---

## Open Gaps and New Defects

### GAP 1 — Arrow Flight: Implemented but with Correctness Defects — **MEDIUM**

**File:** `internal/network/flight_server.go`

The worker added real Arrow IPC serialization (`apache/arrow/go/v15` in `go.mod`, `array.RecordBuilder`, `ipc.NewWriter`/`NewReader`). The dual-stream Gold/Research routing works. This is a meaningful improvement.

However, five defects remain in the streaming implementation:

**Defect 1.1 — Schema field name typo:**
```go
{Name: "lsb_salt", Type: arrow.PrimitiveTypes.Uint32},
```
Should be `lsh_salt` (LSH = Locality-Sensitive Hashing). `lsb_salt` is a different term entirely.

**Defect 1.2 — `drift_score` and `bracket_type` hardcoded in stream:**
```go
driftBuilder.Append(0.0)       // always 0 — real value is in bracket.Meta.DriftScore
typeBuilder.Append("P")        // always "P" — should be bracket.Meta.Type (I or P)
```
Every bracket is emitted as type `"P"` with drift `0.0` regardless of its actual classification. Consumers of the Gold stream cannot distinguish I-brackets from P-brackets.

**Defect 1.3 — `frame_timestamp` set to `SubSecondUS`:**
```go
frameTSBuilder.Append(int64(bracket.SubSecondUS))
```
`SubSecondUS` is a microsecond offset within the second, not a Unix timestamp. This field is misleading as emitted.

**Defect 1.4 — `frame_id` is a counter, not the bracket/frame UUID:**
```go
frameIDBuilder.Append(frameCounter)
```
The spec schema uses `id` (string) for bracket identity. Using an `Int64` row counter loses bracket identity across stream calls.

**Defect 1.5 — Schema inconsistency between `StreamBrackets` and `BracketsToFlightData`:**
`StreamBrackets` uses a 9-field schema (includes `frame_id`, `drift_score`, `bracket_type`, `frame_timestamp`). `BracketsToFlightData` uses a 7-field schema (omits those four). A client that calls one cannot decode data from the other without knowing which schema was used.

Note: The spec required `flight.BaseFlightServer` embedding and gRPC transport. The implementation uses a custom `BracketStreamServer` interface with Arrow IPC over bytes. This is a pragmatic deviation that avoids the full gRPC dependency; it is acceptable as a design choice if documented, but it means the server does not speak the Arrow Flight wire protocol.

---

### GAP 2 — `Raw()`/`RawCollection()` Moved, Not Removed — **LOW**

**File:** `pkg/knirvbase/knirvbase.go:111–127`

The spec says: *"Remove `Raw()` and `RawCollection()` from `DB`."*

The worker moved them to an `accessor` struct behind `db.Access()`:
```go
func (d *DB) Access() *accessor { ... }
type accessor struct { ... }
func (a *accessor) Raw() *db.DistributedDatabase { ... }
func (a *accessor) RawCollection(name string) *coll.DistributedCollection { ... }
```

These methods still exist and still expose internal types — they are just one level deeper. The spirit of the spec was to eliminate these escape hatches. This is not a full fix.

---

### GAP 3 — `drift_score` Filter: Semantically Approximate — **LOW**

**File:** `internal/query/knirvql.go:514–522` and `internal/storage/nrv_storage.go`

The filter now resolves to `payload["brackets"]["avg_drift"]`, which is populated as the mean of all bracket `DriftScore` values for the frame. This is a frame-level aggregate, not per-bracket filtering. The spec implies per-bracket filtering (`WHERE drift_score > 0.1` applied to individual bracket metadata). This is a reasonable approximation but may produce unintuitive results: a frame with one extreme-drift bracket and many zero-drift brackets could pass or fail the filter based on the average.

This gap is lower priority since the spec's exact semantics here are ambiguous — the `matchesFilter` function operates on documents (frames), not individual brackets.

---

### GAP 5 (remaining) — `TestNRVReader_DecodePBracket` Does Not Test XOR Reconstruction — **MEDIUM**

**File:** `internal/storage/nrv_reader_test.go:143–206`

The test was added but does not verify the core requirement: *"P-bracket projections reconstructed correctly from anchor."*

The test writes a P-bracket with the same raw projections as the anchor (no XOR applied before encoding), then reads it back and only checks `Meta.Type` and `Meta.DriftScore`. The actual projection bytes after delta decoding are never compared.

The correct test would:
1. Create an anchor bracket with projections `A`
2. Create a P-bracket delta = `XORProjections(B, A)` for some target `B`
3. Write both to the file
4. Read back via `GetFrame`
5. Assert `brackets[1].Projections == B` (i.e., reconstruction yielded the original absolute values)

As written, the test would pass even if `decodeBrackets` completely skipped P-bracket reconstruction.

---

### GAP 6 — `TestFrameTicker_DriftSpike` Has Broken Float Encoding — **MEDIUM**

**File:** `internal/storage/nrv_ticker_test.go:100–159`

The test is no longer skipped, but the float32 encoding uses a custom bit-manipulation that is not IEEE 754:
```go
bits := int32(val * 256)
bytes[0] = byte(bits & 0xFF)
// ...
```

The correct approach is `math.Float32bits(val)`. Using `int32(val * 256)` writes a fixed-point representation; the resulting bytes will be interpreted by `euclideanDrift` as IEEE 754 floats, yielding completely wrong values.

Additionally, the test only asserts `require.GreaterOrEqual(t, len(writer.registry.Frames), 1)`, which passes trivially even if the drift spike is never detected (frames are created by the ticker regardless of drift). The test should assert that `b2` was classified as `DeltaTypeI` in the BracketIndex, since it was the first bracket after a new tick — or alternatively set the drift spike in a single-tick scenario and verify the I-bracket count exceeds what the 50-bracket interval would produce alone.

---

### GAP 7 — Integration Tests: Three Are Stubs or Test Wrong Things — **HIGH**

**File:** `internal/benchmarks/integration_test.go`

**`TestEndToEnd_ASICPipeline` is a pure placeholder:**
```go
func TestEndToEnd_ASICPipeline(t *testing.T) {
    require.True(t, true, "ASIC pipeline test placeholder - requires full system integration")
}
```
This test asserts `true == true`. It validates nothing. The spec requires: *"Append 1000 brackets over 3 seconds → verify 3 FrameEntries in registry with correct counts, Z3 status, and Dilithium-3 signatures."*

**`TestEndToEnd_CompactionPreservesGold` never runs compaction:**
The test writes 5 "valid" and 3 "invalid" frames, then asserts the initial frame count ≥ 8. It never marks frames INVALID, never calls `MaybeCompact`, never reads the post-compaction file to verify INVALID frames were removed. The test name is entirely misleading.

**`TestEndToEnd_CrashRecovery` tests normal close, not a crash:**
The test appends one frame, closes the writer normally, and verifies 1 frame is in the registry. No crash is simulated, no WAL recovery is exercised. The spec requires: *"Kill writer mid-flush → WAL recovery on re-open → no partial frames in registry."* This test would pass even if WAL recovery was completely deleted.

**`TestEndToEnd_FlightGoldStream` and `TestEndToEnd_FlightResearchStream`** are genuine functional tests that exercise the Flight server correctly. These pass.

---

### GAP 7 — Phase 2 Benchmarks Still Missing — **MEDIUM**

None of the four required benchmarks from §6.3 were added:
- `BenchmarkAppendBracket` — concurrent caller throughput
- `BenchmarkFlush_1000Brackets` — flush latency
- `BenchmarkStreamBrackets_Gold` — Gold stream MB/s
- `BenchmarkDecodePBracket` — XOR reconstruction overhead

---

## Summary Table (Rev 2)

| Category | Spec Items | Fully Correct | Partial/Defective | Missing |
|---|---|---|---|---|
| Core data types | 7 | 7 | 0 | 0 |
| Storage components | 5 | 5 | 0 | 0 |
| Arrow Flight server | 1 | 0 | 1 | 0 |
| KNIRVQL parsing | 1 | 1 | 0 | 0 |
| KNIRVQL filter logic | 4 filters | 3 | 1 (`drift_score` approx) | 0 |
| Public API cleanup | 1 | 0 | 1 (moved, not removed) | 0 |
| Unit tests | 11 | 9 | 2 (DriftSpike, DecodePBracket) | 0 |
| Integration tests | 5 | 2 | 3 (placeholders/wrong assertions) | 0 |
| Benchmarks | 4 | 0 | 0 | 4 |
| **Total** | **~39** | **~27** | **~8** | **~4** |

---

## Prioritized Remaining Work

### Must Fix Before Shipping

1. **`TestEndToEnd_ASICPipeline`** — Replace placeholder with real test: use `NRVDataset.AppendBracket` in a loop over ~3 seconds, verify the resulting FrameEntries, bracket counts, and signatures.

2. **`TestEndToEnd_CompactionPreservesGold`** — Mark some frames INVALID, call `MaybeCompact`, wait for completion, open a new reader, assert only VALID/live frames remain.

3. **`TestEndToEnd_CrashRecovery`** — Simulate crash: write partial data directly to the file after a WAL `Begin` but without `Commit`, then re-open via `NewNRVWriter` and verify the registry contains only the pre-crash frames.

4. **`TestNRVReader_DecodePBracket`** — Rewrite to use `XORProjections` when preparing the P-bracket buffer, then assert the reconstructed projections match the original target values.

5. **Arrow Flight schema defects** (1.1–1.5): Fix `lsb_salt` → `lsh_salt`, populate real `DriftScore` and `Type` per bracket, fix `frame_timestamp`, make schemas consistent between `StreamBrackets` and `BracketsToFlightData`.

### Should Fix

6. **`TestFrameTicker_DriftSpike`** — Replace manual bit-manipulation with `math.Float32bits`. Add assertion that the second bracket resulted in a new I-bracket in the BracketIndex.

7. **Phase 2 benchmarks** — Add `BenchmarkAppendBracket`, `BenchmarkFlush_1000Brackets`, `BenchmarkStreamBrackets_Gold`, `BenchmarkDecodePBracket`.

### Low Priority

8. **`Raw()`/`RawCollection()` on `accessor`** — Remove entirely if the spec intent is to eliminate these escape hatches; or document the `Access()` pattern as the intentional replacement.
