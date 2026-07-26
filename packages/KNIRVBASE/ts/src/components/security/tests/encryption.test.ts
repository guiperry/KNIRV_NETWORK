import { AESEncryption } from '../encryption';

describe('AESEncryption', () => {
  const testKey = Buffer.from('12345678901234567890123456789012'); // 32 bytes

  describe('encrypt/decrypt', () => {
    it('should encrypt and decrypt data correctly', () => {
      const data = Buffer.from('This is test data for encryption');
      const encrypted = AESEncryption.encrypt(data, testKey);
      const decrypted = AESEncryption.decrypt(encrypted, testKey);

      expect(decrypted.equals(data)).toBe(true);
    });

    it('should produce different ciphertexts for same data', () => {
      const data = Buffer.from('Test data');
      const encrypted1 = AESEncryption.encrypt(data, testKey);
      const encrypted2 = AESEncryption.encrypt(data, testKey);

      expect(encrypted1.data).not.toBe(encrypted2.data);
      expect(encrypted1.nonce).not.toBe(encrypted2.nonce);
    });
  });

  describe('encryptString/decryptString', () => {
    it('should encrypt and decrypt strings correctly', () => {
      const text = 'Hello, World! 👋';
      const encrypted = AESEncryption.encryptString(text, testKey);
      const decrypted = AESEncryption.decryptString(encrypted, testKey);

      expect(decrypted).toBe(text);
    });

    it('should handle empty string', () => {
      const text = '';
      const encrypted = AESEncryption.encryptString(text, testKey);
      const decrypted = AESEncryption.decryptString(encrypted, testKey);

      expect(decrypted).toBe(text);
    });

    it('should handle unicode characters', () => {
      const text = '测试中文 🌟 ñoël';
      const encrypted = AESEncryption.encryptString(text, testKey);
      const decrypted = AESEncryption.decryptString(encrypted, testKey);

      expect(decrypted).toBe(text);
    });
  });

  describe('encryptJSON/decryptJSON', () => {
    it('should encrypt and decrypt JSON objects correctly', () => {
      const obj = {
        name: 'John Doe',
        age: 30,
        active: true,
        scores: [95, 87, 92],
        address: {
          street: '123 Main St',
          city: 'Anytown'
        }
      };

      const encrypted = AESEncryption.encryptJSON(obj, testKey);
      const decrypted = AESEncryption.decryptJSON<typeof obj>(encrypted, testKey);

      expect(decrypted).toEqual(obj);
    });

    it('should handle null values', () => {
      const obj = { value: null, other: 'test' };
      const encrypted = AESEncryption.encryptJSON(obj, testKey);
      const decrypted = AESEncryption.decryptJSON(encrypted, testKey);

      expect(decrypted.value).toBeNull();
      expect(decrypted.other).toBe('test');
    });

    it('should handle nested objects', () => {
      const obj = {
        level1: {
          level2: {
            level3: 'deep value'
          }
        }
      };

      const encrypted = AESEncryption.encryptJSON(obj, testKey);
      const decrypted = AESEncryption.decryptJSON(encrypted, testKey);

      expect(decrypted.level1.level2.level3).toBe('deep value');
    });
  });

  describe('error handling', () => {
    it('should throw error for invalid key length', () => {
      const invalidKey = Buffer.from('short-key');
      const data = Buffer.from('test data');

      expect(() => AESEncryption.encrypt(data, invalidKey)).toThrow('Invalid key length');
      expect(() => AESEncryption.decrypt({ data: 'test', nonce: 'test' }, invalidKey)).toThrow('Invalid key length');
    });

    it('should throw error for invalid ciphertext', () => {
      const invalidEncrypted = {
        data: 'invalid-base64',
        nonce: Buffer.from('123456789012').toString('base64')
      };

      expect(() => AESEncryption.decrypt(invalidEncrypted, testKey)).toThrow();
    });

    it('should throw error for too short ciphertext', () => {
      const shortEncrypted = {
        data: Buffer.from('short').toString('base64'),
        nonce: Buffer.from('123456789012').toString('base64')
      };

      expect(() => AESEncryption.decrypt(shortEncrypted, testKey)).toThrow('Invalid encrypted data');
    });
  });

  describe('data integrity', () => {
    it('should detect tampered data', () => {
      const data = Buffer.from('Original data');
      const encrypted = AESEncryption.encrypt(data, testKey);

      // Tamper with the encrypted data
      const encryptedBuffer = Buffer.from(encrypted.data, 'base64');
      encryptedBuffer[0] = encryptedBuffer[0] ^ 0xFF; // Flip some bits
      const tampered = {
        ...encrypted,
        data: encryptedBuffer.toString('base64')
      };

      expect(() => AESEncryption.decrypt(tampered, testKey)).toThrow();
    });
  });
});