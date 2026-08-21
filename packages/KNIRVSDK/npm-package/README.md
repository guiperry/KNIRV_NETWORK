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

The package also exports the versioned `BindingTransport` and `Envelope` API.
Use `SdkClient.create(transport)` to connect a native or remote implementation
of the same Rust binding contract.

## Signing

`@knirv/sdk/signing` is a standalone, browser- and Node-compatible signing
entry point. It does not require WASM initialization:

```js
import { signMessageEnvelope, verifyMessage } from "@knirv/sdk/signing";

const signed = await signMessageEnvelope(privateKey, envelope);
await verifyMessage(signed, envelope.domain, envelope.purpose, envelope.chainId, envelope.nonce, new Date());
```

## Core WASM modules

The four core modules are published as separate `modules/*.wasm` assets with a
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
