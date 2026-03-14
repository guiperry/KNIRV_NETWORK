import { 
  createMemoryEncryption, 
  defaultMemoryEncryption,
  encryptWithPassword,
  decryptWithPassword
} from '../index';

describe('Security Index', () => {
  describe('factory functions', () => {
    it('should create memory encryption with default options', () => {
      const encryption = createMemoryEncryption();
      expect(encryption).toBeDefined();
      const options = encryption.getOptions();
      expect(options.iterations).toBe(100000);
      expect(options.keyLength).toBe(32);
    });

    it('should create memory encryption with custom options', () => {
      const encryption = createMemoryEncryption({
        iterations: 50000,
        keyLength: 64
      });
      const options = encryption.getOptions();
      expect(options.iterations).toBe(50000);
      expect(options.keyLength).toBe(64);
    });
  });

  describe('default instance', () => {
    it('should have default memory encryption instance', () => {
      expect(defaultMemoryEncryption).toBeDefined();
      expect(defaultMemoryEncryption.getOptions().iterations).toBe(100000);
    });
  });

  describe('quick helpers', () => {
    it('should encrypt with password using default instance', async () => {
      const data = 'Test data';
      const password = 'test-password';
      
      const result = await encryptWithPassword(data, password);
      
      expect(result.encryptedData).toBeDefined();
      expect(result.salt).toBeDefined();
      expect(result.salt.length).toBe(16);
    });

    it('should decrypt with password using default instance', async () => {
      const data = 'Test data';
      const password = 'test-password';
      
      const { encryptedData, salt } = await encryptWithPassword(data, password);
      const decrypted = await decryptWithPassword(encryptedData, salt, password);
      
      expect(decrypted.toString()).toBe(data);
    });

    it('should fail to decrypt with wrong password', async () => {
      const data = 'Test data';
      const correctPassword = 'correct-password';
      const wrongPassword = 'wrong-password';
      
      const { encryptedData, salt } = await encryptWithPassword(data, correctPassword);
      
      await expect(
        decryptWithPassword(encryptedData, salt, wrongPassword)
      ).rejects.toThrow();
    });
  });
});