import { toJson } from "./local.js";
import { fetchWasmModule, wasmModuleMetadata } from "./modules.js";
export const BINDING_API_VERSION = 1;
function hostError(kind, message) {
    return JSON.stringify({ error: { kind, message } });
}
/** The default host: global `fetch`, with `timeout_ms` enforced by an abort. */
export function fetchHost(fetchImpl = (...args) => fetch(...args)) {
    return {
        async http_request(requestJson) {
            const req = JSON.parse(requestJson);
            const controller = new AbortController();
            let timedOut = false;
            const timer = req.timeout_ms > 0 ? setTimeout(() => { timedOut = true; controller.abort(); }, req.timeout_ms) : undefined;
            try {
                const body = req.body_base64 ? Uint8Array.from(atob(req.body_base64), (c) => c.charCodeAt(0)) : undefined;
                const response = await fetchImpl(req.url, { method: req.method, headers: req.headers, body: req.method === "GET" || req.method === "HEAD" ? undefined : body, signal: controller.signal });
                const bytes = new Uint8Array(await response.arrayBuffer());
                let binary = "";
                for (let i = 0; i < bytes.length; i += 0x8000)
                    binary += String.fromCharCode(...bytes.subarray(i, i + 0x8000));
                const headers = {};
                response.headers.forEach((value, name) => { headers[name] = value; });
                return JSON.stringify({ status: response.status, headers, body_base64: btoa(binary) });
            }
            catch (error) {
                // fetch does not say why it failed; a timeout we caused is the only kind we can name.
                return hostError(timedOut ? "timeout" : "other", error instanceof Error ? error.message : String(error));
            }
            finally {
                if (timer !== undefined)
                    clearTimeout(timer);
            }
        },
        sleep: (ms) => new Promise((resolve) => setTimeout(resolve, ms)),
    };
}
export class BindingOperationError extends Error {
    bindingError;
    constructor(bindingError) {
        super(bindingError.message);
        this.bindingError = bindingError;
        this.name = "BindingOperationError";
    }
}
const NETWORK_OPERATIONS = new Set(["actuarial.call", "transaction.chain", "gateway.health", "transmission.resolve_uri", "transmission.broadcast", "wallet.account"]);
/** Runs envelopes against the WASM core; network operations use `host`. */
export class WasmBindingTransport {
    core;
    host;
    config;
    constructor(core, host = fetchHost(), config) {
        this.core = core;
        this.host = host;
        this.config = config;
    }
    async call(request) {
        if (request.version !== BINDING_API_VERSION) {
            return failure(request.operation, "INVALID_ARGUMENT", `unsupported binding API version: ${request.version}`);
        }
        switch (request.operation) {
            default:
                if (NETWORK_OPERATIONS.has(request.operation)) {
                    const withConfig = this.config && !request.config ? { ...request, config: this.config } : request;
                    return JSON.parse(await this.core.invoke_async(toJson(withConfig), this.host));
                }
                // Everything else the core can do without a network is forwarded to it;
                // anything it does not recognise answers UNSUPPORTED.
                return JSON.parse(this.core.invoke(JSON.stringify(request)));
        }
    }
}
export class SdkClient {
    transport;
    core;
    constructor(transport, core) {
        this.transport = transport;
        this.core = core;
    }
    /** @internal Creates the client after WASM initialization. */
    static fromWasm(core, options = {}) {
        return new SdkClient(new WasmBindingTransport(core, options.http, options.config), core);
    }
    /** `config` overrides the client's default settings for a network operation. */
    call(operation, payload = {}, config) {
        return this.transport.call({ version: BINDING_API_VERSION, operation, payload, ...(config ? { config } : {}) });
    }
    sha256(data) {
        return this.call("crypto.sha256", { data });
    }
    wasmManifest() {
        return this.call("wasm.manifest");
    }
    /** Returns verified, caller-owned module bytes. Module assets stay separate
     * from the SDK runtime WASM binary to preserve browser and edge size limits. */
    async moduleBytes(id, source) {
        if (!this.core) {
            throw new BindingOperationError({ code: "UNSUPPORTED", message: "module verification requires the WASM core transport", retryable: false });
        }
        const metadata = wasmModuleMetadata(id);
        const bytes = await fetchWasmModule(id, source);
        if (!this.core.sha256_matches(bytes, metadata.sha256)) {
            throw new BindingOperationError({ code: "CRYPTO", message: `digest verification failed for KNIRV WASM module: ${id}`, retryable: false });
        }
        return { metadata, bytes };
    }
    async sha256Hex(data) {
        if (this.core && data instanceof Uint8Array)
            return this.core.sha256_hex(data);
        const response = await this.sha256(typeof data === "string" ? data : new TextDecoder().decode(data));
        if (response.error)
            throw new BindingOperationError(response.error);
        const digest = response.payload?.digest;
        if (!digest)
            throw new BindingOperationError({ code: "INTERNAL_PANIC", message: "crypto.sha256 returned no digest", retryable: false });
        return digest;
    }
    async verifySha256(data, expectedHex) {
        if (this.core)
            return this.core.sha256_matches(typeof data === "string" ? new TextEncoder().encode(data) : data, expectedHex);
        return (await this.sha256Hex(data)).toLowerCase() === expectedHex.toLowerCase();
    }
    coreVersion() {
        return this.core?.core_version();
    }
}
function success(operation, payload) {
    return { version: BINDING_API_VERSION, operation, payload };
}
function failure(operation, code, message) {
    return { version: BINDING_API_VERSION, operation, error: { code, message, retryable: false } };
}
