# KNIRVBASE .nrv Master Specification (v2.2)

## 1. Executive Summary & Scope

This document serves as the absolute technical source of truth for the platform, bridging the physical requirements of the **Bitmain S3 Antminer (BM1382)** hardware with the logical requirements of the **HERO Machine Learning Model** and **Z3 Formal Verification**.

The `.nrv` format is a tiered, high-density binary container optimized for "Noted Resolutions." It is designed to facilitate the transition from raw semantic input to formal logical truth through ASIC-accelerated hashing. The format prioritizes:

1.  **Hardware Alignment**: Direct 80-byte header compatibility with BM1382 chips.
2.  **Formal Verifiability**: Dedicated space for Z3 logic gate results.
3.  **Post-Quantum Security**: Tiered signing using Dilithium-3 and Kyber-768.
4.  **Temporal Integrity**: Microsecond-accurate frame synchronization and delta compression.

---

## 2. The Three-Tier Data Hierarchy
The dataset is structured into three nested tiers of visibility and scope.

### Tier 1: The Global Vault (Dataset Scope)
* **Purpose**: Global identity, security manifest, and statistical aggregate.
* **Storage**: Contained in the JSON "Chunk 0" of the file.
* **Key Data**: PQC signatures, dataset ID, global LSH anchors, and compaction thresholds.

### Tier 2: The Temporal Frame (Environmental Scope)
* **Purpose**: Capturing the physical "moment" of the resolution.
* **Storage**: Aggregates one or more Brackets under a single physical context.
* **Key Data**: Thermodynamics (Temp, Volt, Fan), OS-level timestamps, and the **Drift Score** relative to the previous frame.

### Tier 3: The Data Bracket (Resolution Scope)
* **Purpose**: The atomic unit of semantic resolution.
* **Storage**: The 80-byte ASIC-ready "Bullet" + associated binary modalities (Arrow Vectors).
* **Key Data**: The 12-Slot payload, Z3 status, Golden Seed, and Recursive Context Memory.



---

## 3. Tier 3: Binary Specification (The "Bullet")
Every Bracket contains a mandatory **80-byte binary header** formatted exactly for the Bitmain S3 Antminer's BM1382 hashing core.

### 3.1 The 80-Byte Memory Map
| Byte Offset | Hex Range | Slot Mapping | ASIC Logic Pass | Purpose |
| :--- | :--- | :--- | :--- | :--- |
| **00-31** | `0x00-0x1F` | **Slots 0-3** | 1–7 | Semantic Compass (LSH Projections) |
| **32-35** | `0x20-0x23` | **N/A** | 1–7 | SubSecondUS (Temporal Entropy) |
| **36** | `0x24` | **Slot 4** | 8–14 | Syntactic Steering (Bitmasked) |
| **37-38** | `0x25-0x26` | **N/A** | 8–14 | Reserved CPU Metadata |
| **39** | `0x27` | **Slot 5** | 8–14 | Structural Logic (DepHead) |
| **40** | `0x28` | **Slot 9** | 15–21 | Intent Flags / Delimiter Bit |
| **41-42** | `0x29-0x2A` | **Slot 10** | 15–21 | Domain Signature (Mode) |
| **43-60** | `0x2B-0x3C` | **Slots 6-8** | 15–21 | Recursive Context Bridge (Memory) |
| **61-64** | `0x3D-0x40` | **Slot 11** | 15–21 | **LSH Salt**: `(PosIndex << 16) | TemporalSalt` — Contextual Anchor (warm uniqueness) |
| **65-80** | `0x41-0x4F` | **N/A** | 15–21 | Z3 Prover Space / Reserved |



### 3.2 Slot 4 Bitmask Specification (The Syntactic Byte)
To maximize the 8-bit space of Slot 4, the following bit-allocation is enforced:
* **Bits 0-3 (4-bit, 0-15)**: **POSTag** (Part-of-Speech category).
* **Bits 4-5 (2-bit, 0-3)**: **Tense** (0=N/A, 1=Past, 2=Present, 3=Future).
* **Bits 6-7 (2-bit, 0-3)**: **Plurality** (0=N/A, 1=Singular, 2=Plural, 3=Dual).

