# WASM / SDK re-release steps for 2a.6 + DEC-2 (audit item 6 — DEFERRED)

These steps are required to make the new `signing.sign_bootnode_enroll_*` core ops and the
updated `verify_bootnode_enrollment` (operator binding + nonce consumption) reachable from the
published Go and TS SDKs. They are intentionally NOT performed by the 2a6_dec2 fix (per
audit: "Do NOT do the wasm re-release yourself yet"). The Rust core + Rust ops dispatcher
already implement and unit-test the protocol; only the downstream artifacts lag until a
rebuild.

## Why a rebuild is needed
- The Go SDK (`knirv-sdk-go`) reaches the Rust core through cgo FFI (`internal/ffi.InvokeProto`).
  The currently published/build FFI artifact predates the enrollment ops, so it returns
  `"unsupported operation"` for `signing.sign_bootnode_enroll_challenge` / `_approval` /
  `verify_bootnode_enrollment`. The new Go/TS bindings therefore skip pending this rebuild.
- The npm-package bundles an inlined WASM core (`wasm/inlined_base64.js`) that also predates
  the enrollment ops.

## Steps
1. **Bump core version + digest.** The protocol/serialization of `BootnodeEnrollment` and the
   new `operator` field changed the canonical envelope payload, so the WASM digest is no longer
   0.2.0. Update `core/Cargo.toml` version and the stamped digest constant.
