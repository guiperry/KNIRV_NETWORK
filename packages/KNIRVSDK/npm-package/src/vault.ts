/**
 * Typed facade over the `crypto-core` WASM module: the wallet vault's BIP-39,
 * SLIP-10, Argon2id/AES-GCM and signing, all implemented once in Rust
 * (core/src/vault.rs). No cryptography is written here.
 *
 * Secrets (seed, private keys, derived keys) exist only inside the module. This
 * facade returns session handles, public keys, addresses, signatures and the
 * encrypted envelope. The module is deterministic and has no imports, so this
 * layer supplies the randomness and the clock. Passwords and mnemonics are
 * written into module memory as bytes and wiped by the module once parsed, but a
 * value that was ever a JavaScript string cannot be erased from JavaScript.
 */
import { wireEnvelope, wireRequest, type DirectSignRequest, type MessageEnvelope, type SignedDirectTransaction, type SignedMessageEnvelope } from "./signing.js";
import { toJson } from "./local.js";
import { wasmModuleMetadata } from "./modules.js";

export interface VaultAccount { address: string; public_key: string }
export interface VaultOpened { sessionId: string; account: VaultAccount }
export interface VaultCreated extends VaultOpened { envelope: string; recoveryPhrase: string }
export interface VaultImported extends VaultOpened { envelope: string }
export interface VaultMigrated extends VaultImported { migrated: true }
export interface VaultUnlocked extends VaultOpened { migrated: false }

/** Randomness and time come from the host: the module reads neither. */
export interface VaultHost {
  randomBytes(length: number): Uint8Array;
  nowIso(): string;
  sessionId(): string;
}

export const webVaultHost: VaultHost = {
  randomBytes: (n) => crypto.getRandomValues(new Uint8Array(n)),
  nowIso: () => new Date().toISOString(),
  sessionId: () => crypto.randomUUID(),
};

interface Exports {
  memory: WebAssembly.Memory;
  abi_version(): number;
  module_kind(): number;
  self_test(): number;
  allocate(length: number): number;
  deallocate(ptr: number, length: number): void;
  invoke(op: number, ptr: number, length: number): number;
  result_pointer(handle: number): number;
  result_length(handle: number): number;
  free_result(handle: number): void;
  zeroize(): void;
  session_count(): number;
}

const MODULE_KIND_CRYPTO = 1;
const toB64 = (bytes: Uint8Array): string => {
  let binary = "";
  for (let i = 0; i < bytes.length; i += 0x8000) binary += String.fromCharCode(...bytes.subarray(i, i + 0x8000));
  return btoa(binary);
};
const hex = (bytes: ArrayBuffer): string => [...new Uint8Array(bytes)].map((b) => b.toString(16).padStart(2, "0")).join("");

export class VaultError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "VaultError";
  }
}

export class Vault {
  private constructor(private readonly wasm: Exports, private readonly host: VaultHost) {}

  /**
   * Instantiates the module from verified bytes. The bytes are checked against
   * the manifest digest unless `verify: false` (for a caller who verified them).
   */
  static async fromBytes(bytes: BufferSource, options: { host?: VaultHost; verify?: boolean } = {}): Promise<Vault> {
    if (options.verify !== false) {
      const meta = wasmModuleMetadata("crypto-core");
      const digest = hex(await crypto.subtle.digest("SHA-256", bytes));
      if (digest !== meta.sha256) throw new VaultError("crypto-core digest does not match the SDK manifest");
    }
    const module = await WebAssembly.compile(bytes);
    if (WebAssembly.Module.imports(module).length !== 0) throw new VaultError("crypto-core must not declare imports");
    const { exports } = await WebAssembly.instantiate(module, {});
    const wasm = exports as unknown as Exports;
    if (wasm.abi_version() !== 1 || wasm.module_kind() !== MODULE_KIND_CRYPTO) throw new VaultError("not a crypto-core module");
    if (wasm.self_test() !== 1) throw new VaultError("crypto-core self-test failed");
    return new Vault(wasm, options.host ?? webVaultHost);
  }

