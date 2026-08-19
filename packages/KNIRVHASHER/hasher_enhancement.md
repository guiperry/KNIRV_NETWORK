# KNIRVHASHER Formal Proof Enhancement Plan

## Status

**Proposal / implementation plan**  
**Scope:** KNIRVHASHER only, with a future verifier-worker deployment boundary.  
**Primary objective:** make KNIRVHASHER a neuro-symbolic system in which neural and ASIC-assisted components may *propose, prioritize, and attest* candidate assertions, while a deterministic formal proof kernel is the only authority that may mark an assertion as formally true.

---

## 1. Problem Statement

KNIRVHASHER currently combines several useful but different mechanisms:

- continuous embeddings, hash-based neural inference, and evolutionary search;
- NRV bracket encoding and math-domain slot schemas;
- lexical/structural checks in MATHASHER;
- append-only PoW seed attestations in `seed_writes.jsonl`;
- host/ASIC orchestration over gRPC.

These mechanisms do **not** currently establish a formal mathematical proof. In particular, the MATHASHER endpoint classifies LaTeX-derived role sequences and can return `VERIFIED` without checking that a proposition follows from premises. Its nonce is a deterministic input-derived value, not a proof term or an ASIC-mined proof of theorem validity.

This plan separates four claims which must not be conflated:

| Claim | Authority | Meaning |
|---|---|---|
| Structurally valid | MATHASHER precheck | Input conforms to configured syntax/domain constraints. |
| Candidate proposed | Neural/search layer | A model or user has produced a proof attempt. |
| Formally verified | Lean/Coq kernel | A complete proof term type-checks against a theorem in a pinned environment. |
| Hardware-attested | ASIC PoW verifier | A specific artifact hash was committed under a stated work policy. |

Only the third claim is a statement of formal truth. ASIC work supports provenance, rate limiting, and economic policy; it cannot turn an invalid proof into a valid one.

---

## 2. Target Architecture

```text
User / corpus / LLM / HASHER transformer
                 |
                 | untrusted theorem or proof candidate
                 v
      MATHASHER structural precheck
      - parser / token roles / NRV slots
      - domain and import policy
                 |
                 v
        Formal proof proposal service
      - tactic search / proof repair / retrieval
      - all output remains untrusted
                 |
                 v
      Canonical proof-asset builder
      - canonical source bytes
      - toolchain and dependency digests
      - content-addressed identity
                 |
                 v
      Sandboxed formal checker worker
      - Lean kernel initially
      - accepts or rejects complete artifact
                 |
        +--------+---------+
        |                  |
        v                  v
 FORMALLY_REJECTED   FORMALLY_VERIFIED
 diagnostics ->          |
 search feedback         v
                    Proof asset ledger
                         |
                         v
                 Optional ASIC PoW attestation
                         |
                         v
                Cross-node independently replayable asset
```

### 2.1 Trust boundary

The following components are untrusted with respect to theorem correctness:

- LLMs, HASHER transformer modes, embeddings, retrieval, and evolutionary search;
- candidate parsers, proof repair logic, and canonicalization implementation;
- proof artifact ledger transport and ASIC hardware claims.

The trusted correctness boundary is the selected proof assistant's kernel plus the pinned trusted environment it requires. The integration must retain the complete artifact and environment identity so any node can independently replay verification.

### 2.2 Technology decision

Use **Lean** for the first implementation because it provides constructive dependent type theory, broad mathematical library support, and a practical theorem-proving workflow. Keep the verifier protocol proof-system-neutral so Coq can be added later.

Do not implement a bespoke dependent-type kernel in KNIRVHASHER. A new kernel would be a high-risk security and correctness project. Instead, pin and sandbox a mature checker, then keep the trusted interface as narrow as possible.

---

## 3. Current-Code Mapping

