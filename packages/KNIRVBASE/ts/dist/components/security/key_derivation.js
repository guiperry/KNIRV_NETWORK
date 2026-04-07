import { pbkdf2Sync, randomBytes, timingSafeEqual } from 'crypto';
/**
 * PBKDF2 key derivation implementation
 */
export class KeyDerivation {
    /**
     * Derive a key from user secret and salt using PBKDF2
     */
    static deriveKey(userSecret, salt, iterations = KeyDerivation.DEFAULT_ITERATIONS, keyLength = KeyDerivation.DEFAULT_KEY_LENGTH) {
        try {
            return pbkdf2Sync(userSecret, salt, iterations, keyLength, KeyDerivation.DIGEST);
        }
        catch (error) {
            throw new Error(`Failed to derive key: ${error instanceof Error ? error.message : 'Unknown error'}`);
        }
    }
    /**
     * Generate a random salt for key derivation
     */
    static generateSalt() {
        return randomBytes(KeyDerivation.SALT_LENGTH);
    }
    /**
     * Derive key with options object
     */
    static deriveKeyWithOptions(userSecret, options) {
        const salt = Buffer.from(options.salt, 'base64');
        return KeyDerivation.deriveKey(userSecret, salt, options.iterations || KeyDerivation.DEFAULT_ITERATIONS, options.keyLength || KeyDerivation.DEFAULT_KEY_LENGTH);
    }
    /**
     * Create options for key derivation
     */
    static createOptions(salt, iterations, keyLength) {
        return {
            salt: salt.toString('base64'),
            iterations: iterations || KeyDerivation.DEFAULT_ITERATIONS,
            keyLength: keyLength || KeyDerivation.DEFAULT_KEY_LENGTH
        };
    }
    /**
     * Verify key derivation (timing safe comparison)
     */
    static verifyKey(key1, key2) {
        if (key1.length !== key2.length) {
            return false;
        }
        return timingSafeEqual(key1, key2);
    }
}
KeyDerivation.DEFAULT_ITERATIONS = 100000;
KeyDerivation.DEFAULT_KEY_LENGTH = 32;
KeyDerivation.SALT_LENGTH = 16;
KeyDerivation.DIGEST = 'sha512';
//# sourceMappingURL=key_derivation.js.map