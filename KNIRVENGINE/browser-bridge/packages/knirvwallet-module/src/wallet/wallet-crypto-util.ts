import CryptoJS from 'crypto-js';
import { encryptAES as knirvEncryptAES, decryptAES as knirvDecryptAES, makeCryptKey as knirvMakeCryptKey, sha256 } from '@knirvsdk/crypto';

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

  // Use KNIRV crypto implementation for all KDF operations
  const hexKey = await knirvMakeCryptKey(password, salt);
  // Convert hex string to Uint8Array
  const bytes = new Uint8Array(hexKey.length / 2);
  for (let i = 0; i < hexKey.length; i += 2) {
    bytes[i / 2] = parseInt(hexKey.substr(i, 2), 16);
  }
  return bytes;
}

export const makeCryptKey = async (password: string) => {
  // Use KNIRV crypto implementation with consistent salt
  const SALT_KEY = process.env.SALT_KEY ?? 'knirv-default-salt';
  return await knirvMakeCryptKey(password, SALT_KEY);
};

export const encryptSha256 = async (password: string) => {
  const cryptKey = await makeCryptKey(password);
  return await sha256(cryptKey);
};

export const encryptAES = async (value: string, password: string) => {
  return await knirvEncryptAES(value, password);
};

export const decryptAES = async (encryptedValue: string, password: string) => {
  return await knirvDecryptAES(encryptedValue, password);
};
