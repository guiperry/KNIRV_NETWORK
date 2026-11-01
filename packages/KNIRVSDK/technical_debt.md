# KNIRVSDK: remove the C ABI and finish the Rust-core model

Audit date: 2026-10-31. Scope: everything under `packages/KNIRVSDK/`.

This is a phased plan. The audit was read-only: nothing has been built, run or
changed except the docs noted in Phase 0, and no test status is claimed.

## Decisions

1. **The C ABI is removed.** Rust, Go and TypeScript each run the core as
   WebAssembly.
2. **Network access uses a host-provided `http_request` import.** The core owns
   request construction, retries, failover and error mapping. The host only
   moves bytes.
3. **Python is dropped completely.** No replacement binding.
4. **The Go package downloads the unified WASM core from Cloudflare R2 and no
   longer embeds it.** Upload is part of the Makefile build, using the CLI's R2
   convention, to `knirv/sdk`. Go downloads from
   `https://releases.knirv.com/sdk`, only when SDK code first runs, and keeps it
   in an in-memory cache backed by the application data directory. No path
   setting is required. **TypeScript keeps its inlined base64 copy**, built from
   the same artifact and version as the Go one in the same SDK build.
5. **`crypto-core` is kept and made real.** It becomes the Rust implementation
   of the controller's vault crypto, and `KNIRV_CORP/packages/controller`
   consumes it from the SDK instead of doing that crypto in JavaScript.
