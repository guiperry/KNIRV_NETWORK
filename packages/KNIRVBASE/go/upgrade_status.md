# KNIRVBASE Phase 2 Upgrade — Audit Report

**Auditor:** Claude Sonnet 4.6  
**Date:** 2026-04-08  
**Spec:** `go/KNIRVBASE_Upgrade.md`  
**Verdict:** ⚠️ **Partial — 8 of 15 spec items are complete; 7 gaps remain**

---

## Status Overview

| Section | Component | Status |
|---------|-----------|--------|
| §4.1 | `pkg/nrv/bracket.go` | ✅ Complete |
| §4.2 | `pkg/nrv/frame.go` | ✅ Complete |
| §4.3 | `pkg/nrv/codec.go` | ✅ Complete |
| §4.4 | `internal/storage/nrv_ticker.go` | ✅ Complete |
| §4.5 | `internal/storage/nrv_writer.go` | ✅ Complete |
| §4.6 | `internal/storage/nrv_reader.go` | ✅ Complete |
| §4.7 | `internal/storage/nrv_compactor.go` | ✅ Complete |
| §4.8 | `internal/storage/nrv_storage.go` | ✅ Complete |
| §4.9 | `internal/network/flight_server.go` | ❌ Stub only — Arrow Flight not implemented |
| §4.10 | `internal/query/knirvql.go` — parsing | ✅ Complete |
| §4.10 | `internal/query/knirvql.go` — filter logic | ❌ Two filter keys broken |
| §4.11 | `pkg/knirvbase/knirvbase.go` | ⚠️ Mostly complete — `Raw()`/`RawCollection()` not removed |
| §6.1 | Unit tests | ⚠️ 7 of 11 present; 4 missing; 1 skipped |
| §6.2 | Integration tests | ❌ All 5 missing |
| §6.3 | Benchmarks | ❌ All 4 missing |

---

## Confirmed Complete

### `pkg/nrv/bracket.go` (§4.1)
All required types present and correctly defined: `DeltaType`, `DeltaTypeI`/`P`, `BracketSize = 80`, `Bracket`, `BracketMeta`, `LinguisticMapping`, `ThermoAtmosphere`, `Z3Result`, `BracketBinaryMap`. No deviations from spec.

### `pkg/nrv/frame.go` (§4.2)
`FrameEntry` correctly updated with `Linguistic`, `Thermo`, `Z3`, `Brackets`, `BracketIndex` fields. `GlobalMetrics` correctly updated with `TotalBracketCount`, `ValidFrameCount`, `InvalidFrameCount`, `CompactedAt`. `ThermoData` retained for legacy compatibility per spec instruction. `Registry` schema matches.

### `pkg/nrv/codec.go` (§4.3)
`EncodeBracket`, `DecodeBracket`, `XORProjections`, `ApplyProjectionDelta` all correctly implemented with proper byte offsets (LSHSalt 0–3, Projections 4–67, SubSecondUS 68–71, ASICLoops 72–75, GoldenSeed 76–79). `EncodeFrame` retained for backward compatibility per spec §4.3 deprecation note.

### `internal/storage/nrv_ticker.go` (§4.4)
Fully implemented — not a stub. `euclideanDrift` interprets 64 bytes as 16 × float32 and computes real Euclidean distance. `aggregateThermo` correctly averages all three fields. I/P bracket logic, drift-spike detection, XOR application, and stop-with-flush are all present. The `AppendBracket` signature adds `context.Context` and returns `error` beyond the spec's minimum — this is an acceptable improvement.

### `internal/storage/nrv_writer.go` (§4.5)
New `AppendFrame(frameID, bracketBuf, bracketIndex, thermo, ling)` signature is implemented. WAL begin/commit, flock acquire/release, Dilithium-3 signing, Z3 placeholder, registry update, metric increment, and `saveRegistry` are all present and in the correct order.

### `internal/storage/nrv_reader.go` (§4.6)
`StreamBrackets(goldOnly bool)` correctly skips tombstoned and (when `goldOnly`) INVALID frames. `GetFrame` returns `*nrv.FrameEntry` and `[]*nrv.Bracket`. `decodeBrackets` correctly applies XOR reconstruction for P-brackets using the anchor map. `VerifyFrame` correctly verifies the bracket buffer (not the old multimodal blob).

### `internal/storage/nrv_compactor.go` (§4.7)
`MaybeCompact` correctly computes `deadFrames = TombstoneCount + GlobalMetrics.InvalidFrameCount`. `compact()` correctly skips both tombstoned and INVALID frames. `CompactedAt` is set on the output registry. Rename-and-swap pattern is implemented correctly.

### `internal/storage/nrv_storage.go` (§4.8)
`tickers map[string]*FrameTicker` added. `getOrCreateTicker` calls `getOrCreateWriterLocked` then `NewFrameTicker(w, time.Second)`. `Close()` stops all tickers before closing writers/readers. `GetReader`, `SetLinguistic`, `AppendBracketDirect`, `StreamBrackets`, and `GetFrame` are all implemented. `Insert` routes through `AppendBracketDirect`.

