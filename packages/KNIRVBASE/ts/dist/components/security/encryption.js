import { createCipheriv, createDecipheriv, randomBytes } from 'crypto';
/**
 * AES-256-GCM encryption implementation
 */
export class AESEncryption {
    /**
     * Encrypt data using AES-256-GCM
     */
    static encrypt(data, key) {
        if (key.length !== AESEncryption.KEY_LENGTH) {
            throw new Error(`Invalid key length. Expected ${AESEncryption.KEY_LENGTH} bytes, got ${key.length}`);
        }
        try {
            const nonce = randomBytes(AESEncryption.NONCE_LENGTH);
            const cipher = createCipheriv(AESEncryption.ALGORITHM, key, nonce);
            const encrypted = Buffer.concat([cipher.update(data), cipher.final()]);
            const authTag = cipher.getAuthTag();
            // Combine authTag + encrypted data
            const combined = Buffer.concat([authTag, encrypted]);
            return {
                data: combined.toString('base64'),
                nonce: nonce.toString('base64')
            };
        }
        catch (error) {
            throw new Error(`Failed to encrypt data: ${error instanceof Error ? error.message : 'Unknown error'}`);
        }
    }
    /**
     * Decrypt data using AES-256-GCM
     */
    static decrypt(encryptedData, key) {
        if (key.length !== AESEncryption.KEY_LENGTH) {
            throw new Error(`Invalid key length. Expected ${AESEncryption.KEY_LENGTH} bytes, got ${key.length}`);
        }
        try {
            const combined = Buffer.from(encryptedData.data, 'base64');
            if (combined.length < 16) { // authTag minimum
                throw new Error('Invalid encrypted data: too short');
            }
            const authTag = combined.slice(0, 16);
            const ciphertext = combined.slice(16);
            const decipher = createDecipheriv(AESEncryption.ALGORITHM, key, Buffer.from(encryptedData.nonce, 'base64'));
            decipher.setAuthTag(authTag);
            const decrypted = Buffer.concat([decipher.update(ciphertext), decipher.final()]);
            return decrypted;
        }
        catch (error) {
            throw new Error(`Failed to decrypt data: ${error instanceof Error ? error.message : 'Unknown error'}`);
        }
    }
    /**
     * Encrypt string data
     */
    static encryptString(text, key) {
        return AESEncryption.encrypt(Buffer.from(text, 'utf8'), key);
    }
    /**
     * Decrypt to string data
     */
    static decryptString(encryptedData, key) {
        const decrypted = AESEncryption.decrypt(encryptedData, key);
        return decrypted.toString('utf8');
    }
    /**
     * Encrypt JSON object
     */
    static encryptJSON(obj, key) {
        const jsonString = JSON.stringify(obj);
        return AESEncryption.encryptString(jsonString, key);
    }
    /**
     * Decrypt to JSON object
     */
    static decryptJSON(encryptedData, key) {
        const jsonString = AESEncryption.decryptString(encryptedData, key);
        return JSON.parse(jsonString);
    }
}
AESEncryption.ALGORITHM = 'aes-256-gcm';
AESEncryption.KEY_LENGTH = 32; // 256 bits
AESEncryption.NONCE_LENGTH = 12; // 96 bits for GCM
//# sourceMappingURL=encryption.js.map