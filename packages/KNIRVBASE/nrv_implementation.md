# KNIRVBASE .nrv (Noted Resolution Vector) Implementation Plan

This document outlines the remaining tasks and considerations for fully implementing the `.nrv` (Noted Resolution Vector) format within the KNIRVBASE Go project. The goal is to transition KNIRVBASE from its current discrete file/blob storage model to a unified, single-file `.nrv` container optimized for high-fidelity ML data engineering, streaming, and formal validation.

---

## 1. Introduction

The `.nrv` format represents a significant architectural evolution for KNIRVBASE, consolidating datasets into a single, versioned, cryptographically secure, and memory-mappable file. This plan details the implementation steps required to realize our finalized `.nrv` specification, ensuring seamless integration with existing PQC, CRDT, and networking protocols.

---

## 2. .nrv File Format Specification

This specification defines the **.nrv (Noted Resolution Vector)** file format, a high-fidelity multimodal container for KNIRVBASE. It is designed to bridge the gap between high-speed ASIC computational output and linguistic-temporal reasoning for the HERO model.

### 2.1 Terminology

> **IMPORTANT — Terminology Change:** The term **"Frame"** has been redefined. What was previously called a "Frame" is now called a **"Bracket"**. The term **"Frame"** now refers specifically to the 1-second temporal unit described below.

| Term | Definition |
| :--- | :--- |
| **Frame** | A **1-second snapshot** of the resolution process. Serves as a linguistic container (syllable/word) and holds shared environmental metadata. This is the temporal unit. |
| **Bracket** | A fixed **80-byte binary record** representing one completed ASIC task (LSH projections + solved Golden Seed). This is the computational unit. |

### 2.2 File Architecture

The `.nrv` format uses a binary "envelope" consisting of a fixed-width global header followed by two distinct data chunks.

#### Global Header (12 Bytes)

| Offset | Field | Type | Value / Description |
| :--- | :--- | :--- | :--- |
| `0x00` | **Magic** | `char[4]` | `0x4E525621` (ASCII: `NRV!`) |
| `0x04` | **Version** | `uint32` | `0x00000001` (Spec v1.0) |
| `0x08` | **Length** | `uint32` | Total file size in bytes. |

### 2.3 Chunk 0: Resolution Registry (JSON)

The Registry acts as the index for the file, providing context and validation for the binary data in Chunk 1. Each 1-second **Frame** is indexed with the following metadata:

- **Linguistic Mapping**: The target token and unit (syllable/word) represented.
- **Thermodynamic Atmosphere**: Snapshot of hardware state during that second:
  - `avg_temp_c`: Mean chip temperature.
  - `peak_volt_v`: Highest voltage detected.
  - `clock_mhz`: Operating frequency.
- **Z3 Validation**:
  - `status`: `VALID` or `INVALID` based on formal logic gates.
  - `relevance`: Score correlating this frame's tokens to the previous/next transition.
- **Binary Map**:
  - `count`: Number of 80-byte Brackets found in this 1-second window.
  - `offset`: Byte-coordinate in Chunk 1.
  - `length`: Total bytes for this frame's bracket array (`count × 80`).
- **PQC Signature**: Each frame is signed with **Dilithium-3**.
- **Vector Clocks**: CRDT metadata for multi-node synchronization.

### 2.4 Chunk 1: Multi-Modal Buffer (Binary)

A contiguous stream of **80-byte Brackets**, aligned to 8-byte boundaries for zero-copy Apache Flight streaming and memory-mapping.

#### Bracket Binary Layout (80 Bytes)

| Section | Size | Description |
| :--- | :--- | :--- |
| **LSH Salt** | 4B | Version field used as the forest seed. |
| **Projections A–D** | 32B | First half of the 128-bit LSH projections. |
| **Projections E–H** | 32B | Second half of the 128-bit LSH projections. |
| **Metadata** | 8B | Sub-second timestamp and ASIC Loop Count (1–21). |
| **Golden Seed** | 4B | The solved nonce (the "result"). |

### 2.5 Compression & Maintenance

#### Delta Compression

To optimize storage, `.nrv` supports video-style "Bracket Deltas":

- **I-Brackets (Intra)**: Stores absolute 80-byte values. Occurs every N brackets or upon a significant drift spike.
- **P-Brackets (Predicted)**: Stores the **XOR-diff** of the LSH projections against the last I-Bracket.
- **Drift Score**: Calculated as the Euclidean distance from the anchor; used to trigger new I-Brackets.

The Registry tracks each bracket's `type` (`I` or `P`), `anchor_id`, and `encoding` per modality.

#### Automated Compaction

