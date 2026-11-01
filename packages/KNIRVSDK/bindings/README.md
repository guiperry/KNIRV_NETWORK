# KNIRV SDK bindings

The Rust core in `../core` is the only implementation. This directory holds the
build wrappers that run it as WebAssembly.

- `wasi/` builds the core as a `wasm32-wasip1` module with the three-function
  ABI (`knirv_alloc` / `knirv_free` / `knirv_invoke`). The Go package
  (`../go-package/`) runs it in-process under wazero (`internal/core`), with
  `CGO_ENABLED=0`.
- The TypeScript package (`../npm-package/`, `@knirv/sdk`) runs the wasm-bindgen
  build in `../npm-package/wasm-core`.
- `../bindings-tests/envelopes.json` is the language-neutral fixture set that
  both hosts run.

There is no C ABI and no Python binding; both were removed (the WASM builds cover
every operation, including the network ones).
