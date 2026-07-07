/**
 * Encrypts data using AES encryption with PBKDF2 key derivation
 * @param data - The data to encrypt
 * @param password - The password to use for encryption
 * @returns Promise<string> - The encrypted data as a base64 string
 */
export declare function encryptAES(data: string, password: string): Promise<string>;
/**
 * Decrypts AES encrypted data
 * @param encryptedData - The encrypted data as a base64 string
 * @param password - The password to use for decryption
 * @returns Promise<string> - The decrypted data
 */
export declare function decryptAES(encryptedData: string, password: string): Promise<string>;
/**
 * Generates a cryptographic key using KDF (Key Derivation Function)
 * @param password - The password to derive key from
 * @param salt - Optional salt (will generate random if not provided)
 * @returns Promise<string> - The derived key as hex string
 */
export declare function makeCryptKey(password: string, salt?: string): Promise<string>;
/**
 * Computes SHA256 hash of input data
 * @param data - The data to hash
 * @returns Promise<string> - The SHA256 hash as hex string
 */
export declare function sha256(data: string): Promise<string>;
/**
 * Generates a random salt
 * @param length - Length in bytes (default: 16)
 * @returns string - Random salt as hex string
 */
export declare function generateSalt(length?: number): string;
/**
 * Generates a random IV (Initialization Vector)
 * @param length - Length in bytes (default: 16)
 * @returns string - Random IV as hex string
 */
export declare function generateIV(length?: number): string;
declare const _default: {
    encryptAES: typeof encryptAES;
    decryptAES: typeof decryptAES;
    makeCryptKey: typeof makeCryptKey;
    sha256: typeof sha256;
    generateSalt: typeof generateSalt;
    generateIV: typeof generateIV;
};
export default _default;
//# sourceMappingURL=index.d.ts.map