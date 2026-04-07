import { describe, it, expect, beforeEach } from '@jest/globals';
import { generatePQCKeyPair, loadPQCKeyPair, marshalPublic, marshalWithPrivateKeys, encrypt, decrypt, sign, verify, isActive, isExpired, PQCKeyPair } from '../keys';
import { EncryptionManager } from '../encryption';

describe('PQC Key Management', () => {
  let keyPair: PQCKeyPair;

  beforeEach(() => {
    keyPair = generatePQCKeyPair('test-key', 'encryption');
  });

  describe('generatePQCKeyPair', () => {
    it('should generate a valid PQC key pair', () => {
      expect(keyPair).toBeDefined();
      expect(keyPair.id).toBeDefined();
      expect(keyPair.name).toBe('test-key');
      expect(keyPair.purpose).toBe('encryption');
      expect(keyPair.algorithm).toBe('Kyber-768+Dilithium-3');
      expect(keyPair.status).toBe('active');
      expect(keyPair.createdAt).toBeInstanceOf(Date);

      // Check Kyber-768 keys (public: 1184 bytes, secret: 2400 bytes)
      expect(keyPair.kyberPublicKeyBytes).toBeDefined();
      expect(keyPair.kyberPrivateKeyBytes).toBeDefined();
      expect(keyPair.kyberPublicKeyBytes.length).toBe(1184);
      expect(keyPair.kyberPrivateKeyBytes!.length).toBe(2400);

      // Check Dilithium-3 keys (public: 1952 bytes, secret: 4032 bytes)
      expect(keyPair.dilithiumPublicKeyBytes).toBeDefined();
      expect(keyPair.dilithiumPrivateKeyBytes).toBeDefined();
      expect(keyPair.dilithiumPublicKeyBytes.length).toBe(1952);
      expect(keyPair.dilithiumPrivateKeyBytes!.length).toBeGreaterThan(0);
    });

    it('should generate unique IDs for different keys', () => {
      const keyPair2 = generatePQCKeyPair('test-key-2', 'encryption');
      expect(keyPair.id).not.toBe(keyPair2.id);
    });
  });

  describe('loadPQCKeyPair', () => {
    it('should load a key pair from marshaled data', () => {
      const marshaled = marshalWithPrivateKeys(keyPair);
      const loaded = loadPQCKeyPair(marshaled);

      expect(loaded.id).toBe(keyPair.id);
      expect(loaded.name).toBe(keyPair.name);
      expect(loaded.purpose).toBe(keyPair.purpose);
      expect(loaded.kyberPublicKeyBytes).toEqual(keyPair.kyberPublicKeyBytes);
      expect(loaded.kyberPrivateKeyBytes).toEqual(keyPair.kyberPrivateKeyBytes);
      expect(loaded.dilithiumPublicKeyBytes).toEqual(keyPair.dilithiumPublicKeyBytes);
      expect(loaded.dilithiumPrivateKeyBytes).toEqual(keyPair.dilithiumPrivateKeyBytes);
    });
  });

  describe('marshalPublic', () => {
    it('should marshal only public keys', () => {
      const marshaled = marshalPublic(keyPair);
      const parsed = JSON.parse(marshaled);

      expect(parsed.id).toBe(keyPair.id);
      expect(parsed.name).toBe(keyPair.name);
      expect(parsed.kyberPublicKeyBytes).toBeDefined();
      expect(parsed.dilithiumPublicKeyBytes).toBeDefined();

      // Private keys should not be present
      expect(parsed.kyberPrivateKeyBytes).toBeUndefined();
      expect(parsed.dilithiumPrivateKeyBytes).toBeUndefined();
    });
  });

  describe('isActive and isExpired', () => {
    it('should return true for active non-expired keys', () => {
      expect(isActive(keyPair)).toBe(true);
      expect(isExpired(keyPair)).toBe(false);
    });

    it('should detect expired keys', () => {
      const expiredKey = { ...keyPair, expiresAt: new Date(Date.now() - 1000) };
      expect(isExpired(expiredKey)).toBe(true);
      expect(isActive(expiredKey)).toBe(false);
    });
  });
});

describe('PQC Encryption/Decryption', () => {
  let keyPair: PQCKeyPair;
  const testData = new Uint8Array([1, 2, 3, 4, 5]);

  beforeEach(() => {
    keyPair = generatePQCKeyPair('encryption-test', 'encryption');
  });

  describe('encrypt/decrypt', () => {
    it('should encrypt and decrypt data correctly', () => {
      const encrypted = encrypt(keyPair, testData);
      expect(encrypted).toBeDefined();
      // Output is at least KEM ciphertext (1088) + nonce (12) + auth tag (16) + plaintext
      expect(encrypted.length).toBeGreaterThan(1088 + 12 + 16);

      const decrypted = decrypt(keyPair, encrypted);
      expect(decrypted).toEqual(testData);
    });

    it('should fail decryption with wrong private key', () => {
      const otherKeyPair = generatePQCKeyPair('other-key', 'encryption');
      const encrypted = encrypt(keyPair, testData);

      expect(() => decrypt(otherKeyPair, encrypted)).toThrow();
    });
  });
});

