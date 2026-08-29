# KNIRV SDK

The KNIRV SDK is a Rust core and a set of bindings for working with KNIRV
network services, canonical signing formats, and verified KNIRV WASM modules.
The maintained packages in this directory are listed below.

| Package | Location | What it provides |
| --- | --- | --- |
| Rust SDK | [`core/`](core/README.md) | Async clients for KNIRV services, signing, crypto, transmission, and embedded WASM modules. |
| C ABI | [`bindings/c-abi/`](bindings/c-abi/include/knirv.h) | A C-compatible, versioned JSON/protobuf boundary over the Rust core. |
| Go binding | [`go-package/`](go-package/README.md) | An idiomatic Go facade backed by the C ABI. |
| Python binding | [`bindings/py/`](bindings/py/knirv_sdk/__init__.py) | A `ctypes` adapter over a caller-supplied C ABI library. |
| TypeScript package | [`npm-package/`](npm-package/README.md) | Browser and edge WASM package published as `@knirv/sdk`. |

## What is included

- Unified Rust clients for transaction, gateway, wallet, oracle, governance,
  transmission, actuarial, and supporting KNIRV services.
- Canonical Cosmos `SIGN_MODE_DIRECT` transaction signing, message-envelope
  signing, KNIRV bech32 addresses, and relay-envelope encoding.
- A stable binding envelope (`version: 1`) for local cryptography, signing,
  address, WASM, and selected network operations.
- Four digest-pinned WASM assets: `cognitive-shell`, `controller-relay`,
  `crypto-core`, and `dve-verifier`.

The browser/edge WASM transport currently implements local `crypto.sha256` and
`wasm.manifest`; use a supplied `BindingTransport` when an application needs a
native or remote implementation of additional binding operations.

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

The Go module path is `github.com/guiperry/knirv-sdk-go`. It includes a static
Linux AMD64 archive for the private CGO layer. Build a fresh archive for the
current host with `go-package/scripts/build-native.sh` when necessary.

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

### Python

The Python adapter is source in `bindings/py/`. It requires Python 3.9 or
newer and a platform C ABI library supplied through `KNIRV_SDK_LIBRARY` (or the
`library=` constructor argument):

```python
from knirv_sdk import Client

with Client() as sdk:
    print(sdk.call("crypto.sha256", {"data": "knirv"}))
```

Build the native library from `bindings/c-abi/` before using this adapter.

## Binding contract

The C ABI and language adapters use JSON request envelopes with API version 1:

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

The Rust core embeds the four modules and can inspect or materialize them with:

```sh
cd core
cargo run --bin knirv-sdk -- list
cargo run --bin knirv-sdk -- extract crypto-core ./crypto-core.wasm
cargo run --bin knirv-sdk -- verify crypto-core ./crypto-core.wasm
```

The npm package publishes the same module assets in `modules/` with a
digest-pinned `modules/manifest.json`. `SdkClient.moduleBytes()` verifies an
asset against that manifest before returning it.

## Development and verification

Run checks from the individual package directories:

```sh
(cd core && cargo test)
(cd bindings/c-abi && cargo test)
(cd go-package && go test ./...)
(cd npm-package && npm test)
```

The Go binding's native archive must match the host architecture. The npm test
builds its WASM bindings before running its binding fixtures.
