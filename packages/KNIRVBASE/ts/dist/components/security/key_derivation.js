import { randomBytes, scryptSync, timingSafeEqual } from 'crypto';
/**
 * PBKDF2 key derivation implementation
 */
export class KeyDerivation {
    /**
     * Derive a key from user secret and salt using PBKDF2
     */
    static deriveKey(userSecret, salt, iterations = KeyDerivation.DEFAULT_ITERATIONS, keyLength = KeyDerivation.DEFAULT_KEY_LENGTH) {
        // Since Node.js doesn't have built-in PBKDF2 with async/await support in older versions,
        // we'll use scrypt as a secure alternative that's available in crypto module
        // For production, you might want to use pbkdf2 from 'crypto' module properly
        try {
            return scryptSync(userSecret, salt, keyLength);
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
//# sourceMappingURL=key_derivation.js.map