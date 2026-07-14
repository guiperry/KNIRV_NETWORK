# KNIRV Server Commits

## Status

This document is the implementation contract for KNIRV-native validation
commits. It supersedes the Git-backed proof-ledger transport proposed in
`KNIRVSERVER_Commit_Strategy.md`. That document remains useful background, but
KNIRV does not store proof bundles in bare Git repositories and does not expose
Git smart-HTTP endpoints.

Git remains the developer's source-history tool. KNIRV stores and certifies the
evidence that binds a supervised session to the exact Git commit that resulted
from it.

## Decisions

1. `knirv git commit` creates a normal local Git commit and a KNIRV proof in one
   resumable transaction.
2. The Git commit is created before proof finalization. The proof therefore binds
   the actual commit object rather than a prediction of it.
3. A commit contains only stable KNIRV identity trailers:

   ```text
   KNIRV-Project: <project_id>
   KNIRV-Session: <session_id>
   ```

   A proof root is not embedded in the commit because doing so creates a circular
   dependency between the proof and commit hashes.
4. Proof bytes are encrypted client-side and stored in KNIRV content-addressed
   storage. Plaintext proof data must never be written to server disk, logs, or
   queues.
5. KNIRVCHAIN is canonical for immutable commit/proof bindings and their status.
   The chain stores roots, indexes, validation certificates, and storage
   locations—not bulk source or evidence bytes.
6. KNIRV proofs do not contain a complete reconstructable source snapshot. They
   contain the commit object identity, tree and parent identities, before/after
   workspace hashes, the diff and supervised evidence needed to validate the
   change.
7. Proof certification happens before source push. `knirv git push` refuses to
   push a supervised commit until its KNIRVCHAIN transaction is final.
8. Failure after the Git commit is non-destructive: the local commit and pending
   transaction remain, `knirv git push` stays blocked, and a later command can
   resume upload, verification, replication, or minting.
9. The existing Pod Evidence / Dock Path remains independent from this Bundle
   Sign Path. `/api/dve/*` is not reused as the native commit-proof protocol.

## Local CLI transaction

The CLI owns `.knirv/`; the server never interprets it as a remote repository.

```text
.knirv/
  config.json
  active-session
  policies/
  sessions/<session_id>/
  transactions/<git_commit_sha256>.json
  receipts/<git_commit_sha256>.json
  objects/
  git-hooks/
```

The transaction is:

1. Require an active supervised session and locally registered policy.
2. Validate that `policy.json` is the exact redacted policy whose SHA-256 equals
   the bundle's `policy_hash`.
3. Run the normal Git commit, adding the project and session trailers.
4. Read the raw commit object, object format, OID, tree, parents, and diff. Compute
   an independent SHA-256 digest over the raw commit object.
5. Finalize and sign the proof around those exact values.
6. Generate a random data-encryption key, encrypt the manifest and chunks with
   authenticated encryption, and wrap that key for the owner and validator.
7. Ask KNIRVSERVER which ciphertext objects are missing and upload only those
   objects.
8. Submit the proof descriptor. Poll its durable operation until it is rejected,
   quarantined, or certified.
9. Persist the receipt. A separate `knirv git push` checks chain finality before
   delegating to Git.

The CLI installs a repository-local Git hook while a supervised session is
active. The hook validates a short-lived nonce controlled by `knirv git commit`.
This prevents accidental raw `git commit` during that session. It is not a claim
that Git itself cannot be bypassed; bypassed commits simply cannot receive the
certified supervised binding.

## Native HTTP API

All private routes require a bearer identity and project authorization. Public
lookup returns a sanitized certificate only.

```text
POST /api/v1/knirv/cas/objects/batch
PUT  /api/v1/knirv/cas/objects/{ciphertext_cid}
HEAD /api/v1/knirv/cas/objects/{ciphertext_cid}
GET  /api/v1/knirv/cas/objects/{ciphertext_cid}

GET  /api/v1/knirv/validator-key

POST /api/v1/knirv/projects/{project_id}/proofs
GET  /api/v1/knirv/projects/{project_id}/proofs/{proof_root}
GET  /api/v1/knirv/projects/{project_id}/operations/{operation_id}
POST /api/v1/knirv/projects/{project_id}/proofs/{proof_root}/access-envelope

GET /api/v1/knirv/public/projects/{project_id}/commits/{commit_digest}/proof
```

The CAS batch response identifies missing objects. Object uploads stream to a
temporary file, compute SHA-256 while receiving, reject a CID mismatch, then
atomically install the ciphertext. Re-uploading the same CID is idempotent.

Proof submission names every referenced ciphertext object, the encrypted
manifest, key envelopes, repository fingerprint, Git object format and OID, raw
commit SHA-256, tree and parent OIDs, workspace hashes, policy hash, signing key,
CLI signature, and related proof roots. The server rejects a descriptor whose
referenced objects do not exist or whose deterministic storage root is wrong.

## Encryption and storage

- Plaintext `proof_root` is SHA-256 over the canonical plaintext proof manifest.
- Each ciphertext object CID is SHA-256 over the exact encrypted representation
  uploaded to the server.
- `storage_root` is a deterministic Merkle root over the ordered ciphertext
  object descriptors.
- A per-proof data-encryption key is wrapped independently for the owner and the
  validator. Additional access is granted by appending a wrapped-key envelope;
  stored ciphertext is not rewritten.
- Validation decrypts only into bounded memory and wipes buffers when practical.
- A development deployment may use one storage replica. Production requires at
  least three confirmed replicas before minting.