| Current area | Role after enhancement | Required change |
|---|---|---|
| `pkg/hashing/math/` | Fast structural precheck and theorem-input normalization | Keep role/domain checks; never determine formal truth. |
| `pkg/hashing/api/math_api.go` | API orchestration | Replace ambiguous `VERIFIED` outcome and call formal-verifier client. |
| `internal/proto/hasher/v1/hasher.proto` | Host/device protocol | Add formal-verification messages/RPCs; preserve legacy RPC during migration. |
| `pkg/hashing/transformer/pipeline.go` | Candidate generation and repair | Feed formal checker diagnostics back into untrusted search. |
| `pkg/hashing/transformer/attestation_bridge.go` | Existing span-attestation lookup | Add independent formal-proof lookup keyed by theorem/proof asset IDs. |
| `pipeline/3_DATA_SEEDER/pkg/storage/seed_writer.go` | Seed/PoW ledger persistence | Keep seed ledger intact; add a separate proof-asset ledger. |
| `pipeline/2_DATA_ENCODER/config/math_schema.yaml` | Math input taxonomy | Add policy metadata only; do not encode proof validity in slots. |
| `cmd/driver/hasher-host/main.go` | Host runtime | Configure and expose formal verifier client and status metrics. |

---

## 4. Public Status Model

### 4.1 Replace the overloaded status field

Introduce these states:

```text
STRUCTURALLY_VALID
STRUCTURALLY_REJECTED
PROOF_PENDING
FORMALLY_VERIFIED
FORMALLY_REJECTED
CHECKER_UNAVAILABLE
ATTESTATION_PENDING
HARDWARE_ATTESTED
```

Rules:

1. `FORMALLY_VERIFIED` may only be produced from a successful formal-checker receipt.
2. `HARDWARE_ATTESTED` is additive metadata; it must not replace or imply `FORMALLY_VERIFIED`.
3. A structurally valid input is not necessarily a candidate proof, and a candidate proof is not necessarily formally verified.
4. Existing HTTP/gRPC clients receive legacy status mappings only during the compatibility window; new clients consume the explicit status model.

### 4.2 Legacy endpoint behavior

For `POST /v1/verify/math`:

- retain request compatibility;
- run the current mapper/watchdog first;
- return `STRUCTURALLY_REJECTED` on precheck failure;
- return `PROOF_PENDING` if a valid candidate was accepted for asynchronous checking;
- return `FORMALLY_VERIFIED` only with a checker receipt;
- expose a proof asset ID that clients can poll or retrieve.

The endpoint must not synthesize a nonce from input text and present it as a formal proof witness.

---

## 5. Formal Proof Asset Specification

Create `pkg/hashing/proofasset/` as a pure-Go package with strict validation and canonical serialization.

### 5.1 Core types

```go
type ProofSystem string

const ProofSystemLean ProofSystem = "lean"

type ArtifactRef struct {
    Name   string `json:"name"`
    Digest string `json:"digest"` // algorithm-prefixed, e.g. sha256:...
}

type ProofAsset struct {
    SchemaVersion       uint32        `json:"schema_version"`
    ProofSystem         ProofSystem   `json:"proof_system"`
    ToolchainDigest     string        `json:"toolchain_digest"`
    DependencyLockDigest string       `json:"dependency_lock_digest"`
    TheoremSource       []byte        `json:"theorem_source"`
    ProofSource         []byte        `json:"proof_source"`
    Imports             []ArtifactRef `json:"imports"`
    CandidateProvenance CandidateProvenance `json:"candidate_provenance"`
}

type VerificationReceipt struct {
    SchemaVersion    uint32 `json:"schema_version"`
    ProofAssetID     string `json:"proof_asset_id"`
    Status           string `json:"status"`
    CheckerDigest    string `json:"checker_digest"`
    EnvironmentDigest string `json:"environment_digest"`
    CheckedAt        time.Time `json:"checked_at"`
    DiagnosticDigest string `json:"diagnostic_digest,omitempty"`
}
```

`CandidateProvenance` records model version, prompt/context hashes, NRV-bracket ID, and optional source-document identifiers. It is audit metadata only and must never affect checker acceptance.

### 5.2 Canonical identity

Define a versioned byte serialization and derive:

```text
ProofAssetID = sha256(canonical-asset-bytes)
TheoremID    = sha256(proof-system || environment || canonical-theorem-bytes)
```

Canonicalization requirements:

- valid UTF-8 only, LF line endings, no ambiguous Unicode normalization;
- explicit proof-system and toolchain identity;
- imports sorted by normalized name then digest;
- no network-resolved or floating dependencies;
- bounded artifact size and exact length prefixes;
- raw submitted bytes retained separately for audit if normalization changes them.

Canonical identity supports replay, indexing, and deduplication. It does **not** claim to decide semantic equivalence of arbitrary proofs. The checker is the authority on validity.

### 5.3 Ledger records

Add a separate append-only `proof_writes.jsonl`; do not overload `seed_writes.jsonl`.

Each entry contains:

- `proof_asset_id`, `theorem_id`, schema version, and timestamps;
- canonical asset location/digest and optional encrypted storage reference;
- proof system, checker digest, dependency lock digest, environment digest;
- final verifier status and diagnostic digest;
- optional NRV/LM provenance;
- optional ASIC attestation object.

The ledger must only accept `FORMALLY_VERIFIED` entries when a receipt validates against the exact stored artifact IDs.

---

## 6. Formal Verifier Worker

### 6.1 Worker constraints

Run Lean outside the MIPS ASIC process, initially on the hasher host or a dedicated worker.

- pin the Lean compiler/kernel image and package lockfile by digest;
- execute in a sandbox with no outbound network and a read-only dependency cache;
- use a per-job temporary workspace and delete it after receipt generation;
- enforce CPU, wall-clock, memory, output-size, and import-count limits;
- accept no native build hooks, shell escapes, or unpinned downloads;
- emit structured diagnostics and retain digest-addressed raw logs.

The kernel/checker execution must be deterministic for a pinned artifact and environment. Operational timestamps are not part of the proof asset identity.

### 6.2 Initial verifier API

Add a proof-system-neutral RPC to the host-facing protocol:

```protobuf
service FormalVerificationService {
  rpc SubmitProof(SubmitProofRequest) returns (SubmitProofResponse);
  rpc GetProofStatus(GetProofStatusRequest) returns (GetProofStatusResponse);
  rpc GetProofAsset(GetProofAssetRequest) returns (GetProofAssetResponse);
}

message SubmitProofRequest {
  bytes canonical_proof_asset = 1;
  bool request_hardware_attestation = 2;
}

message SubmitProofResponse {
  string proof_asset_id = 1;
  FormalProofStatus status = 2;
  string diagnostic = 3;
}
```

The existing ASIC `HasherService` remains focused on hardware operations. A formal checker service must not be silently implemented by an ASIC fallback path.

### 6.3 Lean adapter

Add `internal/formal/lean/` with:

- request validation and source assembly;
- a fixed module prelude and explicit import allowlist;
- subprocess/container invocation behind an interface;
- receipt parsing that fails closed on malformed output;
- testable fake process runner used only in unit tests.

The initial scope supports closed Lean theorem files. It must reject declarations that need unavailable imports, implicit network resolution, or unapproved project-local code.

---

## 7. Neural Search and Feedback Loop

### 7.1 Candidate generation

Use existing HASHER/HEART inference as an untrusted proposer:

1. generate theorem statement, tactic sequence, or Lean proof source;
2. run MATHASHER structural precheck and import policy check;
3. submit only well-formed candidates to the verifier worker;
4. persist each candidate's provenance and submission outcome;
5. supply structured rejection diagnostics to a bounded repair/search loop.

The loop may use embeddings for retrieval, NRV slots for routing, and evolutionary scoring to rank candidates. It must never reward or persist a candidate as truth merely because it scores highly.

### 7.2 Diagnostic taxonomy

Normalize checker diagnostics into non-authoritative search labels:

```text
PARSE_ERROR
UNKNOWN_IDENTIFIER
TYPE_MISMATCH
UNSOLVED_GOAL
IMPORT_POLICY_DENIED
RESOURCE_LIMIT
CHECKER_FAILURE
```

