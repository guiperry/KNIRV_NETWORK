# Bootnode validator enrollment

Enrollment has two separate approvals: Marketplace/onboarding approval and
failover-validator admission. A registration ID alone is not cryptographic proof
of validator-key ownership and never appends a validator automatically. The
automatic path below binds the approval to the new boot-node key; the manual
publisher workflow at the end remains the fallback.

## Automatic path (DEC-2)

The enrollment server verifies a challenge signed by the **new boot-node key** and
countersigned by the **root owner**, then appends the validator through the
backend's registry client. The enrollment process never holds the failover
publisher key.

Run the enrollment step on the **root / authorized publisher node**. The registry
Worker only accepts `POST /validators` from `ROOT_PUBLIC_KEY` or the stored
authorized publisher, so a non-root node cannot publish its own admission. (Use
the manual fallback if the root signs offline.)

### Prerequisites

1. The onboarding registration is approved and the operator knows its
   `registration_id` and the validator address (`validator_id`) the new key
   derives to.
2. The root operator writes the approved registration to the root node's
   `bootnode_approved_registrations.json` (default path:
   `<app-data-dir>/bootnode_approved_registrations.json`, override with
   `KNIRV_ENROLL_REGISTRATIONS`). The file is the trust store: a registration
   that is absent from it is rejected. Missing file = nothing approved.
3. `failover.chain_id` (or its config/env equivalent) on the backend is set to
   the validator-set chain. `KNIRV_ENROLL_CHAIN_ID` on the launcher must match it;
   the backend also rejects any admission for a different chain.
4. The root node's backend must already have a genesis validator set
   (`version >= 1`). Admission rejects an uninitialized set rather than inventing
   one.

Registration file formats (both accepted):

```json
{"registrations":[{"registration_id":"reg-123","validator_id":"knirv1...","chain_id":"knirvoracle-1"}]}
```

```json
[{"registration_id":"reg-123","validator_id":"knirv1...","chain_id":"knirvoracle-1"}]
```

`chain_id` is optional; when present it must match the enrollment challenge.
Every entry needs a non-empty `registration_id` and `validator_id`.

### Launcher configuration

The launcher enables the internal enrollment socket only when these are set:

```bash
KNIRV_ENROLL_SOCKET=/run/knirv/enroll.sock
KNIRV_ENROLL_ROOT_PUBKEY=<base64 compressed root public key>
KNIRV_ENROLL_OPERATOR=operator:alice
KNIRV_ENROLL_CHAIN_ID=knirvoracle-1                 # required: challenges are issued only for this chain
KNIRV_ENROLL_REGISTRATIONS=/path/to/approved.json   # optional override
```

`KNIRV_INTERNAL_AUTH_TOKEN` is generated per boot by the launcher and passed to
the backend; the launcher uses the same value to call the backend's internal
admission route over the backend Unix socket. Do not set it manually.

The gateway proxies only `POST /enroll/{step}` to the Unix socket; the backend's
`/internal/v1/knirv/enroll/register` route is not proxied and is reachable only
on the node-local backend socket.

### Enroll the new key

Two steps on the **root node**. The server, not the operator, issues the nonce.

1. **Issue a challenge (on approval).** The registration must already be in the
   approved file, or issuance is refused. The nonce is single-use, expires after
   `--ttl` (default 10 minutes, max 1 hour), and is bound to this registration and
   chain. This route is served only on the internal socket and is not under the
   gateway's public `/enroll/` prefix.

   ```bash
   knirvserver enroll issue \
     --socket /run/knirv/enroll.sock \
     --chain-id knirvoracle-1 \
     --registration-id reg-123
   ```

   It prints `nonce=enroll:...`, `validator_id=...` and `expires_at=...`. Pass the
   nonce to the registrant. The registrant must sign exactly that nonce.

