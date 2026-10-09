/**
 * Arena relay WASM worker — runs the same `controller-relay.wasm` the Controller uses.
 *
 * Operations: validate_envelope, canonicalize, check_sequence, compute_digest,
 * zeroize (relay WASM, module kind 3, ABI 1) and the arena's per-pairing key
 * operations generate_keypair / sign_envelope (ArenaRelayKeys: secp256k1,
 * signed as a knirv.message.v1 relay-request so the CLI can verify it).
 */

import { ArenaRelayKeys } from './arenaRelayKeys';

interface RelayWorkerRequest {
  id: string;
  type: 'init' | 'self_test' | 'validate_envelope' | 'canonicalize' | 'compute_digest' | 'check_sequence' | 'zeroize' | 'sign_envelope' | 'generate_keypair' | 'store_keypair' | 'get_public_key';
  bytes?: ArrayBuffer;
  pairingId?: string;
  requestId?: string;
  issuedAtUnix?: number;
  expiresAtUnix?: number;
  chainId?: string;
}

interface RelayWorkerResponse {
  id: string;
  ok: boolean;
  result?: unknown;
  error?: string;
}

interface WASMExports {
  abi_version: () => number;
  module_kind: () => number;
  allocate: (length: number) => number;
  deallocate: (ptr: number, len: number) => void;
  self_test: (inputPtr: number, inputLen: number) => number;
  invoke: (opPtr: number, opLen: number, inPtr: number, inLen: number) => number;
  result_pointer: (handle: number) => number;
  result_length: (handle: number) => number;
  free_result: (handle: number) => void;
  zeroize: () => void;
  wasm_init: () => void;
}

const OP_VALIDATE_ENVELOPE = 'validate_envelope';
const OP_CANONICALIZE = 'canonicalize';
const OP_COMPUTE_DIGEST = 'compute_digest';
const OP_CHECK_SEQUENCE = 'check_sequence';

let moduleExports: WASMExports | null = null;
let memory: WebAssembly.Memory | null = null;
let initialized = false;

const relayKeys = new ArenaRelayKeys();

function encodeResult(handle: number): { result: unknown; valid: boolean } | null {
  if (!moduleExports || !memory) return null;
  const ptr = moduleExports.result_pointer(handle);
  if (ptr === 0) return { result: null, valid: false };
  const length = moduleExports.result_length(handle);
  if (length === 0) return { result: null, valid: false };
  const bytes = new Uint8Array(memory.buffer, ptr, length);
  const text = new TextDecoder().decode(bytes.slice());
  moduleExports.free_result(handle);
  try {
    return JSON.parse(text) as { result: unknown; valid: boolean };
  } catch {
    return { result: text, valid: false };
  }
}

async function initModule(wasmBytes: ArrayBuffer): Promise<void> {
  const compiled = await WebAssembly.compile(wasmBytes);
  const instance = await WebAssembly.instantiate(compiled, {});
  const exports = instance.exports as Record<string, unknown>;

  const required = ['abi_version', 'module_kind', 'allocate', 'deallocate', 'self_test', 'invoke', 'result_pointer', 'result_length', 'free_result', 'zeroize', 'wasm_init'];
  for (const name of required) {
    if (typeof exports[name] !== 'function') throw new Error(`Missing required export: ${name}`);
  }

  moduleExports = exports as unknown as WASMExports;

  const memExport = Object.values(exports).find((v): v is WebAssembly.Memory => v instanceof WebAssembly.Memory);
  if (!memExport) throw new Error('WASM module must export a Memory instance');
  memory = memExport;

  const version = moduleExports.abi_version();
  const kind = moduleExports.module_kind();
  if (version !== 1) throw new Error(`Unsupported ABI version: ${version}`);
  if (kind !== 3) throw new Error(`Expected relay module kind (3), got: ${kind}`);

  moduleExports.wasm_init();

  const handle = moduleExports.self_test(0, 0);
  const selfTestResult = encodeResult(handle);
  if (!selfTestResult || selfTestResult.valid === false) {
    throw new Error(`Relay WASM self-test failed: ${JSON.stringify(selfTestResult)}`);
  }

  initialized = true;
}