### 3.3 Slot 9 Delimiter Specification (The Space Bracket)
When a Bracket represents a boundary (Space/Punctuation), it is identified as a **Delimiter**:
* **Slot 9 (Bit 7)**: Set to `1`. This signals the CPU to treat the Bracket as a structural anchor rather than a semantic token.
* **LSH Projections (Slots 0-3)**: Initialized to `[0]` or a designated "Null-Plane" vector.

### The "Space" (Delimiter) Logic with Tombstones
If an agent or a user decides a specific "Space" bracket is unnecessary, they flip the tombstone. However, because we are using a Recursive Context Bridge (Slots 6-8), the Compactor must be smart enough to know that removing a bracket—even a space—requires updating the Memory slots of the next bracket to maintain the chain.

### Final Verification of the S3 Antminer Block
The 12 slots in the asic_payload_slots sum exactly to 80 bytes. This is the hard-wired requirement of the Bitmain BM1382 chip. By storing the data this way, you ensure that the "Noted Resolution" (the JSON) and the "Vector" (the Binary) are perfectly synchronized.

This is the definitive build. It contains the logic, the physics, and the address for every single bit in the KNIRVBASE.
---

## 4. The JSON Registry Schema (The "Golden Record")
The Registry is the high-fidelity map used by the CPU to manage the binary data. It is stored as a JSON object at the head of every `.nrv` file.

```json
{
  "tier_1_global_vault": {
    "dataset_id": "string",
    "pqc_manifest": {
      "sig_dilithium3": "byte_array",
      "kem_kyber768": "byte_array"
    },
    "global_metrics": {
      "total_brackets": "float_array[32]",
      "tombstone_ratio": "float",
      "compaction_threshold": 0.20
    }
  },
  "tier_2_temporal_frames": [
    {
      "frame_id": "string",
      "thermodynamics": {
        "temp_c": "float",
        "voltage": "float",
        "drift_score": "float"
      },
      "tier_3_data_brackets": [
        {
          "bracket_id": "string",
          "tombstone": "boolean",
          "type": "I-Bracket | P-Bracket",
          "registry_metadata": {
            "token_id": "string",
            "is_delimiter": "boolean",
            "z3_verified": "boolean"
          },
          "binary_offsets": {
            "asic_header": { "start": "uint64", "len": 80 },
            "arrow_modality": { "start": "uint64", "len": 512 },
            "z3_trace": { "start": "uint64", "len": "uint32" }
          },
          "asic_payload_slots": [
            "byte_array[32]", "byte_array[4]", "byte_array[1]", 
            "byte_array[2]", "byte_array[1]", "byte_array[1]", 
            "byte_array[2]", "byte_array[18]", "byte_array[4]", 
            "byte_array[15]"
          ]
        }
      ]
    }
  ]
}
```

---

### Why Every Field in this "Rich" data structure is Mandatory:

**1. The Tombstone (tombstone: false)**
* **This is the heartbeat of your Compactor.**
* **When you delete a bracket via KNIRVQL, the CPU flips this to true.**
* **Once the tombstone_ratio in Tier 1 hits 0.20 (20%), the background process kicks in, reads the binary_offsets of all false brackets, and rewrites them into a fresh, clean .nrv file.**

**2. The Binary Offsets (binary_offsets)**
* **This is what enables Apache Flight and Zero-Copy Streaming.**
* **The asic_header offset points to the exact 80-byte block on disk.**
* **The arrow_modality offset points to the heavy training data (the high-dimensional vector).**
* **By having these in the JSON, the CPU doesn't have to guess where the binary data starts; it can use a Linux sendfile or Rust mmap to stream the data from disk to the network/GPU with zero CPU overhead.**

