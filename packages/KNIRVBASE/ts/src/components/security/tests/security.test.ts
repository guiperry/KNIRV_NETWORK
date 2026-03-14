import { MemoryEncryption } from '../security';
import { KeyDerivation } from '../key_derivation';
import { AESEncryption } from '../encryption';

describe('MemoryEncryption', () => {
  let encryption: MemoryEncryption;
  const testKey = Buffer.from('12345678901234567890123456789012'); // 32 bytes

  beforeEach(() => {
    encryption = new MemoryEncryption();
  });

  describe('constructor', () => {
    it('should create MemoryEncryption with default options', () => {
      expect(encryption).toBeInstanceOf(MemoryEncryption);
      const options = encryption.getOptions();
      expect(options.iterations).toBe(100000);
      expect(options.keyLength).toBe(32);
    });

    it('should create MemoryEncryption with custom options', () => {
      const customEnc = new MemoryEncryption({
        iterations: 50000,
        keyLength: 64
      });
      const options = customEnc.getOptions();
      expect(options.iterations).toBe(50000);
      expect(options.keyLength).toBe(64);
    });
  });

  describe('key derivation', () => {
    it('should derive key from password and salt', () => {
      const salt = Buffer.from('test-salt-1234567890123456'); // 16 bytes
      const key = encryption.deriveKey('test-secret', salt);

      expect(key.length).toBe(32);
    });

    it('should produce same key for same inputs', () => {
      const salt = Buffer.from('test-salt-1234567890123456');
      const key1 = encryption.deriveKey('test-secret', salt);
      const key2 = encryption.deriveKey('test-secret', salt);

      expect(key1.equals(key2)).toBe(true);
    });

    it('should produce different keys for different passwords', () => {
      const salt = Buffer.from('test-salt-1234567890123456');
      const key1 = encryption.deriveKey('password1', salt);
      const key2 = encryption.deriveKey('password2', salt);

      expect(key1.equals(key2)).toBe(false);
    });
  });

  describe('encryption/decryption', () => {
    const plaintext = Buffer.from('This is a test message for encryption');

    it('should encrypt and decrypt data correctly', () => {
      const encrypted = encryption.encryptMemory(plaintext, testKey);
      const decrypted = encryption.decryptMemory(encrypted, testKey);

      expect(encrypted.data).toBeTruthy();
      expect(encrypted.nonce).toBeTruthy();
      expect(decrypted.equals(plaintext)).toBe(true);
    });

    it('should encrypt and decrypt strings correctly', () => {
      const text = 'Hello, World!';
      const encrypted = encryption.encryptString(text, testKey);
      const decrypted = encryption.decryptString(encrypted, testKey);

      expect(decrypted).toBe(text);
    });

    it('should throw error for invalid key length during encryption', () => {
      const invalidKey = Buffer.from('short-key');
      
      expect(() => {
        encryption.encryptMemory(plaintext, invalidKey);
      }).toThrow('Invalid key length');
    });

    it('should throw error for invalid key length during decryption', () => {
      const invalidKey = Buffer.from('short-key');
      const encrypted = encryption.encryptMemory(plaintext, testKey);
      
      expect(() => {
        encryption.decryptMemory(encrypted, invalidKey);
      }).toThrow('Invalid key length');
    });
  });

  describe('salt generation', () => {
    it('should generate random salt', () => {
      const salt1 = encryption.generateSalt();
      const salt2 = encryption.generateSalt();

      expect(salt1.length).toBe(16);
      expect(salt2.length).toBe(16);
      expect(salt1.equals(salt2)).toBe(false);
    });
  });

  describe('key encoding/decoding', () => {
    it('should encode and decode key correctly', () => {
      const encoded = encryption.encodeKey(testKey);
      const decoded = encryption.decodeKey(encoded);

      expect(decoded.equals(testKey)).toBe(true);
    });

    it('should throw error for invalid base64 key', () => {
      expect(() => {
        encryption.decodeKey('invalid-base64!');
      }).toThrow('Failed to decode key');
    });
  });

  describe('password-based encryption', () => {
    it('should encrypt with password and decrypt with same password', async () => {
      const data = 'Sensitive data';
      const password = 'strong-password';

      const { encryptedData, salt } = await encryption.encryptWithPassword(data, password);
      const decrypted = await encryption.decryptWithPassword(encryptedData, salt, password);

      expect(decrypted.toString()).toBe(data);
    });

    it('should fail to decrypt with wrong password', async () => {
      const data = 'Sensitive data';
      const correctPassword = 'strong-password';
      const wrongPassword = 'wrong-password';

      const { encryptedData, salt } = await encryption.encryptWithPassword(data, correctPassword);

      await expect(
        encryption.decryptWithPassword(encryptedData, salt, wrongPassword)
      ).rejects.toThrow();
    });
  });
});