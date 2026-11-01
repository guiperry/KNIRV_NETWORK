# KNIRV SDK

The KNIRV SDK is a Rust core and a set of bindings for working with KNIRV
network services, canonical signing formats, and verified KNIRV WASM modules.
The maintained packages in this directory are listed below.

| Package | Location | What it provides |
| --- | --- | --- |
| Rust SDK | [`core/`](core/README.md) | Async clients for KNIRV services, signing, crypto, transmission, and embedded WASM modules. |
| WASI module | [`bindings/wasi/`](bindings/wasi/src/lib.rs) | The Rust core as a `wasm32-wasip1` module with a three-function ABI. The artifact the Go package runs. |
| Go binding | [`go-package/`](go-package/README.md) | An idiomatic Go facade that runs the WASI module in-process under wazero (no CGO). |
| TypeScript package | [`npm-package/`](npm-package/README.md) | Browser and edge WASM package published as `@knirv/sdk`. |

## Development convention: Rust core first, bindings only

**The Rust core ([`core/`](core/README.md)) is the single source of truth for
all SDK functionality.** The Go and TypeScript packages are
bindings: they expose what the core implements and nothing more. This applies
to every contributor, human or agent.

Rules:

1. **Implement in `core/` first.** Any new behavior (service clients, signing,
   URI/address handling, crypto, validation, request construction, failover,
   serialization, model/wire contracts) is written in Rust in `core/src/`, with
   Rust tests, before any other language sees it.
2. **Expose it via bindings only.** Add the operation to the binding contract
   in [`core/src/bindings.rs`](core/src/bindings.rs) /
   `core/src/bindings/`, then surface it through the WASM builds:
   `bindings/wasi` (run under wazero by the Go package) and `npm-package/wasm-core`
   (wasm-bindgen, for TypeScript). Add a cross-language fixture to
   [`bindings-tests/envelopes.json`](bindings-tests/envelopes.json).
3. **Go and TypeScript packages are thin facades.** They may add idiomatic
   types, error mapping, context/promise handling, and packaging. They must
   not reimplement business logic, protocol handling, hashing, signing,
   encoding, or validation that exists (or belongs) in the core. If you find
   yourself writing such logic in `go-package/` or `npm-package/`, stop and move
   it to Rust.
4. **Do not leak Rust internals.** Bindings exchange versioned JSON/protobuf
   bytes; never expose Rust pointers, futures, Tokio handles, or Rust error
   types as public language-level APIs. See
   [`core/SDK_Bindings.md`](core/SDK_Bindings.md) for the boundary design.
5. **Fixing a bug means fixing the core.** Patch the Rust implementation and
   rebuild the bindings rather than patching a binding's copy of the logic.
6. **Parity is verified through shared vectors.** Behavior shared by several
   languages is tested against the same fixtures (`bindings-tests/`,
   `testvectors/`), so the languages cannot drift apart.

Pre-existing pure-Go or pure-TypeScript logic that predates this convention is
technical debt to be migrated into the core, not a pattern to copy. Do not add
to it.

Checklist for a new SDK feature:

- [ ] Implemented and unit-tested in `core/`
- [ ] Operation added to the binding envelope contract (`version: 1`)
- [ ] Exposed through the WASM build(s)
- [ ] Thin Go and TypeScript wrappers added with no duplicated logic
- [ ] Shared fixture added; `cargo test`, `go test`, and `npm test` pass

## What is included

- Unified Rust clients for transaction, gateway, wallet, oracle, governance,
  transmission, actuarial, and supporting KNIRV services.
- Canonical Cosmos `SIGN_MODE_DIRECT` transaction signing, message-envelope
  signing, KNIRV bech32 addresses, and relay-envelope encoding.
- A stable binding envelope (`version: 1`) for local cryptography, signing,
  address, WASM, and selected network operations.
- Three digest-pinned WASM assets: `controller-relay`, `crypto-core`, and
  `proof-verifier`.

Local operations run synchronously inside the WASM core. The five network
operations (`transaction.chain`, `gateway.health`, `transmission.resolve_uri`,
`transmission.broadcast`, `wallet.account`) also run in the core, which owns
request construction, retries, failover and error mapping; the host supplies only
an `http_request` function that moves bytes (global `fetch` by default in
TypeScript, `net/http` in Go). See `core/src/http_transport.rs` for the wire
contract.

## Quick starts

### Rust

Use the local crate while developing in this repository:

```toml
[dependencies]
knirv-sdk = { path = "KNIRV_NETWORK/packages/KNIRVSDK/core" }
tokio = { version = "1", features = ["macros", "rt-multi-thread"] }
```

```rust,no_run
use knirv_sdk::{ClientConfig, KnirvClient, Network};

#[tokio::main]
async fn main() -> knirv_sdk::Result<()> {
    let sdk = KnirvClient::new(ClientConfig {
        network: Network::PublicTestnet,
        ..Default::default()
    })?;
    let chain = sdk.transaction.chain().await?;
    println!("{}", chain.chain_id);
    Ok(())
}
```

`ClientConfig` supports API keys, timeouts, retry settings, headers, and service
URL overrides. Its default configuration also reads
`KNIRVCHAIN_TRANSACTION_SDK_API_KEY`,
`KNIRVCHAIN_TRANSACTION_SDK_BASE_URL`, and `KNIRVGATEWAY_BASE_URL`.

### TypeScript and JavaScript

The `@knirv/sdk` package requires Node.js 18 or later. Its default entry point
contains an inline WASM fallback:

```js
import { SdkClient } from "@knirv/sdk";

const sdk = await SdkClient.init();
console.log(await sdk.sha256Hex("knirv"));
```