2. **Rebuild the core WASM asset.** From `packages/KNIRVSDK/core`: `wasm-pack build` (or the
   repo's `Makefile` wasm target) producing `pkg/`; rebuild the `wasm/knirv_sdk_core_bg.wasm`
   and its JS glue.
3. **Update the inlined WASM.** Regenerate `npm-package/wasm/inlined_base64.js` from the new
   `.wasm` so `initSync`/`initWasm` load the enrollment-aware core. Re-run
   `scripts/sdk-dist.mjs manifest` and `... stamp` to refresh `core.json` / manifest + stamp.
4. **Go SDK FFI artifact.** Rebuild the cgo FFI shared library (`internal/ffi`) against the new
   core so `InvokeProto` dispatches the `signing.sign_bootnode_enroll_*` ops. Update
   `go-package/internal/core/stamp.go` to the new digest.
5. **Regenerate bindings in the published packages.** Re-run the binding generators so
   `knirv-sdk-go@<new>` and `@knirv/sdk` (npm) embed the rebuilt core and correct stamps.
   Verify `go-package/signing/enrollment.go` and `npm-package/signing.ts` (already added as
   thin delegators) now resolve the ops.
6. **Publish matching release objects.** Publish the new `knirv-sdk-go` module version and the
   `@knirv/sdk` npm version consumed by SDK loaders; update any lockfiles / `replace`
   directives (e.g. `KNIRVSERVER/.../plugin/go.mod` `replace github.com/guiperry/knirv-sdk-go`).
7. **Post-rebuild verification.** After publish, the previously-skipped tests must pass without
   skipping:
   - `go-package/signing/enrollment_test.go::TestBootnodeEnrollmentVectors`
   - `npm-package/src/enrollment.test.ts::runBootnodeEnrollmentVectors`
    Re-run the Rust `cargo test --lib signing` (already green: 29 passed) as the source of truth.

## Status

**2a6b re-audit (audit_2a6b.md item 6): a local, non-release rebuild was performed so the
bindings run non-skipped. Nothing was published/uploaded.**

- **Rust core + ops: DONE and green.** 29 `cargo test --lib signing` tests pass, including
  the DEC-2 replay-safety regression (`bootnode_enrollment_invalid_approval_does_not_consume_nonce`):
  an invalid approval is rejected *before* the nonce is consumed, and the same nonce can still be
  used by a valid enrollment. The core op `signing.verify_bootnode_enrollment` now authenticates the
  full approval→challenge→root binding FIRST, then atomically consumes the nonce only as the final
  step before accepting.
- **Base64 contract (audit item 5) fixed in ops.rs.** `signing.sign_bootnode_enroll_challenge`
  base64-DEC0DES the `claims` field (Go/TS base64-encode it), so the signed claim matches exactly
  what callers supply. Verified by `TestBootnodeEnrollmentVectors`/`runBootnodeEnrollmentVectors`
  (claims survive the binding base64 contract identically).
- **Durable atomic host nonce store (audit item 4).** Go/TS `verifyBootnodeEnrollment` no longer
  take a caller-supplied `[]string`; they accept a `NonceStore`. The core op returns the consumed
  nonce after authentication; the host persists it through an atomic `consume-if-absent` store.
  Implemented: `MemoryNonceStore` (in-process) and `FileNonceStore`.
  - **Go `FileNonceStore`** uses `syscall.Flock` + atomic rename, persistent across restarts.
    `TestFileNonceStoreConcurrent` proves exactly one of 32 racing callers wins; `TestFileNonceStoreRestart`
    proves durability.
  - **TS `FileNonceStore`** uses `fs.openSync(path, "wx")` (**O_EXCL** — atomically fails if the lock
    file already exists), so exactly one caller — across OS processes AND async tasks — ever holds the
    lock; a stale lock (>10s, e.g. from a crashed holder) is removed before retrying. The previously
    non-exclusive `renameSync` lock is gone. `nonceStore.test.ts` `runNonceStoreTests` proves
    cross-process atomicity by spawning **4 concurrent child processes** racing the same nonce file:
    exactly one wins, the rest observe a replay. (Validated: `evidence_2a6c_ts.log`.)
- **Local wasm (wasi) rebuilt and re-pinned — Go bindings now run NON-SKIPPED.** Built
  `bindings/wasi` for `wasm32-wasip1` and re-pinned `go-package/internal/core/stamp.go` Digest
  (`664f88d5…`) and the plugin embedded digest + `assets/knirv_core.wasm`. `go test` with
  `KNIRV_SDK_DATA_DIR` seeded to the rebuilt core now runs `TestBootnodeEnrollmentVectors`,
  `TestFileNonceStore*`, `TestMemoryVsFileNonceStoreEquivalence`, the plugin `TestProvisionManagedKeyWiring`
  (non-skipped, passing) and `TestManagedKeyHandlerRoutesThroughDEC1Policy` (in-process handler test:
  managed key creation routes through `ProvisionManagedKey` and attaches the correct DEC-1 allowlist).
  - **The `TestProvisionManagedKeyWiring` conditional `t.Skip` (native-unavailable) was removed.** It now
    fails loudly on error by default; skipping is opt-in only via `KNIRV_TEST_ALLOW_SKIP=1` (set by a CI
    that intentionally has no wasm build). The same env-gate was applied to the Go enrollment test's
    `t.Skip` branch. So neither binding test silently skips. (Validated: `evidence_2a6c_go.log`.)
- **Go plugin production wiring (audit item 1): DONE.** `backend.go` `keys/{name}` and `rotate/{name}`
  handlers route `bootnode-master` / `root-owner` / `failover-identity` through `ProvisionManagedKey`
  / `RotateManagedKey`; the handler returns `managed:true` and the narrow server-side policy is applied.
  The end-to-end handler test (`TestManagedKeyHandlerRoutesThroughDEC1Policy`) is non-skipped and passing.
- **Launcher AppRole swap (audit item 2): DONE + wired to the spec 8.6 promotion hook.** `SwapChildToActive`
  (launcher `bao_handoff.go`) is the low-level swap (issue fresh active secret-id → login → revoke old
  token). `PromoteChildToActive` is the **spec 8.6 failover/promotion transition hook** — the function the
  launcher's promotion path calls when a standby is promoted to active (a failover CONFIRMED / RECLAIMED).
  It delegates to `SwapChildToActive`, so there is a single reviewed call site and no dead helper.
  `TestLauncherSwapChildToActive` and `TestLauncherPromoteChildToActive` exercise both against a fake
  OpenBao over a unix socket (non-skipped, passing).
- **ORACLE_VALIDATOR_KEY criterion (audit item 5/7): amended to "no runtime reader".**
  `PendingImplementations.md` §2a.6 Done-when and step (7), and `OpenBao.md` §4.7, now state the
  acceptance criterion is *no runtime reader* (no Go/TS/Rust code reads it from env/args/config/file),
  not literal grep-absence; the legacy name may remain in design docs. Verified: no `os.Getenv` /
  config binding / file read for `ORACLE_VALIDATOR_KEY` exists in the codebase.

**Not done (still deferred / owner-gated, per audit):**
- **No release/upload.** `stamp.go` and the plugin asset digest were changed only locally; they are
  NOT published to R2, and no lockfiles/`replace` directives were bumped to a new published version.
- **Live-vault P1 items remain owner-gated:** AppRole secret-id handoff + launcher policy swap
  end-to-end, `emit-bao key_maker` writing `node/validator`, and the two-node integration test.

**Local (non-release) builds performed for verification (both wasi/Go AND npm/wasm-bindgen cores were
rebuilt locally and re-pinned, so bindings run non-skipped without any publish):**
- `bindings/wasi` → `wasm32-wasip1` rebuilt; `go-package/internal/core/stamp.go` Digest
  (`664f88d5…`) and the plugin `embeddedCoreDigest` + `assets/knirv_core.wasm` re-pinned.
- `npm-package` wasm-bindgen core rebuilt via `node ./scripts/build.mjs`; the TS
  `runBootnodeEnrollmentVectors` + `runNonceStoreTests` now run non-skipped (no unsupported-op skip).
  Seed dir `/tmp/opencode/knirv_go_seed` carries the wasi core for Go; `KNIRV_SDK_DATA_DIR` points at it.

**Verification evidence (saved, timestamped):**
- `evidence_2a6c_rust.log` — `cargo test --lib signing`: 29 passed, 0 failed (includes the DEC-2
  replay-safety regression `bootnode_enrollment_invalid_approval_does_not_consume_nonce`).
- `evidence_2a6c_go.log` — `go test -v` for `go-package/signing`, the plugin, and the launcher:
  `TestBootnodeEnrollmentVectors`, `TestFileNonceStore*`, `TestProvisionManagedKeyWiring`,
  `TestManagedKeyHandlerRoutesThroughDEC1Policy`, `TestLauncherSwapChildToActive`,
  `TestLauncherPromoteChildToActive` all PASS; the only `--- SKIP` is the intentional child-process
  re-exec helper `TestPublisherNonceChild` (its real work runs inside
  `TestPublisherNonceTwoProcessContention`), not a verification skip.
- `evidence_2a6c_ts.log` — `tsc --noEmit` (no type errors) + `node scripts/test-signing.mjs`:
  `signing wrappers`, `enrollment nonce-store (TS)`, and `bootnode enrollment vectors (TS)` all pass.