Store the raw diagnostic digest with the label. The model sees diagnostics as feedback; the raw checker result remains the audit record.

### 7.3 Search controls

- maximum repair attempts per candidate and theorem;
- deduplicate by `ProofAssetID` and theorem/context identity;
- exponential backoff for checker resource failures;
- separate quotas for user-submitted and autonomous candidates;
- never submit unbounded model output directly to the checker.

---

## 8. MATHASHER and Schema Changes

### 8.1 Preserve current value

`pkg/hashing/math` remains responsible for:

- LaTeX/token handling;
- domain routing (`Arithmetic`, `Algebra`, `Calculus`, etc.);
- low-cost sequence and slot checks;
- NRV bracket preparation for training and retrieval.

It must not claim that role transitions, bitmask validity, a temporal lock, or a Golden Seed establish theorem correctness.

### 8.2 Schema metadata

Extend `pipeline/2_DATA_ENCODER/config/math_schema.yaml` with a policy section:

```yaml
formal_verification:
  default_proof_system: lean
  allowed_imports:
    - Mathlib.Algebra.Group.Basic
    - Mathlib.Data.Real.Basic
  max_source_bytes: 65536
  max_checker_seconds: 15
```

This is an admission-control policy. It does not represent proof rules and does not replace the proof assistant environment lock.

### 8.3 API migration

1. Add a `precheck_status` field to new math responses.
2. Add `formal_status`, `proof_asset_id`, and optional `receipt` fields.
3. Deprecate the meaning of the legacy `nonce` field for formal verification.
4. After client migration, remove `VERIFIED` from the precheck-only path.

---

## 9. ASIC Attestation Design

### 9.1 Purpose

After and only after formal verification, submit an ASIC mining job committing to:

```text
proof_asset_id || theorem_id || checker_digest || environment_digest || ledger_epoch
```

Pack this commitment deterministically into the existing Bitcoin-style header mapping. Record:

- full header bytes and header mapping version;
- target/difficulty, nonce range, and first valid nonce;
- computed double-SHA256 result;
- device/firmware identity and time window;
- verifier-side recomputation result.

### 9.2 Acceptance rule

```text
formally valid = verifier receipt validates
hardware attested = formally valid AND PoW witness validates
```

No consumer may infer `formally valid` from a nonce, seed, header, bucket, or ledger membership alone.

### 9.3 Verification

Add a pure software verification function that recomputes header commitment and PoW difficulty. Cross-node verification must not require access to the original ASIC.

---

## 10. Implementation Phases

### Phase 0 — Correctness and terminology (small, blocking)

1. Change MATHASHER response statuses to structural terminology.
2. Add documentation explaining the difference between structural validation, proof verification, and PoW attestation.
3. Mark input-derived temporal lock/nonces as non-proof metadata.
4. Add regression tests proving an arbitrary syntactically valid statement cannot receive `FORMALLY_VERIFIED` without a checker receipt.

**Exit gate:** no precheck-only code path produces a formal-truth status.

### Phase 1 — Proof asset foundation

1. Implement `pkg/hashing/proofasset` canonical model and validation.
2. Define `proof_writes.jsonl` schema and append-only writer/reader.
3. Implement content-addressed local storage for assets and logs.
4. Add theorem/proof lookup separate from `AttestationBridge`'s span witness lookup.

**Exit gate:** asset ID and theorem ID are stable across processes and malformed assets fail closed.

### Phase 2 — Sandboxed Lean verification

1. Produce a pinned Lean worker image/environment and lock digest.
2. Implement Lean adapter, sandbox runner, resource limits, and receipt parser.
3. Add gRPC contract and host configuration for worker endpoint/TLS.
4. Persist accepted and rejected receipts with exact environment identity.

**Exit gate:** known Lean proofs verify; invalid proof, invalid import, timeout, malformed receipt, and unavailable worker are distinct and fail closed.

### Phase 3 — MATHASHER and client integration

