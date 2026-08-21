# SDK Bindings Implementation Plan

## Goal

Make the Rust SDK the shared, performance-critical core for the existing Go,
TypeScript, and Python SDKs. Each language binding will be a thin, idiomatic
adapter around a stable Rust API rather than a separate implementation of
transaction, gateway, signing, wallet, and service behavior.

The Rust crate remains usable directly. It is now both a library and a binary:
`src/lib.rs` owns all reusable behavior, while `src/main.rs` is a thin CLI over
that library. Bindings must not expose Rust pointers, futures, Tokio handles,
or Rust error types as public language-level APIs.

## Decisions and boundaries

- Keep `knirv-sdk` as the domain/core crate. It owns request construction,
  failover, signing, serialization, validation, crypto, and service clients.
- Put each foreign interface in a dedicated crate/package. The C ABI is the
  lowest-level common boundary; PyO3, napi-rs, and wasm-bindgen adapt the core
  directly where that produces a safer native experience.
- TypeScript must support browser and edge runtimes: browsers, Cloudflare
  Workers, Deno, Vercel Edge, Bun, and Node.js. WebAssembly is therefore a
  required distribution target, not a deferred optional package.
- The Rust core owns Tokio. Bindings invoke a private runtime bridge and expose
  native synchronous calls, callbacks, or promises—never a Rust `Future`. The
  WASM build does not include the native Tokio runtime; it uses the host event
  loop through `wasm-bindgen-futures`.
- Use JSON, protobuf, or bincode bytes at FFI boundaries. Do not mirror complex
  Rust structs in C, Go, Python, or JavaScript memory layouts.
- Treat the four WebAssembly modules as versioned SDK artifacts. The core embeds
  immutable bytes for `cognitive-shell`, `controller-relay`, `crypto-core`, and
  `dve-verifier`; it provides a zero-copy registry and a digest-pinned manifest.
  The SDK never executes a module merely because it is embedded.
- Start with the stable service operations that already exist in `KnirvClient`.
  Add new bindings only after Rust tests establish their behavior.

## Proposed layout

```text
KNIRVSDK/
  core/                         # library + knirv-sdk binary
    src/lib.rs                   # reusable Rust SDK surface
    src/main.rs                  # CLI; imports only knirv_sdk library APIs
    src/wasm_modules.rs          # typed embedded-module registry
    wasm-modules/
      assets/                    # four stable, digest-pinned .wasm artifacts
      <module>/                  # source/build definition for each module
  bindings/
    c-abi/                       # cdylib/staticlib and C-compatible API
    python/                      # PyO3 package + maturin configuration
    node/                        # napi-rs package + platform packages
    wasm/                        # wasm-bindgen browser/edge package
    typescript/                  # isomorphic TS facade and conditional exports
    go/                          # Go wrapper, headers, and prebuilt libraries
  bindings-tests/                # language-neutral fixtures and vectors
```

`rust-bindings` may become a Cargo workspace after the first binding is
introduced. Until then, keep this crate independent so the existing direct Rust
SDK stays simple to consume.

## Embedded WASM module contract

The current library surface is the canonical native host contract:

```rust
use knirv_sdk::{materialize_wasm_module, WasmModule};

let bytes: &'static [u8] = WasmModule::CryptoCore.bytes(); // zero-copy
let digest: &'static str = WasmModule::CryptoCore.sha256();
materialize_wasm_module("crypto-core", "./crypto-core.wasm")?;
```

`WasmModule` is the sole stable selector. Its identifier, byte length, SHA-256,
ABI version, module kind, and capabilities must become a versioned manifest;
bindings and external codebases must not reach into `wasm-modules/assets` or a
Rust `target/` directory. `src/main.rs` is intentionally only an operational
tool over this API:

```text
knirv-sdk list
knirv-sdk extract <module> <destination>
```

The CLI adds no alternate implementation and must not become a required runtime
dependency for Rust, Go, Python, TypeScript, or WASM hosts. A host imports the
library directly and uses the bytes with zero copying; file extraction is only
for hosts that explicitly require a module file.

## Phase 1 — Stabilize the Rust core contract

1. Define a binding-safe facade in `knirv-sdk`, separate from the broad public
   client API. It should accept serializable request types and return serializable
   response types for:
   - transaction, gateway, and transmission operations;
   - wallet/controller approval operations;
   - signing, verification, address, crypto, and relay-envelope operations.
2. Create versioned request/response envelopes:

   ```json
   { "version": 1, "operation": "transaction.chain", "payload": {} }
   ```

   The response envelope carries either a JSON payload or a structured error
   code. This keeps compatibility independent of language-specific models.