For a CDN, Worker module binding, or bundler-managed asset, use the slim entry
point and pass the WASM source explicitly:

```js
import { SdkClient } from "@knirv/sdk/slim";

const sdk = await SdkClient.init(
  new URL("./knirv_sdk_core_bg.wasm", import.meta.url),
);
```

`@knirv/sdk/signing` is a separate browser- and Node-compatible entry point;
it does not require WASM initialization. `@knirv/sdk/actuarial` exposes a
fetch-based client for KNIRVSERVER's actuarial API.

### Go

The Go module path is `github.com/guiperry/knirv-sdk-go`. It runs the Rust core as a WASI
module under wazero: no CGO and no native library. The module is downloaded on
first use (see `go-package/README.md`); `make build-wasm` rebuilds it.

```go
package main

import (
    "context"
    "fmt"

    knirv "github.com/guiperry/knirv-sdk-go"
)

func main() {
    client := knirv.NewClient()
    result, err := client.Execute(
        context.Background(),
        "crypto.sha256",
        []byte(`{"data":"knirv"}`),
    )
    if err != nil {
        panic(err)
    }
    fmt.Println(string(result))
}
```

`ModuleManifest`, `ModuleBytes`, and `WriteModule` return or write only
SHA-256-verified WASM module bytes.

## Binding contract

The WASM hosts (Go and TypeScript) use JSON request envelopes with API version 1:

```json
{
  "version": 1,
  "operation": "crypto.sha256",
  "payload": { "data": "knirv" }
}
```

Responses preserve the version and operation, returning either `payload` or a
structured `error` with a code, message, optional HTTP status, and retry flag.
The supported Rust-core operations are defined in
[`core/src/bindings.rs`](core/src/bindings.rs); cross-language fixtures live in
[`bindings-tests/envelopes.json`](bindings-tests/envelopes.json).

## WASM modules

Three raw-ABI modules (no imports, no JS glue) are built from `core/wasm-modules/`
and described by **one generated manifest**, `core/wasm-modules/manifest.json`
(digest, length, kind, ABI, capabilities). The manifest is produced by the build
from `core/wasm-modules/modules.json` plus the built bytes; Rust, the npm package
and the release tooling all read it.

| Module | Kind | What it is |
| --- | --- | --- |
| `crypto-core` | crypto | The wallet vault: BIP-39, SLIP-10, Argon2id + AES-256-GCM envelope, signing. Secrets never leave the module. Bundled in the npm package (`@knirv/sdk/vault`). |
| `controller-relay` | relay | Relay-envelope validation and signing. |
| `proof-verifier` | verifier | Event-log and Merkle verification (the same sources are compiled into the core as `evidence.*`). |

The module bytes are release artifacts, published beside the core at
`https://releases.knirv.com/sdk/<version>/modules/<id>.wasm`. Only `crypto-core`
ships inside the npm tarball; `SdkClient.moduleBytes()` (and
`fetchWasmModule`) fetch the others from the release URL by default and verify
them against the manifest. The Go package no longer serves module bytes.

A native Rust host can embed the bytes (`embedded-modules` feature, on by default
only for native builds) and use them with:

```sh
cd core
cargo run --bin knirv-sdk -- list
cargo run --bin knirv-sdk -- extract crypto-core ./crypto-core.wasm
cargo run --bin knirv-sdk -- verify crypto-core ./crypto-core.wasm
```

## Two builds, one core

Two thin builds of the same core crate exist, because a browser or edge host
cannot make the synchronous host call the WASI build is designed around (`fetch`
is asynchronous). They share the crate, the operation table, the feature set below
and one fixture suite; `make parity` runs `bindings-tests/{parity,network}.json`
through both and fails on any difference.

| Build | Target | Used by | Network seam | Features |
| --- | --- | --- | --- | --- |
| `bindings/wasi` | `wasm32-wasip1` | Go (wazero) | sync import `knirv_host.http_request` | `secrets`, `module-manifest` |
| `npm-package/wasm-core` | `wasm32-unknown-unknown` + wasm-bindgen | TypeScript | async `invoke_async` + `HttpHost` | `secrets`, `module-manifest`, `js-rng` |
| native | host | Rust callers, `knirv-sdk` CLI | reqwest | defaults (`http-reqwest`, `filesystem`, `embedded-modules`, ...) |

`vault` is built only into the `crypto-core` module, so the general cores do not
carry Argon2id or BIP-39. Every build panics as a trap (`panic = "abort"`); Go
discards a trapped instance, and `npm-package/scripts/test-no-traps.mjs` checks
the TypeScript path does the same.

## Limits

* **Cancellation.** Go honours a context while waiting for an instance, before a
  call starts, and during a network operation (the request is aborted in flight).
  Pure computation inside the module is not interruptible: wazero's per-loop
  interruption check measured 30-65% slower on signing and scrypt. This matters
  for `authority.mine_header`, which searches until it finds a witness: do not
  pass it a difficulty you do not control.
* **First run.** Go downloads its core once per machine and version (see
  `go-package/README.md`). A host with no network and an empty data directory
  must be seeded.

## Release

```sh
make build-sdk     # build, stamp, test Rust/Go/npm, check parity and digests
make release-sdk   # the same, then publish to R2 (needs an rclone remote `incline`)
```

## Development and verification

Run checks from the individual package directories, or everything with
`./scripts/run-all-tests.sh`:

```sh
(cd core && cargo test --all-features)
make test-go parity          # Go against the built core; Go vs TypeScript
(cd npm-package && npm test) # builds the WASM bindings first
```
