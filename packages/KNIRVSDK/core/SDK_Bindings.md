# SDK binding design

The Rust core is the only implementation of SDK behaviour. Go and TypeScript run
it as WebAssembly and add only idiomatic types, error mapping and packaging.
Python and a C ABI are not provided.

## Shape

- One JSON envelope contract (`bindings.rs`, `bindings/ops.rs`), API version 1:
  `{version, operation, payload}` in, `{version, operation, payload | error}` out.
- `bindings/wasi` builds the core for `wasm32-wasip1` (Go, via wazero).
- `npm-package/wasm-core` builds it with wasm-bindgen (TypeScript).
- Cross-language fixtures live in `bindings-tests/envelopes.json`.

## Network access

The core owns request construction, retries, failover and error mapping. A WASM
host only moves bytes through a host-provided `http_request` import (see
`http_transport.rs`): JSON request `{method, url, headers, body_base64,
timeout_ms}`, JSON response `{status, headers, body_base64}` or a transport
error (`timeout`, `dns`, `tls`, `refused`, `aborted`).

## Distribution

The Go package downloads the core from `https://releases.knirv.com/sdk/<version>/`
on first use and caches it in the user data directory; TypeScript inlines the
same artifact. See `../technical_debt.md`, Phase 3.