3. Expand Rust test vectors for every bound signing/crypto operation and for
   representative service requests. Store fixtures in `bindings-tests/` and
   consume the same vectors from every language package.
4. Establish semver rules: additive envelope fields are allowed; changing an
   operation, error code, wire format, or ownership rule is a major-version
   change.
5. Split platform-dependent concerns behind internal traits/features before
   introducing bindings:
   - native builds use Tokio, native TLS/HTTP, OS time, and OS entropy;
   - `wasm32-unknown-unknown` uses `wasm-bindgen-futures`, Fetch-compatible
     transport, `web-time` (or equivalent), and `getrandom` with its `js`
     feature;
   - shared signing, serialization, validation, crypto, and state logic must
     compile without native threads, a Tokio reactor, or direct OS APIs.
6. Promote the embedded-module registry into a binding-safe manifest API:
   - add immutable metadata for each module: identifier, semantic artifact
     version, module kind, ABI version, byte length, SHA-256, and capabilities;
   - make `bytes()` the zero-copy native API and offer copy-out APIs only where
     a foreign runtime cannot safely borrow Rust-owned memory;
   - retain `materialize` only for native targets behind a filesystem feature or
     `cfg`, returning a clear unsupported-platform error in WASM builds;
   - add `verify` support to the library and CLI, validating magic bytes,
     embedded length, digest, and expected ABI/module-kind exports before a
     module is released to a host.
7. Add a reproducible module-import pipeline. Each `wasm-modules/<id>` source
   project builds to `wasm32-unknown-unknown`; a checked-in build script copies
   the release artifact to `wasm-modules/assets/<id>.wasm`, regenerates the
   manifest digest/length, and rejects an unexpected artifact. Nested `target/`
   directories remain ignored and are never imported by SDK consumers.

## Phase 2 — Shared C ABI

Build `knirv-sdk-c-abi` as both `cdylib` and `staticlib`, using `cbindgen` to
generate the checked-in C header.

### ABI shape

Expose opaque handles and byte buffers only:

```c
typedef struct knirv_engine knirv_engine_t;
typedef struct { const uint8_t *ptr; size_t len; } knirv_bytes_t;
typedef struct { int32_t code; knirv_bytes_t message; } knirv_error_t;

knirv_status_t knirv_engine_new(const knirv_config_t *, knirv_engine_t **out);
knirv_status_t knirv_engine_call(
    knirv_engine_t *, knirv_bytes_t request_json, knirv_bytes_t *response_json);
void knirv_engine_free(knirv_engine_t *);
void knirv_bytes_free(knirv_bytes_t);
void knirv_error_free(knirv_error_t);
```

- Every allocated handle or output buffer has one explicit matching free
  function. Allocation ownership is documented beside every export.
- Inputs are borrowed only for the duration of a call. Outputs are copied or
  returned as Rust-owned buffers released by `knirv_bytes_free`.
- Null pointers, invalid UTF-8, invalid lengths, and double-free attempts must
  return a structured status; they must never invoke undefined behavior.
- Generate a language-neutral error enum with stable integer codes, for example
  `INVALID_ARGUMENT`, `AUTHENTICATION`, `TIMEOUT`, `TRANSPORT`, `API`,
  `CRYPTO`, `INTERNAL_PANIC`.
- Add C-ABI module APIs that return metadata plus an SDK-owned byte buffer. The
  caller releases copied bytes with `knirv_bytes_free`; do not expose an opaque
  pointer to the static Rust `include_bytes!` allocation as foreign-owned memory.

### Safety boundary

Every exported `extern "C"` function must use a single boundary helper that:

1. validates pointers and lengths before dereferencing;
2. calls `std::panic::catch_unwind(AssertUnwindSafe(...))`;
3. maps every panic to `INTERNAL_PANIC` and a safe message;
4. does not let Rust unwind across FFI; and
5. clears/replaces any output pointer before returning an error.

Native C-ABI artifacts must retain an unwinding-capable panic strategy so the
boundary helper can convert panics to `INTERNAL_PANIC`; `panic = "abort"` would
make `catch_unwind` ineffective and terminate the host. Tests must intentionally
panic inside a boundary operation and prove that the mapped error is returned.
The WASM distribution may use `panic = "abort"` for size, because its public
surface does not cross a native C ABI.

### Tokio isolation

`knirv_engine_t` owns an internal Tokio runtime (or a private runtime service).
C ABI calls block on that runtime and return bytes synchronously. The C ABI
never returns a Tokio task, a `JoinHandle`, or a borrowed response tied to an
async lifetime. Host languages own their own concurrency policy.

