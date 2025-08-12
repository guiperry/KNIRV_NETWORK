import CryptoJS from 'crypto-js';

/**
 * Encrypts data using AES encryption with PBKDF2 key derivation
 * @param data - The data to encrypt
 * @param password - The password to use for encryption
 * @returns Promise<string> - The encrypted data as a base64 string
 */
export async function encryptAES(data: string, password: string): Promise<string> {
  if (data === null || data === undefined || password === null || password === undefined) {
    throw new Error('Data and password are required for encryption');
  }

  // Allow empty strings but not null/undefined
  if (typeof data !== 'string' || typeof password !== 'string') {
    throw new Error('Data and password must be strings');
  }

  // Allow empty data but not empty password
  if (password === '') {
    throw new Error('Password cannot be empty');
  }

  try {
    // Generate a random salt
    const salt = CryptoJS.lib.WordArray.random(128/8);
    
    // Derive key using PBKDF2
    const key = CryptoJS.PBKDF2(password, salt, {
      keySize: 256/32,
      iterations: 10000
    });

    // Generate random IV
    const iv = CryptoJS.lib.WordArray.random(128/8);

    // Encrypt the data
    const encrypted = CryptoJS.AES.encrypt(data, key, {
      iv: iv,
      padding: CryptoJS.pad.Pkcs7,
      mode: CryptoJS.mode.CBC
    });

    // Combine salt, iv, and encrypted data
    const combined = salt.concat(iv).concat(encrypted.ciphertext);
    
    return CryptoJS.enc.Base64.stringify(combined);
  } catch (error) {
    throw new Error(`Encryption failed: ${error instanceof Error ? error.message : String(error)}`);
  }
}

/**
 * Decrypts AES encrypted data
 * @param encryptedData - The encrypted data as a base64 string
 * @param password - The password to use for decryption
 * @returns Promise<string> - The decrypted data
 */
export async function decryptAES(encryptedData: string, password: string): Promise<string> {
  if (encryptedData === null || encryptedData === undefined || password === null || password === undefined) {
    throw new Error('Encrypted data and password are required for decryption');
  }

  if (typeof encryptedData !== 'string' || typeof password !== 'string') {
    throw new Error('Encrypted data and password must be strings');
  }

  if (!encryptedData || !password) {
    throw new Error('Encrypted data and password cannot be empty');
  }

  try {
    // Parse the base64 data
    const combined = CryptoJS.enc.Base64.parse(encryptedData);
    
    // Extract salt (first 16 bytes)
    const salt = CryptoJS.lib.WordArray.create(combined.words.slice(0, 4));
    
    // Extract IV (next 16 bytes)
    const iv = CryptoJS.lib.WordArray.create(combined.words.slice(4, 8));
    
    // Extract encrypted data (remaining bytes)
    const encrypted = CryptoJS.lib.WordArray.create(combined.words.slice(8));

    // Derive key using same parameters
    const key = CryptoJS.PBKDF2(password, salt, {
      keySize: 256/32,
      iterations: 10000
    });

    // Decrypt the data
    const decrypted = CryptoJS.AES.decrypt(
      CryptoJS.enc.Base64.stringify(encrypted),
      key,
      {
        iv: iv,
        padding: CryptoJS.pad.Pkcs7,
        mode: CryptoJS.mode.CBC
      }
    );

    const decryptedText = decrypted.toString(CryptoJS.enc.Utf8);

    // Check if decryption failed by examining the result
    // Note: sigBytes can be 0 for empty string decryption, which is valid
    if (decrypted.sigBytes < 0) {
      throw new Error('Invalid password or corrupted data');
    }

    // Only check for wrong password if we have a very small result from substantial encrypted data
    // Empty string is a valid decryption result, so we need to be more careful
    if (decryptedText === '' && encryptedData.length > 100 && decrypted.sigBytes === 0) {
      // This suggests the decryption completely failed rather than legitimately producing empty string
      throw new Error('Invalid password or corrupted data');
    }

    // Check for invalid UTF-8 sequences or corrupted characters (indicates wrong password/corruption)
    // Only check if we have actual content (not empty string)
    if (decryptedText.length > 0) {
      if (decryptedText.includes('\u0000') ||
          /[\u0080-\u00FF]/.test(decryptedText) || // Invalid UTF-8 high bytes
          /[^\x20-\x7E\s]/.test(decryptedText.replace(/[\r\n\t]/g, ''))) { // Non-printable chars except whitespace
        throw new Error('Invalid password or corrupted data');
      }
    }

    return decryptedText;
  } catch (error) {
    if (error instanceof Error && error.message === 'Invalid password or corrupted data') {
      throw error;
    }
    throw new Error('Invalid password or corrupted data');
  }
}

/**
 * Generates a cryptographic key using KDF (Key Derivation Function)
 * @param password - The password to derive key from
 * @param salt - Optional salt (will generate random if not provided)
 * @returns Promise<string> - The derived key as hex string
 */
export async function makeCryptKey(password: string, salt?: string): Promise<string> {
  if (password === null || password === undefined) {
    throw new Error('Password is required for key derivation');
  }

  if (typeof password !== 'string') {
    throw new Error('Password must be a string');
  }

  if (password === '') {
    throw new Error('Password cannot be empty');
  }

  try {
    const saltWordArray = salt 
      ? CryptoJS.enc.Utf8.parse(salt)
      : CryptoJS.lib.WordArray.random(128/8);

    const key = CryptoJS.PBKDF2(password, saltWordArray, {
      keySize: 256/32,
      iterations: 10000
    });

    return key.toString(CryptoJS.enc.Hex);
  } catch (error) {
    throw new Error(`Key derivation failed: ${error instanceof Error ? error.message : String(error)}`);
  }
}

/**
 * Computes SHA256 hash of input data
 * @param data - The data to hash
 * @returns Promise<string> - The SHA256 hash as hex string
 */
export async function sha256(data: string): Promise<string> {
  if (!data) {
    throw new Error('Data is required for hashing');
  }

  try {
    const hash = CryptoJS.SHA256(data);
    return hash.toString(CryptoJS.enc.Hex);
  } catch (error) {
    throw new Error(`Hashing failed: ${error instanceof Error ? error.message : String(error)}`);
  }
}

/**
 * Generates a random salt
 * @param length - Length in bytes (default: 16)
 * @returns string - Random salt as hex string
 */
export function generateSalt(length: number = 16): string {
  const salt = CryptoJS.lib.WordArray.random(length);
  return salt.toString(CryptoJS.enc.Hex);
}

/**
 * Generates a random IV (Initialization Vector)
 * @param length - Length in bytes (default: 16)
 * @returns string - Random IV as hex string
 */
export function generateIV(length: number = 16): string {
  const iv = CryptoJS.lib.WordArray.random(length);
  return iv.toString(CryptoJS.enc.Hex);
}

// Export all functions for compatibility
export default {
  encryptAES,
  decryptAES,
  makeCryptKey,
  sha256,
  generateSalt,
  generateIV
};
