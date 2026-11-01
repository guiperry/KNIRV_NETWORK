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
import { type DirectSignRequest, type MessageEnvelope, type SignedDirectTransaction, type SignedMessageEnvelope } from "./signing.js";
export interface VaultAccount {
    address: string;
    public_key: string;
}
export interface VaultOpened {
    sessionId: string;
    account: VaultAccount;
}
export interface VaultCreated extends VaultOpened {
    envelope: string;
    recoveryPhrase: string;
}
export interface VaultImported extends VaultOpened {
    envelope: string;
}
export interface VaultMigrated extends VaultImported {
    migrated: true;
}
export interface VaultUnlocked extends VaultOpened {
    migrated: false;
}
/** Randomness and time come from the host: the module reads neither. */
export interface VaultHost {
    randomBytes(length: number): Uint8Array;
    nowIso(): string;
    sessionId(): string;
}
export declare const webVaultHost: VaultHost;
export declare class VaultError extends Error {
    constructor(message: string);
}
export declare class Vault {
    private readonly wasm;
    private readonly host;
    private constructor();
    /**
     * Instantiates the module from verified bytes. The bytes are checked against
     * the manifest digest unless `verify: false` (for a caller who verified them).
     */
    static fromBytes(bytes: BufferSource, options?: {
        host?: VaultHost;
        verify?: boolean;
    }): Promise<Vault>;
    /** Loads the `crypto-core.wasm` shipped next to this package. */
    static load(options?: {
        host?: VaultHost;
    }): Promise<Vault>;
    private call;
    private fresh;
    /** A new vault with a fresh recovery phrase (128 bits of entropy). */
    createVault(password: string): VaultCreated;
    importMnemonic(mnemonic: string, password: string): VaultImported;
    /** Opens a `knirv-vault-v2.` envelope. */
    unlock(serialized: string, password: string): VaultUnlocked;
    /** Seals a mnemonic recovered from a legacy vault into a v2 envelope, verified before it is returned. */
    migrate(mnemonic: string, password: string): VaultMigrated;
    account(sessionId: string, index?: number): VaultAccount;
    signTransaction(sessionId: string, tx: DirectSignRequest, index?: number): SignedDirectTransaction;
    signMessage(sessionId: string, envelope: MessageEnvelope, index?: number): SignedMessageEnvelope;
    addressFromPublicKey(publicKeyBase64: string): string;
    closeSession(sessionId: string): void;
    sessionCount(): number;
    /** Drops every session and pending result. */
    zeroize(): void;
}
