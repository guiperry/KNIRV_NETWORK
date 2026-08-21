import { fetchWasmModule, wasmModuleMetadata, wasmModulesManifest } from "./modules.js";
export const BINDING_API_VERSION = 1;
export class BindingOperationError extends Error {
    bindingError;
    constructor(bindingError) {
        super(bindingError.message);
        this.bindingError = bindingError;
        this.name = "BindingOperationError";
    }
}
export class WasmBindingTransport {
    core;
    constructor(core) {
        this.core = core;
    }
    async call(request) {
        if (request.version !== BINDING_API_VERSION) {
            return failure(request.operation, "INVALID_ARGUMENT", `unsupported binding API version: ${request.version}`);
        }
        switch (request.operation) {
            case "crypto.sha256": {
                const data = payloadString(request.payload, "data");
                if (data === undefined)
                    return failure(request.operation, "INVALID_ARGUMENT", "data must be a string");
                return success(request.operation, { digest: this.core.sha256_hex(new TextEncoder().encode(data)) });
            }
            case "wasm.manifest":
                return success(request.operation, wasmModulesManifest);
            default:
                return failure(request.operation, "UNSUPPORTED", `unsupported operation: ${request.operation}`);
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
    /** Create a client around a native, remote, or application-provided transport. */
    static async create(transport) {
        return new SdkClient(transport);
    }
    /** @internal Creates the default browser/edge client after WASM initialization. */
    static fromWasm(core) {
        return new SdkClient(new WasmBindingTransport(core), core);
    }
    call(operation, payload = {}) {
        return this.transport.call({ version: BINDING_API_VERSION, operation, payload });
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
function payloadString(payload, name) {
    if (!payload || typeof payload !== "object")
        return undefined;
    const value = payload[name];
    return typeof value === "string" ? value : undefined;
}
