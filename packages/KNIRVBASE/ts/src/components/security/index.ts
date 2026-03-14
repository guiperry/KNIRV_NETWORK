import { MemoryEncryption } from './security';
import { KeyDerivation } from './key_derivation';
import { AESEncryption } from './encryption';
import { EncryptionOptions, EncryptedData, KeyDerivationOptions } from './types';

// Export all classes and utilities
export {
  MemoryEncryption,
  KeyDerivation,
  AESEncryption,
  EncryptionOptions,
  EncryptedData,
  KeyDerivationOptions
};

/**
 * Factory function to create a memory encryption instance
 */
export function createMemoryEncryption(options?: EncryptionOptions): MemoryEncryption {
  return new MemoryEncryption(options);
}

/**
 * Default memory encryption instance
 */
export const defaultMemoryEncryption = createMemoryEncryption();

/**
 * Quick encryption helpers
 */
export async function encryptWithPassword(data: Buffer | string, password: string): Promise<{ encryptedData: EncryptedData; salt: Buffer }> {
  return defaultMemoryEncryption.encryptWithPassword(data, password);
}

export async function decryptWithPassword(encryptedData: EncryptedData, salt: Buffer, password: string): Promise<Buffer> {
  return defaultMemoryEncryption.decryptWithPassword(encryptedData, salt, password);
}