6. **The Rust core stays the only implementation** (see "Development
   convention" in `README.md`). Go and TypeScript are thin facades.

## Implementation status (2026-10-31, second pass)

"Verified" means `make build-sdk` (Rust core + vault vectors, Go, npm, Go/TS
parity, digest consistency) passed at the end of this pass.

| Phase | State |
| --- | --- |
| 0 docs | Done. |
| 1 `http_request` seam | Done, both builds, verified. |
| 2 one artifact | **Decision: two thin builds, one crate** (a browser cannot make the WASI build's synchronous host call). Feature sets are aligned and documented (README "Two builds, one core"); `make parity` runs shared requests through both and fails on any difference (it caught TypeScript re-implementing `crypto.sha256`). `npm-package/wasm` is built from source by `make build-wasm` and not committed to the SDK repo; `check-stamp` fails if the inlined module differs from the manifest. |
| 3 R2 + Go download | Code done and tested. **Nothing is uploaded** (`releases.knirv.com/sdk/0.1.0/` does not exist), so a Go program outside the test suite fails until `make release-knirvsdk` runs with working `incline` credentials. The manifest records two digests (WASI for Go, web for npm) under one version. |
| 4 smaller modules | Done. One generated manifest (`core/wasm-modules/modules.json` -> `manifest.json`) read by Rust, npm and the release tooling; modules published under `/sdk/<version>/modules/`; Rust embeds bytes only with `embedded-modules` (native); Go's `ModuleBytes`/`WriteModule` removed (no callers); Go's second wazero runtime and embedded `proof-verifier.wasm` removed (`evidence.*` ops in the core); npm ships only `crypto-core.wasm`, others default to the release URL; KNIRVARENA fetches and digest-checks the published relay module. Module bytes are built with path remapping. |
| 5 `crypto-core` / vault | Rust vault (`core/src/vault.rs`) and real `crypto-core` module (392 KB, zero imports, 64 MiB Argon2id) done. Golden vectors captured from the controller's JS worker pin addresses, signatures and the sealed envelope bit for bit, including libsodium-vs-Rust Argon2id (the audit's open question: they agree). `@knirv/sdk/vault` facade done, with a test that scans module memory for the password, mnemonic and seed after use. Controller migrated behind the switch the plan calls for (`VITE_VAULT_CRYPTO`, default `rust` in test builds and `js` in production; both run on unlock in development) and no longer builds its own `crypto-core`. Verified against the local SDK (temporarily linked, then restored): the controller's `tsc -b` is clean, its vault, envelope and `cryptoWasm` contract tests pass, and `vite build` bundles `crypto-core` as an app asset. **Not done:** the controller depends on `@knirv/sdk@^0.2.0`, but the published 0.2.0 has no `@knirv/sdk/vault`, so a normal `npm install` there gets a package without it until this tree is published as 0.2.0 or the version is bumped; no measurement on low-memory phones or of unlock time; the unlock path has not been exercised in a browser; the legacy-sunset date is not chosen. |
| 6 drop Python | Done in `KNIRVSDK/`; marketing pages under `websites/` and `KNIRV_CORP` docs still mention it. |
| 7 move logic into core | Done: evidence (hash chaining, merkle, verification), noderole (shared vectors pass in Rust and Go), the actuarial route table, `crypto.sha256`. `models` and `models/frame` (row 2): byte-v1, the advice prompt and its redaction (hand-matched, differential-tested against Go's `regexp`), advice mapping, role lines, and frame assembly/hash/check, with the stage and tool tables now written once in `core/data/frame_defaults.json`; Go and TypeScript call the core. Decided, not moved: scrypt (row 5) and the CLI wallet (row 6), below. |
| 8 remove C ABI | Done: `bindings/c-abi` deleted, test runner and docs updated; the only `cgo`/`CGO` hits left are statements that there is none and the `nocgo_test.go` guard. |
| 9 housekeeping | Done: ignores, orphaned `cognitive-shell` directory removed, module count corrected, cancellation limit documented. |

Phase 7 decisions: see the end of this file.

## Where things stand

- Go and TypeScript already run the core as WASM. Go embeds
  `internal/core/knirv_core.wasm` and runs it under wazero (`CGO_ENABLED=0`,
  enforced by `nocgo_test.go`). TypeScript uses the wasm-bindgen build in
  `npm-package/wasm/`. Both call `call_local_json`.
- The C ABI's only remaining consumer is the Python `ctypes` adapter, plus the
  `c-abi` suite in `scripts/run-all-tests.sh`.
- The C ABI is the only path to the network operations. `BindingEngine::call`
  in `core/src/bindings.rs` handles `transaction.chain`, `gateway.health`,
  `transmission.resolve_uri`, `transmission.broadcast` and `wallet.account`
  through the async `KnirvClient`. Both WASM crates build the core with
  `default-features = false`, which drops reqwest, so these answer
  `UNSUPPORTED` there. Other service clients (`services.rs`, `governance.rs`,
  `oracled.rs`, `actuarial.rs`) are reachable only from Rust callers.
- **Neither language downloads the core today.** Go embeds the module in its
  binary (`//go:embed knirv_core.wasm` in `go-package/internal/core/runner.go`).
  TypeScript ships it inside the npm tarball: the default `@knirv/sdk` entry
  inlines it as base64 (`wasm/inlined_base64.js`), and `@knirv/sdk/slim` takes a
  URL or module from the caller, which a bundler or CDN usually supplies. The
  separate `controller-relay`, `crypto-core` and `proof-verifier` modules
  are also bundled; TypeScript fetches them from `./modules/<id>.wasm`
  relative to the package. Both core binaries are committed to git.
- `KNIRVSDK` has no Makefile. The root `Makefile` only has `test-knirvsdk`
  (it runs `scripts/run-all-tests.sh`), and no target builds the SDK or its
  `.wasm` files. They are built by hand (`go-package/scripts/build-wasm.sh`,
  `npm-package/scripts/build.mjs`).
- The three smaller modules exist in four places with three manifests, and
  only one of them is ever executed (Phase 4).
- `crypto-core` is a stub, and the controller's vault does its crypto in
  JavaScript instead of through it (Phase 5).
- Logic still lives outside the core in Go and TypeScript (Phase 7).
- Many docs still describe a Go CGO layer that no longer exists (Phase 0).

## Phase 0: fix the docs now

Cheap, and stops new code following the wrong model. No dependency on any other
phase.

- `README.md`:
  - package table (lines ~10-12): the Go row says "backed by the C ABI"; mark
    the C ABI row as being removed and delete the Python row; add the missing
    `bindings/wasi` row, which is the artifact Go actually uses.
  - Go quick start (~line 139): remove "static Linux AMD64 archive for the
    private CGO layer" and the reference to `go-package/scripts/build-native.sh`
    (it does not exist; `scripts/` holds only `build-wasm.sh`).
  - Python section (~170-185) and the "C ABI and language adapters" sentence
    (~186): remove or reword.
  - dev/verification block (~223): drop `bindings/c-abi` and the Python entry.
- `bindings/README.md`: refers to `internal/ffi` (it is `internal/core`), to CGO,
  and to legacy `../go`, `../py`, `../ts` directories that do not exist. Rewrite
  around the WASM builds.
- `core/SDK_Bindings.md`: the PyO3, napi-rs and cbindgen/CGO sections describe a
  design that was never the shipped Go path. Rewrite to match this plan, or
  mark the file historical.
- `go-package/signing/native.go:16`, `go-package/authority/native.go` (~line
  14) and `go-package/dveevidence/wasmverify.go:18`: comments still say
  "CGO-backed" / "native CGO bridge".
- Remove `test-reports/`, `coverage/` and `tsconfig.tsbuildinfo` from the tree
  if they are tracked, and ignore them.

Exit: no doc or comment under `KNIRVSDK/` mentions CGO or a Go native archive,
and the package table matches the directory tree.

## Phase 1: define the `http_request` seam in the core

Everything later depends on this contract, so settle it first.

- Specify the host import. Proposed shape, JSON in and out, matching the
  existing envelope style:
  - request: `{method, url, headers, body_base64, timeout_ms}`
  - response: `{status, headers, body_base64}` or a transport error
    (`timeout`, `dns`, `tls`, `refused`, `aborted`)
- Add it to the core behind a trait, for example `HttpTransport`, with two
  implementations: the current reqwest one (native Rust callers, default
  features) and a host-import one used when building for WASM.
- Route `KnirvClient` through the trait so retries, failover, auth headers,
  response decoding and error mapping stay in Rust and are identical on every
  host.
- Make the five network operations in `bindings.rs` work under the WASM build.
  This needs an async-to-host story:
  - wasi (wazero): the host import is called synchronously from inside
    `knirv_invoke`. Use a small single-threaded executor in the core that
    blocks on the import, so the three-function ABI is unchanged.
  - wasm-bindgen (browser/edge): a synchronous import cannot await `fetch`.
    Either expose an async export via `wasm-bindgen-futures` (the design in
    `SDK_Bindings.md` already assumed this), or run it through an asyncify-style
    split. Pick one and document it; the async export is the simpler choice.
- Replace the existing `BindingTransport` / `SdkClient.create(transport)` seam
  in `npm-package` with this single model, so there is one transport story and
  not two.
- Fix timeouts and cancellation as part of the contract: pass `timeout_ms`
  to the host, and let the host abort in flight. Today Go honours `context`
  only before a call starts (see `runner.go`), which is not enough for network
  calls.
- Tests: Rust unit tests with a fake transport (retry, failover, error
  mapping), and cross-language fixtures added to `bindings-tests/` for the
  five network operations with recorded responses.

Exit: with a fake host, `transaction.chain`, `gateway.health`,
`transmission.*` and `wallet.account` succeed through `call`, under both the
wasi and wasm-bindgen builds.

## Phase 2: one WASM artifact and one calling convention

Today there are two builds of the same core: `bindings/wasi`
(`wasm32-wasip1`, ~1.4 MB, wazero) and `npm-package/wasm-core`
(`wasm32-unknown-unknown` plus wasm-bindgen, ~0.77 MB). They differ in ABI,
size profile and features (`wasi` enables `secrets`; `wasm-core` leaves default
features off, so key-file operations are missing in the browser build).

- Decide whether to unify on the wasi three-function ABI
  (`knirv_alloc` / `knirv_free` / `knirv_invoke` plus the Phase 1 import) and
  drive it from TypeScript with a small hand-written loader instead of
  wasm-bindgen. That gives one module and one ABI for all hosts. If browser or
  Cloudflare constraints make wasm-bindgen necessary, keep two thin crates but
  share one lib crate, one feature set and one test suite so parity is
  tested rather than assumed.
- Align feature sets explicitly and document what each build exposes.
- Add a parity test that runs the full `bindings-tests/envelopes.json` fixture
  set against the Go (wazero) and TypeScript runners and fails on any
  difference.
- Make sure panic handling matches: both crates use `panic = "abort"`, so a
  panic is a trap. Go discards the instance; confirm the TypeScript path does
  the same (`npm-package/scripts/test-no-traps.mjs` exists for this).
- The committed-binary provenance problem (`knirv_core.wasm` for Go and
  `knirv_sdk_core_bg.wasm` for npm are committed, and nothing verifies they
  match `core/`) is resolved for Go by Phase 3, which stops committing it. The
  npm copy stays committed and embedded, so it still needs a CI
  rebuild-and-compare check.

Exit: one agreed ABI, one feature matrix, one built artifact, and a parity test
that passes on both hosts.

## Phase 3: publish the unified core to R2; Go downloads it

Depends on Phase 2: there must be one artifact to publish.

Scope: **Go downloads; TypeScript does not.** The TypeScript package keeps its
inlined base64 copy and its current `slim` entry. Both are built from the
same artifact in the same SDK build, so the Go and TypeScript cores always have
the same version and digest (see "Keeping Go and TypeScript matched").

### What the CLI does today (the convention to follow)

From `KNIRV_CORP/packages/cli/Makefile` and `scripts/install.js`:

- `rclone` is the uploader (`R2_CMD := rclone`), with a remote named `incline`
  (`R2_REMOTE`) and a prefix variable (`R2_CLI_PREFIX := knirv/cli`). All three
  are overridable from the command line.
- The `upload-release` target depends on the build target, checks that
  `rclone` exists, checks that each expected file exists, then runs
  `rclone copy <file> "$(R2_REMOTE):$(R2_CLI_PREFIX)/<os_arch>/"` per file.
- The installer downloads `<base>/<os_arch>/<binary>`, where the base defaults
  to `https://releases.knirv.com/cli` and can be overridden with
  `KNIRV_CLI_BASE_URL`.

The first segment of the rclone path (`knirv`) is the bucket and is not part
of the public URL. So `incline:knirv/cli` is served at
`https://releases.knirv.com/cli`, and the SDK follows the same rule:
`incline:knirv/sdk` is served at **`https://releases.knirv.com/sdk`**
(confirmed as the public path).

### Layout in R2

Unlike the CLI, which overwrites binaries in place, the Go package pins a
digest, so a published version must never change. Paths below are public
URL paths; the rclone destination is the same path under `knirv/`.

```text
/sdk/<version>/knirv_core.wasm
/sdk/<version>/knirv_core.wasm.sha256
/sdk/<version>/manifest.json          # version, sha256, size, git revision, build time
/sdk/latest/manifest.json             # pointer only; no .wasm copy
```

`<version>` is the `version` in `core/Cargo.toml`. `latest/manifest.json` is
the only object that is overwritten, and it names a version and digest. Go is
pinned to a version and digest, so `latest` is for tooling and humans.

### Build and upload (Makefile)

The SDK build itself publishes the core. Create `packages/KNIRVSDK/Makefile`
(there is none today) and add thin wrappers to the root `Makefile`.

```make
R2_CMD        ?= rclone
R2_REMOTE     ?= incline
R2_SDK_PREFIX ?= knirv/sdk
SDK_VERSION   := $(shell sed -n 's/^version *= *"\(.*\)"/\1/p' core/Cargo.toml | head -1)

build-wasm:        ## build the unified core; write dist/<version>/ with digest and manifest
stamp:      build-wasm   ## write version + digest into the Go constants and the npm build
build-npm:  stamp        ## build @knirv/sdk with the inlined copy of the same artifact
build-go:   stamp        ## build/test the Go package against the same digest
upload-wasm: build-wasm
	# same preflight as the CLI: rclone present, artifact present
	$(R2_CMD) copy --immutable dist/$(SDK_VERSION) "$(R2_REMOTE):$(R2_SDK_PREFIX)/$(SDK_VERSION)/"
	$(R2_CMD) copy dist/latest "$(R2_REMOTE):$(R2_SDK_PREFIX)/latest/"
build-sdk:  build-npm build-go   ## everything, no upload
release-sdk: build-sdk upload-wasm   ## the whole SDK release, upload included
```

Notes:

- `release-sdk` is the one command that builds, stamps, uploads and verifies.
  The upload is part of the SDK build process, not a separate manual step.
- Use `--immutable` for the versioned upload so a re-run cannot silently change
  a published version. A rebuild of the same version with different bytes then
  fails loudly, which is the intended behaviour: bump the version.
- `build-wasm` is the single place the module is built. It replaces
  `go-package/scripts/build-wasm.sh` and the wasm step in
  `npm-package/scripts/build.mjs`, which now consume its output.
- Wire it in like the CLI: the root `Makefile` gets `build-knirvsdk` and
  `release-knirvsdk` targets. Neither is added to `build-all` by default, so a
  normal build does not need R2 credentials.
- Reuse the CLI's checks (rclone present, files present) and its overridable
  variables. Do not add a second uploader or a different naming scheme.

### Keeping Go and TypeScript matched

- `stamp` takes the SHA-256 of `dist/<version>/knirv_core.wasm` and writes it
  to both the Go package (version and digest constants) and the npm package
  (the inlined copy and a `core.json` with version and digest).
- A final check in `build-sdk` fails if the Go digest, the npm digest and the
  `manifest.json` digest are not identical, or if either package's version
  differs from `core/Cargo.toml`.
- `release-sdk` runs `upload-wasm` only after that check passes, so the Go
  package can never pin a digest that is not at its URL.
- The Phase 2 parity test runs the same fixtures against the artifact Go
  would download and the copy npm inlines.

### When the download happens

**Only when SDK code runs, for the first time on a machine.** Not when an end
user installs or builds a program that uses the Go SDK. `go get` and `go build`
download nothing from R2 and nothing is added to the consumer's build or
install steps. A Go module cannot run a download step during `go get` or
`go build` anyway (no install hook, and `go:embed` needs files already in the
module).

The first SDK call that needs the core resolves it through three layers, in this
order, with no configuration:

1. **In-memory cache.** The compiled module held by the process (wazero's
   compiled module and the instance pool). Every call after the first in a
   process uses it. This is the hot path.
2. **Application data directory.** The downloaded and verified bytes, kept on
   disk so later runs of the program, and other programs on the machine, do not
   download again:
   `<user data dir>/knirv/sdk/<version>/knirv_core.wasm`
   - Linux: `$XDG_DATA_HOME`, else `~/.local/share`
   - macOS: `~/Library/Application Support`
   - Windows: `%LocalAppData%`
   Written atomically (temp file, then rename). If the directory cannot be
   written, the SDK keeps working from memory for that process, logs it once,
   and downloads again next run.
3. **Download.** `https://releases.knirv.com/sdk/<version>/knirv_core.wasm`,
   used only when layers 1 and 2 both miss. Then it is verified, written to the
   data directory, and loaded into memory.

Behaviour:

- Concurrent first calls share one download (`sync.Once` or `singleflight`).
- Importing the package does no I/O. Nothing happens until a function that
  needs the core is called, not in `init()`.
- Version and SHA-256 are Go constants in the package, written by `stamp`.
  The digest is checked on every load from disk, not only after a download. A
  corrupted file is discarded and re-fetched. A mismatch after a fresh download
  is a hard error and nothing is cached.
- No path setting is needed. The defaults above are the only configuration for
  normal use. Two optional overrides exist for unusual hosts and are never
  required: `KNIRV_SDK_BASE_URL` (a mirror, mirroring the CLI's
  `KNIRV_CLI_BASE_URL`) and `KNIRV_SDK_DATA_DIR` (a different location for the
  disk cache).
- There is no separate prefetch command and no download step for the end user to
  add. An offline or air-gapped host can be seeded by copying the verified file
  into the data directory.

Go changes:

- Replace `//go:embed knirv_core.wasm` in `internal/core/runner.go` with the
  three-layer loader, and delete the committed `internal/core/knirv_core.wasm`.
- Use wazero's compilation cache under the same data directory so the module is
  compiled once per machine.
- `nocgo_test.go` is unaffected.
- `dveevidence/wasmverify.go` is not changed by this phase (it embeds its own
  proof-verifier module; Phase 7 removes it).
- Tests need the module too. `go test` in the SDK build runs after `stamp`, with
  `dist/<version>/` seeded into a temporary data directory. `runner_test.go`
  adds a fake local server for the download path, a memory hit, a disk hit, a bad
  digest, an unwritable data directory and an unreachable server.

### Consequences to accept

- The first run of a program using the Go SDK needs network access, unless the
  data directory is already seeded. Later runs do not.
- `backend_server` in `KNIRV_CORP` consumes these packages. If it starts with no
  network and an empty data directory, its first SDK call fails. Deploy images
  must either allow that first fetch or have the file seeded into the data
  directory.
- `releases.knirv.com` and R2 become a run-time dependency of Go consumers, once
  per machine and version. The origin must stay stable and versions must stay
  immutable.
- A published Go version is only as durable as its R2 object. Never delete a
  versioned object that a released Go tag pins.
- The `controller-relay`, `crypto-core` and `proof-verifier` modules are not
  moved by this phase; Phase 4 handles them.

Exit: `make release-knirvsdk` builds, stamps and uploads an immutable version
plus a manifest, and fails if the Go and npm digests differ; a clean Go program
downloads, verifies and runs the module on first call with no copy of it in its
source tree and no download during `go build`; its second run starts from the
data directory without network; a corrupted file or download is rejected.

## Phase 4: the three smaller modules

Depends on Phase 3 (the R2 layout, the build and the data-directory cache). Do
step 0 first; it can cut the scope of the rest.

### How they are loaded today

| Where | What happens |
| --- | --- |
| Rust core | `include_bytes!` embeds `controller-relay`, `crypto-core` and `proof-verifier` (`core/src/wasm_modules.rs`), with each SHA-256 hard-coded there. The `wasm.manifest`, `wasm.bytes` and `wasm.verify` operations serve them. |
| Go `modules.go` | `ModuleBytes` and `WriteModule` call `wasm.bytes` on the core, then check the result against the manifest. Nothing executes the bytes. |
| Go `dveevidence/wasmverify.go` | `proof-verifier` is the only module that runs. It has its own embedded copy (`dveevidence/assets/`), its own hard-coded digest, and a second wazero runtime separate from the one that runs the core. |
| TypeScript | `moduleBytes()` fetches `./modules/<id>.wasm` relative to the package (`import.meta.url`) and verifies it against a manifest generated by `npm-package/scripts/build.mjs`. Nothing executes them. |
| KNIRVARENA | `src/relay/arenaRelayWorkerClient.ts` fetches its own `/build/wasm/controller-relay.wasm`. |

Problems:

- **Four copies of the bytes:** `core/wasm-modules/assets/`, `go-package/dveevidence/assets/`
  (proof-verifier only), `npm-package/modules/`, and ARENA's build output. The
  SHA-256 values match today (checked), but nothing enforces it.
- **Three manifests:** the Rust constants, the npm manifest generated by
  `build.mjs` (which also repeats each module's kind and capabilities in its
  own `moduleDetails` table), and one Go constant.
- **Size:** the Rust embed adds about 307 KB to the Go core, which has
  `embedded-modules` on. The npm core build has it off, so there the cost is the
  same 307 KB in the tarball.
- **`crypto-core` is a stub.** It is 549 bytes, `zeroize()` is empty and
  `self_test` returns a constant. Phase 5 replaces it with a real
  implementation.
- **No production callers found.** Searching `KNIRV_NETWORK` and `KNIRV_CORP`
  found no caller of `ModuleBytes`, `WriteModule` or `moduleBytes` outside
  tests. Callers in other repos would not show up.

### Step 0: confirm who needs module bytes

Search every consumer repo (including ones outside these two) for
`ModuleBytes`, `WriteModule`, `moduleBytes`, `wasm.bytes`,
`knirv-sdk extract` and the `controller-relay` / `crypto-core` /
`proof-verifier` file names.

- **If nobody calls the byte-serving API:** (after Phase 5 the controller
  calls the TypeScript one, so for TypeScript this branch no longer applies;
  it can still apply to Go) remove it from the Rust bindings,
  Go and TypeScript (and keep `wasm.manifest` and `wasm.verify` only if
  something uses them). The modules become release artifacts only, published
  to R2 (step 2) and fetched by the programs that run them. Steps 3 and 4 are
  then simpler or unnecessary.
- **If somebody does:** do all the steps below.

### Step 1: one manifest, generated by the build

- `build-wasm` (Phase 3) builds the three modules and computes, once, each
  module's digest, byte length, kind, capabilities, ABI version and artifact
  version. It writes one `manifest.json`.
- Rust, the npm build and Go all read that file. Delete the hard-coded digests
  in `wasm_modules.rs`, the `moduleDetails` table in `build.mjs`, and the
  `proofVerifierSHA256` constant in Go (that constant goes away fully in step
  3).
- `stamp` (Phase 3) writes the manifest into each package, and the existing
  "digests must be identical" check covers the module digests as well as the
  core's.

### Step 2: publish the modules to R2 and fetch them on demand

- Layout, next to the core:

  ```text
  /sdk/<version>/modules/controller-relay.wasm
  /sdk/<version>/modules/crypto-core.wasm        # made real in Phase 5
  /sdk/<version>/modules/proof-verifier.wasm
  /sdk/<version>/modules/manifest.json
  ```

  `upload-wasm` (Phase 3) uploads them in the same immutable `rclone copy`.
- **Go:** `ModuleBytes` uses the Phase 3 lookup, per module: memory, then the
  data directory (`<user data dir>/knirv/sdk/<version>/modules/<id>.wasm`), then
  download. Same rules: digest-checked on every load, lazy on first call, no
  download at build or install time.
- **Rust core:** make `embedded-modules` off by default. Keep it as an opt-in
  feature for native Rust builds so `knirv-sdk extract` keeps working offline.
  The core keeps only the small manifest, so digests and `wasm.verify` still work
  without the bytes. This removes about 307 KB from the Go core.
- **TypeScript:** `moduleBytes()` defaults to the R2 URL instead of a path
  relative to the package. That also removes the `import.meta.url` problems under
  bundlers and Cloudflare Workers. Keep the existing `source` argument for
  callers who supply their own bytes or URL. Verification against the manifest
  stays exactly as it is. Removing the modules from the npm tarball is optional
  and can wait; if they stay, `stamp` guarantees they match.

### Step 3: remove Go's second WASM runtime

`proof-verifier` is Rust already (`core/wasm-modules/proof-verifier/src/lib.rs`)
and does event-log and Merkle verification. Compile that logic into the core as
binding operations and call it through `internal/core`. Then delete
`dveevidence/wasmverify.go`, `dveevidence/assets/proof-verifier.wasm`, the
extra wazero runtime and the extra digest constant. This is the same work as the
first row of Phase 7; do it once, here or there, not twice. Keep
`proof-verifier.wasm` as a published artifact for runtimes that want to use it
directly.

### Step 4: point KNIRVARENA at the published module

Change `arenaRelayWorkerClient.ts` to fetch
`https://releases.knirv.com/sdk/<version>/modules/controller-relay.wasm`,
verify it against the manifest, and stop producing `build/wasm/controller-relay.wasm`.
`KNIRV_CORP/packages/controller` has a test that reads the same file from the
ARENA build directory (`relayWasm.test.ts`); update it to use the published
artifact or a copy from `dist/`.

### Step 5: `crypto-core`

Handled by Phase 5, which makes it a real module and moves the controller onto
it. Steps 1 and 2 above apply to it like the other modules, with one exception:
the npm package keeps `crypto-core.wasm` bundled (see Phase 5, step 4).

### Risks

- Moving the modules out of the core means the byte-serving operations need a
  network or a seeded data directory on first use. If step 0 finds callers that
  run offline, keep `embedded-modules` on for them.
- Step 4 changes a cross-repo build path. Run the `KNIRV_CORP` controller tests
  before merging.
- Removing the byte-serving API (step 0, first branch) is a breaking change to
  the Go and npm public surface. Version it accordingly.

Exit: one manifest generated by the build and read by Rust, Go and npm; the
modules published under `/sdk/<version>/modules/`; the Go core no longer
carries them; no second wazero runtime in `go-package/`; ARENA uses the
published relay module.

## Phase 5: make `crypto-core` real and move the controller's vault onto it

Depends on Phase 3 (the SDK build, manifest and digest) and Phase 4 step 1 (one
generated manifest). The Rust work can start earlier. The controller work needs
the SDK release.

### Where things are today

- **The SDK module is a stub.** `core/wasm-modules/crypto-core` is 549 bytes.
  Its exports are `abi_version`, `module_kind` (1), `crypto_protocol_version`
  (2), `self_test` and an empty `zeroize()`. Its comment says it is "the
  versioned WASM compatibility and zeroization boundary consumed by the vault
  worker". Nothing implements that.
- **The controller builds its own copy.** `controller/scripts/build-wasm.mjs`
  compiles the SDK's `crypto-core` source into
  `wasm-modules/knirv_sdk_crypto_core/knirv_sdk_crypto_core.wasm` (213 bytes, a
  different hash from the SDK's 549-byte one, because the toolchain and strip
  step differ). It is used only by `cryptoWasm.test.ts` and, if
  `WASM_MODULE_KIND=crypto`, by `upload-wasm-r2.mjs`. The vault never loads it.
- **The vault does its crypto in JavaScript.** `src/react-app/wasm/vaultCrypto.worker.ts`
  (165 lines) derives keys and seals vaults using `@cosmjs/crypto` (BIP-39,
  SLIP-10 secp256k1, Argon2id through `libsodium-wrappers-sumo`), `crypto.subtle`
  AES-GCM, `crypto-js` for legacy vaults, and `bech32`. Only the final
  signature goes through the SDK's WASM core (`signDirectTransaction`,
  `signMessageEnvelope`), and the private key is passed in as a plain
  `Uint8Array`. The seed lives in a JS `Map` and is wiped with `fill(0)`.
- **The SDK's Rust core has none of this.** It has no BIP-39, SLIP-10 or
  Argon2 code, so the derivation logic exists only in the controller. That
  breaks the Rust-first rule.
- **The controller already expects a `crypto` module.** The artifact pipeline
  (`wasmValidation.ts`, `artifactLoader.ts`, `abi.ts`, `wasmPublic.ts`,
  `backend_server` `controllerwasm/coordinator.go`) accepts module kind
  `crypto` = 1 and requires: no imports, and exports `abi_version`,
  `module_kind` and `self_test`. `build_plan.go` says crypto "is added when a
  dedicated adapter exists". That adapter does not exist yet.

### Who uses the vault crypto (checked)

- **Controller web app only.** `vaultCrypto.worker.ts` is reached through
  `vaultCryptoClient.ts` by message. `useVault`, `vaultSession`, `vaultEnvelope`
  and the pages that use them (Onboarding, Settings, Scanner, AgentChat,
  VaultPage, `useBackend`, `useVoiceIntegration`) go through that client, so they
  do not change as long as the worker protocol stays the same.
- **Capacitor Android and iOS layers: no.** `MainActivity.java` is an empty
  `BridgeActivity`, the iOS app has no plugin code, and none of the installed
  Capacitor plugins is a keystore, keychain or secure-storage plugin. The vault
  runs in the web layer's worker on every platform, and the native shells load
  that bundle. The Android keystore in `build.gradle` signs the APK.
- **Go desktop client (`KNIRV_CORP/packages/client`): no.** It has no mnemonic,
  BIP-39, SLIP-10, Argon2 or keyring code. `keys.go` stages node key files
  (`root.key`, `boot.key`) and `controller_login.go` does a device-login flow
  over HTTP.
- **Go CLI (`KNIRV_CORP/packages/cli`): not a consumer.** Transaction signing
  goes through the controller (`controller_signing.go`), so the CLI holds no
  vault keys. Its separate `wallet_manager.go` keyfile is covered in Phase 7.

So no Go or native facade is needed for the vault, and the TypeScript facade in
step 4 is the only one.

### What "as intended" means

1. **Secrets stay inside the module.** Seed, private keys and derived keys exist
   only in the module's linear memory. JavaScript gets session handles, public
   keys, addresses, signatures and the encrypted envelope, never a seed or a
   private key. Honest limit: code running in the same worker can still read the
   module's memory object, so this is not a sandbox against that worker. What it
   removes is the many JS copies of secrets (strings, typed arrays, GC
   leftovers), and it makes zeroization explicit and testable.
2. **Existing vaults keep working, bit for bit.** The `knirv-vault-v2.` envelope
   format, the Argon2id parameters, the HD path, the addresses and the signatures
   must not change. A user must be able to unlock a vault created by the current
   JS code.
3. **Zero imports, deterministic.** This is what the controller's validator
   already enforces. The module cannot call `getRandomValues` or read a clock, so
   the host supplies every random value and timestamp as input.
4. **One implementation, in Rust.** The vault logic lives in the SDK core crate
   and is tested natively. The module is a thin wrapper, as `bindings/wasi` is
   for the main core. Unlike the main core, the vault ops are stateful (they
   hold sessions), so they are exposed only through this module and are not
   added to the stateless `local_operation` surface.

### Design

- **Code layout.** `core/src/vault.rs` (new, behind a `vault` feature) holds the
  logic. `core/wasm-modules/crypto-core` becomes a raw-ABI wrapper that
  depends on `knirv-sdk` with default features off, reusing the existing
  `signing` and `address` code instead of re-implementing signing.
- **ABI.** The same raw ABI as `proof-verifier`: `abi_version`, `module_kind` (1),
  `allocate`, `deallocate`, `invoke(op, ptr, len)`, `result_pointer`,
  `result_length`, `free_result`, `self_test`, `knirv_self_test` and `zeroize`.
  Requests and responses are JSON, matching the other modules. `zeroize()` wipes
  every session and every scratch buffer.
- **Operations** (names are proposals): `vault.create`, `vault.import`,
  `vault.unlock`, `vault.migrate`, `session.account`, `session.derive`,
  `session.sign_tx`, `session.sign_msg`, `session.close`, `address.from_pubkey`.
  These map one to one onto the existing worker messages `vault:create`,
  `vault:import`, `vault:unlock`, `session:derive:address`, `session:derive:pubkey`,
  `session:sign:tx`, `session:sign:msg`, `session:close` and
  `session:derive:address:raw`, so the main-thread client does not change.
- **Host-supplied inputs.** Because the module has no imports, each request that
  needs randomness carries it: 16 bytes of entropy for a new mnemonic, a 16-byte
  salt, a 12-byte nonce, and a 16-byte session id. The envelope's `created_at`
  is also passed in. The host reads these from `crypto.getRandomValues` and
  `Date`.
- **Secret inputs.** The password and mnemonic are written by the host straight
  into module memory (as bytes, not JS strings) and are wiped by the module as
  soon as they are used. `deallocate` zeroes the buffer it frees.
- **Primitives.** Argon2id v1.3, 3 passes, 64 MiB, 1 lane, 32-byte output, 16-byte
  salt. AES-256-GCM with a 12-byte nonce. BIP-39 English wordlist with checksum
  validation and the current whitespace normalisation. SLIP-10 secp256k1 on
  `m/44'/118'/0'/0/i` (the SDK's `KNIRV_HD_PATH`). Compressed public key,
  address `knirv` bech32 of `ripemd160(sha256(pubkey))`. All of this matches the
  current worker, and the parameters are checked against the real behaviour by
  the golden vectors in step 1 and not assumed.
- **Legacy vaults.** Old vaults encrypted by `crypto-js` stay readable. The
  decryption stays in the worker's JavaScript as a one-way migration path, and the
  resulting mnemonic is handed to `vault.migrate`, which seals a v2 envelope. A
  sunset date is chosen and `crypto-js` is then deleted. Re-implementing the
  OpenSSL-style legacy format in Rust is possible but is not worth it for a
  format being retired.

### Steps

1. **Capture golden vectors from the current JS worker, before changing it.**
   For a fixed set of mnemonics, with injected randomness, record: derived public
   keys and addresses for indices 0 to N, a sealed v2 envelope for a fixed salt
   and nonce (and that it opens), signatures of a transaction and of a message
   envelope, an invalid-mnemonic case, a wrong-password case, and one legacy
   `crypto-js` vault. Store them in `testvectors/vault/` next to the other shared
   vectors. Everything after this is tested against them. This is the step that
   proves "compatible", so do not skip or shorten it.
2. **Rust implementation** in `core/src/vault.rs` with native tests against the
   vectors. Add `argon2`, `bip39` and `zeroize` as dependencies behind the
   `vault` feature, so the main core and the browser core are unaffected. Use
   `zeroize` on every secret type.
3. **The module.** Rewrite `crypto-core` as the wrapper above. Build it through
   the SDK's `build-wasm` (Phase 3) so it gets a digest in the generated
   manifest and is published under `/sdk/<version>/modules/`. Add checks: zero
   imports, the required exports, the `self_test` result, and a size budget (the
   old stub was 549 bytes; this will be hundreds of kilobytes, which is expected).
   Run the controller's `validateWasmModule` logic against the built file in the
   SDK's CI, so the two repos cannot disagree about the contract.
4. **SDK TypeScript facade.** Add `@knirv/sdk/vault`: a typed wrapper that
   instantiates the module, handles allocate, write, invoke, read and
   `zeroize`, and exposes the operations above. No cryptography is written in
   TypeScript. The npm package keeps `crypto-core.wasm` bundled even if Phase 4
   later stops bundling the other modules or defaults them to R2, because a
   vault must work offline on first use. No Go or native facade is needed:
   the controller's web app is the only consumer.
5. **Controller migration** (`KNIRV_CORP/packages/controller`):
   - Rewrite `vaultCrypto.worker.ts` so each message handler calls
     `@knirv/sdk/vault`. The message protocol and `vaultCryptoClient.ts` stay as
     they are. Remove `@cosmjs/crypto`, `bech32` and WebCrypto AES from the
     worker, and the SDK `initLocal` and signing calls. Keep `crypto-js` only for
     the legacy path until its sunset.
   - Load the module from an app asset bundled by Vite and verify its SHA-256
     against the SDK manifest at load, so the PWA and the Capacitor Android and
     iOS apps work with no network.
   - Change `scripts/build-wasm.mjs` so it no longer compiles
     `knirv_sdk_crypto_core` from SDK source. It takes the SDK's built artifact
     instead. Delete `wasm-modules/knirv_sdk_crypto_core/`.
   - Update `cryptoWasm.test.ts` to validate the SDK artifact and add functional
     tests that run the golden vectors through the worker.
   - Leave the signed-artifact distribution path for `crypto` unwired for now
     (`build_plan.go` and the `WASM_MODULE_KIND=crypto` upload path). The bundled,
     digest-checked copy is the source of truth. If you later want crypto updates
     to ship through the controller's grant pipeline, that is a separate change.
   - Bump `@knirv/sdk` and remove the dependencies the worker no longer uses.
6. **Rollout.** A wrong vault implementation can lock users out, so the first
   release keeps the old JS path behind a switch (default to the new module in
   test builds, to the old path in production) and compares the two on every
   unlock in development and in the golden tests. Flip the production default
   after a release with no mismatches, and remove the JS path one release later.
   Nothing rewrites a stored vault except the existing migration path, and only
   after the new envelope has been opened again successfully.

### Risks

- **Memory on phones.** Argon2id at 64 MiB needs the module to grow its memory.
  Declare a sensible maximum and test on low-memory Android and iOS devices. If
  allocation fails, the unlock fails with a clear error. It must never fall back
  to weaker parameters, because the envelope pins them.
- **Speed.** WebAssembly Argon2 should be close to the current libsodium
  build, but measure unlock time on the slowest supported device.
- **The isolation claim.** Do not describe this as protection from malicious code
  in the worker. See point 1 under "What as intended means".
- **Passwords from the UI.** A password typed into a JS string cannot be wiped
  from JS. The module can only wipe the copy it receives. This is an existing
  limit, not a regression.
- **Correctness depends on the vectors.** The libsodium-compatible Argon2
  parameters in particular are assumed from the current code and from cosmjs's
  use of `libsodium-wrappers-sumo`. Step 1 settles this before anything else is
  written.
- **Cross-repo release order.** The SDK (module, manifest, `@knirv/sdk/vault`)
  must be released before the controller can depend on it.
- **Legacy sunset.** Until `crypto-js` is removed, the legacy path still handles
  plaintext secrets in JS. Choose and publish the date.

Exit: `crypto-core` is a real module built and published by the SDK build; the
golden vectors pass in Rust and through the controller's worker; existing vaults
unlock and produce identical addresses and signatures; the controller no longer
contains vault crypto beyond the legacy path and no longer builds its own
`crypto-core`; `cryptoWasm.test.ts` validates the SDK artifact; `zeroize` and
`session.close` are covered by tests.

## Phase 6: drop Python

- Delete `bindings/py/` (`knirv_sdk/__init__.py`, `pyproject.toml`).
- Remove Python from `README.md`, `bindings/README.md`, `core/SDK_Bindings.md`
  and any CI or Makefile target (the root `Makefile` was only grepped, so check
  it directly).
- Search `KNIRV_NETWORK` and the sibling `KNIRV_CORP` for imports of
  `knirv_sdk` or `KNIRV_SDK_LIBRARY` before deleting. The audit found no
  consumer, but only inside `KNIRVSDK/`.

Exit: no Python in `KNIRVSDK/`, and nothing else in either repo references it.

## Phase 7: move remaining logic into the core

Order by how much each duplicates. Each move needs Rust tests first, then the
binding becomes a thin call, then the existing shared vectors prove parity.

| Order | Area | Today | Action |
| --- | --- | --- | --- |
| 1 | `go-package/dveevidence/` | Pure Go: sha256 hash chaining, ed25519 verification, report derivation (`verify.go`, `sign.go`, `hashcrypto.go`, `policy.go`). Only event-log and Merkle checks call WASM, via `wasmverify.go`, which embeds and runs its own wazero instance of a second copy of `proof-verifier.wasm` with a hard-coded SHA-256. | Add an evidence module to the core. Call it through `internal/core`. Delete `wasmverify.go`, the extra embedded asset and the second wazero path. Shared with Phase 4, step 3: do it once. |
| 2 | `go-package/models/` and `npm-package/src/models/` | Hand-kept Go and TypeScript twins of the KNIRVMODELS wire contract, held together only by `testvectors/models/`. | Move the wire types and status/advice logic into the core. Keep only language types in each binding. |
| 3 | `go-package/noderole/` | Pure Go `Resolve`, encoding root and boot-node policy. Vectors in `testvectors/noderole/`. | Move into the core. |
| 4 | `npm-package/src/actuarial.ts` | A `fetch` client; the header says "the Rust core owns the matching service", but the logic is TypeScript. | Once Phase 1 lands, route through the core's `actuarial.rs` client and delete the TypeScript client. |
| 5 | `go-package/secrets/file.go` | Runs scrypt with `golang.org/x/crypto/scrypt` in Go for speed, duplicating the key-file format in `core/src/secrets.rs`. This is the only reason `golang.org/x/crypto` is a dependency. | Decide whether to accept this as a documented performance exception or move it. Measure scrypt cost inside wazero first. If it stays, record it as the single sanctioned exception. |
| 6 | `KNIRV_CORP/packages/cli/internal/core/wallet_manager.go` | A Go wallet keyfile outside the SDK: Argon2id (64 MiB, 3 passes, **4 lanes**) and AES-256-GCM over a raw private key, in its own `WalletFile` JSON (cipher, KDF params, a SHA-256 "MAC"), using `golang.org/x/crypto/argon2` and go-ethereum. It is not a vault consumer: it cannot open vault envelopes and does not need to. It is a second implementation of the same primitives, in a different format. | Decide if the CLI wallet should move to the SDK. If so, the core must read and write the existing `WalletFile` format and its 4-lane parameters, and the CLI must keep opening wallets already on disk (add vectors from real files first). Do not change its format as a side effect of Phase 5. |

`signing`, `uri`, `authority` and `replication` in Go and TypeScript are already
thin wrappers. `RemoteSigner` orchestration in Go `authority` looks acceptable
but was not audited line by line.

Exit: no business logic, hashing, signing, encoding or validation left in
`go-package/` or `npm-package/`, other than the one sanctioned exception if
kept.

## Phase 8: remove the C ABI

Only after Phases 1 to 6 are done.

- Delete `bindings/c-abi/` (crate, `include/knirv.h`, tests, `Cargo.lock`). First
  check whether anything uses `knirv_sdk_invoke_proto` or the `prost`
  dependency.
- Remove the `c-abi` suite from `scripts/run-all-tests.sh` (line ~114) and its
  header comment (line ~8).
- Remove `knirv-sdk-c-abi` from any Cargo workspace, Makefile or CI step, in
  this repo and in `KNIRV_CORP`.
- Re-grep `KNIRVSDK/` for `c-abi`, `libknirv`, `knirv.h`, `knirv_sdk_core`,
  `cgo` and `CGO`. The only intentional hits left should be the
  `nocgo_test.go` guard.

Exit: the grep above is clean, and `cargo test`, `go test ./...` and
`npm test` pass.

## Phase 9: housekeeping

- Confirm `target/` under `core/`, `bindings/wasi/` and `npm-package/wasm-core/`
  is git-ignored.
- Clarify `core/wasm-modules/cognitive-shell/`. It exists, but the README and
  `wasm_modules.rs` discuss three embedded modules (and the README says "four"
  in one place).
- Document the cancellation limit for long-running operations such as
  `authority.mine_header`, which no WASM host without fuel or epoch
  interruption can cancel. wazero's per-loop interruption check measured
  30-65% slower on signing and scrypt (see `runner.go`).

## Risks

- **Browser async.** The synchronous wasi import model does not fit `fetch`.
  This is the main design risk in Phase 1 and the main reason Phase 2 might keep
  two thin builds.
- **Performance.** Moving logic into WASM can be slower than native Go. scrypt
  is the known case. Measure before moving anything compute-heavy.
- **Size.** The npm bundle inlines the module as base64. Adding the HTTP
  layer and more of the core will grow it. Track the size per phase.
- **R2 as a run-time dependency of the Go package.** After Phase 3 every Go
  consumer needs `releases.knirv.com` on its first run, unless the data
  directory is already seeded. Immutable versions, pinned digests and the
  offline path are what keep this safe.
- **Vault migration.** Phase 5 changes how users' keys are derived and stored.
  A mistake can lock people out. The golden vectors and the staged rollout in
  that phase are not optional.
- **Module distribution.** Phase 4 depends on the Phase 3 cache and R2 layout.
  Do step 0 of Phase 4 before building anything for it, because it may remove
  the byte-serving API entirely.
- **Cross-repo breakage.** `KNIRV_CORP` consumes Go packages from this repo.
  Phases 4 to 7 change their internals. Run its build before merging.

## Not verified by the audit

- Whether anything outside `KNIRV_NETWORK` and `KNIRV_CORP` uses the vault
  crypto. Inside these two repos it was checked (see Phase 5, "Who uses the
  vault crypto"): only the controller's web app does.
- Whether libsodium's Argon2id (as used by `@cosmjs/crypto`) produces the same
  bytes as the Rust `argon2` crate with these parameters. This is expected but
  unproven until the golden vectors run.
- Whether anything outside `KNIRV_NETWORK` and `KNIRV_CORP` calls the
  module-bytes API (`ModuleBytes`, `WriteModule`, `moduleBytes`).
- Whether anything outside `KNIRVSDK/` (including `KNIRV_CORP` and the root
  `Makefile`) uses the C ABI, the Python adapter or `build-native.sh`.
- Whether the committed `.wasm` binaries match the current `core/` source.
- Whether the `incline` remote's credentials can write to the `knirv/sdk`
  prefix (the public path `/sdk` is confirmed by you, not by the repo).
- Whether `target/` directories are ignored.
- Whether any tests currently pass in any package.

## Phase 7 decisions and the open item

**Row 5, `secrets` scrypt: keep in Go, as the single sanctioned exception.**
Measured (`go test ./secrets -bench Scrypt`, this host): 0.44 s per key-file
unlock with native scrypt against 1.32 s with the core's scrypt inside wazero, 3x.
It is paid once per root/boot key unlock, so the cost is real but small; the
format and KDF parameters still come from the core (`secrets.kdf_params`), and
`golang.org/x/crypto/scrypt` is the only reason that dependency remains.

**Row 6, CLI wallet keyfile (`KNIRV_CORP/packages/cli/internal/core/wallet_manager.go`): not moved.**
It is a separate format (Argon2id with 4 lanes, its own `WalletFile` JSON and a
SHA-256 "MAC"), it is not a vault consumer, and wallets already on disk must keep
opening. Moving it needs vectors captured from real wallet files first, and
Phase 5 deliberately did not touch it. Revisit if the CLI should share the vault.

**Row 2, `models` wire contract: moved to the core.** Chosen over keeping Go's
copy. Consequences you should know:

* Go's `models` module was dependency-free; it now requires the SDK (and so
  wazero). I added the `knirv-sdk-go` `require` and `replace` to the 11 modules
  that import it (`go mod tidy`, offline); wazero moved 1.8.1 -> 1.9.0 in
  KNIRVMODELS and its trainer. All of them build; the KNIRVMODELS packages that use
  `frame` (`framecheck`, `head`, `heartqwen`, `models`) and KNIRVINFERENCER pass
  against the built core.
* `frame.Check`/`CheckShape` run on every llama request, now as a WebAssembly call
  carrying the registry (about 10 KB of JSON). The KNIRVMODELS daemon therefore
  needs the core on first start, like every other Go consumer. Not measured under
  load.
* The pure helpers (`EncodeByteV1`, `BuildAdvisePrompt`, `Roles`, `Frame.Build`...)
  panic if the core cannot be loaded; `Client.Advise` abstains instead.
* `Constitution` and `ConstitutionVersion` remain Go constants (the frame text is in
  the core); the shared fixture fails if they drift.
* `RoleLine.String()` stays in Go and TypeScript: it is display formatting, and
  `RoleLine` must stay comparable.
* TypeScript: the `@knirv/sdk/models` functions now need the core initialised
  (`initLocal()`/`SdkClient.init()`); `ModelsClient.advise` does it itself. No
  consumer of `@knirv/sdk/models` exists in either repo.

## Things that now depend on the first release

* `make release-knirvsdk` has not run. Until it does, every Go consumer that
  imports this package (KNIRVSERVER, KNIRVGRAPH, KNIRVORACLE, KNIRVMODELS ...
  via `replace`) fails its first SDK call, including `noderole.Resolve`, which
  panics rather than guess a role. KNIRVGATEWAY's Dockerfile now builds the core
  from source, pins `stamp.go` to it and ships it in the image (not built here).
* The controller and KNIRVARENA now require `@knirv/sdk@^0.2.0` and need this tree's build, not the published 0.2.0, which must be
  published first. KNIRVARENA's relay worker fetches
  `releases.knirv.com/sdk/<version>/modules/controller-relay.wasm`, which does not
  exist until the release.
* `go-package` is its own repository: its changes (loader, evidence/noderole
  ports, removed `ModuleBytes`/`WriteModule`) need a commit and a tag whose
  `stamp.go` matches the published version.

## Initialisation and the wallet extension (follow-up)

* `initLocal()` is now called where a browser main thread or a bare Node route
  uses the core without it: KNIRVARENA (`main.tsx`, before the app), the wallet
  module (before each signature) and KNIRV.COM's registration route. KNIRVARENA's
  Jest setup restores the real `TextDecoder`/`WebAssembly` for suites that run the
  core and transforms the SDK's `import.meta` (`config/jest-sdk-glue-transformer.cjs`).
  ARENA's third copy of the role-line logic (`services/modelStatus.ts`) is gone; it
  re-exports the SDK's.
* `knirvwallet-extension` no longer imports from the SDK root (which never exported
  those names: they came from an Adena SDK). It imports the protocol from
  `@knirv/sdk/wallet`, uses `KnirvWallet`/`KnirvLedgerConnector` from
  `knirvwallet-module` (the `AdenaWallet` alias and the `AdenaLedgerConnector` class
  name are gone), injects `window.knirv` instead of `window.adena`, and imports
  `knirvwallet-torus-signin` instead of `adena-torus-signin`. Internal Adena-named
  classes and hooks were renamed (`KnirvExecutor`, `KnirvStorage`, `KnirvResponse`,
  `useKnirvContext`, ...). The message codes in `wallet_protocol.json` are new
  (1xxx requests, 2xxx success, 3xxx failure, 4xxx reject): the Adena SDK's own
  table was not available to copy, so anything that matched on the old numbers
  must be updated.
* **Gno removed from the wallet extension (follow-up).** No `@gnolang/*` import, Gno RPC,
  Gno indexer, `gno.land` network or Gno-realm call remains in `knirvwallet-extension`
  source. Replaced by `common/provider/knirv` (`KnirvChainProvider`): NRN balance and
  nonce from KNIRVORACLE, signed NRN transfers, chain info and badge-credential /
  event-bundle lookups by id from KNIRVCHAIN, broadcast through the module's
  `/api/transmission/broadcast`, all via the gateway. Default networks, the faucet
  (Oracle test faucet) and the apps list are KNIRV's. Removed outright: the "Gno Connect"
  page-meta-tag feature, the realm helpers (`vm-call-users/boards`), GRC20/GRC721 lookups
  (the repository methods now return empty results or `UnsupportedOnKnirvError`),
  the Gno indexer queries (history is "not available", as in the controller) and gas
  simulation (a nominal fee schedule is shown; KNIRV does not meter gas).
  `window.knirv.request()` is implemented (see the extension README) with 19 unit tests;
  `knirv_signMessage` and the `disconnect` event are not supported yet.
* **The extension type-checks clean** (`tsc --noEmit`, with the module and SDK mapped to
  the local sources). It still has no `node_modules`, so it was not built, its Jest suite
  was not run, and nothing was exercised in a browser. The new code is covered by tests run
  under Vitest from the controller's install.
* **Still Adena- or Gno-flavoured, left on purpose:** persisted storage keys
  (`adena-wallet`, `ADENA_DATA`) and the stored token discriminator `'gno-native'` (renaming
  them would orphan saved data without a migration); the historical migration models v001-v009;
  the `gnoUrl`/`indexerUrl`/`apiUrl` fields on `NetworkMetainfo` (kept, empty); page, route
  and logo names and UI strings ("Adena", "Gnoscan"); and 136 files whose types and tests
  still say GNOT/GRC20/GRC721. Designing KNIRV token and NFT models for those screens is a
  separate piece of work. **Existing installs keep their stored Gno networks**; only fresh
  installs get the KNIRV ones. A migration (v010) is needed before shipping to anyone who
  already has the old extension.
* **The Oracle's address form:** the Oracle accepts only `knirv1...` addresses
  (`types.AddressFromString`), but the controller's `oracleAddress` helper produces `0x...`;
  the extension uses the bech32 address the Oracle requires. The controller's
  `useBackend` transfers look wrong for the same reason and should be checked.

### Wallet extension: storage baseline and NRN sends (follow-up)

- Storage is one baseline model (`KNIRVWALLET_DATA`, version 1); the Adena-era migrations
  v001–v009 are gone. A later schema change adds `Migration` version 2.
- Names are KNIRV's throughout: `knirv-native` / `contract-token` / `nft` token types,
  `ACCOUNT_NFT_*` storage keys, `NetworkMetainfo` without `indexerUrl`/`gnoUrl`/`apiUrl`.
- NRN sends (the transfer screen, and a dApp's `knirv_sendTransaction` for a single NRN
  `MsgSend`) go through `TransactionService.sendNativeTransfer`: a `knirv.oracle` /
  `oracle-transfer` envelope over `transfer:from:to:amount:nonce`, posted to KNIRVORACLE
  `/oracle/v3/token/transfer/signed`. Ledger accounts and other message types still use the
  broadcast path, which is unconfirmed against the gateway.
- Registration: the controller's onboarding now registers the new account with KNIRVORACLE
  (`services/oracleRegistration.ts`; retry or skip on failure). The extension registers the
  current account on first load of the wallet screen (`use-register-wallet`) and offers a
  button on the account-details screen.
- Explorer: "View in explorer" opens the network gateway's own `/chain-explorer` page
  (`chains.json` `linkUrl`); there is no separate scan host and no per-account route.
- NFT screens: the inherited collection/asset/transfer screens are removed. The NFT tab lists
  badge credentials and event bundles the user added by id, read from KNIRVCHAIN.
- Fees: NRN sends show a 0 NRN fee (the Oracle charges none). Ledger accounts cannot send NRN
  (they cannot sign the Oracle envelope) and the screen says so.
- Unconfirmed: nothing in this repo serves `/api/transmission/broadcast`, which the
  non-NRN transaction path (and the SDK's `transmission.broadcast`) posts to.