## Phase 3 — Python binding

1. Create `knirv-sdk-python` with PyO3 and `pyo3-asyncio-0-21`.
2. Wrap the Rust core in Python classes with explicit `close()` plus normal
   PyO3 `Drop` cleanup. `__del__` is only a fallback, never the lifecycle
   guarantee.
3. Map asynchronous core operations to native awaitable Python methods via
   PyO3 async task support; keep CPU-only crypto helpers synchronous.
4. Convert stable Rust errors into a package exception hierarchy while exposing
   code, message, HTTP status, and retryability.
5. Build and publish manylinux, musllinux, macOS (x86_64/aarch64), and Windows
   wheels with `maturin`.
6. Expose module metadata and a `module_bytes(name)` method. The Python binding
   returns Python-owned `bytes`; `materialize_module(name, path)` is native-only
   convenience behavior and never relies on `__del__` for file lifecycle.

Acceptance: `pip install knirv-sdk` provides the same signing vectors and
representative transaction/gateway behavior as direct Rust calls.

## Phase 4 — TypeScript binding

### Distribution strategy

Publish one isomorphic TypeScript API with conditional exports:

- Browser and edge runtimes load the required `wasm-bindgen` build.
- Node.js and Bun load a napi-rs `.node` addon when the matching platform
  package is available, then fall back to the same WASM build when it is not.

This hybrid approach preserves native throughput for Node/Bun while providing a
single supported API for browsers, Cloudflare Workers, Deno, Vercel Edge, and
other WASM-capable hosts. WASM is the portability baseline; the native addon is
an optimization, not a functional dependency.

The TypeScript facade also owns a portable module-loading API. Native Node/Bun
may request copied bytes from the Rust addon or invoke the SDK CLI during build
tooling, but application code uses the TypeScript facade—not filesystem paths
inside the Rust crate.

### Node.js and Bun

1. Create a napi-rs package that owns a `KnirvClient`/engine handle.
2. Map non-blocking core work to JavaScript `Promise`s using `AsyncTask` or
   `napi::Env::spawn`. Do not block the Node event loop.
3. Use `FinalizationRegistry` only as a fallback for native handle cleanup;
   provide an explicit `close()` method.
4. Publish a small dispatcher package and platform packages, for example
   `@knirv/core-darwin-arm64`, `@knirv/core-linux-x64-gnu`, and Windows
   equivalents. The dispatcher selects the correct binary at install/runtime.
5. Expose `getWasmModule(name)` and `verifyWasmModule(name, bytes)` with the
   same manifest identifiers and hashes as Rust. Return a `Uint8Array`/Promise;
   use `FinalizationRegistry` only for native addon handles, never to manage
   WASM byte ownership.

### Browser and edge WASM

1. Create `knirv-sdk-wasm` with `wasm-bindgen` and compile it for
   `wasm32-unknown-unknown` using `wasm-pack`.
2. Expose browser-compatible APIs only. The generated package must not depend
   on a Node native addon, direct OS threads, native TLS, or a Tokio reactor.
3. Define an explicit asynchronous initialization contract for direct browser
   and edge use. The TypeScript facade will expose `SdkClient.create(config)`,
   which awaits the generated WASM `init()` function before creating an engine.
   It must accept the standard wasm-bindgen initialization inputs: a URL,
   `Response`, `ArrayBuffer`, `WebAssembly.Module`, or a bundler-managed module.
4. Publish separate `wasm-pack` outputs for `web`, `bundler`, and Node fallback
   use as appropriate. The isomorphic facade selects the correct entry through
   package export conditions; consumers must not import generated target files
   directly.
5. Make networking an explicit transport abstraction. In WASM it will use host
   Fetch (directly via `web-sys` or a compatible abstraction); native builds can
   retain `reqwest`. No native `reqwest` TLS feature may be enabled in the WASM
   dependency graph.
6. Configure size-focused WASM releases:

   ```toml
   [profile.release]
   opt-level = "z"
   lto = true
   codegen-units = 1
   panic = "abort"
   strip = "symbols"
   ```

   Then run `wasm-opt -Oz` after `wasm-pack build --release`. Record the final
   compressed and uncompressed `.wasm` sizes in CI and enforce a reviewed size
   budget suitable for Cloudflare Workers and browser delivery.
7. Test the package in real browser, Cloudflare Workers, Deno, Vercel Edge, and
   Bun environments. Include explicit initialization, cryptographic vectors,
   host Fetch transport, failed transport, and disposal/GC fallback coverage.
