import { EncryptedData } from './types';
/**
 * AES-256-GCM encryption implementation
 */
export declare class AESEncryption {
    private static readonly ALGORITHM;
    private static readonly KEY_LENGTH;
    private static readonly NONCE_LENGTH;
    /**
     * Encrypt data using AES-256-GCM
     */
    static encrypt(data: Buffer, key: Buffer): EncryptedData;
    /**
     * Decrypt data using AES-256-GCM
     */
    static decrypt(encryptedData: EncryptedData, key: Buffer): Buffer;
    /**
     * Encrypt string data
     */
    static encryptString(text: string, key: Buffer): EncryptedData;
    /**
     * Decrypt to string data
     */
    static decryptString(encryptedData: EncryptedData, key: Buffer): string;
    /**
     * Encrypt JSON object
     */
    static encryptJSON(obj: any, key: Buffer): EncryptedData;
    /**
     * Decrypt to JSON object
     */
    static decryptJSON<T = any>(encryptedData: EncryptedData, key: Buffer): T;
}
//# sourceMappingURL=encryption.d.ts.map