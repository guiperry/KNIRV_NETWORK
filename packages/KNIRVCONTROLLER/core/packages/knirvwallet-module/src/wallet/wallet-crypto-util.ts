import CryptoJS from 'crypto-js';
import { sha256 } from '../crypto';
import { Argon2id, isArgon2idOptions } from '../crypto';
import { toAscii, toHex } from '../encoding';

interface KdfConfiguration {
  /**
   * An algorithm identifier, such as "argon2id" or "scrypt".
   */
  readonly algorithm: string;
  /** A map of algorithm-specific parameters */
  readonly params: Record<string, unknown>;
}

export async function executeKdf(
  salt: string,
  password: string,
  configuration: KdfConfiguration,
): Promise<Uint8Array> {
  // Validate configuration
  if (!configuration || !configuration.algorithm) {
    throw new Error('Invalid KDF configuration: missing algorithm');
  }

  // Only support argon2id for backward compatibility, but use KNIRV crypto implementation
  if (configuration.algorithm !== 'argon2id') {
    throw new Error(`Unsupported KDF algorithm: ${configuration.algorithm}`);
  }

  // Use crypto-js implementation for KDF operations
  const hexKey = await makeCryptKey(password);
  // Convert hex string to Uint8Array
  const bytes = new Uint8Array(hexKey.length / 2);
  for (let i = 0; i < hexKey.length; i += 2) {
    bytes[i / 2] = parseInt(hexKey.substr(i, 2), 16);
  }
  return bytes;
}

export const makeCryptKey = async (password: string) => {
  const SALT_KEY = process.env.SALT_KEY ?? 'knirv-default-salt';
  return CryptoJS.PBKDF2(password, SALT_KEY, { keySize: 256/32, iterations: 1000 }).toString();
};

export const encryptSha256 = async (password: string) => {
  const cryptKey = await makeCryptKey(password);
  const hash = sha256(new TextEncoder().encode(cryptKey));
  return toHex(hash);
};

export const encryptAES = async (value: string, password: string) => {
  return CryptoJS.AES.encrypt(value, password).toString();
};

export const decryptAES = async (encryptedValue: string, password: string) => {
  const bytes = CryptoJS.AES.decrypt(encryptedValue, password);
  return bytes.toString(CryptoJS.enc.Utf8);
};
