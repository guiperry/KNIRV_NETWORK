import { EncryptionOptions, EncryptedData, KeyDerivationOptions } from './types';
/**
 * Memory encryption for sensitive data protection
 */
export declare class MemoryEncryption {
    private iterations;
    private keyLength;
    constructor(options?: EncryptionOptions);
    /**
     * Derive an encryption key from user secret
     */
    deriveKey(userSecret: string, salt: Buffer): Buffer;
    /**
     * Encrypt memory data for storage
     */
    encryptMemory(data: Buffer, key: Buffer): EncryptedData;
    /**
     * Decrypt memory data for retrieval
     */
    decryptMemory(encryptedData: EncryptedData, key: Buffer): Buffer;
    /**
     * Encrypt string data
     */
    encryptString(text: string, key: Buffer): EncryptedData;
    /**
     * Decrypt string data
     */
    decryptString(encryptedData: EncryptedData, key: Buffer): string;
    /**
     * Generate a random salt for key derivation
     */
    generateSalt(): Buffer;
    /**
     * Encode a key to base64 for storage
     */
    encodeKey(key: Buffer): string;
    /**
     * Decode a base64-encoded key
     */
    decodeKey(encoded: string): Buffer;
    /**
     * Create options for key derivation
     */
    createKeyDerivationOptions(salt: Buffer): KeyDerivationOptions;
    /**
     * Full encryption workflow with key derivation
     */
    encryptWithPassword(data: Buffer | string, password: string): Promise<{
        encryptedData: EncryptedData;
        salt: Buffer;
    }>;
    /**
     * Full decryption workflow with key derivation
     */
    decryptWithPassword(encryptedData: EncryptedData, salt: Buffer, password: string): Promise<Buffer>;
    /**
     * Get current configuration
     */
    getOptions(): EncryptionOptions;
}
//# sourceMappingURL=security.d.ts.map