function writeToMemory(data: Uint8Array): number {
  if (!moduleExports) throw new Error('Relay WASM module not initialized');
  const ptr = moduleExports.allocate(data.length);
  if (ptr === 0) throw new Error('allocate returned 0');
  const buf = new Uint8Array(memory!.buffer, ptr, data.length);
  buf.set(data);
  return ptr;
}

function freeMemory(ptr: number, length: number): void {
  if (!moduleExports || ptr === 0) return;
  moduleExports.deallocate(ptr, length);
}

function writeStringToMemory(str: string): { ptr: number; len: number } {
  const encoded = new TextEncoder().encode(str);
  const ptr = writeToMemory(encoded);
  return { ptr, len: encoded.length };
}

function callInvoke(op: string, data: Uint8Array): unknown {
  if (!moduleExports) throw new Error('Relay WASM module not initialized');
  const { ptr: opPtr, len: opLen } = writeStringToMemory(op);
  const inPtr = writeToMemory(data);
  try {
    const handle = moduleExports.invoke(opPtr, opLen, inPtr, data.length);
    const result = encodeResult(handle);
    if (!result || result.valid === false) {
      throw new Error(`Relay WASM invoke failed: ${JSON.stringify(result)}`);
    }
    return result.result;
  } finally {
    freeMemory(opPtr, opLen);
    freeMemory(inPtr, data.length);
  }
}

self.onmessage = async (event: MessageEvent<RelayWorkerRequest>) => {
  const req = event.data;
  if (!req?.id || !req?.type) return;

  const reply = (response: RelayWorkerResponse) => {
    self.postMessage(response);
  };

  try {
    switch (req.type) {
      case 'init':
        await initModule(req.bytes!);
        reply({ id: req.id, ok: true });
        break;
      case 'self_test': {
        if (!initialized) throw new Error('Relay WASM module not initialized');
        const handle = moduleExports!.self_test(0, 0);
        reply({ id: req.id, ok: true, result: encodeResult(handle) });
        break;
      }
      case 'validate_envelope': {
        if (!initialized) throw new Error('Relay WASM module not initialized');
        const payload = new Uint8Array(req.bytes!);
        const result = callInvoke(OP_VALIDATE_ENVELOPE, payload);
        reply({ id: req.id, ok: true, result });
        break;
      }
      case 'canonicalize': {
        if (!initialized) throw new Error('Relay WASM module not initialized');
        const payload = new Uint8Array(req.bytes!);
        const result = callInvoke(OP_CANONICALIZE, payload);
        reply({ id: req.id, ok: true, result });
        break;
      }
      case 'compute_digest': {
        if (!initialized) throw new Error('Relay WASM module not initialized');
        const payload = new Uint8Array(req.bytes!);
        const result = callInvoke(OP_COMPUTE_DIGEST, payload);
        reply({ id: req.id, ok: true, result });
        break;
      }
      case 'check_sequence': {
        if (!initialized) throw new Error('Relay WASM module not initialized');
        const payload = new Uint8Array(req.bytes!);
        const result = callInvoke(OP_CHECK_SEQUENCE, payload);
        reply({ id: req.id, ok: true, result });
        break;
      }
      case 'zeroize':
        if (!initialized) throw new Error('Relay WASM module not initialized');
        moduleExports!.zeroize();
        reply({ id: req.id, ok: true });
        break;
      case 'generate_keypair': {
        if (!req.pairingId) throw new Error('pairingId required for generate_keypair');
        const { publicKey, address } = await relayKeys.ensureKey(req.pairingId);
        reply({ id: req.id, ok: true, result: { public_key: publicKey, address } });
        break;
      }
      case 'sign_envelope': {
        const { pairingId, requestId, issuedAtUnix, expiresAtUnix, chainId } = req;
        if (!pairingId || !requestId || issuedAtUnix === undefined || expiresAtUnix === undefined || !req.bytes) {
          throw new Error('sign_envelope requires pairingId, requestId, issuedAtUnix, expiresAtUnix and the relay envelope bytes');
        }
        const signed = await relayKeys.signRelayRequest(pairingId, new Uint8Array(req.bytes), { requestId, issuedAtUnix, expiresAtUnix, chainId });
        reply({ id: req.id, ok: true, result: signed });
        break;
      }
      default:
        reply({ id: req.id, ok: false, error: `Unknown request type: ${req.type}` });
    }
  } catch (err) {
    reply({ id: req.id, ok: false, error: err instanceof Error ? err.message : String(err) });
  }
};