8. Do not blindly nest the four artifact bytes inside the SDK's own generated
   `wasm-bindgen` binary. That duplicates module payloads and can exceed browser
   and Cloudflare Workers size budgets. Instead publish a companion
   `@knirv/wasm-modules` ES package with content-addressed module assets and a
   manifest; the TypeScript facade loads the requested module by URL, `Response`,
   or bundler asset import and verifies its SHA-256 before instantiation. A
   feature-gated fully embedded browser build is permitted only when its size is
   measured and accepted for the target deployment.

## Phase 5 — Go binding

1. Generate `knirv.h` using `cbindgen` and write a small CGO wrapper that owns
   `knirv_engine_t` handles.
2. Keep each CGO call synchronous. Go callers use goroutines, contexts, and
   channels for concurrency; never export the Rust runtime to Go.
3. Provide idiomatic `Close()` methods and use `runtime.SetFinalizer` only as a
   fallback. Clear finalizers during explicit close.
4. Convert C buffers immediately to Go-owned `[]byte`/strings and call the
   C-ABI free function before returning to the caller.
5. Embed per-target prebuilt static libraries in the Go module, using scoped
   `#cgo LDFLAGS: -L${SRCDIR}/libs/<target> -lknirv_sdk` directives. A Go SDK
   consumer must not need Rust installed.
6. Provide `ModuleBytes(name)` and `WriteModule(name, path)` wrappers. The
   former returns a Go-owned byte slice; the latter is explicit native-only I/O.
   Go must verify the shared manifest digest before handing module bytes to a
   runtime.

## Phase 6 — CI, artifacts, and release

1. Build the C ABI and native bindings for:
   - `x86_64-unknown-linux-gnu`;
   - `aarch64-unknown-linux-gnu`;
   - `x86_64-apple-darwin`;
   - `aarch64-apple-darwin`;
   - Windows x86_64.
2. Use `cargo-zigbuild` where Zig supports the target and `cross` for
   Dockerized builds that need it. Use native macOS runners for Apple artifacts
   when code signing or platform validation requires them.
3. Run ABI compatibility checks against the generated header and bindgen
   smoke tests. Test Python wheels, npm packages, and Go static linking on each
   supported OS/architecture.
4. Rebuild all four source modules from clean `wasm32-unknown-unknown` targets,
   regenerate the embedded manifest, and compare SHA-256/length against the
   committed assets. Fail releases when an artifact changes without an explicit
   manifest/version update. Run `wasm-tools validate` and export/ABI checks for
   every artifact.
5. Generate SBOMs, checksums, provenance attestations, and signed release
   artifacts for the SDK, the module manifest, and each `.wasm` asset. Publish
   only after all language fixtures match the Rust vectors.
6. Version the Rust crate, C ABI, PyPI package, npm dispatcher/platform
   packages, Go module, and WASM module manifest from one release manifest.

## Security and quality gates

- Fuzz C-ABI byte inputs, malformed JSON/protobuf, null-pointer paths, and
  every explicit free function.
- Use Miri/sanitizers where applicable to catch use-after-free and leak paths.
- Add tests that intentionally panic inside a boundary operation and assert the
  host receives `INTERNAL_PANIC` without process termination.
- Test cancellation, timeout, and shutdown of the private Tokio runtime.
- Verify no secret/private-key material is logged, serialized in diagnostics,
  or retained after a failed call.
- Treat embedded WASM as executable supply-chain material: pin module bytes by
  digest, validate their WebAssembly format and declared exports, attach source
  revision/build-tool provenance, and require review for every binary update.
- Do not auto-extract or auto-instantiate a module. The caller selects a module,
  receives verified bytes, and controls its destination/runtime and permissions.
- Keep language bindings thin: business rules remain in Rust, and binding code
  should contain conversion, lifecycle, and language-native async glue only.

## Rollout order

1. Stabilize the Rust facade and fixtures.
2. Ship the C ABI plus Go wrapper, because it defines the lowest common
   ownership contract.
3. Ship Python with PyO3/maturin.
4. Ship Node/Bun with napi-rs.
5. Ship the required wasm-bindgen package, `@knirv/wasm-modules` asset package,
   and isomorphic TypeScript facade.

## Open decision

**Should Rust perform network I/O directly, or should it provide strictly local
state, compute, and crypto while the host runtime performs I/O?**

The recommended default is a transport trait owned by the Rust core: native
bindings use a `reqwest` implementation, while the WASM binding uses host Fetch.
If host-owned I/O is chosen instead, bind requests/responses as raw bytes and
keep all network-policy behavior (timeouts, failover, authentication headers,
and retries) in the TypeScript/Python/Go wrappers. This decision must be made
before the binding-safe facade is finalized because it determines the API and
test-vector boundary.
