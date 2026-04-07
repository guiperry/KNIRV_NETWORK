import { KeyDerivationOptions } from './types';
/**
 * PBKDF2 key derivation implementation
 */
export declare class KeyDerivation {
    private static readonly DEFAULT_ITERATIONS;
    private static readonly DEFAULT_KEY_LENGTH;
    private static readonly SALT_LENGTH;
    private static readonly DIGEST;
    /**
     * Derive a key from user secret and salt using PBKDF2
     */
    static deriveKey(userSecret: string, salt: Buffer, iterations?: number, keyLength?: number): Buffer;
    /**
     * Generate a random salt for key derivation
     */
    static generateSalt(): Buffer;
    /**
     * Derive key with options object
     */
    static deriveKeyWithOptions(userSecret: string, options: KeyDerivationOptions): Buffer;
    /**
     * Create options for key derivation
     */
    static createOptions(salt: Buffer, iterations?: number, keyLength?: number): KeyDerivationOptions;
    /**
     * Verify key derivation (timing safe comparison)
     */
    static verifyKey(key1: Buffer, key2: Buffer): boolean;
}
//# sourceMappingURL=key_derivation.d.ts.map