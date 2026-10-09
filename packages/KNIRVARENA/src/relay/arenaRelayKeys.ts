/**
 * Per-pairing arena relay keys (arena_fixes.md §2.3, D8).
 *
 * The CLI verifies relay requests with the KNIRV SDK's `VerifyMessage`, which
 * accepts only secp256k1 keys and derives the `knirv1…` bech32 address from
 * them, so the arena signs exactly as the Controller does: a
 * `knirv.message.v1` envelope (domain `knirv.controller`, purpose
 * `relay-request`) signed with `signMessageEnvelope`.
 *
 * WebCrypto cannot hold a secp256k1 key as non-extractable, so each pairing's
 * 32-byte private key is stored encrypted under an AES-256-GCM wrapping key
 * that *is* non-extractable. Raw key bytes exist only for the duration of one
 * signature and are zeroed afterwards. The Controller's device key is never
 * used here.
 */

import {
  CONTROLLER_DOMAIN,
  MESSAGE_SCHEMA_VERSION,
  PURPOSE_RELAY_REQUEST,
  signMessageEnvelope,
  type SignedMessageEnvelope,
} from '@knirv/sdk/signing';

/** Chain ID the CLI expects by default (`KNIRV_CHAIN_ID`, default `knirv-1`). */
export const ARENA_RELAY_CHAIN_ID = 'knirv-1';

export interface StoredArenaKey {
  iv: Uint8Array;
  ciphertext: Uint8Array;
  publicKey: string;
  address: string;
}

export interface ArenaKeyStore {
  getKey(pairingId: string): Promise<StoredArenaKey | undefined>;
  putKey(pairingId: string, key: StoredArenaKey): Promise<void>;
  deleteKey(pairingId: string): Promise<void>;
  getWrappingKey(): Promise<CryptoKey | undefined>;
  putWrappingKey(key: CryptoKey): Promise<void>;
}

export interface ArenaKeyInfo {
  publicKey: string;
  address: string;
}

export interface RelaySignatureFields {
  requestId: string;
  issuedAtUnix: number;
  expiresAtUnix: number;
  chainId?: string;
}

const DB_NAME = 'knirv-arena-relay-keys';
const DB_VERSION = 2;
const KEYS = 'pairing-keys';
const META = 'meta';
const WRAPPING_KEY_ID = 'wrapping-key';

/** IndexedDB-backed store (browser and worker). CryptoKeys are structured-cloneable. */
export class IndexedDBArenaKeyStore implements ArenaKeyStore {
  private db: Promise<IDBDatabase> | null = null;

  private open(): Promise<IDBDatabase> {
    this.db ??= new Promise((resolve, reject) => {
      const request = indexedDB.open(DB_NAME, DB_VERSION);
      request.onupgradeneeded = () => {
        const db = request.result;
        // v1 held P-256 keys the CLI cannot verify; drop them.
        for (const name of Array.from(db.objectStoreNames)) db.deleteObjectStore(name);
        db.createObjectStore(KEYS);
        db.createObjectStore(META);
      };
      request.onsuccess = () => resolve(request.result);
      request.onerror = () => reject(request.error);
    });
    return this.db;
  }

  private async run<T>(store: string, mode: IDBTransactionMode, op: (s: IDBObjectStore) => IDBRequest): Promise<T> {
    const db = await this.open();
    return new Promise<T>((resolve, reject) => {
      const request = op(db.transaction(store, mode).objectStore(store));
      request.onsuccess = () => resolve(request.result as T);
      request.onerror = () => reject(request.error);
    });
  }

  getKey(pairingId: string) { return this.run<StoredArenaKey | undefined>(KEYS, 'readonly', (s) => s.get(pairingId)); }
  async putKey(pairingId: string, key: StoredArenaKey) { await this.run(KEYS, 'readwrite', (s) => s.put(key, pairingId)); }
  async deleteKey(pairingId: string) { await this.run(KEYS, 'readwrite', (s) => s.delete(pairingId)); }
  getWrappingKey() { return this.run<CryptoKey | undefined>(META, 'readonly', (s) => s.get(WRAPPING_KEY_ID)); }
  async putWrappingKey(key: CryptoKey) { await this.run(META, 'readwrite', (s) => s.put(key, WRAPPING_KEY_ID)); }
}