2. **Enroll.** The new boot-node key signs the challenge and the root countersigns
   it. The root's approval covers both the challenge bytes and the boot-node public
   key, so an approval cannot be moved onto a challenge re-signed under another key. `--nonce` is required and must be the issued value; `--registration-id` is
   required (`--claims` is no longer accepted).

   ```bash
   knirvserver enroll \
     --socket /run/knirv/enroll.sock \
     --chain-id knirvoracle-1 \
     --operator operator:alice \
     --registration-id reg-123 \
     --nonce enroll:<issued-nonce> \
     --secrets-backend bao \
     --bao-socket /run/knirv/bao.sock \
     --bao-role-id <approle-role-id> \
     --bao-secret-id-file /secure/path/bao-secret-id
   ```

   Keys come from the secrets provider, never from config or env. `--secrets-backend`
   accepts `bao`, `file` or `auto`. The boot key is the node's `boot.key` master
   wallet key and the root key is `root.key`'s signing key.

   With `bao`, the CLI logs in to OpenBao over `--bao-socket` with AppRole
   (`--bao-role-id` plus the secret_id read from `--bao-secret-id-file`). That
   file must be mode 0600 or stricter, and the secret_id should be single-use
   (`secret_id_num_uses=1`). The resulting client token is short-lived, is used
   only for this run, and is revoked (`auth/token/revoke-self`) when the run ends.
   No token file is read. Neither the secret_id nor the token is printed or logged.

   `--dev-raw-keys --boot-key F --root-key F` reads raw 32-byte key files. It is a
   dev/test fallback only and prints a warning. Do not use it on a production root.

The server checks both signatures, expiry, operator, chain, issuance (nonce issued
for this registration, unexpired, not yet redeemed), the approved registration, and
that the challenge public key derives to the registered `validator_id`. Only then
does it consume the nonce and ask the backend to append
`{validatorID, publicKey, active:true}` with `version + 1`, signed by the
root/authorized publisher. A nonce that is invalid, expired, unissued or for another
registration is rejected and counted, and does not burn a valid nonce. A failure
after the nonce is consumed (for example a backend 5xx) burns it, so the root must
issue a new challenge to retry.

Challenge delivery is out-of-band by design. The server never pushes a nonce to the
registrant, and there is no self-serve fetch route. The approver issues the nonce on
the root node and hands it to the registrant through a channel the operator already
trusts. The issue route stays off the gateway's public prefix.

The Rust core returns a stable reason code for every refused enrollment
(`challenge_signature`, `approval_untrusted_root`, `approval_payload_mismatch`, and
so on, as defined by `BootnodeEnrollFailure` in `KNIRVSDK/core/src/signing.rs`). The
Go and TypeScript bindings pass that code through unchanged. The enrollment server
counts each core refusal under `core_<code>` (for example `core_challenge_signature`).
A refusal with no core code, such as a replayed nonce, stays under
`verification_rejected`.

Outcomes are counted. `GET /enroll/stats` on the internal socket returns
`accepted` and per-reason `rejections` (wrong chain, unknown registration,
validator-id mismatch, replay, nonce_not_issued, nonce_expired, nonce_registration_mismatch, admission failure, and so on). Issuance refusals are counted as `issue_<reason>`. The route is not
proxied publicly.

After admission:

1. Fetch the validator set and confirm the exact `version` and key.
2. Wait for a signed heartbeat. Do not count the node as failover-eligible until
   it has fresh replica telemetry.

## Manual fallback

Use this when the root signs offline or the automatic path is unavailable.

1. Obtain the approved onboarding record, validator ID, compressed secp256k1
   public key, and a proof signed by that key over the approval ID and chain ID.
2. An authorized registry publisher fetches the current validator set, verifies
   the proof and that the ID/public key are not already present, appends
   `{validatorID, publicKey, active:true}`, increments the version by exactly
   one, and signs `POST /validators` with the currently authorized root/acting-root
   identity.
3. Fetch the resulting set, confirm the exact version and key, then wait for a
   signed heartbeat. Do not count a node as failover-eligible until it has fresh
   replica telemetry.

The backend seeds only the genesis root set. Automatic admission refuses
uninitialized sets, inactive validators, an ID bound to another key, and a key
bound to another ID; those cases are operator decisions, not enrollee choices.