  /** Loads the `crypto-core.wasm` shipped next to this package. */
  static async load(options: { host?: VaultHost } = {}): Promise<Vault> {
    const url = new URL("./modules/crypto-core.wasm", import.meta.url);
    let bytes: Uint8Array;
    if (url.protocol === "file:") {
      const specifier = "node:fs/promises"; // only reached under Node; bundlers must not resolve it
      const { readFile } = (await import(/* @vite-ignore */ specifier)) as { readFile(path: URL): Promise<Uint8Array> };
      bytes = await readFile(url);
    } else {
      const response = await fetch(url);
      if (!response.ok) throw new VaultError(`could not load crypto-core: HTTP ${response.status}`);
      bytes = new Uint8Array(await response.arrayBuffer());
    }
    return Vault.fromBytes(bytes as BufferSource, options);
  }

  private call<T>(op: string, fields: Record<string, unknown>): T {
    const request = new TextEncoder().encode(toJson({ op, ...fields }));
    const ptr = this.wasm.allocate(request.length);
    new Uint8Array(this.wasm.memory.buffer, ptr, request.length).set(request);
    request.fill(0);
    let handle = 0;
    try {
      handle = this.wasm.invoke(1, ptr, request.length); // the module wipes the input
    } finally {
      this.wasm.deallocate(ptr, request.length);
    }
    const at = this.wasm.result_pointer(handle);
    const length = this.wasm.result_length(handle);
    const out = new Uint8Array(this.wasm.memory.buffer, at, length).slice();
    this.wasm.free_result(handle);
    let reply: { ok?: T; error?: string };
    try {
      reply = JSON.parse(new TextDecoder().decode(out)) as { ok?: T; error?: string };
    } finally {
      out.fill(0);
    }
    if (reply.error !== undefined) throw new VaultError(reply.error);
    return reply.ok as T;
  }

  private fresh() {
    return {
      session_id: this.host.sessionId(),
      salt: toB64(this.host.randomBytes(16)),
      nonce: toB64(this.host.randomBytes(12)),
      created_at: this.host.nowIso(),
    };
  }

  /** A new vault with a fresh recovery phrase (128 bits of entropy). */
  createVault(password: string): VaultCreated {
    const r = this.call<{ session_id: string; account: VaultAccount; envelope: string; recovery_phrase: string }>("vault.create", {
      password, entropy: toB64(this.host.randomBytes(16)), ...this.fresh(),
    });
    return { sessionId: r.session_id, account: r.account, envelope: r.envelope, recoveryPhrase: r.recovery_phrase };
  }

  importMnemonic(mnemonic: string, password: string): VaultImported {
    const r = this.call<{ session_id: string; account: VaultAccount; envelope: string }>("vault.import", { password, mnemonic, ...this.fresh() });
    return { sessionId: r.session_id, account: r.account, envelope: r.envelope };
  }

  /** Opens a `knirv-vault-v2.` envelope. */
  unlock(serialized: string, password: string): VaultUnlocked {
    const r = this.call<{ session_id: string; account: VaultAccount }>("vault.unlock", { serialized, password, session_id: this.host.sessionId() });
    return { sessionId: r.session_id, account: r.account, migrated: false };
  }

  /** Seals a mnemonic recovered from a legacy vault into a v2 envelope, verified before it is returned. */
  migrate(mnemonic: string, password: string): VaultMigrated {
    const r = this.call<{ session_id: string; account: VaultAccount; envelope: string }>("vault.migrate", { password, mnemonic, ...this.fresh() });
    return { sessionId: r.session_id, account: r.account, envelope: r.envelope, migrated: true };
  }

  account(sessionId: string, index = 0): VaultAccount {
    return this.call<VaultAccount>("session.account", { session_id: sessionId, index });
  }

  signTransaction(sessionId: string, tx: DirectSignRequest, index = 0): SignedDirectTransaction {
    return this.call<SignedDirectTransaction>("session.sign_tx", { session_id: sessionId, index, tx: wireRequest(tx) });
  }

  signMessage(sessionId: string, envelope: MessageEnvelope, index = 0): SignedMessageEnvelope {
    return this.call<SignedMessageEnvelope>("session.sign_msg", { session_id: sessionId, index, envelope: wireEnvelope(envelope) });
  }

  addressFromPublicKey(publicKeyBase64: string): string {
    return this.call<{ address: string }>("address.from_pubkey", { pubkey: publicKeyBase64 }).address;
  }

  closeSession(sessionId: string): void {
    this.call("session.close", { session_id: sessionId });
  }

  sessionCount(): number {
    return this.wasm.session_count();
  }

  /** Drops every session and pending result. */
  zeroize(): void {
    this.wasm.zeroize();
  }
}