1. Update REST and gRPC response types.
2. Wire precheck → proof asset builder → verifier queue.
3. Implement status polling/retrieval and audit views.
4. Maintain a time-bounded legacy endpoint compatibility adapter.

**Exit gate:** clients can submit a candidate, receive pending state, retrieve a receipt, and independently reproduce the result.

### Phase 4 — Neural proof search feedback

1. Add proof-candidate adapter to HEART/HASHER pipeline.
2. Feed normalized rejection diagnostics to bounded repair loops.
3. Add candidate quotas, deduplication, and metrics.
4. Evaluate success rate, cost, and checker safety on a fixed benchmark corpus.

**Exit gate:** neural components improve candidate discovery without altering formal acceptance semantics.

### Phase 5 — ASIC attestation

1. Define header-commitment encoding and versioning.
2. Create ASIC submission path only for verified proof assets.
3. Add independent software witness verification.
4. Attach validated PoW attestation records to the proof ledger.

**Exit gate:** a remote node can validate the formal receipt and ASIC witness independently from stored artifacts.

---

## 11. Test Plan

### Unit tests

- canonical serialization is stable under process restarts;
- IDs change when theorem, proof, dependency, or toolchain changes;
- malformed UTF-8, duplicate imports, oversized artifacts, and unknown proof systems reject;
- MATHASHER precheck never generates a formal acceptance receipt;
- ASIC commitment encoding is deterministic and tamper-evident;
- ledger writer rejects a verified record without a matching receipt.

### Integration tests

- valid minimal Lean theorem produces `FORMALLY_VERIFIED` and replayable receipt;
- false theorem and incomplete tactic produce `FORMALLY_REJECTED`;
- import not in allowlist rejects before checker execution;
- worker resource limit yields `CHECKER_UNAVAILABLE` or resource-specific rejection, never verified;
- tampering with asset source, checker digest, or receipt causes replay failure;
- an ASIC witness generated for a verified artifact validates on a second host;
- an ASIC witness for an invalid or mismatched artifact never changes formal status.

### Property and adversarial tests

- fuzz canonical parser and proof-ledger reader;
- fuzz LaTeX precheck inputs and protobuf payloads;
- test Unicode confusables and source-normalization collisions;
- test path traversal, import injection, symlink escape, command injection, and excessive checker output;
- independently replay a random sample of ledger entries in clean verifier environments.

---

## 12. Security and Operational Requirements

- No secret, unrestricted host filesystem, or network access enters verifier jobs.
- Checker images and mathematical dependencies are immutable, signed, and digest-pinned.
- Raw candidate content is treated as untrusted input at every boundary.
- Ledger records are append-only; compaction/indexing must preserve original artifact IDs and receipts.
- Verification results include toolchain/environment digest so upgrades do not silently alter the meaning of prior results.
- Use explicit retention policy for raw candidate text and diagnostics, especially if user data can be embedded in proof prompts.
- Instrument queue depth, verifier error rate, median check time, resource-limit rate, duplicate rate, and independent-replay success rate.

---

## 13. Non-Goals

The first release does not:

- prove arbitrary natural-language claims directly;
- make embeddings, LSH buckets, NRV slots, seeds, or nonces into proof terms;
- replace Lean's kernel with a custom checker;
- require ASIC access to verify a formal proof;
- allow floating dependencies or arbitrary third-party Lean code;
- claim semantic proof equivalence from a source hash.

---

## 14. Acceptance Criteria

The enhancement is complete when all of the following are true:

1. Only a successful pinned formal-checker receipt can produce `FORMALLY_VERIFIED`.
2. Every verified ledger entry is reproducible from a content-addressed proof asset and its exact environment identity.
3. Neural/search components can propose and repair candidates but cannot bypass or modify checker acceptance.
4. MATHASHER’s structural checks remain useful but are correctly labeled as prechecks.
5. Hardware attestation is optional, independently verifiable, and never substitutes for formal verification.
6. A clean second node can replay formal verification and, when present, validate the ASIC witness.
7. Negative, malformed, and adversarial inputs fail closed with auditable diagnostics.

