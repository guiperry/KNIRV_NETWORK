import { KeyDerivation } from './key_derivation';
import { AESEncryption } from './encryption';
/**
 * Memory encryption for sensitive data protection
 */
export class MemoryEncryption {
    constructor(options = {}) {
        this.iterations = options.iterations || 100000;
        this.keyLength = options.keyLength || 32;
    }
    /**
     * Derive an encryption key from user secret
     */
    deriveKey(userSecret, salt) {
        return KeyDerivation.deriveKey(userSecret, salt, this.iterations, this.keyLength);
    }
    /**
     * Encrypt memory data for storage
     */
    encryptMemory(data, key) {
        return AESEncryption.encrypt(data, key);
    }
    /**
     * Decrypt memory data for retrieval
     */
    decryptMemory(encryptedData, key) {
        return AESEncryption.decrypt(encryptedData, key);
    }
    /**
     * Encrypt string data
     */
    encryptString(text, key) {
        return AESEncryption.encryptString(text, key);
    }
    /**
     * Decrypt string data
     */
    decryptString(encryptedData, key) {
        return AESEncryption.decryptString(encryptedData, key);
    }
    /**
     * Generate a random salt for key derivation
     */
    generateSalt() {
        return KeyDerivation.generateSalt();
    }
    /**
     * Encode a key to base64 for storage
     */
    encodeKey(key) {
        return key.toString('base64');
    }
    /**
     * Decode a base64-encoded key
     */
    decodeKey(encoded) {
        try {
            return Buffer.from(encoded, 'base64');
        }
        catch (error) {
            throw new Error(`Failed to decode key: ${error instanceof Error ? error.message : 'Unknown error'}`);
        }
    }
    /**
     * Create options for key derivation
     */
    createKeyDerivationOptions(salt) {
        return KeyDerivation.createOptions(salt, this.iterations, this.keyLength);
    }
    /**
     * Full encryption workflow with key derivation
     */
    async encryptWithPassword(data, password) {
        const salt = this.generateSalt();
        const key = this.deriveKey(password, salt);
        let encryptedData;
        if (Buffer.isBuffer(data)) {
            encryptedData = this.encryptMemory(data, key);
        }
        else {
            encryptedData = this.encryptString(data, key);
        }
        return { encryptedData, salt };
    }
    /**
     * Full decryption workflow with key derivation
     */
    async decryptWithPassword(encryptedData, salt, password) {
        const key = this.deriveKey(password, salt);
        return this.decryptMemory(encryptedData, key);
    }
    /**
     * Get current configuration
     */
    getOptions() {
        return {
            iterations: this.iterations,
            keyLength: this.keyLength
        };
    }
}
//# sourceMappingURL=security.js.map