# @knirv/sdk

Browser and edge distribution of the portable KNIRV Rust core. The package
ships the generated wasm-bindgen glue, declarations, and `*.wasm` binary.

```js
import { SdkClient } from "@knirv/sdk";
const sdk = await SdkClient.init(); // uses the bundled inline fallback
const response = await sdk.sha256("knirv");
console.log(response.payload.digest);
```

For a CDN, bundler asset, or Cloudflare Worker module binding, avoid the inline
copy and provide the asset explicitly:

```js
import { SdkClient } from "@knirv/sdk/slim";
const sdk = await SdkClient.init(new URL("./knirv_sdk_core_bg.wasm", import.meta.url));
```

Run `npm run build`, then `npm run pack:check` before publishing. The latter
must list `wasm/knirv_sdk_core_bg.wasm`.

Network operations (`transaction.chain`, `gateway.health`, `transmission.*`,
`wallet.account`) run in the Rust core, which owns request construction,
retries, failover and error mapping. The host only moves bytes: pass
`SdkClient.init(source, { http, config })` to supply your own
`{ http_request(json): Promise<string>, sleep(ms): Promise<void> }` (default:
global `fetch`) and default client settings. `SdkClient.create` and
`BindingTransport` were removed in favour of this single seam.

## Initialising the core

The signing, URI, relay, model and frame functions all run in the Rust core, a
WebAssembly module that must be instantiated once.

* **Browser main thread:** `await initLocal()` once at start-up (browsers refuse to
  compile a module of this size synchronously). KNIRVARENA and the controller do it
  before rendering; the wallet module does it before it signs.
* **Node, web workers, most edge runtimes:** nothing to do; the first call compiles
  the bundled module. Calling `await initLocal()` anyway is harmless and idempotent
  (KNIRV.COM's registration route does).
* **Cloudflare Workers:** `await initLocal(wasmModule)` with a pre-compiled module.
* `SdkClient.init()` and `ModelsClient.advise()` initialise it for you.

## Wallet protocol

`@knirv/sdk/wallet` carries the message protocol of the KNIRV wallet extension:
the request kinds, the success/failure/reject outcomes with their codes and
messages (`WalletMessageInfo`), `WalletResponse`, and the `KnirvWalletProvider`
interface the extension injects as `window.knirv`. The protocol is written once, in
`core/data/wallet_protocol.json`; the build generates the TypeScript constants and
the core serves the same data as the `wallet.protocol` operation.

## Signing and URIs (Rust-backed since 0.2.0)

`@knirv/sdk/signing` and `@knirv/sdk/uri` are typed wrappers over the Rust core,
which is the single implementation of signing, addresses, message/relay/WASM
envelopes, `knirv://` URIs and authority records. No cryptography is written in
TypeScript, and the package has no JS crypto dependencies. The function names and
types are unchanged from 0.1.x:

```js
import { signMessageEnvelope, verifyMessage } from "@knirv/sdk/signing";

const signed = await signMessageEnvelope(privateKey, envelope);
await verifyMessage(signed, envelope.domain, envelope.purpose, envelope.chainId, envelope.nonce, new Date());
```

### Breaking change from 0.1.x: initialise the core once

The core is a WebAssembly module, so it must be instantiated before the first
call. What that takes depends on the platform:

| Platform | What to do |
| --- | --- |
| Node, web workers, most edge runtimes | Nothing. The first call compiles the bundled module synchronously. |
| Browser main thread | `await initLocal()` once at start-up. Browsers refuse to compile a module this size synchronously on the main thread. |
| Cloudflare Workers | `import wasm from "@knirv/sdk/wasm/knirv_sdk_core_bg.wasm"; await initLocal(wasm);` Workers forbid compiling WASM from bytes, so pass the pre-compiled module. |

`initLocal` is exported from `@knirv/sdk/signing` and `@knirv/sdk/uri`, is
idempotent, and is shared with `SdkClient.init()`. A call made before the core
can be instantiated throws `LocalCoreNotInitializedError` with these
instructions.

Other differences: failures are `BindingOperationError`s carrying the core's
message (they are still `Error`s), `verifyMessage` accepts an envelope whose
`address` field is empty (the address is derived from the public key), and the
WASM binary is about 0.7 MB instead of 28 KB.

## Core WASM modules

The three core modules are published as separate `modules/*.wasm` assets with a
digest-pinned `modules/manifest.json`; they are not nested inside the SDK WASM
runtime. After initialization, load a default asset or provide a CDN/Worker
asset explicitly:

```js
const { bytes, metadata } = await sdk.moduleBytes("crypto-core");
// or: await sdk.moduleBytes("crypto-core", myWorkerModuleBytes)
```

`moduleBytes` copies the bytes and verifies their SHA-256 against the published
manifest before returning them. `wasmModuleURL` and `wasmModulesManifest` are
also exported from `@knirv/sdk/modules` for hosts that manage loading directly.
