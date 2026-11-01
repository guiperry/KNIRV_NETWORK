# KNIRV SDK for Rust

An asynchronous Rust implementation of the KNIRV SDK. It ports the most complete
public capabilities of the Go and TypeScript SDKs: Transaction/OpenAPI operations,
Gateway Economics and PoAuD services, the unified network client, controller-custodied
wallet approvals, and canonical Cosmos `SIGN_MODE_DIRECT` signing/relay wire formats.

## Install

```toml
[dependencies]
knirv-sdk = { path = "KNIRV_NETWORK/packages/KNIRVSDK/core" }
tokio = { version = "1", features = ["macros", "rt-multi-thread"] }
```

## Use

```rust,no_run
use knirv_sdk::{ClientConfig, KnirvClient, Network};

#[tokio::main]
async fn main() -> knirv_sdk::Result<()> {
    let sdk = KnirvClient::new(ClientConfig { network: Network::PublicTestnet, ..Default::default() })?;
    let chain = sdk.transaction.chain().await?;
    println!("{:?}", chain.chain_id);
    Ok(())
}
```

`ClientConfig` accepts an API key, request timeout/retry configuration, headers, and
per-service URL overrides. It reads `KNIRVCHAIN_TRANSACTION_SDK_API_KEY`,
`KNIRVCHAIN_TRANSACTION_SDK_BASE_URL`, and `KNIRVGATEWAY_BASE_URL` by default.

## Modules

- `transaction`: chain, block/transaction submission, pool, URI, node health/info,
  peers, and MCP capability operations.
- `gateway`: Economics skills/LLM/validation plus PoAuD proofs, challenges, reputation,
  routes, health, and integrations.
- `wallet`: controller approval polling, signing requests, URI resolution, and broadcast.
- `signing`: byte-compatible direct-sign document/action encoding, secp256k1 signing,
  KNIRV bech32 addresses, and relay envelope encoding.
- `wasm_modules`: three immutable, compile-time embedded KNIRV WASM modules.

`KnirvClient` exposes the complete surface through `transaction`, `gateway`,
`wallet`, `oracled`, `governance`, `crypto`, `badges`, `dve`, `treasury`, `agents`,
`network`, `factuality`, `health`, `config`, and `transmission`. Environment metadata
is available as `network_info`.

Requests made against KNIRV's canonical gateways automatically fail over in this
order after a transport failure, timeout, or 5xx response: production gateway,
testnet gateway, then local testnet. Custom base URLs never fail over unexpectedly.

## WASM modules

The SDK's three modules (`controller-relay`, `crypto-core`, `proof-verifier`) are
described by `wasm-modules/manifest.json`, which is always compiled in
(`module-manifest` feature), so digests and `WasmModule::verify` work everywhere.
Their bytes (about 0.8 MB) are embedded only with the `embedded-modules` feature,
on by default for native builds:

```rust
use knirv_sdk::WasmModule;

let wasm_bytes = WasmModule::CryptoCore.bytes(); // `embedded-modules`
WasmModule::CryptoCore.verify(&wasm_bytes)?;     // `module-manifest` only needs the bytes you give it
```

```rust,no_run
knirv_sdk::materialize_wasm_module("crypto-core", "./crypto-core.wasm")?;
```

The `knirv-sdk` binary uses the same library API:

```sh
cargo run --bin knirv-sdk -- list
cargo run --bin knirv-sdk -- extract crypto-core ./crypto-core.wasm
```

## Features

| Feature | Default | Adds |
| --- | --- | --- |
| `http-reqwest`, `rustls-tls` | yes | native HTTP transport. Without it a host must supply an `HttpTransport` (WebAssembly builds do) |
| `filesystem`, `secrets` | yes | key-file discovery and scrypt + AES-GCM decryption |
| `module-manifest` | yes | WASM module manifest and verification |
| `embedded-modules` | yes | module bytes (`include_bytes!`) |
| `vault` | no | the wallet vault; built only into the `crypto-core` module |
| `js-rng` | no | `crypto.getRandomValues` entropy for the wasm-bindgen build |

Run checks with `cargo test --all-features` from this directory.
