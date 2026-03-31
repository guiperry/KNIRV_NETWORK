import { MemoryEncryption } from './security';
import { KeyDerivation } from './key_derivation';
import { AESEncryption } from './encryption';
import { EncryptionOptions, EncryptedData, KeyDerivationOptions } from './types';
export { MemoryEncryption, KeyDerivation, AESEncryption, EncryptionOptions, EncryptedData, KeyDerivationOptions };
/**
 * Factory function to create a memory encryption instance
 */
export declare function createMemoryEncryption(options?: EncryptionOptions): MemoryEncryption;
/**
 * Default memory encryption instance
 */
export declare const defaultMemoryEncryption: MemoryEncryption;
/**
 * Quick encryption helpers
 */
export declare function encryptWithPassword(data: Buffer | string, password: string): Promise<{
    encryptedData: EncryptedData;
    salt: Buffer;
}>;
export declare function decryptWithPassword(encryptedData: EncryptedData, salt: Buffer, password: string): Promise<Buffer>;
//# sourceMappingURL=index.d.ts.map