describe('PQC Signatures', () => {
  let keyPair: PQCKeyPair;
  const testMessage = new Uint8Array([72, 101, 108, 108, 111]); // "Hello"

  beforeEach(() => {
    keyPair = generatePQCKeyPair('signature-test', 'signature');
  });

  describe('sign/verify', () => {
    it('should sign and verify messages correctly', () => {
      const signature = sign(keyPair, testMessage);
      expect(signature).toBeDefined();
      // ML-DSA-65 signature is 3309 bytes
      expect(signature.length).toBe(3309);

      const isValid = verify(keyPair, testMessage, signature);
      expect(isValid).toBe(true);
    });

    it('should reject tampered messages', () => {
      const signature = sign(keyPair, testMessage);
      const tamperedMessage = new Uint8Array([72, 101, 108, 108, 111, 33]); // "Hello!"

      const isValid = verify(keyPair, tamperedMessage, signature);
      expect(isValid).toBe(false);
    });

    it('should reject invalid signatures', () => {
      const fakeSignature = new Uint8Array([1, 2, 3, 4, 5]);
      const isValid = verify(keyPair, testMessage, fakeSignature);
      expect(isValid).toBe(false);
    });
  });
});

describe('EncryptionManager', () => {
  let manager: EncryptionManager;
  let keyPair: PQCKeyPair;

  beforeEach(() => {
    manager = new EncryptionManager();
    keyPair = generatePQCKeyPair('manager-test', 'encryption');
    manager.setMasterKey(keyPair);
  });

  describe('encryptData/decryptData', () => {
    it('should encrypt and decrypt data with metadata', async () => {
      const plaintext = Buffer.from('sensitive data');
      const encrypted = await manager.encryptData(plaintext, keyPair.id);

      expect(encrypted).toBeDefined();
      expect(typeof encrypted).toBe('string');

      const decrypted = await manager.decryptData(encrypted);
      expect(Buffer.from(decrypted).toString()).toBe('sensitive data');
    });

    it('should fail decryption with wrong key', async () => {
      const plaintext = Buffer.from('secret');

      const cacheOnlyKey = manager.generateDataEncryptionKey('cache-only-key');
      const encrypted = await manager.encryptData(plaintext, cacheOnlyKey.id);

      manager.removeKey(cacheOnlyKey.id);

      await expect(manager.decryptData(encrypted)).rejects.toThrow(/not found in cache/);
    });
  });

  describe('key management', () => {
    it('should cache and retrieve keys', () => {
      const newKey = manager.generateDataEncryptionKey('cached-key');
      expect(newKey).toBeDefined();
      expect(manager.getMasterKey()).toBe(keyPair);
    });

    it('should handle multiple keys', async () => {
      const key1 = manager.generateDataEncryptionKey('key1');
      const key2 = manager.generateDataEncryptionKey('key2');

      const data1 = Buffer.from('data for key1');
      const data2 = Buffer.from('data for key2');

      const encrypted1 = await manager.encryptData(data1, key1.id);
      const encrypted2 = await manager.encryptData(data2, key2.id);

      const decrypted1 = await manager.decryptData(encrypted1);
      const decrypted2 = await manager.decryptData(encrypted2);

      expect(Buffer.from(decrypted1).toString()).toBe('data for key1');
      expect(Buffer.from(decrypted2).toString()).toBe('data for key2');
    });
  });

  describe('error handling', () => {
    it('should fail encryption without master key', async () => {
      const managerWithoutKey = new EncryptionManager();
      const data = Buffer.from('test');

      await expect(managerWithoutKey.encryptData(data, 'nonexistent')).rejects.toThrow();
    });

    it('should fail with inactive keys', async () => {
      const inactiveKey = generatePQCKeyPair('inactive', 'encryption');
      inactiveKey.status = 'revoked';

      manager.cacheKey(inactiveKey.id, inactiveKey);
      const data = Buffer.from('test');

      await expect(manager.encryptData(data, inactiveKey.id)).rejects.toThrow(/is not active/);
    });
  });
});

describe('PQC Integration Tests', () => {
  it('should perform full encryption workflow', async () => {
    const manager = new EncryptionManager();
    const kp = generatePQCKeyPair('integration-test', 'encryption');
    manager.setMasterKey(kp);

    const originalData = Buffer.from('This is confidential information');
    const encrypted = await manager.encryptData(originalData, kp.id);
    const decrypted = await manager.decryptData(encrypted);

    expect(Buffer.from(decrypted).toString()).toBe('This is confidential information');
  });

  it('should perform full signature workflow', async () => {
    const kp = generatePQCKeyPair('signature-integration', 'signature');

    const message = Buffer.from('Important document content');
    const signature = sign(kp, message);
    const isValid = verify(kp, message, signature);
    expect(isValid).toBe(true);

    const manager = new EncryptionManager();
    manager.setMasterKey(kp);

    const data = Buffer.from('signed and encrypted data');
    const encrypted = await manager.encryptData(data, kp.id);
    const decrypted = await manager.decryptData(encrypted);

    expect(Buffer.from(decrypted).toString()).toBe('signed and encrypted data');
  });
});
