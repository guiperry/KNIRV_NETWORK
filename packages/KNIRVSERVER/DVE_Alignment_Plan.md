# DVE Alignment Plan

## Purpose

This plan redefines the DVE as a validation boundary instead of a third-party
tool runtime.

`DVE` means `Deterministic Validation Environment`. Its primary objective is to
prove what happened during an agent-assisted work session: what state existed at
the start, what actions were requested, what decisions were approved or denied,
what artifacts were produced, and whether the session followed the active
policy.

Under this model, third-party agents such as Claude, Codex, Hermes, and
OpenCode run on the user's host system. The KNIRV CLI supervises those local
tools, records the session evidence, and commits a signed evidence bundle to the
server DVE. KNIRVSERVER then validates, stores, indexes, and reports on that
bundle.

## First Principles

1. The server should not be the universal execution host.
2. The DVE should be the deterministic verifier of supervised execution.
3. User credentials should remain local or scoped to short-lived sessions.
4. The source of truth should be signed evidence, not implicit server trust.
5. Agent reasoning should be captured as observable evidence, not claimed as
   hidden chain-of-thought proof.
6. Validation should be replayable where possible and auditable where replay is
   impossible.
7. Every supervisor decision must be bound to policy, identity, time, input, and
   output hashes.

## Target Architecture

```text
User Host
  Claude / Codex / Hermes / OpenCode
          |
          v
  KNIRV CLI supervisor
    - launches or wraps agent tools
    - binds local workspace
    - records terminal and tool events
    - handles permission decisions
    - captures memvid streams
    - builds signed evidence bundle
          |
          v
KNIRVSERVER
  DVE ingest service
    - verifies signature
    - validates manifest hashes
    - validates policy decisions
    - stores logs, reports, media, proofs
    - indexes session evidence
    - emits validation certificate
```

## DVE Objective

A DVE must provide five boundaries.

### 1. State Boundary

The DVE records the start state of a session:

- workspace base hash
- source file manifest
- active policy hash
- tool names and versions
- CLI version
- user identity
- DVE identity
- project identity
- OS and runtime metadata
- declared network mode
- declared credential scopes

### 2. Decision Boundary

The DVE records every supervised decision:

- command requested
- file write requested
- network request requested
- credential access requested
- permission hook invoked
- approval or denial result
- supervisor identity
- policy rule matched
- decision timestamp
- input hash
- output hash

### 3. Evidence Boundary

The DVE stores evidence artifacts:

- append-only event log
- terminal transcript
- tool invocation log
- source diffs
- final source snapshot or content-addressed file map
- reports
- validation outputs
- memvid reasoning or transcript media
- screenshots or screen recordings when enabled
- dependency and environment manifests

### 4. Proof Boundary

The DVE stores proofs:

- hash chain for event log
- Merkle root for artifacts
- local CLI signature
- user identity signature
- optional hardware-backed signing metadata
- policy hash
- workspace base hash
- workspace final hash
- validation certificate hash

### 5. Replay And Audit Boundary

The DVE supports:

- deterministic verification of hashes and signatures
- policy replay against permission decisions
- source diff reconstruction
- session timeline reconstruction
- partial command replay where safe
- audit report generation

The DVE does not claim perfect replay of non-deterministic LLM outputs.

## Local Workspace Model

The KNIRV CLI creates a locally bound workspace for each supervised session.

```text
.knirv/
  sessions/
    <session_id>/
      session.json
      policy.json
      manifest.json
      workspace/
      events/
        eventlog.jsonl
        eventlog.hashchain
      proofs/
        permission-decisions.jsonl
        artifact-merkle-root.json
        signatures.json
      memvid/
        reasoning-stream.m4v
        index.json
      reports/
      diffs/
        patch.diff
      artifacts/
```

The local workspace should support two modes:

1. Mirror mode: the CLI mirrors the project source into `.knirv/sessions/.../workspace`.
2. Bind mode: the CLI supervises the real workspace and records before/after
   file hashes.

Mirror mode is safer for reproducibility. Bind mode is faster and more natural
for local development.

## Evidence Bundle

The final submitted artifact is a signed evidence bundle.