- **Trigger**: Fires automatically when 20% of the file is consumed by tombstones or frames marked `INVALID` by the Z3 gate.
- **Mechanism**: A background **Rewrite-and-Swap** creates a `.nrv.tmp` file, copies only live/valid brackets, rebuilds the Registry with new offsets, signs the result with Dilithium-3, and atomically replaces the original — maintaining service via `mmap`.

### 2.6 Validation Flow

1. **Input**: 80-byte tasks are sent to the ASIC via the `0x52` protocol.
2. **ASIC**: Performs a 21-loop pass per bracket to resolve the Golden Seed.
3. **Frame Closure**: At the 1-second mark, KNIRVBASE aggregates found seeds, attaches thermodynamic context, and runs Z3 validation.
4. **Persistence**: The validated frame is committed to the `.nrv` dataset.

---

## 3. Current State Analysis

The existing Go codebase provides essential building blocks:
- **Foundations:** Core PQC (Kyber-768, Dilithium-3), CRDT vector clocks, P2P networking, and a functional `FileStorage` system are in place.
- **Initial NRV Integration:** The `NewNRV` constructor, `stor.NRVStorage`, `NRVDataset`, and references to `nrv.Bracket` (formerly `nrv.Frame`) suggest preliminary work on NRV support.
- **Storage Discrepancy:** The current `FileStorage` separates metadata (`.json` files) from blobs (local, unsynchronized files). This contradicts the `.nrv` single-container goal.
- **Missing Features:** Full implementation of the `.nrv` spec (header, registry, modalities, alignment, delta compression, tombstones, auto-compaction, robust concurrency controls, and Arrow Flight integration) is required.
- **Public API:** The `knirvbase.go` file exposes internal details (`Raw()` methods) that should be abstracted.

---

## 4. Implementation Phases

### Phase 1: Core .nrv Container Implementation (`internal/storage`, `pkg/nrv`)

**Goal:** Establish the fundamental structure and storage mechanisms for the `.nrv` file format.

**Task 1.1: Define and Implement .nrv Spec**
- **Header:** Implement the 12-byte header per §2.2.
- **Resolution Registry (Chunk 0):**
  - Implement `Global Metrics` block (`dataset_id`, `version_clock`, `thermo_correlation_score`, `pqc_fingerprint`).
  - Implement `Frame Index` (array of Frame entries) per §2.3.
  - Each Frame entry maps to its Bracket array via `offset`, `count`, and `length`.
  - Integrate Tombstone flagging for frames marked `INVALID`.
- **Multi-Modal Buffer (Chunk 1):**
  - Enforce strict 8-byte alignment for all Bracket data.
  - Implement the full 80-byte Bracket layout per §2.4.
- **`pkg/nrv` Public Definitions:** Public Go structs for `Header`, `Registry`, `Frame`, `Bracket`, `ThermoData`, etc.

**Task 1.2: Refactor `stor.NRVStorage`**
- Replace `FileStorage` with `NRVStorage` for managing `.nrv` files.
- Implement the 1-second **Ticker** in `internal/storage`: buffers ASIC-generated Brackets and flushes them to the `.nrv` file alongside thermodynamic and Z3 metadata at each frame boundary.
- Implement atomic append operations:
  - Write new Bracket data to EOF.
  - Update the Registry (Chunk 0) in its pre-allocated space.
  - Atomically update the 12-byte global header's `Total Length`.
- Utilize `os.OpenFile` with `O_APPEND`/`O_RDWR` flags and `os.WriteAt`.
- Integrate PQC key management (`pqc.PQCKeyPair`) for per-frame signing with Dilithium-3.

### Phase 2: Advanced .nrv Features & Concurrency (`internal/storage`, `internal/resolver`, `internal/query`)

**Goal:** Implement robust data management, compression, and concurrency control.

**Task 2.1: Implement Delta Compression**
- Distinguish between I-Brackets (anchors) and P-Brackets (XOR-deltas) per §2.5.
- Calculate and store `drift_score` (Euclidean distance from anchor) for P-Brackets.
- Update Registry to track Bracket `type`, `anchor_id`, and `encoding`.
- Implement I-Bracket frequency logic (fixed interval of 50, or dynamic based on drift threshold).

**Task 2.2: Implement Tombstone System & Automated Compaction**
- Add a `tombstone` flag to Frame Registry entries.
- Update all read operations (`Find`, `FindAll`, `StreamBrackets`, Flight Producer) to filter tombstoned frames and `INVALID` Z3 frames.
- Implement the background Compactor per §2.5: trigger at 20% tombstone/invalid ratio, Rewrite-and-Swap with Dilithium-3 re-signing.

