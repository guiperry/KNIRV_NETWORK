import { MemoryEncryption } from './security';
import { KeyDerivation } from './key_derivation';
import { AESEncryption } from './encryption';
// Export all classes and utilities
export { MemoryEncryption, KeyDerivation, AESEncryption };
/**
 * Factory function to create a memory encryption instance
 */
export function createMemoryEncryption(options) {
    return new MemoryEncryption(options);
}
/**
 * Default memory encryption instance
 */
export const defaultMemoryEncryption = createMemoryEncryption();
/**
 * Quick encryption helpers
 */
export async function encryptWithPassword(data, password) {
    return defaultMemoryEncryption.encryptWithPassword(data, password);
}
export async function decryptWithPassword(encryptedData, salt, password) {
    return defaultMemoryEncryption.decryptWithPassword(encryptedData, salt, password);
}
//# sourceMappingURL=index.js.map