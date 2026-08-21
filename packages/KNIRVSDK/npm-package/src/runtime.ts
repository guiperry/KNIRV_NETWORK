import type * as WasmCore from "./wasm/knirv_sdk_core.js";
import { fetchWasmModule, wasmModuleMetadata, wasmModulesManifest, type WasmModuleMetadata, type WasmModuleSource } from "./modules.js";

export const BINDING_API_VERSION = 1 as const;

export type BindingErrorCode =
  | "INVALID_ARGUMENT"
  | "AUTHENTICATION"
  | "TIMEOUT"
  | "TRANSPORT"
  | "API"
  | "CRYPTO"
  | "UNSUPPORTED"
  | "INTERNAL_PANIC";

export interface BindingError {
  code: BindingErrorCode;
  message: string;
  http_status?: number;
  retryable: boolean;
}

export interface Envelope<T = unknown> {
  version: number;
  operation: string;
  payload?: T;
  error?: BindingError;
}

export interface BindingTransport {
  call(request: Envelope): Promise<Envelope>;
}

export class BindingOperationError extends Error {
  constructor(public readonly bindingError: BindingError) {
    super(bindingError.message);
    this.name = "BindingOperationError";
  }
}

type Core = typeof WasmCore;

export class WasmBindingTransport implements BindingTransport {
  constructor(private readonly core: Core) {}

  async call(request: Envelope): Promise<Envelope> {
    if (request.version !== BINDING_API_VERSION) {
      return failure(request.operation, "INVALID_ARGUMENT", `unsupported binding API version: ${request.version}`);
    }
    switch (request.operation) {
      case "crypto.sha256": {
        const data = payloadString(request.payload, "data");
        if (data === undefined) return failure(request.operation, "INVALID_ARGUMENT", "data must be a string");
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
  protected constructor(private readonly transport: BindingTransport, private readonly core?: Core) {}

  /** Create a client around a native, remote, or application-provided transport. */
  static async create(transport: BindingTransport): Promise<SdkClient> {
    return new SdkClient(transport);
  }

  /** @internal Creates the default browser/edge client after WASM initialization. */
  static fromWasm(core: Core): SdkClient {
    return new SdkClient(new WasmBindingTransport(core), core);
  }

  call<T = unknown>(operation: string, payload: unknown = {}): Promise<Envelope<T>> {
    return this.transport.call({ version: BINDING_API_VERSION, operation, payload }) as Promise<Envelope<T>>;
  }

  sha256(data: string): Promise<Envelope<{ digest: string }>> {
    return this.call("crypto.sha256", { data });
  }

  wasmManifest(): Promise<Envelope> {
    return this.call("wasm.manifest");
  }

  /** Returns verified, caller-owned module bytes. Module assets stay separate
   * from the SDK runtime WASM binary to preserve browser and edge size limits. */
  async moduleBytes(id: string, source?: WasmModuleSource): Promise<{ metadata: WasmModuleMetadata; bytes: Uint8Array }> {
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

  async sha256Hex(data: string | Uint8Array): Promise<string> {
    if (this.core && data instanceof Uint8Array) return this.core.sha256_hex(data);
    const response = await this.sha256(typeof data === "string" ? data : new TextDecoder().decode(data));
    if (response.error) throw new BindingOperationError(response.error);
    const digest = response.payload?.digest;
    if (!digest) throw new BindingOperationError({ code: "INTERNAL_PANIC", message: "crypto.sha256 returned no digest", retryable: false });
    return digest;
  }

  async verifySha256(data: string | Uint8Array, expectedHex: string): Promise<boolean> {
    if (this.core) return this.core.sha256_matches(typeof data === "string" ? new TextEncoder().encode(data) : data, expectedHex);
    return (await this.sha256Hex(data)).toLowerCase() === expectedHex.toLowerCase();
  }

  coreVersion(): string | undefined {
    return this.core?.core_version();
  }
}

function success<T>(operation: string, payload: T): Envelope<T> {
  return { version: BINDING_API_VERSION, operation, payload };
}

function failure(operation: string, code: BindingErrorCode, message: string): Envelope {
  return { version: BINDING_API_VERSION, operation, error: { code, message, retryable: false } };
}

function payloadString(payload: unknown, name: string): string | undefined {
  if (!payload || typeof payload !== "object") return undefined;
  const value = (payload as Record<string, unknown>)[name];
  return typeof value === "string" ? value : undefined;
}