### `internal/query/knirvql.go` — Parsing (§4.10)
`QueryGetBracketField` added to the `QueryType` enum at the correct ordinal position. `MEMORY.BRACKET(` prefix detection is implemented in `parseGet` and correctly populates `BracketField`. `BracketField` is present on the `Query` struct. `QueryGetBracketField` is handled in `Execute` (routes to `executeGet`). Filter key parsing for `z3_status`, `avg_temp_c`, `drift_score`, and `bracket_type` is wired into `matchesFilter`.

---

## Gaps and Defects

### GAP 1 — Arrow Flight Server is a Channel Wrapper, Not Arrow Flight (§4.9) — **HIGH**

**File:** `internal/network/flight_server.go`

The spec required a real Apache Arrow Flight gRPC server:
- Embed `flight.BaseFlightServer`
- Add `github.com/apache/arrow/go/v15` to `go.mod`
- Implement `DoGet(ticket *flight.Ticket, stream flight.FlightService_DoGetServer) error`
- Build Arrow `RecordBatch` objects from bracket data using Arrow builders
- Define `bracketArrowSchema()` with proper Arrow field types
- Stream batches of 1024 via `ipc.Writer`

**What exists:**
```go
type FlightServer struct {
    storage *stor.NRVStorage
}
func (s *FlightServer) DoGet(ticket string) (<-chan *nrv.Bracket, error) {
    return s.StreamBrackets(context.Background(), ticket)
}
```

This is a plain Go channel wrapper. There is no Arrow dependency in `go.mod`, no gRPC transport, no Arrow record batches, and `DoGet` has the wrong signature entirely. The Gold/Research dual-stream ticket routing logic is correct, but nothing else from the spec is present. **This is an incomplete implementation, not a working Arrow Flight server.**

---

### GAP 2 — `Raw()` and `RawCollection()` Not Removed (§4.11) — **LOW**

**File:** `pkg/knirvbase/knirvbase.go:73–77`

The spec states: *"Remove `Raw()` and `RawCollection()` from `DB`. These exposed internal types and are no longer needed."*

Both methods are still present:
```go
func (d *DB) Raw() *db.DistributedDatabase { return d.db }
func (d *DB) RawCollection(name string) *coll.DistributedCollection {
    return d.db.Collection(name, d.store)
}
```

This is a minor API hygiene gap; these methods expose internal types in violation of the spec's encapsulation intent.

---

### GAP 3 — `drift_score` Filter Maps to Wrong Field (§4.10) — **MEDIUM**

**File:** `internal/query/knirvql.go:513–519`

The spec defines `drift_score` as a per-bracket concept stored in `BracketMeta.DriftScore`. The implementation maps it to the frame-level Z3 relevance score:

```go
case "drift_score":
    if z3, ok := payload["z3"].(map[string]interface{}); ok {
        cmp := compareValues(z3["relevance"], filter.Value)
        // ...
    }
```

`z3["relevance"]` is the frame-level Z3 relevance, not the bracket-level Euclidean drift. These are semantically distinct values. A query like `WHERE drift_score > 0.1` would silently filter by the wrong field. The correct implementation would need to inspect `BracketMeta.DriftScore` values within the frame's `bracket_index`.

---

### GAP 4 — `bracket_type` Filter References Non-Existent Field Path (§4.10) — **MEDIUM**

**File:** `internal/query/knirvql.go:519–523`

```go
case "bracket_type":
    if brackets, ok := payload["brackets"].(map[string]interface{}); ok {
        return fmt.Sprintf("%v", brackets["type"]) == fmt.Sprintf("%v", filter.Value)
    }
```

`Find()` in `nrv_storage.go` returns the brackets field as:
```go
"brackets": map[string]interface{}{
    "count": len(brackets),
}
```

There is no `"type"` key in this map. This filter will **always return false** for any input. A `WHERE bracket_type = I` query silently returns no results.

---

### GAP 5 — Three Required Unit Tests Missing (§6.1) — **HIGH**

**File:** `internal/query/knirvql_test.go`

The following tests named in §6.1 do not exist anywhere in the test suite:

| Required Test | Assertion |
|---|---|
| `TestKNIRVQL_Z3StatusFilter` | `WHERE z3_status = VALID` returns only valid-frame brackets |
| `TestKNIRVQL_ThermoFilter` | `WHERE avg_temp_c < 85` filters frames correctly |
| `TestKNIRVQL_BracketFieldQuery` | `GET MEMORY.BRACKET(golden_seed)` parses to `QueryGetBracketField` |

**File:** `internal/storage/nrv_reader_test.go`

| Required Test | Assertion |
|---|---|
| `TestNRVReader_DecodePBracket` | P-bracket projections reconstructed correctly from anchor |

Note: `TestKNIRVQL_BracketFieldQuery` is particularly easy to add and would immediately catch the parse path. `TestNRVReader_DecodePBracket` is important for verifying the XOR reconstruction chain end-to-end through the reader.

---

### GAP 6 — `TestFrameTicker_DriftSpike` Skipped (§6.1) — **LOW**

**File:** `internal/storage/nrv_ticker_test.go:101–103`