**Task 2.3: Implement Concurrency Mitigation**
- Integrate Advisory File Locking (`flock`) in `NRVStorage`.
- Implement Shadow Header Updates and Atomic Header Swap for registry updates.
- Integrate a Write-Ahead Log (WAL) for crash recovery.
- (Optional) Single-Writer Orchestrator via `internal/database` for strict linearizability.

### Phase 3: Streaming & Integration (`internal/network`, `internal/query`, `pkg/knirvbase`)

**Goal:** Enable high-performance streaming via Arrow Flight and extend KNIRVQL for the `.nrv` format.

**Task 3.1: Implement Arrow Flight Producer**
- Develop a Flight Server in `internal/network` per §2.6 implementation strategy.
- Parse the `.nrv` Registry and map Frame byte-ranges for zero-copy access.
- Implement the Dual-Stream Policy:
  - **"Gold" Stream**: Serves only Brackets within frames where `z3_verify == VALID`.
  - **"Research" Stream**: Serves all Brackets, including unverified ones.
- Enforce Z3 `relevance` scoring on the Gold stream.

**Task 3.2: Extend KNIRVQL**
- Add syntax for querying specific bracket fields (e.g., `GET MEMORY.BRACKET(golden_seed) WHERE ...`).
- Add syntax for filtering by frame validation status, `drift_score`, and bracket type (`I`, `P`).
- Add syntax for filtering by thermodynamic range (`WHERE avg_temp_c < 85`).
- Update `internal/query` to parse and execute these new `.nrv`-specific queries.

**Task 3.3: Refine Public API (`go/pkg/knirvbase`)**
- Remove `Raw()` and `RawCollection()` methods from the `DB` struct.
- Ensure `Collection` interface supports operations for `.nrv` datasets.
- Update `knirvbase.go` to use `stor.NRVStorage` exclusively when `NewNRV` constructor is used.
- Finalize the `NRVDataset` API (`AppendBracket`, `StreamBrackets`, `GetFrame`) for user interaction.

### Phase 4: Testing & Validation

**Goal:** Rigorously test all implemented features for correctness, performance, and security.

**Task 4.1: Unit and Integration Tests**
- Unit tests for `internal/storage` (NRV file operations, concurrency, compaction, delta encoding, tombstone handling, 1-second ticker flush).
- Integration tests for `NRVDataset` methods and the `NewNRV` constructor.
- Test KNIRVQL extensions against `.nrv` files with various frame/bracket configurations.
- Tests for the Arrow Flight Producer, including Gold/Research stream enforcement.

**Task 4.2: Performance Benchmarks**
- Benchmark `.nrv` file append performance at 80 bytes/bracket throughput.
- Benchmark frame flush latency (1-second ticker accuracy under load).
- Benchmark read performance for direct bracket access vs. streaming.
- Benchmark compaction efficiency and trigger thresholds.
- Benchmark Arrow Flight streaming throughput and latency.

**Task 4.3: Security Validation**
- Verify Dilithium-3 signature integrity for frame commits and compaction rewrites.
- Test concurrent write scenarios to ensure no data corruption.
- Validate Gold stream correctly enforces Z3 `VALID` status.

### Phase 5: Documentation & Finalization

**Goal:** Document the new `.nrv` format and its integration into KNIRVBASE.

**Task 5.1: Update `go/README.md`**
- Revise the overview to reflect the `.nrv` single-file container architecture.
- Update "Storage Layout" to describe `.nrv` files (Frames + Brackets) instead of JSON/blobs.
- Provide KNIRVQL examples relevant to `.nrv` features (bracket fields, Z3 status, thermo filters).
- Remove outdated information about local blobs and unsynchronized data.

**Task 5.2: Update Internal Documentation**
- Ensure Go doc comments align with the new Frame/Bracket terminology throughout `pkg/nrv` and `internal/storage`.

---

## 5. Backlog (V2 Development)

The following features are deferred to a later development cycle:

- **Reasoning Ledger:** Immutable agent-access ledger (e.g., `memvid`) in a dedicated `agent_context` field on Frame registry entries.
- **Cross-Dataset Querying:** Multi-file KNIRVQL joins across `.nrv` files.
- **GPU-Native Loaders:** Rust-based Vulkan/CUDA kernels for direct `.nrv` Chunk 1 to VRAM transfers.
- **Advanced Compression:** Zstd or Brotli for Z3 proof traces or compressed Arrow data alongside bracket deltas.
- **Further KNIRVQL Enhancements:** Advanced analytics queries on per-frame thermodynamic metrics.