/** In-memory store for tests and environments without IndexedDB. */
export class MemoryArenaKeyStore implements ArenaKeyStore {
  private keys = new Map<string, StoredArenaKey>();
  private wrapping: CryptoKey | undefined;
  async getKey(pairingId: string) { return this.keys.get(pairingId); }
  async putKey(pairingId: string, key: StoredArenaKey) { this.keys.set(pairingId, key); }
  async deleteKey(pairingId: string) { this.keys.delete(pairingId); }
  async getWrappingKey() { return this.wrapping; }
  async putWrappingKey(key: CryptoKey) { this.wrapping = key; }
}

export class ArenaRelayKeys {
  constructor(
    private readonly store: ArenaKeyStore = new IndexedDBArenaKeyStore(),
    private readonly webCrypto: Crypto = globalThis.crypto,
  ) {}

  private async wrappingKey(): Promise<CryptoKey> {
    const existing = await this.store.getWrappingKey();
    if (existing) return existing;
    const key = await this.webCrypto.subtle.generateKey({ name: 'AES-GCM', length: 256 }, false, ['encrypt', 'decrypt']);
    await this.store.putWrappingKey(key);
    return key;
  }

  /** Returns the pairing's key, creating it on first use. */
  async ensureKey(pairingId: string): Promise<ArenaKeyInfo> {
    if (!pairingId) throw new Error('pairingId is required');
    const existing = await this.store.getKey(pairingId);
    if (existing) return { publicKey: existing.publicKey, address: existing.address };

    const wrapping = await this.wrappingKey();
    for (let attempt = 0; attempt < 4; attempt++) {
      const privateKey = this.webCrypto.getRandomValues(new Uint8Array(32));
      try {
        // Signing a probe derives the compressed public key and the address the
        // CLI will compute; it also rejects the (≈2^-128) invalid scalars.
        const probe = await signMessageEnvelope(privateKey, {
          schemaVersion: MESSAGE_SCHEMA_VERSION,
          domain: CONTROLLER_DOMAIN,
          purpose: 'arena-key-probe',
          chainId: ARENA_RELAY_CHAIN_ID,
          nonce: pairingId,
          issuedAtUnix: BigInt(Math.floor(Date.now() / 1000)),
          expiresAtUnix: BigInt(Math.floor(Date.now() / 1000) + 1),
          payload: new Uint8Array(),
        });
        const iv = this.webCrypto.getRandomValues(new Uint8Array(12));
        const ciphertext = new Uint8Array(
          await this.webCrypto.subtle.encrypt({ name: 'AES-GCM', iv, additionalData: new TextEncoder().encode(pairingId) }, wrapping, privateKey),
        );
        await this.store.putKey(pairingId, { iv, ciphertext, publicKey: probe.public_key, address: probe.address });
        return { publicKey: probe.public_key, address: probe.address };
      } catch (err) {
        if (attempt === 3) throw err;
      } finally {
        privateKey.fill(0);
      }
    }
    throw new Error('could not generate an arena relay key');
  }

  /**
   * Signs a marshaled relay envelope as a `knirv.message.v1` relay-request:
   * nonce = the relay request ID, payload = the relay envelope bytes.
   */
  async signRelayRequest(pairingId: string, relayEnvelope: Uint8Array, fields: RelaySignatureFields): Promise<SignedMessageEnvelope> {
    await this.ensureKey(pairingId);
    const stored = await this.store.getKey(pairingId);
    if (!stored) throw new Error('arena relay key is missing');
    const privateKey = new Uint8Array(
      await this.webCrypto.subtle.decrypt(
        { name: 'AES-GCM', iv: stored.iv, additionalData: new TextEncoder().encode(pairingId) },
        await this.wrappingKey(),
        stored.ciphertext,
      ),
    );
    try {
      return await signMessageEnvelope(privateKey, {
        schemaVersion: MESSAGE_SCHEMA_VERSION,
        domain: CONTROLLER_DOMAIN,
        purpose: PURPOSE_RELAY_REQUEST,
        chainId: fields.chainId ?? ARENA_RELAY_CHAIN_ID,
        nonce: fields.requestId,
        issuedAtUnix: BigInt(fields.issuedAtUnix),
        expiresAtUnix: BigInt(fields.expiresAtUnix),
        payload: relayEnvelope,
      });
    } finally {
      privateKey.fill(0);
    }
  }

  /** Re-pairing rotates the key; revoking a pairing forgets it. */
  forget(pairingId: string): Promise<void> {
    return this.store.deleteKey(pairingId);
  }
}
