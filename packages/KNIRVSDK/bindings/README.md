# KNIRV SDK bindings

This directory contains the new Rust-backed binding boundary. Existing SDKs in
`../go`, `../py`, and `../ts` remain in place as compatible legacy packages;
they are not moved because their public import paths are used across KNIRV.

- `c-abi/` is the common native boundary: opaque engine handles and JSON bytes.
- `py/` contains a thin language-facing adapter over the shared envelope
  contract. The Go facade is published from `../go-package/`, where its CGO
  implementation is compiler-enforced under `internal/ffi`; the TypeScript
  facade is published from `../npm-package/` as `@knirv/sdk`.
- `../bindings-tests/envelopes.json` is a language-neutral fixture set.

The Rust core lives at `../core`. There were no in-repository Cargo consumers
of the prior `rust/` path; the C ABI dependency above is updated atomically with
this layout change.
