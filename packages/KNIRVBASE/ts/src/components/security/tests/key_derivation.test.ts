import { KeyDerivation } from '../key_derivation';

describe('KeyDerivation', () => {
  describe('deriveKey', () => {
    it('should derive key with default parameters', () => {
      const salt = Buffer.from('test-salt-1234567890123456');
      const key = KeyDerivation.deriveKey('test-secret', salt);

      expect(key.length).toBe(32);
    });

    it('should derive same key for same inputs', () => {
      const salt = Buffer.from('test-salt-1234567890123456');
      const key1 = KeyDerivation.deriveKey('test-secret', salt);
      const key2 = KeyDerivation.deriveKey('test-secret', salt);

      expect(KeyDerivation.verifyKey(key1, key2)).toBe(true);
    });

    it('should derive different keys for different secrets', () => {
      const salt = Buffer.from('test-salt-1234567890123456');
      const key1 = KeyDerivation.deriveKey('secret1', salt);
      const key2 = KeyDerivation.deriveKey('secret2', salt);

      expect(KeyDerivation.verifyKey(key1, key2)).toBe(false);
    });

    it('should derive different keys with custom parameters', () => {
      const salt = Buffer.from('test-salt-1234567890123456');
      const key1 = KeyDerivation.deriveKey('test-secret', salt, 50000, 16);
      const key2 = KeyDerivation.deriveKey('test-secret', salt, 100000, 16);

      expect(key1.length).toBe(16);
      expect(key2.length).toBe(16);
      expect(KeyDerivation.verifyKey(key1, key2)).toBe(false);
    });
  });

  describe('generateSalt', () => {
    it('should generate salt of correct length', () => {
      const salt = KeyDerivation.generateSalt();
      expect(salt.length).toBe(16);
    });

    it('should generate different salts', () => {
      const salt1 = KeyDerivation.generateSalt();
      const salt2 = KeyDerivation.generateSalt();

      expect(salt1.equals(salt2)).toBe(false);
    });
  });

  describe('deriveKeyWithOptions', () => {
    it('should derive key using options object', () => {
      const salt = KeyDerivation.generateSalt();
      const options = {
        salt: salt.toString('base64'),
        iterations: 50000,
        keyLength: 24
      };

      const key = KeyDerivation.deriveKeyWithOptions('test-secret', options);
      expect(key.length).toBe(24);
    });
  });

  describe('createOptions', () => {
    it('should create options object', () => {
      const salt = KeyDerivation.generateSalt();
      const options = KeyDerivation.createOptions(salt, 50000, 24);

      expect(options.salt).toBe(salt.toString('base64'));
      expect(options.iterations).toBe(50000);
      expect(options.keyLength).toBe(24);
    });

    it('should use defaults when parameters not provided', () => {
      const salt = KeyDerivation.generateSalt();
      const options = KeyDerivation.createOptions(salt);

      expect(options.iterations).toBe(100000);
      expect(options.keyLength).toBe(32);
    });
  });

  describe('verifyKey', () => {
    it('should verify identical keys', () => {
      const key = KeyDerivation.deriveKey('test-secret', KeyDerivation.generateSalt());
      expect(KeyDerivation.verifyKey(key, key)).toBe(true);
    });

    it('should reject different keys', () => {
      const key1 = KeyDerivation.deriveKey('secret1', KeyDerivation.generateSalt());
      const key2 = KeyDerivation.deriveKey('secret2', KeyDerivation.generateSalt());
      expect(KeyDerivation.verifyKey(key1, key2)).toBe(false);
    });

    it('should reject keys of different lengths', () => {
      const key1 = KeyDerivation.deriveKey('test-secret', KeyDerivation.generateSalt(), 50000, 16);
      const key2 = KeyDerivation.deriveKey('test-secret', KeyDerivation.generateSalt(), 50000, 32);
      expect(KeyDerivation.verifyKey(key1, key2)).toBe(false);
    });
  });
});