The encrypted manifest wire representation is `KNIRV1 || nonce || ciphertext`,
using XChaCha20-Poly1305 and authenticated data
`knirv.proof-manifest.v1|<project_id>|<proof_root>`. The validator envelope is
`ephemeral_x25519_public_key || nonce || sealed_dek`, base64 encoded, under
`knirv.proof-key-envelope.v1`. The public validator-key endpoint publishes the
recipient, key id, algorithm, schema, and X25519 public key required by the CLI.

The server implementation provides a filesystem CAS adapter, durable metadata,
and independently configured filesystem replica stores. Set
`KNIRV_PROOF_REPLICA_DIRS` to an OS-path-list of durable roots. Confirmations are
issued only after each replica re-validates each ciphertext CID. These adapters
remain interfaces so distributed KNIRV storage can replace filesystem roots
without changing the HTTP contract.

Validator trust is fail-closed. KNIRVSERVER loads a stable base64 X25519 private
key from `KNIRV_PROOF_VALIDATOR_X25519_PRIVATE_KEY` or creates it once at
`<proof_store>/trust/validator-x25519.key` with mode `0600`; optional
`KNIRV_PROOF_VALIDATOR_KEY_ID` and `KNIRV_PROOF_VALIDATOR_RECIPIENT_ID` select
its public identity. `KNIRV_PROOF_SIGNING_KEYS_JSON` is a JSON object mapping CLI
signing-key ids to base64 Ed25519 public keys. Missing trust configuration leaves
the durable proof retryable in `verifying` and never certifies it.

## Durable operation model

An accepted proof moves monotonically through:

```text
prepared -> uploading -> uploaded -> verifying -> replicating
         -> mint_pending -> finalizing -> certified
```

Terminal failure states are `rejected` and `quarantined`. Every transition is
persisted before it is returned. Retrying the same submission returns the same
proof/operation when its immutable descriptor matches; conflicting reuse of a
proof root, Git digest, or operation id is rejected.

The server must fail closed. Local CAS presence or successful structural
validation is never reported as `certified`. Certification requires all of:

1. Authorization and project membership.
2. Ciphertext presence and storage-root validation.
3. Memory-only decrypt and proof-root validation.
4. Signature, policy, event-chain, artifact, Git object, trailer, tree, parent,
   workspace-hash, and diff validation.
5. Required storage replication confirmations.
6. A final KNIRVCHAIN receipt for `validation_proof_mint.v1`.

## KNIRVCHAIN transaction

`validation_proof_mint.v1` contains:

- project and session ids;
- repository fingerprint;
- Git object format, Git OID, independent raw-object SHA-256, tree, and parents;
- proof root, ciphertext storage root, policy hash, and bundle schema;
- owner signer and key id;
- validator certificate hash and validator identity;
- confirmed storage locations and related proofs;
- mint time and final transaction identity.

Consensus rules enforce one primary proof for `(project_id,
git_commit_sha256)`, prevent reassignment of a proof root, and forbid deletion or
detachment. Amend/rebase produces a different commit digest and proof. Status
changes such as supersession or revocation are append-only transactions.

## Authorization boundary

KNIRVSERVER routes the native API locally, but delegates identity and project
membership decisions to the backend authorization service over its Unix socket.
The contract is action-based (`cas:read`, `cas:write`, `proof:read`,
`proof:submit`, `proof:share`). The server forwards the caller's bearer token and
fails closed when authorization is unavailable. Public certificate lookup is the
only anonymous route.

## Implemented server phases

1. **Server foundation:** encrypted filesystem CAS, CID validation, durable proof
   descriptors/operations/indexes, idempotency, native routes, strict request
   limits, and tests.
2. **Authorization:** backend project-action authorization endpoint and
   KNIRVSERVER Unix-socket client; route all public traffic through KNIRVGATEWAY.
3. **Verification:** validator-key resolution, bounded memory-only decryption,
   canonical manifest/proof root, Git commit and trailer checks, signature and
   policy replay, and content-hashed validation certificate.
4. **Replication:** KNIRV storage provider interface, independently configured
   encrypted CAS roots, confirmations, idempotent repair, and production quorum
   enforcement.
5. **Chain:** registered `validation_proof_mint.v1`, semantic certificate and
   replica checks, proof/commit uniqueness at block acceptance, durable indexes,
   idempotent minting, and finality polling.

The following work belongs to the separately developed CLI and later protocol
extensions rather than this server implementation:

6. **CLI:** `.knirv` state, policy registration, Git wrapper/hook nonce,
   commit-object capture, encryption, resumable upload/submission, receipt, and
   finality-gated push.
7. **Extensions:** append-only supersession/revocation transactions, quotas,
   rate limits, audit logs without proof plaintext,
   corrupted-object quarantine/repair, key rotation, recovery tests, and
   multi-node end-to-end tests.

## Acceptance criteria

- Object bytes are accepted only when their digest matches the requested CID.
- No proof may reference a missing object or a different storage root.
- Duplicate uploads and identical proof submissions are safe and idempotent.
- Conflicting commit/proof bindings are rejected before chain submission.
- Restarting KNIRVSERVER preserves objects, descriptors, operations, and commit
  indexes and can resume non-terminal operations.
- Private routes fail closed without backend authorization.
- Plaintext proof bytes never appear in persisted server state or logs.
- A public lookup discloses only the chain receipt and sanitized certificate.
- `certified` is impossible without a final KNIRVCHAIN receipt.
- A failed proof transaction never deletes or rewrites the user's Git commit.
