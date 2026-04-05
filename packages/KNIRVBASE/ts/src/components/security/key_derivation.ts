import { pbkdf2Sync, randomBytes, timingSafeEqual } from 'crypto';
import { KeyDerivationOptions } from './types';

/**
 * PBKDF2 key derivation implementation
 */
export class KeyDerivation {
  private static readonly DEFAULT_ITERATIONS = 100000;
  private static readonly DEFAULT_KEY_LENGTH = 32;
  private static readonly SALT_LENGTH = 16;
  private static readonly DIGEST = 'sha512';

  /**
   * Derive a key from user secret and salt using PBKDF2
   */
  static deriveKey(
    userSecret: string,
    salt: Buffer,
    iterations: number = KeyDerivation.DEFAULT_ITERATIONS,
    keyLength: number = KeyDerivation.DEFAULT_KEY_LENGTH
  ): Buffer {
    try {
      return pbkdf2Sync(userSecret, salt, iterations, keyLength, KeyDerivation.DIGEST);
    } catch (error) {
      throw new Error(`Failed to derive key: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  }

  /**
   * Generate a random salt for key derivation
   */
  static generateSalt(): Buffer {
    return randomBytes(KeyDerivation.SALT_LENGTH);
  }

  /**
   * Derive key with options object
   */
  static deriveKeyWithOptions(userSecret: string, options: KeyDerivationOptions): Buffer {
    const salt = Buffer.from(options.salt, 'base64');
    return KeyDerivation.deriveKey(
      userSecret,
      salt,
      options.iterations || KeyDerivation.DEFAULT_ITERATIONS,
      options.keyLength || KeyDerivation.DEFAULT_KEY_LENGTH
    );
  }

  /**
   * Create options for key derivation
   */
  static createOptions(salt: Buffer, iterations?: number, keyLength?: number): KeyDerivationOptions {
    return {
      salt: salt.toString('base64'),
      iterations: iterations || KeyDerivation.DEFAULT_ITERATIONS,
      keyLength: keyLength || KeyDerivation.DEFAULT_KEY_LENGTH
    };
  }

  /**
   * Verify key derivation (timing safe comparison)
   */
  static verifyKey(key1: Buffer, key2: Buffer): boolean {
    if (key1.length !== key2.length) {
      return false;
    }
    return timingSafeEqual(key1, key2);
  }
}