**3. The ASIC Payload Slots (asic_payload_slots)**
* **This is your Bitmain S3 Instruction Set.**
* **These 12 lines of integer arrays are the "Map" of the 80 bytes found at the asic_header offset.**
* **Having them in the JSON allows the HERO model to perform "Semantic Analysis" on the slots without having to physically load and parse the binary chunk.**


```json
{
  "tier_1_global_vault": {
    "dataset_id": "knirv_alpha_001",
    "pqc_manifest": {
      "signature": [212, 45, 12, "...", 88],
      "kem_algorithm": "Kyber768"
    },
    "global_metrics": {
      "total_brackets": 142005,
      "tombstone_ratio": 0.04,
      "compaction_threshold": 0.20
    }
  },
  "tier_2_temporal_frames": [
    {
      "frame_id": "f_101",
      "thermodynamics": {
        "temp_c": 72.4,
        "voltage": 0.85,
        "drift_score": 0.002
      },
      "tier_3_data_brackets": [
        {
          "bracket_id": "b_101_01",
          "tombstone": false,
          "type": "I-Bracket",
          "registry_metadata": {
            "token_id": "HERO_42",
            "is_delimiter": false,
            "z3_verified": true
          },
          "binary_offsets": {
            "asic_header": { "start": 1024, "len": 80 },
            "arrow_modality": { "start": 1104, "len": 512 },
            "z3_proof": { "start": 1616, "len": 256 }
          },
          "asic_payload_slots": [
            [74, 43, 60, 77, "...", 121],   /* Slots 0-3 (32B)  */
            [0, 12, 214, 84],               /* SubSecondUS (4B) */
            [147],                          /* Slot 4 (1B)      */
            [0, 0],                         /* Reserved (2B)    */
            [12],                           /* Slot 5 (1B)      */
            [1],                            /* Slot 9 (1B)      */
            [4, 2],                         /* Slot 10 (2B)     */
            [161, 178, "...", 174],         /* Slot 6-8 (18B)   */
            [79, 45, 26, 155],              /* Slot 11 (4B)     */
            [0, 0, 0, 0, "...", 0]          /* Z3 Space (15B)   */
          ]
        }
      ]
    }
  ]
}
```

---

## 5. Temporal Logic & Delta Compression
The `.nrv` format uses a video-inspired Group of Brackets (GoB) system to minimize redundant data.

### 5.1 I-Brackets (Intra)
* Contain absolute values for all 12 slots.
* Act as the anchor for subsequent P-Brackets.
* Forced every 50 Brackets or whenever the **Euclidean Drift Score** exceeds a designated threshold.

### 5.2 P-Brackets (Predicted/Delta)
* Store only the mathematical difference (XOR or Arithmetic) from the anchor.
* **Drift Calculation**: The Euclidean distance ($D$) between the current LSH vector ($V$) and the anchor ($A$):
    $$D = \sqrt{\sum_{i=1}^{n} (V_i - A_i)^2}$$
* The `drift_score` is stored in Tier 2 to monitor hardware stability and semantic consistency.

---

## 6. Security & Verification (The "V" in KNIRV)
Validation occurs in a multi-stage process before a Frame is committed to the Global Vault.

### 6.1 The Z3 Validation Gate
1.  **ASIC Phase**: Hardware solves for the Golden Seed (Slot 11).
2.  **CPU Phase**: The CPU takes the 80-byte header and the result.
3.  **Formal Logic**: Z3 Prover runs against the Syntactic Steering (Slot 4) and Dependency Head (Slot 5) to ensure the resolution is logically valid.
4.  **Trace Storage**: The SMT-LIB logic trace is stored in a dedicated binary modality at the offset defined in the Registry.

### 6.2 PQC Frame-Signing
Once all Brackets in a Tier 2 Frame return `z3_verified: true`, the CPU signs the entire Frame using **Dilithium-3**. This ensures that the training loader only consumes formally verified and cryptographically secure data.

