import { createCipheriv, createDecipheriv, randomBytes } from 'crypto';
import { EncryptedData } from './types';

/**
 * AES-256-GCM encryption implementation
 */
export class AESEncryption {
  private static readonly ALGORITHM = 'aes-256-gcm';
  private static readonly KEY_LENGTH = 32; // 256 bits
  private static readonly NONCE_LENGTH = 12; // 96 bits for GCM

  /**
   * Encrypt data using AES-256-GCM
   */
  static encrypt(data: Buffer, key: Buffer): EncryptedData {
    if (key.length !== AESEncryption.KEY_LENGTH) {
      throw new Error(`Invalid key length. Expected ${AESEncryption.KEY_LENGTH} bytes, got ${key.length}`);
    }

    try {
      const cipher = createCipheriv(AESEncryption.ALGORITHM, key, null);
      const nonce = randomBytes(AESEncryption.NONCE_LENGTH);
      
      // Set the nonce for the cipher
      cipher.setAAD(nonce);
      
      const encrypted = Buffer.concat([cipher.update(data), cipher.final()]);
      const authTag = cipher.getAuthTag();
      
      // Combine nonce + authTag + encrypted data
      const combined = Buffer.concat([nonce, authTag, encrypted]);
      
      return {
        data: combined.toString('base64'),
        nonce: nonce.toString('base64')
      };
    } catch (error) {
      throw new Error(`Failed to encrypt data: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  }

  /**
   * Decrypt data using AES-256-GCM
   */
  static decrypt(encryptedData: EncryptedData, key: Buffer): Buffer {
    if (key.length !== AESEncryption.KEY_LENGTH) {
      throw new Error(`Invalid key length. Expected ${AESEncryption.KEY_LENGTH} bytes, got ${key.length}`);
    }

    try {
      const combined = Buffer.from(encryptedData.data, 'base64');
      
      if (combined.length < AESEncryption.NONCE_LENGTH + 16) { // nonce + authTag minimum
        throw new Error('Invalid encrypted data: too short');
      }

      const nonce = combined.slice(0, AESEncryption.NONCE_LENGTH);
      const authTag = combined.slice(AESEncryption.NONCE_LENGTH, AESEncryption.NONCE_LENGTH + 16);
      const ciphertext = combined.slice(AESEncryption.NONCE_LENGTH + 16);

      const decipher = createDecipheriv(AESEncryption.ALGORITHM, key, nonce);
      decipher.setAAD(nonce);
      decipher.setAuthTag(authTag);
      
      const decrypted = Buffer.concat([decipher.update(ciphertext), decipher.final()]);
      return decrypted;
    } catch (error) {
      throw new Error(`Failed to decrypt data: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  }

  /**
   * Encrypt string data
   */
  static encryptString(text: string, key: Buffer): EncryptedData {
    return AESEncryption.encrypt(Buffer.from(text, 'utf8'), key);
  }

  /**
   * Decrypt to string data
   */
  static decryptString(encryptedData: EncryptedData, key: Buffer): string {
    const decrypted = AESEncryption.decrypt(encryptedData, key);
    return decrypted.toString('utf8');
  }

  /**
   * Encrypt JSON object
   */
  static encryptJSON(obj: any, key: Buffer): EncryptedData {
    const jsonString = JSON.stringify(obj);
    return AESEncryption.encryptString(jsonString, key);
  }

  /**
   * Decrypt to JSON object
   */
  static decryptJSON<T = any>(encryptedData: EncryptedData, key: Buffer): T {
    const jsonString = AESEncryption.decryptString(encryptedData, key);
    return JSON.parse(jsonString) as T;
  }
}