// Mock implementation for @knirvsdk/crypto
export const encryptAES = jest.fn(async (data: string, key: string): Promise<string> => {
  // Simulate proper encryption behavior
  if (data === null || data === undefined || key === null || key === undefined) {
    throw new Error('Data and password are required for encryption');
  }
  if (typeof data !== 'string' || typeof key !== 'string') {
    throw new Error('Data and password must be strings');
  }
  if (key === '') {
    throw new Error('Password cannot be empty');
  }
  return `encrypted_${data}_with_${key}`;
});

export const decryptAES = jest.fn(async (encryptedData: string, key: string): Promise<string> => {
  // Simulate proper decryption behavior
  if (encryptedData === null || encryptedData === undefined || key === null || key === undefined) {
    throw new Error('Encrypted data and password are required for decryption');
  }
  if (typeof encryptedData !== 'string' || typeof key !== 'string') {
    throw new Error('Encrypted data and password must be strings');
  }
  if (!encryptedData || !key) {
    throw new Error('Encrypted data and password cannot be empty');
  }

  // Check for wrong password simulation
  if (key === 'wrong-password') {
    throw new Error('Invalid password or corrupted data');
  }

  // Check for corrupted data simulation (only if it ends with xxxxx, not contains)
  if (encryptedData.endsWith('xxxxx')) {
    throw new Error('Invalid password or corrupted data');
  }

  return encryptedData.replace(`encrypted_`, '').replace(`_with_${key}`, '');
});

export const makeCryptKey = jest.fn(async (password: string, salt?: string): Promise<string> => {
  // Simulate proper key derivation behavior
  if (password === null || password === undefined) {
    throw new Error('Password is required for key derivation');
  }
  if (typeof password !== 'string') {
    throw new Error('Password must be a string');
  }
  if (password === '') {
    throw new Error('Password cannot be empty');
  }

  // Return different keys for different passwords
  const mockKeys: Record<string, string> = {
    'password1': '7aa9df508b9bd635d62e2d349db1be21eee18ba3a81da5276c6a946bf9896c66',
    'password2': '8bb0e0619c0ce746e73e3d45aec2cf32fef29cb4b92eb6387d7b057c0a97d77',
    'test-password': '7aa9df508b9bd635d62e2d349db1be21eee18ba3a81da5276c6a946bf9896c66'
  };

  return mockKeys[password] || '7aa9df508b9bd635d62e2d349db1be21eee18ba3a81da5276c6a946bf9896c66';
});

export const sha256 = jest.fn(async (data: string): Promise<string> => {
  // Simulate proper SHA256 behavior
  if (!data) {
    throw new Error('Data is required for hashing');
  }

  // Return realistic SHA256 hash for test data
  const mockHashes: Record<string, string> = {
    'test data': '100cb86f7722146b7238374b641c339ccf8b42dc16f72e9e2c71e8d1741f5397',
    'test-password-123': '100cb86f7722146b7238374b641c339ccf8b42dc16f72e9e2c71e8d1741f5397',
    'input1': 'a1b2c3d4e5f6789012345678901234567890123456789012345678901234567890',
    'input2': 'b2c3d4e5f6789012345678901234567890123456789012345678901234567890a1',
    'TESTTESTTESTTESTpassword1{"algorithm":"argon2id","params":{"outputLength":32,"opsLimit":24,"memLimitKib":12288}}': 'c1d2e3f4567890123456789012345678901234567890123456789012345678901234',
    'TESTTESTTESTTESTpassword2{"algorithm":"argon2id","params":{"outputLength":32,"opsLimit":24,"memLimitKib":12288}}': 'd2e3f4567890123456789012345678901234567890123456789012345678901234c1',
    'SALT1SALT1SALT1SAtest-password-123{"algorithm":"argon2id","params":{"outputLength":32,"opsLimit":24,"memLimitKib":12288}}': 'e3f4567890123456789012345678901234567890123456789012345678901234d2e3',
    'SALT2SALT2SALT2SAtest-password-123{"algorithm":"argon2id","params":{"outputLength":32,"opsLimit":24,"memLimitKib":12288}}': 'f4567890123456789012345678901234567890123456789012345678901234e3f4'
  };

  return mockHashes[data] || 'a6a22ebe2861e3c544e18232f0a909cb8b3def839e3ca751b885f220636b0a90';
});

export default {
  encryptAES,
  decryptAES,
  makeCryptKey,
  sha256
};