### 6.3 Difficulty-as-Deterrence Protocol (GoldenSeed Policy Bridge)

The `GoldenSeed` field (Slot 11, bytes `0x29–0x2C`) carries a dual role: it is
both the solved nonce from the ASIC hashing stave and a **hardware-validated policy
signal**. The Evo-GRPO trainer in KNIRVHASHER modulates the SHA-256 target difficulty
before hashing so that nonce magnitude encodes the policy classification of the bracket.

| Policy Context | Target Difficulty | Resulting Nonce Range | Signal Name |
|----------------|-------------------|-----------------------|-------------|
| Approved action | 2^16 (65,536) | `0x00000000` – `0x0001FFFF` | Low nonce — fast resolution |
| Flagged / audit | 2^24 (16,777,216) | `0x00200000` – `0x00EFFFFF` | Mid nonce — meaningful work |
| Denied / sensitive | 2^32 (4,294,967,295) | `0xFF000000` – `0xFFFFFFFF` | High nonce — Hardware-Validated Proof of Denial (HVPD) |

**Why this works:** The BM1382 ASIC cannot cheat the difficulty target. A low nonce
is statistically impossible to produce when the target difficulty is 2^32, and vice
versa. A `GoldenSeed` near zero is therefore an unforgeable proof that the bracket
was trained under an approved-context difficulty budget. A `GoldenSeed` near
`0xFFFFFFFF` is an unforgeable proof of a denied-context computation.

**Reading the signal at enforcement time:** The `NRVEnforcer` in KNIRVSERVER reads
the `GoldenSeed` bytes directly from the `.nrv` bracket without re-running the hash:

```
seed ≤ 0x0001FFFF  →  Approved  (low nonce, 2^16 work)
seed ≥ 0xFF000000  →  Denied    (HVPD, 2^32 work)
otherwise          →  Flagged   (route to human review)
```

**Z3 + Hamming divergence (Logic Trap):** If Z3 formal verification passes (structural
form is valid) but the Arrow KB Hamming guard returns `LowCoherenceFault = 0xDEAD`
(semantic context unresolvable), the enforcement layer issues `SIGKILL` to the
offending process via the `uprobe/NRVEnforcer_stateTransition` eBPF hook. This
divergence means the bracket's syntax passed formal constraints but its semantic
projection has no KB nearest-neighbour within `HammingThreshold = 8 bits` — the
bracket is formally coherent but semantically orphaned, which is the definition of
a Logic Trap.

---

We now transition from the static data structure to the **Active Operational Layer**: how the hardware actually "chews" the 80-byte header and how the software moves that data at scale.

---

## 7. The 21-Pass Hashing Logic (BM1382 Integration)
The Bitmain S3 Antminer (utilizing BM1382 chips) is repurposed to treat the 80-byte header not as a block to be mined, but as a **Constraint Matrix** to be solved. The hashing core cycles through the header in three distinct staves of 7 passes each (21 total).

### 7.1 Stave 1: Semantic Anchoring (Passes 1–7)
* **Input**: Slots 0–3 (The 32-byte LSH Projection).
* **Logic**: The ASIC performs initial rounds of SHA-256 transformations on the semantic vector. This anchors the hash result to the "Semantic Compass," ensuring that words with similar meanings occupy mathematically adjacent spaces in the resulting hash-target.

### 7.2 Stave 2: Syntactic Steering (Passes 8–14)
* **Input**: Slots 4–5 (The Bitmasked Syntactic Byte and Dependency Head).
* **Logic**: The ASIC introduces the grammatical constraints. By iterating through these passes, the chip filters potential "Golden Seeds" that do not satisfy the syntactic parity requirements defined by the `POSTag`, `Tense`, and `Plurality`.

### 7.3 Stave 3: Identity & Resolution (Passes 15–21)
* **Input**: Slots 6–11 (Context Memory, Intent, Domain, and the Golden Seed/Salt).
* **Logic**: The final staves integrate the recursive memory bridge and the domain signatures. The hashing core attempts to find a nonce (The **Golden Seed**) that satisfies the target difficulty, effectively "locking" the semantic, syntactic, and temporal context into a single verifiable 4-byte solution.

