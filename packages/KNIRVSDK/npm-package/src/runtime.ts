import type * as WasmCore from "./wasm/knirv_sdk_core.js";
import { toJson } from "./local.js";
import { fetchWasmModule, wasmModuleMetadata, type WasmModuleMetadata, type WasmModuleSource } from "./modules.js";

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

/** Client settings for the network operations; carried on each request. */
export interface BindingConfig {
  network?: string;
  api_key?: string;
  timeout_ms?: number;
  transaction_url?: string;
  gateway_url?: string;
  controller_url?: string;
  backend_url?: string;
  headers?: Record<string, string>;
}

/**
 * The one network seam. The Rust core builds requests, retries, fails over and
 * maps errors; the host only moves bytes. Wire format: core/src/http_transport.rs.
 * Both functions receive and return JSON strings / milliseconds.
 */
export interface HttpHost {
  http_request(requestJson: string): Promise<string>;
  sleep(ms: number): Promise<void>;
}

type TransportErrorKind = "timeout" | "dns" | "tls" | "refused" | "aborted" | "other";

function hostError(kind: TransportErrorKind, message: string): string {
  return JSON.stringify({ error: { kind, message } });
}

/** The default host: global `fetch`, with `timeout_ms` enforced by an abort. */
export function fetchHost(fetchImpl: typeof fetch = (...args) => fetch(...args)): HttpHost {
  return {
    async http_request(requestJson: string): Promise<string> {
      const req = JSON.parse(requestJson) as { method: string; url: string; headers: Record<string, string>; body_base64: string; timeout_ms: number };
      const controller = new AbortController();
      let timedOut = false;
      const timer = req.timeout_ms > 0 ? setTimeout(() => { timedOut = true; controller.abort(); }, req.timeout_ms) : undefined;
      try {
        const body = req.body_base64 ? Uint8Array.from(atob(req.body_base64), (c) => c.charCodeAt(0)) : undefined;
        const response = await fetchImpl(req.url, { method: req.method, headers: req.headers, body: req.method === "GET" || req.method === "HEAD" ? undefined : body, signal: controller.signal });
        const bytes = new Uint8Array(await response.arrayBuffer());
        let binary = "";
        for (let i = 0; i < bytes.length; i += 0x8000) binary += String.fromCharCode(...bytes.subarray(i, i + 0x8000));
        const headers: Record<string, string> = {};
        response.headers.forEach((value, name) => { headers[name] = value; });
        return JSON.stringify({ status: response.status, headers, body_base64: btoa(binary) });
      } catch (error) {
        // fetch does not say why it failed; a timeout we caused is the only kind we can name.
        return hostError(timedOut ? "timeout" : "other", error instanceof Error ? error.message : String(error));
      } finally {
        if (timer !== undefined) clearTimeout(timer);
      }
    },
    sleep: (ms) => new Promise((resolve) => setTimeout(resolve, ms)),
  };
}

export class BindingOperationError extends Error {
  constructor(public readonly bindingError: BindingError) {
    super(bindingError.message);
    this.name = "BindingOperationError";
  }
}

type Core = typeof WasmCore;

const NETWORK_OPERATIONS = new Set(["actuarial.call", "transaction.chain", "gateway.health", "transmission.resolve_uri", "transmission.broadcast", "wallet.account"]);

/** Runs envelopes against the WASM core; network operations use `host`. */
export class WasmBindingTransport {
  constructor(private readonly core: Core, private readonly host: HttpHost = fetchHost(), private readonly config?: BindingConfig) {}

  async call(request: Envelope & { config?: BindingConfig }): Promise<Envelope> {
    if (request.version !== BINDING_API_VERSION) {
      return failure(request.operation, "INVALID_ARGUMENT", `unsupported binding API version: ${request.version}`);
    }
    switch (request.operation) {
      default:
        if (NETWORK_OPERATIONS.has(request.operation)) {
          const withConfig = this.config && !request.config ? { ...request, config: this.config } : request;
          return JSON.parse(await this.core.invoke_async(toJson(withConfig), this.host)) as Envelope;
        }
        // Everything else the core can do without a network is forwarded to it;
        // anything it does not recognise answers UNSUPPORTED.
        return JSON.parse(this.core.invoke(JSON.stringify(request))) as Envelope;
    }
  }
}

export interface SdkClientOptions {
  /** Moves bytes for the network operations. Defaults to global `fetch`. */
  http?: HttpHost;
  /** Default client settings (network, URLs, API key, timeout) for network operations. */
  config?: BindingConfig;
}

export class SdkClient {
  protected constructor(private readonly transport: WasmBindingTransport, private readonly core: Core) {}

  /** @internal Creates the client after WASM initialization. */
  static fromWasm(core: Core, options: SdkClientOptions = {}): SdkClient {
    return new SdkClient(new WasmBindingTransport(core, options.http, options.config), core);
  }

  /** `config` overrides the client's default settings for a network operation. */
  call<T = unknown>(operation: string, payload: unknown = {}, config?: BindingConfig): Promise<Envelope<T>> {
    return this.transport.call({ version: BINDING_API_VERSION, operation, payload, ...(config ? { config } : {}) }) as Promise<Envelope<T>>;
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

