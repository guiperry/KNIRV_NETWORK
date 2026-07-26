import { KeyDerivation } from './key_derivation';
import { AESEncryption } from './encryption';
import { EncryptionOptions, EncryptedData, KeyDerivationOptions } from './types';

/**
 * Memory encryption for sensitive data protection
 */
export class MemoryEncryption {
  private iterations: number;
  private keyLength: number;

  constructor(options: EncryptionOptions = {}) {
    this.iterations = options.iterations || 100000;
    this.keyLength = options.keyLength || 32;
  }

  /**
   * Derive an encryption key from user secret
   */
  deriveKey(userSecret: string, salt: Buffer): Buffer {
    return KeyDerivation.deriveKey(userSecret, salt, this.iterations, this.keyLength);
  }

  /**
   * Encrypt memory data for storage
   */
  encryptMemory(data: Buffer, key: Buffer): EncryptedData {
    return AESEncryption.encrypt(data, key);
  }

  /**
   * Decrypt memory data for retrieval
   */
  decryptMemory(encryptedData: EncryptedData, key: Buffer): Buffer {
    return AESEncryption.decrypt(encryptedData, key);
  }

  /**
   * Encrypt string data
   */
  encryptString(text: string, key: Buffer): EncryptedData {
    return AESEncryption.encryptString(text, key);
  }

  /**
   * Decrypt string data
   */
  decryptString(encryptedData: EncryptedData, key: Buffer): string {
    return AESEncryption.decryptString(encryptedData, key);
  }

  /**
   * Generate a random salt for key derivation
   */
  generateSalt(): Buffer {
    return KeyDerivation.generateSalt();
  }

  /**
   * Encode a key to base64 for storage
   */
  encodeKey(key: Buffer): string {
    return key.toString('base64url');
  }

  /**
   * Decode a base64-encoded key
   */
  decodeKey(encoded: string): Buffer {
    try {
      // Validate base64url string
      const base64urlRegex = /^[A-Za-z0-9_-]*$/;
      if (!base64urlRegex.test(encoded)) {
        throw new Error('Invalid base64url characters');
      }
      return Buffer.from(encoded, 'base64url');
    } catch (error) {
      throw new Error(`Failed to decode key: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  }

  /**
   * Create options for key derivation
   */
  createKeyDerivationOptions(salt: Buffer): KeyDerivationOptions {
    return KeyDerivation.createOptions(salt, this.iterations, this.keyLength);
  }

  /**
   * Full encryption workflow with key derivation
   */
  async encryptWithPassword(data: Buffer | string, password: string): Promise<{ encryptedData: EncryptedData; salt: Buffer }> {
    const salt = this.generateSalt();
    const key = this.deriveKey(password, salt);
    
    let encryptedData: EncryptedData;
    if (Buffer.isBuffer(data)) {
      encryptedData = this.encryptMemory(data, key);
    } else {
      encryptedData = this.encryptString(data, key);
    }
    
    return { encryptedData, salt };
  }

  /**
   * Full decryption workflow with key derivation
   */
  async decryptWithPassword(encryptedData: EncryptedData, salt: Buffer, password: string): Promise<Buffer> {
    const key = this.deriveKey(password, salt);
    return this.decryptMemory(encryptedData, key);
  }

  /**
   * Get current configuration
   */
  getOptions(): EncryptionOptions {
    return {
      iterations: this.iterations,
      keyLength: this.keyLength
    };
  }
}