---

## 8. Apache Flight Producer & Zero-Copy Streaming
To facilitate the high-speed transfer of `.nrv` datasets to the **HERO model**, KNIRVBASE implements an **Apache Flight Producer** directly on top of the `.nrv` file system.

### 8.1 Zero-Copy Handover
Because the `.nrv` file maintains strict 8-byte alignment for its binary modalities, the Flight Producer utilizes the `DoGet` RPC call to perform a zero-copy handover:
* **The Registry Read**: The CPU reads the JSON Registry once to cache the `binary_offsets`.
* **The Stream**: When a client requests a specific frame, the server uses `mmap` (Memory Mapping) to point the Flight stream at the disk-backed buffer. 
* **The Result**: The data moves from the disk to the network interface card (NIC) without being duplicated in the CPU's memory space, achieving near-wire-speed throughput.

### 8.2 Flight Schema Mapping
The Flight stream presents the Tier 3 Brackets as a `RecordBatch` where each row represents one resolution:
* `Payload_ASIC`: FixedSizeBinary(80)
* `Feature_Vector`: FixedSizeList<Float32>(12)
* `Z3_Trace`: LargeUtf8 (Optional/Lazy-loaded)

---

## 9. The Compactor: Rewrite-and-Swap Algorithm
To maintain file health and prevent the "Swiss Cheese" effect in append-only storage, the Compactor runs as a background maintenance thread.

### 9.1 The 20% Trigger Logic
The storage engine monitors the ratio of tombstones ($T$) to total brackets ($B$):
$$R = \frac{T}{B}$$
If $R > 0.20$, the Compactor is initiated.

### 9.2 Execution Steps
1.  **Read-Check**: The Compactor scans the JSON Registry for all `tombstone: false` entries.
2.  **Shadow File Creation**: A new `.nrv.tmp` file is opened. 
3.  **The Sequential Copy**: Valid Brackets are copied into the shadow file. This process automatically collapses the gaps left by deleted data.
4.  **Vector Clock Update**: The `dataset_version` in Tier 1 is incremented.
5.  **The Atomic Rename**: The OS performs an atomic `rename()` call, replacing the old file with the new, compacted version.

---

## 10. The Implementation Roadmap & Project Structure (Go)
To ship the platform ASAP, the code is divided into stable public interfaces and volatile internal logic.

### 10.1 `pkg/knirvbase` (The Public API)
* **`DB` & `Collection`**: High-level handles for interacting with datasets.
* **`nrv.Spec`**: Public constants for the 80-byte header layout and Slot mappings.

### 10.2 `internal/` (The Engine)
* **`internal/storage`**: Handles the `flock` (file locking), bit-packing Slot 4, and the 80-byte "Bullet" assembler.
* **`internal/network`**: Implements the Apache Flight Producer and P2P sync logic via Vector Clocks.
* **`internal/prover`**: The Go-to-Z3 bridge that populates the `z3_trace` modality before a Frame is signed.

### 10.3 PQC Commitment Cycle
The final security step before the file is closed for transport:
1.  **Hash**: Calculate the SHA-3 hash of all Z3-verified Brackets in the Frame.
2.  **Sign**: Apply the **Dilithium-3** private key to the hash.
3.  **Store**: Append the signature to the `tier_2_temporal_frames` object in the Registry.

---

## 11. Final Constraints & Operational Bounds
* **Max Frame Size**: 10,000 Brackets (to ensure Z3 validation latency stays under 1s/frame).
* **Alignment Requirement**: 8-byte boundary for all binary modalities.
* **Concurrency**: Multiple Readers (Flight/mmap), Single Writer (flock).

---



This document now provides the complete technical blueprint from the hardware registers of the Bitmain S3 to the high-level PQC security layer. 