```go
func TestFrameTicker_DriftSpike(t *testing.T) {
    t.Skip("Drift spike detection requires precise timing - covered by unit tests")
}
```

The test body is empty and unconditionally skipped. The spec requires this test to verify that a drift > 0.25 forces an I-bracket outside of the fixed 50-bracket interval. The `euclideanDrift` function is fully implemented, so this test can be written by populating two `Bracket.Projections` arrays whose float32 distance exceeds `driftThreshold`.

---

### GAP 7 — All Integration Tests and Benchmarks Missing (§6.2, §6.3) — **HIGH**

Neither integration tests nor Phase 2 benchmarks exist anywhere in the repository:

**Missing integration tests (§6.2):**
- `TestEndToEnd_ASICPipeline` — 1000 brackets over 3 seconds → 3 FrameEntries
- `TestEndToEnd_FlightGoldStream` — Flight gold ticket → VALID-only brackets
- `TestEndToEnd_FlightResearchStream` — Flight research ticket → all brackets
- `TestEndToEnd_CompactionPreservesGold` — INVALID frames purged by compaction
- `TestEndToEnd_CrashRecovery` — WAL recovery after mid-flush kill

**Missing benchmarks (§6.3):**
- `BenchmarkAppendBracket` — concurrent callers throughput
- `BenchmarkFlush_1000Brackets` — flush latency
- `BenchmarkStreamBrackets_Gold` — Gold stream MB/s
- `BenchmarkDecodePBracket` — XOR reconstruction overhead

The existing `internal/benchmarks/benchmarks_test.go` covers Phase 1 SLA benchmarks (credential insert, PQC crypto, auth workflow) and is unrelated to Phase 2 bracket operations.

---

## Minor Observations (Non-Blocking)

- **`TestCompactor_InvalidFrameRatio`** (§6.1): The spec requires this test to verify that INVALID frames alone push the compaction ratio ≥ 20%. The closest existing test (`TestCompactorMaybeCompactAboveThreshold`) uses only tombstoned frames to trigger compaction. The underlying code correctly adds `InvalidFrameCount` to `deadFrames`, but there is no test that marks frames INVALID and verifies the ratio triggers compaction via that path.

- **`compareValues` is lexicographic, not numeric**: `compareValues` in `knirvql.go` does string comparison (`aStr < bStr`), meaning `WHERE avg_temp_c < 85` would compare `"80.5"` < `"85"` as strings. This works for simple integer-like values but will produce incorrect results for floats like `89.2` vs `85` (string `"89.2"` < `"85"` is false, correct) or `"9.5"` vs `"85"` (`"9.5"` > `"85"` string-wise, incorrect numerically). The spec does not prescribe the implementation, but a numeric comparison would be more correct for thermodynamic filters.

- **`nrv_storage.go` — `Update` has broken semantics**: `Update` calls `Delete` first, which tombstones the frame, then calls `Find`, which uses the reader cache and will return stale data because the reader is not refreshed after a write. This is pre-existing behavior, not introduced by Phase 2, but it affects the correctness of any write path through `Update`.

---

## Summary Table

| Category | Spec Items | Implemented | Gap |
|---|---|---|---|
| Core data types | 7 | 7 | 0 |
| Storage components | 5 | 5 | 0 |
| Arrow Flight server | 1 | 0 | 1 (stub only) |
| KNIRVQL parsing | 1 | 1 | 0 |
| KNIRVQL filter logic | 4 filters | 2 correct | 2 broken |
| Public API cleanup | 1 | 0 | 1 (`Raw`/`RawCollection`) |
| Unit tests | 11 | 7 | 4 missing, 1 skipped |
| Integration tests | 5 | 0 | 5 |
| Benchmarks | 4 | 0 | 4 |
| **Total** | **~39** | **~22** | **~17** |

---

## Recommended Priority Order for Remaining Work

1. **Arrow Flight server** (§4.9) — Add `apache/arrow/go/v15` dependency, replace `FlightServer` with real `flight.BaseFlightServer` embedding, implement Arrow schema and batch writing.
2. **Integration tests** (§6.2) — `TestEndToEnd_ASICPipeline` and `TestEndToEnd_CrashRecovery` provide the most safety coverage for the core pipeline.
3. **Fix `drift_score` filter** (§4.10) — Map to actual `BracketMeta.DriftScore` values, not Z3 relevance.
4. **Fix `bracket_type` filter** (§4.10) — Either add `type` to the brackets map in `Find()` or remove the filter until bracket-level querying is wired end-to-end.
5. **Missing unit tests** (§6.1) — `TestNRVReader_DecodePBracket`, `TestKNIRVQL_Z3StatusFilter`, `TestKNIRVQL_ThermoFilter`, `TestKNIRVQL_BracketFieldQuery`.
6. **Unskip `TestFrameTicker_DriftSpike`** — Implement the test body using known projection arrays with drift > 0.25.
7. **Phase 2 benchmarks** (§6.3) — Add to `internal/benchmarks/`.
8. **Remove `Raw()`/`RawCollection()`** (§4.11) — Minor cleanup, low risk.