```json
{
  "schema_version": "dve.bundle.v1",
  "session_id": "session_...",
  "dve_id": "dve_...",
  "user_id": "user_...",
  "project_id": "project_...",
  "started_at": "2026-07-07T00:00:00Z",
  "completed_at": "2026-07-07T00:00:00Z",
  "workspace_base_hash": "sha256:...",
  "workspace_final_hash": "sha256:...",
  "policy_hash": "sha256:...",
  "tool_runs": [],
  "permission_decisions": [],
  "artifacts": [],
  "memvid_refs": [],
  "eventlog_root": "sha256:...",
  "artifact_merkle_root": "sha256:...",
  "signature": {
    "key_id": "knirv-local-key-...",
    "algorithm": "ed25519",
    "value": "..."
  }
}
```

## Phased Implementation

### Phase 0: Vocabulary And Scope Lock

Goal: align the product and codebase around the DVE as a validation boundary.

Steps:

1. Define DVE terms in developer docs:
   - DVE
   - supervised session
   - evidence bundle
   - permission decision
   - validation proof
   - memvid stream (https://github.com/memvid/memvid)
   - local bound workspace
2. Update product language from "DVE runs all agents" to "DVE validates
   supervised agent sessions".
3. Mark server-hosted third-party agent execution as optional behavior.
4. Define trust levels:
   - unsupervised
   - locally supervised
   - signed supervised
   - hardware-backed supervised
   - server-executed deterministic

Deliverables:

- glossary
- threat model draft
- bundle schema draft
- validation status vocabulary

### Phase 1: KNIRV CLI Session Skeleton

Goal: create the local session lifecycle without deep agent integration.

CLI commands:

```bash
knirv session start --dve <dve_id> --workspace .
knirv session status
knirv session commit
knirv session abort
```

Steps:

1. Generate a `session_id`.
2. Resolve the active DVE.
3. Create `.knirv/sessions/<session_id>/`.
4. Snapshot workspace file hashes.
5. Write `session.json`.
6. Write initial `manifest.json`.
7. Start append-only `eventlog.jsonl`.
8. Hash-chain every event.
9. On commit, compute final workspace hash.
10. Generate unsigned bundle manifest.

Deliverables:

- local session directory
- event log format
- workspace hash manifest
- bundle manifest generator

### Phase 2: Permission Hook Supervisor

Goal: make supervisor decisions first-class evidence.

Steps:

1. Define permission event types:
   - `command.requested`
   - `command.approved`
   - `command.denied`
   - `file.write.requested`
   - `network.requested`
   - `credential.requested`
   - `escalation.requested`
2. Create local policy rules:
   - allowed command prefixes
   - denied command prefixes
   - writable roots
   - readable roots
   - network policy
   - secret access policy
3. Add a supervisor decision engine.
4. Record every decision with:
   - policy hash
   - matched rule
   - input event hash
   - decision hash
   - approver identity
5. Add manual approval support.
6. Add non-interactive deny-by-default mode.

Deliverables:

- `policy.json`
- `permission-decisions.jsonl`
- local approval UX
- policy replay test fixture

### Phase 3: Tool Wrapper Layer

Goal: run third-party agents locally under KNIRV supervision.

Initial tool targets:

- Codex
- Claude
- Hermes
- OpenCode

CLI commands:

```bash
knirv agent codex
knirv agent claude
knirv agent hermes
knirv agent opencode
```

Steps:

1. Add a tool registry:
   - name
   - binary discovery
   - version command
   - environment allowlist
   - auth home directory
   - transcript capture strategy
2. Launch tools through KNIRV CLI wrappers.
3. Allocate per-session tool home directories where supported.
4. Capture stdout, stderr, terminal I/O, and exit codes.
5. Capture tool version and config metadata.
6. Prevent global environment leakage by using env allowlists.
7. Record all tool invocations as event log entries.

Deliverables:

- `ToolRegistry`
- wrapper commands
- tool run log
- version capture
- supervised execution status

### Phase 4: Local Credential Boundaries

Goal: keep user authentication unique without placing third-party credentials on
KNIRVSERVER.

Steps:

1. Define credential references instead of credential values:
   - `local:keychain:<id>`
   - `local:file:<path>`
   - `env:<name>`
   - `knirv-secret:<id>`
2. Record credential usage events without recording secret values.
3. Support per-tool auth homes:
   - Codex: isolated `CODEX_HOME`
   - Claude: isolated tool-specific config/cache path where available
   - Hermes: isolated tool-specific config/cache path
   - OpenCode: isolated tool-specific config/cache path
4. Redact known secret patterns before logs are finalized.
5. Record redaction events so the audit trail shows that data was removed.
6. Ensure submitted bundles contain references and proofs, not raw tokens.

Deliverables:

- local credential resolver
- redaction engine
- credential access event schema
- per-tool auth isolation

### Phase 5: Memvid Reasoning Stream Capture (https://github.com/memvid/memvid)

Goal: capture observable reasoning and session narrative as searchable evidence.

Important constraint: the system must not claim access to hidden model
chain-of-thought. It can capture observable material:

- prompts
- tool-visible reasoning summaries
- terminal output
- assistant messages
- planner summaries
- approval explanations
- file diffs
- reports

Steps:

1. Define what gets captured into memvid.
2. Build a stream writer that records session timeline events.
3. Generate an `.m4v` or compatible media artifact.
4. Generate a searchable `index.json`.
5. Link memvid timestamps to event log hashes.
6. Add redaction before media finalization.
7. Add a "reasoning evidence" label that avoids claiming hidden
   chain-of-thought capture.

Deliverables:

- `memvid/reasoning-stream.m4v`
- `memvid/index.json`
- event-to-video timestamp links
- redaction pass

### Phase 6: Evidence Bundle Signing

Goal: make local evidence tamper-evident.

Steps:

1. Generate or import a local KNIRV signing key.
2. Store the key in the OS keychain where possible.
3. Support file-based keys only for development.
4. Create an event log hash chain.
5. Create a Merkle tree over artifacts.
6. Sign:
   - session metadata
   - policy hash
   - event log root
   - artifact Merkle root
   - workspace base hash
   - workspace final hash
7. Add optional hardware-backed signing support.

Deliverables:

- signing key management
- hash chain
- artifact Merkle tree
- `signatures.json`
- verification command

### Phase 7: Server DVE Ingest

Goal: let KNIRVSERVER receive and validate evidence bundles.

Proposed endpoints:

```text
POST /api/dve/{dve_id}/sessions/ingest
GET  /api/dve/{dve_id}/sessions/{session_id}
GET  /api/dve/{dve_id}/sessions/{session_id}/evidence
GET  /api/dve/{dve_id}/sessions/{session_id}/proof
GET  /api/dve/{dve_id}/sessions/{session_id}/report
```

Steps:

1. Accept bundle uploads.
2. Store raw bundle in content-addressed storage.
3. Verify schema version.
4. Verify identity and signature.
5. Verify event log hash chain.
6. Verify artifact Merkle root.
7. Verify workspace hashes.
8. Replay policy against permission decisions.
9. Mark session validation status:
   - `accepted`
   - `verified`
   - `verified_with_warnings`
   - `rejected`
   - `quarantined`
10. Generate server-side validation report.

Deliverables:

- ingest route
- verification service
- validation report
- evidence storage layout

### Phase 8: Server-Side DVE Commit

Goal: commit verified local session outputs into the server DVE.

Steps:

1. Require successful bundle verification before commit.
2. Compare submitted `workspace_base_hash` with current server DVE state.
3. If hashes match, apply final diff.
4. If hashes diverge, create a merge review state.
5. Store final source files or patch artifacts.
6. Bind committed source state to the validation certificate.
7. Emit DVE update events for UI consumers.

Deliverables:

- verified commit flow
- conflict detection
- DVE state update
- validation certificate

### Phase 9: UI And Reporting

Goal: make DVE validation understandable from the KNIRVSERVER frontend.

Views:

1. Session timeline
2. Permission decisions
3. Source diff
4. Tool runs
5. Validation proof
6. Memvid playback
7. Redaction report
8. Warnings and rejected evidence

Steps:

1. Add DVE session list to the DVE node view.
2. Add validation status badges.
3. Add proof details panel.
4. Add memvid player with event-linked timestamps.
5. Add policy replay results.
6. Add exportable validation report.

Deliverables:

- DVE session UI
- validation report UI
- proof detail UI
- media evidence viewer

### Phase 10: Hardening And Governance

Goal: close trust gaps and prepare for production use.

Steps:

1. Add bundle size quotas.
2. Add artifact retention policy.
3. Add redaction rules for common secret formats.
4. Add quarantine for suspicious bundles.
5. Add role-based access to evidence artifacts.
6. Add audit logging for bundle reads.
7. Add remote attestation option for high-trust clients.
8. Add policy version pinning.
9. Add per-organization validation profiles.
10. Add tamper-detection warnings for incomplete logs.

Deliverables:

- governance policy
- retention policy
- quarantine flow
- high-trust signing mode
- production threat model

## Validation Status Model

```text
unsupervised
  No KNIRV CLI evidence.

recorded
  Bundle exists, but no signature or incomplete verification.

signed
  Bundle is signed and hash-consistent.

policy_verified
  Permission decisions replay cleanly against policy.

source_verified
  Source base and final hashes match submitted artifacts.

certified
  Full DVE validation certificate issued.

rejected
  Bundle failed verification.

quarantined
  Bundle requires manual/security review.
```

## Caveats And Risks

### Hidden Reasoning Is Not Provable

Third-party agents often do not expose private chain-of-thought. KNIRV should
record observable reasoning evidence, tool output, summaries, prompts,
permission decisions, and artifacts. It should not claim to prove hidden model
reasoning.

### User Host Trust Is Limited

A malicious host can tamper with files, clocks, binaries, or local logs. Signed
hash chains make tampering visible after supervision starts, but they do not
make an untrusted host fully trustworthy.

Mitigations:

- local signing keys
- hardware-backed keys
- append-only logs
- server-side policy replay
- optional remote attestation
- clear trust-level labels

### CLI Bypass Is Possible

Users can run third-party tools outside the KNIRV CLI. The server should only
mark a session as supervised when all submitted file changes are attributable to
recorded events.

### Logs Can Leak Secrets

Terminal logs, memvid streams, and reports may contain credentials or sensitive
data.

Mitigations:

- redaction before finalization
- credential access references instead of raw values
- secret scanning before upload
- explicit redaction events
- encrypted artifact storage

### LLM Outputs Are Non-Deterministic

The DVE can validate provenance, decisions, and artifacts. It cannot guarantee
that a future LLM run will produce the same output.

### Large Artifacts Can Become Expensive

Memvid files, screen recordings, source snapshots, and logs can be large.

Mitigations:

- content-addressed storage
- compression
- deduplication
- retention policies
- upload chunking
- artifact class quotas

### Environment Drift Affects Reproducibility

Local OS, package versions, agent versions, model versions, and environment
variables affect results.

Mitigations:

- capture tool versions
- capture environment manifest
- capture dependency lockfiles
- capture policy version
- capture CLI version

### Server Commit Conflicts

The server DVE may change while a local session is running.

Mitigations:

- base hash check
- patch-based commit
- conflict state
- manual merge flow

### Vendor Terms And Tool Behavior

Some third-party tools may restrict automation, wrapping, transcript capture, or
credential handling. Each tool adapter should be reviewed independently.

## Near-Term Build Order

1. Define bundle schema and validation statuses.
2. Implement `knirv session start/status/commit`.
3. Implement file manifest and hash chain.
4. Implement permission decision logging.
5. Add one tool adapter first, preferably Codex or OpenCode.
6. Add signed bundle generation.
7. Add server ingest verification.
8. Add DVE session report UI.
9. Add memvid capture.
10. Add redaction and governance hardening.

## Success Criteria

The first production-ready version is complete when:

1. A user can run an agent locally through KNIRV CLI.
2. The CLI records all supervised actions and decisions.
3. The CLI creates a signed evidence bundle.
4. KNIRVSERVER verifies the bundle.
5. KNIRVSERVER rejects tampered bundles.
6. KNIRVSERVER can show a human-readable validation report.
7. The server DVE can commit verified session outputs.
8. The UI clearly distinguishes supervised evidence from unsupervised claims.

